"""Read and write TXM1 tensor files.

TXM1 is the container `gpu.mount` reads. It holds named numeric tensors and
nothing else: no ops, no layer types, no strides. The Trux program decides what
the names mean. The byte layout is documented in docs/TXM.md.

Writing needs only the Python standard library. numpy arrays are accepted when
numpy is installed, but it is never imported here and never required.

    import txm

    txm.pack("model.txm", [
        txm.floats("w", [[1.5, -0.25]]),
        txm.floats("b", [0.5]),
    ])

    print(txm.info("model.txm"))

Run `python3 txm.py info model.txm` for a one-off look at a file.
"""

from __future__ import annotations

import array
import os
import struct
import sys
import tempfile

MAGIC = b"TXM1"
VERSION = 1

DTYPE_INT = 1
DTYPE_FLOAT = 2
DTYPE_NAMES = {DTYPE_INT: "int", DTYPE_FLOAT: "float"}

MAX_TENSORS = 256
MAX_NAME_BYTES = 64
MAX_RANK = 4
# The reader rejects a dimension product whose byte size overflows int64.
MAX_ELEMENTS = (1 << 63) // 8

_NUMERIC_KINDS = frozenset("iuf")


class TXMError(Exception):
    """A file or tensor that TXM1 cannot represent."""


class TensorSpec:
    """One tensor ready to be written: metadata plus little-endian payload."""

    __slots__ = ("name", "dims", "dtype", "payload")

    def __init__(self, name: str, dims: tuple[int, ...], dtype: int, payload: bytes):
        self.name = name
        self.dims = dims
        self.dtype = dtype
        self.payload = payload

    @property
    def count(self) -> int:
        total = 1
        for dim in self.dims:
            total *= dim
        return total

    def __repr__(self) -> str:
        return "TensorSpec(%r, %r, %s)" % (
            self.name,
            self.dims,
            DTYPE_NAMES.get(self.dtype, self.dtype),
        )


class Tensor:
    """One tensor read from a file. `values` stays empty unless asked for."""

    __slots__ = ("name", "dims", "dtype", "values")

    def __init__(self, name: str, dims: tuple[int, ...], dtype: int, values=None):
        self.name = name
        self.dims = dims
        self.dtype = dtype
        self.values = values

    @property
    def count(self) -> int:
        total = 1
        for dim in self.dims:
            total *= dim
        return total

    @property
    def shape(self) -> tuple[int, ...]:
        return self.dims

    def __repr__(self) -> str:
        return "Tensor(%r, %r, %s)" % (
            self.name,
            self.dims,
            DTYPE_NAMES.get(self.dtype, self.dtype),
        )


def _sequence_shape(values, name: str):
    """The shape of a plain nested sequence, or None if it is not one."""
    if not hasattr(values, "__len__") or isinstance(values, (str, bytes)):
        return None
    if len(values) == 0:
        return (0,)
    first = values[0]
    if isinstance(first, (list, tuple)):
        inner = _sequence_shape(first, name)
        if inner is None:
            return None
        for row in values:
            if _sequence_shape(row, name) != inner:
                raise TXMError("%s: nested values are not rectangular" % name)
        return (len(values),) + inner
    return (len(values),)


def _shape(name: str, values, dims) -> tuple[int, ...]:
    if dims is not None:
        try:
            dims = tuple(int(dim) for dim in dims)
        except (TypeError, ValueError):
            raise TXMError("%s: dims must be whole numbers" % name) from None
    elif getattr(values, "shape", None) is not None:
        dims = tuple(int(dim) for dim in values.shape)
    else:
        dims = _sequence_shape(values, name)
        if dims is None:
            raise TXMError("%s: pass dims= when values is a flat sequence" % name)

    if not dims:
        raise TXMError("%s: a tensor needs at least one dimension" % name)
    if len(dims) > MAX_RANK:
        raise TXMError("%s: rank %d is above %d" % (name, len(dims), MAX_RANK))
    if any(dim < 1 for dim in dims):
        raise TXMError("%s: every dimension must be at least 1" % name)

    count = 1
    for dim in dims:
        if count > MAX_ELEMENTS // dim:
            raise TXMError("%s: dimension product is too large" % name)
        count *= dim
    return dims


def _numpy_kind(values):
    """'i', 'u', or 'f' for a numpy array, None for anything else."""
    dtype = getattr(values, "dtype", None)
    if dtype is None or not hasattr(values, "tobytes"):
        return None
    return getattr(dtype, "kind", None)


def _flatten(values):
    for value in values:
        if isinstance(value, (list, tuple)):
            yield from _flatten(value)
        else:
            yield value


def _float_payload(name: str, values) -> bytes:
    kind = _numpy_kind(values)
    if kind is not None:
        if kind not in _NUMERIC_KINDS:
            raise TXMError("%s: unsupported dtype %s" % (name, values.dtype))
        return values.astype("<f8", copy=False).tobytes()

    try:
        data = array.array("d", (float(value) for value in _flatten(values)))
    except (TypeError, ValueError):
        raise TXMError("%s: values must be numbers" % name) from None
    if data.itemsize != 8:
        raise TXMError("this platform stores double in %d bytes" % data.itemsize)
    if sys.byteorder != "little":
        data.byteswap()
    return data.tobytes()


def _int_payload(name: str, values) -> bytes:
    kind = _numpy_kind(values)
    if kind is not None:
        if kind not in ("i", "u"):
            raise TXMError("%s: int tensor given %s values" % (name, values.dtype))
        return values.astype("<i8", copy=False).tobytes()

    whole = []
    for value in _flatten(values):
        if isinstance(value, bool) or not isinstance(value, int):
            raise TXMError("%s: int tensor given a non-integer %r" % (name, value))
        whole.append(value)
    data = array.array("q", whole)
    if data.itemsize != 8:
        raise TXMError("this platform stores int64 in %d bytes" % data.itemsize)
    if sys.byteorder != "little":
        data.byteswap()
    return data.tobytes()


def floats(name: str, values, dims=None) -> TensorSpec:
    """A float64 tensor. Shape comes from a numpy array or from `dims`."""
    dims = _shape(name, values, dims)
    spec = TensorSpec(name, dims, DTYPE_FLOAT, b"")
    spec.payload = _float_payload(name, values)
    if len(spec.payload) != spec.count * 8:
        raise TXMError(
            "%s: expected %d values, got %d"
            % (name, spec.count, len(spec.payload) // 8)
        )
    return spec


def ints(name: str, values, dims=None) -> TensorSpec:
    """An int64 tensor. Shape comes from a numpy array or from `dims`."""
    dims = _shape(name, values, dims)
    spec = TensorSpec(name, dims, DTYPE_INT, b"")
    spec.payload = _int_payload(name, values)
    if len(spec.payload) != spec.count * 8:
        raise TXMError(
            "%s: expected %d values, got %d"
            % (name, spec.count, len(spec.payload) // 8)
        )
    return spec


def _validate(specs: list[TensorSpec]) -> None:
    if not 1 <= len(specs) <= MAX_TENSORS:
        raise TXMError("a file holds 1 to %d tensors, got %d" % (MAX_TENSORS, len(specs)))

    seen = set()
    for spec in specs:
        if not isinstance(spec, TensorSpec):
            raise TXMError("pack expects the result of floats() or ints()")
        name_bytes = spec.name.encode("utf-8")
        if not 1 <= len(name_bytes) <= MAX_NAME_BYTES:
            raise TXMError(
                "%s: name is %d bytes, want 1 to %d"
                % (spec.name, len(name_bytes), MAX_NAME_BYTES)
            )
        if spec.name in seen:
            raise TXMError("%s: duplicate tensor name" % spec.name)
        seen.add(spec.name)
        if spec.dtype not in (DTYPE_INT, DTYPE_FLOAT):
            raise TXMError("%s: bad dtype %r" % (spec.name, spec.dtype))
        if not 1 <= len(spec.dims) <= MAX_RANK:
            raise TXMError("%s: rank must be 1 to %d" % (spec.name, MAX_RANK))
        if len(spec.payload) != spec.count * 8:
            raise TXMError("%s: payload does not match its shape" % spec.name)


def pack(path, tensors) -> None:
    """Write `tensors` to `path` as one TXM1 file.

    The file is written beside the destination and renamed into place, so a
    failure never leaves a half-written model behind.
    """
    specs = list(tensors)
    _validate(specs)

    directory = os.path.dirname(os.path.abspath(path)) or "."
    handle, temp = tempfile.mkstemp(dir=directory, prefix=".txm-", suffix=".tmp")
    try:
        with os.fdopen(handle, "wb") as out:
            out.write(MAGIC)
            out.write(struct.pack("<II", VERSION, len(specs)))
            for spec in specs:
                name_bytes = spec.name.encode("utf-8")
                out.write(struct.pack("<I", len(name_bytes)))
                out.write(name_bytes)
                out.write(struct.pack("<II", spec.dtype, len(spec.dims)))
                for dim in spec.dims:
                    out.write(struct.pack("<Q", dim))
                out.write(spec.payload)
        os.chmod(temp, 0o644)
        os.replace(temp, path)
    except BaseException:
        try:
            os.unlink(temp)
        except OSError:
            pass
        raise


def read(path, values: bool = False) -> list[Tensor]:
    """Read and validate a TXM1 file.

    With `values=False` only the metadata is returned, which keeps a large file
    cheap to inspect. With `values=True` each tensor holds a flat list.
    """
    with open(path, "rb") as handle:
        data = handle.read()

    if len(data) < 12:
        raise TXMError("file is smaller than a TXM1 header")
    if data[:4] != MAGIC:
        raise TXMError("bad magic, want TXM1")

    version, count = struct.unpack_from("<II", data, 4)
    if version != VERSION:
        raise TXMError("bad version %d, want %d" % (version, VERSION))
    if not 1 <= count <= MAX_TENSORS:
        raise TXMError("tensor count %d is outside 1..%d" % (count, MAX_TENSORS))

    offset = 12
    tensors = []
    seen = set()
    for _ in range(count):
        if offset + 4 > len(data):
            raise TXMError("file is truncated in a tensor header")
        (name_len,) = struct.unpack_from("<I", data, offset)
        offset += 4
        if not 1 <= name_len <= MAX_NAME_BYTES:
            raise TXMError("tensor name length %d is outside 1..%d" % (name_len, MAX_NAME_BYTES))
        if offset + name_len > len(data):
            raise TXMError("file is truncated in a tensor name")
        name = data[offset : offset + name_len].decode("utf-8", "replace")
        offset += name_len

        if offset + 8 > len(data):
            raise TXMError("file is truncated in a tensor header")
        dtype, rank = struct.unpack_from("<II", data, offset)
        offset += 8
        if dtype not in (DTYPE_INT, DTYPE_FLOAT):
            raise TXMError("%s: bad dtype %d" % (name, dtype))
        if not 1 <= rank <= MAX_RANK:
            raise TXMError("%s: rank %d is outside 1..%d" % (name, rank, MAX_RANK))

        if offset + rank * 8 > len(data):
            raise TXMError("file is truncated in a tensor shape")
        dims = []
        product = 1
        for _axis in range(rank):
            (dim,) = struct.unpack_from("<Q", data, offset)
            offset += 8
            if dim < 1:
                raise TXMError("%s: a dimension is zero" % name)
            if product > MAX_ELEMENTS // dim:
                raise TXMError("%s: dimension product is too large" % name)
            product *= dim
            dims.append(dim)

        nbytes = product * 8
        if offset + nbytes > len(data):
            raise TXMError("%s: payload is truncated" % name)
        offset += nbytes

        if name in seen:
            raise TXMError("%s: duplicate tensor name" % name)
        seen.add(name)

        parsed = None
        if values:
            start = offset - nbytes
            code = "<%d%s" % (product, "d" if dtype == DTYPE_FLOAT else "q")
            parsed = list(struct.unpack(code, data[start:offset]))
        tensors.append(Tensor(name, tuple(dims), dtype, parsed))

    if offset != len(data):
        raise TXMError("%d trailing bytes after the last tensor" % (len(data) - offset))
    return tensors


def verify(path, expected: dict) -> list[str]:
    """Compare a file against `expected`, a mapping of name to shape.

    Returns a list of problems. An empty list means every expected tensor is
    present with the right shape. Extra tensors are allowed.
    """
    found = {tensor.name: tensor for tensor in read(path)}
    problems = []
    for name, dims in expected.items():
        tensor = found.get(name)
        if tensor is None:
            problems.append("missing %s" % name)
            continue
        want = tuple(int(dim) for dim in dims)
        if tensor.dims != want:
            problems.append("%s has shape %s, want %s" % (name, list(tensor.dims), list(want)))
    return problems


def info(path) -> str:
    """A short table of the tensors in a file."""
    tensors = read(path)
    width = max([len("name")] + [len(tensor.name) for tensor in tensors])
    lines = ["%-*s  %-14s  %-5s  %s" % (width, "name", "shape", "dtype", "elements")]
    for tensor in tensors:
        shape = "[%s]" % ", ".join(str(dim) for dim in tensor.dims)
        lines.append(
            "%-*s  %-14s  %-5s  %d"
            % (width, tensor.name, shape, DTYPE_NAMES[tensor.dtype], tensor.count)
        )
    total = os.path.getsize(path)
    lines.append(
        "%d tensors, %d bytes, %s"
        % (len(tensors), total, path)
    )
    return "\n".join(lines)


if __name__ == "__main__":
    if len(sys.argv) != 3 or sys.argv[1] != "info":
        print("usage: python3 txm.py info PATH", file=sys.stderr)
        raise SystemExit(2)
    try:
        print(info(sys.argv[2]))
    except (OSError, TXMError) as err:
        print("txm: %s" % err, file=sys.stderr)
        raise SystemExit(1)
