package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/claudioscheer/trux/internal/modules"
	"github.com/claudioscheer/trux/internal/parser"
	"github.com/claudioscheer/trux/internal/token"
	semtypes "github.com/claudioscheer/trux/internal/types"
	"github.com/spf13/cobra"
)

var runDebug bool

var runCmd = &cobra.Command{
	Use:          "run <file.tx>",
	Short:        "Compile and run a trux program",
	Long:         "Compile the given .tx file to C, invoke the C compiler, execute the resulting binary, and print its output.",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFileWithOptions(cmd.OutOrStdout(), args[0], runOptions{Debug: runDebug, Stdin: cmd.InOrStdin()})
	},
}

func init() {
	runCmd.Flags().BoolVar(&runDebug, "debug", false, "write per-phase compiler output to tmp/trux-debug")
	rootCmd.AddCommand(runCmd)
}

type runOptions struct {
	Debug bool
	Stdin io.Reader
}

func runFile(out io.Writer, path string) error {
	return runFileWithOptions(out, path, runOptions{})
}

func runFileWithOptions(out io.Writer, path string, opts runOptions) error {
	result, err := compileFile(path, compileOptions{Debug: opts.Debug, DebugOut: out})
	if err != nil {
		return err
	}

	return runCompiledResult(out, opts.Stdin, result)
}

func runCompiledResult(out io.Writer, in io.Reader, result *compileResult) error {
	tmpDir, err := os.MkdirTemp("", "trux-run-*")
	if err != nil {
		return fmt.Errorf("create run temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cPath, err := writeGeneratedFiles(tmpDir, "main", result)
	if err != nil {
		return err
	}
	executablePath := filepath.Join(tmpDir, "main")
	if err := compileGenerated(cPath, executablePath, generatedCompileOptions{UsesCUDA: result.UsesCUDA, Debug: result.Debug}); err != nil {
		return err
	}

	return runExecutable(out, in, executablePath)
}

func formatSourceError(path string, src string, sources map[string]string, err error) error {
	var moduleErr *modules.Error
	if errors.As(err, &moduleErr) {
		return formatErrorAt(moduleErr.File, moduleErr.Source, moduleErr.Pos, moduleErr.Msg)
	}

	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		var typeErr *semtypes.Error
		if !errors.As(err, &typeErr) {
			return err
		}

		errorPath, errorSource := sourceForPosition(path, src, sources, typeErr.Pos)
		return formatErrorAt(errorPath, errorSource, typeErr.Pos, typeErr.Msg)
	}

	errorPath, errorSource := sourceForPosition(path, src, sources, parseErr.Pos)
	return formatErrorAt(errorPath, errorSource, parseErr.Pos, parseErr.Msg)
}

func sourceForPosition(path string, src string, sources map[string]string, pos token.Position) (string, string) {
	if pos.File == "" {
		return path, src
	}
	if sources != nil {
		if source, ok := sources[pos.File]; ok {
			return pos.File, source
		}
	}

	return pos.File, src
}

func formatErrorAt(path string, src string, pos token.Position, msg string) error {
	return fmt.Errorf("%s:%d:%d: %s\n%s",
		path,
		pos.Line,
		pos.Column,
		msg,
		sourceContext(src, pos.Line, pos.Column, 2),
	)
}

func sourceContext(src string, lineNumber int, column int, radius int) string {
	if lineNumber < 1 {
		return ""
	}

	lines := strings.Split(src, "\n")
	if lineNumber > len(lines) {
		return ""
	}

	start := lineNumber - radius
	if start < 1 {
		start = 1
	}

	end := lineNumber + radius
	if end > len(lines) {
		end = len(lines)
	}

	width := len(strconv.Itoa(end))
	caretColumn := column
	if caretColumn < 1 {
		caretColumn = 1
	}

	var out bytes.Buffer
	for line := start; line <= end; line++ {
		fmt.Fprintf(&out, "%*d | %s\n", width, line, lines[line-1])
		if line == lineNumber {
			fmt.Fprintf(&out, "%s | %s^\n",
				strings.Repeat(" ", width),
				strings.Repeat(" ", caretColumn-1),
			)
		}
	}

	return strings.TrimRight(out.String(), "\n")
}
