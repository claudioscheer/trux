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
	runtimec "github.com/claudioscheer/trux/internal/runtime/c"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

type compileOptions struct {
	Debug    bool
	DebugOut io.Writer
}

type compileResult struct {
	CSource           string
	RuntimeHeaderName string
	RuntimeHeader     string
	RuntimeHeaders    []runtimeHeader
	UsesCUDA          bool
	UsesHTTP          bool
	Debug             bool
}

type runtimeHeader struct {
	Name   string
	Source string
}

type generatedCompileOptions struct {
	UsesCUDA bool
	UsesHTTP bool
	Debug    bool
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
	if typedIR.UsesHTTP && len(typedIR.Kernels) > 0 {
		return nil, fmt.Errorf("http and gpu cannot be used in the same program yet")
	}
	if debug != nil {
		if err := debug.writeJSON("04-ir.json", typedIR); err != nil {
			return nil, err
		}
	}

	cSource, err := codegenc.GenerateWithOptions(typedIR, codegenc.Options{
		SourceRoot: filepath.Dir(loaded.EntryPath),
	})
	if err != nil {
		return nil, err
	}
	if debug != nil {
		if err := debug.writeText("05-c.c", cSource); err != nil {
			return nil, err
		}
	}

	headers := []runtimeHeader{{Name: runtimec.HeaderName, Source: runtimec.Source}}
	if typedIR.UsesHTTP {
		headers = append(headers, runtimeHeader{Name: runtimec.HTTPHeaderName, Source: runtimec.HTTPSource})
	}

	return &compileResult{
		CSource:           cSource,
		RuntimeHeaderName: runtimec.HeaderName,
		RuntimeHeader:     runtimec.Source,
		RuntimeHeaders:    headers,
		UsesCUDA:          len(typedIR.Kernels) > 0,
		UsesHTTP:          typedIR.UsesHTTP,
		Debug:             opts.Debug,
	}, nil
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

	cPath, err := writeGeneratedFiles(tmpDir, "main", result)
	if err != nil {
		return err
	}

	return compileGenerated(cPath, outputPath, generatedCompileOptions{UsesCUDA: result.UsesCUDA, UsesHTTP: result.UsesHTTP, Debug: result.Debug})
}

func compileC(sourcePath string, outputPath string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read generated source: %w", err)
	}
	return compileGenerated(sourcePath, outputPath, generatedCompileOptions{
		UsesHTTP: strings.Contains(string(content), runtimec.HTTPHeaderName),
	})
}

func writeGeneratedFiles(dir string, stem string, result *compileResult) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	headers := result.RuntimeHeaders
	if len(headers) == 0 && result.RuntimeHeaderName != "" {
		headers = []runtimeHeader{{Name: result.RuntimeHeaderName, Source: result.RuntimeHeader}}
	}
	for _, header := range headers {
		if header.Name == "" {
			continue
		}
		headerPath := filepath.Join(dir, header.Name)
		if err := os.WriteFile(headerPath, []byte(header.Source), 0o644); err != nil {
			return "", fmt.Errorf("write runtime header %s: %w", header.Name, err)
		}
	}

	ext := ".c"
	if result.UsesCUDA {
		ext = ".cu"
	}
	sourcePath := filepath.Join(dir, stem+ext)
	if err := os.WriteFile(sourcePath, []byte(result.CSource), 0o644); err != nil {
		return "", fmt.Errorf("write generated source: %w", err)
	}
	return sourcePath, nil
}

func compileGenerated(sourcePath string, outputPath string, opts generatedCompileOptions) error {
	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "cc"
	}
	args := []string{"-std=c11", sourcePath, "-o", outputPath}
	if opts.Debug {
		args = []string{"-std=c11", "-O0", sourcePath, "-o", outputPath}
	} else {
		args = []string{"-std=c11", "-O2", sourcePath, "-o", outputPath}
	}
	if opts.UsesCUDA {
		compiler = os.Getenv("NVCC")
		if compiler == "" {
			compiler = "nvcc"
		}
		args = []string{"-x", "cu", sourcePath, "-o", outputPath}
	} else if opts.UsesHTTP {
		args = append(args, "-pthread")
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
