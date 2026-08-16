# Gemini CLI

**Confidence: Verified (read path)** — `internal/sessionindex/gemini.go`.
**Documented** for resume semantics and `$rewindTo` on-disk behavior
(R3, 2026-08-12).

| Aspect | Value |
| ------ | ----- |
| Root (override) | `$GEMINI_CLI_HOME` |
| Root (all OSes) | `~/.gemini` / `%USERPROFILE%\.gemini` |
| Sessions | `<root>/tmp/<project-hash>/chats/session-<id>.json` or `.jsonl` |
| Checkpoints | `<root>/tmp/<project-hash>/checkpoints/checkpoint-<name>.json` |
| Format | Legacy: single JSON object with `messages[]`. Current: JSONL with `$set` metadata records and `$rewindTo` rewind records |
| Project scoping | `<project-hash>` is derived from the project root path |
| Subagents | `kind: "subagent"` sessions are excluded |
| Native resume | `gemini --resume` / `-r`; project-scoped |

### `$rewindTo` (R3 — Documented)

On-disk JSONL is **append-only**: prior message lines stay in the file; a
`{"$rewindTo":"<messageId>"}` record is appended. The **active** conversation
truncates (vendor removes from and including the target id). Phase 4 capsule
readers must replay rewinds **before** emitting canonical events, otherwise
the capsule contains turns the user already discarded.

`internal/transcript/gemini.go` (WP-08) aligns with that vendor cut: the
target id and everything after it are excluded from the capsule. The Phase 2
index reader still uses an inclusive slice for search metadata only.

Synthetic fixtures: `testdata/sessionindex/gemini/{macos,windows}/` and
`testdata/handoff/gemini/{rewind,legacy-json,jsonl}/`.
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](../research/2026-08-12-phase-4-r1-r2-r3.md).
