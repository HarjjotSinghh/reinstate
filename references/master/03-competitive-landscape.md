# Competitive Landscape

**Product:** Reinstate aims at the empty quadrant — universal × cross-device × offline × encrypted × path-aware — not single-vendor native sync.

![Traction](../../assets/03_traction.svg)

## Four categories (Kimi framework)

| # | Category | What it solves | Offline if other machine off? | Multi-vendor? |
|---|----------|----------------|-------------------------------|---------------|
| **A** | Native vendor cloud | Same-agent continuity inside one ecosystem | Depends (cloud VM yes; local store no) | No |
| **B** | Remote-access relays | Drive a **running** local session remotely | **No** — origin must stay up | Partial |
| **C** | Same-machine config sync | MCP/skills across tools on **one** PC | N/A | Yes (config only) |
| **D** | Cross-device state sync | Portable local state | **Yes** (if stored sync) | Rarely (usually 1–2 vendors) |

**Empty / under-owned quadrant:**  
**Universal × cross-device × offline-capable × encrypted × path-aware** (sessions + config/skills).

---

## A. Native vendor cloud

| Product | What ships | Limits for third-party positioning |
|---------|------------|------------------------------------|
| **Claude Code web / cloud sessions** | Persist across devices; review from phone | Vendor-locked; different from local CLI store |
| **Claude `--teleport`** | Cloud → local transcript + edited files | Often **one-way** CLI pull; subscription-gated; not API-key path (MiniMax) |
| **Claude Desktop “Continue in”** | Reverse direction in some flows | Clean git tree requirements; tier-gated |
| **Claude Remote Control** | Live control of **running** local session from claude.ai/mobile | **Not stored sync**; process must keep running; session ends if terminal closes |
| **Claude SessionStore adapter** | SDK mirror transcripts to S3/Redis/DB for cross-host resume | Opens door for third-party backends — also reduces moat for “generic S3 mirror” |
| **Claude Routines** | Scheduled tasks; sync across CLI/web/desktop account | Account-scoped, not multi-vendor |
| **Codex cloud / ChatGPT surfaces** | Threads sync web/mobile/desktop; mobile loads live desktop state | Account-level; third party can’t replicate auth |
| **Codex App Server** | `thread/start`, `thread/resume`, `thread/fork`; fails if MCP missing | Local + remote surfaces; import from other agents exists |
| **GitHub Agent HQ** | Claude + Codex across GitHub/mobile/VS Code | Platform control plane, not neutral file sync |
| **Copilot cloud agent** | Mobile start → PR back to repo | GitHub-centric execution |
| **Grok Build** | Local session flags; default uploads traces to GCS `grok-code-session-traces` (research claim) | Privacy mess for third-party syncers |
| **Gemini CLI** | Mature **local** sessions (UUID resume, browser, checkpoints, retention ~30d) | Weak/no first-party cloud story at research time |
| **OpenCode** | Local sessions + headless HTTP session API | No first-party multi-device product |
| **Warp Drive** | Workflows, notebooks, prompts, env vars; MCP config sync **sans secrets** | Terminal product, not multi-agent universal |

**Incumbent risk summary:** Vendors win **intra-ecosystem** sync. They will not unify **competitors**. Design so that single-vendor absorption hurts but does not kill the product.

---

## B. Remote-access relays / control planes

| Product | Notes | Stars / traction (point-in-time) |
|---------|-------|----------------------------------|
| **Happy** | Mobile/web client for Claude Code & Codex; E2EE relay | ~22.8K ★ |
| **Omnara (YC S25)** | Mobile/web command center; cloud migration when laptop disconnects; free 10 sessions / ~$20 Pro claims | Original OSS wrapper **archived** Feb 2026 (CLI wrap unmaintainable) → proprietary SDK path |
| **Syncode** | iOS + macOS companion; LAN pairing | IAP |
| **Grok / community bridges** | e.g. Telegram remote-control tooling | Niche |
| **TokenRip** | Session state as URL-addressable artifact (`/a/<slug>`) | Different architectural bet: don’t sync files; permanent URL |

These answer: *“drive my home machine from my phone.”*  
They do **not** fully answer: *“state exists independent of any one machine.”*

---

## C. Same-machine cross-tool config sync (DevSync’s original neighborhood)

Crowded, shallow; ceiling evidence (mcp-sync ~46★).

| Product | Scope |
|---------|-------|
| **mcp-sync** (ztripez) | MCP across Claude Desktop/Code, Cline, Roo, VS Code, Cursor, Continue |
| **MCP Manager app** | ~9+ clients |
| **MCPHub.nvim** | Neovim centralization |
| **nicepkg/vsync** (~46★) | MCP/Skills/Agents/Commands across Claude/Cursor/opencode/Codex |
| **Leoyang183/sync-agents-settings** | MCP + CLAUDE.md → 12+ agents; drift detection |
| **baranovxyz / spxrogers agentsync** | Canonical `.agents/` → many agents; **spxrogers explicitly single-machine** (use chezmoi for multi-machine) |
| **neon-solutions/add-mcp** | One-command MCP install ~16 agents |
| **AgentSync (Rust / dallay)** | Symlink-style / multi-release project |
| **yelmuratoff/agent_sync** | Multi-tool |
| **skills-sync** | Skills + MCP workspace |
| **DevSynq (devsynq.app / HarjotSinghh)** | Commercial MCP/API-key sync, 14+ IDEs, marketplace 575+ servers, CLI |
| **Everything Claude Code (ECC)** | 100K★ skills/agents pack; multi-machine via git + setup script |

**Takeaway:** Config-sync alone is a **feature**. Sessions are the emotional hook; config is the glue and expansion surface.

---

## D. Cross-device session / state sync

| Product | Vendors | Notes |
|---------|---------|-------|
| **claude-sync** (tawanorg) | Claude only | age encryption; R2/S3/GCS/WebDAV; path-mapping thoughtfulness; ~229★; sessions-only scope option |
| **cc-sync**, **claude-code-sync** | Claude only | Git-backed variants |
| **Dinesh3184/claude-session-sync** | Claude | iCloud GUI |
| **renefichtmueller/claude-sync** | Claude | Multi-backend + snapshots/encryption |
| **coding-agent-sync** (TCTinh) | Claude + OpenCode | Config-first; sessions bolted on; AES Gist |
| **antigravity-storage-manager** | Antigravity | Google Drive + master password |
| **Depot `depot claude`** | Claude | Watches JSONL, uploads to Depot API, restores path before launch; team share; moving toward remote sandboxes — **direct Claude-only competitor** |
| **codex-tabs**, Codex Session Picker | Codex | Resume UX helpers |
| **Agent Sessions** (macOS apps / jazzyalex, kenn-io) | Multi | Local browsers/search — not sync |
| **npow/session-sync** | Cross | Niche context-transfer (partial) |
| **OpenSync** | — | Convex-backed (mentioned as hosted contrast for security) |
| **Amp** | — | Server-stored contrast for E2EE positioning |
| **iHildy/opencode-synced** | OpenCode | ~40–116★ inconsistent reports |
| **dantelex/aisync** | — | Traction unverified (Claude research) |
| **CodeVibe / codex-workspace-sync** | Codex | Spawned from Codex FR discussion |

**No clear winner** of universal path-aware encrypted multi-agent sync.

---

## Strategic / conceptual competitors (ChatGPT emphasis)

| Product | Why it matters |
|---------|----------------|
| **SpecStory** | Direct competitor: cross-project/agent search, portable sessions, retroactive sync, secret redaction, `specstory run` / `sync`, reconstruct for other tools |
| **CASR** (cross_agent_session_resumer) | Canonical intermediate representation; convert provider sessions; atomic writes; read-back verify — reference for Mode 3 migration |
| **Kontinuo** | Deeper handoff: goal, stop point, files, dirty state, Git HEAD, fingerprints — compact vs full transcript; **Mode 2 inspiration** |
| **ACP (Agent Client Protocol)** | Standardizing `session/resume` lifecycle; align rather than fight |

---

## Orchestration neighborhood (not competitors)

Vibe Kanban (~27.5K★), Conductor, Claude Squad, Nimbalyst (née Crystal), Superset, Paneflow, opcode, Mux, …

- Unit of value: **supervise many concurrent agents on one machine**  
- Your unit: **continuity of one developer’s state across machines**  
- They **compose**; some list paid sync as add-on (validates monetization)  
- Cautionary: Vibe Kanban company shutdown Apr 2026 despite stars → stars ≠ business

Steal: worktree isolation mindset for conflict semantics; session library UI as Phase-3, not v1.

---

## Competitive conclusion (ChatGPT)

No product clearly combines **all** of:

1. Cross-device continuity  
2. Multi-agent coverage  
3. Capability/config plane (MCP/skills)  
4. Environment/workspace validation  
5. Local-first E2EE + BYO storage  
6. Honest resume modes (native / handoff / experimental migration)  

Closest single threats: **SpecStory** (session product), **claude-sync** (OSS pattern), **Depot** (Claude operational), **vendor native sync** (absorbs same-agent).

---

## Positioning map (verbal)

```
                    SINGLE-VENDOR          MULTI-VENDOR
                 ┌────────────────────┬────────────────────┐
  LIVE RELAY     │ Remote Control     │ Happy (partial)    │
  (origin on)    │ Omnara             │                    │
                 ├────────────────────┼────────────────────┤
  STORED SYNC    │ claude-sync        │ ★ EMPTY / WEAK ★   │
  (offline OK)   │ Depot Claude       │ coding-agent-sync  │
                 │ Codex account      │ SpecStory (partial)│
                 └────────────────────┴────────────────────┘
```

**Reinstate** should own the bottom-right: path remapping + E2EE + profile (config/skills) + capability-aware resume.
