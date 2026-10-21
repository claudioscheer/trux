---
title: "Variables"
description: "Declaring variables with let, assigning to them, and where they are visible."
icon: "data_object"
weight: 300
---

## Declaring with `let`

**What it is.** `let name type = value` creates a variable with a written type and a starting value.

**Why.** An explicit type and a required initial value mean there is never an uninitialized variable, and the type of
every name is visible where it is declared.

**How to use it.**

```trux
let total int = 0
let label string = "score"
let values list[int] = list[int]{1, 2, 3}
let text string, err error = io.readFile("input.txt")
let value int, _ = getPair()
```

The value's type must match the declared type exactly. Functions returning multiple values can be unpacked with comma-separated `let` declarations or discarded with the blank identifier `_`.

**Common mistakes.** Declaring the same name twice in one block:

{{< include "examples/errors/variables_duplicate.tx" "trux" >}}

{{< include "website/outputs/errors/variables_duplicate.txt" "text" >}}

**Related:** [Types and values](/docs/guide/types/).

## Assignment

**What it is.** `name = value` replaces the value of an existing variable. `xs[i] = value` replaces one element of an
array, slice, or list.

**Why.** Loops and accumulators need to update state in place.

**How to use it.**

```trux
let x int = 10
x = x + 1

let xs [3]int = [3]int{1, 2, 3}
xs[0] = 42

let a int = 0
let b int = 0
a, b = getPair()
a, _ = getPair()
```

The new value must have the variable's type.

**Common mistakes.** There are no compound operators such as `+=` or `++`:

{{< include "examples/errors/variables_compound.tx" "trux" >}}

{{< include "website/outputs/errors/variables_compound.txt" "text" >}}

Write `i = i + 1`. Strings cannot be changed by index; see
[Strings as collections](/docs/guide/collections/#strings-as-collections).

**When you don't need it.** A function's collection *parameters* cannot have their elements assigned unless the
parameter is `mut`. See [`mut` parameters](/docs/guide/functions/#mut-parameters).

## Scope

**What it is.** A variable is visible from its `let` to the end of the enclosing block. Each `if`, `else`, and `for`
body is a new block, and a variable declared in a `for` header belongs to that loop.

**Why.** Short-lived names stay local, so they cannot be used by mistake later in the function.

**How to use it.** An inner block may reuse an outer name; the inner variable hides the outer one until the block ends:

```trux
package main

func main() int {
  let x int = 1
  if true {
    let x int = 2   // a new variable in the inner block
    print(x)
  }
  print(x)
  return 0
}
```

```text
2
1
```

**Common mistakes.** Using a variable after its block has ended:

{{< include "examples/errors/variables_undefined.tx" "trux" >}}

{{< include "website/outputs/errors/variables_undefined.txt" "text" >}}

Declare it before the block and assign it inside.

**Related:** [Control flow](/docs/guide/control-flow/), [Functions](/docs/guide/functions/).
