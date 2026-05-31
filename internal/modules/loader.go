package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/parser"
	"github.com/claudioscheer/trux/internal/stdlib"
	"github.com/claudioscheer/trux/internal/token"
)

const internalPrefix = "__trux_"

type Result struct {
	EntryPath string
	Program   *ast.Program
	Sources   map[string]string
}

type Error struct {
	File   string
	Source string
	Pos    token.Position
	Msg    string
}

func (e *Error) Error() string {
	if e.Pos.Line == 0 {
		return e.Msg
	}
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Pos.Line, e.Pos.Column, e.Msg)
}

type fileUnit struct {
	path    string
	source  string
	program *ast.Program
	index   int
	imports []importRef
}

type importRef struct {
	decl    *ast.ImportDecl
	unit    *fileUnit
	stdName string
}

type functionInfo struct {
	unit         *fileUnit
	decl         *ast.FuncDecl
	internalName string
	public       bool
}

type packageKey struct {
	dir  string
	name string
}

type resolutionContext struct {
	funcs         map[*fileUnit]map[string]*functionInfo
	packageFuncs  map[packageKey]map[string]*functionInfo
	directImports map[*fileUnit]map[string]importSet
	privateDecls  map[string][]*functionInfo
	publicDecls   map[string][]*functionInfo
}

type importSet struct {
	refs []importRef
}

type loader struct {
	loaded   map[string]*fileUnit
	sources  map[string]string
	overlays map[string]string
	order    []*fileUnit
}

func Load(entryPath string) (*Result, error) {
	return load(entryPath, nil)
}

func LoadWithSources(entryPath string, sources map[string]string) (*Result, error) {
	return load(entryPath, normalizeSources(sources))
}

func load(entryPath string, sources map[string]string) (*Result, error) {
	path, err := canonicalEntryPath(entryPath, sources)
	if err != nil {
		return nil, err
	}

	l := &loader{
		loaded:   map[string]*fileUnit{},
		sources:  map[string]string{},
		overlays: sources,
	}
	entry, err := l.loadFile(path, token.Position{}, nil, false)
	if err != nil {
		return nil, err
	}

	if err := l.validateAndMerge(entry); err != nil {
		return nil, err
	}

	program := &ast.Program{PackageName: entry.program.PackageName}
	for _, unit := range l.order {
		program.Functions = append(program.Functions, unit.program.Functions...)
	}

	return &Result{
		EntryPath: entry.path,
		Program:   program,
		Sources:   l.sources,
	}, nil
}

func normalizeSources(sources map[string]string) map[string]string {
	if len(sources) == 0 {
		return nil
	}

	normalized := map[string]string{}
	for path, source := range sources {
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
		path = filepath.Clean(path)
		if real, err := filepath.EvalSymlinks(path); err == nil {
			path = filepath.Clean(real)
		}
		normalized[path] = source
	}
	return normalized
}

func canonicalEntryPath(path string, sources map[string]string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}
	abs = filepath.Clean(abs)

	if _, ok := sources[abs]; ok {
		return abs, nil
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("read %s: is a directory", path)
	}

	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}

	if _, ok := sources[filepath.Clean(real)]; ok {
		return filepath.Clean(real), nil
	}

	return filepath.Clean(real), nil
}

func (l *loader) loadFile(path string, importPos token.Position, stack []string, includeSiblings bool) (*fileUnit, error) {
	if cycleStart := indexPath(stack, path); cycleStart >= 0 {
		return nil, l.errorAt(importPos, "import cycle detected: %s", strings.Join(append(stack[cycleStart:], path), " -> "))
	}
	if unit, ok := l.loaded[path]; ok {
		return unit, nil
	}

	source, err := l.readSource(path)
	if err != nil {
		return nil, err
	}
	l.sources[path] = source

	program, err := parser.ParseFile(path, source)
	if err != nil {
		if parseErr, ok := err.(*parser.ParseError); ok {
			return nil, l.errorWithSource(path, source, parseErr.Pos, parseErr.Msg)
		}
		return nil, err
	}

	unit := &fileUnit{
		path:    path,
		source:  source,
		program: program,
		index:   len(l.order),
	}
	l.loaded[path] = unit
	l.order = append(l.order, unit)

	stack = append(stack, path)
	for _, importDecl := range program.Imports {
		if stdlib.IsPackage(importDecl.Path) {
			unit.imports = append(unit.imports, importRef{decl: importDecl, stdName: importDecl.Path})
			continue
		}

		importPath, err := l.resolveImport(unit, importDecl)
		if err != nil {
			return nil, err
		}
		imported, err := l.loadFile(importPath, importDecl.Pos, stack, true)
		if err != nil {
			return nil, err
		}
		unit.imports = append(unit.imports, importRef{decl: importDecl, unit: imported})
	}

	if includeSiblings {
		if err := l.loadSamePackageSiblings(unit, stack); err != nil {
			return nil, err
		}
	}

	return unit, nil
}

func (l *loader) readSource(path string) (string, error) {
	if source, ok := l.overlays[path]; ok {
		return source, nil
	}

	sourceBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(sourceBytes), nil
}

func (l *loader) loadSamePackageSiblings(unit *fileUnit, stack []string) error {
	for _, path := range l.samePackageSiblingPaths(unit) {
		if _, err := l.loadFile(path, token.Position{}, stack, true); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) samePackageSiblingPaths(unit *fileUnit) []string {
	dir := filepath.Dir(unit.path)
	candidates := map[string]struct{}{}

	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".tx" {
				continue
			}
			path := filepath.Clean(filepath.Join(dir, entry.Name()))
			if real, err := filepath.EvalSymlinks(path); err == nil {
				path = filepath.Clean(real)
			}
			candidates[path] = struct{}{}
		}
	}

	for path := range l.overlays {
		if filepath.Dir(path) == dir && filepath.Ext(path) == ".tx" {
			candidates[path] = struct{}{}
		}
	}

	delete(candidates, unit.path)

	paths := make([]string, 0, len(candidates))
	for path := range candidates {
		if _, loaded := l.loaded[path]; loaded {
			continue
		}
		source, err := l.readSource(path)
		if err != nil {
			continue
		}
		if packageNameFromSource(path, source) != unit.program.PackageName {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func packageNameFromSource(path string, source string) string {
	tokens := lexer.LexFile(path, source)
	if len(tokens) < 2 || tokens[0].Type != token.Package || tokens[1].Type != token.Ident {
		return ""
	}
	return tokens[1].Lexeme
}

func (l *loader) resolveImport(unit *fileUnit, importDecl *ast.ImportDecl) (string, error) {
	if filepath.IsAbs(importDecl.Path) {
		return "", l.errorInUnit(unit, importDecl.Pos, "import path %q must be relative", importDecl.Path)
	}
	if filepath.Ext(importDecl.Path) != ".tx" {
		if isBareImport(importDecl.Path) {
			return "", l.errorInUnit(unit, importDecl.Pos, "unknown standard package %q", importDecl.Path)
		}
		return "", l.errorInUnit(unit, importDecl.Pos, "import path %q must end in .tx", importDecl.Path)
	}

	candidate := filepath.Clean(filepath.Join(filepath.Dir(unit.path), importDecl.Path))
	if _, ok := l.overlays[candidate]; ok {
		return candidate, nil
	}

	info, err := os.Stat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", l.errorInUnit(unit, importDecl.Pos, "cannot find module %q resolved to %s", importDecl.Path, candidate)
		}
		return "", l.errorInUnit(unit, importDecl.Pos, "cannot read module %q resolved to %s: %v", importDecl.Path, candidate, err)
	}
	if info.IsDir() {
		return "", l.errorInUnit(unit, importDecl.Pos, "module import %q resolved to directory %s; expected .tx file", importDecl.Path, candidate)
	}

	real, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", l.errorInUnit(unit, importDecl.Pos, "cannot resolve module %q resolved to %s: %v", importDecl.Path, candidate, err)
	}

	return filepath.Clean(real), nil
}

func (l *loader) validateAndMerge(entry *fileUnit) error {
	needsHygiene := len(l.order) > 1
	ctx := resolutionContext{
		funcs:         map[*fileUnit]map[string]*functionInfo{},
		packageFuncs:  map[packageKey]map[string]*functionInfo{},
		directImports: map[*fileUnit]map[string]importSet{},
		privateDecls:  map[string][]*functionInfo{},
		publicDecls:   map[string][]*functionInfo{},
	}

	for _, unit := range l.order {
		seenInFile := map[string]*ast.FuncDecl{}
		ctx.funcs[unit] = map[string]*functionInfo{}
		key := unit.packageKey()
		if ctx.packageFuncs[key] == nil {
			ctx.packageFuncs[key] = map[string]*functionInfo{}
		}

		for _, fn := range unit.program.Functions {
			if strings.HasPrefix(fn.Name, internalPrefix) {
				return l.errorInUnit(unit, fn.Pos, "function name %q uses reserved compiler prefix %q", fn.Name, internalPrefix)
			}
			if previous := seenInFile[fn.Name]; previous != nil {
				return l.errorInUnit(unit, fn.Pos, "duplicate function %q in %s", fn.Name, unit.path)
			}
			seenInFile[fn.Name] = fn

			if fn.Name == "main" {
				if unit.path != entry.path {
					return l.errorInUnit(unit, fn.Pos, "imported module %s must not define main", unit.path)
				}
				if fn.Public {
					return l.errorInUnit(unit, fn.Pos, "main cannot be public")
				}
			}

			internalName := fn.Name
			if needsHygiene && fn.Name != "main" {
				internalName = fmt.Sprintf("%smod_%d_%s", internalPrefix, unit.index, fn.Name)
			}
			info := &functionInfo{
				unit:         unit,
				decl:         fn,
				internalName: internalName,
				public:       fn.Public,
			}
			ctx.funcs[unit][fn.Name] = info
			if previous := ctx.packageFuncs[key][fn.Name]; previous != nil {
				return l.errorInUnit(unit, fn.Pos, "package %q defines function %q in both %s and %s", key.name, fn.Name, previous.decl.Pos.File, info.decl.Pos.File)
			}
			ctx.packageFuncs[key][fn.Name] = info

			if fn.Name == "main" {
				continue
			}
			if fn.Public {
				ctx.publicDecls[fn.Name] = append(ctx.publicDecls[fn.Name], info)
				continue
			}

			ctx.privateDecls[fn.Name] = append(ctx.privateDecls[fn.Name], info)
		}
	}

	for _, unit := range l.order {
		direct := map[string]importSet{}
		for _, ref := range unit.imports {
			packageName := ref.packageName()
			set := direct[packageName]
			if len(set.refs) > 0 {
				if set.hasTarget(ref) {
					continue
				}
				if set.isStdlib() || ref.isStdlib() {
					return l.errorInUnit(unit, ref.decl.Pos, "package %q imported from both %s and %s", packageName, set.describe(), ref.describe())
				}
			}
			set.refs = append(set.refs, ref)
			direct[packageName] = set
		}
		for packageName, set := range direct {
			if err := l.validateDirectImportSet(unit, packageName, set, ctx); err != nil {
				return err
			}
		}
		ctx.directImports[unit] = direct
	}

	for _, unit := range l.order {
		for _, fn := range unit.program.Functions {
			if err := l.rewriteCalls(unit, fn.Body.Statements, ctx); err != nil {
				return err
			}
			if info := ctx.funcs[unit][fn.Name]; info != nil && info.internalName != fn.Name {
				fn.Name = info.internalName
			}
		}
	}

	return nil
}

func (l *loader) validateDirectImportSet(unit *fileUnit, packageName string, set importSet, ctx resolutionContext) error {
	if set.isStdlib() {
		return nil
	}

	seen := map[string]*functionInfo{}
	seenPackages := map[packageKey]struct{}{}
	for _, ref := range set.refs {
		key := ref.unit.packageKey()
		if _, seenPackage := seenPackages[key]; seenPackage {
			continue
		}
		seenPackages[key] = struct{}{}
		for name, info := range ctx.packageFuncs[key] {
			if !info.public {
				continue
			}
			if previous := seen[name]; previous != nil {
				return l.errorInUnit(unit, ref.decl.Pos, "package %q exports function %q from both %s and %s", packageName, name, previous.decl.Pos.File, info.decl.Pos.File)
			}
			seen[name] = info
		}
	}

	return nil
}

func (l *loader) rewriteCalls(unit *fileUnit, statements []ast.Statement, ctx resolutionContext) error {
	for _, stmt := range statements {
		if err := l.rewriteStmtCalls(unit, stmt, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) rewriteStmtCalls(unit *fileUnit, stmt ast.Statement, ctx resolutionContext) error {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		return l.rewriteExprCalls(unit, stmt.Value, ctx)
	case *ast.ReturnStmt:
		return l.rewriteExprCalls(unit, stmt.Value, ctx)
	case *ast.AssignStmt:
		return l.rewriteExprCalls(unit, stmt.Value, ctx)
	case *ast.IndexAssignStmt:
		if err := l.rewriteExprCalls(unit, stmt.Target, ctx); err != nil {
			return err
		}
		return l.rewriteExprCalls(unit, stmt.Value, ctx)
	case *ast.IfStmt:
		if err := l.rewriteExprCalls(unit, stmt.Condition, ctx); err != nil {
			return err
		}
		if err := l.rewriteCalls(unit, stmt.Then.Statements, ctx); err != nil {
			return err
		}
		if stmt.Else != nil {
			return l.rewriteCalls(unit, stmt.Else.Statements, ctx)
		}
	case *ast.WhileStmt:
		if err := l.rewriteExprCalls(unit, stmt.Condition, ctx); err != nil {
			return err
		}
		return l.rewriteCalls(unit, stmt.Body.Statements, ctx)
	case *ast.ExprStmt:
		return l.rewriteExprCalls(unit, stmt.Expr, ctx)
	}

	return nil
}

func (l *loader) rewriteExprCalls(unit *fileUnit, expr ast.Expression, ctx resolutionContext) error {
	switch expr := expr.(type) {
	case *ast.ArrayLiteral:
		return l.rewriteExprList(unit, expr.Elements, ctx)
	case *ast.ListLiteral:
		return l.rewriteExprList(unit, expr.Elements, ctx)
	case *ast.MakeExpr:
		return l.rewriteExprCalls(unit, expr.Len, ctx)
	case *ast.CallExpr:
		for _, arg := range expr.Args {
			if err := l.rewriteExprCalls(unit, arg, ctx); err != nil {
				return err
			}
		}
		if expr.Package != "" {
			return l.resolveQualifiedCall(unit, expr, ctx)
		}
		if isBuiltin(expr.Callee) {
			return nil
		}
		if info := ctx.funcs[unit][expr.Callee]; info != nil {
			expr.ResolvedCallee = info.internalName
			return nil
		}
		if info := ctx.packageFuncs[unit.packageKey()][expr.Callee]; info != nil {
			expr.ResolvedCallee = info.internalName
			return nil
		}
		if packageName, ok := l.directExportingPackage(unit, expr.Callee, ctx); ok {
			return l.errorInUnit(unit, expr.Start, "imported function %q must be called as %q", expr.Callee, packageName+"."+expr.Callee)
		}
		if private := ctx.privateDecls[expr.Callee]; len(private) > 0 {
			return l.errorInUnit(unit, expr.Start, "cannot call private function %q from %s; it is declared in %s", expr.Callee, unit.path, private[0].decl.Pos.File)
		}
		if public := ctx.publicDecls[expr.Callee]; len(public) > 0 {
			packageName := public[0].unit.program.PackageName
			return l.errorInUnit(unit, expr.Start, "cannot call imported function %q without a direct package import; import package %q and call %s.%s", expr.Callee, packageName, packageName, expr.Callee)
		}
	case *ast.BinaryExpr:
		if err := l.rewriteExprCalls(unit, expr.Left, ctx); err != nil {
			return err
		}
		return l.rewriteExprCalls(unit, expr.Right, ctx)
	case *ast.IndexExpr:
		if err := l.rewriteExprCalls(unit, expr.Collection, ctx); err != nil {
			return err
		}
		return l.rewriteExprCalls(unit, expr.Index, ctx)
	case *ast.SliceExpr:
		if err := l.rewriteExprCalls(unit, expr.Collection, ctx); err != nil {
			return err
		}
		if expr.StartIndex != nil {
			if err := l.rewriteExprCalls(unit, expr.StartIndex, ctx); err != nil {
				return err
			}
		}
		if expr.EndIndex != nil {
			return l.rewriteExprCalls(unit, expr.EndIndex, ctx)
		}
	}

	return nil
}

func (l *loader) rewriteExprList(unit *fileUnit, exprs []ast.Expression, ctx resolutionContext) error {
	for _, expr := range exprs {
		if err := l.rewriteExprCalls(unit, expr, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) resolveQualifiedCall(unit *fileUnit, expr *ast.CallExpr, ctx resolutionContext) error {
	set, ok := ctx.directImports[unit][expr.Package]
	if !ok {
		return l.errorInUnit(unit, expr.Start, "package %q is not imported by %s", expr.Package, unit.path)
	}

	if set.isStdlib() {
		if _, ok := stdlib.LookupMember(expr.Package, expr.Callee); !ok {
			return l.errorInUnit(unit, expr.Start, "package %q has no function %q", expr.Package, expr.Callee)
		}
		expr.Stdlib = true
		return nil
	}

	var private *functionInfo
	seenPackages := map[packageKey]struct{}{}
	for _, ref := range set.refs {
		key := ref.unit.packageKey()
		if _, seenPackage := seenPackages[key]; seenPackage {
			continue
		}
		seenPackages[key] = struct{}{}
		info := ctx.packageFuncs[key][expr.Callee]
		if info == nil {
			continue
		}
		if !info.public {
			if private == nil {
				private = info
			}
			continue
		}
		expr.ResolvedCallee = info.internalName
		return nil
	}
	if private != nil {
		return l.errorInUnit(unit, expr.Start, "cannot call private function %q from %s; it is declared in %s", expr.SourceName(), unit.path, private.decl.Pos.File)
	}
	return l.errorInUnit(unit, expr.Start, "package %q has no function %q", expr.Package, expr.Callee)
}

func (l *loader) directExportingPackage(unit *fileUnit, name string, ctx resolutionContext) (string, bool) {
	packages := make([]string, 0, len(ctx.directImports[unit]))
	for packageName := range ctx.directImports[unit] {
		packages = append(packages, packageName)
	}
	sort.Strings(packages)

	for _, packageName := range packages {
		set := ctx.directImports[unit][packageName]
		if set.isStdlib() {
			if _, ok := stdlib.LookupMember(packageName, name); ok {
				return packageName, true
			}
			continue
		}
		seenPackages := map[packageKey]struct{}{}
		for _, ref := range set.refs {
			key := ref.unit.packageKey()
			if _, seenPackage := seenPackages[key]; seenPackage {
				continue
			}
			seenPackages[key] = struct{}{}
			info := ctx.packageFuncs[key][name]
			if info != nil && info.public {
				return packageName, true
			}
		}
	}

	return "", false
}

func isBuiltin(name string) bool {
	switch name {
	case "print", "len", "clone", "append":
		return true
	default:
		return false
	}
}

func (set importSet) isStdlib() bool {
	return len(set.refs) > 0 && set.refs[0].isStdlib()
}

func (set importSet) hasTarget(other importRef) bool {
	for _, ref := range set.refs {
		if ref.sameTarget(other) {
			return true
		}
	}
	return false
}

func (set importSet) describe() string {
	descriptions := make([]string, 0, len(set.refs))
	for _, ref := range set.refs {
		descriptions = append(descriptions, ref.describe())
	}
	return strings.Join(descriptions, ", ")
}

func (ref importRef) isStdlib() bool {
	return ref.stdName != ""
}

func (ref importRef) packageName() string {
	if ref.isStdlib() {
		return ref.stdName
	}
	return ref.unit.program.PackageName
}

func (ref importRef) sameTarget(other importRef) bool {
	if ref.isStdlib() || other.isStdlib() {
		return ref.stdName == other.stdName
	}
	return ref.unit.path == other.unit.path
}

func (ref importRef) describe() string {
	if ref.isStdlib() {
		return fmt.Sprintf("standard package %q", ref.stdName)
	}
	return ref.unit.path
}

func (unit *fileUnit) packageKey() packageKey {
	return packageKey{
		dir:  filepath.Dir(unit.path),
		name: unit.program.PackageName,
	}
}

func isBareImport(path string) bool {
	return filepath.Clean(path) == path && !strings.ContainsAny(path, `/\`) && filepath.Ext(path) == ""
}

func indexPath(paths []string, path string) int {
	for i, existing := range paths {
		if existing == path {
			return i
		}
	}
	return -1
}

func (l *loader) errorAt(pos token.Position, format string, args ...any) error {
	file := pos.File
	source := l.sources[file]
	return l.errorWithSource(file, source, pos, fmt.Sprintf(format, args...))
}

func (l *loader) errorInUnit(unit *fileUnit, pos token.Position, format string, args ...any) error {
	return l.errorWithSource(unit.path, unit.source, pos, fmt.Sprintf(format, args...))
}

func (l *loader) errorWithSource(file string, source string, pos token.Position, msg string) error {
	if pos.File == "" {
		pos.File = file
	}
	return &Error{
		File:   file,
		Source: source,
		Pos:    pos,
		Msg:    msg,
	}
}
