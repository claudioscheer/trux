package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ast"
)

func TestParsesHelloExample(t *testing.T) {
	src, err := os.ReadFile("../../examples/hello.tx")
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

	stmt, ok := mainFn.Body.Statements[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("statement = %T, want ast.LetStmt", mainFn.Body.Statements[0])
	}
	if stmt.Name != "x" || stmt.Type != ast.IntType {
		t.Fatalf("let statement = %#v, want name x and int type", stmt)
	}

	call, ok := stmt.Value.(*ast.CallExpr)
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

	stmt := program.Functions[0].Body.Statements[0].(*ast.ReturnStmt)
	expr, ok := stmt.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("return value = %T, want ast.BinaryExpr", stmt.Value)
	}
	if expr.Operator != "+" {
		t.Fatalf("root operator = %q, want +", expr.Operator)
	}

	right, ok := expr.Right.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("right expression = %T, want ast.BinaryExpr", expr.Right)
	}
	if right.Operator != "*" {
		t.Fatalf("right operator = %q, want *", right.Operator)
	}
}

func TestParsesPrimitiveTypesAndLiterals(t *testing.T) {
	program := mustParse(t, `package main
func label() string {
    return "trux"
}

func ready() bool {
    return true
}

func main() int {
    let name string = label()
    let ok bool = false
    print(name)
    print(ok)
    return 0
}`)

	label := program.Functions[0]
	if label.ReturnType != ast.StringType {
		t.Fatalf("label return type = %q, want %q", label.ReturnType, ast.StringType)
	}
	stringReturn := label.Body.Statements[0].(*ast.ReturnStmt)
	stringLiteral, ok := stringReturn.Value.(*ast.StringLiteral)
	if !ok || stringLiteral.Value != "trux" {
		t.Fatalf("label return = %#v, want string literal trux", stringReturn.Value)
	}

	ready := program.Functions[1]
	if ready.ReturnType != ast.BoolType {
		t.Fatalf("ready return type = %q, want %q", ready.ReturnType, ast.BoolType)
	}
	boolReturn := ready.Body.Statements[0].(*ast.ReturnStmt)
	boolLiteral, ok := boolReturn.Value.(*ast.BoolLiteral)
	if !ok || !boolLiteral.Value {
		t.Fatalf("ready return = %#v, want true bool literal", boolReturn.Value)
	}

	mainFn := program.Functions[2]
	nameLet := mainFn.Body.Statements[0].(*ast.LetStmt)
	if nameLet.Type != ast.StringType {
		t.Fatalf("name type = %q, want %q", nameLet.Type, ast.StringType)
	}
	okLet := mainFn.Body.Statements[1].(*ast.LetStmt)
	if okLet.Type != ast.BoolType {
		t.Fatalf("ok type = %q, want %q", okLet.Type, ast.BoolType)
	}
}

func TestParsesControlFlowAssignmentFloatAndIn(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    let text string = "trux"
    let x float = 1.5
    if "ru" in text {
        x = x + 1.0
    } else {
        x = 0.0
    }
    while x >= 1.0 {
        x = x - 1.0
    }
    return 0
}`)

	mainFn := program.Functions[0]
	ifStmt, ok := mainFn.Body.Statements[2].(*ast.IfStmt)
	if !ok {
		t.Fatalf("statement = %T, want ast.IfStmt", mainFn.Body.Statements[2])
	}
	if _, ok := ifStmt.Condition.(*ast.BinaryExpr); !ok {
		t.Fatalf("if condition = %T, want ast.BinaryExpr", ifStmt.Condition)
	}
	if len(ifStmt.Then.Statements) != 1 || len(ifStmt.Else.Statements) != 1 {
		t.Fatalf("if branches = %#v/%#v, want one statement each", ifStmt.Then, ifStmt.Else)
	}
	if _, ok := ifStmt.Then.Statements[0].(*ast.AssignStmt); !ok {
		t.Fatalf("then statement = %T, want ast.AssignStmt", ifStmt.Then.Statements[0])
	}

	whileStmt, ok := mainFn.Body.Statements[3].(*ast.WhileStmt)
	if !ok {
		t.Fatalf("statement = %T, want ast.WhileStmt", mainFn.Body.Statements[3])
	}
	if _, ok := whileStmt.Condition.(*ast.BinaryExpr); !ok {
		t.Fatalf("while condition = %T, want ast.BinaryExpr", whileStmt.Condition)
	}
}

func TestParsesCollections(t *testing.T) {
	program := mustParse(t, `package main
func head(xs []int) int {
    return xs[0]
}

func main() int {
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = xs[1:]
    let zs list[int] = list[int]{head(ys), xs[:2][1]}
    let made []int = make([]int, len(xs))
    xs[0] = head(made)
    append(zs, xs[0])
    return head(ys)
}`)

	head := program.Functions[0]
	if !ast.TypeEqual(head.Params[0].Type, &ast.SliceType{Elem: ast.IntType}) {
		t.Fatalf("head param type = %s, want []int", head.Params[0].Type)
	}

	mainFn := program.Functions[1]
	arrayLet := mainFn.Body.Statements[0].(*ast.LetStmt)
	if !ast.TypeEqual(arrayLet.Type, &ast.ArrayType{Length: 3, Elem: ast.IntType}) {
		t.Fatalf("array let type = %s, want [3]int", arrayLet.Type)
	}
	if _, ok := arrayLet.Value.(*ast.ArrayLiteral); !ok {
		t.Fatalf("array let value = %T, want ast.ArrayLiteral", arrayLet.Value)
	}

	sliceLet := mainFn.Body.Statements[1].(*ast.LetStmt)
	if !ast.TypeEqual(sliceLet.Type, &ast.SliceType{Elem: ast.IntType}) {
		t.Fatalf("slice let type = %s, want []int", sliceLet.Type)
	}
	if _, ok := sliceLet.Value.(*ast.SliceExpr); !ok {
		t.Fatalf("slice let value = %T, want ast.SliceExpr", sliceLet.Value)
	}

	listLet := mainFn.Body.Statements[2].(*ast.LetStmt)
	if !ast.TypeEqual(listLet.Type, &ast.ListType{Elem: ast.IntType}) {
		t.Fatalf("list let type = %s, want list[int]", listLet.Type)
	}
	if _, ok := listLet.Value.(*ast.ListLiteral); !ok {
		t.Fatalf("list let value = %T, want ast.ListLiteral", listLet.Value)
	}

	makeLet := mainFn.Body.Statements[3].(*ast.LetStmt)
	if _, ok := makeLet.Value.(*ast.MakeExpr); !ok {
		t.Fatalf("make let value = %T, want ast.MakeExpr", makeLet.Value)
	}
	if _, ok := mainFn.Body.Statements[4].(*ast.IndexAssignStmt); !ok {
		t.Fatalf("statement = %T, want ast.IndexAssignStmt", mainFn.Body.Statements[4])
	}
	appendStmt := mainFn.Body.Statements[5].(*ast.ExprStmt)
	if call, ok := appendStmt.Expr.(*ast.CallExpr); !ok || call.Callee != "append" {
		t.Fatalf("append statement = %#v, want append call", appendStmt.Expr)
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

func TestReportsStringLexErrors(t *testing.T) {
	_, err := Parse(`package main
func main() int {
    print("bad\x")
    return 0
}`)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), `unknown escape \x`) {
		t.Fatalf("error = %q, want unknown escape", err.Error())
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
