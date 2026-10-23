---
title: "Unit tests"
description: "A real package with .test.tx files, run with trux test."
icon: "fact_check"
weight: 60
---

The `scorebook` project keeps its tests in `scores.test.tx` next to the source. `trux test` runs every `test...`
function in the package and reports the file and line on failure.

{{< details summary="`examples/projects/scorebook/scores.test.tx`" >}}

{{< include "examples/projects/scorebook/scores.test.tx" "trux" >}}

{{< /details >}}

```shell-session
$ trux test examples/projects/scorebook/scores.test.tx
PASS examples/projects/scorebook/scores.test.tx testBestAndPassingCount
PASS examples/projects/scorebook/scores.test.tx testCurveCapsScores
PASS examples/projects/scorebook/scores.test.tx testLetterBoundaries
PASS examples/projects/scorebook/scores.test.tx testPrivateHelpers
PASS examples/projects/scorebook/scores.test.tx testTotalAndAverage
5 passed, 0 failed
```

The package under test is the `scorebook` example in [Projects with IO](/docs/examples/projects-io/#scorebook). For
the file naming rules, the assertion behavior, and the exit code, see [Testing](/docs/guide/testing/).
