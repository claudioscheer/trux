package main

import (
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/parser"
	"github.com/claudioscheer/trux/internal/stdlib"
	"github.com/claudioscheer/trux/internal/token"
)

type definitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type referenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      referenceContext       `json:"context"`
}

type completionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type referenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type symbolKind int

const (
	localSymbol symbolKind = iota
	functionSymbol
)

type symbolDefinition struct {
	Kind   symbolKind
	Name   string
	URI    string
	Path   string
	Pos    token.Position
	Public bool
}

type localDefinition struct {
	Name string
	URI  string
	Pos  token.Position
}

type sourceGraph struct {
	files map[string]*sourceFile
	order []*sourceFile
}

type sourceFile struct {
	path    string
	uri     string
	text    string
	program *ast.Program
}

type packageKey struct {
	dir  string
	name string
}

type importedPackage struct {
	name   string
	files  []*sourceFile
	stdlib bool
}

type completionContext struct {
	member      bool
	packageName string
}

type importStringCompletionContext struct {
	prefix string
	edit   TextEdit
}

type sourceModuleCandidate struct {
	packageName  string
	functionName string
	importPath   string
	path         string
	fn           *ast.FuncDecl
}

type importPathReference struct {
	path  string
	token token.Token
}

type LocationLink struct {
	OriginSelectionRange *Range `json:"originSelectionRange,omitempty"`
	TargetURI            string `json:"targetUri"`
	TargetRange          Range  `json:"targetRange"`
	TargetSelectionRange Range  `json:"targetSelectionRange"`
}

func (s *Server) handleDefinition(raw json.RawMessage) (any, error) {
	var params definitionParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}

	if link, ok, err := s.resolveImportPath(params.TextDocument.URI, params.Position); err != nil {
		return nil, err
	} else if ok {
		return []LocationLink{link}, nil
	}

	if link, ok, err := s.resolveImportPackageQualifier(params.TextDocument.URI, params.Position); err != nil {
		return nil, err
	} else if ok {
		return []LocationLink{link}, nil
	}

	def, ok, err := s.resolveSymbol(params.TextDocument.URI, params.Position)
	if err != nil || !ok {
		return nil, err
	}

	return locationFor(def.URI, def.Pos, def.Name), nil
}

func (s *Server) handleReferences(raw json.RawMessage) (any, error) {
	var params referenceParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}

	def, graph, ok, err := s.resolveSymbolWithGraph(params.TextDocument.URI, params.Position)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []Location{}, nil
	}

	switch def.Kind {
	case localSymbol:
		return collectLocalReferences(graph, def, params.Context.IncludeDeclaration), nil
	case functionSymbol:
		return collectFunctionReferences(graph, def, params.Context.IncludeDeclaration), nil
	default:
		return []Location{}, nil
	}
}

func (s *Server) handleCompletion(raw json.RawMessage) (any, error) {
	var params completionParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}

	text, ok := s.documents[params.TextDocument.URI]
	if !ok {
		var err error
		text, err = readURI(params.TextDocument.URI)
		if err != nil {
			return completionListFromItems(baseCompletionItems()), nil
		}
	}

	path := normalizePath(pathFromURI(params.TextDocument.URI))
	if importContext, ok := importStringCompletionAt(text, params.Position); ok {
		return completionListFromItems(importPathCompletionItems(path, importContext)), nil
	}

	imports := directImportedPackages(path, text, s.documents)
	if !expressionCompletionAt(text, params.Position) {
		return completionListFromItems(declarationCompletionItems()), nil
	}

	context := completionContextAt(text, params.Position)
	if context.member {
		items := []CompletionItem{}
		for _, pkg := range imports {
			if pkg.name != context.packageName {
				continue
			}
			items = append(items, packageMemberCompletionItems(pkg)...)
		}
		if !importedPackageName(imports, context.packageName) {
			if pkg, ok := stdlib.LookupPackage(context.packageName); ok {
				items = append(items, standardPackageMemberCompletionItems(pkg, importEdit(text, pkg.Name))...)
			}
		}
		items = append(items, sourceAutoImportMemberCompletionItems(path, text, s.documents, imports, context.packageName)...)
		return completionListFromItems(items), nil
	}

	items := baseCompletionItems()
	seen := completionLabels(items)
	graph := loadSourceGraph(path, text, s.documents)
	file := graph.files[path]
	if file != nil {
		for _, item := range visibleLocalCompletionItems(file, tokenPositionFromLSP(params.Position)) {
			items = appendCompletionItem(items, seen, item)
		}
		for _, item := range sameFileFunctionCompletionItems(file) {
			items = appendCompletionItem(items, seen, item)
		}
		for _, item := range samePackageFunctionCompletionItems(graph, file) {
			items = appendCompletionItem(items, seen, item)
		}
	}
	for _, item := range importedPackageCompletionItems(imports) {
		items = appendCompletionItem(items, seen, item)
	}
	for _, item := range qualifiedImportedFunctionCompletionItems(imports) {
		items = appendCompletionItem(items, seen, item)
	}
	for _, item := range standardAutoImportFunctionCompletionItems(text, imports) {
		items = appendCompletionItem(items, seen, item)
	}
	for _, item := range sourceAutoImportFunctionCompletionItems(path, text, s.documents, imports) {
		if _, ok := seen[item.Label]; ok {
			continue
		}
		items = append(items, item)
	}

	return completionListFromItems(items), nil
}

func (s *Server) hoverImportPath(uri string, pos Position) (Hover, bool, error) {
	ref, ok, err := s.importPathAt(uri, pos)
	if err != nil || !ok {
		return Hover{}, ok, err
	}

	return Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: "`import \"" + ref.path + "\"`",
		},
		Range: importPathRange(ref.token),
	}, true, nil
}

func (s *Server) hoverFunction(uri string, pos Position) (Hover, bool, error) {
	text, err := s.documentText(uri)
	if err != nil {
		return Hover{}, false, err
	}

	word, ok := wordAt(text, pos)
	if !ok {
		return Hover{}, false, nil
	}

	if value, ok := builtinFunctionHoverText[word]; ok {
		return Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: value,
			},
			Range: rangeForWord(text, pos),
		}, true, nil
	}

	if member, ok := stdlibMemberCallAt(text, uri, pos); ok {
		return Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: "`" + member.Signature() + "`",
			},
			Range: rangeForWord(text, pos),
		}, true, nil
	}

	def, graph, ok, err := s.resolveSymbolWithGraph(uri, pos)
	if err != nil || !ok || def.Kind != functionSymbol {
		return Hover{}, ok && def.Kind == functionSymbol, err
	}

	fn, ok := findFunctionDecl(graph, def)
	if !ok {
		return Hover{}, false, nil
	}

	return Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: "`" + functionSignature(fn) + "`",
		},
		Range: rangeForWord(text, pos),
	}, true, nil
}

func stdlibMemberCallAt(text string, uri string, pos Position) (stdlib.Member, bool) {
	path := normalizePath(pathFromURI(uri))
	program, err := parser.ParseFile(path, text)
	if err != nil {
		return stdlib.Member{}, false
	}
	cursor := tokenPositionFromLSP(pos)
	imports := stdlibImports(program)
	for _, fn := range program.Functions {
		if member, ok := stdlibMemberCallInBlock(fn.Body, imports, cursor); ok {
			return member, true
		}
	}
	return stdlib.Member{}, false
}

func stdlibImports(program *ast.Program) map[string]struct{} {
	imports := map[string]struct{}{}
	for _, importDecl := range program.Imports {
		if stdlib.IsPackage(importDecl.Path) {
			imports[importDecl.Path] = struct{}{}
		}
	}
	return imports
}

func stdlibMemberCallInBlock(block ast.Block, imports map[string]struct{}, cursor token.Position) (stdlib.Member, bool) {
	for _, stmt := range block.Statements {
		if member, ok := stdlibMemberCallInStmt(stmt, imports, cursor); ok {
			return member, true
		}
	}
	return stdlib.Member{}, false
}

func stdlibMemberCallInStmt(stmt ast.Statement, imports map[string]struct{}, cursor token.Position) (stdlib.Member, bool) {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		return stdlibMemberCallInExpr(stmt.Value, imports, cursor)
	case *ast.ReturnStmt:
		return stdlibMemberCallInExpr(stmt.Value, imports, cursor)
	case *ast.AssignStmt:
		return stdlibMemberCallInExpr(stmt.Value, imports, cursor)
	case *ast.IndexAssignStmt:
		if member, ok := stdlibMemberCallInExpr(stmt.Target, imports, cursor); ok {
			return member, true
		}
		return stdlibMemberCallInExpr(stmt.Value, imports, cursor)
	case *ast.IfStmt:
		if member, ok := stdlibMemberCallInExpr(stmt.Condition, imports, cursor); ok {
			return member, true
		}
		if member, ok := stdlibMemberCallInBlock(stmt.Then, imports, cursor); ok {
			return member, true
		}
		if stmt.Else != nil {
			return stdlibMemberCallInBlock(*stmt.Else, imports, cursor)
		}
	case *ast.ForStmt:
		if stmt.Init != nil {
			if member, ok := stdlibMemberCallInStmt(stmt.Init, imports, cursor); ok {
				return member, true
			}
		}
		if member, ok := stdlibMemberCallInExpr(stmt.Condition, imports, cursor); ok {
			return member, true
		}
		if member, ok := stdlibMemberCallInBlock(stmt.Body, imports, cursor); ok {
			return member, true
		}
		if stmt.Post != nil {
			return stdlibMemberCallInStmt(stmt.Post, imports, cursor)
		}
	case *ast.ExprStmt:
		return stdlibMemberCallInExpr(stmt.Expr, imports, cursor)
	}
	return stdlib.Member{}, false
}

func stdlibMemberCallInExpr(expr ast.Expression, imports map[string]struct{}, cursor token.Position) (stdlib.Member, bool) {
	switch expr := expr.(type) {
	case *ast.CallExpr:
		if _, imported := imports[expr.Package]; imported && containsName(cursor, callCalleePosition(expr), expr.Callee) {
			if member, ok := stdlib.LookupMember(expr.Package, expr.Callee); ok {
				return member, true
			}
		}
		for _, arg := range expr.Args {
			if member, ok := stdlibMemberCallInExpr(arg, imports, cursor); ok {
				return member, true
			}
		}
	case *ast.ArrayLiteral:
		return stdlibMemberCallInExprs(expr.Elements, imports, cursor)
	case *ast.ListLiteral:
		return stdlibMemberCallInExprs(expr.Elements, imports, cursor)
	case *ast.MakeExpr:
		return stdlibMemberCallInExpr(expr.Len, imports, cursor)
	case *ast.BinaryExpr:
		if member, ok := stdlibMemberCallInExpr(expr.Left, imports, cursor); ok {
			return member, true
		}
		return stdlibMemberCallInExpr(expr.Right, imports, cursor)
	case *ast.IndexExpr:
		if member, ok := stdlibMemberCallInExpr(expr.Collection, imports, cursor); ok {
			return member, true
		}
		return stdlibMemberCallInExpr(expr.Index, imports, cursor)
	case *ast.SliceExpr:
		if member, ok := stdlibMemberCallInExpr(expr.Collection, imports, cursor); ok {
			return member, true
		}
		if expr.StartIndex != nil {
			if member, ok := stdlibMemberCallInExpr(expr.StartIndex, imports, cursor); ok {
				return member, true
			}
		}
		if expr.EndIndex != nil {
			return stdlibMemberCallInExpr(expr.EndIndex, imports, cursor)
		}
	}
	return stdlib.Member{}, false
}

func stdlibMemberCallInExprs(exprs []ast.Expression, imports map[string]struct{}, cursor token.Position) (stdlib.Member, bool) {
	for _, expr := range exprs {
		if member, ok := stdlibMemberCallInExpr(expr, imports, cursor); ok {
			return member, true
		}
	}
	return stdlib.Member{}, false
}

func (s *Server) resolveImportPath(uri string, pos Position) (LocationLink, bool, error) {
	ref, ok, err := s.importPathAt(uri, pos)
	if err != nil || !ok {
		return LocationLink{}, ok, err
	}
	if stdlib.IsPackage(ref.path) {
		return LocationLink{}, false, nil
	}

	importPath := normalizePath(filepath.Join(filepath.Dir(normalizePath(pathFromURI(uri))), ref.path))
	info, err := os.Stat(importPath)
	if err != nil || info.IsDir() {
		return LocationLink{}, false, nil
	}

	targetRange := Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: 0, Character: 0},
	}
	return LocationLink{
		OriginSelectionRange: importPathRange(ref.token),
		TargetURI:            uriFromPath(importPath),
		TargetRange:          targetRange,
		TargetSelectionRange: targetRange,
	}, true, nil
}

func (s *Server) importPathAt(uri string, pos Position) (importPathReference, bool, error) {
	text, err := s.documentText(uri)
	if err != nil {
		return importPathReference{}, false, err
	}

	path := normalizePath(pathFromURI(uri))
	cursor := tokenPositionFromLSP(pos)
	tokens := lexer.LexFile(path, text)
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type != token.Import || tokens[i+1].Type != token.String {
			continue
		}

		importToken := tokens[i+1]
		if !containsStringToken(cursor, importToken) {
			continue
		}
		if !stdlib.IsPackage(importToken.Lexeme) && (filepath.IsAbs(importToken.Lexeme) || filepath.Ext(importToken.Lexeme) != ".tx") {
			return importPathReference{}, false, nil
		}

		return importPathReference{path: importToken.Lexeme, token: importToken}, true, nil
	}

	return importPathReference{}, false, nil
}

func (s *Server) resolveImportPackageQualifier(uri string, pos Position) (LocationLink, bool, error) {
	text, err := s.documentText(uri)
	if err != nil {
		return LocationLink{}, false, err
	}

	word, ok := wordAt(text, pos)
	if !ok {
		return LocationLink{}, false, nil
	}

	originRange := rangeForWord(text, pos)
	if originRange == nil {
		return LocationLink{}, false, nil
	}

	path := normalizePath(pathFromURI(uri))
	graph := loadSourceGraph(path, text, s.documents)
	file := graph.files[path]
	if file == nil || !packageQualifierAt(file, word, tokenPositionFromLSP(pos)) {
		return LocationLink{}, false, nil
	}

	importToken, ok := directImportTokenForPackage(file, graph, word)
	if !ok {
		return LocationLink{}, false, nil
	}

	targetRange := importPathRange(importToken)
	return LocationLink{
		OriginSelectionRange: originRange,
		TargetURI:            file.uri,
		TargetRange:          *targetRange,
		TargetSelectionRange: *targetRange,
	}, true, nil
}

func directImportTokenForPackage(file *sourceFile, graph *sourceGraph, packageName string) (token.Token, bool) {
	for _, importDecl := range file.program.Imports {
		if stdlib.IsPackage(importDecl.Path) {
			if importDecl.Path == packageName {
				return importPathTokenForDecl(file, importDecl)
			}
			continue
		}
		if filepath.IsAbs(importDecl.Path) || filepath.Ext(importDecl.Path) != ".tx" {
			continue
		}

		importPath := normalizePath(filepath.Join(normalizePath(filepath.Dir(file.path)), importDecl.Path))
		importedFile := graph.files[importPath]
		if importedFile == nil || importedFile.program.PackageName != packageName {
			continue
		}

		if importToken, ok := importPathTokenForDecl(file, importDecl); ok {
			return importToken, true
		}
	}

	return token.Token{}, false
}

func importPathTokenForDecl(file *sourceFile, importDecl *ast.ImportDecl) (token.Token, bool) {
	tokens := lexer.LexFile(file.path, file.text)
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type != token.Import || tokens[i+1].Type != token.String {
			continue
		}
		if samePosition(tokens[i].Pos, importDecl.Pos) && tokens[i+1].Lexeme == importDecl.Path {
			return tokens[i+1], true
		}
	}

	return token.Token{}, false
}

func packageQualifierAt(file *sourceFile, packageName string, cursor token.Position) bool {
	for _, fn := range file.program.Functions {
		if packageQualifierInBlock(fn.Body, packageName, cursor) {
			return true
		}
	}
	return false
}

func packageQualifierInBlock(block ast.Block, packageName string, cursor token.Position) bool {
	for _, stmt := range block.Statements {
		switch stmt := stmt.(type) {
		case *ast.LetStmt:
			if packageQualifierInExpr(stmt.Value, packageName, cursor) {
				return true
			}
		case *ast.ReturnStmt:
			if packageQualifierInExpr(stmt.Value, packageName, cursor) {
				return true
			}
		case *ast.AssignStmt:
			if packageQualifierInExpr(stmt.Value, packageName, cursor) {
				return true
			}
		case *ast.IndexAssignStmt:
			if packageQualifierInExpr(stmt.Target, packageName, cursor) || packageQualifierInExpr(stmt.Value, packageName, cursor) {
				return true
			}
		case *ast.IfStmt:
			if packageQualifierInExpr(stmt.Condition, packageName, cursor) || packageQualifierInBlock(stmt.Then, packageName, cursor) {
				return true
			}
			if stmt.Else != nil && packageQualifierInBlock(*stmt.Else, packageName, cursor) {
				return true
			}
		case *ast.ForStmt:
			if stmt.Init != nil && packageQualifierInStmt(stmt.Init, packageName, cursor) {
				return true
			}
			if packageQualifierInExpr(stmt.Condition, packageName, cursor) || packageQualifierInBlock(stmt.Body, packageName, cursor) {
				return true
			}
			if stmt.Post != nil && packageQualifierInStmt(stmt.Post, packageName, cursor) {
				return true
			}
		case *ast.ExprStmt:
			if packageQualifierInExpr(stmt.Expr, packageName, cursor) {
				return true
			}
		}
	}

	return false
}

func packageQualifierInStmt(stmt ast.Statement, packageName string, cursor token.Position) bool {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		return packageQualifierInExpr(stmt.Value, packageName, cursor)
	case *ast.ReturnStmt:
		return packageQualifierInExpr(stmt.Value, packageName, cursor)
	case *ast.AssignStmt:
		return packageQualifierInExpr(stmt.Value, packageName, cursor)
	case *ast.IndexAssignStmt:
		return packageQualifierInExpr(stmt.Target, packageName, cursor) || packageQualifierInExpr(stmt.Value, packageName, cursor)
	case *ast.IfStmt:
		if packageQualifierInExpr(stmt.Condition, packageName, cursor) || packageQualifierInBlock(stmt.Then, packageName, cursor) {
			return true
		}
		return stmt.Else != nil && packageQualifierInBlock(*stmt.Else, packageName, cursor)
	case *ast.ForStmt:
		if stmt.Init != nil && packageQualifierInStmt(stmt.Init, packageName, cursor) {
			return true
		}
		if packageQualifierInExpr(stmt.Condition, packageName, cursor) || packageQualifierInBlock(stmt.Body, packageName, cursor) {
			return true
		}
		return stmt.Post != nil && packageQualifierInStmt(stmt.Post, packageName, cursor)
	case *ast.ExprStmt:
		return packageQualifierInExpr(stmt.Expr, packageName, cursor)
	default:
		return false
	}
}

func packageQualifierInExpr(expr ast.Expression, packageName string, cursor token.Position) bool {
	switch expr := expr.(type) {
	case nil:
		return false
	case *ast.CallExpr:
		if expr.Package == packageName && containsName(cursor, expr.Start, expr.Package) {
			return true
		}
		for _, arg := range expr.Args {
			if packageQualifierInExpr(arg, packageName, cursor) {
				return true
			}
		}
	case *ast.ArrayLiteral:
		return packageQualifierInExprs(expr.Elements, packageName, cursor)
	case *ast.ListLiteral:
		return packageQualifierInExprs(expr.Elements, packageName, cursor)
	case *ast.MakeExpr:
		return packageQualifierInExpr(expr.Len, packageName, cursor)
	case *ast.BinaryExpr:
		return packageQualifierInExpr(expr.Left, packageName, cursor) || packageQualifierInExpr(expr.Right, packageName, cursor)
	case *ast.IndexExpr:
		return packageQualifierInExpr(expr.Collection, packageName, cursor) || packageQualifierInExpr(expr.Index, packageName, cursor)
	case *ast.SliceExpr:
		if packageQualifierInExpr(expr.Collection, packageName, cursor) {
			return true
		}
		if expr.StartIndex != nil && packageQualifierInExpr(expr.StartIndex, packageName, cursor) {
			return true
		}
		return expr.EndIndex != nil && packageQualifierInExpr(expr.EndIndex, packageName, cursor)
	}

	return false
}

func packageQualifierInExprs(exprs []ast.Expression, packageName string, cursor token.Position) bool {
	for _, expr := range exprs {
		if packageQualifierInExpr(expr, packageName, cursor) {
			return true
		}
	}
	return false
}

func (s *Server) documentText(uri string) (string, error) {
	text, ok := s.documents[uri]
	if ok {
		return text, nil
	}
	return readURI(uri)
}

func (s *Server) resolveSymbol(uri string, pos Position) (symbolDefinition, bool, error) {
	def, _, ok, err := s.resolveSymbolWithGraph(uri, pos)
	return def, ok, err
}

func visibleLocalCompletionItems(file *sourceFile, cursor token.Position) []CompletionItem {
	for _, fn := range file.program.Functions {
		if !containsPosition(fn.Body.Start, fn.Body.End, cursor) {
			continue
		}

		scope := map[string]localDefinition{}
		for _, param := range fn.Params {
			scope[param.Name] = localDefinition{Name: param.Name, URI: file.uri, Pos: param.Pos}
		}
		locals := visibleLocalsInBlock(fn.Body, scope, cursor)
		items := make([]CompletionItem, 0, len(locals))
		for name := range locals {
			items = append(items, CompletionItem{Label: name, Kind: completionKindVariable, Detail: "local symbol"})
		}
		return items
	}

	return nil
}

func visibleLocalsInBlock(block ast.Block, scope map[string]localDefinition, cursor token.Position) map[string]localDefinition {
	locals := cloneLocalScope(scope)

	for _, stmt := range block.Statements {
		switch stmt := stmt.(type) {
		case *ast.LetStmt:
			if positionBefore(cursor, stmt.NamePos) {
				return locals
			}
			locals[stmt.Name] = localDefinition{Name: stmt.Name, URI: "", Pos: stmt.NamePos}
		case *ast.IfStmt:
			if containsPosition(stmt.Then.Start, stmt.Then.End, cursor) {
				return visibleLocalsInBlock(stmt.Then, locals, cursor)
			}
			if stmt.Else != nil && containsPosition(stmt.Else.Start, stmt.Else.End, cursor) {
				return visibleLocalsInBlock(*stmt.Else, locals, cursor)
			}
		case *ast.ForStmt:
			loopLocals := cloneLocalScope(locals)
			addForInitLocal(stmt.Init, loopLocals, "")
			if containsPosition(stmt.Body.Start, stmt.Body.End, cursor) {
				return visibleLocalsInBlock(stmt.Body, loopLocals, cursor)
			}
		}
	}

	return locals
}

func addForInitLocal(stmt ast.Statement, locals map[string]localDefinition, uri string) {
	letStmt, ok := stmt.(*ast.LetStmt)
	if !ok {
		return
	}
	locals[letStmt.Name] = localDefinition{Name: letStmt.Name, URI: uri, Pos: letStmt.NamePos}
}

func completionLabels(items []CompletionItem) map[string]struct{} {
	seen := map[string]struct{}{}
	for _, item := range items {
		seen[item.Label] = struct{}{}
	}
	return seen
}

func appendCompletionItem(items []CompletionItem, seen map[string]struct{}, item CompletionItem) []CompletionItem {
	if _, ok := seen[item.Label]; ok {
		return items
	}
	seen[item.Label] = struct{}{}
	return append(items, item)
}

func completionListFromItems(items []CompletionItem) CompletionList {
	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

func importStringCompletionAt(text string, pos Position) (importStringCompletionContext, bool) {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return importStringCompletionContext{}, false
	}

	line := []rune(lines[pos.Line])
	if pos.Character < 0 || pos.Character > len(line) {
		return importStringCompletionContext{}, false
	}

	before := string(line[:pos.Character])
	leading := len([]rune(before)) - len([]rune(strings.TrimLeft(before, " \t")))
	rest := string(line[leading:pos.Character])
	if !strings.HasPrefix(rest, "import") {
		return importStringCompletionContext{}, false
	}
	afterImport := rest[len("import"):]
	if afterImport == "" || isWordRune([]rune(afterImport)[0]) {
		return importStringCompletionContext{}, false
	}
	afterImport = strings.TrimLeft(afterImport, " \t")
	if !strings.HasPrefix(afterImport, "\"") {
		return importStringCompletionContext{}, false
	}

	prefix := afterImport[1:]
	if strings.Contains(prefix, "\"") {
		return importStringCompletionContext{}, false
	}

	start := pos.Character - len([]rune(prefix))
	edit := TextEdit{
		Range: Range{
			Start: Position{Line: pos.Line, Character: start},
			End:   pos,
		},
	}
	return importStringCompletionContext{prefix: prefix, edit: edit}, true
}

func expressionCompletionAt(text string, pos Position) bool {
	offset := offsetFromPosition(text, pos)
	if offset < 0 {
		return false
	}

	depth := 0
	inString := false
	escaped := false
	for i := 0; i < offset && i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		if ch == '/' && i+1 < offset && i+1 < len(text) && text[i+1] == '/' {
			for i < offset && i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth > 0
}

func offsetFromPosition(text string, pos Position) int {
	line := 0
	character := 0
	for offset, r := range text {
		if line == pos.Line && character == pos.Character {
			return offset
		}
		if r == '\n' {
			line++
			character = 0
			continue
		}
		character++
	}
	if line == pos.Line && character == pos.Character {
		return len(text)
	}
	return -1
}

func (s *Server) resolveSymbolWithGraph(uri string, pos Position) (symbolDefinition, *sourceGraph, bool, error) {
	text, ok := s.documents[uri]
	if !ok {
		var err error
		text, err = readURI(uri)
		if err != nil {
			return symbolDefinition{}, nil, false, err
		}
	}

	word, ok := wordAt(text, pos)
	if !ok {
		return symbolDefinition{}, nil, false, nil
	}

	path := normalizePath(pathFromURI(uri))
	graph := loadSourceGraph(path, text, s.documents)
	file := graph.files[path]
	if file == nil {
		return symbolDefinition{}, graph, false, nil
	}

	isCall := isCallAt(text, pos)
	if isCall {
		if def, ok := findFunctionDefinition(graph, path, word); ok {
			return def, graph, true, nil
		}
	}

	cursor := tokenPositionFromLSP(pos)
	if def, ok := findLocalDefinition(file, word, cursor); ok {
		return symbolDefinition{
			Kind: localSymbol,
			Name: def.Name,
			URI:  def.URI,
			Path: path,
			Pos:  def.Pos,
		}, graph, true, nil
	}

	if def, ok := findFunctionDefinition(graph, path, word); ok {
		return def, graph, true, nil
	}

	return symbolDefinition{}, graph, false, nil
}

func loadSourceGraph(activePath string, activeText string, documents map[string]string) *sourceGraph {
	openDocuments := map[string]string{}
	for uri, text := range documents {
		openDocuments[normalizePath(pathFromURI(uri))] = text
	}
	openDocuments[activePath] = activeText

	graph := &sourceGraph{files: map[string]*sourceFile{}}
	var load func(path string, includeSiblings bool)
	load = func(path string, includeSiblings bool) {
		path = normalizePath(path)
		if _, exists := graph.files[path]; exists {
			return
		}

		text, ok := sourceTextForPath(path, openDocuments)
		if !ok {
			return
		}

		program, err := parser.ParseFile(path, text)
		if err != nil {
			return
		}

		file := &sourceFile{
			path:    path,
			uri:     uriFromPath(path),
			text:    text,
			program: program,
		}
		graph.files[path] = file
		graph.order = append(graph.order, file)

		for _, importDecl := range program.Imports {
			if filepath.IsAbs(importDecl.Path) || filepath.Ext(importDecl.Path) != ".tx" {
				continue
			}
			load(filepath.Join(normalizePath(filepath.Dir(path)), importDecl.Path), true)
		}

		if includeSiblings || !hasMain(program) {
			for _, siblingPath := range samePackageSiblingPaths(file, openDocuments, graph.files) {
				load(siblingPath, true)
			}
		}
	}

	load(activePath, false)
	return graph
}

func directImportedPackages(activePath string, activeText string, documents map[string]string) []importedPackage {
	openDocuments := map[string]string{}
	for uri, text := range documents {
		openDocuments[normalizePath(pathFromURI(uri))] = text
	}

	imports := []importedPackage{}
	seenPaths := map[string]struct{}{}
	packageIndex := map[string]int{}
	for _, importDecl := range importDeclsFromText(activePath, activeText) {
		if pkg, ok := stdlib.LookupPackage(importDecl.Path); ok {
			if _, seen := packageIndex[pkg.Name]; seen {
				continue
			}
			packageIndex[pkg.Name] = len(imports)
			imports = append(imports, importedPackage{name: pkg.Name, stdlib: true})
			continue
		}
		if filepath.IsAbs(importDecl.Path) || filepath.Ext(importDecl.Path) != ".tx" {
			continue
		}

		path := normalizePath(filepath.Join(normalizePath(filepath.Dir(activePath)), importDecl.Path))
		if _, ok := seenPaths[path]; ok {
			continue
		}

		files := importedPackageFiles(path, openDocuments)
		if len(files) == 0 {
			continue
		}
		for _, file := range files {
			seenPaths[file.path] = struct{}{}
		}

		packageName := files[0].program.PackageName
		if index, ok := packageIndex[packageName]; ok {
			imports[index].files = append(imports[index].files, files...)
			continue
		}
		packageIndex[packageName] = len(imports)
		imports = append(imports, importedPackage{name: packageName, files: files})
	}

	return imports
}

func sourceTextForPath(path string, openDocuments map[string]string) (string, bool) {
	path = normalizePath(path)
	if text, ok := openDocuments[path]; ok {
		return text, true
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func importedPackageFiles(path string, openDocuments map[string]string) []*sourceFile {
	path = normalizePath(path)
	text, ok := sourceTextForPath(path, openDocuments)
	if !ok {
		return nil
	}
	program, err := parser.ParseFile(path, text)
	if err != nil {
		return nil
	}

	file := &sourceFile{
		path:    path,
		uri:     uriFromPath(path),
		text:    text,
		program: program,
	}
	files := []*sourceFile{file}
	loaded := map[string]*sourceFile{path: file}
	for _, siblingPath := range samePackageSiblingPaths(file, openDocuments, loaded) {
		siblingText, ok := sourceTextForPath(siblingPath, openDocuments)
		if !ok {
			continue
		}
		siblingProgram, err := parser.ParseFile(siblingPath, siblingText)
		if err != nil {
			continue
		}
		sibling := &sourceFile{
			path:    siblingPath,
			uri:     uriFromPath(siblingPath),
			text:    siblingText,
			program: siblingProgram,
		}
		loaded[siblingPath] = sibling
		files = append(files, sibling)
	}
	return files
}

func samePackageSiblingPaths(file *sourceFile, openDocuments map[string]string, loaded map[string]*sourceFile) []string {
	dir := filepath.Dir(file.path)
	candidates := map[string]struct{}{}

	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".tx" {
				continue
			}
			path := normalizePath(filepath.Join(dir, entry.Name()))
			candidates[path] = struct{}{}
		}
	}

	for path := range openDocuments {
		if filepath.Dir(path) == dir && filepath.Ext(path) == ".tx" {
			candidates[path] = struct{}{}
		}
	}

	delete(candidates, file.path)

	paths := make([]string, 0, len(candidates))
	for path := range candidates {
		if _, exists := loaded[path]; exists {
			continue
		}
		text, ok := sourceTextForPath(path, openDocuments)
		if !ok {
			continue
		}
		if packageNameFromText(path, text) != file.program.PackageName {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func packageNameFromText(path string, text string) string {
	tokens := lexer.LexFile(path, text)
	if len(tokens) < 2 || tokens[0].Type != token.Package || tokens[1].Type != token.Ident {
		return ""
	}
	return tokens[1].Lexeme
}

func sameFileFunctionCompletionItems(file *sourceFile) []CompletionItem {
	items := []CompletionItem{}
	for _, fn := range file.program.Functions {
		items = append(items, CompletionItem{Label: fn.Name, Kind: completionKindFunction, Detail: "function"})
	}
	return items
}

func samePackageFunctionCompletionItems(graph *sourceGraph, file *sourceFile) []CompletionItem {
	items := []CompletionItem{}
	for _, packageFile := range samePackageFiles(graph, file) {
		if packageFile.path == file.path {
			continue
		}
		for _, fn := range packageFile.program.Functions {
			items = append(items, CompletionItem{Label: fn.Name, Kind: completionKindFunction, Detail: "function from package " + file.program.PackageName})
		}
	}
	return items
}

func importedPackageCompletionItems(imports []importedPackage) []CompletionItem {
	items := make([]CompletionItem, 0, len(imports))
	for _, pkg := range imports {
		items = append(items, CompletionItem{Label: pkg.name, Kind: completionKindModule, Detail: "imported package"})
	}
	return items
}

func standardPackageImportCompletionItems(context importStringCompletionContext) []CompletionItem {
	if strings.Contains(context.prefix, "/") || strings.Contains(context.prefix, "\\") || strings.HasPrefix(context.prefix, ".") {
		return nil
	}

	packages := stdlib.Packages()
	items := make([]CompletionItem, 0, len(packages))
	for _, pkg := range packages {
		edit := context.edit
		edit.NewText = pkg.Name
		items = append(items, CompletionItem{
			Label:      pkg.Name,
			Kind:       completionKindModule,
			Detail:     "standard package",
			InsertText: pkg.Name,
			TextEdit:   &edit,
		})
	}
	return items
}

func importPathCompletionItems(activePath string, context importStringCompletionContext) []CompletionItem {
	items := standardPackageImportCompletionItems(context)
	items = append(items, sourceImportPathCompletionItems(activePath, context)...)
	return items
}

func sourceImportPathCompletionItems(activePath string, context importStringCompletionContext) []CompletionItem {
	activeDir := normalizePath(filepath.Dir(activePath))
	dirPrefix, basePrefix := splitImportPrefix(context.prefix)
	targetDir := normalizePath(filepath.Join(activeDir, filepath.FromSlash(dirPrefix)))

	if root, ok := findRepoRoot(activeDir); ok && !pathWithinRoot(targetDir, root) {
		return nil
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil
	}

	items := []CompletionItem{}
	for _, entry := range entries {
		name := entry.Name()
		if shouldSkipCompletionDir(name) {
			continue
		}
		if !strings.HasPrefix(name, basePrefix) {
			continue
		}

		entryPath := filepath.Join(targetDir, name)
		if entry.IsDir() {
			if !directoryCanLeadToTx(entryPath) {
				continue
			}
			importPath := joinImportPath(dirPrefix, name) + "/"
			edit := context.edit
			edit.NewText = importPath
			items = append(items, CompletionItem{
				Label:      name + "/",
				Kind:       completionKindModule,
				Detail:     "directory",
				InsertText: name + "/",
				TextEdit:   &edit,
			})
			continue
		}

		if filepath.Ext(name) != ".tx" || normalizePath(entryPath) == activePath {
			continue
		}
		importPath := joinImportPath(dirPrefix, name)
		edit := context.edit
		edit.NewText = importPath
		items = append(items, CompletionItem{
			Label:      name,
			Kind:       completionKindModule,
			Detail:     "source module",
			InsertText: name,
			TextEdit:   &edit,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

func splitImportPrefix(prefix string) (string, string) {
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	index := strings.LastIndex(prefix, "/")
	if index < 0 {
		return "", prefix
	}
	return prefix[:index+1], prefix[index+1:]
}

func joinImportPath(dirPrefix string, name string) string {
	dirPrefix = strings.ReplaceAll(dirPrefix, "\\", "/")
	if dirPrefix == "" {
		return name
	}
	return dirPrefix + name
}

func directoryCanLeadToTx(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if entry.IsDir() {
			if path != dir && shouldSkipCompletionDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) == ".tx" {
			found = true
		}
		return nil
	})
	return found
}

func shouldSkipCompletionDir(name string) bool {
	switch name {
	case ".git", "node_modules", "tmp":
		return true
	default:
		return false
	}
}

func qualifiedImportedFunctionCompletionItems(imports []importedPackage) []CompletionItem {
	items := []CompletionItem{}
	for _, pkg := range imports {
		if pkg.stdlib {
			for _, member := range stdlibMembers(pkg.name) {
				label := pkg.name + "." + member.Name
				items = append(items, CompletionItem{
					Label:      label,
					Kind:       completionKindFunction,
					Detail:     member.Detail,
					InsertText: label,
					FilterText: label + " " + member.Name,
				})
			}
			continue
		}
		for _, file := range pkg.files {
			for _, fn := range file.program.Functions {
				if !fn.Public {
					continue
				}
				label := pkg.name + "." + fn.Name
				items = append(items, CompletionItem{
					Label:      label,
					Kind:       completionKindFunction,
					Detail:     "imported function",
					InsertText: label,
					FilterText: label + " " + fn.Name,
				})
			}
		}
	}
	return items
}

func standardAutoImportFunctionCompletionItems(text string, imports []importedPackage) []CompletionItem {
	items := []CompletionItem{}
	for _, pkg := range stdlib.Packages() {
		if importedPackageName(imports, pkg.Name) {
			continue
		}
		for _, member := range pkg.Members {
			label := pkg.Name + "." + member.Name
			items = append(items, CompletionItem{
				Label:               label,
				Kind:                completionKindFunction,
				Detail:              member.Detail,
				InsertText:          label,
				FilterText:          label + " " + member.Name,
				AdditionalTextEdits: []TextEdit{importEdit(text, pkg.Name)},
			})
		}
	}
	return items
}

func standardPackageMemberCompletionItems(pkg stdlib.Package, edit TextEdit) []CompletionItem {
	items := make([]CompletionItem, 0, len(pkg.Members))
	for _, member := range pkg.Members {
		items = append(items, CompletionItem{
			Label:               member.Name,
			Kind:                completionKindFunction,
			Detail:              member.Signature(),
			InsertText:          member.Name,
			AdditionalTextEdits: []TextEdit{edit},
		})
	}
	return items
}

func packageMemberCompletionItems(pkg importedPackage) []CompletionItem {
	items := []CompletionItem{}
	if pkg.stdlib {
		for _, member := range stdlibMembers(pkg.name) {
			items = append(items, CompletionItem{
				Label:      member.Name,
				Kind:       completionKindFunction,
				Detail:     member.Signature(),
				InsertText: member.Name,
			})
		}
		return items
	}
	for _, file := range pkg.files {
		for _, fn := range file.program.Functions {
			if !fn.Public {
				continue
			}
			items = append(items, CompletionItem{
				Label:      fn.Name,
				Kind:       completionKindFunction,
				Detail:     "function from " + pkg.name,
				InsertText: fn.Name,
			})
		}
	}
	return items
}

func sourceAutoImportFunctionCompletionItems(activePath string, activeText string, documents map[string]string, imports []importedPackage) []CompletionItem {
	candidates := sourceAutoImportCandidates(activePath, activeText, documents, imports, "")
	items := make([]CompletionItem, 0, len(candidates))
	for _, candidate := range candidates {
		label := candidate.packageName + "." + candidate.functionName
		items = append(items, CompletionItem{
			Label:               label,
			Kind:                completionKindFunction,
			Detail:              "import \"" + candidate.importPath + "\"",
			InsertText:          label,
			FilterText:          label + " " + candidate.functionName,
			AdditionalTextEdits: []TextEdit{importEdit(activeText, candidate.importPath)},
		})
	}
	return items
}

func sourceAutoImportMemberCompletionItems(activePath string, activeText string, documents map[string]string, imports []importedPackage, packageName string) []CompletionItem {
	candidates := sourceAutoImportCandidates(activePath, activeText, documents, imports, packageName)
	items := make([]CompletionItem, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, CompletionItem{
			Label:               candidate.functionName,
			Kind:                completionKindFunction,
			Detail:              functionSignature(candidate.fn) + " from " + candidate.importPath,
			InsertText:          candidate.functionName,
			AdditionalTextEdits: []TextEdit{importEdit(activeText, candidate.importPath)},
		})
	}
	return items
}

func sourceAutoImportCandidates(activePath string, activeText string, documents map[string]string, imports []importedPackage, packageName string) []sourceModuleCandidate {
	activeDir := normalizePath(filepath.Dir(activePath))
	root, ok := findRepoRoot(activeDir)
	if !ok {
		root = activeDir
	}

	openDocuments := map[string]string{}
	for uri, text := range documents {
		openDocuments[normalizePath(pathFromURI(uri))] = text
	}
	openDocuments[activePath] = activeText

	importedPaths := importedSourcePaths(imports)
	importedFunctions := importedFunctionNamesByPackage(imports)
	importedStdlibPackages := importedStdlibPackageNames(imports)

	seenPaths := map[string]struct{}{}
	candidates := []sourceModuleCandidate{}
	visit := func(path string) {
		path = normalizePath(path)
		if _, seen := seenPaths[path]; seen {
			return
		}
		seenPaths[path] = struct{}{}
		if path == activePath {
			return
		}
		if _, imported := importedPaths[path]; imported {
			return
		}

		text, ok := openDocuments[path]
		if !ok {
			data, err := os.ReadFile(path)
			if err != nil {
				return
			}
			text = string(data)
		}

		program, err := parser.ParseFile(path, text)
		if err != nil {
			return
		}
		if packageName != "" && program.PackageName != packageName {
			return
		}
		if stdlib.IsPackage(program.PackageName) {
			return
		}
		if _, conflict := importedStdlibPackages[program.PackageName]; conflict {
			return
		}

		importPath, ok := relativeImportPath(activeDir, path)
		if !ok {
			return
		}
		for _, fn := range program.Functions {
			if !fn.Public {
				continue
			}
			if names := importedFunctions[program.PackageName]; names != nil {
				if _, duplicate := names[fn.Name]; duplicate {
					continue
				}
			}
			candidates = append(candidates, sourceModuleCandidate{
				packageName:  program.PackageName,
				functionName: fn.Name,
				importPath:   importPath,
				path:         path,
				fn:           fn,
			})
		}
	}

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && shouldSkipCompletionDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) == ".tx" {
			visit(path)
		}
		return nil
	})

	for path := range openDocuments {
		if filepath.Ext(path) != ".tx" || !pathWithinRoot(path, root) {
			continue
		}
		visit(path)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i].packageName + "." + candidates[i].functionName + " " + candidates[i].importPath
		right := candidates[j].packageName + "." + candidates[j].functionName + " " + candidates[j].importPath
		return left < right
	})
	return candidates
}

func stdlibMembers(packageName string) []stdlib.Member {
	pkg, ok := stdlib.LookupPackage(packageName)
	if !ok {
		return nil
	}
	return pkg.Members
}

func importedPackageName(imports []importedPackage, name string) bool {
	for _, pkg := range imports {
		if pkg.name == name {
			return true
		}
	}
	return false
}

func importedSourcePaths(imports []importedPackage) map[string]struct{} {
	paths := map[string]struct{}{}
	for _, pkg := range imports {
		if pkg.stdlib {
			continue
		}
		for _, file := range pkg.files {
			paths[file.path] = struct{}{}
		}
	}
	return paths
}

func importedStdlibPackageNames(imports []importedPackage) map[string]struct{} {
	names := map[string]struct{}{}
	for _, pkg := range imports {
		if pkg.stdlib {
			names[pkg.name] = struct{}{}
		}
	}
	return names
}

func importedFunctionNamesByPackage(imports []importedPackage) map[string]map[string]struct{} {
	out := map[string]map[string]struct{}{}
	for _, pkg := range imports {
		if pkg.stdlib {
			continue
		}
		names := out[pkg.name]
		if names == nil {
			names = map[string]struct{}{}
			out[pkg.name] = names
		}
		for _, file := range pkg.files {
			for _, fn := range file.program.Functions {
				if fn.Public {
					names[fn.Name] = struct{}{}
				}
			}
		}
	}
	return out
}

func importEdit(text string, importPath string) TextEdit {
	lines := strings.Split(text, "\n")
	insertLine := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") && insertLine == 0 {
			insertLine = i + 1
		}
		if strings.HasPrefix(trimmed, "import ") {
			insertLine = i + 1
		}
	}
	pos := Position{Line: insertLine, Character: 0}
	return TextEdit{
		Range:   Range{Start: pos, End: pos},
		NewText: "import \"" + importPath + "\"\n",
	}
}

func declarationCompletionItems() []CompletionItem {
	items := []CompletionItem{}
	for _, item := range baseCompletionItems() {
		switch item.Label {
		case "package", "import", "pub", "func", "int", "float", "string", "bool", "list":
			items = append(items, item)
		}
	}
	return items
}

func findRepoRoot(start string) (string, bool) {
	dir := normalizePath(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func pathWithinRoot(path string, root string) bool {
	path = normalizePath(path)
	root = normalizePath(root)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func relativeImportPath(activeDir string, targetPath string) (string, bool) {
	activeDir = normalizePath(activeDir)
	targetPath = normalizePath(targetPath)
	rel, err := filepath.Rel(activeDir, targetPath)
	if err != nil || rel == "." {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func completionContextAt(text string, pos Position) completionContext {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return completionContext{}
	}

	line := []rune(lines[pos.Line])
	if pos.Character < 0 || pos.Character > len(line) {
		return completionContext{}
	}

	prefixStart := pos.Character
	for prefixStart > 0 && isWordRune(line[prefixStart-1]) {
		prefixStart--
	}
	if prefixStart == 0 || line[prefixStart-1] != '.' {
		return completionContext{}
	}

	qualifierEnd := prefixStart - 1
	qualifierStart := qualifierEnd
	for qualifierStart > 0 && isWordRune(line[qualifierStart-1]) {
		qualifierStart--
	}
	if qualifierStart == qualifierEnd {
		return completionContext{}
	}

	return completionContext{
		member:      true,
		packageName: string(line[qualifierStart:qualifierEnd]),
	}
}

func importDeclsFromText(activePath string, text string) []*ast.ImportDecl {
	tokens := lexer.LexFile(activePath, text)
	imports := []*ast.ImportDecl{}
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type != token.Import || tokens[i+1].Type != token.String {
			continue
		}
		imports = append(imports, &ast.ImportDecl{Pos: tokens[i].Pos, Path: tokens[i+1].Lexeme})
	}
	return imports
}

func findLocalDefinition(file *sourceFile, word string, cursor token.Position) (localDefinition, bool) {
	for _, fn := range file.program.Functions {
		if containsName(cursor, fn.NamePos, fn.Name) && word == fn.Name {
			return localDefinition{}, false
		}
		for _, param := range fn.Params {
			if containsName(cursor, param.Pos, param.Name) && word == param.Name {
				return localDefinition{Name: param.Name, URI: file.uri, Pos: param.Pos}, true
			}
		}
		if !containsPosition(fn.Body.Start, fn.Body.End, cursor) {
			continue
		}

		scope := map[string]localDefinition{}
		for _, param := range fn.Params {
			scope[param.Name] = localDefinition{Name: param.Name, URI: file.uri, Pos: param.Pos}
		}
		if def, ok := findLocalDefinitionInBlock(file.uri, fn.Body, scope, word, cursor); ok {
			return def, true
		}
	}

	return localDefinition{}, false
}

func findLocalDefinitionInBlock(uri string, block ast.Block, scope map[string]localDefinition, word string, cursor token.Position) (localDefinition, bool) {
	locals := cloneLocalScope(scope)

	for _, stmt := range block.Statements {
		switch stmt := stmt.(type) {
		case *ast.LetStmt:
			if containsName(cursor, stmt.NamePos, stmt.Name) && word == stmt.Name {
				return localDefinition{Name: stmt.Name, URI: uri, Pos: stmt.NamePos}, true
			}
			if positionBefore(cursor, stmt.NamePos) {
				return locals[word], locals[word].Name != ""
			}
			locals[stmt.Name] = localDefinition{Name: stmt.Name, URI: uri, Pos: stmt.NamePos}
		case *ast.IfStmt:
			if containsPosition(stmt.Then.Start, stmt.Then.End, cursor) {
				return findLocalDefinitionInBlock(uri, stmt.Then, locals, word, cursor)
			}
			if stmt.Else != nil && containsPosition(stmt.Else.Start, stmt.Else.End, cursor) {
				return findLocalDefinitionInBlock(uri, *stmt.Else, locals, word, cursor)
			}
		case *ast.ForStmt:
			loopLocals := cloneLocalScope(locals)
			if letStmt, ok := stmt.Init.(*ast.LetStmt); ok {
				if containsName(cursor, letStmt.NamePos, letStmt.Name) && word == letStmt.Name {
					return localDefinition{Name: letStmt.Name, URI: uri, Pos: letStmt.NamePos}, true
				}
				if positionBefore(cursor, letStmt.NamePos) {
					return locals[word], locals[word].Name != ""
				}
				loopLocals[letStmt.Name] = localDefinition{Name: letStmt.Name, URI: uri, Pos: letStmt.NamePos}
			}
			if containsPosition(stmt.Body.Start, stmt.Body.End, cursor) {
				return findLocalDefinitionInBlock(uri, stmt.Body, loopLocals, word, cursor)
			}
		}
	}

	def, ok := locals[word]
	return def, ok
}

func cloneLocalScope(scope map[string]localDefinition) map[string]localDefinition {
	clone := make(map[string]localDefinition, len(scope))
	for name, def := range scope {
		clone[name] = def
	}
	return clone
}

func findFunctionDefinition(graph *sourceGraph, activePath string, word string) (symbolDefinition, bool) {
	activeFile := graph.files[activePath]
	if activeFile != nil {
		if def, ok := findFunctionInFile(activeFile, word, true); ok {
			return def, true
		}
		for _, file := range samePackageFiles(graph, activeFile) {
			if file.path == activePath {
				continue
			}
			if def, ok := findFunctionInFile(file, word, true); ok {
				return def, true
			}
		}
	}

	for _, file := range graph.order {
		if file.path == activePath {
			continue
		}
		if activeFile != nil && samePackage(file, activeFile) {
			continue
		}
		if def, ok := findFunctionInFile(file, word, false); ok {
			return def, true
		}
	}

	return symbolDefinition{}, false
}

func findFunctionInFile(file *sourceFile, word string, includePrivate bool) (symbolDefinition, bool) {
	for _, fn := range file.program.Functions {
		if fn.Name != word {
			continue
		}
		if !includePrivate && !fn.Public {
			continue
		}
		return symbolDefinition{
			Kind:   functionSymbol,
			Name:   fn.Name,
			URI:    file.uri,
			Path:   file.path,
			Pos:    fn.NamePos,
			Public: fn.Public,
		}, true
	}
	return symbolDefinition{}, false
}

func findFunctionDecl(graph *sourceGraph, def symbolDefinition) (*ast.FuncDecl, bool) {
	file := graph.files[def.Path]
	if file == nil {
		return nil, false
	}
	for _, fn := range file.program.Functions {
		if fn.Name == def.Name && samePosition(fn.NamePos, def.Pos) {
			return fn, true
		}
	}
	return nil, false
}

func functionSignature(fn *ast.FuncDecl) string {
	prefix := "func"
	if fn.Public {
		prefix = "pub func"
	}

	params := make([]string, 0, len(fn.Params))
	for _, param := range fn.Params {
		params = append(params, param.Name+" "+param.Type.String())
	}
	return prefix + " " + fn.Name + "(" + strings.Join(params, ", ") + ") " + fn.ReturnType.String()
}

func collectFunctionReferences(graph *sourceGraph, def symbolDefinition, includeDeclaration bool) []Location {
	locations := []Location{}
	if includeDeclaration {
		locations = append(locations, locationFor(def.URI, def.Pos, def.Name))
	}

	defFile := graph.files[def.Path]
	for _, file := range graph.order {
		if !def.Public && (defFile == nil || !samePackage(file, defFile)) {
			continue
		}
		if def.Public && file.path != def.Path && hasPrivateFunction(file, def.Name) {
			continue
		}
		for _, fn := range file.program.Functions {
			collectFunctionReferencesInBlock(file.uri, fn.Body, def.Name, &locations)
		}
	}

	return locations
}

func samePackageFiles(graph *sourceGraph, file *sourceFile) []*sourceFile {
	files := []*sourceFile{}
	for _, candidate := range graph.order {
		if samePackage(candidate, file) {
			files = append(files, candidate)
		}
	}
	return files
}

func samePackage(left *sourceFile, right *sourceFile) bool {
	return sourcePackageKey(left) == sourcePackageKey(right)
}

func sourcePackageKey(file *sourceFile) packageKey {
	return packageKey{
		dir:  filepath.Dir(file.path),
		name: file.program.PackageName,
	}
}

func hasPrivateFunction(file *sourceFile, name string) bool {
	for _, fn := range file.program.Functions {
		if fn.Name == name && !fn.Public {
			return true
		}
	}
	return false
}

func collectFunctionReferencesInBlock(uri string, block ast.Block, name string, locations *[]Location) {
	for _, stmt := range block.Statements {
		collectFunctionReferencesInStmt(uri, stmt, name, locations)
	}
}

func collectFunctionReferencesInStmt(uri string, stmt ast.Statement, name string, locations *[]Location) {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		collectFunctionReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.ReturnStmt:
		collectFunctionReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.AssignStmt:
		collectFunctionReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.IndexAssignStmt:
		collectFunctionReferencesInExpr(uri, stmt.Target, name, locations)
		collectFunctionReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.IfStmt:
		collectFunctionReferencesInExpr(uri, stmt.Condition, name, locations)
		collectFunctionReferencesInBlock(uri, stmt.Then, name, locations)
		if stmt.Else != nil {
			collectFunctionReferencesInBlock(uri, *stmt.Else, name, locations)
		}
	case *ast.ForStmt:
		if stmt.Init != nil {
			collectFunctionReferencesInStmt(uri, stmt.Init, name, locations)
		}
		collectFunctionReferencesInExpr(uri, stmt.Condition, name, locations)
		collectFunctionReferencesInBlock(uri, stmt.Body, name, locations)
		if stmt.Post != nil {
			collectFunctionReferencesInStmt(uri, stmt.Post, name, locations)
		}
	case *ast.ExprStmt:
		collectFunctionReferencesInExpr(uri, stmt.Expr, name, locations)
	}
}

func collectFunctionReferencesInExpr(uri string, expr ast.Expression, name string, locations *[]Location) {
	switch expr := expr.(type) {
	case *ast.ArrayLiteral:
		for _, elem := range expr.Elements {
			collectFunctionReferencesInExpr(uri, elem, name, locations)
		}
	case *ast.ListLiteral:
		for _, elem := range expr.Elements {
			collectFunctionReferencesInExpr(uri, elem, name, locations)
		}
	case *ast.MakeExpr:
		collectFunctionReferencesInExpr(uri, expr.Len, name, locations)
	case *ast.CallExpr:
		if expr.Callee == name {
			*locations = append(*locations, locationFor(uri, callCalleePosition(expr), expr.Callee))
		}
		for _, arg := range expr.Args {
			collectFunctionReferencesInExpr(uri, arg, name, locations)
		}
	case *ast.BinaryExpr:
		collectFunctionReferencesInExpr(uri, expr.Left, name, locations)
		collectFunctionReferencesInExpr(uri, expr.Right, name, locations)
	case *ast.IndexExpr:
		collectFunctionReferencesInExpr(uri, expr.Collection, name, locations)
		collectFunctionReferencesInExpr(uri, expr.Index, name, locations)
	case *ast.SliceExpr:
		collectFunctionReferencesInExpr(uri, expr.Collection, name, locations)
		if expr.StartIndex != nil {
			collectFunctionReferencesInExpr(uri, expr.StartIndex, name, locations)
		}
		if expr.EndIndex != nil {
			collectFunctionReferencesInExpr(uri, expr.EndIndex, name, locations)
		}
	}
}

func callCalleePosition(expr *ast.CallExpr) token.Position {
	pos := expr.Start
	if expr.Package == "" {
		return pos
	}

	offset := len(expr.Package) + 1
	pos.Offset += offset
	pos.Column += offset
	return pos
}

func collectLocalReferences(graph *sourceGraph, def symbolDefinition, includeDeclaration bool) []Location {
	file := graph.files[def.Path]
	if file == nil {
		return nil
	}

	locations := []Location{}
	for _, fn := range file.program.Functions {
		if !containsPosition(fn.Body.Start, fn.Body.End, def.Pos) {
			for _, param := range fn.Params {
				if samePosition(param.Pos, def.Pos) {
					collectLocalReferencesInFunc(file.uri, fn, def, includeDeclaration, &locations)
					return locations
				}
			}
			continue
		}
		collectLocalReferencesInFunc(file.uri, fn, def, includeDeclaration, &locations)
		return locations
	}

	return locations
}

func collectLocalReferencesInFunc(uri string, fn *ast.FuncDecl, def symbolDefinition, includeDeclaration bool, locations *[]Location) {
	if includeDeclaration {
		*locations = append(*locations, locationFor(def.URI, def.Pos, def.Name))
	}
	collectLocalReferencesInBlock(uri, fn.Body, def.Name, locations)
}

func collectLocalReferencesInBlock(uri string, block ast.Block, name string, locations *[]Location) {
	for _, stmt := range block.Statements {
		switch stmt := stmt.(type) {
		case *ast.LetStmt:
			collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
		case *ast.ReturnStmt:
			collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
		case *ast.AssignStmt:
			if stmt.Name == name {
				*locations = append(*locations, locationFor(uri, stmt.Start, stmt.Name))
			}
			collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
		case *ast.IndexAssignStmt:
			collectLocalReferencesInExpr(uri, stmt.Target, name, locations)
			collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
		case *ast.IfStmt:
			collectLocalReferencesInExpr(uri, stmt.Condition, name, locations)
			collectLocalReferencesInBlock(uri, stmt.Then, name, locations)
			if stmt.Else != nil {
				collectLocalReferencesInBlock(uri, *stmt.Else, name, locations)
			}
		case *ast.ForStmt:
			if stmt.Init != nil {
				collectLocalReferencesInStmt(uri, stmt.Init, name, locations)
			}
			collectLocalReferencesInExpr(uri, stmt.Condition, name, locations)
			collectLocalReferencesInBlock(uri, stmt.Body, name, locations)
			if stmt.Post != nil {
				collectLocalReferencesInStmt(uri, stmt.Post, name, locations)
			}
		case *ast.ExprStmt:
			collectLocalReferencesInExpr(uri, stmt.Expr, name, locations)
		}
	}
}

func collectLocalReferencesInStmt(uri string, stmt ast.Statement, name string, locations *[]Location) {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.ReturnStmt:
		collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.AssignStmt:
		if stmt.Name == name {
			*locations = append(*locations, locationFor(uri, stmt.Start, stmt.Name))
		}
		collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.IndexAssignStmt:
		collectLocalReferencesInExpr(uri, stmt.Target, name, locations)
		collectLocalReferencesInExpr(uri, stmt.Value, name, locations)
	case *ast.IfStmt:
		collectLocalReferencesInExpr(uri, stmt.Condition, name, locations)
		collectLocalReferencesInBlock(uri, stmt.Then, name, locations)
		if stmt.Else != nil {
			collectLocalReferencesInBlock(uri, *stmt.Else, name, locations)
		}
	case *ast.ForStmt:
		if stmt.Init != nil {
			collectLocalReferencesInStmt(uri, stmt.Init, name, locations)
		}
		collectLocalReferencesInExpr(uri, stmt.Condition, name, locations)
		collectLocalReferencesInBlock(uri, stmt.Body, name, locations)
		if stmt.Post != nil {
			collectLocalReferencesInStmt(uri, stmt.Post, name, locations)
		}
	case *ast.ExprStmt:
		collectLocalReferencesInExpr(uri, stmt.Expr, name, locations)
	}
}

func collectLocalReferencesInExpr(uri string, expr ast.Expression, name string, locations *[]Location) {
	switch expr := expr.(type) {
	case *ast.IdentExpr:
		if expr.Name == name {
			*locations = append(*locations, locationFor(uri, expr.Start, expr.Name))
		}
	case *ast.ArrayLiteral:
		for _, elem := range expr.Elements {
			collectLocalReferencesInExpr(uri, elem, name, locations)
		}
	case *ast.ListLiteral:
		for _, elem := range expr.Elements {
			collectLocalReferencesInExpr(uri, elem, name, locations)
		}
	case *ast.MakeExpr:
		collectLocalReferencesInExpr(uri, expr.Len, name, locations)
	case *ast.CallExpr:
		for _, arg := range expr.Args {
			collectLocalReferencesInExpr(uri, arg, name, locations)
		}
	case *ast.BinaryExpr:
		collectLocalReferencesInExpr(uri, expr.Left, name, locations)
		collectLocalReferencesInExpr(uri, expr.Right, name, locations)
	case *ast.IndexExpr:
		collectLocalReferencesInExpr(uri, expr.Collection, name, locations)
		collectLocalReferencesInExpr(uri, expr.Index, name, locations)
	case *ast.SliceExpr:
		collectLocalReferencesInExpr(uri, expr.Collection, name, locations)
		if expr.StartIndex != nil {
			collectLocalReferencesInExpr(uri, expr.StartIndex, name, locations)
		}
		if expr.EndIndex != nil {
			collectLocalReferencesInExpr(uri, expr.EndIndex, name, locations)
		}
	}
}

func isCallAt(text string, pos Position) bool {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return false
	}
	line := []rune(lines[pos.Line])
	_, end, ok := wordBoundsAt(text, pos)
	if !ok {
		return false
	}
	for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
		end++
	}
	return end < len(line) && line[end] == '('
}

func wordBoundsAt(text string, pos Position) (int, int, bool) {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return 0, 0, false
	}

	line := []rune(lines[pos.Line])
	if pos.Character < 0 || pos.Character > len(line) {
		return 0, 0, false
	}

	start := pos.Character
	if start == len(line) && start > 0 {
		start--
	}
	for start > 0 && isWordRune(line[start-1]) {
		start--
	}

	end := pos.Character
	for end < len(line) && isWordRune(line[end]) {
		end++
	}

	return start, end, start < end
}

func rangeForWord(text string, pos Position) *Range {
	start, end, ok := wordBoundsAt(text, pos)
	if !ok {
		return nil
	}

	return &Range{
		Start: Position{Line: pos.Line, Character: start},
		End:   Position{Line: pos.Line, Character: end},
	}
}

func importPathRange(tok token.Token) *Range {
	start := positionFromToken(tok.Pos)
	start.Character++
	end := start
	end.Character += len([]rune(tok.Lexeme))
	return &Range{Start: start, End: end}
}

func locationFor(uri string, pos token.Position, name string) Location {
	start := positionFromToken(pos)
	return Location{
		URI: uri,
		Range: Range{
			Start: start,
			End: Position{
				Line:      start.Line,
				Character: start.Character + len([]rune(name)),
			},
		},
	}
}

func tokenPositionFromLSP(pos Position) token.Position {
	return token.Position{Line: pos.Line + 1, Column: pos.Character + 1}
}

func containsName(cursor token.Position, start token.Position, name string) bool {
	if cursor.Line != start.Line {
		return false
	}
	return cursor.Column >= start.Column && cursor.Column <= start.Column+len([]rune(name))
}

func containsStringToken(cursor token.Position, tok token.Token) bool {
	if cursor.Line != tok.Pos.Line {
		return false
	}
	return cursor.Column >= tok.Pos.Column && cursor.Column <= tok.Pos.Column+len([]rune(tok.Lexeme))+1
}

func containsPosition(start token.Position, end token.Position, pos token.Position) bool {
	return !positionBefore(pos, start) && !positionBefore(end, pos)
}

func positionBefore(left token.Position, right token.Position) bool {
	if left.Line != right.Line {
		return left.Line < right.Line
	}
	return left.Column < right.Column
}

func samePosition(left token.Position, right token.Position) bool {
	return left.Line == right.Line && left.Column == right.Column
}

func normalizePath(path string) string {
	if path == "" {
		return path
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	return filepath.Clean(path)
}

func uriFromPath(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = "/" + strings.ReplaceAll(path, "\\", "/")
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}
