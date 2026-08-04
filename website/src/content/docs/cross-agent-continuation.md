---
title: "Cross-agent continuation across coding agents"
navTitle: "Cross-agent continuation"
description: "Review Reinstate's Phase 4 design for explicit, fidelity-reported task handoffs between Claude Code, Codex, and additional coding-agent harnesses."
order: 5
author: "Harjot Singh Rana"
status: planned
schemaType: tech-article
version: "roadmap"
updatedAt: 2026-08-04
tags: ["roadmap", "cross-agent", "handoff", "claude-code", "codex"]
targetQuery: "continue a coding agent conversation in another agent"
searchIntent: "solution"
draft: false
noindex: false
---

Cross-agent continuation is a **core Reinstate Phase 4 feature**. It is not in
the current Phase 1 CLI.

The flagship scenario is simple: Claude Code reaches its usage limit in the
middle of a task, so the developer hands the work to Codex without calling
Claude again or explaining the project from zero. Codex → Claude Code must work
too. Gemini CLI and OpenCode follow; other agents need adapter and acceptance
evidence.

## Product promise

> Continue the same task in another supported agent through an explicit,
> inspectable handoff with a fidelity report.

The destination is normally a **new native session linked to the source**. It
is not the same vendor session ID and Reinstate does not claim that hidden
runtime state became portable.

## Continuity modes

| Mode | Meaning |
| ---- | ------- |
| **Native resume** | Same harness/vendor; highest fidelity |
| **Structured handoff** | Task state + selected verbatim history + workspace/tool/test evidence; default cross-agent path |
| **Reconstructed conversation** | Portable visible history projected into a new native session; experimental and pair/version-specific |

Each component is labeled `exact`, `normalized`, `summarized`, `referenced`, or
`omitted` with a reason.

## What carries

- User messages and visible assistant replies, subject to explicit redaction
- Historical tool calls/results as **inert evidence**, never actions to replay
- Current goal, latest user intent, decisions, rejected approaches, constraints,
  completed/pending work, and next action
- Canonical project identity, branch/HEAD, dirty files, diffs, commands, tests,
  and errors verified against the destination workspace
- Supported attachments, timestamps, source lineage, and capability differences

## What does not carry

- Credentials, cookies, API keys, account state, or source approvals
- Hidden chain-of-thought or system prompts that were never available
- Opaque reasoning/signatures across vendors
- Live processes, shell state, or remote sandboxes
- Source system/developer instructions as destination authority

The raw source artifact remains immutable and may be retained locally or in
encrypted BYO storage for audit. Retained does not mean every byte is injected
into the destination model context.

## Planned flow

```text
source session
  → freeze latest complete record and hash it
  → version-gated transcript parse
  → continuity capsule (task + events + workspace + fidelity + lineage)
  → destination capability and security preview
  → new destination-native session
  → acknowledgement before mutation
```

The quota path cannot depend on the source model. Reinstate builds deterministic
workspace, message, tool, and test evidence locally. Large histories use an
explicit `checkpoint`, `balanced`, or `full` projection policy and a private
sidecar rather than silently overflowing the destination context window.

Writing target-native session files or databases is experimental, uses a new
destination ID, and is exact-version gated. The safe default is a supported
new-session launch with an inspectable capsule.

## Planned delivery

1. Capsule/event schema, fidelity vocabulary, lineage, and adversarial fixtures
2. Bidirectional Claude Code ↔ Codex structured handoff with dry-run
3. Reconstructed portable visible history and context budgeting
4. Gemini CLI and OpenCode
5. Additional agents and ACP integrations based on evidence

This remains continuity infrastructure: agents execute; Reinstate captures,
verifies, projects, launches, and records lineage.

Read the
[full product, architecture, security, research, and release-gate specification](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/cross-agent-continuation.md).
