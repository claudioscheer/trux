---
title: "Types and values"
description: "int, float, string, bool, error, structs, pointers, their literals, and why Trux never converts between them."
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

**Why.** A small, fixed set of scalar types that map one-to-one onto C keeps the generated code predictable and fast.

**How to use it.** Every variable states its type:

{{< include "examples/primitives.tx" "trux" >}}

{{< include "website/outputs/primitives.txt" "text" >}}

`print` writes floats with up to 15 significant digits and drops a trailing `.0`, so `print(2.0)` prints `2`.

**When you don't need it.** For named fields, use a [struct](#structs). For sequences or key-value tables, use a
[collection](/docs/guide/collections/). For GPU memory, use [`gpu.Buffer`](/docs/guide/gpu/#device-buffers).

**Related:** [Variables](/docs/guide/variables/), [Operators](/docs/guide/operators/).

## Literals

**What it is.** The written forms of values.

**Why.** Each literal has an explicit type, so the type of every expression is obvious from the code.

**How to use it.**

- Integers are decimal digits: `0`, `9223372036854775807`. Negative numbers use unary minus: `-3`.
- Floats need digits on both sides of the dot: `1.0`, `0.5`. `1.`, `.5`, `1e9`, and hex are not supported.
- Strings use double quotes, fit on one line, and support the escapes `\n`, `\t`, `\"`, and `\\`. `len(s)` counts
  bytes.
- `nil` is the zero literal for both `error` (no error) and pointer types `*T` (null pointer).

**Common mistakes.** Any other escape is a compile error:

{{< include "examples/errors/types_bad_escape.tx" "trux" >}}

{{< include "website/outputs/errors/types_bad_escape.txt" "text" >}}

**Related:** [Strings as collections](/docs/guide/collections/#strings-as-collections).

## Structs {#structs}

**What it is.** `type Name struct { ... }` defines a composite value type with named, explicitly typed fields. Prefix
with `pub` (`pub type Name struct`) to export the struct and its fields to files that import your package.

**Why.** Structs group related fields into a single value that can be passed, returned, stored in collections, or
referenced by pointer.

**How to use it.** Construct a struct with named fields (`Name{field: value}`). Omitted fields receive their zero value
(`0`, `0.0`, `""`, `false`, `nil`, zero-filled array, empty slice, empty list, or empty map):

```trux
type Point struct {
  x int
  y int
}

func main() int {
  let p Point = Point{x: 3, y: 4}
  let origin Point = Point{}
  p.x = p.x + 1
  print(p.x, " ", p.y, " ", origin.x)   // 4 4 0
  return 0
}
```

- Structs are passed by value unless you pass a pointer `*Point`.
- Structs whose fields are comparable (scalars, pointers, or arrays/structs of comparable types) can be compared with
  `==` and `!=`, and used as map keys (`map[Point]int`).
- Recursive structs must go through a pointer (e.g. `type Node struct { val int; next *Node }`) or a dynamic collection;
  direct by-value cycles are rejected at compile time.

**Related:** [Pointers](#pointers), [Modules: `pub type`](/docs/guide/modules/#pub-func).

## Pointers {#pointers}

**What it is.** `*T` is a typed pointer to a value of type `T`. `&x` takes the address of an assignable variable, field,
index, or struct literal; `*p` dereferences `p`; `new(T)` allocates a zero-initialized `T` in the durable arena and
returns `*T`; and `nil` is the null pointer literal.

**Why.** Pointers let functions mutate caller state in place (`func advance(c *Counter)`), build linked data structures
(`*Node`), and allocate heap values (`new(T)`) that outlive the current function frame.

**How to use it.**

```trux
type Counter struct {
  value int
}

func advance(c *Counter) int {
  c.value = c.value + 1   // auto-dereferences `c`
  return c.value
}

func main() int {
  let local Counter = Counter{value: 10}
  print(advance(&local), " ", local.value)   // 11 11

  let heap *Counter = new(Counter)
  *heap = Counter{value: 41}
  print(advance(heap))                       // 42
  return 0
}
```

- Accessing a field on a struct pointer (`c.value`) automatically dereferences one pointer level, just like `(*c).value`.
- Dereferencing or accessing a field through a `nil` pointer stops the program immediately with
  `trux runtime error: nil pointer dereference`.
- Returning the address of a stack local (`let x int = 1; return &x`) is rejected at compile time; use `new(T)` or
  `&StructLiteral{...}` when returning a pointer from a function.
- There is no pointer arithmetic.

**Related:** [Memory and ownership](/docs/guide/memory/), [`mut` parameters](/docs/guide/functions/#mut-parameters).

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

**What it is.** Types built from other types: `[N]T` arrays, `[]T` slices, `list[T]` lists, `map[K]V` maps, and, in GPU
programs, `gpu.Buffer[int]` and `gpu.Buffer[float]`.

**Why.** They cover fixed data, views, growable sequences, key-value lookup tables, and GPU memory.

**How to use it.** Element types can nest and combine with structs and pointers: `[2][2]int`, `[][]float`,
`list[Item]`, `map[string]list[int]`. See [Arrays, slices, lists, and maps](/docs/guide/collections/) and
[GPU programming](/docs/guide/gpu/).

**Related:** [Memory and ownership](/docs/guide/memory/).
