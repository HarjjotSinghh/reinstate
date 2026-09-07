# Verified-range widening — OpenCode, native Windows x64, 2026-09-07

`AGENT-TIER-JOURNEY-V1` · one agent, single platform.

Reproduces the method used to widen the Claude Code and Qwen Code ranges
([`2026-09-06-windows-range-widening-v060.md`](2026-09-06-windows-range-widening-v060.md),
[`2026-09-07-windows-range-widening-qwen-v060.md`](2026-09-07-windows-range-widening-qwen-v060.md)),
under [ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md)
D3, for the OpenCode ceiling that same acceptance host's own self-update
pushed past a second time this cycle.

**macOS evidence: pending (ADR 0005).**

## 0. Why this record exists

The acceptance host's OpenCode installation self-updated past the ceiling
`2026-09-06-windows-range-widening-v060.md` had just established:
`opencode --version` now prints `1.18.29`, while the range in tree was
`1.18.21`–`1.18.27` (`internal/agents/catalog/opencode.go`). Every OpenCode
resume/fork/handoff row refuses with exit `5` (`agent.version` block,
`native agent version 1.18.29 is outside the verified range 1.18.21 to
1.18.27 inclusive`) — correct, fail-closed behavior, and the reason for this
record.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-07 |
| Device | `windows-amd64`, native — not WSL |
| OS | Microsoft Windows 11 Pro, 10.0.26200 |
| Branch | `v060/rc5` |
| Branch tip at time of this record | `6d4113783d9db76ea5b21a14949af584aa4d2079` |
| Go toolchain | `go1.25.13` (`GOTOOLCHAIN=go1.25.13`) |
| OpenCode installed | `1.18.29` (managed self-update, default `XDG_DATA_HOME`) |
| Lab root | `<lab-project>` (`D:\ReinstateAcceptanceProjects\rc5-opencode\`, a real, throwaway git repository) |

OpenCode was run as the real installed vendor binary (`opencode.exe`) inside
the throwaway lab project. No real transcript is quoted anywhere below; the
one token named is a marker planted by this record for the sole purpose of
proving continuation, redacted as `<token>`.

## 2. Isolation — no redirected `XDG_DATA_HOME` this time, by design

Every prior widening record in this series (Claude Code, Qwen Code) ran
against a redirected, throwaway credential/session root
(`CLAUDE_CONFIG_DIR`, `QWEN_HOME`). OpenCode's storage root variable is
`XDG_DATA_HOME` (it names the parent; the store is `$XDG_DATA_HOME/opencode`
— `internal/agents/catalog/opencode.go`'s `RootEnv`/`RootEnvSuffix`), and this
task's ground rules name `XDG_DATA_HOME` a live home this run must leave
alone rather than redirect, because other work on this shared acceptance
host reads from it concurrently. `rein doctor --agents --json` confirms
OpenCode declares no `OPENCODE_*` alternative root variable — `XDG_DATA_HOME`
is its only one — so there was no redirect target available even if one had
been wanted.

The isolation this record uses instead: a session was created inside a
throwaway, single-purpose git repository (`<lab-project>`) and identified
**only** by a token planted in its own first turn — never by browsing,
listing in full, or quoting any other session already in the shared store.
Every query below that touches the shared `opencode.db` is scoped to either
that one session id or that one project directory; no other session's title,
id, or content is read or reproduced here. `REINSTATE_HOME` was still an
isolated, throwaway directory of its own, so Reinstate's own index and
handoff artifacts never touched the operator's real Reinstate home.

## 3. Rows exercised

| Step | Command (argv) | Result |
| ---- | --------------- | ------ |
| Create session | `opencode run --model opencode/big-pickle "Reply with exactly this text and nothing else: <token>" --format json` (cwd = `<lab-project>`, a real git repo) | PASS — real session created, vendor returned `<token>`; `sessionID` = `<id>` |
| Session row shape | `session` table, scoped to `id = <id>` only | PASS — same columns as the `1.18.27` record: `directory`, `title`, `version` (reads `1.18.29`), `agent`, `model`; no schema change |
| Message/part row shape | `message`/`part` tables, scoped to `session_id = <id>` only | PASS — same snake_case columns the reader already queries (`session_id`, `message_id`, `time_created`); 2 `message` rows (`user`/`assistant`), 5 `part` rows (`text`/`step-start`/`reasoning`/`text`/`step-finish`); `message.data` carries `agent`/`model`/`role`/`summary`/`time` only — `id`, `sessionID`, `messageID` stay stripped into columns, unchanged from the documented `1.18.21` shape |
| Find it | `rein sessions --agent opencode --json` (before the code change) | PASS — the planted-token session appeared among the store's rows; only its own id and title were read, no other row's content |
| Find it by content | `rein search <token> --json` | PASS — found by message body text, not just title: confirms the `v0.6.0-rc.2` search-text fix (part rows joined to `role":"user"` messages) still works on `1.18.29` |
| Inspect before the code change | `rein inspect opencode:<id> --json` | PASS — `agent.version.status=untested`, `actual=1.18.29`, message names range `1.18.21 to 1.18.27 inclusive`, `decision=blocked`, `block_exit_code=5` |
| Dry-run launch plan before the code change | `rein resume opencode:<id> --dry-run --json` | PASS (fail-closed) — refused `compatibility`, `environment preflight is blocked`, exit `5` |
| Handoff dry-run before the code change | `rein handoff opencode:<id> --to qwen --dry-run --json` | PASS (fail-closed) — refused `compatibility`, `handoff: environment preflight is blocked` |
| Widen the ceiling | `Max` (`internal/agents/catalog/opencode.go`) `1.18.27` → `1.18.29`; rebuild | PASS — `gofmt`/`go vet`/`go build` clean |
| Inspect after the code change | `rein inspect opencode:<id> --json` | PASS — `agent.version.status=match`, `actual=1.18.29`, `message="the native agent version is in the verified range"`, `decision=confirmation_required` |
| Dry-run launch plan | `rein resume opencode:<id> --dry-run --json` | PASS — plan `opencode --session <id>`, `cwd` = the session's own workspace; `decision=confirmation_required` (first-launch baseline warning only; no block) |
| Non-`--dry-run` guard | `rein resume opencode:<id> --json` (no `--dry-run`) | PASS — refused `--json requires --dry-run for native agent launches`, exit `2`, identical to the Claude Code/OpenCode/Qwen rows in the earlier 2026-09-06 and 2026-09-07 records |
| Fork plan | `rein resume opencode:<id> --fork --dry-run --json` | PASS — plan `opencode --session <id> --fork` |
| **Physical resume** | `opencode --session <id> run "What token did you reply with earlier in this conversation? Reply with only the token, nothing else." --format json` (the dry-run plan's `--session` argument, plus the vendor's own non-interactive `run` form) | **PASS** — same `sessionID` as `<id>`; result was exactly `<token>`, which exists nowhere but the original session's first turn |
| **Physical fork** | `opencode run --session <id> --fork "Reply OK" --format json` (the dry-run fork plan's argv, vendor's `run` form) | **PASS** — a new session id appeared, carrying the original session's 2 turns (4 messages) plus 1 new turn (2 messages: 6 total); the original session's own row was unmodified by the fork itself |
| Handoff dry-run (T4, session-scoped digest path) | `rein handoff opencode:<id> --to qwen --dry-run --json` (opencode as the handoff *source*) | PASS — capsule + fidelity report produced, scoped to this session only (`parse.Events=4`, matching this session's own row count at capture time); destination argv `qwen --session-id <new-id> --prompt-interactive "..."` |
| Version-output parsing | `opencode --version` | PASS — bare line `1.18.29` on stdout, empty stderr; matches the existing `opencodeVersionPattern` regex unmodified |
| `opencode session list --format json` | not exercised | not applicable — the shipped index source reads the embedded SQLite store directly (`internal/agents/sources/opencode/sqlite.go`) and never shells out to this vendor command, per `docs/session-storage/opencode.md`'s own opening paragraph; unchanged in this record |
| New-session id / id-collision form | not exercised | not applicable — OpenCode assigns its own session id; the catalog's `NewSession` flag is `--prompt` (`internal/handoff/target_opencode.go`), not a caller-chosen id, so there is no id-collision row to re-run (unchanged from the documented `1.18.21`–`1.18.27` behavior) |

## 4. What changed between `1.18.27` and `1.18.29`, and what did not

Unchanged: session/message/part table and column shapes, project-bucket
directory association, `message.data`'s stripped-identity fields,
`--session`/`--session … --fork`/`--continue`, the non-interactive `run`
form, the `search_text` join added in `v0.6.0-rc.2`, the structured-handoff
capsule and fidelity path, and the bare-semver `--version` line the existing
regex already parses.

One observation, newly recorded rather than newly changed (no `1.18.27`
build remained installed on this host to diff against, since the host had
already self-updated past it before this record began): a native `--fork`
does not set the child session's `parent_id` column — that column read back
`null` for the forked session in this record. The vendor instead threads
lineage through `session.title`, appending `" (fork #1)"` to the original
session's own title. `internal/agents/sources/opencode/sqlite.go` never
reads `parent_id` for anything today, so this is not a regression against
any existing reader behavior; it is recorded here because no prior device
journey inspected a forked row's schema this closely. `docs/compatibility.md`
and `docs/session-storage/opencode.md` continue to source fork *capability*
(`capabilities.fork`) from `Native.Fork` being non-empty, not from this
column, so nothing downstream depends on it.

## 5. Range declaration moved

| Location | Old | New |
| -------- | --- | --- |
| `internal/agents/catalog/opencode.go` (`VersionSpec.Max`) | `1.18.27` | `1.18.29` |

`internal/agentcheck/agent.go`'s `testFallbackDefinitions()` has no OpenCode
entry (only `claude` and `codex`), so there is nothing to move there.

Tests moved with the ceiling:

- `internal/agents/catalog/opencode_resume_test.go`
  `TestParseOpenCodeVersion` — adds a `1.18.29` case.
- `internal/agents/catalog/opencode_resume_test.go`
  `TestOpenCodeDescriptorIsT5` — the measured-range assertion moves to
  `1.18.21`–`1.18.29`.
- `internal/agents/catalog/catalog_test.go`
  `TestShippedAgentsRegisterAtDeclaredTiers` — the OpenCode row's expected
  max moves to `1.18.29`.

Nothing else in the matrix moves: `internal/agentcheck/agent_embedded_store_test.go`'s
`1.18.21` values are synthetic fixture literals unrelated to the shipped
catalog range, not touched.

Fail-closed boundary: `1.18.30` (one past the new ceiling) has no literal
unit-test case on either side of this change and was not re-verified
against the real vendor binary, since it is not installed on this host —
the same allowance the `2026-09-06` and `2026-09-07` (Qwen) records used for
their own untested one-past-ceiling boundaries.

## 6. Gates

| Gate | Command | Result |
| ---- | ------- | ------ |
| Format | `gofmt -l internal/agents/catalog/opencode.go internal/agents/catalog/opencode_resume_test.go internal/agents/catalog/catalog_test.go` | PASS — empty |
| Vet (owned packages) | `GOTOOLCHAIN=go1.25.13 go vet ./internal/agents/catalog/... ./internal/agentcheck/...` | PASS |
| Vet (whole tree) | `GOTOOLCHAIN=go1.25.13 go vet ./...` | pre-existing failure unrelated to this task's files: `internal/agents/probe/collect_test.go` (a different in-progress task's uncommitted edit, sharing this worktree — `internal/agents/probe/collect.go`, `collect_test.go`, `shape.go` modified and `zzz_regen_gemini_test.go` added, none by this record) fails to build; no failure in any path this record touches |
| Tidy | `GOTOOLCHAIN=go1.25.13 go mod tidy -diff` | PASS — empty |
| Unit suite | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test -p 4 ./... -count=1` | PASS on every package this record's files touch (`internal/agents/catalog`, `internal/doctest`, and the whole tree otherwise); the only failure was a transient `tmp_probe_repro` build error from another executor's scratch directory disappearing mid-run in this shared worktree, gone by the time of the next run, and not a path this record owns |
| Lint | `make lint` | pre-existing failure, same unrelated file as the vet row: `internal/agents/probe/zzz_regen_gemini_test.go:102` (`staticcheck SA9003`), an uncommitted addition from a different in-progress task sharing this worktree; nothing in this record's own files is flagged |
| Doc gate | `go test ./internal/doctest/... -count=1` | PASS |
| Doc-link script | `bash scripts/check-docs.sh` | PASS |
| Website unit tests | `cd website && npx vitest run` | PASS — 47 files / 321 tests |

## 7. Verdict

The widening is supported on **native Windows only**. Apple Silicon macOS
evidence for `1.18.21`–`1.18.29` is deferred (tracked in `#403`, the same
macOS-deferred issue every other v0.6.0 native-Windows-only widening in this
release cites), consistent with ADR 0005 D3.
