package types

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/modules"
	"github.com/claudioscheer/trux/internal/parser"
)

func TestCheckValidIntegerProgram(t *testing.T) {
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

func TestCheckValidPrimitiveProgram(t *testing.T) {
	program := mustParse(t, `package main
func label() string {
    return "trux"
}

func ready() bool {
    return true
}

func main() int {
    let name string = label()
    let ok bool = ready()
    print(name, " ", 1)
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
	wantStringPrint := []ast.Type{ast.StringType, ast.StringType, ast.IntType}
	if !sameTypes(info.PrintCalls[stringPrint], wantStringPrint) {
		t.Fatalf("string print types = %q, want %q", info.PrintCalls[stringPrint], wantStringPrint)
	}

	boolPrint := mainFn.Body.Statements[3].(*ast.ExprStmt).Expr.(*ast.CallExpr)
	if !sameTypes(info.PrintCalls[boolPrint], []ast.Type{ast.BoolType}) {
		t.Fatalf("bool print types = %q, want %q", info.PrintCalls[boolPrint], []ast.Type{ast.BoolType})
	}
}

func TestCheckValidControlFlowProgram(t *testing.T) {
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
    } else if i == 0 {
        total = total + 1.0
    } else {
        total = 0.0
    }

    for i < 3 {
        i = i + 1
    }

    print(total, " ", i == 3, " ", text != "other")
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
	if !sameTypes(info.PrintCalls[printCall], []ast.Type{ast.FloatType, ast.StringType, ast.BoolType, ast.StringType, ast.BoolType}) {
		t.Fatalf("print call types = %q, want float string bool string bool", info.PrintCalls[printCall])
	}
}

func TestCheckAllowsCStyleForLoopScopedLocal(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    let sum int = 0
    for let i int = 0; i < 3; i = i + 1 {
        sum = sum + i
    }
    for let i int = 0; i < 2; i = i + 1 {
        sum = sum + i
    }
    return sum
}`)

	if _, err := Check(program); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsStringConcatenation(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    let first string = "tru"
    let second string = "x"
    let name string = first + second
    print(name)
    return 0
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}

	mainFn := program.Functions[0]
	nameLet := mainFn.Body.Statements[2].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[nameLet.Value], ast.StringType) {
		t.Fatalf("concat type = %q, want string", info.ExprTypes[nameLet.Value])
	}
}

func TestCheckAllowsCollections(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = xs[1:]
    ys[0] = 9
    let zs list[int] = list[int]{xs[1]}
    append(zs, ys[0])
    let made []int = make([]int, len(zs))
    made[0] = zs[1]
    let ch string = "abc"[1]
    let sub string = "abcd"[1:3]
    let rows list[list[int]] = list[list[int]]{list[int]{1, 2}, list[int]{3}}
    append(rows, list[int]{4, 5, 6})
    rows[1] = list[int]{7}
    let fixed [2][2]int = [2][2]int{[2]int{1, 2}, [2]int{3, 4}}
    let jagged [][]float = make([][]float, 2)
    jagged[0] = make([]float, 1)
    jagged[0][0] = 1.5
    print(len(xs), " ", len(ys), " ", len(zs), " ", xs[1], " ", made[0], " ", ch, " ", sub, " ", len(rows[0]), " ", fixed[1][1], " ", jagged[0][0])
    return 0
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}

	mainFn := program.Functions[0]
	sliceLet := mainFn.Body.Statements[1].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[sliceLet.Value], &ast.SliceType{Elem: ast.IntType}) {
		t.Fatalf("slice expression type = %s, want []int", info.ExprTypes[sliceLet.Value])
	}
	listLet := mainFn.Body.Statements[3].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[listLet.Value], &ast.ListType{Elem: ast.IntType}) {
		t.Fatalf("list literal type = %s, want list[int]", info.ExprTypes[listLet.Value])
	}
	appendCall := mainFn.Body.Statements[4].(*ast.ExprStmt).Expr.(*ast.CallExpr)
	if !ast.TypeEqual(info.AppendCalls[appendCall].ElemType, ast.IntType) {
		t.Fatalf("append elem type = %s, want int", info.AppendCalls[appendCall].ElemType)
	}
	chLet := mainFn.Body.Statements[7].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[chLet.Value], ast.StringType) {
		t.Fatalf("string index type = %s, want string", info.ExprTypes[chLet.Value])
	}
	rowsLet := mainFn.Body.Statements[9].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[rowsLet.Value], &ast.ListType{Elem: &ast.ListType{Elem: ast.IntType}}) {
		t.Fatalf("nested list literal type = %s, want list[list[int]]", info.ExprTypes[rowsLet.Value])
	}
	fixedLet := mainFn.Body.Statements[12].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[fixedLet.Value], &ast.ArrayType{Length: 2, Elem: &ast.ArrayType{Length: 2, Elem: ast.IntType}}) {
		t.Fatalf("nested array literal type = %s, want [2][2]int", info.ExprTypes[fixedLet.Value])
	}
	jaggedLet := mainFn.Body.Statements[13].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[jaggedLet.Value], &ast.SliceType{Elem: &ast.SliceType{Elem: ast.FloatType}}) {
		t.Fatalf("nested slice make type = %s, want [][]float", info.ExprTypes[jaggedLet.Value])
	}
}

func TestCheckRejectsParameterOwnedCollectionMutation(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "append list parameter",
			src: `package main
func push(xs list[int], value int) int {
    append(xs, value)
    return len(xs)
}
func main() int {
    return 0
}`,
		},
		{
			name: "append list parameter alias",
			src: `package main
func push(xs list[int], value int) int {
    let alias list[int] = xs
    append(alias, value)
    return len(alias)
}
func main() int {
    return 0
}`,
		},
		{
			name: "assign through slice parameter",
			src: `package main
func write(xs []int) int {
    xs[0] = 1
    return xs[0]
}
func main() int {
    return 0
}`,
		},
		{
			name: "assign through parameter slice view",
			src: `package main
func write(xs []int) int {
    let view []int = xs[:]
    view[0] = 1
    return view[0]
}
func main() int {
    return 0
}`,
		},
		{
			name: "append nested list parameter element",
			src: `package main
func push(xs list[list[int]], value int) int {
    append(xs[0], value)
    return len(xs[0])
}
func main() int {
    return 0
}`,
		},
		{
			name: "assign through nested slice parameter element",
			src: `package main
func write(xs [][]int) int {
    xs[0][0] = 1
    return xs[0][0]
}
func main() int {
    return 0
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := mustParse(t, tt.src)
			_, err := Check(program)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "cannot mutate parameter-owned collection") {
				t.Fatalf("error = %q, want parameter-owned collection mutation error", err.Error())
			}
		})
	}
}

func TestCheckAllowsMutableParameterMutation(t *testing.T) {
	program := mustParse(t, `package main
func push(mut xs list[int], value int) int {
    append(xs, value)
    let alias list[int] = xs
    append(alias, value + 1)
    return len(xs)
}

func write(mut xs []int) int {
    xs[0] = 1
    let view []int = xs[:]
    view[1] = 2
    return view[1]
}

func pushNested(mut rows list[list[int]], value int) int {
    append(rows[0], value)
    return len(rows[0])
}

func main() int {
    let items list[int] = list[int]{}
    let rows list[list[int]] = list[list[int]]{list[int]{}}
    let xs [2]int = [2]int{0, 0}
    print(push(items, 1), " ", write(xs[:]), " ", pushNested(rows, 3))
    return 0
}`)

	if _, err := Check(program); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsInvalidMutableParameters(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "scalar",
			src: `package main
func write(mut value int) int {
    return value
}
func main() int {
    return 0
}`,
			want: `mut parameter "value" must be array, slice, or list, got int`,
		},
		{
			name: "string",
			src: `package main
func write(mut value string) string {
    return value
}
func main() int {
    return 0
}`,
			want: `mut parameter "value" must be array, slice, or list, got string`,
		},
		{
			name: "kernel",
			src: `package main
import "gpu"

kernel func fill(mut out gpu.Buffer[int], n int) {
    let i int = gpu.globalX()
    if i < n {
        out[i] = 1
    }
}

func main() int {
    return 0
}`,
			want: `kernel parameter "out" cannot be mut`,
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
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestCheckRejectsNonMutableArgumentsForMutableParameters(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "borrowed parameter",
			src: `package main
func push(mut xs list[int], value int) int {
    append(xs, value)
    return len(xs)
}

func outer(xs list[int]) int {
    return push(xs, 1)
}

func main() int {
    return 0
}`,
			want: `cannot pass borrowed collection to mut parameter "xs"`,
		},
		{
			name: "unknown call result",
			src: `package main
func push(mut xs list[int], value int) int {
    append(xs, value)
    return len(xs)
}

func build() list[int] {
    let xs list[int] = list[int]{}
    return xs
}

func main() int {
    return push(build(), 1)
}`,
			want: `cannot pass collection with unknown ownership to mut parameter "xs"`,
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
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestCheckAllowsLocalCollectionMutationInFunction(t *testing.T) {
	program := mustParse(t, `package main
func build(value int) int {
    let xs list[int] = list[int]{}
    append(xs, value)
    xs[0] = xs[0] + 1
    return xs[0]
}
func main() int {
    return build(1)
}`)

	if _, err := Check(program); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAllowsCloneForDynamicValues(t *testing.T) {
	program := mustParse(t, `package main
func id(xs []int) []int {
    return xs
}
func main() int {
    let name string = clone("trux")
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = clone(xs[:])
    let zs []int = clone(id(xs[:]))
    let items list[string] = clone(list[string]{name})
    let rows list[list[int]] = clone(list[list[int]]{list[int]{1, 2}, list[int]{3}})
    ys[0] = 9
    zs[0] = 8
    append(items, "x")
    append(rows[0], 4)
    return ys[0]
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}
	mainFn := program.Functions[1]
	nameClone := mainFn.Body.Statements[0].(*ast.LetStmt).Value.(*ast.CallExpr)
	if !ast.TypeEqual(info.CloneCalls[nameClone].Type, ast.StringType) {
		t.Fatalf("clone type = %s, want string", info.CloneCalls[nameClone].Type)
	}
	if info.ExprOrigins[nameClone] != OriginFrameOwned {
		t.Fatalf("clone origin = %s, want %s", info.ExprOrigins[nameClone], OriginFrameOwned)
	}
	sliceClone := mainFn.Body.Statements[2].(*ast.LetStmt).Value.(*ast.CallExpr)
	if !ast.TypeEqual(info.CloneCalls[sliceClone].Type, &ast.SliceType{Elem: ast.IntType}) {
		t.Fatalf("slice clone type = %s, want []int", info.CloneCalls[sliceClone].Type)
	}
	if info.ExprOrigins[sliceClone] != OriginFrameOwned {
		t.Fatalf("slice clone origin = %s, want %s", info.ExprOrigins[sliceClone], OriginFrameOwned)
	}
	rowsClone := mainFn.Body.Statements[5].(*ast.LetStmt).Value.(*ast.CallExpr)
	if !ast.TypeEqual(info.CloneCalls[rowsClone].Type, &ast.ListType{Elem: &ast.ListType{Elem: ast.IntType}}) {
		t.Fatalf("nested clone type = %s, want list[list[int]]", info.CloneCalls[rowsClone].Type)
	}
	if info.ExprOrigins[rowsClone] != OriginFrameOwned {
		t.Fatalf("nested clone origin = %s, want %s", info.ExprOrigins[rowsClone], OriginFrameOwned)
	}
}

func TestCheckAllowsStandardIOPackages(t *testing.T) {
	program := mustLoadProgram(t, `package main
import "io"
import "csv"

func main() int {
    let line string = io.readLine()
    let count int = io.readInt()
    let ratio float = io.readFloat()
    let ready bool = io.readBool()
    let text string = io.readFile("input.txt")
    io.writeFile("out.txt", text + line)
    let cells list[string] = csv.read("in.csv", 2)
    csv.write("out.csv", cells, 2)
    print(line, " ", count, " ", ratio, " ", ready, " ", len(cells))
    return 0
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}

	kinds := map[IOCallKind]int{}
	for _, sig := range info.IOCalls {
		kinds[sig.Kind]++
	}
	for _, kind := range []IOCallKind{
		IOCallReadLine,
		IOCallReadInt,
		IOCallReadFloat,
		IOCallReadBool,
		IOCallReadFile,
		IOCallWriteFile,
		IOCallReadCSV,
		IOCallWriteCSV,
	} {
		if kinds[kind] != 1 {
			t.Fatalf("IO call count for %s = %d, want 1", kind, kinds[kind])
		}
	}

	mainFn := program.Functions[0]
	readCsvLet := mainFn.Body.Statements[6].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[readCsvLet.Value], &ast.ListType{Elem: ast.StringType}) {
		t.Fatalf("readCsv type = %s, want list[string]", info.ExprTypes[readCsvLet.Value])
	}
	if info.ExprOrigins[readCsvLet.Value] != OriginFrameOwned {
		t.Fatalf("readCsv origin = %s, want %s", info.ExprOrigins[readCsvLet.Value], OriginFrameOwned)
	}
}

func TestCheckAllowsStandardImagePackage(t *testing.T) {
	program := mustLoadProgram(t, `package main
import "image"

func main() int {
    let width int = image.width("in.ppm")
    let height int = image.height("in.ppm")
    let pixels []int = image.readPpm("in.ppm")
    image.writePpm("out.ppm", pixels, width, height)
    print(width, " ", height, " ", len(pixels))
    return 0
}`)

	info, err := Check(program)
	if err != nil {
		t.Fatal(err)
	}

	kinds := map[IOCallKind]int{}
	for _, sig := range info.IOCalls {
		kinds[sig.Kind]++
	}
	for _, kind := range []IOCallKind{
		IOCallImageWidth,
		IOCallImageHeight,
		IOCallReadPPM,
		IOCallWritePPM,
	} {
		if kinds[kind] != 1 {
			t.Fatalf("image call count for %s = %d, want 1", kind, kinds[kind])
		}
	}

	mainFn := program.Functions[0]
	readPPMLet := mainFn.Body.Statements[2].(*ast.LetStmt)
	if !ast.TypeEqual(info.ExprTypes[readPPMLet.Value], &ast.SliceType{Elem: ast.IntType}) {
		t.Fatalf("readPpm type = %s, want []int", info.ExprTypes[readPPMLet.Value])
	}
	if info.ExprOrigins[readPPMLet.Value] != OriginFrameOwned {
		t.Fatalf("readPpm origin = %s, want %s", info.ExprOrigins[readPPMLet.Value], OriginFrameOwned)
	}
}

func TestCheckRejectsUnsupportedClone(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    return clone(1)
}`)

	_, err := Check(program)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "clone does not support int") {
		t.Fatalf("error = %q, want unsupported clone", err.Error())
	}
}

func TestCheckRejectsUnknownOwnershipMutation(t *testing.T) {
	program := mustParse(t, `package main
func id(xs []int) []int {
    return xs
}
func main() int {
    let xs [2]int = [2]int{1, 2}
    let ys []int = id(xs[:])
    ys[0] = 9
    return ys[0]
}`)

	_, err := Check(program)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot mutate collection with unknown ownership") {
		t.Fatalf("error = %q, want unknown ownership mutation error", err.Error())
	}
}

func TestCheckRejectsBlockLocalEscape(t *testing.T) {
	program := mustParse(t, `package main
func main() int {
    if true {
        let x int = 1
    }
    return x
}`)

	_, err := Check(program)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `undefined variable "x"`) {
		t.Fatalf("error = %q, want undefined block local", err.Error())
	}
}

func TestCheckRejectsSemanticErrors(t *testing.T) {
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
			name: "mixed string concatenation",
			src: `package main
func main() int {
    let x string = "a" + 1
    return 0
}`,
			want: `operator "+" requires matching numeric operands or string operands, got string and int`,
		},
		{
			name: "bool arithmetic",
			src: `package main
func main() int {
    let x int = true + false
    return x
}`,
			want: `operator "+" requires matching numeric operands or string operands, got bool and bool`,
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
			name: "for condition not bool",
			src: `package main
func main() int {
    for "bad" {
        print("bad")
    }
    return 0
}`,
			want: `for condition must be bool, got string`,
		},
		{
			name: "for init local outside loop",
			src: `package main
func main() int {
    for let i int = 0; i < 3; i = i + 1 {
        print(i)
    }
    return i
}`,
			want: `undefined variable "i"`,
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
			want: `operator "+" requires matching numeric operands or string operands, got float and int`,
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
		{
			name: "array literal length mismatch",
			src: `package main
func main() int {
    let xs [2]int = [2]int{1}
    return 0
}`,
			want: `array literal for [2]int has 1 elements, want 2`,
		},
		{
			name: "invalid index type",
			src: `package main
func main() int {
    let xs [1]int = [1]int{1}
    return xs[true]
}`,
			want: `index must be int, got bool`,
		},
		{
			name: "invalid slice bound type",
			src: `package main
func main() int {
    let xs [1]int = [1]int{1}
    let ys []int = xs["bad":]
    return 0
}`,
			want: `slice start must be int, got string`,
		},
		{
			name: "append wrong value type",
			src: `package main
func main() int {
    let xs list[int] = list[int]{}
    append(xs, "bad")
    return 0
}`,
			want: `append value has type string, want int`,
		},
		{
			name: "append expression",
			src: `package main
func main() int {
    let xs list[int] = list[int]{}
    return append(xs, 1)
}`,
			want: `append can only be used as a statement`,
		},
		{
			name: "string index assignment",
			src: `package main
func main() int {
    let text string = "abc"
    text[0] = "x"
    return 0
}`,
			want: `cannot assign through string index`,
		},
		{
			name: "make non-slice",
			src: `package main
func main() int {
    let x int = make(int, 1)
    return x
}`,
			want: `make expects slice type, got int`,
		},
		{
			name: "readLine arity",
			src: `package main
import "io"

func main() int {
    let line string = io.readLine("prompt")
    return 0
}`,
			want: `io.readLine expects 0 arguments, got 1`,
		},
		{
			name: "readFile path type",
			src: `package main
import "io"

func main() int {
    let text string = io.readFile(1)
    return 0
}`,
			want: `io.readFile path has type int, want string`,
		},
		{
			name: "writeFile expression",
			src: `package main
import "io"

func main() int {
    return io.writeFile("out.txt", "contents")
}`,
			want: `io.writeFile can only be used as a statement`,
		},
		{
			name: "readCsv columns type",
			src: `package main
import "csv"

func main() int {
    let cells list[string] = csv.read("in.csv", "2")
    return 0
}`,
			want: `csv.read columns has type string, want int`,
		},
		{
			name: "writeCsv cells type",
			src: `package main
import "csv"

func main() int {
    let cells list[int] = list[int]{1}
    csv.write("out.csv", cells, 1)
    return 0
}`,
			want: `csv.write cells has type list[int], want list[string]`,
		},
		{
			name: "legacy io builtin",
			src: `package main
func main() int {
    let text string = readFile("in.txt")
    return len(text)
}`,
			want: `undefined function "readFile"; use io.readFile after import "io"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := mustParseOrLoadProgram(t, tt.src)
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
		if !ast.TypeEqual(got[i], want[i]) {
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

func mustLoadProgram(t *testing.T, src string) *ast.Program {
	t.Helper()

	path := filepath.Join(t.TempDir(), "main.tx")
	result, err := modules.LoadWithSources(path, map[string]string{path: src})
	if err != nil {
		t.Fatal(err)
	}
	return result.Program
}

func mustParseOrLoadProgram(t *testing.T, src string) *ast.Program {
	t.Helper()

	if strings.Contains(src, "\nimport ") {
		return mustLoadProgram(t, src)
	}
	return mustParse(t, src)
}
