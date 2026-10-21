---
title: "Memory and ownership"
description: "Arenas instead of a garbage collector, and the ownership rules that follow."
icon: "memory"
weight: 800
---

Trux has no garbage collector and no `free` for ordinary values. Instead, the generated program allocates strings and
collections in **arenas**: big blocks of memory that are released all at once. The compiler decides which arena each
value lives in, and a few simple rules in the language keep that safe.

You don't have to think about arenas to write Trux. You do need to know three rules, because the compiler enforces them.

## Rule 1: functions borrow their collection arguments

When you pass an array, slice, list, or string to a function, the function **borrows** it. It can read it, slice it, and
return views of it, but it cannot change it:

```trux
func bad(xs list[int]) int {
  append(xs, 1) // rejected: cannot mutate parameter-owned collection
  return len(xs)
}
```

To let a function change the caller's collection, mark the parameter [`mut`](/docs/guide/functions/#mut-parameters).

## Rule 2: returned values are safe to use

A function's return value always stays valid after the function returns:

- If it returns a view of one of its **parameters** (for example `return xs[1:3]`), the result is a *borrowed* view of the
  caller's data. It's cheap, and it sees later changes to that data.
- If it returns something built **locally** (a local array, a list literal, a concatenated string), the compiler copies it
  into memory owned by the caller during `return`.

## Rule 3: use `clone` when you need your own copy

`clone(x)` makes an independent deep copy of a string, array, slice, or list.

```trux
func borrowedMiddle(xs []int) []int {
  return xs[1:3]          // a view of the caller's data
}

func ownedMiddle(xs []int) []int {
  return clone(xs[1:3])   // a copy the caller owns
}
```

Two common situations call for `clone`:

1. **Editing a parameter's data.** Clone it into a local, then change the local:

    ```trux
    func cloneThenEdit(xs []int) []int {
      let owned []int = clone(xs[:])
      owned[0] = 42
      return owned
    }
    ```

2. **Changing a collection returned by a function.** The compiler treats a function's returned collection as having
   unknown ownership, so `append` or element assignment on it is rejected
   (`cannot mutate collection with unknown ownership`). Clone it first:

    ```trux
    let ys list[int] = clone(makeList())
    append(ys, 4)
    ```

## The full walkthrough

{{< include "examples/ownership_clone.tx" "trux" >}}

{{< include "website/outputs/ownership_clone.txt" "text" >}}

## Under the hood

The generated C uses:

- a **durable arena** (`trux_ctx->arena`) that lives for the whole program,
- a **frame arena** (`trux_frame`) for functions that allocate local dynamic values, created on entry and released when
  the function returns, and
- a **result arena** (`trux_result_arena`) passed in by each caller; values that are returned are copied there.

Loops whose allocations don't escape an iteration rewind the frame arena at the end of each iteration, so a loop that
builds temporary strings doesn't keep growing memory. Lists are heap-backed handles tracked by the runtime and freed when
their owning arena is reset or released.

For the full rationale see [Arenas](/docs/design-notes/arenas/), and for how it looks in generated code see
[Compiling to C and CUDA](/docs/internals/compilation/#memory-in-generated-code).
