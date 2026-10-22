---
title: "Codebase tour"
description: "Where each part of the compiler lives and what it does."
icon: "account_tree"
weight: 200
---

The compiler is a Go module, `github.com/claudioscheer/trux`, with one third-party dependency
([Cobra](https://github.com/spf13/cobra) for the CLI).

```text
trux/
├── cmd/trux/                 the trux command: run, build, emit-c, fmt, test
├── internal/
│   ├── token/                token types, keywords, and source positions
│   ├── lexer/                source text -> tokens
│   ├── ast/                  syntax tree node types and type representations
│   ├── parser/               tokens -> AST (recursive descent)
│   ├── modules/              import loader, visibility checks, private-name hygiene, merge
│   ├── stdlib/               signatures of the io, csv, image, time, and http packages
│   ├── types/                type checker and ownership-origin analysis
│   ├── ir/                   typed intermediate representation
│   ├── codegen/c/            C and CUDA code generator
│   ├── runtime/c/            the C runtime headers, embedded as Go strings
│   ├── formatter/            source formatter used by trux fmt and the language server
│   └── language/             language version, and a test that keeps the docs in sync
├── tooling/
│   ├── lsp/                  trux-lsp, the language server
│   └── vscode-trux/          VS Code extension (grammar, theme, LSP client)
├── examples/                 runnable example programs and small projects
├── docs/                     design notes (published on this site under "Design notes")
├── website/                  this documentation site: Hugo + Lotus Docs config, pages, example outputs
└── Makefile                  convenience targets
```

## Command layer: `cmd/trux`

| File | Responsibility |
| --- | --- |
| `main.go`, `root.go` | entry point and the root Cobra command (`trux --version` reads `internal/language.Version`) |
| `compile.go` | the pipeline: `modules.Load` → `types.Check` → `ir.Build` → `codegen.GenerateWithOptions`; writing generated files; invoking `cc`/`nvcc`; running executables |
| `run.go` | `trux run`, and formatting compile errors with file, line, column, and source context |
| `build.go` | `trux build -o` |
| `emit_c.go` | `trux emit-c --out-dir` |
| `fmt.go` | `trux fmt [-r]` |
| `test.go` | `trux test`: discovers `.test.tx` files, builds one harness program per test, and reports results |
| `debug.go` | `--debug` phase dumps (tokens, AST JSON, types, IR JSON, C) |

## Front end

- **`internal/token`** defines token types and the keyword table. Adding a keyword starts here.
- **`internal/lexer`** produces tokens with file/line/column positions. It decodes string escapes and turns problems
  (unterminated strings, unknown escapes, stray characters) into `Illegal` tokens that the parser reports.
- **`internal/ast`** holds the node types (`Program`, `StructDecl`, `FuncDecl`, statements, expressions) and the type
  representations (`ScalarType`, `StructType`, `PointerType`, `ArrayType`, `SliceType`, `ListType`, `MapType`,
  `GPUBufferType`, `TupleType`) with `TypeEqual`, `IsComparable`, and `ElementType` helpers.
- **`internal/parser`** is a recursive-descent parser with precedence climbing for binary operators.
- **`internal/modules`** loads the entry file and its imports, applies the package and visibility rules, renames private
  functions and structs to unique internal names, resolves qualified calls and types, and merges everything into one
  `ast.Program`. `LoadTest` is the variant used by `trux test`, which includes the entry `.test.tx` file.

## Middle

- **`internal/stdlib`** is the single source of truth for the standard package members (name, parameters, return type,
  whether the call is statement-only, and how its result is owned). The type checker and the language server both read
  it. The `gpu` package is checked by dedicated code in `internal/types` because its calls take kernels and buffers.
- **`internal/types`** is the type checker (`types.Check`). It returns an `Info` with expression types, ownership
  origins, resolved calls, and per-builtin side tables. Kernel bodies are checked by separate functions
  (`checkKernelFunc`, `checkKernelStmt`, `checkKernelExpr`).
- **`internal/ir`** (`ir.Build`) turns AST + `Info` into typed IR nodes that are convenient for code generation.

## Back end

- **`internal/codegen/c`** emits the C or CUDA program from the IR: type mapping, name mangling, frame-arena decisions,
  return copy-out, loop rewinding, checked arithmetic, `#line` directives, kernels, and launches.
- **`internal/runtime/c`** contains `trux_runtime.h` (`rt.go`) and `trux_http_runtime.h` (`http.go`) as Go string
  constants. The runtime covers arenas, strings, the collection families, bounds checks, checked integer arithmetic,
  printing, stdin/file/CSV/PPM IO, clocks, CUDA helpers, and the HTTP server.

## Tooling

- **`internal/formatter`** formats source using the lexer and parser.
- **`tooling/lsp`** implements diagnostics, formatting, completion, hover, definition, and references on top of the
  compiler packages.
- **`tooling/vscode-trux`** is a thin VS Code client plus a TextMate grammar.

## Where to start reading

1. `cmd/trux/compile.go`, `compileLoaded`: one function that shows the whole pipeline.
2. `internal/parser/parser.go`: small and self-contained.
3. `internal/types/types.go`, `checkStmt` and `checkCall`: where most language rules live.
4. `internal/codegen/c/c.go`, `GenerateWithOptions` and `emitFunc`: the shape of every generated file.
5. Run `trux run --debug examples/hello.tx` and read the files in `tmp/trux-debug/hello/` side by side with the code.

## Tests

Every package has Go tests next to it (`*_test.go`). Notable ones:

| Tests | What they cover |
| --- | --- |
| `cmd/trux/run_test.go` | end-to-end: compiles and runs programs (including many files in `examples/`) and checks their output |
| `cmd/trux/test_test.go` | the `trux test` runner |
| `internal/codegen/c/c_test.go` | shape of the generated C |
| `internal/runtime/c/rt_test.go` | runtime helpers, compiled with the system C compiler |
| `internal/types/types_test.go`, `internal/parser/parser_test.go`, `internal/lexer/lexer_test.go`, `internal/modules/loader_test.go` | the individual compiler stages |
| `internal/language/docs_alignment_test.go` | fails if `README.md`, `docs/SPECS.md`, `docs/MODULES.md`, or `docs/GPU.md` drift from the implemented language |
| `tooling/lsp/server_test.go` | the language server |

Tests that need `nvcc` or POSIX-only features are skipped when those aren't available.
