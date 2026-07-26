# Development workflow

## Prerequisites

- Go 1.25.12 or newer
- Git
- Make on macOS, Linux, or WSL2
- PowerShell 7 is helpful when changing Windows scripts

No Node.js toolchain is required.

## Setup

```bash
git clone https://github.com/HarjjotSinghh/reinstate.git
cd reinstate
make deps
make build
./bin/reinstate version
```

Use a focused branch or Git worktree for each change. Do not develop against
real session fixtures. The committed `testdata/` tree must remain synthetic.

## Required local gate

```bash
make quick
make verify
```

`make quick` is the fast edit-loop gate: formatting, vet, and focused product
packages. It intentionally omits the slow production-KDF and
subprocess/document packages and reuses Go's cache, so it cannot approve a
merge or release.

That gate runs formatting, vet, pinned lint, unit/integration tests, race tests,
`govulncheck`, documentation contracts, fixture secret scanning, and a build.
Documentation and fixture contracts run once through the full test target; the
race target filters documentation because its shell/document subprocesses are
not Go product race surfaces. It also omits the stateless crypto wrapper.
High-level CLI, doctor, and sync tests retain the age envelope format with a
reduced test-only scrypt cost, while the ordinary full test target covers the
production crypto default.
Run focused package tests while iterating, but run the full gate before a PR.

For installer or release work, also run:

```bash
goreleaser release --snapshot --clean
sh scripts/test-install.sh dist
```

Native Windows changes must pass the Windows CI row. Record any manual
Windows/WSL/macOS evidence in the PR without including private paths.

## Project boundaries

- `internal/adapter/`: vendor detection, discovery, transformation, restore
- `internal/sync/`: encrypted snapshots, manifests, conflicts
- `internal/backend/`: storage contracts and implementations
- `internal/cli/`: command behavior and stable exit codes
- `internal/schema/`: persisted formats; changes require migration notes
- `docs/prompts/`: copy-paste agent setup contracts

Open an issue or RFC before changing encrypted schemas, remote key layout,
security defaults, stable exit codes, or phase scope.
