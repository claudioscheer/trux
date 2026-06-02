# Arenas as the Memory Model

Trux uses arena-based allocation for dynamic memory.

Generated programs have a durable arena and compiler-managed function-frame arenas:

- **durable arena**: values that must survive across function calls or for the lifetime of the program/context
- **function-frame arena**: scratch and frame-owned memory used inside one function call

Functions are one-way unless a parameter explicitly opts into caller-visible mutation:

- ordinary parameters are inputs
- `mut` collection parameters may mutate caller-owned arrays, slices, and lists
- the return value is the output
- functions cannot mutate caller-owned containers through non-`mut` parameters

Function returns are either:

- **borrowed**: a parameter value or parameter-backed slice/view
- **owned**: a value that remains valid after the callee returns, because it was copied into `trux_result_arena`, already lives in the durable arena, or was frame-owned and copied into `trux_result_arena` during return

Local temporary dynamic values allocate in the current function-frame arena when possible.

Local collections use function-frame memory:

- array literals allocate in the current function-frame arena
- list literals allocate in the current function-frame arena
- `make([]T, n)` allocates backing storage in the current function-frame arena
- if local frame-backed data is returned, it is copied into `trux_result_arena`

Slices are views:

- returning a slice of parameter-owned data is okay
- returning a slice of local frame-backed data requires copying into `trux_result_arena`

Mutable parameters are an ownership escape hatch:

- `func push(mut xs list[int], value int) int` can append to the caller's list
- `func setFirst(mut xs []int, value int) int` can write through a caller-backed slice
- assigning a new collection to the parameter name itself is still local rebinding, not caller replacement
- dynamic values appended through a `mut` list parameter use durable memory when needed

`clone(x)` creates an owned dynamic value in the selected ownership target for the expression context:

- `return clone(xs[:])` allocates directly in `trux_result_arena`
- `let ys []int = clone(xs[:])` allocates in the current function-frame arena
- returning a frame-owned dynamic value copies it into `trux_result_arena`

For collections containing strings, `clone` deep-copies each string element.

There is no per-object `free`. Arena memory is reclaimed by rewinding, resetting, or destroying the arena.
