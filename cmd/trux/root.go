package main

import (
	"fmt"
	"os"

	"github.com/claudioscheer/trux/internal/language"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "trux",
	Short:         "trux is a small Go-inspired language that compiles to C",
	Version:       language.Version,
	SilenceErrors: true,
	Long: `trux is a small Go-inspired language that compiles to C.
The goal is to learn.

See https://github.com/claudioscheer/trux for the spec.`,
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
