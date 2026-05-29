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
- typed IR
- C code generation and execution through `cc` or `$CC`
- `print(...)` with one or more `int`, `float`, `string`, or `bool` arguments

Imports, modules, and string concatenation are not implemented yet.

## Building

```bash
make build
```

This produces `bin/trux`.

## Running

Run a source file:

```bash
make run FILE=examples/v1/functions.tx
```

Run with debug output for each compiler phase:

```bash
make run FILE=examples/v1/functions.tx DEBUG=1
```

Debug files are written to `tmp/trux-debug/<source-name>/`.

## Examples

Minimal integer program:

```trux
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

Strings, booleans, and multi-argument print:

```trux
package main

func project() string {
    return "trux"
}

func stable() bool {
    return false
}

func main() int {
    let name string = project()
    let isStable bool = stable()

    print(name, " ", 1, " ", isStable)
    print(isStable)

    return 0
}
```

Expected output:

```text
trux 1 false
false
```

More examples live in [examples/](examples/).

## Commands

```bash
make test
make emit-c FILE=examples/v1/functions.tx
make build-bin FILE=examples/v1/functions.tx OUT=bin/functions
```

## Documentation

See [docs/SPECS.md](docs/SPECS.md) and the other docs in [docs/](docs/) for design decisions, including arenas, modules, and GPU support.
