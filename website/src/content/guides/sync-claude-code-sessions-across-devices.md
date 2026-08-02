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
searchIntent: "agent-specific"
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
estimatedTaskMinutes: 30
prerequisites:
  - "Claude Code installed on the source and destination devices"
  - "An S3-compatible bucket and its endpoint and credentials"
  - "The same long encryption passphrase available privately on both devices"
howToSteps:
  - name: "Install and check the source device"
    text: "Install the pinned Reinstate release, confirm its version, and run the read-only setup check before creating local configuration."
    anchor: "install-source"
  - name: "Map the source repository"
    text: "Initialize Reinstate with one stable project ID mapped to the source repository's absolute path, then save the non-secret profile UUID."
    anchor: "configure-source"
  - name: "Dry-run and push one Claude session"
    text: "List Claude Code sessions, select one exact session ID, inspect a non-mutating push plan, and upload only that encrypted session snapshot."
    anchor: "push-session"
  - name: "Join the destination to the profile"
    text: "Install Reinstate on the destination and reuse the source profile UUID, storage settings, passphrase, and project ID with the destination path."
    anchor: "configure-destination"
  - name: "Dry-run, pull, and resume natively"
    text: "Inspect the destination restore plan, close Claude Code when replacement is required, pull the session, and resume the exact ID with Claude Code."
    anchor: "pull-and-resume"
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

The current public installer pins `v0.2.0-rc.3`. It is release-candidate
software while the remaining native platform and two-device acceptance rows
are completed. Check the [compatibility page](/compatibility) before using a
newer Claude Code version.

## Key points

- Reinstate restores **Claude Code sessions into Claude Code**. It does not
  translate transcripts into Codex.
- Use the same non-secret `profile_id` and canonical project ID on both
  devices, while mapping each device's real local path separately.
- Select one `SESSION_ID` and run both push and pull with `--dry-run` before
  either mutating command.
- Snapshots and manifests are encrypted locally; storage credentials stay in
  the OS keyring, and the passphrase is not stored.
- `v0.2.0-rc.3` is not a stable release and this guide is not evidence that the
  outstanding physical two-device acceptance matrix has passed.

## Before you begin

You need:

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

### Platform and version qualifications

| Environment | Installer path | Current qualification |
| --- | --- | --- |
| macOS native arm64 | POSIX installer | Current source compatibility evidence covers the documented Claude Code range. |
| macOS native amd64 | POSIX installer | A Phase 1 release gate remains open; do not infer certification from installer success. |
| Windows 11 native amd64 | PowerShell installer | Covered by the two-device Phase 1 acceptance run, which passed. |
| Linux native | POSIX installer | Installer-compatible; not covered by the two-device acceptance run. |
| WSL2 amd64 | POSIX installer | Installer-compatible and documented for smoke testing; not covered by the two-device acceptance run. WSL1 is unsupported. |

The repository currently records Claude Code `2.1.219`–`2.1.220` as the tested
stable range. `rein setup check` is authoritative for the installed version:
`UNTESTED` and `UNSUPPORTED` block transfer. These facts describe committed
evidence, not completed physical Reinstate acceptance on every platform.

## Command placeholders and parameters

| Value or flag | Meaning |
| --- | --- |
| `local/my-project` | A stable, non-secret project ID reused on every device. It is not a filesystem path. |
| `/absolute/path/to/my-project` | The source device's real absolute checkout path. Replace it, including on Windows. |
| `DEVICE_A_PROFILE_UUID` | The exact non-secret `profile_id` printed by the first successful `rein init`. |
| `SESSION_ID` | The exact Claude session identifier selected from `rein list --agent claude`. |
| `--project PROJECT_ID=ABSOLUTE_PATH` | Creates one canonical project mapping. The ID is shared; the absolute path is device-specific. |
| `--profile-id UUID` | Joins an additional device to the first device's existing encrypted sync profile. |
| `--agent claude` | Restricts discovery or transfer to the Claude Code adapter. |
| `--session SESSION_ID` | Restricts transfer to one session. It is safer than selecting every session with `--all`. |
| `--dry-run` | Authenticates, validates, and prints the plan without uploading or restoring the selected session. |
| `--json` | Requests machine-readable output; this guide uses it only for the version check. |

<h2 id="install-source">1. Install and verify Reinstate on the source</h2>

On macOS, Linux, or WSL2:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

On native Windows PowerShell:

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

If your policy prohibits executing a downloaded script directly, use the
download-and-inspect variants in the
[getting-started documentation](/docs/getting-started) before running it.

Verify the installed binary and the local environment:

```sh
rein version --json
rein setup check
```

**Expected result:** `rein version --json` returns a JSON object whose version
is `v0.2.0-rc.3` for the currently pinned installer. Before initialization,
`rein setup check` exits with code `3` and reports `config missing`. That one
pre-init failure is expected; a platform, keyring, or Claude Code compatibility
failure is a separate blocker that must be resolved.

<h2 id="configure-source">2. Configure the source project path</h2>

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

**Expected result:** successful initialization prints that `config.toml` and
`state.json` were created, followed by
`profile_id=<UUID> (use this exact ID on every device)`. Afterward, when
configuration and every installed-agent compatibility check pass,
`rein setup check` and `rein doctor --self-test` exit `0`. The list command
prints each discovered Claude session as agent, session ID, and project ID; an
empty list means there is not yet a discoverable Claude session in the mapped
project.

The `--project` value has the form `PROJECT_ID=ABSOLUTE_PATH`. The left side is
portable and identical on both devices; the right side is local to this
device. Do not put storage credentials or a passphrase in that value.

<h2 id="push-session">3. Push one Claude Code session</h2>

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

**Expected result:** the dry run includes
`would push ... snapshot(s)` and `dry_run=true`; the counts depend on whether
the session is unchanged. The mutating command reports
`pushed ... snapshot(s)` and `dry_run=false`. A successful push does not prove
the other device can resume the session; complete the destination verification
below.

Use an explicit session ID for the first transfer. `--all` exists, but neither
Reinstate nor a setup agent should select every session without your deliberate
choice.

<h2 id="configure-destination">4. Configure the destination with the same identity</h2>

Install Reinstate on the second device, then map the same project ID to that
device's actual repository path:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path
```

Enter the same endpoint, bucket, storage credentials, and encryption
passphrase. Reinstate verifies that the existing encrypted remote manifest belongs to
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

**Expected result:** initialization verifies and joins the existing encrypted
manifest instead of creating a different sync set. `rein status` reports the
remote revision and session count. The pull preview reports
`would pull ... snapshot(s), dry_run=true` and prints the selected
`claude:SESSION_ID`, destination path, and backup root. The destination must be
derived from this device's mapped repository, not copied from the source.

<h2 id="pull-and-resume">5. Pull and resume in Claude Code</h2>

Close Claude Code before a pull that will replace an existing local copy of the
same session. Then restore:

```sh
rein pull --agent claude --session SESSION_ID
rein list --agent claude
claude --resume SESSION_ID
```

Reinstate checks the planned Claude project directory after restore. Finding the same
session ID somewhere else under Claude Code's project storage is not accepted
as successful restoration.

**Expected result:** the pull reports
`pulled ... snapshot(s), dry_run=false` and the resolved destination. The list
command then includes the exact session ID, and Claude Code opens that session
when given `claude --resume SESSION_ID`. Claude Code's own screen text varies
by version, so the durable success condition is the exact resumed session and
mapped project, not a particular banner.

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

## Failure modes and common errors

### What does `config missing` (exit `3`) mean?

**Meaning:** Local Reinstate configuration does not exist. This is expected
only before first-device initialization.

**Safe next action:** Run `rein init` with the intended project mapping; do not
create config files by hand.

### What should I do after compatibility exit `5` or `UNTESTED`?

**Meaning:** The installed Claude layout or version lacks release evidence.

**Safe next action:** Stop writes, check the
[compatibility matrix](/compatibility), and use a documented supported
version.

### Why does Reinstate report `no matching local sessions found` (exit `2`)?

**Meaning:** The selected ID or adapter is not discoverable on the source.

**Safe next action:** Run `rein list --agent claude` from the mapped project
and copy the exact ID.

### Why does the destination report `remote profile manifest not found` (exit `4`)?

**Meaning:** The destination profile, bucket, prefix, or endpoint does not
identify the first device's manifest.

**Safe next action:** Recheck the copied `profile_id`, bucket, prefix, and
service endpoint. Do not create an empty manifest.

### What should I do after a wrong-passphrase or authentication failure (exit `4`)?

**Meaning:** Reinstate cannot decrypt or authenticate the remote state.

**Safe next action:** Wait for the hidden prompt and retry with the original
passphrase; never put it in a flag, environment value, or chat.

### Why does pull report `remote session not found` (exit `2`)?

**Meaning:** The selected session was not pushed to this remote profile.

**Safe next action:** Confirm `rein status`, the profile identity, and the
exact source push result.

### What should I do after conflict exit `6`?

**Meaning:** Local and remote histories diverged.

**Safe next action:** Preserve both histories and the conflict record; inspect
them with `rein conflicts` before choosing a resolution.

### What should I do after safety exit `7` while pulling?

**Meaning:** Claude Code may still be writing the target session.

**Safe next action:** Close every Claude Code process and rerun the exact
scoped dry run before the pull.

### Why can pull succeed while native resume misses the session?

**Meaning:** The path mapping or destination project key is wrong.

**Safe next action:** Confirm the destination `local_root`, rerun the scoped
dry run, and follow the [troubleshooting guide](/docs/troubleshooting).

## Safe rollback and undo

Reinstate does not provide a general `rein undo` or per-session remote
delete command. Use these recovery boundaries instead:

1. Prefer `--dry-run`: it does not upload or restore the selected session, so
   there is nothing to undo.
2. If initialization used the wrong values, stop before a push. A later
   `rein init --force` backs up existing initialization state before
   replacement, but review `rein init --help` and re-enter secrets privately.
3. A push creates encrypted remote state. Do not delete `manifest.age` or an
   individual storage object manually; that can leave the profile
   inconsistent. Stop syncing and preserve the bucket while you determine the
   correct cleanup procedure.
4. Before replacing an existing session, pull plans name the backup root and a
   mutating pull backs up the target when applicable. Close Claude Code,
   preserve that backup, and do not copy files over the live session until you
   have identified the authoritative copy.
5. A conflict is a refusal, not a partial success. Keep the conflict record and
   use `rein conflicts list`, `rein conflicts show <id>`, and an explicit
   resolution only after reviewing which side contains the needed work.

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

## Claude Code sync FAQ

### Can Reinstate resume this Claude session in Codex?

No. Reinstate Phase 1 restores a Claude Code session for native Claude Code
resume. Cross-agent work requires an explicit portable handoff in a later
phase; Reinstate does not silently translate transcripts.

### Do both computers need the same repository path?

No. Both devices reuse the same canonical project ID, while each maps that ID
to its own absolute checkout path. That mapping is the mechanism that supports
different Windows and macOS paths.

### Does the destination need Anthropic credentials from the source?

No. Authenticate Claude Code normally on the destination. Reinstate neither
requests nor syncs Anthropic credentials, auth files, OS keyring contents, or
OAuth material.

### Is Linux or WSL2 a certified Phase 1 resume target?

No. The POSIX installer works on Linux and WSL2, and WSL2 has a documented
smoke-test path, but neither is a certified Phase 1 agent-resume target in the
current committed evidence. WSL1 is unsupported.

### Does a successful personal transfer mea Reinstate passed release acceptance?

No. It proves only the devices, versions, storage, and session you tested.
Release qualification requires the committed acceptance runbook and recorded,
sanitized evidence for every required matrix row.
