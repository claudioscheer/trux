package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFileParsesValidSourceWithNoOutput(t *testing.T) {
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

func TestRunFileWithDebugWritesPhaseFiles(t *testing.T) {
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
		path + ":4:5: print expects 1 argument, got 0",
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
