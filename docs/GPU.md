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

Kernel code is restricted. It supports scalar locals, arithmetic, comparisons, `if`, `for`, GPU buffer indexing, and GPU coordinate helpers. It does not support strings, lists, host slices, allocation, `print`, IO, time calls, or normal function calls.

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

## Example

`examples/gpu_matrix_multiply.tx` runs square matrix multiplication twice in the same Trux program: once on the CPU and once with `gpu.launch2d`. `examples/performance_stencil.tx` repeats a Jacobi step on two device buffers. Both print CPU and GPU checksums, a mismatch count, and elapsed time from the `time` package.
