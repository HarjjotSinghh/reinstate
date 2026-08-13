---
title: "Restore and resume one coding-agent session"
navTitle: "Restore a session"
description: "Dry-run, safely restore, verify, and natively resume one encrypted Claude Code or Codex session on another configured device."
order: 13
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.4.0-rc.6"
updatedAt: 2026-08-01
tags: ["pull", "restore", "native-resume", "backup", "conflicts"]
targetQuery: "restore a coding agent session with Reinstate"
searchIntent: "how-to"
draft: false
noindex: false
---

On the destination device, inspect a session-scoped `pull --dry-run`, close the
matching agent if an existing session will be replaced, run the approved pull,
verify the restored ID with `rein list`, and resume through the same vendor's
native command.

> **Current scope:** Claude Code sessions restore to Claude Code, and Codex
> sessions restore to Codex. Reinstate does not translate native transcripts,
> prepare the Git repository, install dependencies, copy agent credentials, or
> recreate the complete source-machine environment.

## Prerequisites

- The same reviewed Reinstate version installed and initialized on the destination.
- The source profile's exact `profile_id`, storage bucket, prefix, and
  encryption passphrase.
- The same canonical project ID mapped to this device's real absolute path.
- The selected agent installed in a `SUPPORTED` version and native layout.
- The corresponding repository and required dependencies prepared separately.
- The exact remote agent and session ID.

Additional-device `init` must already have verified the existing encrypted
manifest. Do not continue from an empty or newly invented replacement profile.

## Inspect available remote metadata

```sh
rein setup check --json
rein status --json
rein diff --agent AGENT --session SESSION_ID --json
```

`status` and `diff` request the hidden encryption passphrase because they read
the remote manifest. Confirm that the remote key uses the intended agent and
session. Metadata does not prove that the repository checkout matches the
source. The stable `v0.3.0` verified-resume report checks the local repository
before native launch; it does not replace review of this remote restore plan.

## Dry-run the restore

For Claude Code:

```sh
rein pull --agent claude --session SESSION_ID --dry-run --json
```

For Codex:

```sh
rein pull --agent codex --session SESSION_ID --dry-run --json
```

The dry-run fetches, authenticates, decrypts, and validates the selected
artifact; checks adapter compatibility; expands known structural path tokens;
and constructs the restore plan. It omits filesystem and state mutation.

Review:

- `dry_run` is `true`;
- `pulled` is one;
- the snapshot ID matches remote status;
- every planned destination belongs to the destination agent's native layout;
- the destination uses this device's project mapping, not the source path; and
- the reported backup root belongs to this device's Reinstate home.

## Close the agent before replacement

A mutating pull refuses to replace an existing session while Claude Code or
Codex may still be writing it. Save current work and exit every process for the
selected agent normally. Do not force-kill a process during a session write.

A dry-run and a restore of a genuinely new native session remain available
because they do not overwrite an active existing target. Conflict
`--keep-both` also restores a distinct identity, but use it only for a recorded
conflict.

## Restore the selected snapshot

Repeat the reviewed command without `--dry-run`:

```sh
# Claude Code
rein pull --agent claude --session SESSION_ID --json

# Codex
rein pull --agent codex --session SESSION_ID --json
```

Reinstate validates the full artifact before mutation. When the destination
exists, it creates a timestamped private backup, writes a temporary sibling,
syncs it, atomically replaces the target, verifies that the adapter rediscovers
the exact restored session, and only then updates local state.

If the local and remote revisions diverge, the pull records a conflict instead
of silently choosing a winner.

## Verify and resume natively

Claude Code:

```sh
rein list --agent claude --json
claude --resume SESSION_ID
```

Codex:

```sh
rein list --agent codex --json
codex resume SESSION_ID
```

For Claude Code, success requires discovery beneath the exact recomputed
destination project directory key. Finding the same ID elsewhere under the
Claude session tree is not equivalent. For Codex, Reinstate preserves the native
date-partitioned rollout layout and expands the structural
`session_meta.cwd` through the configured local project root.

## Resolve an explicit conflict

Inspect conflict metadata without printing the transcript:

```sh
rein conflicts list --json
rein conflicts show CONFLICT_ID
```

Choose exactly one strategy:

- `--keep-local` publishes the local branch on the current remote head;
- `--keep-remote` backs up local state and restores the remote branch; or
- `--keep-both` preserves local state and restores a vendor-safe fork with a
  distinct structural session ID.

When uncertain, `--keep-both` preserves both native branches:

```sh
rein conflicts resolve CONFLICT_ID --keep-both
```

The conflict record remains if resolution fails. Do not delete it simply to
make the list appear clean.

## Expected evidence

- The dry-run reports one validated plan, the correct destination, and
  `dry_run=true` without creating agent files.
- The mutating pull reports one pulled snapshot and `dry_run=false`.
- An existing target has one timestamped backup under the reported backup
  root.
- `rein list --agent AGENT --json` rediscovers the exact restored or forked ID.
- `claude --resume` or `codex resume` opens the session in the same vendor.
- No transcript content is printed as part of verification.

## Failure paths

- If the mutating pull reports an active process, use
  [active-agent troubleshooting](/docs/troubleshooting#why-does-a-pull-fail-while-the-coding-agent-is-running).
- If Claude cannot see a correctly pulled ID, use
  [Claude path-remapping troubleshooting](/docs/troubleshooting#why-does-claude---resume-not-see-a-pulled-session).
- If local and remote state diverged, follow
  [conflict troubleshooting](/docs/troubleshooting#why-does-reinstate-create-a-session-conflict).
- If authentication fails, do not test passphrases through shell arguments;
  use [passphrase troubleshooting](/docs/troubleshooting#why-does-passphrase-verification-fail-on-a-second-device).

## Security boundaries

Restores necessarily create plaintext session data on the trusted destination
because the native agent must read it. Client-side encryption protects the
remote artifact, not a compromised source or destination OS. Local backups are
also plaintext vendor session files protected by owner-only filesystem
permissions.

Agent account credentials are never restored. Authenticate Claude Code or
Codex independently on the destination. A restored transcript can contain
secrets previously printed inside it; rotate exposed credentials and follow
the [transcript-credential response](/docs/troubleshooting#can-reinstate-upload-credentials-from-a-transcript).

## Related pages

- [Push one selected session](/docs/sync-a-session)
- [Understand adapters and path mapping](/docs/adapters)
- [Review the backup and conflict architecture](/docs/architecture)
- [Check current compatibility](/compatibility)
- [Read all current limitations](/docs/limitations)
