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
