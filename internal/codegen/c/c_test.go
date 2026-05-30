package c

import (
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ir"
	"github.com/claudioscheer/trux/internal/parser"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

func TestGenerateCreatesCForIntegerProgram(t *testing.T) {
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
		"rt_arena_chunk* chunks;",
		"} rt_arena;",
		"static RT_UNUSED void rt_arena_init(rt_arena* arena)",
		"static RT_UNUSED void rt_arena_reset(rt_arena* arena)",
		"static RT_UNUSED void rt_arena_deinit(rt_arena* arena)",
		"static RT_UNUSED rt_arena_mark rt_arena_mark_current(rt_arena* arena)",
		"static RT_UNUSED void rt_arena_rewind(rt_arena* arena, rt_arena_mark mark)",
		"static RT_UNUSED void rt_print_int(int64_t value)",
		"int64_t trux_add(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t trux_v_1_a, int64_t trux_v_1_b);",
		"int64_t trux_main(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"int64_t trux_add(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t trux_v_1_a, int64_t trux_v_1_b) {",
		"rt_arena trux_frame;",
		"rt_arena_init(&trux_frame);",
		"int64_t trux_return_value;",
		"trux_return_value = (trux_v_1_a + trux_v_1_b);",
		"goto trux_return;",
		"trux_return:",
		"rt_arena_deinit(&trux_frame);",
		"return trux_return_value;",
		"int64_t trux_v_1_x = trux_add(trux_ctx, &trux_frame, 1, 2);",
		"rt_print_int(trux_v_1_x);",
		"rt_print_newline();",
		"int main(void) {",
		"rt_arena_init(&trux_arena);",
		"rt_context trux_ctx = {&trux_arena};",
		"int64_t trux_exit_code = trux_main(&trux_ctx, &trux_arena);",
		"rt_arena_deinit(&trux_arena);",
		"return (int)trux_exit_code;",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForPrimitiveProgram(t *testing.T) {
	program, err := parser.Parse(`package main
func label() string {
    return "trux"
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
		"static RT_UNUSED void rt_print_string(rt_string value)",
		"static RT_UNUSED void rt_print_bool(bool value)",
		"rt_string trux_label(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"bool trux_ready(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"rt_string trux_return_value;",
		"trux_return_value = (rt_string){(const uint8_t*)\"trux\", 4};",
		"bool trux_return_value;",
		"trux_return_value = true;",
		"rt_string trux_v_4_name = trux_label(trux_ctx, &trux_frame);",
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

func TestGenerateCreatesCForControlFlowProgram(t *testing.T) {
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
		"static RT_UNUSED void rt_print_float(double value)",
		"static RT_UNUSED bool rt_string_contains(rt_string needle, rt_string haystack)",
		"static RT_UNUSED bool rt_string_equal(rt_string left, rt_string right)",
		"double trux_v_1_x = 1.5;",
		"if (rt_string_contains((rt_string){(const uint8_t*)\"ru\", 2}, trux_v_4_text)) {",
		"trux_v_1_x = (trux_v_1_x + 1.0);",
		"} else {",
		"trux_v_1_x = 0.0;",
		"while (trux_v_1_x > 0.0) {",
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
		"static RT_UNUSED rt_string rt_string_concat(rt_arena* arena, rt_string left, rt_string right)",
		"rt_string trux_greet(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_string trux_v_4_name);",
		"trux_return_value = rt_string_clone(trux_result_arena, rt_string_concat(&trux_frame, (rt_string){(const uint8_t*)\"hello \", 6}, trux_v_4_name));",
		"rt_string trux_v_4_name = trux_greet(trux_ctx, &trux_frame, (rt_string){(const uint8_t*)\"trux\", 4});",
		"rt_print_string(rt_string_concat(&trux_frame, trux_v_4_name, (rt_string){(const uint8_t*)\"!\", 1}));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateAllocatesReturnedStringLocalInResultArena(t *testing.T) {
	program, err := parser.Parse(`package main
func bang(name string) string {
    let s string = name + "!"
    return s
}

func main() int {
    print(bang("a"))
    print(bang("b"))
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
		"rt_string trux_bang(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_string trux_v_4_name);",
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, trux_v_4_name, (rt_string){(const uint8_t*)\"!\", 1});",
		"trux_return_value = rt_string_clone(trux_result_arena, trux_v_1_s);",
		"rt_print_string(trux_bang(trux_ctx, &trux_frame, (rt_string){(const uint8_t*)\"a\", 1}));",
		"rt_print_string(trux_bang(trux_ctx, &trux_frame, (rt_string){(const uint8_t*)\"b\", 1}));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateReturnsStringParameterAsIs(t *testing.T) {
	program, err := parser.Parse(`package main
func id(s string) string {
    return s
}

func main() int {
    print(id("x"))
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
		"trux_return_value = trux_v_1_s;",
		"rt_print_string(trux_id(trux_ctx, &trux_frame, (rt_string){(const uint8_t*)\"x\", 1}));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateAllocatesStringLocalUsedByReturnedCallInResultArena(t *testing.T) {
	program, err := parser.Parse(`package main
func id(s string) string {
    return s
}

func wrap(name string) string {
    let s string = name + "!"
    return id(s)
}

func main() int {
    print(wrap("x"))
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
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, trux_v_4_name, (rt_string){(const uint8_t*)\"!\", 1});",
		"trux_return_value = rt_string_clone(trux_result_arena, trux_id(trux_ctx, &trux_frame, trux_v_1_s));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateAllocatesStringLocalStoredInScratchCollectionInFrameArena(t *testing.T) {
	program, err := parser.Parse(`package main
func stash(name string) string {
    let items list[string] = list[string]{}
    let s string = name + "!"
    append(items, s)
    return "ok"
}

func main() int {
    print(stash("x"))
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
		"rt_list_string* trux_v_5_items = rt_list_string_from_values(&trux_frame, NULL, 0);",
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, trux_v_4_name, (rt_string){(const uint8_t*)\"!\", 1});",
		"rt_list_string_append(trux_v_5_items, trux_v_1_s);",
		"rt_print_string(trux_stash(trux_ctx, &trux_frame, (rt_string){(const uint8_t*)\"x\", 1}));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateAllocatesStringLocalUsedInScratchCollectionLiteralInFrameArena(t *testing.T) {
	program, err := parser.Parse(`package main
func main() int {
    let s string = "a" + "!"
    let items list[string] = list[string]{s}
    print(len(items))
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
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, (rt_string){(const uint8_t*)\"a\", 1}, (rt_string){(const uint8_t*)\"!\", 1});",
		"rt_list_string* trux_v_5_items = rt_list_string_from_values(&trux_frame, (rt_string[]){trux_v_1_s}, 1);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateAllocatesStringLocalPassedToCollectionReturningCallInFrameArena(t *testing.T) {
	program, err := parser.Parse(`package main
func wrap(s string) list[string] {
    return list[string]{s}
}

func main() int {
    let s string = "a" + "!"
    let items list[string] = wrap(s)
    print(len(items))
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
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, (rt_string){(const uint8_t*)\"a\", 1}, (rt_string){(const uint8_t*)\"!\", 1});",
		"rt_list_string* trux_v_5_items = trux_wrap(trux_ctx, &trux_frame, trux_v_1_s);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForCollections(t *testing.T) {
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
		"rt_array_int trux_v_2_xs = rt_array_int_from_values(&trux_frame, (int64_t[]){1, 2, 3}, 3);",
		"rt_slice_int trux_v_4_view = rt_array_int_slice(trux_v_2_xs, true, 1, false, 0);",
		"rt_slice_int_set(trux_v_4_view, 0, 9);",
		"rt_list_int* trux_v_5_items = rt_list_int_from_values(&trux_frame, (int64_t[]){rt_array_int_get(trux_v_2_xs, 1)}, 1);",
		"rt_list_int_append(trux_v_5_items, 4);",
		"rt_slice_int trux_v_4_made = rt_make_slice_int(&trux_frame, rt_checked_len_i64(trux_v_5_items->len));",
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

func TestGenerateCreatesCForNestedCollections(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func build() list[list[int]] {
    let rows list[list[int]] = list[list[int]]{list[int]{1, 2}, list[int]{3}}
    append(rows, list[int]{4, 5, 6})
    return rows
}

func main() int {
    let rows list[list[int]] = build()
    let copied list[list[int]] = clone(rows)
    append(copied[0], 9)
    let fixed [2][2]int = [2][2]int{[2]int{1, 2}, [2]int{3, 4}}
    let slices [][]float = make([][]float, 2)
    slices[0] = make([]float, 2)
    slices[0][1] = 2.5
    print(copied[0][2], " ", rows[2][2], " ", fixed[1][1], " ", slices[0][1])
    return 0
}`)

	wantParts := []string{
		"#define RT_CLONE_VALUE_list_int(ARENA, VALUE) rt_list_int_clone((ARENA), (VALUE))",
		"RT_DEFINE_COLLECTIONS(list_int, rt_list_int*)",
		"#define RT_CLONE_VALUE_array_int(ARENA, VALUE) rt_array_int_clone((ARENA), (VALUE))",
		"RT_DEFINE_COLLECTIONS(array_int, rt_array_int)",
		"#define RT_CLONE_VALUE_slice_float(ARENA, VALUE) rt_slice_float_clone((ARENA), (VALUE))",
		"RT_DEFINE_COLLECTIONS(slice_float, rt_slice_float)",
		"rt_list_list_int* trux_build(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"rt_list_list_int* trux_v_4_rows = rt_list_list_int_from_values(&trux_frame, (rt_list_int*[]){rt_list_int_from_values(&trux_frame, (int64_t[]){1, 2}, 2), rt_list_int_from_values(&trux_frame, (int64_t[]){3}, 1)}, 2);",
		"rt_list_list_int_append(trux_v_4_rows, rt_list_int_from_values(&trux_frame, (int64_t[]){4, 5, 6}, 3));",
		"trux_return_value = rt_list_list_int_clone(trux_result_arena, trux_v_4_rows);",
		"rt_list_list_int* trux_v_4_rows = trux_build(trux_ctx, &trux_frame);",
		"rt_list_list_int* trux_v_6_copied = rt_list_list_int_clone(&trux_frame, trux_v_4_rows);",
		"rt_list_int_append(rt_list_list_int_get(trux_v_6_copied, 0), 9);",
		"rt_array_array_int trux_v_5_fixed = rt_array_array_int_from_values(&trux_frame, (rt_array_int[]){rt_array_int_from_values(&trux_frame, (int64_t[]){1, 2}, 2), rt_array_int_from_values(&trux_frame, (int64_t[]){3, 4}, 2)}, 2);",
		"rt_slice_slice_float trux_v_6_slices = rt_make_slice_slice_float(&trux_frame, 2);",
		"rt_slice_slice_float_set(trux_v_6_slices, 0, rt_make_slice_float(&trux_frame, 2));",
		"rt_slice_float_set(rt_slice_slice_float_get(trux_v_6_slices, 0), 1, 2.5);",
		"rt_print_int(rt_list_int_get(rt_list_list_int_get(trux_v_6_copied, 0), 2));",
		"rt_print_int(rt_list_int_get(rt_list_list_int_get(trux_v_4_rows, 2), 2));",
		"rt_print_int(rt_array_int_get(rt_array_array_int_get(trux_v_5_fixed, 1), 1));",
		"rt_print_float(rt_slice_float_get(rt_slice_slice_float_get(trux_v_6_slices, 0), 1));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func mustGenerateC(t *testing.T, src string) string {
	t.Helper()

	program, err := parser.Parse(src)
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
	return cSource
}

func generatedFunctionBody(t *testing.T, cSource string, name string) string {
	t.Helper()

	marker := "trux_" + name + "("
	searchStart := 0
	for {
		idx := strings.Index(cSource[searchStart:], marker)
		if idx < 0 {
			t.Fatalf("generated C missing function %s:\n%s", name, cSource)
		}
		idx += searchStart
		lineStart := strings.LastIndex(cSource[:idx], "\n") + 1
		lineEnd := strings.Index(cSource[idx:], "\n")
		if lineEnd < 0 {
			lineEnd = len(cSource)
		} else {
			lineEnd += idx
		}
		line := cSource[lineStart:lineEnd]
		if strings.HasSuffix(line, " {") {
			open := strings.Index(cSource[lineStart:], "{") + lineStart
			depth := 0
			for i := open; i < len(cSource); i++ {
				switch cSource[i] {
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						return cSource[open+1 : i]
					}
				}
			}
			t.Fatalf("generated C function %s has no matching closing brace:\n%s", name, cSource)
		}
		searchStart = idx + len(marker)
	}
}

func TestGenerateOnlyEmitsVoidCastsForUnusedRuntimeParameters(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func id(x int) int {
    return x
}

func borrowedMiddle(xs []int) []int {
    return xs[1:2]
}

func ownedMiddle(xs []int) []int {
    return clone(xs[1:2])
}

func callsCtx(x int) int {
    return id(x)
}

func main() int {
    let xs [3]int = [3]int{1, 2, 3}
    let borrowed []int = borrowedMiddle(xs[:])
    let owned []int = ownedMiddle(xs[:])
    print(borrowed[0], " ", owned[0], " ", callsCtx(1))
    return 0
}`)

	borrowed := generatedFunctionBody(t, cSource, "borrowedMiddle")
	if !strings.Contains(borrowed, "(void)trux_ctx;") {
		t.Fatalf("borrowedMiddle should mark unused trux_ctx:\n%s", borrowed)
	}
	if !strings.Contains(borrowed, "(void)trux_result_arena;") {
		t.Fatalf("borrowedMiddle should mark unused trux_result_arena:\n%s", borrowed)
	}

	owned := generatedFunctionBody(t, cSource, "ownedMiddle")
	if !strings.Contains(owned, "(void)trux_ctx;") {
		t.Fatalf("ownedMiddle should mark unused trux_ctx:\n%s", owned)
	}
	if strings.Contains(owned, "(void)trux_result_arena;") {
		t.Fatalf("ownedMiddle should not mark used trux_result_arena unused:\n%s", owned)
	}

	callsCtx := generatedFunctionBody(t, cSource, "callsCtx")
	if strings.Contains(callsCtx, "(void)trux_ctx;") {
		t.Fatalf("callsCtx should not mark used trux_ctx unused:\n%s", callsCtx)
	}
	if !strings.Contains(callsCtx, "(void)trux_result_arena;") {
		t.Fatalf("callsCtx should mark unused trux_result_arena:\n%s", callsCtx)
	}
}

func TestGenerateCopiesScratchSliceReturnToResultArena(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func build() []int {
    let xs [3]int = [3]int{1, 2, 3}
    return xs[:]
}

func main() int {
    let xs []int = build()
    print(xs[0], " ", xs[2])
    return 0
}`)

	wantParts := []string{
		"rt_array_int trux_v_2_xs = rt_array_int_from_values(&trux_frame, (int64_t[]){1, 2, 3}, 3);",
		"trux_return_value = rt_slice_int_clone(trux_result_arena, rt_array_int_slice(trux_v_2_xs, false, 0, false, 0));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateKeepsParameterSliceReturnBorrowed(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func mid(xs []int) []int {
    return xs[1:2]
}

func main() int {
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = mid(xs[:])
    print(ys[0])
    return 0
}`)

	want := "trux_return_value = rt_slice_int_slice(trux_v_2_xs, true, 1, true, 2);"
	if !strings.Contains(cSource, want) {
		t.Fatalf("generated C missing borrowed return %q:\n%s", want, cSource)
	}
}

func TestGenerateClonesParameterSliceWhenRequested(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func midOwned(xs []int) []int {
    return clone(xs[1:2])
}

func main() int {
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = midOwned(xs[:])
    print(ys[0])
    return 0
}`)

	want := "trux_return_value = rt_slice_int_clone(trux_result_arena, rt_slice_int_slice(trux_v_2_xs, true, 1, true, 2));"
	if !strings.Contains(cSource, want) {
		t.Fatalf("generated C missing owned return %q:\n%s", want, cSource)
	}
}

func TestGenerateLocalCloneUsesFrameArenaAndCopiesOutOnReturn(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func ownedLocal() []int {
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = clone(xs[:])
    return ys
}

func main() int {
    let ys []int = ownedLocal()
    print(ys[0], " ", ys[2])
    return 0
}`)

	wantParts := []string{
		"rt_slice_int trux_v_2_ys = rt_slice_int_clone(&trux_frame, rt_array_int_slice(trux_v_2_xs, false, 0, false, 0));",
		"trux_return_value = rt_slice_int_clone(trux_result_arena, trux_v_2_ys);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCopiesMutatedFrameOwnedCloneOnReturn(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func edit(xs []int) []int {
    let ys []int = clone(xs[:])
    ys[0] = 42
    return ys
}

func main() int {
    let xs [2]int = [2]int{1, 2}
    let ys []int = edit(xs[:])
    print(ys[0])
    return 0
}`)

	wantParts := []string{
		"rt_slice_int trux_v_2_ys = rt_slice_int_clone(&trux_frame, rt_slice_int_slice(trux_v_2_xs, false, 0, false, 0));",
		"rt_slice_int_set(trux_v_2_ys, 0, 42);",
		"trux_return_value = rt_slice_int_clone(trux_result_arena, trux_v_2_ys);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCopiesScratchStringListReturnToResultArena(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func build() list[string] {
    let xs list[string] = list[string]{}
    append(xs, "a" + "b")
    return xs
}

func main() int {
    let xs list[string] = build()
    print(xs[0])
    return 0
}`)

	wantParts := []string{
		"rt_list_string* trux_v_2_xs = rt_list_string_from_values(&trux_frame, NULL, 0);",
		"rt_list_string_append(trux_v_2_xs, rt_string_concat(&trux_frame, (rt_string){(const uint8_t*)\"a\", 1}, (rt_string){(const uint8_t*)\"b\", 1}));",
		"trux_return_value = rt_list_string_clone(trux_result_arena, trux_v_2_xs);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}
