---
title: "Contributing"
description: "Building, testing, changing the language, and working on this site."
icon: "handshake"
weight: 300
---

Trux is a learning project, and contributions that keep it small and readable are welcome.

## Set up

```bash
git clone https://github.com/claudioscheer/trux.git
cd trux
make build      # bin/trux
make test       # go test ./...
```

You need Go and a C compiler; see [Install and build](/docs/getting-started/install/). CUDA is only needed to run GPU
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
8. Documentation: the relevant page under `docs/guide/` or `docs/stdlib/`, [`docs/SPECS.md`](/docs/design-notes/specs/), and an example
   in `examples/` if it helps.

Use `trux run --debug` to inspect each stage while you work.

## Conventions

- Source-level names use lower camelCase; internal Go and C names follow their own language's conventions. See
  [Naming](/docs/design-notes/naming/).
- The compiler must not hardcode `gcc` or `clang`; it uses `cc`/`$CC` and `nvcc`/`$NVCC`. See
  [Using cc or $CC](/docs/design-notes/compiler/).
- Run `make fmt` (or `trux fmt -r`) after editing `.tx` files, and `gofmt` on Go code.
- Write down the reasoning for non-obvious design choices in `docs/` (the existing design notes use a short
  "Tradeoff" section).

## Working on the documentation site

The site is built with [Hugo](https://gohugo.io) (extended edition, 0.140 or newer; CI pins 0.168.0) and the
[Lotus Docs](https://lotusdocs.dev) theme, which Hugo downloads as a Go module, so you also need Go. Everything lives in
`website/`: the configuration in `hugo.toml`, the pages in `content/docs/`, and the landing page in `data/landing.yaml`.
The sidebar is generated from each page's `weight` front matter.

```bash
cd website
hugo server         # live preview at http://localhost:1313/trux/
hugo --gc --minify  # production build into website/public/
```

- Show files from `examples/` with the `include` shortcode, for example `{{</* include "examples/hello.tx" */>}}` or a
  line range such as `{{</* include "examples/gpu_matrix_multiply.tx:15:23" */>}}`, so the site always shows the real
  files. Prefer that over copying code into a page.
- The design notes in `docs/*.md` are published unchanged under "Design notes" by
  `website/content/docs/design-notes/_content.gotmpl`. Edit them where they are; add a new note to the list in that file.
- Internal links use site paths such as `[Memory](/docs/guide/memory/)`. The build fails on a link or `#anchor` that does
  not resolve, so `hugo` doubles as a link checker.
- Every page is also published as Markdown (`…/index.md`, plus `llms.txt` and `llms-full.txt` at the site root) for
  pasting into AI tools. `website/layouts/partials/page-markdown.md` turns the shortcodes into plain Markdown; when you
  add a new shortcode to the content, add a pattern for it there. The build fails if shortcode syntax is left over, and
  CI runs `website/check-markdown-output.sh` on the built site.
- The program outputs shown on the site live in `website/outputs/`. After changing an example or the compiler's output,
  run `website/update-example-outputs.sh` to regenerate them. CI runs the same script with `--check`.
- Trux syntax highlighting comes from a small Prism grammar in `website/static/docs/js/components/prism-trux.min.js`.
  Update its keyword lists when the language gains keywords, builtins, or packages.
- `internal/language/docs_alignment_test.go` reads `README.md`, `docs/SPECS.md`, `docs/MODULES.md`, and `docs/GPU.md`.
  Keep those files in place, and run `go test ./internal/language` after editing them.

The site is deployed to GitHub Pages by `.github/workflows/docs.yml` on every push to `main`.

## License

Trux is released under the MIT License. See [`LICENSE`](https://github.com/claudioscheer/trux/blob/main/LICENSE).
