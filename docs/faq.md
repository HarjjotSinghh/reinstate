# FAQ

## What is Reinstate?

The **continuity layer for coding-agent work**: find, search, resume, and hand
off sessions across agents, projects, environments, and devices. Multi-device
sync uses end-to-end encryption and bring-your-own storage. Single-device users
still get a universal session index and verified resume.

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

**Native resume:** no — same-vendor only (Claude → Claude Code, Codex → Codex).

**Portable handoff (roadmap):** yes, as an *explicit* checkpoint (goal,
decisions, files touched, tests, next action) — not a silent perfect transcript
translation. See [product-strategy.md](product-strategy.md).

## Do I need two computers?

**No.** Multi-device sync is the flagship wedge, but one machine with multiple
agents, sessions, projects, or worktrees is a first-class user. Phase 2+ local
index/search/resume does not require remote storage.

## Is Reinstate another ADE / agent IDE?

**No.** We do not replace Claude Code, Codex, Orca, Conductor, or Cursor as
execution environments. We make work **discoverable, verifiable, portable, and
syncable** across those tools.

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

Pre-1.0. See [ROADMAP.md](../ROADMAP.md) and [CHANGELOG.md](../CHANGELOG.md).
Use with backups; report bugs via GitHub Issues.

## How do I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md).

## How do I report a security issue?

See [SECURITY.md](../SECURITY.md) — private disclosure only.
