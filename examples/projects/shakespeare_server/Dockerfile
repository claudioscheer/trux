FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    liburing2 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY server /app/server
COPY model.txm /app/model.txm

EXPOSE 8080

CMD ["/app/server"]
