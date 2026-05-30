# Modules

## Current State

Trux now has initial module support. Programs can load multiple `.tx` source files through relative imports, while the backend still compiles the loaded program into a single generated `.c` file.

The implemented stage supports `import "relative/path.tx"`, `pub func`, private file-local functions, private-name hygiene, transitive public function visibility, import de-duplication, different package names per loaded file, and cycle detection. Qualified names, directory-based packages, reusable package artifacts, and separate C compilation are still deferred.

## Generated C Code and Modules

The C backend currently emits everything into a single `.c` file: runtime helpers plus user code. This remains acceptable for the initial module stage because the loader merges modules before type checking, IR building, and C generation.

### Why Single-File Works Today

- The project is small.
- The runtime is compact.
- Codegen is much simpler with one output buffer.
- Compile times are fast for small programs.

### Why Single-File Becomes a Problem With Modules

The initial module implementation deliberately stays with single-file C output (using a loader + hygiene pass). However, even this limited form of modules makes the long-term limitations of single-file generation clearer:

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
| Current / initial modules | Multiple `.tx` source files through relative imports (`import "foo.tx"`) + `pub` visibility. Private functions are file-local and may share names across files. Single flat namespace for public names. Cycle detection. | Single `.c` file (via loader + private name hygiene merge into existing pipeline) | Keep for simplicity. Preserves the existing type checker, IR, and codegen with minimal changes. |
| Real modules | Module-aware compilation, proper boundaries | One `.c` file per module plus runtime files | Switch here |
| Full package system | Package exports and reusable artifacts | Proper multi-file output with headers | Required |

## Initial Modules (Approach)

The first cut of module support uses the following concrete design:

- **Source model**: Multiple `.tx` source files are supported, but the backend still emits exactly one `.c` file for the whole loaded program.
- **Import syntax**: `import "relative/path/to/mod.tx"`. Paths are resolved relative to the directory of the file containing the `import` statement. The `.tx` extension is required and explicit.
- **Import placement**: Imports are top-level declarations after `package` and before any `func` declarations. Grouped imports and imports inside functions are not part of the initial implementation.
- **Visibility**: `pub func` makes a function visible to importers (Go-style). Functions without `pub` are private to their declaring source file.
- **Name rules**:
  - No two functions in the same file may share the same source name.
  - Public function names must be unique across the entire program.
  - Private functions may have the same name in different files (they are file-local).
- **Implementation technique**: A loader parses the entry file and all transitive imports, detects cycles, and performs a small hygiene/merge pass. Private functions receive compiler-unique internal names only for the duration of type checking and code generation. The result is a single flat `Program` fed to the existing type checker, IR builder, and code generator. This keeps changes to the existing type checker and codegen small.
- **Output**: Still one concatenated `.c` file containing the runtime plus all functions from all modules.
- **Error reporting**: Parse and type errors carry the originating file path so diagnostics from imported modules are clear.
- **Package clause**: Currently decorative (still required by the parser for compatibility). Loaded files may use different package names. The entry file's package name is kept on the merged `Program`, but package names are not used for resolution or qualification in this stage.
- **Entrypoint**: Exactly one `main` function is allowed, and it must be declared in the entry file. It is the compiler entrypoint, not a module export. Imported files must not define `main`.

This approach deliberately stays inside the "controlled change, not a rewrite" constraint. The existing single-file `Parse` → `Check` → `ir.Build` → `Generate` pipeline continues to work for both single-file programs and the merged multi-file case.

Qualified names (e.g. `math.add`), directory-based packages, and separate C compilation are deferred to later stages.

## Loader and Import Rules

The loader sits in front of the existing compiler pipeline:

```text
entry file
  -> load entry and transitive imports
  -> parse each file
  -> validate imports, names, and visibility
  -> rewrite private names
  -> merge into one ast.Program
  -> Check
  -> ir.Build
  -> Generate
```

Concrete loader rules:

- Import paths must be relative string literals. Absolute paths are rejected.
- Import paths are normalized with `filepath.Clean` and resolved to canonical absolute paths before de-duplication and cycle detection.
- A source file is loaded at most once, even if reached through multiple import paths such as `foo.tx` and `./foo.tx`.
- Import cycles are reported with the import chain, including file paths.
- Missing imports, directories, and non-`.tx` paths are compile errors.
- Public functions from all transitively loaded files live in one flat public namespace for this initial stage.

This treats imports closer to controlled source inclusion than to real package namespaces. The tradeoff is intentional: it avoids module-aware type checking while still allowing multi-file programs. The cost is that public names are globally visible once loaded, so this is not the final package model.

## Visibility and Name Hygiene

The merged program must not depend on C-level collisions or checker-level duplicate private names.

Rules:

- Reserve a compiler-internal prefix, such as `__trux_`, for generated names. User function names beginning with that prefix are rejected.
- Public functions keep their source name in the merged `Program`.
- Private functions, except the entry file's `main`, are rewritten to compiler-unique internal names before type checking.
- Calls to a same-file private function are rewritten to that function's internal name.
- Calls to public functions keep the public source name.
- If a file has a private function with the same name as a loaded public function, calls from that file resolve to the same-file private function. There is no syntax yet to call the hidden public function from that file.
- Calls to another file's private function are rejected because the private function name is never exposed in the merged public namespace.

The private-name rewrite is a temporary compatibility layer for the current flat checker and code generator. Real module boundaries should replace this with explicit name resolution later.

## Suggested Implementation Order

1. Add `import` and `pub` tokens, AST fields, and parser support.
2. Add file-aware source positions or an equivalent source map before adding the loader. Diagnostics are harder to retrofit after AST rewrites exist.
3. Add a loader package that parses the entry file, resolves imports, canonicalizes paths, de-duplicates files, and detects cycles.
4. Add name and visibility validation before merging files.
5. Add private-name hygiene and call rewriting.
6. Wire the loader into `compileFile` while preserving the existing single-file path.
7. Add integration tests through `trux run` and `trux emit-c`.

The main tradeoff is doing diagnostics early. It adds plumbing before visible module behavior exists, but it avoids a more expensive rework once imported-file errors and rewritten names start flowing through the compiler.

## Diagnostics and Source Provenance

Initial modules require file-aware diagnostics, not just line and column.

The command path now formats parser and type errors against the source file that produced the error. With imports, errors must point at the imported file when that file produced the diagnostic. The implementation carries the source file path in positions and keeps a source map through parsing, hygiene, and type checking.

Diagnostics should include:

- The file path, line, and column of parse and type errors.
- Source context from the file that actually contains the error.
- Import-cycle errors that show the cycle chain.
- Name-conflict errors that show both conflicting file paths when possible.

## Required Tests for Initial Modules

Before treating module support as complete, add focused tests for:

- Lexing and parsing `import` and `pub func`.
- Rejecting imports after function declarations.
- Resolving relative imports from the importing file's directory.
- Allowing imported files to use package names that differ from the entry file.
- De-duplicating the same file reached through different relative paths.
- Detecting import cycles with a useful chain.
- Rejecting missing imports, absolute imports, and non-`.tx` imports.
- Allowing same-named private functions in different files.
- Rejecting duplicate public function names across loaded files.
- Rejecting user function names that use the reserved compiler-internal prefix.
- Rewriting same-file private calls to internal names.
- Rejecting calls to another file's private function.
- Resolving public functions from transitively loaded imports according to the initial flat public namespace rule.
- Resolving a same-file private function before a loaded public function with the same source name.
- Keeping the entry file's `main` as the generated program entrypoint.
- Rejecting `main` in imported files.
- Reporting parse and type errors from imported files with the imported file path and source context.
- Compiling and running a small multi-file program through `trux run`.

## Key Principle

Design codegen so moving from single-file to multi-file output is a controlled change, not a rewrite.

For the initial modules stage we are additionally using a **private name hygiene** technique (plus a thin loader + merge pass) so that the existing type checker, IR builder, and code generator can continue to operate on a single flat `Program`. This keeps the blast radius small while still delivering real `import` + `pub` semantics.

Even while staying single-file, the code generator should:

- Keep runtime code clearly separated from user code, even if concatenated.
- Avoid hard assumptions that only one file will ever be emitted.
- Preserve a path to per-module files and proper `#include` relationships.

## Design Considerations

- **Runtime separation**: The runtime should eventually live in its own `.c`/`.h` pair rather than being inlined into every generated program.
- **Module boundaries (initial cut)**: For the first implementation, each `.tx` file is a module. Private name hygiene lets us support per-file privacy without changing the flat checker/codegen. Real per-module C files come later.
- **Public vs private**: `pub func` controls cross-file visibility. Private functions are file-local. The initial implementation uses compiler-internal renaming for private functions to avoid name clashes while keeping a single flat namespace for public names.
- **Build model**: The `trux` driver will eventually need to coordinate compilation and linking of multiple generated C files. In the initial modules stage the driver grows a relative-import loader and hygiene pass but still emits one `.c`.
- **Error provenance**: Parse and type errors must carry originating file paths (not just line/column) so diagnostics from imported modules are actionable.

## Relationship to Other Decisions

- **Arenas**: A multi-file model may make per-module or per-package arenas easier to explore later.
- **GPU kernels**: Kernel code and host code may naturally live in different files once modules exist.
- **Packages**: Packages are the main driver that will force proper multi-file C generation.

## Summary

Single-file C output (augmented with a loader + private name hygiene) is the right strategy for the initial module implementation. It delivers useful `import "..."` + `pub` functionality with minimal disruption to the existing type checker, IR, and code generator.

It must not become a permanent architectural assumption. The code generator and driver must keep a clear, low-friction migration path toward one-file-per-module output and proper separate compilation.

This decision (and the hygiene technique) should be revisited once real modules are in use and the pain of single-file C becomes concrete.
