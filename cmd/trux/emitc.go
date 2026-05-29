package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var emitCCmd = &cobra.Command{
	Use:   "emit-c <file.tx>",
	Short: "Emit C source for a trux program (no compilation)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		fmt.Printf("trux emit-c: not yet implemented (would print C for %s)\n", src)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(emitCCmd)
}
