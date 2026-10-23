---
title: "gpu"
description: "Device buffers, copies, and kernel launches."
icon: "developer_board"
weight: 500
---

```trux
import "gpu"
```

**What it is.** CUDA device buffers, host/device copies, kernel launches, and thread coordinates.

**Why.** It is the whole interface between host code and the `kernel func`s you write.

**When you don't need it.** For CPU-only programs. A program that uses `gpu` buffers must also declare a kernel; see the
[guide](/docs/guide/gpu/#device-buffers).

This page is the reference. For explanations, examples, and common mistakes, read
[GPU programming](/docs/guide/gpu/).

## Types

| Type | Description |
| --- | --- |
| `gpu.Buffer[int]` | device memory holding 64-bit integers (`int64_t*` in CUDA) |
| `gpu.Buffer[float]` | device memory holding doubles (`double*` in CUDA) |

## Host functions

| Call | Returns | Description |
| --- | --- | --- |
| `gpu.from(host []T)` | `gpu.Buffer[T]` | Allocates a device buffer of `len(host)` elements and copies `host` into it. |
| `gpu.alloc(n int)` | `gpu.Buffer[T]` | Allocates `n` elements. Must be the value of a `let` with an explicit `gpu.Buffer[T]` type. |
| `gpu.to(device, host []T)` | | Copies the device buffer into the host slice and waits for the copy (statement only). |
| `gpu.copyToDevice(host []T, device)` | | Copies host to device into an existing buffer (statement only). Note the argument order. |
| `gpu.copyToHost(device, host []T)` | | Same as `gpu.to` (statement only). |
| `gpu.launch1d(kernel, n int, args...)` | | Runs `kernel` for indexes `0`..`n-1`; the kernel's first `int` parameter receives the index. Blocks of 256 threads (statement only). |
| `gpu.launch2d(kernel, width int, height int, args...)` | | Runs `kernel` over a `height` x `width` grid; the first two `int` parameters receive row and column. 16 x 16 blocks (statement only). |
| `gpu.launch(kernel, gridX, gridY, gridZ, blockX, blockY, blockZ, args...)` | | Launches with an explicit CUDA grid and block. No coordinates are injected and no bounds check is added (statement only). |
| `gpu.sync()` | | Host-side `cudaDeviceSynchronize`: waits until all queued GPU work has finished (statement only). Not a barrier between kernel threads; see [`gpu.sync`](/docs/guide/gpu/#gpu-sync). |
| `gpu.free(device)` | | Frees a device buffer (statement only). Fails for a mounted tensor or a workspace. |
| `gpu.mount(path string)` | | Loads a TXM1 file once and copies each tensor to the device (statement only). |
| `gpu.workspaceFloat(name string, n int)` | | Allocates one reused float workspace per worker (statement only). Call it before `http.listen`. |
| `gpu.workspaceInt(name string, n int)` | | Same for `int` (statement only). |
| `gpu.floats(name string)` | `gpu.Buffer[float]` | Device memory of a mounted float tensor. |
| `gpu.ints(name string)` | `gpu.Buffer[int]` | Device memory of a mounted int tensor. |
| `gpu.dim(name string, axis int)` | `int` | One dimension of a mounted tensor. |
| `gpu.scalar(name string)` | `float` | The value of a one-element float tensor. |
| `gpu.hostFloats(name string)` | `[]float` | Host copy of a mounted float tensor. |
| `gpu.hostInts(name string)` | `[]int` | Host copy of a mounted int tensor. |
| `gpu.borrowFloat(name string)` | `gpu.Buffer[float]` | This worker's float workspace. |
| `gpu.borrowInt(name string)` | `gpu.Buffer[int]` | This worker's int workspace. |
| `gpu.ready(name string)` | `[]float` | Pinned host mirror of a float workspace. Read it from the `gpu.finish` continuation. |
| `gpu.finish(request int, continuation)` | | Queues the copy back and runs `continuation` when the GPU work finishes (statement only). |

`T` is `int` or `float`, and host and device element types must match. The first argument of a launch must be the name of
a `kernel func`, and the remaining arguments must match the kernel's (remaining) parameters exactly.

## Coordinate helpers

Use these **inside a kernel**; they all return `int`. The compiler currently does not stop you from calling them in
host code, where the program then fails to build with a C compiler error such as
`error: 'blockIdx' undeclared` ([details](/docs/guide/gpu/#coordinate-helpers)).

| Call | CUDA equivalent |
| --- | --- |
| `gpu.globalX()`, `gpu.globalY()`, `gpu.globalZ()` | `blockIdx.* * blockDim.* + threadIdx.*` |
| `gpu.threadX()`, `gpu.threadY()`, `gpu.threadZ()` | `threadIdx.x`, `.y`, `.z` |
| `gpu.blockX()`, `gpu.blockY()`, `gpu.blockZ()` | `blockIdx.x`, `.y`, `.z` |
| `gpu.blockDimX()`, `gpu.blockDimY()`, `gpu.blockDimZ()` | `blockDim.x`, `.y`, `.z` |

`gpu.sync`, `gpu.free`, `gpu.launch*`, and the copy and allocation calls cannot be used inside kernels.

## Errors

CUDA API failures stop the program with `trux runtime error: CUDA <operation> failed: <CUDA error text>`, where
`<operation>` is `alloc`, `copyToDevice`, `copyToHost`, `launch`, `sync`, or `free`. Copying a host slice longer than
the device buffer fails with `gpu copyToHost host slice is larger than device buffer` (or `copyToDevice`). Faults inside
a running kernel are reported by the next `gpu.sync` or copy back; see [Kernel errors](/docs/guide/gpu/#kernel-errors).
These messages come from the generated runtime code; they were not produced on a GPU while writing these docs.
