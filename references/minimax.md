Let me search for the factual background on this topic.

Let me do more searches to round out the picture.

I have a strong factual base. Let me now read Step 2.

I have the direction. Now let me read Step 3.

Let me check a few more specific items to round out the picture, then do the deep analysis.

I have enough material to build the analysis and execute the research. Let me work through the deep analysis and then drive Step 4.

# Step 3: Deep Analysis

## 1. Deep sub-topic breakdown

This question is a build/validate/steer question for a developer/founder, not a usage guide. It decomposes into six sub-topics that each demand real research:

**S1. The vendor headwind problem (high-priority)**
The biggest strategic question. The user wants to "store sessions on cloud and sync across devices" — but Anthropic, OpenAI, Google, and xAI are all racing to ship the *exact* same feature natively. Anthropic already has `claude --teleport`, `--cloud` flag, web sessions, scheduled cloud tasks, and a "Continue in" menu in the desktop app. OpenAI's Codex Cloud has an "Import legacy sessions" feature requested by users. GitHub has Copilot Cloud Agent on mobile. Need to research: how much of the cross-device pain does each vendor already cover? Where are the explicit gaps? Is the third-party role going to be the universal glue across vendors, or the redundant layer that vendors are racing to make obsolete?
*Keywords*: "claude --teleport", "Claude Code cloud sessions", "Codex Cloud", "GitHub Copilot cloud agent mobile", "Anthropic scheduled tasks", "Anthropic auto memory"
*Why it matters*: Determines the entire product positioning. If Anthropic is doing it for free for paid users, the third-party tool must either (a) cover vendors Anthropic doesn't, (b) target users who don't have Pro/Max, (c) be better/faster/cheaper, or (d) solve a different layer entirely.
*Connects to*: S2 (what's actually different), S6 (business model)

**S2. The technical reality of "session sync"**
Across Claude Code, Codex CLI, Gemini CLI, opencode, Grok Build, Cursor Agent, GitHub Copilot CLI, Antigravity CLI — each stores sessions in a different location, format, and lifecycle. Claude Code uses append-only JSONL with project-encoded slugs. Codex uses `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`. Gemini uses `~/.gemini/tmp/<project_hash>/chats/`. opencode uses `~/.local/share/opencode/storage/`. The format "is internal to Claude Code and changes between versions" per Anthropic's own docs. The path slug is lossy (dash and dot both become dash, so two real paths can collide). Need to research: how reliable is each tool's storage contract? What's the migration story when storage format changes? What does a multi-vendor sync tool actually have to know?
*Keywords*: "Claude Code JSONL format", "claude-code session recovery", "opencode storage XDG_DATA_HOME", "Codex rollout session", "Grok session storage GCS", "session state format changelog"
*Why it matters*: A "universal" session sync requires either (a) reverse-engineering and parsing each tool's internal format with high fragility, or (b) sitting at a higher level (hooks, MCP, shell wrappers) to capture session data in a vendor-neutral way. The second path is much harder for sessions (the data lives inside the tool, not flowing through a hook) but vastly more durable.
*Connects to*: S1 (what layer are you playing at?), S3 (which existing tools chose which layer)

**S3. The competitive map (open source + cloud)**
There are at least 15+ relevant tools that touch this space, roughly grouped:
- *MCP-only config sync*: Devsynq (most visible), mcp-sync (multiple forks), mcpsync-cli, mcp-sync (sodeyama), agents (amtiYo), agentsync (spxrogers, 31 agents, Go)
- *Claude Code–specific session sync*: claude-sync (tawanorg, R2/S3/GCS + age encryption)
- *Multi-tool config sync*: coding-agent-sync (TCTinh, GitHub Gist), skills-sync (Ryan Reh, symlink-based), Memora (R2/S3 sync of memory)
- *Memory layer*: context0 (local SQLite), Memora, the broader MCP memory server pattern
- *Cloud handoff/agent runtimes*: Omnara (Cloud Sessions, explicit "migrates local sessions to the cloud when laptop disconnects")
- *Config framework that absorbs the space*: ECC (100K stars, 28 agents, 119 skills, 60 commands, supports Claude Code/Cursor/Codex/OpenCode)
- *The "externalize context" school*: workos, Obsidian-vault approach
Need to research: which of these is closest to the user's idea, what's their distribution, traction, and business model. Which is the user effectively competing with head-to-head vs. complementing?
*Keywords*: "Devsynq github stars", "claude-sync npm", "agentsync 31 agents", "Omnara Cloud Sessions", "everything-claude-code stars adoption", "skills-sync vs claude-sync"
*Why it matters*: A solo founder needs to know: am I building the 12th "sync MCP" tool, or carving out a new position? And is "universal session sync" actually a different product or a feature add to the existing MCP-sync tools?
*Connects to*: S1, S4, S6

**S4. The universal-claim problem (scope discipline)**
"Universal" means covering Claude Code, Codex, Gemini, opencode, Grok, Cursor, Copilot, Antigravity, etc. Each has a different storage location, format, lifecycle, retention policy, and most importantly — different *vendor strategy*. Some are moving to cloud (Anthropic), some are still local-only (opencode, Cursor), some are actively hostile to local-only (Grok uploading repos to GCS by default). Covering all of them in v1 is impossible. Covering 2-3 well is a real product.
*Keywords*: "agent SDK session storage", "Cursor agent storage", "Antigravity CLI brain", "opencode portable storage"
*Why it matters*: The user's framing is "sessions across all the tools." But a generic tool that pretends to handle 8 vendors will handle 0 vendors well. The scope decision (which 2-3 tools to support deeply) is the single most important product choice.
*Connects to*: S2 (what can you actually do per vendor?), S3 (what's the comparable competitor scope?)

**S5. The "Dev Sync" brand/pivot**
The user already named the product "Dev Sync" and it's an existing concept that overlaps with Devsynq (different spelling, MCP-sync tool by Harjot Singh Rana). The user wants to "pivot" Dev Sync from MCP-only to universal. Need to research: is the brand name usable? Is "Dev Sync" trademarked? Does the user have prior Dev Sync code/users? Is the pivot credible (do current Dev Sync users care about session sync, or is this a rebrand)?
*Keywords*: "Dev Sync" trademark, "devsynq" github, "devsync" OR "dev-sync" coding agent
*Why it matters*: A pivot is much easier if you have distribution; a rebrand is a marketing cost. If Dev Sync already has a few hundred users, that's distribution to migrate. If it doesn't exist yet, the brand is just a name to pick.
*Connects to*: S6 (does the existing user base change the answer?)

**S6. The business model question**
The user hasn't said "make money" — they said "scratch my own itch." But for any product with cloud infrastructure, the question of business model is unavoidable. Most of the open-source competitors are free (Devsynq, claude-sync, ECC). The cloud players (Anthropic) bundle it free for paying users. Where's the room? Possibly: hosted multi-vendor cloud sync with encryption for users who don't want to set up R2 themselves, plus a paid team tier. Or: the meta-orchestration layer (an "agent orchestrator" that watches sessions across all tools). Or: privacy-first / self-host-first with paid cloud convenience.
*Keywords*: "agent cloud sync pricing", "R2 encrypted session sync SaaS", "MCP server marketplace business"
*Why it matters*: Determines whether the project is a 2-week hackathon project, a serious product, or a venture-backable startup.
*Connects to*: S1, S3, S5

## 2. Precise concept disambiguation

The terminology in this space is genuinely confusing, and conflating terms will lead to a sloppy product.

- **Session** vs **Memory** vs **History** vs **Context** vs **Transcript**: A *session* is the conversation + tool calls + file edits for one working session; it's a JSONL (or per-vendor equivalent). *Memory* is the distilled/curated knowledge that an agent carries forward (e.g., `~/.claude/projects/<p>/memory/`, AGENTS.md, GEMINI.md, .cursorrules). *History* is the index/listing of past sessions (e.g., `history.jsonl`). *Context* is the working memory of the current prompt (in-memory, not stored). *Transcript* is roughly synonymous with session in this domain. **Misconception to flag**: syncing sessions ≠ syncing memory. They need different sync cadences, sizes, and tools.

- **Sync** vs **Migrate** vs **Handoff** vs **Teleport**: *Sync* is bidirectional background replication. *Migrate* is one-time move. *Handoff* is "I'm done here, hand this off to that device." *Teleport* is Anthropic's branded term for pulling a cloud session to local. Each implies different UX and architecture. The user said "sync" but the actual user need is closer to "handoff."

- **MCP server config** vs **MCP marketplace** vs **Agent Skills** vs **Plugins** vs **Slash commands** vs **Subagents** vs **Rules** vs **Hooks**: Different artifacts, different per-vendor support. ECC enumerates 28 agents, 119 skills, 60 commands, rules, hooks, MCP configs. Mapping these across vendors is a research project on its own. **Misconception to flag**: claiming "universal" coverage when in practice only MCP servers and skills translate cleanly; agents, commands, and hooks are vendor-specific.

- **~/.claude** vs **~/.claude.json** vs **~/.claude/projects/**: Three distinct things. The single `~/.claude.json` is global config; `~/.claude/projects/<encoded-cwd>/` is per-project sessions; `~/.claude/agents/`, `skills/`, `plugins/`, `commands/` are shared assets. Sync tools often blur these and end up syncing the wrong thing.

- **"Local session" vs "Cloud session" vs "Web session"**: A local session is the JSONL on disk. A cloud session is hosted by Anthropic / OpenAI / GitHub on vendor infra. A web session is the same as a cloud session but accessed via browser. With `--teleport`, a cloud session becomes a local one in the new terminal.

- **The path-slug collision**: A real, documented bug-class in Claude Code where two unrelated real paths slugify to the same directory name. Any sync tool that naively uses the slug will get confused.

- **"Format is internal and changes"**: Anthropic explicitly says scripts that parse JSONL transcripts "can break on any release." This is a deprecation warning. A sync tool that parses JSONL content is therefore inherently fragile; a sync tool that only moves the file bytes (or only writes via sanctioned interfaces like `/export` or the SessionStore adapter) is more durable.

## 3. Rough scope

**In scope (must cover):**
- The current state of each major coding agent's session storage (Claude Code, Codex, Gemini, opencode, Grok, Cursor, Copilot, Antigravity) — at the level of: where, in what format, with what retention, and what the vendor's own cloud strategy is.
- Existing open-source tools in this space (Devsynq, claude-sync, coding-agent-sync, agentsync, ECC, skills-sync, Memora, context0, Omnara) — positioning, traction, business model, what they cover, what they don't.
- Anthropic's own cloud session features (--teleport, --cloud, Continue in menu, scheduled tasks) — what they do, what they don't, and where the explicit gap is for a third-party tool.
- The technical landmines: format instability, path-slug collisions, secrets leakage, encryption, hook-based vs file-based capture, retention policy conflicts.
- A clear product-strategy recommendation: what to build, what not to build, who to target, how to position, what to charge.
- A list of "non-obvious insights" — things the user will hit that aren't obvious from the surface description.

**Out of scope (do not cover):**
- A specific tech stack recommendation (which language/framework/DB to use).
- A specific implementation plan / code architecture.
- A specific marketing plan.
- A specific pricing plan (numbers, tiers).
- A full list of every minor agent (Aider, Cody, Sourcegraph, etc.) — only the ones the user explicitly named and the dominant ones.
- Speculation about what Anthropic will do in 2027.
- Detailed comparison of cloud storage providers (R2 vs S3 vs GCS).

**Borderline areas:**
- Cloud agents (Devin, Replit Agent, Continue Cloud Agents) — these are "cloud sessions" too, but in a different sense. Mention briefly as adjacent, don't deep-dive.
- The MCP protocol itself — only touch on it where relevant to the sync problem.
- Security/vulnerability topics (prompt injection via CLAUDE.md) — relevant because cross-device sync can move attack surface, but the user didn't ask about security. Flag as a risk to consider, don't deep-dive.

## 4. Question type and capabilities needed

This question leans most heavily on:

- **Critical annotation (HIGH)** — the user has a specific idea and a specific positioning, and needs honest pushback on where it's redundant, where the vendor is already winning, and where the actual gap is. Use this everywhere there's a claim like "we sync sessions" — ask: yes, but the vendor already does, and a third-party is a worse version of that for paid users.

- **Non-obvious insight (HIGH)** — the technical landmines (path-slug collision, format instability, secrets, retention, hook-based capture) are not obvious from a high-level description. The user said "essentially we're just storing sessions in S3" — that framing is wrong in important ways that need to be surfaced. Use this to make the answer feel worth reading.

- **Framework building (HIGH)** — give the user a mental model for thinking about scope (which vendors, which artifacts, which layer of capture) and for thinking about vendor trajectory (where each vendor is heading, who will own what). Use this to give the answer structure beyond a list of facts.

- **Scenario forking (MEDIUM-HIGH)** — the user is implicitly choosing a position. There are at least 3-4 defensible positions: (a) narrow Claude-Code-only with the best UX, (b) multi-vendor cloud bridge for the long tail of tools, (c) the memory/skills/agents layer (not sessions), (d) the orchestration/observability layer (a dashboard of all your agent sessions). Lay out the branches so the user can pick.

- **Causal reasoning (MEDIUM)** — explain *why* Anthropic's --teleport strategy is a particular threat (because it's free for paid users, integrated into the CLI, has GitHub integration built in), and *why* the path-slug collision happens. Use this to make the analysis feel earned, not asserted.

- **Comparative analysis (MEDIUM)** — the competitive map needs a real comparison, not just a list. Compare on: scope (how many tools), layer (config vs memory vs session vs orchestrator), business model (free OSS vs cloud), traction (stars/users/ARR). Use this for the "what to actually compete with" section.

- **Data-driven argumentation (LOW-MEDIUM)** — there are some good numbers (ECC 100K stars, Devsynq 14+ IDEs, market $7.37B in 2025, Cursor $3.4B valuation) that should anchor claims. Use sparingly to avoid the report feeling like a market-research deck.

- **Historical analogy (LOW)** — could draw analogies to dotfile managers (chezmoi, GNU Stow), password manager 1Password (local-first, cloud sync, encryption), or to early SaaS vs OSS battles. Use only if it sharpens a point, not as decoration.

## 5. Key facts and verification checklist

The following are facts I've gathered and want to double-check in Step 4 (most are already well-sourced, but some are time-sensitive or come from a single source):

- **Anthropic's --teleport feature is real and live** (high confidence, multiple sources, docs link). Cross-check: is the handoff still one-way from CLI? Does the Desktop "Continue in" menu still allow push? Does it require Max plan?
- **ECC has 100K+ stars and is the dominant config framework** (high confidence, multiple sources). Cross-check: is it really 100K at the time of writing (2026-07-24), or is this a moving target? Is it 28 agents / 119 skills / 60 commands or 125 skills?
- **Devsynq supports 14+ IDEs including CLI tools** (medium-high confidence, single creator Harjot Singh Rana, GitHub link). Cross-check: is Devsynq's "Dev Sync" the same as the user's "Dev Sync"? Probably not — the user said MCP-only at first, and Devsynq is also MCP-focused but has pivoted. Different products.
- **claude-sync uses age encryption and R2/S3/GCS** (high confidence, GitHub README). Cross-check: is it actively maintained as of mid-2026? How many users (npm downloads)?
- **agentsync by spxrogers covers 31 agents** (high confidence, single GitHub source). Notable: it explicitly says "agentsync is single-machine. To sync `~/.agentsync/` across machines, use chezmoi" — this is a real, unfilled gap.
- **Claude Code session JSONL format is unstable** (high confidence, official docs). Cross-check: has Anthropic actually published a changelog for it, or is the warning just a warning?
- **Grok Build uploads entire repos to `grok-code-session-traces` GCS bucket** (high confidence, security researcher wire-level analysis). This is a privacy/positioning fact — Grok is a hostile territory for "local session" sync, but also a huge market.
- **Claude Code uses path-encoded slugs with documented collisions** (high confidence, multiple sources). Cross-check: the algorithm is documented in the BasedGPT/Claude Code session recovery tool — `:` `\` `/` `.` ` ` all become `-`.
- **Cursor at $3.4B valuation, $3.4B raised** (high confidence, market data). Cross-check: as of Nov 2025, $3.4B raised, $2.3B Series D in Nov 2025. Replit $872M raised. Anysphere (Cursor) is the breakout.
- **GitHub Copilot cloud agent is live on mobile** (high confidence, GitHub docs).
- **Claude Code on the web, scheduled tasks, auto-memory, Cowork on mobile, "Continue in" menu, dispatch, channels (Telegram/Discord)** (high confidence, multiple sources, Q1 2026 changelog).

## 6. User profile

The user is a **technical builder / developer** with the following characteristics:
- **Hands-on expert** with MCP, Claude Code, Codex, Gemini, opencode, Grok, coding agents in general. Uses the same tools on Windows desktop and MacBook daily.
- **Self-deprecating, casual tone** — "scratching my own itch", "I am going to build it anyways", mixed Hindi-English idiom. Comfortable with informal speech.
- **Not a strategy/marketing person** — has framed the project as "essentially just storing sessions in S3" (a developer's framing, not a product manager's). Hasn't asked about pricing, market size, or positioning. Needs those to be surfaced.
- **Has already committed** to building. Doesn't need motivation, needs a sharper version of the plan.
- **Probably an indie hacker or solo founder**, or a senior engineer with side-project energy. Not a corporate PM.
- **Wants a thinking partner**, not a survey. Wants me to push back where needed and surface the non-obvious things they'll hit.
- **Likely small existing footprint** — mentions "Dev sync" with a small "d" and lower-cased branding, which suggests it's a personal project, not yet a public product. No indication of users, revenue, or distribution.

How this should influence the answer:
- Skip the basics. Skip the "MCP is a protocol that..." lecture. Skip the "cloud is great for..." primer.
- Don't hedge. The user is technical and can handle directness.
- Use the same casual register (without the Hindi idiom — the language of the conversation is English, and the user asked in English).
- Surface the 2-3 non-obvious landmines that will save them weeks of work, and the 1-2 non-obvious strategic insights that will change the product.
- Give a clear recommendation at the end. Not "here are 5 options, you choose." Pick the most defensible one and say why.
- Don't write a 5000-word market-research report. A focused 1500-2200 words is the right length.

## 7. Research direction and search strategy

The background is already strong. The remaining research in Step 4 should be **targeted gap-filling**, not broad coverage.

**Gaps to fill:**
- Is there a real, working product that does *exactly* what the user is describing (multi-vendor session sync, not just MCP-config sync)? If yes, what's its traction?
- What's the actual npm/HF download count for the existing tools? (claude-sync, agentsync, coding-agent-sync, Memora)
- Does Anthropic's --teleport have a "push from local" mode? (I've read it's one-way from CLI but Desktop has "Continue in" — confirm)
- Is there an explicit "session cross-device" feature request with significant community engagement on any of the agent GitHub repos?
- What's the current state of the ECC "sessions" feature — does it sync sessions across machines, or only config?
- How does GitHub Copilot cloud agent's mobile flow actually work? (deep enough to compare to what the user wants to build)

**Search angles (in priority order):**

1. "Claude Code --teleport push from CLI 2026" — confirm one-way limitation
2. "coding agent session sync stars github" — find any tool doing exactly this
3. "agentsync downloads npm pypi" — see if any of these have real traction
4. "ECC sessions cross machine" — does the dominant config framework already cover this
5. "Omnara cloud session mobile" — the closest direct competitor
6. "Anthropic claude code cross-device feature request" — community signal
7. "sync coding agent sessions reddit hacker news" — community demand signal
8. "agentsync single machine chezmoi" — the explicit gap

**Evidence to capture per subtopic (Step 4 handoff):**
- For each existing tool, capture: name, owner, what it syncs, what it doesn't, business model, traction signal.
- For Anthropic's cloud, capture: what's covered, what requires Max plan, what's one-way, what's the open gap.
- For each vendor, capture: where they store sessions, what format, whether the format is stable, whether they have a first-party cloud solution.

## 8. Specific writing guidance

**Reader**: developer/founder who already knows the space. Treat as a thinking partner, not a customer.

**Length**: 1500-2200 words. Long enough to actually argue, short enough to read in one sitting.

**Style**: research-report-meets-conversation. Direct, opinionated, willing to push back. Not a survey. Not a market deck.

**Structure** (suggested, not handcuffs):
1. **TL;DR / Verdict** — 1-2 paragraphs. The bottom line.
2. **The vendor headwind** — Anthropic is already doing this; lay out exactly what they cover and where the gap is.
3. **What you're actually competing with** — competitive map. Position the 4-5 closest tools.
4. **The technical landmines** — format instability, path-slug collision, secrets, hook-based capture. The things that will bite you.
5. **Scope: what to actually build** — the single most important choice, with a recommendation.
6. **The pivot question** — is "Dev Sync" the right name and the right framing, or should you go a different direction?
7. **What I'd actually do** — concrete next step.

**Insights that MUST be in the final answer (at least 5)**:
1. Anthropic's --teleport + --cloud + scheduled tasks + Cowork already does most of this for Claude Code, free for Max users. The third-party tool must explicitly target either (a) non-Anthropic agents, (b) the long tail of users without Max plans, or (c) the multi-vendor glue.
2. The user's framing of "just storing sessions in S3" is the wrong abstraction layer. The right layer is either (a) capture sessions in a vendor-neutral format at the hook/SDK level, or (b) be a thin file-mover that doesn't try to parse JSONL content. Naive file-movers that parse JSONL will break on every Claude Code release.
3. The path-slug collision is a real, documented bug-class. A sync tool that uses the directory name to identify the project will conflate two unrelated real paths. Must read `cwd` from inside the transcript.
4. The Devsynq / agentsync / coding-agent-sync / claude-sync / mcp-sync cluster is a fragmented market with 15+ active tools. Another "MCP config sync" tool is not a product. A *focused, well-executed* session sync for 2-3 vendors is.
5. The skills/agents/rules layer is the highest-leverage pivot from "Dev Sync" — not sessions. The market for synced skills/agents is bigger, less commoditized, and the artifacts are stable (markdown) instead of fragile (JSONL). The user mentioned "skills" in passing but probably hasn't thought through that this might be a better bet than sessions.
6. agentsync explicitly says it's single-machine and tells users to use chezmoi. That's a real, unfilled gap — a multi-vendor, multi-machine, encrypted sync that just works. If the user can deliver that for sessions, they've found the position.
7. Grok Build is actively hostile territory — it uploads repos to GCS by default. Trying to "sync Grok sessions" without addressing that first means shipping a privacy nightmare. The Grok market is real (people use it) but the legal/ethical question is sharp.
8. The "Dev Sync" name overlap with Devsynq is a real problem. If the user goes to market with that name, they'll be confused with Harjot Singh Rana's MCP-sync tool. Pick a different name.

**Concluding statements that show depth:**
- "The market doesn't need another MCP config sync tool. It needs one well-executed session sync that actually beats the vendor cloud for one specific use case."
- "The real competitor is not claude-sync or Devsynq. It's `claude --teleport` and what Anthropic ships next quarter."
- "The pivot from MCP-config to universal-session is correct in spirit (multi-vendor, multi-device) but probably wrong in target (sessions are fragile, skills/rules are durable)."

**Shallow-answer patterns to avoid:**
- "Great question! Let me first explain what MCP is..."
- "The market is $7.37B, projected to $23.97B by 2030..." (market-size deck filler)
- "There are pros and cons to each approach, ultimately the decision is yours..." (analysis avoidance)
- "Here's a list of 15 tools you should consider..." (a list is not analysis)
- "Build it, ship it, iterate!" (unearned enthusiasm)

**The single core takeaway** (if I could keep only one):
*Build for the multi-vendor, multi-machine gap, but pick a smarter artifact than raw session JSONL. Skills/rules/MCP configs are the durable bet; sessions are the marketing-friendly headline but the technical debt will be enormous.*

---

# Step 4 Handoff (Todo list for research)

I now have enough to convert into Step 4's research Todo list. Here's the handoff:

| # | Label | Subtopic | What to verify | Suggested keywords |
|---|-------|----------|----------------|---------------------|
| T1 | Anthropic --teleport one-way vs Desktop push | S1 | Confirm CLI is one-way pull; Desktop "Continue in" menu can push; Max plan requirement | "claude --teleport push local", "Continue in another surface", "Claude Max plan required teleport" |
| T2 | Community demand signal for cross-device session sync | S3 | Find any GitHub feature request / HN / Reddit thread with significant engagement for cross-device session sync that the vendors haven't shipped | "cross-device session sync feature request Claude Code", "Codex thread sync github issue", "AI coding session sync reddit" |
| T3 | Existing multi-vendor session sync tools (not just MCP) | S3 | Find tools that sync *sessions* (not just configs) across 2+ coding agents | "multi vendor session sync", "universal coding agent session", "synced across Cursor Claude Codex sessions" |
| T4 | agentsync traction + whether it covers sessions | S3,S6 | Confirm the "31 agents" claim; check downloads; confirm it explicitly tells users to use chezmoi for multi-machine | "agentsync spxrogers downloads", "agentsync chezmoi multi-machine", "agentsync 31 agents" |
| T5 | ECC sessions support vs config only | S3,S4 | Does the 100K-star config framework actually sync sessions across machines, or only config? | "everything-claude-code sessions", "ECC cross machine sync" |
| T6 | Omnara Cloud Sessions positioning and traction | S3 | The closest direct competitor — verify it does what the user wants, and how it's positioned | "Omnara cloud session migration", "Omnara local to cloud" |
| T7 | GitHub Copilot cloud agent mobile flow depth | S1,S3 | Confirm cross-device mobile flow; check limitations | "Copilot cloud agent mobile continuation", "github mobile continue session" |
| T8 | Per-vendor format stability / format docs | S2 | Has Anthropic / OpenAI / Google published a stable format for sessions, or all "internal and changes"? | "Claude Code jsonl schema documented", "Codex session schema", "opencode session stable format" |
| T9 | Hook-based session capture availability per vendor | S2 | Which agents expose hooks to capture session data in real time, vs file-based polling? | "Claude Code PreCompact hook", "Codex session end hook", "Gemini CLI hook event" |
| T10 | Dev Sync / Devsynq brand collision | S5 | Is the user's "Dev Sync" the same as Harjot Singh Rana's "Devsynq"? Is the name usable? | "Devsynq trademark", "Dev Sync vs Devsynq", "Harjot Singh Rana devsynq" |
| T11 | "Dev Sync" / "dev-sync" existing OSS projects | S5 | Is there a well-known existing project with the same name? | "dev-sync" github MCP, "DevSync" npm |
| T12 | Grep/Reddit/HN signal for "I built this and here's the traction" | S6 | Find the most-cited indie launches in this space to estimate what's possible | "claude-sync hacker news launch", "agentsync launch HN", "Omnara launch traction" |
| T13 | The "skills/rules as the real product" thesis verification | S1,S4 | Are skills/rules actually the bigger market than sessions? Evidence? | "Claude Code skills marketplace", "AGENTS.md adoption", "skills-sync traction" |
| T14 | Existing per-vendor cross-device workarounds that the user is solving for | S3 | iCloud+symlink, Syncthing, git+symlink, rclone+git — what % of users actually do this manually? | "iCloud Claude Code symlink", "Syncthing claude code", "rclone codex session" |

# Step 4: Execute the research Todo list

Now let me read Step 4 and execute.

The research understanding is complete. Now let me read Step 5 and write the final answer.

Now let me write the final answer.

Let me run the audit and make a few small fixes.

# Cloud Session Sync for Coding Agents

## 1. The Bottom Line

Build it, but the headline artifact (sessions) is the wrong one to center the product on. The vendor cloud already covers most of the user-visible pain for Claude Code subscribers, and the third-party tools that try to do "universal session sync" are landing in a narrow, contested lane. The MCP config sync pivot from "Dev Sync" is correct in spirit, but the higher-leverage move is to widen the artifact set to skills, rules, agents, commands, hooks, and MCP configs, and treat sessions as one feature among many. The Windows + Mac, two-OS, two-vendor case you actually live in is a real, under-served gap. Pick a fresh name while you're still small, and build the cross-machine layer for the durable artifacts (markdown files) rather than the fragile ones (per-tool JSONL transcripts).

The rest of this walks through why, and what to actually build.

## 2. The Vendor Cloud Coverage Today

Anthropic has been shipping, every quarter, more of the feature you described. `claude --teleport` pulls a cloud session into your local terminal and is documented as a one-way operation from the CLI: web → local, never local → web [1]. The Desktop app's "Continue in" menu does allow the reverse direction, but it requires a clean git tree and is bound to a single Anthropic subscription tier (Pro, Max, Team, or Enterprise) — API key users can't use it [2]. Anthropic also shipped `--cloud` (kick off a new cloud session from your repo), the `&` prefix for background cloud sessions, scheduled tasks ("Routines") running on Anthropic-managed infrastructure, auto-memory enabled by default, Cowork spanning web + mobile, Remote Control, and the Channels integration for Telegram/Discord [3].

The quietly decisive piece is the new SessionStore adapter in the Agent SDK: it lets a third party mirror Claude Code transcripts to S3, Redis, or a database so a session created on one host can be resumed on another [4]. Anthropic has explicitly opened the door to a vendor-neutral session backend, and the door is not closing.

OpenAI's Codex Cloud is similar: cloud threads with cross-device continuity, plus a community request for an "Import legacy sessions" path from local JSONL into cloud threads [5]. GitHub's Copilot cloud agent runs on a phone-started task and pushes a PR back to your repo [6]. xAI's Grok Build is the awkward case: it uploads your whole repo to a GCS bucket named `grok-code-session-traces` by default, and disabling "Improve the model" doesn't actually turn that off [7]. Any third-party tool that touches Grok sessions inherits that mess.

The practical effect: a "sync Claude Code sessions across my Mac and Windows" tool that runs alongside an Anthropic subscription is replacing a feature that is shipping now, in pieces, for free, with the explicit blessing of the vendor. The third-party tool has to be cheaper, faster, more private, more multi-vendor, or cover an actual gap (Windows + Mac, no Max plan, non-Claude agents, hybrid cloud + local control). Generic "store JSONL in S3" does not do that.

## 3. The Closest Direct Competitors

The space is more crowded than it looks, and the closest hits are not the MCP config tools.

**Depot.dev's `depot claude`** is a direct answer to the same problem you described. It wraps the local `claude` binary, watches the session JSONL on disk, uploads changes to the Depot API, and on resume fetches the transcript and writes it back to the right path before launching Claude. Sessions are named, shareable across a team, and resumable from any machine or CI environment [8]. Their blog post says it plainly: "We had the same problem." Depot is now also running remote agent sandboxes by default, so this is closer to a cloud coding runtime than a sync utility. If you ship a Claude-only session sync, you are head-to-head with Depot.

**Omnara** (YC S25) is a mobile and web command center for Claude Code and Codex. Free tier of 10 sessions per month, $20/month Pro, 250K+ agent interactions in its first week, 4.3/5 stars, and a "cloud migration" feature that moves a session to a remote sandbox when your laptop disconnects [9]. Omnara is positioned as the control plane, not the file sync, but from a user's perspective the outcome is similar: same session, different surface.

**claude-sync** (tawanorg) is the most cited open-source option: it syncs the entire `~/.claude/` directory to Cloudflare R2, AWS S3, or GCS using age encryption, with the same passphrase on every device producing the same key [10]. **cc-sync** (ikook-wang) and **claude-code-sync** (perfectra1n) are git-backed variants with similar scope [11][12]. All three are single-vendor (Claude Code only).

For config-only, the market is fragmented past the point of usefulness: Devsynq (HarjotSinghh), at least three projects named "agentsync" (dallay, spxrogers, yelmuratoff), `agents` (amtiYo), `coding-agent-sync` (TCTinh), `mcpsync-cli`, `mcp-sync` (multiple forks), `agentsync-cli` (PyPI), and `skills-sync` [13][14][15][16]. The one named "agentsync" that explicitly calls out the cross-machine gap is spxrogers/agentsync, which tells users in its own README: "agentsync is single-machine. To sync `~/.agentsync/` across machines, use chezmoi (or any dotfile manager)" [14]. That is a real, self-acknowledged hole.

For context on the dominant non-OSS benchmark: Everything Claude Code (ECC) hit 100K stars with 28 agents, 125 skills, and 60 commands, all version-controlled in a single repo, and it explicitly uses `git` + a `setup-sync.sh` script for multi-machine install [17]. ECC's pattern is essentially "put the agent state in a git repo and symlink it in." It works. It also doesn't try to be a cloud product.

## 4. The Format and Path Landmines

Two landmines are easy to miss until you hit them.

**Format instability.** The Claude Code docs are explicit: "The entry format is internal to Claude Code and changes between versions, so scripts that parse these files directly can break on any release" [18]. Codex, Gemini, opencode, and Grok have their own storage layouts and no published schema stability guarantees. A sync tool that does anything smart with the transcript (summarization, search, structured export) is fragile by design. A sync tool that only moves file bytes and uses the sanctioned hook or SDK surfaces is durable.

**Path-slug collisions.** Claude Code derives a project directory name from the absolute working directory, replacing every non-alphanumeric character with `-` [19]. Because dash and dot both become dash, two unrelated real paths can slugify to the same directory [20]. A sync tool that identifies projects by the directory name will conflate them. The `cwd` field inside each transcript is the only unambiguous identifier. This is a real bug class, not a theoretical one.

A practical implication: if you build around hooks instead of file parsing, you dodge both problems. Claude Code's hook system is mature (SessionStart, SessionEnd, UserPromptSubmit, PreToolUse, PostToolUse, PreCompact, PostCompact, Stop, SubagentStop, Notification) [21]. Codex CLI ships hooks in beta with the same lifecycle (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop) gated behind `codex_hooks = true` [22]. Gemini CLI's hook surface is even wider (SessionStart, SessionEnd, BeforeAgent, AfterAgent, BeforeModel, AfterModel, BeforeToolSelection, BeforeTool, AfterTool, PreCompress, Notification) and supports OpenTelemetry for streaming events out [23]. Across the three biggest agents, a real-time, vendor-shaped event stream exists. Using that stream as the source of truth is more durable than parsing JSONL after the fact.

## 5. The Brand and the Pivot Question

The name "Dev Sync" has a real collision: Devsynq (one word, by HarjotSinghh) is the most visible MCP config sync tool on the market, with 14+ IDEs, a marketplace of 575+ MCP servers, and CLI support shipped earlier this year [13]. Beyond that, GitHub has at least three repos named `devsync` or `dev-sync` with varying degrees of activity (maxime-aknin/devsync, troylar/devsync with 21 releases, jrTilak/dev-sync archived) [24]. None of these are blocking you legally — the names are common English words and there is no registered trademark that a search surfaces — but you will be confused with them in search and conversation.

Since you don't have an existing user base to migrate, the rename cost is zero. Pick a name that signals "skills + sessions + configs across machines" rather than "MCP config sync," because the second is now a saturated, low-margin category.

## 6. The Strategic Question

The thing the user wants — sync every artifact a coding agent touches, across machines and OSes, without losing context — is real. The framing of "essentially we're just storing sessions in a bucket" undersells it. The full artifact set across the major agents is:

| Category | Claude Code | Codex | Gemini | opencode | Grok | Format |
|---|---|---|---|---|---|---|
| Sessions | `~/.claude/projects/<slug>/*.jsonl` | `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` | `~/.gemini/tmp/<hash>/chats/` | `~/.local/share/opencode/storage/` | `~/.grok/sessions/<encoded>/<uuid>/` | Fragile, format-unstable |
| MCP config | `~/.claude.json` | `~/.codex/config.toml` | `.gemini/settings.json` | `~/.config/opencode/` | varies | Stable, schema-published |
| Skills | `~/.claude/skills/` | `.codex/skills/` | `~/.gemini/skills/` | `~/.config/opencode/skills/` | limited | Markdown, stable |
| Agents / subagents | `~/.claude/agents/` | `.codex/agents/` | `~/.gemini/agents/` | varies | limited | Markdown, stable |
| Commands | `~/.claude/commands/` | `.codex/prompts/` | `~/.gemini/commands/` | varies | limited | Markdown, stable |
| Rules | `~/.claude/rules/` | `AGENTS.md` | `GEMINI.md` | `AGENTS.md` | `AGENTS.md` | Markdown, stable |
| Hooks | `settings.json` hooks block | `hooks.json` (beta) | `settings.json` hooks | varies | limited | Declarative JSON |
| Memory | `~/.claude/projects/<p>/memory/` | `~/.codex/memories/` | `~/.gemini/GEMINI.md` | varies | varies | Mixed |

The bottom half of this table is the durable, high-leverage pivot. Skills launched in October 2025 and the ecosystem is now in the thousands; Anthropic's official marketplace has 101 plugins as of March 2026, with 33 Anthropic-built and 68 partner; ECC alone has 125 skills; community indexes report over 14,000 indexed plugins across marketplaces; the Firecrawl skill alone claims 110K weekly installs across Claude Code, Codex, and Gemini CLI [25][26]. The top half (sessions) is what users will ask for first because it's the obvious, attention-grabbing feature, but it is the wrong center of gravity because the underlying data is fragile, the format will change, and the vendor cloud is converging on the same job.

The honest answer to "what should I sync" is: sync the stable stuff first (skills, rules, agents, commands, MCP configs, hooks), use hooks to optionally capture session metadata, and stay away from reproducing raw transcript content as a primary feature. When Anthropic, OpenAI, and Google converge on cloud session continuity, a third-party that re-implements that is racing the vendor. A third-party that gives you a portable, multi-vendor, multi-OS home for the agent state that the vendors don't yet treat as cloud-native is not.

## 7. The Three Defensible Positions

### The Universal Session Sync Wrapper

Cover all major agents (Claude Code, Codex, Gemini, opencode, Cursor, Copilot, Grok, Antigravity) with one CLI, hooks for capture, R2/S3/GCS for storage, end-to-end encryption, and a per-vendor adapter layer. This is the literal version of your stated idea. It is technically possible, and the architecture is the same as claude-sync but for eight vendors. The problem is scope: each vendor has a different storage layout, a different hook system, a different retention policy, and a different vendor-cloud roadmap. Covering all of them in v1 is impossible; covering 2–3 well is a real product. The market for "I want my sessions on all eight agents synced" is much smaller than the market for "I want my skills and rules to follow me."

### The Skills, Rules, and Agents Center

Rebrand "Dev Sync" as a multi-vendor, multi-machine home for the durable agent artifacts. The store is a git repo or an S3 bucket; the per-vendor output is the agent's native config layout. The CLI is a single `devsync sync` that handles MCP configs, skills, agents, commands, rules, hooks, and optionally recent sessions. Hook-based capture for session summary, not full transcript; per-agent retention rules; cross-OS path remapping; and an open marketplace on top. This is the position that picks up the existing Dev Sync distribution (MCP-only users), widens to skills (a much bigger market), and doesn't fight the vendor cloud on its strongest ground. ECC at 100K stars proves the demand for "a structured layer of skills/rules/hooks/memory across tools" [17]. Nobody has shipped the cross-machine version of that yet.

### The Dev-Environment-as-Code Path

Pure OSS, no cloud, no SaaS. A chezmoi / GNU Stow for AI dev environments. The unit is a dotfiles-style repo; the CLI is `devsync apply` and `devsync diff`; the cloud story is "bring your own git and your own S3 if you want." Lightweight, composable, owned by the user. The market is small but the moat is honesty: a tool that doesn't try to host anything is hard to disrupt by anyone, including the vendors. The downside is that nobody gets paid, and the "scratching your own itch" energy fades after six months unless there is a path to a paid tier.

## 8. The Recommended First Move

Build a multi-vendor, multi-machine sync that treats skills, rules, agents, commands, MCP configs, and hooks as first-class, and session history as an opt-in feature with a 30-day default retention and clear warnings that the format is unstable. Use the hook systems to capture session metadata (working directory, project, duration, summary), not full transcripts. Stay session-light: don't try to out-cloud Anthropic. Ship R2/S3/GCS as a convenience, with chezmoi-style git as the default. Pick a name that isn't "Dev Sync" and isn't "Devsynq." Cover Claude Code and Codex in v1 with hooks-first adapters; add Gemini in v1.5 because its hook surface is wider than Codex's and the parity comes for free. Skip Grok until the privacy story is clean, or ship a Grok adapter that strips uploaded content before it touches the sync store. Treat the format-instability landmine as a design constraint, not a problem to solve: every adapter declares which fields it depends on, and you ship a fuzz test that fails when a vendor changes the schema.

The opportunity is not "store sessions in S3." The opportunity is the durable, portable, multi-vendor home for everything else that an AI coding agent reads and writes. Sessions are a feature, not a product.

## References

[1] Claude Code Docs — Use Claude Code on the web: "From the CLI, session handoff is one-way: you can pull cloud sessions into your terminal with `--teleport`, but you can't push an existing terminal session to the web." https://code.claude.com/docs/en/claude-code-on-the-web

[2] Claude Code Docs — Feature availability: "These require signing in with a claude.ai account and are not reachable with an Anthropic Console API key or from a third-party provider." https://code.claude.com/docs/en/feature-availability

[3] LinkedIn — "Anthropic Releases 28 Features in Q1 2026" (Charlie Hills), covering Cowork on mobile/web, Remote Control, Scheduled Tasks, Auto-Memory, Channels, Dispatch, etc. https://www.linkedin.com/posts/charlie-hills_anthropic-shipped-28-features-in-q1-2026-activity-7446134478407684096-U-B9

[4] Claude Code Docs — Persist sessions to external storage: "A `SessionStore` adapter lets you mirror those transcripts to your own backend, such as S3, Redis, or a database, so a session created on one host can be resumed on another." https://code.claude.com/docs/en/agent-sdk/session-storage

[5] OpenAI Community — "Legacy Codex local JSONL session import into cloud-synced Codex desktop." https://community.openai.com/t/legacy-codex-local-jsonl-session-import-into-cloud-synced-codex-desktop/1379835

[6] GitHub Docs — "Using Copilot cloud agent on GitHub Mobile." https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/use-cloud-agent-on-mobile

[7] Cereblab gist — Wire-level analysis of Grok Build CLI 0.2.93: "The storage destination is a Google Cloud Storage bucket, `grok-code-session-traces`." https://gist.github.com/cereblab/dc9a40bc26120f4540e4e09b75ffb547

[8] dev.to / Depot — "Now available: Claude Code sessions in Depot." https://dev.to/depot/now-available-claude-code-sessions-in-depot-33kd

[9] Y Combinator — Omnara company page: "Free: up to 10 agent sessions per month. $20/month: unlimited sessions." https://www.ycombinator.com/companies/omnara

[10] GitHub — tawanorg/claude-sync, README. https://github.com/tawanorg/claude-sync

[11] GitHub — ikook-wang/cc-sync, README. https://github.com/ikook-wang/cc-sync

[12] GitHub — perfectra1n/claude-code-sync, README. https://github.com/perfectra1n/claude-code-sync

[13] LinkedIn — Harjot Singh Rana, "CLI tool support in Devsynq." https://www.linkedin.com/posts/harjjotsinghh_new-feature-shipped-cli-tool-support-in-activity-7415222034223259649-Bkr4

[14] pkg.go.dev — github.com/spxrogers/agentsync: "agentsync is single-machine. To sync `~/.agentsync/` across machines, use chezmoi (or any dotfile manager)." https://pkg.go.dev/github.com/spxrogers/agentsync

[15] GitHub — dallay/agentsync (Rust, 73 releases, latest v1.46.2 May 24 2026). https://github.com/dallay/agentsync

[16] GitHub — yelmuratoff/agent_sync, README (11 tools, 265 commits). https://github.com/yelmuratoff/agent_sync

[17] Augment Code — "Everything Claude Code hits 100K stars." https://www.augmentcode.com/learn/everything-claude-code-github

[18] Claude Code Docs — Manage sessions: "The entry format is internal to Claude Code and changes between versions, so scripts that parse these files directly can break on any release." https://code.claude.com/docs/en/sessions

[19] Claude Code Docs — Work with sessions: "Sessions are stored under `~/.claude/projects/<encoded-cwd>/*.jsonl`, or under `$CLAUDE_CONFIG_DIR/projects/<encoded-cwd>/*.jsonl` if you set the `CLAUDE_CONFIG_DIR` environment variable, where `<encoded-cwd>` is the absolute working directory with every non-alphanumeric character replaced by `-`." https://code.claude.com/docs/en/agent-sdk/sessions

[20] Nate J Swenson — "Parsing the session logs Claude Code warns you not to parse": "Two unrelated projects can collide on the same session directory name." https://natejswenson.com/devlog/ghostwriter/v0.6.0/

[21] Claude Code Docs — Hooks reference (SessionStart, SessionEnd, UserPromptSubmit, PreToolUse, PostToolUse, PreCompact, PostCompact, Stop, SubagentStop, Notification). https://code.claude.com/docs/en/hooks

[22] Codex CLI — Hooks feature behind `codex_hooks = true`, events: SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop. https://hjarni.com/docs/codex-hooks

[23] Gemini CLI — Hooks reference (SessionStart, SessionEnd, BeforeAgent, AfterAgent, BeforeModel, AfterModel, BeforeToolSelection, BeforeTool, AfterTool, PreCompress, Notification) and OpenTelemetry support. https://geminicli.com/docs/hooks/

[24] GitHub — troylar/devsync (21 releases, v0.15.0 Feb 22 2026); jrTilak/dev-sync (archived). https://github.com/troylar/devsync

[25] Anthropic — claude-plugins-official, "101 plugins (33 Anthropic-built, 68 partner) as of March 2026." https://github.com/anthropics/claude-plugins-official

[26] Firecrawl — "Best Claude Code Skills to Try in 2026," citing "over 110k weekly installs across Claude Code, Codex, and Gemini CLI" for the Firecrawl skill. https://www.firecrawl.dev/blog/best-claude-code-skills
