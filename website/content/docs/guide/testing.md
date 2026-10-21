---
title: "Testing"
description: "Unit tests in .test.tx files, assert, and trux test."
icon: "fact_check"
weight: 1000
---

Trux has built-in unit tests. Tests are ordinary Trux functions in files ending with `.test.tx`, and `trux test` finds,
compiles, and runs them.

## Writing a test file

{{< include "examples/projects/scorebook/scores.test.tx" "trux" >}}

The rules:

- The file name ends in `.test.tx`.
- A test function's name starts with `test`.
- Test functions take **no parameters**, **omit the return type**, and contain **no `return` statements**.
- A test passes if it finishes normally. It fails if an `assert` fails, a runtime error happens (for example an index out
  of bounds), or the file doesn't compile.
- A test file must not define `main`.
- Other functions in the file that don't start with `test` are ordinary helpers (and must follow the normal rules,
  including a return type).

### Testing a package's private functions

Notice that `scores.test.tx` above declares `package scores` and imports `scores.tx`, which lives next to it. Because the
test file is in the same directory and the same package, it can call the package's functions unqualified, including
private helpers such as `capScore` and `isPassing`.

## Running tests

```bash
trux test                                              # every .test.tx under the current directory
trux test examples/projects/scorebook/scores.test.tx   # one file
trux test --fail-fast                                  # stop at the first failure
```

When searching a directory, `trux test` skips `.git`, `node_modules`, `bin`, and `tmp`. Test files are run in sorted path
order, and the tests in each file in alphabetical order.

```shell-session
$ trux test examples/projects/scorebook/scores.test.tx
PASS examples/projects/scorebook/scores.test.tx testBestAndPassingCount
PASS examples/projects/scorebook/scores.test.tx testCurveCapsScores
PASS examples/projects/scorebook/scores.test.tx testLetterBoundaries
PASS examples/projects/scorebook/scores.test.tx testPrivateHelpers
PASS examples/projects/scorebook/scores.test.tx testTotalAndAverage
5 passed, 0 failed
```

Passing lines are printed in green and failures in red. After the summary, each failure is listed with its error and any
output the test printed. The command exits with a non-zero status if any test failed.

## How it works

Each test runs as its own small program: `trux` takes the test file, gives the selected test function an `int` return
type, and adds a generated `main` that calls it. That program is compiled and run exactly like `trux run` would, so a
crash in one test can't affect another.

## Testing the compiler itself

The Go test suite for the compiler is separate: run it with `make test` or `go test ./...`. See
[Contributing](/docs/internals/contributing/).
