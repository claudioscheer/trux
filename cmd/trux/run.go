package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <file.tx>",
	Short: "Compile and run a trux program",
	Long:  "Compile the given .tx file to C, invoke the C compiler, execute the resulting binary, and print its output.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		fmt.Printf("trux run: not yet implemented (would compile and execute %s)\n", src)
		// When you implement the pipeline:
		//   1. lex + parse + typecheck + lower to IR
		//   2. codegen C
		//   3. write temp .c
		//   4. clang/gcc it
		//   5. exec the binary and stream stdout/stderr
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
