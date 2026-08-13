# Local session storage map

**Status:** Phase 4 research input
**Last researched:** 2026-08-12
**Complements:** [cross-agent-continuation.md](cross-agent-continuation.md),
[Phase 4 plan](superpowers/plans/2026-08-12-phase-4-cross-agent-handoff-plan.md)

This is the ground truth Phase 4 parsing depends on: where each supported
coding agent keeps its local conversation state, on every operating system
Reinstate targets, and how confident we are about each row.

Nothing here is a support claim. A layout being documented does not mean
Reinstate parses it, and parsing it does not mean handoff is supported. See
[compatibility.md](compatibility.md) for support states.

## Confidence vocabulary

Every claim below carries one of these:

| Level | Meaning |
| ----- | ------- |
| **Verified** | Reinstate already reads this layout in shipped code, with fixtures |
| **Documented** | The vendor or its docs state this; Reinstate has not shipped a reader |
| **Observed** | Community/third-party evidence only; no vendor statement |
| **Unverified** | Assumed from a sibling platform; **must** be confirmed before a reader ships |

An `Unverified` row may not be turned into a `SUPPORTED` compatibility state.
It must first be confirmed on a physical device and captured as a synthetic
fixture under `testdata/`.

---

## 1. Claude Code

**Confidence: Verified** — `internal/sessionindex/claude.go`,
`internal/adapter/claude/claude.go`, fixtures in
`testdata/sessionindex/claude/{macos,windows,wsl}/`.

| Aspect | Value |
| ------ | ----- |
| Root (override) | `$CLAUDE_CONFIG_DIR` |
| Root (macOS/Linux) | `~/.claude`, fallback `~/.config/claude` |
| Root (Windows) | `%USERPROFILE%\.claude`, fallback `%USERPROFILE%\.config\claude` |
| Sessions | `<root>/projects/<project-key>/<session-uuid>.jsonl` |
| Format | JSON Lines, one event per line, appended live |
| Session identity | File stem is the session UUID; events also carry `sessionId` |
| Subagents | `<project-key>/subagents/` — **excluded** from the top-level list |
| Native resume | `claude --resume <session-id>` |
| Native fork | `claude --resume <session-id> --fork-session` |
| New session with pinned ID | `claude --session-id <uuid>` (UUID must be valid) |
| Initial prompt | Positional argument: `claude "<prompt>"` |

### `--session-id` collision policy (R5 — Unverified / fail closed)

Vendor docs used by Reinstate confirm that `claude --session-id <uuid>` pins a
new session ID, but **do not** state what happens when that UUID already exists
on disk or in the local index (resume vs append vs refuse vs overwrite).

Phase 4 therefore fails closed:

1. Allocate UUID v4 with `crypto/rand`.
2. Refuse any ID that collides with an **indexed** Claude session.
3. Regenerate up to 8 times; if all collide, escalate
   (`ErrClaudeSessionIDCollision`) and do not launch.
4. Never assume silent overwrite. Reinstate still writes **no** files under
   `~/.claude/projects` (ADR 0003).

Research note:
[research/2026-08-12-phase-4-r5-claude-session-id-collision.md](research/2026-08-12-phase-4-r5-claude-session-id-collision.md).

### Project key derivation

The `<project-key>` directory name is derived from the **absolute project path
on that device**, which is why it differs between macOS and Windows:

| Device | Project path | Directory key |
| ------ | ------------ | ------------- |
| macOS | `/Users/fixture-user/code/demo` | `-Users-fixture-user-code-demo` |
| Windows | `C:\Users\fixture-user\code\demo` | `C--Users-fixture-user-code-demo` |
| WSL2 | `/home/fixture-user/code/demo` | `-home-fixture-user-code-demo` |

Reinstate never reuses a source device's directory key as a destination path.
It records the canonical project ID and recomputes the key from the destination
device's `local_root`. Phase 4 inherits this rule unchanged.

### Fields Phase 4 relies on

Per-line, on the top-level object or its nested `message`:

- `type` — `user`, `assistant`, `summary`, `session_meta`, `metadata`, …
- `sessionId`, `cwd`, `gitBranch`, `timestamp`, `customTitle`, `isMeta`
- `message.role`, `message.content` — content is a string or a block array
- Blocks: `{type: "text"|"tool_use"|"tool_result"|"image", …}`
- `tool_use` blocks carry `id`, `name`, `input`; `tool_result` blocks carry
  `tool_use_id`, `content`, `is_error`

`isMeta: true` records are harness-injected and must not be treated as a
human-authored prompt.

### Context-window ceiling (R7 — Omitted)

No Claude Code **harness** token ceiling is published in the vendor docs
Reinstate trusts. Model context sizes are not treated as a CLI constant.
Capability-diff summaries emit `context_ceiling: omitted` with reason
`no_vendor_published_harness_token_ceiling`. See
[research/2026-08-12-phase-4-r7-context-ceilings.md](research/2026-08-12-phase-4-r7-context-ceilings.md).

### Attachments (R8)

Claude Code image blocks use the Anthropic Messages `image` shape. Two source
forms appear in project JSONL:

1. **Inline base64** — `source.type: "base64"` with `media_type` + `data`.
   Reinstate does not re-embed the bytes; the event is `omitted` with reason
   `attachment_unavailable`.
2. **Path references** — a local file path on the image block or
   `source` (`path` / `file` / `file_path`). When the file exists on disk, the
   event is `referenced` (sha256 + mime + size only; no absolute path). When
   the path is missing, it is `omitted` with `attachment_unavailable`.

---

## 2. OpenAI Codex CLI

**Confidence: Verified** — `internal/sessionindex/codex.go`,
`internal/adapter/codex/codex.go`, fixtures in
`testdata/sessionindex/codex/forks/`.

| Aspect | Value |
| ------ | ----- |
| Root (override) | `$CODEX_HOME` |
| Root (macOS/Linux) | `~/.codex`, fallback `~/.config/codex` |
| Root (Windows) | `%USERPROFILE%\.codex` |
| Sessions | `<root>/sessions/YYYY/MM/DD/rollout-<RFC3339-ish>-<uuid>.jsonl` |
| Format | JSON Lines "rollout" records |
| Session identity | Trailing UUID of the **filename** is authoritative |
| Native resume | `codex resume <session-id>`, `codex resume --last` |
| Native fork | `codex fork <session-id>` |
| Initial prompt | Positional argument: `codex "<prompt>"` |
| Non-interactive | `codex exec …`, `codex exec --last` |

### Initial-prompt argv ceiling (R6 — Documented / Unverified)

Codex accepts a new-session prompt as `codex "<bootstrap>"` (**Documented**).
No Codex-published Windows-specific argv byte ceiling was found in vendor docs
Reinstate trusts (**Unverified**). Destination handoffs therefore enforce
`TargetCapabilities.MaxArgvBytes`, defaulting to `DefaultMaxArgvBytes`
(24 KiB) from the Phase 4 architecture plan — a Reinstate conservative budget,
not a vendor constant. Over-budget plans fall back to a short bootstrap that
references `projection.md` only. See
[research/2026-08-12-phase-4-r6-codex-argv.md](research/2026-08-12-phase-4-r6-codex-argv.md).

### Context-window ceiling (R7 — Omitted)

No Codex CLI **harness** token ceiling is published in the vendor docs
Reinstate trusts. Capability-diff summaries omit it with the same R7 reason
as Claude Code. See
[research/2026-08-12-phase-4-r7-context-ceilings.md](research/2026-08-12-phase-4-r7-context-ceilings.md).

### Why the filename wins

Codex names every rollout file after **that file's own** session, including
forks — but a fork also replays the source's records, so its `session_meta` can
carry the source ID. Pinning identity to the filename keeps a fork addressable
and stops it collapsing into its parent. Phase 4 readers must reuse
`codexSessionIDFromFilename` semantics rather than trusting in-file IDs.

### Record shapes Phase 4 relies on

Two coexisting representations; readers must handle both and prefer the first:

1. `{"type":"event_msg","payload":{"type":"user_message"|"agent_message", …}}`
2. `{"type":"response_item","payload":{"type":"message","role":"user"|"assistant","content":[…]}}`

`session_meta` carries `payload.git.{branch,repository_url,commit_hash}` and
`payload.cwd`. Tool activity appears as typed response items with call IDs;
reasoning items may be opaque or encrypted and are **never** translated.

### Reasoning items (R4 — Documented)

Codex rollouts may include Responses API `reasoning` items under
`response_item`. Phase 4 classifies **all** of the following as
`portability: omitted` with reason `vendor_opaque_state` and never copies
payload bodies into capsule blocks:

1. `{"type":"response_item","payload":{"type":"reasoning","encrypted_content":…}}`
2. `{"type":"response_item","payload":{"type":"reasoning",…}}` (including
   summary-only or empty-summary forms)
3. Any other `response_item` payload that carries `encrypted_content` /
   `encryptedContent`

Visible assistant text remains on `event_msg`/`agent_message` (preferred) or
`response_item`/`message`/`role=assistant` when no `event_msg` exists.
Synthetic fixtures: `testdata/handoff/codex/reasoning-items/`.

---

## 3. Gemini CLI

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
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](research/2026-08-12-phase-4-r1-r2-r3.md).

---

## 4. OpenCode

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
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](research/2026-08-12-phase-4-r1-r2-r3.md).

---

## 5. Grok Build CLI (xAI)

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

For v0.4.0-rc.3, Grok is a **source only**: you may hand off *from* Grok, and
Grok sessions appear in the local index. Grok is not a destination.

### Remaining omissions for a Grok reader

- Exact ACP envelope wrapping for every `updates.jsonl` line variant —
  treat unknown lines as opaque.
- Whether file snapshots are inline or content-addressed side files —
  still **omitted** (no confirmed vendor schema in this pass).

Synthetic fixtures: `testdata/sessionindex/grok/{macos,windows}/`.
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](research/2026-08-12-phase-4-r1-r2-r3.md).

---

## 6. Cross-OS summary

| Agent | macOS | Windows | WSL2 | Env override |
| ----- | ----- | ------- | ---- | ------------ |
| Claude Code | `~/.claude/projects/` | `%USERPROFILE%\.claude\projects\` | `~/.claude/projects/` | `CLAUDE_CONFIG_DIR` |
| Codex CLI | `~/.codex/sessions/` | `%USERPROFILE%\.codex\sessions\` | `~/.codex/sessions/` | `CODEX_HOME` |
| Gemini CLI | `~/.gemini/tmp/` | `%USERPROFILE%\.gemini\tmp\` | `~/.gemini/tmp/` | `GEMINI_CLI_HOME` |
| OpenCode | `~/.local/share/opencode/storage/` | `%USERPROFILE%\.local\share\opencode\storage\` | `~/.local/share/opencode/storage/` | `XDG_DATA_HOME` |
| Grok Build | `~/.grok/sessions/` | `%USERPROFILE%\.grok\sessions\` | `~/.grok/sessions/` | `GROK_HOME` |

Native Windows and WSL2 are **different devices** with different agent trees.
Reinstate never treats one agent-state directory as shared between them.

## 7. Rules every Phase 4 reader must follow

1. **Never parse a partially appended record.** Take the boundary at the last
   complete line; record the byte offset and the SHA-256 of everything up to it.
2. **Never mutate a source.** Readers open read-only and never write, rename,
   truncate, or lock a vendor file.
3. **Never guess an unknown record.** Preserve it as an opaque, hashed source
   reference with `portability: referenced`.
4. **Fail closed on an unknown layout version.** An unrecognized shape produces
   a compatibility refusal (exit `5`), not a best-effort parse.
5. **Never use a contributor's real agent tree.** Fixtures under `testdata/`
   only; the fixture secret scanner gates every commit.
6. **Bound everything.** Reuse the existing `MaxJSONLineBytes`,
   `MaxSearchTextBytes`, and `MaxFileReferences` ceilings; add explicit capsule
   ceilings rather than inventing new unbounded reads.

## Sources

- [Claude Code sessions](https://code.claude.com/docs/en/sessions) · [headless mode](https://code.claude.com/docs/en/headless)
- [OpenAI Codex CLI](https://github.com/openai/codex) · [Codex session lifecycle](https://codex.danielvaughan.com/2026/06/05/codex-cli-session-lifecycle-archive-resume-fork-compact-management/)
- [Gemini CLI session management](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/session-management.md) · [checkpointing](https://google-gemini.github.io/gemini-cli/docs/cli/checkpointing.html) · [rewind](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/rewind.md) · [`chatRecordingService.ts`](https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/services/chatRecordingService.ts)
- [OpenCode CLI](https://opencode.ai/docs/cli/) · [OpenCode troubleshooting/storage](https://opencode.ai/docs/troubleshooting/) · [storage.ts](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/storage/storage.ts) · [message-v2.ts](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/session/message-v2.ts) · [global.ts](https://github.com/anomalyco/opencode/blob/dev/packages/core/src/global.ts)
- [xAI Grok Build sessions](https://docs.x.ai/build/features/sessions) · [Grok Build changelog](https://x.ai/build/changelog) · [17-sessions.md (source)](https://github.com/xai-org/grok-build/blob/main/crates/codegen/xai-grok-pager/docs/user-guide/17-sessions.md) · [`encode_cwd_dirname`](https://github.com/xai-org/grok-build/blob/main/crates/codegen/xai-grok-config/src/paths.rs)
- [Grok Build CLI wire-level analysis (privacy)](https://gist.github.com/cereblab/dc9a40bc26120f4540e4e09b75ffb547)
- Phase 4 R1–R3 research note: [research/2026-08-12-phase-4-r1-r2-r3.md](research/2026-08-12-phase-4-r1-r2-r3.md)
- Phase 4 R7 context ceilings: [research/2026-08-12-phase-4-r7-context-ceilings.md](research/2026-08-12-phase-4-r7-context-ceilings.md)
