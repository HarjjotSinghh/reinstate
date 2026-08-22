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

## Write-ahead journalling in the embedded store (measured, 2026-08-22)

OpenCode `1.18.21` keeps its sessions in an embedded SQLite database and
journals in write-ahead mode. It does **not** checkpoint on exit — measured
after quitting the TUI through its own UI and, separately, after `SIGINT`.

Immediately after a session was created in a throwaway root:

| File | Size |
| ---- | ---- |
| `opencode.db` | 4096 bytes, no `session` table at all |
| `opencode.db-wal` | 543872 bytes |

Everything the user had just done was in the log. Recent commits stay there
until SQLite's automatic checkpoint threshold — 1000 pages, roughly 4 MB of
writes — is crossed by later vendor activity, which on a lightly used install
can be a long time.

This matters because the obvious way to read a vendor store safely is the wrong
one here. Opening with `immutable=1` promises SQLite the file cannot change,
which is what stops it taking a lock or creating `-wal`/`-shm` sidecars under
the agent's root — and, as a direct consequence, makes it ignore the log. A
reader built that way reports the sessions the user most recently worked in as
absent, and on a new install reports the store as empty.

Reinstate therefore reads through `internal/vendorsqlite`, which copies the
database and its log into a private directory and opens the copy. Nothing is
written beside the vendor's database, and the log's contents are visible. A
store with no log is still read in place, immutable, with no copy at all.

Copying a live log is safe in a specific, bounded way. SQLite checksums every
frame and stops recovery at the first invalid one, so a copy taken while the
vendor is writing degrades to the last complete commit rather than to a corrupt
database. Truncating a copied log at nine points from 100% down to 0% left
`PRAGMA integrity_check` reporting `ok` every time, with the visible session
count falling monotonically. The one hazard that is not self-correcting is a
checkpoint landing between the two files being copied, so the database is
stat-ed before and after and a change retries; exhausting the retries falls back
to the in-place read and reports the listing as incomplete rather than
presenting a short list as the whole store.
