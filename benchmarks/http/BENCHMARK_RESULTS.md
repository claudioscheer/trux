# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **173,596** | 0.99ms | 476.00us | 1.15ms | 10.40ms | 91.7MiB |
| Plaintext Baseline (/plaintext) | 1000 | **179,170** | 6.39ms | 5.51ms | 13.83ms | 27.01ms | 94.93MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **37,405** | 3.54ms | 1.83ms | 9.09ms | 20.55ms | 99.82MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **38,948** | 53.53ms | 17.27ms | 159.50ms | 301.04ms | 103.9MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **110,655** | 797.37us | 643.00us | 1.41ms | 3.05ms | 104.1MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **106,370** | 14.46ms | 6.98ms | 41.78ms | 81.86ms | 106.2MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **45,157** | 3.04ms | 1.40ms | 8.18ms | 17.60ms | 108.2MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **46,486** | 45.04ms | 2.86ms | 135.57ms | 273.89ms | 111MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **192,070** | 763.92us | 430.00us | 1.12ms | 3.37ms | 87.7MiB |
| Plaintext Baseline (/plaintext) | 1000 | **322,706** | 3.79ms | 2.86ms | 8.40ms | 19.24ms | 95.31MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **38,515** | 2.81ms | 1.83ms | 7.20ms | 11.37ms | 101.1MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **37,797** | 23.48ms | 23.55ms | 30.91ms | 41.36ms | 105.5MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **146,633** | 625.42us | 499.00us | 1.18ms | 2.45ms | 104.8MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **188,478** | 5.82ms | 4.40ms | 12.99ms | 26.70ms | 108.7MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **40,513** | 2.63ms | 1.59ms | 7.02ms | 11.01ms | 109.2MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **41,417** | 21.12ms | 21.31ms | 29.84ms | 40.09ms | 111.6MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **243,186** | 846.47us | 313.00us | 1.04ms | 6.10ms | 91.7MiB |
| Plaintext Baseline (/plaintext) | 1000 | **431,069** | 2.69ms | 1.88ms | 5.92ms | 13.35ms | 95.18MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **21,909** | 2.81ms | 2.11ms | 5.67ms | 12.31ms | 98.1MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **21,381** | 27.95ms | 27.94ms | 37.25ms | 46.80ms | 102.1MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **115,832** | 539.52us | 409.00us | 1.06ms | 2.39ms | 102MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **298,624** | 2.03ms | 1.29ms | 4.74ms | 9.37ms | 103.4MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **23,219** | 2.83ms | 2.10ms | 5.94ms | 11.60ms | 103MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **22,651** | 26.38ms | 26.29ms | 35.47ms | 44.20ms | 106.4MiB |

