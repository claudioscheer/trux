package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

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
	UsesSQLite        bool
	UsesJSON          bool
	UsesJWT           bool
	Debug             bool
}

type runtimeHeader struct {
	Name   string
	Source string
}

type generatedCompileOptions struct {
	UsesCUDA   bool
	UsesHTTP   bool
	UsesSQLite bool
	UsesJSON   bool
	UsesJWT    bool
	Debug      bool
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
	if typedIR.UsesSQLite {
		headers = append(headers, runtimeHeader{Name: runtimec.SQLiteHeaderName, Source: runtimec.SQLiteHeaderSource})
		headers = append(headers, runtimeHeader{Name: "sqlite3.h", Source: runtimec.SQLite3HeaderSource})
	}
	if typedIR.UsesJSON {
		headers = append(headers, runtimeHeader{Name: runtimec.JSONHeaderName, Source: runtimec.JSONHeaderSource})
	}
	if typedIR.UsesJSON || typedIR.UsesJWT {
		headers = append(headers, runtimeHeader{Name: "yyjson.h", Source: runtimec.YYJSONHeaderSource})
	}
	if typedIR.UsesJWT {
		headers = append(headers, runtimeHeader{Name: runtimec.JWTHeaderName, Source: runtimec.JWTHeaderSource})
	}

	return &compileResult{
		CSource:           cSource,
		RuntimeHeaderName: runtimec.HeaderName,
		RuntimeHeader:     runtimec.Source,
		RuntimeHeaders:    headers,
		UsesCUDA:          len(typedIR.Kernels) > 0,
		UsesHTTP:          typedIR.UsesHTTP,
		UsesSQLite:        typedIR.UsesSQLite,
		UsesJSON:          typedIR.UsesJSON,
		UsesJWT:           typedIR.UsesJWT,
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

	return compileGenerated(cPath, outputPath, generatedCompileOptions{
		UsesCUDA:   result.UsesCUDA,
		UsesHTTP:   result.UsesHTTP,
		UsesSQLite: result.UsesSQLite,
		UsesJSON:   result.UsesJSON,
		UsesJWT:    result.UsesJWT,
		Debug:      result.Debug,
	})
}

func compileC(sourcePath string, outputPath string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read generated source: %w", err)
	}
	return compileGenerated(sourcePath, outputPath, generatedCompileOptions{
		UsesHTTP:   strings.Contains(string(content), runtimec.HTTPHeaderName),
		UsesSQLite: strings.Contains(string(content), runtimec.SQLiteHeaderName),
		UsesJSON:   strings.Contains(string(content), runtimec.JSONHeaderName),
		UsesJWT:    strings.Contains(string(content), runtimec.JWTHeaderName),
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
		args = []string{"-std=c11", "-O2", "-march=native", "-mtune=native", sourcePath, "-o", outputPath}
	}
	if opts.UsesCUDA {
		compiler = os.Getenv("NVCC")
		if compiler == "" {
			compiler = "nvcc"
		}
		// The generated file is .cu, so nvcc treats it as CUDA. A global `-x cu`
		// would also force later inputs, such as the yyjson object, to be compiled
		// as source.
		args = []string{sourcePath, "-o", outputPath}
		// nvcc leaves host code at -O0 unless asked. Match the C path so host
		// loops (argmax, comparisons, CPU reference kernels) are optimized too.
		if opts.Debug {
			args = append(args, "-O0")
		} else {
			args = append(args, "-O2", "-arch=native", "-use_fast_math", "--extra-device-vectorization", "-Xcompiler", "-O2,-march=native")
		}
		if opts.UsesHTTP {
			// nvcc rejects -pthread ("Unknown option '-pthread'" on CUDA 12).
			// The host compiler accepts it through -Xcompiler.
			args = append(args, "-D_GNU_SOURCE", "-Xcompiler", "-pthread", "-luring")
		}
		if opts.UsesSQLite {
			host := os.Getenv("CC")
			if host == "" {
				host = "cc"
			}
			sqliteObj, err := ensureSQLiteObjectCached(host)
			if err != nil {
				return err
			}
			args = append(args, sqliteObj)
			if !opts.UsesHTTP {
				args = append(args, "-Xcompiler", "-pthread")
			}
		}
		if opts.UsesJSON || opts.UsesJWT {
			host := os.Getenv("CC")
			if host == "" {
				host = "cc"
			}
			yyjsonObj, err := ensureYYJSONObjectCached(host)
			if err != nil {
				return err
			}
			args = append(args, yyjsonObj)
		}
		if opts.UsesJWT {
			prefix := jwtPrefix()
			args = appendJWTLinkArgs(args, prefix, true)
		}
	} else {
		if opts.UsesHTTP {
			args = append(args, "-D_GNU_SOURCE", "-pthread", "-luring")
		}
		if opts.UsesSQLite {
			sqliteObj, err := ensureSQLiteObjectCached(compiler)
			if err != nil {
				return err
			}
			args = append(args, sqliteObj, "-pthread")
		}
		if opts.UsesJSON || opts.UsesJWT {
			yyjsonObj, err := ensureYYJSONObjectCached(compiler)
			if err != nil {
				return err
			}
			args = append(args, yyjsonObj)
		}
		if opts.UsesJWT {
			prefix := jwtPrefix()
			args = appendJWTLinkArgs(args, prefix, false)
		}
		// Host math calls (math.exp, math.sqrt) lower to libm functions.
		// Keep -lm after any cached object files so the linker resolves them.
		args = append(args, "-lm")
	}

	cmd := exec.Command(compiler, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		var compileErr error
		if msg == "" {
			compileErr = fmt.Errorf("%s failed: %w", compiler, err)
		} else {
			compileErr = fmt.Errorf("%s failed: %w\n%s", compiler, err, msg)
		}
		if opts.UsesJWT {
			return fmt.Errorf("%w\n\njwt programs need libjwt 3, jansson, and OpenSSL. Install them, or set TRUX_LIBJWT_PREFIX to a directory with include/jwt.h and lib/libjwt.a.", compileErr)
		}
		return compileErr
	}

	return nil
}

func jwtPrefix() string {
	if p := strings.TrimSpace(os.Getenv("TRUX_LIBJWT_PREFIX")); p != "" {
		return p
	}
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		return ""
	}
	p := filepath.Join(base, "trux", "libjwt")
	header := filepath.Join(p, "include", "jwt.h")
	lib := filepath.Join(p, "lib", "libjwt.a")
	if info, statErr := os.Stat(header); statErr != nil || info.IsDir() {
		return ""
	}
	if info, statErr := os.Stat(lib); statErr != nil || info.IsDir() {
		return ""
	}
	return p
}

func appendJWTLinkArgs(args []string, prefix string, nvcc bool) []string {
	if prefix != "" {
		args = append(args,
			"-I"+filepath.Join(prefix, "include"),
			"-L"+filepath.Join(prefix, "lib"),
			"-DJWT_STATIC_DEFINE",
		)
		rpath := "-rpath," + filepath.Join(prefix, "lib")
		if nvcc {
			// nvcc rejects -Wl,... ("Unknown option"). -Xlinker forwards one
			// option to the host linker.
			args = append(args, "-Xlinker", rpath)
		} else {
			args = append(args, "-Wl,"+rpath)
		}
	}
	if nvcc {
		args = append(args, "-Xlinker", "-Bstatic", "-ljwt", "-Xlinker", "-Bdynamic", "-ljansson", "-lcrypto")
	} else {
		args = append(args, "-Wl,-Bstatic", "-ljwt", "-Wl,-Bdynamic", "-ljansson", "-lcrypto")
	}
	return args
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

var sqliteCacheMu sync.Mutex

func ensureSQLiteObjectCached(compiler string) (string, error) {
	sqliteCacheMu.Lock()
	defer sqliteCacheMu.Unlock()

	baseDir, err := os.UserCacheDir()
	if err != nil || baseDir == "" {
		baseDir = os.TempDir()
	}
	cacheDir := filepath.Join(baseDir, "trux", "sqlite")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create sqlite cache dir: %w", err)
	}

	compileFlags := []string{
		"-c",
		"-O3",
		"-DNDEBUG",
		"-DSQLITE_THREADSAFE=2",
		"-DSQLITE_DEFAULT_MEMSTATUS=0",
		"-DSQLITE_DEFAULT_WAL_SYNCHRONOUS=1",
		"-DSQLITE_LIKE_DOESNT_MATCH_BLOBS",
		"-DSQLITE_MAX_EXPR_DEPTH=0",
		"-DSQLITE_OMIT_DECLTYPE",
		"-DSQLITE_OMIT_DEPRECATED",
		"-DSQLITE_OMIT_PROGRESS_CALLBACK",
		"-DSQLITE_OMIT_SHARED_CACHE",
		"-DSQLITE_USE_ALLOCA",
		"-DSQLITE_ENABLE_FTS5",
		"-DSQLITE_ENABLE_RTREE",
		"-DSQLITE_OMIT_LOAD_EXTENSION",
	}

	h := sha256.New()
	h.Write([]byte(compiler))
	h.Write([]byte(strings.Join(compileFlags, " ")))
	h.Write([]byte(runtimec.SQLite3CSource))
	sum := hex.EncodeToString(h.Sum(nil))[:16]
	objName := fmt.Sprintf("sqlite3-3.47.2-%s.o", sum)
	objPath := filepath.Join(cacheDir, objName)

	if info, err := os.Stat(objPath); err == nil && info.Size() > 0 {
		return objPath, nil
	}

	cPath := filepath.Join(cacheDir, fmt.Sprintf("sqlite3-%s.c", sum))
	hPath := filepath.Join(cacheDir, "sqlite3.h")
	if err := os.WriteFile(hPath, []byte(runtimec.SQLite3HeaderSource), 0o644); err != nil {
		return "", fmt.Errorf("write sqlite3.h: %w", err)
	}
	if err := os.WriteFile(cPath, []byte(runtimec.SQLite3CSource), 0o644); err != nil {
		return "", fmt.Errorf("write sqlite3.c: %w", err)
	}

	tmpObjPath := filepath.Join(cacheDir, fmt.Sprintf("sqlite3-%s.tmp-%d.o", sum, os.Getpid()))
	args := append(append([]string(nil), compileFlags...), cPath, "-o", tmpObjPath)

	cmd := exec.Command(compiler, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpObjPath)
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return "", fmt.Errorf("precompile sqlite3.c failed: %w", err)
		}
		return "", fmt.Errorf("precompile sqlite3.c failed: %w\n%s", err, msg)
	}

	_ = os.Remove(objPath)
	if err := os.Rename(tmpObjPath, objPath); err != nil {
		if info, statErr := os.Stat(objPath); statErr == nil && info.Size() > 0 {
			_ = os.Remove(tmpObjPath)
			return objPath, nil
		}
		return "", fmt.Errorf("cache sqlite object: %w", err)
	}
	_ = os.Remove(cPath)

	return objPath, nil
}

var yyjsonCacheMu sync.Mutex

func ensureYYJSONObjectCached(compiler string) (string, error) {
	yyjsonCacheMu.Lock()
	defer yyjsonCacheMu.Unlock()

	baseDir, err := os.UserCacheDir()
	if err != nil || baseDir == "" {
		baseDir = os.TempDir()
	}
	cacheDir := filepath.Join(baseDir, "trux", "yyjson")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create yyjson cache dir: %w", err)
	}

	compileFlags := []string{
		"-c",
		"-O3",
		"-DNDEBUG",
	}

	h := sha256.New()
	h.Write([]byte(compiler))
	h.Write([]byte(strings.Join(compileFlags, " ")))
	h.Write([]byte(runtimec.YYJSONCSource))
	sum := hex.EncodeToString(h.Sum(nil))[:16]
	objName := fmt.Sprintf("yyjson-0.11.1-%s.o", sum)
	objPath := filepath.Join(cacheDir, objName)

	if info, err := os.Stat(objPath); err == nil && info.Size() > 0 {
		return objPath, nil
	}

	cPath := filepath.Join(cacheDir, fmt.Sprintf("yyjson-%s.c", sum))
	hPath := filepath.Join(cacheDir, "yyjson.h")
	if err := os.WriteFile(hPath, []byte(runtimec.YYJSONHeaderSource), 0o644); err != nil {
		return "", fmt.Errorf("write yyjson.h: %w", err)
	}
	if err := os.WriteFile(cPath, []byte(runtimec.YYJSONCSource), 0o644); err != nil {
		return "", fmt.Errorf("write yyjson.c: %w", err)
	}

	tmpObjPath := filepath.Join(cacheDir, fmt.Sprintf("yyjson-%s.tmp-%d.o", sum, os.Getpid()))
	args := append(append([]string(nil), compileFlags...), cPath, "-o", tmpObjPath)

	cmd := exec.Command(compiler, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpObjPath)
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return "", fmt.Errorf("precompile yyjson.c failed: %w", err)
		}
		return "", fmt.Errorf("precompile yyjson.c failed: %w\n%s", err, msg)
	}

	_ = os.Remove(objPath)
	if err := os.Rename(tmpObjPath, objPath); err != nil {
		if info, statErr := os.Stat(objPath); statErr == nil && info.Size() > 0 {
			_ = os.Remove(tmpObjPath)
			return objPath, nil
		}
		return "", fmt.Errorf("cache yyjson object: %w", err)
	}
	_ = os.Remove(cPath)

	return objPath, nil
}
