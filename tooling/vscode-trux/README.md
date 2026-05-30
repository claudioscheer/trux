# Trux Language Support

VS Code support for `.tx` Trux files.

## Features

- Syntax highlighting for Trux keywords, types, strings, numbers, comments, functions, and operators.
- Diagnostics, formatting, completion, and hover through `trux-lsp`.

## Language Server

The extension starts `trux-lsp` from `PATH` by default. Build it from the repository root:

```bash
make lsp
```

Then copy `bin/trux-lsp` to a directory in your `PATH`.

If the binary lives somewhere else, set `trux.languageServer.path` to the full path.
