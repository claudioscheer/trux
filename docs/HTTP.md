# HTTP Package

Trux can serve plaintext HTTP through the imported `http` standard package.

```trux
import "http"

http.serve(host string, port int, workers int, handler)
http.method(request int) string
http.path(request int) string
http.query(request int) string
http.header(request int, name string) string
http.body(request int) string
http.respond(request int, status int, contentType string, body string)
```

`http.serve` is statement-only and blocks forever. The fourth argument must be a bare handler function name. A handler must have this shape:

```trux
func handle(request int) int {
    http.respond(request, 200, "text/plain", "ok")
    return 0
}
```

Example:

```trux
package main

import "http"

func handle(request int) int {
    let method string = http.method(request)
    let path string = http.path(request)

    if method == "GET" {
        if path == "/" {
            http.respond(request, 200, "text/plain", "hello from trux\n")
            return 0
        }
    }

    http.respond(request, 404, "text/plain", "not found\n")
    return 0
}

func main() int {
    http.serve("127.0.0.1", 8080, 32, handle)
    return 0
}
```

See [examples/projects/http_server](../examples/projects/http_server) for a runnable server with `GET /`, `GET /health`, and `POST /echo`.

## Runtime Model

The generated C includes `trux_http_runtime.h` only when an HTTP call is used. The runtime uses POSIX sockets and pthreads: one accept loop feeds a bounded connection queue, and worker threads parse requests and call the Trux handler.

Each handled request gets a fresh arena and `rt_context`, so request-local allocations are isolated between worker threads. The current language does not expose general light threads, channels, shared state primitives, or async IO.

## Limits

- POSIX/macOS/Linux only. Windows builds fail at runtime for `http`.
- Plain HTTP only. TLS should sit in front of the process for now.
- `host` must be an IPv4 literal such as `"127.0.0.1"` or `"0.0.0.0"`.
- `port` must be between `1` and `65535`.
- `workers` must be between `1` and `1024`.
- Request bodies use `Content-Length` and are capped at 1 MiB.
- Chunked request bodies are rejected.
- If a handler returns without calling `http.respond`, the runtime sends `500 text/plain`.
- If a handler calls `http.respond` more than once, only the first response is sent.

## Tradeoff

This design adds high-performance webserver support without first designing general goroutine-like language features. The cost is that concurrency is currently owned by the HTTP runtime, not exposed as a reusable language primitive. That is the right tradeoff for this stage because it keeps the language surface small while giving web programs a real worker pool.
