---
title: "Benchmarks"
description: "Measured HTTP throughput, and HTTP requests that launch a GPU kernel."
icon: "speed"
weight: 450
---

Two runs, kept separate. The HTTP numbers are the cgroup matrix. The HTTP + GPU numbers are the handwritten-digits server on one machine.

1. **[HTTP](/docs/benchmarks/http/)**: `wrk` against containerized servers at 1, 2, and 4 vCPUs. Plaintext, JSON, and on-disk SQLite.
2. **[HTTP + GPU](/docs/benchmarks/http-gpu/)**: `POST /predict` on the digits server. Each request launches one RBF kernel and returns on the `io_uring` loop.
