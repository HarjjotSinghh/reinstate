---
title: "Claude Code and Codex session adapters"
navTitle: "Session adapters"
description: "Learn how Reinstate adapters discover, normalize, and restore Claude Code and Codex sessions without translating native transcripts across vendors."
order: 3
author: "Harjot Singh Rana"
status: current
schemaType: tech-article
version: "v0.5.2-rc.1"
updatedAt: 2026-08-16
tags: ["adapters", "claude-code", "codex", "same-vendor-resume"]
targetQuery: "Reinstate supported coding agents"
searchIntent: "agent-specific"
draft: false
noindex: false
---

Reinstate's adapters index vendor-native Claude Code, Codex, Gemini CLI, and
OpenCode sessions without translating sessions across agents. Claude Code and
Codex additionally support encrypted export/restore and native same-vendor
resume/fork; Gemini CLI and OpenCode remain read-only.

## Current v0.4.0 scope

| Adapter | Local index | Native resume/fork | Encrypted sync | Structured handoff |
| ------- | ----------- | ------------------ | -------------- | ------------------ |
| Claude Code | Full | Same-vendor | Supported | Destination and source |
| OpenAI Codex CLI | Full | Same-vendor | Supported | Destination and source |
| Gemini CLI | Read-only | No | No | Source-only |
| OpenCode | Read-only | No | No | Source-only |

Stable `v0.5.1` passed dual-platform tagged-artifact acceptance on Apple
Silicon macOS and native Windows x64. Intel macOS and Linux/WSL2 packages are
preview and unverified; check the
[compatibility page](/compatibility) before relying on them.

## Later adapters

| Adapter | Status |
| ------- | ------ |
| Cursor CLI | T1 Discover (read-only) |
| Cline | T1 Discover (read-only) |
| GitHub Copilot CLI | T1 Discover (read-only) |
| Qwen Code | T1 Discover (read-only) |
| Pi | T1 Discover (read-only) |
| Kimi Code CLI | T1 Discover (read-only) |
| Grok Build | Exploring |

Configuration support is capability-specific and planned separately. Session
support never implies support for MCP servers, skills, hooks, plugins,
marketplaces, or settings. See
[Universal agent configuration](/docs/universal-configuration).

## How does the Claude Code adapter remap a project?

Claude Code stores a project beneath a directory key derived from that
device's absolute project path. Reinstate records the configured canonical
project ID in a snapshot and recomputes Claude's directory key from the
destination device's `local_root`.

Reinstate validates the exact planned destination after restore. Finding the same
session ID elsewhere in `~/.claude/projects` does not count as success.

## How does the Codex adapter remap a project?

Codex stores the source working directory in each rollout's structural
`session_meta.cwd`. When project mappings are configured, Reinstate resolves that
directory to the canonical project ID during discovery and excludes rollouts
outside mapped roots.

Export normalizes the resolved source root to `${REPO:<id>}`. Restore expands
that token through the destination device's `local_root` while preserving
Codex's native date-partitioned rollout layout. Phase 1 transfers full
snapshots; append-aware delta transfer remains roadmap work.

## What do adapters exclude?

Sync adapters hard-exclude authentication material, credentials, tokens,
caches, logs, and regenerable dependencies. The local index additionally
excludes assistant reasoning/messages, tool output, environment dumps, and
auth stores while retaining bounded user-authored search text and metadata.
Future configuration profiles may carry secret references, but never raw
secret values. Tests use deterministic synthetic fixtures that are scanned for
secrets.

## Contributing an adapter

See [CONTRIBUTING.md](https://github.com/HarjjotSinghh/reinstate/blob/main/CONTRIBUTING.md#adapter-contributions).

Minimum pull-request requirements:

1. Implementation + fixtures
2. Defensive parsing (skip unknown fields/types)
3. Explicit credential excludes
4. Docs row in this matrix + README
5. No network in unit tests

## Adapter request template

Use [Adapter request](https://github.com/HarjjotSinghh/reinstate/issues/new?template=adapter_request.yml)
and include:

- Agent name + version
- Session file locations
- Sample **redacted** session header (no secrets)
- How resume works
