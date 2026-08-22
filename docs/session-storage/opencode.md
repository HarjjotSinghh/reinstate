# OpenCode

**Confidence: Verified (embedded store)** — the shipped index is
`internal/agents/sources/opencode/sqlite.go`, which reads the embedded SQLite
store read-only and immutable. The vendor CLI is not invoked to list sessions:
`opencode session list` answers only for the directory it runs in, and running
it opened the store read-write and left `-wal` and `-shm` files under the agent
root.
**Documented** for the historical on-disk `storage/message/` layout and the
Windows root (R1, 2026-08-12).

| Aspect | Value |
| ------ | ----- |
| Storage family | F3, embedded SQLite session store |
| Layout id | `embedded-sqlite-session-store` |
| Marker | `opencode.db` — a regular file, not a directory |
| Storage root (Linux/macOS) | `~/.local/share/opencode` |
| Storage root (Windows) | `%USERPROFILE%\.local\share\opencode` (same XDG layout; **not** `%LOCALAPPDATA%`) |
| Env override | `XDG_DATA_HOME`, which names the **parent**; the root is `$XDG_DATA_HOME/opencode` |
| Tables read | `session`, `project`, and whichever of `message` / `session_message` exists |
| Tables never opened | `credential`, `account`, `control_account`, `account_state` |
| Project key | opaque 40-hex vendor id; never used as a display name |
| Session ID shape | `ses_…` |

## Verified resume (T3)

Argv below is quoted from `opencode --help` as printed by the installed binary,
measured on macOS from OpenCode `1.18.21`:

```text
  -c, --continue      continue the last session
  -s, --session       session id to continue
      --fork          fork the session when continuing (use with --continue or --session)
```

OpenCode has **no `resume` or `fork` verb**. Continuation is an option on the
default command, and `--fork` is a modifier on `--session` or `--continue`. The
default command's only positional is a project path, so an argv shaped like
another vendor's `<verb> <id>` would silently start a new session in a directory
named after the session id.

| Action | Argv |
| ------ | ---- |
| Resume | `opencode --session <id>` |
| Fork | `opencode --session <id> --fork` |
| Continue newest | `opencode --continue` |

`opencode --version` prints a bare `MAJOR.MINOR.PATCH` on stdout with an empty
stderr, and answers unchanged under the sanitized probe environment. The
verified range is the single build physically measured, and widens only when
another build is measured on a device.

A session row the vendor recorded without a working directory stays read-only:
OpenCode is launched into a directory, so such a row has nowhere to go.

Device journey:
[../testing/results/2026-08-22-macos-opencode-t3-journey.md](../testing/results/2026-08-22-macos-opencode-t3-journey.md)
(macOS only; native Windows pending).

## Handoff destination (T4)

`rein handoff <session> --to opencode` starts a **new** OpenCode session. It is
never a cross-agent resume.

| Aspect | Value |
| ------ | ----- |
| New session with an initial prompt | `opencode --prompt "<briefing>"`, run in the verified workspace |
| Pinned session id | **not possible** — `opencode --session <unknown-id>` refuses with `Session not found` and creates nothing |
| Vendor files written by Reinstate | none; the vendor creates the session, its first message and its parts |
| Directory-trust pre-acceptance | not needed; the TUI starts straight into a fresh directory |
| Session id reconciliation | after launch, by verified workspace + the session's own `time_created` + the SHA-256 of its first human turn |

### Reconciliation is usually `unresolved`

OpenCode journals in SQLite WAL mode and does not checkpoint on exit. Measured
on macOS from OpenCode `1.18.21`: after a session was created and the TUI quit
through its own UI, `opencode.db` was 4096 bytes with no `session` table and
543 KB sat in `opencode.db-wal`.

The `immutable=1` guard that stops Reinstate creating files beside a store it
does not own is also what makes those rows invisible. A just-created destination
session is therefore normally recorded as `unresolved`. The handoff still
happens — only the recorded destination session id is unknown.

The same limitation applies well before any handoff: an OpenCode session written
recently may not appear in `rein sessions` until the vendor's own later writes
cross SQLite's automatic checkpoint threshold.

Device journey:
[../testing/results/2026-08-22-macos-opencode-t4-journey.md](../testing/results/2026-08-22-macos-opencode-t4-journey.md)
(macOS only; native Windows pending).

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
