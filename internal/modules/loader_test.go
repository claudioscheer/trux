package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudioscheer/trux/internal/ast"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

func TestLoadResolvesRelativeImportsAndDeduplicatesFiles(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "math.tx"
import "./math.tx"

func main() int {
    return math.add(1, 2)
}`)
	writeSource(t, dir, "math.tx", `package math
pub func add(a int, b int) int {
    return a + b
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Program.Functions) != 2 {
		t.Fatalf("function count = %d, want 2", len(result.Program.Functions))
	}
	if len(result.Sources) != 2 {
		t.Fatalf("source count = %d, want 2", len(result.Sources))
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAllowsDifferentPackageNames(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package app
import "math.tx"

func main() int {
    return calc.add(1, 2)
}`)
	writeSource(t, dir, "math.tx", `package calc
pub func add(a int, b int) int {
    return a + b
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Program.PackageName != "app" {
		t.Fatalf("merged package = %q, want entry package app", result.Program.PackageName)
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDetectsImportCycle(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"

func main() int {
    return 0
}`)
	writeSource(t, dir, "a.tx", `package main
import "b.tx"

pub func a() int {
    return 1
}`)
	writeSource(t, dir, "b.tx", `package main
import "a.tx"

pub func b() int {
    return 2
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, "import cycle detected:")
	assertErrorContains(t, err, filepath.Join(dir, "a.tx"))
	assertErrorContains(t, err, filepath.Join(dir, "b.tx"))
}

func TestLoadRejectsInvalidImportPaths(t *testing.T) {
	dir := t.TempDir()
	absImport := filepath.Join(dir, "abs.tx")
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "missing",
			path: "missing.tx",
			want: "cannot find module \"missing.tx\"",
		},
		{
			name: "absolute",
			path: absImport,
			want: "must be relative",
		},
		{
			name: "non tx",
			path: "notes.txt",
			want: "must end in .tx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeSource(t, dir, "main.tx", `package main
import "`+tt.path+`"

func main() int {
    return 0
}`)

			_, err := Load(filepath.Join(dir, "main.tx"))
			if err == nil {
				t.Fatal("expected error")
			}
			assertErrorContains(t, err, tt.want)
		})
	}
}

func TestLoadResolvesStandardLibraryImports(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "io"
import "csv"

func main() int {
    let line string = io.readLine()
    let cells list[string] = csv.read("input.csv", 2)
    io.writeFile("out.txt", line)
    csv.write("out.csv", cells, 2)
    return 0
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("source count = %d, want only entry source for stdlib imports", len(result.Sources))
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsUnknownStandardLibraryImport(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "net"

func main() int {
    return 0
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `unknown standard package "net"`)
}

func TestLoadRejectsStandardLibraryCallsWithoutDirectImport(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"

func main() int {
    let line string = io.readLine()
    return len(line)
}`)
	writeSource(t, dir, "a.tx", `package a
import "io"

pub func value() string {
    return io.readLine()
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `package "io" is not imported`)
}

func TestLoadRejectsSourcePackageNameConflictingWithStandardLibraryImport(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "io"
import "local_io.tx"

func main() int {
    return 0
}`)
	writeSource(t, dir, "local_io.tx", `package io
pub func value() int {
    return 1
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `package "io" imported from both standard package "io"`)
}

func TestLoadRejectsMissingStandardLibraryFunction(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "io"

func main() int {
    io.missing()
    return 0
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `package "io" has no function "missing"`)
}

func TestLoadRejectsDirectoryImport(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "module.tx"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSource(t, dir, "main.tx", `package main
import "module.tx"

func main() int {
    return 0
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, "resolved to directory")
	assertErrorContains(t, err, "expected .tx file")
}

func TestLoadAllowsSameNamedPrivateFunctionsInDifferentFiles(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"
import "b.tx"

func main() int {
    return alpha.a() + beta.b()
}`)
	writeSource(t, dir, "a.tx", `package alpha
func value() int {
    return 1
}

pub func a() int {
    return value()
}`)
	writeSource(t, dir, "b.tx", `package beta
func value() int {
    return 2
}

pub func b() int {
    return value()
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}

	if !hasFunctionPrefix(result.Program, "__trux_mod_1_value") || !hasFunctionPrefix(result.Program, "__trux_mod_2_value") {
		t.Fatalf("functions = %#v, want private functions rewritten with module prefixes", functionNames(result.Program))
	}
}

func TestLoadAllowsSameNamedPublicFunctionsInDifferentPackages(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"
import "b.tx"

func main() int {
    return alpha.value() + beta.value()
}`)
	writeSource(t, dir, "a.tx", `package alpha
pub func value() int {
    return 1
}`)
	writeSource(t, dir, "b.tx", `package beta
pub func value() int {
    return 2
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAllowsSamePackageDirectImports(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"
import "b.tx"

func main() int {
    return util.a() + util.b()
}`)
	writeSource(t, dir, "a.tx", `package util
pub func a() int {
    return 1
}`)
	writeSource(t, dir, "b.tx", `package util
pub func b() int {
    return 2
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsDuplicatePublicFunctionsInSamePackageDirectImports(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"
import "b.tx"

func main() int {
    return util.value()
}`)
	writeSource(t, dir, "a.tx", `package util
pub func value() int {
    return 1
}`)
	writeSource(t, dir, "b.tx", `package util
pub func value() int {
    return 2
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `package "util" exports function "value" from both`)
}

func TestLoadRejectsReservedFunctionPrefixAndImportedMain(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "reserved prefix",
			files: map[string]string{
				"main.tx": `package main
func __trux_user() int {
    return 1
}

func main() int {
    return __trux_user()
}`,
			},
			want: `reserved compiler prefix "__trux_"`,
		},
		{
			name: "imported main",
			files: map[string]string{
				"main.tx": `package main
import "lib.tx"

func main() int {
    return 0
}`,
				"lib.tx": `package main
func main() int {
    return 1
}`,
			},
			want: "must not define main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, src := range tt.files {
				writeSource(t, dir, name, src)
			}

			_, err := Load(filepath.Join(dir, "main.tx"))
			if err == nil {
				t.Fatal("expected error")
			}
			assertErrorContains(t, err, tt.want)
		})
	}
}

func TestLoadRejectsCallsToOtherFilePrivateFunction(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "lib.tx"

func main() int {
    return lib.secret()
}`)
	writeSource(t, dir, "lib.tx", `package lib
func secret() int {
    return 42
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `cannot call private function "lib.secret"`)
	assertErrorContains(t, err, filepath.Join(dir, "lib.tx"))
}

func TestLoadResolvesTransitivePublicFunctions(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"

func main() int {
    return a.a()
}`)
	writeSource(t, dir, "a.tx", `package a
import "b.tx"

pub func a() int {
    return b.b()
}`)
	writeSource(t, dir, "b.tx", `package b
pub func b() int {
    return 7
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsTransitivePackageCallsWithoutDirectImport(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "a.tx"

func main() int {
    return b.b()
}`)
	writeSource(t, dir, "a.tx", `package a
import "b.tx"

pub func a() int {
    return b.b()
}`)
	writeSource(t, dir, "b.tx", `package b
pub func b() int {
    return 7
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `package "b" is not imported`)
}

func TestLoadRejectsUnqualifiedImportedPublicFunction(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "math.tx"

func main() int {
    return add(1, 2)
}`)
	writeSource(t, dir, "math.tx", `package math
pub func add(a int, b int) int {
    return a + b
}`)

	_, err := Load(filepath.Join(dir, "main.tx"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorContains(t, err, `imported function "add" must be called as "math.add"`)
}

func TestLoadResolvesSameFilePrivateBeforeLoadedPublic(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "main.tx", `package main
import "lib.tx"

func value() int {
    return 1
}

func main() int {
    return value()
}`)
	writeSource(t, dir, "lib.tx", `package lib
pub func value() int {
    return 2
}`)

	result, err := Load(filepath.Join(dir, "main.tx"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := semtypes.Check(result.Program); err != nil {
		t.Fatal(err)
	}

	call := findMainReturnCall(t, result.Program)
	if !strings.HasPrefix(call.ResolvedCallee, "__trux_mod_0_value") {
		t.Fatalf("resolved callee = %q, want entry private function rewrite", call.ResolvedCallee)
	}
}

func writeSource(t *testing.T, dir string, name string, src string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func hasFunctionPrefix(program *ast.Program, prefix string) bool {
	for _, fn := range program.Functions {
		if strings.HasPrefix(fn.Name, prefix) {
			return true
		}
	}
	return false
}

func functionNames(program *ast.Program) []string {
	names := make([]string, 0, len(program.Functions))
	for _, fn := range program.Functions {
		names = append(names, fn.Name)
	}
	return names
}

func findMainReturnCall(t *testing.T, program *ast.Program) *ast.CallExpr {
	t.Helper()

	for _, fn := range program.Functions {
		if fn.Name != "main" {
			continue
		}
		stmt := fn.Body.Statements[len(fn.Body.Statements)-1].(*ast.ReturnStmt)
		call, ok := stmt.Value.(*ast.CallExpr)
		if !ok {
			t.Fatalf("main return = %T, want call", stmt.Value)
		}
		return call
	}

	t.Fatal("missing main")
	return nil
}
