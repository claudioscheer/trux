---
title: "math"
description: "Exponential and square root for host code."
icon: "calculate"
weight: 250
---

```trux
import "math"
```

**What it is.** Two `float` functions for host code: `e` raised to a power, and a square root.

**Why.** Layer normalization needs a square root and a softmax needs `e^x`. Those are the two operations a transformer
cannot express with the arithmetic operators alone.

**When you don't need it.** GPU kernels have bare [`exp` and `sqrt`](/docs/guide/gpu/) builtins. Use those inside a
kernel.

| Function | Returns | Description |
| --- | --- | --- |
| `math.exp(x float)` | `float` | `e` raised to the power `x`. |
| `math.sqrt(x float)` | `float` | Square root of `x`. |

```trux
package main

import "math"

func main() int {
  print(math.exp(0.0) == 1.0)
  print(math.sqrt(4.0) == 2.0)
  print(math.sqrt(2.0))
  return 0
}
```

## Values and failure

Both functions return a `float` and have no `error` result:

- `math.sqrt(x)` for `x < 0` follows IEEE/C behavior and returns NaN. Layer norm always passes `variance + epsilon`,
  which is positive.
- `math.exp(x)` overflows to infinity for large `x`. The softmax subtracts the row maximum first, so its exponent is
  never positive.
- Neither function sets `errno` or stops the program. Check the result if your input can be outside the domain.

Wrong argument counts and non-`float` arguments are compile errors.

## Kernels

Inside a `kernel func`, use the bare builtins:

```trux
let inv float = 1.0 / sqrt(variance + 0.00001)
let weight float = exp(score - best)
```

They compile to CUDA's double-precision `sqrt` and `exp`. The qualified host calls, `math.sqrt` and `math.exp`, are not
allowed in a kernel.

An import is visible to the whole file, including its kernels. Importing `math` makes the bare names `sqrt` and `exp`
resolve to the imported functions, which the kernel checker rejects. When one program needs host math and kernels, put
the host calls in a small module such as
[`hostmath.tx`](https://github.com/thea1lab/trux/blob/main/examples/projects/tiny_transformer/hostmath.tx), as the
[Tiny transformer](/docs/examples/tiny-transformer/) does.

## Related

- [Tiny transformer](/docs/examples/tiny-transformer/): the `math` calls in layer norm and softmax, checked against
  PyTorch.
- [GPU programming](/docs/guide/gpu/): kernels and launches.
- [`time`](/docs/stdlib/time/): the other module used by the transformer timings.
