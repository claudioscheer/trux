# TXM1

TXM1 is a file of named numeric tensors. `gpu.mount` reads one file, copies each tensor to the GPU, and keeps that device memory for the rest of the process. The reader does not interpret names. A digits SVM, a small network, and a transformer are the same container. The Trux program decides what each name means.

The file is little-endian and has no padding and no text.

```
magic          4 bytes  "TXM1"
version        uint32   1
tensor count   uint32   1..256
```

Each tensor follows immediately:

```
name_len       uint32   1..64
name           bytes    UTF-8, not null-terminated, unique in the file
dtype          uint32   1 = int (int64), 2 = float (float64)
rank           uint32   1..4
dims           uint64   rank times, each dimension >= 1
payload        raw      product(dims) values, int64 or float64
```

A one-element float tensor is a scalar. There is no separate scalar table and no workspace table. Workspace sizes are chosen by the program with `gpu.workspaceFloat` and `gpu.workspaceInt`.

## Two files

The bytes start the same way for every model. `examples/projects/digits_server/sample.txm` begins:

```
54 58 4d 31   01 00 00 00   05 00 00 00
T  X  M  1    version 1     5 tensors
```

A linear regression `y = w0*x0 + w1*x1 + b` is two tensors. The program multiplies `w` by the input and adds `b`. The file does not store that formula.

| Name | Shape | Dtype | Values |
| --- | --- | --- | --- |
| `w` | `[2]` | float | 1.5, -0.25 |
| `b` | `[1]` | float | 0.5 |

An SVM uses other names. The digits table below is what `examples/projects/digits_server` and `benchmarks/digits` read. `sample.txm` has those names with `n_sv = 2`. `benchmarks/digits/weights.txm` is the trained file, with `n_sv = 519`.

The reader rejects:

- a magic other than `TXM1`
- a version other than 1
- a tensor count of 0 or greater than 256
- a truncated file
- a name length outside 1..64
- a duplicate name
- a dtype other than 1 or 2
- a rank outside 1..4
- a dimension of 0
- a dimension product that overflows
- bytes left over after the last payload

`examples/projects/digits_server` uses these names for the handwritten-digits SVM. Another model uses other names and another program.

| Name | Shape | Dtype | Meaning in that program |
| --- | --- | --- | --- |
| `gamma` | `[1]` | float | RBF gamma |
| `support` | `[n_sv, 64]` | float | support vectors, one image per row |
| `dual` | `[9, n_sv]` | float | one-vs-one dual coefficients |
| `intercept` | `[45]` | float | one-vs-one intercepts |
| `n_support` | `[10]` | int | support vectors per digit |

The checked-in `sample.txm` has `n_sv = 2` so the fixture stays small. The full training file is produced by `export_svm.py` and is not committed.
