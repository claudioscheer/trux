---
title: "time"
description: "Clocks and sleeping."
icon: "schedule"
weight: 300
---

```trux
import "time"
```

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

`examples/performance_integer.tx` uses `time.monotonicNanos` to time a CPU-bound loop. See [Time package](/docs/design-notes/time/).
