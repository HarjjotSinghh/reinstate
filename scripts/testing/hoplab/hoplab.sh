#!/usr/bin/env bash
# One-line wrapper: `./scripts/testing/hoplab/hoplab.sh <command> [flags]`
# from the repository root, in Git Bash or any POSIX shell. See README.md
# next to this file.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
exec go run ./scripts/testing/hoplab "$@"
