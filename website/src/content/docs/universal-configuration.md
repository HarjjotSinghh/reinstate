---
title: "Universal agent configuration roadmap"
navTitle: "Universal configuration"
description: "Explore the planned non-secret desired-state model for rendering MCP servers, skills, hooks, plugins, marketplaces, and safe settings across coding agents."
order: 6
author: "Harjot Singh Rana"
status: planned
schemaType: tech-article
version: "roadmap"
updatedAt: 2026-08-04
tags: ["roadmap", "mcp", "skills", "plugins", "agent-configuration"]
targetQuery: "sync MCP servers and coding-agent configuration"
searchIntent: "solution"
draft: false
noindex: false
---

Cross-agent task handoff is a separate Phase 4 continuity-capsule capability.
Configuration reconciliation can repair supported non-secret destination
capabilities before launch, but it never converts raw vendor config, source
credentials, or approvals into destination session state. See
[Cross-agent continuation](/docs/cross-agent-continuation).

> Status: planned after Phase 1. These commands are design direction, not part
> of the current `v0.1` CLI.

Reinstate will eventually make the **agent development environment** portable,
not only session history. Define an MCP server, skill, instruction, hook/loop,
plugin, marketplace, or safe setting once, then reconcile it across supported
harnesses and devices.

A developer using Claude Code, Codex, Grok, OpenCode, or Gemini CLI should not
have to paste the same MCP JSON into every tool. Reinstate will keep one
versioned, non-secret desired-state profile and use configuration adapters to
render each harness's real native format.

```bash
# Proposed workflow; exact syntax is not stable.
rein mcp add mobbin --url https://example.invalid/mcp --target all
rein config diff --target claude,codex,grok,opencode,gemini
rein config apply --all
rein config sync
```

## Normalize, then render

Harness schemas and installation mechanisms differ. Reinstate will not copy one
tool's raw configuration tree into another.

```text
native harness config
        ↕ import / render
configuration adapter
        ↕
Reinstate desired-state profile
        ↕ encrypt / sync
other devices
```

Adapters must declare supported capabilities, report unsupported or lossy
mappings, preserve unrelated native settings, preview changes, back up files,
write atomically, and fail closed on unverified schemas.

## Authentication stays safe

The intended outcome is **configure once, authenticate as few times as safely
possible**:

- synced profiles contain secret references, never raw API keys, OAuth tokens,
  cookies, or vendor credential stores;
- local values may come from the OS keychain or an explicit secret provider;
- authentication status should be visible per device and harness;
- Reinstate may coordinate official login flows; and
- token reuse happens only when the protocol, provider, or harness explicitly
  supports it.

Some targets may still require separate official login when safe reuse is not
available.

## Executable extensions

Skills, plugins, hooks, loops, and marketplaces may introduce executable code.
Applying them will require source/version pinning, digest or signature checks
where available, a dry-run of commands and permissions, explicit consent, and
rollback.

Reinstate coordinates native harness mechanisms. It does not become a coding
harness, plugin runtime, or agent marketplace.

Full design:
[docs/universal-configuration.md](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/universal-configuration.md).
