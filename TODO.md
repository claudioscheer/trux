Now let me look at the C runtime — that's where arena semantics, string handling, and the collection macros live.Now let me check the IR ownership model and how `void`/returns are handled, then build and run the test suite.It builds and runs with Go 1.22 (the `go 1.26.3` directive was stricter than needed). Now the test suites:All green. Now let me probe the suspected reliability holes with adversarial programs. First: a function that doesn't return on every path.Both confirmed — valid Trux programs that leak raw C compiler errors. Next: argument evaluation order with side-effecting expressions.Division by zero is a raw SIGFPE — no trux error despite the language bounds-checking everything else. Now the subtle one: arena/ownership correctness across calls (dangling views):Ownership analysis holds up. One string-borrowing probe, then performance:Solid project — and the part I expected to be broken (the arena/ownership model) actually survived everything I threw at it. I cloned the repo, built it (note: your `go.mod` says `go 1.26.3` but it compiles fine on 1.22 — you're demanding a newer toolchain than you use), ran both test suites (all green), and then probed the generated C with adversarial programs. The real problems are elsewhere. Here's what I found, all verified by running code.

## Reliability: four confirmed bugs

**1. No return-path analysis → undefined behavior.** This compiles and prints stack garbage:

```go
func pick(flag bool) int {
  if flag {
    return 42
  }
}
```

`pick(false)` printed `64` on my machine. Codegen declares `trux_return_value` uninitialized and falls through to the `trux_return:` label. Fix it in `types.Check` with a standard "block terminates" analysis (a return statement terminates; an `if` terminates only if both branches do and an `else` exists), and add a codegen backstop — emit `rt_runtime_fail("missing return")` just before the label so any future checker gap becomes a clean crash instead of UB.

**2. Evaluation order is unspecified.** This is the classic transpiler-to-C bug:

```go
print(sub(io.readInt(), io.readInt()))   // input: 10 then 3
```

prints **-7**. gcc evaluated the arguments right-to-left, so the *first* parameter received the *second* line of input. C leaves argument and binary-operand order unspecified, and your codegen emits nested expressions directly, so Trux inherits that. Your binary-expr probe happened to go left-to-right on my gcc — which is worse, because it means behavior varies by compiler and optimization level. The fix is to lower side-effecting subexpressions (calls, reads) into ordered temporaries, best done as a normalization pass in `ir.Build`: `f(g(), h())` becomes `let t0 = g(); let t1 = h(); f(t0, t1)`. Doing it in the IR also fixes kernels for free and removes the need for the `trimParens`/`emitCondition` string surgery, which is currently safe but is exactly the kind of pattern-matching-on-emitted-C that breaks silently later.

**3. Integer division is unguarded.** `10 / 0` dies with a raw SIGFPE and no trux error message; `INT64_MIN / -1` is also UB. It's inconsistent — you bounds-check every index and overflow-check every allocation, then leave the one arithmetic op that traps completely bare. Route `/` through a helper:

```c
static inline int64_t rt_div_int(int64_t a, int64_t b) {
    if (b == 0) rt_runtime_fail("integer division by zero");
    if (a == INT64_MIN && b == -1) rt_runtime_fail("integer division overflow");
    return a / b;
}
```

Once you add `-O2` (see below) this inlines to a perfectly-predicted compare-and-branch.

**4. Function-name mangling collides with codegen internals.** `func frame()`, `func ctx()` — valid Trux — fail with leaked C compiler errors, because `mangleFunc` only prepends `trux_` while your generated locals live in the same namespace (`trux_frame` shadows the function `trux_frame`). Worse, `func kernel_scale()` alongside `kernel func scale` emits two conflicting declarations of `trux_kernel_scale` (I verified: lines 1201 and 1203 of the emitted source, one `int64_t(...)`, one `__global__ void(...)`). You already solved this correctly for variables with `trux_v_%d_%s` — apply the same scheme to functions (`trux_f_%d_%s`) and kernels (`trux_k_%d_%s`), and the internal names can never collide again.

One thing that *passed*: I tried hard to get a dangling arena pointer — returning a frame-backed slice through nested calls, returning a borrowed view of a parameter that was itself a frame-local temporary, `id(msg + "!")` in return position — and the ownership analysis copied out correctly every time. That's the hard part of this design and it works.

## Performance: two big wins, both measured

**Add `-O2`. It's an 18× speedup.** `compileGenerated` passes only `-std=c11`, so everything runs at `-O0`. Your runtime's whole design — tiny `static` helpers for every get/set/alloc — only works when the C compiler inlines them; at `-O0` every `xs[i]` is a real function call with a real bounds-check call inside it. My array-sum benchmark: 198 ms at `-O0`, 11 ms at `-O2`. One line. Keep `-O0` behind your existing `DEBUG=1` path if you want debuggable C.

**Pool the frame-arena chunks.** Every function that touches dynamic memory mallocs a 4 KB chunk on first allocation and frees it at `trux_return`. For a hot string-returning function called 2M times, that's 165 ms at `-O2`; patching the generated C to reuse the chunk drops it to 64 ms (2.6×). The right fix isn't a static arena (that breaks recursion) — it's a chunk freelist on `rt_context`: `rt_arena_deinit` donates chunks to the pool, `rt_arena_new_chunk` checks the pool first. Frame init/teardown becomes a few pointer moves with zero allocator traffic in steady state.

**Loop temporaries accrete.** That same 2M-iteration benchmark peaks at **33 MB RSS** for ~2 bytes of live data: each iteration clones the returned string into the *caller's* frame arena, which only resets when the caller returns. You already wrote `rt_arena_mark_current` / `rt_arena_rewind` — codegen never calls them. Mark at the top of a loop body and rewind at the bottom whenever the body's frame allocations don't escape the iteration (your ownership info can tell you when an allocation is bound to a variable declared outside the loop, which is the case where rewinding is unsafe). This is more delicate than the first two, so I'd do it last, but it's the difference between O(live data) and O(iterations) memory.

Smaller items: each `list[T]` creation costs three mallocs (registry node + struct + data) — the struct could live in the arena with only the data pointer registered; `rt_read_file` double-buffers (stat the file, allocate once in the arena, single `fread`); arena chunks never grow geometrically, so large frames malloc many 4 KB chunks.

## Simplifying the Go

Three changes would shrink `internal/codegen/c/c.go` (1,486 lines) by roughly half:

**Delete `funcUsage` entirely.** Its only purpose is deciding whether to emit `(void)trux_ctx;` / `(void)trux_result_arena;`, and that's why you render the body into a second buffer and thread `usage *funcUsage` through eight function signatures. Casting a parameter to void is valid C *whether or not it's used* — emit both casts unconditionally at the top of every function and the tracking struct, the `noteArena` calls, the double buffer, and a parameter on every emit function all disappear.

**Kill the 293-line `collect*NestedCollectionFamilies` walker.** It's a full hand-rolled IR traversal that exists only to discover which nested element types appear, and it must be extended every time you add an IR node (a guaranteed future bug — forget one case and you get a C compile error about a missing `rt_list_slice_int` typedef). `ir.Build` already visits every node exactly once: record types into a set on `ir.Program` as you construct nodes, and codegen just reads `program.UsedTypes`. Alternatively, write one generic `ir.Walk(node, visit)` that future passes (your eval-order lowering, for instance) can reuse.

**Stop returning errors for unreachable states.** 99 `if err != nil` blocks in this file, and nearly all guard cases like "unsupported type" that the type checker has already rejected — by the time IR reaches codegen, those are internal compiler errors, not user errors. The Go compiler's own backend `panic`s on impossible states; do the same (`panic("compiler bug: unsupported type " + ...)`), with one `recover` in `Generate` that wraps it as an error if you want graceful CLI output. Real logic becomes visible again. This is a taste call — some people prefer explicit errors everywhere — but for a post-typecheck backend the panic convention is the established one, and it's the single biggest readability win available here.

Minor: `emitKernelPrototype` and `emitKernelFunc` duplicate the signature loop — give kernels the same `prototype bool` treatment as `emitSignature`; `emitCondition` reimplements `trimParens` inline (moot if you do the temporaries lowering); and one forward-looking trap: you emit the `for` post-statement at the end of the body, which will silently break the day you add `continue` — the usual fix is a `goto`-able continue-label before the post statement.

If you want a priority order: `-O2` (one line, 18×), function-name mangling (small, fixes compile failures), missing-return check, division guards, then the eval-order lowering (the most work, but it's a semantic hole), then chunk pooling, and the loop-rewind last.