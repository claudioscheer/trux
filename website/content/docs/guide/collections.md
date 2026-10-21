---
title: "Arrays, slices, and lists"
description: "Fixed arrays, slice views, and growable lists."
icon: "view_list"
weight: 700
---

Trux has three collection shapes. They all support `len`, indexing, and slicing, and any of them can hold scalars or other
collections.

| | Array `[N]T` | Slice `[]T` | List `list[T]` |
| --- | --- | --- | --- |
| Length | fixed at compile time | fixed when created | grows with `append` |
| Created with | `[N]T{...}` | slicing, `make([]T, n)` | `list[T]{...}` |
| Element assignment | yes | yes (writes through to the underlying storage) | yes |
| `append` | no | no | yes |

## Arrays

```trux
let xs [3]int = [3]int{1, 2, 3}
xs[1] = 9
print(len(xs), " ", xs[0], " ", xs[1])   // 3 1 9
```

The length `N` is a positive integer literal and is part of the type: `[3]int` and `[4]int` are different types. An
array literal must list exactly `N` elements.

## Slices

A slice is a **view** into an array, another slice, or a list. Slicing never copies:

```trux
let ys [4]int = [4]int{1, 2, 3, 4}
let view []int = ys[1:3]   // elements 1 and 2 of ys
view[0] = 8                // ys[1] is now 8
```

Slice expressions follow the usual half-open `[start:end]` rule, and either bound may be left out: `xs[1:]`, `xs[:2]`,
`xs[:]`.

`make([]T, n)` creates new zero-filled storage of length `n` and returns a slice over it:

```trux
let scratch []int = make([]int, 3)   // 0 0 0
```

`make` only creates slices. Its length must be an `int`.

## Lists

Lists are growable. `append(list, value)` adds one element at the end:

```trux
let items list[int] = list[int]{1}
append(items, 2)
print(len(items))   // 2
```

A list variable is a **handle**: assigning or passing a list shares the same underlying list, so a change through one
name is visible through the other.

```trux
let alias list[int] = items
append(alias, 3)
print(len(items))   // 3
```

There is no way to remove elements from a list yet. `examples/mut_parameters.tx` shows a queue that advances a head index
instead.

## Strings as collections

Strings support `len`, indexing, and slicing too. Indexing a string gives a one-byte `string`, not a number:

```trux
let text string = "trux"
print(text[1], " ", text[1:3], " ", len(text))   // r ru 4
```

Strings are immutable: `text[0] = "x"` is rejected.

## Nested collections

Collections can contain collections:

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

## Bounds checks

Every index and slice is checked at run time:

```text
trux runtime error: index 5 out of bounds for length 3
trux runtime error: slice 2:9 out of bounds for length 4
```

## Builtins summary

| Builtin | Works on | Returns |
| --- | --- | --- |
| `len(x)` | string, array, slice, list | `int` |
| `append(list, value)` | list (statement only) | |
| `make([]T, n)` | slice types | a new zero-filled `[]T` |
| `clone(x)` | string, array, slice, list | an owned deep copy, see [Memory and ownership](/docs/guide/memory/) |

## Full example

{{< include "examples/collections.tx" "trux" >}}

{{< include "website/outputs/collections.txt" "text" >}}
