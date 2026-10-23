# Trux HTTP io_uring Benchmark Results

*Kernel: 7.0.0-38-generic*
*Docker Engine: 29.9.0*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **140,731** | 693.87us | 655.00us | 805.00us | 1.11ms | 6.043MiB |
| Plaintext Baseline (/plaintext) | 1000 | **118,127** | 8.45ms | 8.19ms | 9.72ms | 12.72ms | 13.96MiB |
| JSON Serialization (/json/serialize) | 100 | **137,110** | 697.92us | 684.00us | 787.00us | 1.04ms | 13.77MiB |
| JSON Serialization (/json/serialize) | 1000 | **120,816** | 8.20ms | 7.99ms | 9.70ms | 12.51ms | 13.62MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **42,425** | 3.87ms | 1.92ms | 8.32ms | 29.93ms | 24.57MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **39,821** | 24.91ms | 21.66ms | 40.37ms | 52.57ms | 31.19MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **105,662** | 0.90ms | 0.90ms | 0.99ms | 1.19ms | 31.31MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **85,923** | 11.53ms | 11.35ms | 13.13ms | 16.78ms | 31.38MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **44,721** | 3.83ms | 1.81ms | 9.27ms | 29.61ms | 31.8MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **40,099** | 24.92ms | 21.46ms | 48.27ms | 54.93ms | 32.63MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **252,505** | 409.74us | 371.00us | 445.00us | 558.00us | 10.33MiB |
| Plaintext Baseline (/plaintext) | 1000 | **247,527** | 4.06ms | 3.92ms | 4.73ms | 6.10ms | 18.76MiB |
| JSON Serialization (/json/serialize) | 100 | **246,800** | 385.59us | 374.00us | 456.00us | 579.00us | 18.75MiB |
| JSON Serialization (/json/serialize) | 1000 | **240,009** | 4.14ms | 4.06ms | 4.78ms | 6.03ms | 23.19MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **50,121** | 4.89ms | 1.57ms | 8.46ms | 53.90ms | 26.16MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **47,623** | 24.59ms | 17.92ms | 51.95ms | 101.11ms | 28.73MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **170,164** | 570.58us | 559.00us | 757.00us | 1.05ms | 28.74MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **162,594** | 6.10ms | 5.95ms | 7.16ms | 9.14ms | 28.86MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **50,665** | 6.35ms | 1.51ms | 19.34ms | 70.40ms | 29.14MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **51,285** | 21.45ms | 16.75ms | 36.53ms | 76.06ms | 30.85MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **469,148** | 215.19us | 195.00us | 263.00us | 342.00us | 18.9MiB |
| Plaintext Baseline (/plaintext) | 1000 | **466,623** | 2.12ms | 2.06ms | 2.58ms | 3.18ms | 27.46MiB |
| JSON Serialization (/json/serialize) | 100 | **463,319** | 206.95us | 199.00us | 270.00us | 342.00us | 28.57MiB |
| JSON Serialization (/json/serialize) | 1000 | **456,498** | 2.17ms | 2.15ms | 2.61ms | 3.08ms | 34.04MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **44,199** | 5.73ms | 1.85ms | 8.69ms | 63.92ms | 37.39MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **46,802** | 23.93ms | 18.61ms | 42.61ms | 92.56ms | 43.45MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **271,051** | 370.44us | 340.00us | 436.00us | 1.45ms | 42.9MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **267,936** | 3.70ms | 3.61ms | 4.60ms | 7.88ms | 46.43MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **46,239** | 6.75ms | 1.75ms | 19.49ms | 73.64ms | 47.47MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **50,856** | 21.81ms | 17.16ms | 38.31ms | 81.20ms | 51.75MiB |

