# Design notes

These documents explain *why* Trux works the way it does. They are written for contributors and curious readers, and
they predate this site, so some of them use a terser style than the guide.

| Note | Topic |
| --- | --- |
| [Language specification](../SPECS.md) | the compact, complete reference for the current language |
| [Arenas](../ARENAS.md) | the arena memory model, borrowed and owned returns, `mut`, and `clone` |
| [Modules design](../MODULES.md) | how imports and packages are loaded and merged, and the plan toward separate compilation |
| [GPU design](../GPU.md) | the CUDA model: device buffers, kernels, and launches |
| [IO packages](../IO.md) | why IO lives in the `io` and `csv` packages and how they are lowered |
| [HTTP package](../HTTP.md) | the HTTP server's runtime model and limits |
| [Image package](../IMAGE.md) | the PPM-based image package |
| [Time package](../TIME.md) | clocks and sleeping |
| [Naming](../NAMING.md) | lower camelCase for source names, and how internal names differ |
| [Using cc or $CC](../COMPILER.md) | why the compiler never hardcodes `gcc` or `clang` |
| [Open issues](../OPEN_ISSUES.md) | known backend improvements |

The user-facing explanations of the same topics are in the [language guide](../guide/index.md) and the
[standard library](../stdlib/index.md) section.
