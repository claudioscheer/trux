---
title: "Basics"
description: "Hello, arithmetic, primitives, and strings: the smallest complete programs."
icon: "code"
weight: 10
---

## hello

A function, a variable, and `print`.

{{< details summary="`examples/hello.tx`" >}}

{{< include "examples/hello.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/hello.tx
```

{{< include "website/outputs/hello.txt" "text" >}}

## arithmetic

Small integer functions and nested calls.

{{< details summary="`examples/arithmetic.tx`" >}}

{{< include "examples/arithmetic.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/arithmetic.tx
```

{{< include "website/outputs/arithmetic.txt" "text" >}}

## primitives

All four scalar types, string escapes, and multi-argument `print`.

{{< details summary="`examples/primitives.tx`" >}}

{{< include "examples/primitives.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/primitives.tx
```

{{< include "website/outputs/primitives.txt" "text" >}}

## strings

Concatenation, equality, and the `in` operator.

{{< details summary="`examples/strings.tx`" >}}

{{< include "examples/strings.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/strings.tx
```

{{< include "website/outputs/strings.txt" "text" >}}
