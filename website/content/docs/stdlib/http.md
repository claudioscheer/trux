---
title: "http"
description: "A declarative, high-performance HTTP server powered by Linux io_uring."
icon: "dns"
weight: 400
---

```trux
import "http"
```

**What it is.** A native HTTP/1.1 server built directly on Linux **`io_uring`**. It provides declarative routing, path parameter extraction, middleware chains, request context, custom response headers, and zero-allocation networking.

**Why.** Web services written in Trux achieve exceptional throughput and sub-millisecond latencies through an asynchronous, shared-nothing thread-per-core runtime. See [HTTP benchmarks](/docs/benchmarks/http/) for detailed performance metrics.

**When you don't need it.** For TLS/HTTPS, place a reverse proxy (such as Caddy or Nginx) in front. There is no HTTP client.

## A complete server

{{< include "examples/projects/http_server/main.tx" "trux" >}}

```shell-session
$ trux run examples/projects/http_server/main.tx
# in another terminal:
$ curl http://127.0.0.1:8080/
hello from trux
$ curl http://127.0.0.1:8080/health
ok
$ curl -X POST --data 'hello' http://127.0.0.1:8080/echo
hello
```

The server blocks forever; stop it with ++ctrl+c++.

## Routing & Server Lifecycle

| Function | Returns | Description |
| --- | --- | --- |
| `http.listen(port int)` | | Binds to `0.0.0.0` on `port`, spawns worker threads matching available cores, and starts the server (statement only). |
| `http.get(path string, handler)` | | Registers a `GET` route handler. |
| `http.post(path string, handler)` | | Registers a `POST` route handler. |
| `http.put(path string, handler)` | | Registers a `PUT` route handler. |
| `http.delete(path string, handler)` | | Registers a `DELETE` route handler. |
| `http.patch(path string, handler)` | | Registers a `PATCH` route handler. |
| `http.route(method string, path string, handler)` | | Registers a route handler for an arbitrary HTTP method. |
| `http.notFound(handler)` | | Registers a custom 404 fallback handler. |

### Dynamic Path Parameters

Routes can define named parameter segments with `:param`:

```trux
func getUser(request int) int {
  let id string = http.param(request, "id")
  http.json(request, 200, "{\"id\":\"" + id + "\"}\n")
  return 0
}

func main() int {
  http.get("/users/:id", getUser)
  http.listen(8080)
  return 0
}
```

## Middleware & Context

| Function | Returns | Description |
| --- | --- | --- |
| `http.use(handler)` | | Registers global middleware executed on all incoming requests. |
| `http.useAt(prefix string, handler)` | | Registers middleware executed only when the request path starts with `prefix`. |
| `http.set(request int, key string, val string)` | | Stores a key-value string pair in the request-scoped context dictionary. |
| `http.context(request int, key string)` | `string` | Retrieves a value previously stored via `http.set`. |

Middleware functions share the same signature as route handlers (`func(request int) int`). Returning `0` allows the pipeline to continue; returning non-zero halts execution:

```trux
func authMiddleware(request int) int {
  let token string = http.header(request, "Authorization")
  if token == "Bearer secret-token" {
    http.set(request, "userId", "user-42")
    return 0 // continue to next handler
  }
  http.setHeader(request, "WWW-Authenticate", "Bearer")
  http.json(request, 401, "{\"error\":\"unauthorized\"}\n")
  return 1 // abort pipeline
}
```

## Request & Response Inspection

| Function | Returns | Description |
| --- | --- | --- |
| `http.method(request int)` | `string` | Request method, such as `"GET"` or `"POST"`. |
| `http.path(request int)` | `string` | Request path, without the query string. |
| `http.query(request int)` | `string` | Raw query string, without the leading `?`. |
| `http.header(request int, name string)` | `string` | Value of a request header (matched case-insensitively). |
| `http.param(request int, name string)` | `string` | Value of a matched path parameter (`:name`). |
| `http.body(request int)` | `string` | Request body. |
| `http.setHeader(request int, name string, val string)` | | Attaches a custom response header. |
| `http.text(request int, status int, body string)` | | Sends a `text/plain` response. |
| `http.json(request int, status int, body string)` | | Sends an `application/json` response. |
| `http.status(request int, status int)` | | Sends an empty response with the specified status code. |
| `http.respond(request int, status int, contentType string, body string)` | | Sends a response with custom content type and body. |

## The Handler Signature

Every route handler and middleware function must have this shape:

```trux
func handle(request int) int
```

- `request` is an opaque handle representing the active connection.
- Handlers send a response using `http.text`, `http.json`, `http.status`, or `http.respond`.
- If a handler returns without responding, the runtime automatically sends `500 text/plain`.
- If a handler attempts to respond multiple times, only the first response is emitted.
- Each request receives a fresh, dedicated memory arena; memory allocations do not escape or leak across requests or threads.

## Limits

- Linux server environment required for `io_uring` networking. Windows/macOS stubs fail gracefully at runtime.
- In Docker containers, specify `--security-opt seccomp=unconfined --ulimit memlock=-1:-1` so the kernel can allocate buffer rings.
- Plain HTTP only; place a TLS reverse proxy in front for HTTPS.
- Request bodies are read using `Content-Length` and are capped at 1 MiB. Chunked request bodies are rejected.
- A handler may call `gpu.finish` after launching a kernel. The worker keeps accepting and reading other sockets while that kernel runs. The continuation sends the response. See [GPU programming](/docs/guide/gpu/#models).
