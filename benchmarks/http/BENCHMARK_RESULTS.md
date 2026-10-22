# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **168,601** | 1.02ms | 489.00us | 1.20ms | 7.82ms | 91.48MiB |
| Plaintext Baseline (/plaintext) | 1000 | **182,692** | 6.07ms | 5.43ms | 12.83ms | 24.13ms | 94.73MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **1,393** | 26.33ms | 3.63ms | 5.84ms | 915.55ms | 98.89MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **1,457** | 47.98ms | 30.38ms | 37.12ms | 881.62ms | 105.4MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **113,213** | 1.54ms | 725.00us | 1.60ms | 4.16ms | 105.2MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **110,632** | 14.89ms | 8.28ms | 42.83ms | 81.60ms | 108.8MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **1,534** | 40.61ms | 4.01ms | 5.57ms | 1.25s | 109.8MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **1,576** | 49.38ms | 36.92ms | 40.91ms | 948.47ms | 113.3MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **187,769** | 813.80us | 435.00us | 1.18ms | 4.95ms | 91.62MiB |
| Plaintext Baseline (/plaintext) | 1000 | **318,316** | 3.72ms | 2.89ms | 8.27ms | 17.54ms | 94.71MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **1,315** | 131.62ms | 4.01ms | 546.36ms | 1.49s | 98.93MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **1,352** | 209.24ms | 36.87ms | 867.66ms | 1.81s | 103.6MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **121,268** | 600.45us | 454.00us | 1.18ms | 2.59ms | 103.8MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **189,167** | 3.60ms | 2.92ms | 7.72ms | 14.24ms | 106.2MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **1,517** | 103.06ms | 3.64ms | 347.50ms | 1.50s | 106.9MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **1,545** | 116.20ms | 31.64ms | 264.53ms | 1.46s | 109.1MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **248,440** | 762.55us | 307.00us | 1.05ms | 5.57ms | 91.3MiB |
| Plaintext Baseline (/plaintext) | 1000 | **436,352** | 2.60ms | 1.83ms | 5.65ms | 12.52ms | 95.54MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **1,318** | 170.80ms | 5.49ms | 661.51ms | 1.53s | 101MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **1,375** | 324.28ms | 52.86ms | 1.08s | 1.79s | 106.7MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **157,146** | 648.72us | 509.00us | 1.23ms | 2.63ms | 107.1MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **310,436** | 3.36ms | 2.35ms | 7.68ms | 15.26ms | 111MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **1,428** | 76.88ms | 4.24ms | 248.18ms | 1.09s | 111.9MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **1,452** | 108.44ms | 34.14ms | 212.10ms | 1.42s | 115.7MiB |

