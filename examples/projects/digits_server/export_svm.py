#!/usr/bin/env python3
"""Write a TXM1 file for the handwritten-digits SVM.

The default writes the full model next to this script as digits.txm and checks
the host vote against sklearn. That file is generated locally and is not part
of the repository. --sample writes the small sample.txm fixture, which only
has the tensor names and shapes the server expects.
"""

import argparse
import struct
import sys


def write_u32(out, value):
    out.extend(struct.pack("<I", value))


def write_u64(out, value):
    out.extend(struct.pack("<Q", value))


def add_tensor(out, name, dtype, dims, payload):
    raw = name.encode("utf-8")
    if not 1 <= len(raw) <= 64:
        raise SystemExit("bad tensor name length")
    write_u32(out, len(raw))
    out.extend(raw)
    write_u32(out, dtype)
    write_u32(out, len(dims))
    for dim in dims:
        if dim < 1:
            raise SystemExit("dimension must be at least 1")
        write_u64(out, dim)
    out.extend(payload)


def write_file(path, tensors):
    out = bytearray(b"TXM1")
    write_u32(out, 1)
    write_u32(out, len(tensors))
    for name, dtype, dims, payload in tensors:
        add_tensor(out, name, dtype, dims, payload)
    with open(path, "wb") as handle:
        handle.write(out)


def sample_tensors():
    gamma = struct.pack("<d", 0.001)
    support = bytearray()
    for sv in range(2):
        for feature in range(64):
            support.extend(struct.pack("<d", float(sv + 1) * float(feature)))
    dual = bytearray()
    for row in range(9):
        for col in range(2):
            dual.extend(struct.pack("<d", 0.01 * float(row + 1)))
    intercept = struct.pack("<d", 0.0) * 45
    n_support = struct.pack("<q", 1) + struct.pack("<q", 1) + (struct.pack("<q", 0) * 8)
    return [
        ("gamma", 2, (1,), gamma),
        ("support", 2, (2, 64), bytes(support)),
        ("dual", 2, (9, 2), bytes(dual)),
        ("intercept", 2, (45,), intercept),
        ("n_support", 1, (10,), n_support),
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
    clf = SVC(gamma=0.001)
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
        ("gamma", 2, (1,), struct.pack("<d", gamma)),
        ("support", 2, support.shape, support.tobytes()),
        ("dual", 2, dual.shape, dual.tobytes()),
        ("intercept", 2, intercept.shape, intercept.tobytes()),
        ("n_support", 1, n_support.shape, n_support.tobytes()),
    ]
    return tensors


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sample", action="store_true")
    parser.add_argument("--output", default="")
    args = parser.parse_args()
    if args.sample:
        path = args.output or "sample.txm"
        write_file(path, sample_tensors())
        print("wrote %s" % path)
        return
    try:
        tensors = full_tensors()
    except ImportError as err:
        print("full export needs numpy and scikit-learn: %s" % err, file=sys.stderr)
        print("pass --sample to write the small fixture", file=sys.stderr)
        return 1
    path = args.output or "digits.txm"
    write_file(path, tensors)
    print("wrote %s" % path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
