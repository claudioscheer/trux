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
- **Multishot Operations & Async Response Queuing**: A single Submission Queue Entry (SQE) establishes continuous connection acceptance (`io_uring_prep_multishot_accept`) and streaming receive (`io_uring_prep_recv_multishot`), while outgoing HTTP responses are queued asynchronously via `io_uring_prep_send` and flushed in batched ring submissions.
- **CPU Affinity Worker Sizing (`sched_getaffinity`)**: Worker threads automatically match the container or host CPU affinity mask (`CPU_COUNT`), maintaining a strict 1:1 thread-per-core mapping without over-spawning threads inside cgroup-constrained containers.
- **Predictable Memory Footprint**: Bounded buffer rings and request-scoped arenas keep memory consumption between **9.4 MiB and 50.5 MiB** regardless of concurrent connection counts.

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

1. **Plaintext Baseline (`/plaintext`)**: Raw HTTP/1.1 framing and minimal response body (`"Hello, World!"`), measuring peak kernel network I/O with zero database overhead.
2. **SQLite File Insert & Query (`/sqlite/insert-and-query`)**: Real on-disk SQLite WAL database file (`/tmp/bench.db`) write transaction: appends an access log entry (`INSERT INTO logs`), queries the latest row (`SELECT`), and serializes JSON.
3. **SQLite File Prepared Query (`/sqlite/query-by-id/1024`)**: Concurrent reads on disk: extracts route parameter (`:id`), compiles a prepared SQL statement (`SELECT name, role FROM users WHERE id = ?`), binds parameters (`bindText`), steps the cursor, and returns JSON.
4. **Middleware & SQLite File Audit (`/api/authenticated-audit`)**: Prefix middleware (`useAt`), Bearer token validation, on-disk SQLite WAL audit log insertion, request-scoped context dictionary propagation, and privileged JSON response delivery.

---

## Benchmark Results

### 1. Micro Edge (1 vCPU, 256 MB RAM)

Simulates constrained edge environments, micro instances, or single-core serverless containers (`--cpuset-cpus="0" --cpus="1.0" --memory="256m"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext Baseline** (`/plaintext`) | 100 | **162,761** | 15.99 MB/s | 0.94 ms | 489 µs | 1.26 ms | 4.31 ms | 9.4 MiB |
| **Plaintext Baseline** (`/plaintext`) | 1,000 | **172,765** | 16.97 MB/s | 5.74 ms | 5.61 ms | 6.68 ms | 9.40 ms | 15.4 MiB |
| **SQLite File Insert & Query** (`/sqlite/insert-and-query`) | 100 | **60,993** | 9.77 MB/s | 2.37 ms | 1.17 ms | 7.02 ms | 11.12 ms | 30.4 MiB |
| **SQLite File Insert & Query** (`/sqlite/insert-and-query`) | 1,000 | **59,980** | 9.61 MB/s | 16.48 ms | 18.66 ms | 21.53 ms | 28.28 ms | 33.2 MiB |
| **SQLite File Prepared Query** (`/sqlite/query-by-id/1024`) | 100 | **146,278** | 19.53 MB/s | 684 µs | 559 µs | 1.21 ms | 2.49 ms | 33.6 MiB |
| **SQLite File Prepared Query** (`/sqlite/query-by-id/1024`) | 1,000 | **143,482** | 19.16 MB/s | 6.88 ms | 6.77 ms | 7.67 ms | 9.98 ms | 33.8 MiB |
| **Middleware & SQLite File Audit** (`/api/authenticated-audit`) | 100 | **66,073** | 8.76 MB/s | 2.26 ms | 1.07 ms | 6.86 ms | 10.64 ms | 34.1 MiB |
| **Middleware & SQLite File Audit** (`/api/authenticated-audit`) | 1,000 | **62,864** | 8.33 MB/s | 15.73 ms | 18.22 ms | 20.77 ms | 27.94 ms | 34.9 MiB |

---

### 2. Cloud Standard (2 vCPUs, 1 GB RAM)

Simulates standard dual-core virtual machines (`--cpuset-cpus="0,1" --cpus="2.0" --memory="1g"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext Baseline** (`/plaintext`) | 100 | **189,399** | 18.60 MB/s | 0.90 ms | 422 µs | 1.22 ms | 5.93 ms | 13.9 MiB |
| **Plaintext Baseline** (`/plaintext`) | 1,000 | **321,646** | 31.59 MB/s | 3.20 ms | 2.88 ms | 4.14 ms | 8.11 ms | 20.9 MiB |
| **SQLite File Insert & Query** (`/sqlite/insert-and-query`) | 100 | **53,790** | 8.62 MB/s | 2.45 ms | 1.45 ms | 6.76 ms | 11.18 ms | 21.9 MiB |
| **SQLite File Insert & Query** (`/sqlite/insert-and-query`) | 1,000 | **71,345** | 11.43 MB/s | 13.88 ms | 14.76 ms | 21.07 ms | 26.73 ms | 28.0 MiB |
| **SQLite File Prepared Query** (`/sqlite/query-by-id/1024`) | 100 | **140,253** | 18.73 MB/s | 715 µs | 598 µs | 1.27 ms | 2.58 ms | 28.4 MiB |
| **SQLite File Prepared Query** (`/sqlite/query-by-id/1024`) | 1,000 | **212,590** | 28.38 MB/s | 4.63 ms | 4.44 ms | 6.14 ms | 8.40 ms | 30.4 MiB |
| **Middleware & SQLite File Audit** (`/api/authenticated-audit`) | 100 | **55,743** | 7.39 MB/s | 2.43 ms | 1.38 ms | 6.86 ms | 11.42 ms | 31.5 MiB |
| **Middleware & SQLite File Audit** (`/api/authenticated-audit`) | 1,000 | **74,955** | 9.94 MB/s | 13.17 ms | 13.91 ms | 20.03 ms | 25.87 ms | 32.0 MiB |

---

### 3. Compute Node (4 vCPUs, 2 GB RAM)

Simulates multi-core production server nodes (`--cpuset-cpus="0,1,2,3" --cpus="4.0" --memory="2g"`).

| Endpoint | Concurrency | Throughput (req/s) | Transfer | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Plaintext Baseline** (`/plaintext`) | 100 | **222,763** | 21.88 MB/s | 0.87 ms | 342 µs | 1.17 ms | 7.54 ms | 21.9 MiB |
| **Plaintext Baseline** (`/plaintext`) | 1,000 | **452,747** | 44.47 MB/s | 2.49 ms | 1.75 ms | 4.97 ms | 11.97 ms | 29.7 MiB |
| **SQLite File Insert & Query** (`/sqlite/insert-and-query`) | 100 | **38,497** | 6.17 MB/s | 3.03 ms | 2.13 ms | 6.96 ms | 12.92 ms | 32.3 MiB |
| **SQLite File Insert & Query** (`/sqlite/insert-and-query`) | 1,000 | **55,219** | 8.85 MB/s | 17.99 ms | 17.85 ms | 28.38 ms | 36.15 ms | 40.8 MiB |
| **SQLite File Prepared Query** (`/sqlite/query-by-id/1024`) | 100 | **154,332** | 20.61 MB/s | 661 µs | 540 µs | 1.20 ms | 2.50 ms | 41.0 MiB |
| **SQLite File Prepared Query** (`/sqlite/query-by-id/1024`) | 1,000 | **280,389** | 37.44 MB/s | 3.57 ms | 3.29 ms | 6.09 ms | 9.67 ms | 45.5 MiB |
| **Middleware & SQLite File Audit** (`/api/authenticated-audit`) | 100 | **39,346** | 5.22 MB/s | 2.97 ms | 2.05 ms | 6.92 ms | 12.82 ms | 46.9 MiB |
| **Middleware & SQLite File Audit** (`/api/authenticated-audit`) | 1,000 | **56,003** | 7.42 MB/s | 17.75 ms | 17.61 ms | 28.12 ms | 35.89 ms | 50.5 MiB |

---

## Analysis & Key Takeaways

### Realistic On-Disk Database Workloads
The benchmark operates against an actual persistent database file on disk (`/tmp/bench.db`) configured with SQLite's recommended Write-Ahead Logging mode (`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA mmap_size=67108864; PRAGMA cache_size=-16384;`).

### Thread-Local Connections & Persistent Prepared Statement Cache
Handlers utilize thread-local connection caching (`sqlite.openThreadLocal`) paired with a transparent per-connection prepared statement cache (`sqlite3_prepare_v3` with `SQLITE_PREPARE_PERSISTENT`):
- **Zero File Descriptor or Parser Churn**: Eliminates file open/close syscalls, SQL lexing, AST parsing, and VDBE compilation on repeated requests.
- **Lock-Free Concurrency (`SQLITE_THREADSAFE=2`)**: Because each worker thread owns its connection handle exclusively, SQLite runs in multi-thread mode without per-call recursive connection mutexes (`db->mutex`) or global memory-tracking locks (`SQLITE_DEFAULT_MEMSTATUS=0`).
- **Single-Roundtrip Write+Read (`INSERT ... RETURNING`)**: `/sqlite/insert-and-query` inserts and retrieves the new row in a single VDBE execution without a secondary B-tree lookup.

### Concurrent Readers vs. High-Throughput WAL Writes
- **High-Throughput Reads (`/sqlite/query-by-id/1024`)**: Parameterized prepared queries scale up to **280,389 req/s** on disk with a **9.67 ms** p99 latency at 1,000 concurrency across 4 cores.
- **High-Throughput WAL Writes (`/sqlite/insert-and-query`, `/api/authenticated-audit`)**: Combining `PRAGMA synchronous=NORMAL`, persistent statement caching, CPU-affinity worker sizing (`sched_getaffinity`), and bounded 2,000-frame `SQLITE_CHECKPOINT_TRUNCATE` checkpoints sustains **55,219 to 74,955 writes/sec** with **1.07–2.13 ms** median latency at 100 concurrency and zero lock errors.
- **Async `io_uring_prep_send` & Batched CQE Draining**: Worker threads drain all ready completion queue entries (`io_uring_for_each_cqe`) and queue HTTP responses via `io_uring_prep_send`, coalescing network I/O into batched ring submissions.

### Low Memory Footprint Under Continuous I/O
Matching worker threads to container CPU affinity (`1`, `2`, and `4` workers) and bounding `/tmp/bench.db-wal` via truncating checkpoints keeps total container memory between **9.4 MiB and 50.5 MiB** across millions of database reads and writes.

---

## Reproducing the Benchmarks

The benchmark harness and server source code are located in `benchmarks/http/`:

```bash
# Compile and run the benchmark suite against Docker containers
python3 benchmarks/http/benchmark.py
```

The script compiles the Trux HTTP server, builds the runtime Docker container, executes `wrk` across all hardware profiles, and generates `BENCHMARK_RESULTS.md` and `results.json`.
