# Grok Build CLI (xAI)

**Confidence: Documented** (vendor docs + source; R2/R3 resolved 2026-08-12).
A Reinstate transcript reader and index source ship; the native launch argv
below is measured but the physical resume journey is still pending.

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
| Native resume | `grok --resume <session-uuid>` |
| Native fork | `grok --resume <session-uuid> --fork-session` |
| Continue newest here | `grok --continue` |
| New session with pinned ID | `grok --session-id <uuid>` (valid UUID, must not already exist) |
| Initial prompt | Positional argument: `grok "<prompt>"` |
| Version probe | `grok --version` → `grok 1.0.5 (5115b46bc909)` on stdout |
| In-TUI picker | `/resume` lists recent sessions for the current workspace |
| Compaction | `/compact [context]` rewrites `chat_history.jsonl`; preserves request/checkpoint side files |

### `--resume` takes an ID **or a title** (why Reinstate pins the shape)

Measured from `grok --help` on Grok Build 1.0.5:

> `-r, --resume [<SESSION_ID_OR_TITLE>]` — Resume a session by ID or title, or
> the most recent if omitted. Non-ID values match session titles for the
> current directory (ignoring letter case; a sole renamed match wins among
> duplicates, otherwise ambiguity errors; UUID-shaped values always mean IDs).

A title is not a stable identifier and two sessions can share one, so a
non-UUID value in that position can address a session the operator never
selected. The catalog descriptor therefore declares
`NativeSpec.SessionIDPattern` as the 8-4-4-4-12 hex UUID shape, and:

1. an indexed Grok session whose recorded id is not UUID-shaped is marked
   `can_resume: false` with the reason
   `Grok Build session id is not a UUID; --resume would address it by title`;
   and
2. the argv builder refuses to substitute any value of another shape, so no
   route to a launch plan can put a title on the command line.

`--session-id` is the reverse direction: it creates a **new** conversation with
a caller-chosen UUID, and the vendor requires that the UUID not already exist
under the target session directory. It never resumes.

### Version range

`grok --version` prints one stdout line, `grok <semver>` with an optional
parenthesised build id, and nothing on stderr. The catalog pins the inclusive
range `1.0.5`–`1.0.5`, measured on the macOS acceptance host on 2026-08-22
(`grok 1.0.5 (5115b46bc909)`). The 2026-08-17 native Windows probe recorded
`0.2.101`; that build predates this measurement and its `--version` shape has
not been measured, so a Windows host still on `0.2.101` is reported `UNTESTED`
and refused with exit `5` until it is upgraded or a second build is physically
measured and the range widens.

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

Grok sessions appear in the local index and are handoff sources. Native resume
and fork use the vendor's own CLI against the vendor's own session, and
Reinstate writes nothing under `~/.grok`.

### Remaining omissions for a Grok reader

- Exact ACP envelope wrapping for every `updates.jsonl` line variant —
  treat unknown lines as opaque.
- Whether file snapshots are inline or content-addressed side files —
  still **omitted** (no confirmed vendor schema in this pass).

Synthetic fixtures: `testdata/sessionindex/grok/{macos,windows}/`.
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](../research/2026-08-12-phase-4-r1-r2-r3.md).

### Device evidence (2026-08-17, native Windows amd64)

Artifact:
[`2026-08-17-windows-grok.json`](../testing/results/agent-probes/2026-08-17-windows-grok.json)

`grok` 0.2.101. After excluding `bundled/`, `marketplace-cache/`, `bin/`,
`downloads/`, `docs/`, and `auth.json`, the walk reached `sessions/`: 32
session directories, `summary.json`, `chat_history.jsonl` (`content`,`type`),
`updates.jsonl` (`method`,`params`,`timestamp`), and `events.jsonl`. That
matches the shipped T2 reader. The first dump the same day never left the
installer trees and is not committed.

The tree still lists `mcp_credentials.json` (filename only). Exclude it on
the next catalog pass; do not open it.

### Native resume evidence status

The committed Grok device rows to date cover index, search, inspect and
handoff-source behaviour. The physical native-resume journey T3 requires —
real agent, real session, resumed, continuation observed, on macOS **and**
native Windows — is specified in
[testing/grok-native-resume-acceptance.md](../testing/grok-native-resume-acceptance.md)
and has not been recorded on either platform.
