---
title: "HTTP + GPU"
description: "One handwritten-digit classification per HTTP request, with the RBF kernel on the GPU."
icon: "developer_board"
weight: 20
---

This page is one measured run of `make bench-digits`. Each request sends one **8×8** image from scikit-learn's [`load_digits`](https://scikit-learn.org/stable/modules/generated/sklearn.datasets.load_digits.html) set. The server answers with the digit it classified. The [HTTP](/docs/benchmarks/http/) page is a different program and a different machine setup.

These files are the whole benchmark input:

| What you are looking at | Where it is |
| --- | --- |
| The 8×8 image, as 64 numbers | `benchmarks/digits/image.json` |
| The dataset row and the correct digit | `benchmarks/digits/label.txt` (`1728` and `7`) |
| The trained model the server loads | `benchmarks/digits/weights.txm` |
| The script that writes that model | `examples/projects/digits_server/export_svm.py` |

## What is measured

The server is `benchmarks/digits/server.tx`. It listens on `127.0.0.1:8090` and handles `POST /predict`.

One request does this:

1. Read 64 numbers from the JSON body.
2. Copy those numbers into that worker's input workspace on the GPU.
3. Launch one RBF kernel over the support vectors. The kernel's first parameter is the vector index. `gpu.launch1d` groups the threads in blocks of 256.
4. Return to the `io_uring` loop while the kernel runs. Other connections on that worker still accept and receive.
5. When the kernel finishes, vote on the host and send `{"digit":N}`.

The weights stay on the GPU for the whole process. A request does not read the file again and does not allocate the weights again. One kernel is in flight per worker.

`make bench` does not start this server. That target runs the in-process integer, matrix, stencil, and conv-net programs.

## The dataset

[`load_digits`](https://scikit-learn.org/stable/modules/generated/sklearn.datasets.load_digits.html) is the UCI optical recognition of handwritten digits set, as shipped by scikit-learn. It has **1,797** images. Each image is **8×8**, so **64** pixels. Each pixel is an integer from **0** to **16**. MNIST is a different collection: 28×28 images and 784 pixels. This benchmark uses the 8×8 set only.

`export_svm.py` calls `train_test_split(..., test_size=0.5, shuffle=False)`. Rows **0–897** (898 images) train the SVM. Rows **898–1796** (899 images) stay out of the fit. The benchmark image is row **1728**, inside that held-out half. `label.txt` stores the dataset's own label for that row, which is **7**.

## The 8×8 image

`benchmarks/digits/image.json` is the request body. The harness posts that file on every request. There is no PNG and no PPM. The 64 numbers are the 8 rows of row 1728, left to right:

```
 0  0  2 14 16  8  0  0
 0  0  4 12 16 11  0  0
 0  0  0  0 16 12  0  0
 0  0  0  3 16  9  0  0
 0  2  5 10 16 12  2  0
 0 16 16 16 16 14  3  0
 0  4  4 14 12  0  0  0
 0  0  2 16  7  0  0  0
```

The same pixels, in the file, are one JSON array:

```json
[0,0,2,14,16,8,0,0,0,0,4,12,16,11,0,0,0,0,0,0,16,12,0,0,0,0,0,3,16,9,0,0,0,2,5,10,16,12,2,0,0,16,16,16,16,14,3,0,0,4,4,14,12,0,0,0,0,0,2,16,7,0,0,0]
```

`pixelsFrom` in `server.tx` accepts that bare array. It also accepts `{"pixels":[...]}` with 64 numbers. Any other body gets HTTP 400 and the text `expected 64 pixels`.

## The trained model

Open `benchmarks/digits/weights.txm`. That is the model this benchmark serves. It is a binary TXM1 file, so a text editor will not show the coefficients. The readable description is the five tensors below, and the script that packed them is `examples/projects/digits_server/export_svm.py`.

The harness copies `weights.txm` to `benchmarks/digits/digits.txm` for the run, because `server.tx` calls `gpu.mount("digits.txm")`. The copy is deleted when the run ends. `digits.txm` is in `.gitignore`. The committed model is `weights.txm`.

The fit keeps 519 support vectors. `n_support` is 34, 64, 55, 52, 43, 50, 35, 56, 65, 65 for digits 0 through 9.

| Name | Shape | What got exported |
| --- | --- | --- |
| `gamma` | `[1]` float | `0.001`, from `SVC(gamma=0.001)` |
| `support` | `[519, 64]` float | `clf.support_vectors_`, one 8×8 image per row, flattened |
| `dual` | `[9, 519]` float | `clf.dual_coef_`, the one-vs-one coefficients |
| `intercept` | `[45]` float | `clf.intercept_`, one intercept per pair of digits |
| `n_support` | `[10]` int | `clf.n_support_`, how many support vectors belong to each digit |

`examples/projects/digits_server/sample.txm` has the same five names with two support vectors. Open that file only to see a tiny TXM1 header. It is not the trained model.

## How the numbers are exported

From `examples/projects/digits_server`:

```bash
python3 export_svm.py
```

That needs numpy and scikit-learn. The committed `weights.txm` was produced by this script with scikit-learn 1.9.1. The script writes `digits.txm` in that directory. That local file is not committed. To refresh the benchmark model, write the export straight to the committed path:

```bash
python3 export_svm.py --output ../../../benchmarks/digits/weights.txm
```

`export_svm.py` does this:

1. `load_digits()` loads the 1,797 images. Each image is reshaped from 8×8 to 64 floats.
2. The first 898 rows train `SVC(gamma=0.001, decision_function_shape="ovo")`. `ovo` makes `decision_function` return the 45 pairwise sums. It does not change the fitted weights.
3. For every held-out row, the script's own vote is checked against `predict` and against those 45 sums. One mismatch aborts the export.
4. The five arrays are packed little-endian into a TXM1 file: float64 for `gamma`, `support`, `dual`, and `intercept`, int64 for `n_support`. The byte layout is on the [TXM1](/docs/design-notes/txm/) page.

`python3 export_svm.py --sample` writes `sample.txm` and does not fit a model. `--sample` needs only the Python standard library.

## What the model does

This is a 10-class SVM with an RBF kernel, one-vs-one. There are 45 pairs among the digits 0 through 9.

The kernel runs once for each support vector. For vector `sv` and image `x` it writes:

```
score[sv] = exp(-gamma * ||x - support[sv]||^2)
```

`vote` then walks the 45 pairs. Each pair adds a signed sum of `dual * score` plus that pair's intercept. A positive sum votes for the lower digit. A zero or negative sum votes for the higher digit. The response is the digit with the most votes.

On this image the dataset label is 7, and the fitted vote is 7. The closest pairwise sum is about 0.13 away from zero, so a small difference between device `exp` and host `exp` does not flip the winner.

## How a TXM file is laid out

`gpu.mount` reads a TXM1 file: a little-endian list of named tensors. The reader does not know what the names mean. The program does. The byte rules are on the [TXM1](/docs/design-notes/txm/) page. Two shapes are enough to see the idea.

A linear regression `y = w0*x0 + w1*x1 + b` is two tensors. The formula stays in the program.

| Name | Shape | Dtype | Example values |
| --- | --- | --- | --- |
| `w` | `[2]` | float | 1.5, -0.25 |
| `b` | `[1]` | float | 0.5 |

The digits SVM is the five names in the table above. Every TXM1 file starts with the same header. `sample.txm` begins:

```
54 58 4d 31   01 00 00 00   05 00 00 00
T  X  M  1    version 1     5 tensors
```

## Concurrency

Concurrency is the number of clients posting at the same time. Each client holds one keep-alive connection and sends the next copy of `image.json` as soon as the previous response arrives. The default run uses 1, then 8, then 32 connections. Each level lasts 6 seconds.

Concurrency is not the worker count. The server starts one worker per CPU in the process affinity mask. This machine allowed 24 workers. Each worker may have one kernel in flight. At 32 connections, some requests wait for a worker before they launch.

`-duration` and `-concurrency` change those two knobs. `-model` replaces `benchmarks/digits/weights.txm`. The posted image stays `image.json`.

## The check

The correct digit is the dataset label, 7.

Before the timer, the harness votes `image.json` on the host with the same one-vs-one loop as `server.tx`. That vote must be 7. It then posts the image and requires the body `{"digit":7}`. The timer starts only after that response. Every timed response must be the same body. A wrong digit, a non-200 status, or a failed request increments the error count and fails the run.

`-model` still posts this image and still requires digit 7. A file that votes for another digit fails before the timer. Device `exp` and host `exp` can land on opposite sides of a sum that is almost zero, and that run fails on purpose.

## This run

- **Host**: Linux Mint 22.3, kernel 7.0.0-38-generic, AMD Ryzen 9 5900X
- **Workers**: 24, one per hardware thread in the process affinity mask
- **GPU**: NVIDIA GeForce GTX 1660, 6 GiB, driver 595.99, CUDA 12.0 (`nvcc` 12.0.140)
- **Library**: liburing 2.5
- **Image**: `benchmarks/digits/image.json`, `load_digits` row 1728, dataset label 7
- **Weights**: `benchmarks/digits/weights.txm`, trained `SVC(gamma=0.001)`, 519 support vectors
- **Duration**: 6 seconds at 1, 8, and 32 keep-alive connections
- **Check**: every response was `{"digit":7}`, 0 errors

| Concurrency | Requests | Throughput (req/s) | p50 | p99 |
| :---: | :---: | :---: | :---: | :---: |
| 1 | 6,424 | **1,070.6** | 0.369 ms | 1.839 ms |
| 8 | 30,922 | **4,548.9** | 0.409 ms | 2.119 ms |
| 32 | 59,923 | **8,832.8** | 0.693 ms | 3.794 ms |

From 1 to 8 connections, throughput follows the extra in-flight kernels. At 32 connections there are still 24 workers, so some requests wait for a worker before they launch.

## Run it

Linux, liburing, and CUDA:

```bash
make bench-digits
```

That runs `go run ./benchmarks/digits/`.

```bash
go run ./benchmarks/digits/ -duration 10s -concurrency 1,4,16
go run ./benchmarks/digits/ -model path/to/other.txm
```

The default model is `benchmarks/digits/weights.txm`. `-model` still posts `image.json` and still requires digit 7.
