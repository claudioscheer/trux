# Shakespeare HTTP + GPU Server

An HTTP server written in Trux that serves the Tiny Shakespeare character-level
transformer (`examples/projects/tiny_transformer/model.txm`) over `io_uring`
with CUDA acceleration, packaged for a resource-limited Docker container
(`2 vCPUs`, `4 GB RAM`, local NVIDIA GPU).

## Architecture

- **Startup (`main.tx`):**
  - Calls `gpu.mount("model.txm")` once before listening so the 3.5 MB float64
    transformer weights stay resident in both host and device memory.
  - Declares 10 per-worker GPU workspaces (`tokens`, `x`, `xn`, `q`, `k`, `v`,
    `attn`, `scores`, `h`, `logits`) so each HTTP worker thread allocates its
    scratch buffers and non-blocking CUDA stream once at startup (zero
    `cudaMalloc`/`cudaFree` per request).
  - Starts the HTTP/1.1 `io_uring` server on `0.0.0.0:8080` with 2 worker
    threads (`http.serve("0.0.0.0", 8080, 2)`).
- **Endpoints:**
  - `GET /health` — returns `{"status":"ok","model":"tiny_transformer"}`
  - `POST /generate` — accepts JSON `{"prompt": "...", "tokens": 60, "device": "gpu"}`
    (`"device"` defaults to `"gpu"` and also accepts `"cpu"`).

## Run in Docker (2 vCPUs, 4 GB RAM, Local GPU)

From the repository root:

```bash
make shakespeare-docker
```

Or run the script directly:

```bash
./examples/projects/shakespeare_server/run-docker.sh
```

What this does:
1. Builds `bin/trux`.
2. Copies `examples/projects/tiny_transformer/model.txm` into this folder and
   compiles `main.tx` into `./server`.
3. Builds the `trux-shakespeare` Ubuntu 24.04 Docker image (`Dockerfile`).
4. Runs the container with `--cpus="2" --memory="4g"`, `io_uring` permissions
   (`--security-opt seccomp=unconfined --ulimit memlock=-1:-1`), and local
   NVIDIA GPU access (`/dev/nvidia*` devices plus the host `libcuda.so.1` and
   `libnvidia-ptxjitcompiler.so.1` driver libraries).

## Test with `curl`

Health check:

```bash
curl http://127.0.0.1:8080/health
```

Generate 60 tokens on the GPU:

```bash
curl -s http://127.0.0.1:8080/generate \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"what is your name?\n","tokens":60,"device":"gpu"}'
```

Example response (~137 ms on GTX 1660):

```json
{
  "prompt": "what is your name?\n",
  "generated": "\nCORIOLANUS:\nWhat says the state of the state of the state,\n",
  "text": "what is your name?\n\nCORIOLANUS:\nWhat says the state of the state of the state,\n",
  "tokens": 60,
  "device": "gpu",
  "elapsed_ms": 137
}
```

Compare the same request on CPU inside the 2-vCPU container (~1,586 ms):

```bash
curl -s http://127.0.0.1:8080/generate \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"what is your name?\n","tokens":60,"device":"cpu"}'
```

## Files

| File | Purpose |
| --- | --- |
| `main.tx` | HTTP + CUDA server (`GET /health`, `POST /generate`) |
| `hostmath.tx` | Host `hexp` and `hsqrt` wrappers for the CPU path |
| `Dockerfile` | Ubuntu 24.04 runtime image with `liburing2` |
| `run-docker.sh` | Builds `server`, builds the Docker image, and runs the container with GPU + resource limits |
| `README.md` | Documentation and `curl` examples |
