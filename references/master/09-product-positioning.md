# Product Positioning & Business — Reinstate

## Canonical positioning

> **Reinstate restores your coding-agent workspace on any device.**

Supporting lines:

- Resume any agent, anywhere — with the right conversation, repo state, and toolchain.  
- Verified continuity, not cloud chat storage.  
- Your agent workspace, carried forward and made ready to continue.

**Not:**

- “S3 for session JSONL”  
- “Dropbox for coding-agent transcripts”  
- “Cloud sessions for AI development” (vague AWS-sounding)  
- “Another Claude Code”  

---

## Product hierarchy (what Reinstate *is*)

1. **Synchronize capabilities** — MCP, skills, instructions, hooks, plugins, profiles *(DevSync heritage)*  
2. **Capture continuity** — sessions, checkpoints, decisions, tasks, artifacts  
3. **Validate reality** — repository, commit, working tree, environment, missing capabilities  
4. **Resume safely** — native same-agent → portable handoff → experimental migration  
5. **Protect everything** — local-first, E2EE, secret references, device control  

The storage bucket is plumbing.  
The product answers:

> *Can this agent, on this machine, in this repository, with these tools, safely continue the work that stopped elsewhere?*

---

## Relationship to DevSync / Devsynq

| Term | Meaning in this corpus |
|------|------------------------|
| **DevSync** | Original idea: centralized MCP config sync across coding tools |
| **Devsynq** | Existing commercial MCP sync product (name collision caution; founder-adjacent branding history in sources) |
| **Reinstate** | **Chosen product name** for the expanded platform: sessions + capabilities + verified resume |

Expansion, not a thin rename of “MCP only.” MiniMax warned against staying in the saturated MCP-sync category under a “Dev Sync” name.

---

## Competitive framing for landing pages

| vs | Message |
|----|---------|
| **Native vendor sync** | They sync *their* agent. Reinstate is universal and works when the other machine is off. |
| **Remote Control / Happy** | Those drive a live machine. Reinstate stores portable state. |
| **claude-sync** | Great for Claude-only. Reinstate is multi-agent + capability plane + path-aware restore. |
| **SpecStory** | Adjacent; Reinstate centers verified resume + environment/capability validation. |
| **Syncthing DIY** | Works until absolute paths and secrets bite. Reinstate is resume-aware and safe-by-default. |
| **Orchestrators** | They run many agents on one box. Reinstate continues *you* across boxes. |

Seed comparison content early: *Reinstate vs claude-sync vs native sync vs Syncthing*.

---

## Three defensible positions (MiniMax) — mapped to Reinstate

| Position | Reinstate stance |
|----------|------------------|
| Universal session sync wrapper | **Wedge / Phase 1–2**, not sole identity |
| Skills/rules/agents center | **Core platform layer** (durable artifacts) |
| Dev-env-as-code (chezmoi) | **Local-first / BYO mode** always available |

Default: hybrid — durable profile + session hook + checkpoints under one brand: **Reinstate**.

---

## Business models scored (synthesized)

| Model | Score | Notes |
|-------|------:|-------|
| Personal OSS tool | 10/10 | Guaranteed utility; dogfood |
| Indie open-core | 8/10 | Free core + paid cloud/teams |
| Venture on transcript sync alone | 2–3/10 | Vendors absorb |
| Open control plane for portable agent envs | ~7/10 | Needs adapter + trust moat |

Realistic early trajectory (Kimi): thousands–tens of thousands OSS users; hundreds of paid seats.

---

## Recommended commercial split (ChatGPT — post-name)

> **Open-source local engine and adapters. Proprietary hosted sync and commercial control plane.**

```text
Reinstate Core (OSS)     → tool: local, inspectable, BYO storage
Reinstate Cloud (product)→ automatic multi-device sync, managed storage
Reinstate Teams          → shared handoffs, RBAC, SSO, audit
```

### Why not closed-source-first

Audience will demand proof of redaction/E2EE. Closed binary + filesystem access = trust nightmare.

### Why not fully open everything day one

Hosted coordination, billing, device policy, abuse, ops — commercial surface; early churn of backend is operational tax if public.

**Tailscale-like pattern:** open client; managed coordination can stay proprietary initially.

### Free forever core should include

- Local discovery, search, doctor  
- Export/import, checkpoints  
- Adapters, profiles, workspace validation  
- Manual sync + user-owned storage  
- No account required for local-only mode  

### Cloud paid after free tier

- Auto cross-device sync  
- Managed encrypted storage  
- Device management, retention, recovery  
- Hosted indexing / notifications  

### Teams per seat

- Shared sessions/handoffs  
- Org profiles, RBAC, SSO, audit, compliance  

---

## Monetization sketches from other sources

| Source | Idea |
|--------|------|
| Kimi | OSS CLI marketing channel; charge for hosted layer day one of offering it |
| Claude research | Free core; paid managed storage, multi-machine dashboard, team profiles |
| DeepSeek | Team plans: shared sessions, central MCP, admin |
| Perplexity | Who pays: multi-machine power users; later teams |

---

## Metrics while dogfooding

| Metric | Insight |
|--------|---------|
| Desktop→laptop time-to-resume | Core value |
| READY / DEGRADED / BLOCKED rates | Capability plane quality |
| Conflict rate | Sync model |
| Adapter breaks per agent release | Maintenance |
| Secrets redacted | Security |
| Time to first success | Onboarding |
| % using BYO vs managed cloud | Monetization signal |

---

## Messaging do’s and don’ts

**Do:** restore, reinstate, workspace, verified, capability-aware, local-first, encrypted, any agent  
**Don’t:** perfect cross-agent memory, replace Claude/Codex, “just a bucket,” guarantee dirty-tree magic without validation  

---

## Build-in-public note (Kimi)

Audience lives on GitHub, X, agent subreddits. Narrative post (“deep in a session… switch machines… gone”) converts. Seed comparison SEO early.
