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

---

## 3. Gemini CLI

**Confidence: Verified (read path)** — `internal/sessionindex/gemini.go`.
**Documented** for resume semantics.

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

`$rewindTo` truncates the message list back to a named message ID. Any Phase 4
reader must replay rewinds **before** emitting canonical events, otherwise the
capsule contains turns the user already discarded.

---

## 4. OpenCode

**Confidence: Verified (list path)** — `internal/sessionindex/opencode.go` uses
the supported `opencode session list --format json` command and deliberately
does not read private storage.
**Observed** for the on-disk layout below.

| Aspect | Value |
| ------ | ----- |
| Supported read API | `opencode session list --format json` |
| Storage root (Linux/macOS) | `~/.local/share/opencode/storage` |
| Storage root (Windows) | **Unverified** — likely `%LOCALAPPDATA%\opencode` or `%USERPROFILE%\.local\share\opencode` |
| Session index | `storage/session/<project-hash>/<session-id>.json` |
| Messages | `storage/message/<session-id>/msg_<message-id>.json` |
| Message parts | `storage/part/…` |
| Session diffs | `storage/session_diff/…` |
| Session ID shape | `ses_…` |
| Resume | `opencode run "<prompt>" --session <id>` / `--continue` (`-c`) |

**Phase 4 constraint:** the session list command returns metadata only — no
message bodies. Building a capsule **from** an OpenCode session therefore
requires reading `storage/message/<session-id>/` directly. That is a new,
private, undocumented surface. Treat it as a version-gated reader that fails
closed on an unrecognized layout, and keep the metadata-only path as the
fallback. Confirm the Windows root physically before shipping.

---

## 5. Grok Build CLI (xAI)

**Confidence: Documented** (vendor docs + changelog).
**No Reinstate reader exists yet.**

| Aspect | Value |
| ------ | ----- |
| Root (macOS/Linux) | `~/.grok` |
| Root (Windows) | `%USERPROFILE%\.grok` |
| Config | `<root>/config.toml` |
| Sessions | `<root>/sessions/` — auto-saved, keyed by working directory |
| Contents | Prompts, responses, tool calls, and file snapshots |
| Resume | `grok --resume <session-id>`, `grok --continue` |
| In-TUI picker | `/resume` lists recent sessions for the current workspace |
| Compaction | `/compact [context]` rewrites history in place |

### Required privacy warning

Grok Build CLI has a documented history (mid-2026) of transmitting repository
contents — including Git history and unredacted `.env` material — to xAI cloud
storage. Phase 4 must therefore:

1. surface an explicit warning before **any** handoff whose destination is
   Grok, naming the upload behavior and the redaction that was applied;
2. run capsule redaction unconditionally on the Grok path, never `--no-redact`;
   and
3. keep Grok out of the default target set until a target packet ships.

For v0.4.0-rc.1, Grok is a **source only**: you may hand off *from* Grok, and
Grok sessions appear in the local index. Grok is not a destination.

### Open research (blocking a Grok reader)

- Exact per-session filename and on-disk schema (`session-<timestamp>.json`
  is community-observed, not vendor-confirmed).
- How the working-directory key is encoded in the path.
- Whether `/compact` destroys pre-compaction turns or preserves them.
- Whether file snapshots are inline or content-addressed side files.

Resolve these against a real install on both macOS and Windows, then commit
synthetic fixtures. Never inspect a contributor's real `~/.grok` tree into the
repository.

---

## 6. Cross-OS summary

| Agent | macOS | Windows | WSL2 | Env override |
| ----- | ----- | ------- | ---- | ------------ |
| Claude Code | `~/.claude/projects/` | `%USERPROFILE%\.claude\projects\` | `~/.claude/projects/` | `CLAUDE_CONFIG_DIR` |
| Codex CLI | `~/.codex/sessions/` | `%USERPROFILE%\.codex\sessions\` | `~/.codex/sessions/` | `CODEX_HOME` |
| Gemini CLI | `~/.gemini/tmp/` | `%USERPROFILE%\.gemini\tmp\` | `~/.gemini/tmp/` | `GEMINI_CLI_HOME` |
| OpenCode | `~/.local/share/opencode/storage/` | **Unverified** | `~/.local/share/opencode/storage/` | none known |
| Grok Build | `~/.grok/sessions/` | `%USERPROFILE%\.grok\sessions\` | `~/.grok/sessions/` | none known |

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
- [Gemini CLI session management](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/session-management.md) · [checkpointing](https://google-gemini.github.io/gemini-cli/docs/cli/checkpointing.html)
- [OpenCode CLI](https://opencode.ai/docs/cli/) · [OpenCode troubleshooting/storage](https://opencode.ai/docs/troubleshooting/)
- [xAI Grok Build sessions](https://docs.x.ai/build/features/sessions) · [Grok Build changelog](https://x.ai/build/changelog)
- [Grok Build CLI wire-level analysis (privacy)](https://gist.github.com/cereblab/dc9a40bc26120f4540e4e09b75ffb547)
