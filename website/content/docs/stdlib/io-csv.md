---
title: "io and csv"
description: "Reading stdin and files, writing files, and CSV."
icon: "input"
weight: 100
---

## io

```trux
import "io"
```

| Function | Returns | Description |
| --- | --- | --- |
| `io.readLine()` | `string` | Reads one line from stdin, without the trailing newline (a trailing `\r` is also dropped). Returns `""` at end of input. |
| `io.readInt()` | `int` | Reads one line and parses it as a base-10 integer. Surrounding spaces are ignored. |
| `io.readFloat()` | `float` | Reads one line and parses it as a floating-point number. |
| `io.readBool()` | `bool` | Reads one line; it must be `true` or `false`. |
| `io.readFile(path string)` | `string` | Returns the entire contents of a file. |
| `io.writeFile(path string, contents string)` | | Creates or overwrites a file (statement only). |

Invalid typed input (`io.readInt` on `abc`, say) is a runtime error, as is failing to open or write a file.

Relative paths are resolved against the **current working directory** of the running program, not against the `.tx`
file. The example projects use paths such as `examples/projects/io_basics/input.txt`, so run them from the repository
root.

{{< include "examples/projects/interactive_counter/main.tx" "trux" >}}

```shell-session
$ printf 'Ada\n7\n' | trux run examples/projects/interactive_counter/main.tx
counter for Ada
7
8
9
```

## csv

```trux
import "csv"
```

| Function | Returns | Description |
| --- | --- | --- |
| `csv.read(path string, columns int)` | `list[string]` | Reads a CSV file into a flat, row-major list of cells. Every row must have exactly `columns` cells. |
| `csv.write(path string, cells list[string], columns int)` | | Writes a flat row-major list as CSV (statement only). `len(cells)` must be divisible by `columns`. |

The table is flat: cell `(row, col)` is `cells[row * columns + col]`, and the number of rows is `len(cells) / columns`.
Quoted fields and doubled quotes (`""`) inside quoted fields are supported.

{{< include "examples/projects/csv_roster/main.tx" "trux" >}}

```shell-session
$ printf 'Cara\nops\n' | trux run examples/projects/csv_roster/main.tx
8 cells
name team
Cara ops
```

## Putting it together

`examples/projects/io_basics/main.tx` uses stdin, files, and CSV in one program:

{{< include "examples/projects/io_basics/main.tx" "trux" >}}

```shell-session
$ printf 'Ada\n41\n' | trux run examples/projects/io_basics/main.tx
hello Ada
42
6 name kind
```

See [IO packages](/docs/design-notes/io/) for the design notes.
