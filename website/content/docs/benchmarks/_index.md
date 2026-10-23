---
title: "Benchmarks"
description: "Measured HTTP throughput, and HTTP requests that launch a GPU kernel."
icon: "speed"
weight: 450
---

Two runs, kept separate. The HTTP numbers are the cgroup matrix. The HTTP + GPU numbers are the handwritten-digits server on one machine.

1. **[HTTP](/docs/benchmarks/http/)**: `wrk` against containerized servers at 1, 2, and 4 vCPUs. Plaintext, JSON, and on-disk SQLite.
2. **[HTTP + GPU](/docs/benchmarks/http-gpu/)**: `POST /predict` with one 8×8 image, `load_digits` row 1728, a handwritten 7. The page links the dataset, the image file, and the trained TXM1 export.
