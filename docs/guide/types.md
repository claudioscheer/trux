# Types and values

## Scalar types

| Type | Meaning | Literals | In generated C |
| --- | --- | --- | --- |
| `int` | 64-bit signed integer | `0`, `42`, `1000000` | `int64_t` |
| `float` | 64-bit IEEE 754 floating point | `1.5`, `2.0`, `0.25` | `double` |
| `string` | immutable sequence of bytes | `"hello"`, `"line\n"` | `rt_string` (pointer + length) |
| `bool` | truth value | `true`, `false` | `bool` |

```trux
package main

func main() int {
  let count int = 3
  let ratio float = 0.75
  let name string = "trux"
  let ready bool = true

  print(name, " ", count, " ", ratio, " ", ready)
  return 0
}
```

```text
trux 3 0.75 true
```

### Number literals

- Integer literals are decimal digits: `0`, `7`, `9223372036854775807`.
- Float literals need digits on both sides of the dot: `1.0`, `0.5`. Forms such as `1.`, `.5`, `1e9`, or hex are not
  supported.
- Negative numbers use the unary minus operator: `-3`, `-2.5`.

### String literals

Strings use double quotes and must fit on one line. Supported escapes:

| Escape | Meaning |
| --- | --- |
| `\n` | newline |
| `\t` | tab |
| `\"` | double quote |
| `\\` | backslash |

Any other escape is a compile error. A Trux string is a byte string (data pointer plus length), not a NUL-terminated
C string. `len(s)` counts bytes.

### Printing floats

`print` writes floats with up to 15 significant digits and drops a trailing `.0`, so `print(2.0)` prints `2` and
`print(1.0 / 3.0)` prints `0.333333333333333`.

## No implicit conversions

Trux never converts between types for you:

- `int` and `float` cannot be mixed in arithmetic or comparisons: `1 + 2.0` is rejected.
- Conditions must be `bool`: `if 1 { ... }` is rejected.
- There are currently **no conversion functions** between `int`, `float`, and `string`. The examples work around this,
  for example `examples/gpu_matrix_multiply.tx` prints a seconds value by splitting an integer number of nanoseconds into
  whole seconds and milliseconds.

## Collection types

| Type | What it is |
| --- | --- |
| `[N]T` | fixed-size array of exactly `N` elements (`N` is a positive integer literal) |
| `[]T` | slice: a view into an array, another slice, or a list |
| `list[T]` | growable list; assigning or passing it shares the same list |

Element types can be any type above, including other collections: `[2][2]int`, `[][]float`, and `list[list[int]]` are all
valid. Collections are covered in [Arrays, slices, and lists](collections.md).

## GPU buffer type

`gpu.Buffer[int]` and `gpu.Buffer[float]` are handles to memory on an NVIDIA GPU. They only exist in programs that
`import "gpu"`. See [GPU programming](gpu.md).

## What's not there

Trux has no structs, maps, tuples, enums, pointers, function values, generics, or user-defined types. Top-level code is
functions only.
