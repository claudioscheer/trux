# Handwritten digits HTTP + GPU

Times `POST /predict` on the digits server. Each request is one 8x8 image. The weights are a TXM1 file mounted once, before listen. A request copies 64 floats into that worker's workspace, launches the RBF kernel, and finishes on the `io_uring` loop. The worker does not load the file again and does not allocate the weights again.

The default file matches the notebook model shape: 519 support vectors, 64 features, dual coefficients `9 x 519`, 45 intercepts, gamma `0.001`. The notebook prints that support-vector count after `SVC(gamma=0.001)` on the 898-image training split. The numbers in the default file are synthetic. Class 0 owns the first 52 vectors, all zeros, so an image of 64 zeros is digit 0. Pass `-model` with the `digits.txm` from `export_svm.py` to time the trained weights instead.

The server is `server.tx`. It is the same predict and vote path as `examples/projects/digits_server`, bound to `127.0.0.1:8090`, mounting `digits.txm`.

## Run

Linux, `liburing`, and CUDA:

```
make bench-digits
```

That runs `go run ./benchmarks/digits/`. The harness builds the server, writes `digits.txm`, checks one prediction, then measures keep-alive posts. Default levels are 1, 8, and 32 connections, 6 seconds each. One kernel is in flight per worker. Workers follow the process CPU affinity mask.

```
go run ./benchmarks/digits/ -duration 10s -concurrency 1,4,16
go run ./benchmarks/digits/ -model examples/projects/digits_server/digits.txm
```

`make bench` does not run this. That target is the in-process integer, matrix, stencil, and conv-net programs, including on Windows. This server needs the Linux HTTP runtime.

The check image is 64 zeros. The harness computes the digit with the same host vote as `server.tx` and requires the response body to be that digit. A trained file can disagree if `exp` on the device and `exp` on the host land on opposite sides of a decision that is almost zero.
