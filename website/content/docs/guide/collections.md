---
title: "Arrays, slices, lists, and maps"
description: "Fixed arrays, slice views, growable lists, hash maps, collection builtins, and bounds checks."
icon: "view_list"
weight: 700
---

Trux has four collection shapes: fixed-size arrays, slice views, growable lists, and key-value maps. Arrays, slices, and
lists are indexed by `int` and support slicing; maps are indexed by any comparable key type `K`.

| | Array `[N]T` | Slice `[]T` | List `list[T]` | Map `map[K]V` |
| --- | --- | --- | --- | --- |
| Size | fixed at compile time | fixed when created | grows with `append` | grows on key assignment, shrinks with `delete` |
| Created with | `[N]T{...}` | slicing, `make([]T, n)` | `list[T]{...}` | `map[K]V{...}`, `make(map[K]V)` |
| Element assignment | yes | yes, writes to underlying storage | yes | yes (`m[k] = v`) |
| `len` | yes | yes | yes | yes |

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

**Common mistakes.** Slicing past the end; see [Bounds checks](#bounds-checks).

**When you don't need it.** If you need your own independent copy, use [`clone`](/docs/guide/memory/#clone).

**Related:** [Memory and ownership](/docs/guide/memory/).

## Lists {#lists}

**What it is.** `list[T]` is a growable sequence. `append(list, value)` adds one element at the end.

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

There is no builtin to remove elements from the middle of a list. `examples/mut_parameters.tx` shows a `Queue` struct
that advances a `head` index instead.

**Common mistakes.** `append` only works on lists:

{{< include "examples/errors/collections_append_array.tx" "trux" >}}

{{< include "website/outputs/errors/collections_append_array.txt" "text" >}}

Appending to a list parameter needs [`mut`](/docs/guide/functions/#mut-parameters).

**When you don't need it.** If the size is fixed, an array or `make([]T, n)` is simpler and cannot be shared by accident.

**Related:** [`mut` parameters](/docs/guide/functions/#mut-parameters), [`csv`](/docs/stdlib/csv/).

## Maps {#maps}

**What it is.** `map[K]V` is a growable hash table mapping keys of type `K` to values of type `V`. Any comparable type
can be a key (`int`, `float`, `string`, `bool`, `error`, pointers `*T`, and arrays or structs composed of comparable
types); `V` can be any type.

**Why.** Fast O(1) lookup, counting, deduplication, and indexing by name or ID.

**How to use it.** Create a map with a map literal `map[K]V{k: v, ...}` or `make(map[K]V)` (optionally with an initial
capacity hint `make(map[K]V, cap)`):

```trux
let stock map[string]int = map[string]int{"bolts": 6, "nuts": 10}
stock["washers"] = 5
delete(stock, "nuts")

// Single-value lookup returns the value or V's zero value when missing:
let count int = stock["bolts"]

// Comma-ok lookup also reports whether the key was present:
let missing int, ok bool = stock["nuts"]   // 0, false

// Snapshot keys or values in insertion order as a list:
let ks list[string] = keys(stock)
let vs list[int] = values(stock)
```

Like lists, a map variable is a **shared handle**: assigning a map copies the handle, use `clone(m)` for an independent
deep copy, and mark map parameters `mut` when a function inserts, updates, or deletes entries.

**Related:** [`mut` parameters](/docs/guide/functions/#mut-parameters), [`clone`](/docs/guide/memory/#clone).

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

**What it is.** Collections whose elements are collections or structs: `[2][2]int`, `list[list[int]]`, `[][]float`,
`map[string]list[int]`.

**Why.** Grids, jagged rows, and grouped records.

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

## Bounds checks {#bounds-checks}

**What it is.** Every array, slice, list, and string index or slice expression is checked at run time.

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
| `len(x)` | string, array, slice, list, map | `int` |
| `append(list, value)` | list (statement only) | |
| `delete(map, key)` | map (statement only) | |
| `keys(map)` | `map[K]V` | `list[K]` in insertion order |
| `values(map)` | `map[K]V` | `list[V]` in insertion order |
| `make([]T, n)` | slice types | a new zero-filled `[]T` |
| `make(map[K]V)` / `make(map[K]V, cap)` | map types | a new empty `map[K]V` |
| `new(T)` | any host type `T` | a durable zero-initialized `*T` |
| `clone(x)` | string, array, slice, list, map, struct | an independent deep copy, see [Memory and ownership](/docs/guide/memory/#clone) |

## Full example

{{< include "examples/collections.tx" "trux" >}}

{{< include "website/outputs/collections.txt" "text" >}}
