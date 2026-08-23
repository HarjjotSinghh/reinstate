---
title: "Reinstate features and commands from v0.1.0 to v0.4.0"
navTitle: "Features and commands"
description: "See which Reinstate features and CLI commands shipped in stable v0.1.0 through v0.4.0, including encrypted sync, local search, verified resume, and structured handoff."
order: 2
author: "Harjot Singh Rana"
status: current
schemaType: tech-article
version: "v0.5.2-rc.1"
updatedAt: 2026-08-16
tags: ["cli", "features", "session-sync", "handoff", "verified-resume"]
targetQuery: "Reinstate features and commands"
searchIntent: "navigational"
draft: false
noindex: false
---

Stable **`v0.4.0`** is the current release. Use this page as the versioned
feature map. Universal configuration, a Reinstate console, and team continuity
remain later phases.

A structured handoff starts a **new destination session continuing the same
task**. Native resume stays same-vendor. Reinstate does not reconstruct a
vendor session or translate one agent's transcript into another.

The `rein` and `reinstate` names run the same binary.

## Stable lines

| Line | Shipped | Surface |
| ---- | ------- | ------- |
| `v0.1.0` | 2026-07-30 | Encrypted Claude Code / Codex sync across devices |
| `v0.2.0` | 2026-08-05 | Configless local index, search, inspect, last, resume, fork |
| `v0.3.0` | 2026-08-11 | Verified resume environment report and launch gate |
| `v0.4.0` | 2026-08-16 | Structured handoff into a new Claude Code or Codex session |

Mandatory verified platforms remain Apple Silicon macOS and native Windows
x64. Intel macOS and Linux/WSL2 stay preview/unverified.

## Command map

| Command | Since | Needs `init` |
| ------- | ----- | :----------: |
| `rein version`, `doctor`, `setup check`, `completion` | 0.1 | no |
| `rein init`, `list`, `status`, `diff`, `push`, `pull`, `conflicts` | 0.1 | yes after init |
| `rein sessions`, `search`, `inspect` | 0.2 | no |
| `rein last`, `resume`, `fork`, bare `rein` | 0.2 | no |
| `--allow-environment-warning` | 0.3 | no |
| `rein handoff`, `handoff list`, `inspect`, `export` | 0.4 | no |
| `rein resume --with claude\|codex` | 0.4 | no |

Flag-level syntax lives in the [CLI reference](/docs/cli-reference).

## v0.1.0 — encrypted sync

```sh
rein init --project github.com/acme/app=/absolute/path/to/app
rein push --agent claude --session SESSION_ID --dry-run
rein pull --agent claude --session SESSION_ID --dry-run
```

Client-side age encryption, user-owned S3-compatible storage, path remapping,
credential exclusion, atomic restore, and conflict forks.

## v0.2.0 — local continuity

```sh
rein sessions
rein search "webhook retry" --agent claude
rein inspect claude:SESSION_ID
rein resume codex:SESSION_ID --dry-run
```

No bucket required. Gemini CLI and OpenCode are read-only. Native resume is
Claude Code → Claude Code and Codex → Codex only.

## v0.3.0 — verified resume

```sh
rein resume claude:SESSION_ID --dry-run --json
rein resume claude:SESSION_ID --allow-environment-warning baseline.unavailable
```

Workspace, agent, capability, and runtime checks. Blockers cannot be
overridden.

## v0.4.0 — structured handoff

```sh
rein handoff claude:SESSION_ID --to codex --dry-run --json
rein handoff list --json
rein resume claude:SESSION_ID --with codex --dry-run --json
```

Sources: Claude Code, Codex, Gemini CLI, OpenCode, Grok Build. Destinations:
Claude Code and Codex only. The destination first reply restates five
acknowledgement bullets. Capsules stay local and are excluded from sync.

Fail-closed ranges: Claude Code `2.1.219`–`2.1.238`, Codex CLI
`0.133.0`–`0.149.0`. Dual-platform tagged-artifact acceptance passed on Apple
Silicon macOS and native Windows x64 (44/44 on both devices).

## Not in v0.4.0

Reconstructed cross-agent sessions, Gemini/OpenCode/Grok destinations,
universal configuration, and Intel macOS or Linux as verified platforms.

See [handoff](/docs/handoff), [getting started](/docs/getting-started),
[CLI reference](/docs/cli-reference), and the [roadmap](/roadmap).
