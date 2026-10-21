---
title: "image"
description: "Reading and writing PPM images."
icon: "image"
weight: 200
---

```trux
import "image"
```

**What it is.** Reads and writes **P3 PPM** files, an uncompressed plain-text RGB format with channel values up to
`255`, as flat `[]int` slices.

**Why.** PPM is simple enough to parse without a library, which makes it a good format for pixel experiments on the CPU
and GPU.

**When you don't need it.** For PNG or JPEG files, convert them first (see below); there is no general image decoder.

| Function | Returns | Description |
| --- | --- | --- |
| `image.width(path string)` | `int` | Width in pixels, read from the file header. |
| `image.height(path string)` | `int` | Height in pixels, read from the file header. |
| `image.readPpm(path string)` | `[]int` | All channel values as a flat slice: `R G B R G B ...`, row by row. |
| `image.writePpm(path string, pixels []int, width int, height int)` | | Writes a P3 PPM file (statement only). Requires `len(pixels) == width * height * 3`. |

For pixel number `p` (counting left to right, top to bottom), red is at `p * 3`, green at `p * 3 + 1`, and blue at
`p * 3 + 2`. Pixel `(row, col)` is `p = row * width + col`.

## Example: invert an image

```trux
package main

import "image"

func main() int {
  let path string = "examples/projects/rotate_image/source.ppm"
  let width int = image.width(path)
  let height int = image.height(path)
  let pixels []int = image.readPpm(path)

  for let i int = 0; i < len(pixels); i = i + 1 {
    pixels[i] = 255 - pixels[i]
  }

  image.writePpm("inverted.ppm", pixels, width, height)
  print(width, "x", height)
  return 0
}
```

Run it from the repository root; it writes `inverted.ppm` to the current directory.

## Errors

**Common mistakes.** A pixel slice whose length is not `width * height * 3`:

{{< include "examples/errors/image_pixel_count.tx" "trux" >}}

{{< include "website/outputs/errors/image_pixel_count.txt" "text" >}}

Reading a missing or malformed file also stops the program with a `trux runtime error`.

## Converting images to PPM

General PNG and JPEG decoding is not implemented. `examples/projects/rotate_image/to_ppm.sh` converts an image to the
P3 PPM format this package reads:

```bash
examples/projects/rotate_image/to_ppm.sh input.png output.ppm
```

See `examples/projects/rotate_image` for a CPU and GPU rotation example, and [Image package](/docs/design-notes/image/) for the design
notes.
