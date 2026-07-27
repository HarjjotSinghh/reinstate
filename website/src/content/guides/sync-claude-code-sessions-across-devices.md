---
title: "How to Sync Claude Code Sessions Across Devices"
description: "Set up encrypted Claude Code session sync between two computers with Reinstate, including project path mapping, dry runs, restore, and native resume."
answer: "To sync a Claude Code session across devices, install Reinstate on both computers, map the same canonical project ID to each local repository path, push one explicit Claude session from the source, dry-run and pull it on the destination, then resume it with Claude Code."
author: "Harjot Singh Rana"
publishedAt: 2026-07-27
updatedAt: 2026-07-27
reviewedAt: 2026-07-27
tags: ["Claude Code", "session sync", "multi-device", "path mapping", "end-to-end encryption"]
targetQuery: "how to sync Claude Code sessions across devices"
searchIntent: "how-to"
related:
  - title: "Claude Code integration details"
    path: "/integrations/claude-code"
  - title: "Getting started documentation"
    path: "/docs/getting-started"
  - title: "Compatibility and tested versions"
    path: "/compatibility"
  - title: "Security model"
    path: "/docs/security-model"
draft: false
noindex: false
agent: "claude-code"
difficulty: "intermediate"
estimatedMinutes: 12
prerequisites:
  - "Claude Code installed on the source and destination devices"
  - "An S3-compatible bucket and its endpoint and credentials"
  - "The same long encryption passphrase available privately on both devices"
---

## What this workflow does

This workflow moves one supported Claude Code session through storage you
control. Reinstate discovers the vendor-native session, rewrites only known
structural paths into portable project tokens, encrypts the snapshot locally,
and uploads ciphertext. On the second device it validates, decrypts, remaps,
backs up when necessary, and restores the session into Claude Code's native
project layout.

This is **Claude Code to Claude Code** continuity. Reinstate Phase 1 does not
translate a Claude transcript into Codex or any other agent format.

The current public installer pins `v0.1.0-rc.6`. It is release-candidate
software while the remaining native platform and two-device acceptance rows
are completed. Check the [compatibility page](/compatibility) before using a
newer Claude Code version.

## Before you begin

You need:

- macOS, native 64-bit Windows, Linux, or WSL2 on each device;
- Claude Code installed on both devices;
- a Cloudflare R2, Amazon S3, or compatible bucket you control;
- the service endpoint, bucket name, access-key ID, and secret access key; and
- one long encryption passphrase you can enter through a hidden prompt on both
  devices.

Keep the bucket name separate from the service endpoint. Reinstate does not
need your Anthropic credentials, and those credentials must never be pasted
into a coding-agent conversation.

Choose one stable project ID before setup. The examples use
`local/my-project`; replace it with a value you will reuse exactly on every
device.

## 1. Install and verify Reinstate on the source

On macOS, Linux, or WSL2:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

On native Windows PowerShell:

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

Verify the installed binary and the local environment:

```sh
rein version --json
rein setup check
```

Before initialization, `rein setup check` should report that Reinstate
configuration is missing. Resolve any platform, keyring, or Claude Code
compatibility failure before continuing.

## 2. Configure the source project path

Map the canonical project ID to the source device's real absolute repository
path:

```sh
rein init \
  --project local/my-project=/absolute/path/to/my-project
```

`rein init` prompts for the storage endpoint, bucket, credentials, and related
settings. Credential input is hidden and stored through the operating system
keyring. The encryption passphrase is not stored.

Save the printed `profile_id`. It is not a secret. The second device must reuse
that exact UUID to join the same encrypted sync set.

Validate the configuration:

```sh
rein setup check
rein doctor --self-test
rein list --agent claude
```

## 3. Push one Claude Code session

Create or resume a harmless Claude Code session in the mapped repository, then
list the discoverable sessions:

```sh
rein list --agent claude
```

Copy the intended session ID. Preview the upload before mutating remote state:

```sh
rein push --agent claude --session SESSION_ID --dry-run
```

Review the selected agent, session, project, and action. If they are correct,
push that one session:

```sh
rein push --agent claude --session SESSION_ID
```

Use an explicit session ID for the first transfer. `--all` exists, but neither
Reinstate nor a setup agent should select every session without your deliberate
choice.

## 4. Configure the destination with the same identity

Install Reinstate on the second device, then map the same project ID to that
device's actual repository path:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path
```

Enter the same endpoint, bucket, storage credentials, and encryption
passphrase. RC6 verifies that the existing encrypted remote manifest belongs to
the supplied profile before it saves local configuration.

The paths may be different. For example, the project might be
`C:\src\my-project` on Windows and `/Users/you/src/my-project` on macOS. The
shared `local/my-project` identifier is the portable identity Reinstate uses to
compute the correct destination layout.

Validate without restoring:

```sh
rein setup check
rein doctor --self-test
rein status
rein pull --agent claude --session SESSION_ID --dry-run
```

Stop if compatibility is `UNTESTED` or `UNSUPPORTED`, the session is not
associated with the expected project, or the dry-run destination is wrong.

## 5. Pull and resume in Claude Code

Close Claude Code before a pull that will replace an existing local copy of the
same session. Then restore:

```sh
rein pull --agent claude --session SESSION_ID
rein list --agent claude
claude --resume SESSION_ID
```

RC6 checks the planned Claude project directory after restore. Finding the same
session ID somewhere else under Claude Code's project storage is not accepted
as successful restoration.

## How path remapping preserves native resume

Claude Code groups sessions beneath a project directory derived from the
device's absolute project path. A raw copy from one operating system can land
under a directory key that does not match the destination checkout.

Reinstate associates the source session with the configured project ID. During
restore it resolves that ID through the destination device's `local_root` and
recomputes Claude Code's destination project key. This preserves same-vendor
native resume without rewriting path-like text in ordinary conversation prose.

## Security checks for this transfer

- Remote manifests and snapshots are encrypted before upload.
- Storage credentials stay in the OS keyring; they are not session content.
- The encryption passphrase is entered through a hidden prompt and is not
  stored by Reinstate.
- Auth files, OAuth material, credential files, caches, and logs are excluded.
- Pull validates before mutation and backs up an existing target when needed.
- Divergent session histories create conflict records instead of a silent
  overwrite.

A coding-agent transcript can contain a secret that a person or tool printed
into the conversation. Encryption protects the uploaded artifact from the
storage provider, but it does not make a leaked production credential safe.
Keep secrets out of agent chats and read the full
[security model](/docs/security-model).

## Verification checklist

The transfer is complete only when all of these are true:

1. `rein setup check` succeeds on both devices.
2. The source push targets one intended Claude Code session.
3. The destination pull dry-run shows the expected local project.
4. The mutating pull completes without a compatibility or conflict refusal.
5. `rein list --agent claude` finds the restored session on the destination.
6. `claude --resume SESSION_ID` opens the same Claude Code context.
7. No transcript text, credential, or passphrase was printed merely to prove
   success.

For release qualification rather than personal evaluation, use the full
[MacBook and Windows Phase 1 acceptance runbook](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/testing/phase-1-mac-windows-acceptance.md).

## If the session does not resume

Check the following in order:

1. Confirm both devices use the same `profile_id`, bucket, prefix, and canonical
   project ID.
2. Confirm each device maps that project ID to its own absolute local path.
3. Run `rein setup check` and resolve any Claude Code compatibility block.
4. Run `rein status` and the pull dry-run again.
5. Do not bypass a conflict or overwrite refusal until you understand which
   copy contains the work you need.
6. Follow the [troubleshooting guide](/docs/troubleshooting) and preserve the
   generated backups.
