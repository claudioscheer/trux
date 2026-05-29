package main

import (
	"fmt"
	"io"
	"os"

	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/token"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <file.tx>",
	Short: "Compile and run a trux program",
	Long:  "Compile the given .tx file to C, invoke the C compiler, execute the resulting binary, and print its output.",
	Args:  cobra.ExactArgs(1),
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

	for _, tok := range lexer.Lex(string(src)) {
		fmt.Fprintf(out, "%d:%d %s %q\n", tok.Pos.Line, tok.Pos.Column, tok.Type, tok.Lexeme)
		if tok.Type == token.Illegal {
			return fmt.Errorf("lex error at %d:%d: illegal character %q", tok.Pos.Line, tok.Pos.Column, tok.Lexeme)
		}
	}

	return nil
}
