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
	if !sameTypes(printStmt.Types, []ast.Type{ast.IntType}) {
		t.Fatalf("print types = %q, want %q", printStmt.Types, []ast.Type{ast.IntType})
	}
	if len(printStmt.Args) != 1 || printStmt.Args[0].Type() != ast.IntType {
		t.Fatalf("print args = %#v, want one int arg", printStmt.Args)
	}
}

func TestBuildCreatesTypedIRForV1Program(t *testing.T) {
	program := mustParse(t, `package main
func label() string {
    return "Kern"
}

func ready() bool {
    return true
}

func main() int {
    let name string = label()
    let ok bool = false
    print(name, 1)
    print(ok)
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

	label := irProgram.Functions[0]
	if label.ReturnType != ast.StringType {
		t.Fatalf("label return type = %q, want %q", label.ReturnType, ast.StringType)
	}
	stringReturn := label.Body[0].(*ReturnStmt)
	stringLiteral := stringReturn.Value.(*StringLiteral)
	if stringLiteral.Value != "Kern" || stringLiteral.Type() != ast.StringType {
		t.Fatalf("string literal = %#v, want typed Kern", stringLiteral)
	}

	ready := irProgram.Functions[1]
	boolReturn := ready.Body[0].(*ReturnStmt)
	boolLiteral := boolReturn.Value.(*BoolLiteral)
	if !boolLiteral.Value || boolLiteral.Type() != ast.BoolType {
		t.Fatalf("bool literal = %#v, want typed true", boolLiteral)
	}

	mainFn := irProgram.Functions[2]
	nameLet := mainFn.Body[0].(*LetStmt)
	nameCall := nameLet.Value.(*CallExpr)
	if nameLet.Type != ast.StringType || nameCall.Type() != ast.StringType {
		t.Fatalf("name let/call = %#v/%#v, want string", nameLet, nameCall)
	}

	stringPrint := mainFn.Body[2].(*PrintStmt)
	if !sameTypes(stringPrint.Types, []ast.Type{ast.StringType, ast.IntType}) {
		t.Fatalf("string print types = %q, want string and int", stringPrint.Types)
	}
	if len(stringPrint.Args) != 2 || stringPrint.Args[0].Type() != ast.StringType || stringPrint.Args[1].Type() != ast.IntType {
		t.Fatalf("string print args = %#v, want string and int", stringPrint.Args)
	}

	boolPrint := mainFn.Body[3].(*PrintStmt)
	if !sameTypes(boolPrint.Types, []ast.Type{ast.BoolType}) || len(boolPrint.Args) != 1 || boolPrint.Args[0].Type() != ast.BoolType {
		t.Fatalf("bool print = %#v, want bool print", boolPrint)
	}
}

func TestBuildCreatesTypedIRForV2Program(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    let text string = "trux"
    let x float = 1.5
    if "ru" in text {
        x = x + 1.0
    }
    while x > 0.0 {
        x = x - 1.0
    }
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

	mainFn := irProgram.Functions[0]
	floatLet := mainFn.Body[1].(*LetStmt)
	if floatLet.Type != ast.FloatType || floatLet.Value.Type() != ast.FloatType {
		t.Fatalf("float let = %#v, want float", floatLet)
	}

	ifStmt := mainFn.Body[2].(*IfStmt)
	if ifStmt.Condition.Type() != ast.BoolType {
		t.Fatalf("if condition type = %q, want bool", ifStmt.Condition.Type())
	}
	if _, ok := ifStmt.Then[0].(*AssignStmt); !ok {
		t.Fatalf("then statement = %T, want AssignStmt", ifStmt.Then[0])
	}

	whileStmt := mainFn.Body[3].(*WhileStmt)
	if whileStmt.Condition.Type() != ast.BoolType {
		t.Fatalf("while condition type = %q, want bool", whileStmt.Condition.Type())
	}

	printStmt := mainFn.Body[4].(*PrintStmt)
	if !sameTypes(printStmt.Types, []ast.Type{ast.FloatType}) {
		t.Fatalf("print types = %q, want float", printStmt.Types)
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
