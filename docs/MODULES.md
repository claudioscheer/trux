# Modules

## Current State

Trux has no module system yet. All code lives in a single file and is compiled together.

The current compiler only supports single-file programs.

## Generated C Code and Modules

The C backend currently emits everything into a single `.c` file: runtime helpers plus user code. This is acceptable while the language has no modules.

### Why Single-File Works Today

- The project is small.
- The runtime is compact.
- Codegen is much simpler with one output buffer.
- Compile times are fast for small programs.

### Why Single-File Becomes a Problem With Modules

Once modules are introduced, single-file C generation creates real issues:

- Every module's code gets combined into one giant file.
- No separate compilation: the entire program must be recompiled together.
- Poor scalability as programs grow.
- Difficult support for incremental builds or partial recompilation.
- Harder integration with existing C codebases and build systems.
- Debug information and compilation speed degrade.

A proper module system will eventually require the C backend to emit multiple `.c` files: one per module, plus runtime separation.

## Recommended Evolution Path

| Stage | Module Support | Generated C Output | Recommendation |
|-------|----------------|--------------------|----------------|
| Current | None | Single `.c` file | Acceptable |
| Initial modules | Basic imports without separate compilation | Mostly single file | Keep for simplicity |
| Real modules | Module-aware compilation | One `.c` file per module plus runtime files | Switch here |
| Full package system | Package exports and reusable artifacts | Proper multi-file output with headers | Required |

## Key Principle

Design codegen so moving from single-file to multi-file output is a controlled change, not a rewrite.

Even while staying single-file, the code generator should:

- Keep runtime code clearly separated from user code, even if concatenated.
- Avoid hard assumptions that only one file will ever be emitted.
- Preserve a path to per-module files and proper `#include` relationships.

## Design Considerations

- **Runtime separation**: The runtime should eventually live in its own `.c`/`.h` pair rather than being inlined into every generated program.
- **Module boundaries**: Each module should ideally map to at least one generated C file.
- **Public vs private**: The module system will need to control what gets exposed across generated C files.
- **Build model**: The `trux` driver will eventually need to coordinate compilation and linking of multiple generated C files.

## Relationship to Other Decisions

- **Arenas**: A multi-file model may make per-module or per-package arenas easier to explore later.
- **GPU kernels**: Kernel code and host code may naturally live in different files once modules exist.
- **Packages**: Packages are the main driver that will force proper multi-file C generation.

## Summary

Single-file C output is a reasonable strategy while the language has no modules. It should not become a permanent architectural assumption. The code generator should keep a clear migration path toward one-file-per-module output.

This decision should be revisited when module design work begins.
