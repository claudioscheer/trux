# Functions

## Declaring functions

```trux
func add(a int, b int) int {
  return a + b
}
```

- Every parameter has a type, written after its name. There is no shorthand like `a, b int`.
- The return type comes after the parameter list and is **required** for ordinary functions. A function returns exactly
  one value.
- Every path through the body must end in `return`. The compiler rejects a function whose last statement is an `if`
  without an `else`, or a loop, if the code could fall off the end:

    ```text
    function "f" must return on all paths
    ```

    The usual fix is a final `return` after the loop or `if`.

- `return` always takes a value. Functions that exist only for their side effects conventionally return `0`, as in
  `examples/fibonacci.tx`.

## Calling functions

```trux
let x int = add(1, 2)
print(add(add(1, 2), 3))
```

Arguments must match the parameter types exactly. Functions can be declared in any order in a file, and recursion works:

```trux
--8<-- "examples/fibonacci.tx"
```

A call can be used as a statement when you don't need its result (`printIterative(count)` above).

## Returning collections

Functions can take and return arrays, slices, and lists:

```trux
--8<-- "examples/matrix_multiply_2x2.tx"
```

Whether a returned collection is a cheap view or a copy is decided by the compiler's ownership rules. See
[Memory and ownership](memory.md).

## `mut` parameters

By default a function can **read** the collections you pass it but cannot change them. Writing `xs[0] = 1` or
`append(xs, 1)` on an ordinary collection parameter is a compile error:

```text
cannot mutate parameter-owned collection
```

Mark the parameter `mut` to allow the function to change the caller's array, slice, or list:

```trux
func push(mut xs list[int], value int) int {
  append(xs, value)
  return len(xs)
}

func main() int {
  let items list[int] = list[int]{1}
  print(push(items, 2), " ", len(items))   // 2 2
  return 0
}
```

Rules:

- `mut` is only allowed on array, slice, and list parameters, and not on kernel parameters.
- The call site does **not** repeat `mut`: you write `push(items, 2)`.
- The argument must be something the caller is allowed to mutate: a local collection, or the caller's own `mut`
  parameter. Passing a borrowed collection (for example the caller's non-`mut` parameter) is rejected.
- Assigning a whole new collection to a `mut` parameter name only rebinds the local name; it does not replace the
  caller's variable.

The complete walkthrough is in `examples/mut_parameters.tx`:

```trux
--8<-- "examples/mut_parameters.tx"
```

```text
--8<-- "website/outputs/mut_parameters.txt"
```

## Public functions

In multi-file programs, `pub func` makes a function callable from files that import it, as `package.name(...)`. See
[Modules and packages](modules.md).

## Kernel functions

`kernel func` declares a function that runs on the GPU. Kernels have no return type. See [GPU programming](gpu.md).
