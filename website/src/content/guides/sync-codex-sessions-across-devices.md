---
title: "How to Sync Codex Sessions Across Devices"
description: "Set up encrypted Codex CLI session sync between two computers with Reinstate, including project mapping, dry runs, safe restore, and native resume."
answer: "To sync a Codex CLI session across devices, install Reinstate on both computers, assign the repository one canonical project ID, push one explicit Codex session, dry-run and pull it on the destination, then continue it with the native codex resume command."
author: "Harjot Singh Rana"
publishedAt: 2026-07-27
updatedAt: 2026-07-27
reviewedAt: 2026-07-27
tags: ["Codex CLI", "session sync", "multi-device", "path mapping", "end-to-end encryption"]
targetQuery: "how to sync Codex sessions across devices"
searchIntent: "how-to"
related:
  - title: "Codex integration details"
    path: "/integrations/codex"
  - title: "Getting started documentation"
    path: "/docs/getting-started"
  - title: "Compatibility and tested versions"
    path: "/compatibility"
  - title: "Security model"
    path: "/docs/security-model"
draft: false
noindex: false
agent: "codex"
difficulty: "intermediate"
estimatedMinutes: 12
prerequisites:
  - "Codex CLI installed on the source and destination devices"
  - "An S3-compatible bucket and its endpoint and credentials"
  - "The same long encryption passphrase available privately on both devices"
---

## What this workflow does

This workflow transfers one supported Codex CLI session through object storage
you control. Reinstate discovers the native rollout, normalizes the structural
project working directory through a canonical project token, encrypts the
snapshot locally, and uploads ciphertext. It reverses that mapping on the
destination before restoring the rollout to Codex's native date-partitioned
session layout.

This is **Codex to Codex** same-vendor continuity. Phase 1 does not turn a Codex
rollout into a Claude Code transcript, and it does not silently reconstruct a
session for another coding agent.

The current public installer pins `v0.1.0-rc.6`. It remains release-candidate
software while the remaining native platform and physical two-device
acceptance rows are completed. Confirm the supported Codex CLI range on the
[compatibility page](/compatibility) before transferring real work.

## Before you begin

Prepare:

- macOS, native 64-bit Windows, Linux, or WSL2 on each device;
- Codex CLI installed on both devices;
- a Cloudflare R2, Amazon S3, or compatible bucket you control;
- the service endpoint, bucket name, access-key ID, and secret access key; and
- one long encryption passphrase available through private entry on both
  devices.

Do not append the bucket name to the service endpoint. Reinstate does not need
your OpenAI account credentials. Keep all credentials and the encryption
passphrase out of prompts, shell history, screenshots, and issue reports.

The examples use `local/my-project` as the canonical project ID. Choose your
own stable identifier and reuse it exactly on each device.

## 1. Install and check the source device

On macOS, Linux, or WSL2:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

On native Windows PowerShell:

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

Check the binary and local compatibility:

```sh
rein version --json
rein setup check
```

Before configuration exists, `rein setup check` should identify the missing
Reinstate config. Fix platform, keyring, or Codex compatibility errors before
you initialize sync.

## 2. Give the repository a portable identity

Configure the source device with a canonical project ID and its actual absolute
path:

```sh
rein init \
  --project local/my-project=/absolute/path/to/my-project
```

Follow the hidden prompts for the storage configuration and credentials.
Storage keys are placed in the supported operating-system keyring. Reinstate
does not store the encryption passphrase.

Save the printed `profile_id`. This non-secret UUID identifies the remote sync
set and must be reused by every additional device.

Verify the result:

```sh
rein setup check
rein doctor --self-test
rein list --agent codex
```

## 3. Select and push one Codex session

Create or resume a harmless Codex session in the mapped repository. List the
sessions Reinstate can discover:

```sh
rein list --agent codex
```

Copy the exact intended session ID. Preview the operation:

```sh
rein push --agent codex --session SESSION_ID --dry-run
```

If the agent, session, and project are correct, push that session:

```sh
rein push --agent codex --session SESSION_ID
```

Start with one explicit session. The broader `--all` option must be a conscious
human choice, not an automatic selection by an installer or coding agent.

## 4. Join the same profile from the destination

Install Reinstate on the destination, then reuse the source profile UUID and
canonical project ID:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path
```

Enter the same endpoint, bucket, storage credentials, and encryption
passphrase. RC6 requires the encrypted remote manifest for an additional
device; it fails without creating a new local profile if that manifest is
missing.

Different local paths are expected. A checkout might live at
`C:\work\my-project` on Windows and `/Users/you/work/my-project` on macOS.
`local/my-project` connects those paths without making either machine's
absolute path the portable session identity.

Run read-only checks:

```sh
rein setup check
rein doctor --self-test
rein status
rein pull --agent codex --session SESSION_ID --dry-run
```

Do not continue if Codex compatibility is not `SUPPORTED` or the planned
destination project is wrong.

## 5. Pull and resume with Codex CLI

Close Codex before replacing an existing local copy of the selected session.
Then restore and verify discovery:

```sh
rein pull --agent codex --session SESSION_ID
rein list --agent codex
codex resume SESSION_ID
```

The final command is Codex CLI's native resume path. Reinstate prepares the
vendor-native session; it does not execute or replace the Codex agent loop.

## Why Codex working-directory remapping matters

Codex rollouts contain structural session metadata, including the source
working directory. That path is normally different after a repository moves
from Windows to macOS or between two user accounts.

For mapped projects, Reinstate resolves the source working directory to the
configured project ID during discovery. Export replaces that structural root
with a `${REPO:<id>}` token. Restore expands the token through the destination
device's configured path while preserving the native date-partitioned rollout
layout.

Reinstate rewrites allow-listed structural fields. It does not treat
path-looking prose in ordinary conversation text as a filesystem destination.

## Security and restore boundaries

- Session snapshots and manifests are encrypted locally with age passphrase
  encryption before they reach the bucket.
- Object storage receives ciphertext and opaque object keys.
- Storage credentials remain in the operating-system keyring.
- The encryption passphrase is entered through a hidden prompt and is not
  stored.
- Codex auth data, API credentials, `.env` files, caches, and logs are excluded.
- A pull validates before mutation and preserves an existing target through
  backup behavior when applicable.
- Divergent histories are not silently overwritten.

Session transcripts can still contain secrets that were typed or printed
during the conversation. Encryption protects remote storage, but it cannot
undo disclosure inside the transcript or protect a compromised local machine.
Review the [security model](/docs/security-model) before syncing sensitive
work.

## Verification checklist

Treat the handoff as verified only when:

1. Both devices pass `rein setup check`.
2. The source push names one intended Codex session.
3. The destination dry-run resolves the expected repository mapping.
4. The pull finishes without a compatibility, active-session, or conflict
   refusal.
5. `rein list --agent codex` discovers the restored session locally.
6. `codex resume SESSION_ID` opens the expected native Codex context.
7. Verification did not print transcript contents, storage credentials, or the
   passphrase.

Use the repository's
[Phase 1 MacBook and Windows acceptance runbook](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/testing/phase-1-mac-windows-acceptance.md)
when certifying a release rather than evaluating a personal setup.

## If resume fails

Work through these checks:

1. Confirm both devices share the same `profile_id`, bucket, prefix, and
   canonical project ID.
2. Confirm the destination maps that project ID to the real local repository
   path.
3. Check that the installed Codex CLI version is inside the currently supported
   stable range.
4. Run `rein status` and repeat the pull dry-run.
5. Preserve any conflict record or backup until you know which session copy is
   authoritative.
6. Continue with the [troubleshooting documentation](/docs/troubleshooting).
