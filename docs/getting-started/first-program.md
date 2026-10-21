# Your first program

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

```console
$ trux run greet.tx
0: hello, trux
1: hello, trux
2: hello, trux
```

`trux run` compiles the program to C in a temporary directory, builds it with your C compiler, runs it, and removes the
temporary files. Standard input is passed through to the program, so interactive programs work too.

!!! note "Exit codes"

    If `main` returns something other than `0`, `trux run` reports it, for example
    `execute /tmp/trux-run-.../main: exit status 3`, and exits with status `1` itself.

## Build an executable

```console
$ trux build -o greet greet.tx
$ ./greet
0: hello, trux
1: hello, trux
2: hello, trux
```

`-o` is required.

## Look at the generated C

```console
$ trux emit-c greet.tx
out/greet/greet.c
out/greet/trux_runtime.h
```

`emit-c` writes the generated program plus the runtime header it includes. Open `out/greet/greet.c` to see what your
program became: each Trux function turns into a C function, `int` arithmetic goes through overflow-checked helpers such
as `rt_add_i64`, and `#line` directives point back to your `.tx` file. [Compiling to C and CUDA](../internals/compilation.md)
walks through it.

## When something is wrong

Compile errors show the file, line, and column, with the offending line marked:

```console
$ trux run bad.tx
bad.tx:4:15: cannot assign float to int
2 |
3 | func main() int {
4 |   let x int = 1.5
  |               ^
5 |   return 0
6 | }
```

Problems that can only be detected while the program runs (an index out of range, integer overflow, a failed `assert`)
stop it with a `trux runtime error` message on stderr:

```text
trux runtime error: index 5 out of bounds for length 3
```

The [FAQ](../faq.md) lists the most common errors and what they mean.

## Format your code

```bash
trux fmt greet.tx     # format one file in place
trux fmt -r           # format every .tx file under the current directory
```

## Where next

- The [language guide](../guide/index.md) explains each feature with small examples.
- The [examples gallery](../examples.md) has complete programs you can run from the repository.
