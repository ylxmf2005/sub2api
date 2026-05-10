.PHONY: build ccgo-binaries generate test test-unit test-integration test-e2e

VERSION ?= $(shell tr -d '\r\n' < ./cmd/server/VERSION)
LDFLAGS ?= -s -w -X main.Version=$(VERSION)
CCGO_BIN_DIR ?= bin

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o bin/server ./cmd/server

ccgo-binaries:
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(CCGO_BIN_DIR)/ccgo ./cmd/ccgo
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(CCGO_BIN_DIR)/ccgo-wrapper ./cmd/ccgo-wrapper

generate:
	go generate ./ent
	go generate ./cmd/server

test:
	go test ./...
	golangci-lint run ./...

test-unit:
	go test -tags=unit ./...

test-integration:
	go test -tags=integration ./...

test-e2e:
	./scripts/e2e-test.sh

test-e2e-local:
	go test -tags=e2e -v -timeout=300s ./internal/integration/...
