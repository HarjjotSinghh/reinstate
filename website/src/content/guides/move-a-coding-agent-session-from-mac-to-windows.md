---
title: "Move a Coding-Agent Session from Mac to Windows"
description: "Move one encrypted Claude Code or Codex session from macOS to native Windows with Reinstate path remapping, dry-runs, backups, and native resume."
answer: "To move a coding-agent session from Mac to Windows with Reinstate, map one canonical project ID to the Mac and Windows checkout paths, push one selected Claude Code or Codex session from the Mac, join the same encrypted storage profile on native Windows, preview the pull, restore it, and resume it with the same vendor."
author: "Harjot Singh Rana"
publishedAt: 2026-07-27
updatedAt: 2026-07-27
reviewedAt: 2026-07-27
tags: ["macOS", "Windows", "path remapping", "session sync", "Claude Code", "Codex CLI"]
targetQuery: "move coding agent session from Mac to Windows"
searchIntent: "platform-specific"
related:
  - title: "Claude Code integration"
    path: "/integrations/claude-code"
  - title: "Codex CLI integration"
    path: "/integrations/codex"
  - title: "Desktop and laptop continuity"
    path: "/use-cases/desktop-and-laptop"
  - title: "macOS and Windows use case"
    path: "/use-cases/macos-and-windows"
  - title: "Compatibility and tested versions"
    path: "/compatibility"
  - title: "Security model"
    path: "/docs/security-model"
draft: false
noindex: false
agent: "general"
difficulty: "intermediate"
estimatedMinutes: 17
estimatedTaskMinutes: 45
prerequisites:
  - "One native macOS computer and one native 64-bit Windows 11 computer you are authorized to configure"
  - "Reinstate v0.2.0-rc.2 plus a tested Claude Code or Codex CLI version on both computers"
  - "The same Git project checked out normally on both computers, even if its absolute paths differ"
  - "A private S3-compatible profile, its non-secret coordinates, and credentials for both OS keyrings"
  - "One harmless source session ID and a long encryption passphrase stored outside Reinstate"
howToSteps:
  - name: "Verify the two native environments"
    text: "Confirm the Reinstate binary, same-vendor agent version, native operating system, and separate absolute project path on each computer before configuring synchronization."
    anchor: "verify-native-environments"
  - name: "Initialize and verify the Mac profile"
    text: "Initialize the first device with one canonical project ID mapped to the Mac checkout, enter storage credentials privately, record its profile UUID, and run setup and self-tests."
    anchor: "initialize-mac-profile"
  - name: "Preview and push one selected session"
    text: "List metadata, choose exactly one harmless Claude Code or Codex session, inspect the non-mutating push plan, push it, and verify the encrypted remote revision."
    anchor: "push-selected-session"
  - name: "Join the profile on native Windows"
    text: "Initialize native Windows with the first device profile UUID and the same project ID mapped to the Windows checkout, then verify storage, crypto, and adapter checks."
    anchor: "join-windows-profile"
  - name: "Preview, pull, and resume on Windows"
    text: "Close the destination agent, preview the exact restore and backup paths, pull the selected session, verify its native destination, and resume it through the same vendor CLI."
    anchor: "pull-and-resume"
---

## What this guide proves—and what it does not

This is an operator workflow for moving one supported, vendor-native session
from macOS to **native Windows 11** through Reinstate `v0.2.0-rc.2`. It uses
the same commands and stop conditions as the repository's
[Phase 1 Mac/Windows acceptance runbook](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/testing/phase-1-mac-windows-acceptance.md),
but completing it on personal devices is not release certification.

Reinstate transfers an age-encrypted snapshot and known structural project
paths. Git remains responsible for source code and working-tree state. Claude
Code or Codex CLI remains responsible for native resume.

| Concern | macOS source | Reinstate | Windows destination |
| --- | --- | --- | --- |
| Repository checkout | Existing Mac path | Maps it to a canonical project ID | Existing Windows path |
| Session selection | Human chooses one ID | Discovers and snapshots that ID | Restores the same vendor ID |
| Encryption | Passphrase entered locally | Encrypts before upload | Decrypts after download |
| Credentials | Stored in macOS Keychain | Stores only a credential reference in config | Stored in Windows Credential Manager |
| Resume | Source vendor writes the session | Does not execute the agent | Same vendor resumes natively |

## Key points

- Use one stable project ID on both devices even though paths such as
  `/Users/me/Code/app` and `C:\src\app` differ.
- Use **native Windows PowerShell** for this workflow. WSL2 is a separate Linux
  Reinstate device with different path semantics; WSL1 is unsupported.
- Choose exactly one `claude` or `codex` session and preview both transfer
  directions with `--dry-run`.
- Resume remains **same-vendor**: Claude Code to Claude Code, or Codex CLI to
  Codex CLI. Reinstate does not convert one vendor's transcript into the other
  vendor's native format.
- Credentials, vendor authentication, API keys, OAuth material, `.env` files,
  Git state, MCP servers, skills, plugins, and hooks do not move with the
  session.
- Keep independent backups. The encrypted remote profile is a continuity
  artifact, not a replacement for repository backups or a tested retention
  plan.
- macOS amd64 and WSL2 acceptance still
  contain open release gates. Do not describe a personal success as stable
  Phase 1 certification.

## Before you begin

Use a harmless session in a disposable or non-sensitive repository first.
Confirm that moving its contents to the selected bucket and Windows computer
is permitted by employer, client, data-residency, and device policies.

Configure a private provider bucket before this guide. Use either the
[Amazon S3 storage guide](/guides/use-s3-for-coding-agent-session-storage) or
the
[Cloudflare R2 storage guide](/guides/use-cloudflare-r2-for-coding-agent-session-storage).
Never paste the storage secret, encryption passphrase, transcript, or
downloaded snapshot into a coding-agent prompt, shell argument, commit,
screenshot, or test report.

### Platform and release qualifications

| Environment | Current qualification |
| --- | --- |
| macOS native arm64 | Current source compatibility evidence covers the documented Claude Code and Codex ranges; a real second device is still required for transfer evidence. |
| macOS native amd64 | A required stable-release row that remains open. |
| Windows 11 native amd64 | A primary Phase 1 target whose native installer, restore, and resume acceptance gate remains open until recorded evidence passes. |
| WSL2 amd64 | Installer-compatible and documented for smoke testing, but it is not native Windows and its Phase 1 gate remains open. |
| Linux native | Installer-compatible; not covered by the two-device acceptance run. |
| WSL1 | Unsupported. |

Current Reinstate source evidence recognizes Claude Code `2.1.219`–`2.1.220` and
Codex CLI `0.133.0`–`0.146.0`. A result outside those inclusive ranges is
`UNTESTED`; `rein setup check` exits with compatibility code `5` and mutating
push or pull is blocked.

### Native Windows and WSL are different devices

Do not point native Windows and WSL at one shared agent-state directory or one
Reinstate home:

| Property | Native Windows | WSL2 |
| --- | --- | --- |
| Shell | PowerShell | Linux shell |
| Default Reinstate home | `%USERPROFILE%\.reinstate` | `~/.reinstate` inside WSL |
| Paths | `C:\src\app` | `/home/me/app` or `/mnt/c/src/app` |
| Phase 1 role in this guide | Destination | Out of scope |

If the agent runs inside WSL, follow a separately configured Linux/WSL
workflow; do not label it native Windows evidence.

## Command placeholders and parameters

| Placeholder or flag | Meaning |
| --- | --- |
| `AGENT` | Replace with exactly `claude` or `codex`, and use the same value on both devices. |
| `SESSION_ID` | The exact harmless source ID returned by `rein list --agent AGENT`. |
| `PROFILE_ID` | The non-secret UUID printed by first-device `rein init`; copy it exactly to Windows. |
| `PROJECT_ID` | A stable ID such as `github.com/acme/app`, identical on both devices. |
| `MAC_PROJECT` | The absolute macOS checkout path, such as `/Users/me/Code/app`. |
| `$WindowsProject` | A PowerShell variable containing the absolute native Windows checkout path. |
| `STORAGE_ENDPOINT` | The S3-compatible service endpoint, without the bucket name. |
| `STORAGE_REGION` | The provider signing Region; use `auto` for Cloudflare R2. |
| `STORAGE_BUCKET` | The private bucket name only. |
| `--profile-id PROFILE_ID` | Joins the existing encrypted remote profile instead of generating an unrelated one. |
| `--session SESSION_ID` | Limits transfer to the one selected session. |
| `--dry-run` | Reads, authenticates, validates, and plans without writing a remote snapshot or local session. |

<h2 id="verify-native-environments">1. Verify the two native environments</h2>

Install Reinstate on the Mac:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
rein version --json
sw_vers
uname -m
```

Install Reinstate from **native Windows PowerShell**, not a WSL shell:

```powershell
irm https://reinstate.dev/install.ps1 | iex
rein version --json
Get-ComputerInfo |
  Select-Object WindowsProductName, WindowsVersion, OsBuildNumber
[Environment]::Is64BitOperatingSystem
```

Run `claude --version` or `codex --version` on both computers for the one
vendor being moved. Clone or update the repository through Git, then record
the two absolute checkout paths. Ensure the Windows path is a native path and
not a WSL mount.

Do not transfer a session while either checkout is on an unexpected branch,
commit, or unreviewed dirty working tree. Reinstate maps structural paths; it does
not synchronize Git state or prove environment equivalence.

**Expected result:** both `rein version --json` commands report
`0.1.0`, the Mac and 64-bit native Windows environments are identified
separately, the same agent vendor is installed in a recognized range, and the
same repository exists at two recorded absolute paths.

<h2 id="initialize-mac-profile">2. Initialize and verify the Mac profile</h2>

Replace every placeholder, then run this from a private Mac terminal:

```sh
rein init \
  --endpoint STORAGE_ENDPOINT \
  --region STORAGE_REGION \
  --bucket STORAGE_BUCKET \
  --project "PROJECT_ID=MAC_PROJECT"
```

Enter the storage access-key ID and secret access key only through the two
hidden prompts. Initialization does not request or store the encryption
passphrase. Reinstate creates a profile UUID, writes and deletes a two-byte storage
probe, stores the credential in the OS keyring, and saves `config.toml` plus
`state.json` only after the probe succeeds.

Record only the printed non-secret `profile_id`, then run:

```sh
rein setup check
rein doctor --self-test
```

Stop if the selected adapter is not `SUPPORTED`, the self-test fails, or the
project mapping does not resolve to the intended Mac checkout. Do not use an
unsafe compatibility override; Reinstate has none.

**Expected result:** init prints
`initialized reinstate home (config.toml + state.json)` and
`profile_id=<UUID>`, setup and self-test exit `0`, and the temporary storage
probe is gone. No persistent `manifest.age` exists until the first successful
push.

<h2 id="push-selected-session">3. Preview and push one selected session</h2>

Close the Mac agent cleanly and choose one harmless session:

```sh
rein list --agent AGENT
rein push --agent AGENT --session SESSION_ID --dry-run
rein push --agent AGENT --session SESSION_ID
rein status
```

Enter the long encryption passphrase only at Reinstate's hidden prompt and
store it in a password manager. You will need the exact same passphrase on
Windows. Do not use `--all` for the first transfer.

The dry-run must identify one snapshot without mutation. The real push
encrypts the snapshot locally, writes the encrypted snapshot and manifest
beneath `profiles/PROFILE_ID/`, and updates local synchronization state.

**Expected result:** dry-run begins with
`would push 1 snapshot(s)` and `dry_run=true`; the real command reports
`pushed 1 snapshot(s)` and `dry_run=false`; status reports one remote
revision; and the private bucket contains `manifest.age` plus an opaque
`snapshots/<id>.age`, not a plaintext transcript or credential file.

<h2 id="join-windows-profile">4. Join the profile on native Windows</h2>

In native Windows PowerShell, set a local variable to the real checkout:

```powershell
$WindowsProject = "C:\absolute\path\to\project"
```

Initialize the second device with the exact first-device profile UUID:

```powershell
rein init `
  --profile-id PROFILE_ID `
  --endpoint STORAGE_ENDPOINT `
  --region STORAGE_REGION `
  --bucket STORAGE_BUCKET `
  --project "PROJECT_ID=$WindowsProject"
```

Enter the same storage credential through hidden prompts. Additional-device
init requires the existing encrypted `manifest.age` before it saves local
configuration; a wrong profile ID, endpoint, Region, bucket, prefix, or
credential fails closed.

Then validate the native destination:

```powershell
rein setup check
rein doctor --self-test
rein status
```

`rein status` asks for the encryption passphrase. A deliberately wrong phrase
must produce a decryption/authentication failure without restoring a session,
changing an agent file, or creating a backup. Retry with the correct phrase.

**Expected result:** second-device init reuses `PROFILE_ID`, setup and
self-test exit `0`, status with the correct passphrase reports the selected
remote session, and the project mapping resolves `PROJECT_ID` to the intended
native Windows path without creating any agent session file yet.

<h2 id="pull-and-resume">5. Preview, pull, and resume on Windows</h2>

Close every process for the selected agent. Preview the exact destination and
backup root:

```powershell
rein pull --agent AGENT --session SESSION_ID --dry-run
rein pull --agent AGENT --session SESSION_ID
rein list --agent AGENT
Get-ChildItem -Recurse `
  (Join-Path $env:USERPROFILE ".reinstate\backups") |
  Select-Object -First 20 FullName
```

The dry-run should say `would pull`, name the native vendor destination, and
show the backup root without mutation. The real pull validates and decrypts
the complete artifact before writing, backs up an existing destination when
one exists, writes through a private temporary sibling, and atomically
renames the restored file.

Resume through the same native vendor:

```powershell
claude --resume SESSION_ID
```

or:

```powershell
codex resume SESSION_ID
```

Verify visually that this is the intended prior session and that it opens from
the mapped Windows project. Do not copy transcript content into public
evidence.

**Expected result:** dry-run reports `would pull 1 snapshot(s)` with
`dry_run=true`; the real command reports `pulled 1 snapshot(s)` with
`dry_run=false`; the exact ID is discoverable at its native Windows path; an
existing target has a timestamped backup; and the same-vendor CLI resumes it
without a structural path error.

## How path remapping works

The two local paths map to one portable identity:

| Device | Local configuration | Portable structural form |
| --- | --- | --- |
| Mac | `PROJECT_ID=/Users/me/Code/app` | `${REPO:PROJECT_ID}/...` |
| Windows | `PROJECT_ID=C:\src\app` | `${REPO:PROJECT_ID}/...` |

The Claude Code adapter recomputes the destination project-directory key from
the Windows root. The Codex adapter rewrites the known structural
`session_meta.cwd` field and preserves Codex's native date-partitioned rollout
layout. Reinstate does not blindly replace path-shaped text inside prompts,
responses, shell output, or unknown vendor fields.

Path remapping also does not install runtimes, recreate environment variables,
move uncommitted code, align Git branches, copy agent authentication, or
synchronize MCP servers and skills.

## Visible evidence to retain

Record a private, redacted result containing:

- OS versions and architectures;
- Reinstate and agent versions;
- the non-secret profile UUID and session ID;
- canonical project ID plus redacted source and destination paths;
- command exit codes and `would push`, `pushed`, `would pull`, and `pulled`
  counts;
- remote object names and a boolean that selected plaintext markers were
  absent from a privately downloaded `.age` object;
- whether an existing target received a timestamped backup; and
- whether the vendor-native resume opened the intended session.

Never retain the passphrase, storage secret, transcript text, downloaded
snapshot, or unredacted sensitive paths as evidence.

## Failure modes and common errors

### What does pre-init `config missing` (exit `3`) mean?

**Meaning:** The Reinstate home is intentionally unconfigured.

**Safe recovery:** Run the reviewed init command; do not treat this expected
pre-init failure as a pass.

### What should I do after compatibility exit `5` or `UNTESTED`?

**Meaning:** The agent version or layout lacks release evidence.

**Safe recovery:** Stop mutation and use the
[compatibility page](/compatibility) to choose a recognized version.

### Why does Windows report `remote profile manifest not found`?

**Meaning:** Windows used the wrong profile or storage coordinates, or the Mac
has not pushed yet.

**Safe recovery:** Confirm the Mac push, then copy the exact profile UUID,
endpoint, Region, bucket, and prefix. Never create a substitute empty profile.

### What should I do after a decryption or authentication failure?

**Meaning:** The passphrase is wrong or ciphertext is damaged.

**Safe recovery:** Stop, verify the passphrase privately, preserve remote and
local copies, and retry without changing session files.

### Why does pull report `remote session not found` (exit `2`)?

**Meaning:** `AGENT` or `SESSION_ID` does not match the selected remote entry.

**Safe recovery:** Recheck Mac metadata and `rein status`; do not broaden the
pull to `--all`.

### What should I do after safety exit `7`?

**Meaning:** The destination agent appears active and an existing target could
be overwritten.

**Safe recovery:** Close the same-vendor agent completely and rerun the
dry-run; do not bypass the refusal.

### What should I do after conflict exit `6`?

**Meaning:** The Windows-local and remote copies diverged.

**Safe recovery:** Preserve both, then inspect `rein conflicts list` and
`rein conflicts show CONFLICT_ID`; use `--keep-both` when neither branch may
be lost.

### Why does resume open from the wrong project?

**Meaning:** The canonical ID or Windows mapping is wrong.

**Safe recovery:** Stop the agent, preserve the restored file and backup,
correct configuration through a reviewed re-init, and pull again only after
previewing the destination.

## Safe rollback and undo

Reinstate does not provide a general `rein undo` or remote-session delete
command.

1. A push or pull `--dry-run` mutates neither remote session state nor local
   vendor files.
2. A failed first-device init does not save config or state after a failed
   storage probe.
3. A failed additional-device init does not save config when the existing
   remote manifest cannot be verified.
4. A wrong-passphrase pull fails before restore and creates no backup.
5. If a real pull replaced an existing target, stop the agent and preserve
   both the restored file and the timestamped copy under
   `%USERPROFILE%\.reinstate\backups\` before deciding which one to keep.
6. If local and remote state diverged, do not delete conflict metadata.
   Inspect it and resolve explicitly with `--keep-local`, `--keep-remote`, or
   `--keep-both`.
7. Remote cleanup is a provider operation. Stop all participating devices,
   retain needed independent backups, verify the exact
   `profiles/PROFILE_ID/` target, and revoke the dedicated credential after
   approved deletion.

Never delete only `manifest.age` while leaving snapshots behind. A second
device requires that encrypted manifest and will fail closed if it disappears.

## Verification checklist

- [ ] Both machines report Reinstate, native OS identity, and a recognized version
      of the same agent vendor.
- [ ] The repository exists on both machines, and one canonical project ID
      maps to the two reviewed absolute paths.
- [ ] Mac setup and self-test pass; the remote profile did not exist before
      the first successful selected-session push.
- [ ] Push and pull dry-runs each select exactly `AGENT:SESSION_ID` and show no
      mutation.
- [ ] The bucket contains only encrypted `manifest.age` and `.age` snapshot
      objects under the recorded profile prefix; credentials are absent.
- [ ] Windows second-device init reuses the exact Mac profile UUID and verifies
      the existing manifest before saving config.
- [ ] The pull reports the intended native destination and backup root, and an
      existing target is backed up before replacement.
- [ ] Claude Code or Codex resumes its own session ID from the mapped Windows
      project without a structural path error.
- [ ] No claim says the repository's still-open physical two-device acceptance
      matrix passed merely because this personal workflow succeeded.

## Current limitations

- Reinstate is explicit push/pull synchronization, not background automatic sync.
- It moves complete immutable snapshots rather than deltas and has no remote
  retention or garbage-collection policy.
- Native resume is same-vendor only. Portable handoffs between vendors are a
  later phase.
- Path mapping covers known structural fields, not prose, tool output, or
  arbitrary strings that resemble paths.
- Git state, build tools, shells, environment variables, agent authentication,
  MCP servers, skills, plugins, hooks, and settings remain device-local.
- WSL2 is a separate installer-compatible environment with its own Linux
  paths and home; it is not the native Windows destination in this guide.
- macOS amd64 and WSL2 smoke results remain
  open gates for stable Phase 1. This guide is operational documentation, not
  a certified Phase 1 agent-resume target result.
- Reinstate's encryption protects remote confidentiality, but it cannot
  protect a compromised endpoint or recover a lost passphrase.

## Mac-to-Windows session FAQ

### Does the Mac need to stay online after the push?

No. After the selected push succeeds, native Windows downloads from your
S3-compatible bucket. The Mac can be offline, but Windows still needs network
access to the bucket and the same passphrase.

### Can I use WSL instead of native Windows?

Not for the workflow described here. WSL2 can be configured as a separate
Linux Reinstate device with its own paths, home, agent installation, and
acceptance status. WSL1 and shared native/WSL agent-state folders are
unsupported.

### Can I move a Claude Code session into Codex?

No. Reinstate restores Claude Code state for Claude Code and Codex state for Codex
CLI. It does not silently reconstruct one vendor's native transcript format
for another vendor.

### What if the checkout paths are different?

That is the intended case. Map the same canonical `PROJECT_ID` to the Mac path
on the source and the Windows path on the destination. Reinstate normalizes
known structural fields during snapshot creation and expands them through the
destination mapping during restore.

### Why did pull refuse while the agent was open?

When an existing destination may be replaced, Reinstate checks whether that vendor
process is active. Safety exit `7` protects the live agent file. Close the
agent fully, rerun the dry-run, confirm the destination and backup root, and
then pull.

### Is the encrypted bucket a complete backup?

No. It is a user-owned continuity store for supported session artifacts.
Maintain independent backups of repositories and any session history your
policy requires, preserve restore-tested copies, and remember that losing the
passphrase makes remote ciphertext unrecoverable by design.
