---
title: "Reinstate frequently asked questions"
navTitle: "FAQ"
description: "Get direct answers about Reinstate's current session-sync scope, supported agents, encryption, storage, offline behavior, cross-agent handoffs, and roadmap."
order: 7
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.3.0"
updatedAt: 2026-08-01
tags: ["faq", "session-sync", "claude-code", "codex", "security"]
targetQuery: "what is Reinstate"
searchIntent: "answer"
draft: false
noindex: false
---

Reinstate answers common continuity questions with one present-scope rule:
supported Claude Code and Codex sessions resume in the same vendor, while
cross-agent translation and broader continuity features remain later roadmap
work.

## What is Reinstate?

Reinstate is an open-source continuity layer for coding-agent work. Phase 1
implements encrypted, bring-your-own-storage sync for same-vendor Claude Code
and Codex sessions. Phase 2 adds configless local indexing, literal search,
metadata inspection, a TTY switcher, and same-vendor native resume/fork.
Stable `v0.3.0` adds verified resume; its tagged-artifact
acceptance is pending. Portable handoffs and cross-harness configuration remain
later phases.

See [What is Reinstate?](/about/reinstate) for current product facts, non-goals,
roadmap boundaries, and maintainer information.

## What is a coding-agent session?

A coding-agent session is the vendor-native record of an ongoing agent task,
including its session identifier, conversation events, tool activity, working
directory, and enough tool-specific state for that same agent to resume it.
Reinstate preserves supported native session artifacts; it does not convert a
Claude Code transcript into a Codex rollout or vice versa. See
[adapter internals](/docs/adapters).

## Is Reinstate free and open source?

Yes. Reinstate is available under the Apache-2.0 license and the CLI does not
require a Reinstate account. You provide your own S3-compatible storage and are
responsible for any storage-provider charges.

See the [open-source project page](/open-source) for the repository, license,
governance, security policy, and contribution paths.

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

See [Reinstate compared with Git](/docs/comparison) for the distinct roles of
source history and coding-agent session continuity.

## Why not copy the session files manually?

Manual copying can work for a narrow same-machine experiment, but it does not
provide Reinstate's canonical project identity, structural macOS ↔ Windows
path remapping, client-side encryption, credential exclusions, immutable
snapshots, conflict checks, or atomic restore backup. Reinstate remains
same-vendor: those safeguards do not translate one agent's transcript into
another format. See [the architecture](/docs/architecture).

## Can I restore a session without copying the whole repository?

Yes. Reinstate transfers supported session artifacts, not the repository
itself. The destination still needs an appropriate checkout of the source
repository, branch, and dependencies through Git or the developer's normal
workflow. Map that local checkout to the same canonical project ID before
restoring the session. See [installation and sync](/docs/getting-started).

## Will this resume a Claude session inside Codex?

**Native resume:** no — same-vendor only. A later explicit portable handoff can
carry a lossy task checkpoint without pretending to translate native history.

## Do I need two computers?

No. Multi-device sync is the first release wedge, while Phase 2 local
index/search/resume helps developers manage fragmented sessions and agents on
one computer without a remote-storage dependency.

## Is Reinstate a cloud IDE or coding-agent harness?

No. Claude Code, Codex, and other coding agents execute the work. Reinstate
finds, verifies, restores, hands off, and syncs continuity state around those
tools; it does not provide an editor, terminal emulator, or agent scheduler.

## Is Reinstate a remote desktop or live terminal mirror?

No. Remote desktop streams or controls another machine's live environment.
Reinstate transfers supported encrypted session state into the destination
agent's native local layout so work can continue on that device. It does not
stream a screen, terminal, process, or active agent. Review the
[work-and-personal-computer use case](/use-cases/work-and-personal-computers).

## Is Reinstate a general backup tool?

No. Reinstate creates encrypted, immutable snapshots of supported coding-agent
session artifacts, but it is not a whole-computer, repository, credential, or
general file backup system. Keep independent backups and use Git for source
history. The [security model](/security) explains what is deliberately
excluded.

## Will Reinstate configure the same MCP server in every harness?

**That is planned after Phase 1.** The target is to define an MCP server once,
preview the native changes, and apply it across Claude Code, Codex, and future
verified adapters. The model also covers
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

Session files remain local. Phase 2 `sessions`, `search`, and `inspect` work
offline without sync configuration. The `status`, `diff`, `push`, and `pull`
commands still read the remote manifest and need access to your storage backend.

## Does Reinstate support Windows and macOS?

Windows ↔ macOS is the primary design target, and the structural path-remapping
implementation shipped in v0.1.0, and v0.2.0 physical acceptance passed on
Apple Silicon macOS and native Windows x64. Intel macOS, WSL2, and other POSIX
packages are preview and unverified; check the
[roadmap](https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md).

## What happens when Claude Code or Codex changes its session format?

Reinstate adapters validate known formats and tested agent-version ranges. An
unknown or untested layout fails closed for writes instead of guessing.
Maintainers update fixtures, adapter logic, compatibility evidence, and
release notes before expanding the supported range. Check the
[compatibility matrix](/compatibility) and [changelog](/changelog) before
syncing after an agent upgrade.

## Is this affiliated with Anthropic / OpenAI / Google / xAI?

**No.** Independent Apache-2.0 project by Harjot Singh Rana.

## Production ready?

`v0.3.0` is the current pre-1.0 stable release on Apple Silicon macOS and
native Windows x64, with Phase 3 verified resume. Intel macOS, WSL2, and other
POSIX packages are optional and unverified. See the
[roadmap](https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md)
and [changelog](https://github.com/HarjjotSinghh/reinstate/blob/main/CHANGELOG.md),
use it with backups, and report bugs through GitHub Issues.

## How do I contribute?

See [CONTRIBUTING.md](https://github.com/HarjjotSinghh/reinstate/blob/main/CONTRIBUTING.md).

## How do I report a security issue?

See [SECURITY.md](https://github.com/HarjjotSinghh/reinstate/blob/main/SECURITY.md) — private disclosure only.
