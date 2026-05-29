# Modules

## Current State

Trux has no module system yet. All code lives in a single file and is compiled together.

Early versions of the compiler will only support single-file programs.

## Generated C Code and Modules

During the early phases, the C backend emits everything into a **single `.c` file** (runtime helpers + user code). This is acceptable while the language has no modules.

### Why Single-File Works Today

- The project is small and early.
- The runtime is minimal.
- Codegen is much simpler (one output buffer).
- Compile times are fast for small programs.

### Why Single-File Becomes a Problem With Modules

Once modules are introduced, single-file C generation creates real issues:

- Every module’s code gets smashed into one giant file.
- No separate compilation — the entire program must be recompiled together.
- Poor scalability as programs grow.
- Difficult to support incremental builds or partial recompilation.
- Harder to integrate with existing C codebases and build systems.
- Debug information and compilation speed degrade.

A proper module system will eventually require the C backend to emit **multiple `.c` files** (one per module, plus runtime separation).

## Recommended Evolution Path

| Stage | Module Support | Generated C Output | Recommendation |
|-------|----------------|--------------------|----------------|
| Now – Early | None | Single `.c` file | Acceptable |
| Stage 1 | Basic modules (no separate compilation) | Still mostly single file | Keep for simplicity |
| Stage 2 | Real modules | One `.c` file per module + runtime files | Switch here |
| Stage 3 | Full package system | Proper multi-file output with headers | Required |

### Key Principle

**Design the codegen so that moving from single-file to multi-file output is a controlled change, not a rewrite.**

Even while staying single-file, the code generator should:
- Keep runtime code clearly separated from user code (logically, even if concatenated).
- Avoid hard assumptions that only one file will ever be emitted.
- Support the future ability to emit per-module files and proper `#include` relationships.

## Design Considerations

- **Runtime separation**: The runtime (arenas, strings, GPU helpers, etc.) should eventually live in its own `.c`/`.h` pair rather than being inlined into every generated program.
- **Module boundaries**: Each module should ideally map to at least one generated C file.
- **Public vs private**: The module system will need to control what gets exposed across generated C files (similar to `pub` in the long-term spec).
- **Build model**: The `trux` driver will eventually need to coordinate compilation and linking of multiple generated C files.

## Relationship to Other Decisions

- **Arenas**: A multi-file model makes it easier to have per-module or per-package arenas if desired later.
- **GPU kernels**: Kernel code and host code may naturally live in different files once modules exist.
- **Packages** (long-term goal in SPECS.md): This is the main driver that will force proper multi-file C generation.

## Summary

Single-file C output is a reasonable temporary strategy while the language has no modules. It should not become a permanent architectural assumption. The code generator should be written with a clear migration path toward one-file-per-module output.

This decision should be revisited as soon as module design work begins.
