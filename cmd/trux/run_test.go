package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFileParsesValidSourceWithNoOutput(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}
}

func TestRunFileCompilesAndExecutesIntegerProgram(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func add(a int, b int) int {
    return a + b
}

func main() int {
    let x int = add(1, 2)
    print(x)
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "3\n" {
		t.Fatalf("output = %q, want 3\\n", out.String())
	}
}

func TestRunFileCompilesAndExecutesPrimitiveProgram(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
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
    print("line\nquote: \"")
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "trux 1\ntrue\nline\nquote: \"\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileKeepsTempStringReturnAliveForCaller(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func bang(name string) string {
    let s string = name + "!"
    return s
}

func main() int {
    print(bang("a"))
    print(bang("b"))
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "a!\nb!\n" {
		t.Fatalf("output = %q, want a!\\nb!\\n", out.String())
	}
}

func TestRunFileCompilesAndExecutesControlAndStringExamples(t *testing.T) {
	requireCC(t)

	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../examples/control_flow.tx",
			want: "big\n0\n1\n2\ndone\n",
		},
		{
			path: "../../examples/primitives.tx",
			want: "trux 3 false\n2\nline\nquote: \"\n",
		},
		{
			path: "../../examples/fibonacci.tx",
			want: "iterative\n0\n1\n1\n2\n3\n5\n8\n13\n21\n34\nrecursive\n0\n1\n1\n2\n3\n5\n8\n13\n21\n34\n",
		},
		{
			path: "../../examples/strings.tx",
			want: "trux compiler\nhello, trux compiler\ntrux compiler!\nempty\ntrue\ntrue\ntrue\ntrue\nfalse\ntrue\ntrue\ntrue\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var out bytes.Buffer
			err := runFile(&out, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunFileCompilesAndExecutesCollectionExamples(t *testing.T) {
	requireCC(t)

	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../examples/collections.tx",
			want: "3\n1 3\n9 2\n8 8 2\n5 0 7\nr ru 4\n2 2\n3 2 3\n8 2\n",
		},
		{
			path: "../../examples/ownership_clone.tx",
			want: "99 2\n20 30\n1 42\n7 9\nab cd 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var out bytes.Buffer
			err := runFile(&out, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunFileCompilesAndExecutesModuleExamples(t *testing.T) {
	requireCC(t)

	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../examples/modules/basic/main.tx",
			want: "7\n",
		},
		{
			path: "../../examples/modules/transitive/main.tx",
			want: "hello modules 42\n",
		},
		{
			path: "../../examples/modules/private_names/main.tx",
			want: "11 22\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var out bytes.Buffer
			err := runFile(&out, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunFileCompilesAndExecutesNestedCollections(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func buildRows() list[list[int]] {
    let rows list[list[int]] = list[list[int]]{list[int]{1, 2}, list[int]{3}}
    append(rows, list[int]{4, 5, 6})
    return rows
}

func main() int {
    let rows list[list[int]] = buildRows()
    let copied list[list[int]] = clone(rows)
    append(copied[0], 9)

    let fixed [2][2]int = [2][2]int{[2]int{1, 2}, [2]int{3, 4}}

    let slices [][]float = make([][]float, 2)
    slices[0] = make([]float, 2)
    slices[0][1] = 2.5

    print(len(rows[0]), " ", len(copied[0]), " ", copied[0][2], " ", rows[2][2])
    print(fixed[1][1], " ", slices[0][1])
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "2 3 9 6\n4 2.5\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileCompilesAndExecutesOwnershipCopyOut(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func middle() []int {
    let xs [3]int = [3]int{1, 2, 3}
    return xs[1:2]
}

func build() list[string] {
    let xs list[string] = list[string]{}
    append(xs, "a" + "b")
    return xs
}

func mid(xs []int) []int {
    return xs[1:2]
}

func midOwned(xs []int) []int {
    return clone(xs[1:2])
}

func frameOwnedLocal() []int {
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = clone(xs[:])
    return ys
}

func main() int {
    let local []int = middle()
    print(local[0])

    let words list[string] = build()
    print(words[0], " ", len(words))

    let xs [3]int = [3]int{1, 2, 3}
    let borrowed []int = mid(xs[:])
    let owned []int = midOwned(xs[:])
    xs[1] = 9
    print(borrowed[0], " ", owned[0])

    let framed []int = frameOwnedLocal()
    print(framed[0], " ", framed[2])
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "2\nab 1\n9 2\n1 3\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileCompilesAndExecutesModuleProgram(t *testing.T) {
	requireCC(t)

	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "math.tx"
import "words.tx"

func main() int {
    print(label(), " ", add(2, 5))
    return 0
}`)
	writeTempFile(t, dir, "math.tx", `package math
func base() int {
    return 2
}

pub func add(a int, b int) int {
    return a + b + base()
}`)
	writeTempFile(t, dir, "words.tx", `package words
pub func label() string {
    return "module"
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "module 9\n" {
		t.Fatalf("output = %q, want module 9\\n", out.String())
	}
}

func TestRunFileReturnsRuntimeBoundsError(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    let xs [1]int = [1]int{1}
    print(xs[1])
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}
	if !strings.Contains(err.Error(), "trux runtime error: index 1 out of bounds for length 1") {
		t.Fatalf("error = %q, want bounds error", err.Error())
	}
}

func TestRunFileReturnsMissingImportWithSourceContext(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "missing.tx"

func main() int {
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		mainPath + ":2:1: cannot find module \"missing.tx\"",
		"1 | package main",
		"2 | import \"missing.tx\"",
		"  | ^",
		"4 | func main() int {",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsImportedTypeErrorWithImportedSourceContext(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "lib.tx"

func main() int {
    return bad()
}`)
	libPath := writeTempFile(t, dir, "lib.tx", `package main
pub func bad() int {
    return "wrong"
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		libPath + ":3:12: cannot return string from function returning int",
		"1 | package main",
		"2 | pub func bad() int {",
		"3 |     return \"wrong\"",
		"  |            ^",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsImportedParseErrorWithImportedSourceContext(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "lib.tx"

func main() int {
    return 0
}`)
	libPath := writeTempFile(t, dir, "lib.tx", `package main
pub func bad( int {
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		libPath + ":2:15: expected IDENT, got \"int\"",
		"1 | package main",
		"2 | pub func bad( int {",
		"  |               ^",
		"3 |     return 0",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileWithDebugWritesPhaseFiles(t *testing.T) {
	requireCC(t)

	t.Cleanup(func() {
		_ = os.RemoveAll("tmp")
	})

	path := writeTempSource(t, `package main
func main() int {
    let x int = 1
    print(x)
    return 0
}`)

	var out bytes.Buffer
	err := runFileWithOptions(&out, path, runOptions{Debug: true})
	if err != nil {
		t.Fatal(err)
	}

	debugDir := filepath.Join("tmp", "trux-debug", "main")
	if !strings.Contains(out.String(), "debug files: "+debugDir) {
		t.Fatalf("output = %q, want debug dir", out.String())
	}

	wantFiles := []string{
		"00-source.tx",
		"01-tokens.txt",
		"02-ast.json",
		"03-types.txt",
		"04-ir.json",
		"05-c.c",
	}
	for _, file := range wantFiles {
		path := filepath.Join(debugDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected debug file %s: %v", path, err)
		}
	}

	tokens := readFile(t, filepath.Join(debugDir, "01-tokens.txt"))
	if !strings.Contains(tokens, "IDENT\t\"print\"") {
		t.Fatalf("tokens = %q, want print token", tokens)
	}

	typeInfo := readFile(t, filepath.Join(debugDir, "03-types.txt"))
	if !strings.Contains(typeInfo, "print -> print(int)") {
		t.Fatalf("type info = %q, want resolved print call", typeInfo)
	}

	typedIR := readFile(t, filepath.Join(debugDir, "04-ir.json"))
	if !strings.Contains(typedIR, `"Type": "int"`) {
		t.Fatalf("IR = %q, want int print type", typedIR)
	}

	cSource := readFile(t, filepath.Join(debugDir, "05-c.c"))
	if !strings.Contains(cSource, "rt_print_int(trux_v_1_x);") || !strings.Contains(cSource, "rt_print_newline();") {
		t.Fatalf("generated C = %q, want int print call with newline", cSource)
	}
}

func TestRunFileWithDebugWritesPartialFilesOnTypeError(t *testing.T) {
	t.Cleanup(func() {
		_ = os.RemoveAll("tmp")
	})

	path := writeTempSource(t, `package main
func main() int {
    print()
    return 0
}`)

	var out bytes.Buffer
	err := runFileWithOptions(&out, path, runOptions{Debug: true})
	if err == nil {
		t.Fatal("expected error")
	}

	debugDir := filepath.Join("tmp", "trux-debug", "main")
	if !strings.Contains(out.String(), "debug files: "+debugDir) {
		t.Fatalf("output = %q, want debug dir", out.String())
	}
	for _, file := range []string{"00-source.tx", "01-tokens.txt", "02-ast.json"} {
		path := filepath.Join(debugDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected debug file %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(debugDir, "03-types.txt")); !os.IsNotExist(err) {
		t.Fatalf("types debug file should not exist after type error, stat err = %v", err)
	}
}

func TestRunFileReturnsLexError(t *testing.T) {
	path := writeTempSource(t, "@\n")

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}

	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		path + ":1:1: lex error: illegal character \"@\"",
		"1 | @",
		"  | ^",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsSyntaxErrorWithSourceContext(t *testing.T) {
	src := `package main

func add(a int, b int) int {
    return a + b
}

func main( int {
    return 0
}`
	path := writeTempSource(t, src)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		path + ":7:12: expected IDENT, got \"int\"",
		"5 | }",
		"6 | ",
		"7 | func main( int {",
		"  |            ^",
		"8 |     return 0",
		"9 | }",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsTypeErrorWithSourceContext(t *testing.T) {
	src := `package main

func main() int {
    print()
    return 0
}`
	path := writeTempSource(t, src)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		path + ":4:5: print expects at least 1 argument, got 0",
		"2 | ",
		"3 | func main() int {",
		"4 |     print()",
		"  |     ^",
		"5 |     return 0",
		"6 | }",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunCommandDoesNotPrintUsageForParseErrors(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Fatal("root command should let execute print errors once")
	}
	if !runCmd.SilenceUsage {
		t.Fatal("run command should not print usage for source parse errors")
	}
}

func TestEmitCFileWritesGeneratedC(t *testing.T) {
	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)

	result, err := compileFile(path, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.CSource, "int main(void)") {
		t.Fatalf("generated C = %q, want C main", result.CSource)
	}
}

func TestEmitCFileSupportsModuleProgram(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "math.tx"

func main() int {
    return add(1, 2)
}`)
	writeTempFile(t, dir, "math.tx", `package math
pub func add(a int, b int) int {
    return a + b
}`)

	result, err := compileFile(mainPath, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.CSource, "int64_t trux_add(") {
		t.Fatalf("generated C = %q, want module function add", result.CSource)
	}
	if !strings.Contains(result.CSource, "int main(void)") {
		t.Fatalf("generated C = %q, want C main", result.CSource)
	}
}

func TestBuildCommandRequiresOutput(t *testing.T) {
	oldOutput := buildOutput
	t.Cleanup(func() {
		buildOutput = oldOutput
	})
	buildOutput = ""

	err := buildCmd.RunE(buildCmd, []string{"main.tx"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "build requires -o <output>" {
		t.Fatalf("error = %q, want output requirement", err.Error())
	}
}

func TestBuildFileRespectsCCEnvironmentVariable(t *testing.T) {
	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "program")
	logPath := filepath.Join(dir, "cc.log")
	fakeCC := filepath.Join(dir, "fake-cc")
	writeFakeCompiler(t, fakeCC, logPath)
	t.Setenv("CC", fakeCC)

	if err := buildFile(path, outputPath); err != nil {
		t.Fatal(err)
	}

	log := readFile(t, logPath)
	if !strings.Contains(log, "-std=c11") {
		t.Fatalf("compiler args = %q, want C11 standard flag", log)
	}
	if !strings.Contains(log, "-o "+outputPath) {
		t.Fatalf("compiler args = %q, want output path", log)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output executable: %v", err)
	}
}

func TestGeneratedExamplesCompileWithStrictCWarnings(t *testing.T) {
	requireCC(t)

	paths := exampleEntrypoints(t)
	if len(paths) == 0 {
		t.Fatal("no example programs found")
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			result, err := compileFile(path, compileOptions{})
			if err != nil {
				t.Fatal(err)
			}

			dir := t.TempDir()
			cPath := filepath.Join(dir, "main.c")
			if err := os.WriteFile(cPath, []byte(result.CSource), 0o644); err != nil {
				t.Fatal(err)
			}

			compiler := os.Getenv("CC")
			if compiler == "" {
				compiler = "cc"
			}
			objPath := filepath.Join(dir, "main.o")
			cmd := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-pedantic", "-c", cPath, "-o", objPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compile generated C: %v\n%s", err, output)
			}
			if strings.TrimSpace(string(output)) != "" {
				t.Fatalf("strict C compiler emitted warnings:\n%s", output)
			}
		})
	}
}

func exampleEntrypoints(t *testing.T) []string {
	t.Helper()

	paths, err := filepath.Glob("../../examples/*.tx")
	if err != nil {
		t.Fatal(err)
	}

	err = filepath.WalkDir("../../examples", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "main.tx" && filepath.Dir(path) != "../../examples" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return paths
}

func writeTempSource(t *testing.T, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "main.tx")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func writeTempFile(t *testing.T, dir string, name string, src string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

func requireCC(t *testing.T) {
	t.Helper()

	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "cc"
	}
	if _, err := exec.LookPath(compiler); err != nil {
		t.Skipf("C compiler %q not found", compiler)
	}
}

func writeFakeCompiler(t *testing.T, path string, logPath string) {
	t.Helper()

	script := fmt.Sprintf(`#!/bin/sh
echo "$@" > %q
while [ "$#" -gt 0 ]; do
    if [ "$1" = "-o" ]; then
        shift
        touch "$1"
        exit 0
    fi
    shift
done
exit 2
`, logPath)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
