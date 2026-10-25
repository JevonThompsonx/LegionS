# rog — top-level convenience Makefile.
# The Go module lives in ./gamepack, so most targets delegate there.
# This thin wrapper lets `make build/test/lint/clean` work from the repo root
# and is what the CI workflow exercises.

GO ?= go
MODULE_DIR := gamepack

.PHONY: build test lint clean

build:
	$(MAKE) -C $(MODULE_DIR) build

test:
	$(GO) test ./$(MODULE_DIR)/...

lint:
	$(GO) vet ./$(MODULE_DIR)/...
	@test -z "$$(gofmt -l $(MODULE_DIR))" || { \
		echo "gofmt: these files are not formatted:"; \
		gofmt -l $(MODULE_DIR); \
		exit 1; \
	}

clean:
	$(MAKE) -C $(MODULE_DIR) clean
	$(GO) clean -testcache
	rm -f coverage.out
