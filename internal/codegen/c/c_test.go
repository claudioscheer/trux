package c

import (
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ir"
	"github.com/claudioscheer/trux/internal/parser"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

func TestGenerateCreatesCForV0Program(t *testing.T) {
	program, err := parser.Parse(`package main
func add(a int, b int) int {
    return a + b
}

func main() int {
    let x int = add(1, 2)
    print(x)
    return 0
}`)
	if err != nil {
		t.Fatal(err)
	}
	info, err := semtypes.Check(program)
	if err != nil {
		t.Fatal(err)
	}
	typedIR, err := ir.Build(program, info)
	if err != nil {
		t.Fatal(err)
	}

	cSource, err := Generate(typedIR)
	if err != nil {
		t.Fatal(err)
	}

	wantParts := []string{
		"#include <stdint.h>",
		"static void rt_print_int(int64_t value)",
		"int64_t trux_add(int64_t a, int64_t b);",
		"int64_t trux_main(void);",
		"int64_t trux_add(int64_t a, int64_t b) {",
		"return (a + b);",
		"int64_t x = trux_add(1, 2);",
		"rt_print_int(x);",
		"int main(void) {",
		"return (int)trux_main();",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}
