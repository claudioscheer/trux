---
title: "Control flow"
description: "if/else, the three for loops, early returns, recursion, and runtime asserts."
icon: "alt_route"
weight: 20
---

## control_flow

`if` / `else if` / `else` and a condition loop.

{{< details summary="`examples/control_flow.tx`" >}}

{{< include "examples/control_flow.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/control_flow.tx
```

{{< include "website/outputs/control_flow.txt" "text" >}}

## else_if

A chain of `else if` branches.

{{< details summary="`examples/else_if.tx`" >}}

{{< include "examples/else_if.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/else_if.tx
```

{{< include "website/outputs/else_if.txt" "text" >}}

## loops

All three `for` forms.

{{< details summary="`examples/loops.tx`" >}}

{{< include "examples/loops.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/loops.tx
```

{{< include "website/outputs/loops.txt" "text" >}}

## return_paths

Returning from branches and loops.

{{< details summary="`examples/return_paths.tx`" >}}

{{< include "examples/return_paths.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/return_paths.tx
```

{{< include "website/outputs/return_paths.txt" "text" >}}

## fibonacci

Recursive and iterative Fibonacci.

{{< details summary="`examples/fibonacci.tx`" >}}

{{< include "examples/fibonacci.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/fibonacci.tx
```

{{< include "website/outputs/fibonacci.txt" "text" >}}

## assertions

Runtime checks with `assert`.

{{< details summary="`examples/assertions.tx`" >}}

{{< include "examples/assertions.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/assertions.tx
```

{{< include "website/outputs/assertions.txt" "text" >}}
