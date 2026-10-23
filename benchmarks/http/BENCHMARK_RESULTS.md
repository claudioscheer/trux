# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **150,124** | 697.42us | 610.00us | 763.00us | 1.44ms | 7.441MiB |
| Plaintext Baseline (/plaintext) | 1000 | **114,305** | 8.73ms | 8.41ms | 9.83ms | 12.58ms | 15.1MiB |
| JSON Serialization (/json/serialize) | 100 | **149,968** | 632.82us | 608.00us | 775.00us | 1.21ms | 14.77MiB |
| JSON Serialization (/json/serialize) | 1000 | **121,905** | 8.13ms | 8.06ms | 8.92ms | 10.50ms | 14.86MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **63,155** | 3.08ms | 1.24ms | 8.32ms | 24.22ms | 31.05MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **57,662** | 17.39ms | 15.21ms | 23.37ms | 42.12ms | 33.28MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **126,580** | 753.03us | 724.00us | 0.91ms | 1.35ms | 34.31MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **100,104** | 9.87ms | 9.63ms | 10.96ms | 13.72ms | 34.16MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **66,870** | 2.84ms | 1.16ms | 8.18ms | 22.14ms | 34.17MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **61,824** | 16.55ms | 13.74ms | 25.86ms | 43.35ms | 34.14MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **278,853** | 388.13us | 317.00us | 480.00us | 1.05ms | 12.07MiB |
| Plaintext Baseline (/plaintext) | 1000 | **228,951** | 4.36ms | 4.15ms | 5.27ms | 6.71ms | 20.33MiB |
| JSON Serialization (/json/serialize) | 100 | **266,365** | 367.63us | 330.00us | 506.00us | 1.00ms | 20.73MiB |
| JSON Serialization (/json/serialize) | 1000 | **226,984** | 4.37ms | 4.22ms | 5.22ms | 6.41ms | 24.89MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **85,280** | 4.50ms | 0.90ms | 15.59ms | 47.50ms | 27.25MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **76,184** | 18.38ms | 10.27ms | 50.26ms | 89.85ms | 31.87MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **193,071** | 497.80us | 463.00us | 690.00us | 1.24ms | 32MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **180,777** | 5.49ms | 5.25ms | 6.69ms | 8.49ms | 32.92MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **90,254** | 4.79ms | 827.00us | 17.95ms | 48.56ms | 33.61MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **87,907** | 14.28ms | 9.27ms | 34.37ms | 69.39ms | 33.92MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext Baseline (/plaintext) | 100 | **690,048** | 206.87us | 106.00us | 226.00us | 538.00us | 20.03MiB |
| Plaintext Baseline (/plaintext) | 1000 | **703,956** | 1.39ms | 1.26ms | 2.08ms | 3.66ms | 29.3MiB |
| JSON Serialization (/json/serialize) | 100 | **675,776** | 143.42us | 111.00us | 221.00us | 467.00us | 29.8MiB |
| JSON Serialization (/json/serialize) | 1000 | **695,237** | 1.39ms | 1.28ms | 2.04ms | 3.30ms | 36.12MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 100 | **59,584** | 3.75ms | 1.36ms | 4.56ms | 43.06ms | 38.04MiB |
| SQLite File Insert & Query (/sqlite/insert-and-query) | 1000 | **70,082** | 16.00ms | 12.30ms | 28.60ms | 63.43ms | 45.67MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 100 | **250,387** | 378.30us | 330.00us | 642.00us | 1.01ms | 45.27MiB |
| SQLite File Prepared Query (/sqlite/query-by-id/1024) | 1000 | **254,878** | 3.89ms | 3.71ms | 5.52ms | 7.92ms | 48.28MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 100 | **59,644** | 3.88ms | 1.36ms | 4.74ms | 44.85ms | 49.11MiB |
| Middleware & SQLite File Audit (/api/authenticated-audit) | 1000 | **76,429** | 15.08ms | 11.09ms | 30.38ms | 60.99ms | 55.93MiB |

