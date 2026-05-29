package ir

import (
	"testing"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/parser"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

func TestBuildCreatesTypedIRForV0Program(t *testing.T) {
	program := mustParse(t, `package main
func add(a int, b int) int {
    return a + b
}

func main() int {
    let x int = add(1, 2)
    print(x)
    return 0
}`)
	info, err := semtypes.Check(program)
	if err != nil {
		t.Fatal(err)
	}

	irProgram, err := Build(program, info)
	if err != nil {
		t.Fatal(err)
	}

	if irProgram.PackageName != "main" {
		t.Fatalf("package = %q, want main", irProgram.PackageName)
	}
	if len(irProgram.Functions) != 2 {
		t.Fatalf("function count = %d, want 2", len(irProgram.Functions))
	}

	add := irProgram.Functions[0]
	if add.Name != "add" || add.ReturnType != ast.IntType {
		t.Fatalf("add function = %#v, want int-returning add", add)
	}
	returnStmt := add.Body[0].(*ReturnStmt)
	binary := returnStmt.Value.(*BinaryExpr)
	if binary.Operator != "+" || binary.Type() != ast.IntType {
		t.Fatalf("binary = %#v, want typed + expression", binary)
	}

	mainFn := irProgram.Functions[1]
	letStmt := mainFn.Body[0].(*LetStmt)
	call := letStmt.Value.(*CallExpr)
	if letStmt.Name != "x" || letStmt.Type != ast.IntType {
		t.Fatalf("let = %#v, want x int", letStmt)
	}
	if call.Callee != "add" || call.Type() != ast.IntType {
		t.Fatalf("call = %#v, want resolved int add call", call)
	}

	printStmt := mainFn.Body[1].(*PrintStmt)
	if printStmt.Type != ast.IntType {
		t.Fatalf("print type = %q, want %q", printStmt.Type, ast.IntType)
	}
	if printStmt.Arg.Type() != ast.IntType {
		t.Fatalf("print arg type = %q, want %q", printStmt.Arg.Type(), ast.IntType)
	}
}

func mustParse(t *testing.T, src string) *ast.Program {
	t.Helper()

	program, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	return program
}
