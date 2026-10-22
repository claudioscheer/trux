# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **162,761** | 0.94ms | 489.00us | 1.26ms | 4.31ms | 9.367MiB |
| Plaintext Baseline (/plaintext) | 1000 | **172,765** | 5.74ms | 5.61ms | 6.68ms | 9.40ms | 15.43MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **60,993** | 2.37ms | 1.17ms | 7.02ms | 11.12ms | 30.36MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **59,980** | 16.48ms | 18.66ms | 21.53ms | 28.28ms | 33.24MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **146,278** | 684.49us | 559.00us | 1.21ms | 2.49ms | 33.63MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **143,482** | 6.88ms | 6.77ms | 7.67ms | 9.98ms | 33.79MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **66,073** | 2.26ms | 1.07ms | 6.86ms | 10.64ms | 34.14MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **62,864** | 15.73ms | 18.22ms | 20.77ms | 27.94ms | 34.88MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **189,399** | 0.90ms | 422.00us | 1.22ms | 5.93ms | 13.88MiB |
| Plaintext Baseline (/plaintext) | 1000 | **321,646** | 3.20ms | 2.88ms | 4.14ms | 8.11ms | 20.94MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **53,790** | 2.45ms | 1.45ms | 6.76ms | 11.18ms | 21.87MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **71,345** | 13.88ms | 14.76ms | 21.07ms | 26.73ms | 27.95MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **140,253** | 715.37us | 598.00us | 1.27ms | 2.58ms | 28.44MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **212,590** | 4.63ms | 4.44ms | 6.14ms | 8.40ms | 30.41MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **55,743** | 2.43ms | 1.38ms | 6.86ms | 11.42ms | 31.48MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **74,955** | 13.17ms | 13.91ms | 20.03ms | 25.87ms | 32.02MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **222,763** | 0.87ms | 342.00us | 1.17ms | 7.54ms | 21.91MiB |
| Plaintext Baseline (/plaintext) | 1000 | **452,747** | 2.49ms | 1.75ms | 4.97ms | 11.97ms | 29.74MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **38,497** | 3.03ms | 2.13ms | 6.96ms | 12.92ms | 32.25MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **55,219** | 17.99ms | 17.85ms | 28.38ms | 36.15ms | 40.83MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **154,332** | 661.45us | 540.00us | 1.20ms | 2.50ms | 40.97MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **280,389** | 3.57ms | 3.29ms | 6.09ms | 9.67ms | 45.51MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **39,346** | 2.97ms | 2.05ms | 6.92ms | 12.82ms | 46.86MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **56,003** | 17.75ms | 17.61ms | 28.12ms | 35.89ms | 50.54MiB |

