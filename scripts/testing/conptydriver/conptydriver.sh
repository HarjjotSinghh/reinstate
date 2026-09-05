#!/usr/bin/env bash
# One-line wrapper: `./scripts/testing/conptydriver/conptydriver.sh [flags] -- <command> [args...]`
# from the repository root, in Git Bash. Windows only (ConPTY); see README.md
# next to this file.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
exec go run ./scripts/testing/conptydriver "$@"
