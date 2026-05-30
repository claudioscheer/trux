# Arenas as the Memory Model

Trux uses arena-based allocation for dynamic memory.

Generated programs have two arenas:

- **durable arena**: values that must survive across function calls
- **temp arena**: scratch memory used inside functions

Functions are one-way:

- parameters are inputs
- return value is the output
- functions cannot mutate caller-owned containers

Local temporary dynamic values allocate in the temp arena when possible.

Function returns are either:

- **borrowed**: a parameter value or parameter-backed slice/view
- **owned**: data copied into the result arena, data already owned by the durable arena, or data produced by `clone`

Local collections use function scratch memory:

- array literals allocate in `trux_ctx->temp`
- list literals allocate in `trux_ctx->temp`
- `make([]T, n)` allocates backing storage in `trux_ctx->temp`
- if local scratch-backed data is returned, it is copied into `trux_result_arena`

Slices are views:

- returning a slice of parameter-owned data is okay
- returning a slice of local temp data requires copying

`clone(x)` creates an owned dynamic value in the selected ownership target for the expression context:

- `return clone(xs[:])` allocates in `trux_result_arena`
- `let ys []int = clone(xs[:])` allocates in `trux_ctx->arena`

For collections containing strings, `clone` deep-copies each string element.

There is no per-object `free`. Arena memory is reclaimed by rewinding, resetting, or destroying the arena.
