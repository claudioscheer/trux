# HTTP Server

This project serves plaintext HTTP on localhost with the `http` standard package.

Run from the repository root:

```bash
trux run examples/projects/http_server/main.tx
```

Then call it from another terminal:

```bash
curl http://127.0.0.1:8080/
curl http://127.0.0.1:8080/health
curl -X POST --data 'hello' http://127.0.0.1:8080/echo
```

The server uses a pthread-backed worker pool through the runtime. It blocks forever until you stop the process.
