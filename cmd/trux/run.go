package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/claudioscheer/trux/internal/ir"
	"github.com/claudioscheer/trux/internal/lexer"
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
		return runFileWithOptions(cmd.OutOrStdout(), args[0], runOptions{Debug: runDebug})
	},
}

func init() {
	runCmd.Flags().BoolVar(&runDebug, "debug", false, "write per-phase compiler output to tmp/trux-debug")
	rootCmd.AddCommand(runCmd)
}

type runOptions struct {
	Debug bool
}

func runFile(out io.Writer, path string) error {
	return runFileWithOptions(out, path, runOptions{})
}

func runFileWithOptions(out io.Writer, path string, opts runOptions) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var debug *debugWriter
	if opts.Debug {
		debug, err = newDebugWriter(path)
		if err != nil {
			return err
		}
		if err := debug.writeText("00-source.tx", string(src)); err != nil {
			return err
		}
		tokens := lexer.Lex(string(src))
		if err := debug.writeText("01-tokens.txt", formatTokens(tokens)); err != nil {
			return err
		}
		fmt.Fprintf(out, "debug files: %s\n", debug.dir)
	}

	program, err := parser.Parse(string(src))
	if err != nil {
		return formatSourceError(path, string(src), err)
	}
	if debug != nil {
		if err := debug.writeJSON("02-ast.json", program); err != nil {
			return err
		}
	}
	info, err := semtypes.Check(program)
	if err != nil {
		return formatSourceError(path, string(src), err)
	}
	if debug != nil {
		if err := debug.writeText("03-types.txt", formatTypeInfo(program, info)); err != nil {
			return err
		}
	}
	typedIR, err := ir.Build(program, info)
	if err != nil {
		return err
	}
	if debug != nil {
		if err := debug.writeJSON("04-ir.json", typedIR); err != nil {
			return err
		}
	}
	return nil
}

func formatSourceError(path string, src string, err error) error {
	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		var typeErr *semtypes.Error
		if !errors.As(err, &typeErr) {
			return err
		}

		return formatErrorAt(path, src, typeErr.Pos, typeErr.Msg)
	}

	return formatErrorAt(path, src, parseErr.Pos, parseErr.Msg)
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
