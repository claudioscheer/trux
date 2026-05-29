package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var emitCCmd = &cobra.Command{
	Use:          "emit-c <file.tx>",
	Short:        "Compile a trux program to C",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := compileFile(args[0], compileOptions{})
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(cmd.OutOrStdout(), result.CSource)
		return err
	},
}

func init() {
	rootCmd.AddCommand(emitCCmd)
}
