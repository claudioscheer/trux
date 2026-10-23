---
title: "GPU programming"
description: "Kernels, device buffers, launches, coordinate helpers, and gpu.sync."
icon: "developer_board"
weight: 1100
---

Trux can run code on NVIDIA GPUs through CUDA. You write the GPU code, a **kernel**, in Trux; the compiler emits it as a
CUDA `__global__` function and builds the program with `nvcc`.

## Requirements {#requirements}

GPU programs need an NVIDIA CUDA-capable GPU and driver, and the CUDA toolkit's `nvcc` on your `PATH` (or
`NVCC=/path/to/nvcc`). As soon as a program declares a `kernel func`, the whole program is compiled with `nvcc` instead
of `cc`. Without `nvcc` the build stops with:

```text
nvcc failed: exec: "nvcc": executable file not found in $PATH
```

{{% alert context="info" %}}

**What was run for this page**

The compile errors below were produced by the real compiler. The machine that records them has no GPU or `nvcc`, so
behavior on the device is explained from the generated CUDA code shown here, and every such part is marked
**not executed**.

{{% /alert %}}

## The shape of a GPU program

Every GPU program follows the same four steps:

```trux
package main

import "gpu"

kernel func fill(i int, out gpu.Buffer[int]) {
  out[i] = i * 2
}

func main() int {
  let n int = 8
  let host []int = make([]int, n)

  let device gpu.Buffer[int] = gpu.from(host)   // 1. allocate on the GPU and copy host -> device
  gpu.launch1d(fill, n, device)                 // 2. run fill for i = 0 .. n-1
  gpu.to(device, host)                          // 3. copy device -> host (waits for the GPU)
  gpu.free(device)                              // 4. free the device memory

  print(host[0], " ", host[7])
  return 0
}
```

On a CUDA machine this prints `0 14` (**not executed** here).

## Kernels {#kernels}

**What it is.** A `kernel func` is a function that runs on the GPU, once per thread, in parallel.

**Why.** Data-parallel work, such as one output cell per thread in a matrix multiply, runs on thousands of GPU threads at
once.

**How to use it.**

{{< include "examples/gpu_matrix_multiply.tx:15:23" "trux" >}}

- No return type and no `return` with a value; a kernel's results are what it writes into buffers.
- Parameters may be `int`, `float`, `bool`, or `gpu.Buffer[...]`, and cannot be `mut`.
- Inside a kernel you can use `int`, `float`, and `bool` locals, arithmetic, comparisons, `&&`, `||`, `!`, `if`, `for`,
  `break`, `continue`, buffer indexing and element assignment, and the [coordinate helpers](#coordinate-helpers).
- Kernels **cannot** use strings, arrays, slices, lists, `make`, `print`, `assert`, `io`, `time`, or call any function.

**Common mistakes.** Declaring a return type; the parser expects the body:

{{< include "examples/errors/gpu_kernel_return.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_kernel_return.txt" "text" >}}

Returning a value:

{{< include "website/outputs/errors/gpu_kernel_return_value.txt" "text" >}}

Calling a function, including `print`:

{{< include "examples/errors/gpu_kernel_call.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_kernel_call.txt" "text" >}}

{{< include "website/outputs/errors/gpu_kernel_print.txt" "text" >}}

Write the computation inline. A local of an unsupported type:

{{< include "website/outputs/errors/gpu_kernel_string.txt" "text" >}}

**Indexing out of range (not executed).** Buffer indexing inside a kernel is **not** bounds-checked. The generated code
indexes the raw device pointer, `trux_v_3_out[...] = trux_v_3_sum` above, with no length check. The bounds check that
`launch1d` and `launch2d` add only compares the thread's coordinates with the extents you passed, not with the buffer
size. Launching with `n` larger than the buffer, or computing a wrong index, writes outside the buffer: undefined
behavior in CUDA, usually reported later as a failed CUDA call or silently wrong results.

**When you don't need it.** Small or sequential work is faster on the CPU: every GPU run pays for copies to and from the
device. The examples print CPU and GPU timings side by side so you can compare.

**Related:** [Launches](#launches), [Checked integer arithmetic](/docs/guide/operators/#checked-integer-arithmetic).

## Device buffers {#device-buffers}

**What it is.** `gpu.Buffer[int]` and `gpu.Buffer[float]` are handles to GPU memory, created from a host slice or empty.

**Why.** Kernels can only read and write device memory, so data has to be copied there and back.

**How to use it.**

```trux
let a gpu.Buffer[float] = gpu.from(hostValues)   // len(hostValues) elements, copied from the host
let out gpu.Buffer[float] = gpu.alloc(total)     // total elements, contents unspecified
gpu.to(out, hostResult)                          // copy back into a []float
gpu.free(a)
gpu.free(out)
```

- Host slices must be `[]int` or `[]float`, matching the buffer's element type. Pass an array as `xs[:]`.
- `gpu.alloc` must be the value of a `let` with an explicit `gpu.Buffer[...]` type.
- Host code cannot index a buffer or take its `len`.
- Buffers are not in an arena: free each one with `gpu.free`.
- For multi-step algorithms, keep data on the device and swap buffer variables between launches, as
  `examples/performance_stencil.tx` does:

```trux
for let step int = 0; step < steps; step = step + 1 {
  gpu.launch2d(jacobiKernel, width, height, dCurrent, dNext, width, height)
  let swap gpu.Buffer[int] = dCurrent
  dCurrent = dNext
  dNext = swap
}
```

**Common mistakes.** Using `gpu.alloc` anywhere but a typed `let`:

{{< include "examples/errors/gpu_alloc_untyped.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_alloc_untyped.txt" "text" >}}

Reading a buffer on the host; copy it back with `gpu.to` first:

{{< include "examples/errors/gpu_buffer_host_index.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_buffer_host_index.txt" "text" >}}

Mixing element types:

{{< include "website/outputs/errors/gpu_element_type.txt" "text" >}}

**Using buffers in a program without kernels (compiler bug).** The type checker accepts it, but the program is built with
`cc`, which has no CUDA runtime, so the C compiler fails:

{{< include "examples/errors/gpu_buffers_without_kernel.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_buffers_without_kernel.txt" "text" >}}

Buffers are only useful with a kernel, so add one, or use host slices.

**Copying into a slice that is larger than the buffer (not executed).** `gpu.to`, `gpu.copyToHost`, and
`gpu.copyToDevice` check the sizes before copying:

{{< include "website/outputs/generated/gpu_sync_runtime.h:1:4" "cpp" >}}

A slice longer than the buffer stops the program with
`trux runtime error: gpu copyToHost host slice is larger than device buffer`. A shorter slice copies only its own
length.

**Forgetting `gpu.free` (not executed).** Nothing reports it. The memory stays allocated until the process exits, which
only matters for programs that allocate buffers in a loop.

**When you don't need it.** Host-only programs never need buffers. Inside a multi-step GPU algorithm, don't copy back
after every step; copy once at the end.

**Related:** [`gpu` reference](/docs/stdlib/gpu/), [Memory and ownership](/docs/guide/memory/).

## Launches {#launches}

**What it is.** A launch runs a kernel on a grid of GPU threads. Trux has three forms:

| Call | Threads | Coordinates |
| --- | --- | --- |
| `gpu.launch1d(kernel, n, args...)` | `n`, in blocks of 256 | first parameter receives `0` .. `n-1`; threads past `n` do nothing |
| `gpu.launch2d(kernel, width, height, args...)` | `width` x `height`, in 16 x 16 blocks | first two parameters receive **row** (`0` .. `height-1`) and **col** (`0` .. `width-1`); threads outside do nothing |
| `gpu.launch(kernel, gridX, gridY, gridZ, blockX, blockY, blockZ, args...)` | exactly the grid and block you give | none injected and no bounds check; use the [coordinate helpers](#coordinate-helpers) |

**Why.** `launch1d` and `launch2d` cover the common cases without grid arithmetic. `launch` is there when you need to
choose the block size yourself.

**How to use it.** Pass the kernel's name, the extents, and then the kernel's remaining arguments:

{{< include "examples/gpu_matrix_multiply.tx:103:113" "trux" >}}

The generated host code computes the grid from `n` and checks the launch:

{{< include "website/outputs/generated/matmul_launch.cu" "cpp" >}}

With `gpu.launch`, the kernel checks its own bounds:

```trux
kernel func fillAt(out gpu.Buffer[int], n int) {
  let i int = gpu.globalX()
  if i < n {
    out[i] = 42
  }
}

// ...
gpu.launch(fillAt, gridX, 1, 1, 256, 1, 1, out, n)
```

**Common mistakes.** A `launch1d` kernel whose first parameter is not the `int` index:

{{< include "examples/errors/gpu_launch1d_coordinate.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_launch1d_coordinate.txt" "text" >}}

Passing the wrong number of kernel arguments:

{{< include "website/outputs/errors/gpu_launch_arguments.txt" "text" >}}

Launching one kernel with two different shapes:

{{< include "examples/errors/gpu_launch_two_shapes.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_launch_two_shapes.txt" "text" >}}

Write a second kernel for the other shape.

**When you don't need it.** Prefer `launch1d` or `launch2d`; use `launch` only when the block size matters to you.

**Related:** [Kernels](#kernels), [`gpu.sync`](#gpu-sync).

## Coordinate helpers {#coordinate-helpers}

**What it is.** `gpu.globalX()`, `gpu.threadX()`, `gpu.blockX()`, `gpu.blockDimX()`, and their `Y` and `Z` forms return
the current thread's position as `int`. They map to CUDA's `blockIdx`, `blockDim`, and `threadIdx`.

**Why.** Kernels started with `gpu.launch` get no injected coordinates, so they read their position with these helpers.

**How to use it.** Call them inside a kernel, as `fillAt` does above. The full list is in the
[`gpu` reference](/docs/stdlib/gpu/#coordinate-helpers).

**Common mistakes.** Calling one in host code. This is a **compiler bug**: the type checker accepts it, and the C compiler
then fails because CUDA's thread variables do not exist on the host:

{{< include "examples/errors/gpu_coordinate_on_host.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_coordinate_on_host.txt" "text" >}}

That is the output with `cc`, because the program has no kernel. In a program that does have a kernel, the host code
goes to `nvcc` instead; what `nvcc` does with these names in host code was **not verified**, and the value would be
meaningless on the host anyway. Only call the helpers inside a `kernel func`.

**When you don't need it.** In `launch1d` and `launch2d` kernels, use the injected `i` or `row` and `col` parameters
instead.

## `gpu.sync` {#gpu-sync}

**What it is.** `gpu.sync()` makes the host wait until all GPU work queued so far has finished. It is a host-side call
to `cudaDeviceSynchronize`, not a barrier between threads inside a kernel:

{{< include "website/outputs/generated/gpu_sync_runtime.h:5:7" "cpp" >}}

**Why.** A launch only queues the kernel and returns at once; the host keeps running while the GPU works. `gpu.sync` is
how the host waits for that work without copying anything.

**How to use it.** The typical use is timing. In `examples/gpu_matrix_multiply.tx`, the sync sits between the launch and
the copy back so the timing includes the kernel:

{{< include "website/outputs/generated/matmul_sync.cu" "cpp" >}}

**Common mistakes.** Calling it inside a kernel:

{{< include "examples/errors/gpu_sync_in_kernel.tx" "trux" >}}

{{< include "website/outputs/errors/gpu_sync_in_kernel.txt" "text" >}}

**Expecting it to synchronize threads (not executed).** Trux has no in-kernel barrier (`__syncthreads`), no shared
memory, and no atomics. Threads of one launch run in no particular order, so a kernel that reads an element another thread
writes in the same launch is a data race. For example, updating a grid in place:

```trux
kernel func smoothInPlace(i int, grid gpu.Buffer[int], n int) {
  if i > 0 && i < n - 1 {
    grid[i] = (grid[i - 1] + grid[i] + grid[i + 1]) / 3   // neighbours may already be updated
  }
}
```

This compiles, but each thread may see its neighbours before or after they were updated, so results can change from run
to run. Several threads adding into one element, such as `out[0] = out[0] + values[i]`, lose updates the same way. Write
into a second buffer and swap them between launches, as `performance_stencil.tx` does. `gpu.sync` does not help here,
because the race is inside a single launch.

**Timing without it (not executed).** Without the sync, a timer read right after `gpu.launch2d` measures only how long
it took to queue the kernel, not to run it.

**When you don't need it.**

- **Before `gpu.to` or `gpu.copyToHost`.** They use a blocking `cudaMemcpy` (shown above), which waits for earlier
  kernels and for the copy itself.
- **Between launches.** Launches run in the order they were issued, so a kernel that reads another kernel's output can
  be launched right after it.
- **Before `gpu.free`.**

Every way of getting data back to the host waits for the GPU, so leaving `gpu.sync` out does not change results
(reasoned from the generated code above, **not executed**). Use it for timing and to make an error show up at a known
point.

**Related:** [Kernel errors](#kernel-errors), [`time`](/docs/stdlib/time/).

## Kernel errors {#kernel-errors}

**What it is.** How failures on the GPU reach your program.

**Why it matters.** Device code cannot print, so errors look different from host runtime errors.

**How it works (not executed).**

- Integer overflow and division by zero are still checked in kernels; a failed check executes a `trap` instruction
  instead of printing a message.
- A failed CUDA call stops the program with `trux runtime error: CUDA <operation> failed: <CUDA error text>`. The launch
  is checked right away (`cudaGetLastError`), which catches bad launch configurations.
- A fault while the kernel runs, such as a trap or an out-of-range write, is only reported by the **next** call that waits
  for the GPU: `gpu.sync` (`CUDA sync failed: …`) or a copy back (`CUDA copyToHost failed: …`). Put a `gpu.sync` right
  after a launch to find out which kernel failed.

## Models {#models}

**What it is.** A way to load a numeric model once, keep it on the device, and answer HTTP requests without blocking the worker on the GPU.

**Why it matters.** Copying weights on every request costs more than the arithmetic for a small model. The weights should move to the GPU at startup. Each request only copies its own input.

**How it works.** `gpu.mount` reads a TXM1 file. The byte layout is `docs/TXM.md` in the repository. The file is a list of named tensors, `int` or `float`, rank 1 to 4. The reader does not know what the names mean. The program does. `gpu.floats` and `gpu.ints` return the device memory. `gpu.dim` and `gpu.scalar` read the shape and a one-element float. `gpu.hostFloats` and `gpu.hostInts` are the host copy that stayed behind after the upload.

`gpu.workspaceFloat` and `gpu.workspaceInt` allocate a buffer for each worker, once, before `http.listen`. `gpu.borrowFloat` and `gpu.borrowInt` return that worker's buffer. A later request reuses it. `gpu.free` on a mounted tensor or a workspace fails.

Inside a kernel, `exp(x)` is the one call you can make. It takes a `float` and returns a `float`.

A request handler copies the input into the borrowed workspace, launches, then calls `gpu.finish(request, continuation)`. The continuation is `func(request int) int`. The worker goes back to the `io_uring` loop while the kernel runs, so other sockets on that worker still accept and receive. One request is in flight per worker. The continuation runs when the kernel finishes, reads `gpu.ready`, and sends the response. `examples/projects/digits_server` is the handwritten-digits SVM on that path. `benchmarks/digits` times it (`make bench-digits` on Linux with liburing and CUDA). The measured run is on the [HTTP + GPU](/docs/benchmarks/http-gpu/) page.

`gpu.to` outside an HTTP worker is still a blocking copy. Existing GPU examples keep using it.

## Restrictions

- Kernels must be written in Trux; there is no way to load precompiled CUDA code.

## Examples

| Example | What it does |
| --- | --- |
| `examples/gpu_matrix_multiply.tx` | square matrix multiply on CPU and with `gpu.launch2d`, then compares results |
| `examples/performance_stencil.tx` | repeated Jacobi stencil steps on two device buffers |
| `examples/performance_cnn.tx` | a four-layer float convolution network with `gpu.launch1d` |
| `examples/projects/rotate_image` | rotates a PPM image on CPU and GPU |

Each prints CPU and GPU checksums, a mismatch count, and timings. See the [`gpu` reference](/docs/stdlib/gpu/) for every
call and [GPU design](/docs/design-notes/gpu/) for the design notes.
