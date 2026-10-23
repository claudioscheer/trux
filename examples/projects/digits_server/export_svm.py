#!/usr/bin/env python3
"""Write a TXM1 file for the handwritten-digits SVM.

The default writes the full model next to this script as digits.txm and checks
the host vote against sklearn. That file is generated locally and is not part
of the repository. --sample writes the small sample.txm fixture, which only
has the tensor names and shapes the server expects.
"""

import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "python"))

import txm


def sample_tensors():
    support = []
    for sv in range(2):
        for feature in range(64):
            support.append(float(sv + 1) * float(feature))
    dual = []
    for row in range(9):
        for col in range(2):
            dual.append(0.01 * float(row + 1))
    n_support = [1, 1, 0, 0, 0, 0, 0, 0, 0, 0]
    return [
        txm.floats("gamma", [0.001]),
        txm.floats("support", support, (2, 64)),
        txm.floats("dual", dual, (9, 2)),
        txm.floats("intercept", [0.0] * 45),
        txm.ints("n_support", n_support),
    ]


def vote_digit(x, gamma, support, dual, intercept, n_support):
    import numpy as np

    diff = support - x
    dist = np.sum(diff * diff, axis=1)
    kernel = np.exp(-gamma * dist)
    n_classes = len(n_support)
    start = np.cumsum(n_support) - n_support
    n_sv = support.shape[0]
    votes = np.zeros(n_classes, dtype=int)
    decisions = []
    pair = 0
    for i in range(n_classes):
        for j in range(i + 1, n_classes):
            total = float(intercept[pair])
            for k in range(int(n_support[i])):
                idx = int(start[i] + k)
                total += float(dual[j - 1, idx]) * float(kernel[idx])
            for k in range(int(n_support[j])):
                idx = int(start[j] + k)
                total += float(dual[i, idx]) * float(kernel[idx])
            decisions.append(total)
            if total > 0.0:
                votes[i] += 1
            else:
                votes[j] += 1
            pair += 1
    if n_sv != dual.shape[1]:
        raise SystemExit("dual width does not match support rows")
    return int(np.argmax(votes)), np.asarray(decisions, dtype=float)


def full_tensors():
    import numpy as np
    from sklearn.datasets import load_digits
    from sklearn.model_selection import train_test_split
    from sklearn.svm import SVC

    digits = load_digits()
    images = digits.images.reshape((len(digits.images), -1))
    train_x, test_x, train_y, test_y = train_test_split(
        images, digits.target, test_size=0.5, shuffle=False
    )
    # ovo matches the 45 pairwise sums in vote_digit. The default ovr shape
    # is 10 numbers and does not change the fitted weights.
    clf = SVC(gamma=0.001, decision_function_shape="ovo")
    clf.fit(train_x, train_y)
    gamma = float(clf._gamma)
    if abs(gamma - 0.001) > 1e-15:
        raise SystemExit("sklearn gamma is %r, want 0.001" % gamma)

    support = np.ascontiguousarray(clf.support_vectors_, dtype="<f8")
    dual = np.ascontiguousarray(clf.dual_coef_, dtype="<f8")
    intercept = np.ascontiguousarray(clf.intercept_, dtype="<f8")
    n_support = np.ascontiguousarray(clf.n_support_, dtype="<i8")
    if support.shape != (n_support.sum(), 64):
        raise SystemExit("support shape %s" % (support.shape,))
    if dual.shape != (9, support.shape[0]):
        raise SystemExit("dual shape %s" % (dual.shape,))
    if intercept.shape != (45,):
        raise SystemExit("intercept shape %s" % (intercept.shape,))
    if n_support.shape != (10,):
        raise SystemExit("n_support shape %s" % (n_support.shape,))

    mismatches = 0
    for i in range(len(test_x)):
        digit, decisions = vote_digit(
            test_x[i], gamma, support, dual, intercept, n_support
        )
        if digit != int(clf.predict(test_x[i : i + 1])[0]):
            mismatches += 1
        sk = clf.decision_function(test_x[i : i + 1])[0]
        if not np.allclose(decisions, sk):
            raise SystemExit("decision mismatch at test row %d" % i)
    if mismatches:
        raise SystemExit("%d vote mismatches" % mismatches)
    print("checked %d test rows against sklearn" % len(test_x))

    tensors = [
        txm.floats("gamma", [gamma]),
        txm.floats("support", support),
        txm.floats("dual", dual),
        txm.floats("intercept", intercept),
        txm.ints("n_support", n_support),
    ]
    return tensors


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sample", action="store_true")
    parser.add_argument("--output", default="")
    args = parser.parse_args()
    if args.sample:
        path = args.output or "sample.txm"
        txm.pack(path, sample_tensors())
        print("wrote %s" % path)
        return
    try:
        tensors = full_tensors()
    except ImportError as err:
        print("full export needs numpy and scikit-learn: %s" % err, file=sys.stderr)
        print("pass --sample to write the small fixture", file=sys.stderr)
        return 1
    path = args.output or "digits.txm"
    txm.pack(path, tensors)
    print("wrote %s" % path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
