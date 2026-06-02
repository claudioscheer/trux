package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	codegenc "github.com/claudioscheer/trux/internal/codegen/c"
	"github.com/claudioscheer/trux/internal/ir"
	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/modules"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

type compileOptions struct {
	Debug    bool
	DebugOut io.Writer
}

type compileResult struct {
	CSource  string
	UsesCUDA bool
}

func compileFile(path string, opts compileOptions) (*compileResult, error) {
	loaded, err := modules.Load(path)
	if err != nil {
		return nil, formatSourceError(path, "", nil, err)
	}
	return compileLoaded(path, loaded, opts)
}

func compileLoaded(path string, loaded *modules.Result, opts compileOptions) (*compileResult, error) {
	src := loaded.Sources[loaded.EntryPath]
	if src == "" {
		return nil, fmt.Errorf("missing loaded source for %s", loaded.EntryPath)
	}

	var err error
	var debug *debugWriter
	if opts.Debug {
		debug, err = newDebugWriter(path)
		if err != nil {
			return nil, err
		}
		if err := debug.writeText("00-source.tx", string(src)); err != nil {
			return nil, err
		}
		tokens := lexer.LexFile(loaded.EntryPath, string(src))
		if err := debug.writeText("01-tokens.txt", formatTokens(tokens)); err != nil {
			return nil, err
		}
		if opts.DebugOut != nil {
			fmt.Fprintf(opts.DebugOut, "debug files: %s\n", debug.dir)
		}
	}

	program := loaded.Program
	if debug != nil {
		if err := debug.writeJSON("02-ast.json", program); err != nil {
			return nil, err
		}
	}

	info, err := semtypes.Check(program)
	if err != nil {
		return nil, formatSourceError(path, string(src), loaded.Sources, err)
	}
	if debug != nil {
		if err := debug.writeText("03-types.txt", formatTypeInfo(program, info)); err != nil {
			return nil, err
		}
	}

	typedIR, err := ir.Build(program, info)
	if err != nil {
		return nil, err
	}
	if debug != nil {
		if err := debug.writeJSON("04-ir.json", typedIR); err != nil {
			return nil, err
		}
	}

	cSource, err := codegenc.Generate(typedIR)
	if err != nil {
		return nil, err
	}
	if debug != nil {
		if err := debug.writeText("05-c.c", cSource); err != nil {
			return nil, err
		}
	}

	return &compileResult{CSource: cSource, UsesCUDA: len(typedIR.Kernels) > 0}, nil
}

func buildFile(sourcePath string, outputPath string) error {
	result, err := compileFile(sourcePath, compileOptions{})
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "trux-build-*")
	if err != nil {
		return fmt.Errorf("create build temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cPath := filepath.Join(tmpDir, "main.c")
	if result.UsesCUDA {
		cPath = filepath.Join(tmpDir, "main.cu")
	}
	if err := os.WriteFile(cPath, []byte(result.CSource), 0o644); err != nil {
		return fmt.Errorf("write generated source: %w", err)
	}

	return compileGenerated(cPath, outputPath, result.UsesCUDA)
}

func compileC(sourcePath string, outputPath string) error {
	return compileGenerated(sourcePath, outputPath, false)
}

func compileGenerated(sourcePath string, outputPath string, usesCUDA bool) error {
	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "cc"
	}
	args := []string{"-std=c11", sourcePath, "-o", outputPath}
	if usesCUDA {
		compiler = os.Getenv("NVCC")
		if compiler == "" {
			compiler = "nvcc"
		}
		args = []string{"-x", "cu", sourcePath, "-o", outputPath}
	}

	cmd := exec.Command(compiler, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return fmt.Errorf("%s failed: %w", compiler, err)
		}
		return fmt.Errorf("%s failed: %w\n%s", compiler, err, msg)
	}

	return nil
}

func runExecutable(out io.Writer, in io.Reader, executablePath string) error {
	cmd := exec.Command(executablePath)
	cmd.Stdout = out
	if in == nil {
		in = os.Stdin
	}
	cmd.Stdin = in
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return fmt.Errorf("execute %s: %w", executablePath, err)
		}
		return fmt.Errorf("execute %s: %w\n%s", executablePath, err, msg)
	}

	return nil
}
