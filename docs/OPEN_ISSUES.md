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
