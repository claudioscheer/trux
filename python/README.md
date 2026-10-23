# TXM tools

`txm.py` reads and writes TXM1 files, the container `gpu.mount` loads. It is one
file, it uses only the Python standard library, and it enforces the rules in
[`docs/TXM.md`](../../docs/TXM.md) when writing. A bad tensor fails in Python
with a name and a shape instead of failing at `gpu.mount` on a machine with a
GPU.

numpy arrays are accepted when numpy is installed. The module never imports
numpy, so `--sample` exports keep working on a bare interpreter.

## Use

Add this directory to `sys.path` or copy `txm.py` next to your script.

```python
import txm

txm.pack("model.txm", [
    txm.floats("w", [[1.5, -0.25]]),
    txm.floats("b", [0.5]),
])

print(txm.info("model.txm"))
```

`txm.floats(name, values, dims=None)` and `txm.ints(name, values, dims=None)`
take a numpy array, a flat sequence with `dims`, or a nested sequence whose
shape can be inferred.

Reading:

```python
tensors = {tensor.name: tensor for tensor in txm.read("model.txm", values=True)}
print(tensors["w"].shape, tensors["w"].values)

problems = txm.verify("model.txm", {"w": (1, 2), "b": (1,)})
```

`read(path)` without `values=True` returns only names, shapes, and dtypes, so a
60 MB model is cheap to inspect. `verify` returns a list of problems and does
not raise.

From a shell:

```console
$ python3 python/txm.py info examples/projects/mnist_mlp/sample.txm
name             shape           dtype  elements
w1               [8, 784]        float  6272
...
```

## Tests

```console
$ python3 -m unittest discover -s python
```

The tests include a repack of `examples/projects/digits_server/sample.txm` that
must reproduce the committed bytes exactly.

## Status

This is not a published package. It exists so the example exporters do not each
hand-roll `struct.pack`. It is the part of a future `trux-txm` distribution that
does not depend on a model framework.
