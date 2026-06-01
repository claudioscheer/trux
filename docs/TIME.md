# Time Package

Trux exposes basic time operations through `import "time"`.

```trux
import "time"

let wall int = time.nowUnixMillis()
let start int = time.monotonicNanos()
time.sleepMillis(10)
let elapsed int = time.monotonicNanos() - start
```

## Functions

```trux
time.nowUnixMillis() int
time.monotonicNanos() int
time.sleepMillis(ms int)
```

`nowUnixMillis` returns wall-clock Unix time in milliseconds. Use it for timestamps.

`monotonicNanos` returns a monotonic timestamp in nanoseconds. Use it for elapsed-time measurement.

`sleepMillis` is statement-only and sleeps for a non-negative number of milliseconds.
