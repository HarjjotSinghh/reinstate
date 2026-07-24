# FAQ

## What is Reinstate?

A local-first, open-source CLI that syncs AI coding agent **sessions** (and
config: MCP servers, skills, settings) across your devices with end-to-end
encryption and bring-your-own storage.

## Why not just use git?

Git is **source** truth. Sessions are **context** truth — the reasoning trail,
tool outputs, and decisions that are not in `git log`. Pulling commits and
asking a new agent to re-derive context is slow and incomplete.

## Will this resume a Claude session inside Codex?

**No.** Same-vendor resume only. Cross-agent value is portable **config /
skills / environment**, not replaying one model's transcript into another.

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
