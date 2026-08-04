# FAQ

## What is Reinstate?

The **continuity layer for coding-agent work**. Phase 1 implements encrypted,
bring-your-own-storage sync for same-vendor Claude Code and Codex sessions.
Universal search, verified resume, core cross-agent continuation, and
cross-harness configuration are later phases.

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

Git is **source** truth. Sessions are **context** truth — the reasoning trail,
tool outputs, and decisions that are not in `git log`. Pulling commits and
asking a new agent to re-derive context is slow and incomplete.

## Will this resume a Claude session inside Codex?

**Current CLI:** no—native resume is same-vendor only.

**Core Phase 4 roadmap:** yes. Claude Code ↔ Codex quota/outage continuation is
the first pair, with Gemini CLI and OpenCode following. Reinstate creates a new
linked destination session from task state, portable visible history,
workspace/tool/test evidence, capability differences, and a fidelity report.

It preserves the immutable source but cannot claim an identical foreign
runtime. Credentials, approvals, hidden reasoning, and live processes do not
transfer. Source system/developer messages remain audit history, and historical
tools never re-execute. Reconstructed native history is experimental and
pair/version-specific.

See [Cross-agent continuation](cross-agent-continuation.md).

## Will Reinstate configure the same MCP server in every harness?

**That is planned after Phase 1.** The target is to define an MCP server once,
preview the native changes, and apply it across selected Claude Code, Codex,
Grok, OpenCode, Gemini CLI, and future adapters. The model also covers
skills/instructions, hooks/loops, plugins, marketplaces, and safe settings.

See [Universal agent configuration](universal-configuration.md).

## Will MCP authentication also carry across tools and devices?

Reinstate should report authentication state and coordinate supported official
login flows. Raw API keys, OAuth tokens, cookies, and vendor credential stores
will not be synced. Safe reuse is possible only where the protocol, provider,
or harness explicitly supports it; otherwise a target may still require its
own login.

## Is my data sent to Reinstate servers?

**No** for the open-source CLI. You point at **your** R2/S3/WebDAV/etc. A future
optional hosted convenience layer would still be zero-knowledge (ciphertext
only); it is not required.

## What if I lose my passphrase?

You cannot decrypt remote data. That is intentional (zero-knowledge). Keep the
passphrase in a password manager.

## Does this work offline?

Local status/diff from the manifest works offline. Push/pull need network to
your storage backend.

## Windows + Mac?

Yes — that dual setup is a primary design target. Path remapping is the hard
problem we optimize for.

## Is this affiliated with Anthropic / OpenAI / Google / xAI?

**No.** Independent Apache-2.0 project by Harjot Singh Rana.

## Production ready?

Pre-1.0. See [ROADMAP.md](https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md) and [CHANGELOG.md](https://github.com/HarjjotSinghh/reinstate/blob/main/CHANGELOG.md).
Use with backups; report bugs via GitHub Issues.

## How do I contribute?

See [CONTRIBUTING.md](https://github.com/HarjjotSinghh/reinstate/blob/main/CONTRIBUTING.md).

## How do I report a security issue?

See [SECURITY.md](https://github.com/HarjjotSinghh/reinstate/blob/main/SECURITY.md) — private disclosure only.
