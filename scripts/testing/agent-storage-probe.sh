#!/usr/bin/env bash
# Emit AGENT-PROBE-V1 via the installed Reinstate binary.
# Output is byte-identical to: rein doctor --agents --json
set -euo pipefail
if command -v rein >/dev/null 2>&1; then
	exec rein doctor --agents --json "$@"
fi
if command -v reinstate >/dev/null 2>&1; then
	exec reinstate doctor --agents --json "$@"
fi
echo "rein/reinstate not found on PATH" >&2
exit 1
