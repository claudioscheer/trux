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

func TestRunFileCompilesAndExecutesV0Program(t *testing.T) {
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

func TestRunFileCompilesAndExecutesV1Program(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func label() string {
    return "Kern"
}

func ready() bool {
    return true
}

func main() int {
    let name string = label()
    let ok bool = ready()
    print(name, 1)
    print(ok)
    print("line\nquote: \"")
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "Kern1\ntrue\nline\nquote: \"\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileCompilesAndExecutesV2Examples(t *testing.T) {
	requireCC(t)

	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../examples/v2/control_flow.tx",
			want: "big\n0\n1\n2\ndone\n",
		},
		{
			path: "../../examples/v2/floats.tx",
			want: "2\n5\nhigh\n",
		},
		{
			path: "../../examples/v2/mixed.tx",
			want: "3.75\ntrue true\n",
		},
		{
			path: "../../examples/v2/strings_in.tx",
			want: "true\ntrue\ntrue\nfalse\ntrue\ntrue\ntrue\n",
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
	if !strings.Contains(cSource, "rt_print_int(x);") || !strings.Contains(cSource, "rt_print_newline();") {
		t.Fatalf("generated C = %q, want int print call with newline", cSource)
	}
}

func TestRunFileWithDebugWritesEarlyPhaseFilesOnTypeError(t *testing.T) {
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
	if !strings.Contains(log, "-o "+outputPath) {
		t.Fatalf("compiler args = %q, want output path", log)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output executable: %v", err)
	}
}

func writeTempSource(t *testing.T, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "main.tx")
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
