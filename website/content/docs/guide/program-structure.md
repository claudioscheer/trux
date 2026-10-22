---
title: "Program structure"
description: "Packages, imports, structs, main, comments, names, and layout rules."
icon: "account_tree"
weight: 100
---

A Trux file always has the same parts, in this order:

```trux
package main              // 1. package clause

import "time"             // 2. imports (optional)

type RunStats struct {    // 3. struct and function declarations
  elapsed int
}

func elapsedSince(start int) RunStats {
  return RunStats{elapsed: time.monotonicNanos() - start}
}

func main() int {
  let start int = time.monotonicNanos()
  print("hello")
  let stats RunStats = elapsedSince(start)
  print("took ", stats.elapsed, " ns")
  return 0
}
```

## Package clause

**What it is.** `package name` is the first line of code in every file. It names the package the file belongs to.

**Why.** The package name is the qualifier other files use to reference your exported functions and structs
(`math.add(...)`, `math.Pair{...}`), and files with the same package name in one directory form a single package.

**How to use it.** The file you run declares `package main`. Imported files declare their own name, such as
`package math`. See [Modules and packages](/docs/guide/modules/).

**When you don't need it.** Never: every file needs one.

**Related:** [Modules and packages](/docs/guide/modules/).

## Imports

**What it is.** `import "..."` makes a standard package or another source file available to this file.

**Why.** Nothing outside the file is visible by default, so each file lists what it uses.

**How to use it.** One import per line, after `package` and before the first `type` or `func` declaration:

```trux
import "io"          // a standard package: io, csv, image, time, http, gpu, errors
import "math.tx"     // another Trux file, relative to this one
```

**Common mistakes.** An import after a function or struct declaration is rejected:

{{< include "examples/errors/program_import_order.tx" "trux" >}}

{{< include "website/outputs/errors/program_import_order.txt" "text" >}}

Using a standard package without importing it is also an error; see
[`io`](/docs/stdlib/io/#errors).

**When you don't need it.** Single-file programs that only use the builtins (`print`, `len`, `append`, `make`, `new`,
`delete`, `keys`, `values`, `clone`, `assert`) need no imports.

**Related:** [Modules and packages](/docs/guide/modules/), [Standard library](/docs/stdlib/).

## The `main` function

**What it is.** `func main() int` is where a program starts. Its return value becomes the process exit code.

**Why.** Returning an `int` makes success (`0`) and failure (anything else) explicit.

**How to use it.** Declare it in the entry file (the file you pass to `trux run`), with no parameters:

{{< include "examples/hello.tx" "trux" >}}

{{< include "website/outputs/hello.txt" "text" >}}

Only the entry file may define `main`. Programs cannot read command-line arguments.

**Common mistakes.** Forgetting `main`:

{{< include "examples/errors/program_missing_main.tx" "trux" >}}

{{< include "website/outputs/errors/program_missing_main.txt" "text" >}}

Giving `main` parameters:

{{< include "website/outputs/errors/program_main_params.txt" "text" >}}

**When you don't need it.** Imported files and `.test.tx` files must not define `main`.

**Related:** [Testing](/docs/guide/testing/), [Errors and failures](/docs/guide/errors/#exit-codes).

## Comments

**What it is.** `//` starts a comment that runs to the end of the line. There are no block comments.

**Why.** One comment form keeps the lexer and the formatter simple.

**How to use it.** `// explain why, not what`

**When you don't need it.** When a well-named function or variable says the same thing.

## Names

**What it is.** Identifiers start with an ASCII letter or `_` and continue with letters, digits, or `_`.

**Why.** Short, predictable names map directly onto C identifiers in the generated code.

**How to use it.** By convention, functions, variables, struct fields, and standard package members use lower camelCase
(`io.readLine`, `fibRecursive`), struct types use UpperCamelCase (`RowSummary`, `Pair`), and package names are short and
lowercase. Names that start with `__trux_` are reserved for the compiler. These words are reserved:

```text
package  import  pub  kernel  func  type  struct  return  let  mut
if  else  for  break  continue  in
int  float  string  bool  error  list  map  true  false  nil
```

**Related:** [Naming](/docs/design-notes/naming/).

## Statements and layout

**What it is.** A newline ends a statement. Blocks always use braces, and `if` and `for` conditions have no
parentheses.

**Why.** Fewer optional forms means fewer ways to write the same thing, and `trux fmt` can format any file.

**How to use it.** Write one statement per line. Semicolons appear only inside a three-part `for` header. Run
`trux fmt` to apply the standard two-space layout.

**Common mistakes.** Starting an `if` on the line where the previous block closes is rejected; see
[Control flow](/docs/guide/control-flow/#if-else-if-else).

**Related:** [Control flow](/docs/guide/control-flow/), [`trux fmt`](/docs/cli/#trux-fmt).
