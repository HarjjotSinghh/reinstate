# Problem Definition

**Product:** [Reinstate](./10-naming.md) — restores coding-agent workspaces across devices.

## The user story (foundational)

- Work all day on a **Windows desktop** across many agent sessions and projects.
- Switch to a **MacBook** (e.g. evening / bed) using the same coding agents.
- Want to **continue the exact session** — full conversational and tool-call context — not only `git pull` + a fresh agent.
- Machines are configured differently: different MCP servers, skills, paths, shells, credentials.

Original framing (all sources): sessions are “threads of messages, tool calls, tool outputs, decisions, and reasoning” trapped on the machine of origin.

---

## Why the transcript is the asset

A session is **not** a consumer chat log. It is an execution record:

| Contained in typical session stores | Why it matters on resume |
|-------------------------------------|---------------------------|
| User prompts & model responses | Continuity of intent |
| Tool calls + inputs/outputs | What was already tried |
| Reasoning / thinking blocks | Decisions & rejected paths |
| Token usage / compaction events | Cost & context shape |
| `cwd`, branch, sometimes git snapshots | Environment anchors |
| Skills/MCP invocations | Capability history |

**What git cannot restore:**

- Why approach B was rejected  
- Which files the agent already read three times  
- Mid-thread style constraints (“never touch the legacy migration path”)  
- Orphaned task / subagent IDs  
- The user’s half-remembered conversation state  

Re-deriving that from a diff is impossible; re-establishing it costs **tokens + user attention** (user must remember what they remember).

Gemini deep research names this the **“developer retrieval problem”** and argues AI context must be treated as a **portable continuous asset**, not a single-node cache.

---

## Why “session ≠ safe continuation state”

Multiple sources stress the same correction:

| Layer | Examples | If missing on target machine… |
|-------|----------|-------------------------------|
| Conversation | Transcript / thread | Lost narrative |
| Workspace | Git HEAD, dirty files, patches | Agent reasons about wrong tree |
| Capability | MCP, skills, hooks, permissions | Tools fail mid-resume |
| Environment | OS, paths, package managers, WSL | Commands break |
| Provenance | Source agent, schema version | Wrong adapter / silent corruption |

Claude docs (cited widely): sessions persist **conversation**, not filesystem state.  
ACP `session/resume`: needs session ID **+ working directory + MCP server reconnection**, not just a chat blob.  
Kontinuo: checkpoints goal, stop point, changed files, dirty state, Git HEAD, workspace fingerprints.

### Dirty-state desync (Gemini.md “final boss”)

1. Agent edits files for hours on desktop; user leaves **uncommitted** changes.  
2. Session syncs to laptop.  
3. Laptop repo lacks those dirty files (or has different HEAD).  
4. Resumed agent “knows” about edits that **aren’t on disk** → hallucinated continuity / broken tools.

**Implication:** product must either:

- Sync / warn on dirty workspace (hard, privacy-sensitive), or  
- **Detect and surface** dirty/HEAD mismatch before resume (minimum bar), or  
- Prefer **portable handoffs** that restate ground truth rather than blind replay.

---

## Scope of agents in research

| Agent / surface | Mentioned by |
|-----------------|--------------|
| Claude Code (CLI, desktop, web, VS Code, JetBrains, mobile) | All major sources |
| OpenAI Codex (CLI, app-server, desktop, mobile, cloud) | All major |
| Gemini CLI | All major |
| OpenCode | Most |
| Grok Build / Grok Code / xAI CLI | Most |
| Cursor / Antigravity / Copilot CLI | Claude, Gemini deep, MiniMax, Kimi |
| Warp | ChatGPT, Claude adjacent |

**Founder case to optimize for first:** Windows desktop ↔ macOS MacBook, Claude Code + Codex, multi-project daily use.

---

## Related but different problems (do not conflate)

| Problem | Product that solves it | Not the same as… |
|---------|------------------------|------------------|
| Drive a **running** machine from phone | Happy, Remote Control, Omnara | Offline portable state |
| Many agents in parallel on **one** machine | Vibe Kanban, Conductor, orchestrators | Cross-device continuity |
| MCP config on **one** machine across tools | mcp-sync, vsync, original DevSync | Cross-device session store |
| Cloud **execution** (VM sandboxes) | Claude web, Depot sandboxes, Codex cloud | Local state portability |
| Session **search / history UX** | SpecStory, agent-sessions viewers | Sync plumbing |

---

## Timing why now

Sources converge on three crossing curves (esp. Kimi, Gemini deep research):

1. **Agent CLI scale:** Claude Code large ARR / commit share claims; Codex multi-million weekly users; Gemini CLI & Codex huge star counts (point-in-time 2026).  
2. **Multi-agent fragmentation:** Power users run 2–5+ agents; orchestrators advertise 10+ agent support. Single-vendor sync leaves them cold.  
3. **Vendors validating the behavior:** Partial native cross-device experiences ship → market education done; **universal** layer still missing.

Surface count keeps growing (CLI + IDE + desktop + web + mobile). Every new surface increases expectation that state “follows the account” — but local CLI transcript stores often remain outside vendor sync perimeters.
