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
		"typedef struct {",
		"rt_arena_block* blocks;",
		"} rt_arena;",
		"static void rt_arena_init(rt_arena* arena)",
		"static void rt_arena_deinit(rt_arena* arena)",
		"static void rt_print_int(int64_t value)",
		"int64_t trux_add(rt_arena* trux_arena, int64_t trux_v_1_a, int64_t trux_v_1_b);",
		"int64_t trux_main(rt_arena* trux_arena);",
		"int64_t trux_add(rt_arena* trux_arena, int64_t trux_v_1_a, int64_t trux_v_1_b) {",
		"return (trux_v_1_a + trux_v_1_b);",
		"int64_t trux_v_1_x = trux_add(trux_arena, 1, 2);",
		"rt_print_int(trux_v_1_x);",
		"rt_print_newline();",
		"int main(void) {",
		"rt_arena_init(&trux_arena);",
		"int64_t trux_exit_code = trux_main(&trux_arena);",
		"rt_arena_deinit(&trux_arena);",
		"return (int)trux_exit_code;",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForV1Program(t *testing.T) {
	program, err := parser.Parse(`package main
func label() string {
    return "Kern"
}

func ready() bool {
    return true
}

func main() int {
    let name string = label()
    let ok bool = false
    print(name, " ", 1)
    print(ok)
    print("quote: \"")
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
		"#include <stdbool.h>",
		"typedef struct {",
		"const uint8_t* data;",
		"size_t len;",
		"} rt_string;",
		"static void rt_print_string(rt_string value)",
		"static void rt_print_bool(bool value)",
		"rt_string trux_label(rt_arena* trux_arena);",
		"bool trux_ready(rt_arena* trux_arena);",
		"return (rt_string){(const uint8_t*)\"Kern\", 4};",
		"return true;",
		"rt_string trux_v_4_name = trux_label(trux_arena);",
		"bool trux_v_2_ok = false;",
		"rt_print_string(trux_v_4_name);",
		"rt_print_string((rt_string){(const uint8_t*)\" \", 1});",
		"rt_print_int(1);",
		"rt_print_newline();",
		"rt_print_bool(trux_v_2_ok);",
		"rt_print_string((rt_string){(const uint8_t*)\"quote: \\\"\", 8});",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForV2Program(t *testing.T) {
	program, err := parser.Parse(`package main
func main() int {
    let text string = "trux"
    let x float = 1.5
    if "ru" in text {
        x = x + 1.0
    } else {
        x = 0.0
    }
    while x > 0.0 {
        x = x - 1.0
    }
    print(x, " ", text == "trux")
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
		"static void rt_print_float(double value)",
		"static bool rt_string_contains(rt_string needle, rt_string haystack)",
		"static bool rt_string_equal(rt_string left, rt_string right)",
		"double trux_v_1_x = 1.5;",
		"if (rt_string_contains((rt_string){(const uint8_t*)\"ru\", 2}, trux_v_4_text)) {",
		"trux_v_1_x = (trux_v_1_x + 1.0);",
		"} else {",
		"trux_v_1_x = 0.0;",
		"while ((trux_v_1_x > 0.0)) {",
		"trux_v_1_x = (trux_v_1_x - 1.0);",
		"rt_print_float(trux_v_1_x);",
		"rt_print_bool(rt_string_equal(trux_v_4_text, (rt_string){(const uint8_t*)\"trux\", 4}));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForStringConcatenation(t *testing.T) {
	program, err := parser.Parse(`package main
func greet(name string) string {
    return "hello " + name
}

func main() int {
    let name string = greet("trux")
    print(name + "!")
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
		"static rt_string rt_string_concat(rt_arena* arena, rt_string left, rt_string right)",
		"rt_string trux_greet(rt_arena* trux_arena, rt_string trux_v_4_name);",
		"return rt_string_concat(trux_arena, (rt_string){(const uint8_t*)\"hello \", 6}, trux_v_4_name);",
		"rt_string trux_v_4_name = trux_greet(trux_arena, (rt_string){(const uint8_t*)\"trux\", 4});",
		"rt_print_string(rt_string_concat(trux_arena, trux_v_4_name, (rt_string){(const uint8_t*)\"!\", 1}));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForV3Collections(t *testing.T) {
	program, err := parser.Parse(`package main
func main() int {
    let xs [3]int = [3]int{1, 2, 3}
    let view []int = xs[1:]
    view[0] = 9
    let items list[int] = list[int]{xs[1]}
    append(items, 4)
    let made []int = make([]int, len(items))
    made[0] = items[1]
	print(xs[1], " ", view[0], " ", made[0], " ", "abc"[1], " ", "abcd"[1:3])
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
		"RT_DEFINE_COLLECTIONS(int, int64_t)",
		"rt_array_int trux_v_2_xs = rt_array_int_from_values(trux_arena, (int64_t[]){1, 2, 3}, 3);",
		"rt_slice_int trux_v_4_view = rt_array_int_slice(trux_v_2_xs, true, 1, false, 0);",
		"rt_slice_int_set(trux_v_4_view, 0, 9);",
		"rt_list_int* trux_v_5_items = rt_list_int_from_values(trux_arena, (int64_t[]){rt_array_int_get(trux_v_2_xs, 1)}, 1);",
		"rt_list_int_append(trux_v_5_items, 4);",
		"rt_slice_int trux_v_4_made = rt_make_slice_int(trux_arena, ((int64_t)trux_v_5_items->len));",
		"rt_slice_int_set(trux_v_4_made, 0, rt_list_int_get(trux_v_5_items, 1));",
		"rt_print_int(rt_array_int_get(trux_v_2_xs, 1));",
		"rt_print_string((rt_string){(const uint8_t*)\" \", 1});",
		"rt_print_string(rt_string_index((rt_string){(const uint8_t*)\"abc\", 3}, 1));",
		"rt_print_string(rt_string_slice((rt_string){(const uint8_t*)\"abcd\", 4}, true, 1, true, 3));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}
