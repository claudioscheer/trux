---
title: "Language guide"
description: "Every language feature, one page per topic, each with examples and the real errors it produces."
icon: "menu_book"
weight: 200
cards: false
---

This guide covers every feature of the language. If you know Go or C, most of Trux will look familiar. The differences
that matter most:

- **Types are always written.** `let count int = 0`, never `count := 0`.
- **No implicit conversions.** `1 + 2.0` is a compile error, and there is no cast between `int` and `float`.
- **Newlines end statements.** Semicolons appear only in the three-part `for` header.
- **Functions cannot change the collections you pass them** unless the parameter is marked `mut` (or you pass a pointer `*T`).
- **Mistakes stop the program.** Overflow, division by zero, a bad index, a nil pointer dereference, or a failed `assert`
  ends the run with a `trux runtime error`.

## How each feature is described

Every feature section uses the same parts, so you can skip to the one you need:

| Part | Answers |
| --- | --- |
| **What it is** | the feature in one or two sentences |
| **Why** | the problem it solves, or why Trux does it this way |
| **How to use it** | the syntax and a small program you can run |
| **Common mistakes** | the error you get when it goes wrong, copied from a real run of the compiler |
| **When you don't need it** | when to reach for something simpler, or leave it out |
| **Related** | links to the features it works with |

The failing programs live in [`examples/errors/`](https://github.com/thea1lab/trux/tree/main/examples/errors), and
their output is recorded by `website/update-example-outputs.sh`. CI re-runs them, so a message shown here is the message
the current compiler prints. GPU behavior that needs a CUDA device is explained from the generated CUDA code and is
marked **not executed**.

## Pages

| Page | Features |
| --- | --- |
| [Program structure](/docs/guide/program-structure/) | `package`, `import`, `type ... struct`, `main`, comments, names |
| [Types and values](/docs/guide/types/) | `int`, `float`, `string`, `bool`, `error`, structs, pointers `*T`, `new(T)`, literals, no implicit conversions |
| [Variables](/docs/guide/variables/) | `let`, assignment, field/pointer/map assignment, comma-ok lookup, scope |
| [Operators and expressions](/docs/guide/operators/) | arithmetic, checked integer math, `&x`, `*p`, comparison, logic, `in` |
| [Functions](/docs/guide/functions/) | parameters, returns, recursion, `mut` parameters, pointer parameters |
| [Control flow](/docs/guide/control-flow/) | `if`, the three `for` forms, `break`, `continue`, `assert` |
| [Arrays, slices, lists, and maps](/docs/guide/collections/) | `[N]T`, `[]T`, `list[T]`, `map[K]V`, `make`, `append`, `delete`, `keys`, `values`, bounds checks |
| [Memory and ownership](/docs/guide/memory/) | arenas, borrowing, `new(T)`, `clone` |
| [Modules and packages](/docs/guide/modules/) | multi-file programs, `pub func`, `pub type ... struct`, imports |
| [Testing](/docs/guide/testing/) | `.test.tx` files and `trux test` |
| [Errors and failures](/docs/guide/errors/) | compile errors, runtime errors, exit codes |
| [GPU programming](/docs/guide/gpu/) | `kernel func`, device buffers, launches, `gpu.sync` |

The [standard library](/docs/stdlib/) covers the built-in packages `io`, `csv`, `image`, `time`, `http`, `gpu`, and `errors`. For
one dense reference, see the [language specification](/docs/design-notes/specs/).
