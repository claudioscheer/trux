---
title: "Control flow"
description: "if and else, the three kinds of for loop, break, continue, and assert."
icon: "alt_route"
weight: 600
---

## `if`, `else if`, `else` {#if-else-if-else}

**What it is.** Runs a block when a `bool` condition is `true`, with optional `else if` and `else` branches.

**Why.** It is the only branching statement; there is no `switch` or ternary operator.

**How to use it.** No parentheses around the condition, braces always required:

{{< include "examples/else_if.tx" "trux" >}}

{{< include "website/outputs/else_if.txt" "text" >}}

**Common mistakes.** Writing `} if` on one line, which usually means a forgotten `else`:

{{< include "examples/errors/control_if_same_line.tx" "trux" >}}

{{< include "website/outputs/errors/control_if_same_line.txt" "text" >}}

A non-`bool` condition is also rejected; see [No implicit conversions](/docs/guide/types/#no-implicit-conversions).

**Related:** [Comparison](/docs/guide/operators/#comparison), [Logic](/docs/guide/operators/#logic).

## `for` loops

**What it is.** `for` is the only loop keyword. It has three forms: condition only (like `while`), three-part
(`init; condition; post`), and infinite.

**Why.** One keyword covers every loop shape, as in Go.

**How to use it.**

{{< include "examples/loops.tx" "trux" >}}

{{< include "website/outputs/loops.txt" "text" >}}

- A variable declared in the three-part header belongs to the loop.
- Any of the three parts may be empty: `for ; i < 5; { ... }`.
- The post statement is an assignment such as `i = i + 1`; it cannot be a `let`.

**When you don't need it.** To process every element of a collection, a three-part loop over `len(xs)` is the only way;
there is no `range`.

**Related:** [Scope](/docs/guide/variables/#scope), [Arrays, slices, and lists](/docs/guide/collections/).

## `break` and `continue`

**What it is.** `break` leaves the innermost loop. `continue` skips to its next iteration; in a three-part loop the post
statement still runs.

**Why.** They stop or skip work without extra flag variables.

**How to use it.**

```trux
package main

func main() int {
  let total int = 0
  for let i int = 0; i < 20; i = i + 1 {
    if i % 2 == 0 {
      continue
    }
    if i > 9 {
      break
    }
    total = total + i
  }
  print(total)
  return 0
}
```

```text
25
```

**Common mistakes.** Using them outside a loop:

{{< include "examples/errors/control_break_outside.tx" "trux" >}}

{{< include "website/outputs/errors/control_break_outside.txt" "text" >}}

To leave a function early, use `return`.

**Related:** [Returning on every path](/docs/guide/functions/#returning-on-every-path).

## `assert`

**What it is.** `assert(condition, message)` stops the program with exit status `1` and prints `message` when
`condition` is `false`.

**Why.** It turns an assumption into a check that fails loudly at the line where it is broken. It is also how
[tests](/docs/guide/testing/) fail.

**How to use it.** `assert` is a statement:

{{< include "examples/assertions.tx" "trux" >}}

{{< include "website/outputs/assertions.txt" "text" >}}

**Common mistakes.** A false condition stops the program, with the file, line, and column:

{{< include "examples/errors/control_assert.tx" "trux" >}}

{{< include "website/outputs/errors/control_assert.txt" "text" >}}

**When you don't need it.** Don't assert what the runtime already checks: indexes, integer overflow, and division by
zero fail on their own. `assert` is not for expected conditions such as bad user input; there is no way to recover from
it. Check the input with `if` and handle it instead.

**Related:** [Testing](/docs/guide/testing/), [Errors and failures](/docs/guide/errors/).
