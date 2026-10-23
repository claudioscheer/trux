# MNIST MLP

A 784 → 128 → 64 → 10 ReLU network trained in scikit-learn, exported to a TXM1
file, and run in Trux. The whole 10,000-row test set is classified in three
`gpu.launch1d` calls, then argmax runs on the host. `bench.tx` runs the same
network with host loops and with the kernels, so the CPU and GPU timings use the
same weights and the same value check.

Running either program needs an NVIDIA GPU and `nvcc`, because `gpu.mount` is
the only TXM reader. The CPU section of `bench.tx` launches no kernels. The
export does not need a GPU.

This is the end-to-end model example: real data, real weights, and the last
layer's ten values for every test row that the program has to reproduce.

## Run

```console
$ make mnist                    # from the repository root
$ make mnist-bench              # CPU vs GPU timings
```

`make mnist` uses the committed `sample.txm` fixture when no `model.txm` exists,
so it runs without training. `make mnist-bench` needs the trained file. To build
the real model:

```console
$ cd examples/projects/mnist_mlp
$ python3 export_mlp.py
```

The first run downloads `mnist_784` from OpenML into scikit-learn's cache (a
15 MB compressed file) and needs `numpy` and `scikit-learn`:

```console
$ python3 -m venv .venv && . .venv/bin/activate
$ pip install numpy scikit-learn
```

`export_mlp.py` trains, writes `model.txm` next to this README, and checks every
test row against `clf.predict` before the file is written. The file carries the
last-layer values and the class for every row, so the Trux run can compare both.
`model.txm` is generated and gitignored.

## Output

```
rows 10000
errors 237
expected errors 237
predictions mismatched 0
logits compared 100000
max logit difference 1.77635683940025e-14
logits outside 1e-9 0
accuracy 0.9763
millis 86
```

That run is a 784 → 128 → 64 → 10 network trained for 12 epochs, on a GTX
1660. The ~86 ms covers the three launches, the sync, and the copy of the
10,000 × 10 logits back to the host. It is the whole test set, not one image.

The last layer is checked value by value. The file stores numpy's ten outputs
for every row as `expected_logits`, and the program compares all 100,000 of
them with a 1e-9 absolute tolerance. The largest difference here is 1.78e-14,
which is float64 rounding: numpy's matrix multiply and the kernel add the 784
products in a different order, and the compiler may fuse a multiply and an add
into one FMA. A transposed weight or a bias on the wrong layer changes the
values by many orders of magnitude more than that, so the check catches real
mistakes rather than double rounding.

`predictions mismatched` is the check on the decision. The program compares its
own argmax with `expected_predictions` for every row; 0 means all 10,000
classes match. The smallest top-2 logit margin in the test set is 0.00434632,
so a difference of 1.78e-14 cannot flip any row.

As a check on the check, setting one output weight to 1e6 produces
`predictions mismatched 2327`, `max logit difference 9983023`, and
`logits outside 1e-9 2408`, then exits 1. Restoring the weight returns the
mismatches and the outside count to zero, and the maximum difference to
1.78e-14.

`sample.txm` has two rows and one deliberate error, so a run without a trained
file still exercises both checks. Its logits match exactly, not just within
1e-9.

## CPU vs GPU

`bench.tx` runs the same three layers twice: once with host loops and once with
the three launches. It mounts the file once, runs a warm-up, times the forward
pass only, and checks both results against the same expected arrays.

This is one 10,000-row run:

```text
rows 10000
timed runs 3
cpu nanos 1188463319
gpu compute nanos 86026937
gpu total nanos 86367673
cpu predictions mismatched 0
cpu logits outside 1e-9 0
gpu predictions mismatched 0
gpu logits outside 1e-9 0
```

The table is the median of three program runs on an AMD Ryzen 9 5900X and a
GTX 1660 with CUDA 12.0. Each program run reports its best pass after a
warm-up (3, 4, or 9 timed passes, depending on the row count):

| Rows | CPU | GPU total | CPU / GPU |
| ---: | ---: | ---: | ---: |
| 1 | 0.115 ms | 0.153 ms | 0.75x |
| 10 | 1.15 ms | 0.183 ms | 6.3x |
| 100 | 11.5 ms | 0.696 ms | 16.6x |
| 1,000 | 124 ms | 7.31 ms | 17.0x |
| 10,000 | 1.19 s | 86.4 ms | 13.7x |

The last column is CPU time divided by GPU time, so values above 1 mean the GPU
is faster. The full test set is about 14x faster on the GPU, and 100 to 1,000
rows are about 17x faster. One row is the exception: three launches, a sync,
and a copy cost more than the 109,184 multiply-adds that row needs, so the GPU
is about 1.3x slower. The crossover is between one and ten rows. Run-to-run
variation is a few percent, and is largest at the small row counts.

`gpu total` includes the launches, `gpu.sync`, and the copy of the logits back
to the host. `gpu compute` stops before the copy; at 10,000 rows the copy adds
about 0.3 ms. The CPU side has no copy because `gpu.hostFloats` already returns
the host arrays, but its loops are plain, bounds-checked Trux code, not a
hand-tuned BLAS call. Both outputs pass the same 1e-9 check, so this is the same
arithmetic.

The `-O2` host build is the same level as a C-only Trux program. The CUDA
compile path used to leave host code at `-O0`; the full-set CPU pass then took
about 13 s, which overstated the GPU speedup. Both paths now pass the same
optimization flag.

The CPU half still needs `gpu.mount`, because it is the only TXM reader in Trux
today. It therefore needs a CUDA build, but the timed CPU section launches no
kernels. Mount is outside every timer.

## Tensor layout

Weights are stored `[out, in]` so the kernel's inner loop walks the weight row
and the input row in the same order. scikit-learn stores `[in, out]`, so the
exporter transposes.

| Name | Shape | Dtype | Meaning |
| --- | --- | --- | --- |
| `w1` | `[128, 784]` | float | layer 1 weights |
| `b1` | `[128]` | float | layer 1 bias |
| `w2` | `[64, 128]` | float | layer 2 weights |
| `b2` | `[64]` | float | layer 2 bias |
| `w3` | `[10, 64]` | float | output weights |
| `b3` | `[10]` | float | output bias |
| `test_images` | `[N, 784]` | float | pixels, already divided by 255 |
| `test_labels` | `[N]` | int | digits 0 through 9 |
| `expected_logits` | `[N, 10]` | float | last layer from numpy, before softmax |
| `expected_predictions` | `[N]` | int | class per row from the numpy check |

`main.tx` reads every size from the file with `gpu.dim`, including the hidden
widths, so `sample.txm` can use an 8/4 network without changing the program.

The pixels live in the same file as the weights because `gpu.mount` accepts one
file per process, and a batched launch needs them on the device anyway. The
exporter divides by 255 before writing, because that is what the model was
trained on. Feeding raw 0 through 255 pixels to the same weights would produce
confident nonsense.

There is no softmax kernel. Argmax of the logits is the same class as argmax of
the softmax. The exporter still writes the raw last-layer values as
`expected_logits`, because those are what the program compares. Softmax would
only rescale them and hide absolute differences.

## Files

| File | |
| --- | --- |
| `main.tx` | mount, three launches, host argmax, compare every logit and prediction |
| `bench.tx` | CPU and GPU forward passes at five row counts, with the same checks |
| `export_mlp.py` | train, check against sklearn, write TXM1 |
| `sample.txm` | committed fixture: 8/4 hidden, two rows, one known error |
| `model.txm` | generated by `export_mlp.py`, gitignored |

`export_mlp.py --sample` rebuilds `sample.txm` with only the Python standard
library. It uses deterministic weights and picks a seed whose top two logits are
far enough apart that the fixture cannot flip on rounding.

## Notes

- The file stores float64, so the weights are twice the size and the kernels run
  at a fraction of float32 throughput on consumer cards. A 1660 class GPU still
  classifies the whole test set in milliseconds because the network is small.
- `--hidden 128,64 --max-iter 20` changes the training run. `--rows 1000` caps
  the test set for a quicker export.
- `--data mnist.npz` reads a local file with `x` and `y` arrays instead of
  downloading.
