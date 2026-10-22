# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **167,368** | 0.97ms | 472.00us | 1.25ms | 7.01ms | 9.547MiB |
| Plaintext Baseline (/plaintext) | 1000 | **177,898** | 5.73ms | 5.39ms | 6.70ms | 11.66ms | 15.78MiB |
| JSON Serialization (/json/serialize) | 100 | **161,407** | 661.98us | 494.00us | 1.25ms | 2.90ms | 16.49MiB |
| JSON Serialization (/json/serialize) | 1000 | **171,448** | 5.75ms | 5.62ms | 6.82ms | 9.19ms | 15.62MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **60,884** | 2.42ms | 1.18ms | 7.07ms | 13.61ms | 30.38MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **39,332** | 26.19ms | 26.15ms | 47.06ms | 78.43ms | 33.84MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **139,661** | 733.52us | 581.00us | 1.30ms | 2.84ms | 33.59MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **140,875** | 7.01ms | 6.84ms | 8.07ms | 10.58ms | 33.93MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **66,057** | 2.20ms | 1.08ms | 6.57ms | 10.25ms | 34.18MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **63,745** | 15.52ms | 17.17ms | 20.35ms | 27.63ms | 33.88MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **180,050** | 1.69ms | 437.00us | 1.29ms | 41.19ms | 13.65MiB |
| Plaintext Baseline (/plaintext) | 1000 | **311,405** | 3.26ms | 2.94ms | 4.37ms | 7.93ms | 21.14MiB |
| JSON Serialization (/json/serialize) | 100 | **176,968** | 608.99us | 458.00us | 1.22ms | 2.68ms | 20.53MiB |
| JSON Serialization (/json/serialize) | 1000 | **296,372** | 3.32ms | 3.10ms | 4.67ms | 7.34ms | 25.43MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **51,594** | 2.48ms | 1.51ms | 6.65ms | 11.26ms | 26.38MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **70,624** | 13.99ms | 14.63ms | 21.24ms | 27.73ms | 30.38MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **131,149** | 766.26us | 635.00us | 1.37ms | 2.82ms | 30.81MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **207,490** | 4.74ms | 4.55ms | 6.42ms | 8.62ms | 31.57MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **54,067** | 2.47ms | 1.43ms | 6.90ms | 11.37ms | 31.11MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **73,031** | 13.59ms | 14.33ms | 20.85ms | 27.34ms | 31.99MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **223,732** | 1.05ms | 340.00us | 1.18ms | 13.01ms | 22.18MiB |
| Plaintext Baseline (/plaintext) | 1000 | **430,467** | 2.76ms | 1.83ms | 5.38ms | 13.99ms | 29.78MiB |
| JSON Serialization (/json/serialize) | 100 | **207,778** | 565.51us | 380.00us | 1.20ms | 3.04ms | 30.35MiB |
| JSON Serialization (/json/serialize) | 1000 | **420,551** | 2.62ms | 1.89ms | 5.43ms | 11.70ms | 36.25MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **37,408** | 3.10ms | 2.18ms | 7.10ms | 13.31ms | 38.55MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **52,727** | 18.74ms | 18.47ms | 29.42ms | 37.48ms | 45.23MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **146,638** | 695.69us | 570.00us | 1.26ms | 2.53ms | 45.59MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **252,317** | 3.95ms | 3.64ms | 6.66ms | 10.93ms | 48.13MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **38,225** | 3.06ms | 2.16ms | 7.10ms | 13.05ms | 48.77MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **53,626** | 18.50ms | 18.21ms | 29.47ms | 37.70ms | 52.16MiB |

