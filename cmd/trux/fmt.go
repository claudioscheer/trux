package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudioscheer/trux/internal/formatter"
	"github.com/spf13/cobra"
)

var fmtRecursive bool

var fmtCmd = &cobra.Command{
	Use:          "fmt [-r] [file.tx]",
	Short:        "Format trux source files",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if fmtRecursive {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			return formatRecursive(root)
		}

		if len(args) != 1 {
			return fmt.Errorf("fmt requires a file unless -r is set")
		}

		return formatPath(args[0])
	},
}

func init() {
	fmtCmd.Flags().BoolVarP(&fmtRecursive, "recursive", "r", false, "format .tx files recursively")
	rootCmd.AddCommand(fmtCmd)
}

func formatPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("read %s: is a directory", path)
	}
	if filepath.Ext(path) != ".tx" {
		return fmt.Errorf("fmt expects a .tx file, got %s", path)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	formatted, err := formatter.Format(path, string(src))
	if err != nil {
		return formatSourceError(path, string(src), nil, err)
	}
	if formatted == string(src) {
		return nil
	}

	if err := os.WriteFile(path, []byte(formatted), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func formatRecursive(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("read %s: %w", root, err)
	}
	if !info.IsDir() {
		return formatPath(root)
	}

	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if filepath.Ext(path) != ".tx" {
			return nil
		}
		return formatPath(path)
	})
}
