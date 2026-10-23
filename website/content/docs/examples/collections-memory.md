---
title: "Collections and memory"
description: "Arrays, slices, lists, maps, structs, pointers, mut parameters, and clone."
icon: "memory"
weight: 30
---

## collections

Arrays, slices, `make`, string indexing, list handles, structs, pointers, and maps.

{{< details summary="`examples/collections.tx`" >}}

{{< include "examples/collections.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/collections.tx
```

{{< include "website/outputs/collections.txt" "text" >}}

## jagged_lists

A `list[list[int]]` with rows of different lengths, summarized with a `RowSummary` struct.

{{< details summary="`examples/jagged_lists.tx`" >}}

{{< include "examples/jagged_lists.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/jagged_lists.tx
```

{{< include "website/outputs/jagged_lists.txt" "text" >}}

## nested_slices

`[][]float` built with `make`, a deep `clone`, and a `CellPos` struct.

{{< details summary="`examples/nested_slices.tx`" >}}

{{< include "examples/nested_slices.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/nested_slices.tx
```

{{< include "website/outputs/nested_slices.txt" "text" >}}

## matrix_multiply_2x2

Passing and returning `Matrix2x2` structs and `*Matrix2x2` pointers.

{{< details summary="`examples/matrix_multiply_2x2.tx`" >}}

{{< include "examples/matrix_multiply_2x2.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/matrix_multiply_2x2.tx
```

{{< include "website/outputs/matrix_multiply_2x2.txt" "text" >}}

## ownership_clone

Borrowed versus owned returns, heap allocation with `new(T)`, and deep-cloning structs and maps with `clone`.

{{< details summary="`examples/ownership_clone.tx`" >}}

{{< include "examples/ownership_clone.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/ownership_clone.tx
```

{{< include "website/outputs/ownership_clone.txt" "text" >}}

## mut_parameters

Mutating caller collections and maps through `mut` parameters, and mutating a `Queue` struct through `*Queue`.

{{< details summary="`examples/mut_parameters.tx`" >}}

{{< include "examples/mut_parameters.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/mut_parameters.tx
```

{{< include "website/outputs/mut_parameters.txt" "text" >}}
