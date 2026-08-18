# Claude Code

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
[research/2026-08-12-phase-4-r5-claude-session-id-collision.md](../research/2026-08-12-phase-4-r5-claude-session-id-collision.md).

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
[research/2026-08-12-phase-4-r7-context-ceilings.md](../research/2026-08-12-phase-4-r7-context-ceilings.md).

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
