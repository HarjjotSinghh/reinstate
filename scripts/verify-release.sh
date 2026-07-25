#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dist_dir=${1:-"$repo_dir/dist"}

exec "$repo_dir/scripts/check-release-artifacts.sh" "$dist_dir"
