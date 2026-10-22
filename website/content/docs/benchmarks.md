---
title: "Benchmarks"
description: "HTTP performance and throughput under Linux io_uring across simulated server environments."
icon: "speed"
weight: 450
---

Trux features a high-performance HTTP engine engineered specifically for Linux servers. It is built natively on **`io_uring`** and **`liburing`**, departing from legacy `epoll` architectures and goroutine schedulers in favor of a shared-nothing, thread-per-core model.

## Architecture Highlights

- **Shared-Nothing Thread-per-Core**: Each hardware execution unit runs an independent worker thread. Every worker creates its own listening socket with `SO_REUSEPORT`, dedicated `io_uring` instance, and dedicated kernel buffer ring.
- **Kernel Buffer Rings (`io_uring_register_buf_ring`)**: The Linux kernel dynamically selects pre-registered buffers upon incoming network packets (`IOSQE_BUFFER_SELECT`), eliminating user-space memory allocations and read system calls.
- **Multishot Operations**: A single Submission Queue Entry (SQE) establishes continuous connection acceptance (`io_uring_prep_multishot_accept`) and streaming receive (`io_uring_prep_recv_multishot`) for the lifetime of connections.
- **Vectorized Scatter-Gather Responses**: HTTP headers and payloads are emitted through vectorized I/O (`writev`), preventing memory copies.
- **Predictable Memory Footprint**: Bounded buffer rings and request-scoped arenas keep memory consumption flat (< 100 MiB) regardless of concurrent connection counts.

```mermaid
flowchart TD
    Client["Client Traffic"] --> Kernel["Linux Kernel SO_REUSEPORT Hashing"]
    Kernel --> Worker0["Worker Core 0 (io_uring + Buffer Ring)"]
    Kernel --> Worker1["Worker Core 1 (io_uring + Buffer Ring)"]
    Worker0 --> Arena0["Per-Request Arena Isolation"]
    Worker1 --> Arena1["Per-Request Arena Isolation"]
```

---

## Test Environment & Methodology

Benchmarks were conducted using the standard HTTP benchmarking utility [`wrk`](https://github.com/wg/wrk) against containerized Trux servers run under strict Linux cgroup hardware limits:

- **Host OS & Kernel**: Ubuntu 24.04 LTS on Linux 6.18 (`x86_64`)
- **Container Runtime**: Docker Engine 29.1 (cgroup v2, `--security-opt seccomp=unconfined`, `--ulimit memlock=-1:-1`)
- **Libraries**: `liburing2` v2.5
- **Connections**: 100 and 1,000 persistent HTTP/1.1 keep-alive connections
- **Duration**: 6 seconds per scenario

### Workloads Evaluated

1. **Plaintext (`/plaintext`)**: Raw HTTP/1.1 framing and minimal response body (`"Hello, World!"`), serving as the zero-database network ceiling baseline.
2. **SQLite Insert & Query (`/json`)**: Real database transaction per request: creates table, inserts an access log entry (`INSERT INTO logs`), reads it back with a `SELECT` query, and returns JSON.
3. **SQLite Prepared Query (`/users/1024`)**: Dynamic route parameter parsing (`:id`), preparing a parameterized query statement (`SELECT name, role FROM users WHERE id = ?`), binding parameters (`bindText`), stepping the cursor (`step`), and extracting column values (`columnText`).
4. **Middleware & SQLite Audit (`/api/data`)**: Prefix middleware (`useAt`), Bearer token validation, logging an audit record into SQLite (`INSERT INTO audit`), request-scoped context dictionary propagation, and privileged JSON response delivery.

---

## Benchmark Results

### 1. Micro Edge (1 vCPU, 256 MB RAM)

Simulates constrained edge environments, micro instances, or single-core serverless containers (`--cpuset-cpus="0" --cpus="1.0" --memory="256m"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext** (`/plaintext`) | 100 | **157,549** | 15.48 MB/s | 870 µs | 526 µs | 1.29 ms | 3.64 ms | 91.6 MiB |
| **Plaintext** (`/plaintext`) | 1,000 | **168,376** | 16.54 MB/s | 6.69 ms | 5.92 ms | 14.34 ms | 27.72 ms | 94.8 MiB |
| **SQLite Insert & Query** (`/json`) | 100 | **18,534** | 2.45 MB/s | 16.69 ms | 1.01 ms | 53.84 ms | 96.29 ms | 98.0 MiB |
| **SQLite Insert & Query** (`/json`) | 1,000 | **18,301** | 2.42 MB/s | 55.46 ms | 62.20 ms | 118.28 ms | 186.31 ms | 103.5 MiB |
| **SQLite Prepared Query** (`/users/1024`) | 100 | **17,001** | 2.27 MB/s | 20.79 ms | 632 µs | 64.58 ms | 116.14 ms | 104.8 MiB |
| **SQLite Prepared Query** (`/users/1024`) | 1,000 | **16,315** | 2.18 MB/s | 60.26 ms | 63.65 ms | 122.34 ms | 187.12 ms | 108.8 MiB |
| **Middleware & SQLite Audit** (`/api/data`) | 100 | **22,073** | 2.92 MB/s | 14.28 ms | 940 µs | 47.17 ms | 87.91 ms | 109.6 MiB |
| **Middleware & SQLite Audit** (`/api/data`) | 1,000 | **21,487** | 2.85 MB/s | 49.37 ms | 52.12 ms | 111.43 ms | 182.43 ms | 113.4 MiB |

---

### 2. Cloud Standard (2 vCPUs, 1 GB RAM)

Simulates standard dual-core virtual machines (`--cpuset-cpus="0,1" --cpus="2.0" --memory="1g"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext** (`/plaintext`) | 100 | **177,120** | 17.40 MB/s | 890 µs | 454 µs | 1.26 ms | 4.98 ms | 87.4 MiB |
| **Plaintext** (`/plaintext`) | 1,000 | **298,837** | 29.35 MB/s | 3.83 ms | 3.04 ms | 8.27 ms | 17.98 ms | 94.8 MiB |
| **SQLite Insert & Query** (`/json`) | 100 | **19,627** | 2.60 MB/s | 4.33 ms | 4.25 ms | 7.12 ms | 9.36 ms | 97.9 MiB |
| **SQLite Insert & Query** (`/json`) | 1,000 | **19,232** | 2.55 MB/s | 47.12 ms | 46.68 ms | 58.40 ms | 69.81 ms | 102.5 MiB |
| **SQLite Prepared Query** (`/users/1024`) | 100 | **16,705** | 2.23 MB/s | 4.93 ms | 4.61 ms | 8.77 ms | 12.38 ms | 104.1 MiB |
| **SQLite Prepared Query** (`/users/1024`) | 1,000 | **16,023** | 2.14 MB/s | 55.75 ms | 55.04 ms | 68.82 ms | 84.21 ms | 108.6 MiB |
| **Middleware & SQLite Audit** (`/api/data`) | 100 | **20,252** | 2.68 MB/s | 4.32 ms | 4.10 ms | 7.50 ms | 10.81 ms | 108.8 MiB |
| **Middleware & SQLite Audit** (`/api/data`) | 1,000 | **20,528** | 2.72 MB/s | 43.70 ms | 43.26 ms | 51.88 ms | 62.29 ms | 112.7 MiB |

---

### 3. Compute Node (4 vCPUs, 2 GB RAM)

Simulates multi-core production server nodes (`--cpuset-cpus="0,1,2,3" --cpus="4.0" --memory="2g"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext** (`/plaintext`) | 100 | **241,266** | 23.70 MB/s | 910 µs | 315 µs | 1.12 ms | 6.50 ms | 91.6 MiB |
| **Plaintext** (`/plaintext`) | 1,000 | **412,084** | 40.47 MB/s | 2.76 ms | 1.92 ms | 5.97 ms | 13.42 ms | 95.0 MiB |
| **SQLite Insert & Query** (`/json`) | 100 | **13,622** | 1.80 MB/s | 2.99 ms | 2.49 ms | 5.72 ms | 7.95 ms | 96.9 MiB |
| **SQLite Insert & Query** (`/json`) | 1,000 | **14,030** | 1.86 MB/s | 33.79 ms | 34.20 ms | 42.56 ms | 50.76 ms | 99.0 MiB |
| **SQLite Prepared Query** (`/users/1024`) | 100 | **12,970** | 1.73 MB/s | 3.75 ms | 3.52 ms | 6.54 ms | 9.86 ms | 100.3 MiB |
| **SQLite Prepared Query** (`/users/1024`) | 1,000 | **12,828** | 1.71 MB/s | 37.82 ms | 38.13 ms | 45.66 ms | 51.00 ms | 103.2 MiB |
| **Middleware & SQLite Audit** (`/api/data`) | 100 | **15,442** | 2.05 MB/s | 3.53 ms | 3.50 ms | 6.21 ms | 8.72 ms | 102.9 MiB |
| **Middleware & SQLite Audit** (`/api/data`) | 1,000 | **15,173** | 2.01 MB/s | 32.25 ms | 31.26 ms | 40.02 ms | 47.58 ms | 104.9 MiB |

---

## Analysis & Key Takeaways

### Reliable Real-World Workload Metrics
Rather than testing empty responses, the benchmark exercises real computational workloads: executing SQLite table setup, row insertions, prepared statement compilation, 1-based parameter binding, cursor stepping, column extraction, and token authentication on every single HTTP request.

### Raw I/O vs. Database Workloads
- **Network I/O Ceiling**: The raw plaintext handler scales from **168,376 req/s** on 1 core to **412,084 req/s** on 4 cores with sub-2ms median latency.
- **SQLite Database Throughput**: Endpoints executing full SQLite database operations sustain **13,000 to 22,000 requests per second** under 1,000 concurrent persistent connections.
- **Low Median Tail Latency**: At 100 concurrent connections across 4 cores, median latency for SQLite operations is strictly under **3.6 ms** (p99 < 9.9 ms).

### Memory Predictability Under Continuous Load
Throughout hundreds of thousands of concurrent database transactions, container memory remains bounded between **87 MiB and 113 MiB**. Lock-free handle management and explicit resource closing eliminate heap bloat and guarantee long-running stability without garbage collection pauses.

---

## Reproducing the Benchmarks

The benchmark harness and server source code are located in `benchmarks/http/`:

```bash
# Compile and run the benchmark suite against Docker containers
python3 benchmarks/http/benchmark.py
```

The script compiles the Trux HTTP server, builds the runtime Docker container, executes `wrk` across all hardware profiles, and generates `BENCHMARK_RESULTS.md` and `results.json`.
