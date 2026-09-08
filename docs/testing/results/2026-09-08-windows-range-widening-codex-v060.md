# Verified-range widening — OpenAI Codex CLI, native Windows x64, 2026-09-08

`AGENT-TIER-JOURNEY-V1` · one agent, single platform, read-only evidence only.

Reproduces the method used to widen the Claude Code, Qwen Code, OpenCode, and
Grok Build ranges
([`2026-09-06-windows-range-widening-v060.md`](2026-09-06-windows-range-widening-v060.md),
[`2026-09-07-windows-range-widening-qwen-v060.md`](2026-09-07-windows-range-widening-qwen-v060.md),
[`2026-09-07-windows-range-widening-opencode-v060.md`](2026-09-07-windows-range-widening-opencode-v060.md),
[`2026-09-08-windows-range-widening-grok-v060.md`](2026-09-08-windows-range-widening-grok-v060.md)),
under [ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md)
D3, for the OpenAI Codex CLI ceiling — with the same disclosed limitation the
Grok record carried: this record's evidence is **read-only**. The
completed-turn resume/fork journey (`codex:E1`–`E3`) cannot be produced here,
for a reason independent of this change (below).

**macOS evidence: pending (ADR 0005).**
**Completed-turn evidence (`codex:E1`–`E3` against `0.153.4`): deferred to
the `v0.6.0-rc.7` tagged run's codex rows, collected once the acceptance
host's Codex account usage limit resets. See §5.**

## 0. Why this record exists

The acceptance host's Codex CLI installation self-updated past the ceiling
the catalog carried: `codex --version` now prints `codex-cli 0.153.4`, while
the in-tree range was `0.133.0`–`0.149.0`
(`internal/adapter/codex/codex.go`, `internal/agents/catalog/codex.go`,
`internal/agentcheck/agent.go`). This matches the `v0.6.0-rc.6` tagged
report's own findings: `codex:E1`–`E3` were recorded `NOT TESTED (version
drift)`, with `rein resume codex:<id> --dry-run --json` correctly refusing
with exit `5` (`agent.version`, `native agent version 0.153.4 is outside the
verified range 0.133.0 to 0.149.0 inclusive`) against a real, `rein`-visible
Codex session on that same host — correct, fail-closed behavior, and the
reason this record exists. That report also noted a second, in-range Codex
CLI install (`0.146.0`) coexists on the host and was used to exercise the
version-gated rows (`E4`, `E6`) that need an in-range binary.

Orthogonal to the version ceiling, the host's Codex account is presently
usage-limit exhausted: `codex exec` prints `You've hit your usage limit ...
try again at 10:13 AM`, reproduced identically against both the drifted
`0.153.4` build and the in-range `0.146.0` build during the `v0.6.0-rc.6` run
(`MatrixG:G1`'s Codex half, `PARTIAL`). No completed Codex turn is possible
from this harness before that reset. The maintainer's standing policy
(2026-09-07, Q27) is that when a vendor CLI self-updates past the verified
ceiling, Reinstate widens the range on native Windows evidence rather than
block — Reinstate always wants to support the latest version — so this
record widens the ceiling on the read-only evidence that this environment
*can* produce today, and defers the completed-turn rows to the `v0.6.0-rc.7`
tagged run's own codex rows, collected once the account limit resets.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-07 (2026-09-08 local) |
| Device | `windows-amd64`, native — not WSL |
| OS | Microsoft Windows 11 Pro, 10.0.26200 |
| Worktree | `D:\Projects\reinstate-worktrees\rc7`, branch `v060/rc7` |
| Branch tip at time of this record | `6d5de83d5e14f6a690ca986f12ff79b81c00bacb` |
| Go toolchain | `go1.25.13` (`GOTOOLCHAIN=go1.25.13`) |
| Codex CLI installed (default, drifted) | `0.153.4` (`codex-cli 0.153.4`) |
| Codex CLI installed (second, in-range) | `0.146.0`, also present on this host (used for `v0.6.0-rc.6`'s `G2`/`E4`/`E6` rows) |
| Codex CLI verified range before this record | `0.133.0`–`0.149.0` |
| Codex CLI verified range after this record | `0.133.0`–`0.153.4` |
| Account state | Usage-limit exhausted; resets `10:13` local, `2026-09-08` |

No real transcript, session id, project path, or session title from the
operator's own `~/.codex` tree is quoted anywhere below. Every observation
about the real Codex home is scoped to command output shape, sanitized
`rein doctor` probe fields, and file-naming shape only, per this task's
ground rules; no session content or identifier from a real, operator-created
session is reproduced here. The launch-plan and doctor probes below that
needed an actual session used only the committed synthetic fixture at
`testdata/adapters/codex/windows/sessions/rollout-syn-001.jsonl`
(`fixture-user`/`demo`), never a real session.

## 2. Isolation

`CODEX_HOME` is Codex CLI's root override
(`internal/agents/catalog/codex.go`'s `RootEnv`). This task's ground rules
leave `CODEX_HOME` untouched (never unset or redirected), so the read-only
probes below were run in three forms:

- `codex --version`, `codex --help`, `codex exec --help`,
  `codex resume --help`, and `codex fork --help`: read-only, stateless
  vendor commands that touch no session store; run directly against the
  installed `0.153.4` binary.
- `rein doctor --agents --json`: run against a fresh, throwaway
  `REINSTATE_HOME` (this task's own `unset REINSTATE_BACKEND
  REINSTATE_MEMORY_BACKEND_DIR` rule applied), reading whatever real
  `CODEX_HOME` this task's ground rules leave in effect the way `rein
  doctor` always does — but its own output is already a sanitized,
  aggregated shape probe (glob-level directory/file counts, median byte
  sizes, and a small sample of first-line JSON key names with no values),
  never raw file content or identifiers, which is why this is the one probe
  in this record that reads a live tree directly.
- `rein resume codex:<id> --dry-run --json`: a real, non-mutating launch-plan
  build against the installed `0.153.4` binary, run twice (before and after
  this record's code change) against an isolated `REINSTATE_HOME` and a
  `CODEX_HOME` seeded only with the committed synthetic fixture
  `testdata/adapters/codex/windows/sessions/rollout-syn-001.jsonl` — a real,
  completed conversational turn against a session of the executor's own
  making is impossible without a live Codex turn (§0), so the committed
  fixture stands in for one, per this task's ground rules.

No Codex session was created by this record, and nothing was written under
any real Codex home by any command run in this record.

## 3. Read-only evidence gathered

| Step | Command / probe | Result |
| ---- | ---------------- | ------ |
| Version output shape | `codex --version` | `codex-cli 0.153.4` — same shape as `0.149.0`: bare `codex-cli <semver>`, nothing on stderr. Parses under the existing `codexVersionPattern` regex unmodified — no pattern change needed to widen the range. |
| Top-level command surface | `codex --help` | Same top-level usage form (`codex [OPTIONS] [PROMPT]` / `codex [OPTIONS] <COMMAND> [ARGS]`) and the `resume`/`fork`/`exec` subcommands `rein`'s native launch plan depends on are all present, with the same one-line descriptions. Additional subcommands not used by any launch plan (`agents`, `plugin`, `app-server`, `remote-control`, `app`, `queue`, `archive`, `delete`, `migrate-rollouts`, `unarchive`, `cloud`, `exec-server`, `features`) are new surface area outside anything `rein` invokes. |
| Native resume argv | `codex resume --help` | `Usage: codex resume [OPTIONS] [SESSION_ID] [PROMPT]` — a bare positional `[SESSION_ID]` and optional `[PROMPT]`, no required flag. Identical shape to what `rein`'s `Resume: []string{"resume", "{{.SessionID}}"}` template builds. `--last` (continue most recent) is present and unused by `rein`'s explicit-id path. Unchanged. |
| Native fork argv | `codex fork --help` | `Usage: codex fork [OPTIONS] [SESSION_ID] [PROMPT]` — same bare-positional shape `rein`'s `Fork: []string{"fork", "{{.SessionID}}"}` template builds. Unchanged. |
| Initial-prompt argv | `codex --help` | `Usage: codex [OPTIONS] [PROMPT]` — the positional `[PROMPT]` argument `rein`'s `agents.PromptArgv` initial-prompt mode targets is unchanged. |
| Non-interactive form | `codex exec --help` | `Usage: codex exec [OPTIONS] [PROMPT]` with `resume`/`fork`/`review` subcommands. Present and unchanged in shape from the vendor contract `docs/session-storage/codex.md` already documents (`codex exec …`, `codex exec --last`); not exercised for a completion in this record (§0, §5). |
| Session directory / file naming shape | `rein doctor --agents --json`, `name_shapes` field, real `CODEX_HOME` | `sessions/*/*/*/*` → shape `<slug>-<n>-<uuid-v4>.jsonl`, 423 samples — the same `rollout-<timestamp>-<uuid>.jsonl` naming convention `docs/session-storage/codex.md` already documents, confirmed unchanged under the live, `0.153.4`-written tree this task's ground rules leave in scope. |
| `rein doctor --agents --json`, Codex row | isolated `REINSTATE_HOME`, real `CODEX_HOME` | `"key":"codex"`, `"executable_on_path":true`, `"version_raw":"codex-cli 0.153.4"`; the sanitized `tree`/`name_shapes` blocks report glob-level shape (directory/file counts, median byte sizes, path-segment shapes) consistent with the documented layout. No id, path, or file content appears in this output — it is Reinstate's own sanitized shape probe. |
| Committed-fixture record shape | `rein sessions --agent codex --json`, `CODEX_HOME` seeded only with `testdata/adapters/codex/windows/sessions/rollout-syn-001.jsonl` | Discovers exactly `codex:rollout-syn-001` (`project: "demo"`, `workspace: "C:\Users\fixture-user\code\demo"`) — the committed synthetic fixture, addressed correctly with the same filename-derived identity `docs/session-storage/codex.md` documents. |
| Launch-plan build, **before** this change | `rein resume codex:rollout-syn-001 --dry-run --json` (pre-widening binary, `Max: "0.149.0"`) | Exit `5`. `agent.version` check: `status:"unknown"`, `severity:"block"`, `actual:"0.153.4"`, `message:"native agent version 0.153.4 is outside the verified range 0.133.0 to 0.149.0 inclusive"`. Fail-closed, as expected — the same refusal `v0.6.0-rc.6`'s `codex:E1`–`E3` recorded against a real session. |
| Launch-plan build, **after** this change | same command (post-widening binary, `Max: "0.153.4"`) | Exit `0`, `decision:"confirmation_required"`. `agent.version` check: `status:"match"`, `severity:"info"`, `actual:"0.153.4"`, `message:"the native agent version is in the verified range"`. The produced plan's `args` are exactly `["resume", "rollout-syn-001"]` against `executable:"codex"` — the same argv shape §3's `codex resume --help` measurement confirms is still accepted by the real `0.153.4` binary. |

## 4. What changed between `0.149.0` and `0.153.4`, and what did not

Unchanged: the `codex --version` output shape (bare `codex-cli <semver>`,
nothing on stderr — the same regex the catalog has parsed since the T3
promotion); every launch-plan-relevant `--help` surface `rein`'s native argv
depends on (`resume [SESSION_ID] [PROMPT]`, `fork [SESSION_ID] [PROMPT]`,
the positional `[PROMPT]` for a new session, `exec [PROMPT]` for the
non-interactive form); the session file naming convention at the level the
index source globs (`sessions/YYYY/MM/DD/rollout-<timestamp>-<uuid>.jsonl`);
and, confirmed end-to-end, the whole version-gate → launch-plan pipeline
against the real `0.153.4` binary and the committed synthetic fixture (§3's
before/after `--dry-run` rows).

Newly observed, not newly broken: several new top-level subcommands
(`agents`, `plugin`, `app-server`, `remote-control`, `app`, `queue`,
`archive`, `delete`, `migrate-rollouts`, `unarchive`, `cloud`,
`exec-server`, `features`) that no `rein` launch plan invokes, including a
`migrate-rollouts` command ("Inspect or migrate legacy local sessions to
paginated thread history") that is opt-in (`--apply` required to write
anything) and was not run in this record. No `rollout-*.jsonl` file from a
session created under `0.153.4` specifically was independently re-verified
for its first-line JSON key names in this record: the real, operator-owned
`~/.codex/sessions` tree carried no entry from today at the time of this
record, and reading real session content — even filtered to key names — is
out of scope for this executor per this task's own agent-store protections.
This is a disclosed gap, not a claim of verification: the naming convention
(§3) and the reader-consumed fields (`session_meta`'s `payload.{cwd,git}`,
`event_msg`/`response_item` message shapes) were confirmed unchanged only
against the committed synthetic fixture and the existing (pre-`0.153.4`)
documented contract, not against a fresh `0.153.4`-authored rollout file.

## 5. Completed-turn evidence — deferred to the `v0.6.0-rc.7` tagged run

`codex:E1` (resume), `codex:E2` (fork), and `codex:E3` (recall) each require
a real, completed conversational turn against the live `codex` binary: a
session is created, a token is planted in the first turn, the session is
resumed or forked through `rein`'s own launch plan, and the resumed agent
must complete a real turn that recalls the token. That is exactly the step
the `v0.6.0-rc.6` tagged report recorded blocked, independent of `rein`: the
host's live, authenticated Codex account returns a usage-limit refusal
(`codex exec` prints `You've hit your usage limit ... try again at 10:13
AM`), reproduced identically against both the drifted `0.153.4` build and
the in-range `0.146.0` build.

Per this task's ground rules, that limit resets at `10:13` local on
`2026-09-08` and is not yet reset at the time of this record — no completed
Codex turn is possible from this harness before then. Read-only evidence
(§3) is everything this environment can produce today.

`codex:E1`, `codex:E2`, and `codex:E3` against `0.153.4` will therefore be
collected from the `v0.6.0-rc.7` tagged run's own codex rows, once the
account limit resets — the same division of labor this task's ground rules
specify. This record does not claim those rows as collected.

## 6. Range declaration moved

| Location | Old | New |
| -------- | --- | --- |
| `internal/adapter/codex/codex.go` (`maximumVerifiedCodexVersion`) | `0.149.0` | `0.153.4` |
| `internal/agents/catalog/codex.go` (`VersionSpec.Max`) | `0.149.0` | `0.153.4` |
| `internal/agentcheck/agent.go` (`testFallbackDefinitions()["codex"].Max`) | `0.149.0` | `0.153.4` |

Tests moved with the ceiling:

- `internal/adapter/codex/codex_test.go` `TestCodexSupportedVersionRange` —
  `0.149.1` moves from `want: false` to `want: true` (now inside the
  widened range); adds `0.153.4` (`want: true`) and `0.153.5`
  (`want: false`, the new one-past-ceiling boundary).
- `internal/agents/catalog/catalog_test.go`
  `TestShippedAgentsRegisterAtDeclaredTiers` — the Codex row's expected max
  moves to `0.153.4`; `TestVersionParsersMatchAgentcheckShape` adds a
  `codex-cli 0.153.4` parse case.

Fail-closed boundary: no build between `0.149.1` and `0.153.3` inclusive,
and no build past `0.153.4`, was installed on this host or physically
measured, so none of them gained a literal test case beyond the boundary
values above — the same allowance the prior widening records in this series
used for their own untested interior and one-past-ceiling versions.

## 7. Gates

| Gate | Command | Result |
| ---- | ------- | ------ |
| Format | `gofmt -l internal/adapter/codex/codex.go internal/adapter/codex/codex_test.go internal/agents/catalog/codex.go internal/agents/catalog/catalog_test.go internal/agentcheck/agent.go` | see the run recorded with this commit |
| Vet (owned packages) | `GOTOOLCHAIN=go1.25.13 go vet ./internal/adapter/codex/... ./internal/agents/catalog/... ./internal/agentcheck/...` | see the run recorded with this commit |
| Vet (whole tree) | `GOTOOLCHAIN=go1.25.13 go vet ./...` | see the run recorded with this commit |
| Tidy | `GOTOOLCHAIN=go1.25.13 go mod tidy -diff` | see the run recorded with this commit |
| Unit suite | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test -p 4 ./... -count=1` | see the run recorded with this commit |
| Lint | `make lint` | see the run recorded with this commit |
| Doc gate | `go test ./internal/doctest/... -count=1` | see the run recorded with this commit |
| Doc-link script | `bash scripts/check-docs.sh` | see the run recorded with this commit |
| Website unit tests | `cd website && npm ci && npx vitest run` | see the run recorded with this commit |
| Acceptance-matrix row count | `rein doctor --agents --acceptance-matrix --json` (`"row_count"`) | still `178` — widening `Min`/`Max` on an already-registered agent changes no tier, family, or agent count, so the generated matrix's row count is unaffected |

## 8. Verdict

The verified range is widened to `0.133.0`–`0.153.4` on **read-only** native
Windows evidence, plus a real before/after launch-plan build against a
committed synthetic fixture (§3), under the maintainer's standing
self-update policy (2026-09-07, Q27). Version-output parsing and every
launch-plan-relevant `--help` surface are confirmed unchanged from
`0.149.0`; the version-gate → launch-plan pipeline itself is confirmed to
flip from a correct fail-closed refusal (exit `5`) to a correct, fully-built
plan (exit `0`) across the code change in this record. Apple Silicon macOS
evidence is deferred (ADR 0005 D3, tracked in the same `#403` every other
`v0.6.0` native-Windows-only widening in this release cites). The
completed-turn `codex:E1`–`E3` rows against `0.153.4` are explicitly **not**
collected by this record — they are deferred to the `v0.6.0-rc.7` tagged
run's own codex rows, to be collected once the acceptance host's Codex
account usage limit resets at `10:13` local, `2026-09-08`.
