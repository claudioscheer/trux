# GPU Programming

Trux has an initial CUDA-backed GPU path. It is intentionally small: device memory is explicit, kernels are written in Trux source, and the compiler uses `nvcc` only for programs that declare kernels.

## Requirements

GPU programs require:

- an NVIDIA CUDA-capable GPU and driver
- `nvcc` on `PATH`, or `NVCC=/path/to/nvcc`

Non-GPU programs still compile with `cc` or `$CC`.

## Device Buffers

GPU memory is represented with `gpu.Buffer[T]`, where `T` is currently `int` or `float`.

```trux
import "gpu"

let n int = 1024
let host []int = make([]int, n)
let device gpu.Buffer[int] = gpu.from(host)

gpu.to(device, host)
gpu.free(device)
```

`gpu.from` allocates a device buffer and copies the host slice in. `gpu.to` copies it back. `gpu.alloc`, `gpu.copyToDevice`, and `gpu.copyToHost` remain for a buffer that should start empty, or for a copy that is separate from allocation. Device buffers are not host slices. Host code cannot index a `gpu.Buffer`.

Multi-step kernels should keep the device buffers and launch again, instead of copying on every step.

## Kernels

Kernels are declared with `kernel func` and do not return values. For `gpu.launch1d`, the first parameter is the element index. For `gpu.launch2d`, the first parameter is the row and the second is the column. Threads outside the launch extents do nothing, so the body does not repeat that check.

```trux
kernel func fill(i int, out gpu.Buffer[int]) {
  out[i] = 42
}

kernel func paint(row int, col int, out gpu.Buffer[int], width int) {
  out[row * width + col] = row + col
}
```

Kernel code is restricted. It supports scalar locals, arithmetic, comparisons, `if`, `for`, GPU buffer indexing, and GPU coordinate helpers. It does not support strings, lists, host slices, allocation, `print`, IO, time calls, or normal function calls. The one exception is `exp(x)`, which takes a `float` and returns a `float`, and only inside a kernel.

## Launching

```trux
let device gpu.Buffer[int] = gpu.from(host)
gpu.launch1d(fill, n, device)
gpu.launch2d(paint, width, height, device, width)
gpu.to(device, host)
gpu.free(device)
```

A 1D launch uses 256-thread blocks. A 2D launch uses 16 by 16 blocks. `gpu.launch` is still available when the grid and block need to be chosen by hand:

```trux
kernel func fillAt(out gpu.Buffer[int], n int) {
  let i int = gpu.globalX()
  if i < n {
    out[i] = 42
  }
}

gpu.launch(fillAt, gridX, 1, 1, 256, 1, 1, out, n)
gpu.sync()
```

That form does not inject coordinates. The kernel reads them itself.

Coordinate helpers inside kernels:

```trux
gpu.globalX()
gpu.globalY()
gpu.globalZ()
gpu.threadX()
gpu.threadY()
gpu.threadZ()
gpu.blockX()
gpu.blockY()
gpu.blockZ()
gpu.blockDimX()
gpu.blockDimY()
gpu.blockDimZ()
```

`gpu.to` and `gpu.copyToHost` wait until the copied data is back on the host. `gpu.sync()` is available when source code needs an explicit synchronization point, such as timing a kernel.

Integer arithmetic inside a kernel uses the same overflow and division checks as host code. On device, a failing check traps the kernel. That trap does not print a Trux runtime error message.

## Models

`gpu.mount(path)` reads a TXM1 file, described in [TXM.md](TXM.md). Each tensor is uploaded once. Later requests use `gpu.floats`, `gpu.ints`, `gpu.dim`, `gpu.scalar`, `gpu.hostFloats`, and `gpu.hostInts`. Those lookups do not call CUDA and do not take a lock. A second `gpu.mount` fails.

`gpu.workspaceFloat(name, n)` and `gpu.workspaceInt(name, n)` allocate a device buffer and a pinned host mirror for each worker. Call them before `http.listen`. `gpu.borrowFloat` and `gpu.borrowInt` return the buffer for the worker that is running the handler. `gpu.free` fails if the pointer is a mounted tensor or a workspace.

`gpu.copyToDevice` into a workspace on an HTTP worker copies into the pinned mirror and then queues the device copy on that worker's stream. It returns without waiting. `gpu.to` outside that path still waits.

`gpu.finish(request, continuation)` is for an HTTP handler. The continuation is `func(request int) int`. The runtime queues a copy of the workspaces back to the pinned mirrors, then returns the worker to its `io_uring` loop. Other sockets on that worker still accept and receive. The continuation runs after the kernel finishes. `gpu.ready(name)` is the pinned `[]float` to read there. The continuation sends the HTTP response. One GPU request is in flight on each worker. Request batching is not done.

If the kernel fails, the ring does not wait forever. A timeout queries the stream and the request gets a 500.

`examples/projects/digits_server` mounts a digits SVM and serves one image per request on this path. `benchmarks/digits` times that request path. `make bench-digits` needs Linux, liburing, and CUDA. It is not part of `make bench`.

## Example

`examples/gpu_matrix_multiply.tx` runs square matrix multiplication twice in the same Trux program: once on the CPU and once with `gpu.launch2d`. `examples/performance_stencil.tx` repeats a Jacobi step on two device buffers. Both print CPU and GPU checksums, a mismatch count, and elapsed time from the `time` package.
