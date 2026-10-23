---
title: "Programs that fail on purpose"
description: "One small program per common mistake, with the real compiler and runtime errors."
icon: "report"
weight: 100
---

[`examples/errors/`](https://github.com/thea1lab/trux/tree/main/examples/errors) holds one small program per common
mistake. Each one is compiled or run by `website/update-example-outputs.sh`, and the recorded output is shown on the
guide page for the feature it belongs to. CI re-runs them, so the message on the page is the message the current
compiler prints.

| Kind | What it covers | Where to read it |
| --- | --- | --- |
| Compile errors | missing `main`, mixed numbers, bad escapes, duplicate and undefined names | [Program structure](/docs/guide/program-structure/), [Types](/docs/guide/types/), [Variables](/docs/guide/variables/) |
| Runtime errors | overflow, division by zero, a bad index, a nil dereference, a failed `assert` | [Errors and failures](/docs/guide/errors/) |
| Collections and memory | out-of-bounds slices, `append` on an array, unknown ownership | [Collections](/docs/guide/collections/), [Memory](/docs/guide/memory/) |
| Modules and tests | bare imports, private names, import cycles, failing tests | [Modules](/docs/guide/modules/), [Testing](/docs/guide/testing/) |
| GPU kernels | returns, strings, calls, and `print` inside a kernel; bad launch shapes | [GPU programming](/docs/guide/gpu/) |

The directory names match the guide pages, so `examples/errors/types_mixed_numbers.tx` is quoted on the types page.
