Short answer, GOAT: **yes, you should build this** – but not as “S3 for session logs”, as a **cross-device, cross-agent dev state layer**. The naive version already exists; the good version doesn’t.

***

## Reality check: what already exists

Most of the big CLIs already have **local** session persistence:

- Gemini CLI has full session management: automatic saving of prompts, responses, tool outputs, and activity logs, with a session browser (`/resume`), `--resume`, `--list-sessions`, and retention policies configured via `.gemini/settings.json`. All of this is stored under `~/.gemini/tmp/<project_hash>/chats/` on your machine, not synced anywhere. [geminicli](https://geminicli.com/docs/cli/session-management/)
- Codex CLI persists every session as JSONL transcripts in `~/.codex/sessions/YYYY/MM/DD/`, including prompts, responses, tool calls, and results, and gives you `codex resume`, `codex resume --last`, `/resume`, etc. Again, strictly local. [verdent](https://www.verdent.ai/guides/codex-cli-resume-continue-save-chat)
- Gemini’s dev blog explicitly markets this as “pick up exactly where you left off,” but the scope is one machine: automatic saving plus resume flags; no multi-device or cross-agent story. [developers.googleblog](https://developers.googleblog.com/pick-up-exactly-where-you-left-off-with-session-management-in-gemini-cli/)
- Claude Code has session primitives too and even a “Remote Control” feature that lets you continue a **local** session from phone, tablet, or browser by remotely driving that same machine; it’s basically remote desktop for a single host, not multi-device sync of state. [code.claude](https://code.claude.com/docs/en/remote-control)

On top of that, hackers have already started patching the gap you are describing:

- There is an OSS tool, Claude Sync, that literally syncs `~/.claude` (settings, projects, sessions) across devices using encrypted cloud storage (R2, S3, GCS, etc.). It positions itself as “sync your Claude Code sessions across all your devices”. [github](https://github.com/tawanorg/claude-sync)
- People are doing DIY setups with NAS + symlinks + Syncthing to share Claude settings/projects across multiple machines while explicitly keeping “session state” local because they’re nervous about corruption and privacy. [steeman](https://www.steeman.be/posts/syncing-claude-code-across-multiple-machines/)
- Codex users have built their own session browsers because the native resume flow is clunky for long history, but these are still single-machine tools over the local JSONL directory. [reddit](https://www.reddit.com/r/ChatGPTCoding/comments/1nvnzze/built_a_session_browser_for_codex_cli_because/)
- There is even an open feature request for account-level settings sync in Claude Code, because today everything lives in `~/.claude/` with no official sync mechanism. [github](https://github.com/anthropics/claude-code/issues/22648)

So: **“sync CLI sessions via cloud” is not a unique idea anymore.** You’ll be competing with vendor-native features plus niche OSS like Claude Sync. [dev](https://dev.to/tawanorg/claude-sync-sync-your-claude-code-sessions-across-all-your-devices-simplified-49bl)

***

## Why your idea still has teeth

Where your idea **does stand out**:

1. **Cross-agent, not just Claude**
   - Claude Sync is Claude-only; Gemini’s session management is Gemini-only; Codex’ JSONL is Codex-only. [geminicli](https://geminicli.com/docs/cli/session-management/)
   - You live in a world with Claude Code, Codex CLI, Gemini CLI, Grok Code/Build, Antigravity, OpenCode, etc. and use multiple on both MacBook and Windows in the same day. [perplexity](https://www.perplexity.ai/search/29593b04-6458-48d6-a5e8-e2773f1a73d4)
   - A **neutral, agent-agnostic layer** that ingests and normalizes state from all of them is not really a thing yet.

2. **Environment + config parity, not just chat logs**
   - Your original DevSync idea was already about syncing MCP configuration across tools (central MCP config that fans out to Claude, Codex, Gemini, etc.). [geminicli](https://geminicli.com/docs/tools/mcp-server/)
   - Current tools either:
     - Store config locally (`~/.claude`, `.gemini/settings.json`, `.codex/config.toml`) with no cloud sync. [github](https://github.com/openai/codex/blob/main/docs/config.md)
     - Or rely on you wiring NAS/Syncthing hacks. [steeman](https://www.steeman.be/posts/syncing-claude-code-across-multiple-machines/)
   - Having **one source of truth** for MCP servers, skills, hooks, and agent settings, and then compiling it per-tool is a genuine DX upgrade over the current mess. [code.claude](https://code.claude.com/docs/en/plugins-reference)

3. **Project handoff vs. raw transcript**
   - Codex and Gemini can replay transcripts to reconstruct context, but they don’t give a portable “here’s the current task, decisions, changed files, and next steps” artifact for cross-agent handoff. [developers.googleblog](https://developers.googleblog.com/pick-up-exactly-where-you-left-off-with-session-management-in-gemini-cli/)
   - For your use case (20 sessions across 4 projects, then moving to bed), what you actually want is:
     - A handoff summary (what you were doing, what changed).
     - A pointer to relevant files/diffs.
     - A reproducible environment manifest (tools, versions, MCPs).
   - That high-level “context capsule” is missing in the landscape; everything today is either raw transcripts or full-code reread.

4. **Multi-device reality**
   - Claude Remote Control solves “drive the same machine from another screen”; it does not solve “I shut down the desktop, now I’m on the laptop, let me continue with native performance and local paths.” [code.claude](https://code.claude.com/docs/en/remote-control)
   - Gemini/Codex sessions are bound to local directories and local toolchains; nothing today automatically bridges Windows desktop and macOS laptop without hacks. [verdent](https://www.verdent.ai/guides/codex-cli-resume-continue-save-chat)

So yeah, GOAT: **you’re not the only one thinking about session persistence; you ARE early on cross-agent, environment-aware, multi-device Dev state.**

***

## What you should actually build (and not build)

If you just build:

> “We upload `~/.claude`, `~/.codex`, `~/.gemini` to S3 and sync them back down.”

you’ve made **Claude Sync + rsync**, which is fine but not special. [github](https://github.com/tawanorg/claude-sync)

If you build:

> “DevSync: the portable state layer for AI dev tools – sessions + handoff + MCP config + environment parity”

then you’ve found a real wedge.

I’d structure your product as **four distinct layers**:

1. **Raw artifacts (per tool)**
   - Claude Code: whatever they store under `~/.claude/projects/` and session files. [codeagentswarm](https://www.codeagentswarm.com/en/guides/claude-code-history-complete-guide)
   - Codex: JSONL transcripts in `~/.codex/sessions/...`. [inventivehq](https://inventivehq.com/knowledge-base/openai/how-to-resume-sessions)
   - Gemini CLI: chats and session metadata in `~/.gemini/tmp/<project_hash>/chats/`. [habr](https://habr.com/en/articles/977390/)
   - These are stored encrypted in your cloud; you never rely on them directly for cross-tool logic.

2. **Normalized index**
   - For each session: project path, Git ref, agent type, timestamps, short description, machine ID.
   - You use this to present “20 sessions across 4 projects” in a unified view across tools/devices.

3. **Context capsule (handoff)**
   - A structured doc per project capturing:
     - Active task and next steps.
     - Key decisions, constraints.
     - Changed files (paths, diffs summaries).
     - Runtime assumptions (Node version, DB, external services, MCP servers used).
   - Computed from raw artifacts plus Git plus maybe your own CLI hooks.

4. **Environment manifest**
   - MCP servers, skills, hooks, plugins installed.
   - Tool config (flags like YOLO/sandbox, context policies, etc.). [perplexity](https://www.perplexity.ai/search/f545eea4-e083-445a-af7b-b5130066ea93)
   - OS + language runtimes (Node, Python, etc.).
   - Used to detect drift: “desktop has Supabase MCP, laptop does not”, “Node 22 vs Node 20”, etc.

That last two layers (context capsule + manifest) are where your idea actually **adds value** beyond “vendor might eventually add cloud session sync.”

***

## Table: how you’d differ from existing stuff

| Thing | Scope | Where state lives | Devices | Agent support | Your wedge |
| --- | --- | --- | --- | --- | --- |
| Gemini CLI sessions | Gemini-only | `~/.gemini/tmp/...` | Single machine | Gemini | Local resume only  [geminicli](https://geminicli.com/docs/cli/session-management/) |
| Codex CLI sessions | Codex-only | `~/.codex/sessions/...` | Single machine | Codex | Local resume only  [verdent](https://www.verdent.ai/guides/codex-cli-resume-continue-save-chat) |
| Claude Remote Control | Claude-only | Local machine | Any screen, same host | Claude | Remote driving, no state sync  [code.claude](https://code.claude.com/docs/en/remote-control) |
| Claude Sync (OSS) | Claude-only | Cloud-backed `~/.claude` | Multi-device | Claude | Encrypted directory sync only  [github](https://github.com/tawanorg/claude-sync) |
| DIY NAS + Syncthing | User-specific | NAS + local copies | Multi-device | Mainly Claude Code | Config+projects sync, manual  [steeman](https://www.steeman.be/posts/syncing-claude-code-across-multiple-machines/) |
| DevSync (what you should build) | All coding agents | Encrypted cloud + normalized index | Multi-device | Claude / Codex / Gemini / Grok / OpenCode | Cross-agent context capsules + MCP config parity |

***

## Technical shape (high level)

For an MVP, I’d keep it fairly boring:

1. **Local adapters per agent**
   - Read session metadata and transcripts from known paths (Codex: `~/.codex/sessions`, Gemini: `~/.gemini/tmp`, Claude: `~/.claude/projects`). [codeagentswarm](https://www.codeagentswarm.com/en/guides/claude-code-history-complete-guide)
   - Read config from `.codex/config.toml`, `.gemini/settings.json`, `~/.claude/settings.json` etc. [github](https://github.com/openai/codex/blob/main/docs/config.md)
   - Turn these into:
     - Session records.
     - Config objects.
     - Simple diffs for code (maybe via Git).

2. **Local daemon / CLI**
   - Runs on each device.
   - Watches for changes (file system + Git).
   - Generates/updates:
     - Session index.
     - Context capsule per project.
     - Environment manifest per device.

3. **Cloud sync**
   - Uploads encrypted raw artifacts + normalized state to your backend (R2/AWS S3 + Postgres).
   - Decryption keys live client-side; server only sees ciphertext plus metadata.

4. **Restore workflow**
   - On laptop:
     - `devsync projects` – list repos and sessions across agents.
     - `devsync restore <project>` – pulls context capsule + manifest.
     - `devsync doctor` – checks what’s missing and suggests install/actions.
     - Then you launch whichever CLI agent you like, injecting the capsule (manually via paste, or via integration as you build deeper).

This gives you your own UX and differentiation **before** you start trying to rewrite native session-resume of each vendor.

***

## Security and MCP risks (you cannot ignore this)

Sessions and MCP configs are **highly sensitive**:

- Codex sessions include shell commands and tool outputs – that’s sometimes secrets, DB queries, internal URLs. [verdent](https://www.verdent.ai/guides/codex-cli-resume-continue-save-chat)
- Gemini sessions preserve plans, task trackers, and tool outputs, not just friendly chat. [geminicli](https://geminicli.com/docs/cli/session-management/)
- MCP configs can be defined in project-level files (`.codex/config.toml`, `.mcp.json`, etc); security research has already flagged that malicious repos can use these to trigger dangerous behavior or remote code execution by misconfiguring MCP. [mindgard](https://mindgard.ai/disclosures/openai-codex-cli-mcp-configuration-remote-code-execution)

Your base posture should be:

- Default: **no secret sync** unless explicitly enabled and encrypted.
- Scan transcripts for obvious secrets (API keys patterns, `BEGIN PRIVATE KEY`, etc.) and:
  - Mask or drop those parts from cloud artifacts unless user overrides.
  - At minimum, warn loudly.
- MCP configs synced as **pending**; require explicit per-device enable.
- Never auto-run hooks, shell commands, migrations, or YOLO mode on a new device.
- Provide clear provenance:
  - Which device generated a config.
  - Which repo it came from.
  - When it changed.

If you ship “sync YOLO configs and transcripts to cloud in plaintext,” your strict Indian parents (and your users’ CISOs) will beat your idea with a chappal.

***

## Business / traction side

Brutal honesty, GOAT:

- Claude Sync already exists and does multi-provider storage for `~/.claude`; you’re not inventing the concept of “sync sessions to S3”. [dev](https://dev.to/tawanorg/claude-sync-sync-your-claude-code-sessions-across-all-your-devices-simplified-49bl)
- Vendors can, in theory, add their own cloud sync. Codex could push sessions into OpenAI accounts; Gemini could mirror CLI sessions to their web UI; Claude Code already has Remote Control for pseudo-cloud control. [code.claude](https://code.claude.com/docs/en/remote-control)
- Your **defensible wedge** is:
  - Cross-vendor.
  - Multi-device.
  - Environment-aware.
  - Actually designed around dev workflow (handoff, doctor, MCP parity), not just “history browser”.

Given your own stack (Moonshift, heavy agent use, multi-device dev, already built stuff like env sync and MCP sync), you have **founder–problem fit**. [perplexity](https://www.perplexity.ai/search/ee682135-0393-4bfd-a56d-167643d61dba)

For traction:

- Initial ICP should be:
  - Agent power users (Claude Code + Codex + Gemini).
  - Poly-device devs (desktop + laptop).
  - AI-native teams using these CLIs daily.
- Pricing path:
  - Free: single user, 1–2 devices, basic MCP sync + project capsule.
  - Pro: more devices, encrypted raw artifacts, environment doctor, semantic search over sessions.
  - Team: shared project handoffs, onboarding templates, audit logs.

***

## Should you build it?

Given all of the above: **yes, but build the hard version, not the trivial version.**

- If you just mirror `~/.claude` to S3, you get overshadowed by Claude Sync and any future first-party sync. [github](https://github.com/tawanorg/claude-sync)
- If you own MCP config normalization + environment parity + cross-agent context capsules and make “switch device and continue dev” truly seamless, you have a legit shot at a niche but real dev tool business, and at minimum a very useful internal tool.

You already live this pain: two devices, many agents, MCP chaos, multi-hour sessions you don’t want to rehydrate from scratch each time.  That’s exactly when building a focused tool **makes sense**. [perplexity](https://www.perplexity.ai/search/29593b04-6458-48d6-a5e8-e2773f1a73d4)

So, build it, GOAT – but treat **“cloud session storage” as an implementation detail**, and **“portable AI dev state” as the actual product**.
