---
title: "Language guide"
description: "A tour of the language, one small topic at a time."
icon: "menu_book"
weight: 200
cards: false
---

This guide walks through Trux one feature at a time. Each page is short and has examples you can paste into a `.tx` file
and run with `trux run`.

If you know Go, C, or any curly-brace language, most of Trux will look familiar. The main differences:

- **Types are always explicit.** `let count int = 0`, never `count := 0`.
- **No implicit conversions.** `1 + 2.0` is a compile error, and there is no cast between `int` and `float`.
- **Statements are separated by newlines**, not semicolons. Semicolons appear only in the three-part `for` header.
- **Collections have ownership rules.** Functions can read the collections you pass them, but only change them when the
  parameter is marked `mut`.

| Page | What you'll learn |
| --- | --- |
| [Program structure](/docs/guide/program-structure/) | `package`, `import`, `main`, comments, and identifiers |
| [Types and values](/docs/guide/types/) | `int`, `float`, `string`, `bool`, literals, and collection types |
| [Variables](/docs/guide/variables/) | `let`, assignment, and scope |
| [Operators and expressions](/docs/guide/operators/) | arithmetic, comparison, logic, `in`, and precedence |
| [Functions](/docs/guide/functions/) | parameters, return values, recursion, and `mut` parameters |
| [Control flow](/docs/guide/control-flow/) | `if` / `else if` / `else`, the three `for` forms, `break`, `continue` |
| [Arrays, slices, and lists](/docs/guide/collections/) | `[N]T`, `[]T`, `list[T]`, `make`, `len`, `append`, slicing |
| [Memory and ownership](/docs/guide/memory/) | arenas, borrowed vs owned values, and `clone` |
| [Modules and packages](/docs/guide/modules/) | multi-file programs, `pub func`, and package-qualified calls |
| [Testing](/docs/guide/testing/) | `.test.tx` files, `assert`, and `trux test` |
| [GPU programming](/docs/guide/gpu/) | `kernel func`, device buffers, and launches |

The [standard library](/docs/stdlib/) section covers the built-in `io`, `csv`, `image`, `time`, `http`, and `gpu`
packages. For a single dense reference, see the [language specification](/docs/design-notes/specs/).
