# IO Builtins

Trux IO is implemented as compiler-recognized builtins, matching the existing treatment of `print`, `len`, `clone`, and `append`.

## Builtins

Input:

```trux
read_line() string
read_int() int
read_float() float
read_bool() bool
```

Files:

```trux
read_file(path string) string
write_file(path string, contents string)
```

CSV:

```trux
read_csv(path string, columns int) list[string]
write_csv(path string, cells list[string], columns int)
```

CSV data is flat and row-major. `columns` defines the row width, so row count is `len(cells) / columns`.

## Design

The right implementation is builtin calls, not `main(args)`. The type checker still rejects parameters on `main`, and generated C still emits `int main(void)`. User input comes from runtime helpers that read `stdin`; file and CSV helpers use runtime C functions.

Dynamic IO results allocate into the caller's current arena. `read_line`, `read_file`, and `read_csv` are frame-owned in local expression context and are copied out through the existing return-copy rules when returned from a function.

Write operations are statement-only, like `append`. They fatal on runtime errors, matching the current runtime style for allocation, bounds, and output failures.

## Tradeoff

Returning CSV as `list[string]` is intentionally simple. It avoids adding structs, tuples, nested collections, or a table type in the same change. The cost is that callers must manage row shape through `columns` and index math. That is acceptable for the first IO surface because it works with the language's current type system.
