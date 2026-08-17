# Grok Build CLI (xAI)

**Confidence: Documented** (vendor docs + source; R2/R3 resolved 2026-08-12).
**No Reinstate reader exists yet.**

| Aspect | Value |
| ------ | ----- |
| Root (override) | `$GROK_HOME` |
| Root (macOS/Linux) | `~/.grok` |
| Root (Windows) | `%USERPROFILE%\.grok` |
| Config | `<root>/config.toml` |
| Sessions | `<root>/sessions/<encoded-cwd>/<session-uuid>/` (directory, not a single file) |
| Index entry | `summary.json` (`Info { id, cwd }` + counts/timestamps/model) |
| Authoritative log | `updates.jsonl` (append-only ACP/update stream) |
| Model-facing history | `chat_history.jsonl` (`ConversationItem` JSONL; `chat_format_version` 0 legacy / 1 current) |
| Compaction artifacts | `compaction_checkpoints/`, `compaction_requests/` |
| Resume | `grok --resume <session-id>`, `grok --continue` |
| In-TUI picker | `/resume` lists recent sessions for the current workspace |
| Compaction | `/compact [context]` rewrites `chat_history.jsonl`; preserves request/checkpoint side files |

### Workspace key encoding (R2 — Documented)

`encode_cwd_dirname` (`xai-grok-config/src/paths.rs`):

1. URL-encode the absolute working directory.
2. If ≤ 255 bytes → use that as the directory name.
3. If longer → `{slug}-{blake3_hex16}` and write the original path to `.cwd`.

### `/compact` (R3 — Documented)

Active `chat_history.jsonl` is **atomically replaced** (prior turns removed
from that file). Pre-compaction turns are **preserved** in
`compaction_requests/` (full request payload) and compaction markers are
**appended** to `updates.jsonl` pointing at `compaction_checkpoints/`.

### Required privacy warning

Grok Build CLI has a documented history (mid-2026) of transmitting repository
contents — including Git history and unredacted `.env` material — to xAI cloud
storage. Phase 4 must therefore:

1. surface an explicit warning before **any** handoff whose destination is
   Grok, naming the upload behavior and the redaction that was applied;
2. run capsule redaction unconditionally on the Grok path, never `--no-redact`;
   and
3. keep Grok out of the default target set until a target packet ships.

For v0.4.0, Grok is a **source only**: you may hand off *from* Grok, and
Grok sessions appear in the local index. Grok is not a destination.

### Remaining omissions for a Grok reader

- Exact ACP envelope wrapping for every `updates.jsonl` line variant —
  treat unknown lines as opaque.
- Whether file snapshots are inline or content-addressed side files —
  still **omitted** (no confirmed vendor schema in this pass).

Synthetic fixtures: `testdata/sessionindex/grok/{macos,windows}/`.
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](../research/2026-08-12-phase-4-r1-r2-r3.md).

### Device observation (2026-08-17, native Windows amd64)

Not committed as an artifact. `grok` 0.2.101 was on `PATH` and `%USERPROFILE%\.grok`
resolved, but the probe never reached `sessions/`. The walk spent its budget on
`bundled/` (installer payload, including 137 MB `bin/*.exe`) and
`marketplace-cache/` (cloned plugin git trees). Those directories, plus
`downloads/`, `docs/`, and `auth.json`, are now excluded. Re-probe with a binary
that includes those exclusions before treating Windows as a native artifact.
