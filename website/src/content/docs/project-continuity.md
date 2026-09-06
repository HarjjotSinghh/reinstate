---
title: "Project continuity roadmap"
navTitle: "Project continuity"
description: "Explore the planned model for one project understanding across agents: durable rules and decisions in the repository, plus learned memory with provenance."
order: 18
author: "Harjot Singh Rana"
status: planned
schemaType: tech-article
version: "roadmap"
updatedAt: 2026-09-06
tags: ["roadmap", "project-continuity", "memory", "context", "mcp"]
targetQuery: "share project rules and memory across coding agents"
searchIntent: "solution"
draft: false
noindex: false
---

> Status: planned as Phase 7, targeted at `v0.8.0`, after Phase 6A/6B ships in
> `v0.7.0`. Nothing on this page exists in the current CLI; commands shown
> here are design direction only.

Reinstate carries a *session* from one agent, device, or environment to
another. Phase 6 will extend that to the *configuration* of an agent
environment. Project continuity is the third kind of state: what the project
**knows**.

## The problem

A developer states a rule to one agent — never use em dashes in user-facing
copy, use `pnpm` and never `npm`, all API handlers validate with Zod. That
rule binds that agent only. Claude Code writes it to `CLAUDE.md` and its own
per-project memory; Codex, Cursor, Grok, OpenCode, and Gemini CLI keep
separate instruction files and separate stores, and none of them learn it.
Decisions and hard-won knowledge fracture the same way, until one instruction
file grows stale and contradictory and the next agent re-derives what was
already known, or silently contradicts it.

## Three kinds of state, three storage models

Conflating these is what makes one enormous instruction file rot.

| Kind | Example | Authored by | Lives in |
| ---- | ------- | ----------- | -------- |
| **Durable context** | rules, conventions, architecture, decisions | a person, reviewed | the repository, in git |
| **Learned memory** | gotchas, workarounds, rejected approaches | an agent, proposed | the local encrypted store, synced |
| **Configuration** | MCP servers, skills, hooks, plugins, settings | a person, declared | the Reinstate desired-state profile |

Durable context is reviewable and useful to a teammate who has never
installed Reinstate. Learned memory is provisional and provenance-carrying —
it does not belong in a pull request until a person promotes it.

## Canonical context, rendered

The canonical project tree — rules, conventions, architecture, and decisions
— will be committed to the repository, then rendered into `AGENTS.md`,
`CLAUDE.md`, `GEMINI.md`, and editor rule files through the Phase 6A
configuration adapters:

```text
native instruction files
        ↕ import / render
configuration adapter
        ↕
canonical project context (in the repository)
```

Generated content will sit in delimited regions; hand-written prose in the
same file stays untouched. A rule scoped to a path renders into a harness's
scoped mechanism where one exists, or into the global file with a recorded
lossy mapping where it does not. Uninstall Reinstate and the rules, decisions,
and conventions remain plain reviewable markdown.

`AGENTS.md` and the native instruction files stay render targets, not a
format Reinstate replaces.

## Learned memory, with provenance

A memory will be a typed, structured record, not a paragraph appended to a
file: type (fact, decision, gotcha, workaround, preference, or rejected
approach), scope, content, and status (proposed, active, superseded, or
expired). Every record carries its source — agent, session, device, branch,
commit, timestamp — so a wrong memory is removable and a stale one is
traceable. Retrieval is scoped to the task at hand; an agent never receives
the whole store.

## Capture and promotion

```text
agent learns something → explicit capture → proposed memory → review queue
                                                     │
                                       promoted ──────┴────► reviewed context
                                                              in the repository
```

Two invariants: **capture is explicit** — a tool call or `rein remember`,
never mining transcripts in the background — and **promotion is human**. An
agent proposes; it does not promote itself into durable project truth.

## A local Reinstate MCP server

Agents will read and write memory through an MCP server Reinstate runs
locally: `memory.search`, `memory.add`, `memory.update`, `memory.supersede`,
and `memory.promote` (never auto-applied). This stays a store with
provenance, not an execution ecosystem — it will not run an agent loop,
schedule work, or execute plugins. Harnesses that cannot speak MCP will still
receive durable context through rendered instruction files; they read, they
cannot contribute.

## Conflict and staleness

Because memories will carry dates, provenance, and supersede links, Reinstate
can report a collision — a memory that a later decision contradicts —
instead of serving both as true. Staleness works the same way: age, explicit
expiry, and evidence that no longer matches the tree are meant to surface as
a count in `rein status`, not as silent decay.

## Security and sync

Project continuity is designed to inherit the existing security contract and
add to it: durable context is committed to the repository and is as public as
the repo; learned memory is meant to be stored locally encrypted and synced
as non-secret project state, the way sessions are; every record is intended
to pass the secret scanner before it is stored or synced; and no credentials,
tokens, or vendor auth stores are meant to enter memory or context.

## What this is not

| Not this | Why |
| -------- | --- |
| Silent transcript mining | Durable state is created by an explicit act, not inferred from conversation |
| A vector-database or knowledge-graph product | Retrieval stays scoped and explainable; every record names its source |
| A replacement for git | Reviewed truth is committed; only learned memory and provenance live in the encrypted store |
| Parallel-agent arbitration | Reinstate records who learned what; harnesses own concurrent execution and locks |
| A credential store or MCP gateway | Authentication stays Phase 6B; a gateway is exploratory and gated on real demand |

Full design:
[docs/project-continuity.md](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/project-continuity.md).
