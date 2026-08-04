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
> terminals, containers, machines, and harness-specific configurations.**

A developer with **one laptop** may still use multiple agents, many sessions,
worktrees, WSL/containers/SSH, and agent-specific MCP/skills layouts — and still
lose the thread.

Cross-device encrypted sync is a **sharp entry wedge**. It is not the whole
product identity.

A second sharp problem is **quota-forced agent switching**. A developer may hit
Claude Code's or Codex's usage window while the task is unfinished, then open a
different harness because that subscription still has capacity. Today the code
is present but the task thread, rejected approaches, tool evidence, and latest
intent are stranded. Cross-agent continuation is therefore core continuity,
not an export accessory.

The long-term environment problem is also larger than verification. Developers
currently repeat MCP, skill, plugin, hook/loop, marketplace, and setting changes
for every harness and device. Reinstate should provide a canonical non-secret
desired-state profile and translate it through per-harness configuration
adapters. Configure once; preview, reconcile, and verify everywhere.

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
3. Environment restoration and configuration portability
4. Device synchronization
5. (Later) Team continuity

### One-line spine

> **Reinstate is not another place to code. It makes every place you code continuous.**

Reinstate owns continuity **before and after** execution.
Coding agents own the **execution loop**.

For cross-agent work, the precise promise is **same task, new linked native
session**. Reinstate must show fidelity and capability differences; it must not
claim that inaccessible hidden reasoning, system state, approvals, or
credentials became portable.

---

## Do we serve single-device users?

**Yes — without abandoning the multi-device wedge.**

| Audience | How they get value |
| -------- | ------------------ |
| One machine, multi-agent / multi-session | Local index, search, `last`, resume, quota/outage handoffs, verified environment, cross-harness configuration |
| Multi-machine | Everything above + encrypted session and non-secret config sync / cloud continuity |
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
2. **Switch agents without re-explaining** — continuity capsules with portable
   visible history, task state, evidence, fidelity, and lineage
3. **Crash / reboot / context limit recovery** — `rein last`
4. **Parallel tasks** — task-level control plane across terminals
5. **Environment drift** — pre-resume mismatch report
6. **Configure once** — reconcile MCP servers, skills, loops, plugins, and safe
   settings across supported harnesses

---

## Continuity modes (honesty contract)

| Mode | Fidelity | Label |
| ---- | -------- | ----- |
| Native resume (same vendor) | Highest; vendor semantics retained | Default when possible |
| Structured handoff | Portable task state + selected verbatim history + evidence | Primary cross-agent path |
| Reconstructed conversation | Portable visible history projected into a new native session | Experimental; pair/version-specific |

Never claim perfect Claude ↔ Codex native transcript translation. User messages
and visible assistant/tool evidence should be retained where portable, but
source system/developer messages remain audit history, historical tools are
never re-executed, and unavailable hidden state is reported as omitted.

Full design: [cross-agent-continuation.md](cross-agent-continuation.md).

---

## Product surfaces

### 1. Continuity core (library / engine)

Adapters, indexing, canonical capsule/event schema, workspace fingerprints,
checkpoints, fidelity reports, lineage, encryption, sync protocol, destination
projections, and universal configuration normalization/rendering.

### 2. Reinstate CLI / TUI (primary)

```bash
rein                  # session switcher
rein search …
rein inspect …
rein resume …
rein fork …
rein handoff …
rein sync | push | pull
rein mcp add …
rein config diff | apply | sync
```

Launches sessions in **native agents by default**.

The configuration commands are later-phase direction, not part of the current
`v0.1` CLI. See [universal-configuration.md](universal-configuration.md).

### 3. Reinstate Console (optional, thin client)

Unified UI for search, transcript preview, agent selection, capability and
workspace status. May later speak **ACP** (`session/resume`). Still delegates
coding to external agents.

### 4. Reinstate Cloud

Encrypted multi-device sync, backup, device management, later team handoffs.

### Cross-agent continuation is not a Reinstate runtime

The quota-switch flow captures a source session, verifies the workspace,
projects a portable capsule into a **new destination session**, launches the
chosen agent, and records lineage. Claude Code, Codex, Gemini, or OpenCode still
owns prompting, permissions, tools, and execution after launch.

The first GA pair is Claude Code ↔ Codex. The flow must work when the source CLI
is closed or rate-limited and cannot depend on a source model summary. Gemini
CLI and OpenCode follow. Other agents require directional adapter and
acceptance evidence.

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
- Reinstate-owned plugin runtime
- Full IDE / ADE replacement

Integrate with Orca, Conductor, T3 Code, editors, and agents instead.
Reinstate may synchronize declarations for third-party marketplaces; it does
not operate a marketplace or execute the agent loop.

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
- Continue in another agent after a usage limit or outage
- Back up sessions automatically
- Configure MCP servers once across harnesses and devices
- Install the same skills, loops, plugins, and marketplaces across harnesses
- Recover after crashes or reinstalls
- Hand work to another developer

Opt-in local telemetry (metadata only, never transcripts): search, resume,
handoff attempts/acknowledgements, source-target support state, mismatch
detections, remote resumes, and time to first useful destination action.

---

## Relationship to Phase 0 / Phase 1

Phase 0–1 implement the **trustworthy engine**: adapters, path map, encryption,
sync, doctor, and installers. Local verification is green; native platform,
two-device, remote CI, and public release gates remain. That engine is the
substrate for:

- Phase 2 local index / switcher
- Phase 3 verified resume
- Phase 4 core cross-agent continuation via portable handoffs
- Phase 5 universal configuration + automated multi-device habit
- Phase 6 thin console / ACP

Do not rewrite the core for marketing pivots. **Reuse primitives.**
