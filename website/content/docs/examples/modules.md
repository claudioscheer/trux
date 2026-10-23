---
title: "Modules"
description: "Multi-file programs, relative imports, pub names, and package layout."
icon: "account_tree"
weight: 40
---

## modules_basic

One import, a `pub type Pair struct`, and `pub func`.

{{< details summary="`examples/modules/basic/main.tx`" >}}

{{< include "examples/modules/basic/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/basic/main.tx
```

{{< include "website/outputs/modules_basic.txt" "text" >}}

## modules_transitive

A chain of imports (`main` → `app` → `text`) returning an exported `app.Status` struct.

{{< details summary="`examples/modules/transitive/main.tx`" >}}

{{< include "examples/modules/transitive/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/transitive/main.tx
```

{{< include "website/outputs/modules_transitive.txt" "text" >}}

## modules_private_names

Same-named private structs and helpers in different packages.

{{< details summary="`examples/modules/private_names/main.tx`" >}}

{{< include "examples/modules/private_names/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/private_names/main.tx
```

{{< include "website/outputs/modules_private_names.txt" "text" >}}

## modules_same_package_math

One package spread over several files, sharing an exported `Pair` struct.

{{< details summary="`examples/modules/same_package_math/main.tx`" >}}

{{< include "examples/modules/same_package_math/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/same_package_math/main.tx
```

{{< include "website/outputs/modules_same_package_math.txt" "text" >}}

## modules_mut_parameters

A stack and a `Queue` struct in separate packages, using `mut` parameters and `*queue.Queue` pointers.

{{< details summary="`examples/modules/mut_parameters/main.tx`" >}}

{{< include "examples/modules/mut_parameters/main.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux run examples/modules/mut_parameters/main.tx
```

{{< include "website/outputs/modules_mut_parameters.txt" "text" >}}
