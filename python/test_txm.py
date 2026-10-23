"""Tests for txm.py. Run with: python3 -m unittest discover -s python"""

import os
import struct
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import txm

try:
    import numpy as np
except ImportError:  # the format module must work without numpy
    np = None

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DIGITS_SAMPLE = os.path.join(REPO, "examples", "projects", "digits_server", "sample.txm")


class RoundTrip(unittest.TestCase):
    def test_float_and_int(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "model.txm")
            txm.pack(
                path,
                [
                    txm.floats("w", [[1.5, -0.25], [0.0, 2.0]]),
                    txm.floats("b", [0.5]),
                    txm.ints("n", [3, 4]),
                ],
            )
            tensors = {tensor.name: tensor for tensor in txm.read(path, values=True)}
            self.assertEqual(tensors["w"].shape, (2, 2))
            self.assertEqual(tensors["w"].values, [1.5, -0.25, 0.0, 2.0])
            self.assertEqual(tensors["w"].dtype, txm.DTYPE_FLOAT)
            self.assertEqual(tensors["b"].shape, (1,))
            self.assertEqual(tensors["n"].values, [3, 4])
            self.assertEqual(tensors["n"].dtype, txm.DTYPE_INT)

    def test_metadata_only_skips_values(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "model.txm")
            txm.pack(path, [txm.floats("w", [1.0, 2.0])])
            (tensor,) = txm.read(path)
            self.assertIsNone(tensor.values)

    def test_info_lists_every_tensor(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "model.txm")
            txm.pack(path, [txm.floats("w", [1.0]), txm.ints("n", [2])])
            text = txm.info(path)
            self.assertIn("w", text)
            self.assertIn("n", text)
            self.assertIn("[1]", text)

    def test_committed_digits_sample_repacks_byte_for_byte(self):
        specs = []
        for tensor in txm.read(DIGITS_SAMPLE, values=True):
            make = txm.floats if tensor.dtype == txm.DTYPE_FLOAT else txm.ints
            specs.append(make(tensor.name, tensor.values, tensor.dims))
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "repacked.txm")
            txm.pack(path, specs)
            with open(DIGITS_SAMPLE, "rb") as original, open(path, "rb") as made:
                self.assertEqual(original.read(), made.read())


class NumpyInputs(unittest.TestCase):
    @unittest.skipIf(np is None, "numpy is not installed")
    def test_transposed_view_is_written_row_major(self):
        base = np.arange(6, dtype=np.float64).reshape(2, 3)
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "model.txm")
            txm.pack(path, [txm.floats("w", base.T, (3, 2))])
            (tensor,) = txm.read(path, values=True)
            self.assertEqual(tensor.dims, (3, 2))
            self.assertEqual(tensor.values, [0.0, 3.0, 1.0, 4.0, 2.0, 5.0])

    @unittest.skipIf(np is None, "numpy is not installed")
    def test_uint8_becomes_float(self):
        pixels = np.arange(4, dtype=np.uint8)
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "model.txm")
            txm.pack(path, [txm.floats("x", pixels)])
            (tensor,) = txm.read(path, values=True)
            self.assertEqual(tensor.dims, (4,))
            self.assertEqual(tensor.values, [0.0, 1.0, 2.0, 3.0])


class Rejects(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.directory.name, "model.txm")

    def tearDown(self):
        self.directory.cleanup()

    def test_duplicate_name(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [txm.floats("w", [1.0]), txm.floats("w", [2.0])])

    def test_name_too_long(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [txm.floats("x" * 65, [1.0])])

    def test_no_tensors(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [])

    def test_rank_above_four(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [txm.floats("w", [1.0], (1, 1, 1, 1, 1))])

    def test_zero_dimension(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [txm.floats("w", [], (0,))])

    def test_shape_and_values_disagree(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [txm.floats("w", [1.0, 2.0], (3,))])

    def test_float_values_in_int_tensor(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [txm.ints("n", [1.5])])

    def test_reader_rejects_trailing_bytes(self):
        txm.pack(self.path, [txm.floats("w", [1.0])])
        with open(self.path, "ab") as handle:
            handle.write(b"\x00")
        with self.assertRaises(txm.TXMError):
            txm.read(self.path)

    def test_reader_rejects_bad_magic(self):
        with open(self.path, "wb") as handle:
            handle.write(b"XXXX" + struct.pack("<II", 1, 1))
        with self.assertRaises(txm.TXMError):
            txm.read(self.path)

    def test_failed_pack_leaves_no_file(self):
        with self.assertRaises(txm.TXMError):
            txm.pack(self.path, [txm.floats("w", [1.0]), txm.floats("w", [2.0])])
        self.assertFalse(os.path.exists(self.path))
        self.assertEqual([name for name in os.listdir(self.directory.name) if name.endswith(".tmp")], [])


class Verify(unittest.TestCase):
    def test_reports_missing_and_wrong_shape(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "model.txm")
            txm.pack(path, [txm.floats("w", [[1.0, 2.0]])])
            self.assertEqual(txm.verify(path, {"w": (1, 2)}), [])
            self.assertEqual(len(txm.verify(path, {"w": (2, 1)})), 1)
            self.assertEqual(len(txm.verify(path, {"nope": (1,)})), 1)


if __name__ == "__main__":
    unittest.main()
