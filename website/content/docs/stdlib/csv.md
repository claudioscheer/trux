---
title: "csv"
description: "Reading and writing CSV files as flat list[string] tables."
icon: "table_chart"
weight: 110
---

```trux
import "csv"
```

| Function | Returns | Description |
| --- | --- | --- |
| `csv.read(path string, columns int)` | `list[string]` | Reads a whole CSV file into a flat, row-major list of cells. Every row must have exactly `columns` cells. |
| `csv.write(path string, cells list[string], columns int)` | | Creates or overwrites a file with `cells` as CSV, `columns` cells per row (statement only). `len(cells)` must be divisible by `columns`. |

`csv` builds on the file handling of the [`io` package](/docs/stdlib/io/): `csv.read` reads the whole file the way
`io.readFile` does, and `csv.write` replaces it the way `io.writeFile` does. Paths are resolved against the current
working directory in the same way, and the two packages are often imported together.

## The table layout

There is no table type. The cells come back as one flat `list[string]`, row after row:

- cell `(row, col)` is `cells[row * columns + col]`
- the number of rows is `len(cells) / columns`
- a header row, if the file has one, is just the first `columns` cells

All cells are strings; convert them yourself if you need numbers.

## Reading and writing

`csv.read` returns the cells as an ordinary list, so you can `append` rows to it, one cell at a time, and write it
back with `csv.write`. This example reads the roster with `csv.read`, gets a new row from stdin with
[`io.readLine`](/docs/stdlib/io/#reading-stdin), and writes the result to `out.csv`:

{{< include "examples/projects/csv_roster/roster.csv" "text" >}}

{{< include "examples/projects/csv_roster/main.tx" "trux" >}}

```shell-session
$ printf 'Cara\nops\n' | trux run examples/projects/csv_roster/main.tx
```

{{< include "website/outputs/csv_roster.txt" "text" >}}

## Quoting

The format follows the common CSV rules:

- Fields are separated by `,` and rows by `\n`, `\r\n`, or `\r`. A final newline at the end of the file is optional.
- A field that starts with `"` is quoted. It can contain commas, newlines, and doubled quotes (`""`), which stand for
  one `"`. A quoted field must be followed by `,`, a newline, or the end of the file.
- A `"` inside an unquoted field is an error.
- Spaces are kept as part of the cell; nothing is trimmed.
- An empty file gives an empty list.

When writing, `csv.write` quotes a cell only if it contains `,`, `"`, `\n`, or `\r`, and doubles any `"` inside it.
Rows end with `\n`, including the last one. This example reads a file with quoted fields and an empty cell, then
writes the cells back and prints the new file with `io.readFile`:

{{< include "examples/projects/csv_quoting/items.csv" "text" >}}

{{< include "examples/projects/csv_quoting/main.tx" "trux" >}}

```shell-session
$ trux run examples/projects/csv_quoting/main.tx
```

{{< include "website/outputs/csv_quoting.txt" "text" >}}

## Putting it together

`examples/projects/io_basics/main.tx` uses stdin, files, and CSV in one program:

{{< include "examples/projects/io_basics/main.tx" "trux" >}}

```shell-session
$ printf 'Ada\n41\n' | trux run examples/projects/io_basics/main.tx
```

{{< include "website/outputs/io_basics.txt" "text" >}}

## Errors

Failures stop the program with exit status `1` and a message on stderr:

| Message | Cause |
| --- | --- |
| `trux runtime error: csv row has 1 columns, want 2` | a row has a different number of cells than `columns`; a blank line in the middle of a file is a row with one empty cell |
| `trux runtime error: csv columns must be positive, got 0` | `columns` was `0` or negative |
| `trux runtime error: csv cell count 5 is not divisible by columns 2` | `csv.write` got a list that doesn't fill whole rows |
| `trux runtime error: unexpected quote in csv field` | a `"` appeared in the middle of an unquoted field |
| `trux runtime error: unterminated csv quoted field` | the file ended inside a quoted field |
| `trux runtime error: expected csv delimiter after quoted field` | text followed a closing quote, as in `"a"b` |
| `trux runtime error: cannot open file for reading data.csv: No such file or directory` | `csv.read` couldn't open the file; `csv.write` reports `cannot open csv for writing …` |

See [IO packages](/docs/design-notes/io/) for the design notes.
