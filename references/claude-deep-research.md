# Cross-Device Sync for Terminal AI Coding Agents: Competitive & Technical Landscape

## TL;DR
- **The durable opportunity is a cross-vendor abstraction layer with OS-aware path remapping as its headline feature — NOT another "sync my Claude sessions" tool.** Every major vendor is now shipping single-ecosystem sync natively (Claude Code Remote Control/teleport, Codex's unified App Server backend, Amp threads, Copilot CLI cloud sync), so the defensible wedge is the one thing no vendor will ever build: normalizing sessions + MCP + skills + instruction files across *competing* tools and remapping Windows↔macOS paths.
- **Mid-session cross-vendor replay (e.g., Claude Code → Codex) is effectively impossible and should be dropped from scope;** config/skills/instruction portability and *same-vendor* cross-device session continuity are the achievable, valuable core. The single sharpest, currently-unsolved technical pain — which even the leading competitor (claude-sync) explicitly punts on — is Claude Code's absolute-path project indexing breaking on Windows↔macOS sync.
- **Recommendation: build a "universal profile" that folds DevSync's MCP-sync foundation in, adds E2E-encrypted session sync with first-class path remapping for Claude Code first, then Codex/Gemini/opencode.** Ship path remapping as the MVP differentiator; treat E2E encryption as table stakes; do not try to out-run vendors on same-vendor live handoff.

## Key Findings

1. **Native vendor sync is real but ecosystem-locked and gap-riddled.** Claude Code teleport is one-way (cloud→local) and requires a clean git tree; Remote Control needs the origin machine to stay awake and online. Codex has the most mature multi-surface story (shared App Server backend) but it's OpenAI-only and cloud-account-bound. None of these help a developer who mixes Claude Code on a Windows desktop and Codex or Claude on a MacBook.

2. **The path-mapping problem is the crux and is largely unsolved.** Claude Code, Gemini CLI, and Cursor all key sessions to the absolute project path (or a hash of it). Syncing raw files between `C:\Users\...` and `/Users/...` produces orphaned, unresumable sessions. The leading sync tool (claude-sync) openly admits "there's no path remapping logic." Only a couple of niche tools (cursaves for Cursor, claudepath for local moves) attempt rewriting.

3. **Demand is loud and repeatedly expressed** in vendor issue trackers — multiple high-engagement Claude Code and Codex feature requests, Copilot CLI requests, and Cursor forum threads all ask for exactly this. Vendors are responding, but only within their own walls.

4. **The competitor field is fragmented, early, and mostly config-only or single-vendor.** No tool today delivers a polished, cross-vendor, path-aware, E2E-encrypted unified profile. That is the open lane.

## Details

### 1. Native vendor capability matrix (as of July 2026)

**Claude Code (Anthropic).** Ships the richest consumer-facing multi-device story:
- **`/teleport`** (`claude --teleport`): pulls a *cloud* session (started at claude.ai/code or the mobile app) down into the local terminal. It is **one-way** — you cannot push a terminal session to the web via teleport. Anthropic's docs state: "From the CLI, session handoff is one-way: you can pull cloud sessions into your terminal with `--teleport`, but you can't push an existing terminal session to the web." The Desktop app's "Continue in" menu is the only exception. Teleport requires **clean git state** ("Your working directory must have no uncommitted changes. Teleport prompts you to stash changes if needed"), the same repo checkout, the branch pushed to remote, and the same claude.ai account. Teleport also gives the terminal its own copy — "new work there stays local and doesn't appear in the cloud session."
- **Remote Control** (`claude remote-control` / `/remote-control`, `--rc`): pushes a *live local* session to phone/web. The session runs locally the whole time; web/mobile are "a window into that local session." Limitations, verbatim from the docs: "Remote Control runs as a local process. If you close the terminal, quit VS Code, or otherwise stop the claude process, the session ends," and "if your machine is awake but unable to reach the network for more than roughly 10 minutes, the session times out and the process exits." (Separately, issue #39970 reports an *undocumented* ~15-minute idle timeout on the Desktop/Code surface that "is not documented and the timeout is not communicated to the user before it happens.") Research preview; Pro/Max/Team/Enterprise only (no API keys); disabled on Bedrock/Vertex/Foundry and when `ANTHROPIC_BASE_URL` is redirected; unavailable under Zero Data Retention.
- **Claude Code on the web** (claude.ai/code): runs on Anthropic cloud infra, no local filesystem.
- **Residual gap:** No true device-to-device *local* session migration (Windows desktop ↔ MacBook) without the origin machine staying online. No cross-vendor anything. Anthropic has begun syncing conversation history across devices on the same account (reported in issue #71794 as an *unexpected*, un-opted-in behavior), suggesting account-level sync is coming — but it will be Claude-only.

**Codex CLI (OpenAI).** The most architecturally mature cross-surface model:
- CLI, Desktop app (macOS + Windows), IDE/VS Code extension, and Cloud runtime all connect to a shared **App Server** (long-lived process exposing a JSON-RPC 2.0 API over stdio or WebSocket). A thread started in the CLI can be resumed in Desktop "without any export/import dance." Sessions are JSONL under `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`, with a `session_meta` first line.
- Rich lifecycle: `codex resume` (picker), `--last`, `--resume-session-id`, `/fork`, archive/unarchive (v0.136), delete (v0.140). Imports histories from other agents via an `external-agent-sessions` crate.
- **Known gaps/bugs:** Desktop may not list CLI sessions if the app-server wasn't running at creation; ephemeral sessions aren't persisted; sessions containing `encrypted_content` often can't be continued after crossing provider/account (`invalid_encrypted_content`); Desktop's session list only pulls the recent ~50. Open requests (#12593, #14722, discussion #14067) ask for account-linked cross-device sync and CLI↔app-server live sync.
- **Residual gap:** Sync is OpenAI-account-bound and cloud-mediated; no cross-vendor; local file portability still hits path issues.

**Gemini CLI (Google).** Auto-saves sessions to `~/.gemini/tmp/<project_hash>/chats/` (project-scoped, hash of project root). `/resume` session browser, `--resume`/`-r` flag, `/chat save` checkpoints, and a shadow-git checkpointing system under `~/.gemini/history/<project_hash>`. No native cross-device cloud sync. Note: Google announced Gemini CLI is being replaced by "Antigravity CLI" for unpaid/Google One tiers. **Residual gap:** everything local + path-hash-keyed; no sync.

**opencode (sst).** SQLite + file storage under `~/.local/share/opencode/` (`opencode.db`, `storage/session|message|part`). Native **session sharing** publishes to Cloudflare Durable Objects for a public web viewer (read/playback, not cross-device resume). A healthy third-party ecosystem already exists (opencode-synced, opencode-sync, opencode-sync-plugin/OpenSync, opencode-session-backup). **Residual gap:** sharing ≠ syncing; DB concurrency and cross-OS path issues remain.

**Cursor CLI (cursor-agent).** Chats live in local SQLite; CLI chats under `~/.cursor/chats` and resume via `cursor-agent --resume`. **IDE chats and CLI chats do NOT share a session store** (confirmed by Cursor staff in the forum). Cloud handoff exists (prepend `&` to push to a Cloud Agent, pick up at cursor.com/agents), and "My Machines" lets cloud agents run on your registered machine. Strong user demand for account-level chat sync (forum). **Residual gap:** no local cross-device sync, IDE↔CLI split, path issues.

**Grok / Grok Build (xAI).** Sessions auto-saved to `~/.grok/sessions/`, keyed by working directory. `/resume`, `--resume`, `-c`, `/fork`; subscription-gated (SuperGrok/X Premium+). Config in `~/.grok/config.toml`. No native cross-device sync. Still beta (v0.2.x as of July 2026). **Residual gap:** no sync at all.

**GitHub Copilot CLI.** Sessions in `~/.copilot/` (`session-state/<uuid>/events.jsonl`). Notably, Copilot CLI **syncs sessions to your GitHub account by default** and offers **remote control** from GitHub.com/GitHub Mobile (origin machine must stay online, `/keep-alive`). But **cross-device *resume* is still not supported** — manually copying a session dir and running `--resume` overwrites `events.jsonl` and destroys history (issue #1635); cloud-synced cross-device continuity is an open request (#1947). The Copilot SDK supports resumable sessions via a user-supplied `session_id`. **Residual gap:** the sync is telemetry/history, not resumable working state across machines.

**Amp (Sourcegraph).** Threads sync to ampcode.com by default and can be continued across devices; "orbs" (remote machines) keep running after you close your laptop. This is the closest to "cloud-native by default," but it's Amp-only and stores threads on Sourcegraph servers (a privacy consideration; MCP config is GUI-global, not file-based). **Residual gap:** vendor-locked, server-stored, no cross-vendor.

**Cross-vendor summary:** Every vendor is building *within* its ecosystem. The white space is precisely between ecosystems and across OSes.

### 2. Session storage formats & technical feasibility

| Tool | Location | Format | Path keying | Cross-device resume native? |
|---|---|---|---|---|
| Claude Code | `~/.claude/projects/<encoded-path>/<session>.jsonl` | JSONL, append-only event stream | **Absolute path, non-alphanumerics→dashes** | Via cloud only (teleport/RC) |
| Codex | `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` | JSONL + `session_meta` header | date-based; cwd in meta | Yes, via App Server |
| Gemini CLI | `~/.gemini/tmp/<project_hash>/chats/` | JSON | **Hash of project root** | No |
| opencode | `~/.local/share/opencode/` (SQLite + files) | SQLite/JSON | project-keyed | Sharing only |
| Cursor | local SQLite | DB rows | workspace/path metadata | No (cloud handoff only) |
| Grok | `~/.grok/sessions/` | JSON | working directory | No |
| Copilot CLI | `~/.copilot/session-state/<uuid>/events.jsonl` | JSONL events | uuid | Account history sync; no resume |

**The path-mapping problem (the crux).** Claude Code derives the project folder name by escaping the absolute path: `/Users/me/app` → `~/.claude/projects/-Users-me-app/`, while the same repo at `C:\Users\me\app` (or a different macOS username/location) produces a *different* folder. Consequently, after a naive file sync, `claude --resume` on the second machine cannot find the first machine's sessions. This is confirmed by the claude-sync author's own documentation and raised pointedly by users in the tool's DEV Community thread: "does the tool handle path remapping somehow, or is this a known limitation? Because if project session continuity doesn't work across different paths, the core use case feels significantly weaker." Community writeups (Rockford Lhotka, Nick Ang) "sidestep" it by keeping identical paths everywhere. Gemini CLI has the same issue via `project_hash`; Cursor stores path metadata in chat rows (cursaves rewrites these; it also matches projects by git remote URL). Tools that solve it either (a) require identical paths on all machines, or (b) rewrite the path key + in-file path references on pull. **This is the single highest-value, currently-unmet technical feature.**

**Format churn risk.** Anthropic explicitly warns: "The entry format is internal to Claude Code and changes between versions, so scripts that parse these files directly can break on any release." This is a real, ongoing maintenance tax for any file-parsing approach and must be designed for (adapters, schema versioning, graceful degradation).

**Mid-session cross-device resume feasibility (same vendor):** Achievable for Claude Code, Codex, Gemini, Grok, opencode by copying/rewriting the session file + ensuring the repo/branch is present — *if* path keys are remapped. Codex's App Server makes it cloud-native already.

**Cross-vendor session replay (e.g., Claude Code session → Codex):** **Fundamentally impractical.** Each tool encodes tool-call schemas, thinking blocks, provider-specific message formats, and (in Codex's case) `encrypted_content` that can't even survive an account change. npow/session-sync attempts this by translating Claude Code tool calls into other agents' native session dirs via hooks, but this is best-effort context transfer, not faithful state replay. **Recommendation: treat cross-vendor as config/skills/instruction portability + context handoff, never as live session replay.**

### 3. Competitor teardown

**Session-continuity tools:**
- **tawanorg/claude-sync** (~100 stars, actively maintained, 26 releases, latest ~April 2026): Go binary; syncs `~/.claude` (sessions, agents, skills, plugins, rules, settings) to R2/S3/GCS/S3-compatible/WebDAV; gzip + **age E2E encryption** ("Every file is encrypted before it leaves your machine using age, a modern encryption tool by Filippo Valsorda"), passphrase-derived keys via Argon2 (a memory-hard KDF). The reference implementation for this space. **Punts on path remapping** (its explicit, acknowledged weakness). Open-source, no business model.
- **npow/session-sync** (~1 star, very new): the only cross-vendor *session* translator — Claude Code → Codex/Amp/Gemini via PostToolUse/Stop hooks. Proof the idea is wanted; nascent.
- **dantelex/aisync ("ss")**: multi-device continuity for Claude/Codex/Cursor/opencode via a provider-backed sync folder (iCloud/Dropbox/OneDrive/etc.); claims auto path detection and handoff blocks. (Repo was not retrievable at time of writing — treat traction as unverified.)
- **codsync/codsync** (~2 stars, alpha): encrypted Git-backed Codex session + config sync; Windows/macOS; conservative conflict handling; self-describes as alpha.
- **kroxiksut/codexSync**: local-first "cold-sync" Codex state handoff, Windows-first; explicitly does NOT transfer auth tokens (OpenAI licensing).
- **waynesutton/opencode-sync-plugin & codex-sync-plugin (OpenSync)** (~10 stars): Convex-backed cloud sync/search/share; opensync.dev SaaS option.
- **iHildy/opencode-synced** (~40 stars, active): opencode config + optional session sync via Git or Turso (concurrency-safe); opt-in sessions.
- **Dinesh3184/claude-session-sync**: iCloud Drive sync GUI for Claude Code (macOS + Windows).
- **renefichtmueller/claude-sync**: multi-backend (Git/iCloud/Dropbox/Syncthing/rsync) with snapshots/rollback and encryption.

**Config/MCP/skills sync (DevSync's neighborhood):**
- **Leoyang183/sync-agents-settings** (~8 stars): MCP + CLAUDE.md sync from Claude Code to 12+ agents; drift detection.
- **nicepkg/vsync** (~46 stars, active): MCP/Skills/Agents/Commands across Claude Code/Cursor/opencode/Codex, one source of truth.
- **baranovxyz/agentsync** and **spxrogers/agentsync** (beta): canonical `.agents/` config synced to 9–31 agents.
- **neon-solutions/add-mcp**: one-command MCP install across ~16 agents (Neon).
- **ztripez/mcp-sync**: MCP config sync across tools.
- **AgentSync (Rust)**: symlink-based config sync.
- **skills-sync**: skills + MCP workspace.
- **DevSynq** (devsynq.app): commercial-looking MCP/API-key sync across IDEs + launcher/marketplace.

**URL-addressable / mobile-control / viewer approaches:**
- **TokenRip** (tokenrip.com): makes session state a URL-addressable artifact (`/a/<slug>`) any agent can fetch — a genuinely different architectural bet ("don't sync files; give state a permanent URL").
- **Omnara** (YC S25): mobile control of Claude Code/Codex with cloud sync + local handoff; the original open-source wrapper repo (~2.6k stars) was **archived in February 2026** because "it was built as a wrapper around the Claude Code CLI, which became unfeasible to maintain with Claude Code's constant updates" — the company pivoted to a proprietary Claude Agent SDK–based voice-first platform at omnara.com. (A ~4.3★ mobile-app rating was seen but is unverified.)
- **Syncode – AI Programming Tool**: iOS + macOS companion, LAN pairing, cross-device (Mac execute → iPhone follow-up).
- **jazzyalex/agent-sessions** & **kenn-io/agentsview**: local-first session browsers/search across 20+ agents (read/resume-command, not sync).
- **LAN-workspace approaches** and Syncthing/NAS/dotfiles/chezmoi+age: DIY approaches that work for config but hit the path problem for sessions.

**Takeaway:** The config-sync lane (where DevSync sits) is crowded but shallow; the session-sync lane has a clear leader (claude-sync) that explicitly punts on the hardest problem (path remapping) and is single-vendor. No one owns "unified, cross-vendor, path-aware, encrypted profile."

### 4. Market demand signals

Vendor issue trackers show sustained, high-engagement demand:
- **Claude Code:** #47926 ("Allow resuming Claude Code sessions across devices"), #52052 (opt-in cloud sync of transcripts/plan/memory), #26045 (cross-device memory sync), #45358 (cross-machine session sync, Mac+Windows+Linux), #51816 (Mac+Windows instance-to-instance), #60058 (connect from any client), #22648 (account-level settings sync, with a long list of duplicates: #6037, #19634, #13461, #12119, #17682). #71794 shows Anthropic quietly began syncing history across devices.
- **Codex:** #12593 (VS Code account-linked cross-device sync), #14722 (sync CLI ↔ app-server sessions), discussion #14067 (cross-device thread/context sync; spawned third-party CodeVibe and codex-workspace-sync), #4514 (forking).
- **Copilot CLI:** #1947 (cloud-synced sessions), #1635 (cross-environment resume).
- **Cursor forum:** multiple threads demanding account-level chat/agent sync, especially for Remote SSH users.

Recurring workarounds users cite (scp, symlinks to iCloud/Dropbox, git repos, identical paths everywhere) all "require user setup and maintenance." The volume of feature requests, the explosion of DIY tools, and paid mobile apps (Omnara IAP, Syncode IAP) are evidence of willingness to pay for a frictionless solution.

### 5. Strategic assessment

**Where the durable moat is.** Vendors will keep building *intra*-ecosystem sync (and doing it better than any third party, because they control the backend and auth). Competing there is a losing race. The durable, vendor-proof value is:
1. **Cross-vendor unified profile** — one normalized model for MCP servers + skills + custom agents + instruction files (CLAUDE.md/AGENTS.md/GEMINI.md) + settings, mapped into each tool's native format. This is DevSync's existing strength, extended.
2. **OS-aware path remapping as a first-class feature** — the concrete, unsolved pain that makes same-vendor session sync actually work across Windows↔macOS. This is the wedge no competitor has nailed.
3. **E2E encryption by default** — table stakes given session files contain code, secrets, and full transcripts.

**Risks:**
- **Format churn:** Claude Code's format "changes between versions" by Anthropic's own admission. Mitigate with per-tool adapters, schema versioning, and conservative fail-safe behavior (never corrupt local state; always back up before write — the pattern codexSync/cursaves use).
- **Vendor lock-out / ToS:** Storing session data is user-owned data (fine), but *auth tokens* are not transferable (codexSync explicitly refuses; Codex `encrypted_content` breaks across accounts). Never sync credentials; never depend on undocumented cloud endpoints. Remote Control/teleport are gated to subscriptions and blocked under ZDR — don't build on them.
- **Vendors closing the gap:** Anthropic's account sync (#71794), Codex's App Server, Copilot's GitHub sync are all encroaching on single-vendor sync. This *validates* the demand while *confirming* that only the cross-vendor + cross-OS layer is defensible.
- **Security/privacy expectations:** E2E encryption (age or equivalent), local-first defaults, explicit opt-in per repo, no plaintext in any remote — all mandatory. This is also a differentiator vs. Amp (server-stored) and OpenSync (Convex).
- **Honest limits:** Cross-vendor mid-session replay is impossible; over-promising it will burn trust. Sell config/skills portability + same-vendor session continuity + best-effort context handoff.

## Recommendations

**Stage 0 — Fold in DevSync (immediate).** Keep and harden the existing MCP-config sync as the "config" half of the unified profile. Extend its normalized model to cover skills, custom agents, instruction files (CLAUDE.md/AGENTS.md/GEMINI.md), and settings across Claude Code, Codex, Gemini CLI, opencode, Cursor, Copilot, Grok, Amp. This is shippable now and already differentiated from single-vendor sync tools (and from crowded but shallow rivals like vsync and sync-agents-settings).

**Stage 1 — MVP: Claude Code session sync with path remapping + E2E encryption (the wedge).** Target the founder's exact case (Windows desktop ↔ macOS MacBook, both daily). Deliver:
- Encrypted (age/passphrase-derived) sync of `~/.claude/projects/` and global config to user-owned storage (S3/R2/GCS/WebDAV), matching claude-sync's proven approach.
- **First-class OS-aware path remapping:** on pull, rewrite the project folder key AND in-file absolute path references (cwd, git snapshots, file refs) from the source machine's path to the target machine's path, using the git remote URL as the project identity (the technique cursaves uses for Cursor). This is the feature claude-sync explicitly lacks — lead marketing with it.
- Safe-by-default writes: back up before overwrite, atomic writes, conflict preservation.
- **Benchmark to advance:** `claude --resume` on the MacBook reliably finds and resumes a session created on the Windows desktop (different absolute paths) in <10s, with no corruption, across 3 consecutive Claude Code releases.

**Stage 2 — Second and third vendors: Codex + Gemini CLI + opencode.** Add adapters. For Codex, cooperate with (don't fight) the App Server — sync what's local and useful, avoid `encrypted_content`/token pitfalls. For Gemini, remap `project_hash`. For opencode, use safe SQLite snapshotting (the `sqlite3 .backup` pattern the community scripts use).
- **Benchmark:** each new adapter must pass the same cross-OS resume test as Claude Code before GA.

**Stage 3 — Unified profile + cross-vendor context handoff.** One "profile" = sessions + MCP + skills + agents + instructions, portable across machines and (for config) across vendors. Add best-effort context handoff (summarize + re-inject) for cross-vendor moves — explicitly labeled as context transfer, not replay.

**Monetization & thresholds.** Free/open-source core (build trust, match claude-sync) with a paid tier for managed encrypted storage, multi-machine dashboards, and team profiles (DevSynq/OpenSync validate willingness to pay). **Kill/pivot signal:** if Anthropic ships account-level *local* session sync with automatic path remapping across OSes (not just history mirroring, per #71794), the single-vendor wedge collapses — at that point double down on cross-vendor config/skills/profile portability, which no vendor will build.

**What NOT to do:** Don't build on Remote Control/teleport (gated, one-way, origin-online, with documented ~10-min network and undocumented ~15-min idle timeouts). Don't promise cross-vendor session replay. Don't sync auth tokens. Don't store plaintext anywhere remote.

## Caveats
- Star counts and release dates are point-in-time (mid-2026) and volatile; several competitors are days-old and could stall or surge. iHildy/opencode-synced was reported at both ~40 (GitHub profile listing) and ~116 stars across sources — treat as ~40–116. dantelex/aisync traction is unverified (repo not retrievable at writing). Omnara's ~4.3★ mobile rating is unverified.
- Session file formats — especially Claude Code's — are explicitly documented as unstable across versions; any parsing approach carries ongoing maintenance risk.
- Vendor capabilities (teleport, Remote Control, Codex App Server, Copilot sync, Gemini→Antigravity migration) are evolving fast; specifics may have changed since the cited docs.
- "Impossible" cross-vendor replay is an engineering judgment based on schema/encryption divergence, not a vendor statement; niche context-transfer tools (npow/session-sync) show partial approaches exist.