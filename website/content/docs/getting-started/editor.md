---
title: "Editor support"
description: "The trux-lsp language server and the VS Code extension."
icon: "edit"
weight: 300
---

The repository ships editor tooling under [`tooling/`](https://github.com/claudioscheer/trux/tree/main/tooling).

## Language server (`trux-lsp`)

`tooling/lsp` is a small Language Server Protocol server written in Go. It reuses the compiler's own packages, so what it
reports matches what `trux` does. It provides:

- diagnostics for parse and type errors as you type
- document formatting (the same formatter as `trux fmt`)
- completion for keywords, builtins, and standard package members
- hover text for core language forms, functions, and standard package calls
- go to definition and find references

Build it from the repository root and put it on your `PATH`:

```bash
make lsp                      # or: go build -o bin/trux-lsp ./tooling/lsp
cp bin/trux-lsp ~/.local/bin/ # any directory on your PATH
```

Any editor that can start an LSP server over stdio can use `trux-lsp` for `.tx` files.

## VS Code extension

`tooling/vscode-trux` registers the `.tx` language, adds a TextMate grammar for syntax highlighting, an optional
**Trux Dark** color theme, and a client that starts `trux-lsp`.

To run it from the checkout:

```bash
cd tooling/vscode-trux
npm install
```

Then open that folder in VS Code and launch it as an extension (for example with **Run > Start Debugging**), or, with
[`vsce`](https://github.com/microsoft/vscode-vsce) installed, package it with `npm run compile`. Without `npm install`, syntax highlighting and the theme still work,
but language-server features are disabled.

If `trux-lsp` is not on your `PATH`, point the extension at it:

```json
{
  "trux.languageServer.path": "/path/to/trux-lsp"
}
```

The extension also sets two-space indentation for Trux files, which matches `trux fmt`.
