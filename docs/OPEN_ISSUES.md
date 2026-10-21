# Open Issues

## C backend: elide frame arenas when a function has no frame allocations

Status: resolved. `functionNeedsFrame` in `internal/codegen/c/c.go` now skips `trux_frame` for functions that do not allocate into it, so `borrowedMiddle` in `examples/ownership_clone.tx` no longer creates a frame arena. It still returns through the shared `trux_return` label instead of the direct `return` shown below.

Generated C used to create and destroy `trux_frame` for every Trux function. This was safe, but unnecessary for functions that only return borrowed data or scalar expressions and do not allocate into the frame arena.

Example target shape:

```c
rt_slice_int trux_f_14_borrowedMiddle(rt_context* trux_ctx, rt_arena* trux_result_arena, rt_slice_int trux_v_2_xs) {
    (void)trux_ctx;
    (void)trux_result_arena;
    return rt_slice_int_slice(trux_v_2_xs, true, 1, true, 3);
}
```

Tradeoff: this would make generated C smaller and avoid unnecessary arena init/deinit calls, but it requires changing return lowering because the current backend uses a shared `trux_return` cleanup label for all functions.

## Type checker: GPU coordinate helpers are accepted in host code

Status: open

`gpu.globalX()` and the other coordinate helpers type-check outside a `kernel func`, but host code has no `blockIdx` or `threadIdx`, so the generated C fails to compile with an error such as `'blockIdx' undeclared`. The checker should reject these calls outside kernels with a Trux diagnostic.
