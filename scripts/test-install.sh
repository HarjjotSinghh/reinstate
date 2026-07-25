#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dist_dir=${1:-"$repo_dir/dist"}

"$repo_dir/scripts/verify-release.sh" "$dist_dir"
cd "$repo_dir"
: "${GOTOOLCHAIN:=go1.25.12}"
export GOTOOLCHAIN
go test ./internal/doctest -run TestInstaller -count=1
