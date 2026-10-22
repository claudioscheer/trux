# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **167,308** | 0.96ms | 486.00us | 1.21ms | 6.16ms | 91.7MiB |
| Plaintext (/plaintext) | 1000 | **178,487** | 6.23ms | 5.58ms | 13.27ms | 24.79ms | 94.95MiB |
| SQLite File Insert & Query (/json) | 100 | **760** | 59.72ms | 5.26ms | 10.99ms | 1.44s | 99.28MiB |
| SQLite File Insert & Query (/json) | 1000 | **572** | 384.75ms | 173.71ms | 1.20s | 1.84s | 122.5MiB |
| SQLite File Prepared Query (/users/1024) | 100 | **15,262** | 27.64ms | 514.00us | 61.65ms | 440.28ms | 106MiB |
| SQLite File Prepared Query (/users/1024) | 1000 | **14,473** | 64.74ms | 59.74ms | 124.69ms | 196.35ms | 111.3MiB |
| Middleware & SQLite File Audit (/api/data) | 100 | **774** | 89.20ms | 7.04ms | 179.07ms | 1.58s | 111.8MiB |
| Middleware & SQLite File Audit (/api/data) | 1000 | **804** | 98.65ms | 56.22ms | 82.64ms | 1.08s | 122.2MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **188,964** | 753.63us | 427.00us | 1.15ms | 3.28ms | 91.5MiB |
| Plaintext (/plaintext) | 1000 | **331,785** | 3.66ms | 2.80ms | 8.10ms | 17.91ms | 95.03MiB |
| SQLite File Insert & Query (/json) | 100 | **773** | 200.10ms | 6.62ms | 807.51ms | 1.61s | 99.64MiB |
| SQLite File Insert & Query (/json) | 1000 | **806** | 439.41ms | 131.41ms | 1.38s | 1.95s | 110.4MiB |
| SQLite File Prepared Query (/users/1024) | 100 | **21,730** | 5.07ms | 3.71ms | 7.32ms | 12.92ms | 106.8MiB |
| SQLite File Prepared Query (/users/1024) | 1000 | **18,421** | 48.43ms | 48.25ms | 59.19ms | 68.02ms | 111.9MiB |
| Middleware & SQLite File Audit (/api/data) | 100 | **807** | 153.31ms | 6.24ms | 616.49ms | 1.42s | 111.9MiB |
| Middleware & SQLite File Audit (/api/data) | 1000 | **780** | 455.41ms | 209.23ms | 1.36s | 1.86s | 117.4MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **256,123** | 0.86ms | 294.00us | 1.03ms | 8.91ms | 91.46MiB |
| Plaintext (/plaintext) | 1000 | **446,259** | 2.56ms | 1.77ms | 5.64ms | 12.25ms | 95.32MiB |
| SQLite File Insert & Query (/json) | 100 | **808** | 114.64ms | 7.04ms | 435.90ms | 1.42s | 97.71MiB |
| SQLite File Insert & Query (/json) | 1000 | **734** | 324.12ms | 76.43ms | 995.02ms | 1.98s | 100.3MiB |
| SQLite File Prepared Query (/users/1024) | 100 | **12,792** | 3.09ms | 3.15ms | 4.89ms | 6.63ms | 100.6MiB |
| SQLite File Prepared Query (/users/1024) | 1000 | **12,903** | 32.07ms | 31.77ms | 37.73ms | 42.79ms | 103.2MiB |
| Middleware & SQLite File Audit (/api/data) | 100 | **794** | 104.50ms | 5.45ms | 361.77ms | 1.44s | 103.4MiB |
| Middleware & SQLite File Audit (/api/data) | 1000 | **785** | 254.64ms | 67.20ms | 1.01s | 1.91s | 106.3MiB |

