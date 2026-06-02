package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/modules"
	"github.com/claudioscheer/trux/internal/token"
	"github.com/spf13/cobra"
)

var testFailFast bool

const (
	testColorGreen = "\033[32m"
	testColorRed   = "\033[31m"
	testColorReset = "\033[0m"
)

var testCmd = &cobra.Command{
	Use:          "test [file.test.tx]",
	Short:        "Run trux unit tests",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) == 1 {
			root = args[0]
		}
		return runTests(cmd.OutOrStdout(), root, testOptions{FailFast: testFailFast})
	},
}

func init() {
	testCmd.Flags().BoolVar(&testFailFast, "fail-fast", false, "stop after the first failing test")
	rootCmd.AddCommand(testCmd)
}

type testOptions struct {
	FailFast bool
}

type testCase struct {
	Path         string
	Name         string
	InternalName string
}

type testResult struct {
	Test   testCase
	Passed bool
	Output string
	Err    error
}

type failedTestDetails struct {
	Path   string
	Name   string
	Output string
	Err    error
}

func runTests(out io.Writer, root string, opts testOptions) error {
	files, err := testFiles(root)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		_, err := fmt.Fprintln(out, "no tests found")
		return err
	}

	passed := 0
	failed := 0
	var failures []failedTestDetails
	for _, path := range files {
		tests, err := loadTestCases(path)
		if err != nil {
			failed++
			printFileFailureLine(out, path)
			failures = append(failures, failedTestDetails{Path: path, Err: err})
			if opts.FailFast {
				printTestSummary(out, passed, failed)
				printFailureDetails(out, failures)
				return errors.New(colorRed(fmt.Sprintf("%d test(s) failed", failed)))
			}
			continue
		}

		for _, tc := range tests {
			result := runSingleTest(tc)
			if result.Passed {
				passed++
				fmt.Fprintln(out, colorGreen(fmt.Sprintf("PASS %s %s", result.Test.Path, result.Test.Name)))
				continue
			}

			failed++
			printTestFailureLine(out, result)
			failures = append(failures, failedTestDetails{
				Path:   result.Test.Path,
				Name:   result.Test.Name,
				Output: result.Output,
				Err:    result.Err,
			})
			if opts.FailFast {
				printTestSummary(out, passed, failed)
				printFailureDetails(out, failures)
				return errors.New(colorRed(fmt.Sprintf("%d test(s) failed", failed)))
			}
		}
	}

	printTestSummary(out, passed, failed)
	if failed > 0 {
		printFailureDetails(out, failures)
		return errors.New(colorRed(fmt.Sprintf("%d test(s) failed", failed)))
	}
	return nil
}

func testFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", root, err)
	}
	if !info.IsDir() {
		if !isTestSource(root) {
			return nil, fmt.Errorf("test expects a .test.tx file, got %s", root)
		}
		return []string{root}, nil
	}

	var paths []string
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "bin", "tmp":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if isTestSource(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func loadTestCases(path string) ([]testCase, error) {
	loaded, err := modules.LoadTest(path)
	if err != nil {
		return nil, formatSourceError(path, "", nil, err)
	}

	if err := rejectEntryMain(loaded); err != nil {
		return nil, err
	}

	var tests []testCase
	for _, fn := range loaded.Program.Functions {
		if !isEntryFunction(loaded, fn) {
			continue
		}
		name := displayTestName(fn.Name)
		if !strings.HasPrefix(name, "test") {
			continue
		}
		if err := validateTestFunction(path, fn, name); err != nil {
			return nil, err
		}
		tests = append(tests, testCase{
			Path:         path,
			Name:         name,
			InternalName: fn.Name,
		})
	}
	sort.Slice(tests, func(i, j int) bool {
		return tests[i].Name < tests[j].Name
	})
	if len(tests) == 0 {
		return nil, fmt.Errorf("%s: no test functions found", path)
	}
	return tests, nil
}

func runSingleTest(tc testCase) testResult {
	result := testResult{Test: tc}
	compiled, err := compileTestHarness(tc)
	if err != nil {
		result.Err = err
		return result
	}

	var output bytes.Buffer
	err = runCompiledResult(&output, nil, compiled)
	result.Output = output.String()
	if err != nil {
		result.Err = err
		return result
	}

	result.Passed = true
	return result
}

func compileTestHarness(tc testCase) (*compileResult, error) {
	loaded, err := modules.LoadTest(tc.Path)
	if err != nil {
		return nil, formatSourceError(tc.Path, "", nil, err)
	}
	if err := rejectEntryMain(loaded); err != nil {
		return nil, err
	}

	found := false
	for _, fn := range loaded.Program.Functions {
		if !isEntryFunction(loaded, fn) {
			continue
		}
		name := displayTestName(fn.Name)
		if !strings.HasPrefix(name, "test") {
			continue
		}
		if err := validateTestFunction(tc.Path, fn, name); err != nil {
			return nil, err
		}
		if fn.Name == tc.InternalName {
			found = true
		}
		fn.ReturnType = ast.IntType
		fn.Body.Statements = append(fn.Body.Statements, &ast.ReturnStmt{
			Start: fn.Body.End,
			Value: &ast.IntLiteral{Start: fn.Body.End, Value: "0"},
		})
	}
	if !found {
		return nil, fmt.Errorf("%s: test %s not found", tc.Path, tc.Name)
	}

	loaded.Program.Functions = append(loaded.Program.Functions, syntheticMain(tc.InternalName, token.Position{
		File:   loaded.EntryPath,
		Line:   1,
		Column: 1,
	}))
	return compileLoaded(tc.Path, loaded, compileOptions{})
}

func syntheticMain(testName string, pos token.Position) *ast.FuncDecl {
	return &ast.FuncDecl{
		Pos:        pos,
		NamePos:    pos,
		Name:       "main",
		ReturnType: ast.IntType,
		Body: ast.Block{
			Start: pos,
			End:   pos,
			Statements: []ast.Statement{
				&ast.ExprStmt{Expr: &ast.CallExpr{Start: pos, Callee: testName}},
				&ast.ReturnStmt{Start: pos, Value: &ast.IntLiteral{Start: pos, Value: "0"}},
			},
		},
	}
}

func validateTestFunction(path string, fn *ast.FuncDecl, name string) error {
	if fn.Kernel {
		return fmt.Errorf("%s:%d:%d: test function %s must not be a kernel", path, fn.NamePos.Line, fn.NamePos.Column, name)
	}
	if len(fn.Params) != 0 {
		return fmt.Errorf("%s:%d:%d: test function %s must not have parameters", path, fn.NamePos.Line, fn.NamePos.Column, name)
	}
	if fn.ReturnType != nil {
		return fmt.Errorf("%s:%d:%d: test function %s must not declare a return type", path, fn.NamePos.Line, fn.NamePos.Column, name)
	}
	if hasReturn(fn.Body.Statements) {
		return fmt.Errorf("%s:%d:%d: test function %s must not return explicitly", path, fn.NamePos.Line, fn.NamePos.Column, name)
	}
	return nil
}

func hasReturn(stmts []ast.Statement) bool {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *ast.ReturnStmt:
			return true
		case *ast.IfStmt:
			if hasReturn(stmt.Then.Statements) {
				return true
			}
			if stmt.Else != nil && hasReturn(stmt.Else.Statements) {
				return true
			}
		case *ast.ForStmt:
			if stmt.Init != nil && stmtHasReturn(stmt.Init) {
				return true
			}
			if stmt.Post != nil && stmtHasReturn(stmt.Post) {
				return true
			}
			if hasReturn(stmt.Body.Statements) {
				return true
			}
		}
	}
	return false
}

func stmtHasReturn(stmt ast.Statement) bool {
	switch stmt := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		return hasReturn(stmt.Then.Statements) || (stmt.Else != nil && hasReturn(stmt.Else.Statements))
	case *ast.ForStmt:
		return hasReturn(stmt.Body.Statements)
	default:
		return false
	}
}

func rejectEntryMain(loaded *modules.Result) error {
	for _, fn := range loaded.Program.Functions {
		if isEntryFunction(loaded, fn) && displayTestName(fn.Name) == "main" {
			return fmt.Errorf("%s:%d:%d: test file must not define main", loaded.EntryPath, fn.NamePos.Line, fn.NamePos.Column)
		}
	}
	return nil
}

func isEntryFunction(loaded *modules.Result, fn *ast.FuncDecl) bool {
	return filepath.Clean(fn.NamePos.File) == loaded.EntryPath
}

func displayTestName(name string) string {
	return strings.TrimPrefix(name, "__trux_mod_0_")
}

func isTestSource(path string) bool {
	return strings.HasSuffix(filepath.Base(path), ".test.tx")
}

func printFileFailureLine(out io.Writer, path string) {
	fmt.Fprintln(out, colorRed(fmt.Sprintf("FAIL %s", path)))
}

func printTestFailureLine(out io.Writer, result testResult) {
	fmt.Fprintln(out, colorRed(fmt.Sprintf("FAIL %s %s", result.Test.Path, result.Test.Name)))
}

func printFailureDetails(out io.Writer, failures []failedTestDetails) {
	if len(failures) == 0 {
		return
	}
	fmt.Fprintln(out, colorRed("failures:"))
	for _, failure := range failures {
		if failure.Name == "" {
			fmt.Fprintf(out, "\n%s\n", colorRed(fmt.Sprintf("FAIL %s", failure.Path)))
		} else {
			fmt.Fprintf(out, "\n%s\n", colorRed(fmt.Sprintf("FAIL %s %s", failure.Path, failure.Name)))
		}
		printTrace(out, failure)
		if strings.TrimSpace(failure.Output) != "" {
			fmt.Fprintf(out, "stdout:\n%s", failure.Output)
			if !strings.HasSuffix(failure.Output, "\n") {
				fmt.Fprintln(out)
			}
		}
		if failure.Err != nil {
			fmt.Fprintf(out, "%s\n%s\n", colorRed("error:"), cleanTestError(failure.Err))
		}
	}
}

func printTrace(out io.Writer, failure failedTestDetails) {
	fmt.Fprintln(out, "trace:")
	if failure.Name == "" {
		fmt.Fprintf(out, "  at %s\n", failure.Path)
		return
	}
	fmt.Fprintf(out, "  at %s (%s)\n", failure.Name, failure.Path)
	for _, line := range strings.Split(cleanTestError(failure.Err), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "at ") {
			fmt.Fprintf(out, "  %s\n", line)
		}
	}
}

func printTestSummary(out io.Writer, passed int, failed int) {
	line := fmt.Sprintf("%d passed, %d failed", passed, failed)
	if failed > 0 {
		fmt.Fprintln(out, colorRed(line))
		return
	}
	fmt.Fprintln(out, colorGreen(line))
}

func cleanTestError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if strings.HasPrefix(msg, "execute ") {
		if _, rest, ok := strings.Cut(msg, "\n"); ok {
			trimmed := strings.TrimSpace(rest)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return msg
}

func colorGreen(text string) string {
	return testColorGreen + text + testColorReset
}

func colorRed(text string) string {
	return testColorRed + text + testColorReset
}
