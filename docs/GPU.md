# GPU Programming Model

**Core rule:** CPU and GPU have separate memory address spaces. All data movement must be explicit.

## Recommended Abstraction

Use `gpu.Buffer[T]` as the primary type for device memory.

```trux
let d_data gpu.Buffer[int] = gpu.alloc(n)
```

`gpu.Buffer[T]` owns device memory and is the only way to pass data to kernels.

## Basic Pattern: CPU → GPU → CPU

```trux
func main() int {
    let n int = 1024 * 1024

    // 1. Data on CPU
    let h_data []int = make([]int, n)
    // ... fill h_data on CPU ...

    // 2. Allocate on GPU
    let d_data gpu.Buffer[int] = gpu.alloc(n)

    // 3. Send data to GPU (explicit copy)
    gpu.copy_to_device(h_data, d_data)

    // 4. Launch kernel
    gpu.launch(fill, d_data, n)

    // 5. Get results back (explicit copy)
    gpu.copy_to_host(d_data, h_data)

    // 6. Use results on CPU
    print(h_data[0])

    return 0
}
```

## Defining a Kernel

Kernels are declared with the `kernel` keyword. They can only be called from the host via a launch, not normal function calls.

```trux
kernel func fill(out gpu.Buffer[int], n int) {
    let i int = gpu.global_id()

    if i < n {
        out[i] = 42
    }
}
```

Kernels have restrictions:
- No dynamic allocation
- Limited language features (see SPECS.md)
- Operate on `gpu.Buffer` and primitive types

## Launching a Kernel

```trux
gpu.launch(fill, d_data, n)
```

The launch is asynchronous by default. Use explicit synchronization when you need to wait for completion before copying results back.

## Complete Example

```trux
kernel func fill(out gpu.Buffer[int], n int) {
    let i int = gpu.global_id()
    if i < n {
        out[i] = 42
    }
}

func main() int {
    let n int = 1024

    let h_data []int = make([]int, n)

    let d_data gpu.Buffer[int] = gpu.alloc(n)
    gpu.copy_to_device(h_data, d_data)

    gpu.launch(fill, d_data, n)

    gpu.copy_to_host(d_data, h_data)

    // h_data now contains results from the GPU
    return 0
}
```

## Key Operations

| Operation                  | Description                              | Direction       |
|---------------------------|------------------------------------------|-----------------|
| `gpu.alloc(n)`            | Allocate `n` elements on the device      | -               |
| `gpu.copy_to_device(h, d)`| Copy from CPU slice to `gpu.Buffer`      | Host → Device   |
| `gpu.copy_to_host(d, h)`  | Copy from `gpu.Buffer` to CPU slice      | Device → Host   |
| `gpu.launch(kernel, ...)` | Launch a GPU kernel                      | -               |
| `gpu.free(buf)`           | Free device memory                       | -               |

## Mixing CPU and GPU Work

You can freely interleave CPU and GPU execution:

```trux
let input  []int = load_data_on_cpu()
let device = gpu.alloc(len(input))

gpu.copy_to_device(input, device)
gpu.launch(heavy_parallel_work, device, len(input))

// Only copy back what the CPU actually needs
let results = make([]int, 256)
gpu.copy_to_host_partial(device, results, 0, 256)

let final = cpu_reduce(results)   // CPU post-processing
save_results(final)
```

**Guideline:** Only copy data back when the CPU actually needs it. Avoid round-tripping large buffers unnecessarily.

## Synchronization Rules

- `gpu.copy_to_host` is **synchronous** by default. It blocks until the kernel and copy complete.
- For performance, use explicit synchronization or streams later:
  - `gpu.sync()`
  - Stream-based async copies

Do not read from a host buffer that was the target of a previous `copy_to_host` until the copy has completed.

## Design Decisions

- **Explicit copies only** (default). Unified memory (`cudaMallocManaged`) is not the primary model and should only be offered as an opt-in later if needed.
- `gpu.Buffer[T]` owns its memory. It is not a view.
- String and other complex host types are **not** supported inside kernels in the initial GPU support.
- Kernels are restricted. They operate on `gpu.Buffer` and primitive types only.
- Host-side arenas are recommended for managing temporary staging buffers and other CPU-side GPU resources.

## Generated Code Shape (C level)

The compiler will emit code similar to:

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

- CPU and GPU memory are separate.
- Use `gpu.Buffer[T]` + explicit `copy_to_device` / `copy_to_host`.
- Minimize data movement.
- CPU work and GPU work can be mixed, but synchronization is required before reading copied-back data.

This model prioritizes control and performance over convenience.