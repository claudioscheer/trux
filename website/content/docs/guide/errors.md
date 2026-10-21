---
title: "Errors and failures"
description: "What compile errors and runtime errors look like, and how a Trux program reports failure."
icon: "report"
weight: 1050
---

Trux has no exceptions and no error values. A mistake is either caught by the compiler, or it stops the running program
with a message. Either way the command exits with a non-zero status.

## Compile errors

**What it is.** When the compiler rejects a program it prints `path:line:column: message`, the surrounding source with a
caret under the problem, and exits with status `1`. Nothing is built or run.

**Why.** Type mistakes, missing returns, and ownership violations are cheaper to fix before the program runs.

**How to use it.** Read the first line, then the caret. Only the **first** error is reported; fix it and compile again.

{{< include "examples/errors/functions_missing_return.tx" "trux" >}}

{{< include "website/outputs/errors/functions_missing_return.txt" "text" >}}

Errors in an imported file point at that file. Errors go to stderr.

**Related:** [FAQ: common errors](/docs/faq/#common-errors). Every guide page lists the errors for its features.

## Runtime errors {#runtime-errors}

**What it is.** A check that fails while the program runs prints `trux runtime error: <message>` to stderr and exits
with status `1`. Integer overflow, division by zero, and `assert` add an `at file:line` line; bounds checks and library
errors currently don't.

**Why.** Failing at the faulty operation is easier to debug than a wrong result found much later.

**How to use it.** These checks are always on:

| Check | Message | Page |
| --- | --- | --- |
| integer overflow | `integer overflow in +` | [Operators](/docs/guide/operators/#checked-integer-arithmetic) |
| division or remainder by zero | `integer division by zero` | [Operators](/docs/guide/operators/#checked-integer-arithmetic) |
| index or slice out of range | `index 5 out of bounds for length 3` | [Collections](/docs/guide/collections/#bounds-checks) |
| failed `assert` | `assertion failed: <message>` | [Control flow](/docs/guide/control-flow/#assert) |
| bad input, missing file, malformed CSV or PPM | `invalid int input`, `cannot open file …` | [Standard library](/docs/stdlib/#runtime-errors) |
| CUDA call failed | `CUDA <operation> failed: …` | [GPU](/docs/guide/gpu/#kernel-errors) |

Under `trux run`, a failed program is followed by `execute <temp path>: exit status N`, printed by `trux` itself:

{{< include "website/outputs/errors/operators_divide_by_zero.txt" "text" >}}

**Common mistakes.** Expecting to recover. There is no `try`, and library calls do not return errors. Check what you can
before the call (for example, read input with `io.readLine` and validate it yourself), and treat a runtime error as a
bug to fix.

**When you don't need it.** Don't add `assert`s that repeat the built-in checks above.

## Exit codes {#exit-codes}

**What it is.** The value `main` returns is the program's exit code.

**Why.** Scripts and CI can tell success from failure.

**How to use it.** Return `0` for success. A runtime error always exits with `1`.

- A binary made with `trux build -o app main.tx` exits with exactly the value `main` returned.
- `trux run` exits with `0` when the program returned `0`, and with `1` otherwise, after printing
  `execute …: exit status N` with the program's real code `N`.
- `trux test` exits with `1` when any test fails.

**Related:** [CLI: exit statuses](/docs/cli/#exit-statuses).

## C compiler errors

**What it is.** A message starting with `cc failed:` or `nvcc failed:` comes from the C or CUDA compiler, after Trux
accepted the program.

**Why it matters.** For a program that passed the Trux checks, this is either a missing tool or a compiler bug.

- `nvcc failed: exec: "nvcc": executable file not found in $PATH` means the program has a `kernel func` and the CUDA
  toolkit is not installed. See [GPU requirements](/docs/guide/gpu/#requirements).
- Known compiler bugs that surface as `cc failed`: [GPU coordinate helpers in host code](/docs/guide/gpu/#coordinate-helpers)
  and [GPU buffers in a program without kernels](/docs/guide/gpu/#device-buffers).

Please [report](https://github.com/claudioscheer/trux/issues) any other `cc failed` error.
