This is a very good idea — and the pivot from "MCP-only" to "cloud sessions" is exactly right.

You described it simply as "store sessions in S3 and sync them" but the real insight is bigger: **your AI coding context is now more valuable than your code checkout**. Losing 8 hours of Claude/Codex reasoning hurts more than losing uncommitted files. Your Windows desktop + MacBook split is the norm now, not the edge case.

I've audited what's already out there. You should build this, but not as a generic backup tool.

### 1. Your pain is validated — everyone is building a slice of it

I found 15+ tools all circling the same problem:

**Config-only sync (what your original Dev Sync was):**
* `mcpsync`, `mcp-sync`, `skills-sync`, `agent-sync`, `agentry`, `vek-sync`, `sync-conf.dev` — all do: define MCP servers / skills once, generate vendor configs for each agent
* Example: `agent-sync` syncs MCP, skills, and AGENTS.md across Claude Code, Codex, Copilot, Gemini, Antigravity, OpenCode

**Session-only sync:**
* `claude-sync` — The closest to your idea for Claude-only. Syncs your entire `~/.claude` directory across devices using encrypted cloud storage to R2/S3/GCS, with `push` / `pull`
* `codexSync` — Local-first cold-sync handoff using any cloud folder (Dropbox, OneDrive, Syncthing). Detects if Codex is running, compares local vs cloud, backup-before-overwrite. Windows-to-Windows only validated, macOS not yet
* `claude-backup`, `agent-sessions` TUI, `aisessions.dev` uploader — search, backup, export

**Heavy platform approach:**
* **Omnigent (Databricks, Apache 2.0)** — Open-source meta-harness. Sessions follow you across terminal, browser, phone. Messages, sub-agents, terminals, files stay in sync across every surface. Architecture is runner wraps each agent + server manages policies and sharing. Requires you to run *their* server and use *their* wrappers, not native `claude` / `codex` commands.

No one is doing universal, lightweight, native-CLI session + config sync that works Windows <-> Mac.

### 2. What you'd actually need to sync — it's 4 layers

This is why config-only tools feel insufficient:

**Layer 1 — Identity / Config (your original Dev Sync):**
`~/.claude.json` global index, `~/.claude/settings.json`, `~/.claude/agents/`, `skills/`, `rules/`, `plugins/`, `CLAUDE.md`, `~/.codex/config.toml`, `~/.gemini/settings.json`, etc.

**Layer 2 — Sessions:**
* Claude Code: `~/.claude/projects/[project-name]/[session-id].jsonl` — one JSON object per line, every message + tool call. Officially `~/.claude/projects/<project>/<session-id>.jsonl` where project is cwd with non-alphanumeric replaced by `-`
* Codex: `~/.codex/sessions/{Y}/{M}/{D}/rollout-*.jsonl` and `~/.codex/archived_sessions/` plus `~/.codex/sessions/**/*.jsonl`
* Gemini CLI: `~/.gemini/tmp/[PROJECT_HASH]/chats/session-*.json` — automatically saved in background, project-specific
* OpenCode: Linux/macOS `~/.local/share/opencode/storage/`, Windows `%LOCALAPPDATA%/opencode/storage/`, SQLite `opencode.db` + snapshots

**Layer 3 — Runtime artifacts:** `~/.claude/session-env/{session-id}/`, OpenCode `snapshot/{projectId}/` bare git repos with trees+blobs.

**Layer 4 — Secrets:** MCP env vars with API keys. Can't sync plaintext.

### 3. The hard problems nobody is solving well

**a) Path canonicalization is the killer for your use case.**
Claude encodes `/Users/xxx/my-project` as `-Users-xxx-my-project`. Windows is `C:\Users\...`. If you just rsync `~/.claude/projects/` from Windows to Mac, resume will never find the project. You need to store canonical key = `git remote url + git branch + relative project hash`, then re-encode to local OS on restore.

**b) Session format is volatile.**
Anthropic docs explicitly say: "The entry format is internal to Claude Code and changes between versions, so scripts that parse these files directly can break on any release.". Don't build a parser. Treat JSONL as an opaque blob for v1, with a version field.

**c) Size and retention.**
A single afternoon can generate 50MB+ of session data. Claude deletes sessions after 30 days by default. Your tool is both a sync *and* an infinite backup — that's actually a feature.

**d) Conflict resolution.**
What if you edited same session on both devices offline? `codexSync` handles this with policies: `manual_abort`, `prefer_cloud`, `prefer_local`, `prefer_newer_mtime`. You need the same, plus backup-before-overwrite.

**e) Cross-OS MCP binaries.**
Your MCP config on Windows might be `command: "npx"` vs Mac `command: "/opt/homebrew/bin/npx"`, python paths differ. Need per-OS override layer.

### 4. Proposed architecture for Dev Sync v2

Don't build a hosted cloud on day 1. Copy what works: BYOB + E2E encryption.

**Storage:** Bring-your-own-bucket — Cloudflare R2, S3, GCS, or even a local folder (Dropbox/OneDrive/Syncthing like codexSync). This solves privacy and cost instantly. Claude Sync proves free tier is enough.

**Security:** `age` encryption with passphrase-derived key, same model as claude-sync. Secrets never leave device unencrypted.

**Sync Engine:**

1. `devsync init` -> creates `~/.config/devsync/config.json` with device ID (desktop-win, macbook-air), bucket creds, encryption passphrase.
2. Manifest: `manifest.json` in bucket stores `{file_path, hash, mtime, device, canonical_project_key}`.
3. Three-way diff: local hash vs last-synced hash vs remote hash. If both changed -> conflict -> backup both versions to `~/.devsync/backups/` and apply policy.
4. Path translator: `canonicalize(path) = gitRemote + ":" + relativePath`. On pull, `denormalize(canonical)` to local absolute + re-encode to agent's expected format.

**CLI v1 (2 weeks):**
```
devsync status # what would push/pull, drift
devsync push # encrypted upload
devsync pull # download + re-encode paths + backup
devsync doctor # check claude/codex/gemini installed, detect projects
```

**v2 (daemon):** File watcher with 5s debounce, auto-push. On terminal focus, auto-pull. Menu bar icon.

**v3 (cloud):** Optional hosted control plane for real-time: WebSocket relay so you can `devsync resume <session-id>` on laptop and continue exactly where desktop left off, no manual push/pull.

### 5. Positioning — don't compete with Omnigent

Omnigent is infrastructure: "run agents in cloud sandboxes via Modal, Daytona, E2B, Kubernetes", team collaboration, policy enforcement. It's heavy, enterprise.

Your wedge is the opposite: **personal, local-first, works with native CLIs you already use, zero migration**. Tagline: "Dotfiles for your AI brain."

Open source the core sync engine (MIT), charge for:
* Hosted encrypted vault (no R2 setup)
* Team shared skills/MCP library
* Infinite searchable session history (like `agent-sessions` TUI but cloud)

### Verdict: Should you build it?

Yes. But build it in this order:

**Week 1-2: Solve your own pain.** Claude Code + Codex CLI session sync Windows <-> Mac with correct path remapping. That's already better than `claude-sync` (Claude-only) and `codexSync` (Windows-only, not path-aware).

**Week 3: Add config layer.** MCP + skills sync on top — that's your original Dev Sync, now for free because you already have the engine.

**Week 4: Polish for release.** E2E encryption, conflict UI, `doctor`.

If it works for you for 2 weeks without you thinking about it, ship it. The market is fragmented, the official CLIs have no native sync (Gemini just added local auto-save but not cloud), and everyone with 2 devices feels this.

Want me to draft the data model for the manifest and the path canonicalization algorithm for Claude vs Codex vs Gemini?
