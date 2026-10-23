# Digits server

An HTTP server that classifies one 8x8 digit per request. The weights come from a TXM1 file. `gpu.mount` sends them to the GPU once, before `http.listen`. Each request reuses that device memory and the worker's input and score workspaces.

## Request

`POST /predict` with a JSON array of 64 numbers, or `{"pixels":[...]}`. The numbers are the sklearn digit pixels, 0 through 16. The response is `{"digit":N}`.

`predict` copies those 64 floats into the pinned input workspace, launches one kernel, and calls `gpu.finish`. It does not send the response. The worker returns to `io_uring` while the kernel runs. Other sockets on that worker still accept and receive. `vote` runs when the scores are back in the pinned mirror, does the 45 one-vs-one votes on the host, and sends the JSON.

One request is launched at a time on each worker. Several workers can each have one kernel in flight. This server does not batch requests into one launch.

## Files

`sample.txm` is a tiny model with two support vectors. It has the names and shapes the program reads. It is not a trained SVM.

`export_svm.py` fits the same model as the handwritten-digits notebook: `load_digits`, flatten to 64 features, `train_test_split(..., test_size=0.5, shuffle=False)`, `SVC(gamma=0.001)`. It checks the host vote against `predict` and `decision_function`, then writes `digits.txm`. The training split is 898 images. The notebook's fitted SVM keeps 519 support vectors. That file is not committed. Point `gpu.mount` at it when you want the real model.

```
python3 export_svm.py
python3 export_svm.py --sample
```

The full export needs numpy and scikit-learn. `--sample` needs only the Python standard library.

## What costs what

Startup, once: read the file, `cudaMalloc` the tensors, copy them to the device, and allocate each worker's workspaces.

A request: copy 64 floats into the pinned input, queue that copy and the kernel on the worker stream, queue the score copy back, then vote on the host. The file is not read again. The weights are not allocated again.

`gpu.to` is still the blocking copy used by the other GPU examples. This server does not use it on the request path.

## Check on Linux with a GPU

This machine can compile the program and parse `sample.txm` without a GPU. The running server needs Linux, `liburing`, and CUDA. After it is listening:

- Send several `/predict` requests. Device memory reported by `nvidia-smi` should stay flat. A second request must not allocate another copy of the weights.
- While one request is in flight, a keep-alive read on another connection to the same worker should still complete. The kernel for one 64-pixel image is too small for that overlap to be easy to see. The check that matters for this model is that the weights stay resident and the worker is not stuck inside `cudaDeviceSynchronize`.

A second `copyToDevice` into the same workspace before the first device copy finishes is not guarded. This handler copies the input once.

## Benchmark

`make bench-digits` times this predict path on Linux with liburing and CUDA. The harness is `benchmarks/digits`. It serves a TXM1 file with the notebook shape, 519 support vectors, and reports request rate and latency. `make bench` does not start this server.
