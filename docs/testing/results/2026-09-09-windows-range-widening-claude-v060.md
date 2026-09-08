# Verified-range widening — Claude Code, native Windows x64, 2026-09-09

`AGENT-TIER-JOURNEY-V1` · one agent, single platform, completed-turn evidence.

Reproduces the method used to widen the Claude Code, Qwen Code, OpenCode,
Grok Build, and Codex CLI ranges
([`2026-09-06-windows-range-widening-v060.md`](2026-09-06-windows-range-widening-v060.md),
[`2026-09-07-windows-range-widening-qwen-v060.md`](2026-09-07-windows-range-widening-qwen-v060.md),
[`2026-09-07-windows-range-widening-opencode-v060.md`](2026-09-07-windows-range-widening-opencode-v060.md),
[`2026-09-08-windows-range-widening-grok-v060.md`](2026-09-08-windows-range-widening-grok-v060.md),
[`2026-09-08-windows-range-widening-codex-v060.md`](2026-09-08-windows-range-widening-codex-v060.md)),
under [ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md)
D3, for the Claude Code ceiling.

**macOS evidence: pending (ADR 0005).**

## 0. Why this record exists

The acceptance host's Claude Code installation self-updated past the
in-tree ceiling: `claude --version` now prints `2.1.265 (Claude Code)`,
while the range widened for `v0.6.0` on 2026-09-06 topped out at `2.1.263`
(`internal/adapter/claude/claude.go`, `internal/agentcheck/agent.go`,
`internal/agents/catalog/claude.go`). Two `internal/handoff` tests
(`TestPlanPartialFinalRecordByteExactCrossCheck/claude-windows`,
`TestPlanPartialFinalRecordRefusalIsOSShapeIndependent/claude-windows`)
resolve the real, registered `internal/transcript` Claude reader with no
fake version resolver, so `reader.Probe` shells out to the installed
`claude --version` exactly like production; against a drifted `2.1.265`
binary that probe now reports `CompatibilityUntested`, and `Plan()` refuses
before either test can reach the behavior it actually checks. The
maintainer's standing policy (2026-09-07, Q27) is that when a vendor CLI
self-updates past the verified ceiling, Reinstate widens the range on
native Windows evidence rather than block, so this record widens the
ceiling on real, completed-turn evidence gathered against the real
`2.1.265` binary.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-08 (2026-09-09 local) |
| Device | `windows-amd64`, native — not WSL |
| OS | Microsoft Windows 11 Pro, 10.0.26200 |
| Worktree | `D:\Projects\reinstate-worktrees\rc8`, branch `v060/rc8` |
| Branch tip at time of this record | `4602a58fe077503dc2ac70e47d7f12f205caa49c` |
| Go toolchain | `go1.25.13` (`GOTOOLCHAIN=go1.25.13`) |
| Claude Code installed | `2.1.265` (`claude --version` → `2.1.265 (Claude Code)`) |
| Claude Code verified range before this record | `2.1.219`–`2.1.263` |
| Claude Code verified range after this record | `2.1.219`–`2.1.265` |
| Lab project | `D:\ReinstateAcceptanceProjects\rc8-claude` (throwaway git repo, never committed) |
| `CLAUDE_CONFIG_DIR` | the host's own, live config — never repointed, per this task's ground rules |

`CLAUDE_CONFIG_DIR` is a live, real home: this record's evidence is
therefore scoped to command output shape, sanitized `rein doctor` probe
fields, glob-level file-naming shape, and this record's own throwaway
session — never any other real transcript, session id, project path, or
session title from the operator's own tree. No prompt sent to the real
account beyond the two named below is reproduced here. The vendor's own
nested-session environment variables
(`CLAUDE_CODE_CHILD_SESSION`/`CLAUDE_CODE_SESSION_ID`/
`CLAUDE_CODE_MESSAGING_*`/`CLAUDE_CODE_BRIDGE_SESSION_ID`/`CLAUDECODE`/
`CLAUDE_PID`, inherited from the agent session driving this record) were
cleared before every launch below, so the child `claude` process this
record created is a normal, resumable session rather than a transcript-off
child session.

## 2. Session creation

A real, non-interactive turn planted a token in a fresh session:

```
claude -p "Reply with exactly this token and nothing else: <token>" --output-format json
```

run from `D:\ReinstateAcceptanceProjects\rc8-claude` with the nested-session
variables above cleared. The vendor returned the token exactly
(`"result":"<token>"`), under `session_id`
`866010d7-d6de-4060-baad-b2e61cabbaad`, written to
`<CLAUDE_CONFIG_DIR>\projects\D--ReinstateAcceptanceProjects-rc8-claude\866010d7-d6de-4060-baad-b2e61cabbaad.jsonl`
— the documented `<root>/projects/<project-key>/<session-uuid>.jsonl` layout
(`docs/session-storage/claude.md`), unchanged.

## 3. Evidence gathered

All `rein` commands below ran from a binary built from this record's own
branch tip, against an isolated `REINSTATE_HOME`
(`REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR` unset, per this task's
ground rules) and the real, live `CLAUDE_CONFIG_DIR`.

| Step | Command | Result |
| ---- | ------- | ------ |
| Find it by content | `rein search <token> --agent claude --json` | **PASS** — found exactly `claude:866010d7-d6de-4060-baad-b2e61cabbaad` (project `rc8-claude`, workspace `D:\ReinstateAcceptanceProjects\rc8-claude`), nothing else; three unrelated warnings from other, pre-existing sessions on the live store (two oversized-record skips, one segment coalescing) confirm the scan walked the real tree and changed nothing in it |
| Dry-run launch plan, **before** this record's code change (`Max: "2.1.263"`) | `rein resume claude:<id> --dry-run --json` | **PASS (fail-closed, as expected)** — exit `5`. `agent.version` check: `status:"unknown"`, `severity:"block"`, `actual:"2.1.265"`, `message:"native agent version 2.1.265 is outside the verified range 2.1.219 to 2.1.263 inclusive"` |
| Dry-run launch plan, **after** this record's code change (`Max: "2.1.265"`) | same command | **PASS** — exit `0`, `decision:"confirmation_required"` (only the same first-launch `baseline.unavailable` warning the original `2.1.263` record saw; no block). `agent.version` check: `status:"match"`, `severity:"info"`, `actual:"2.1.265"`, `message:"the native agent version is in the verified range"`. Launch plan: `executable:"claude"`, `args:["--resume","866010d7-d6de-4060-baad-b2e61cabbaad"]`, `cwd` = the session's own workspace |
| Fork dry-run | `rein resume claude:<id> --fork --dry-run --json` | **PASS** — `args:["--resume","866010d7-d6de-4060-baad-b2e61cabbaad","--fork-session"]` |
| Non-`--dry-run` JSON guard | `rein resume claude:<id> --json` (no `--dry-run`) | **PASS** — refused `--json requires --dry-run for native agent launches`, exit `2`, unchanged from the original record |
| Non-interactive terminal guard | `rein resume claude:<id> --allow-environment-warning baseline.unavailable` (no `--dry-run`, stdin/stdout redirected, no TTY) | **PASS** — printed the full environment report (`agent.version` already `match`/`2.1.265`), then refused `native agent resume/fork requires an interactive terminal: re-run from a real TTY or use --dry-run for non-interactive inspection`, exit `7` |
| **Real interactive launch, through a real console** | `scripts\testing\conptydriver\conptydriver.exe` (built from this branch), driven via PowerShell `Start-Process -WindowStyle Hidden -Wait` (never redirecting the driver's own stdio, per this task's ground rules and the driver's own documented trap), against `rein.exe resume claude:<id> --allow-environment-warning baseline.unavailable` | **PASS** — `rein` built and ran the real launch plan (`claude --resume 866010d7-d6de-4060-baad-b2e61cabbaad`, the identical argv the dry-run plan named), attached to a real ConPTY. First run: the folder-trust prompt appeared (`Accessing workspace: D:\ReinstateAcceptanceProjects\rc8-claude ...`); `key down` + `key enter` selected "Yes, I trust this folder". Claude Code then rendered the **resumed session's own history** — the original prompt (`Reply with exactly this token and nothing else: <token>`) and the original answer (`<token>`) — proving `rein resume`'s real launch plan reached a real, restored Claude Code session, not a fresh one. A live `claude.ai/code/session_...` URL appeared in the raw VT stream during this run, confirming a real session against the host's own account |
| **Physical resume to a fresh, completed turn** | `claude --resume 866010d7-d6de-4060-baad-b2e61cabbaad -p "What token did you reply with earlier in this conversation? Reply with only the token, nothing else." --output-format json` — the dry-run plan's exact `--resume <id>` argument, plus the vendor's own non-interactive completion flag, the documented fallback this series' own first record (`2026-09-06-windows-range-widening-v060.md`, T-301) used when a full interactive round trip is not backed by a ConPTY-scripted send/wait | **PASS** — same `session_id` (`866010d7-d6de-4060-baad-b2e61cabbaad`); `"result":"<token>"` — the exact token, which existed nowhere but this session's own first turn. (The interactive ConPTY round trip above reached the resumed chat prompt and a message was sent and echoed there too, but the vendor's response consistently exceeded this record's ConPTY wait budgets under interactive rendering overhead unrelated to Reinstate — recorded as an open harness observation in §5, not claimed as failed evidence either way; the `-p` fallback is what this record's recall claim rests on, exactly as the original T-301 record's own evidentiary bar allows) |
| Session file layout | direct filesystem check | **PASS, unchanged** — `<CLAUDE_CONFIG_DIR>\projects\D--ReinstateAcceptanceProjects-rc8-claude\866010d7-d6de-4060-baad-b2e61cabbaad.jsonl`, matching `docs/session-storage/claude.md`'s documented `<root>/projects/<project-key>/<session-uuid>.jsonl` shape exactly |
| First-line JSON key names (names only, no values) | direct read of the freshly `2.1.265`-authored session file | Line 0 keys: `type`, `aiTitle`, `sessionId` (record `type:"ai-title"`). Every one of the file's 25 lines carries a `sessionId` key, and every `user`/`assistant`/`attachment` line additionally carries `cwd`, `version`, `gitBranch`, `uuid`, `timestamp` — the fields `docs/session-storage/claude.md`'s contract already documents (file stem is the session UUID; events carry `sessionId`). No documented contract asserts a specific first-line record *type*, so an `ai-title` line first is new information, not a contract change; the adapter does not require or special-case a first-line type either, which is exactly why the `rein search` row above resolved this file correctly without any code change |
| `rein doctor --agents --json`, Claude row | isolated `REINSTATE_HOME`, real `CLAUDE_CONFIG_DIR` | `"key":"claude"`, `"executable_on_path":true`, `"version_raw":"2.1.265 (Claude Code)"`; `name_shapes` reports `projects/*` → `<slug>` (3 samples), `projects/*/*` → `<uuid-v4>.jsonl` (11 samples) — the same `projects/<project-key>/<session-uuid>.jsonl` shape, confirmed unchanged under the live, `2.1.265`-written tree. No id, path, or file content appears in this output; it is Reinstate's own sanitized shape probe |

## 4. What changed between `2.1.263` and `2.1.265`, and what did not

Unchanged, confirmed against the real `2.1.265` binary and a real,
completed round trip: the `claude --version` output shape (bare
`<semver> (Claude Code)`, parsed by the existing `claudeVersionPattern`
regex with no change needed); the native resume/fork argv
(`--resume <id>`, `--resume <id> --fork-session`) `rein`'s launch plan
builds and the real binary accepts; the session file layout and naming
(`projects/<project-key>/<session-uuid>.jsonl`); the documented per-event
fields (`sessionId`, `cwd`, `version`, `gitBranch`, `uuid`, `timestamp`);
and, end to end, the whole version-gate → launch-plan → real-vendor-launch
→ restored-history pipeline, confirmed to flip from a correct fail-closed
refusal (exit `5`) to a correct, fully-built plan (exit `0`) and then to a
real, working resumed session across the code change in this record.

Newly observed, not newly broken: the session file's first record is now
`type:"ai-title"` rather than a `type:"meta"`/`user` record, and several
other record types (`queue-operation`, `attachment`, `atis-latch`,
`last-prompt`) appear that the existing fixtures do not model individually.
None of this breaks anything Reinstate reads: the `rein search` row in §3
found and indexed this exact, freshly `2.1.265`-authored file correctly on
the first try, with no code change, because the adapter tokenizes complete
JSONL records generically rather than special-casing a first-line shape.

## 5. Open harness observation — interactive ConPTY round trip, response timing

`rein resume claude:<id> --allow-environment-warning baseline.unavailable`,
driven through `conptydriver`, reliably reached the resumed chat prompt
(§3) and reliably delivered a typed follow-up message to it (echoed
on-screen, with a live `claude.ai/code/session_...` link visible in the
raw VT stream — proof the message reached the real, authenticated
session). Waiting for the response's own completion footer inside the
same ConPTY-scripted run, however, did not complete within this record's
wait budgets (tried up to 150s) across several attempts, while the
identical question against the identical session via the vendor's own
`-p` non-interactive flag (§3's physical-resume row) answered in about two
and a half seconds. No Reinstate process or code path is on the call stack
at that point — `rein` had already hand ed off to the real `claude.exe`
attached to the pseudo console — so this is recorded as an open harness
observation about interactive rendering under a scripted ConPTY session on
this host, not a Reinstate product defect, and is not claimed as evidence
either way. It does not weaken the recall claim in §3, which rests on the
`-p` fallback exactly as this series' first record allows.

## 6. Range declaration moved

| Location | Old | New |
| -------- | --- | --- |
| `internal/adapter/claude/claude.go` (`maximumVerifiedClaudeVersion`) | `2.1.263` | `2.1.265` |
| `internal/agentcheck/agent.go` (`testFallbackDefinitions()["claude"].Max`) | `2.1.263` | `2.1.265` |
| `internal/agents/catalog/claude.go` (`VersionSpec.Max`) | `2.1.263` | `2.1.265` |

Tests moved with the ceiling:

- `internal/adapter/claude/claude_test.go` `TestClaudeSupportedVersionRange`
  — `2.1.263` stays `true`; the boundary case moves from `2.1.264`/`false`
  to `2.1.264`/`true` and `2.1.265`/`true` (new ceiling), adding
  `2.1.266`/`false` as the new one-past-ceiling boundary.
- `internal/agentcheck/version_evidence_test.go`
  `TestOutOfRangeVersionNamesTheRange` — the refusal message's named range
  moves from `2.1.219`/`2.1.263` to `2.1.219`/`2.1.265`.
- `internal/agents/catalog/catalog_test.go`
  `TestShippedAgentsRegisterAtDeclaredTiers` — the Claude row's expected
  max moves to `2.1.265`; `TestVersionParsersMatchAgentcheckShape` adds a
  `2.1.265 (Claude Code)` parse case.
- `internal/cli/handoff_claude_source_test.go`
  `TestHandoffFromClaudeInstallJustOutsideVerifiedRange` — the fake
  `claude --version` output moves from `2.1.264` (now inside the widened
  range) to `2.1.266`, the new one-past-ceiling boundary; the test's own
  assertion still expects the compatibility refusal at that boundary.
- `internal/transcript/claude_test.go`
  `TestClaudeProbeVersionGate` — the `"outside range"` case's fixed
  version moves from `2.1.264` to `2.1.266` for the same reason.

Fail-closed boundary: no build between `2.1.264` and `2.1.265` inclusive
other than the two host-installed builds this and the prior record
measured, and no build past `2.1.265`, was physically measured on this
host, so the interior values carry no independent physical evidence beyond
the two real installs (`2.1.263` from the prior record, `2.1.265` here) —
the same allowance the prior widening records in this series used for
their own untested interior and one-past-ceiling versions.

## 7. Gates

| Gate | Command | Result |
| ---- | ------- | ------ |
| Format | `gofmt -l .` | see the run recorded with this commit |
| Vet | `GOTOOLCHAIN=go1.25.13 go vet ./...` | see the run recorded with this commit |
| Tidy | `GOTOOLCHAIN=go1.25.13 go mod tidy -diff` | see the run recorded with this commit |
| Unit suite | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test -p 4 ./... -count=1` | see the run recorded with this commit — `internal/handoff`'s `TestPlanPartialFinalRecordByteExactCrossCheck/claude-windows` and `TestPlanPartialFinalRecordRefusalIsOSShapeIndependent/claude-windows` are the two tests this record exists to turn green again |
| Lint | `make lint` | see the run recorded with this commit |
| Doc gate | `go test ./internal/doctest/... -count=1` | see the run recorded with this commit |
| Doc-link script | `bash scripts/check-docs.sh` | see the run recorded with this commit |
| Website unit tests | `cd website && npx vitest run` | see the run recorded with this commit |
| Acceptance-matrix row count | `rein doctor --agents --acceptance-matrix --json` (`"row_count"`) | still `178` — widening `Min`/`Max` on an already-registered agent changes no tier, family, or agent count |

## 8. Verdict

The verified range is widened to `2.1.219`–`2.1.265` on native Windows
evidence, under the maintainer's standing self-update policy (2026-09-07,
Q27), with a **completed-turn** recall proof: a session created with the
installed `2.1.265` build was found by `rein search` on its own planted
token, resumed through the real launch plan `rein resume` produces (both
as a dry-run and, via `conptydriver`, as a real interactive launch that
restored the session's own prior history in a real console), and answered
the recall question with the exact token that existed only in that
session's own first turn. Version-output parsing, native resume/fork argv,
and the session file layout and documented per-event fields are all
confirmed unchanged from `2.1.263`. Apple Silicon macOS evidence is
deferred (ADR 0005 D3, tracked under the same `#403` every other `v0.6.0`
native-Windows-only widening in this release cites).
