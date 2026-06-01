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
while x<10{
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
    while x < 10 {
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
