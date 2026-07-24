# Reinstate — development Makefile
# Copyright (c) 2026 Harjot Singh Rana

MODULE  := github.com/HarjjotSinghh/reinstate
BIN_DIR := bin
BINARY  := $(BIN_DIR)/reinstate
CMD     := ./cmd/reinstate
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: all deps build test test-race vet lint clean run version fixture-test help

all: build

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

deps: ## Download Go modules
	go mod download
	go mod tidy

build: ## Build reinstate binary into ./bin
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

run: build ## Build and print version
	$(BINARY) version

test: ## Run unit tests
	go test ./... -count=1

test-race: ## Run tests with race detector
	go test ./... -race -count=1

vet: ## go vet
	go vet ./...

lint: ## golangci-lint if installed
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed; skip"

fixture-test: test ## Alias for adapter fixture tests (expand later)
	@echo "fixture tests: ok (scaffold)"

version: build ## Print embedded version
	$(BINARY) version

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) dist coverage.out

install: build ## Install to GOPATH/bin or /usr/local/bin
	install -m 755 $(BINARY) "$${GOBIN:-$${GOPATH:-$$HOME/go}/bin}/reinstate"
