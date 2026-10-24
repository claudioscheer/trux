# HTTP, CPU, and GPU Hardware Extraction: Issue Map & Optimization Plan

**Date:** 2026-10-23  
**Scope:** `internal/runtime/c/http.go`, `internal/runtime/c/rt.go`, `internal/runtime/c/json.go`, `internal/runtime/c/sqlite.go`, `internal/codegen/c/c.go`, `cmd/trux/compile.go`, `benchmarks/digits/`, `examples/projects/shakespeare_server/`

---

## 1. Complete Map of Identified Issues

### 1.1 HTTP & `io_uring` Runtime (`internal/runtime/c/http.go`)

| ID | Severity | Location | Issue & Root Cause | Hardware / Runtime Impact |
| :--- | :--- | :--- | :--- | :--- |
| **HTTP-1** | **High (Perf)** | `http.go:711-762` | `rt_http_dispatch` initializes `rt_arena arena` (`owner = NULL`) and `rt_context ctx` on the stack and calls `rt_context_free_chunks(&ctx)` + `rt_arena_deinit(&arena)` at the end of **every request**. | Every request that allocates even 1 byte in the request or frame arena calls libc `malloc(4120)` and `free()` instead of reusing worker-local memory. |
| **HTTP-2** | **Critical (Blocking)** | `http.go:773` | `rt_http_drain_conn` begins with `if (g_rt_http_parked != NULL) return;`. | When **one** connection parks on `gpu.finish`, **all other connections on that worker thread are frozen**—even pure-CPU routes (`/health`, `/plaintext`, `/json`) cannot parse or dispatch until the GPU finishes. |
| **HTTP-3** | **High (Perf / Fairness)** | `http.go:840-846`, `890` | `rt_http_drain_ready` linearly scans `for (int fd = 0; fd < RT_MAX_CONNS; fd++)` across all 65,536 `rt_http_conn` structs (4 MiB per worker) on every GPU completion. | Thrashes L1/L2/L3 cache on every GPU completion and starves higher FDs whenever lower FDs submit back-to-back GPU work. |
| **HTTP-4** | **High (Bug / Leak)** | `http.go:151`, `633-647`, `707` | `rt_http_arm_gpu_timeout` submits a 1s `io_uring_prep_timeout` SQE on every `gpu.finish`, cycling over `g_rt_http_gpu_ts[8]`, and **never cancels it** when the GPU completes early. | At 5,000 GPU req/s, 5,000 dead timers per second accumulate in the kernel, mutating live `timespec` structs and flooding the CQ ring with stale timeout CQEs. |
| **HTTP-5** | **High (Correctness)** | `http.go:468-517`, `1026-1042` | 1. `RT_HTTP_UDATA_SEND` completion clears `tx_in_flight` without checking if `cqe->res < total_len` (partial TCP send).<br>2. Pipelined requests in `rt_http_drain_conn` see `tx_in_flight == true` on the 2nd request and fall back to synchronous `writev`, racing with the queued `io_uring` send. | 1. Responses larger than socket send buffer space are silently truncated.<br>2. HTTP/1.1 pipelined responses can interleave or drop connections on `EAGAIN`. |
| **HTTP-6** | **Medium (DoS / Leak)** | `http.go:43-45`, `1051-1074` | `RT_HTTP_MAX_HEADER_BYTES` (65536) and `RT_HTTP_READ_TIMEOUT_SECONDS` (15) are defined but **never used**. Per-FD `c->buffer` and `c->tx_buf` are never freed or shrunk on `close(client_fd)`. | Unbounded `realloc` on header flood (OOM); permanent heap high-water mark across FDs. Also `docs/HTTP.md:115` says 1 MiB max body while `RT_HTTP_MAX_BODY_BYTES` is 10 MiB. |
| **HTTP-7** | **Medium (Reliability)** | `http.go:1049-1055` | Multishot `recv` treats `res <= 0` as fatal and closes `client_fd`, including `res == -ENOBUFS` when the 1024-buffer ring temporarily empties during traffic spikes. | Active connections are dropped under bursty load instead of re-arming `recv`. |
| **HTTP-8** | **Medium (Perf)** | `http.go:355-394`, `431-486`, `1057-1080` | **4x redundant memory copies per request:**<br>1. Kernel buffer ring `in_data` $\to$ `memcpy` into `c->buffer` (`1072`).<br>2. `c->buffer` $\to$ `rt_http_clone_string` on every `http.body`/`path`/`param`/`header` call (`355-394`).<br>3. `snprintf` into stack `char header[4096]` (`432`).<br>4. `memcpy` of `header` + `body` into `c->tx_buf` (`483-485`). | Wastes memory bandwidth, L1 cache capacity, and CPU cycles (`snprintf` format parsing) on every request. |
| **HTTP-9** | **Medium (Perf)** | `http.go:920-1105`, `1149` | 1. `io_uring_queue_init` uses flags `0` (missing `SINGLE_ISSUER`, `COOP_TASKRUN`, `DEFER_TASKRUN`).<br>2. Uses separate `io_uring_wait_cqe` + `io_uring_submit` syscalls per loop turn.<br>3. Calls `io_uring_cq_advance(&ring, 1)` and `io_uring_buf_ring_advance(br, 1)` inside the CQE loop.<br>4. Worker threads are not pinned to CPU cores (`pthread_setaffinity_np`). | Extra syscalls, cross-core IPIs preempting user handlers, repeated atomic store-release barriers per CQE, and scheduler thread migrations. |

---

### 1.2 CPU Runtime, JSON, SQLite, & Compiler Codegen

| ID | Severity | Location | Issue & Root Cause | Hardware / Runtime Impact |
| :--- | :--- | :--- | :--- | :--- |
| **CPU-1** | **Critical (Perf)** | `rt.go:147-166` | `rt_mul_i64_at` checks signed 64-bit multiplication overflow using 6 branches and **64-bit integer division** (`left > INT64_MAX / right`) on every `int * int` operation, and calls `exit(1)` on failure. | `idiv` costs 15–40 cycles on every index calculation (`row * width + col`) and **blocks GCC/Clang from auto-vectorizing (AVX2/AVX-512) any loop**. |
| **CPU-2** | **High (Perf)** | `json.go:276-291`, `317-355` | `rt_json_new_doc` calls `yyjson_mut_doc_new(NULL)` (heap `malloc`) + `malloc(sizeof(rt_list_allocation))` instead of using `yyjson_alc` on the request arena. `rt_json_add_*` copies compile-time key literals via `yyjson_mut_strncpy`. | 4 heap `malloc`s and 4 `free`s on every JSON serialization response. |
| **CPU-3** | **Medium (Perf)** | `sqlite.go:175-203`, `587-593` | `rt_sqlite_open_thread_local` calls `rt_string_to_c_string` (`malloc` + NUL scan + `memcpy` + `free`) on every cache hit. `rt_sqlite_column_text` uses `strlen` instead of `sqlite3_column_bytes`. | Unnecessary heap allocation on every SQLite HTTP request. |
| **CPU-4** | **Medium (Bug)** | `cmd/trux/compile.go:239-275` | In `compileGenerated`, the `if opts.UsesCUDA` branch links HTTP, JSON, and JWT, but **omits `if opts.UsesSQLite`**. | Programs that use both a GPU `kernel func` and `sqlite` fail to link under `nvcc`. |
| **CPU-5** | **Medium (Perf)** | `cmd/trux/compile.go:237, 253` | Host C/C++ compilation uses `-O2` without `-O3 -march=native`. | Misses FMA, AVX2/AVX-512 loop vectorization, and aggressive inlining on the target CPU. |
| **CPU-6** | **Low (Docs)** | `docs/OPEN_ISSUES.md:21-25` | Lists `gpu.globalX()` in host code as an open issue, though `internal/types/types.go:1702` already rejects it. | Stale documentation. |

---

### 1.3 GPU Runtime, CUDA Codegen, & Kernels

| ID | Severity | Location | Issue & Root Cause | Hardware / Runtime Impact |
| :--- | :--- | :--- | :--- | :--- |
| **GPU-1** | **Critical (Perf)** | `rt.go:1990-1996` | `rt_gpu_submit_signal` (`gpu.finish`) loops over **all** declared workspaces (`0 .. g_rt_gpu_ws_decl_count`) and queues `cudaMemcpyAsync(..., cudaMemcpyDeviceToHost)` for every single one. | Copies input buffers and all intermediate activation buffers across PCIe on every request, even when only the final output buffer is read by `gpu.ready`. |
| **GPU-2** | **Critical (Perf)** | `rt.go:203-219` | `rt_mul_i64_device` executes `INT64_MAX / right` (64-bit signed division) on every `int * int` inside CUDA kernels. | NVIDIA GPUs have **no hardware 64-bit integer divider**; every index multiply (`sv * features`, `row * inDim`) compiles to a ~20-instruction software division subroutine. |
| **GPU-3** | **High (Perf)** | `c.go:3185-3198`, `compile.go:247` | 1. `emitKernelType` emits raw `double*` / `int64_t*` without `__restrict__`.<br>2. `nvcc` is invoked without `-arch=native`. | 1. `nvcc` assumes pointer aliasing across all buffers, disabling read-only cache (`__ldg`) loads and loop unrolling.<br>2. `nvcc` compiles for the oldest default virtual architecture instead of the host GPU's SM version. |
| **GPU-4** | **High (Blocking)** | `shakespeare_server/main.tx:472, 484`, `rt.go:1792, 1809` | 1. `gpu.sync()` calls `cudaDeviceSynchronize()`, stalling **all streams across the entire GPU** and blocking the HTTP worker thread for ~137 ms.<br>2. `gpu.to` uses blocking `cudaMemcpy` on stream `0` and copies all `n * vocabSize` rows instead of the last row (`65` floats). | Blocks the HTTP worker thread for the entire 60-token generation and serializes multi-worker GPU execution. |
| **GPU-5** | **High (Perf)** | `rt.go:660-708`, `1999` | 1. `rt_http_retain_parked` + `rt_http_park_begin` performs 4 heap `malloc`/`free` calls per `gpu.finish` and frees arena chunks on completion.<br>2. Completion relies solely on `cudaLaunchHostFunc` $\to$ driver callback thread $\to$ `write(eventfd)` $\to$ `io_uring` wake $\to$ `read(eventfd)`. | 4 heap allocations + 2 thread context switches + 2 syscalls added to every GPU request's latency. |
| **GPU-6** | **High (Perf)** | `digits/server.tx:13-20`, `shakespeare_server/main.tx:20-117` | 1. **Uncoalesced global memory access:** Adjacent threads `sv` and `sv+1` access `support[sv * 64 + i]` (512 bytes apart = 32 memory transactions per warp instead of 1). Same in `linearK`.<br>2. **SM underutilization:** `layerNormK` launches `n` threads (e.g. 8 threads = 1/4 warp on 1 SM!) with 248 idle threads in the block.<br>3. **No KV cache:** `shakespeare_server` recomputes all $1..n$ tokens on every step (1,500 kernel launches per request).<br>4. **CPU-side reduction:** `digits/server.tx` copies all 519 `scores` to host to run `vote` on CPU instead of reducing on GPU. | Leaves >90% of GPU memory bandwidth and SM compute idle. |
| **GPU-7** | **High (Hardware Limit)** | `rt.go:1709-1710` | GPU buffers and TXM tensors only support 64-bit `double` (FP64) and `int64_t`. | On standard NVIDIA GPUs (Turing/Ampere/Ada/Hopper consumer & L4/A10 datacenter), FP64 throughput is **1/32 to 1/64 of FP32**, and uses **2x PCIe and VRAM bandwidth**. |

---

## 2. Phased Implementation Plan

### Phase 1: Compiler, Arithmetic, & Codegen Hardware Unlock (CPU + GPU)
**Goal:** Eliminate the 64-bit division bottleneck in integer multiplication on both CPU and GPU, enable C/CUDA pointer aliasing optimizations, pass native architecture flags, and fix CUDA+SQLite linking.

#### 1.1 Fast Hardware Overflow Checks (`internal/runtime/c/rt.go:133-234`)
- **Host (`rt_add_i64_at`, `rt_sub_i64_at`, `rt_mul_i64_at`):**
  Use compiler intrinsics `__builtin_add_overflow`, `__builtin_sub_overflow`, and `__builtin_mul_overflow` when compiling with GCC/Clang (`#if defined(__GNUC__) || defined(__clang__)`):
  ```c
  static RT_UNUSED int64_t rt_mul_i64_at(int64_t left, int64_t right, const char* file, int line) {
  #if defined(__GNUC__) || defined(__clang__)
      int64_t res;
      if (__builtin_expect(__builtin_mul_overflow(left, right, &res), 0)) {
          rt_arithmetic_fail("overflow in *", file, line);
      }
      return res;
  #else
      // existing fallback
  #endif
  }
  ```
- **Device (`rt_add_i64_device`, `rt_sub_i64_device`, `rt_mul_i64_device` in `rt.go:189-220`):**
  Replace `INT64_MAX / right` software division in `rt_mul_i64_device` with CUDA's 2-instruction `__mul64hi` intrinsic:
  ```c
  __device__ static int64_t rt_mul_i64_device(int64_t left, int64_t right) {
      int64_t res = (int64_t)((uint64_t)left * (uint64_t)right);
      int64_t hi = __mul64hi(left, right);
      if (hi != (res >> 63)) {
          asm("trap;");
      }
      return res;
  }
  ```

#### 1.2 Kernel `__restrict__` & Native Compiler Flags (`internal/codegen/c/c.go`, `cmd/trux/compile.go`)
- **`internal/codegen/c/c.go:3185-3198` (`emitKernelType`):**
  Emit `elem + "* __restrict__"` for `*ast.GPUBufferType` kernel parameters so `nvcc` uses read-only cache loads (`__ldg`) and unrolls loops.
- **`cmd/trux/compile.go:233-300` (`compileGenerated`):**
  - Non-debug C builds: pass `-O3 -march=native`.
  - Non-debug CUDA builds: pass `-O3 -arch=native -Xcompiler -O3,-march=native`.
  - Add the missing `if opts.UsesSQLite` block inside `if opts.UsesCUDA` (using `host` compiler to build `sqliteObj` and passing `sqliteObj` to `nvcc`).

---

### Phase 2: Zero-Allocation HTTP Request Lifecycle (CPU Memory & JSON/SQLite)
**Goal:** Achieve **zero `malloc`/`free` calls per request** across plaintext, JSON, SQLite, and GPU endpoints.

#### 2.1 Worker-Local Arena Pooling (`internal/runtime/c/http.go`, `internal/runtime/c/rt.go`)
- In `rt_http_worker_thread` (`http.go:945`), initialize a persistent worker `rt_context worker_ctx` with a pre-warmed `chunk_pool`.
- In `rt_http_dispatch` (`http.go:710`):
  - Bind the request's `arena.owner = &worker_ctx`.
  - On request completion (`http.go:761`), call `rt_arena_deinit(&arena)` which returns all request chunks directly to `worker_ctx.chunk_pool` (instead of `rt_context_free_chunks` which `free`d them to libc!).
  - Cap `worker_ctx.chunk_pool` at a reasonable high-water mark (e.g., 16 MiB per worker) so oversized one-off chunks don't stay pinned forever.
- Pre-allocate the parked GPU request struct (`rt_http_gpu_park`, `rt_http_request`, `rt_arena`, `rt_context`) inline on the worker thread instead of calling `malloc`/`free` 4 times per `gpu.finish` in `rt_http_retain_parked` (`http.go:660`) and `rt_http_park_begin` (`rt.go:386`).

#### 2.2 Arena-Backed `json.newDoc()` (`internal/runtime/c/json.go:276-291`)
- Change `rt_json_new_doc(rt_arena* arena)` and `codegen` so `json.newDoc()` allocates via `yyjson_alc alc = { rt_yyjson_malloc, rt_yyjson_realloc, rt_yyjson_free, trux_ctx->arena }` on the durable request arena (which is never rewound by inner loop marks).
- Remove `malloc(sizeof(rt_list_allocation))` and `yyjson_mut_doc_free` tracking for `MutDoc` since the request arena reclaims all chunks in $O(1)$ at the end of the request.

#### 2.3 Zero-Malloc SQLite Thread-Local Lookup (`internal/runtime/c/sqlite.go:175-203`, `587-593`)
- In `rt_sqlite_open_thread_local`, compare `path.len` and `memcmp(entry.path, path.data, path.len)` directly against `g_trux_sqlite_tls_cache` before calling `rt_string_to_c_string`.
- In `rt_sqlite_column_text`, use `sqlite3_column_bytes(stmt->handle, (int)col)` instead of `strlen`.

---

### Phase 3: Non-Blocking HTTP + GPU Continuation Pipeline (`io_uring` $\leftrightarrow$ CUDA)
**Goal:** Eliminate worker head-of-line blocking, fix the $O(65,536)$ FD scan, fix the `io_uring` timeout leak, and eliminate redundant PCIe DMA transfers.

#### 3.1 Unblock CPU Routes & Replace $O(65,536)$ Scan (`internal/runtime/c/http.go:770-891`)
- **Remove line 773 (`if (g_rt_http_parked != NULL) return;`) from `rt_http_drain_conn`:**
  Allow all non-GPU requests on the worker to parse, dispatch, and respond immediately even while a GPU job is in flight.
- **Intrusive Ready Queue:**
  Add `int next_ready_fd;` and `bool in_ready_queue;` to `rt_http_conn`, plus per-worker `ready_head` / `ready_tail`.
  If a connection has unparsed requests waiting because the GPU slot is busy, push `fd` onto the FIFO ready queue in $O(1)$.
  Replace `rt_http_drain_ready`'s 65,536-element loop with popping from `ready_head` in $O(1)$.

#### 3.2 Fix GPU Timeout SQE Leak with Linked Timeouts (`internal/runtime/c/http.go:619-647`)
- When arming the `eventfd` poll in `rt_http_arm_gpu_poll` while a GPU job is parked, link the timeout directly to the poll SQE using `IOSQE_IO_LINK` and `io_uring_prep_link_timeout`.
- When `eventfd` becomes readable, the kernel automatically cancels the linked timeout with zero stale timer accumulation.

#### 3.3 Fast-Path User-Space `cudaStreamQuery` (`internal/runtime/c/http.go:963-1105`)
- Before calling `io_uring_submit_and_wait`, if `g_rt_http_parked != NULL`, check `rt_gpu_query_job()` (`cudaStreamQuery` in user space, ~50 ns).
- If the kernel already completed while the worker was processing other HTTP requests, run `rt_http_gpu_release` immediately without waiting for `cudaLaunchHostFunc` + `eventfd` wakeup.

#### 3.4 Selective Workspace D2H Copies (`internal/runtime/c/rt.go:1719-2001`)
- Track which workspaces are written by kernels or read by `gpu.ready` (or add a `dirty_on_device` flag set when a workspace is passed to `gpu.launch*` and cleared when copied, or only D2H-copy workspaces that are not purely input-only).
- Even simpler: mark workspaces that receive `gpu.copyToDevice` and are only read in kernels vs. workspaces written in kernels, or copy D2H on demand / only for output workspaces.
- Make `rt_gpu_sync()` call `cudaStreamSynchronize(rt_gpu_launch_stream())` when on an HTTP worker thread instead of `cudaDeviceSynchronize()`, and make `rt_gpu_copy_to_host_*` use `cudaMemcpyAsync(..., g_rt_gpu_stream)` + `cudaStreamSynchronize(g_rt_gpu_stream)` so workers never stall each other's streams.

---

### Phase 4: Zero-Copy HTTP Data Path & Modern `io_uring` Setup
**Goal:** Maximize single-core and multi-core HTTP throughput and minimize p99 tail latency.

#### 4.1 Zero-Copy Parsing & Request String Views (`internal/runtime/c/http.go:355-417`, `1057-1080`)
- **Direct Buffer-Ring Dispatch:** In the `recv` CQE handler (`http.go:1057`), when `c->len == 0`, run `rt_http_find_header_end` and `rt_http_dispatch` **directly on `in_data`** (the kernel buffer ring page) before returning `bid` to the buffer ring. Only copy into `c->buffer` if the request is partial (split across TCP segments).
- **Zero-Copy String Views:** Change `rt_http_method`, `rt_http_path`, `rt_http_query`, `rt_http_body`, `rt_http_header`, `rt_http_param`, and `rt_http_context` to return `RT_LIT(rt_string, ptr, len)` views directly without `rt_http_clone_string` (since `rt_http_retain_parked` already deep-copies them into the arena if the handler parks on `gpu.finish`).
- **Fast Header Formatter & Partial Send Handling (`http.go:426-518`, `1026-1042`):**
  - Replace `snprintf` in `rt_http_respond` with direct formatting into `c->tx_buf` using `memcpy` for static header fragments and a fast inline `u64_to_ascii` helper for status code and `Content-Length`.
  - Track `c->tx_sent` and `c->tx_len` on `rt_http_conn`. If `RT_HTTP_UDATA_SEND` returns `0 < res < remaining`, submit a follow-up `io_uring_prep_send` for `c->tx_buf + c->tx_sent` (`remaining - res`). If a second pipelined response is written while `tx_in_flight` is true, append it to `c->tx_buf` instead of calling synchronous `writev`.
  - Enforce `RT_HTTP_MAX_HEADER_BYTES` during `recv` and handle `res == -ENOBUFS` by re-arming `io_uring_prep_recv_multishot` instead of closing the connection.

#### 4.2 `io_uring` Advanced Flags, Batch Ring Advances, & Core Pinning (`http.go:919-1158`)
- Pin each worker thread $i$ to the $i$-th available CPU in the affinity mask via `pthread_setaffinity_np`.
- Initialize `io_uring` with `IORING_SETUP_SINGLE_ISSUER | IORING_SETUP_COOP_TASKRUN | IORING_SETUP_DEFER_TASKRUN` (with graceful fallback to `0` if running on an older kernel) and call `io_uring_register_ring_fd(&ring)`.
- Batch `io_uring_cq_advance` and `io_uring_buf_ring_advance` once per event-loop turn instead of once per CQE.
- Use `io_uring_submit_and_wait(&ring, 1)` to fuse submission and waiting into a single syscall per loop turn.

---

### Phase 5: GPU Kernel Coalescing, Reductions, KV-Cache, & Micro-Batching
**Goal:** Extract maximum SM compute and memory bandwidth on GPU workloads (`benchmarks/digits`, `shakespeare_server`).

#### 5.1 `benchmarks/digits` & `examples/projects/digits_server`
- **GPU-Side Vote & Argmax:** Add a second kernel (`voteK`) or fuse the 45-pair voting so the GPU writes the winning digit directly into a 1-element workspace, reducing D2H copy size from 4,152 bytes to 8 bytes and removing the $O(\text{classes}^2 \times n_{sv})$ host loop from the HTTP worker thread.

#### 5.2 `examples/projects/shakespeare_server`
- **KV Cache for Autoregressive Decode:** Store per-layer `K` and `V` caches in GPU workspaces so each decode step after prefill only projects and attends for the **1 new token** ($n = 1$) instead of recomputing all $n \le 128$ tokens (~30x reduction in FLOPs and memory traffic).
- **Only Copy the Last Token's Logits:** Copy only the `vocabSize` (65) floats for position `n - 1` instead of `n * vocabSize` floats, or run greedy `argmax` on the GPU and copy 1 token ID (`8` bytes).
- **Non-Blocking Continuation:** Convert `generateHandler` to use `gpu.finish` so `/health` and other connections on the same worker are never blocked during token generation.

---

## 3. Verification & Benchmark Plan

1. **Compiler & Runtime Unit Tests:**
   - `make test` (`go test ./...`) — verify all lexer, parser, typechecker, IR, C/CUDA codegen, and runtime C tests pass with `-Wall -Wextra -pedantic`.
2. **In-Process CPU & GPU Benchmarks:**
   - `make bench` (`performance_integer.tx`, `gpu_matrix_multiply.tx`, `performance_stencil.tx`, `performance_cnn.tx`)
   - `make mnist-bench` and `make transformer` — verify exact numerical parity (`1e-9` logit tolerance) and measure CPU & GPU speedup from fast overflow intrinsics, `-O3 -march=native`, and `nvcc -arch=native` + `__restrict__`.
3. **HTTP + CPU Container Benchmarks:**
   - `python3 benchmarks/http/benchmark.py` — measure req/s, p50/p99 latency, and memory footprint across `/plaintext`, `/json/serialize`, `/sqlite/insert-and-query`, `/sqlite/query-by-id/1024`, and `/api/authenticated-audit`.
4. **HTTP + GPU Benchmarks:**
   - `make bench-digits` (`go run ./benchmarks/digits/ -concurrency 1,8,32,128`) — measure req/s and p50/p99 latency improvements on the HTTP+GPU continuation path.
   - `examples/projects/shakespeare_server` — measure token generation latency (`elapsed_ms`) on both `"device": "gpu"` and `"device": "cpu"`, and verify concurrent `/health` responsiveness while `/generate` is running.
