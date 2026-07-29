---
title: "Reinstate limitations and release-candidate boundaries"
description: "Review Reinstate RC8 limitations for agents, operating systems, native resume, storage, snapshots, path remapping, security, and release certification."
order: 15
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.1.0-rc.8"
updatedAt: 2026-07-27
tags: ["limitations", "release-candidate", "compatibility", "same-vendor-resume", "roadmap"]
targetQuery: "Reinstate limitations"
searchIntent: "evaluation"
draft: false
noindex: false
---

Reinstate `v0.1.0-rc.8` is a pre-1.0 continuity layer for encrypted,
same-vendor Claude Code and Codex session sync. It is not yet a certified
stable release, local universal session index, cross-agent translator,
repository synchronizer, remote desktop, coding harness, or universal agent
configuration system.

> **Release gates remain open:** current implementation and local synthetic and
> macOS-arm64 verification are green, while native Windows, macOS amd64, WSL2,
> physical two-device resume, remote CI, and stable-release certification still
> require evidence. This page does not convert planned or locally tested work
> into a certified platform claim.

## Prerequisites

Before evaluating Reinstate, record the exact Reinstate version, agent version,
operating system, architecture, storage provider, and transfer direction.
Check the live [compatibility matrix](/compatibility) and keep independent
backups.

Treat every limitation below as applying to RC8 unless a later changelog and
compatibility update explicitly replaces it. Repository roadmap items are
direction, not current CLI syntax or support.

## Current support boundary

| Area | RC8 boundary |
| --- | --- |
| Agents | Tested Claude Code and Codex CLI layouts only |
| Resume | Claude Code → Claude Code; Codex → Codex |
| Primary platforms | macOS and native Windows, with open release gates |
| WSL | WSL2 documented and smoke-tested target with an open gate; WSL1 unsupported |
| Storage | User-owned S3-compatible object storage; R2 recommended |
| Transfer model | Manual push/pull of full immutable snapshots |
| Configuration | Reinstate session-sync config only |
| Release status | Pre-1.0 release candidate; formats and interfaces may change |

Versions outside the tested stable ranges, including prereleases, are
`UNTESTED`. Recognizable untested sessions may be discovered read-only, but
push and restore fail closed with compatibility exit code `5`. Phase 1 has no
unsafe compatibility override.

## Native resume is same-vendor only

Reinstate preserves each supported vendor's native representation. It does not
make a Claude Code transcript natively resumable in Codex or a Codex rollout
natively resumable in Claude Code. Explicit portable handoffs and experimental
reconstructed conversations are later-phase work and will be labeled as
lossy—not presented as native resume.

Reinstate also does not call vendor agent APIs, execute sessions, schedule
agents, provide an editor or terminal, or replace Claude Code and Codex.

## The development environment is not synchronized

Phase 1 does not transfer:

- Git commits, branches, worktrees, uncommitted files, or repository hosting;
- dependencies, build outputs, virtual environments, containers, or runtimes;
- shell history, environment variables, running processes, or terminal state;
- MCP servers, skills, instructions, hooks, plugins, marketplaces, or settings;
- Anthropic or OpenAI accounts, logins, OAuth tokens, or API credentials.

Prepare the repository and authenticate the same vendor independently on the
destination. A restored session can carry assumptions about a branch, commit,
dependency, tool, or service that is absent there. Automatic environment
fingerprinting and repair are roadmap work.

## Path remapping is structural, not semantic

Adapters normalize recognized structural paths with `${HOME}`,
`${REPO:<id>}`, and configured work-root tokens, then expand them through the
destination mapping. Unknown fields, arbitrary prose, prompts, tool output, and
path-like strings inside transcript text are left unchanged.

The same canonical project ID must map to the correct local root on every
device. Native Windows and WSL2 are distinct devices and must not share one
agent-state directory. A bad or missing mapping can prevent native discovery
even when bytes transfer successfully.

## Full snapshots, manual commands, and conflicts

RC8 transfers complete immutable session snapshots. Append-aware deltas,
chunking, retention controls, garbage collection, and continuous background
sync are not current features. Large Codex rollouts can therefore take longer,
and `--all` can select more data than intended.

There is no semantic transcript merge. When the same session diverges,
Reinstate records a conflict and requires an explicit `--keep-local`,
`--keep-remote`, or `--keep-both` resolution. Conditional manifest updates,
local backups, and conflict forks reduce overwrite risk; they do not replace
an independent backup and recovery plan.

## Storage and availability limits

Current `status`, `diff`, `push`, and `pull` require the configured remote
manifest and its passphrase. The Phase 2 local universal index is not yet
implemented. Reinstate does not operate a hosted storage service or recover a
provider account.

Storage compatibility depends on S3 object operations, conditional request
semantics, credentials, region/endpoint correctness, availability, lifecycle
rules, and quota. Phase 1 has no built-in retention UI or remote snapshot
deletion command. Removing a referenced object through provider tooling can
make a session unrestorable.

## Security and privacy limits

- The passphrase is not stored and cannot be recovered. Losing it means losing
  access to existing remote ciphertext.
- Hard exclusions block known credential artifacts, but a secret printed or
  pasted into ordinary transcript text remains inside the encrypted snapshot.
- Plaintext necessarily exists on trusted local machines during native use and
  restore; a compromised local OS is outside the threat model.
- Object storage can observe access timing, size, bucket and endpoint metadata,
  and opaque object keys even though payloads are encrypted.
- Source availability and checksums are not a formal independent security
  audit. No formal audit is claimed for pre-1.0.
- Local backups contain plaintext vendor session files protected by local
  filesystem permissions.

Never treat encrypted transcript storage as permission to include production
secrets in agent conversations.

## Expected evidence

A supported RC8 workflow should provide all of the following:

- `rein setup check` reports the selected adapter as `SUPPORTED`;
- a scoped push dry-run and pull dry-run each plan one explicit session;
- object storage contains ciphertext-only manifest and snapshot objects;
- the destination plan uses the correct local project root;
- a mutating restore backs up an existing target and is rediscovered by the
  adapter; and
- the same vendor's native resume command opens the exact restored session.

Missing evidence leaves the corresponding platform, version, storage, or
resume claim unverified. A successful synthetic test is not a physical
two-device acceptance result.

## Failure paths

- Start with [troubleshooting](/docs/troubleshooting) for installation, mapping,
  passphrase, manifest, conflict, large-session, transcript-secret, and
  active-agent failures.
- Check [compatibility](/compatibility) before reporting an untested or
  unsupported agent version as a defect.
- Use a private
  [security report](https://github.com/HarjjotSinghh/reinstate/security/policy)
  for exclusion bypass, plaintext remote storage, or another vulnerability.
- Use the [changelog](/changelog) to confirm whether a later release has
  changed a stated boundary.

Do not work around a safety or compatibility refusal by manually moving vendor
files, weakening credential exclusions, inventing an empty manifest, or
claiming a roadmap command exists.

## Security boundaries

Reinstate protects selected session transport with local encryption,
credential exclusions, authenticated artifacts, conditional updates, private
backups, and atomic restore. Users still own endpoint trust, bucket policy,
credential lifecycle, passphrase strength, local-machine security, repository
preparation, transcript sensitivity, provider retention, and backup recovery.

The product is independent of Anthropic and OpenAI. Local adapter compatibility
does not imply vendor partnership, endorsement, account access, or a guarantee
that a future vendor release will preserve its current file layout.

## Related pages

- [Review the exact compatibility matrix](/compatibility)
- [Understand the security model](/docs/security-model)
- [Inspect the architecture and roadmap boundaries](/docs/architecture)
- [Compare Reinstate with adjacent workflows](/compare)
- [Review release changes and open gates](/changelog)
