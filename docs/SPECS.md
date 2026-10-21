# `trux` Language Specification

## Goal

`trux` is a small Go-inspired language that compiles to C.

`trux` is not Go-compatible. It borrows simple syntax ideas from Go while keeping the compiler small enough to understand end to end.

The compiler is written in Go. The current backend emits C and invokes `cc` or `$CC` for normal programs. Programs that declare GPU kernels are emitted as CUDA source and invoke `nvcc` or `$NVCC`.

Source-level identifiers use lower camelCase. See [Naming](NAMING.md) for the builtin naming pattern and the distinction between source names and internal implementation names.

## Compiler Pipeline

`trux` uses this pipeline:

```text
source code
  -> lexer
  -> parser
  -> AST
  -> type checker
  -> typed IR
  -> C or CUDA code
  -> cc/$CC or nvcc/$NVCC
  -> executable
```

The typed IR is a separate checked representation consumed by the C backend. It stores resolved function calls, resolved builtin calls, variable types, expression types, function signatures, and ownership metadata for dynamic values.

## Program Shape

Every source file starts with a package declaration:

```trux
package main
```

The compiler accepts an entry file plus transitive relative `.tx` imports. A runnable program must define:

```trux
func main() int
```

The generated C program contains a real `main` function that creates the runtime context, calls the Trux `main`, and returns its integer exit code.

Importing a source file also loads same-directory sibling `.tx` files that declare the same package. The entry file does not auto-load siblings. Functions without `pub` are package-private inside that same-directory package group and are called unqualified. `pub func` is required for external package-qualified calls.

## Types

Supported scalar types:

```text
int
float
string
bool
error
```

Supported multiple return types:

```text
(T1, T2, ...)
```

Functions can return multiple values using parenthesized return types, e.g. `(int, error)` or `(string, int, bool)`. The `error` type represents an explicit error value; `nil` represents the absence of an error.

Supported collection types:

```text
[N]T
[]T
list[T]
```

Collection element types may be scalars or further collections. Nested lists, slices, and arrays are supported. `examples/jagged_lists.tx` and `examples/nested_slices.tx` cover that shape. `gpu.Buffer` elements stay `int` or `float`.

Supported GPU device buffer types:

```text
gpu.Buffer[int]
gpu.Buffer[float]
```

GPU buffers require `import "gpu"` for operations. They are device memory handles, not host collections.

Current C mappings:

```text
int            -> int64_t
float          -> double
bool           -> bool
string         -> rt_string
error          -> rt_error (rt_error_data*)
(T1, T2, ...)  -> trux_tuple_T1_T2
```

A `trux` string is not a raw C `char*`. It is represented as:

```c
typedef struct {
    const uint8_t* data;
    size_t len;
} rt_string;
```

## Functions and Variables

Functions must declare parameter types and return type:

```trux
func add(a int, b int) int {
  return a + b
}
```

Collection parameters are read-only unless the parameter is marked `mut`:

```trux
func push(mut xs list[int], value int) int {
  append(xs, value)
  return len(xs)
}
```

`mut` is valid only on array, slice, and list parameters in ordinary functions. Calls do not repeat `mut`; the type checker enforces that the argument has mutable ownership.

Variables use `let` and require explicit types:

```trux
let x int = 10
```

Assignment updates an existing variable and must preserve its type:

```trux
x = x + 1
```

Block-local variables are scoped to the block where they are declared.

## Expressions

Supported expressions include:

```text
integer literals
float literals
string literals
boolean literals
variables
function calls
qualified imported function calls
parenthesized expressions
binary arithmetic
comparisons
string containment
collection literals
indexing
slicing
make([]T, n)
clone(x)
len(x)
nil
errors.new(message)
errors.message(err)
io.readLine()
io.readInt()
io.readFloat()
io.readBool()
io.readFile(path)
io.writeFile(path, contents)
csv.read(path, columns)
csv.write(path, cells, columns)
image.width(path)
image.height(path)
image.readPpm(path)
image.writePpm(path, pixels, width, height)
time.nowUnixMillis()
time.monotonicNanos()
gpu.alloc(n)
gpu.from(host)
gpu.to(device, host)
gpu.launch1d(kernel, n, args)
gpu.launch2d(kernel, width, height, args)
gpu.globalX()
gpu.globalY()
gpu.globalZ()
gpu.threadX()
gpu.threadY()
gpu.threadZ()
gpu.blockX()
gpu.blockY()
gpu.blockZ()
gpu.blockDimX()
gpu.blockDimY()
gpu.blockDimZ()
```

Imported public functions must be called through the package declared by the imported file. Multiple direct imports may declare the same package name, and the shared package qualifier resolves across all uniquely named public functions in those package groups:

```trux
import "math/add.tx"
import "math/mul.tx"

let total int = math.add(1, 2)
let product int = math.mul(3, 4)
```

Arithmetic operators:

```text
+ - * /
```

Arithmetic requires matching numeric operands. The compiler does not implicitly promote `int` to `float`.

Integer `+`, `-`, `*`, and `/` trap on signed overflow. Integer `/` also traps on division by zero. Both failures are Trux runtime errors on the host. Float arithmetic follows IEEE 754. Float division by zero produces an infinity rather than a Trux runtime error.

Comparison operators:

```text
== != < <= > >=
```

Comparisons return `bool`. Numeric ordering comparisons require matching numeric operands. Equality supports matching `int`, `float`, `string`, and `bool` operands.

String containment uses `in`:

```trux
let found bool = "ux" in "trux"
```

The left operand is the needle. The right operand is the haystack. Both operands must be `string`, and the result is `bool`.

String concatenation uses `+`:

```trux
let name string = "trux" + " compiler"
```

Concatenated strings are dynamic strings allocated in compiler-managed memory. String literals point at static storage.

## Statements

Supported statements:

```text
let
assignment
indexed assignment
return
if / else if / else
for
print(...)
assert(condition, message)
append(list, value)
time.sleepMillis(ms)
gpu.copyToDevice(host, device)
gpu.copyToHost(device, host)
gpu.launch(kernel, gridX, gridY, gridZ, blockX, blockY, blockZ, args...)
gpu.launch1d(kernel, n, args...)
gpu.launch2d(kernel, width, height, args...)
gpu.from(host)
gpu.to(device, host)
gpu.sync()
gpu.free(device)
```

`if` and `for` conditions require `bool` values. `for` supports condition-only, infinite, and init/condition/post forms:

```trux
if x > 5 {
  print("big")
} else if x == 5 {
  print("exact")
} else {
  print("small")
}

for x < 10 {
  x = x + 1
}

for {
  print("forever")
}

for let i int = 0; i < 10; i = i + 1 {
  print(i)
}
```

`print` supports one or more scalar arguments:

```trux
print("count: ", 3, " ", true)
```

The typed IR decides which runtime function each argument needs:

```text
print(int)    -> rt_print_int
print(float)  -> rt_print_float
print(string) -> rt_print_string
print(bool)   -> rt_print_bool
print(error)  -> rt_print_error
```

`assert` is statement-only. It requires a `bool` condition and a `string` message. If the condition is false at runtime, the generated app writes a runtime assertion error to stderr and exits with status `1`:

```trux
assert(len(items) > 0, "items must not be empty")
```

## Error Handling and Multiple Returns

Trux uses Go-style explicit error handling and multiple return values.

### The `error` Type and `nil`

`error` is a first-class scalar type representing fallible operation results. The `nil` keyword represents the absence of an error.

`error` values can be compared to `nil` using `==` and `!=`:

```trux
if err != nil {
  print("operation failed: ", errors.message(err))
}
```

Printing an error value with `print(err)` prints the error's message string, or `<nil>` if the error is nil.

### The `errors` Package

The standard library includes `errors`:

```trux
import "errors"

errors.new(msg string) error
errors.message(err error) string
```

- `errors.new(msg)` allocates a new error value with the given message.
- `errors.message(err)` returns the error message string (or `""` if `nil`).

### Multiple Return Values

Functions can return multiple values using parenthesized return types in their signature:

```trux
func divmod(a int, b int) (int, int) {
  return a / b, a % b
}

func parse(s string) (int, error) {
  if len(s) == 0 {
    return 0, errors.new("empty string")
  }
  return 42, nil
}
```

A return statement with multiple values uses comma separation: `return val, nil`.
Forwarding a call returning matching multiple values is supported:

```trux
func forward(a int, b int) (int, int) {
  return divmod(a, b)
}
```

Multi-return forwarding into function argument lists (e.g. `foo(bar())`) is forbidden.

### Unpacking and the Blank Identifier

Multiple returns are unpacked into explicit typed `let` bindings:

```trux
let q int, r int = divmod(14, 3)
let value int, err error = parse(input)
```

The blank identifier `_` discards unwanted values:

```trux
let _, rem int = divmod(14, 3)
let val int, _ error = parse(input)
```

Multi-variable assignment updates existing variables:

```trux
q, r = divmod(20, 6)
_, r = divmod(25, 4)
```

### Statement Restrictions

Calling a function that returns multiple values or an `error` as a bare statement without assigning or discarding its results is a compile-time error. All errors must be handled, assigned, or explicitly discarded:

```trux
// Compile-time error:
// io.writeFile("out.txt", data)

// Allowed:
let err error = io.writeFile("out.txt", data)
_ = io.writeFile("out.txt", data)
```

## Unit Tests

Unit tests live in files ending with `.test.tx` and run through:

```bash
trux test
trux test path/to/file.test.tx
```

The first form recursively discovers `.test.tx` files from the current directory. The second runs one file. Test functions are no-argument functions whose names start with `test` and that omit the return type:

```trux
package math

import "math.tx"

func testDouble() {
  assert(double(4) == 8, "double should multiply by two")
}
```

Tests pass by returning normally. Failed `assert` calls, runtime errors, compile errors, or invalid test declarations fail the test run. `trux test --fail-fast` stops after the first failure. Normal program loading skips `.test.tx` files.

`append` is statement-only and mutates a list:

```trux
append(items, 2)
```

## IO

Input and file operations require `import "io"`. CSV operations require `import "csv"`. P3 PPM image operations require `import "image"`:

```trux
import "io"
import "csv"
import "image"

let line string = io.readLine()
let count int = io.readInt()
let ratio float = io.readFloat()
let ok bool = io.readBool()

let contents string, readErr error = io.readFile("input.txt")
let writeErr error = io.writeFile("copy.txt", contents)

let cells list[string], csvErr error = csv.read("input.csv", 2)
let writeCsvErr error = csv.write("copy.csv", cells, 2)

let width int = image.width("input.ppm")
let height int = image.height("input.ppm")
let pixels []int, ppmErr error = image.readPpm("input.ppm")
let writePpmErr error = image.writePpm("copy.ppm", pixels, width, height)
```

`io.readLine` reads one line from stdin without the trailing newline. `io.readInt`, `io.readFloat`, and `io.readBool` read one line from stdin and parse it as the requested scalar type. Invalid typed input is a runtime error.

`io.readFile` returns `(string, error)` with the complete file contents or a descriptive error. `io.writeFile` writes the target path with the provided contents and returns `error` (`nil` on success).

`csv.read` returns `(list[string], error)` as flat row-major cells. The `columns` argument must be positive and each CSV row must have exactly that many columns. `csv.write` writes a flat row-major `list[string]` using the same column count and returns `error` (`nil` on success).

CSV supports quoted fields and escaped quotes using doubled quotes.

`image.readPpm` returns `([]int, error)` with flat RGB channel values from a P3 PPM image with max value `255`. `image.writePpm` writes a flat RGB `[]int` as P3 PPM, requiring `len(pixels) == width * height * 3`, and returns `error` (`nil` on success).

## Time

Time operations require `import "time"`:

```trux
import "time"

let wall int = time.nowUnixMillis()
let start int = time.monotonicNanos()
time.sleepMillis(10)
let elapsed int = time.monotonicNanos() - start
```

`time.nowUnixMillis` returns wall-clock Unix time in milliseconds. `time.monotonicNanos` returns a monotonic timestamp for elapsed-time measurement. `time.sleepMillis` is statement-only and sleeps for a non-negative number of milliseconds.

## GPU

GPU operations require `import "gpu"` and a CUDA toolchain. `gpu.from` copies a host slice to a new device buffer. `gpu.to` copies it back. `gpu.alloc`, `gpu.copyToDevice`, and `gpu.copyToHost` remain when those steps should be separate.

```trux
import "gpu"

let n int = 1024
let host []int = make([]int, n)
let device gpu.Buffer[int] = gpu.from(host)

gpu.launch1d(fill, n, device)
gpu.to(device, host)
gpu.free(device)
```

Kernels are declared with `kernel func` and do not return values. `gpu.launch1d` fills the first parameter with the element index. `gpu.launch2d` fills the first parameter with the row and the second with the column. Threads outside the extents do nothing.

```trux
kernel func fill(i int, out gpu.Buffer[int]) {
  out[i] = 42
}
```

Kernels support scalar locals, arithmetic, comparisons, `if`, `for`, GPU buffer indexing, and GPU coordinate helpers. They do not support strings, lists, host slices, allocation, `print`, IO, time calls, or normal function calls.

`gpu.launch` still takes an explicit grid and block, and does not inject coordinates. `gpu.sync` waits for the device.

## Collections

Arrays use `[N]T`, where `N` is a positive integer literal:

```trux
let xs [3]int = [3]int{1, 2, 3}
```

Array literals must provide exactly `N` elements.

Slices use `[]T` and are borrowed views over arrays, slices, or lists:

```trux
let tail []int = xs[1:]
let all []int = xs[:]
```

Lists use `list[T]` and are growable shared handles:

```trux
let items list[int] = list[int]{1}
append(items, 2)
```

Assigning or passing a list copies the handle. Mutating through any alias observes the same list header and backing buffer.

`make([]T, n)` creates zero-filled arena-backed storage and returns a slice view:

```trux
let scratch []int = make([]int, 10)
```

`len(x)` returns an `int` for strings, arrays, slices, and lists.

Indexing works for strings, arrays, slices, and lists:

```trux
let first int = xs[0]
let ch string = "abc"[1]
```

String indexing returns a one-byte `string` view. Strings are immutable, so string index assignment is rejected.

Slicing works for strings, arrays, slices, and lists:

```trux
let sub []int = xs[1:3]
let prefix string = "trux"[:2]
```

Indexed assignment works for arrays, slices, and lists:

```trux
xs[0] = 42
view[1] = 7
items[0] = 9
```

Runtime bounds checks trap invalid indexes and slices with a `trux runtime error`.

## Ownership and Memory Model

Dynamic allocation uses arenas for strings, arrays, and slice backing storage. Lists are heap-backed shared handles tracked by the runtime owner and freed at program exit, arena reset, or arena rewind.

Generated code uses a durable arena plus compiler-managed function-frame arenas. Local dynamic values may use frame memory.

Function returns are either borrowed or owned:

- Borrowed returns are parameter values or parameter-backed slices/views.
- Owned returns remain valid after the callee returns because they were copied into `trux_result_arena`, already live in the durable arena, or were frame-owned and copied into `trux_result_arena` during return.

Returning a parameter or a parameter-backed slice is borrowed by default:

```trux
func mid(xs []int) []int {
  return xs[1:2]
}
```

Use `clone(x)` when an owned copy is required:

```trux
func midOwned(xs []int) []int {
  return clone(xs[1:2])
}
```

`clone(x)` supports `string`, arrays, slices, and lists. In direct return context, it allocates into the result arena. In local context, it allocates into the current function-frame arena, and returning that frame-owned value copies it into the result arena:

```trux
func frameOwnedLocal() []int {
  let xs [3]int = [3]int{1, 2, 3}
  let ys []int = clone(xs[:])
  return ys
}
```

For collections containing strings, `clone` deep-copies each string element.

Inside a function, ordinary collection parameters are borrowed. The function may read them and return values derived from them, but it may not mutate the parameter-owned collection or an alias/view of it:

```trux
func bad(xs list[int]) int {
  append(xs, 1) // rejected
  return len(xs)
}
```

Use `mut` when a function is allowed to mutate the caller-owned collection:

```trux
func good(mut xs list[int]) int {
  append(xs, 1)
  return len(xs)
}
```

Values appended through a `mut` list parameter allocate dynamic element data in the durable arena when needed, so caller-visible list elements remain valid after the callee returns.

See [ARENAS.md](ARENAS.md) for the full memory model rationale. See [examples/ownership_clone.tx](https://github.com/claudioscheer/trux/blob/main/examples/ownership_clone.tx) and [examples/mut_parameters.tx](https://github.com/claudioscheer/trux/blob/main/examples/mut_parameters.tx) for executable walkthroughs.

## Compiler Checks

The compiler rejects:

```text
syntax errors
undefined variables
undefined functions
duplicate function names in the same file
duplicate function names in the same-directory package group
duplicate local variables
wrong number of function arguments
missing main function
main functions with parameters
invalid return type
invalid expression type
assigning a value to the wrong variable type
returning the wrong type
calling functions with wrong argument types
using arithmetic with incompatible operands
using comparisons with incompatible operands
using in with non-string operands
if or for conditions that are not bool
assignment to undefined variables
array literals with the wrong element count
collection literal elements with the wrong type
index expressions whose index is not int
slice bounds that are not int
indexing or slicing non-collection values
string index assignment
append outside statement position
append with a non-list first argument
append values with the wrong element type
assert outside statement position
assert with wrong arity or non-bool/non-string arguments
mutation of non-`mut` parameter-owned collections or aliases/views of them
`mut` parameters with non-collection types
borrowed or unknown-owned arguments passed to `mut` parameters
make with a non-slice type or non-int length
clone with the wrong arity or unsupported type
qualified calls to private package functions
standard `io`, `csv`, `time`, and `gpu` package calls with the wrong arity or argument types
time.sleepMillis, and statement-only gpu calls outside statement position
calling a function returning multiple values or an error as a bare statement without assignment or discard
unsupported host-only constructs inside kernels
```

## Build Commands

Minimum command:

```bash
trux run main.tx
```

Optional commands:

```bash
trux build -o main main.tx
trux emit-c main.tx
```

`emit-c` writes `<stem>.c` or `<stem>.cu` plus the runtime headers needed by that program to `out/<stem>/` by default. Pass `--out-dir DIR` to choose the output directory.

## Future Ideas

`pub func` and package-qualified calls are implemented. See [MODULES.md](MODULES.md).

These ideas are not implemented and should not drive current language usage:

- recursive directory-based packages
- multi-file C output
