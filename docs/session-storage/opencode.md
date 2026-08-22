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
