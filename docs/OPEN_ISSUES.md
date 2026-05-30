# Open Issues

## C backend: elide frame arenas when a function has no frame allocations

Status: open

Generated C currently creates and destroys `trux_frame` for every Trux function. This is safe, but unnecessary for functions that only return borrowed data or scalar expressions and do not allocate into the frame arena.

Example target shape:

```c
rt_slice_int trux_borrowedMiddle(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_slice_int trux_v_2_xs) {
    (void)trux_ctx;
    (void)trux_result_arena;
    return rt_slice_int_slice(trux_v_2_xs, true, 1, true, 3);
}
```

Tradeoff: this would make generated C smaller and avoid unnecessary arena init/deinit calls, but it requires changing return lowering because the current backend uses a shared `trux_return` cleanup label for all functions.

## Language: no input I/O primitives (only print output exists)

Status: open

The language provides only one output mechanism (`print` of scalars, which lowers to rt_print_* + newline in generated C). There are no input facilities whatsoever:

- `func main() int` — parameters are rejected at type-check time (`internal/types/types.go:161`: "main must not have parameters").
- No `read`, `input`, `scan`, or similar builtin (the only CallExpr special forms are print/len/clone/append; see types.go:489 and ir.go:376).
- No stdin/file reads, no argv, no env access, no line-based or typed input.
- Runtime (rt.go) contains only printf/fwrite writers for output + stderr fatals; zero read helpers.

Programs are therefore limited to pure computation that produces output. Interactive programs, data ingestion, or any stdin-driven logic are impossible.

```trux
func main() int {
    // no syntax or builtin exists to populate x from input
    let x int = /* ??? */
    print(x + 1)
    return 0
}
```

Tradeoff: adding input I/O requires choosing a model (e.g. `read_line() string` owning a frame string, or typed `read_int()` that fatals on bad input to match existing error style, or a streaming iterator) that respects the arena/ownership rules without introducing new failure modes the rest of the language does not have. It also means new AST/IR nodes, checker cases, codegen emission, and runtime C helpers — a larger surface-area increase than the current single print statement. The cost is language completeness for real-world utility; the benefit of staying minimal is a tiny compiler and simple mental model.

## Language: no support for nested collections (jagged lists, slices, and arrays)

Status: open

Collection element types are restricted to scalars. Any attempt to use a nested collection type is rejected:

```
collection element type must be scalar, got list[int]
```

(See `internal/types/types.go:715` (`validateElementType`) and its callers from `validateType` for array/slice/list types, plus the literal and make checks.)

```trux
let bad list[list[int]] = list[list[int]]{list[int]{1, 2}, list[int]{3}}  // rejected
let fixed [2][2]int = [2][2]int{[2]int{1, 2}, [2]int{3, 4}}               // rejected
let worse [][]float = make([][]float, 3)                                  // rejected
```

The desired direction is jagged nested collections, not a matrix abstraction. Each inner collection should carry its own length and backing storage, and the user is responsible for any row-shape invariant. The compiler does not need to enforce rectangular dimensions or equal row lengths. Existing runtime bounds checks are enough for invalid indexing.

This is still more than deleting `validateElementType`. Current codegen and runtime only support scalar collection element families:

- `emitValueArray` lowers collection literals through `emitScalarType`, so collection-valued elements cannot currently be materialized.
- `elemRuntimeName` only names `int`, `float`, `bool`, and `string`, so helper names such as `rt_list_list_int` or `rt_slice_slice_float` do not exist.
- The runtime only instantiates `RT_DEFINE_COLLECTIONS` for scalar element types.
- Ownership must propagate through nested indexing. If `xs` is a borrowed `list[list[int]]`, then `xs[0]` must be treated as borrowed too; it cannot become an owned mutable inner list.
- `clone` must either deep-clone nested collections or reject clone on nested collection types until deep clone exists.

Tradeoff: allowing nested collections gives users natural jagged 2D and higher-rank data without a rectangular matrix type, and it avoids flat collection plus manual index math. The cost is recursive collection support across type checking, IR/codegen, runtime helper generation, ownership propagation, and clone semantics. Skipping rectangular validation removes shape complexity, but it does not remove the representation and aliasing work.
