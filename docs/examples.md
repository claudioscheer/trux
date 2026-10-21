# Examples

Every program on this page lives in the [`examples/`](https://github.com/claudioscheer/trux/tree/main/examples) directory
of the repository. The source shown here is included straight from those files, and each output was produced by running
the program with `trux run` (see `website/update-example-outputs.sh`).

Run any of them from the repository root after [building `trux`](getting-started/install.md):

```bash
trux run examples/hello.tx
```

!!! tip "Learning path"

    Start with **hello**, then **control_flow**, **loops**, **collections**, and **ownership_clone**. Those five cover
    most of the language.

## Basics

### hello

A function, a variable, and `print`.

??? example "`examples/hello.tx`"

    ```trux
    --8<-- "examples/hello.tx"
    ```

```console
$ trux run examples/hello.tx
```

```text
--8<-- "website/outputs/hello.txt"
```

### arithmetic

Small integer functions and nested calls.

??? example "`examples/arithmetic.tx`"

    ```trux
    --8<-- "examples/arithmetic.tx"
    ```

```console
$ trux run examples/arithmetic.tx
```

```text
--8<-- "website/outputs/arithmetic.txt"
```

### primitives

All four scalar types, string escapes, and multi-argument `print`.

??? example "`examples/primitives.tx`"

    ```trux
    --8<-- "examples/primitives.tx"
    ```

```console
$ trux run examples/primitives.tx
```

```text
--8<-- "website/outputs/primitives.txt"
```

### strings

Concatenation, equality, and the `in` operator.

??? example "`examples/strings.tx`"

    ```trux
    --8<-- "examples/strings.tx"
    ```

```console
$ trux run examples/strings.tx
```

```text
--8<-- "website/outputs/strings.txt"
```

## Control flow

### control_flow

`if` / `else if` / `else` and a condition loop.

??? example "`examples/control_flow.tx`"

    ```trux
    --8<-- "examples/control_flow.tx"
    ```

```console
$ trux run examples/control_flow.tx
```

```text
--8<-- "website/outputs/control_flow.txt"
```

### else_if

A chain of `else if` branches.

??? example "`examples/else_if.tx`"

    ```trux
    --8<-- "examples/else_if.tx"
    ```

```console
$ trux run examples/else_if.tx
```

```text
--8<-- "website/outputs/else_if.txt"
```

### loops

All three `for` forms.

??? example "`examples/loops.tx`"

    ```trux
    --8<-- "examples/loops.tx"
    ```

```console
$ trux run examples/loops.tx
```

```text
--8<-- "website/outputs/loops.txt"
```

### return_paths

Returning from branches and loops.

??? example "`examples/return_paths.tx`"

    ```trux
    --8<-- "examples/return_paths.tx"
    ```

```console
$ trux run examples/return_paths.tx
```

```text
--8<-- "website/outputs/return_paths.txt"
```

### fibonacci

Recursive and iterative Fibonacci.

??? example "`examples/fibonacci.tx`"

    ```trux
    --8<-- "examples/fibonacci.tx"
    ```

```console
$ trux run examples/fibonacci.tx
```

```text
--8<-- "website/outputs/fibonacci.txt"
```

### assertions

Runtime checks with `assert`.

??? example "`examples/assertions.tx`"

    ```trux
    --8<-- "examples/assertions.tx"
    ```

```console
$ trux run examples/assertions.tx
```

```text
--8<-- "website/outputs/assertions.txt"
```

## Collections and memory

### collections

Arrays, slices, `make`, string indexing, and list handles.

??? example "`examples/collections.tx`"

    ```trux
    --8<-- "examples/collections.tx"
    ```

```console
$ trux run examples/collections.tx
```

```text
--8<-- "website/outputs/collections.txt"
```

### jagged_lists

A `list[list[int]]` with rows of different lengths.

??? example "`examples/jagged_lists.tx`"

    ```trux
    --8<-- "examples/jagged_lists.tx"
    ```

```console
$ trux run examples/jagged_lists.tx
```

```text
--8<-- "website/outputs/jagged_lists.txt"
```

### nested_slices

`[][]float` built with `make`, and a deep `clone`.

??? example "`examples/nested_slices.tx`"

    ```trux
    --8<-- "examples/nested_slices.tx"
    ```

```console
$ trux run examples/nested_slices.tx
```

```text
--8<-- "website/outputs/nested_slices.txt"
```

### matrix_multiply_2x2

Passing and returning `[2][2]int` arrays.

??? example "`examples/matrix_multiply_2x2.tx`"

    ```trux
    --8<-- "examples/matrix_multiply_2x2.tx"
    ```

```console
$ trux run examples/matrix_multiply_2x2.tx
```

```text
--8<-- "website/outputs/matrix_multiply_2x2.txt"
```

### ownership_clone

Borrowed versus owned returns, and `clone`.

??? example "`examples/ownership_clone.tx`"

    ```trux
    --8<-- "examples/ownership_clone.tx"
    ```

```console
$ trux run examples/ownership_clone.tx
```

```text
--8<-- "website/outputs/ownership_clone.txt"
```

### mut_parameters

Mutating caller collections through `mut` parameters.

??? example "`examples/mut_parameters.tx`"

    ```trux
    --8<-- "examples/mut_parameters.tx"
    ```

```console
$ trux run examples/mut_parameters.tx
```

```text
--8<-- "website/outputs/mut_parameters.txt"
```

## Modules

### modules_basic

One import and one `pub func`.

??? example "`examples/modules/basic/main.tx`"

    ```trux
    --8<-- "examples/modules/basic/main.tx"
    ```

```console
$ trux run examples/modules/basic/main.tx
```

```text
--8<-- "website/outputs/modules_basic.txt"
```

### modules_transitive

A chain of imports: `main` → `app` → `text`.

??? example "`examples/modules/transitive/main.tx`"

    ```trux
    --8<-- "examples/modules/transitive/main.tx"
    ```

```console
$ trux run examples/modules/transitive/main.tx
```

```text
--8<-- "website/outputs/modules_transitive.txt"
```

### modules_private_names

Same-named private helpers in different packages.

??? example "`examples/modules/private_names/main.tx`"

    ```trux
    --8<-- "examples/modules/private_names/main.tx"
    ```

```console
$ trux run examples/modules/private_names/main.tx
```

```text
--8<-- "website/outputs/modules_private_names.txt"
```

### modules_same_package_math

One package spread over several files.

??? example "`examples/modules/same_package_math/main.tx`"

    ```trux
    --8<-- "examples/modules/same_package_math/main.tx"
    ```

```console
$ trux run examples/modules/same_package_math/main.tx
```

```text
--8<-- "website/outputs/modules_same_package_math.txt"
```

### modules_mut_parameters

A stack and a queue in separate packages, using `mut`.

??? example "`examples/modules/mut_parameters/main.tx`"

    ```trux
    --8<-- "examples/modules/mut_parameters/main.tx"
    ```

```console
$ trux run examples/modules/mut_parameters/main.tx
```

```text
--8<-- "website/outputs/modules_mut_parameters.txt"
```

## Projects with IO

### scorebook

A small package with unit tests (`scores.test.tx`).

??? example "`examples/projects/scorebook/main.tx`"

    ```trux
    --8<-- "examples/projects/scorebook/main.tx"
    ```

```console
$ trux run examples/projects/scorebook/main.tx
```

```text
--8<-- "website/outputs/scorebook.txt"
```

### io_basics

stdin, file IO, and CSV in one program.

??? example "`examples/projects/io_basics/main.tx`"

    ```trux
    --8<-- "examples/projects/io_basics/main.tx"
    ```

```console
$ printf 'Ada\n41\n' | trux run examples/projects/io_basics/main.tx
```

```text
--8<-- "website/outputs/io_basics.txt"
```

### interactive_counter

`io.readLine` and `io.readInt`.

??? example "`examples/projects/interactive_counter/main.tx`"

    ```trux
    --8<-- "examples/projects/interactive_counter/main.tx"
    ```

```console
$ printf 'Ada\n7\n' | trux run examples/projects/interactive_counter/main.tx
```

```text
--8<-- "website/outputs/interactive_counter.txt"
```

### file_notes

`io.readFile` and `io.writeFile`.

??? example "`examples/projects/file_notes/main.tx`"

    ```trux
    --8<-- "examples/projects/file_notes/main.tx"
    ```

```console
$ printf 'review IO examples\n' | trux run examples/projects/file_notes/main.tx
```

```text
--8<-- "website/outputs/file_notes.txt"
```

### csv_roster

`csv.read`, `append`, and `csv.write`.

??? example "`examples/projects/csv_roster/main.tx`"

    ```trux
    --8<-- "examples/projects/csv_roster/main.tx"
    ```

```console
$ printf 'Cara\nops\n' | trux run examples/projects/csv_roster/main.tx
```

```text
--8<-- "website/outputs/csv_roster.txt"
```

## Unit tests

```console
$ trux test examples/projects/scorebook/scores.test.tx
PASS examples/projects/scorebook/scores.test.tx testBestAndPassingCount
PASS examples/projects/scorebook/scores.test.tx testCurveCapsScores
PASS examples/projects/scorebook/scores.test.tx testLetterBoundaries
PASS examples/projects/scorebook/scores.test.tx testPrivateHelpers
PASS examples/projects/scorebook/scores.test.tx testTotalAndAverage
5 passed, 0 failed
```

See [Testing](guide/testing.md).

## HTTP server

`examples/projects/http_server/main.tx` serves `GET /`, `GET /health`, and `POST /echo` on `127.0.0.1:8080`
(Linux and macOS). It runs until you stop it. See the [`http` package](stdlib/http.md) for the full listing.

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

??? example "`examples/gpu_matrix_multiply.tx`"

    ```trux
    --8<-- "examples/gpu_matrix_multiply.tx"
    ```
