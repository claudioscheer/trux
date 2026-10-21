<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo/trux-wordmark-light.svg">
    <img src="assets/logo/trux-wordmark.svg" alt="Trux" height="64">
  </picture>
</h1>

**Trux is a small, Go-inspired programming language that compiles to C, and to CUDA for GPU kernels.**

The compiler is written in Go and builds every stage by hand: lexer, parser, type checker, typed IR, and a C/CUDA
backend. The generated C is compiled with your system's C compiler into an ordinary native executable. Trux is a
learning project; it borrows syntax ideas from Go but is not Go-compatible.

📖 **Documentation: <https://claudioscheer.github.io/trux/>**

## A quick look

```go
package main

import "io"

func greet(name string) string {
  return "hello, " + name
}

func main() int {
  let name string = io.readLine()
  let scores list[int] = list[int]{72, 88, 95}

  let total int = 0
  for let i int = 0; i < len(scores); i = i + 1 {
    total = total + scores[i]
  }

  print(greet(name))
  print("average: ", total / len(scores))
  return 0
}
```

```console
$ echo Ada | trux run greet.tx
hello, Ada
average: 85
```

Highlights:

- explicit types: `int`, `float`, `string`, `bool`, arrays `[N]T`, slices `[]T`, and lists `list[T]`
- no implicit conversions; conditions must be `bool`
- checked integer overflow, division by zero, and bounds, reported as `trux runtime error`
- arena-based memory with simple ownership rules (`mut` parameters, `clone`) instead of a garbage collector
- multi-file programs with relative imports and `pub func`
- built-in unit tests: `.test.tx` files run by `trux test`
- standard packages for stdin and files (`io`), CSV (`csv`), PPM images (`image`), clocks (`time`), and HTTP servers (`http`)
- GPU kernels written in Trux with `kernel func`, compiled as CUDA

## Install and build

Requirements:

- [Go](https://go.dev/dl/) to build the compiler (`go.mod` declares Go 1.26.3)
- a C11 compiler, invoked as `cc` or `$CC`
- optional: the NVIDIA CUDA toolkit (`nvcc` or `$NVCC`) and an NVIDIA GPU for programs with kernels
- optional: `make` for the convenience targets

```bash
git clone https://github.com/claudioscheer/trux.git
cd trux
make build              # or: go build -o bin/trux ./cmd/trux
./bin/trux --version    # trux version 0.1.0
```

## Quick start

```bash
./bin/trux run examples/hello.tx                  # compile and run
./bin/trux build -o hello examples/hello.tx       # build an executable
./bin/trux emit-c examples/hello.tx               # write the generated C to out/hello/
./bin/trux run --debug examples/hello.tx          # dump every compiler phase to tmp/trux-debug/hello/
./bin/trux test                                   # run all .test.tx unit tests
./bin/trux fmt -r                                 # format all .tx files
```

Programs that declare a `kernel func` are emitted as CUDA and compiled with `nvcc`; everything else goes through `cc`:

```text
source -> lexer -> parser -> AST -> type checker -> typed IR -> C or CUDA -> cc/$CC or nvcc/$NVCC -> executable
```

## Documentation

The website covers everything from installation to compiler internals:

| Section | |
| --- | --- |
| [Getting started](https://claudioscheer.github.io/trux/docs/getting-started/) | install, first program, editor setup |
| [Language guide](https://claudioscheer.github.io/trux/docs/guide/) | types, functions, control flow, collections, memory, modules, tests, GPU |
| [Standard library](https://claudioscheer.github.io/trux/docs/stdlib/) | `io`, `csv`, `image`, `time`, `http`, `gpu` |
| [Examples](https://claudioscheer.github.io/trux/docs/examples/) | every program in `examples/`, with its output |
| [CLI reference](https://claudioscheer.github.io/trux/docs/cli/) | `run`, `build`, `emit-c`, `fmt`, `test` |
| [How it works](https://claudioscheer.github.io/trux/docs/internals/) | the compilation pipeline, generated C and CUDA, codebase tour |
| [FAQ](https://claudioscheer.github.io/trux/docs/faq/) | limitations and common errors |

The site is built with [Hugo](https://gohugo.io) and the [Lotus Docs](https://lotusdocs.dev) theme from the pages in
[`website/`](website/). The compact language specification is [`docs/SPECS.md`](docs/SPECS.md), and the other files in
[`docs/`](docs/) are design notes (arenas, modules, GPU, IO, HTTP, ...); the site publishes them under "Design notes".

## Project layout

```text
cmd/trux/            the trux command line tool
internal/            compiler: lexer, parser, ast, modules, types, ir, codegen/c, runtime/c, formatter, stdlib
examples/            runnable examples, module examples, and small projects
tooling/             trux-lsp language server and the VS Code extension
docs/                language specification and design notes
website/             documentation site (Hugo + Lotus Docs): config, pages, recorded example outputs
```

## Contributing

```bash
make test    # go test ./... (also compiles and runs generated C)
```

See the [contributing guide](https://claudioscheer.github.io/trux/docs/internals/contributing/) for how a language change flows
through the compiler, the project conventions, and how to preview the documentation site locally
(Hugo extended and Go: `cd website && hugo server`, then open <http://localhost:1313/trux/>).

## License

[MIT](LICENSE)
