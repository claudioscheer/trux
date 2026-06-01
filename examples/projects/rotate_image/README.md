# Rotate Image

This project rotates an image twice in the same Trux program: once on the CPU
and once with a CUDA kernel.

The original downloaded asset is `source.jpg`. Trux currently reads the
converted `source.ppm` file because PPM is a simple uncompressed RGB format.

Convert any local image to the PPM format used by this example:

```sh
examples/projects/rotate_image/to_ppm.sh input.png output.ppm
```

Set `turns` in `main.tx` to choose the number of clockwise quarter turns. For
example, `1` is 90 degrees, `2` is 180 degrees, `3` is 270 degrees, and larger
or negative values are normalized to the same four cases.

Run from the repository root:

```sh
make run FILE=./examples/projects/rotate_image/main.tx
```

The program writes:

- `rotated_cpu.ppm`
- `rotated_gpu.ppm`

It also prints width, height, CPU/GPU checksums, mismatch count, and timings.
