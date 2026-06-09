# Modules

## Current State

Trux now has initial module support. Programs can load multiple `.tx` source files through relative imports, while the backend still compiles the loaded program into one generated program source (`.c` or `.cu`) plus the shared `trux_runtime.h` header.

The implemented stage supports `import "relative/path.tx"`, standard package imports such as `import "io"` and `import "csv"`, `pub func`, package-qualified calls such as `math.add(...)`, package-private functions across same-directory files that declare the same package, private-name hygiene, import de-duplication, different package names per loaded file, multiple direct imports that share one package name, and cycle detection. Recursive directory-based packages, reusable package artifacts, and separate C compilation are still deferred.

## Generated C Code and Modules

The C backend currently emits one program source file for all loaded Trux modules and a companion runtime header. This remains acceptable for the initial module stage because the loader merges modules before type checking, IR building, and C generation.

### Why One Program Source Works Today

- The project is small.
- The runtime is shared through a header instead of being copied into the generated program source.
- Codegen is much simpler with one program output buffer.
- Compile times are fast for small programs.

### Why One Program Source Becomes a Problem With Modules

The initial module implementation deliberately stays with one generated program source (using a loader + hygiene pass). However, even this limited form of modules makes the long-term limitations of that generation model clearer:

- Every module's code gets combined into one generated program source.
- No separate compilation: the entire program must be recompiled together.
- Poor scalability as programs grow.
- Difficult support for incremental builds or partial recompilation.
- Harder integration with existing C codebases and build systems.
- Debug information and compilation speed degrade.

A proper module system will eventually require the C backend to emit multiple `.c` files: one per module, plus the shared runtime header.

## Recommended Evolution Path

| Stage | Module Support | Generated C Output | Recommendation |
|-------|----------------|--------------------|----------------|
| Current / initial modules | Multiple `.tx` source files through relative imports (`import "foo.tx"`) + package-qualified `pub` visibility. Imported source files load same-directory siblings that declare the same package. Private functions are package-private within that group and may share names only across different package groups. Public names may overlap across packages. Cycle detection. | One generated program source plus `trux_runtime.h` (via loader + private name hygiene merge into existing pipeline) | Keep for simplicity. Preserves the existing type checker, IR, and codegen with minimal changes. |
| Real modules | Module-aware compilation, proper boundaries | One `.c` file per module plus runtime files | Switch here |
| Full package system | Package exports and reusable artifacts | Proper multi-file output with headers | Required |

## Initial Modules (Approach)

The first cut of module support uses the following concrete design:

- **Source model**: Multiple `.tx` source files are supported, but the backend still emits exactly one program source file for the whole loaded program.
- **Import syntax**: `import "relative/path/to/mod.tx"` for source modules, plus bare standard package imports such as `import "io"` and `import "csv"`. Source paths are resolved relative to the directory of the file containing the `import` statement. The `.tx` extension is required and explicit for source modules.
- **Package loading**: Importing a source file also loads same-directory sibling `.tx` files that declare the same package. Entry files do not auto-load siblings.
- **Import placement**: Imports are top-level declarations after `package` and before any `func` declarations. Grouped imports and imports inside functions are not part of the initial implementation.
- **Visibility**: `pub func` makes a function visible to direct importers through `package.function(...)`. Functions without `pub` are package-private: callable unqualified from loaded files in the same directory with the same package name, but not callable by external importers.
- **Name rules**:
  - No two functions in the same-directory package group may share the same source name.
  - Public function names may overlap across packages because imported calls are package-qualified.
  - A file may directly import multiple files that declare the same package name when their public function names are unique across the shared package qualifier.
  - Private functions may have the same name in different package groups.
- **Implementation technique**: A loader parses the entry file and all transitive imports, detects cycles, and performs a small hygiene/merge pass. Private functions receive compiler-unique internal names only for the duration of type checking and code generation. The result is a single flat `Program` fed to the existing type checker, IR builder, and code generator. This keeps changes to the existing type checker and codegen small.
- **Output**: One generated `.c` or `.cu` file containing all functions from all loaded modules, plus `trux_runtime.h`.
- **Error reporting**: Parse and type errors carry the originating file path so diagnostics from imported modules are clear.
- **Package clause**: Loaded files may use different package names. An imported file's package name is the qualifier used by direct importers.
- **Entrypoint**: Exactly one `main` function is allowed, and it must be declared in the entry file. It is the compiler entrypoint, not a module export. Imported files must not define `main`.

This approach deliberately stays inside the "controlled change, not a rewrite" constraint. The existing flat `Parse` → `Check` → `ir.Build` → `Generate` pipeline continues to work for both single-source programs and the merged multi-file case.

Directory-based packages and separate C compilation are deferred to later stages.

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

- Source import paths must be relative string literals. Absolute paths are rejected.
- Bare imports are reserved for standard packages. Currently supported standard packages are `io` and `csv`; other bare imports are rejected.
- Import paths are normalized with `filepath.Clean` and resolved to canonical absolute paths before de-duplication and cycle detection.
- A source file is loaded at most once, even if reached through multiple import paths such as `foo.tx` and `./foo.tx`.
- Loading an imported source file also loads same-directory siblings whose package clause matches that imported file.
- Import cycles are reported with the import chain, including file paths.
- Missing source imports, directories, and non-`.tx` source paths are compile errors.
- Public functions from directly imported package groups are visible only through `package.function(...)`. When multiple direct imports declare the same package name, that package qualifier resolves across all uniquely named public functions in those package groups.
- Transitive imports are loaded for dependencies, but they do not introduce callable package qualifiers into the importing file.

This still treats imports closer to controlled source inclusion than to reusable package artifacts. The tradeoff is intentional: it avoids separate module compilation while still giving source code a real package-qualified call surface.

## Visibility and Name Hygiene

The merged program must not depend on C-level collisions or checker-level duplicate private names.

Rules:

- Reserve a compiler-internal prefix, such as `__trux_`, for generated names. User function names beginning with that prefix are rejected.
- Public and private functions, except the entry file's `main`, are rewritten to compiler-unique internal names before type checking when multiple files are loaded.
- Calls to same-package functions stay unqualified in source and are rewritten to the target function's internal name.
- Calls to imported public functions must use the direct import package qualifier, such as `math.add(...)`.
- If a file has a private function with the same name as an imported public function, unqualified calls from that file resolve to the same-file private function. The imported public function remains callable through `package.function(...)`.
- Qualified calls to private functions are rejected, including same-package private functions. Same-package private functions are called unqualified.
- Calls to another package group's private function are rejected.

The private-name rewrite is a temporary compatibility layer for the current flat checker and code generator. Real module boundaries should replace this with explicit name resolution later.

## Suggested Implementation Order

1. Add `import` and `pub` tokens, AST fields, and parser support.
2. Add file-aware source positions or an equivalent source map before adding the loader. Diagnostics are harder to retrofit after AST rewrites exist.
3. Add a loader package that parses the entry file, resolves imports, canonicalizes paths, de-duplicates files, and detects cycles.
4. Add name and visibility validation before merging files.
5. Add private-name hygiene and call rewriting.
6. Wire the loader into `compileFile` while preserving the existing single-program-source path.
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
- Using imported package names as call qualifiers, even when the file name differs.
- De-duplicating the same file reached through different relative paths.
- Detecting import cycles with a useful chain.
- Rejecting missing imports, absolute imports, and non-`.tx` imports.
- Allowing same-named private functions in different package groups.
- Loading same-directory package siblings for imported source files.
- Resolving unqualified package-private calls across same-directory package siblings.
- Rejecting duplicate function names in a same-directory package group.
- Keeping entry-file siblings out of the load graph unless explicitly imported.
- Allowing duplicate public function names across different packages.
- Allowing multiple direct imports with the same package name when public export names are unique.
- Rejecting duplicate public function names across same-package direct imports.
- Rejecting user function names that use the reserved compiler-internal prefix.
- Rewriting same-package calls to internal names.
- Rejecting qualified calls to private functions.
- Rejecting calls to another package group's private function.
- Rejecting unqualified calls to imported public functions.
- Rejecting qualified calls to transitively imported packages unless the calling file imports that package directly.
- Resolving a same-file private function before a loaded public function with the same source name.
- Keeping the entry file's `main` as the generated program entrypoint.
- Rejecting `main` in imported files.
- Reporting parse and type errors from imported files with the imported file path and source context.
- Compiling and running a small multi-file program through `trux run`.

## Key Principle

Design codegen so moving from one generated program source to multi-file output is a controlled change, not a rewrite.

For the initial modules stage we are additionally using a **private name hygiene** technique (plus a thin loader + merge pass) so that the existing type checker, IR builder, and code generator can continue to operate on a single flat `Program`. This keeps the blast radius small while still delivering real `import` + `pub` semantics.

Even while staying with one generated program source, the code generator should:

- Keep runtime code clearly separated from user code.
- Avoid hard assumptions that only one file will ever be emitted.
- Preserve a path to per-module files and proper `#include` relationships.

## Design Considerations

- **Runtime separation**: The runtime currently lives in `trux_runtime.h`; a future `.c`/`.h` split may still be useful if the runtime stops being header-only.
- **Module boundaries (initial cut)**: For the first implementation, each imported `.tx` file anchors a same-directory package group. Private name hygiene lets us support package-private calls without changing the flat checker/codegen. Real per-module C files come later.
- **Public vs private**: `pub func` controls external visibility through direct package-qualified calls. Private functions are package-private within their same-directory package group. The initial implementation uses compiler-internal renaming to avoid name clashes while preserving one merged compiler pipeline.
- **Build model**: The `trux` driver will eventually need to coordinate compilation and linking of multiple generated C files. In the initial modules stage the driver grows a relative-import loader and hygiene pass but still emits one program source plus `trux_runtime.h`.
- **Error provenance**: Parse and type errors must carry originating file paths (not just line/column) so diagnostics from imported modules are actionable.

## Relationship to Other Decisions

- **Arenas**: A multi-file model may make per-module or per-package arenas easier to explore later.
- **GPU kernels**: Kernel code and host code may naturally live in different files once modules exist.
- **Packages**: Packages are the main driver that will force proper multi-file C generation.

## Summary

One generated program source (augmented with a loader + private name hygiene) is the right strategy for the initial module implementation. It delivers useful `import "..."`, `pub`, and package-private same-directory functionality with minimal disruption to the existing type checker, IR, and code generator.

It must not become a permanent architectural assumption. The code generator and driver must keep a clear, low-friction migration path toward one-file-per-module output and proper separate compilation.

This decision (and the hygiene technique) should be revisited once real modules are in use and the pain of one generated program source becomes concrete.
