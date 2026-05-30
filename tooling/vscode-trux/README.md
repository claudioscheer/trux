# Trux Language Support

VS Code support for `.tx` Trux files.

## Features

- Syntax highlighting for Trux keywords, types, strings, numbers, comments, functions, and operators.
- Optional `Trux Dark` color theme tuned for the Trux grammar scopes.
- Diagnostics, formatting, completion, and hover through `trux-lsp`.

## Language Server

The extension starts `trux-lsp` from `PATH` by default. Build it from the repository root:

```bash
make lsp
```

Then copy `bin/trux-lsp` to a directory in your `PATH`.

If the binary lives somewhere else, set `trux.languageServer.path` to the full path.

When running this extension directly from the repository checkout, run `npm install` in `tooling/vscode-trux` first so the VS Code language client dependency is available. Without it, syntax highlighting and themes still load, but LSP features are disabled.
