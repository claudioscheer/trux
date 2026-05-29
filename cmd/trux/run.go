package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/claudioscheer/trux/internal/parser"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:          "run <file.tx>",
	Short:        "Compile and run a trux program",
	Long:         "Compile the given .tx file to C, invoke the C compiler, execute the resulting binary, and print its output.",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFile(cmd.OutOrStdout(), args[0])
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runFile(out io.Writer, path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	if _, err := parser.Parse(string(src)); err != nil {
		return formatSourceError(path, string(src), err)
	}

	_ = out
	return nil
}

func formatSourceError(path string, src string, err error) error {
	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	return fmt.Errorf("%s:%d:%d: %s\n%s",
		path,
		parseErr.Pos.Line,
		parseErr.Pos.Column,
		parseErr.Msg,
		sourceContext(src, parseErr.Pos.Line, parseErr.Pos.Column, 2),
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
