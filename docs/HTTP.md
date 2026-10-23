# HTTP Package

Trux provides a native HTTP/1.1 server built on Linux `io_uring` through the imported `http` standard package.

```trux
import "http"

http.listen(port int)
http.get(path string, handler)
http.post(path string, handler)
http.put(path string, handler)
http.delete(path string, handler)
http.patch(path string, handler)
http.route(method string, path string, handler)
http.notFound(handler)

http.use(handler)
http.useAt(prefix string, handler)

http.set(request int, key string, val string)
http.context(request int, key string) string

http.param(request int, name string) string
http.method(request int) string
http.path(request int) string
http.query(request int) string
http.header(request int, name string) string
http.body(request int) string

http.setHeader(request int, name string, val string)
http.text(request int, status int, body string)
http.json(request int, status int, body string)
http.status(request int, status int)
http.respond(request int, status int, contentType string, body string)
```

`http.listen` is statement-only and blocks forever. Route handlers and middleware must have the signature:

```trux
func handle(request int) int {
    http.text(request, 200, "ok\n")
    return 0
}
```

Example:

```trux
package main

import "http"

func index(request int) int {
    http.text(request, 200, "hello from trux\n")
    return 0
}

func health(request int) int {
    http.text(request, 200, "ok\n")
    return 0
}

func echo(request int) int {
    let body string = http.body(request)
    http.text(request, 200, body)
    return 0
}

func main() int {
    http.get("/", index)
    http.get("/health", health)
    http.post("/echo", echo)
    http.listen(8080)
    return 0
}
```

See [examples/projects/http_server](https://github.com/claudioscheer/trux/tree/main/examples/projects/http_server) and [examples/projects/http_multi_file](https://github.com/claudioscheer/trux/tree/main/examples/projects/http_multi_file) for runnable servers.

## Runtime Model

The generated C includes `trux_http_runtime.h` when HTTP calls are used. On Linux, the runtime employs a shared-nothing thread-per-core architecture:

1. **`SO_REUSEPORT`**: Each worker thread binds its own socket; incoming connections are load-balanced across threads in kernel space.
2. **`io_uring`**: Each thread runs an independent submission and completion queue ring.
3. **Kernel Buffer Rings (`io_uring_register_buf_ring`)**: The kernel selects receive buffers on packet arrival (`IOSQE_BUFFER_SELECT`), eliminating user-space memory allocations and read system calls.
4. **Multishot Operations**: Single SQEs handle continuous `accept` and `recv` operations across connections.
5. **Memory Isolation**: Each request receives its own arena and `rt_context`, isolating allocations between requests and worker threads.
6. **GPU requests**: A handler may call `gpu.finish` instead of responding. The arena stays alive until the continuation copies the response into the connection buffer. The worker keeps polling the ring while that kernel runs. See [GPU.md](GPU.md) and [TXM.md](TXM.md).

## Limits

- Linux server environment required for `io_uring` networking. Windows/macOS stubs fail gracefully at runtime.
- In Docker containers, require `--security-opt seccomp=unconfined --ulimit memlock=-1:-1` for kernel buffer ring allocation.
- Plain HTTP only. TLS should sit in front of the process (e.g. reverse proxy).
- `port` must be between `1` and `65535`.
- Request bodies use `Content-Length` and are capped at 1 MiB. Chunked request bodies are rejected.
- If a handler returns without calling a respond helper, the runtime sends `500 text/plain`.
- If a handler calls a respond helper more than once, only the first response is sent.
