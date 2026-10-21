---
title: "http"
description: "A small HTTP server with routes and handlers."
icon: "dns"
weight: 400
---

```trux
import "http"
```

**What it is.** A plaintext HTTP/1.1 server. Your program supplies one **handler function**; the runtime accepts
connections, parses requests, and calls the handler from a pool of worker threads.

**Why.** A small web service is a realistic program to write in Trux, and the worker pool is the only concurrency the
language offers.

**When you don't need it.** For HTTPS, put a TLS proxy in front. There is no HTTP client.

## A complete server

{{< include "examples/projects/http_server/main.tx" "trux" >}}

```shell-session
$ trux run examples/projects/http_server/main.tx
# in another terminal:
$ curl http://127.0.0.1:8080/
hello from trux
$ curl -X POST --data 'hello' http://127.0.0.1:8080/echo
hello
```

The server blocks forever; stop it with ++ctrl+c++.

## Functions

| Function | Returns | Description |
| --- | --- | --- |
| `http.serve(host string, port int, workers int, handler)` | | Starts the server and never returns (statement only). |
| `http.method(request int)` | `string` | Request method, such as `"GET"` or `"POST"`. |
| `http.path(request int)` | `string` | Request path, without the query string. |
| `http.query(request int)` | `string` | Raw query string, without the leading `?`. |
| `http.header(request int, name string)` | `string` | Value of a request header; the name is matched case-insensitively. |
| `http.body(request int)` | `string` | Request body. |
| `http.respond(request int, status int, contentType string, body string)` | | Sends the response (statement only). |

## The handler

- The fourth argument to `http.serve` must be the **name** of a function (not a call), with exactly this signature:

    ```trux
    func handle(request int) int
    ```

- `request` is an opaque handle; pass it to the other `http` functions.
- Call `http.respond` once. If the handler returns without responding, the client gets `500 text/plain`. If it responds
  more than once, only the first response is sent.
- The handler's return value is not sent anywhere; return `0`.
- Each request gets its own fresh arena, so allocations in one request never leak into another.

## Limits

- Linux and macOS only (POSIX sockets and pthreads). It does not work on Windows.
- Plain HTTP only. Put a TLS proxy in front for HTTPS.
- `host` must be an IPv4 literal such as `"127.0.0.1"` or `"0.0.0.0"`.
- `port` must be `1`-`65535`, and `workers` `1`-`1024`.
- Request bodies are read using `Content-Length` and are capped at 1 MiB. Chunked request bodies are rejected.
- A program cannot use `http` together with GPU kernels yet:

    {{< include "website/outputs/errors/gpu_with_http.txt" "text" >}}

The `curl` session above was run by hand against `examples/projects/http_server`; it is not re-recorded by CI, because
the server never exits.

Programs that use `http` are compiled with `-pthread` and include the extra `trux_http_runtime.h` header. See
[HTTP package](/docs/design-notes/http/) for the runtime model.
