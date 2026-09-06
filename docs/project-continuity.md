# Project continuity (roadmap)

> Status: planned as Phase 7, targeted at `v0.8.0`, after Phase 6A/6B ships in
> `v0.7.0`. Nothing on this page exists in the current CLI. Commands shown here
> are design direction and will be stabilized through an RFC before
> implementation; see [cli-reference.md](cli-reference.md) for what actually
> ships today.

Reinstate carries a *session* from one agent, device, or environment to
another. Phase 6 extends that to the *configuration* of an agent environment —
MCP servers, skills, instructions, hooks, plugins, and safe settings declared
once and rendered into each harness.

Project continuity is the third kind of state: what the project **knows**.

## The problem

A developer states a rule to one agent — never use em dashes in user-facing
copy, use `pnpm` and never `npm`, all API handlers validate with Zod. That rule
binds that agent. Claude Code writes it to `CLAUDE.md` and its own per-project
memory; Codex, Cursor, Grok, OpenCode, and Gemini CLI keep separate instruction
files and separate stores, and none of them learn it.

The same fracture applies to everything a project accumulates:

- **decisions** — we chose Postgres over SQLite, and why; we rejected Redis for
  queues; auth belongs in middleware, not controllers
- **knowledge earned by working** — the billing tests need Redis on port 6380;
  this vendor API returns `200` on a failure; the checkout bug was a race in
  cart mutation
- **conventions** — the design system's spacing rules, the generated files
  nothing may touch

Today each of those lives wherever the agent that learned it keeps memory, or
in one instruction file that grows until it is stale, contradictory, and long
enough that agents skim it. The next agent then re-derives what was already
known, or contradicts a decision recorded three weeks ago, and nobody notices
until review.

This is a continuity problem, not a memory-product problem. It has the same
shape as the one Reinstate already solves: state that should survive a boundary
does not, because every tool keeps its own copy.

## Three kinds of state, three storage models

Conflating these is what makes one enormous instruction file rot.

| Kind | Example | Authored by | Lives in | Phase |
| ---- | ------- | ----------- | -------- | ----- |
| **Durable context** | rules, conventions, architecture, decisions | a person, reviewed | the repository, in git | 7A |
| **Learned memory** | gotchas, workarounds, rejected approaches, observations | an agent, proposed | the local encrypted store, synced | 7B/7C |
| **Configuration** | MCP servers, skills, hooks, plugins, settings | a person, declared | the Reinstate desired-state profile | 6A |

Durable context is reviewable, diffable, and useful to a teammate who has never
installed Reinstate. Learned memory is provisional, provenance-carrying, and
subject to being superseded — it does not belong in a pull request until a
person promotes it.

## Canonical context, rendered

The canonical project tree is committed to the repository. The exact layout is
subject to the RFC; the shape is:

```text
.reinstate/
├── project.toml              # identity, enabled targets, scopes
├── context/
│   ├── project.md            # what this project is
│   ├── architecture.md
│   ├── conventions.md
│   └── decisions/
│       ├── 0001-auth-strategy.md
│       └── 0002-database-choice.md
├── rules/
│   ├── global.md
│   ├── frontend.md           # scope: apps/web/**
│   └── backend.md            # scope: internal/**
└── skills/
```

Reinstate is a compiler here, not a file copier. The canonical tree is the
source; `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and editor rule files are render
targets, written through the Phase 6A configuration adapters:

```text
native instruction files
        ↕ import / render
configuration adapter
        ↕
canonical project context (in the repository)
```

Three rules govern rendering:

1. **Delimited regions.** Generated content sits between explicit markers.
   Hand-written prose in the same file is preserved exactly, and a file with no
   markers is never rewritten without a preview and a backup.
2. **Scope survives translation, or the loss is reported.** A rule scoped to
   `apps/web/**` renders into a harness's path-scoped mechanism where one
   exists, and into the global instruction file with a recorded lossy mapping
   where it does not.
3. **The tree stands alone.** Uninstall Reinstate and the rules, decisions, and
   conventions remain plain reviewable markdown in the repository.

`rein context adopt` reads whatever already exists — `CLAUDE.md`, `AGENTS.md`,
`GEMINI.md`, editor rule files — and normalizes it into the canonical tree, so
adoption is not a rewrite.

### Why not make `AGENTS.md` itself canonical

Because a single flat file cannot carry path scoping, per-decision provenance,
supersede links, or expiry — and because a file that every agent reads in full
on every task is exactly the artifact that bloats until it is ignored.
`AGENTS.md` remains the interoperability format Reinstate writes *into*, not a
format Reinstate competes with.

## Memory records

A memory is a typed, structured record, not a paragraph appended to a file.

| Field | Purpose |
| ----- | ------- |
| `type` | fact · decision · gotcha · workaround · preference · rejected approach |
| `scope` | project, path glob, or task |
| `content` | the claim itself |
| `source` | agent, session id, device, branch, commit, timestamp |
| `confidence` | how strongly it is held |
| `status` | proposed · active · superseded · expired |
| `supersedes` | the record this one replaces |
| `expires_at` | when it should be re-checked |

Provenance is the point. An agent that reads "checkout serializes cart
mutations" is told less than an agent that reads: Claude Code learned this in
session `claude/01893` on 2026-08-28, Codex confirmed it the next day, and
commit `7bd18ac` implements it. Provenance is also what makes a wrong memory
removable — you can see where it came from and what it was based on.

Retrieval is scoped. An agent asks for what is relevant to the task at hand; it
never receives the whole store.

## Capture and promotion

The write path is the part of this problem that existing tools leave open.
Distribution is well served — several projects render one rules file into many
harnesses. What nothing closes is the loop back: something learned in one
agent's session becoming durable project state that every agent honors.

The lifecycle:

```text
agent learns something
        │  explicit capture (MCP tool call, or `rein remember`)
        ▼
proposed memory  ──── review queue ────►  active memory
        │                                      │
        │                                      │  proves durable / you say so
        │                                      ▼
        └──────────────────────────►  `rein memory promote`
                                               │
                                               ▼
                                     reviewed context in the repository
                                     (a rule, or a decision record)
```

Two invariants:

- **Capture is explicit.** A memory is created by a tool call or a command,
  never by mining transcripts in the background. Reinstate already indexes
  session history; deriving durable project state from conversation content
  without an explicit act is a different privacy contract, and not one this
  project makes.
- **Promotion is human.** An agent proposes; it does not promote on its own
  authority. Durable project truth enters the repository through review, the
  same way code does.

Not everything deserves the same permanence, and the distinction matters:
"never use em dashes in user-facing copy" is deterministic policy and belongs
in a rule; "we rejected Redis for queues" is a decision record; "the billing
tests need Redis on 6380" is a memory that may be false next month.

## Conflict and staleness

Instruction files rot. A decision changes, the old sentence stays, and agents
are handed two contradictory truths with no way to tell which is current.

Because memories carry dates, provenance, and supersede links, Reinstate can
report the collision instead of serving both:

```text
CONTEXT CONFLICT

memory/18        use Redis for job queues
                 Codex · session codex/7182 · 2026-07-19 · confidence medium

decision/0041    Redis replaced by SQS
                 reviewed · 2026-08-14 · commit 4c1e9ab

Suggested        mark memory/18 superseded by decision/0041
```

Staleness works the same way: age, explicit expiry, and evidence that no longer
matches the tree — a memory anchored to a file or commit that has since
changed — surface as a count in `rein status`, not as silent decay.

Detection is deterministic. Explicit supersede links, overlapping scope,
declared subject tags, expiry, and evidence anchors are what raise a conflict.
Reinstate makes no model calls anywhere today, including in handoffs, and this
phase is not where that changes; the cost is that a contradiction sharing no
scope or tag with its predecessor will be missed. See
[the open questions](planning/v0.8.0-project-continuity/clarifications.md) for
what would have to be true to revisit that.

## The Reinstate MCP server

Agents read and write memory through an MCP server that Reinstate runs locally,
alongside the daemon that already keeps sync current:

```text
memory.search      scoped retrieval for the current task
memory.add         propose a memory (enters the review queue)
memory.update      revise a memory this agent authored
memory.supersede   mark a memory replaced by another record
memory.promote     request promotion to reviewed context (never auto-applied)
```

This stays inside the "client, not harness" boundary. The server is a store
with provenance, not an execution ecosystem: it does not run the agent loop,
schedule work, route models, or execute plugins. Every write is attributable,
every write is subject to secret scanning, and the review queue means no agent
silently changes what the next agent believes.

Harnesses that cannot speak MCP still receive durable context through rendered
instruction files. They read; they cannot contribute.

## Security and sync contract

Project continuity inherits the existing contract
([security-model.md](security-model.md)) and adds to it:

- Durable context is committed to the repository. It is as public as the repo.
- Learned memory is stored locally encrypted and synced as non-secret project
  state, the way sessions are. It is not device-local like handoff capsules,
  because "the same project understanding on the other machine" is the point.
- Every record passes the secret scanner before it is stored or synced. A
  memory is prose written by an agent, which makes it exactly the kind of place
  a token gets pasted by accident.
- No credentials, tokens, or vendor auth stores enter memory or context.
  Authentication remains Phase 6B: secret references, keychain resolution, and
  official login flows.
- Memory is redacted and bounded in every output surface, like transcripts are.

## Cross-agent readiness

`rein status` is the control-plane view — what every agent on this device
actually has:

```text
Project    reinstate            Branch  main

Context    14 rules · 8 decisions       1 conflict
Memory     31 active · 2 stale          4 unreviewed

             Context  Memory  Skills  MCP
Claude Code  current  current   8/8    6/6
Codex        current  current   8/8    6/6
Cursor       drift    current   8/8    5/6
OpenCode     current  current   8/8    6/6
```

Verified resume (Phase 3) gains the same signal: a session that cannot see the
project's current rules is reported before launch, next to a missing MCP server
or a mismatched runtime.

## What this is not

| Not this | Why |
| -------- | --- |
| Silent transcript mining | Durable state is created by an explicit act, not inferred from conversation |
| A vector-database or knowledge-graph product | Retrieval stays scoped and explainable; every record names its source |
| A replacement for git | Reviewed truth is committed; only learned memory and provenance live in the encrypted store |
| Parallel-agent arbitration | Reinstate records who learned what; harnesses own concurrent execution, file claims, and locks |
| A rules-file copier | Copying `CLAUDE.md` over `.cursor/rules` discards scope and produces silent drift; adapters normalize and render |
| A credential store or MCP gateway | Authentication stays Phase 6B; a gateway is exploratory and gated on real demand |

## Prior art

This is a crowded area, and the crowding is informative: distribution of static
rules across harnesses is well served, and cross-agent memory servers exist.
What is not solved is the write path with provenance, tied to session and
project identity, reconciled across devices. See
[research/2026-09-06-phase-7-cross-agent-context-landscape.md](research/2026-09-06-phase-7-cross-agent-context-landscape.md)
for the verified survey and what each tool does and does not cover.

## Relationship to the rest of Reinstate

1. Reinstate finds the task ([Phase 2](../ROADMAP.md#phase-2--local-universal-session-index-)).
2. Verified resume reports whether the environment can continue it correctly.
3. Universal configuration reconciles the environment that was missing.
4. **Project continuity supplies what the project knows, to whichever agent
   picks the task up.**
5. Encrypted sync carries all of it to the next device.

Sessions remain the wedge. Project continuity is what makes the second agent as
informed as the first.

See [ROADMAP.md](../ROADMAP.md),
[universal-configuration.md](universal-configuration.md),
[architecture.md](architecture.md),
[security-model.md](security-model.md), and
[ADR 0006](adr/0006-project-continuity-scope.md).
