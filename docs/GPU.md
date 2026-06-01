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
let device gpu.Buffer[int] = gpu.alloc(n)

gpu.copyToDevice(host, device)
gpu.copyToHost(device, host)
gpu.free(device)
```

Device buffers are not host slices. Host code cannot index a `gpu.Buffer`; it must copy data back to a host slice first.

## Kernels

Kernels are declared with `kernel func` and do not return values.

```trux
kernel func fill(out gpu.Buffer[int], n int) {
  let i int = gpu.globalX()

  if i < n {
    out[i] = 42
  }
}
```

Kernel code is restricted. It supports scalar locals, arithmetic, comparisons, `if`, `for`, GPU buffer indexing, and GPU coordinate helpers. It does not support strings, lists, host slices, allocation, `print`, IO, time calls, or normal function calls.

## Launching

Launch geometry is explicit and mirrors CUDA's grid/block model:

```trux
gpu.launch(fill, gridX, gridY, gridZ, blockX, blockY, blockZ, out, n)
gpu.sync()
```

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

`gpu.copyToHost` is synchronous for the copied data. `gpu.sync()` is available when source code needs an explicit synchronization point, such as timing a kernel.

## Example

`examples/gpu_matrix_multiply.tx` runs square matrix multiplication twice in the same Trux program: once on the CPU and once with a Trux GPU kernel. It computes launch geometry from `n`, then prints CPU and GPU checksums, mismatch count, and elapsed time from the `time` package.
