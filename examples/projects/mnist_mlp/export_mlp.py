#!/usr/bin/env python3
"""Write a TXM1 file for the MNIST MLP example.

The default trains a 784 -> 128 -> 64 -> 10 ReLU network on MNIST, writes
model.txm next to this script, and checks every test row against sklearn
before the file is written. The last-layer values and the class for every row
go into the file as `expected_logits` and `expected_predictions`, and main.tx
compares its own numbers row by row.

    python3 export_mlp.py
    python3 export_mlp.py --hidden 128,64 --max-iter 20

The first run downloads mnist_784 from OpenML into scikit-learn's cache
(15 MB compressed). Later runs load it from that cache. Pass --data to read a
local .npz with `x` and `y` arrays instead.

--sample writes the small committed fixture sample.txm. It uses deterministic
weights and needs only the Python standard library.
"""

import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "python"))

import txm

FEATURES = 784
CLASSES = 10
SAMPLE_HIDDEN = (8, 4)
TRAIN_ROWS = 60000


def _ramp(seed, count, scale):
    """Deterministic pseudo-random values in [-scale/2, scale/2]."""
    values = []
    state = seed
    for _ in range(count):
        state = (state * 1103515245 + 12345) & 0x7FFFFFFF
        values.append((state / 0x7FFFFFFF - 0.5) * scale)
    return values


def _dense(values, weights, bias, in_dim, out_dim):
    out = []
    for oc in range(out_dim):
        total = bias[oc]
        base = oc * in_dim
        for ic in range(in_dim):
            total += weights[base + ic] * values[ic]
        out.append(total)
    return out


def _relu(values):
    return [value if value > 0.0 else 0.0 for value in values]


def _predict(row, model):
    w1, b1, w2, b2, w3, b3, hidden1, hidden2 = model
    h1 = _relu(_dense(row, w1, b1, FEATURES, hidden1))
    h2 = _relu(_dense(h1, w2, b2, hidden1, hidden2))
    logits = _dense(h2, w3, b3, hidden2, CLASSES)
    return max(range(CLASSES), key=lambda index: logits[index]), logits


def _margin(logits):
    ranked = sorted(logits, reverse=True)
    return ranked[0] - ranked[1]


def sample_tensors():
    """The committed fixture: tiny weights, two rows, one known error."""
    hidden1, hidden2 = SAMPLE_HIDDEN
    rows = [[0.0] * FEATURES, [((i * 7) % 32) / 31.0 for i in range(FEATURES)]]

    for seed in range(1, 128):
        model = (
            _ramp(seed + 1, hidden1 * FEATURES, 0.002),
            _ramp(seed + 2, hidden1, 0.01),
            _ramp(seed + 3, hidden2 * hidden1, 0.05),
            _ramp(seed + 4, hidden2, 0.01),
            _ramp(seed + 5, CLASSES * hidden2, 0.1),
            _ramp(seed + 6, CLASSES, 0.01),
            hidden1,
            hidden2,
        )
        predictions, logits = zip(*(_predict(row, model) for row in rows))
        if min(_margin(values) for values in logits) > 0.001:
            break
    else:
        raise SystemExit("could not build a sample with wide argmax margins")

    labels = [predictions[0], (predictions[1] + 1) % CLASSES]

    return [
        txm.floats("w1", model[0], (hidden1, FEATURES)),
        txm.floats("b1", model[1]),
        txm.floats("w2", model[2], (hidden2, hidden1)),
        txm.floats("b2", model[3]),
        txm.floats("w3", model[4], (CLASSES, hidden2)),
        txm.floats("b3", model[5]),
        txm.floats("test_images", rows, (len(rows), FEATURES)),
        txm.ints("test_labels", labels),
        txm.ints("expected_predictions", list(predictions)),
        txm.floats("expected_logits", logits, (len(rows), CLASSES)),
    ]


def load_mnist(data_path=None):
    import numpy as np

    if data_path:
        bundle = np.load(data_path)
        return (
            np.asarray(bundle["x"], dtype=np.float64),
            np.asarray(bundle["y"], dtype=np.int64),
        )

    from sklearn.datasets import fetch_openml

    print("loading mnist_784 from openml (cached after the first download)", flush=True)
    bunch = fetch_openml("mnist_784", version=1, as_frame=False, parser="liac-arff")
    return (
        np.asarray(bunch.data, dtype=np.float64),
        np.asarray(bunch.target, dtype=np.int64),
    )


def full_tensors(hidden, max_iter, data_path, rows):
    import numpy as np
    from sklearn.neural_network import MLPClassifier

    images, labels = load_mnist(data_path)
    if len(images) != len(labels):
        raise SystemExit("x and y have different row counts")

    images = images / 255.0
    train_x, train_y = images[:TRAIN_ROWS], labels[:TRAIN_ROWS]
    test_x, test_y = images[TRAIN_ROWS:], labels[TRAIN_ROWS:]
    if rows:
        test_x, test_y = test_x[:rows], test_y[:rows]

    print(
        "training 784 -> %s -> 10 on %d rows"
        % (" -> ".join(str(size) for size in hidden), len(train_y)),
        flush=True,
    )
    clf = MLPClassifier(
        hidden_layer_sizes=hidden,
        activation="relu",
        solver="adam",
        max_iter=max_iter,
        early_stopping=True,
        n_iter_no_change=3,
        random_state=0,
    )
    clf.fit(train_x, train_y)
    print("trained in %d epochs" % clf.n_iter_, flush=True)

    def layer(values, weights, bias):
        # clf.coefs_ is [in, out]; the file stores the transpose.
        return values @ weights + bias

    h1 = np.maximum(0.0, layer(test_x, clf.coefs_[0], clf.intercepts_[0]))
    h2 = np.maximum(0.0, layer(h1, clf.coefs_[1], clf.intercepts_[1]))
    logits = layer(h2, clf.coefs_[2], clf.intercepts_[2])
    predicted = logits.argmax(axis=1)

    sklearn_predicted = clf.predict(test_x)
    differing = np.flatnonzero(predicted != sklearn_predicted)
    if len(differing):
        raise SystemExit(
            "%d rows differ from sklearn predict, first at %d"
            % (len(differing), int(differing[0]))
        )

    errors = int((predicted != test_y).sum())
    accuracy = 100.0 * (len(test_y) - errors) / len(test_y)
    top_two = np.partition(logits, -2, axis=1)[:, -2:]
    margin = float((top_two[:, 1] - top_two[:, 0]).min())
    print(
        "checked %d rows against sklearn: %d errors, %.2f%% accuracy"
        % (len(test_y), errors, accuracy),
        flush=True,
    )
    print("smallest top-2 logit margin %.6g" % margin, flush=True)

    hidden1, hidden2 = hidden
    tensors = [
        txm.floats("w1", clf.coefs_[0].T, (hidden1, FEATURES)),
        txm.floats("b1", clf.intercepts_[0]),
        txm.floats("w2", clf.coefs_[1].T, (hidden2, hidden1)),
        txm.floats("b2", clf.intercepts_[1]),
        txm.floats("w3", clf.coefs_[2].T, (CLASSES, hidden2)),
        txm.floats("b3", clf.intercepts_[2]),
        txm.floats("test_images", test_x),
        txm.ints("test_labels", test_y.astype("<i8")),
        txm.ints("expected_predictions", predicted.astype("<i8")),
        txm.floats("expected_logits", logits),
    ]
    return tensors, errors, accuracy, len(test_y)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sample", action="store_true")
    parser.add_argument("--output", default="")
    parser.add_argument("--hidden", default="128,64")
    parser.add_argument("--max-iter", type=int, default=20)
    parser.add_argument("--rows", type=int, default=0, help="cap the test rows")
    parser.add_argument("--data", default="", help="local .npz with x and y")
    args = parser.parse_args()

    if args.sample:
        path = args.output or str(Path(__file__).with_name("sample.txm"))
        txm.pack(path, sample_tensors())
        print(txm.info(path))
        return 0

    try:
        hidden = tuple(int(part) for part in args.hidden.split(","))
    except ValueError:
        print("--hidden wants sizes like 128,64", file=sys.stderr)
        return 2
    if len(hidden) != 2 or any(size < 1 for size in hidden):
        print("--hidden wants two positive sizes, like 128,64", file=sys.stderr)
        return 2

    try:
        tensors, errors, accuracy, n_test = full_tensors(
            hidden, args.max_iter, args.data, args.rows
        )
    except ImportError as err:
        print("full export needs numpy and scikit-learn: %s" % err, file=sys.stderr)
        print("pass --sample to write the small fixture", file=sys.stderr)
        return 1

    path = args.output or str(Path(__file__).with_name("model.txm"))
    txm.pack(path, tensors)

    problems = txm.verify(
        path,
        {
            "w1": (hidden[0], FEATURES),
            "b1": (hidden[0],),
            "w2": (hidden[1], hidden[0]),
            "b2": (hidden[1],),
            "w3": (CLASSES, hidden[1]),
            "b3": (CLASSES,),
            "test_images": (n_test, FEATURES),
            "test_labels": (n_test,),
            "expected_predictions": (n_test,),
            "expected_logits": (n_test, CLASSES),
        },
    )
    if problems:
        for problem in problems:
            print("verify: %s" % problem, file=sys.stderr)
        return 1

    print(txm.info(path))
    print("wrote %s: %d errors, %.2f%% accuracy" % (path, errors, accuracy))
    print("run main.tx to compare every logit and prediction")
    return 0


if __name__ == "__main__":
    sys.exit(main())
