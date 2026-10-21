---
title: "Operators and expressions"
description: "Arithmetic, checked integer math, comparison, logic, string containment, and precedence."
icon: "calculate"
weight: 400
---

## Arithmetic

**What it is.** `+ - * / %` and unary `-` on numbers, and `+` to join strings.

| Operator | `int` | `float` | `string` |
| --- | --- | --- | --- |
| `+` | addition | addition | concatenation |
| `-` | subtraction | subtraction | |
| `*` | multiplication | multiplication | |
| `/` | division, truncates toward zero | division | |
| `%` | remainder | | |
| unary `-` | negation | negation | |

**Why.** These are the operators C has, with the same meaning, so the generated code is direct.

**How to use it.** Both operands must have the same type:

{{< include "examples/arithmetic.tx" "trux" >}}

{{< include "website/outputs/arithmetic.txt" "text" >}}

**Common mistakes.** Mixing `int` and `float`, or a string and a number, is a compile error. See
[No implicit conversions](/docs/guide/types/#no-implicit-conversions).

**Related:** [Checked integer arithmetic](#checked-integer-arithmetic).

## Checked integer arithmetic {#checked-integer-arithmetic}

**What it is.** Integer `+`, `-`, `*`, `/`, `%`, and unary `-` are checked at run time. Signed overflow, division by
zero, and remainder by zero stop the program with exit status `1`.

**Why.** In C, signed overflow is undefined behavior and division by zero crashes or gives garbage. Trux turns both into
a clear error with the source line, so a wrong result never goes unnoticed.

**How to use it.** Nothing to write: every integer operation is checked. The checks also run inside GPU kernels, where a
failure traps the kernel instead of printing a message (see [GPU programming](/docs/guide/gpu/#kernel-errors)).

**Common mistakes.** Overflowing the 64-bit range:

{{< include "examples/errors/operators_overflow.tx" "trux" >}}

{{< include "website/outputs/errors/operators_overflow.txt" "text" >}}

Dividing by a value that turns out to be zero:

{{< include "examples/errors/operators_divide_by_zero.tx" "trux" >}}

{{< include "website/outputs/errors/operators_divide_by_zero.txt" "text" >}}

The `at` line names the file and line of the failing operation. The `execute …` line is printed by `trux run`; see
[Errors and failures](/docs/guide/errors/#runtime-errors).

**When you don't need it.** Float arithmetic is not checked; it follows IEEE 754, so `1.0 / 0.0` is `inf`. If a value
can legitimately be zero, test it before dividing.

**Related:** [Bounds checks](/docs/guide/collections/#bounds-checks), [Errors and failures](/docs/guide/errors/).

## Comparison

**What it is.** `==` and `!=` on two values of the same scalar type, and `<`, `<=`, `>`, `>=` on two `int` or two
`float` values. The result is a `bool`.

**Why.** Comparisons feed `if` and `for` conditions, which must be `bool`.

**How to use it.** Strings compare by content: `"ab" == "a" + "b"` is `true`.

**Common mistakes.** Collections cannot be compared with `==`:

{{< include "examples/errors/operators_compare_collections.tx" "trux" >}}

{{< include "website/outputs/errors/operators_compare_collections.txt" "text" >}}

Compare element by element in a loop instead.

**Related:** [Control flow](/docs/guide/control-flow/).

## Logic

**What it is.** `&&` (and), `||` (or), and `!` (not) on `bool` values. `&&` and `||` short-circuit: the right side runs
only when it can change the result.

**Why.** Short-circuiting lets you guard a check, as in `i < len(xs) && xs[i] > 0`, without an out-of-bounds error.

**How to use it.** All operands must be `bool`. There is no truthiness: `if count` is rejected, write `if count > 0`.

**Related:** [Comparison](#comparison).

## String containment: `in`

**What it is.** `needle in haystack` is `true` when the string `needle` appears in the string `haystack`.

**Why.** Substring search is common enough to deserve an operator.

**How to use it.** Both sides must be `string`. An empty needle is always found.

{{< include "examples/strings.tx" "trux" >}}

{{< include "website/outputs/strings.txt" "text" >}}

**When you don't need it.** To compare whole strings, use `==`.

## Precedence

From lowest to highest binding. Operators on the same row are left-associative.

| Level | Operators |
| --- | --- |
| 1 (lowest) | `||` |
| 2 | `&&` |
| 3 | `==` `!=` `<` `<=` `>` `>=` `in` |
| 4 | `+` `-` |
| 5 | `*` `/` `%` |
| 6 (highest) | unary `-` and `!`, then calls, indexing `x[i]`, and slicing `x[a:b]` |

Use parentheses whenever they make the intent clearer: `(a + b) / 2.0`.

## Statement-only builtins

**What it is.** Calls that produce no value and can only stand alone on a line: `print`, `assert`, `append`,
`time.sleepMillis`, `http.serve`, `http.respond`, and the GPU copy,
launch, `sync`, and `free` calls. (File writes like `io.writeFile`, `csv.write`, and `image.writePpm` return `error`.)

**Why.** They exist for their effect. Using one inside an expression is always a mistake, so the compiler rejects it.

**How to use it.** Put each on its own line. A call to your own function may also be used as a statement when its result
is discarded (unless it returns multiple values or an `error`, which must be explicitly handled or bound to `_`).

**Common mistakes.**

{{< include "examples/errors/operators_print_value.tx" "trux" >}}

{{< include "website/outputs/errors/operators_print_value.txt" "text" >}}

**Related:** [Standard library](/docs/stdlib/).
