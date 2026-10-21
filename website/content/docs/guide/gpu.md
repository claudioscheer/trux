---
title: "GPU programming"
description: "Writing CUDA kernels in Trux and launching them on the GPU."
icon: "developer_board"
weight: 1100
---

Trux can run code on NVIDIA GPUs through CUDA. You write the GPU code (a **kernel**) in Trux, and the compiler emits it as
a CUDA `__global__` function.

{{% alert context="warning" %}}

**Requirements**

GPU programs need an NVIDIA CUDA-capable GPU and driver, and the CUDA toolkit's `nvcc` on your `PATH` (or set
`NVCC=/path/to/nvcc`). As soon as a program declares a `kernel func`, the whole program is compiled with `nvcc`
instead of `cc`. Programs without kernels are unaffected.

{{% /alert %}}

## The shape of a GPU program

```trux
package main

import "gpu"

kernel func fill(i int, out gpu.Buffer[int]) {
  out[i] = i * 2
}

func main() int {
  let n int = 8
  let host []int = make([]int, n)

  let device gpu.Buffer[int] = gpu.from(host)   // allocate on the GPU and copy host -> device
  gpu.launch1d(fill, n, device)                 // run fill for i = 0 .. n-1
  gpu.to(device, host)                          // copy device -> host (waits for the GPU)
  gpu.free(device)                              // device memory is freed explicitly

  print(host[0], " ", host[7])
  return 0
}
```

Four steps, always the same:

1. **Move data to the device** with `gpu.from(slice)`, or create an empty buffer with `gpu.alloc(n)`.
2. **Launch a kernel** with `gpu.launch1d` or `gpu.launch2d`.
3. **Copy results back** with `gpu.to(buffer, slice)`.
4. **Free device buffers** with `gpu.free(buffer)`.

## Device buffers

`gpu.Buffer[int]` and `gpu.Buffer[float]` are handles to GPU memory. They are not host collections: host code cannot
index them or take their `len`. Only `int` and `float` elements are supported.

```trux
let a gpu.Buffer[float] = gpu.from(hostValues)   // size and contents from a []float slice
let out gpu.Buffer[float] = gpu.alloc(total)     // total elements, contents unspecified
```

`gpu.alloc` must be used directly as the value of a `let` with an explicit `gpu.Buffer[...]` type, so the compiler knows
the element type. Host slices passed to `gpu.from`, `gpu.to`, and the lower-level copy calls must be `[]int` or
`[]float` (use `xs[:]` to pass an array).

For multi-step algorithms, keep data on the device and launch repeatedly instead of copying back after every step.
Buffer variables can be reassigned, which makes ping-pong buffers easy (from `examples/performance_stencil.tx`):

```trux
for let step int = 0; step < steps; step = step + 1 {
  gpu.launch2d(jacobiKernel, width, height, dCurrent, dNext, width, height)
  let swap gpu.Buffer[int] = dCurrent
  dCurrent = dNext
  dNext = swap
}
```

## Writing kernels

```trux
kernel func paint(row int, col int, out gpu.Buffer[int], width int) {
  out[row * width + col] = row + col
}
```

- Kernels are declared with `kernel func` and have **no return type** and no `return` statement.
- Parameters may be `int`, `float`, `bool`, or `gpu.Buffer[...]`. Parameters cannot be `mut`.
- Inside a kernel you can use `int`, `float`, and `bool` locals, arithmetic, comparisons, `&&`, `||`, `!`, `if`, `for`,
  `break`, `continue`, buffer indexing and element assignment, and the [coordinate helpers](/docs/stdlib/gpu/#coordinate-helpers).
- Kernels **cannot** use strings, arrays, slices, lists, `make`, `print`, `assert`, IO, `time`, or call other functions.
- Integer arithmetic is still overflow- and zero-division-checked. On the device a failed check traps the kernel, and
  that trap does not print a Trux runtime error message.

## Launching

### 1D: `gpu.launch1d(kernel, n, args...)`

The kernel's **first parameter** must be an `int`. It receives the element index, from `0` to `n - 1`. The remaining
arguments are passed to the remaining parameters. Threads past `n` do nothing, so the kernel doesn't need its own bounds
check. The launch uses 256-thread blocks.

### 2D: `gpu.launch2d(kernel, width, height, args...)`

The kernel's **first two parameters** must be `int`: they receive the **row** (`0` to `height - 1`) and the **column**
(`0` to `width - 1`). The launch uses 16 x 16 blocks.

{{< include "examples/gpu_matrix_multiply.tx:15:23" "trux" >}}

### Manual: `gpu.launch(kernel, gridX, gridY, gridZ, blockX, blockY, blockZ, args...)`

Use this when you want to pick the grid and block sizes yourself. No coordinates are injected and no bounds check is
added; the kernel reads its position with helpers such as `gpu.globalX()` and checks bounds itself:

```trux
kernel func fillAt(out gpu.Buffer[int], n int) {
  let i int = gpu.globalX()
  if i < n {
    out[i] = 42
  }
}

// ...
gpu.launch(fillAt, gridX, 1, 1, 256, 1, 1, out, n)
gpu.sync()
```

A given kernel must always be launched with the same shape (1D, 2D, or manual) within a program.

## Synchronization

`gpu.to` and `gpu.copyToHost` wait until the data is back on the host. Call `gpu.sync()` when you need an explicit wait,
for example to time only the kernel.

## Restrictions

- A program cannot use both `http` and GPU kernels yet.
- Kernels must be declared in the program (there is no way to load precompiled CUDA code).

## Examples

| Example | What it does |
| --- | --- |
| `examples/gpu_matrix_multiply.tx` | square matrix multiply on CPU and with `gpu.launch2d`, then compares results |
| `examples/performance_stencil.tx` | repeated Jacobi stencil steps on two device buffers |
| `examples/performance_cnn.tx` | a four-layer float convolution network with `gpu.launch1d` |
| `examples/projects/rotate_image` | rotates a PPM image on CPU and GPU |

Each prints CPU and GPU checksums, a mismatch count, and timings.

See the [`gpu` package reference](/docs/stdlib/gpu/) for every call, and [GPU design](/docs/design-notes/gpu/) for the design notes.
