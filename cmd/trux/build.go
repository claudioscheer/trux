package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var buildOutput string

var buildCmd = &cobra.Command{
	Use:          "build -o <output> <file.tx>",
	Short:        "Compile a trux program to an executable",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if buildOutput == "" {
			return fmt.Errorf("build requires -o <output>")
		}
		return buildFile(args[0], buildOutput)
	},
}

func init() {
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "output executable path")
	rootCmd.AddCommand(buildCmd)
}
