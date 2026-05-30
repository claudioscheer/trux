package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/parser"
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
}

type loader struct {
	loaded  map[string]*fileUnit
	sources map[string]string
	order   []*fileUnit
}

func Load(entryPath string) (*Result, error) {
	path, err := canonicalEntryPath(entryPath)
	if err != nil {
		return nil, err
	}

	l := &loader{
		loaded:  map[string]*fileUnit{},
		sources: map[string]string{},
	}
	entry, err := l.loadFile(path, token.Position{}, nil)
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

func canonicalEntryPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}
	abs = filepath.Clean(abs)

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

	return filepath.Clean(real), nil
}

func (l *loader) loadFile(path string, importPos token.Position, stack []string) (*fileUnit, error) {
	if cycleStart := indexPath(stack, path); cycleStart >= 0 {
		return nil, l.errorAt(importPos, "import cycle detected: %s", strings.Join(append(stack[cycleStart:], path), " -> "))
	}
	if unit, ok := l.loaded[path]; ok {
		return unit, nil
	}

	sourceBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	source := string(sourceBytes)
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
		importPath, err := l.resolveImport(unit, importDecl)
		if err != nil {
			return nil, err
		}
		if _, err := l.loadFile(importPath, importDecl.Pos, stack); err != nil {
			return nil, err
		}
	}

	return unit, nil
}

func (l *loader) resolveImport(unit *fileUnit, importDecl *ast.ImportDecl) (string, error) {
	if filepath.IsAbs(importDecl.Path) {
		return "", l.errorInUnit(unit, importDecl.Pos, "import path %q must be relative", importDecl.Path)
	}
	if filepath.Ext(importDecl.Path) != ".tx" {
		return "", l.errorInUnit(unit, importDecl.Pos, "import path %q must end in .tx", importDecl.Path)
	}

	candidate := filepath.Clean(filepath.Join(filepath.Dir(unit.path), importDecl.Path))
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
	publicFuncs := map[string]*ast.FuncDecl{}
	privateDecls := map[string][]*ast.FuncDecl{}
	privateNames := map[*fileUnit]map[string]string{}

	for _, unit := range l.order {
		seenInFile := map[string]*ast.FuncDecl{}
		privateNames[unit] = map[string]string{}

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
				continue
			}

			if fn.Public {
				if previous := publicFuncs[fn.Name]; previous != nil {
					return l.errorInUnit(unit, fn.Pos, "duplicate public function %q; first declared in %s", fn.Name, previous.Pos.File)
				}
				publicFuncs[fn.Name] = fn
				continue
			}

			privateDecls[fn.Name] = append(privateDecls[fn.Name], fn)
			privateNames[unit][fn.Name] = fmt.Sprintf("%smod_%d_%s", internalPrefix, unit.index, fn.Name)
		}
	}

	publicNames := map[string]struct{}{}
	for name := range publicFuncs {
		publicNames[name] = struct{}{}
	}

	needsHygiene := len(l.order) > 1
	if !needsHygiene {
		return nil
	}

	for _, unit := range l.order {
		for _, fn := range unit.program.Functions {
			if err := l.rewriteCalls(unit, fn.Body.Statements, privateNames[unit], publicNames, privateDecls); err != nil {
				return err
			}
			if internal, ok := privateNames[unit][fn.Name]; ok {
				fn.Name = internal
			}
		}
	}

	return nil
}

func (l *loader) rewriteCalls(unit *fileUnit, statements []ast.Statement, privateNames map[string]string, publicNames map[string]struct{}, privateDecls map[string][]*ast.FuncDecl) error {
	for _, stmt := range statements {
		if err := l.rewriteStmtCalls(unit, stmt, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) rewriteStmtCalls(unit *fileUnit, stmt ast.Statement, privateNames map[string]string, publicNames map[string]struct{}, privateDecls map[string][]*ast.FuncDecl) error {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		return l.rewriteExprCalls(unit, stmt.Value, privateNames, publicNames, privateDecls)
	case *ast.ReturnStmt:
		return l.rewriteExprCalls(unit, stmt.Value, privateNames, publicNames, privateDecls)
	case *ast.AssignStmt:
		return l.rewriteExprCalls(unit, stmt.Value, privateNames, publicNames, privateDecls)
	case *ast.IndexAssignStmt:
		if err := l.rewriteExprCalls(unit, stmt.Target, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
		return l.rewriteExprCalls(unit, stmt.Value, privateNames, publicNames, privateDecls)
	case *ast.IfStmt:
		if err := l.rewriteExprCalls(unit, stmt.Condition, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
		if err := l.rewriteCalls(unit, stmt.Then.Statements, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
		if stmt.Else != nil {
			return l.rewriteCalls(unit, stmt.Else.Statements, privateNames, publicNames, privateDecls)
		}
	case *ast.WhileStmt:
		if err := l.rewriteExprCalls(unit, stmt.Condition, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
		return l.rewriteCalls(unit, stmt.Body.Statements, privateNames, publicNames, privateDecls)
	case *ast.ExprStmt:
		return l.rewriteExprCalls(unit, stmt.Expr, privateNames, publicNames, privateDecls)
	}

	return nil
}

func (l *loader) rewriteExprCalls(unit *fileUnit, expr ast.Expression, privateNames map[string]string, publicNames map[string]struct{}, privateDecls map[string][]*ast.FuncDecl) error {
	switch expr := expr.(type) {
	case *ast.ArrayLiteral:
		return l.rewriteExprList(unit, expr.Elements, privateNames, publicNames, privateDecls)
	case *ast.ListLiteral:
		return l.rewriteExprList(unit, expr.Elements, privateNames, publicNames, privateDecls)
	case *ast.MakeExpr:
		return l.rewriteExprCalls(unit, expr.Len, privateNames, publicNames, privateDecls)
	case *ast.CallExpr:
		for _, arg := range expr.Args {
			if err := l.rewriteExprCalls(unit, arg, privateNames, publicNames, privateDecls); err != nil {
				return err
			}
		}
		if isBuiltin(expr.Callee) {
			return nil
		}
		if internal, ok := privateNames[expr.Callee]; ok {
			expr.Callee = internal
			return nil
		}
		if _, ok := publicNames[expr.Callee]; ok {
			return nil
		}
		if private := privateDecls[expr.Callee]; len(private) > 0 {
			return l.errorInUnit(unit, expr.Start, "cannot call private function %q from %s; it is declared in %s", expr.Callee, unit.path, private[0].Pos.File)
		}
	case *ast.BinaryExpr:
		if err := l.rewriteExprCalls(unit, expr.Left, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
		return l.rewriteExprCalls(unit, expr.Right, privateNames, publicNames, privateDecls)
	case *ast.IndexExpr:
		if err := l.rewriteExprCalls(unit, expr.Collection, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
		return l.rewriteExprCalls(unit, expr.Index, privateNames, publicNames, privateDecls)
	case *ast.SliceExpr:
		if err := l.rewriteExprCalls(unit, expr.Collection, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
		if expr.StartIndex != nil {
			if err := l.rewriteExprCalls(unit, expr.StartIndex, privateNames, publicNames, privateDecls); err != nil {
				return err
			}
		}
		if expr.EndIndex != nil {
			return l.rewriteExprCalls(unit, expr.EndIndex, privateNames, publicNames, privateDecls)
		}
	}

	return nil
}

func (l *loader) rewriteExprList(unit *fileUnit, exprs []ast.Expression, privateNames map[string]string, publicNames map[string]struct{}, privateDecls map[string][]*ast.FuncDecl) error {
	for _, expr := range exprs {
		if err := l.rewriteExprCalls(unit, expr, privateNames, publicNames, privateDecls); err != nil {
			return err
		}
	}
	return nil
}

func isBuiltin(name string) bool {
	switch name {
	case "print", "len", "clone", "append":
		return true
	default:
		return false
	}
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
