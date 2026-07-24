# Master Reference Library

**Synthesized from all raw research in `references/` (July 2026).**

This directory consolidates every major claim, competitor, technical detail, strength, weakness, risk, and recommendation from the original multi-model research corpus. Original files are left untouched under `references/`; this folder is the structured synthesis.

## What this product is about

**Product name:** **Reinstate** (`reinstate.dev`)  
**Short CLI alias:** **`rein`** (same binary as `reinstate`)  
**Lineage:** Expands the earlier **DevSync** MCP-config idea into sessions + skills + configs + verified resume.  
**Core problem:** AI coding-agent sessions (Claude Code, Codex, Gemini CLI, OpenCode, Grok Build, …) and their surrounding environment (MCP, skills, hooks, configs) are trapped on the machine where they were created. Multi-device developers lose reasoning context when switching machines.

**Canonical pitch:** *Reinstate restores your coding-agent workspace on any device.*

**Consensus verdict across sources:** **Yes, build Reinstate** — but not as naive “Dropbox for transcripts.” Build a **vendor-neutral, local-first control plane for verified continuity** (sessions + capabilities + workspace/environment validation).

## Directory map

| File | Contents |
|------|----------|
| [00-executive-summary.md](./00-executive-summary.md) | Unified verdict, disagreements, decision table |
| [01-problem-definition.md](./01-problem-definition.md) | Pain, why transcripts matter, dirty-state problem |
| [02-market-demand.md](./02-market-demand.md) | Issue trackers, DIY workarounds, traction signals |
| [03-competitive-landscape.md](./03-competitive-landscape.md) | Four quadrants, full competitor inventory |
| [04-vendor-capabilities.md](./04-vendor-capabilities.md) | What each vendor already ships |
| [05-session-formats-and-tech.md](./05-session-formats-and-tech.md) | Storage formats, path landmines, hard problems |
| [06-architecture-and-mvp.md](./06-architecture-and-mvp.md) | State model, resume modes, phased MVP |
| [07-security-and-trust.md](./07-security-and-trust.md) | E2EE, secrets, MCP risks, device trust |
| [08-strengths-weaknesses-risks.md](./08-strengths-weaknesses-risks.md) | SWOT + risk register |
| [09-product-positioning.md](./09-product-positioning.md) | Positioning, monetization, business models |
| [10-naming.md](./10-naming.md) | **Reinstate** decision, rejected alternatives, domain/TM notes |
| [11-build-recommendations.md](./11-build-recommendations.md) | Phased plan, what not to build, validation |
| [12-source-map.md](./12-source-map.md) | Every original file → what it contributed |
| [assets/](./assets/) | Landscape / demand / traction / market / architecture diagrams |

## How to use

1. Start with **00-executive-summary** for the decision.
2. Use **03**, **05**, **06**, **07** as implementation playbooks.
3. Use **08** and **11** for risk-aware scope.
4. Trace any claim back via **12-source-map** to raw sources under `../`.

## Synthesis method

- **Union of facts:** every competitor, path, format, and FR cited in any source appears here if unique.
- **Intersection of judgment:** “build / don’t build” and moat claims are marked by agreement level.
- **Preserved disagreements:** sources that recommend building a *new agent* vs *sync wrappers* vs *skills-first* are all retained, not collapsed.
- **No silent drops:** source map lists coverage; if something feels missing, check the raw file.

## Original sources included

| Source | Path |
|--------|------|
| ChatGPT deep research | `../chatgpt-deep-research.md` |
| ChatGPT conversation (design, naming, arch) | `../chatgpt.md` |
| Claude deep research | `../claude-deep-research.md` |
| Claude conversation (large) | `../claude.md` / `../claude.pdf` |
| DeepSeek | `../deepseek.md` |
| Gemini deep research | `../gemini-deep-research.md` |
| Gemini (blunt tech risks) | `../gemini.md` |
| GLM | `../glm.md` |
| Grok | `../grok.md` |
| Kimi deep research (+ assets) | `../kimi/kimi.md` |
| Meta AI | `../metaai.md` |
| MiniMax deep analysis | `../minimax.md` |
| Perplexity deep research | `../perplexity-deep-research.md` |
| Perplexity | `../perplexity.md` |

*Research window: ~July 2025–July 2026 ecosystem state. Star counts and vendor features are point-in-time and will drift.*
