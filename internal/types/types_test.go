package types

import (
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/parser"
)

func TestCheckValidV0Program(t *testing.T) {
	program := mustParse(t, `package main
func add(a int, b int) int {
    return a + b
}

func main() int {
    let x int = add(1, 2)
    print(x)
    return 0
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}

	mainFn := program.Functions[1]
	letStmt := mainFn.Body.Statements[0].(*ast.LetStmt)
	call := letStmt.Value.(*ast.CallExpr)
	if info.ExprTypes[call] != ast.IntType {
		t.Fatalf("call type = %q, want %q", info.ExprTypes[call], ast.IntType)
	}
	if info.ResolvedCalls[call].Name != "add" {
		t.Fatalf("resolved call = %q, want add", info.ResolvedCalls[call].Name)
	}

	printStmt := mainFn.Body.Statements[1].(*ast.ExprStmt)
	printCall := printStmt.Expr.(*ast.CallExpr)
	if info.PrintCalls[printCall] != ast.IntType {
		t.Fatalf("print call type = %q, want %q", info.PrintCalls[printCall], ast.IntType)
	}
}

func TestCheckRejectsV0SemanticErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "undefined variable",
			src: `package main
func main() int {
    return missing
}`,
			want: `undefined variable "missing"`,
		},
		{
			name: "undefined function",
			src: `package main
func main() int {
    return missing()
}`,
			want: `undefined function "missing"`,
		},
		{
			name: "duplicate function",
			src: `package main
func main() int {
    return 0
}
func main() int {
    return 0
}`,
			want: `duplicate function "main"`,
		},
		{
			name: "duplicate local",
			src: `package main
func main() int {
    let x int = 1
    let x int = 2
    return x
}`,
			want: `duplicate local variable "x"`,
		},
		{
			name: "wrong argument count",
			src: `package main
func add(a int, b int) int {
    return a + b
}
func main() int {
    return add(1)
}`,
			want: `add expects 2 arguments, got 1`,
		},
		{
			name: "missing main",
			src: `package main
func add(a int, b int) int {
    return a + b
}`,
			want: `missing main function`,
		},
		{
			name: "main parameter",
			src: `package main
func main(x int) int {
    return x
}`,
			want: `main must not have parameters`,
		},
		{
			name: "print arity",
			src: `package main
func main() int {
    print()
    return 0
}`,
			want: `print expects 1 argument, got 0`,
		},
		{
			name: "print expression",
			src: `package main
func main() int {
    return print(1)
}`,
			want: `print can only be used as a statement`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := mustParse(t, tt.src)
			_, err := Check(program)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.want)
			}
		})
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
