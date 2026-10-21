---
title: "Examples"
description: "Every program in examples/, with its source and its real output."
icon: "code"
weight: 400
---

Every program on this page lives in the [`examples/`](https://github.com/claudioscheer/trux/tree/main/examples) directory
of the repository. The source shown here is included straight from those files, and each output was produced by running
the program with `trux run` (see `website/update-example-outputs.sh`).

Run any of them from the repository root after [building `trux`](/docs/getting-started/install/):

```bash
trux run examples/hello.tx
```

{{% alert context="success" %}}

**Learning path**

Start with **hello**, then **control_flow**, **loops**, **collections**, and **ownership_clone**. Those five cover
most of the language.

{{% /alert %}}

## Basics

### hello

A function, a variable, and `print`.

{{< details summary="`examples/hello.tx`" >}}

{{< include "examples/hello.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/hello.tx
```

{{< include "website/outputs/hello.txt" "text" >}}

### arithmetic

Small integer functions and nested calls.

{{< details summary="`examples/arithmetic.tx`" >}}

{{< include "examples/arithmetic.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/arithmetic.tx
```

{{< include "website/outputs/arithmetic.txt" "text" >}}

### primitives

All four scalar types, string escapes, and multi-argument `print`.

{{< details summary="`examples/primitives.tx`" >}}

{{< include "examples/primitives.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/primitives.tx
```

{{< include "website/outputs/primitives.txt" "text" >}}

### strings

Concatenation, equality, and the `in` operator.

{{< details summary="`examples/strings.tx`" >}}

{{< include "examples/strings.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/strings.tx
```

{{< include "website/outputs/strings.txt" "text" >}}

## Control flow

### control_flow

`if` / `else if` / `else` and a condition loop.

{{< details summary="`examples/control_flow.tx`" >}}

{{< include "examples/control_flow.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/control_flow.tx
```

{{< include "website/outputs/control_flow.txt" "text" >}}

### else_if

A chain of `else if` branches.

{{< details summary="`examples/else_if.tx`" >}}

{{< include "examples/else_if.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/else_if.tx
```

{{< include "website/outputs/else_if.txt" "text" >}}

### loops

All three `for` forms.

{{< details summary="`examples/loops.tx`" >}}

{{< include "examples/loops.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/loops.tx
```

{{< include "website/outputs/loops.txt" "text" >}}

### return_paths

Returning from branches and loops.

{{< details summary="`examples/return_paths.tx`" >}}

{{< include "examples/return_paths.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/return_paths.tx
```

{{< include "website/outputs/return_paths.txt" "text" >}}

### fibonacci

Recursive and iterative Fibonacci.

{{< details summary="`examples/fibonacci.tx`" >}}

{{< include "examples/fibonacci.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/fibonacci.tx
```

{{< include "website/outputs/fibonacci.txt" "text" >}}

### assertions

Runtime checks with `assert`.

{{< details summary="`examples/assertions.tx`" >}}

{{< include "examples/assertions.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/assertions.tx
```

{{< include "website/outputs/assertions.txt" "text" >}}

## Collections and memory

### collections

Arrays, slices, `make`, string indexing, and list handles.

{{< details summary="`examples/collections.tx`" >}}

{{< include "examples/collections.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/collections.tx
```

{{< include "website/outputs/collections.txt" "text" >}}

### jagged_lists

A `list[list[int]]` with rows of different lengths.

{{< details summary="`examples/jagged_lists.tx`" >}}

{{< include "examples/jagged_lists.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/jagged_lists.tx
```

{{< include "website/outputs/jagged_lists.txt" "text" >}}

### nested_slices

`[][]float` built with `make`, and a deep `clone`.

{{< details summary="`examples/nested_slices.tx`" >}}

{{< include "examples/nested_slices.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/nested_slices.tx
```

{{< include "website/outputs/nested_slices.txt" "text" >}}

### matrix_multiply_2x2

Passing and returning `[2][2]int` arrays.

{{< details summary="`examples/matrix_multiply_2x2.tx`" >}}

{{< include "examples/matrix_multiply_2x2.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/matrix_multiply_2x2.tx
```

{{< include "website/outputs/matrix_multiply_2x2.txt" "text" >}}

### ownership_clone

Borrowed versus owned returns, and `clone`.

{{< details summary="`examples/ownership_clone.tx`" >}}

{{< include "examples/ownership_clone.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/ownership_clone.tx
```

{{< include "website/outputs/ownership_clone.txt" "text" >}}

### mut_parameters

Mutating caller collections through `mut` parameters.

{{< details summary="`examples/mut_parameters.tx`" >}}

{{< include "examples/mut_parameters.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/mut_parameters.tx
```

{{< include "website/outputs/mut_parameters.txt" "text" >}}

## Modules

### modules_basic

One import and one `pub func`.

{{< details summary="`examples/modules/basic/main.tx`" >}}

{{< include "examples/modules/basic/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/basic/main.tx
```

{{< include "website/outputs/modules_basic.txt" "text" >}}

### modules_transitive

A chain of imports: `main` → `app` → `text`.

{{< details summary="`examples/modules/transitive/main.tx`" >}}

{{< include "examples/modules/transitive/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/transitive/main.tx
```

{{< include "website/outputs/modules_transitive.txt" "text" >}}

### modules_private_names

Same-named private helpers in different packages.

{{< details summary="`examples/modules/private_names/main.tx`" >}}

{{< include "examples/modules/private_names/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/private_names/main.tx
```

{{< include "website/outputs/modules_private_names.txt" "text" >}}

### modules_same_package_math

One package spread over several files.

{{< details summary="`examples/modules/same_package_math/main.tx`" >}}

{{< include "examples/modules/same_package_math/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/same_package_math/main.tx
```

{{< include "website/outputs/modules_same_package_math.txt" "text" >}}

### modules_mut_parameters

A stack and a queue in separate packages, using `mut`.

{{< details summary="`examples/modules/mut_parameters/main.tx`" >}}

{{< include "examples/modules/mut_parameters/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/mut_parameters/main.tx
```

{{< include "website/outputs/modules_mut_parameters.txt" "text" >}}

## Projects with IO

### scorebook

A small package with unit tests (`scores.test.tx`).

{{< details summary="`examples/projects/scorebook/main.tx`" >}}

{{< include "examples/projects/scorebook/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/projects/scorebook/main.tx
```

{{< include "website/outputs/scorebook.txt" "text" >}}

### io_basics

stdin, file IO, and CSV in one program. See the [`io`](/docs/stdlib/io/) and [`csv`](/docs/stdlib/csv/) packages.

{{< details summary="`examples/projects/io_basics/main.tx`" >}}

{{< include "examples/projects/io_basics/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'Ada\n41\n' | trux run examples/projects/io_basics/main.tx
```

{{< include "website/outputs/io_basics.txt" "text" >}}

### interactive_counter

`io.readLine` and `io.readInt` from the [`io` package](/docs/stdlib/io/).

{{< details summary="`examples/projects/interactive_counter/main.tx`" >}}

{{< include "examples/projects/interactive_counter/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'Ada\n7\n' | trux run examples/projects/interactive_counter/main.tx
```

{{< include "website/outputs/interactive_counter.txt" "text" >}}

### file_notes

`io.readFile` and `io.writeFile` from the [`io` package](/docs/stdlib/io/).

{{< details summary="`examples/projects/file_notes/main.tx`" >}}

{{< include "examples/projects/file_notes/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'review IO examples\n' | trux run examples/projects/file_notes/main.tx
```

{{< include "website/outputs/file_notes.txt" "text" >}}

### csv_roster

`csv.read`, `append`, and `csv.write` from the [`csv` package](/docs/stdlib/csv/).

{{< details summary="`examples/projects/csv_roster/main.tx`" >}}

{{< include "examples/projects/csv_roster/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ printf 'Cara\nops\n' | trux run examples/projects/csv_roster/main.tx
```

{{< include "website/outputs/csv_roster.txt" "text" >}}

### csv_quoting

Quoted fields, a doubled quote, and an empty cell, read with `csv.read` and written back with `csv.write`. See
[Quoting](/docs/stdlib/csv/#quoting).

{{< details summary="`examples/projects/csv_quoting/main.tx`" >}}

{{< include "examples/projects/csv_quoting/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/projects/csv_quoting/main.tx
```

{{< include "website/outputs/csv_quoting.txt" "text" >}}

## Unit tests

```shell-session
$ trux test examples/projects/scorebook/scores.test.tx
PASS examples/projects/scorebook/scores.test.tx testBestAndPassingCount
PASS examples/projects/scorebook/scores.test.tx testCurveCapsScores
PASS examples/projects/scorebook/scores.test.tx testLetterBoundaries
PASS examples/projects/scorebook/scores.test.tx testPrivateHelpers
PASS examples/projects/scorebook/scores.test.tx testTotalAndAverage
5 passed, 0 failed
```

See [Testing](/docs/guide/testing/).

## HTTP server

`examples/projects/http_server/main.tx` serves `GET /`, `GET /health`, and `POST /echo` on `127.0.0.1:8080`
(Linux and macOS). It runs until you stop it. See the [`http` package](/docs/stdlib/http/) for the full listing.

```bash
trux run examples/projects/http_server/main.tx
curl http://127.0.0.1:8080/health
```

## Performance and GPU

These programs time their work, so their output varies from machine to machine. The GPU ones need an NVIDIA GPU and
`nvcc`.

| Example | Needs a GPU | What it does |
| --- | --- | --- |
| `examples/performance_integer.tx` | no | A CPU-bound integer workload. Compare `trux run` (built with `-O2`) and `trux run --debug` (`-O0`). |
| `examples/gpu_matrix_multiply.tx` | yes | 500 x 500 integer matrix multiply on the CPU and with `gpu.launch2d`, with a full result comparison. |
| `examples/performance_stencil.tx` | yes | 200 Jacobi stencil steps on a 1024 x 1024 grid, swapping two device buffers. |
| `examples/performance_cnn.tx` | yes | A four-layer float convolutional network using `gpu.launch1d`. |
| `examples/projects/rotate_image/main.tx` | yes | Rotates a PPM image on CPU and GPU and writes both results. |

`make bench` runs the first four in sequence.

{{< details summary="`examples/gpu_matrix_multiply.tx`" >}}

{{< include "examples/gpu_matrix_multiply.tx" "trux" >}}

{{< /details >}}
