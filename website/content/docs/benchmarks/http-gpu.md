---
title: "HTTP + GPU"
description: "Handwritten-digits requests that launch one CUDA kernel and answer on the io_uring loop."
icon: "developer_board"
weight: 20
---

`benchmarks/digits` times `POST /predict` on the handwritten-digits SVM. The server mounts a TXM1 file once. Each request copies 64 floats into that worker's workspace, launches one RBF kernel, and finishes on the `io_uring` loop. The weights stay on the device. One kernel is in flight per worker.

```bash
make bench-digits
```

That runs `go run ./benchmarks/digits/`. Linux, liburing, and CUDA. It is not part of `make bench`, and it is not the cgroup matrix on the [HTTP](/docs/benchmarks/http/) page.

`-duration` and `-concurrency` change the run. `-model` points at a trained `digits.txm`. The default file is synthetic, with the notebook shape: 519 support vectors, 64 features, gamma 0.001. An image of 64 zeros is digit 0. The harness checks that response before it starts the timer.

## This run

- **Host**: Linux Mint 22.3, kernel 7.0.0-38-generic, AMD Ryzen 9 5900X
- **Workers**: 24, one per hardware thread in the process affinity mask
- **GPU**: NVIDIA GeForce GTX 1660, 6 GiB, driver 595.99, CUDA 12.0 (`nvcc` 12.0.140)
- **Library**: liburing 2.5
- **Duration**: 6 seconds at 1, 8, and 32 keep-alive connections
- **Check**: `{"digit":0}` for 64 zeros, 0 errors

| Concurrency | Requests | Throughput (req/s) | p50 | p99 |
| :---: | :---: | :---: | :---: | :---: |
| 1 | 4,279 | **713.1** | 0.362 ms | 2.042 ms |
| 8 | 43,666 | **6,439.2** | 0.377 ms | 2.281 ms |
| 32 | 62,664 | **8,961.5** | 0.509 ms | 3.359 ms |

From 1 to 8 connections, throughput follows the extra in-flight kernels. At 32 connections there are still 24 workers, so some requests wait for a worker before they launch.
