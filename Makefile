# Reinstate — development Makefile
# Copyright (c) 2026 Harjot Singh Rana

MODULE  := github.com/HarjjotSinghh/reinstate
BIN_DIR := bin
BINARY  := $(BIN_DIR)/reinstate
ALIAS   := $(BIN_DIR)/rein
CMD     := ./cmd/reinstate
GO      ?= go
GOTOOLCHAIN ?= go1.25.12
GOENV   := GOTOOLCHAIN=$(GOTOOLCHAIN)
FAST_PACKAGES := $(shell $(GOENV) $(GO) list -f '{{if and (ne .ImportPath "$(MODULE)/internal/doctest") (ne .ImportPath "$(MODULE)/internal/crypto")}}{{.ImportPath}}{{end}}' ./...)
GOLANGCI_LINT_VERSION := v2.11.4
GOVULNCHECK_VERSION   := v1.6.0
VERSION ?= $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo 0.0.0-dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: all deps build quick test test-race vet lint fmt fmt-check docs-check fixture-scan vuln verify clean run version help snapshot

all: build

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

deps: ## Download Go modules
	$(GOENV) $(GO) mod download
	$(GOENV) $(GO) mod tidy

build: ## Build reinstate binary into ./bin (+ rein symlink)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)
	@ln -sfn reinstate $(ALIAS)

run: build ## Build and print version
	$(BINARY) version

quick: fmt-check vet ## Fast development gate; release work must use verify
	$(GOENV) $(GO) test $(FAST_PACKAGES)

test: ## Run unit tests
	$(GOENV) $(GO) test ./... -count=1

test-race: ## Run tests with race detector
	$(GOENV) $(GO) test $(FAST_PACKAGES) -race -count=1 -timeout=20m

vet: ## go vet
	$(GOENV) $(GO) vet ./...

fmt: ## gofmt write
	gofmt -w .

fmt-check: ## Fail if gofmt needed
	@test -z "$$(gofmt -l . | tee /dev/stderr)"

lint: ## Run the pinned golangci-lint release
	$(GOENV) $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --timeout=5m ./...

docs-check: ## Documentation consistency tests
	./scripts/check-docs.sh

fixture-scan: ## Scan fixtures for secrets
	$(GOENV) $(GO) test ./internal/fixture -count=1

vuln: ## Run the pinned govulncheck release
	$(GOENV) $(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

verify: fmt-check vet lint test test-race vuln ## Full local merge gate; test includes docs and fixture scan
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
