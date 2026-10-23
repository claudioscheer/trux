---
title: "Design notes"
description: "The design documents behind the language, kept in docs/."
icon: "description"
weight: 800
cards: false
---

These documents explain *why* Trux works the way it does. They are written for contributors and curious readers, and
they predate this site, so some of them use a terser style than the guide.

| Note | Topic |
| --- | --- |
| [Language specification](/docs/design-notes/specs/) | the compact, complete reference for the current language |
| [Arenas](/docs/design-notes/arenas/) | the arena memory model, borrowed and owned returns, `mut`, and `clone` |
| [Modules design](/docs/design-notes/modules/) | how imports and packages are loaded and merged, and the plan toward separate compilation |
| [GPU design](/docs/design-notes/gpu/) | the CUDA model: device buffers, kernels, and launches |
| [IO packages](/docs/design-notes/io/) | why IO lives in the `io` and `csv` packages and how they are lowered |
| [HTTP package](/docs/design-notes/http/) | the HTTP server's runtime model, limits, and HTTPS through a reverse proxy |
| [Image package](/docs/design-notes/image/) | the PPM-based image package |
| [Time package](/docs/design-notes/time/) | clocks and sleeping |
| [Naming](/docs/design-notes/naming/) | lower camelCase for source names, and how internal names differ |
| [Using cc or $CC](/docs/design-notes/compiler/) | why the compiler never hardcodes `gcc` or `clang` |
| [Open issues](/docs/design-notes/open-issues/) | known backend improvements |

The user-facing explanations of the same topics are in the [language guide](/docs/guide/) and the
[standard library](/docs/stdlib/) section.
