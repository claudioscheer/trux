---
title: "Arrays, slices, and lists"
description: "Fixed arrays, slice views, growable lists, the collection builtins, and bounds checks."
icon: "view_list"
weight: 700
---

Trux has three collection shapes. All of them support `len`, indexing, and slicing, and any of them can hold scalars or
other collections.

| | Array `[N]T` | Slice `[]T` | List `list[T]` |
| --- | --- | --- | --- |
| Length | fixed at compile time | fixed when created | grows with `append` |
| Created with | `[N]T{...}` | slicing, `make([]T, n)` | `list[T]{...}` |
| Element assignment | yes | yes, writes to the underlying storage | yes |
| `append` | no | no | yes |

## Arrays

**What it is.** `[N]T` holds exactly `N` values of type `T`. `N` is a positive integer literal and part of the type, so
`[3]int` and `[4]int` are different types.

**Why.** Fixed-size data such as a 2x2 matrix or an RGB triple has a known size, and the compiler can check literals
against it.

**How to use it.**

```trux
let xs [3]int = [3]int{1, 2, 3}
xs[1] = 9
print(len(xs), " ", xs[0], " ", xs[1])   // 3 1 9
```

**Common mistakes.** A literal with the wrong number of elements:

{{< include "examples/errors/collections_array_length.tx" "trux" >}}

{{< include "website/outputs/errors/collections_array_length.txt" "text" >}}

**When you don't need it.** If the size is only known at run time, use `make([]T, n)`. If it grows, use a list.

**Related:** [Slices](#slices).

## Slices

**What it is.** `[]T` is a **view** into an array, another slice, or a list. Slicing never copies.

**Why.** Functions can work on part of a collection without copying it, and one function can accept arrays of any
length through `xs[:]`.

**How to use it.** `xs[start:end]` is half-open; either bound may be left out (`xs[1:]`, `xs[:2]`, `xs[:]`). Writes
through a slice change the underlying storage:

```trux
let ys [4]int = [4]int{1, 2, 3, 4}
let view []int = ys[1:3]   // elements 1 and 2 of ys
view[0] = 8                // ys[1] is now 8
```

`make([]T, n)` creates new zero-filled storage of length `n` and returns a slice over it:

```trux
let scratch []int = make([]int, 3)   // 0 0 0
```

`make` only creates slices, and its length must be an `int`.

**Common mistakes.** Slicing past the end; see [Bounds checks](#bounds-checks).

**When you don't need it.** If you need your own independent copy, use [`clone`](/docs/guide/memory/#clone).

**Related:** [Memory and ownership](/docs/guide/memory/).

## Lists

**What it is.** `list[T]` is a growable collection. `append(list, value)` adds one element at the end.

**Why.** Reading input of unknown size, such as CSV rows or lines from stdin, needs a collection that grows.

**How to use it.**

```trux
let items list[int] = list[int]{1}
append(items, 2)
print(len(items))   // 2
```

A list variable is a **handle**: assigning or passing a list shares the same list, so a change through one name is
visible through the other:

```trux
let alias list[int] = items
append(alias, 3)
print(len(items))   // 3
```

There is no way to remove elements yet. `examples/mut_parameters.tx` shows a queue that advances a head index instead.

**Common mistakes.** `append` only works on lists:

{{< include "examples/errors/collections_append_array.tx" "trux" >}}

{{< include "website/outputs/errors/collections_append_array.txt" "text" >}}

Appending to a list parameter needs [`mut`](/docs/guide/functions/#mut-parameters).

**When you don't need it.** If the size is fixed, an array or `make([]T, n)` is simpler and cannot be shared by accident.

**Related:** [`mut` parameters](/docs/guide/functions/#mut-parameters), [`csv`](/docs/stdlib/csv/).

## Strings as collections {#strings-as-collections}

**What it is.** Strings support `len`, indexing, and slicing. Indexing gives a one-byte `string`, not a number.

**Why.** Simple text processing works with the same tools as other collections.

**How to use it.**

```trux
let text string = "trux"
print(text[1], " ", text[1:3], " ", len(text))   // r ru 4
```

**Common mistakes.** Strings are immutable:

{{< include "examples/errors/collections_string_assign.tx" "trux" >}}

{{< include "website/outputs/errors/collections_string_assign.txt" "text" >}}

Build a new string with `+` instead: `"T" + text[1:]`.

## Nested collections

**What it is.** Collections whose elements are collections: `[2][2]int`, `list[list[int]]`, `[][]int`.

**Why.** Grids and rows of different lengths without a struct type.

**How to use it.**

{{< tabs tabTotal="3" >}}
{{% tab tabName="Arrays of arrays" %}}

```trux
let m [2][2]int = [2][2]int{[2]int{1, 2}, [2]int{3, 4}}
print(m[1][0])   // 3
```

{{% /tab %}}
{{% tab tabName="Jagged lists" %}}

{{< include "examples/jagged_lists.tx" "trux" >}}

{{% /tab %}}
{{% tab tabName="Slices of slices" %}}

{{< include "examples/nested_slices.tx" "trux" >}}

{{% /tab %}}
{{< /tabs >}}

**When you don't need it.** For numeric grids that go to the GPU or to an image file, use one flat slice indexed as
`row * width + col`: that is the layout [`gpu`](/docs/guide/gpu/) buffers and [`image`](/docs/stdlib/image/) use.

## Bounds checks

**What it is.** Every index and slice expression is checked at run time.

**Why.** An out-of-range access in C reads or overwrites unrelated memory. In Trux it stops the program at once.

**How to use it.** Nothing to write. Keep indexes below `len(xs)`.

**Common mistakes.**

{{< include "examples/errors/collections_index_out_of_bounds.tx" "trux" >}}

{{< include "website/outputs/errors/collections_index_out_of_bounds.txt" "text" >}}

{{< include "website/outputs/errors/collections_slice_out_of_bounds.txt" "text" >}}

**When you don't need it.** The checks are always on for host code. Indexing a `gpu.Buffer` inside a kernel is **not**
bounds-checked; see [GPU programming](/docs/guide/gpu/#kernels).

**Related:** [Checked integer arithmetic](/docs/guide/operators/#checked-integer-arithmetic).

## Builtins summary

| Builtin | Works on | Returns |
| --- | --- | --- |
| `len(x)` | string, array, slice, list | `int` |
| `append(list, value)` | list (statement only) | |
| `make([]T, n)` | slice types | a new zero-filled `[]T` |
| `clone(x)` | string, array, slice, list | an independent deep copy, see [Memory and ownership](/docs/guide/memory/#clone) |

## Full example

{{< include "examples/collections.tx" "trux" >}}

{{< include "website/outputs/collections.txt" "text" >}}
