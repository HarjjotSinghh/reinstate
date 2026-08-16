# OpenCode

**Confidence: Verified (list path)** — `internal/sessionindex/opencode.go` uses
the supported `opencode session list --format json` command and deliberately
does not read private storage.
**Documented** for on-disk `storage/message/` layout and Windows root (R1,
2026-08-12).

| Aspect | Value |
| ------ | ----- |
| Supported read API | `opencode session list --format json` |
| Storage root (Linux/macOS) | `~/.local/share/opencode` (`storage/` underneath) |
| Storage root (Windows) | `%USERPROFILE%\.local\share\opencode` (same XDG layout; **not** `%LOCALAPPDATA%`) |
| Env override | `XDG_DATA_HOME` (when set) |
| Session index | `storage/session/<project-id>/<session-id>.json` |
| Messages | `storage/message/<session-id>/<message-id>.json` (`msg…` ids) |
| Message parts | `storage/part/<message-id>/<part-id>.json` (`prt…` ids; body text lives here) |
| Session diffs | `storage/session_diff/…` |
| Session ID shape | `ses_…` |
| Resume | `opencode run "<prompt>" --session <id>` / `--continue` (`-c`) |

### Message record schema (R1 — Documented)

Each message file is a MessageV2 `Info` object discriminated on `role`
(`user` | `assistant`). Required user fields include `id`, `sessionID`,
`role`, `time.created`, `agent`, and `model.{providerID,modelID}`. Assistant
records add `parentID`, `modelID`, `providerID`, `mode`, `agent`,
`path.{cwd,root}`, `cost`, and `tokens`. Evidence: anomalyco/opencode
`packages/opencode/src/session/message-v2.ts` and
`packages/opencode/src/storage/storage.ts` (`Storage.write` →
`<data>/storage/<key…>.json`). Windows root evidence: vendor troubleshooting
docs + `packages/core/src/global.ts` + `xdg-basedir`.

**Phase 4 constraint:** the session list command returns metadata only — no
message bodies. Building a capsule **from** an OpenCode session therefore
requires reading `storage/message/<session-id>/` (plus parts) directly.
Treat it as a version-gated reader that fails closed on an unrecognized
layout. Newest OpenCode also keeps messages in SQLite
(`SessionMessageTable`); if `storage/message/` is absent, omit capsule body
rather than guessing a SQL schema.

Synthetic fixtures: `testdata/sessionindex/opencode/{macos,windows}/`.
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](../research/2026-08-12-phase-4-r1-r2-r3.md).
