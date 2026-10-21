# Getting started

This section takes you from a fresh clone to a running Trux program.

1. **[Install and build](install.md)**: what you need on your machine and how to build the `trux` binary.
2. **[Your first program](first-program.md)**: write, run, build, and inspect a small program.
3. **[Editor support](editor.md)**: syntax highlighting, diagnostics, and formatting in VS Code through the Trux language server.

!!! tip "In a hurry?"

    ```bash
    git clone https://github.com/claudioscheer/trux.git
    cd trux
    go build -o bin/trux ./cmd/trux
    ./bin/trux run examples/hello.tx
    ```

    If that prints `3`, you are ready for the [language guide](../guide/index.md).
