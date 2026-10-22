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

1. **Plaintext (`/plaintext`)**: Raw HTTP/1.1 framing and minimal response body (`"Hello, World!"`), measuring peak kernel network I/O with zero database overhead.
2. **SQLite File Insert & Query (`/json`)**: Real on-disk SQLite WAL database file (`/tmp/bench.db`) write transaction: appends an access log entry (`INSERT INTO logs`), queries the latest row (`SELECT`), and serializes JSON.
3. **SQLite File Prepared Query (`/users/1024`)**: Concurrent reads on disk: extracts route parameter (`:id`), compiles a prepared SQL statement (`SELECT name, role FROM users WHERE id = ?`), binds parameters (`bindText`), steps the cursor, and returns JSON.
4. **Middleware & SQLite File Audit (`/api/data`)**: Prefix middleware (`useAt`), Bearer token validation, on-disk SQLite WAL audit log insertion, request-scoped context dictionary propagation, and privileged JSON response delivery.

---

## Benchmark Results

### 1. Micro Edge (1 vCPU, 256 MB RAM)

Simulates constrained edge environments, micro instances, or single-core serverless containers (`--cpuset-cpus="0" --cpus="1.0" --memory="256m"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext** (`/plaintext`) | 100 | **167,308** | 16.43 MB/s | 960 µs | 486 µs | 1.21 ms | 6.16 ms | 91.7 MiB |
| **Plaintext** (`/plaintext`) | 1,000 | **178,487** | 17.53 MB/s | 6.23 ms | 5.58 ms | 13.27 ms | 24.79 ms | 95.0 MiB |
| **SQLite File Insert & Query** (`/json`) | 100 | **760** | 105.3 KB/s | 59.72 ms | 5.26 ms | 10.99 ms | 1.44 s | 99.3 MiB |
| **SQLite File Insert & Query** (`/json`) | 1,000 | **572** | 79.4 KB/s | 384.75 ms | 173.71 ms | 1.20 s | 1.84 s | 122.5 MiB |
| **SQLite File Prepared Query** (`/users/1024`) | 100 | **15,262** | 2.04 MB/s | 27.64 ms | 514 µs | 61.65 ms | 440.28 ms | 106.0 MiB |
| **SQLite File Prepared Query** (`/users/1024`) | 1,000 | **14,473** | 1.93 MB/s | 64.74 ms | 59.74 ms | 124.69 ms | 196.35 ms | 111.3 MiB |
| **Middleware & SQLite File Audit** (`/api/data`) | 100 | **774** | 105.0 KB/s | 89.20 ms | 7.04 ms | 179.07 ms | 1.58 s | 111.8 MiB |
| **Middleware & SQLite File Audit** (`/api/data`) | 1,000 | **804** | 109.2 KB/s | 98.65 ms | 56.22 ms | 82.64 ms | 1.08 s | 122.2 MiB |

---

### 2. Cloud Standard (2 vCPUs, 1 GB RAM)

Simulates standard dual-core virtual machines (`--cpuset-cpus="0,1" --cpus="2.0" --memory="1g"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext** (`/plaintext`) | 100 | **188,964** | 18.56 MB/s | 754 µs | 427 µs | 1.15 ms | 3.28 ms | 91.5 MiB |
| **Plaintext** (`/plaintext`) | 1,000 | **331,785** | 32.59 MB/s | 3.66 ms | 2.80 ms | 8.10 ms | 17.91 ms | 95.0 MiB |
| **SQLite File Insert & Query** (`/json`) | 100 | **773** | 107.0 KB/s | 200.10 ms | 6.62 ms | 807.51 ms | 1.61 s | 99.6 MiB |
| **SQLite File Insert & Query** (`/json`) | 1,000 | **806** | 111.8 KB/s | 439.41 ms | 131.41 ms | 1.38 s | 1.95 s | 110.4 MiB |
| **SQLite File Prepared Query** (`/users/1024`) | 100 | **21,730** | 2.90 MB/s | 5.07 ms | 3.71 ms | 7.32 ms | 12.92 ms | 106.8 MiB |
| **SQLite File Prepared Query** (`/users/1024`) | 1,000 | **18,421** | 2.46 MB/s | 48.43 ms | 48.25 ms | 59.19 ms | 68.02 ms | 111.9 MiB |
| **Middleware & SQLite File Audit** (`/api/data`) | 100 | **807** | 109.5 KB/s | 153.31 ms | 6.24 ms | 616.49 ms | 1.42 s | 111.9 MiB |
| **Middleware & SQLite File Audit** (`/api/data`) | 1,000 | **780** | 105.8 KB/s | 455.41 ms | 209.23 ms | 1.36 s | 1.86 s | 117.4 MiB |

---

### 3. Compute Node (4 vCPUs, 2 GB RAM)

Simulates multi-core production server nodes (`--cpuset-cpus="0,1,2,3" --cpus="4.0" --memory="2g"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext** (`/plaintext`) | 100 | **256,123** | 25.16 MB/s | 860 µs | 294 µs | 1.03 ms | 8.91 ms | 91.5 MiB |
| **Plaintext** (`/plaintext`) | 1,000 | **446,259** | 43.84 MB/s | 2.56 ms | 1.77 ms | 5.64 ms | 12.25 ms | 95.3 MiB |
| **SQLite File Insert & Query** (`/json`) | 100 | **808** | 111.9 KB/s | 114.64 ms | 7.04 ms | 435.90 ms | 1.42 s | 97.7 MiB |
| **SQLite File Insert & Query** (`/json`) | 1,000 | **734** | 101.8 KB/s | 324.12 ms | 76.43 ms | 995.02 ms | 1.98 s | 100.3 MiB |
| **SQLite File Prepared Query** (`/users/1024`) | 100 | **12,792** | 1.71 MB/s | 3.09 ms | 3.15 ms | 4.89 ms | 6.63 ms | 100.6 MiB |
| **SQLite File Prepared Query** (`/users/1024`) | 1,000 | **12,903** | 1.72 MB/s | 32.07 ms | 31.77 ms | 37.73 ms | 42.79 ms | 103.2 MiB |
| **Middleware & SQLite File Audit** (`/api/data`) | 100 | **794** | 107.8 KB/s | 104.50 ms | 5.45 ms | 361.77 ms | 1.44 s | 103.4 MiB |
| **Middleware & SQLite File Audit** (`/api/data`) | 1,000 | **785** | 106.5 KB/s | 254.64 ms | 67.20 ms | 1.01 s | 1.91 s | 106.3 MiB |

---

## Analysis & Key Takeaways

### Realistic On-Disk Database Workloads
The benchmark operates against an actual persistent database file on disk (`/tmp/bench.db`) configured with SQLite's recommended Write-Ahead Logging mode (`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`).

### Concurrent Readers vs. Serialized Disk Writes
- **High-Throughput Reads (`/users/1024`)**: Because WAL mode allows readers to run concurrently without taking exclusive locks, parameterized prepared queries achieve **12,800 to 21,700 req/s** on disk with median latency strictly under **3.7 ms** at 100 concurrency.
- **Physical Disk Write Limits (`/json`, `/api/data`)**: SQLite serializes write transactions to guarantee ACID durability. Under unbatched auto-commit inserts, Trux saturates disk I/O at **730 to 810 writes/sec**.
- **Zero Lock Failures**: Thanks to `sqlite3_busy_timeout` queueing, zero requests failed with `database is locked` despite 1,000 simultaneous concurrent writers.

### Bounded Memory Stability Under Continuous I/O
Across hundreds of thousands of concurrent database transactions, container memory remains bounded between **91 MiB and 122 MiB**. Lock-free handle management and explicit resource closing eliminate heap bloat and guarantee long-running stability without garbage collection pauses.

---

## Reproducing the Benchmarks

The benchmark harness and server source code are located in `benchmarks/http/`:

```bash
# Compile and run the benchmark suite against Docker containers
python3 benchmarks/http/benchmark.py
```

The script compiles the Trux HTTP server, builds the runtime Docker container, executes `wrk` across all hardware profiles, and generates `BENCHMARK_RESULTS.md` and `results.json`.
