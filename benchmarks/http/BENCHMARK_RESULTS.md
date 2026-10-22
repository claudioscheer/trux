# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **162,943** | 1.00ms | 504.00us | 1.22ms | 4.95ms | 91.69MiB |
| Plaintext (/plaintext) | 1000 | **179,249** | 6.23ms | 5.54ms | 13.36ms | 25.14ms | 95.03MiB |
| SQLite Insert & Query (/json) | 100 | **19,990** | 15.51ms | 646.00us | 50.76ms | 90.00ms | 100.5MiB |
| SQLite Insert & Query (/json) | 1000 | **20,340** | 48.17ms | 53.99ms | 106.93ms | 178.15ms | 109.6MiB |
| SQLite Prepared Query (/users/1024) | 100 | **17,509** | 17.57ms | 483.00us | 56.78ms | 101.91ms | 119MiB |
| SQLite Prepared Query (/users/1024) | 1000 | **16,721** | 55.29ms | 62.31ms | 114.56ms | 171.40ms | 131.5MiB |
| Middleware & SQLite Audit (/api/data) | 100 | **23,316** | 12.06ms | 1.38ms | 40.23ms | 82.92ms | 135.8MiB |
| Middleware & SQLite Audit (/api/data) | 1000 | **22,728** | 43.66ms | 46.16ms | 98.64ms | 162.56ms | 144MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **188,426** | 1.13ms | 431.00us | 1.18ms | 16.12ms | 91.37MiB |
| Plaintext (/plaintext) | 1000 | **323,778** | 3.66ms | 2.86ms | 8.20ms | 17.05ms | 94.18MiB |
| SQLite Insert & Query (/json) | 100 | **20,293** | 4.58ms | 3.73ms | 7.71ms | 13.90ms | 101.3MiB |
| SQLite Insert & Query (/json) | 1000 | **19,321** | 44.67ms | 44.47ms | 55.28ms | 69.53ms | 109.6MiB |
| SQLite Prepared Query (/users/1024) | 100 | **17,226** | 5.05ms | 4.77ms | 8.98ms | 12.94ms | 119.4MiB |
| SQLite Prepared Query (/users/1024) | 1000 | **17,676** | 49.39ms | 49.17ms | 60.30ms | 67.70ms | 133MiB |
| Middleware & SQLite Audit (/api/data) | 100 | **21,679** | 3.84ms | 3.72ms | 6.49ms | 9.15ms | 138MiB |
| Middleware & SQLite Audit (/api/data) | 1000 | **21,079** | 42.64ms | 42.84ms | 51.14ms | 61.92ms | 146.3MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **237,913** | 841.86us | 313.00us | 1.09ms | 6.00ms | 91.54MiB |
| Plaintext (/plaintext) | 1000 | **425,079** | 2.65ms | 1.86ms | 5.78ms | 12.55ms | 94.39MiB |
| SQLite Insert & Query (/json) | 100 | **14,958** | 2.92ms | 2.79ms | 4.79ms | 6.26ms | 98.93MiB |
| SQLite Insert & Query (/json) | 1000 | **14,869** | 32.18ms | 32.04ms | 38.49ms | 47.61ms | 105MiB |
| SQLite Prepared Query (/users/1024) | 100 | **13,319** | 2.84ms | 2.76ms | 3.94ms | 5.07ms | 112.2MiB |
| SQLite Prepared Query (/users/1024) | 1000 | **13,234** | 37.94ms | 37.31ms | 46.07ms | 50.51ms | 120.6MiB |
| Middleware & SQLite Audit (/api/data) | 100 | **15,970** | 2.61ms | 2.51ms | 4.32ms | 5.94ms | 123.8MiB |
| Middleware & SQLite Audit (/api/data) | 1000 | **15,824** | 31.79ms | 32.27ms | 37.87ms | 44.15ms | 129.4MiB |

