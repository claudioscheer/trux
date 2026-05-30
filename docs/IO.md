# IO Packages

Trux IO is exposed through imported standard packages. Stdin and text-file operations live in `io`; CSV operations live in `csv`.

## Standard Packages

Input and files:

```trux
import "io"

io.readLine() string
io.readInt() int
io.readFloat() float
io.readBool() bool
io.readFile(path string) string
io.writeFile(path string, contents string)
```

CSV:

```trux
import "csv"

csv.read(path string, columns int) list[string]
csv.write(path string, cells list[string], columns int)
```

CSV data is flat and row-major. `columns` defines the row width, so row count is `len(cells) / columns`.

## Design

The source-language boundary is a real import boundary, not a global alias. A file must import `io` before calling `io.readLine` or `io.readFile`, and must import `csv` before calling `csv.read` or `csv.write`.

The implementation is still privileged behind that facade. The type checker records `io` and `csv` calls as runtime-backed operations, and the IR/codegen path lowers them to the existing C runtime helpers. This avoids pretending the current language can implement file handles, stdin parsing, or CSV parsing as ordinary Trux functions.

The type checker still rejects parameters on `main`, and generated C still emits `int main(void)`. User input comes from runtime helpers that read `stdin`; file and CSV helpers use runtime C functions.

Dynamic IO results allocate into the caller's current arena. `io.readLine`, `io.readFile`, and `csv.read` are frame-owned in local expression context and are copied out through the existing return-copy rules when returned from a function.

Write operations are statement-only, like `append`. They fatal on runtime errors, matching the current runtime style for allocation, bounds, and output failures.

The old global names such as `readLine`, `readFile`, `readCsv`, and `writeCsv` are not part of the language surface. Use `io.readLine`, `io.readFile`, `csv.read`, and `csv.write`.

## Tradeoff

Keeping the runtime-backed lowering is simpler than building ordinary stdlib source packages today. The cost is that `io` and `csv` are privileged packages rather than reusable `.tx` modules. That is acceptable because it keeps side-effectful capabilities explicit in source while preserving the existing runtime implementation.

Returning CSV as `list[string]` is intentionally simple. It avoids adding structs, tuples, nested collections, or a table type in the same change. The cost is that callers must manage row shape through `columns` and index math. That is acceptable for the first CSV surface because it works with the language's current type system.
