# Trux language audit

Date: 2026-10-20.

This is the working list of what the language can do, what is still missing, and which simplifications are worth the change. `TODO.md` is an older review transcript. `TODO.checklist.json` tracks that review. Several items in both files are already in the compiler. Use this document for the next round of work.

Complexity means how hard the change is in this codebase:

- **Low.** One area. The behavior is already decided. A focused change and tests.
- **Medium.** Two or three compiler phases, or a semantic choice that can break programs.
- **High.** The type model, the C ABI, or ownership. Easy to get wrong.

## What exists

Trux is a small Go-inspired language. It is not Go. The compiler is Go. Ordinary programs become one C file and link through `cc` or `$CC` at `-O2`. Programs that declare a kernel become CUDA and link through `nvcc` or `$NVCC`.

```text
source
  -> lexer
  -> parser
  -> module loader
  -> type checker
  -> typed IR
  -> C or CUDA
  -> cc or nvcc
```

A program is `package`, then imports, then functions. `func main() int` is the entry point. `trux run`, `trux build`, `trux emit-c`, `trux fmt`, and `trux test` are the commands.

### Types and values

Scalars are `int` (`int64_t`), `float` (`double`), `bool`, and `string` (`rt_string`, a pointer plus a length). Strings are bytes. Indexing a string returns a one-byte string view. Literals support `\"`, `\\`, `\n`, and `\t`. A newline inside a quote is an error. Comments are `//` line comments.

Collections are `[N]T`, `[]T`, and `list[T]`. Element types may themselves be collections. `gpu.Buffer[int]` and `gpu.Buffer[float]` are device handles. There is no `nil`, no pointer type, no map, no struct, and no user-defined type.

`let` requires an explicit type. Assignment updates an existing variable. `mut` is allowed only on collection parameters. Calls do not repeat `mut`.

### Expressions and statements

Expressions are literals, names, calls, `package.function(...)`, parentheses, `+ - * /`, comparisons, `needle in haystack`, indexing, slicing, collection literals, `make([]T, n)`, `clone`, and `len`.

Statements are `let`, assignment, indexed assignment, `return`, `if` / `else if` / `else`, `for`, `print`, `assert`, `append`, and the statement-only standard calls (`io.writeFile`, `csv.write`, `image.writePpm`, `time.sleepMillis`, `gpu.copyToDevice`, `gpu.copyToHost`, `gpu.to`, `gpu.launch`, `gpu.launch1d`, `gpu.launch2d`, `gpu.sync`, `gpu.free`, `http.serve`, `http.respond`).

`for` has three forms: `for {`, `for condition {`, and `for init; condition; post {`. The init may be `let` or an assignment. The post may not be `let`.

Integer `+`, `-`, `*`, and `/` go through checked runtime helpers. Division by zero and signed overflow fail as Trux runtime errors on the host. On device they trap. Float arithmetic is ordinary C `double` arithmetic.

### Memory

Dynamic strings, arrays, and slice storage live in arenas. Lists are heap handles registered with the current arena and freed on reset, rewind, or destroy. Each function gets a frame arena. A returned frame value is copied into the caller result arena. A returned parameter, or a slice of a parameter, is borrowed. `clone` makes an owned copy and deep-copies string elements. The ownership checks held up under the probes recorded in `TODO.md`.

### Modules, tests, and standard library

`import "relative/file.tx"` loads that file and same-directory siblings with the same package name. `pub func` is the export. Other functions are package-private inside that directory group. The loader detects cycles, rejects `.test.tx` imports, and merges everything into one program before type checking. Output is still one `.c` or `.cu` file.

Bare imports are `io`, `csv`, `image`, `time`, `http`, and `gpu`. Image IO is P3 PPM only. HTTP is plaintext POSIX sockets with a pthread worker pool. HTTP and GPU cannot be combined in one program. Kernels may use scalar locals, arithmetic, comparisons, `if`, `for`, buffer indexing, and the `gpu.global*` / `thread*` / `block*` helpers. Kernels cannot allocate, print, or call ordinary functions.

`.test.tx` files run under `trux test`. A test function starts with `test`, takes no parameters, omits the return type, and must not contain `return`. The test driver inserts `return 0` and a synthetic `main` before type checking.

Editor support is `trux fmt`, a language server (diagnostics, format, completion, hover, definition, references), and a VS Code extension with TextMate highlighting.

## Already in the compiler

These were open in the older review. The code and tests now cover them. Do not schedule them again.

- Non-void functions must return on every path. `blockTerminates` treats `return` as terminal and `if` as terminal only when both branches terminate and an `else` exists. Codegen still emits `rt_runtime_fail("missing return")` before the return label.
- Function names mangle as `trux_f_<len>_<name>`. Kernels mangle as `trux_k_<len>_<name>`.
- Host integer arithmetic uses `rt_add_i64`, `rt_sub_i64`, `rt_mul_i64`, and `rt_div_i64`. The helpers reject overflow and division by zero.
- Calls and other side-effecting operands are emitted left to right through statement preludes (`trux_tmp_N`).
- Normal builds pass `-O2`. Debug builds stay at `-O0`.

## Docs that disagree with the code

Fix these before using the docs as a spec. Each one is **Low**.

| Doc | What it says | What the code does |
| --- | --- | --- |
| `docs/SPECS.md` types | Nested collections are rejected | `list[list[int]]`, `[][]T`, and `[N][M]T` type-check and generate code. `examples/jagged_lists.tx` and `examples/nested_slices.tx` use them |
| `docs/SPECS.md` future ideas | `pub func` and packages are unimplemented | `pub`, package-qualified calls, and the loader are implemented |
| `docs/MODULES.md` loader rules | Bare imports are only `io` and `csv` | `image`, `time`, `http`, and `gpu` are bare imports too |
| `README.md` pipeline | The pipeline ends at C and `cc` | Kernel programs emit CUDA and call `nvcc` |
| `TODO.md` | Missing returns, name collisions, raw `SIGFPE`, and right-to-left calls are open bugs | Those four are fixed. The performance and codegen-cleanup notes in that file are still open |

`docs/OPEN_ISSUES.md` only records empty frame-arena elision. That item is still open. It is listed below.

## Parser

`internal/parser/parser.go` is about 820 lines. It is a recursive-descent parser with a Pratt loop for binary operators. It is the simple part of the compiler. The expensive duplication is in the type checker and the C backend, described in the next section.

Keep these parser choices:

- `make([]T, n)` is parsed specially because the first argument is a type. `len`, `clone`, `append`, `print`, and `assert` stay ordinary calls and are resolved later.
- `return` always takes an expression. Every non-kernel function has a return type. Test functions gain `return 0` after parsing.
- `else if` is a nested `if` stored in the else block. The type checker, IR, and backend already walk that shape. Replacing it with an else-if list would touch every later phase for a small gain. Leave it.
- `gpu.Buffer` is the only qualified type, and `parseType` matches it by name. Generalize that when a second qualified type exists.
- The same-line `if` check rejects `} if` on one line and tells the user to write `else if` or a newline. It is tested. Leave it.
- The parser reports the first error and stops. That is the right batch-compiler behavior.

Three small parser cleanups are worth doing when that file is next edited:

| ID | Change | Complexity | Why |
| --- | --- | --- | --- |
| P1 | Delete `Parser.peek`. Nothing calls it | Low | Dead code |
| P2 | Choose the `for` form before parsing the header | Low | `parseForStmt` parses an init statement, then unwraps an `ExprStmt` back into a condition when `{` follows. Look ahead for `;` and parse `for {`, `for condition {`, and `for init; condition; post {` as three productions. Keep the current errors, including "post cannot be `let`" |
| P3 | Accept a trailing comma in parameter lists | Low | Expression lists already accept one. Parameter lists do not. One rule is enough |

`pub` and `kernel` are mutually exclusive in `parseFuncDecl`. `pub kernel func` does not parse. Kernels are visible by name after the module merge, so `pub` on a kernel has no job today. Leave the syntax until kernels need a visibility rule.

## Simplifications outside the parser

The type checker has two statement walkers and two expression walkers: one for ordinary functions and one for kernels. The C backend has the same split, plus a usage tracker and a hand-rolled IR walk that exists only to discover nested collection types.

| ID | Change | Complexity | Why |
| --- | --- | --- | --- |
| C1 | Delete `funcUsage` | Low | It exists to decide whether to emit `(void)trux_ctx` and `(void)trux_result_arena`. A void cast is valid when the parameter is used. Emit both casts for every ordinary function. The usage struct, `noteArena`, and the extra parameter on the emit helpers go away |
| C2 | Emit a kernel signature once | Low | `emitKernelPrototype` and `emitKernelFunc` each write the parameter list. Ordinary functions already share one signature helper. Give kernels the same `prototype bool` switch |
| C3 | Record nested collection families while building IR | Medium | `nestedCollectionFamilies` and its statement and expression walkers re-traverse the whole IR to find element types that need runtime typedefs. A new IR node has to be added there or codegen fails later with a missing C type. `ir.Build` already visits every node. Record the set on `ir.Program` and let codegen read it. Delete the walker |
| C4 | Elide the frame arena when a function does not allocate | Medium | `docs/OPEN_ISSUES.md`. Every function currently creates and destroys `trux_frame`, and every return goes through a shared cleanup label. Functions that only return scalars or borrowed views do not need that. The label and the return path have to change together |
| C5 | Panic on states the type checker already rejected | Medium | Codegen still returns `unsupported type` and `unsupported statement` for IR that cannot appear after a successful check. Those are compiler bugs. A panic with a fixed message, recovered once in `Generate`, removes a large amount of `if err != nil` plumbing. Keep real failures as errors: I/O, `cc` or `nvcc` failures, and the HTTP-plus-GPU rejection. Do this only if the surrounding code is being edited. It is a wide mechanical change |

Leave the host and kernel checkers split. Their allowed statements differ on purpose. A shared walker with a mode flag tends to hide a kernel rejection in a default branch. Share small helpers, such as "this binary operator on these two types", and keep the kernel `switch` explicit.

Leave side-effect ordering in the C emitter. It is tested, it covers calls, reads, indexes, slices, and literals, and it produces left-to-right preludes. Moving it into `ir.Build` is attractive and it is also a rewrite of working code. Do it when a new expression form cannot be ordered safely in the emitter. `&&` and `||` are that form. See L3.

## Missing language

These are gaps in the language a user can see. The first group is the everyday arithmetic and control flow that the current types almost imply. The second group is larger design work. The third group would fight the "small enough to read end to end" goal.

### Everyday gaps

| ID | Gap | Complexity | Notes |
| --- | --- | --- | --- |
| L1 | Unary minus | Low | `-1` and `-x` do not parse. `-` is only an infix token. Parse a prefix `-` on a numeric operand. For `int`, emit `rt_sub_i64(0, x)` so negating the minimum integer takes the existing overflow path. For `float`, emit C negation |
| L2 | Boolean not | Low | `!` is an illegal token today. `!(a == b)` is the useful form. Parse prefix `!`, require `bool`, emit C `!`. Do this with L1. The two prefixes share one parse function |
| L3 | `&&` and `||` | Medium | There is no way to combine conditions. Nested `if` is the only substitute. These operators must short-circuit. The current emitter evaluates both operands of a binary operator before the operator, which is correct for `+` and wrong for `&&`. `io.readInt() != 0 && io.readInt() != 0` must not read the second line when the first value is zero. Implement them in the emitter as ordered conditionals, or lower them in IR to `if` form before ordinary binary emission. Add them to kernels too |
| L4 | Remainder `%` | Low | New token, same precedence as `*` and `/`. Host and device helpers should reject a zero divisor. Also reject `INT64_MIN % -1`, which is undefined in C because the matching division is undefined |
| L5 | `break` and `continue` | Medium | Neither statement exists. Both emitters place the `for` post statement at the end of the body. A C `continue` would skip that post. Give each loop a continue label immediately before the post statement, in the host emitter and the kernel emitter, and implement `continue` as a jump to that label. `break` leaves the loop. Nested loops need a stack of labels. The host emitter already rewrites a side-effecting condition into `for (;;) { prelude; if (!cond) break; body; post; }`. `continue` has to land on `post`, not on the top of that loop, or the condition runs twice. There is no `switch`, so `break` means "leave the loop" only |
| L6 | Float division policy | Low | `1.0 / 0.0` follows IEEE and becomes an infinity. Integer division traps. Pick one and write it in `docs/SPECS.md`. Trapping float division is a small runtime helper. Documenting the split is smaller and is enough |
| L7 | `for { return 1 }` is rejected | Low | `blockTerminates` ignores `for`. A function whose only exit is `return` inside `for { }` fails with "must return on all paths" even though the loop body cannot fall through. The check is safe. Special-case an infinite `for` whose body terminates if that program should be legal |

### Larger design work

Do these when a real program is stuck, not as cleanup.

| ID | Gap | Complexity | Why it is large |
| --- | --- | --- | --- |
| L8 | `range` loops | Medium | Needs an iteration rule for arrays, slices, lists, and strings, plus the loop labels from L5. Strings are bytes, so a range over a string is a byte index unless the language grows runes at the same time |
| L9 | `switch` | Medium | New statement, new IR, two emitters. Use `break` only for loops if both features land, or the two meanings collide |
| L10 | Small `math` package | Low | `abs`, `min`, and `max` for `int` and `float` fit the existing stdlib pattern. No language change. `int` `abs` of the minimum integer must use the overflow failure |
| L11 | `args` and environment | Low | A small `os` package. `os.args` as `list[string]`, `os.getenv` as `string`. No language change. Decide what a missing variable returns before writing the runtime |
| L12 | Printing collections, or one format function | Medium | `print` accepts scalars only. A format string needs a checked argument list in the type checker, which is a new call form rather than another `print` overload |
| L13 | Structs or tuples | High | CSV rows and HTTP requests are awkward because the language has no product type. CSV is a flat `list[string]` plus a column count. An HTTP request is an `int` handle. A struct needs layout, literals, field access, ownership of string and collection fields, and C codegen. That is a language version, not a patch |
| L14 | Maps | High | A new container, a hashing story, and the same ownership questions as lists. The arena model does not make this cheap |
| L15 | Errors as values | High | IO, bounds failures, and arithmetic failures exit the process. That is consistent. Returning a result would change every standard signature and the generated ABI. Keep traps until a program needs to recover |
| L16 | Directory packages and one C file per module | High | Already the deferred module stage in `docs/MODULES.md`. The loader, name hygiene, and single-file backend are doing the job for the current programs |
| L17 | First-class functions | High | `http.serve` already takes a bare function name and checks its shape. Closures would capture frame memory and fight the arena model. Stop at the bare-name rule |
| L18 | PNG and JPEG | Medium | `docs/IMAGE.md` defers this. PPM is enough for the CPU and GPU examples. Adding a decoder is a runtime project, not a language project |
| L19 | HTTP on Windows | Medium | `docs/HTTP.md` says Windows builds fail at runtime. A second socket backend, or an explicit compile error on Windows when `http` is used. The second is the smaller fix |
| L20 | HTTP and GPU in one program | Medium | `compile.go` rejects the combination. The two runtimes want different compilers and different threading stories. Leave the rejection until there is a program that needs both |

### Out of scope

These would make the compiler harder to read without making the current programs possible. They are not on the list:

- Type inference on `let`. Explicit types are a source of the checker's simplicity.
- Generics beyond the built-in collections.
- Methods, interfaces, and operator overloading.
- Pointers, `nil`, `defer`, and manual `free`.
- Goroutines and channels. HTTP workers are the concurrency mechanism.
- Closures and nested functions.
- A general macro or compile-time evaluator.

Lexer extras that can land any time, each **Low**, and none of them unblock real programs: block comments, raw string literals, hex integer literals, underscores in numbers. Scientific notation (`1e10`) is the most useful of that set because float literals cannot express it today.

## Runtime and performance

Measured notes from the older review that are still true in the code:

| ID | Change | Complexity | Notes |
| --- | --- | --- | --- |
| R1 | Pool frame-arena chunks on `rt_context` | Medium | `rt_arena_new_chunk` still `malloc`s. `rt_arena_deinit` still `free`s. A freelist on the context lets a hot function reuse its chunk. Keep the freelist off a process-global so recursion stays correct. The old probe saw a hot string function drop from 165 ms to 64 ms with reuse |
| R2 | Rewind non-escaping loop temporaries | High | `rt_arena_mark_current` and `rt_arena_rewind` exist. Codegen never calls them. A loop that builds a temporary string each iteration keeps every temporary until the function returns. The old probe peaked near 33 MB for a couple of live bytes. Rewinding is safe only when nothing allocated in the iteration is stored into a variable that outlives the iteration, appended to a list, or returned. Getting that proof wrong is a dangling pointer. Do this after R1, and add both an escaping and a non-escaping stress test |
| R3 | Fewer mallocs on the small paths | Low | Three separate changes, benchmarked one at a time. Put the list header in the arena and register only the data pointer. Make `rt_read_file` size the file, allocate once, and read once. Grow arena chunks geometrically so a large frame does not take a 4 KB chunk per allocation |
| R4 | Device arithmetic failures have no message | Low | Host overflow prints `trux runtime error`. Device overflow executes `asm("trap;")`. Document that in `docs/GPU.md`. A device `printf` is optional and is not worth a new failure path by itself |

Generated code also emits `#line` directives back to `.tx` files. That is enough for a C debugger to land on Trux source. A Trux-level debugger is out of scope.

## Tooling

| ID | Gap | Complexity | Notes |
| --- | --- | --- | --- |
| T1 | Type diagnostics for files without `main` | Medium | `diagnosticsForDocuments` returns no type errors when the open file has no `main`. Library files and `.test.tx` files only get parse errors. Test files also need the same return-type insertion the test driver performs, or the checker reports "must declare return type" for a legal test |
| T2 | More than one diagnostic | Low | A file with several type errors reports the first one. Collecting them means continuing the checker after an error, which needs a policy for poison types. Start with "record the error and skip the bad statement" before building a poison-type lattice |
| T3 | Formatter and highlight coverage for new syntax | Low | Any of L1 through L5 has to update `trux fmt`, the TextMate grammar, and the LSP keyword list in the same change. Treat that as part of the feature, not a follow-up |

The language server already does completion, hover, definition, references, and format for the syntax that exists.

## Order

1. Correct `docs/SPECS.md`, `docs/MODULES.md`, and the README pipeline. Mark `TODO.md` as a historical review so the fixed bugs are not reopened.
2. L1 and L2 together, then L4. Unary minus, boolean not, and remainder. Small, local, and easy to test.
3. L3. `&&` and `||` with real short-circuit. This is the highest-value language hole.
4. C1 and C2. Delete `funcUsage` and share the kernel signature helper. Do these before more emitter work.
5. L5. `break` and `continue`, including the continue label in both emitters.
6. R1. Frame-chunk freelist.
7. C3. Record nested collection families during IR build, then delete the walker.
8. T1. Type diagnostics for library files and tests.
9. C4 and R3. Frame-arena elision, then the small allocation cuts, each with a benchmark.
10. R2 last. Loop rewind is the one performance change that can silently corrupt a correct program.

L8 and later wait until one of the examples cannot be written cleanly with the language above.
