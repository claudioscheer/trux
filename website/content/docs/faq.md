---
title: "FAQ and troubleshooting"
description: "Common questions, limitations, and error messages explained."
icon: "help"
weight: 700
---

## General

### Is Trux Go?

No. Trux borrows some syntax from Go (`package`, `func`, `for`, slices) but it is a different, much smaller language and
is not Go-compatible. The compiler happens to be written in Go.

### Is Trux ready for production?

No. Trux is a learning project: a complete, readable compiler pipeline built by hand. The language version is `0.1.0`
and the language is still changing.

### Why compile to C?

C compilers exist everywhere and optimize well, and generated C is easy to read, which makes the compiler easy to debug
and to learn from. CUDA is "C plus kernels", so the same backend can target NVIDIA GPUs.

### What's missing?

Not implemented today, so don't expect these to work:

- structs, maps, tuples, enums, pointers, or user-defined types
- generics, closures, or function values (except naming a handler for `http.serve` or a kernel for `gpu.launch*`)
- type inference (`x := 1`), compound assignment (`+=`), and `++` / `--`
- conversions between `int`, `float`, and `string`
- global variables and constants
- removing elements from a list
- command-line arguments for programs (`main` takes no parameters)
- error values: library calls stop the program on failure
- threads or async code outside the HTTP server's worker pool
- recursive directory packages and separate compilation

## Common errors

### `cannot assign float to int` / `operator "+" requires matching numeric operands ...`

Trux never converts between `int` and `float`. Use literals of the right kind (`2.0`, not `2`, in float expressions) and
keep each expression in one numeric type.

### `if condition must be bool, got int`

Conditions must be `bool`. Write `if count > 0`, not `if count`.

### `function "f" must return on all paths`

The compiler doesn't analyze whether a loop always returns, and an `if` without `else` might be skipped. Add a final
`return` at the end of the function.

### `cannot mutate parameter-owned collection`

A function tried to `append` to, or assign an element of, a collection it received as an ordinary parameter. Either mark
the parameter `mut` (if the caller's collection should change), or `clone` it into a local first (if the function should
work on its own copy). See [Memory and ownership](/docs/guide/memory/).

### `cannot mutate collection with unknown ownership`

You tried to change a collection that was returned by a function call. Clone it first:

```trux
let ys list[int] = clone(makeList())
append(ys, 4)
```

The same applies when passing such a value to a `mut` parameter
(`cannot pass collection with unknown ownership to mut parameter ...`).

### `print can only be used as a statement` (or `append`, `assert`, ...)

Some builtins and library calls don't produce a value and must be on their own line. See
[statement-only builtins](/docs/guide/operators/#statement-only-builtins).

### `undefined function "readLine"; use io.readLine after import "io"`

The standard library lives in packages. Add `import "io"` and call `io.readLine()`.

### `package "text" is not imported by main.tx`

You can only call packages that the current file imports directly, even if another imported file already imports them.
Add the import to this file.

### `unexpected "if" after block on the same line`

Write `else if`, or put the second `if` on a new line.

### `nvcc failed: exec: "nvcc": executable file not found in $PATH`

The program declares a `kernel func`, so it must be compiled with CUDA. Install the CUDA toolkit, or set `NVCC` to the
full path of `nvcc`. A program with kernels cannot be built with `cc`.

### `cc failed: ...`

`trux` couldn't find or run a C compiler. Install one (gcc or clang), or set `CC` to its path.

### `http and gpu cannot be used in the same program yet`

That combination isn't supported. Split the work into two programs.

## Runtime errors

All runtime errors start with `trux runtime error:` and exit with status `1`.

| Message | Cause |
| --- | --- |
| `integer overflow in +` (or `-`, `*`, `/`) | the result doesn't fit in a 64-bit `int` |
| `integer division by zero`, `integer remainder by zero` | `x / 0` or `x % 0` on integers |
| `index 5 out of bounds for length 3` | indexing past the end (or a negative index) |
| `slice 2:9 out of bounds for length 4` | slice bounds outside the collection, or start after end |
| `assertion failed: <message>` | an `assert` condition was `false` |
| `invalid int input`, `invalid float input`, `invalid bool input` | `io.readInt` and friends couldn't parse the line |
| `cannot open file for reading notes.txt: No such file or directory` and similar | a file operation failed |
| `csv row has 3 columns, want 2` | a CSV row didn't match the `columns` argument |

The overflow and assertion messages include the Trux file and line, for example `at main.tx:5`.

## The documentation site

### How do I preview the site locally?

See [Working on the documentation site](/docs/internals/contributing/#working-on-the-documentation-site).
