---
title: "Getting started"
description: "Install the compiler, run your first program, and set up your editor."
icon: "rocket_launch"
weight: 100
cards: false
---

This section takes you from a fresh clone to a running Trux program.

1. **[Install and build](/docs/getting-started/install/)**: what you need on your machine and how to build the `trux` binary.
2. **[Your first program](/docs/getting-started/first-program/)**: write, run, build, and inspect a small program.
3. **[Editor support](/docs/getting-started/editor/)**: syntax highlighting, diagnostics, and formatting in VS Code through the Trux language server.

{{% alert context="success" %}}

**In a hurry?**

```bash
git clone https://github.com/thea1lab/trux.git
cd trux
go build -o bin/trux ./cmd/trux
./bin/trux run examples/hello.tx
```

If that prints `3`, you are ready for the [language guide](/docs/guide/).

{{% /alert %}}
