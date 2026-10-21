---
title: "Functions"
description: "Declaring and calling functions, returns, recursion, and mut parameters."
icon: "functions"
weight: 500
---

## Declaring and calling functions

**What it is.** `func name(param type, ...) returnType { ... }` declares a function. A call is `name(args)`.

**Why.** Functions are the only top-level declarations in Trux; all code lives in them.

**How to use it.**

```trux
func add(a int, b int) int {
  return a + b
}
```

- Every parameter has its own type. There is no shorthand like `a, b int`.
- The return type is **required**. A function returns a single type or multiple types in parentheses (e.g. `(int, error)`).
  Functions that exist for their side effects conventionally return `0` or `error`.
- Arguments must match the parameter types exactly.
- Functions can be declared in any order, and they can call themselves:

{{< include "examples/fibonacci.tx" "trux" >}}

{{< include "website/outputs/fibonacci.txt" "text" >}}

A call returning a single non-error value can stand alone as a statement when you don't need its result (`printIterative(count)` above). Calls returning multiple values or an `error` must be unpacked or explicitly ignored with `_`.

**Common mistakes.** Leaving out the return type:

{{< include "examples/errors/functions_no_return_type.tx" "trux" >}}

{{< include "website/outputs/errors/functions_no_return_type.txt" "text" >}}

Passing an argument of the wrong type:

{{< include "website/outputs/errors/functions_argument_type.txt" "text" >}}

**When you don't need it.** Kernels (`kernel func`) and test functions are the two kinds of function that have no
return type. See [GPU programming](/docs/guide/gpu/#kernels) and [Testing](/docs/guide/testing/).

**Related:** [Returning on every path](#returning-on-every-path), [Modules: `pub func`](/docs/guide/modules/#pub-func).

## Returning on every path

**What it is.** Every path through a function body must end in `return value`.

**Why.** A function that falls off its end would return garbage in C. The compiler rules that out instead of guessing.

**How to use it.** End the function with a `return`, or make the last statement an `if` / `else` whose branches all
return:

{{< include "examples/return_paths.tx" "trux" >}}

{{< include "website/outputs/return_paths.txt" "text" >}}

**Common mistakes.** An `if` / `else if` chain without a final `else`, or a loop, as the last statement:

{{< include "examples/errors/functions_missing_return.tx" "trux" >}}

{{< include "website/outputs/errors/functions_missing_return.txt" "text" >}}

The compiler does not work out whether a loop always returns. Add a final `return` after it.

**Related:** [Control flow](/docs/guide/control-flow/).

## Returning collections

**What it is.** Functions can take and return arrays, slices, lists, and strings.

**Why.** Building a result in a function and handing it back is the normal way to structure work in a language without
structs.

**How to use it.**

{{< include "examples/matrix_multiply_2x2.tx" "trux" >}}

{{< include "website/outputs/matrix_multiply_2x2.txt" "text" >}}

The returned value always stays valid after the function returns. Whether it is a view or a copy is decided by the
ownership rules in [Memory and ownership](/docs/guide/memory/#returned-values).

**Common mistakes.** Changing a returned collection directly is rejected; see
[unknown ownership](/docs/guide/memory/#clone).

## `mut` parameters {#mut-parameters}

**What it is.** `mut` on an array, slice, or list parameter lets the function change the caller's collection.

**Why.** By default a function can only **read** the collections it is given. That makes every call that can change your
data visible in the function's signature.

**How to use it.** Mark the parameter, not the call:

```trux
func push(mut xs list[int], value int) int {
  append(xs, value)
  return len(xs)
}

func main() int {
  let items list[int] = list[int]{1}
  print(push(items, 2), " ", len(items))   // 2 2
  return 0
}
```

- The argument must be something the caller may change: its own local collection, or its own `mut` parameter.
- Assigning a whole new collection to a `mut` parameter only rebinds the local name; the caller's variable is unchanged.

The full walkthrough:

{{< include "examples/mut_parameters.tx" "trux" >}}

{{< include "website/outputs/mut_parameters.txt" "text" >}}

**Common mistakes.** Changing an ordinary collection parameter:

{{< include "examples/errors/functions_mutate_parameter.tx" "trux" >}}

{{< include "website/outputs/errors/functions_mutate_parameter.txt" "text" >}}

Passing a collection the caller only borrowed to a `mut` parameter:

{{< include "examples/errors/functions_mut_borrowed.tx" "trux" >}}

{{< include "website/outputs/errors/functions_mut_borrowed.txt" "text" >}}

Using `mut` on a scalar:

{{< include "website/outputs/errors/functions_mut_scalar.txt" "text" >}}

Scalars are passed by value; return the new value instead.

**When you don't need it.** If the function only reads the collection, leave `mut` off. If it needs its own modified
copy, [`clone`](/docs/guide/memory/#clone) the parameter into a local instead. Kernel parameters cannot be `mut`.

**Related:** [Memory and ownership](/docs/guide/memory/), [Modules: `mut` across packages](/docs/guide/modules/).
