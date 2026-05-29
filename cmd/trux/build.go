package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build <file.tx>",
	Short: "Compile a trux program to an executable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		fmt.Printf("trux build: not yet implemented (would produce executable from %s)\n", src)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
