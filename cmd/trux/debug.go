package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/token"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

type debugWriter struct {
	dir string
}

func newDebugWriter(sourcePath string) (*debugWriter, error) {
	name := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	if name == "" {
		name = "source"
	}

	dir := filepath.Join("tmp", "trux-debug", name)
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("reset debug dir %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create debug dir %s: %w", dir, err)
	}

	return &debugWriter{dir: dir}, nil
}

func (d *debugWriter) writeText(name string, content string) error {
	path := filepath.Join(d.dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write debug file %s: %w", path, err)
	}

	return nil
}

func (d *debugWriter) writeJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode debug file %s: %w", name, err)
	}
	data = append(data, '\n')

	return d.writeText(name, string(data))
}

func formatTokens(tokens []token.Token) string {
	var out bytes.Buffer
	for _, tok := range tokens {
		fmt.Fprintf(&out, "%d:%d\t%s\t%q\n", tok.Pos.Line, tok.Pos.Column, tok.Type, tok.Lexeme)
	}

	return out.String()
}

func formatTypeInfo(program *ast.Program, info *semtypes.Info) string {
	var out bytes.Buffer

	fmt.Fprintln(&out, "functions")
	functionNames := make([]string, 0, len(info.Funcs))
	for name := range info.Funcs {
		functionNames = append(functionNames, name)
	}
	sort.Strings(functionNames)
	for _, name := range functionNames {
		sig := info.Funcs[name]
		fmt.Fprintf(&out, "  %s(", name)
		for i, param := range sig.Params {
			if i > 0 {
				fmt.Fprint(&out, ", ")
			}
			fmt.Fprintf(&out, "%s %s", param.Name, param.Type)
		}
		fmt.Fprintf(&out, ") %s\n", sig.ReturnType)
	}

	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "locals")
	for _, fn := range program.Functions {
		fmt.Fprintf(&out, "  %s\n", fn.Name)
		names := make([]string, 0, len(info.Locals[fn]))
		for name := range info.Locals[fn] {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(&out, "    %s %s\n", name, info.Locals[fn][name])
		}
	}

	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "resolved calls")
	for _, fn := range program.Functions {
		formatResolvedCalls(&out, fn.Body.Statements, info)
	}

	return out.String()
}

func formatResolvedCalls(out *bytes.Buffer, statements []ast.Statement, info *semtypes.Info) {
	for _, stmt := range statements {
		switch stmt := stmt.(type) {
		case *ast.LetStmt:
			formatExprCalls(out, stmt.Value, info)
		case *ast.ReturnStmt:
			formatExprCalls(out, stmt.Value, info)
		case *ast.ExprStmt:
			formatExprCalls(out, stmt.Expr, info)
		}
	}
}

func formatExprCalls(out *bytes.Buffer, expr ast.Expression, info *semtypes.Info) {
	switch expr := expr.(type) {
	case *ast.CallExpr:
		if sig, ok := info.ResolvedCalls[expr]; ok {
			fmt.Fprintf(out, "  %d:%d %s -> %s\n", expr.Start.Line, expr.Start.Column, expr.Callee, sig.ReturnType)
		}
		if typ, ok := info.PrintCalls[expr]; ok {
			fmt.Fprintf(out, "  %d:%d print -> print(%s)\n", expr.Start.Line, expr.Start.Column, typ)
		}
		for _, arg := range expr.Args {
			formatExprCalls(out, arg, info)
		}
	case *ast.BinaryExpr:
		formatExprCalls(out, expr.Left, info)
		formatExprCalls(out, expr.Right, info)
	}
}
