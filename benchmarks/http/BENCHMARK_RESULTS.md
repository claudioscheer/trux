# Trux HTTP io_uring Benchmark Results

*Kernel: 6.18.54.1-microsoft-standard-WSL2*
*Docker Engine: 29.1.3*
*Duration per test: 6s*

### Micro Edge (1 vCPU, 256MB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **168,967** | 1.01ms | 492.00us | 1.20ms | 9.09ms | 91.57MiB |
| Plaintext (/plaintext) | 1000 | **182,076** | 6.22ms | 5.53ms | 13.35ms | 25.43ms | 94.78MiB |
| JSON Serialization (/json) | 100 | **162,049** | 590.73us | 459.00us | 1.13ms | 2.53ms | 95.58MiB |
| JSON Serialization (/json) | 1000 | **176,102** | 5.73ms | 4.95ms | 12.42ms | 23.22ms | 98.64MiB |
| URL Param (/users/1024) | 100 | **155,291** | 600.73us | 469.00us | 1.14ms | 2.50ms | 98.36MiB |
| URL Param (/users/1024) | 1000 | **173,273** | 5.79ms | 5.20ms | 12.46ms | 21.69ms | 101.8MiB |
| Middleware & Auth (/api/data) | 100 | **152,164** | 598.38us | 471.00us | 1.14ms | 2.42ms | 102.4MiB |
| Middleware & Auth (/api/data) | 1000 | **170,151** | 5.88ms | 5.16ms | 12.93ms | 22.93ms | 106.4MiB |

### Cloud Standard (2 vCPUs, 1GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **177,334** | 835.66us | 453.00us | 1.26ms | 4.04ms | 91.56MiB |
| Plaintext (/plaintext) | 1000 | **315,934** | 3.74ms | 2.92ms | 8.23ms | 17.98ms | 94.85MiB |
| JSON Serialization (/json) | 100 | **179,575** | 558.56us | 418.00us | 1.13ms | 2.54ms | 95.36MiB |
| JSON Serialization (/json) | 1000 | **314,041** | 3.57ms | 2.73ms | 7.87ms | 16.75ms | 98.8MiB |
| URL Param (/users/1024) | 100 | **162,708** | 573.82us | 417.00us | 1.17ms | 2.74ms | 99.29MiB |
| URL Param (/users/1024) | 1000 | **313,081** | 3.63ms | 2.84ms | 8.13ms | 16.73ms | 103.1MiB |
| Middleware & Auth (/api/data) | 100 | **171,384** | 599.60us | 441.00us | 1.20ms | 2.79ms | 103.6MiB |
| Middleware & Auth (/api/data) | 1000 | **304,410** | 3.76ms | 2.89ms | 8.56ms | 17.61ms | 107.1MiB |

### Compute Node (4 vCPUs, 2GB)

| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Plaintext (/plaintext) | 100 | **239,323** | 671.02us | 314.00us | 1.10ms | 4.43ms | 91.48MiB |
| Plaintext (/plaintext) | 1000 | **419,373** | 2.75ms | 1.89ms | 5.93ms | 13.83ms | 94.77MiB |
| JSON Serialization (/json) | 100 | **65,073** | 433.20us | 266.00us | 0.90ms | 2.81ms | 95.06MiB |
| JSON Serialization (/json) | 1000 | **407,081** | 0.92ms | 524.00us | 2.05ms | 5.89ms | 95.86MiB |
| URL Param (/users/1024) | 100 | **83,670** | 444.59us | 279.00us | 0.96ms | 2.72ms | 96.36MiB |
| URL Param (/users/1024) | 1000 | **397,040** | 0.94ms | 530.00us | 2.05ms | 5.98ms | 98MiB |
| Middleware & Auth (/api/data) | 100 | **81,342** | 464.75us | 288.00us | 1.00ms | 2.93ms | 98.17MiB |
| Middleware & Auth (/api/data) | 1000 | **399,442** | 0.96ms | 544.00us | 2.18ms | 6.09ms | 99.8MiB |

