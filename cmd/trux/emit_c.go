package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var emitCOutDir string

var emitCCmd = &cobra.Command{
	Use:          "emit-c <file.tx>",
	Short:        "Emit generated C or CUDA source for a trux program",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := compileFile(args[0], compileOptions{})
		if err != nil {
			return err
		}
		stem := strings.TrimSuffix(filepath.Base(args[0]), filepath.Ext(args[0]))
		outDir := emitCOutDir
		if outDir == "" {
			outDir = filepath.Join("out", stem)
		}
		sourcePath, err := writeGeneratedFiles(outDir, stem, result)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), sourcePath)
		if result.RuntimeHeaderName != "" {
			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(outDir, result.RuntimeHeaderName))
		}
		return nil
	},
}

func init() {
	emitCCmd.Flags().StringVar(&emitCOutDir, "out-dir", "", "directory for generated source and runtime header")
	rootCmd.AddCommand(emitCCmd)
}
