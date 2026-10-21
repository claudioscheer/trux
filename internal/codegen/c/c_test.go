package c

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ir"
	"github.com/claudioscheer/trux/internal/modules"
	"github.com/claudioscheer/trux/internal/parser"
	runtimec "github.com/claudioscheer/trux/internal/runtime/c"
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
		"#include \"trux_runtime.h\"",
		"int64_t trux_f_3_add(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t trux_v_1_a, int64_t trux_v_1_b);",
		"int64_t trux_f_4_main(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"int64_t trux_f_3_add(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t trux_v_1_a, int64_t trux_v_1_b) {",
		"int64_t trux_return_value;",
		"trux_return_value = rt_add_i64(trux_v_1_a, trux_v_1_b);",
		"goto trux_return;",
		"trux_return:",
		"return trux_return_value;",
		"int64_t trux_v_1_x = trux_f_3_add(trux_ctx, trux_result_arena, 1, 2);",
		"rt_print_int(trux_v_1_x);",
		"rt_print_newline();",
		"int main(void) {",
		"rt_arena_init(&trux_arena);",
		"rt_context trux_ctx = {&trux_arena, NULL};",
		"int64_t trux_exit_code = trux_f_4_main(&trux_ctx, &trux_arena);",
		"rt_arena_deinit(&trux_arena);",
		"return (int)trux_exit_code;",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateUsesCheckedIntegerArithmetic(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func main() int {
    let a int = 7 + 3
    let b int = a - 2
    let c int = b * 4
    let d int = c / 5
    return d
}`)

	wantParts := []string{
		"int64_t trux_v_1_a = rt_add_i64(7, 3);",
		"int64_t trux_v_1_b = rt_sub_i64(trux_v_1_a, 2);",
		"int64_t trux_v_1_c = rt_mul_i64(trux_v_1_b, 4);",
		"int64_t trux_v_1_d = rt_div_i64(trux_v_1_c, 5);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateOrdersSideEffectingCallArguments(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "io"

func sub(a int, b int) int {
    return a - b
}

func main() int {
    print(sub(io.readInt(), io.readInt()))
    return 0
}`)

	wantParts := []string{
		"int64_t trux_tmp_0 = rt_read_int(&trux_frame);",
		"int64_t trux_tmp_1 = rt_read_int(&trux_frame);",
		"rt_print_int(trux_f_3_sub(trux_ctx, &trux_frame, trux_tmp_0, trux_tmp_1));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
	if strings.Contains(cSource, "trux_f_3_sub(trux_ctx, &trux_frame, rt_read_int(&trux_frame), rt_read_int(&trux_frame))") {
		t.Fatalf("generated C still nests side-effecting call arguments:\n%s", cSource)
	}
}

func TestGenerateOrdersSideEffectingBinaryOperands(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "io"

func main() int {
    print(io.readInt() - io.readInt())
    return 0
}`)

	wantParts := []string{
		"int64_t trux_tmp_0 = rt_read_int(&trux_frame);",
		"int64_t trux_tmp_1 = rt_read_int(&trux_frame);",
		"rt_print_int(rt_sub_i64(trux_tmp_0, trux_tmp_1));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateOrdersSideEffectingForCondition(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "io"

func main() int {
    for io.readInt() < io.readInt() {
        print("loop")
    }
    return 0
}`)

	wantParts := []string{
		"for (;;) {",
		"int64_t trux_tmp_0 = rt_read_int(&trux_frame);",
		"int64_t trux_tmp_1 = rt_read_int(&trux_frame);",
		"if (!(trux_tmp_0 < trux_tmp_1)) {",
		"break;",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateEmitsMissingReturnBackstop(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func pick(flag bool) int {
    if flag {
        return 42
    } else {
        return 7
    }
}

func main() int {
    return pick(true)
}`)

	if !strings.Contains(cSource, "rt_runtime_fail(\"missing return\");\ntrux_return:") {
		t.Fatalf("generated C missing return backstop before return label:\n%s", cSource)
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
		"#include \"trux_runtime.h\"",
		"rt_string trux_f_5_label(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"bool trux_f_5_ready(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"rt_string trux_return_value;",
		"trux_return_value = RT_LIT(rt_string, (const uint8_t*)\"trux\", 4);",
		"bool trux_return_value;",
		"trux_return_value = true;",
		"rt_string trux_v_4_name = trux_f_5_label(trux_ctx, &trux_frame);",
		"bool trux_v_2_ok = false;",
		"rt_print_string(trux_v_4_name);",
		"rt_print_string(RT_LIT(rt_string, (const uint8_t*)\" \", 1));",
		"rt_print_int(1);",
		"rt_print_newline();",
		"rt_print_bool(trux_v_2_ok);",
		"rt_print_string(RT_LIT(rt_string, (const uint8_t*)\"quote: \\\"\", 8));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForHTTPProgram(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "http"

func handle(request int) int {
    let path string = http.path(request)
    http.respond(request, 200, "text/plain", path)
    return 0
}

func main() int {
    http.serve("127.0.0.1", 8080, 4, handle)
    return 0
}`)

	wantParts := []string{
		"#include \"trux_runtime.h\"",
		"#include \"trux_http_runtime.h\"",
		"int64_t trux_f_6_handle(rt_context* trux_ctx, rt_arena* trux_result_arena, int64_t trux_v_7_request);",
		"rt_http_path(&trux_frame, trux_v_7_request)",
		"rt_http_respond(trux_v_7_request, 200, RT_LIT(rt_string, (const uint8_t*)\"text/plain\", 10), trux_v_4_path);",
		"rt_http_serve(RT_LIT(rt_string, (const uint8_t*)\"127.0.0.1\", 9), 8080, 4, trux_f_6_handle);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateUsesCheckedIntegerArithmeticInKernels(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "gpu"

kernel func fill(out gpu.Buffer[int], n int) {
    let i int = gpu.globalX()
    if i < n {
        out[i] = (i + 1) * 2
    }
}

func main() int {
    return 0
}`)

	wantParts := []string{
		"#define TRUX_RUNTIME_CUDA 1",
		"#include \"trux_runtime.h\"",
		"__global__ void trux_k_4_fill(int64_t* trux_v_3_out, int64_t trux_v_1_n)",
		"trux_v_3_out[trux_v_1_i] = rt_mul_i64(rt_add_i64(trux_v_1_i, 1), 2);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateManglesFunctionsAwayFromRuntimeLocals(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func frame() int {
    return 1
}

func ctx() int {
    return 2
}

func main() int {
    return frame() + ctx()
}`)

	wantParts := []string{
		"int64_t trux_f_5_frame(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"int64_t trux_f_3_ctx(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"int64_t trux_tmp_0 = trux_f_5_frame(trux_ctx, trux_result_arena);",
		"int64_t trux_tmp_1 = trux_f_3_ctx(trux_ctx, trux_result_arena);",
		"trux_return_value = rt_add_i64(trux_tmp_0, trux_tmp_1);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
	if strings.Contains(cSource, "trux_frame(") || strings.Contains(cSource, "trux_ctx(") {
		t.Fatalf("generated C used collision-prone function names:\n%s", cSource)
	}
}

func TestGenerateShapedLaunchInjectsCoordinates(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "gpu"

kernel func fill(i int, out gpu.Buffer[int]) {
    out[i] = i
}

kernel func paint(row int, col int, out gpu.Buffer[int], width int) {
    out[row * width + col] = row + col
}

func main() int {
    let host []int = make([]int, 4)
    let device gpu.Buffer[int] = gpu.from(host)
    gpu.launch1d(fill, 4, device)
    gpu.launch2d(paint, 2, 2, device, 2)
    gpu.to(device, host)
    gpu.free(device)
    return 0
}`)

	wantParts := []string{
		"__global__ void trux_k_4_fill(int64_t trux_dim_x, int64_t* trux_v_3_out)",
		"int64_t trux_v_1_i = ((int64_t)(blockIdx.x * blockDim.x + threadIdx.x));",
		"if (trux_v_1_i < trux_dim_x)",
		"trux_k_4_fill<<<dim3((unsigned int)(((4) + 255) / 256), 1u, 1u), dim3(256u, 1u, 1u)>>>(4, trux_v_6_device.data)",
		"__global__ void trux_k_5_paint(int64_t trux_dim_x, int64_t trux_dim_y, int64_t* trux_v_3_out, int64_t trux_v_5_width)",
		"int64_t trux_v_3_row = ((int64_t)(blockIdx.y * blockDim.y + threadIdx.y));",
		"int64_t trux_v_3_col = ((int64_t)(blockIdx.x * blockDim.x + threadIdx.x));",
		"if (trux_v_3_row < trux_dim_y && trux_v_3_col < trux_dim_x)",
		"trux_k_5_paint<<<dim3((unsigned int)(((2) + 15) / 16), (unsigned int)(((2) + 15) / 16), 1u), dim3(16u, 16u, 1u)>>>(2, 2, trux_v_6_device.data, 2)",
		"rt_gpu_copy_to_device_int(",
		"rt_gpu_copy_to_host_int(",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestShapedLaunchRejectsMixedShapes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.tx")
	src := `package main
import "gpu"

kernel func fill(i int, out gpu.Buffer[int]) {
    out[i] = 1
}

func main() int {
    let out gpu.Buffer[int] = gpu.alloc(1)
    gpu.launch1d(fill, 1, out)
    gpu.launch(fill, 1, 1, 1, 1, 1, 1, out)
    gpu.free(out)
    return 0
}`
	result, err := modules.LoadWithSources(path, map[string]string{path: src})
	if err != nil {
		t.Fatal(err)
	}
	_, err = semtypes.Check(result.Program)
	if err == nil || !strings.Contains(err.Error(), "different shapes") {
		t.Fatalf("error = %v, want mixed launch shapes", err)
	}
}

func TestGenerateManglesKernelsAwayFromFunctions(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "gpu"

kernel func scale(out gpu.Buffer[int], n int) {
    let i int = gpu.globalX()
    if i < n {
        out[i] = i
    }
}

func kernel_scale() int {
    return 7
}

func main() int {
    return kernel_scale()
}`)

	wantParts := []string{
		"int64_t trux_f_12_kernel_scale(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"__global__ void trux_k_5_scale(int64_t* trux_v_3_out, int64_t trux_v_1_n);",
		"trux_return_value = trux_f_12_kernel_scale(trux_ctx, trux_result_arena);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
	if strings.Contains(cSource, "trux_kernel_scale(") {
		t.Fatalf("generated C used collision-prone kernel/function name:\n%s", cSource)
	}
}

func TestGenerateCreatesCForControlFlowProgram(t *testing.T) {
	program, err := parser.Parse(`package main
func main() int {
    let text string = "trux"
    let x float = 1.5
    if x > 10.0 {
        x = 10.0
    } else if "ru" in text {
        x = x + 1.0
    } else {
        x = 0.0
    }
    for x > 0.0 {
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
		"#include \"trux_runtime.h\"",
		"double trux_v_1_x = 1.5;",
		"if (trux_v_1_x > 10.0) {",
		"trux_v_1_x = 10.0;",
		"} else {",
		"if (rt_string_contains(RT_LIT(rt_string, (const uint8_t*)\"ru\", 2), trux_v_4_text)) {",
		"trux_v_1_x = (trux_v_1_x + 1.0);",
		"} else {",
		"trux_v_1_x = 0.0;",
		"for (; trux_v_1_x > 0.0;) {",
		"trux_v_1_x = (trux_v_1_x - 1.0);",
		"rt_print_float(trux_v_1_x);",
		"rt_print_bool(rt_string_equal(trux_v_4_text, RT_LIT(rt_string, (const uint8_t*)\"trux\", 4)));",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForCStyleFor(t *testing.T) {
	program, err := parser.Parse(`package main
func main() int {
    let sum int = 0
    for let i int = 0; i < 3; i = i + 1 {
        sum = sum + i
    }
    print(sum)
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
		"{",
		"int64_t trux_v_1_i = 0;",
		"for (; trux_v_1_i < 3;) {",
		"trux_v_3_sum = rt_add_i64(trux_v_3_sum, trux_v_1_i);",
		"trux_v_1_i = rt_add_i64(trux_v_1_i, 1);",
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
		"#include \"trux_runtime.h\"",
		"rt_string trux_f_5_greet(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_string trux_v_4_name);",
		"trux_return_value = rt_string_clone(trux_result_arena, rt_string_concat(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"hello \", 6), trux_v_4_name));",
		"rt_string trux_v_4_name = trux_f_5_greet(trux_ctx, &trux_frame, RT_LIT(rt_string, (const uint8_t*)\"trux\", 4));",
		"rt_print_string(rt_string_concat(&trux_frame, trux_v_4_name, RT_LIT(rt_string, (const uint8_t*)\"!\", 1)));",
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
		"rt_string trux_f_4_bang(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_string trux_v_4_name);",
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, trux_v_4_name, RT_LIT(rt_string, (const uint8_t*)\"!\", 1));",
		"trux_return_value = rt_string_clone(trux_result_arena, trux_v_1_s);",
		"rt_print_string(trux_f_4_bang(trux_ctx, &trux_frame, RT_LIT(rt_string, (const uint8_t*)\"a\", 1)));",
		"rt_print_string(trux_f_4_bang(trux_ctx, &trux_frame, RT_LIT(rt_string, (const uint8_t*)\"b\", 1)));",
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
		"rt_print_string(trux_f_2_id(trux_ctx, &trux_frame, RT_LIT(rt_string, (const uint8_t*)\"x\", 1)));",
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
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, trux_v_4_name, RT_LIT(rt_string, (const uint8_t*)\"!\", 1));",
		"trux_return_value = rt_string_clone(trux_result_arena, trux_f_2_id(trux_ctx, &trux_frame, trux_v_1_s));",
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
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, trux_v_4_name, RT_LIT(rt_string, (const uint8_t*)\"!\", 1));",
		"rt_list_string_append(trux_v_5_items, trux_v_1_s);",
		"rt_print_string(trux_f_5_stash(trux_ctx, &trux_frame, RT_LIT(rt_string, (const uint8_t*)\"x\", 1)));",
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
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"a\", 1), RT_LIT(rt_string, (const uint8_t*)\"!\", 1));",
		"rt_list_string* trux_v_5_items = rt_list_string_from_values(&trux_frame, RT_ARRAY(rt_string, 1, trux_v_1_s), 1);",
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
		"rt_string trux_v_1_s = rt_string_concat(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"a\", 1), RT_LIT(rt_string, (const uint8_t*)\"!\", 1));",
		"rt_list_string* trux_v_5_items = trux_f_4_wrap(trux_ctx, &trux_frame, trux_v_1_s);",
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
		"#include \"trux_runtime.h\"",
		"rt_array_int trux_v_2_xs = rt_array_int_from_values(&trux_frame, RT_ARRAY(int64_t, 3, 1, 2, 3), 3);",
		"rt_slice_int trux_v_4_view = rt_array_int_slice(trux_v_2_xs, true, 1, false, 0);",
		"rt_slice_int_set(trux_v_4_view, 0, 9);",
		"rt_list_int* trux_v_5_items = rt_list_int_from_values(&trux_frame, RT_ARRAY(int64_t, 1, rt_array_int_get(trux_v_2_xs, 1)), 1);",
		"rt_list_int_append(trux_v_5_items, 4);",
		"rt_slice_int trux_v_4_made = rt_make_slice_int(&trux_frame, rt_checked_len_i64(trux_v_5_items->len));",
		"rt_slice_int_set(trux_v_4_made, 0, rt_list_int_get(trux_v_5_items, 1));",
		"rt_print_int(rt_array_int_get(trux_v_2_xs, 1));",
		"rt_print_string(RT_LIT(rt_string, (const uint8_t*)\" \", 1));",
		"rt_print_string(rt_string_index(RT_LIT(rt_string, (const uint8_t*)\"abc\", 3), 1));",
		"rt_print_string(rt_string_slice(RT_LIT(rt_string, (const uint8_t*)\"abcd\", 4), true, 1, true, 3));",
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
		"rt_list_list_int* trux_f_5_build(rt_context* trux_ctx, rt_arena* trux_result_arena);",
		"rt_list_list_int* trux_v_4_rows = rt_list_list_int_from_values(&trux_frame, RT_ARRAY(rt_list_int*, 2, rt_list_int_from_values(&trux_frame, RT_ARRAY(int64_t, 2, 1, 2), 2), rt_list_int_from_values(&trux_frame, RT_ARRAY(int64_t, 1, 3), 1)), 2);",
		"rt_list_list_int_append(trux_v_4_rows, rt_list_int_from_values(&trux_frame, RT_ARRAY(int64_t, 3, 4, 5, 6), 3));",
		"trux_return_value = rt_list_list_int_clone(trux_result_arena, trux_v_4_rows);",
		"rt_list_list_int* trux_v_4_rows = trux_f_5_build(trux_ctx, &trux_frame);",
		"rt_list_list_int* trux_v_6_copied = rt_list_list_int_clone(&trux_frame, trux_v_4_rows);",
		"rt_list_int_append(rt_list_list_int_get(trux_v_6_copied, 0), 9);",
		"rt_array_array_int trux_v_5_fixed = rt_array_array_int_from_values(&trux_frame, RT_ARRAY(rt_array_int, 2, rt_array_int_from_values(&trux_frame, RT_ARRAY(int64_t, 2, 1, 2), 2), rt_array_int_from_values(&trux_frame, RT_ARRAY(int64_t, 2, 3, 4), 2)), 2);",
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

func TestGenerateCreatesCForIO(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "io"
import "csv"

func main() int {
    let line string = io.readLine()
    let count int = io.readInt()
    let ratio float = io.readFloat()
    let ready bool = io.readBool()
    let text string, err1 error = io.readFile("input.txt")
    let err2 error = io.writeFile("out.txt", text + line)
    let cells list[string], err3 error = csv.read("in.csv", 2)
    let err4 error = csv.write("out.csv", cells, 2)
    print(line, " ", count, " ", ratio, " ", ready, " ", len(cells))
    return 0
}`)

	wantParts := []string{
		"rt_string trux_v_4_line = rt_read_line(&trux_frame);",
		"int64_t trux_v_5_count = rt_read_int(&trux_frame);",
		"double trux_v_5_ratio = rt_read_float(&trux_frame);",
		"bool trux_v_5_ready = rt_read_bool(&trux_frame);",
		"rt_read_file(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"input.txt\", 9))",
		"rt_write_file(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"out.txt\", 7), rt_string_concat(&trux_frame, trux_v_4_text, trux_v_4_line))",
		"rt_read_csv(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"in.csv\", 6), 2)",
		"rt_write_csv(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"out.csv\", 7), trux_v_5_cells, 2)",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateCreatesCForAssert(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func main() int {
    let x int = 1
    assert(x > 0, "x must be positive")
    return x
}`)

	wantParts := []string{
		"#line 2 \"main.tx\"",
		"#line 4 \"main.tx\"",
		"if (!(trux_v_1_x > 0)) {",
		"rt_assert_fail_at(RT_LIT(rt_string, (const uint8_t*)\"x must be positive\", 18), \"main.tx\", 4, 5);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func mustGenerateC(t *testing.T, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "main.tx")
	result, err := modules.LoadWithSources(path, map[string]string{path: src})
	if err != nil {
		t.Fatal(err)
	}
	program := result.Program
	info, err := semtypes.Check(program)
	if err != nil {
		t.Fatal(err)
	}
	typedIR, err := ir.Build(program, info)
	if err != nil {
		t.Fatal(err)
	}
	cSource, err := GenerateWithOptions(typedIR, Options{SourceRoot: filepath.Dir(path)})
	if err != nil {
		t.Fatal(err)
	}
	return cSource
}

func generatedFunctionBody(t *testing.T, cSource string, name string) string {
	t.Helper()

	marker := mangleFunc(name) + "("
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

func TestGenerateAlwaysVoidCastsRuntimeParameters(t *testing.T) {
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
	if !strings.Contains(owned, "(void)trux_ctx;") || !strings.Contains(owned, "(void)trux_result_arena;") {
		t.Fatalf("ownedMiddle should void-cast both runtime parameters:\n%s", owned)
	}

	callsCtx := generatedFunctionBody(t, cSource, "callsCtx")
	if !strings.Contains(callsCtx, "(void)trux_ctx;") || !strings.Contains(callsCtx, "(void)trux_result_arena;") {
		t.Fatalf("callsCtx should void-cast both runtime parameters:\n%s", callsCtx)
	}
	if strings.Contains(cSource, "funcUsage") {
		t.Fatal("generated source should not mention a usage tracker")
	}
}

func TestKernelPrototypeMatchesDefinition(t *testing.T) {
	cSource := mustGenerateC(t, `package main
import "gpu"

kernel func fill(out gpu.Buffer[int], n int) {
    out[0] = n
}

func main() int {
    return 0
}`)

	proto := "__global__ void trux_k_4_fill(int64_t* trux_v_3_out, int64_t trux_v_1_n);"
	def := "__global__ void trux_k_4_fill(int64_t* trux_v_3_out, int64_t trux_v_1_n) {"
	if !strings.Contains(cSource, proto) || !strings.Contains(cSource, def) {
		t.Fatalf("kernel prototype and definition diverged:\n%s", cSource)
	}
}

func TestGenerateRewindsNonEscapingLoopTemps(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func temps(n int) int {
    let i int = 0
    let count int = 0
    for ; i < n; i = i + 1 {
        let temp string = "a" + "b"
        if temp == "ab" {
            count = count + 1
        }
    }
    return count
}

func sumViews(n int) int {
    let xs [3]int = [3]int{1, 2, 3}
    let i int = 0
    let total int = 0
    for ; i < n; i = i + 1 {
        let view []int = clone(xs[:])
        total = total + view[0] + view[2]
    }
    return total
}

func mix(n int) int {
    let nums list[int] = list[int]{}
    let i int = 0
    for ; i < n; i = i + 1 {
        let temp string = "a" + "b"
        append(nums, len(temp))
    }
    return nums[0] + len(nums)
}

func kept(n int) string {
    let acc string = ""
    let i int = 0
    for ; i < n; i = i + 1 {
        acc = acc + "x"
    }
    return acc
}

func collect(n int) int {
    let words list[string] = list[string]{}
    let i int = 0
    for ; i < n; i = i + 1 {
        append(words, "a" + "b")
    }
    return len(words)
}

func pushWord(mut words list[string], word string) int {
    append(words, word)
    return len(words)
}

func fill(n int) int {
    let words list[string] = list[string]{}
    let i int = 0
    for ; i < n; i = i + 1 {
        pushWord(words, "a" + "b")
    }
    return len(words)
}

func hot(n int) int {
    let checksum int = 1
    for let i int = 0; i < n; i = i + 1 {
        checksum = checksum + i
    }
    return checksum
}

func skipped(n int) int {
    let i int = 0
    let count int = 0
    for ; i < n; i = i + 1 {
        let temp string = "a" + "b"
        if i == 1 {
            continue
        }
        if temp == "ab" {
            count = count + 1
        }
    }
    return count
}

func main() int {
    return temps(1) + sumViews(1) + mix(1) + len(kept(1)) + collect(1) + fill(1) + hot(1) + skipped(1)
}`)

	for _, name := range []string{"temps", "sumViews", "mix", "skipped"} {
		body := generatedFunctionBody(t, cSource, name)
		if !strings.Contains(body, "rt_arena_mark_current(&trux_frame)") || !strings.Contains(body, "rt_arena_rewind(&trux_frame,") {
			t.Fatalf("%s should rewind non-escaping loop memory:\n%s", name, body)
		}
	}
	skipped := generatedFunctionBody(t, cSource, "skipped")
	labelAt := strings.Index(skipped, "trux_continue_")
	rewindAt := strings.LastIndex(skipped, "rt_arena_rewind(&trux_frame,")
	if labelAt < 0 || rewindAt < 0 || labelAt > rewindAt {
		t.Fatalf("continue should reach rewind, label %d rewind %d:\n%s", labelAt, rewindAt, skipped)
	}
	for _, name := range []string{"kept", "collect", "fill"} {
		body := generatedFunctionBody(t, cSource, name)
		if strings.Contains(body, "rt_arena_rewind") {
			t.Fatalf("%s should keep memory that escapes the iteration:\n%s", name, body)
		}
	}
	hot := generatedFunctionBody(t, cSource, "hot")
	if strings.Contains(hot, "rt_arena_mark") || strings.Contains(hot, "rt_arena_init_frame") {
		t.Fatalf("integer loop should not touch a frame arena:\n%s", hot)
	}
}

func TestNonEscapingLoopReusesOneChunk(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func main() int {
    let i int = 0
    let count int = 0
    for ; i < 5000; i = i + 1 {
        let temp string = "a" + "b"
        if temp == "ab" {
            count = count + 1
        }
    }
    print(count)
    return 0
}`)
	stdout, mallocs := compileAndRunGeneratedC(t, cSource)
	if stdout != "5000\n" {
		t.Fatalf("stdout = %q, want 5000", stdout)
	}
	if mallocs != 1 {
		t.Fatalf("mallocs = %d, want 1 reused frame chunk", mallocs)
	}
}

func TestEscapingLoopKeepsGrowingString(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func main() int {
    let acc string = ""
    let i int = 0
    for ; i < 200; i = i + 1 {
        acc = acc + "x"
    }
    print(len(acc))
    return 0
}`)
	if strings.Contains(generatedFunctionBody(t, cSource, "main"), "rt_arena_rewind") {
		t.Fatal("growing string loop should not rewind")
	}
	stdout, mallocs := compileAndRunGeneratedC(t, cSource)
	if stdout != "200\n" {
		t.Fatalf("stdout = %q, want 200", stdout)
	}
	if mallocs < 2 {
		t.Fatalf("mallocs = %d, want the growing string to allocate more than one chunk", mallocs)
	}
}

func compileAndRunGeneratedC(t *testing.T, source string) (string, int) {
	t.Helper()

	const stdlibInclude = "#include <stdlib.h>\n"
	const mallocHooks = `#include <stdlib.h>

static size_t rt_test_malloc_calls = 0;

static void* rt_test_malloc(size_t size) {
    rt_test_malloc_calls++;
    return malloc(size);
}

static void rt_test_free(void* ptr) {
    free(ptr);
}

static void* rt_test_realloc(void* ptr, size_t size) {
    return realloc(ptr, size);
}

#define malloc rt_test_malloc
#define free rt_test_free
#define realloc rt_test_realloc
`
	header := runtimec.Source
	if !strings.Contains(header, stdlibInclude) {
		t.Fatal("runtime header is missing stdlib include")
	}
	header = strings.Replace(header, stdlibInclude, mallocHooks, 1)
	const ret = "return (int)trux_exit_code;"
	if !strings.Contains(source, ret) {
		t.Fatal("generated C is missing main return")
	}
	source = strings.Replace(source, ret, "fflush(stdout);\n    fprintf(stderr, \"mallocs %zu\\n\", rt_test_malloc_calls);\n    "+ret, 1)

	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "cc"
	}
	if _, err := exec.LookPath(compiler); err != nil {
		t.Skipf("C compiler %q not found", compiler)
	}
	dir := t.TempDir()
	cPath := filepath.Join(dir, "main.c")
	exePath := filepath.Join(dir, "main")
	if err := os.WriteFile(filepath.Join(dir, runtimec.HeaderName), []byte(header), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cPath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	compile := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-pedantic", cPath, "-o", exePath)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile generated C: %v\n%s", err, output)
	}
	run := exec.Command(exePath)
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run generated C: %v\n%s", err, output)
	}
	text := string(output)
	const marker = "mallocs "
	idx := strings.LastIndex(text, marker)
	if idx < 0 {
		t.Fatalf("missing malloc count:\n%s", text)
	}
	stdout := text[:idx]
	rest := strings.TrimSpace(text[idx+len(marker):])
	var mallocs int
	if _, err := fmt.Sscan(rest, &mallocs); err != nil {
		t.Fatalf("malloc count %q: %v", rest, err)
	}
	return stdout, mallocs
}

func TestGenerateElidesFrameForBorrowedSliceReturn(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func mid(xs []int) []int {
    return xs[1:2]
}

func owned(xs []int) []int {
    return clone(xs[1:2])
}

func main() int {
    let xs [3]int = [3]int{1, 2, 3}
    let borrowed []int = mid(xs[:])
    let copied []int = owned(xs[:])
    print(borrowed[0], copied[0])
    return 0
}`)
	mid := generatedFunctionBody(t, cSource, "mid")
	if strings.Contains(mid, "rt_arena_init_frame") {
		t.Fatalf("borrowed return should not create a frame:\n%s", mid)
	}
	owned := generatedFunctionBody(t, cSource, "owned")
	if !strings.Contains(owned, "rt_arena_init_frame") || !strings.Contains(owned, "rt_arena_deinit") {
		t.Fatalf("owned return should create a frame:\n%s", owned)
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
		"rt_array_int trux_v_2_xs = rt_array_int_from_values(&trux_frame, RT_ARRAY(int64_t, 3, 1, 2, 3), 3);",
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
		"rt_list_string_append(trux_v_2_xs, rt_string_concat(&trux_frame, RT_LIT(rt_string, (const uint8_t*)\"a\", 1), RT_LIT(rt_string, (const uint8_t*)\"b\", 1)));",
		"trux_return_value = rt_list_string_clone(trux_result_arena, trux_v_2_xs);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateUsesExistingHandlesForMutableParameters(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func push(mut xs list[int], value int) int {
    append(xs, value)
    return len(xs)
}

func setFirst(mut xs []int) int {
    xs[0] = 42
    return xs[0]
}

func main() int {
    let xs list[int] = list[int]{}
    let fixed [2]int = [2]int{0, 0}
    print(push(xs, 1), " ", setFirst(fixed[:]))
    return 0
}`)

	wantParts := []string{
		"int64_t trux_f_4_push(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_list_int* trux_v_2_xs, int64_t trux_v_5_value);",
		"int64_t trux_f_8_setFirst(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_slice_int trux_v_2_xs);",
		"rt_list_int_append(trux_v_2_xs, trux_v_5_value);",
		"trux_return_value = rt_checked_len_i64(trux_v_2_xs->len);",
		"rt_slice_int_set(trux_v_2_xs, 0, 42);",
	}
	for _, part := range wantParts {
		if !strings.Contains(cSource, part) {
			t.Fatalf("generated C missing %q:\n%s", part, cSource)
		}
	}
}

func TestGenerateAppendsMutableParameterStringValuesToDurableArena(t *testing.T) {
	cSource := mustGenerateC(t, `package main
func addGreeting(mut words list[string], name string) int {
    append(words, "hello " + name)
    return len(words)
}

func main() int {
    let words list[string] = list[string]{}
    return addGreeting(words, "trux")
}`)

	addGreeting := generatedFunctionBody(t, cSource, "addGreeting")
	want := `rt_list_string_append(trux_v_5_words, rt_string_concat(trux_ctx->arena, RT_LIT(rt_string, (const uint8_t*)"hello ", 6), trux_v_4_name));`
	if !strings.Contains(addGreeting, want) {
		t.Fatalf("generated C missing durable append %q:\n%s", want, addGreeting)
	}
	if strings.Contains(addGreeting, "rt_string_concat(&trux_frame") {
		t.Fatalf("addGreeting should not append frame-owned string data:\n%s", addGreeting)
	}
}

func TestGenerateMultiReturnAndErrorHandling(t *testing.T) {
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
    if err != nil {
        print(errors.message(err))
        return 1
    }
    print(err)
    let _, err2 error = divide(10, 0)
    result, err = divide(30, 3)
    _, err = divide(40, 4)
    return 0
}
`
	path := filepath.Join(t.TempDir(), "main.tx")
	result, err := modules.LoadWithSources(path, map[string]string{path: src})
	if err != nil {
		t.Fatal(err)
	}
	info, err := semtypes.Check(result.Program)
	if err != nil {
		t.Fatal(err)
	}
	typedIR, err := ir.Build(result.Program, info)
	if err != nil {
		t.Fatal(err)
	}
	cSource, err := Generate(typedIR)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(cSource, "trux_tuple_") {
		t.Fatalf("generated C missing tuple struct definition:\n%s", cSource)
	}
	if !strings.Contains(cSource, "rt_error") {
		t.Fatalf("generated C missing rt_error:\n%s", cSource)
	}
}
