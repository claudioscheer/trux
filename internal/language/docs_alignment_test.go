package language

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLanguageDocsMatchTheCompiler(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")

	specs := readDoc(t, filepath.Join(root, "docs", "SPECS.md"))
	modules := readDoc(t, filepath.Join(root, "docs", "MODULES.md"))
	readme := readDoc(t, filepath.Join(root, "README.md"))
	gpu := readDoc(t, filepath.Join(root, "docs", "GPU.md"))

	for _, phrase := range []string{
		"Nested lists, slices, and arrays are supported",
		"pub func` and package-qualified calls are implemented",
		"Integer `+`, `-`, `*`, and `/` trap on signed overflow",
		"Integer `/` also traps on division by zero",
		"Float arithmetic follows IEEE 754",
	} {
		if !strings.Contains(specs, phrase) {
			t.Fatalf("SPECS.md missing %q", phrase)
		}
	}
	if strings.Contains(specs, "Nested collections are rejected") {
		t.Fatal("SPECS.md still rejects nested collections")
	}
	if strings.Contains(specs, "nested collection element types") {
		t.Fatal("SPECS.md still lists nested collections as a compiler rejection")
	}

	for _, pkg := range []string{"`io`", "`csv`", "`image`", "`time`", "`http`", "`gpu`"} {
		if !strings.Contains(modules, pkg) {
			t.Fatalf("MODULES.md missing standard package %s", pkg)
		}
	}
	if strings.Contains(modules, "Currently supported standard packages are `io` and `csv`") {
		t.Fatal("MODULES.md still limits bare imports to io and csv")
	}

	if !strings.Contains(readme, "nvcc") {
		t.Fatal("README pipeline missing nvcc")
	}
	if !strings.Contains(readme, "CUDA") {
		t.Fatal("README pipeline missing CUDA")
	}

	if !strings.Contains(gpu, "does not print a Trux runtime error message") {
		t.Fatal("GPU.md missing the device trap note")
	}
}

func readDoc(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
