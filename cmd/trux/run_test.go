package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFilePrintsTokenDump(t *testing.T) {
	path := writeTempSource(t, "package main\n")

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "1:1 PACKAGE \"package\"\n" +
		"1:9 IDENT \"main\"\n" +
		"2:1 EOF \"\"\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileReturnsLexError(t *testing.T) {
	path := writeTempSource(t, "@\n")

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}

	wantOut := "1:1 ILLEGAL \"@\"\n"
	if out.String() != wantOut {
		t.Fatalf("output = %q, want %q", out.String(), wantOut)
	}

	wantErr := "lex error at 1:1: illegal character \"@\""
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), wantErr)
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
