---
title: "Types and values"
description: "int, float, string, bool, their literals, and why Trux never converts between them."
icon: "category"
weight: 200
---

## Scalar types

**What it is.** Built-in value types:

| Type | Meaning | Literals | In generated C |
| --- | --- | --- | --- |
| `int` | 64-bit signed integer | `0`, `42`, `-7` | `int64_t` |
| `float` | 64-bit IEEE 754 number | `1.5`, `2.0`, `0.25` | `double` |
| `string` | immutable sequence of bytes | `"hello"`, `"line\n"` | `rt_string` (pointer + length) |
| `bool` | truth value | `true`, `false` | `bool` |
| `error` | error value or nil | `nil` | `rt_error` |

**Why.** A small, fixed set of types that map one-to-one onto C keeps the generated code predictable and fast.

**How to use it.** Every variable states its type:

{{< include "examples/primitives.tx" "trux" >}}

{{< include "website/outputs/primitives.txt" "text" >}}

`print` writes floats with up to 15 significant digits and drops a trailing `.0`, so `print(2.0)` prints `2`.

**When you don't need it.** For a group of values, use a [collection](/docs/guide/collections/). For GPU memory, use
[`gpu.Buffer`](/docs/guide/gpu/#device-buffers).

**Related:** [Variables](/docs/guide/variables/), [Operators](/docs/guide/operators/).

## Literals

**What it is.** The written forms of values.

**Why.** Each literal has exactly one type, so the type of every expression is obvious from the code.

**How to use it.**

- Integers are decimal digits: `0`, `9223372036854775807`. Negative numbers use unary minus: `-3`.
- Floats need digits on both sides of the dot: `1.0`, `0.5`. `1.`, `.5`, `1e9`, and hex are not supported.
- Strings use double quotes, fit on one line, and support the escapes `\n`, `\t`, `\"`, and `\\`. `len(s)` counts
  bytes.
- `nil` is the literal for an `error` that represents success (no error).

**Common mistakes.** Any other escape is a compile error:

{{< include "examples/errors/types_bad_escape.tx" "trux" >}}

{{< include "website/outputs/errors/types_bad_escape.txt" "text" >}}

**Related:** [Strings as collections](/docs/guide/collections/#strings-as-collections).

## No implicit conversions {#no-implicit-conversions}

**What it is.** Trux never changes a value's type for you. `int` and `float` cannot be mixed, and conditions must be
`bool`.

**Why.** Silent conversions hide precision loss and overflow. Here every conversion would have to be visible in the code,
and the compiler catches the rest.

**How to use it.** Keep each expression in one type: write `1.0 / 2.0` in float code, and `if count > 0`, not
`if count`. There are currently **no conversion functions** between `int`, `float`, and `string`.

**Common mistakes.** Mixing `int` and `float` in arithmetic:

{{< include "examples/errors/types_mixed_numbers.tx" "trux" >}}

{{< include "website/outputs/errors/types_mixed_numbers.txt" "text" >}}

Using an integer literal where a float is expected:

{{< include "website/outputs/errors/types_int_to_float.txt" "text" >}}

Using a number as a condition:

{{< include "examples/errors/types_if_int.tx" "trux" >}}

{{< include "website/outputs/errors/types_if_int.txt" "text" >}}

**When you don't need it.** If you need to print a computed float as whole units, keep the value in `int` (for example
nanoseconds) and print parts of it, as `examples/gpu_matrix_multiply.tx` does for seconds.

**Related:** [Operators](/docs/guide/operators/), [FAQ: what's missing](/docs/faq/#whats-missing).

## Collection and buffer types

**What it is.** Types built from other types: `[N]T` arrays, `[]T` slices, `list[T]` lists, and, in GPU programs,
`gpu.Buffer[int]` and `gpu.Buffer[float]`.

**Why.** They cover fixed data, views, growable data, and GPU memory without a general type system.

**How to use it.** Element types can nest: `[2][2]int`, `[][]float`, `list[list[int]]`. See
[Arrays, slices, and lists](/docs/guide/collections/) and [GPU programming](/docs/guide/gpu/).

**When you don't need it.** There are no structs, maps, tuples, enums, pointers, function values, generics, or
user-defined types. Use parallel lists or a flat list with computed indexes instead, as [`csv`](/docs/stdlib/csv/) does.

**Related:** [Memory and ownership](/docs/guide/memory/).
