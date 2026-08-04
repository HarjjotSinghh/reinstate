# FAQ

## What is Reinstate?

The **continuity layer for coding-agent work**. Phase 1 implements encrypted,
bring-your-own-storage sync for same-vendor Claude Code and Codex sessions.
Universal local search, verified resume, and core cross-agent continuation are
later phases. A later universal configuration layer will also reconcile supported
MCP servers, skills, hooks/loops, plugins, marketplaces, and safe settings
across harnesses and devices.

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

## Why not just use git?

Git is **source** truth. Sessions are **context** truth — the reasoning trail,
tool outputs, and decisions that are not in `git log`. Pulling commits and
asking a new agent to re-derive context is slow and incomplete.

## Will this resume a Claude session inside Codex?

**In the current CLI:** no. Phase 1 native resume is same-vendor only (Claude →
Claude Code, Codex → Codex).

**Core Phase 4 roadmap:** yes. The flagship case is hitting the Claude Code
usage limit and continuing the same task in Codex without another Claude model
call; Codex → Claude must work too. Reinstate will create a new linked
destination session from a portable continuity capsule containing task state,
workspace/test truth, selected or full portable visible history, tool evidence,
capability differences, lineage, and a component-level fidelity report.

See [cross-agent continuation](cross-agent-continuation.md).

## Will it preserve the exact conversation, tool calls, and system messages?

It will preserve the immutable raw source artifact and carry every **portable,
visible** element it safely can: user messages, visible assistant replies, tool
calls/results as inert evidence, timestamps, files, decisions, tests, and
attachments supported by the destination.

It cannot honestly promise an identical foreign runtime. Vendor message roles,
tools, IDs, sandboxes, policies, context compaction, and reasoning state differ.
Hidden chain-of-thought or unavailable system prompts cannot be recovered;
credentials, approvals, and live process state must not transfer. Source
system/developer messages are audit history and are never promoted to
destination policy automatically. Historical tool calls are never re-executed.

The default cross-agent mode is a structured handoff. A fuller reconstructed
conversation is experimental and available only for tested source/target/version
pairs. The CLI will label each component `exact`, `normalized`, `summarized`,
`referenced`, or `omitted` with a reason.

## Do I need two computers?

**No.** Multi-device sync is the flagship wedge, but one machine with multiple
agents, sessions, projects, or worktrees is a first-class user. Phase 2+ local
index/search/resume does not require remote storage.

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
Offline indexing and search are Phase 2 work.

## Windows + Mac?

It is the primary design target, and path remapping is implemented. Exact
native Windows, macOS amd64, WSL2, and two-device release-candidate acceptance
are still open gates; see the [roadmap](../ROADMAP.md).

## Is this affiliated with Anthropic / OpenAI / Google / xAI?

**No.** Independent Apache-2.0 project by Harjot Singh Rana.

## Production ready?

Pre-1.0. See [ROADMAP.md](../ROADMAP.md) and [CHANGELOG.md](../CHANGELOG.md).
Use with backups; report bugs via GitHub Issues.

## How do I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md).

## How do I report a security issue?

See [SECURITY.md](../SECURITY.md) — private disclosure only.
