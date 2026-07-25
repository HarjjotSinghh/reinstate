#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_dir"

: "${GOTOOLCHAIN:=go1.25.12}"
export GOTOOLCHAIN
go test ./internal/doctest -count=1
