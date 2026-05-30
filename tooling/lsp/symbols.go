package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/parser"
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

	items := baseCompletionItems()
	seen := completionLabels(items)

	text, ok := s.documents[params.TextDocument.URI]
	if !ok {
		var err error
		text, err = readURI(params.TextDocument.URI)
		if err != nil {
			return completionListFromItems(items), nil
		}
	}

	path := normalizePath(pathFromURI(params.TextDocument.URI))
	graph := loadSourceGraph(path, text, s.documents)
	file := graph.files[path]
	if file == nil {
		for _, item := range importedFunctionCompletionItems(path, text, s.documents) {
			items = appendCompletionItem(items, seen, item)
		}
		return completionListFromItems(items), nil
	}

	for _, item := range visibleLocalCompletionItems(file, tokenPositionFromLSP(params.Position)) {
		items = appendCompletionItem(items, seen, item)
	}
	for _, item := range functionCompletionItems(graph, path) {
		items = appendCompletionItem(items, seen, item)
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

func (s *Server) resolveImportPath(uri string, pos Position) (LocationLink, bool, error) {
	ref, ok, err := s.importPathAt(uri, pos)
	if err != nil || !ok {
		return LocationLink{}, ok, err
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
		if filepath.IsAbs(importToken.Lexeme) || filepath.Ext(importToken.Lexeme) != ".tx" {
			return importPathReference{}, false, nil
		}

		return importPathReference{path: importToken.Lexeme, token: importToken}, true, nil
	}

	return importPathReference{}, false, nil
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
		case *ast.WhileStmt:
			if containsPosition(stmt.Body.Start, stmt.Body.End, cursor) {
				return visibleLocalsInBlock(stmt.Body, locals, cursor)
			}
		}
	}

	return locals
}

func functionCompletionItems(graph *sourceGraph, activePath string) []CompletionItem {
	items := []CompletionItem{}
	for _, file := range graph.order {
		includePrivate := file.path == activePath
		for _, fn := range file.program.Functions {
			if !includePrivate && !fn.Public {
				continue
			}
			detail := "function"
			if file.path != activePath {
				detail = "imported function"
			}
			items = append(items, CompletionItem{Label: fn.Name, Kind: completionKindFunction, Detail: detail})
		}
	}
	return items
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
	var load func(path string)
	load = func(path string) {
		path = normalizePath(path)
		if _, exists := graph.files[path]; exists {
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
			load(filepath.Join(filepath.Dir(path), importDecl.Path))
		}
	}

	load(activePath)
	return graph
}

func importedFunctionCompletionItems(activePath string, activeText string, documents map[string]string) []CompletionItem {
	openDocuments := map[string]string{}
	for uri, text := range documents {
		openDocuments[normalizePath(pathFromURI(uri))] = text
	}

	graph := &sourceGraph{files: map[string]*sourceFile{}}
	var load func(path string)
	load = func(path string) {
		path = normalizePath(path)
		if _, exists := graph.files[path]; exists {
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
			load(filepath.Join(filepath.Dir(path), importDecl.Path))
		}
	}

	for _, importPath := range importPathsFromText(activePath, activeText) {
		load(importPath)
	}

	return functionCompletionItems(graph, activePath)
}

func importPathsFromText(activePath string, text string) []string {
	tokens := lexer.LexFile(activePath, text)
	paths := []string{}
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type != token.Import || tokens[i+1].Type != token.String {
			continue
		}
		importPath := tokens[i+1].Lexeme
		if filepath.IsAbs(importPath) || filepath.Ext(importPath) != ".tx" {
			continue
		}
		paths = append(paths, filepath.Join(filepath.Dir(activePath), importPath))
	}
	return paths
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
		case *ast.WhileStmt:
			if containsPosition(stmt.Body.Start, stmt.Body.End, cursor) {
				return findLocalDefinitionInBlock(uri, stmt.Body, locals, word, cursor)
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
	if file := graph.files[activePath]; file != nil {
		if def, ok := findFunctionInFile(file, word, true); ok {
			return def, true
		}
	}

	for _, file := range graph.order {
		if file.path == activePath {
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

	for _, file := range graph.order {
		if !def.Public && file.path != def.Path {
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
	case *ast.WhileStmt:
		collectFunctionReferencesInExpr(uri, stmt.Condition, name, locations)
		collectFunctionReferencesInBlock(uri, stmt.Body, name, locations)
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
			*locations = append(*locations, locationFor(uri, expr.Start, expr.Callee))
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
		case *ast.WhileStmt:
			collectLocalReferencesInExpr(uri, stmt.Condition, name, locations)
			collectLocalReferencesInBlock(uri, stmt.Body, name, locations)
		case *ast.ExprStmt:
			collectLocalReferencesInExpr(uri, stmt.Expr, name, locations)
		}
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
