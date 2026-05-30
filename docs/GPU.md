# GPU Programming Model Idea

This document is a future design note. GPU support is not implemented in the compiler, parser, type checker, runtime, or examples.

The idea here is to preserve a possible direction: CPU and GPU have separate memory address spaces, and all data movement should be explicit.

## Recommended Abstraction

Use `gpu.Buffer[T]` as the primary type for device memory:

```trux
let d_data gpu.Buffer[int] = gpu.alloc(n)
```

`gpu.Buffer[T]` would own device memory and would be the only way to pass data to kernels.

## Basic Pattern: CPU to GPU to CPU

```trux
func main() int {
  let n int = 1024 * 1024

  // Data starts on the CPU.
  let h_data []int = make([]int, n)

  // Device memory is explicit.
  let d_data gpu.Buffer[int] = gpu.alloc(n)

  // Copies are explicit so cost is visible in source.
  gpu.copy_to_device(h_data, d_data)

  gpu.launch(fill, d_data, n)

  // Copy results back only when host code needs them.
  gpu.copy_to_host(d_data, h_data)

  print(h_data[0])

  return 0
}
```

## Defining a Kernel

Possible kernel syntax:

```trux
kernel func fill(out gpu.Buffer[int], n int) {
  let i int = gpu.global_id()

  if i < n {
    out[i] = 42
  }
}
```

Kernels would be restricted:

- no dynamic allocation
- limited language features
- operate on `gpu.Buffer` and primitive types only
- no host strings or complex host-owned values

## Launching a Kernel

```trux
gpu.launch(fill, d_data, n)
```

Launches would likely be asynchronous by default. Source code would need explicit synchronization when it needs completion before copying results back.

## Key Operations

| Operation | Description | Direction |
|-----------|-------------|-----------|
| `gpu.alloc(n)` | Allocate `n` elements on the device | none |
| `gpu.copy_to_device(h, d)` | Copy from CPU slice to `gpu.Buffer` | Host to device |
| `gpu.copy_to_host(d, h)` | Copy from `gpu.Buffer` to CPU slice | Device to host |
| `gpu.launch(kernel, ...)` | Launch a GPU kernel | none |
| `gpu.free(buf)` | Free device memory | none |

## Mixing CPU and GPU Work

The intended model allows CPU and GPU execution to be interleaved:

```trux
let input []int = load_data_on_cpu()
let device gpu.Buffer[int] = gpu.alloc(len(input))

gpu.copy_to_device(input, device)
gpu.launch(heavy_parallel_work, device, len(input))

let results []int = make([]int, 256)
gpu.copy_to_host_partial(device, results, 0, 256)

let final int = cpu_reduce(results)
save_results(final)
```

Guideline: only copy data back when the CPU actually needs it. Avoid round-tripping large buffers unnecessarily.

## Synchronization Rules

Likely default rules:

- `gpu.copy_to_host` is synchronous by default.
- Explicit `gpu.sync()` or stream-based async copies can be added later.
- Host code must not read from a host buffer that is the target of an unfinished copy.

## Design Decisions

- Explicit copies are the default model.
- Unified memory should only be offered later as an opt-in if it is needed.
- `gpu.Buffer[T]` owns its memory. It is not a view.
- Kernels operate on `gpu.Buffer` and primitive types only.
- Host-side arenas may help manage temporary staging buffers and other CPU-side GPU resources.

## Generated Code Shape

A C/CUDA backend could emit code similar to:

```c
int *h_data = ...;
int *d_data;
cudaMalloc(&d_data, n * sizeof(int));
cudaMemcpy(d_data, h_data, n * sizeof(int), cudaMemcpyHostToDevice);

my_kernel<<<grid, block>>>(d_data, n);

cudaMemcpy(h_data, d_data, n * sizeof(int), cudaMemcpyDeviceToHost);
cudaFree(d_data);
```

## Summary

This idea prioritizes control and performance over convenience. The central rule is explicit movement between CPU and GPU memory.
