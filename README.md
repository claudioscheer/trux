# trux

`trux` is a small Go-inspired language that compiles to C.

The goal is to learn compiler construction by building the full pipeline by hand:

```text
source code
  -> lexer
  -> parser
  -> AST
  -> type checker
  -> typed IR
  -> C code
  -> cc (or $CC)
  -> executable
```

`trux` is not Go-compatible. It only borrows simple syntax ideas from Go.

## Current Status

The compiler currently supports:

- `package main`
- `func`, parameters, return types, and function calls
- `let` bindings with explicit types
- `int`, `float`, `string`, and `bool`
- integer and float arithmetic with `+`, `-`, `*`, and `/`
- comparison operators with `==`, `!=`, `<`, `<=`, `>`, and `>=`
- `if`, `else`, and `while`
- assignment to existing variables
- string and boolean literals
- string containment with `needle in haystack`
- string concatenation with `+`
- fixed arrays `[N]T`, slices `[]T`, and mutable lists `list[T]` for scalar element types
- array and list literals, `make([]T, n)`, `len(x)`, indexing, slicing, list `append`, and indexed assignment
- explicit `clone(x)` for owned copies of strings, arrays, slices, and lists
- relative module imports with `import "path/to/file.tx"` across files that may declare different package names
- `pub func` exports called as `package.function(...)`, file-local private functions, and import-cycle detection
- typed IR
- C code generation and execution through `cc` or `$CC`
- `print(...)` with one or more `int`, `float`, `string`, or `bool` arguments
- stdin, file, and CSV IO through builtins

Directory-based packages, separate module compilation, reusable package artifacts, and GPU kernels are not implemented.

## Building

```bash
make build
```

This produces `bin/trux`.

## Running

Run a source file:

```bash
make run FILE=examples/hello.tx
```

Run with debug output for each compiler phase:

```bash
make run FILE=examples/hello.tx DEBUG=1
```

Debug files are written to `tmp/trux-debug/<source-name>/`.

## Examples

Minimal integer program:

```go trux
package main

func add(a int, b int) int {
    return a + b
}

func main() int {
    let x int = add(1, 2)
    print(x)
    return 0
}
```

Expected output:

```text
3
```

Strings, booleans, floats, and multi-argument print:

```go trux
package main

func mean(a float, b float) float {
    return (a + b) / 2.0
}

func main() int {
    let name string = "trux"
    let stable bool = false
    let average float = mean(1.5, 2.5)

    print(name, " ", stable)
    print(average)

    return 0
}
```

Expected output:

```text
trux false
2
```

More examples live in [examples/](examples/). Start with [examples/hello.tx](examples/hello.tx), then read [examples/collections.tx](examples/collections.tx) and [examples/ownership_clone.tx](examples/ownership_clone.tx) for the current collection and ownership model. Module examples live under [examples/modules/](examples/modules/).

The IO basics project reads a name and number from stdin, reads/writes text files, and reads/writes flat row-major CSV:

```bash
printf 'Ada\n41\n' | make run FILE=examples/projects/io_basics/main.tx
```

More complete IO examples live under [examples/projects/](examples/projects/):

- `io_basics`: stdin, file IO, and CSV IO in one small program
- `interactive_counter`: stdin with `read_line` and `read_int`
- `file_notes`: `read_file` and `write_file`
- `csv_roster`: `read_csv`, list mutation, and `write_csv`

## Commands

```bash
make test
make emit-c FILE=examples/hello.tx
make build-bin FILE=examples/hello.tx OUT=bin/hello
./bin/trux fmt examples/hello.tx
./bin/trux fmt -r
make lsp
```

## Documentation

See [docs/SPECS.md](docs/SPECS.md) and the other docs in [docs/](docs/) for design decisions, including arenas, IO, modules, and future GPU ideas.
Editor tooling lives in [tooling/](tooling/), including the Go language server and VS Code extension.
