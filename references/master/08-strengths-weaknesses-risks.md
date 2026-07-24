# Strengths, Weaknesses & Risk Register — Reinstate

## Product strengths (why build)

| Strength | Evidence |
|----------|----------|
| **Validated multi-device pain** | FRs across Claude, Codex, Gemini/Antigravity; Reddit multi-PC threads |
| **Empty quadrant** | Universal × offline stored sync × path-aware × encrypted is under-owned |
| **Vendors won’t multi-agent sync** | Structural moat; each vendor locks its own cloud |
| **Demoable magic moment** | Windows session → `reinstate` → Mac resume in a Loom |
| **DevSync heritage** | Capability plane (MCP/skills) already part of the idea; differentiates from transcript-only tools |
| **Friendly technical substrate** | JSONL/append logs, hooks, SessionStore, OpenCode APIs, proven claude-sync patterns |
| **Modest user asks** | Storage, export/import, path mapping — plumbing, not multiplayer magic |
| **Timing** | Agent CLIs scaled; vendors educating market on continuity |
| **Name fit (Reinstate)** | Verb for restore-to-usable-state; CLI-native (`reinstate resume`) |
| **Personal founder ICP** | Exact Windows + MacBook + multi-agent daily use |

---

## Product weaknesses / hard truths

| Weakness | Implication |
|----------|-------------|
| **Same-agent sync is being absorbed** | OpenAI already; Anthropic marching; pure Claude-only sync is a trap |
| **Transcript-only is table stakes** | SpecStory, SessionStore, claude-sync, Depot already exist |
| **Format churn** | Claude docs: parsers break across versions |
| **Path / dirty-state complexity** | Hardest engineering; easy to ship a broken “sync” |
| **Cross-agent replay is brittle** | Over-promising burns trust |
| **Trust bar is extreme** | One leak kills the brand |
| **Maintenance tax of N adapters** | Omnara OSS wrapper archived for this reason |
| **Small early paying market** | Power-user intersection; not automatic venture scale |
| **Name collision** | `reinstate.app` (Shopify appeals) — different market, TM work needed |
| **“Reinstate” HR/account connotation** | Some users hear “unsuspend account” first — tagline must teach |

---

## SWOT (compressed)

### Strengths
Universal multi-agent scope · path remapping as wedge · E2EE/BYO · capability-aware resume · open inspectable local core · founder dogfood

### Weaknesses  
Adapter surface · format instability · dirty workspace hard · marketing vs vendor free features

### Opportunities  
Neutral Switzerland of agent wars · ACP alignment · open adapter ecosystem · teams handoffs · skills marketplace adjacency

### Threats  
Vendor native path-aware local sync · SpecStory expansion · Depot/Omnara productization · trust incident · DIY Syncthing “good enough”

---

## Risk register (ranked)

| Risk | Severity | Evidence | Mitigation |
|------|----------|----------|------------|
| **Vendor ships native same-agent sync** | High per-agent; low if multi-agent | Codex account sync; Claude cloud/teleport/SessionStore; FRs cite OpenAI | Multi-agent + profile + BYO E2EE + path-aware offline; never Claude-only |
| **Secrets / source leak** | Existential | Transcripts + auth files | E2EE, redaction, denylist, published model |
| **Corrupt local session store** | High reputation | Users treat history as irreplaceable | Atomic writes, backups, dry-run first pull |
| **Format churn breaks adapters** | Medium recurring | Claude format warning; SQLite versions | Defensive parse, fixtures, hooks-first where possible |
| **“Just use Syncthing/OneDrive”** | Medium | Documented DIY | Sell 2-min setup vs afternoon of junctions; path rewrite; secret safety |
| **Market too small for venture** | Medium | Sync tools low stars vs CLIs | OSS-first; indie open-core; control-plane ambition only if evidence |
| **ToS / trademark friction** | Low–medium | Local user-owned files OK; `reinstate.app` exists | No unofficial branding as vendor; TM search; read-only disk access |
| **Scope explosion (12 agents + UI)** | High execution | Orchestrator tarpits | Phase gates; 2 agents first |
| **Closed binary trust failure** | High if closed-source-first | Power-user audience | Open-source local engine (recommended) |
| **Grok privacy inheritance** | Medium | GCS trace upload claims | Delay adapter or strip |
| **Over-promise cross-agent replay** | High brand | Schema divergence | Three modes; handoff default for cross-agent |
| **Remote Control dependency** | Medium | Origin-online, timeouts, gated | Not core path |

---

## Minority / dissenting strengths of alternate strategies

| View | Strength of that view | Why Reinstate still differs |
|------|----------------------|------------------------------|
| **DeepSeek: build own agent** | Avoids reverse-engineering | Huge feature race vs vendors; Reinstate is meta-layer |
| **MiniMax: durable artifacts first** | Format-stable, less vendor contested | Reinstate can center profile while using sessions as hook |
| **Claude-only clone of claude-sync** | Faster ship | No moat vs Anthropic + existing OSS |

---

## What “good” looks like (success criteria)

1. Cross-OS same-agent resume works reliably for Claude + Codex  
2. Doctor reports missing MCP/skills/workspace mismatch before broken resume  
3. Zero plaintext session content on any hosted path by default  
4. Strangers complete first success in &lt;5 minutes  
5. Survives agent version upgrades without data loss  
6. Users describe Reinstate as *restore workspace*, not *Dropbox for chats*
