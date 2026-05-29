# Using cc or $CC

The trux compiler must not hardcode `clang` or `gcc`.

## Decision

Default to the POSIX `cc` command.  
Respect the `CC` environment variable if set.

Hardcode a specific compiler only for narrow, documented reasons (e.g. a sanitizer that requires it).

## Origin of the Question

**[SPECS.md](SPECS.md)** (original pipeline diagram):
```
  → C code
  → clang/gcc
  → executable
```

**cmd/trux/run.go:19**:
```go
Long: "Compile the given .tx file to C, invoke the C compiler, execute the resulting binary, and print its output.",
```

## Tradeoffs

| Approach                  | Benefit                                         | Cost                                              |
|---------------------------|-------------------------------------------------|---------------------------------------------------|
| Hardcode `clang`          | Better diagnostics on macOS and many Linux boxes| Breaks on systems that only have gcc              |
| Hardcode `gcc`            | Traditional default on classic Linux            | Worse diagnostics; on macOS `gcc` is usually clang |
| Use `cc` + respect `$CC`  | Portable; follows long-standing Unix convention | Slightly less "explicit"                          |

For a learning compiler that emits straightforward C (`int64_t`, small structs), portability wins. This is the same choice made by cgo, nim, and other tools that emit C. On the current macOS machine the practical difference is small, but the code must not assume the author's environment is universal.

## When to Revisit

Only if the project ships its own toolchain or gains a hard requirement for compiler-specific features.
