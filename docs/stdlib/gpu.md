# gpu

```trux
import "gpu"
```

Reference for the `gpu` package. For a tutorial-style introduction, read [GPU programming](../guide/gpu.md) first.

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
| `gpu.sync()` | | Waits until all queued GPU work has finished (statement only). |
| `gpu.free(device)` | | Frees a device buffer (statement only). |

`T` is `int` or `float`, and host and device element types must match. The first argument of a launch must be the name of
a `kernel func`, and the remaining arguments must match the kernel's (remaining) parameters exactly.

## Coordinate helpers

These may only be called **inside a kernel**. They all return `int`.

| Call | CUDA equivalent |
| --- | --- |
| `gpu.globalX()`, `gpu.globalY()`, `gpu.globalZ()` | `blockIdx.* * blockDim.* + threadIdx.*` |
| `gpu.threadX()`, `gpu.threadY()`, `gpu.threadZ()` | `threadIdx.x`, `.y`, `.z` |
| `gpu.blockX()`, `gpu.blockY()`, `gpu.blockZ()` | `blockIdx.x`, `.y`, `.z` |
| `gpu.blockDimX()`, `gpu.blockDimY()`, `gpu.blockDimZ()` | `blockDim.x`, `.y`, `.z` |

`gpu.sync`, `gpu.free`, `gpu.launch*`, and the copy and allocation calls cannot be used inside kernels.

## Errors

CUDA API failures (allocation, copies, launches) stop the program with `trux runtime error: CUDA <operation> failed: ...`.
