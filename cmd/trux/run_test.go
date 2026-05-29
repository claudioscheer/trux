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
