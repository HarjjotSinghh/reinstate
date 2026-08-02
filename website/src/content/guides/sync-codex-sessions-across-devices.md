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
searchIntent: "agent-specific"
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
estimatedTaskMinutes: 30
prerequisites:
  - "Codex CLI installed on the source and destination devices"
  - "An S3-compatible bucket and its endpoint and credentials"
  - "The same long encryption passphrase available privately on both devices"
howToSteps:
  - name: "Install and check the source device"
    text: "Install the pinned Reinstate release, confirm its version, and run the read-only setup check before creating local configuration."
    anchor: "install-source"
  - name: "Map the source repository"
    text: "Initialize Reinstate with one stable project ID mapped to the source repository's absolute path, then save the non-secret profile UUID."
    anchor: "configure-source"
  - name: "Dry-run and push one Codex session"
    text: "List Codex sessions, select one exact session ID, inspect a non-mutating push plan, and upload only that encrypted native rollout snapshot."
    anchor: "push-session"
  - name: "Join the destination to the profile"
    text: "Install Reinstate on the destination and reuse the source profile UUID, storage settings, passphrase, and project ID with the destination path."
    anchor: "configure-destination"
  - name: "Dry-run, pull, and resume natively"
    text: "Inspect the destination restore plan, close Codex when replacement is required, pull the session, and resume the exact ID with Codex CLI."
    anchor: "pull-and-resume"
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

The current public installer pins `v0.2.0-rc.2`. It remains release-candidate
software while the remaining native platform and physical two-device
acceptance rows are completed. Confirm the supported Codex CLI range on the
[compatibility page](/compatibility) before transferring real work.

## Key points

- Reinstate restores **Codex sessions into Codex CLI**. It does not translate
  rollouts into Claude Code transcripts.
- Use the same non-secret `profile_id` and canonical project ID on both
  devices, while mapping each device's real local path separately.
- Select one `SESSION_ID` and run both push and pull with `--dry-run` before
  either mutating command.
- Snapshots and manifests are encrypted locally; storage credentials stay in
  the OS keyring, and the passphrase is not stored.
- `v0.2.0-rc.2` is not a stable release and this guide is not evidence that the
  outstanding physical two-device acceptance matrix has passed.

## Before you begin

Prepare:

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

### Platform and version qualifications

| Environment | Installer path | Current qualification |
| --- | --- | --- |
| macOS native arm64 | POSIX installer | Current source compatibility evidence covers the documented Codex CLI range. |
| macOS native amd64 | POSIX installer | A Phase 1 release gate remains open; do not infer certification from installer success. |
| Windows 11 native amd64 | PowerShell installer | Covered by the two-device Phase 1 acceptance run, which passed. |
| Linux native | POSIX installer | Installer-compatible; not covered by the two-device acceptance run. |
| WSL2 amd64 | POSIX installer | Installer-compatible and documented for smoke testing; not covered by the two-device acceptance run. WSL1 is unsupported. |

The repository currently records Codex CLI `0.133.0`–`0.146.0` as the tested
stable range. `rein setup check` is authoritative for the installed version:
`UNTESTED` and `UNSUPPORTED` block transfer. These facts describe committed
evidence, not completed physical Reinstate acceptance on every platform.

## Command placeholders and parameters

| Value or flag | Meaning |
| --- | --- |
| `local/my-project` | A stable, non-secret project ID reused on every device. It is not a filesystem path. |
| `/absolute/path/to/my-project` | The source device's real absolute checkout path. Replace it, including on Windows. |
| `DEVICE_A_PROFILE_UUID` | The exact non-secret `profile_id` printed by the first successful `rein init`. |
| `SESSION_ID` | The exact Codex session identifier selected from `rein list --agent codex`. |
| `--project PROJECT_ID=ABSOLUTE_PATH` | Creates one canonical project mapping. The ID is shared; the absolute path is device-specific. |
| `--profile-id UUID` | Joins an additional device to the first device's existing encrypted sync profile. |
| `--agent codex` | Restricts discovery or transfer to the Codex adapter. |
| `--session SESSION_ID` | Restricts transfer to one session. It is safer than selecting every session with `--all`. |
| `--dry-run` | Authenticates, validates, and prints the plan without uploading or restoring the selected session. |
| `--json` | Requests machine-readable output; this guide uses it only for the version check. |

<h2 id="install-source">1. Install and check the source device</h2>

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

Check the binary and local compatibility:

```sh
rein version --json
rein setup check
```

**Expected result:** `rein version --json` returns a JSON object whose version
is `v0.2.0-rc.2` for the currently pinned installer. Before initialization,
`rein setup check` exits with code `3` and reports `config missing`. That one
pre-init failure is expected; a platform, keyring, or Codex compatibility
failure is a separate blocker that must be resolved.

<h2 id="configure-source">2. Give the repository a portable identity</h2>

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

**Expected result:** successful initialization prints that `config.toml` and
`state.json` were created, followed by
`profile_id=<UUID> (use this exact ID on every device)`. Afterward, when
configuration and every installed-agent compatibility check pass,
`rein setup check` and `rein doctor --self-test` exit `0`. The list command
prints each discovered Codex session as agent, session ID, and project ID; an
empty list means there is not yet a discoverable Codex session in the mapped
project.

The `--project` value has the form `PROJECT_ID=ABSOLUTE_PATH`. The left side is
portable and identical on both devices; the right side is local to this
device. Do not put storage credentials or a passphrase in that value.

<h2 id="push-session">3. Select and push one Codex session</h2>

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

**Expected result:** the dry run includes
`would push ... snapshot(s)` and `dry_run=true`; the counts depend on whether
the session is unchanged. The mutating command reports
`pushed ... snapshot(s)` and `dry_run=false`. A successful push does not prove
the other device can resume the session; complete the destination verification
below.

Start with one explicit session. The broader `--all` option must be a conscious
human choice, not an automatic selection by an installer or coding agent.

<h2 id="configure-destination">4. Join the same profile from the destination</h2>

Install Reinstate on the destination, then reuse the source profile UUID and
canonical project ID:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path
```

Enter the same endpoint, bucket, storage credentials, and encryption
passphrase. Reinstate requires the encrypted remote manifest for an additional
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

**Expected result:** initialization verifies and joins the existing encrypted
manifest instead of creating a different sync set. `rein status` reports the
remote revision and session count. The pull preview reports
`would pull ... snapshot(s), dry_run=true` and prints the selected
`codex:SESSION_ID`, destination path, and backup root. The destination must be
derived from this device's mapped repository, not copied from the source.

<h2 id="pull-and-resume">5. Pull and resume with Codex CLI</h2>

Close Codex before replacing an existing local copy of the selected session.
Then restore and verify discovery:

```sh
rein pull --agent codex --session SESSION_ID
rein list --agent codex
codex resume SESSION_ID
```

The final command is Codex CLI's native resume path. Reinstate prepares the
vendor-native session; it does not execute or replace the Codex agent loop.

**Expected result:** the pull reports
`pulled ... snapshot(s), dry_run=false` and the resolved destination. The list
command then includes the exact session ID, and Codex CLI opens that rollout
when given `codex resume SESSION_ID`. Codex's own screen text varies by
version, so the durable success condition is the exact resumed session and
mapped project, not a particular banner.

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

## Failure modes and common errors

### What does `config missing` (exit `3`) mean?

**Meaning:** Local Reinstate configuration does not exist. This is expected
only before first-device initialization.

**Safe next action:** Run `rein init` with the intended project mapping; do not
create config files by hand.

### What should I do after compatibility exit `5` or `UNTESTED`?

**Meaning:** The installed Codex layout or version lacks release evidence.

**Safe next action:** Stop writes, check the
[compatibility matrix](/compatibility), and use a documented supported
version.

### Why does Reinstate report `no matching local sessions found` (exit `2`)?

**Meaning:** The selected ID or adapter is not discoverable on the source.

**Safe next action:** Run `rein list --agent codex` from the mapped project and
copy the exact ID.

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

**Meaning:** Codex may still be writing the target rollout.

**Safe next action:** Close every Codex process and rerun the exact scoped dry
run before the pull.

### Why can pull succeed while native resume misses the session?

**Meaning:** The path mapping, date partition, or destination working
directory is wrong.

**Safe next action:** Confirm the destination mapping, rerun the scoped dry
run, and follow the [troubleshooting guide](/docs/troubleshooting).

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
4. Before replacing an existing rollout, pull plans name the backup root and a
   mutating pull backs up the target when applicable. Close Codex, preserve
   that backup, and do not copy files over the live rollout until you have
   identified the authoritative copy.
5. A conflict is a refusal, not a partial success. Keep the conflict record and
   use `rein conflicts list`, `rein conflicts show <id>`, and an explicit
   resolution only after reviewing which side contains the needed work.

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

## Codex session sync FAQ

### Can Reinstate resume this Codex rollout in Claude Code?

No. Reinstate Phase 1 restores a Codex rollout for native Codex CLI resume.
Cross-agent work requires an explicit portable handoff in a later phase;
Reinstate does not silently translate transcripts.

### Do both computers need the same repository path?

No. Both devices reuse the same canonical project ID, while each maps that ID
to its own absolute checkout path. Reinstate expands the portable project token
to the destination path while preserving Codex's native date partition.

### Does the destination need OpenAI credentials from the source?

No. Authenticate Codex normally on the destination. Reinstate neither requests
nor syncs OpenAI credentials, Codex auth data, OS keyring contents, or API
keys.

### Is Linux or WSL2 a certified Phase 1 resume target?

No. The POSIX installer works on Linux and WSL2, and WSL2 has a documented
smoke-test path, but neither is a certified Phase 1 agent-resume target in the
current committed evidence. WSL1 is unsupported.

### Does a successful personal transfer mea Reinstate passed release acceptance?

No. It proves only the devices, versions, storage, and session you tested.
Release qualification requires the committed acceptance runbook and recorded,
sanitized evidence for every required matrix row.
