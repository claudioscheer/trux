# Image Package

Trux has a first image package for simple RGB image experiments.

```trux
import "image"

image.width(path string) int
image.height(path string) int
image.readPpm(path string) []int
image.writePpm(path string, pixels []int, width int, height int)
```

`image.readPpm` reads a P3 PPM file and returns flat RGB channel values:

```text
R G B R G B R G B ...
```

For pixel index `p`, the red channel starts at `p * 3`, green at `p * 3 + 1`,
and blue at `p * 3 + 2`.

The current runtime supports uncompressed text P3 PPM with max value `255`.
This is intentionally small because PPM is easy to parse and works well for
CPU/GPU pixel examples. General PNG/JPEG decoding is not implemented yet.

See [examples/projects/rotate_image](../examples/projects/rotate_image) for a
CPU and GPU image rotation example.
