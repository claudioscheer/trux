# Contributing

Trux is a learning project, and contributions that keep it small and readable are welcome.

## Set up

```bash
git clone https://github.com/claudioscheer/trux.git
cd trux
make build      # bin/trux
make test       # go test ./...
```

You need Go and a C compiler; see [Install and build](../getting-started/install.md). CUDA is only needed to run GPU
programs.

## Making a language change

A new feature usually touches the stages in order:

1. **`internal/token`** and **`internal/lexer`**, if it needs a new keyword or operator.
2. **`internal/ast`** and **`internal/parser`**, for the syntax.
3. **`internal/types`**, for the rules, error messages, and ownership origins.
4. **`internal/ir`**, if code generation needs a new node.
5. **`internal/codegen/c`** and, if needed, **`internal/runtime/c`**.
6. **`internal/formatter`** and **`tooling/lsp`** (completions, hover text) so tooling understands it.
7. Tests at each layer, plus an end-to-end test in `cmd/trux/run_test.go`.
8. Documentation: the relevant page under `docs/guide/` or `docs/stdlib/`, [`docs/SPECS.md`](../SPECS.md), and an example
   in `examples/` if it helps.

Use `trux run --debug` to inspect each stage while you work.

## Conventions

- Source-level names use lower camelCase; internal Go and C names follow their own language's conventions. See
  [Naming](../NAMING.md).
- The compiler must not hardcode `gcc` or `clang`; it uses `cc`/`$CC` and `nvcc`/`$NVCC`. See
  [Using cc or $CC](../COMPILER.md).
- Run `make fmt` (or `trux fmt -r`) after editing `.tx` files, and `gofmt` on Go code.
- Write down the reasoning for non-obvious design choices in `docs/` (the existing design notes use a short
  "Tradeoff" section).

## Working on the documentation site

The site is built with [MkDocs](https://www.mkdocs.org/) and the [Material](https://squidfunk.github.io/mkdocs-material/)
theme. Pages are Markdown files in `docs/`, and the navigation is in `mkdocs.yml`.

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install -r website/requirements.txt
mkdocs serve        # live preview at http://127.0.0.1:8000/trux/
mkdocs build --strict
```

- Code samples from `examples/` are included with snippet markers such as `--8<-- "examples/hello.tx"`, so the site always
  shows the real files. Prefer that over copying code into a page.
- The program outputs shown on the site live in `website/outputs/`. After changing an example or the compiler's output,
  run `website/update-example-outputs.sh` to regenerate them. CI runs the same script with `--check`.
- Trux syntax highlighting comes from a small Pygments lexer in `website/pygments-trux/`. Update its keyword lists when
  the language gains keywords, builtins, or packages.
- `internal/language/docs_alignment_test.go` reads `README.md`, `docs/SPECS.md`, `docs/MODULES.md`, and `docs/GPU.md`.
  Keep those files in place, and run `go test ./internal/language` after editing them.

The site is deployed to GitHub Pages by `.github/workflows/docs.yml` on every push to `main`.

## License

Trux is released under the MIT License. See [`LICENSE`](https://github.com/claudioscheer/trux/blob/main/LICENSE).
