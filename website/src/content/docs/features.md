---
title: "Reinstate features and commands from v0.1.0 to v0.5.1"
navTitle: "Features and commands"
description: "See which Reinstate features and CLI commands shipped in stable v0.1.0 through v0.5.1, including encrypted sync, verified resume, structured handoff, and agent coverage."
order: 2
author: "Harjot Singh Rana"
status: current
schemaType: tech-article
version: "v0.6.0-rc.6"
updatedAt: 2026-09-05
tags: ["cli", "features", "session-sync", "handoff", "verified-resume"]
targetQuery: "Reinstate features and commands"
searchIntent: "navigational"
draft: false
noindex: false
---

Stable **`v0.5.1`** is the current release. It includes every shipped surface
from Phase 1 through Phase 5. Use this page as the versioned feature map. The
`v0.6.0-rc.6` release candidate — Reinstate Hop (cloud continuity) and the
interactive CLI — is covered in its own section below and is not yet a
stable claim: its acceptance is native Windows x64 only, with Apple Silicon
macOS deferred. Universal configuration, a Reinstate console, and team
continuity remain later phases.

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
| `v0.5.0` | 2026-08-21 | Universal agent coverage: 18-agent catalog, `rein doctor --agents` |
| `v0.5.1` | 2026-08-21 | Patch: pure-Go SQLite driver bump; no product-surface change |

Mandatory verified platforms remain Apple Silicon macOS and native Windows
x64. Intel macOS and Linux/WSL2 stay preview/unverified. The `v0.6.0-rc.6`
release candidate is not in this table because it is not yet a stable line;
see the release-candidate section near the bottom of this page.

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
| `rein doctor --agents [--acceptance-matrix]` | 0.5 | no |
| `rein login`, `rein whoami` | 0.6.0-rc.6 (candidate) | no |
| `rein init --hop`, `rein account init\|recover\|join\|status` | 0.6.0-rc.6 (candidate) | yes |
| `rein devices [approve\|revoke]`, `rein hop status\|credentials` | 0.6.0-rc.6 (candidate) | yes |
| `rein sync verify`, `rein sync migrate --to byo` | 0.6.0-rc.6 (candidate) | yes |
| `rein daemon run\|install\|start\|stop\|uninstall\|status` | 0.6.0-rc.6 (candidate) | yes |
| bare `rein` interactive switcher, `ctrl+k` palette, `--plain` | 0.6.0-rc.6 (candidate) | no |

Flag-level syntax lives in the [CLI reference](/docs/cli-reference). Rows
marked "candidate" ship in `v0.6.0-rc.6`, not in stable `v0.5.1`.

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

## v0.5.0 — universal agent coverage

```sh
rein doctor --agents --json
rein doctor --agents --acceptance-matrix --json
```

One catalog of eighteen coding agents, one support-tier ladder (T0–T5), and a
conformance suite that holds each descriptor's capabilities to its declared
tier. Session discovery lands for six T1 agents (Kimi Code CLI, Qwen Code, Pi,
Cursor CLI, GitHub Copilot CLI, Cline) and three more T2 handoff sources
(Gemini CLI, OpenCode, Grok Build). Same-vendor native resume, fork, and
encrypted sync stay Claude Code and Codex CLI only. Verified ranges reach
Claude Code `2.1.238` and Codex CLI `0.149.0` on dual-platform physical resume
evidence. Dual-platform tagged-artifact acceptance passed across the full
150-row matrix on Apple Silicon macOS and native Windows x64.

`v0.5.1` is a patch on top of `v0.5.0`: a pure-Go SQLite driver bump with no
product-surface change, re-verified on both platforms.

## Not in stable v0.5.1

Reconstructed cross-agent sessions; Gemini CLI, OpenCode, Grok Build, Qwen
Code, or Kimi Code CLI as handoff destinations or native-resume agents;
encrypted sync beyond Claude Code and Codex CLI; Reinstate Hop; the
interactive CLI; universal configuration; and Intel macOS or Linux as
verified platforms.

## v0.6.0-rc.6 — Reinstate Hop and the interactive CLI (release candidate)

`v0.6.0-rc.6` is a release candidate, not a stable claim: stable remains
`v0.5.1`, and this candidate's acceptance is native Windows x64 only, with
Apple Silicon macOS deferred until that hardware returns.

```sh
rein login
rein init --hop
rein account init
rein push --agent claude --session SESSION_ID   # or --all for every discovered session
rein hop status
rein sync verify
```

Reinstate Hop ships as ordinary `rein` commands with no build tag or feature
flag: passwordless sign-in (`rein login` / `rein whoami`), the locker and
keyring (`rein init --hop`, `rein account init\|recover\|join\|status`),
device pairing and revocation (`rein devices`, `approve`, `revoke`),
verification (`rein sync verify`), leaving Hop (`rein sync migrate --to
byo`), and a resident daemon (`rein daemon`). The hosted control plane is
not yet open — `rein login` against the default URL reports that in one
sentence, `REINSTATE_HOP_URL` / `[hop] url` point the client elsewhere, and
no pricing, billing, or sign-up ships. **OpenCode reaches T5** (encrypted
sync, the first embedded-SQLite agent to sync) and **Kimi Code CLI reaches
T2** (handoff source). See [Reinstate Hop](/docs/hop) for the full command
reference.

This candidate also carries the interactive CLI experience introduced by an
earlier, uncertified candidate, unchanged: bare `rein` opens a session
switcher with a readiness verdict
per row, a handoff studio, a setup wizard with `--link` / `--paste` pairing
codes, and a `ctrl+k` command palette. Every `--json` document and non-TTY
byte stream stays byte-identical to `v0.5.1`; `--plain` and
`REINSTATE_NO_TUI` freeze the old output. OpenCode and Grok Build gain
verified native resume, and OpenCode, Grok Build, and Qwen Code become
handoff destinations alongside Claude Code and Codex.

See [handoff](/docs/handoff), [getting started](/docs/getting-started),
[CLI reference](/docs/cli-reference), [Reinstate Hop](/docs/hop), and the
[roadmap](/roadmap).
