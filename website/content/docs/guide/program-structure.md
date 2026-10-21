---
title: "Program structure"
description: "Packages, imports, functions, statements, and comments."
icon: "account_tree"
weight: 100
---

## A complete program

```trux
package main

import "time"

// Comments start with two slashes and run to the end of the line.
func elapsedSince(start int) int {
  return time.monotonicNanos() - start
}

func main() int {
  let start int = time.monotonicNanos()
  print("hello")
  print("took ", elapsedSince(start), " ns")
  return 0
}
```

A Trux source file has three parts, always in this order:

1. **A package clause**: `package main`. Every file has exactly one, as its first line of code.
2. **Imports** (optional): one `import "..."` per line. Imports must come after `package` and before any function.
3. **Functions**: `func`, `pub func`, or `kernel func` declarations. Functions are the only top-level declarations.
   There are no global variables, constants, or type declarations.

## The `main` function

A runnable program needs `func main() int` in its entry file (the file you pass to `trux run`):

- it takes no parameters (command-line arguments are not available)
- it must return `int`; that value becomes the process exit code
- only the entry file may define `main`

## Comments

Only line comments exist: `//` to the end of the line. There are no block comments.

## Identifiers and naming

Identifiers start with a letter or `_` and continue with letters, digits, or `_`. They are ASCII only.

By convention Trux code uses **lower camelCase** for functions, variables, and standard package members
(`io.readLine`, `time.monotonicNanos`, `fibRecursive`). Package names are short and lowercase. See
[Naming](/docs/design-notes/naming/) for the reasoning.

Names that start with `__trux_` are reserved for the compiler.

Reserved words:

```text
package  import  pub  kernel  func  return  let  mut
if  else  for  break  continue  in
int  float  string  bool  list  true  false
```

## Statements and layout

Statements end at the end of a construct, so you don't write semicolons. Semicolons appear only in a
three-part `for` header (`for let i int = 0; i < n; i = i + 1 { ... }`).

Blocks always use braces, and `if`/`for` conditions are written without parentheses. One layout rule is enforced: an
`if` cannot start on the same line as the closing brace of a previous `if` or `for` block. Use `else if` to chain, or put
the next `if` on its own line.

`trux fmt` formats files with two-space indentation.

## Imports at a glance

```trux
import "io"          // a standard package: io, csv, image, time, http, gpu
import "math.tx"     // another Trux source file, relative to this file
```

See [Modules and packages](/docs/guide/modules/) and the [standard library](/docs/stdlib/).
