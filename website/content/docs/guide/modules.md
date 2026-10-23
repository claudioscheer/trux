---
title: "Modules and packages"
description: "Splitting a program across files with relative imports, pub func, and pub type."
icon: "folder_open"
weight: 900
---

## Source imports

**What it is.** `import "file.tx"` loads another Trux file. The compiler follows imports from the entry file and
compiles everything into one program.

**Why.** Larger programs stay readable when related types and functions live in their own files.

**How to use it.**

{{< tabs tabTotal="2" >}}
{{% tab tabName="main.tx" %}}

{{< include "examples/modules/basic/main.tx" "trux" >}}

{{% /tab %}}
{{% tab tabName="math.tx" %}}

{{< include "examples/modules/basic/math.tx" "trux" >}}

{{% /tab %}}
{{< /tabs >}}

{{< include "website/outputs/modules_basic.txt" "text" >}}

- The path is relative to the **importing file's directory** and ends in `.tx`: `"math.tx"`, `"math/add.tx"`,
  `"../shared/text.tx"`.
- Callers use the file's **package name** (`package math`), not its file name, as the qualifier for functions
  (`math.add(...)`) and struct types (`math.Pair`).
- Each file is loaded once, however many times it is imported. `.test.tx` files are skipped.

**Common mistakes.** A bare name that isn't a standard package:

{{< include "examples/errors/modules_bare_import.tx" "trux" >}}

{{< include "website/outputs/errors/modules_bare_import.txt" "text" >}}

Write `import "calc.tx"` for a source file. Import cycles are rejected with the full chain:

{{< include "website/outputs/errors/modules_cycle.txt" "text" >}}

Errors in an imported file are reported with that file's path, line, and column.

**When you don't need it.** A program that fits comfortably in one file. Imports are not a way to share code between
projects: there are no package artifacts or search paths.

**Related:** [Program structure: imports](/docs/guide/program-structure/#imports).

## `pub func` and `pub type` {#pub-func}

**What it is.** `pub func` makes a function callable from files that import its package, as `pkg.name(...)`.
`pub type Name struct` exports a struct type and all of its fields so importers can write `pkg.Name` and construct
`pkg.Name{...}`. Plain `func` and `type` declarations are private to their package.

**Why.** Each package decides what it exposes. Private helpers and internal structs can change without breaking importers.

**How to use it.**

| Declared as | Visible from |
| --- | --- |
| `pub func f` / `pub type S struct` | files that **directly** import its package, as `pkg.f(...)` / `pkg.S`; and, unqualified, from its own package |
| `func f` / `type S struct` | only its own package, unqualified |

Different packages may each have a private function or private struct with the same name (`examples/modules/private_names`).

**Common mistakes.** Calling a private function or referencing a private struct from another package:

{{< tabs tabTotal="2" >}}
{{% tab tabName="program.tx" %}}

{{< include "examples/errors/modules_private/program.tx" "trux" >}}

{{% /tab %}}
{{% tab tabName="text.tx" %}}

{{< include "examples/errors/modules_private/text.tx" "trux" >}}

{{% /tab %}}
{{< /tabs >}}

{{< include "website/outputs/errors/modules_private.txt" "text" >}}

A qualified call to a private function is rejected even inside its own package; call it unqualified there.

**When you don't need it.** Functions and structs used only inside their own package should stay private.

**Related:** [Functions](/docs/guide/functions/), [Structs](/docs/guide/types/#structs).

## Direct imports only

**What it is.** A file can call only the packages it imports itself. Importing `app.tx`, which imports `text.tx`, does
not give you `text`.

**Why.** Every file's dependencies are listed at its top, so moving or deleting an import never silently breaks a
different file.

**How to use it.** Import each package you call. `examples/modules/transitive` shows the working version.

**Common mistakes.** Calling a package that is only imported indirectly:

{{< include "examples/errors/modules_transitive/program.tx" "trux" >}}

{{< include "website/outputs/errors/modules_transitive.txt" "text" >}}

Add `import "text.tx"` to the file.

## Packages across several files

**What it is.** Importing a file also loads its **sibling files in the same directory that declare the same package**,
so one package can be split across files.

**Why.** A package can grow without its importers having to list every file.

**How to use it.**

```text
examples/modules/same_package_math/
├── main.tx          package main, imports "math/add.tx" and "math/mul.tx"
└── math/
    ├── add.tx       package math
    ├── double.tx    package math
    └── mul.tx       package math
```

{{< include "examples/modules/same_package_math/main.tx" "trux" >}}

{{< include "website/outputs/modules_same_package_math.txt" "text" >}}

Private functions and structs are shared by all files of the package, and every function or struct name in the package
must be unique. Several imports may declare the same package name; the qualifier then covers the public functions and
types of all of them.

**When you don't need it.** The entry file is special: it does **not** load its siblings, and only it may declare
`main`.

## More examples

| Directory | Shows |
| --- | --- |
| `examples/modules/basic` | one import, `pub type Pair struct`, and `pub func` |
| `examples/modules/transitive` | `main` → `app` → `text`, `pub type Status struct`, and the direct-import rule |
| `examples/modules/private_names` | two packages with same-named private structs and helpers, and a package split over two files |
| `examples/modules/same_package_math` | one package across several files sharing `Pair`, imported through two paths |
| `examples/modules/mut_parameters` | `mut` collection/map parameters and `*Queue` struct pointers across package boundaries |

## Limits

All files are merged into **one** generated C file. Directory packages, separate compilation, and reusable package
artifacts are not implemented. See [Modules design](/docs/design-notes/modules/).
