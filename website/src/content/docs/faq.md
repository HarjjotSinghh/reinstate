---
title: "Reinstate frequently asked questions"
description: "Get direct answers about Reinstate's current session-sync scope, supported agents, encryption, storage, offline behavior, cross-agent handoffs, and roadmap."
order: 7
updatedAt: 2026-07-27
tags: ["faq", "session-sync", "claude-code", "codex", "security"]
targetQuery: "what is Reinstate"
searchIntent: "navigational"
draft: false
noindex: false
---

## What is Reinstate?

Reinstate is an open-source continuity layer for coding-agent work. Phase 1
implements encrypted, bring-your-own-storage sync for same-vendor Claude Code
and Codex sessions; universal search, verified resume, portable handoffs, and
cross-harness configuration are later phases.

## Is Reinstate free and open source?

Yes. Reinstate is available under the Apache-2.0 license and the CLI does not
require a Reinstate account. You provide your own S3-compatible storage and are
responsible for any storage-provider charges.

## What is `rein` vs `reinstate`?

**Same tool.**

| Name | Role |
| ---- | ---- |
| **Reinstate** | Product / brand / docs / repo |
| **`reinstate`** | Full CLI binary name |
| **`rein`** | Short alias (preferred day-to-day) |

```bash
rein version
reinstate version   # identical behavior
```

Config and data live under `~/.reinstate/` either way.

## Why not just use git?

Git stores source history; it does not store the complete vendor-native
coding-agent session. Reinstate handles session context while Git remains the
source of truth for commits, branches, and repository collaboration.

## Will this resume a Claude session inside Codex?

**Native resume:** no — same-vendor only. A later explicit portable handoff can
carry a lossy task checkpoint without pretending to translate native history.

## Do I need two computers?

No. Multi-device sync is the first release wedge, while later local
index/search/resume features are intended to help developers manage fragmented
sessions and agents on one computer without a remote-storage dependency.

## Is Reinstate a cloud IDE or coding-agent harness?

No. Claude Code, Codex, and other coding agents execute the work. Reinstate
finds, verifies, restores, hands off, and syncs continuity state around those
tools; it does not provide an editor, terminal emulator, or agent scheduler.

## Will Reinstate configure the same MCP server in every harness?

**That is planned after Phase 1.** The target is to define an MCP server once,
preview the native changes, and apply it across selected Claude Code, Codex,
Grok, OpenCode, Gemini CLI, and future adapters. The model also covers
skills/instructions, hooks/loops, plugins, marketplaces, and safe settings.

See [Universal agent configuration](/docs/universal-configuration).

## Will MCP authentication also carry across tools and devices?

Reinstate should report authentication state and coordinate supported official
login flows. Raw API keys, OAuth tokens, cookies, and vendor credential stores
will not be synced. Safe reuse is possible only where the protocol, provider,
or harness explicitly supports it; otherwise a target may still require its
own login.

## Is my data sent to Reinstate servers?

No. The open-source CLI sends ciphertext to the S3-compatible bucket you
configure, such as Cloudflare R2 or Amazon S3. Reinstate does not operate a
required storage service.

## What if I lose my passphrase?

You cannot decrypt remote data. That is intentional (zero-knowledge). Keep the
passphrase in a password manager.

## Does this work offline?

Session files remain local, but the current `status`, `diff`, `push`, and
`pull` commands read the remote manifest and need access to your storage
backend. Offline indexing and search are Phase 2 roadmap work.

## Does Reinstate support Windows and macOS?

Windows ↔ macOS is the primary design target, and the structural path-remapping
implementation is in the release candidate. Exact native Windows, macOS amd64,
WSL2, and two-device certification remain open release gates; check the
[roadmap](https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md).

## Is this affiliated with Anthropic / OpenAI / Google / xAI?

**No.** Independent Apache-2.0 project by Harjot Singh Rana.

## Production ready?

No. Reinstate is a pre-1.0 release candidate while native acceptance gates are
open. See the [roadmap](https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md)
and [changelog](https://github.com/HarjjotSinghh/reinstate/blob/main/CHANGELOG.md),
use it with backups, and report bugs through GitHub Issues.

## How do I contribute?

See [CONTRIBUTING.md](https://github.com/HarjjotSinghh/reinstate/blob/main/CONTRIBUTING.md).

## How do I report a security issue?

See [SECURITY.md](https://github.com/HarjjotSinghh/reinstate/blob/main/SECURITY.md) — private disclosure only.
