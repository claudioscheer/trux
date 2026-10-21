---
title: "Your first program"
description: "Write, run, build, and test a small Trux program step by step."
icon: "play_arrow"
weight: 200
---

## Write it

Create a file called `greet.tx`:

```trux
package main

func greet(name string) string {
  return "hello, " + name
}

func main() int {
  let who string = "trux"
  let times int = 3

  for let i int = 0; i < times; i = i + 1 {
    print(i, ": ", greet(who))
  }

  return 0
}
```

A few things to notice:

- Every file starts with a `package` clause. A runnable program's entry file uses `package main`.
- The program starts at `func main() int`. Its return value becomes the process exit code.
- Variables are introduced with `let name type = value`. The type is always written out.
- `print` takes one or more `int`, `float`, `string`, or `bool` values, prints them side by side, and ends the line.

## Run it

```shell-session
$ trux run greet.tx
0: hello, trux
1: hello, trux
2: hello, trux
```

`trux run` compiles the program to C in a temporary directory, builds it with your C compiler, runs it, and removes the
temporary files. Standard input is passed through to the program, so interactive programs work too.

{{% alert context="info" %}}

**Exit codes**

If `main` returns something other than `0`, `trux run` reports it, for example
`execute /tmp/trux-run-.../main: exit status 3`, and exits with status `1` itself.

{{% /alert %}}

## Build an executable

```shell-session
$ trux build -o greet greet.tx
$ ./greet
0: hello, trux
1: hello, trux
2: hello, trux
```

`-o` is required.

## Look at the generated C

```shell-session
$ trux emit-c greet.tx
out/greet/greet.c
out/greet/trux_runtime.h
```

`emit-c` writes the generated program plus the runtime header it includes. Open `out/greet/greet.c` to see what your
program became: each Trux function turns into a C function, `int` arithmetic goes through overflow-checked helpers such
as `rt_add_i64`, and `#line` directives point back to your `.tx` file. [Compiling to C and CUDA](/docs/internals/compilation/)
walks through it.

## When something is wrong

Compile errors show the file, line, and column, with the offending line marked. Nothing is built:

{{< include "examples/errors/types_int_to_float.tx" "trux" >}}

```shell-session
$ trux run examples/errors/types_int_to_float.tx
```

{{< include "website/outputs/errors/types_int_to_float.txt" "text" >}}

(`trux` prints the file's full path; it is shortened here.) Problems that can only be found while the program runs, such
as an index out of range, integer overflow, or a failed `assert`, stop it with a `trux runtime error` on stderr:

{{< include "website/outputs/errors/collections_index_out_of_bounds.txt" "text" >}}

[Errors and failures](/docs/guide/errors/) explains both kinds, and each guide page shows the errors for its features.

## Format your code

```bash
trux fmt greet.tx     # format one file in place
trux fmt -r           # format every .tx file under the current directory
```

## Where next

- The [language guide](/docs/guide/) explains each feature: what it is, how to use it, and how it fails.
- The [examples gallery](/docs/examples/) has complete programs you can run from the repository.
