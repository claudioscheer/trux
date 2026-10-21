---
title: "Install and build"
description: "Requirements, building the compiler from source, and checking the toolchain."
icon: "download"
weight: 100
---

Trux is distributed as source. You build the `trux` compiler with Go, and `trux` then uses a C compiler on your machine to
produce executables.

## Requirements

| Tool | Needed for | Notes |
| --- | --- | --- |
| [Go](https://go.dev/dl/) | Building the compiler | `go.mod` declares `go 1.26.3`. Recent Go releases download a matching toolchain automatically when needed. |
| A C11 compiler | Running any Trux program | `trux` runs `cc` by default. Set `CC` to use another compiler, for example `CC=clang`. |
| `make` (optional) | Convenience targets | Every `make` target is a thin wrapper around `go` or `trux` commands, listed below. |
| NVIDIA CUDA toolkit (optional) | Programs that declare a `kernel func` | `trux` runs `nvcc` (or `$NVCC`) for these, and they need an NVIDIA GPU and driver to run. |
| POSIX sockets and pthreads (optional) | Programs that use the `http` package | Linux and macOS. The HTTP runtime is not available on Windows. |

## Build the compiler

```bash
git clone https://github.com/claudioscheer/trux.git
cd trux
make build
```

`make build` runs `go build -o bin/trux ./cmd/trux` and leaves the compiler at `bin/trux`. If you don't have `make`, run
that `go build` command directly.

Check that it works:

```shell-session
$ ./bin/trux --version
trux version 0.1.0
$ ./bin/trux run examples/hello.tx
3
```

{{% alert context="info" %}}

**Putting `trux` on your PATH**

The examples on this site write `trux` for short. Either call `./bin/trux` from the repository root, or copy
`bin/trux` to a directory on your `PATH`. You can also run the compiler straight from source with
`go run ./cmd/trux <command>`.

{{% /alert %}}

## Make targets

The `Makefile` has small convenience wrappers:

| Target | What it runs |
| --- | --- |
| `make build` | `go build -o bin/trux ./cmd/trux` |
| `make run FILE=path.tx` | builds, then `./bin/trux run path.tx`. Add `DEBUG=1` for `--debug`, and `RUN_FLAGS=...` for extra flags. |
| `make build-bin FILE=path.tx OUT=bin/app` | builds, then `./bin/trux build -o bin/app path.tx` |
| `make emit-c FILE=path.tx` | builds, then `./bin/trux emit-c path.tx` |
| `make test` | `go test ./...` (the compiler's own Go test suite) |
| `make fmt` | builds, then `./bin/trux fmt -r` (formats every `.tx` file under the current directory) |
| `make bench` | builds, then runs the integer, GPU matrix multiply, stencil, and CNN examples |
| `make lsp` | `go build -o bin/trux-lsp ./tooling/lsp` (the language server) |
| `make clean` | removes `bin/`, `tmp/`, and stray generated files, then runs `go clean -cache` |

## Choosing the C compiler

`trux` never hardcodes `gcc` or `clang`:

- Normal programs are compiled with `$CC` if it is set, otherwise `cc`, using `-std=c11 -O2` (or `-O0` with `trux run --debug`).
- Programs that use the `http` package also get `-pthread`.
- Programs that declare at least one `kernel func` are compiled as CUDA with `$NVCC` if set, otherwise `nvcc`, using `-x cu`.

```bash
CC=clang trux run examples/hello.tx
NVCC=/usr/local/cuda/bin/nvcc trux run examples/gpu_matrix_multiply.tx
```

{{% alert context="info" %}}

**Windows**

The runtime has Windows-specific code paths (binary stdout, MSVC support for CUDA builds), and the `Makefile` comment
for `make bench` says to run the Visual Studio `vs` environment first so `nvcc` can find `cl.exe`. The `http` package
is POSIX-only.

{{% /alert %}}

## Verify your setup

```bash
make test                 # the compiler's Go tests (they also compile and run C)
./bin/trux test           # the Trux unit tests in examples/
```

Next: [write your first program](/docs/getting-started/first-program/).
