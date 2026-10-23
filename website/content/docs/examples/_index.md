---
title: "Examples"
description: "Every program in examples/, with its source and its real output."
icon: "code"
weight: 400
cards: false
---

Every program in this section lives in the [`examples/`](https://github.com/thea1lab/trux/tree/main/examples) directory
of the repository. The source shown on each page is included straight from those files, and the outputs are real runs
from this checkout (see `website/update-example-outputs.sh`).

Run any of them from the repository root after [building `trux`](/docs/getting-started/install/):

```bash
trux run examples/hello.tx
```

{{% alert context="success" %}}

**Learning path**

Start with [Basics](/docs/examples/basics/), [Control flow](/docs/examples/control-flow/),
[Collections and memory](/docs/examples/collections-memory/), and [Modules](/docs/examples/modules/). Those four cover
most of the language. Then read [Projects with IO](/docs/examples/projects-io/) and
[Unit tests](/docs/examples/testing/), look at the [HTTP server](/docs/examples/http-server/), and finish with the
larger [MNIST MLP](/docs/examples/mnist-mlp/) project.

{{% /alert %}}

## Pages

| Page | What it covers |
| --- | --- |
| [Basics](/docs/examples/basics/) | hello, arithmetic, primitives, strings |
| [Control flow](/docs/examples/control-flow/) | if/else, loops, early returns, recursion, assert |
| [Collections and memory](/docs/examples/collections-memory/) | arrays, slices, lists, maps, structs, pointers, mut, clone |
| [Modules](/docs/examples/modules/) | imports, pub names, packages, and multi-file programs |
| [Projects with IO](/docs/examples/projects-io/) | scorebook, stdin, files, and CSV |
| [Unit tests](/docs/examples/testing/) | `.test.tx` files and `trux test` |
| [HTTP server](/docs/examples/http-server/) | three routes on the `http` package |
| [MNIST MLP](/docs/examples/mnist-mlp/) | a trained model on the GPU, checked value by value against numpy, with CPU vs GPU timings |
| [Performance and GPU](/docs/examples/performance-gpu/) | matrix multiply, stencil, CNN, and image rotation |
| [Programs that fail on purpose](/docs/examples/failures/) | the `errors/` programs and the messages they produce |

The sidebar lists the pages in this order. The shorter pages are single programs; the project pages build on the
language pages before them.
