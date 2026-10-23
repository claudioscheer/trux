---
title: "Standard library"
description: "The packages built into the compiler."
icon: "inventory_2"
weight: 300
cards: false
---

Trux ships a handful of built-in packages. Each one has to be imported by its bare name before use:

```trux
import "io"
import "csv"
```

| Package | What it's for |
| --- | --- |
| [`io`](/docs/stdlib/io/) | reading stdin line by line, reading and writing whole text files, and reading the environment |
| [`csv`](/docs/stdlib/csv/) | reading and writing CSV files as flat `list[string]` tables |
| [`image`](/docs/stdlib/image/) | reading and writing plain-text P3 PPM images as flat RGB values |
| [`time`](/docs/stdlib/time/) | wall-clock timestamps, monotonic timers, and sleeping |
| [`math`](/docs/stdlib/math/) | `exp` and `sqrt` for float host code; kernels use the built-in `exp` and `sqrt` |
| [`sqlite`](/docs/stdlib/sqlite/) | embedded relational database with transactions and prepared statements |
| [`json`](/docs/stdlib/json/) | high-performance JSON parsing, inspection, and generation |
| [`jwt`](/docs/stdlib/jwt/) | HS256 signing and verification for API authentication |
| [`http`](/docs/stdlib/http/) | a plaintext HTTP/1.1 server with a worker pool (Linux/macOS). No HTTPS; put a TLS reverse proxy in front |
| [`gpu`](/docs/stdlib/gpu/) | CUDA device buffers, kernel launches, and thread coordinates |
| [`errors`](/docs/guide/errors/) | creating errors with `errors.new` and retrieving messages with `errors.message` |

{{% alert context="info" %}}

**Built into the compiler**

These packages are not written in Trux. The type checker knows their signatures, and the code generator lowers each
call to a helper in the C runtime header (`trux_runtime.h`, plus `trux_http_runtime.h` for `http` and
`trux_jwt_runtime.h` for `jwt`). Calls that produce no value are statement-only.

{{% /alert %}}

Besides packages, a few **builtins** are always available without an import:

| Builtin | Description |
| --- | --- |
| `print(a, b, ...)` | print one or more `int`, `float`, `string`, or `bool` values on one line, then a newline (statement only) |
| `assert(cond, message)` | stop with exit status `1` and an error message if `cond` is `false` (statement only) |
| `len(x)` | length of a string, array, slice, or list |
| `append(list, value)` | add an element to a list (statement only) |
| `make([]T, n)` | create a zero-filled slice of length `n` |
| `clone(x)` | deep copy of a string, array, slice, or list |

## Errors and failure {#runtime-errors}

Stdin input parsing and invalid runtime state stop the program with a message on stderr such as
`trux runtime error: invalid int input` and exit status `1`. File operations in `io`, `csv`, and `image` return explicit
`error` values that callers must handle or unpack. Each package page shows its real error messages;
[Errors and failures](/docs/guide/errors/) explains the format.
