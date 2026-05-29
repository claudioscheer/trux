# Arenas as the Memory Model

## Decision

**Trux uses arena-based (region-based) allocation as the primary strategy for dynamic memory.**

All heap-allocated data — strings, slices, and future compound types — will be allocated from arenas rather than through individual `malloc`/`free` or a garbage collector.

## Why Arenas

The project has these constraints and goals:

- First backend targets C.
- The compiler itself should stay relatively small and understandable.
- The generated C should remain readable and debuggable by humans.
- The language is intentionally "small" (see [SPECS.md](SPECS.md)).
- The primary purpose is learning compiler construction and language design.

Under these constraints, arenas are the best fit:

- Allocation is extremely cheap (bump pointer).
- There is no per-object deallocation overhead or complexity.
- The runtime stays tiny.
- Generated code stays simple (no write barriers, no refcount inc/dec, no safepoints).
- The approach is proven in real systems languages and high-performance codebases (Odin, Jai, C3, game engines, compilers, PostgreSQL memory contexts, etc.).

## Current State

As of the early implementation:

- v0 only supports `int`. No dynamic allocation exists.
- v1 adds `string` literals and `print(string)`. These are backed by static storage (`const uint8_t*` pointing into read-only data).
- No arenas are present in the runtime or generated code yet.

The first pressure appears when the language gains operations that produce *new* dynamic string data at runtime (concatenation, formatting, etc.).

## Planned Runtime Shape

When dynamic allocation is introduced:

- A single primary `rt_program_arena` (or a small number of well-known arenas) will back dynamic values.
- `rt_string` remains a length-based immutable view:

  ```c
  typedef struct {
      const uint8_t* data;
      size_t len;
  } rt_string;
  ```

  Its `data` pointer may refer either to static literal storage or to arena-allocated dynamic storage. Arena memory is writable during construction but exposed as immutable through `rt_string`.

- String literals continue to use static storage for efficiency.
- Dynamic strings are allocated by copying into the arena.
- There is **no individual `free`** for strings or other heap objects.

A recommended separation is to keep `rt_string` strictly immutable and introduce a separate builder type for construction (e.g. `rt_string_builder` with capacity). This avoids giving the language-level string type surprising mutability behavior depending on provenance.

Early generated code will likely look like:

```c
rt_string greet(rt_string name, rt_arena* arena) {
    rt_string lit1 = rt_string_lit("Hello, ");
    rt_string lit2 = rt_string_lit("!");
    rt_string tmp = rt_string_concat(lit1, name, arena);
    return rt_string_concat(tmp, lit2, arena);
}
```

Two important details must be decided before heavy use of dynamic strings:

- Whether `rt_string` values are strictly length-based or also null-terminated (affects printing, FFI, and `rt_string_concat` implementation).
- The exact invalidation rules for arena-allocated data (when a returned `rt_string` may become invalid after an arena reset). These rules belong in the language specification, not just the runtime.

## Tradeoffs

**Benefits**
- Extremely fast allocation and bulk deallocation.
- Very small runtime surface.
- No hidden costs from a GC or pervasive reference counting.
- Predictable performance (no pauses).
- Simple mental model once the lifetime rules are understood.

**Costs and Risks**
- All dynamic data lives until its arena is reset or destroyed.
- Memory usage can grow monotonically in long-running loops unless arenas are reset.
- A single global arena is not reentrant and not thread-safe. It should be avoided as soon as any generated function can perform allocation.
- The language must define clear rules about when values allocated into an arena become invalid. These invalidation rules should be specified *before* introducing aggressive temp-arena resets.
- Some patterns (mixing very different lifetimes without multiple arenas) become awkward.
- "Graveyard" waste can occur when growing containers (dynamic arrays, string builders) inside an arena.

The biggest risk is **under-specifying lifetimes in the language semantics**. If the language does not clearly state when a string returned from a function may become invalid, every C representation is just a temporary workaround. The hard problem is not the arena implementation — it is defining when allocated values become invalid.

## Evolution Path

We expect to evolve the arena strategy in stages rather than jumping to a complex model immediately.

### Stage 0 (initial dynamic allocation)
A single explicit arena (ideally passed down from `main` rather than a true global). All dynamic data lives until the arena is reset. This is acceptable for very early experimentation and short-lived programs, but a true global arena should be phased out as soon as any generated function can perform allocation.

### Stage 1 (compiler-managed temporary arena)
Introduce a distinguished `temp_arena` used for expression temporaries. The compiler inserts resets at safe boundaries (after statements, after loop iterations, after calls when results are not captured). This dramatically reduces memory growth in common cases while remaining invisible to the source language.

### Stage 2 (user-visible control)
Add minimal user-facing mechanisms so programmers can control when memory is reclaimed for larger batches:
- `pool { ... }` blocks (scoped temporary arenas), or
- An ambient temporary context that can be reset at application-defined points (e.g. per request, per frame, per file).

At this stage the language surface changes for the first time.

### Stage 3 (first-class arenas)
Only if the language outgrows the small scope: expose an `Arena` / `Allocator` type so users can create and manage their own regions explicitly. This is a significant increase in language complexity and is not a near-term goal.

## Relationship to Garbage Collection

GC was explicitly considered and rejected for the early lifetime of the project.

Adding a real GC (even a conservative one) would:
- Dramatically increase the size and complexity of both the runtime and the compiler.
- Pollute generated C with write barriers, stack maps, and safepoints.
- Change the fundamental character of the project from "a small compiler that emits C" to "a language with its own runtime and memory management."
- Inherit an entire category of hard problems (pauses, tuning, rooting, FFI interactions) that do not exist with arenas.

Reference counting was also evaluated. It tends to produce more generated code noise than arenas and still requires a solution for cycles.

Arenas are not a temporary hack before "real" memory management. They are a coherent, production-used strategy for systems languages and performance-oriented code. Many successful projects never move beyond well-managed arenas.

GC (or a hybrid) remains a possible *later experiment* (e.g. an optional `--gc` backend using Boehm), but it is not the default path.

## GPU Kernels and CUDA Support

The long-term roadmap includes GPU kernels (see [SPECS.md](SPECS.md)). Arenas interact differently with host-side and device-side code.

### Host side
Host-side arenas remain compatible and are often a good fit when managing GPU resources. Staging buffers, device memory descriptors, command streams, and per-launch temporaries are natural candidates for arena allocation with bulk reset at stream synchronization points. This pattern is common in real CUDA codebases.

### Device side (inside kernels)
Dynamic allocation inside CUDA kernels is fundamentally different:

- Most high-performance GPU code deliberately avoids it.
- A device-side arena requires warp-cooperative atomics, careful memory ordering, and has much worse performance characteristics than CPU arenas.
- The current spec shows restricted kernels that operate on pre-allocated `gpu.Buffer` objects. This style aligns well with arenas on the host and explicit buffer management on the device.

If the language later wants rich dynamic data structures inside kernels, arenas will not be sufficient by themselves. A separate device-side allocation strategy (or strong restrictions) will be needed. The host and device memory models should not be unified casually.

### Recommendation
- Continue using arenas for host-side management of GPU-related resources.
- Treat device memory as mostly pre-allocated buffers for the initial CUDA support.
- Design any future device-side dynamic allocation as a distinct mechanism rather than an extension of the CPU arena model.

This keeps the CPU-side arena decision intact while acknowledging the very different constraints of GPU execution.

## Implications

### For Generated Code
The C backend will be written to make arena usage explicit and controllable rather than hidden behind a global. Passing an arena (or a small context struct containing arenas) to functions that allocate is preferred over implicit globals as soon as multiple lifetime categories appear.

### For Language Semantics
The most important work is not in the runtime — it is defining what the language guarantees:

- When is a value returned from a function that performed allocation still valid?
- What operations can invalidate previously valid strings/slices?
- Are there distinguished "temporary" values whose lifetime is shorter than a named `let` binding?
- What are the precise invalidation rules for arena-allocated data (especially once temp arenas and resets are introduced)?

These questions should be answered in the language specification before the compiler starts inserting aggressive arena resets or relying on escape analysis. Until the invalidation model is explicit, the generated C shape remains provisional.

### For the Learning Goal
Using arenas forces the project to confront lifetime reasoning, escape analysis, and the gap between language semantics and backend implementation. These are more valuable lessons for a systems-oriented compiler project than integrating an off-the-shelf GC.

## References

- [SPECS.md](SPECS.md) — current language scope and roadmap
- Odin, Jai, and C3 memory models (for examples of successful arena-centric designs)
- Region-based memory management literature (Cyclone, ML Kit, Tofte-Talpin)
- CUDA memory pools and async allocation patterns (for host-side GPU resource management)
- [MODULES.md](MODULES.md) — impact of modules on generated C output strategy

---

This document should be updated as the implementation and language semantics evolve. The decision is deliberate and staged, not accidental.