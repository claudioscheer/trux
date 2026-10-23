# trux Makefile
# All compiler logic lives in the packages you write by hand.
# These targets are just convenience wrappers.

GO := go
BINARY := trux
CMD := ./cmd/trux
RUN_BIN := ./bin/$(BINARY)

.PHONY: build run test bench bench-digits fmt emit-c lsp clean

build:
	$(GO) build -o $(RUN_BIN) $(CMD)

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

# Usage: make test           (verbose, prints every test)
#        make test VERBOSE=0 (quiet, summary only)
VERBOSE ?= 1

test:
	$(GO) test $(if $(filter 1,$(VERBOSE)),-v )$(GO_TEST_FLAGS) ./...

# Integer workload, GPU matrix multiply, Jacobi stencil, and a four-layer conv net.
bench: build
	@echo ===== examples/performance_integer.tx =====
	$(RUN_BIN) run examples/performance_integer.tx
	@echo ===== examples/gpu_matrix_multiply.tx =====
	$(RUN_BIN) run examples/gpu_matrix_multiply.tx
	@echo ===== examples/performance_stencil.tx =====
	$(RUN_BIN) run examples/performance_stencil.tx
	@echo ===== examples/performance_cnn.tx =====
	$(RUN_BIN) run examples/performance_cnn.tx

# Handwritten-digits HTTP+GPU. Linux, liburing, and CUDA. Not part of `bench`.
bench-digits:
	$(GO) run ./benchmarks/digits/

fmt: build
	./bin/$(BINARY) fmt -r

lsp:
	$(GO) build -o bin/trux-lsp ./tooling/lsp

clean:
	rm -rf bin/ tmp/ *.tx.c *.tx
	go clean -cache
