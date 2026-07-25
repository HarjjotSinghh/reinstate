# Product

## Register

brand

## Positioning (canonical)

**Reinstate** is the **continuity layer for coding-agent work**.

Lead:

> **Pick up any coding task exactly where you left it.**

Expand:

> Search, resume, and hand off coding-agent sessions across agents, projects,
> environments, and devices — verifying that the workspace and capabilities
> still match before you continue.

Spine:

> **Reinstate is not another place to code. It makes every place you code continuous.**

Multi-device encrypted sync is the **entry wedge** and flagship demo.  
**Single-device multi-agent / multi-session continuity** is first-class product
value on the same engine. Full strategy: [docs/product-strategy.md](docs/product-strategy.md).

## Users

### Primary ICP (narrow)

AI developers and power users who run **multiple terminal coding agents**
(Claude Code, Codex, Gemini CLI, OpenCode, …) and maintain **several concurrent
sessions or projects**. They may use one machine or many.

They care about:

- finding the right session later  
- resuming without re-prompting from zero  
- switching agents without losing the task thread  
- catching environment drift (branch, MCP, skills, runtime)  
- optionally moving work between Windows desktop and macOS laptop  

### Secondary

- Multi-device users (desktop + laptop) for whom push/pull is the hero path  
- Later: teams handing off checkpoints and onboarding from prior agent work  

Primary job: **keep coding-agent task continuity**.  
Secondary jobs: security/BYO storage trust, install/init, compare alternatives,
early access.

## Product purpose

Reinstate provides **continuity infrastructure**:

1. **Session recovery** — discover, search, preview, resume, fork  
2. **Agent portability** — portable checkpoints / handoffs (not silent translation)  
3. **Environment continuity** — verify repo/branch/MCP/skills/runtime before resume  
4. **Cloud continuity** — E2EE BYO storage (R2/S3 first), path remapping, conflict-safe sync  
5. **Team continuity** (later) — shared checkpoints, provenance, policy  

Agents remain the executors. Reinstate does **not** become a full ADE/IDE.

Site purpose: convert continuity-aware agent users into waitlist/docs/GitHub
stars; establish trust (E2EE, BYO storage, open source); position against
single-vendor history UIs, live remote-control tools, DIY file sync, and
full agentic IDEs that own the coding surface.

Success: clear value in one viewport for **one-device and multi-device** users;
docs accurate to the repo; brand feels like infrastructure you trust.

## Brand personality

**Precise · portable · zero-knowledge**

Voice: concrete, CLI-native, no hype verbs. Prefer specific claims (path
remapping, age encryption, verified resume, portable handoff) over abstract
empowerment. Verb-friendly language: *reinstate this session*, *pull on the
laptop*, short alias **`rein`**.

Emotional goals: confidence when switching **tasks, agents, or machines**;
calm technical authority; zero-knowledge trust without security theater.

Visually: **sharp, flat-vector isometric world** — workroom in bold outlines;
product as a physical object that is pulled apart, sealed, and put back.
Illustration is the brand, not decoration.

## Anti-references

- Generic AI-purple SaaS mesh gradients and glassmorphism soup  
- Drop shadows, blur, glow; the system is sharp and vector  
- Claude / Cursor-style chat-product marketing (we are continuity infrastructure)  
- Busy crypto / Web3 neon excess  
- "Dropbox for session JSONL" or vague "cloud for AI chats" only  
- Positioning as “another ADE” (Orca/Conductor/T3 Code competitor)  
- Identical three-feature card grids and fake dashboard heroes  
- Decorative terminal window as the only hero  

## Design principles

1. **Show the mechanism.** Continuity: find → verify → resume/handoff → (optional) sync.  
2. **Infrastructure honesty.** Real CLI, real docs, real security defaults.  
3. **Sharp, never soft.** Depth from outlines and flat faces.  
4. **One accent.** Chartreuse is the only loud colour.  
5. **Docs are the proof.** Marketing points into getting-started, security, adapters, strategy.  
6. **Waitlist without friction.** Simple email capture; GitHub star secondary.  

Full visual system: [DESIGN.md](DESIGN.md).

## Accessibility & inclusion

Target WCAG 2.2 AA on marketing and docs. Full keyboard paths. Honor
`prefers-reduced-motion`. Light default / dark twin; body text ≥4.5:1.
Focus-visible rings. Illustrations carry `<title>` and `<desc>`.

## Metrics

North star: **previously started coding tasks successfully resumed per active user.**

Not primary: devices connected, raw storage bytes, vanity DAU.
