package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunTestsRunsPassingTestFile(t *testing.T) {
	requireCC(t)

	path := writeTempTestFile(t, `package main
func testAdds() {
    assert(1 + 1 == 2, "addition should work")
}`)

	var out bytes.Buffer
	err := runTests(&out, path, testOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, "PASS "+path+" testAdds") {
		t.Fatalf("output = %q, want pass line", got)
	}
	if !strings.Contains(got, testColorGreen+"PASS "+path+" testAdds"+testColorReset) {
		t.Fatalf("output = %q, want green pass line", got)
	}
	if !strings.Contains(got, "1 passed, 0 failed") {
		t.Fatalf("output = %q, want summary", got)
	}
}

func TestRunTestsReportsFailureAndContinues(t *testing.T) {
	requireCC(t)

	path := writeTempTestFile(t, `package main
func testFails() {
    print("before failure")
    assert(false, "expected failure")
}

func testPasses() {
    assert(true, "ok")
}`)

	var out bytes.Buffer
	err := runTests(&out, path, testOptions{})
	if err == nil {
		t.Fatal("expected failure")
	}

	got := out.String()
	for _, want := range []string{
		"FAIL " + path + " testFails",
		"trux runtime error: assertion failed: expected failure",
		"PASS " + path + " testPasses",
		"1 passed, 1 failed",
		"failures:",
		"trace:",
		"at testFails (" + path + ")",
		"at main.test.tx:4:5",
		"stdout:\nbefore failure\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
	if !strings.Contains(got, testColorRed+"FAIL "+path+" testFails"+testColorReset) {
		t.Fatalf("output = %q, want red fail line", got)
	}
	if strings.Contains(got, "execute /tmp/trux-run-") {
		t.Fatalf("output = %q, want cleaned runtime error without temp executable path", got)
	}
	summaryIndex := strings.Index(got, "1 passed, 1 failed")
	detailsIndex := strings.Index(got, "failures:")
	if summaryIndex < 0 || detailsIndex < 0 || detailsIndex < summaryIndex {
		t.Fatalf("output = %q, want failure details after summary", got)
	}
	details := got[detailsIndex:]
	if strings.Contains(details, "testPasses") {
		t.Fatalf("failure details = %q, want only failed tests", details)
	}
}

func TestRunTestsFailFastStopsAfterFirstFailure(t *testing.T) {
	requireCC(t)

	path := writeTempTestFile(t, `package main
func testA() {
    assert(false, "stop here")
}

func testB() {
    assert(true, "would pass")
}`)

	var out bytes.Buffer
	err := runTests(&out, path, testOptions{FailFast: true})
	if err == nil {
		t.Fatal("expected failure")
	}

	got := out.String()
	if !strings.Contains(got, "FAIL "+path+" testA") {
		t.Fatalf("output = %q, want first failure", got)
	}
	if strings.Contains(got, "testB") {
		t.Fatalf("output = %q, want fail-fast to skip testB", got)
	}
	if !strings.Contains(got, "0 passed, 1 failed") {
		t.Fatalf("output = %q, want fail-fast summary", got)
	}
}

func TestRunTestsSupportsImportsLikeNormalSourceFiles(t *testing.T) {
	requireCC(t)

	dir := t.TempDir()
	testPath := writeTempFile(t, dir, "math.test.tx", `package math
import "math.tx"

func testPrivateHelper() {
    assert(double(4) == 8, "same-package private helper should be visible")
}`)
	writeTempFile(t, dir, "math.tx", `package math
func double(x int) int {
    return x * 2
}`)

	var out bytes.Buffer
	err := runTests(&out, testPath, testOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "PASS "+testPath+" testPrivateHelper") {
		t.Fatalf("output = %q, want pass line", out.String())
	}
}

func TestRunFileDoesNotLoadTestSiblings(t *testing.T) {
	requireCC(t)

	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
func main() int {
    return 0
}`)
	writeTempFile(t, dir, "main.test.tx", `package main
func testWouldFail() {
    assert(false, "normal run must ignore tests")
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}
}

func TestLoadTestCasesRejectsInvalidTestDeclarations(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "params",
			src: `package main
func testBad(x int) {
    assert(true, "ok")
}`,
			want: "must not have parameters",
		},
		{
			name: "return type",
			src: `package main
func testBad() int {
    return 0
}`,
			want: "must not declare a return type",
		},
		{
			name: "explicit return",
			src: `package main
func testBad() {
    return 0
}`,
			want: "must not return explicitly",
		},
		{
			name: "main",
			src: `package main
func main() int {
    return 0
}

func testBad() {
    assert(true, "ok")
}`,
			want: "test file must not define main",
		},
		{
			name: "no tests",
			src: `package main
func helper() int {
    return 1
}`,
			want: "no test functions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempTestFile(t, tt.src)
			_, err := loadTestCases(path)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestScorebookProjectTestsPass(t *testing.T) {
	requireCC(t)

	path := "../../examples/projects/scorebook/scores.test.tx"
	var out bytes.Buffer
	err := runTests(&out, path, testOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "5 passed, 0 failed") {
		t.Fatalf("output = %q, want scorebook summary", out.String())
	}
}

func writeTempTestFile(t *testing.T, src string) string {
	t.Helper()

	return writeTempFile(t, t.TempDir(), "main.test.tx", src)
}

func TestTestFilesDiscoversSortedTestFiles(t *testing.T) {
	dir := t.TempDir()
	first := writeTempFile(t, dir, filepath.Join("a", "alpha.test.tx"), `package main
func testAlpha() {
    assert(true, "ok")
}`)
	second := writeTempFile(t, dir, filepath.Join("b", "beta.test.tx"), `package main
func testBeta() {
    assert(true, "ok")
}`)
	writeTempFile(t, dir, "not_a_test.tx", `package main
func main() int {
    return 0
}`)

	got, err := testFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{first, second}
	if len(got) != len(want) {
		t.Fatalf("files = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("files = %q, want %q", got, want)
		}
	}
}
