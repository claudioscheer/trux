package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ast"
)

func TestParsesV0Example(t *testing.T) {
	src, err := os.ReadFile("../../examples/v0/hello.tx")
	if err != nil {
		t.Fatal(err)
	}

	program, err := Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}

	if program.PackageName != "main" {
		t.Fatalf("package = %q, want %q", program.PackageName, "main")
	}
	if len(program.Functions) != 2 {
		t.Fatalf("function count = %d, want 2", len(program.Functions))
	}

	add := program.Functions[0]
	if add.Name != "add" {
		t.Fatalf("first function = %q, want %q", add.Name, "add")
	}
	if add.ReturnType != ast.IntType {
		t.Fatalf("add return type = %q, want %q", add.ReturnType, ast.IntType)
	}
	wantParams := []ast.Param{
		{Name: "a", Type: ast.IntType},
		{Name: "b", Type: ast.IntType},
	}
	if len(add.Params) != len(wantParams) {
		t.Fatalf("param count = %d, want %d", len(add.Params), len(wantParams))
	}
	for i := range wantParams {
		if add.Params[i] != wantParams[i] {
			t.Fatalf("param %d = %#v, want %#v", i, add.Params[i], wantParams[i])
		}
	}
}

func TestParsesLetWithCallExpression(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    let x int = add(1, 2)
}`)

	mainFn := program.Functions[0]
	if len(mainFn.Body.Statements) != 1 {
		t.Fatalf("statement count = %d, want 1", len(mainFn.Body.Statements))
	}

	stmt, ok := mainFn.Body.Statements[0].(ast.LetStmt)
	if !ok {
		t.Fatalf("statement = %T, want ast.LetStmt", mainFn.Body.Statements[0])
	}
	if stmt.Name != "x" || stmt.Type != ast.IntType {
		t.Fatalf("let statement = %#v, want name x and int type", stmt)
	}

	call, ok := stmt.Value.(ast.CallExpr)
	if !ok {
		t.Fatalf("let value = %T, want ast.CallExpr", stmt.Value)
	}
	if call.Callee != "add" {
		t.Fatalf("callee = %q, want %q", call.Callee, "add")
	}
	if len(call.Args) != 2 {
		t.Fatalf("arg count = %d, want 2", len(call.Args))
	}
}

func TestParsesBinaryOperatorPrecedence(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    return a + b * 2
}`)

	stmt := program.Functions[0].Body.Statements[0].(ast.ReturnStmt)
	expr, ok := stmt.Value.(ast.BinaryExpr)
	if !ok {
		t.Fatalf("return value = %T, want ast.BinaryExpr", stmt.Value)
	}
	if expr.Operator != "+" {
		t.Fatalf("root operator = %q, want +", expr.Operator)
	}

	right, ok := expr.Right.(ast.BinaryExpr)
	if !ok {
		t.Fatalf("right expression = %T, want ast.BinaryExpr", expr.Right)
	}
	if right.Operator != "*" {
		t.Fatalf("right operator = %q, want *", right.Operator)
	}
}

func TestReportsSyntaxErrorWithExpectedActualAndPosition(t *testing.T) {
	_, err := Parse(`package main
func main( int {
    return 0
}`)
	if err == nil {
		t.Fatal("expected error")
	}

	want := `parse error at 2:12: expected IDENT, got "int"`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestReportsIllegalTokenAsLexError(t *testing.T) {
	_, err := Parse("@")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), `lex error: illegal character "@"`) {
		t.Fatalf("error = %q, want lex error", err.Error())
	}
}

func mustParse(t *testing.T, src string) *ast.Program {
	t.Helper()

	program, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	return program
}
