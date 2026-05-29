# `trux` Early Specification

## Goal

`trux` is a small Go-inspired language that compiles to C.

`trux` is not Go-compatible. It only borrows simple syntax ideas from Go.

The compiler is written in Go.

The first backend is C.

---

# Compiler Pipeline

`trux` uses this pipeline:

```text
source code
  → lexer
  → parser
  → AST
  → type checker
  → typed IR
  → C code
  → cc (or $CC)
  → executable
```

For v0, typed IR can be simple.

A typed IR means the compiler has already checked what every expression means and what type it has before generating C.

---

# v0: Minimal Language

## Goal

v0 proves the full compiler pipeline.

It supports only enough syntax to compile simple integer programs.

## Supported Features

```text
package main
func
return
let
int
function calls
integer literals
+ - * /
print
```

## Example

```go
package main

func add(a int, b int) int {
    return a + b
}

func main() int {
    let x int = add(1, 2)
    print(x)
    return 0
}
```

Expected output:

```text
3
```

## Types

v0 supports one type:

```text
int
```

`int` maps to C as:

```c
int64_t
```

## Functions

Functions must declare parameter types and return type.

```go
func add(a int, b int) int {
    return a + b
}
```

## Variables

Variables use `let`.

```go
let x int = 10
```

All variables require explicit types.

## Expressions

Supported expressions:

```text
integer literals
variables
function calls
parenthesized expressions
binary arithmetic
```

Supported operators:

```text
+ - * /
```

All operators work only on `int`.

## Print

`print(expr)` prints an integer.

```go
print(x)
```

## Entry Point

Every program must have:

```go
func main() int
```

The compiler generates a real C `main` function that calls `trux` `main`.

## v0 Compiler Checks

The compiler must reject:

```text
syntax errors
undefined variables
undefined functions
duplicate function names
duplicate local variables
wrong number of function arguments
missing main function
invalid return type
invalid expression type
```

---

# v1: Strings, Booleans, Typed IR

## Goal

v1 adds basic strings, booleans, and a real typed IR.

The compiler should no longer treat typed IR as just a checked AST. It should build a separate typed representation that the C backend consumes.

## New Supported Features

```text
string
bool
true
false
string literals
boolean literals
print(string)
print(bool)
typed IR
```

## Example

```go
package main

func main() int {
    let name string = "Kern"
    let ready bool = true

    print(name)
    print(ready)

    return 0
}
```

Expected output:

```text
Kern
true
```

## Types

v1 supports:

```text
int
string
bool
```

## String Type

A `trux` string is not a raw C `char*`.

It should be represented as:

```c
typedef struct {
    const uint8_t* data;
    size_t len;
} rt_string;
```

## Boolean Type

A `trux` bool maps to C as:

```c
bool
```

Generated C should include:

```c
#include <stdbool.h>
```

## Print

`print` supports:

```text
print(int)
print(string)
print(bool)
```

The typed IR decides which runtime function to call:

```text
print(int)    → rt_print_int
print(string) → rt_print_string
print(bool)   → rt_print_bool
```

## Runtime Functions

The runtime should provide:

```c
void rt_print_int(int64_t value);
void rt_print_string(rt_string value);
void rt_print_bool(bool value);
```

## Typed IR

v1 should introduce a real typed IR.

The typed IR must store:

```text
function names
function parameters
function return types
variable types
expression types
resolved function calls
resolved print calls
```

Example source:

```go
let name string = "Kern"
print(name)
```

Typed IR should know:

```text
name has type string
print(name) calls print(string)
```

## v1 Compiler Checks

The compiler must reject:

```text
assigning string to int
assigning int to string
assigning bool to int
using string with + - * /
using bool with + - * /
returning the wrong type
calling functions with wrong argument types
```

---

# v2: Control Flow

## Goal

v2 adds basic control flow.

## New Supported Features

```text
if
else
while
comparison operators
boolean conditions
variable assignment
```

## Example

```go
package main

func main() int {
    let x int = 10

    if x > 5 {
        print("big")
    } else {
        print("small")
    }

    return 0
}
```

Expected output:

```text
big
```

## While Example

```go
package main

func main() int {
    let i int = 0

    while i < 3 {
        print(i)
        i = i + 1
    }

    return 0
}
```

Expected output:

```text
0
1
2
```

## Comparison Operators

v2 supports:

```text
== != < <= > >=
```

Comparisons return:

```text
bool
```

Example:

```go
let result bool = 1 < 2
```

## If / Else

`if` requires a `bool` condition.

```go
if condition {
    ...
}
```

`else` is optional.

```go
if condition {
    ...
} else {
    ...
}
```

Invalid:

```go
if 123 {
    print("bad")
}
```

The compiler must reject this because `123` is an `int`, not a `bool`.

## While

`while` requires a `bool` condition.

```go
while i < 10 {
    i = i + 1
}
```

## Assignment

v2 adds assignment to existing variables.

```go
i = i + 1
```

The assigned value must match the variable type.

Invalid:

```go
let x int = 1
x = "hello"
```

## v2 Compiler Checks

The compiler must reject:

```text
if condition that is not bool
while condition that is not bool
assignment to undefined variable
assignment with wrong type
comparison between incompatible types
```

---

# Build Commands

Minimum command:

```bash
trux run main.tx
```

Optional commands:

```bash
trux build -o main main.tx
trux emit-c main.tx
```

---

# Summary

## v0

```text
minimal int language
functions
let
return
print(int)
C backend
```

## v1

```text
string
bool
print(string)
print(bool)
real typed IR
```

## v2

```text
if
else
while
comparisons
assignment
```

---

# Memory Model

Dynamic allocation (starting with strings that are not literals) uses arenas rather than individual `malloc`/`free` or a garbage collector.

See [arenas.md](arenas.md) for the full rationale, tradeoffs, staged evolution plan, and why GC was deferred.

This is a deliberate design choice aligned with the project's goals of a small compiler, readable generated C, and learning compiler construction and systems language tradeoffs.

---

# Future Direction

Long-term goal: support packages, imports, public functions, and GPU kernels.

These are not near-term priorities and will not drive early language design.

## Public Functions

```trux
package math

pub func add(a int, b int) int {
    return a + b
}

func double(x int) int {
    return x * 2
}
````

`pub func` makes a function visible to other packages.

Plain `func` stays private to the current package.

## GPU Kernels

```trux
kernel func fill(out gpu.Buffer[int], n int) {
    let i int = gpu.global_id()

    if i < n {
        out[i] = 42
    }
}
```

Kernel functions will be restricted and compiled through a separate GPU backend.
