---
title: "Structured handoff between Claude Code and Codex"
navTitle: "Structured handoff"
description: "Continue the same coding-agent task in a new Claude Code or Codex session with Reinstate v0.4.0 structured handoff, without native resume or transcript translation."
order: 16
author: "Harjot Singh Rana"
status: current
schemaType: tech-article
version: "v0.5.0"
updatedAt: 2026-08-16
tags: ["handoff", "claude-code", "codex", "cli"]
targetQuery: "Reinstate structured handoff"
searchIntent: "how-to"
draft: false
noindex: false
---

Stable `v0.5.0` lets you continue the same task in a **new** Claude Code or
Codex session. This is not native resume, not a transferred session, and not a
lossless copy of the source transcript.

## Commands

```sh
rein handoff --last --from claude --to codex --dry-run
rein handoff claude:SESSION_ID --to codex
rein handoff claude:SESSION_ID --to codex --no-launch
rein handoff list --json
rein handoff inspect HANDOFF_ID --json
rein handoff export HANDOFF_ID --format markdown --out ./handoff.md
rein resume claude:SESSION_ID --with codex --dry-run
```

`rein resume --with` is a structured handoff convenience alias. It still starts
a new destination session.

## Directions in v0.4.0

| Source | Claude dest | Codex dest |
| ------ | ----------- | ---------- |
| Claude Code | same-vendor native resume | structured handoff |
| Codex CLI | structured handoff | same-vendor native resume |
| Gemini CLI | structured handoff (source-only) | structured handoff (source-only) |
| OpenCode | structured handoff (source-only) | structured handoff (source-only) |
| Grok Build | structured handoff (source-only) | structured handoff (source-only) |

Gemini CLI, OpenCode, and Grok Build are not handoff destinations.

## First reply

The destination's first reply must restate five bullets before mutation:
current goal and latest user request; critical constraints; changed files and
test state; missing capabilities or uncertain evidence; proposed next action.

## Safety

- `--dry-run` plans without launching.
- `--allow-warning ID` acknowledges one exact warning. No wildcards.
- Wrong Git repository: exit `5`.
- Non-TTY dest launch: exit `7`.
- Untested agent version: exit `5` unless `--allow-untested`.
- Capsules stay under `$REINSTATE_HOME/handoffs/` and are excluded from sync.

See the [feature map](/docs/features), [CLI reference](/docs/cli-reference),
and [getting started](/docs/getting-started).

## Prerequisites

- Stable `v0.5.0` installed (`rein version --json`)
- Apple Silicon macOS or native Windows x64 for certified dest-ack
- A logged-in destination Claude Code or Codex CLI in the fail-closed range
- The source session on this device; dest launch needs a real TTY

## Expected evidence

- `--dry-run --json` reports `mode` `structured handoff` and a dest argv without CR/LF
- Live dest first-reply restates the five bullets before mutation
- `rein handoff list --json` keys are `mode` and `handoffs`
- Marker files in the workspace are unchanged unless you later ask the dest to edit

## Failure paths

| Symptom | Cause | Recovery |
| ------- | ----- | -------- |
| Exit `5` | Wrong repo, untested agent, or unsupported dest | Use the source workspace; pin a supported Claude/Codex version |
| Exit `7` | Unacknowledged warning or non-TTY dest launch | Pass each `--allow-warning ID`; run in a real terminal |
| Dest theme/trust hang | Dest home not trusted | Keep dest `CLAUDE_CONFIG_DIR` / `CODEX_HOME`; do not copy oauth files |
| No dest session file | Dest TUI never started | Launch from a local console, not a detached SSH PTY |

See [troubleshooting](/docs/troubleshooting).

## Security boundaries

- Capsules are owner-only under `$REINSTATE_HOME/handoffs/` and excluded from sync
- Known credentials stay excluded; session text can still contain secrets — review the projection
- `--no-redact` is refused for Grok sources
- Dest argv never includes `--dangerously-skip-permissions`

## Related pages

- [Features and commands](/docs/features)
- [CLI reference](/docs/cli-reference)
- [Getting started](/docs/getting-started)
- [Limitations](/docs/limitations)
- [Troubleshooting](/docs/troubleshooting)
