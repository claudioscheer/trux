#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
PORT="${PORT:-8080}"

cd "$ROOT"
make build

cp "$ROOT/examples/projects/tiny_transformer/model.txm" "$HERE/model.txm"
"$ROOT/bin/trux" build -o "$HERE/server" "$HERE/main.tx"

cd "$HERE"
docker build -t trux-shakespeare .

LIBCUDA="$(readlink -f /lib/x86_64-linux-gnu/libcuda.so.1)"
LIBPTX="$(readlink -f /lib/x86_64-linux-gnu/libnvidia-ptxjitcompiler.so.1)"

echo "Starting trux-shakespeare on http://127.0.0.1:${PORT} (2 vCPUs, 4 GB RAM, CUDA enabled)..."
docker rm -f trux-shakespeare >/dev/null 2>&1 || true
exec docker run --rm --name trux-shakespeare \
  --security-opt seccomp=unconfined --ulimit memlock=-1:-1 \
  -p "${PORT}:8080" \
  --cpus="2" --memory="4g" \
  --device /dev/nvidia0 \
  --device /dev/nvidiactl \
  --device /dev/nvidia-uvm \
  --device /dev/nvidia-uvm-tools \
  -v "${LIBCUDA}:/lib/x86_64-linux-gnu/libcuda.so.1:ro" \
  -v "${LIBPTX}:/lib/x86_64-linux-gnu/libnvidia-ptxjitcompiler.so.1:ro" \
  trux-shakespeare
