# Reinstate — development Makefile
# Copyright (c) 2026 Harjot Singh Rana

MODULE  := github.com/HarjjotSinghh/reinstate
BIN_DIR := bin
BINARY  := $(BIN_DIR)/reinstate
ALIAS   := $(BIN_DIR)/rein
CMD     := ./cmd/reinstate
GOTOOLCHAIN ?= go1.25.13

# Resolving the Go toolchain
#
# make runs recipes with /bin/sh, which does not read an interactive shell's
# configuration. A toolchain installed by a version manager therefore reaches
# the prompt but not necessarily a recipe, and the failure that produces —
# "/bin/sh: go: command not found" followed by "Error 127" — says nothing about
# what to do.
#
# Two constraints shape how this is written. Nothing here may fail at parse
# time, because `make -n` must work anywhere, including on a machine where the
# toolchain really is absent. And the probing must not assume a POSIX shell:
# `$(shell command -v …)` on Windows makes make spawn a shell it does not have,
# which prints CreateProcess errors and yields nothing useful. So the search is
# pure make where it can be, shell-assisted only where a shell exists, and the
# diagnosis lives in a recipe rather than in a conditional.
GO ?= go

ifeq ($(OS),Windows_NT)
GO_HOME := $(or $(USERPROFILE),$(HOME))
else
# `echo ~` resolves from the password database, so this still works when the
# invoking environment did not export HOME — in which case every path below
# would otherwise become a search under "/".
GO_HOME := $(or $(HOME),$(shell echo ~))
GO_ON_PATH := $(shell command -v $(GO) 2>/dev/null)
# Hand the resolved home to the recipes too. Go derives GOPATH and the module
# cache from HOME, so without this a build that got this far fails a second
# time with "module cache not found" — a message about a different subject
# entirely. This restores the real home from the password database; it does not
# invent one.
ifeq ($(strip $(HOME)),)
export HOME := $(GO_HOME)
endif
endif

# mise and asdf both install a shim directory that works without their shell
# hook having run. That is the reliable entry point when the hook is the thing
# that did not happen. $(wildcard) needs no shell, so this is safe everywhere.
GO_SHIM := $(firstword $(wildcard \
	$(GO_HOME)/.local/share/mise/shims/go \
	$(GO_HOME)/.asdf/shims/go \
	/opt/homebrew/bin/go \
	/usr/local/go/bin/go \
	$(GO_HOME)/go/bin/go \
	$(GO_HOME)/.local/share/mise/installs/go/*/bin/go \
	$(GO_HOME)/.asdf/installs/golang/*/go/bin/go))

ifeq ($(strip $(GO_ON_PATH)),)
ifneq ($(strip $(GO_SHIM)),)
GO := $(GO_SHIM)
endif
endif

GOENV   := GOTOOLCHAIN=$(GOTOOLCHAIN)
# Deferred (=, not :=) so only the targets that need the package list pay for a
# `go list`, and so `make help` works without a toolchain at all.
FAST_PACKAGES = $(shell $(GOENV) $(GO) list -f '{{if and (ne .ImportPath "$(MODULE)/internal/doctest") (ne .ImportPath "$(MODULE)/internal/crypto")}}{{.ImportPath}}{{end}}' ./...)
GOLANGCI_LINT_VERSION := v2.11.4
GOVULNCHECK_VERSION   := v1.6.0
VERSION ?= $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo 0.0.0-dev)
COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: all deps build quick test test-race vet lint fmt fmt-check docs-check fixture-scan vuln verify clean run version help snapshot

# `help` and `clean` are exactly what someone reaches for when the build is
# broken, so the diagnosis lives here rather than in a parse-time $(error) that
# would take those down too.
.PHONY: require-go
require-go:
	@$(GO) version >/dev/null 2>&1 || { \
	  echo ""; \
	  echo "No Go toolchain found: $(GO)"; \
	  echo ""; \
	  echo "  make runs recipes with /bin/sh, which does not read ~/.zshrc, so a"; \
	  echo "  toolchain your prompt can see may still be invisible here."; \
	  echo ""; \
	  echo "  Check with:  /bin/sh -c 'command -v go'"; \
	  echo "  If that is empty but 'go version' works in your shell, your PATH is"; \
	  echo "  set by an interactive-only hook. Using mise, 'mise activate' must"; \
	  echo "  appear exactly ONCE in your shell configuration; repeated copies"; \
	  echo "  register competing hooks that can strip the toolchain from PATH."; \
	  echo ""; \
	  echo "  Searched: $(GO_HOME)/.local/share/mise/shims, $(GO_HOME)/.asdf/shims,"; \
	  echo "  /opt/homebrew/bin, /usr/local/go/bin, $(GO_HOME)/go/bin, and the"; \
	  echo "  mise and asdf install trees.  home=$(GO_HOME)"; \
	  echo ""; \
	  echo "  Or point make at it directly:  make GO=\"$$(command -v go)\" build"; \
	  echo ""; \
	  exit 1; }

all: build

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

deps: ## Download Go modules
	$(GOENV) $(GO) mod download
	$(GOENV) $(GO) mod tidy

build: require-go ## Build reinstate binary into ./bin (+ rein symlink)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOENV) $(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)
	@ln -sfn reinstate $(ALIAS)

run: build ## Build and print version
	$(BINARY) version

quick: fmt-check vet ## Fast development gate; release work must use verify
	$(GOENV) $(GO) test $(FAST_PACKAGES)

test: ## Run unit tests
	CGO_ENABLED=0 $(GOENV) $(GO) test ./... -count=1

test-race: ## Run tests with race detector
	CGO_ENABLED=1 $(GOENV) $(GO) test $(FAST_PACKAGES) -race -count=1 -timeout=20m

vet: ## go vet
	$(GOENV) $(GO) vet ./...

fmt: ## gofmt write
	gofmt -w .

fmt-check: ## Fail if gofmt needed
	@files="$$(gofmt -l .)"; test -z "$$files" || { printf '%s\n' "$$files" >&2; exit 1; }

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

snapshot: ## goreleaser snapshot (requires goreleaser; Windows: scripts/snapshot.ps1)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser required for snapshot (Windows: scripts/snapshot.ps1)"; exit 1; }
	@current_tag="$$(git describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null)" || { echo "git describe failed; run: git fetch --tags"; exit 1; }; \
	previous_tag="$$(git describe --tags --match 'v[0-9]*' --abbrev=0 "$$current_tag^" 2>/dev/null || true)"; \
	echo "snapshot: GORELEASER_CURRENT_TAG=$$current_tag GORELEASER_PREVIOUS_TAG=$$previous_tag"; \
	GORELEASER_CURRENT_TAG="$$current_tag" GORELEASER_PREVIOUS_TAG="$$previous_tag" goreleaser release --snapshot --clean

version: build ## Print embedded version
	$(BINARY) version

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) dist coverage.out

install: build ## Install to GOPATH/bin
	install -m 755 $(BINARY) "$${GOBIN:-$${GOPATH:-$$HOME/go}/bin}/reinstate"
