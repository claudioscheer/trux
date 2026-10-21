---
title: "Memory and ownership"
description: "Arenas instead of a garbage collector, borrowing, and clone."
icon: "memory"
weight: 800
---

## Arenas

**What it is.** Strings and collections live in **arenas**: blocks of memory that are released all at once. The compiler
decides which arena each value goes into. There is no garbage collector and no `free` for host values.

**Why.** Arenas make allocation cheap and freeing automatic, with no collector pauses and no manual freeing in your code. The price is a few ownership rules, below, that the compiler enforces.

**How to use it.** Nothing to write. A function's temporary values are released when it returns, and a loop whose
temporaries don't escape an iteration reuses the same memory on every pass, so building strings in a loop doesn't keep
growing memory.

**When you don't need it.** GPU buffers are the exception: they are not in an arena and must be freed with
[`gpu.free`](/docs/guide/gpu/#device-buffers).

**Related:** [Arenas design](/docs/design-notes/arenas/),
[Memory in generated code](/docs/internals/compilation/#memory-in-generated-code).

## Borrowed parameters

**What it is.** A function **borrows** the arrays, slices, lists, and strings passed to it. It can read them, slice them,
and return views of them, but not change them.

**Why.** The compiler has to know which arena owns each collection. Read-only parameters keep that simple, and `mut`
makes every exception visible in the function's signature.

**How to use it.** Read freely. To change the caller's collection, declare the parameter
[`mut`](/docs/guide/functions/#mut-parameters). To work on a changed copy, [`clone`](#clone) it.

**Common mistakes.** `append` or element assignment on a parameter fails with
`cannot mutate parameter-owned collection`; the full example is under
[`mut` parameters](/docs/guide/functions/#mut-parameters).

## Returned values {#returned-values}

**What it is.** A function's return value always stays valid after the function returns.

**Why.** Values built in a function would otherwise disappear with its arena.

**How to use it.** Return whatever you like; the compiler picks the cheap option:

- A view of a **parameter** (`return xs[1:3]`) is returned as a borrowed view of the caller's data. It is free, and it
  sees later changes to that data.
- Something built **locally** (a local array, a list literal, a joined string) is copied into the caller's memory during
  `return`.

**Related:** [Returning collections](/docs/guide/functions/#returning-collections).

## `clone` {#clone}

**What it is.** `clone(x)` makes an independent deep copy of a string, array, slice, or list.

**Why.** It gives you a value you own, which you can change without affecting, or being affected by, anyone else.

**How to use it.**

```trux
func borrowedMiddle(xs []int) []int {
  return xs[1:3]          // a view of the caller's data
}

func ownedMiddle(xs []int) []int {
  return clone(xs[1:3])   // a copy the caller owns
}

func cloneThenEdit(xs []int) []int {
  let owned []int = clone(xs[:])
  owned[0] = 42
  return owned
}
```

The full walkthrough:

{{< include "examples/ownership_clone.tx" "trux" >}}

{{< include "website/outputs/ownership_clone.txt" "text" >}}

**Common mistakes.** The compiler treats a collection returned by a function as having unknown ownership, so it cannot be
changed directly:

{{< include "examples/errors/memory_unknown_ownership.tx" "trux" >}}

{{< include "website/outputs/errors/memory_unknown_ownership.txt" "text" >}}

Clone it first: `let ys list[int] = clone(makeList())`.

**When you don't need it.** Only reading a value never needs `clone`. If a function should change the caller's
collection rather than a copy, use `mut`. Each `clone` copies the whole value, nested collections included, so avoid it
in hot loops.

**Related:** [`mut` parameters](/docs/guide/functions/#mut-parameters), [Lists](/docs/guide/collections/#lists).

## Under the hood

The generated C uses a **durable arena** (`trux_ctx->arena`) for the whole program, a **frame arena** (`trux_frame`)
per function that allocates, released on return, and a **result arena** (`trux_result_arena`) passed in by each caller,
where returned values are copied. Lists are heap-backed handles freed when their owning arena is reset or released. See
[Compiling to C and CUDA](/docs/internals/compilation/#memory-in-generated-code).
