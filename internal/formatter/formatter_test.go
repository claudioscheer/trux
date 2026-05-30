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
