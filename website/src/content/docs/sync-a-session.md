---
title: "Push one coding-agent session"
navTitle: "Push a session"
description: "Select, dry-run, encrypt, and push one supported Claude Code or Codex session to user-owned S3-compatible storage with Reinstate."
order: 12
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.2.0-rc.2"
updatedAt: 2026-08-01
tags: ["push", "session-sync", "claude-code", "codex", "dry-run"]
targetQuery: "push a coding agent session with Reinstate"
searchIntent: "how-to"
draft: false
noindex: false
---

List local sessions, choose one explicit Claude Code or Codex session ID,
inspect `push --dry-run`, and then repeat the same scoped command without
`--dry-run`. Reinstate normalizes supported structural paths, encrypts the
snapshot locally, uploads ciphertext, and conditionally advances the encrypted
remote manifest.

> **Current scope:** Reinstate pushes full immutable snapshots for supported
> same-vendor sessions. It does not continuously watch agent directories,
> synchronize Git state, or translate Claude Code sessions into Codex
> sessions.

## Prerequisites

- Reinstate `v0.2.0-rc.2` initialized on the source device.
- A successful `rein setup check` with the selected adapter marked
  `SUPPORTED`.
- An existing encrypted profile and reachable S3-compatible storage.
- A configured canonical project mapping for the session's repository.
- The profile passphrase available for private entry.
- One harmless Claude Code or Codex session that has finished writing.

Commit or otherwise preserve the repository separately. Reinstate transfers
the supported agent session artifact, not source files, dependencies,
credentials, shell state, or running processes.

## Discover the exact session

List metadata for the agent you intend to sync:

```sh
rein list --agent claude --json
rein list --agent codex --json
```

Choose the ID, agent, and canonical project deliberately. `list` reads local
vendor metadata; it does not contact storage or print the transcript.

Use Claude Code IDs only with `--agent claude` and Codex IDs only with
`--agent codex`. Same-vendor identity is part of the snapshot contract.

## Inspect the push without mutation

For Claude Code:

```sh
rein push --agent claude --session SESSION_ID --dry-run --json
```

For Codex:

```sh
rein push --agent codex --session SESSION_ID --dry-run --json
```

Wait for Reinstate's hidden passphrase prompt before typing. The dry-run reads
and authenticates the current remote manifest, checks the local revision,
plans adapter export, builds and validates the portable artifact, and reports
what would be uploaded. It does not publish a snapshot, advance the manifest,
or update local sync state.

Review these fields:

- `dry_run` is `true`;
- exactly one candidate snapshot is planned, unless the session is unchanged;
- `skipped` reflects an already synchronized local/remote revision;
- the selected agent and session match your intended scope; and
- no compatibility, credential-exclusion, storage, or conflict error appears.

## Push the selected session

Exit the selected agent normally so its file is not changing during export,
then repeat the approved command without `--dry-run`:

```sh
# Claude Code
rein push --agent claude --session SESSION_ID --json

# Codex
rein push --agent codex --session SESSION_ID --json
```

The command exports one supported vendor-native session, replaces recognized
structural project paths with portable tokens, hard-excludes known credential
artifacts, encrypts the snapshot with age passphrase encryption, uploads the
immutable `.age` object, and conditionally updates `manifest.age`.

If the local and remote session already match recorded state, Reinstate skips
the unchanged snapshot instead of creating a new revision.

## About `--all`

`rein push --all` is available, but neither Reinstate nor a coding agent should
select every discovered session on your behalf. `--all` can include large,
old, irrelevant, or unexpectedly sensitive transcripts and increases transfer
time under the current full-snapshot model.

Prefer one agent and one session until you have reviewed the complete discovery
set and deliberately accept its storage and disclosure scope.

## Verify remote state

```sh
rein status --json
rein diff --agent AGENT --session SESSION_ID --json
```

Replace `AGENT` with `claude` or `codex`. Both commands currently read the
remote manifest and request the passphrase. They compare metadata; they do not
print transcript content.

## Expected evidence

- The approved dry-run reports one would-be snapshot and `dry_run=true`, or
  reports the selected session as unchanged.
- The mutating command reports one pushed snapshot and `dry_run=false`.
- `status` includes the `AGENT:SESSION_ID` remote key, snapshot ID, and remote
  revision.
- A repeated push with no local changes reports the session as skipped.
- Object storage contains an encrypted immutable snapshot and encrypted
  `manifest.age`; it never receives a plaintext session artifact.

Do not decrypt or print a transcript merely to prove that upload succeeded.

## Failure paths

- A conflict exit code `6` means the remote head changed or both sides
  diverged; follow
  [session-conflict troubleshooting](/docs/troubleshooting#why-does-reinstate-create-a-session-conflict).
- A slow large Codex rollout is subject to the
  [full-snapshot limitation](/docs/troubleshooting#why-is-a-large-codex-session-slow-to-sync).
- A missing remote manifest on an established device is a storage/profile
  failure, not permission to create an empty profile. Use
  [remote-manifest troubleshooting](/docs/troubleshooting#why-does-reinstate-report-a-remote-profile-manifest-is-missing).
- If the adapter is `UNTESTED` or `UNSUPPORTED`, stop at compatibility exit
  code `5`; Phase 1 has no unsafe write override.

## Security boundaries

Known auth files, OAuth material, credential stores, tokens, caches, logs, and
regenerable dependency trees are hard-excluded. A secret that an agent printed
inside ordinary transcript text remains part of the selected encrypted
snapshot. Reinstate does not semantically redact transcript prose.

Never put a passphrase in a normal environment variable, argument, prompt
transcript, or issue. Remote encryption reduces storage-provider exposure, but
the local source, destination, passphrase, storage credentials, and transcript
handling remain your responsibility.

## Related pages

- [Restore the session on another device](/docs/restore-a-session)
- [Review storage behavior and object layout](/docs/storage)
- [Check supported agents and versions](/compatibility)
- [Follow the Claude Code outcome guide](/guides/sync-claude-code-sessions-across-devices)
- [Follow the Codex outcome guide](/guides/sync-codex-sessions-across-devices)
