# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **160,763** | 0.91ms | 496.00us | 1.28ms | 4.35ms | 9.305MiB |
| Plaintext Baseline (/plaintext) | 1000 | **179,512** | 5.52ms | 5.36ms | 6.64ms | 9.39ms | 14.84MiB |
| JSON Serialization (/json/serialize) | 100 | **162,415** | 640.60us | 498.00us | 1.22ms | 2.59ms | 16MiB |
| JSON Serialization (/json/serialize) | 1000 | **173,301** | 5.70ms | 5.58ms | 6.77ms | 9.32ms | 16.55MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **60,085** | 2.35ms | 1.20ms | 6.88ms | 10.93ms | 30.58MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **60,090** | 16.46ms | 18.55ms | 21.45ms | 28.62ms | 33.78MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **138,780** | 724.60us | 586.00us | 1.27ms | 2.75ms | 33.76MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **140,120** | 7.07ms | 6.94ms | 8.10ms | 10.23ms | 34.08MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **63,859** | 2.28ms | 1.09ms | 6.88ms | 10.70ms | 34.04MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **62,761** | 15.84ms | 17.72ms | 21.28ms | 27.76ms | 35.2MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **180,543** | 1.30ms | 436.00us | 1.28ms | 27.15ms | 13.55MiB |
| Plaintext Baseline (/plaintext) | 1000 | **308,128** | 3.24ms | 2.95ms | 4.46ms | 7.93ms | 20.51MiB |
| JSON Serialization (/json/serialize) | 100 | **174,804** | 623.01us | 459.00us | 1.26ms | 2.88ms | 21.08MiB |
| JSON Serialization (/json/serialize) | 1000 | **300,312** | 3.26ms | 3.08ms | 4.57ms | 6.90ms | 24MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **51,151** | 2.55ms | 1.52ms | 6.93ms | 11.55ms | 25.99MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **67,381** | 14.65ms | 15.58ms | 22.16ms | 28.63ms | 29.43MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **128,390** | 784.52us | 657.00us | 1.39ms | 2.77ms | 29.56MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **197,050** | 5.00ms | 4.80ms | 6.85ms | 9.26ms | 30.48MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **53,159** | 2.52ms | 1.44ms | 7.06ms | 11.89ms | 31.25MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **73,039** | 13.55ms | 14.27ms | 20.66ms | 26.94ms | 31.62MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **217,847** | 1.03ms | 350.00us | 1.23ms | 14.72ms | 22.7MiB |
| Plaintext Baseline (/plaintext) | 1000 | **431,177** | 2.57ms | 1.79ms | 5.49ms | 11.64ms | 30.36MiB |
| JSON Serialization (/json/serialize) | 100 | **219,225** | 537.80us | 357.00us | 1.15ms | 2.97ms | 30.34MiB |
| JSON Serialization (/json/serialize) | 1000 | **427,035** | 2.60ms | 1.82ms | 5.48ms | 12.05ms | 36.09MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **37,348** | 3.11ms | 2.18ms | 7.12ms | 13.45ms | 38.35MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **52,767** | 18.74ms | 18.63ms | 29.57ms | 37.45ms | 45.74MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **149,781** | 676.07us | 555.00us | 1.24ms | 2.55ms | 45.83MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **262,263** | 3.80ms | 3.52ms | 6.44ms | 10.14ms | 48.19MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **38,507** | 3.01ms | 2.12ms | 6.88ms | 12.92ms | 49.12MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **54,467** | 18.21ms | 17.87ms | 28.41ms | 36.74ms | 51.35MiB |

