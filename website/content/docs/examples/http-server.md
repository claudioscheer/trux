---
title: "HTTP server"
description: "A plaintext HTTP server with three routes, built on the http package."
icon: "dns"
weight: 70
---

`examples/projects/http_server/main.tx` is a complete server in 27 lines: three routes on `127.0.0.1:8080`, plaintext
HTTP, Linux and macOS. It runs until you stop it.

{{< details summary="`examples/projects/http_server/main.tx`" >}}

{{< include "examples/projects/http_server/main.tx" "trux" >}}

{{< /details >}}

| Route | Response |
| --- | --- |
| `GET /` | `hello from trux` |
| `GET /health` | `ok` |
| `POST /echo` | the request body, unchanged |

Run it from the repository root, then use `curl`:

```bash
trux run examples/projects/http_server/main.tx
curl http://127.0.0.1:8080/health
curl -X POST --data 'hello' http://127.0.0.1:8080/echo
```

The `http` package does not support HTTPS; put a reverse proxy in front for TLS. See the [`http` package](/docs/stdlib/http/#https).
