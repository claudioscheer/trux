# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **171,472** | 1.35ms | 467.00us | 1.24ms | 26.18ms | 9.84MiB |
| Plaintext Baseline (/plaintext) | 1000 | **181,616** | 5.55ms | 5.34ms | 6.29ms | 8.94ms | 17.08MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **62,174** | 2.28ms | 1.16ms | 6.71ms | 10.52ms | 30.93MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **61,224** | 16.18ms | 18.83ms | 21.02ms | 28.85ms | 34.16MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **148,280** | 683.10us | 558.00us | 1.20ms | 2.50ms | 34.29MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **144,907** | 6.84ms | 6.72ms | 7.73ms | 10.16ms | 33.93MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **57,428** | 3.84ms | 1.12ms | 10.69ms | 28.03ms | 34.23MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **46,161** | 21.44ms | 23.28ms | 37.02ms | 42.74ms | 34.52MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **187,385** | 0.89ms | 422.00us | 1.20ms | 4.83ms | 13.85MiB |
| Plaintext Baseline (/plaintext) | 1000 | **321,562** | 3.11ms | 2.87ms | 4.19ms | 7.27ms | 20.73MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **52,168** | 2.81ms | 1.47ms | 7.81ms | 19.13ms | 22.33MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **70,855** | 13.97ms | 15.01ms | 21.16ms | 30.68ms | 28.25MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **139,039** | 723.05us | 611.00us | 1.27ms | 2.51ms | 27.75MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **211,987** | 4.65ms | 4.46ms | 6.24ms | 8.74ms | 30.37MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **41,320** | 5.92ms | 1.59ms | 19.46ms | 43.76ms | 30.67MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **56,472** | 18.61ms | 16.28ms | 34.46ms | 53.94ms | 31.59MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **242,837** | 831.70us | 317.00us | 1.06ms | 5.02ms | 22.08MiB |
| Plaintext Baseline (/plaintext) | 1000 | **440,072** | 2.55ms | 1.79ms | 5.30ms | 12.02ms | 29.88MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **38,644** | 2.95ms | 2.11ms | 6.61ms | 12.46ms | 31.61MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **54,240** | 18.20ms | 17.98ms | 28.60ms | 37.06ms | 40.37MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **153,217** | 669.22us | 549.00us | 1.22ms | 2.53ms | 40.34MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **262,383** | 3.78ms | 3.57ms | 6.25ms | 9.55ms | 45.09MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **39,400** | 2.97ms | 2.09ms | 6.90ms | 12.81ms | 45.59MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **56,217** | 17.64ms | 17.59ms | 27.92ms | 34.58ms | 49.3MiB |

