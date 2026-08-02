# FAQ

## What is Reinstate?

The **continuity layer for coding-agent work**. Phase 1 implements encrypted,
bring-your-own-storage sync for same-vendor Claude Code and Codex sessions.
Current Phase 2 source adds universal local indexing, literal
search, metadata inspection, and same-vendor resume/fork without cloud
configuration. Verified resume and cross-agent handoffs remain later phases. A
later universal configuration layer will reconcile supported MCP servers,
skills, hooks/loops, plugins, marketplaces, and safe settings across harnesses
and devices.

Spine: *Reinstate is not another place to code — it makes every place you code continuous.*

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

## Do local search and resume require `rein init` or a bucket?

**No.** In Phase 2 source builds:

```bash
rein sessions
rein search "webhook retry"
rein inspect claude:SESSION_ID
rein resume claude:SESSION_ID --dry-run
```

These commands use a private derived index at
`$REINSTATE_HOME/cache/session-index-v1.sqlite`. They do not need a sync
profile, storage credentials, an encryption passphrase, keyring access, or a
network backend. The `v0.2.0-rc.2` public installers contain both the Phase 1
sync surface and Phase 2 local continuity. Development acceptance passed, but
signed tagged-artifact acceptance is still pending before stable promotion.

## Why not just use git?

Git is **source** truth. Sessions are **context** truth — the reasoning trail,
tool outputs, and decisions that are not in `git log`. Pulling commits and
asking a new agent to re-derive context is slow and incomplete.

## Will this resume a Claude session inside Codex?

**Native resume:** no — same-vendor only (Claude → Claude Code, Codex → Codex).

**Portable handoff (roadmap):** yes, as an *explicit* checkpoint (goal,
decisions, files touched, tests, next action) — not a silent perfect transcript
translation. See [product-strategy.md](product-strategy.md).

## Do I need two computers?

**No.** Multi-device sync is the flagship wedge, but one machine with multiple
agents, sessions, projects, or worktrees is a first-class user. Phase 2 local
index/search/resume is built for that workflow and does not require remote
storage.

## Does local search upload or semantically analyze my prompts?

**No.** Search is literal, case-insensitive, and local. The index stores bounded
user-authored prompt text and known metadata/file references with owner-only
permissions. It excludes assistant messages/reasoning, tool output,
environment dumps, credentials, and auth stores. `search` does not print the
matching passage, and `inspect` caps its terminal-safe user preview at 160
Unicode code points.

User prompts can themselves contain sensitive text. The local index is not a
redaction or DLP product, and a compromised local machine remains outside the
threat model.

## Is Reinstate another ADE / agent IDE?

**No.** We do not replace Claude Code, Codex, Orca, Conductor, or Cursor as
execution environments. We make work **discoverable, verifiable, portable, and
syncable** across those tools.

## Will Reinstate configure the same MCP server in every harness?

**That is planned after Phase 1.** The target is to define a server once with
`rein mcp add`, preview native changes, and apply it across selected Claude
Code, Codex, Grok, OpenCode, Gemini CLI, and future adapters. The same model
extends to skills, instructions, hooks/loops, plugins, marketplaces, and safe
settings.

Reinstate will normalize intent and render each harness's real schema; it will
not blindly copy one tool's config file. See
[universal-configuration.md](universal-configuration.md).

## Will MCP authentication also carry across tools and devices?

Reinstate should make authentication status visible and coordinate supported
login flows, reducing repetition where the MCP protocol, provider, or harness
allows safe reuse. It will not sync raw API keys, OAuth tokens, cookies, or
vendor credential stores. When safe reuse is not supported, each target may
still require its official login flow.

## Is my data sent to Reinstate servers?

**No** for the open-source CLI. You point at **your** R2/S3-compatible bucket. A future
optional hosted convenience layer would still be zero-knowledge (ciphertext
only); it is not required.

## What if I lose my passphrase?

You cannot decrypt remote data. That is intentional (zero-knowledge). Keep the
passphrase in a password manager.

## Does this work offline?

Session files remain local, but the current `status`, `diff`, `push`, and `pull`
commands read the remote manifest and need access to your storage backend.
Phase 2 `sessions`, `search`, and `inspect` work offline and without sync
configuration. Native resume/fork needs only the local vendor executable and
recorded workspace.

## Windows + Mac?

It is the primary design target. Stable Phase 1 path remapping passed its
23-row physical RC8 matrix on native macOS arm64 and Windows 11 amd64. Phase 2
development acceptance passed all 30 required rows on those devices at
`b952d38`; signed release artifacts must repeat the release matrix. macOS amd64
and WSL2 evidence is reported separately rather than fabricated.

## Is this affiliated with Anthropic / OpenAI / Google / xAI?

**No.** Independent Apache-2.0 project by Harjot Singh Rana.

## Production ready?

Pre-1.0. Stable `v0.1.0` completed Phase 1. The `v0.2.0-rc.2` candidate contains
development-accepted Phase 2; exact tagged-artifact certification and stable
publication are next. See
[ROADMAP.md](../ROADMAP.md) and [CHANGELOG.md](../CHANGELOG.md). Use with
backups; report bugs via GitHub Issues.

## How do I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md).

## How do I report a security issue?

See [SECURITY.md](../SECURITY.md) — private disclosure only.
