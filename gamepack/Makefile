GO ?= go

.PHONY: build deploy test coverage vet lint

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags "-s -w" -o gamepack ./cmd/gamepack

deploy: build
	scp gamepack rog:/tmp/gamepack.new
	ssh rog 'chmod 755 /tmp/gamepack.new && mv /tmp/gamepack.new /run/media/system/Games_Data/Tools/gamepack/gamepack'

test:
	$(GO) test ./internal/...

coverage:
	bash scripts/coverage-check.sh

vet:
	$(GO) vet ./...

clean:
	rm -f gamepack coverage.out
	$(GO) clean -testcache

lint: vet test coverage
