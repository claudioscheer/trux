# trux Makefile
# All compiler logic lives in the packages you write by hand.
# These targets are just convenience wrappers.

GO := go
BINARY := trux
CMD := ./cmd/trux

.PHONY: build run test emit-c lsp clean

build:
	$(GO) build -o bin/$(BINARY) $(CMD)

# Usage: make run FILE=examples/hello.tx
run: build
	@if [ -z "$(FILE)" ]; then echo "Usage: make run FILE=path/to/file.tx"; exit 1; fi
	./bin/$(BINARY) run $(if $(DEBUG),--debug )$(RUN_FLAGS) $(FILE)

# Usage: make build-bin FILE=examples/hello.tx OUT=bin/hello
build-bin: build
	@if [ -z "$(FILE)" ]; then echo "Usage: make build-bin FILE=path/to/file.tx"; exit 1; fi
	@if [ -z "$(OUT)" ]; then echo "Usage: make build-bin FILE=path/to/file.tx OUT=path/to/output"; exit 1; fi
	./bin/$(BINARY) build -o $(OUT) $(FILE)

# Usage: make emit-c FILE=examples/hello.tx
emit-c: build
	@if [ -z "$(FILE)" ]; then echo "Usage: make emit-c FILE=path/to/file.tx"; exit 1; fi
	./bin/$(BINARY) emit-c $(FILE)

test:
	$(GO) test ./...

lsp:
	$(GO) build -o bin/trux-lsp ./tooling/lsp

clean:
	rm -rf bin/ tmp/ *.tx.c *.tx
	go clean -cache
