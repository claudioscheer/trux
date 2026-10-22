# Multi-File Modular HTTP Server Example

This example demonstrates how to build a modular, multi-file HTTP service in Trux utilizing Linux `io_uring`:

- **Modular Packages**: Separated into `handlers/` and `middleware/` packages.
- **Middleware Pipeline**: Global middleware (`auth.serverHeader`) and path-scoped middleware (`http.useAt("/api", auth.authenticate)`).
- **Request Context**: Context propagation across middleware and handlers using `http.set()` and `http.context()`.
- **Dynamic Routing**: Path parameters extracted via `http.param(request, "id")`.
- **Custom Error Handling**: Fallback 404 handler via `http.notFound()`.
- **Zero-Allocation I/O**: High-performance request processing driven by `io_uring` kernel buffer rings and multishot operations.

## Running

On a Linux machine:

```bash
trux run examples/projects/http_multi_file/main.tx
```

## Testing Endpoints

```bash
# Health check
curl -i http://localhost:8080/health

# Unauthorized request to protected route
curl -i http://localhost:8080/api/users/42

# Authorized request to protected route
curl -i -H "Authorization: Bearer secret-token" http://localhost:8080/api/users/42

# PUT request with body
curl -i -X PUT -H "Authorization: Bearer secret-token" -d '{"name":"Alice"}' http://localhost:8080/api/users/99

# Custom 404 route
curl -i http://localhost:8080/unknown
```
