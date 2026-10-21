# Control flow

## `if`, `else if`, `else`

```trux
--8<-- "examples/else_if.tx"
```

```text
--8<-- "website/outputs/else_if.txt"
```

- The condition must be a `bool`. There are no parentheses around it, and the braces are required.
- Chain conditions with `else if`. Writing `} if` on the same line is rejected with a hint to use `else if`.

## `for` loops

`for` is the only loop keyword. It has three forms:

=== "Condition"

    Runs while the condition is `true`, like `while` in other languages.

    ```trux
    let i int = 0
    for i < 3 {
      print(i)
      i = i + 1
    }
    ```

=== "Three-part"

    An init statement, a condition, and a post statement, separated by semicolons. A variable declared in the init
    statement is scoped to the loop.

    ```trux
    for let i int = 0; i < 5; i = i + 1 {
      print(i)
    }
    ```

    Any of the three parts may be left empty, for example `for ; i < 5; { ... }`. The post statement cannot be a
    `let`.

=== "Infinite"

    Loops until `break` or `return`.

    ```trux
    for {
      if done() {
        break
      }
    }
    ```

All three forms appear in `examples/loops.tx`:

```trux
--8<-- "examples/loops.tx"
```

```text
--8<-- "website/outputs/loops.txt"
```

## `break` and `continue`

`break` leaves the innermost loop. `continue` skips to the next iteration; in a three-part loop the post statement still
runs. Both are compile errors outside a loop.

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

## Returning from loops

`return` can appear anywhere, including inside loops. Remember that a function must return on every path, so a loop that
might finish without returning needs a `return` after it:

```trux
--8<-- "examples/return_paths.tx"
```

## Assertions

`assert(condition, message)` stops the program with exit status `1` if `condition` is `false`:

```trux
assert(count > 0, "count must be positive")
```

```text
trux runtime error: assertion failed: count must be positive
  at main.tx:9:3
```

`assert` is a statement and is the building block of [unit tests](testing.md).
