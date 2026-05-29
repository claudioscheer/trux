# trux

`trux` is a small Go-inspired language that compiles to C.

The goal is to learn.

## Current Status

The lexer is implemented and functional.  
`trux run` currently lexes source files and prints tokens (useful for early development and testing).

Full parsing, type checking, and C code generation are not yet implemented.

## Building

```bash
make build
```

This produces `bin/trux`.

## Running

The primary command right now is `run`, which lexes a `.tx` file and dumps tokens:

```bash
./bin/trux run examples/v0/hello.tx
```

Or using make:

```bash
make run FILE=examples/v0/hello.tx
```

### Example

Given this source ([examples/v0/hello.tx](examples/v0/hello.tx)):

```go
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

`trux run` will output tokens like:

```
1:1 PACKAGE "package"
1:9 IDENT "main"
2:1 EOF ""
...
```

It will also report errors on illegal characters.

## Documentation

See [docs/SPECS.md](docs/SPECS.md) and the other docs in [docs/](docs/) for design decisions (arenas, modules, GPU support, etc.).