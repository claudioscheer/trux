---
title: "Testing"
description: "Unit tests in .test.tx files, assert, and trux test."
icon: "fact_check"
weight: 1000
---

## Test files and test functions

**What it is.** A file ending in `.test.tx` holds test functions: functions whose names start with `test`. `trux test`
finds, compiles, and runs them.

**Why.** Tests are plain Trux with `assert`, so there is no framework to learn and no extra dependency.

**How to use it.**

{{< include "examples/projects/scorebook/scores.test.tx" "trux" >}}

- A test function takes **no parameters**, has **no return type**, and has **no `return` statement**.
- A test passes if it finishes. It fails on a false `assert`, any runtime error, or a compile error.
- The file must not define `main`. Other functions in it are ordinary helpers and follow the normal rules.
- A test file in the same directory and package as the code it tests (here `package scores` next to `scores.tx`) can call
  that package's private functions unqualified.

**Common mistakes.** Writing a test like a normal function, with a return type:

{{< include "examples/errors/testing_rules/math.test.tx" "trux" >}}

{{< include "website/outputs/errors/testing_rules.txt" "text" >}}

**When you don't need it.** For a quick check inside a program, an [`assert`](/docs/guide/control-flow/#assert) in the
code is enough. The compiler's own Go tests are separate: `make test` or `go test ./...` (see
[Contributing](/docs/internals/contributing/)).

**Related:** [`assert`](/docs/guide/control-flow/#assert), [Modules](/docs/guide/modules/).

## Running tests

**What it is.** `trux test` runs every test function and reports each one as `PASS` or `FAIL`.

**Why.** Each test runs as its own small program: `trux` gives the test function an `int` return type, adds a `main`
that calls it, and compiles and runs it like `trux run`. A crash in one test cannot affect another.

**How to use it.**

```bash
trux test                                              # every .test.tx under the current directory
trux test examples/projects/scorebook/scores.test.tx   # one file
trux test --fail-fast                                  # stop at the first failure
```

Directory search skips `.git`, `node_modules`, `bin`, and `tmp`. Files run in sorted path order, and tests within a file
in alphabetical order. `PASS` lines are green and failures red.

```shell-session
$ trux test examples/projects/scorebook/scores.test.tx
PASS examples/projects/scorebook/scores.test.tx testBestAndPassingCount
PASS examples/projects/scorebook/scores.test.tx testCurveCapsScores
PASS examples/projects/scorebook/scores.test.tx testLetterBoundaries
PASS examples/projects/scorebook/scores.test.tx testPrivateHelpers
PASS examples/projects/scorebook/scores.test.tx testTotalAndAverage
5 passed, 0 failed
```

**Common mistakes.** When a test fails, the summary is followed by each failure with its trace and error, and the
command exits with status `1`:

{{< include "examples/errors/testing_failure/math.test.tx" "trux" >}}

{{< include "website/outputs/errors/testing_failure.txt" "text" >}}

The first failing `assert` ends that test, so later asserts in the same function do not run.

**Related:** [CLI: `trux test`](/docs/cli/#trux-test).
