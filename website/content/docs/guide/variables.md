---
title: "Variables"
description: "Declaring, assigning, and scoping variables with let."
icon: "data_object"
weight: 300
---

## Declaring with `let`

```trux
let name type = value
```

Every variable has an explicit type and an initial value:

```trux
let total int = 0
let label string = "score"
let values list[int] = list[int]{1, 2, 3}
```

The value's type must match the declared type exactly.

## Assigning

Assignment updates an existing variable. The new value must have the same type.

```trux
let x int = 10
x = x + 1
```

There are no compound operators such as `+=` or `++`. Write `i = i + 1`.

Elements of arrays, slices, and lists can be assigned by index:

```trux
let xs [3]int = [3]int{1, 2, 3}
xs[0] = 42
```

Strings are immutable, so `s[0] = "x"` is rejected.

## Scope

A variable is visible from its `let` to the end of the enclosing block. Each `if`, `else`, and `for` body is a new block,
and a variable declared in a three-part `for` header belongs to that loop.

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

Declaring the same name twice in the **same** block is an error (`duplicate local variable "x"`). A declaration in an
inner block may reuse an outer name, as above.

## Function parameters

Parameters behave like variables that are initialized by the caller, and you can assign a new value to a parameter name
inside the function. The *contents* of a collection parameter, however, are read-only (no `append`, no element
assignment) unless the parameter is declared `mut`. See
[Functions](/docs/guide/functions/#mut-parameters) and [Memory and ownership](/docs/guide/memory/).
