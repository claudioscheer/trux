---
title: "Projects with IO"
description: "Programs that read stdin, write files, and parse and write CSV."
icon: "input"
weight: 50
---

## scorebook

A small package exporting a `Summary` struct, a `map[string]int` grade counter, and unit tests (`scores.test.tx`).

{{< details summary="`examples/projects/scorebook/main.tx`" >}}

{{< include "examples/projects/scorebook/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/projects/scorebook/main.tx
```

{{< include "website/outputs/scorebook.txt" "text" >}}

## io_basics

stdin, file IO, and CSV in one program. See the [`io`](/docs/stdlib/io/) and [`csv`](/docs/stdlib/csv/) packages.

{{< details summary="`examples/projects/io_basics/main.tx`" >}}

{{< include "examples/projects/io_basics/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'Ada\n41\n' | trux run examples/projects/io_basics/main.tx
```

{{< include "website/outputs/io_basics.txt" "text" >}}

## interactive_counter

`io.readLine` and `io.readInt` from the [`io` package](/docs/stdlib/io/).

{{< details summary="`examples/projects/interactive_counter/main.tx`" >}}

{{< include "examples/projects/interactive_counter/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'Ada\n7\n' | trux run examples/projects/interactive_counter/main.tx
```

{{< include "website/outputs/interactive_counter.txt" "text" >}}

## file_notes

`io.readFile` and `io.writeFile` from the [`io` package](/docs/stdlib/io/).

{{< details summary="`examples/projects/file_notes/main.tx`" >}}

{{< include "examples/projects/file_notes/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'review IO examples\n' | trux run examples/projects/file_notes/main.tx
```

{{< include "website/outputs/file_notes.txt" "text" >}}

## csv_roster

`csv.read`, `append`, and `csv.write` from the [`csv` package](/docs/stdlib/csv/).

{{< details summary="`examples/projects/csv_roster/main.tx`" >}}

{{< include "examples/projects/csv_roster/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'Cara\nops\n' | trux run examples/projects/csv_roster/main.tx
```

{{< include "website/outputs/csv_roster.txt" "text" >}}

## csv_quoting

Quoted fields, a doubled quote, and an empty cell, read with `csv.read` and written back with `csv.write`. See
[Quoting](/docs/stdlib/csv/#quoting).

{{< details summary="`examples/projects/csv_quoting/main.tx`" >}}

{{< include "examples/projects/csv_quoting/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/projects/csv_quoting/main.tx
```

{{< include "website/outputs/csv_quoting.txt" "text" >}}
