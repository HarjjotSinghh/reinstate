# Product strategy

**Status:** accepted direction (2026-07-25)
**Complements:** [ROADMAP.md](../ROADMAP.md), [PRODUCT.md](../PRODUCT.md),
[architecture.md](architecture.md)

---

## Problem we actually solve

Not only:

> My session is on my desktop; I am now on my laptop.

Deeper:

> **Coding work is fragmented across agents, sessions, projects, branches,
> terminals, containers, machines, and configurations.**

A developer with **one laptop** may still use multiple agents, many sessions,
worktrees, WSL/containers/SSH, and agent-specific MCP/skills layouts — and still
lose the thread.

Cross-device encrypted sync is a **sharp entry wedge**. It is not the whole
product identity.

---

## Positioning

### Lead with

# Reinstate

**Pick up any coding task exactly where you left it.**

### Then explain

> Search, resume, and hand off coding-agent sessions across agents, projects,
> environments, and devices.

Hierarchy of value:

1. Task continuity
2. Agent interoperability
3. Environment restoration
4. Device synchronization
5. (Later) Team continuity

### One-line spine

> **Reinstate is not another place to code. It makes every place you code continuous.**

Reinstate owns continuity **before and after** execution.
Coding agents own the **execution loop**.

---

## Do we serve single-device users?

**Yes — without abandoning the multi-device wedge.**

| Audience | How they get value |
| -------- | ------------------ |
| One machine, multi-agent / multi-session | Local index, search, `last`, resume, verified environment, handoffs |
| Multi-machine | Everything above + encrypted push/pull / cloud continuity |
| Teams (later) | Shared checkpoints, onboarding, audit |

Do **not** contort Reinstate into a generic developer dashboard to “include
everybody.” Expand the **problem statement**, not random feature surface.

---

## Differentiator vs session browsers

Tools like SpecStory validate demand for unified session browse/search/resume.
Reinstate must not win by recreating “another picker.”

**Signature claim:**

> Reinstate does not only recover the conversation. It **verifies and restores
> the environment** required to continue correctly.

---

## Feature themes for same-device users

1. **Where was that conversation?** — searchable development memory
2. **Switch agents without re-explaining** — portable checkpoints
3. **Crash / reboot / context limit recovery** — `rein last`
4. **Parallel tasks** — task-level control plane across terminals
5. **Environment drift** — pre-resume mismatch report

---

## Continuity modes (honesty contract)

| Mode | Fidelity | Label |
| ---- | -------- | ----- |
| Native resume (same vendor) | Highest | Default when possible |
| Portable handoff (checkpoint) | Lossy, explicit | Primary cross-agent path |
| Reconstructed conversation | Experimental | Never silent |

Never claim perfect Claude ↔ Codex native transcript translation.

---

## Product surfaces

### 1. Continuity core (library / engine)

Adapters, indexing, canonical schema, workspace fingerprints, checkpoints,
encryption, sync protocol, config normalization.

### 2. Reinstate CLI / TUI (primary)

```bash
rein                  # session switcher
rein search …
rein inspect …
rein resume …
rein fork …
rein handoff …
rein sync | push | pull
```

Launches sessions in **native agents by default**.

### 3. Reinstate Console (optional, thin client)

Unified UI for search, transcript preview, agent selection, capability and
workspace status. May later speak **ACP** (`session/resume`). Still delegates
coding to external agents.

### 4. Reinstate Cloud

Encrypted multi-device sync, backup, device management, later team handoffs.

---

## Explicitly out of scope (for now)

Do not build unless usage forces a revisit:

- Custom code editor
- Full terminal emulator
- Browser automation environment
- Worktree orchestration platform
- Multi-agent scheduler / PR suite
- Proprietary model router
- Agent marketplace
- Full IDE / ADE replacement

Integrate with Orca, Conductor, T3 Code, editors, and agents instead.

Long-term strategic option:

> Reinstate becomes the **continuity infrastructure** that ADEs and agents
> integrate with — they own *where work happens*; Reinstate owns *whether work
> survives and can move*.

### Harness revisit criteria

Build a Reinstate-owned runtime only if **all** are true:

1. Users frequently choose cross-agent continuation
2. Existing agents cannot consume portable handoffs reliably
3. A Reinstate runtime would improve continuity, not just UI duplication

---

## ICP (first)

> Developers who use **multiple terminal coding agents** and maintain several
> concurrent sessions or projects.

Multiple physical devices are **not** required for ICP fit.

---

## Metrics and validation

**North star:** previously started coding tasks successfully resumed per active user.

Landing survey options:

- Continue sessions across computers
- Find and resume old sessions
- Move sessions between coding agents
- Back up sessions automatically
- Sync MCP / skills / configuration
- Recover after crashes or reinstalls
- Hand work to another developer

Opt-in local telemetry (metadata only, never transcripts): search, resume,
handoff attempts, mismatch detections, remote resumes.

---

## Relationship to Phase 0 / Phase 1

Phase 0–1 implement the **trustworthy engine**: adapters, path map, encryption,
sync, doctor, and installers. Local verification is green; native platform,
two-device, remote CI, and public release gates remain. That engine is the
substrate for:

- Phase 2 local index / switcher
- Phase 3 verified resume
- Phase 4 handoffs
- Phase 5 automated multi-device habit
- Phase 6 thin console / ACP

Do not rewrite the core for marketing pivots. **Reuse primitives.**
