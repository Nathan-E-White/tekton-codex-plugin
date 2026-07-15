GO_CACHE ?= /tmp/tekton-go-cache
GO_MOD_CACHE ?= /tmp/tekton-go-mod-cache
GO_ENV = GOTMPDIR=/tmp GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) GOFLAGS=-p=1

.PHONY: build test test-race vet validate package kind-smoke

build:
	mkdir -p bin
	$(GO_ENV) go build -trimpath -o bin/tekton-mcp ./cmd/tekton-mcp

test:
	$(GO_ENV) go test -p 1 ./...

test-race:
	$(GO_ENV) go test -race -p 1 ./internal/...

vet:
	$(GO_ENV) go vet ./...

validate: test vet
	python3 scripts/validate-package.py

package:
	scripts/package-release.sh 0.1.0

kind-smoke:
	scripts/kind-smoke.sh
