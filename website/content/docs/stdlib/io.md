---
title: "io"
description: "Reading stdin and whole text files, and writing files."
icon: "input"
weight: 100
# The old combined "io and csv" page lived here.
aliases: ["/docs/stdlib/io-csv/"]
---

```trux
import "io"
```

**What it is.** Line-by-line reads from standard input, typed reads (`int`, `float`, `bool`), and whole-file reads and
writes.

**Why.** Programs need input and output. These calls cover stdin and whole files, with no file handles to open or
close.

**When you don't need it.** For output to the terminal, `print` is a builtin and needs no import. For comma-separated
data, use [`csv`](/docs/stdlib/csv/).

| Function | Returns | Description |
| --- | --- | --- |
| `io.readLine()` | `string` | Reads one line from stdin, without the trailing `\n` (a trailing `\r` is also dropped). Returns `""` at end of input. |
| `io.readInt()` | `int` | Reads one line and parses it as a base-10 64-bit integer. Leading and trailing whitespace is ignored; a leading `+` or `-` is allowed. |
| `io.readFloat()` | `float` | Reads one line and parses it as a floating-point number, such as `2.5`, `-0.1`, or `1e3`. Surrounding whitespace is ignored. |
| `io.readBool()` | `bool` | Reads one line, which must be exactly `true` or `false` after surrounding whitespace is removed. |
| `io.readFile(path string)` | `string` | Returns the entire contents of a file, byte for byte. |
| `io.writeFile(path string, contents string)` | | Creates or overwrites a file with exactly `contents`; no newline is added (statement only). |

## Reading stdin

Every read consumes exactly one line. `io.readInt`, `io.readFloat`, and `io.readBool` read a line the same way
`io.readLine` does and then parse it, so put each value on its own line:

{{< include "examples/projects/interactive_counter/main.tx" "trux" >}}

```shell-session
$ printf 'Ada\n7\n' | trux run examples/projects/interactive_counter/main.tx
```

{{< include "website/outputs/interactive_counter.txt" "text" >}}

At end of input `io.readLine` returns `""`, the same as for an empty line, so there's no way to tell the two apart.
The typed reads fail at end of input, because `""` isn't a valid number or boolean.

## Reading and writing files

`io.readFile` returns the whole file as one `string`, including its final newline if it has one. `io.writeFile`
replaces the file's contents in one call; there's no append mode, so to add to a file, read it, extend the string,
and write it back:

{{< include "examples/projects/file_notes/main.tx" "trux" >}}

```shell-session
$ printf 'review IO examples\n' | trux run examples/projects/file_notes/main.tx
```

{{< include "website/outputs/file_notes.txt" "text" >}}

The blank line in the output comes from `seed.txt`: the file ends with a newline, and the program adds another
before the line it read from stdin.

Relative paths are resolved against the **current working directory** of the running program, not against the
`.tx` file. The example projects use paths such as `examples/projects/file_notes/seed.txt`, so run them from the
repository root.

## Errors

**Common mistakes.** Typed reads stop the program when the line doesn't parse:

{{< include "examples/errors/io_read_int.tx" "trux" >}}

```shell-session
$ printf 'abc\n' | trux run examples/errors/io_read_int.tx
```

{{< include "website/outputs/errors/io_read_int.txt" "text" >}}

A missing file:

{{< include "website/outputs/errors/io_missing_file.txt" "text" >}}

Using `io` without importing it is a compile error:

{{< include "website/outputs/errors/io_not_imported.txt" "text" >}}

All runtime failures exit with status `1`:

| Message | Cause |
| --- | --- |
| `trux runtime error: invalid int input` | `io.readInt` got a line that isn't a whole base-10 number, or one that doesn't fit in an `int` |
| `trux runtime error: invalid float input` | `io.readFloat` got a line that isn't a number |
| `trux runtime error: invalid bool input` | `io.readBool` got anything other than `true` or `false` |
| `trux runtime error: cannot open file for reading missing.txt: No such file or directory` | `io.readFile` couldn't open the file; `io.writeFile` reports `cannot open file for writing …` |
| `trux runtime error: file path contains NUL byte` | a path contained a `\0` character |

There is no way to catch these. To accept bad input gracefully, read with `io.readLine` and check the text yourself.

## Reading CSV {#csv}

For comma-separated files, use the [`csv` package](/docs/stdlib/csv/). It reads and writes whole files the same way
`io.readFile` and `io.writeFile` do, and returns the parsed cells as a `list[string]`.

See [IO packages](/docs/design-notes/io/) for the design notes.
