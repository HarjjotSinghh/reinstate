# Reinstate — development Makefile
# Copyright (c) 2026 Harjot Singh Rana

MODULE  := github.com/HarjjotSinghh/reinstate
BIN_DIR := bin
BINARY  := $(BIN_DIR)/reinstate
ALIAS   := $(BIN_DIR)/rein
CMD     := ./cmd/reinstate
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: all deps build test test-race vet lint fmt fmt-check docs-check fixture-scan vuln verify clean run version help snapshot

all: build

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

deps: ## Download Go modules
	go mod download
	go mod tidy

build: ## Build reinstate binary into ./bin (+ rein symlink)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)
	@ln -sfn reinstate $(ALIAS)

run: build ## Build and print version
	$(BINARY) version

test: ## Run unit tests
	go test ./... -count=1

test-race: ## Run tests with race detector
	go test ./... -race -count=1

vet: ## go vet
	go vet ./...

fmt: ## gofmt write
	gofmt -w .

fmt-check: ## Fail if gofmt needed
	@test -z "$$(gofmt -l . | tee /dev/stderr)"

lint: ## golangci-lint (required when installed)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint required"; exit 1; }
	golangci-lint run --timeout=5m ./...

docs-check: ## Documentation consistency tests
	go test ./internal/doctest -count=1

fixture-scan: ## Scan fixtures for secrets
	go test ./internal/fixture -count=1

vuln: ## govulncheck when installed
	@if command -v govulncheck >/dev/null 2>&1; then govulncheck ./...; else echo "govulncheck not installed; skip"; fi

verify: fmt-check vet test docs-check fixture-scan ## Full local gate (lint/vuln soft if tools missing)
	@if command -v golangci-lint >/dev/null 2>&1; then $(MAKE) lint; else echo "lint: golangci-lint not installed (CI enforces)"; fi
	@$(MAKE) vuln
	@$(MAKE) build
	@echo "verify ok"

snapshot: ## goreleaser snapshot (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser required for snapshot"; exit 1; }
	goreleaser release --snapshot --clean

version: build ## Print embedded version
	$(BINARY) version

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) dist coverage.out

install: build ## Install to GOPATH/bin
	install -m 755 $(BINARY) "$${GOBIN:-$${GOPATH:-$$HOME/go}/bin}/reinstate"
