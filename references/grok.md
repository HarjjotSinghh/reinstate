**GOAT, ye idea solid hai — validated pain, not some random brainfart.**

You’re describing exactly the shit every dual-setup dev (desktop Windows + MacBook) hits daily with Claude Code, Cursor, Codex CLI, Gemini CLI, OpenCode, etc. Local sessions/config/MCP/skills live in silos. You grind 8 hours on desktop, create 15-20 sessions across projects, switch to laptop on bed, and either pull git + re-onboard the agent (waste of tokens and brain cells) or lose full conversational context. That’s real friction.

### Reality Check from the Field (2025-2026)

The problem is **not theoretical**. It’s being screamed about and hacked around:

- **Claude Code**: Multiple open-source tools already exist specifically for this. Biggest is *Claude Sync* (tawanorg) — CLI that encrypts and pushes/pulls `~/.claude/` (sessions, agents, skills, memory, settings) to R2/S3/WebDAV with age encryption and passphrase keys. Another is GitHub-based variants and memory-only daemons. GitHub issue on Anthropic repo: “No native way to sync Claude Code config and skills across machines.” People use Syncthing + NAS + careful exclusion of sessions/caches because path indexing breaks continuity.
- **Cursor**: Chat/agent transcripts are purely local (`~/.cursor/projects/.../agent-transcripts/`). Feature requests for cloud sync are old and recurring. Official reply (late 2025/2026): “On our radar, no timeline.” Devs built their own CLIs and extensions (Cursor Chat Transfer, etc.). Dropbox/iCloud + symlink hacks are common.
- **Others**: Depot.dev ships remote Claude Code sessions you can create/resume/share across machines and teammates (more cloud sandbox than pure local sync). OpenCode has multi-session + share links built-in. Config-only tools exist for MCP/skills across Claude + Codex + Cursor (`ai-config-sync-manager`, `mcp-sync`).
- Broader pattern: Most AI coding tools still treat context as machine-local. Switching devices = context death. Articles and X posts talk about externalizing to Obsidian/git or using Tmux + Tailscale + Termius as workarounds.

No polished **universal** layer exists that cleanly handles sessions + MCP servers + skills + memory across the major CLIs/IDEs with good path remapping, conflict handling, and E2E encryption. Tool-specific hacks and file-sync bandaids dominate.

**Market context**: AI coding tools are massive. Cursor hit multi-billion ARR territory. Claude Code is extremely loved. 80-90%+ of developers use these daily; tool-stacking (2-4 tools) is normal. The broader AI code tools market sits in the multi-billion range. Your original Dev Sync (central MCP config) is still relevant — MCP support is now table stakes across Claude Code, Cursor, Codex, etc., and configs still fragment.

### Is This a Good Idea?

**Yes, with sharp edges.**

**Pros (why it can work):**
- You’re solving your own exact daily pain → highest signal product.
- Natural expansion of Dev Sync. MCP/skills/config was the right starting point; sessions are the higher-value layer people will actually pay for or stick with.
- Indie builders keep shipping partial solutions → demand is proven, distribution possible via GitHub/HN/Reddit/X.
- Technical core is straightforward: detect tool storage locations → normalize/serialize sessions + configs → encrypt → store (S3/R2 or DB) → pull + remap paths on other devices. Selective sync, conflict detection, shell hooks for auto push/pull.
- Upside if you nail path remapping (Claude’s absolute-path indexing is a known footgun) and multi-tool support.

**Cons / Risks (brutal):**
- Big players can (and probably will) ship native account-level sync. Cursor already acknowledges the requests. Anthropic could add it tomorrow. When they do, pure file-sync tools become less necessary.
- Privacy is a landmine. Sessions contain full code context, prompts, potentially secrets, architecture decisions. Zero-knowledge E2E + self-host option is non-negotiable or enterprise/devs with IP will ghost you. One leak and you’re done.
- Maintenance hell. Every Claude Code / Cursor / Codex update can change storage format or paths. You’ll be reverse-engineering forever.
- Adoption friction. Devs hate yet another CLI unless it’s dead simple and works better than their current Syncthing/Dropbox hack. “Just use git + consistent paths” crowd will dunk.
- Monetization is hard. Open-source core is almost required for trust. Free tier + paid storage/team/sharing/priority? Devs are cheap and already paying Cursor/Claude/Codex. Pure free-forever dies.
- Scope trap: Trying to support every tool perfectly on day one = death. Start with Claude Code + Cursor + MCP/skills layer.

**Odds assessment (assuming decent execution):** Decent chance of useful personal tool + small paid niche (power users with dual machines + teams). Low chance of “blow up and print money” unless you hit perfect timing before natives ship and nail distribution. High chance it stays a solid side project that solves your life and a few hundred others’. Pivot/kill signals: if Anthropic or Cursor ships good native sync in the next 6-9 months and usage stays flat.

### Should You Build It?

**Yes. You’re going to anyway, so do it right.**

1. **Narrow MVP first**. Pick Claude Code sessions + MCP/skills/config sync. Add Cursor transcripts second. Ignore everything else until those two feel magical. Path remapping is the real technical win — nail that or the product is half-broken.
2. **Architecture**: Local watchers or CLI push/pull → E2E encrypt (age or similar) → R2/S3 or simple Postgres + object storage. Optional self-host. Dashboard later for browsing sessions.
3. **Differentiate or die**: Universal > single-tool. Security theater-free (real zero-knowledge). Conflict UI that doesn’t suck. Optional team share (Depot-style) as paid feature.
4. **Distribution**: Open-source the core hard. Post the exact dual-machine pain story on HN/Reddit/X. Ship a one-command install. Demo the “worked 6 hours on desktop, continued exact same session on MacBook in bed” video.
5. **Monetization reality**: Charge for cloud storage beyond free tier, team features, or priority support. Don’t make it free forever just because “devs like free.”
6. **Timeline pressure**: Build the scratch-your-itch version in 2-3 weeks max. Validate with 10-20 real dual-setup users before expanding.

This is a classic “I hate this enough to fix it” product. The market has already confirmed the pain with multiple indie tools. Your edge is making the *universal* version that also covers the original Dev Sync MCP vision instead of yet another Claude-only sync.

Just don’t over-engineer the perfect multi-agent orchestration platform on day one. Ship the session continuity that makes *you* stop losing context between Windows desktop and MacBook. Everything else is secondary.

Now go drink some chai and start mapping the `~/.claude` and Cursor transcript layouts. The window is open, but it won’t stay open forever.