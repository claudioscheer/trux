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

## Language: I/O should live in an imported package instead of global builtins

Status: open

I/O now works, but it is exposed as compiler-recognized global builtins (`read_line`, `read_int`, `read_file`, `write_file`, `read_csv`, and `write_csv`). That keeps examples small, but it makes I/O part of the always-available global language surface.

Current shape:

```trux
package main

func main() int {
    let name string = read_line()
    let note string = read_file("input.txt")
    write_file("copy.txt", note + "\n" + name)
    return 0
}
```

Desired direction:

```trux
package main

import "io"

func main() int {
    let name string = io.read_line()
    let note string = io.read_file("input.txt")
    io.write_file("copy.txt", note + "\n" + name)
    return 0
}
```

This should be a real package boundary, not only a cosmetic alias. Module support now uses relative `.tx` imports and qualified package member resolution for source files, but `import "io"` still needs standard-library package resolution and a decision about how low-level runtime-backed operations are exposed behind that package facade.

Tradeoff: keeping I/O as builtins is simple and matches the current lowering path for `print`, `len`, `clone`, and `append`. Moving I/O into an imported package makes side-effectful capabilities explicit and keeps the global namespace smaller, but it requires standard-library package resolution and a decision about whether low-level runtime-backed operations can be implemented as ordinary package functions or need privileged compiler lowering behind the package facade.
