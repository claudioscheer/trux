# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **157,549** | 0.87ms | 526.00us | 1.29ms | 3.64ms | 91.63MiB |
| Plaintext (/plaintext) | 1000 | **168,376** | 6.69ms | 5.92ms | 14.34ms | 27.72ms | 94.75MiB |
| SQLite Insert & Query (/json) | 100 | **18,534** | 16.69ms | 1.01ms | 53.84ms | 96.29ms | 97.98MiB |
| SQLite Insert & Query (/json) | 1000 | **18,301** | 55.46ms | 62.20ms | 118.28ms | 186.31ms | 103.5MiB |
| SQLite Prepared Query (/users/1024) | 100 | **17,001** | 20.79ms | 632.00us | 64.58ms | 116.14ms | 104.8MiB |
| SQLite Prepared Query (/users/1024) | 1000 | **16,315** | 60.26ms | 63.65ms | 122.34ms | 187.12ms | 108.8MiB |
| Middleware & SQLite Audit (/api/data) | 100 | **22,073** | 14.28ms | 0.94ms | 47.17ms | 87.91ms | 109.6MiB |
| Middleware & SQLite Audit (/api/data) | 1000 | **21,487** | 49.37ms | 52.12ms | 111.43ms | 182.43ms | 113.4MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **177,120** | 0.89ms | 454.00us | 1.26ms | 4.98ms | 87.37MiB |
| Plaintext (/plaintext) | 1000 | **298,837** | 3.83ms | 3.04ms | 8.27ms | 17.98ms | 94.78MiB |
| SQLite Insert & Query (/json) | 100 | **19,627** | 4.33ms | 4.25ms | 7.12ms | 9.36ms | 97.89MiB |
| SQLite Insert & Query (/json) | 1000 | **19,232** | 47.12ms | 46.68ms | 58.40ms | 69.81ms | 102.5MiB |
| SQLite Prepared Query (/users/1024) | 100 | **16,705** | 4.93ms | 4.61ms | 8.77ms | 12.38ms | 104.1MiB |
| SQLite Prepared Query (/users/1024) | 1000 | **16,023** | 55.75ms | 55.04ms | 68.82ms | 84.21ms | 108.6MiB |
| Middleware & SQLite Audit (/api/data) | 100 | **20,252** | 4.32ms | 4.10ms | 7.50ms | 10.81ms | 108.8MiB |
| Middleware & SQLite Audit (/api/data) | 1000 | **20,528** | 43.70ms | 43.26ms | 51.88ms | 62.29ms | 112.7MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **241,266** | 0.91ms | 315.00us | 1.12ms | 6.50ms | 91.61MiB |
| Plaintext (/plaintext) | 1000 | **412,084** | 2.76ms | 1.92ms | 5.97ms | 13.42ms | 95.03MiB |
| SQLite Insert & Query (/json) | 100 | **13,622** | 2.99ms | 2.49ms | 5.72ms | 7.95ms | 96.9MiB |
| SQLite Insert & Query (/json) | 1000 | **14,030** | 33.79ms | 34.20ms | 42.56ms | 50.76ms | 98.97MiB |
| SQLite Prepared Query (/users/1024) | 100 | **12,970** | 3.75ms | 3.52ms | 6.54ms | 9.86ms | 100.3MiB |
| SQLite Prepared Query (/users/1024) | 1000 | **12,828** | 37.82ms | 38.13ms | 45.66ms | 51.00ms | 103.2MiB |
| Middleware & SQLite Audit (/api/data) | 100 | **15,442** | 3.53ms | 3.50ms | 6.21ms | 8.72ms | 102.9MiB |
| Middleware & SQLite Audit (/api/data) | 1000 | **15,173** | 32.25ms | 31.26ms | 40.02ms | 47.58ms | 104.9MiB |

