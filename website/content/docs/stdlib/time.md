---
title: "time"
description: "Clocks and sleeping."
icon: "schedule"
weight: 300
---

```trux
import "time"
```

**What it is.** A wall clock, a monotonic clock for measuring durations, and sleep.

**Why.** Timing code (the examples compare CPU and GPU runs) needs a clock that never jumps; logs and timestamps need the
real date.

**When you don't need it.** To measure elapsed time, never subtract `nowUnixMillis` readings, because the wall clock can
be adjusted; use `monotonicNanos`. To time GPU work, call [`gpu.sync`](/docs/guide/gpu/#gpu-sync) before reading the
clock.

| Function | Returns | Description |
| --- | --- | --- |
| `time.nowUnixMillis()` | `int` | Wall-clock time as milliseconds since the Unix epoch. Use it for timestamps. |
| `time.monotonicNanos()` | `int` | A monotonic clock in nanoseconds. Use differences between two readings to measure elapsed time. |
| `time.sleepMillis(ms int)` | | Sleeps for `ms` milliseconds (statement only). `ms` must not be negative. |

```trux
package main

import "time"

func main() int {
  let start int = time.monotonicNanos()
  time.sleepMillis(10)
  let elapsed int = time.monotonicNanos() - start

  print("slept at least 10 ms: ", elapsed >= 10000000)
  return 0
}
```

## Errors

**Common mistakes.** A negative sleep:

{{< include "examples/errors/time_negative_sleep.tx" "trux" >}}

{{< include "website/outputs/errors/time_negative_sleep.txt" "text" >}}

## Related

`examples/performance_integer.tx` uses `time.monotonicNanos` to time a CPU-bound loop. See [Time package](/docs/design-notes/time/).
