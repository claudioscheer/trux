# Trux Tooling

This folder contains editor and language tooling for Trux.

## Language Server

`tooling/lsp` is a small Go LSP server. It supports:

- parse diagnostics for every `.tx` file
- type diagnostics for single-file `main` programs
- document formatting through the same formatter used by `trux fmt`
- keyword, type, builtin, local, function, struct, and standard-package completions
- hover text for core language forms, functions, and standard-package calls
- go to definition and find references for local bindings, including multi-return bindings and multi-assignment targets

It reads standard packages and their members from `internal/stdlib`, so imports such as `io`, `csv`, `errors`, `time`, `http`,
`image`, `gpu`, `sqlite`, `json`, and `jwt` complete and hover without a separate tooling list.

Build it from the repository root:

```bash
make lsp
```

Then copy `bin/trux-lsp` to a directory in your `PATH`. The VS Code extension expects `trux-lsp` to be available in `PATH` by default.

## VS Code Extension

`tooling/vscode-trux` contributes `.tx` language registration, syntax highlighting, and a client that starts `trux-lsp`.

To use a custom language server location, set:

```json
{
  "trux.languageServer.path": "/path/to/trux-lsp"
}
```
