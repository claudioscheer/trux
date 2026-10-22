package formatter

import (
	"strings"
	"testing"
)

func TestFormatNormalizesWhitespaceAndIndentation(t *testing.T) {
	src := `package   main
import   "math.tx"

pub   func add( a   int,b int)int{
let x int= a+b
if x>2{
print( "big" , x)
}else if x==2{
print("exact")
}else{
for x<10{
x=x+1
}
}
return x
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

import "math.tx"

pub func add(a int, b int) int {
  let x int = a + b
  if x > 2 {
    print("big", x)
  } else if x == 2 {
    print("exact")
  } else {
    for x < 10 {
      x = x + 1
    }
  }
  return x
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatKeepsBreakAndContinue(t *testing.T) {
	got, err := Format("main.tx", "package main\nfunc main() int {\nfor {break\ncontinue\n}\nreturn 0\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "break") || !strings.Contains(got, "continue") {
		t.Fatalf("formatted = %q, want break and continue", got)
	}
}

func TestFormatKeepsLogicalOperators(t *testing.T) {
	got, err := Format("main.tx", "package main\nfunc main() int {\nlet flag bool=true&&false||true\nreturn 0\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "let flag bool = true && false || true") {
		t.Fatalf("formatted = %q, want spaced logical operators", got)
	}
}

func TestFormatKeepsRemainder(t *testing.T) {
	got, err := Format("main.tx", "package main\nfunc main() int {\nlet n int=10%3\nreturn n\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "let n int = 10 % 3") {
		t.Fatalf("formatted = %q, want spaced remainder", got)
	}
}

func TestFormatKeepsPrefixOperatorsTight(t *testing.T) {
	src := "package main\nfunc main() int {\nlet n int=-1\nlet flag bool=!false\nreturn 0\n}\n"
	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "let n int = -1") {
		t.Fatalf("formatted = %q, want tight prefix minus", got)
	}
	if !strings.Contains(got, "let flag bool = !false") {
		t.Fatalf("formatted = %q, want tight boolean not", got)
	}
}

func TestFormatNormalizesCStyleFor(t *testing.T) {
	src := `package main
func main() int{
for let i int=0;i<10;i=i+1{
print(i)
}
return 0
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

func main() int {
  for let i int = 0; i < 10; i = i + 1 {
    print(i)
  }
  return 0
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatNormalizesMutableParameters(t *testing.T) {
	src := `package main
func push(mut   xs list[int],value int)int{
append(xs,value)
return len(xs)
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

func push(mut xs list[int], value int) int {
  append(xs, value)
  return len(xs)
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatKeepsQualifiedCallsTight(t *testing.T) {
	src := `package main
import "math.tx"

func main() int{
print(math . add(3,4))
return 0
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

import "math.tx"

func main() int {
  print(math.add(3, 4))
  return 0
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatKeepsIndexAndSliceExpressionsTight(t *testing.T) {
	src := `package main
func mid(xs []int)[]int{
return xs [1:2]
}
func main() int{
let xs [3]int=[3]int{1,2,3}
let items list[int]=list[int]{ }
print(xs [0], " ", xs [2], " ", "abc" [1], " ", len(xs [:]))
return 0
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

func mid(xs []int) []int {
  return xs[1:2]
}
func main() int {
  let xs [3]int = [3]int{1, 2, 3}
  let items list[int] = list[int]{}
  print(xs[0], " ", xs[2], " ", "abc"[1], " ", len(xs[:]))
  return 0
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatKeepsCollectionLiteralBracesTight(t *testing.T) {
	src := `package main
func main() int{
let result [2][2]int=[2][2]int{[2]int{0,0},[2]int{0,0}}
let items list[int]=list[int]{1,2}
return result [0][0]+items [0]
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

func main() int {
  let result [2][2]int = [2][2]int{[2]int{0, 0}, [2]int{0, 0}}
  let items list[int] = list[int]{1, 2}
  return result[0][0] + items[0]
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatPreservesLineCommentsAndStringContent(t *testing.T) {
	src := `package main
// file comment
func main() int{
print("https://trux.test", "line\nquote: \"") // trailing comment
return 0
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

// file comment
func main() int {
  print("https://trux.test", "line\nquote: \"") // trailing comment
  return 0
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatNormalizesPackageAndImportSpacing(t *testing.T) {
	src := `package   math
import   "double.tx"
pub   func add(a int,b int)int{
return a+b
}`

	got, err := Format("math.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package math

import "double.tx"

pub func add(a int, b int) int {
  return a + b
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatGroupsImports(t *testing.T) {
	src := `package main
import "io"

import "csv"
func main() int{
return 0
}`

	got, err := Format("main.tx", src)
	if err != nil {
		t.Fatal(err)
	}

	want := `package main

import "io"
import "csv"

func main() int {
  return 0
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatDoesNotAddPackageGapAtEOF(t *testing.T) {
	got, err := Format("main.tx", "package main\n\n")
	if err != nil {
		t.Fatal(err)
	}

	want := "package main\n"
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatReturnsParseError(t *testing.T) {
	_, err := Format("bad.tx", `package main
func main( int {
    return 0
}`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `expected IDENT, got "int"`) {
		t.Fatalf("error = %q, want parse error", err.Error())
	}
}

func TestFormatMultiReturnAndErrors(t *testing.T) {
	src := `package main

import "errors"

func divide(a int, b int) (int, error) {
  if b == 0 {
    return 0, errors.new("division by zero")
  }
  return a / b, nil
}

func main() int {
  let result int, err error = divide(10, 2)
  let _, err2 error = divide(10, 0)
  return 0
}
`
	formatted, err := Format("test.tx", src)
	if err != nil {
		t.Fatal(err)
	}
	if formatted != src {
		t.Fatalf("unexpected formatting:\n%s\nwant:\n%s", formatted, src)
	}
}

func TestFormatStructsPointersAndMaps(t *testing.T) {
	src := `package main

pub type Point struct {
  x int
  y int
}

func clonePtr(p *Point) *Point {
  let out *Point = new(Point)
  *out = *p
  return out
}

func main() int {
  let p Point = Point{x: 1, y: 2}
  let ptr *Point = &p
  let m map[string]*Point = map[string]*Point{"origin": ptr}
  return m["origin"].x
}
`
	formatted, err := Format("test.tx", src)
	if err != nil {
		t.Fatal(err)
	}
	if formatted != src {
		t.Fatalf("unexpected formatting:\n%s\nwant:\n%s", formatted, src)
	}
}
