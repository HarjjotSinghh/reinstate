# File ownership — v0.6.0

Who may edit what. A branch that touches a path outside its workstream's row
is rejected at Gate 0 before anyone reads the code. Two workstreams needing
the same file talk to the coordinator, who sequences them.

| Path | Owner | Notes |
| ---- | ----- | ----- |
| `CHANGELOG.md` | W1 (then W8 for the date line) | W2/W3/W6 hand their bullets to W1 in the PR description, not by editing the file |
| `RELEASING.md` | W1 (then W8, W9) | |
| `ROADMAP.md` | W1 | |
| `README.md` | W1 | |
| `docs/hop.md`, `docs/hop/**`, `docs/security-model.md`, `docs/getting-started.md`, `docs/cli-reference.md`, `docs/faq.md`, `docs/features.md`, `docs/troubleshooting.md` | W1 | W2 may add the exact message text of T-202 to `docs/hop.md` and `docs/troubleshooting.md` |
| `docs/compatibility.md` | W3 (numbers) then W1 (wording) | Sequenced: W3 merges first |
| `docs/adapters.md`, `docs/agent-support-tiers.md`, `docs/session-storage/**` | W3 | Only the rows for Claude Code and OpenCode versions |
| `docs/planning/v0.6.0-hop/**` | Coordinator | Executors append to `clarifications.md` only via the "executor questions" section |
| `docs/adr/0005-*.md` | Coordinator | |
| `docs/testing/v0.6.0-windows-acceptance.md` | Coordinator | W7 may append the "run notes" section |
| `docs/testing/windows-acceptance-host.md`, `docs/testing/windows-probe-runbook.md` | W4 | |
| `docs/testing/v0.6.0-rc.1-agent-verification-prompts.md` | W8 | |
| `docs/testing/results/2026-09-*-windows-range-widening-v060.md` | W3 | |
| `docs/testing/results/2026-09-*-windows-hop-parity-v060.md` | W6 | |
| `docs/testing/results/2026-09-*-windows-v060rc1-pretag.md` and the tagged-run report | W7 / W7b | |
| `internal/preflight/**` | W2 | |
| `internal/hop/**`, `internal/cli/login*.go`, `internal/cli/whoami*.go` | W2 | T-202 only; W6 fixes go through the coordinator |
| `internal/adapter/claude/**`, `internal/agentcheck/**`, `internal/agents/catalog/claude.go`, `internal/agents/catalog/opencode.go`, `internal/adapter/opencode/**`, `internal/sessionindex/**` | W3 | Version constants and their tests only; behaviour changes go to the coordinator |
| `scripts/testing/hoplab/**`, `scripts/testing/conptydriver/**`, `scripts/testing/fakelocker/**` | W4 | |
| `scripts/tuisandbox/**` | W4 | Only if a row needs a new synthetic state |
| `website/src/data/**`, `website/src/lib/*.test.ts`, `website/src/content/docs/**`, `website/src/pages/contact.astro`, `website/src/pages/roadmap.astro` | W5 (then W8/W9 for pins) | `website/public/install.*` are W9 only |
| `CITATION.cff`, `internal/doctest/bootstrap_install_contract_test.go`, `.github/workflows/*.yml` | W8 / W9 | |
| `internal/cli/**` (everything else), `internal/keyring/**`, `internal/daemon/**`, `internal/verify/**`, `internal/sync/**`, `internal/backend/**` | Coordinator | Defects found by W6/W7 are filed in the results doc; the coordinator assigns a fix branch with its own card |
| `testdata/**` | Owner of the workstream that needs the fixture; synthetic only; secret scanner runs | |
| `go.mod`, `go.sum` | Coordinator | An executor that needs a dependency stops and asks |

Never touched by anyone in this release: `internal/version/version.go`
(ldflags set it), `scripts/install.sh`, `scripts/install.ps1` (their pins
move only in W9), `website/public/**` (W9), `.github/allowed_signers`.
