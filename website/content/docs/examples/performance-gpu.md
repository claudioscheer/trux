---
title: "Performance and GPU"
description: "Matrix multiply, stencil, CNN, and image rotation, with CPU and GPU timings."
icon: "speed"
weight: 90
---

These programs time their work, so their output varies from machine to machine. The GPU ones need an NVIDIA GPU and
`nvcc`. The larger [MNIST MLP](/docs/examples/mnist-mlp/) project has its own page.

| Example | Needs a GPU | What it does |
| --- | --- | --- |
| `examples/performance_integer.tx` | no | A CPU-bound integer workload. Compare `trux run` (built with `-O2`) and `trux run --debug` (`-O0`). |
| `examples/gpu_matrix_multiply.tx` | yes | 500 x 500 integer matrix multiply on the CPU and with `gpu.launch2d`, with a full result comparison. |
| `examples/performance_stencil.tx` | yes | 200 Jacobi stencil steps on a 1024 x 1024 grid, swapping two device buffers. |
| `examples/performance_cnn.tx` | yes | A four-layer float convolutional network using `gpu.launch1d`. |
| `examples/projects/rotate_image/main.tx` | yes | Rotates a PPM image on CPU and GPU and writes both results. |

`make bench` runs the first four in sequence.

{{< details summary="`examples/gpu_matrix_multiply.tx`" >}}

{{< include "examples/gpu_matrix_multiply.tx" "trux" >}}

{{< /details >}}
