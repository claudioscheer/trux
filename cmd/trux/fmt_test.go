package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatPathFormatsSingleFile(t *testing.T) {
	path := writeTempSource(t, `package main
func main()int{
return 0
}`)

	if err := formatPath(path); err != nil {
		t.Fatal(err)
	}

	got := readFile(t, path)
	want := `package main
func main() int {
  return 0
}
`
	if got != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatPathRequiresTxFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := formatPath(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "fmt expects a .tx file") {
		t.Fatalf("error = %q, want .tx requirement", err.Error())
	}
}

func TestFormatRecursiveFormatsTxFilesUnderRoot(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
func main()int{
return helper()
}`)
	libPath := writeTempFile(t, dir, "nested/lib.tx", `package lib
pub func helper()int{
return 1
}`)
	textPath := writeTempFile(t, dir, "nested/readme.txt", "not trux")

	if err := formatRecursive(dir); err != nil {
		t.Fatal(err)
	}

	if got := readFile(t, mainPath); got != "package main\nfunc main() int {\n  return helper()\n}\n" {
		t.Fatalf("main source = %q", got)
	}
	if got := readFile(t, libPath); got != "package lib\npub func helper() int {\n  return 1\n}\n" {
		t.Fatalf("lib source = %q", got)
	}
	if got := readFile(t, textPath); got != "not trux" {
		t.Fatalf("text source = %q, want unchanged", got)
	}
}

func TestFormatCommandRequiresFileWithoutRecursive(t *testing.T) {
	err := fmtCmd.RunE(fmtCmd, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "fmt requires a file unless -r is set" {
		t.Fatalf("error = %q, want missing file error", err.Error())
	}
}
