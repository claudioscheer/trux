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
	if !sameTypes(info.PrintCalls[printCall], []ast.Type{ast.IntType}) {
		t.Fatalf("print call types = %q, want %q", info.PrintCalls[printCall], []ast.Type{ast.IntType})
	}
}

func TestCheckValidV1Program(t *testing.T) {
	program := mustParse(t, `package main
func label() string {
    return "Kern"
}

func ready() bool {
    return true
}

func main() int {
    let name string = label()
    let ok bool = ready()
    print(name, 1)
    print(ok)
    print(1)
    return 0
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}

	mainFn := program.Functions[2]
	nameLet := mainFn.Body.Statements[0].(*ast.LetStmt)
	labelCall := nameLet.Value.(*ast.CallExpr)
	if info.ExprTypes[labelCall] != ast.StringType {
		t.Fatalf("label call type = %q, want %q", info.ExprTypes[labelCall], ast.StringType)
	}

	stringPrint := mainFn.Body.Statements[2].(*ast.ExprStmt).Expr.(*ast.CallExpr)
	wantStringPrint := []ast.Type{ast.StringType, ast.IntType}
	if !sameTypes(info.PrintCalls[stringPrint], wantStringPrint) {
		t.Fatalf("string print types = %q, want %q", info.PrintCalls[stringPrint], wantStringPrint)
	}

	boolPrint := mainFn.Body.Statements[3].(*ast.ExprStmt).Expr.(*ast.CallExpr)
	if !sameTypes(info.PrintCalls[boolPrint], []ast.Type{ast.BoolType}) {
		t.Fatalf("bool print types = %q, want %q", info.PrintCalls[boolPrint], []ast.Type{ast.BoolType})
	}
}

func TestCheckValidV2Program(t *testing.T) {
	program := mustParse(t, `package main
func more(value float) float {
    return value + 1.25
}

func main() int {
    let text string = "trux compiler"
    let ok bool = "compiler" in text
    let total float = more(1.5)
    let i int = 0

    if ok {
        total = total * 2.0
    } else {
        total = 0.0
    }

    while i < 3 {
        i = i + 1
    }

    print(total, i == 3, text != "other")
    return 0
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}

	mainFn := program.Functions[1]
	okLet := mainFn.Body.Statements[1].(*ast.LetStmt)
	if info.ExprTypes[okLet.Value] != ast.BoolType {
		t.Fatalf("in expression type = %q, want bool", info.ExprTypes[okLet.Value])
	}

	totalLet := mainFn.Body.Statements[2].(*ast.LetStmt)
	if info.ExprTypes[totalLet.Value] != ast.FloatType {
		t.Fatalf("float call type = %q, want float", info.ExprTypes[totalLet.Value])
	}

	printCall := mainFn.Body.Statements[6].(*ast.ExprStmt).Expr.(*ast.CallExpr)
	if !sameTypes(info.PrintCalls[printCall], []ast.Type{ast.FloatType, ast.BoolType, ast.BoolType}) {
		t.Fatalf("print call types = %q, want float bool bool", info.PrintCalls[printCall])
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
			want: `print expects at least 1 argument, got 0`,
		},
		{
			name: "print expression",
			src: `package main
func main() int {
    return print(1)
}`,
			want: `print can only be used as a statement`,
		},
		{
			name: "assign string to int",
			src: `package main
func main() int {
    let x int = "bad"
    return x
}`,
			want: `cannot assign string to int`,
		},
		{
			name: "assign int to string",
			src: `package main
func main() int {
    let x string = 1
    return 0
}`,
			want: `cannot assign int to string`,
		},
		{
			name: "assign bool to int",
			src: `package main
func main() int {
    let x int = true
    return x
}`,
			want: `cannot assign bool to int`,
		},
		{
			name: "return wrong type",
			src: `package main
func bad() bool {
    return "bad"
}
func main() int {
    return 0
}`,
			want: `cannot return string from function returning bool`,
		},
		{
			name: "wrong argument type",
			src: `package main
func takesString(value string) int {
    return 0
}
func main() int {
    return takesString(true)
}`,
			want: `argument 1 to takesString has type bool, want string`,
		},
		{
			name: "string arithmetic",
			src: `package main
func main() int {
    let x int = "a" + "b"
    return x
}`,
			want: `operator "+" requires matching numeric operands, got string and string`,
		},
		{
			name: "bool arithmetic",
			src: `package main
func main() int {
    let x int = true + false
    return x
}`,
			want: `operator "+" requires matching numeric operands, got bool and bool`,
		},
		{
			name: "if condition not bool",
			src: `package main
func main() int {
    if 1 {
        print("bad")
    }
    return 0
}`,
			want: `if condition must be bool, got int`,
		},
		{
			name: "while condition not bool",
			src: `package main
func main() int {
    while "bad" {
        print("bad")
    }
    return 0
}`,
			want: `while condition must be bool, got string`,
		},
		{
			name: "assignment undefined variable",
			src: `package main
func main() int {
    missing = 1
    return 0
}`,
			want: `undefined variable "missing"`,
		},
		{
			name: "assignment wrong type",
			src: `package main
func main() int {
    let x float = 1.0
    x = 1
    return 0
}`,
			want: `cannot assign int to float`,
		},
		{
			name: "mixed numeric arithmetic",
			src: `package main
func main() int {
    let x float = 1.0 + 1
    return 0
}`,
			want: `operator "+" requires matching numeric operands, got float and int`,
		},
		{
			name: "mixed numeric comparison",
			src: `package main
func main() int {
    let ok bool = 1.0 == 1
    return 0
}`,
			want: `operator "==" requires matching comparable operands, got float and int`,
		},
		{
			name: "string in wrong type",
			src: `package main
func main() int {
    let ok bool = "x" in 1
    return 0
}`,
			want: `operator "in" requires string operands, got string and int`,
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

func sameTypes(got []ast.Type, want []ast.Type) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func mustParse(t *testing.T, src string) *ast.Program {
	t.Helper()

	program, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	return program
}
