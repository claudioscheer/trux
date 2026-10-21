---
title: "Operators and expressions"
description: "Arithmetic, comparison, and logical operators, and checked arithmetic."
icon: "calculate"
weight: 400
---

## Arithmetic

| Operator | `int` | `float` | `string` |
| --- | --- | --- | --- |
| `+` | addition | addition | concatenation |
| `-` | subtraction | subtraction | |
| `*` | multiplication | multiplication | |
| `/` | division (truncates toward zero) | division | |
| `%` | remainder | | |
| unary `-` | negation | negation | |

Both operands must have the **same** type. `1 + 2` and `1.0 + 2.0` are fine, `1 + 2.0` is a compile error.

### Checked integer arithmetic

Integer `+`, `-`, `*`, `/`, `%`, and unary `-` are checked at run time. Signed overflow, division by zero, and remainder by
zero stop the program:

```text
trux runtime error: integer overflow in +
  at main.tx:5
```

Float arithmetic follows IEEE 754: dividing a float by zero gives an infinity instead of an error.

## Comparison

| Operator | Works on |
| --- | --- |
| `==`, `!=` | two values of the same scalar type: `int`, `float`, `string`, or `bool` |
| `<`, `<=`, `>`, `>=` | two `int` or two `float` values |

Comparisons produce `bool`. Strings compare by content (`"ab" == "a" + "b"` is `true`). Collections cannot be compared
with `==`.

## Logic

| Operator | Meaning |
| --- | --- |
| `&&` | and (short-circuits: the right side runs only if the left is `true`) |
| `||` | or (short-circuits: the right side runs only if the left is `false`) |
| `!` | not |

All operands must be `bool`. There is no "truthiness": `if count { ... }` is rejected, write `if count > 0 { ... }`.

## String containment: `in`

```trux
let found bool = "ux" in "trux"   // true
```

The left side is the needle, the right side is the haystack. Both must be `string`. An empty needle is always found.

## Precedence

From lowest to highest binding. Operators on the same row are left-associative.

| Level | Operators |
| --- | --- |
| 1 (lowest) | `||` |
| 2 | `&&` |
| 3 | `==` `!=` `<` `<=` `>` `>=` `in` |
| 4 | `+` `-` |
| 5 | `*` `/` `%` |
| 6 (highest) | unary `-` and `!`, then calls, indexing `x[i]`, and slicing `x[a:b]` |

Use parentheses whenever it makes intent clearer: `(a + b) / 2.0`.

## Other expressions

- **Calls**: `add(1, 2)`, `math.add(1, 2)` (a function from an imported package), `io.readLine()` (a standard package).
- **Indexing and slicing**: `xs[0]`, `xs[1:3]`, `text[:2]`. See [collections](/docs/guide/collections/).
- **Collection literals**: `[3]int{1, 2, 3}`, `list[string]{"a", "b"}`.
- **Builtins**: `len(x)`, `clone(x)`, `make([]T, n)`.

## Statement-only builtins

Some operations produce no useful value and may only be used as a statement on their own line, never inside an
expression: `print(...)`, `assert(...)`, `append(...)`, `io.writeFile`, `csv.write`, `image.writePpm`,
`time.sleepMillis`, `http.serve`, `http.respond`, and the GPU copy, launch, `sync`, and `free` calls.

A call to one of your own functions can also be used as a statement; its return value is discarded:

```trux
printIterative(count)
```
