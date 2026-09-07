# Verified-range widening — Qwen Code, native Windows x64, 2026-09-07

`AGENT-TIER-JOURNEY-V1` · one agent, single platform.

Reproduces the method `v0.6.0-rc.1` used to widen the Claude Code and
OpenCode ranges
([`2026-09-06-windows-range-widening-v060.md`](2026-09-06-windows-range-widening-v060.md)),
under [ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md)
D3, for the one agent that record did not cover.

**macOS evidence: pending (ADR 0005).**

## 0. Why this record exists

The acceptance host's Qwen Code installation self-updated past the verified
range: `qwen --version` now prints `0.23.0`, while the range in tree was
`0.21.12`–`0.21.13` (`internal/agents/catalog/qwen.go`). Every Qwen
resume/fork/handoff row refused with exit `5` (`agent.version` block,
`native agent version 0.23.0 is outside the verified range 0.21.12 to 0.21.13
inclusive`) — correct, fail-closed behavior, and the reason for this record.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-07 |
| Device | `windows-amd64`, native — not WSL |
| OS | Microsoft Windows 11 Pro, 10.0.26200 |
| Branch | `v060/rc4` |
| Branch tip at time of this record | `202157c7877d33105dec700604dd23893b4d8b51` |
| Go toolchain | `go1.25.13` (`GOTOOLCHAIN=go1.25.13`) |
| Qwen Code installed | `0.23.0` (managed self-update, default root); `0.21.12` (bundled npm global install, `@qwen-code/qwen-code`) |
| Lab root | `<lab-project>` (`D:\ReinstateAcceptanceProjects\rc4-qwen\`, a real, throwaway git repository) |

Qwen was run as the real installed vendor code both ways: through the
`qwen` binary on `PATH` (answers `0.21.12` once `QWEN_HOME` is redirected,
since the redirected root has no self-update tree of its own) and, to reach
the actually-installed `0.23.0` build under a redirected root, by invoking
that build's own `cli-entry.js` directly out of the default root's
`updates/npm/<install-id>/versions/0.23.0/` tree — application code, not
operator session data — with `QWEN_HOME` still pointed at the isolated root
below. Every session file this record produced lives under that isolated
root. No real `~/.qwen` tree was read and no real transcript is quoted
anywhere below; the one token named is a marker planted by this record for
the sole purpose of proving continuation, redacted as `<token>`.

## 2. Isolation

| Redirect | Value |
| -------- | ----- |
| `QWEN_HOME` | an isolated, throwaway directory, seeded with only `settings.json` copied from the host's real `~/.qwen` (the file the vendor docs say may hold API keys under `env`; no other file was copied) |
| `REINSTATE_HOME` | a separate isolated, throwaway directory |
| Workspace | `<lab-project>`, a real Git repository containing only a README |

Isolation was verified before any session was created: `qwen -p "Reply
PONG"` against the isolated `QWEN_HOME` answered `PONG`, exit `0`, and the
vendor's own project-bucket sanitizer resolved the workspace to
`d--reinstateacceptanceprojects-rc4-qwen` (lower-cased, non-alphanumeric
bytes replaced with `-`) under that isolated root, confirming the redirect
took effect before any credential-bearing turn ran.

## 3. Rows exercised

| Step | Command (argv) | Result |
| ---- | --------------- | ------ |
| Isolation check | `qwen -p "Reply PONG"` (`QWEN_HOME` = isolated root, `<lab-project>` cwd, stdin redirected from empty) | PASS — `PONG`, exit `0` |
| Create session (0.23.0) | `qwen -p "Reply with exactly this text and nothing else: <token>"` (0.23.0 build, `QWEN_HOME` = isolated root) | PASS — real session created, vendor returned `<token>` |
| Session file location and naming | — | PASS — `projects/d--reinstateacceptanceprojects-rc4-qwen/chats/<uuid-v4>.jsonl`, unchanged from the `0.21.13` layout `docs/session-storage/qwen.md` documents |
| First-user-message shape | — | PASS — first record's keys are `uuid, parentUuid, sessionId, timestamp, type, provenance, cwd, version, gitBranch, message{role,parts}`, matching the `0.21.13` record shape; `version` reads `0.23.0` |
| Find it | `rein sessions --agent qwen --json` | PASS — the session listed, `size_bytes`/`message_count` computed |
| Find it and inspect before the code change | `rein inspect qwen:<id> --json` | PASS — `agent.version.status=unknown/untested`, `actual=0.23.0`, message names range `0.21.12 to 0.21.13 inclusive`, `decision=blocked`, `block_exit_code=5` |
| Widen the ceiling | `Max` (`internal/agents/catalog/qwen.go`) `0.21.13` → `0.23.0`; rebuild | PASS — `gofmt`/`go vet`/`go build` clean |
| Inspect after the code change | `rein inspect qwen:<id> --json` | PASS — `agent.version.status=match`, `actual=0.23.0`, `message="the native agent version is in the verified range"` |
| Dry-run launch plan | `rein resume qwen:<id> --dry-run --json` | PASS — plan `qwen --resume <id>`, `cwd` = the session's own workspace; `decision=confirmation_required` (first-launch baseline warning only; no block) |
| Non-`--dry-run` guard | `rein resume qwen:<id> --json` (no `--dry-run`) | PASS — refused `--json requires --dry-run for native agent launches`, exit `2`, identical to the Claude Code and OpenCode rows in the 2026-09-06 record |
| Fork plan | `rein resume qwen:<id> --fork --dry-run --json` | PASS — plan `qwen --resume <id> --fork-session` |
| **Physical resume** | `qwen --resume <id> -p "What token did you reply with earlier in this conversation? Reply with only the token, nothing else."` (0.23.0 build, `QWEN_HOME` = isolated root) — the dry-run plan's exact argv, plus the vendor's own non-interactive completion flag (`-p`) | **PASS** — result was exactly `<token>`, which exists nowhere but the original session's first turn |
| **Physical fork** | `qwen --resume <id> --fork-session -p "Reply OK"` (0.23.0 build) — the dry-run plan's exact argv | **PASS** — a new `chats/<uuid>.jsonl` appeared, its first record carries `"forkedFrom":{"sessionId":"<id>","messageUuid":"<uuid>"}`; the original session file was unmodified by the fork itself (only the earlier physical-resume row had appended to it) |
| New session at a chosen id (T4 handoff form) | `qwen --session-id <uuid>` (0.23.0 build) | PASS — created `chats/<uuid>.jsonl` at exactly that id |
| Id-collision refusal | `qwen --session-id <uuid>` again, same id (0.23.0 build) | PASS — `Error: Session Id <uuid> already exists (active or archived). Delete or unarchive it first.` |
| Non-UUID id refusal | not re-exercised this record | documented behavior unchanged from `0.21.13`; not re-run |
| Session listing | `qwen sessions list --json` (0.23.0 build) | PASS — one JSON object per line, `sessionId`/`filePath`/`cwd`/`prompt` present, unchanged shape |
| Version-output parsing | `qwen --version` (0.23.0 build) | PASS — bare line `0.23.0`, matches the existing `qwenVersionPattern` regex unmodified |

## 4. What changed between `0.21.13` and `0.23.0`, and what did not

Unchanged: session file location and naming, project-bucket sanitizing,
first-user-message shape, `--resume`, `--resume … --fork-session` and its
`forkedFrom` marker, `--session-id` (new-session and id-collision refusal),
`qwen sessions list --json`, and the bare-semver `--version` line the
existing regex already parses.

Changed, and evaluated for impact:

- `assistant` records gained a `contextWindowSize` field. `transcript.QwenReader`
  (`internal/transcript/qwen.go`) decodes each line with `json.Unmarshal`
  into a fixed struct with no `DisallowUnknownFields`, so an unrecognized
  field is silently ignored. No reader change needed.
- Each project bucket (`projects/<slug>/`) now also holds `meta.json` and
  `extract-cursor.json`, and a sibling `memory/` directory. None of these
  match the `projects/**/chats/*.jsonl` session glob
  (`internal/agents/sources/qwen.SessionGlob`), so they do not enter the
  index. No source change needed.
- No `<sessionId>.runtime.json` sidecar was observed anywhere under the
  isolated `QWEN_HOME` after the non-interactive (`-p`, `--resume`,
  `--fork-session`, `--session-id`) rows above. This is recorded as an open
  observation rather than a regression: the `0.21.13` sidecar evidence in
  `docs/session-storage/qwen.md` was also gathered without a non-interactive
  baseline to compare against, so there is no prior evidence that a
  non-interactive run ever wrote one either. `Excluded` already excludes
  `settings.json` and `.env`; the sidecar was never part of the session glob
  regardless of whether it appears.

## 5. Range declaration moved

| Location | Old | New |
| -------- | --- | --- |
| `internal/agents/catalog/qwen.go` (`VersionSpec.Max`) | `0.21.13` | `0.23.0` |

`internal/agentcheck/agent.go`'s `testFallbackDefinitions()` has no Qwen
entry (only `claude` and `codex`), so there is nothing to move there.

Tests moved with the ceiling:

- `internal/agents/catalog/qwen_test.go` `TestQwenVersionRangeSpansTheSelfUpdater`
  — range assertion moves to `0.21.12`–`0.23.0`; `0.21.15` (previously the
  documented unverified boundary) now sits inside the widened range's
  endpoints and is asserted `SUPPORTED`; the fail-closed boundary moves to
  `0.23.1` (one past the new ceiling), asserted `UNTESTED`. `0.23.1` was not
  re-verified against the real vendor binary since it is not installed on
  this host — same allowance the 2026-09-06 record used for OpenCode's
  untested boundary.
- `internal/agents/catalog/qwen_test.go` `TestParseQwenVersion` — adds a
  `0.23.0` case.

## 6. Gates

| Gate | Command | Result |
| ---- | ------- | ------ |
| Format | `gofmt -l internal/agents/catalog/qwen.go internal/agents/catalog/qwen_test.go` | PASS — empty |
| Vet | `GOTOOLCHAIN=go1.25.13 go vet ./...` | PASS |
| Tidy | `GOTOOLCHAIN=go1.25.13 go mod tidy -diff` | PASS — empty |
| Unit suite | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test -p 4 ./... -count=1` | PASS on every package this task's files touch; `internal/doctest` shows one unrelated failure (a different in-progress task's link on `docs/session-storage/pi.md`, sharing this worktree) and none from this task's own paths |
| Doc gate | `go test ./internal/doctest/... -count=1` | See above — the Qwen-related links this record introduces resolve; the `pi.md` failure belongs to other in-flight work in this worktree |
| Lint | `make lint` | see command output at commit time |
| Doc-link script | `bash scripts/check-docs.sh` | see command output at commit time |
| Website unit tests | `cd website && npx vitest run` | see command output at commit time |

## 7. Verdict

The widening is supported on **native Windows only**. Apple Silicon macOS
evidence for `0.21.12`–`0.23.0` is deferred (tracked in `#403`, the same
macOS-deferred issue every other v0.6.0 native-Windows-only widening in this
release cites), consistent with ADR 0005 D3.
