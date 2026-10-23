# Handwritten digits HTTP + GPU

Times `POST /predict` on `server.tx`. Each request posts `image.json`: one 8×8 image from scikit-learn [`load_digits`](https://scikit-learn.org/stable/modules/generated/sklearn.datasets.load_digits.html) (1,797 images, 64 pixels, values 0–16). Row 1728 is a handwritten 7 from the held-out half. Rows 0–897 train the model. The dataset label is the first line of `label.txt`.

The trained model is `weights.txm`. That file is binary. `examples/projects/digits_server/export_svm.py` is the script that packs `SVC(gamma=0.001)` into it (`support_vectors_`, `dual_coef_`, `intercept_`, `n_support_`, gamma). Regenerate with:

```
python3 examples/projects/digits_server/export_svm.py --output benchmarks/digits/weights.txm
```

The harness copies `weights.txm` to `digits.txm` for `gpu.mount` and deletes the copy when the run ends. `digits.txm` is gitignored.

A request copies the 64 floats into that worker's workspace, launches the RBF kernel, and finishes on the `io_uring` loop. The worker does not load the file again and does not allocate the weights again.

The correct digit is 7. The harness votes the image on the host and requires that vote to be 7. It then requires every response, including the ones inside the timer, to be `{"digit":7}`. `-model` still has to classify this image as 7. Device `exp` and host `exp` can disagree when a pair sum is almost zero, and that run fails.

`server.tx` is the same predict and vote path as `examples/projects/digits_server`, bound to `127.0.0.1:8090`.

## Run

Linux, `liburing`, and CUDA:

```
make bench-digits
```

That runs `go run ./benchmarks/digits/`. The harness builds the server, copies `weights.txm` to `digits.txm`, checks the digit, then measures keep-alive posts. Default levels are 1, 8, and 32 connections, 6 seconds each. Concurrency is the number of clients posting at once. It is separate from the worker count. Workers follow the process CPU affinity mask. One kernel is in flight per worker.

```
go run ./benchmarks/digits/ -duration 10s -concurrency 1,4,16
go run ./benchmarks/digits/ -model path/to/other.txm
```

`make bench` does not run this. That target is the in-process integer, matrix, stencil, and conv-net programs. This server needs the Linux HTTP runtime.
