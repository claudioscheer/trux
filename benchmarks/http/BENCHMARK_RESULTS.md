# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **163,983** | 1.06ms | 497.00us | 1.25ms | 12.90ms | 91.75MiB |
| Plaintext (/plaintext) | 1000 | **184,936** | 6.04ms | 5.44ms | 12.94ms | 23.88ms | 95.42MiB |
| SQLite File Insert & Query (/json) | 100 | **1,451** | 74.00ms | 2.81ms | 85.44ms | 1.40s | 99.75MiB |
| SQLite File Insert & Query (/json) | 1000 | **1,443** | 82.11ms | 33.28ms | 45.37ms | 1.52s | 104.1MiB |
| SQLite File Prepared Query (/users/1024) | 100 | **110,870** | 0.88ms | 731.00us | 1.58ms | 3.40ms | 104.7MiB |
| SQLite File Prepared Query (/users/1024) | 1000 | **108,854** | 15.12ms | 8.38ms | 43.33ms | 82.83ms | 108.4MiB |
| Middleware & SQLite File Audit (/api/data) | 100 | **1,555** | 13.04ms | 3.16ms | 3.95ms | 373.49ms | 109MiB |
| Middleware & SQLite File Audit (/api/data) | 1000 | **1,547** | 46.95ms | 30.24ms | 33.83ms | 871.78ms | 113.3MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **184,787** | 0.97ms | 436.00us | 1.20ms | 5.19ms | 91.41MiB |
| Plaintext (/plaintext) | 1000 | **325,641** | 3.75ms | 2.82ms | 8.32ms | 19.40ms | 95.45MiB |
| SQLite File Insert & Query (/json) | 100 | **1,370** | 134.68ms | 4.49ms | 544.89ms | 1.31s | 101MiB |
| SQLite File Insert & Query (/json) | 1000 | **1,433** | 266.85ms | 36.82ms | 1.04s | 1.92s | 105.9MiB |
| SQLite File Prepared Query (/users/1024) | 100 | **127,252** | 588.99us | 451.00us | 1.14ms | 2.49ms | 105.7MiB |
| SQLite File Prepared Query (/users/1024) | 1000 | **188,125** | 4.47ms | 3.38ms | 9.98ms | 19.61ms | 109.1MiB |
| Middleware & SQLite File Audit (/api/data) | 100 | **1,504** | 51.11ms | 3.20ms | 42.26ms | 961.91ms | 109.4MiB |
| Middleware & SQLite File Audit (/api/data) | 1000 | **1,579** | 84.38ms | 32.23ms | 45.35ms | 1.53s | 113.2MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **244,235** | 0.99ms | 308.00us | 1.08ms | 15.81ms | 91.8MiB |
| Plaintext (/plaintext) | 1000 | **428,091** | 2.62ms | 1.78ms | 5.75ms | 12.83ms | 95.86MiB |
| SQLite File Insert & Query (/json) | 100 | **1,433** | 119.52ms | 3.46ms | 457.03ms | 1.63s | 98.21MiB |
| SQLite File Insert & Query (/json) | 1000 | **1,395** | 149.20ms | 35.61ms | 417.99ms | 1.48s | 101.8MiB |
| SQLite File Prepared Query (/users/1024) | 100 | **86,035** | 504.68us | 354.00us | 0.99ms | 2.61ms | 102.3MiB |
| SQLite File Prepared Query (/users/1024) | 1000 | **292,992** | 1.81ms | 1.23ms | 4.02ms | 7.89ms | 104.5MiB |
| Middleware & SQLite File Audit (/api/data) | 100 | **1,459** | 110.74ms | 3.74ms | 459.71ms | 1.43s | 105.3MiB |
| Middleware & SQLite File Audit (/api/data) | 1000 | **1,517** | 84.35ms | 28.53ms | 39.52ms | 1.66s | 106.5MiB |

