---
title: "Modules and packages"
description: "Splitting a program across files with relative imports and pub func."
icon: "folder_open"
weight: 900
---

A Trux program can be split across several `.tx` files. The compiler loads the entry file, follows its imports, and
compiles everything into one program.

## A two-file program

{{< tabs tabTotal="2" >}}
{{% tab tabName="main.tx" %}}

{{< include "examples/modules/basic/main.tx" "trux" >}}

{{% /tab %}}
{{% tab tabName="math.tx" %}}

{{< include "examples/modules/basic/math.tx" "trux" >}}

{{% /tab %}}
{{< /tabs >}}

```shell-session
$ trux run examples/modules/basic/main.tx
7
```

Three things make this work:

1. `import "math.tx"` loads the file `math.tx` from the **same directory as the importing file**.
2. `math.tx` declares `package math`. The package name, not the file name, is what callers use as a qualifier.
3. `add` is declared `pub func`, so importers can call it as `math.add(...)`.

## Import rules

- Source imports are **relative paths** to `.tx` files, written as string literals: `import "math.tx"`,
  `import "math/add.tx"`, `import "../shared/text.tx"`. Absolute paths, directories, and paths without `.tx` are errors.
- Bare names are reserved for the standard packages: `import "io"`, `"csv"`, `"image"`, `"time"`, `"http"`, `"gpu"`.
  Any other bare name is an error.
- Imports go after `package` and before the first function, one per line. Grouped `import ( ... )` blocks are not
  supported.
- Each file is loaded once, even if it is reached through several import paths.
- Import cycles are reported with the full chain of files.
- `.test.tx` files are skipped when loading a normal program.

## Visibility: `pub func`

| Declared as | Callable from |
| --- | --- |
| `pub func f` | files that **directly** import its package, as `pkg.f(...)`; and, unqualified, from its own package |
| `func f` | only its own package (same directory, same package name), unqualified |

- A qualified call to a private function is rejected, even from inside the same package.
- You can only call packages you import directly. If `main.tx` imports `app.tx` and `app.tx` imports `text.tx`,
  `main.tx` cannot call `text.prefix()` unless it imports `text.tx` too (see `examples/modules/transitive`).
- Different packages may each have a private function with the same name; the compiler keeps them apart.

## Packages spread over several files

Importing a file also loads its **sibling files in the same directory that declare the same package**. That lets you
split one package across files:

```text
examples/modules/same_package_math/
├── main.tx          package main, imports "math/add.tx" and "math/mul.tx"
└── math/
    ├── add.tx       package math
    ├── double.tx    package math
    └── mul.tx       package math
```

{{< include "examples/modules/same_package_math/main.tx" "trux" >}}

Inside the `math` package, private functions are shared by all three files. Within one package group, every function
name must be unique.

Several direct imports may declare the same package name (here `math/add.tx` and `math/mul.tx`). The qualifier `math`
then covers the public functions of all of them, as long as those names don't clash.

{{% alert context="info" %}}

**The entry file is special**

The file you pass to `trux run` does **not** automatically load its siblings; only imported files do. And only the
entry file may declare `main`.

{{% /alert %}}

## Error messages point at the right file

Parse and type errors in an imported file are reported with that file's path, line, and column, so you can jump straight
to the problem.

## More examples

| Directory | Shows |
| --- | --- |
| `examples/modules/basic` | one import, one public function |
| `examples/modules/transitive` | `main` → `app` → `text`, and the direct-import rule |
| `examples/modules/private_names` | two packages with same-named private helpers, and a package split over two files |
| `examples/modules/same_package_math` | one package across several files, imported through two paths |
| `examples/modules/mut_parameters` | `mut` parameters across package boundaries |

## Limits

All loaded files are merged and compiled into **one** generated C file. Recursive directory packages, separate
compilation, and reusable package artifacts are not implemented. The design and the planned evolution are described in
[Modules design](/docs/design-notes/modules/).
