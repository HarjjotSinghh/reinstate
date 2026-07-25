# Phase 1 MacBook + Windows Acceptance Runbook

Use this runbook to decide whether Reinstate Phase 1 is actually functional.
It tests the public installers, Claude Code and Codex setup prompts, encrypted
two-device synchronization, restore safety, and failure behavior.

**Release under test:** `v0.1.0-rc.3`
**Device A:** macOS  
**Device B:** native 64-bit Windows  
**Scope:** Claude Code and Codex CLI sessions only

Passing a build or watching one happy-path demo is not enough. Every mandatory
row in the final checklist must pass before Phase 1 is complete.

## 1. Rules and stop conditions

- Use a disposable project created for this run.
- Use an isolated Reinstate home on both devices.
- Never paste the R2/S3 secret key or encryption passphrase into an AI prompt,
  command argument, ordinary environment variable, screenshot, or test report.
- Do not use `--all`. Select only the two disposable session IDs.
- Do not inspect or publish transcript contents as evidence. Session IDs,
  counts, versions, exit codes, snapshot IDs, and redacted paths are enough.
- Do not delete real Claude Code or Codex data during cleanup.
- Stop immediately on a checksum mismatch, plaintext remote session object,
  silent overwrite, missing backup, or unexplained exit code.

Warnings are not automatically failures. A `SUPPORTED` adapter is mandatory;
an `UNTESTED` or `UNSUPPORTED` adapter blocks mutating acceptance.

## 2. Record the test

Create a private result note with no secrets:

| Field | Value |
| ----- | ----- |
| Date/time | |
| Mac model and macOS version | |
| Mac architecture | |
| Windows edition/build | |
| Claude Code version | |
| Codex CLI version | |
| Reinstate version | |
| GitHub PR/check run | |
| Device A profile ID | |
| Claude test session ID | |
| Codex test session ID | |

For each command, record `PASS` or `FAIL`, its exit code, and a redacted
screenshot or copied output. Never record hidden inputs.

## 3. Prerequisites

You need:

- a Cloudflare R2 or other S3-compatible bucket dedicated to this test;
- its HTTPS endpoint, bucket name, access-key ID, and secret access key;
- permission to put, get, list, and delete objects in that bucket;
- a long test encryption passphrase stored in a password manager;
- Claude Code and Codex CLI installed on both devices; and
- Git installed on both devices.

The RC compatibility evidence currently recognizes these inclusive stable
version ranges:

- Claude Code `2.1.219`–`2.1.220`
- Codex CLI `0.133.0`–`0.145.0`

Check the installed tools.

macOS:

```sh
sw_vers
uname -m
claude --version
codex --version
git --version
```

Windows PowerShell:

```powershell
Get-ComputerInfo |
  Select-Object WindowsProductName, WindowsVersion, OsBuildNumber
[Environment]::Is64BitOperatingSystem
claude --version
codex --version
git --version
```

Mandatory result:

- macOS reports `arm64` or `x86_64`;
- Windows reports a 64-bit operating system; and
- both agent CLIs run on both devices.

If an agent version falls outside the recognized range, continue through
read-only checks, but do not call Phase 1 complete when `rein setup check`
reports `UNTESTED`.

## 4. Create the disposable mapped project

Use the same canonical ID on both devices:

```text
local/reinstate-phase1-acceptance
```

### Device A — macOS

Run in a new terminal:

```sh
export REINSTATE_HOME="$HOME/.reinstate-phase1-acceptance"
export PHASE1_PROJECT="$HOME/Projects/reinstate-phase1-acceptance"
mkdir -p "$PHASE1_PROJECT"
cd "$PHASE1_PROJECT"
git init
printf '# Reinstate Phase 1 acceptance\n' > README.md
```

Keep this terminal open. Any Claude Code or Codex process launched from it
inherits the isolated `REINSTATE_HOME`.

### Device B — Windows PowerShell

Run in a new PowerShell:

```powershell
$env:REINSTATE_HOME = Join-Path $HOME ".reinstate-phase1-acceptance"
$Phase1Project = Join-Path $HOME "Projects\reinstate-phase1-acceptance"
New-Item -ItemType Directory -Force -Path $Phase1Project | Out-Null
Set-Location $Phase1Project
git init
Set-Content -Path README.md -Value "# Reinstate Phase 1 acceptance"
```

Keep this PowerShell open for the same reason.

Mandatory result: both project paths are absolute, writable, and different
across operating systems.

## 5. Test the live public installers

The website routes must be live before continuing.

### Device A — macOS

```sh
curl -fsSI https://reinstate.dev/install.sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

If the script tells you that `~/.local/bin` was added to a shell file, open a
new terminal and re-export the acceptance environment:

```sh
export REINSTATE_HOME="$HOME/.reinstate-phase1-acceptance"
export PHASE1_PROJECT="$HOME/Projects/reinstate-phase1-acceptance"
cd "$PHASE1_PROJECT"
```

Verify:

```sh
command -v rein
command -v reinstate
rein version --json
```

Expected:

- HTTP status `200`;
- `rein` and `reinstate` resolve under `~/.local/bin`;
- the installer reports both checksum checks as successful; and
- JSON contains `"version": "0.1.0-rc.3"`.

Run the same one-liner again. It must report the same version already installed
and must not duplicate its PATH entry.

### Device B — native Windows PowerShell

```powershell
(Invoke-WebRequest -Method Head https://reinstate.dev/install.ps1).StatusCode
irm https://reinstate.dev/install.ps1 | iex
```

Verify in the same PowerShell:

```powershell
(Get-Command rein).Source
(Get-Command reinstate).Source
rein version --json
```

Expected:

- status `200`;
- both commands resolve under
  `%LOCALAPPDATA%\Programs\Reinstate\bin`;
- checksum verification succeeds;
- JSON contains `"version": "0.1.0-rc.3"`; and
- no elevated PowerShell prompt appears.

Run the one-liner again. It must not duplicate the user PATH entry:

```powershell
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\Reinstate\bin"
$NormalizedInstallDir = $InstallDir.TrimEnd("\").ToLowerInvariant()
(
  [Environment]::GetEnvironmentVariable("Path", "User") -split ";" |
    Where-Object {
      $_.Trim().Trim('"').TrimEnd("\").ToLowerInvariant() -eq
        $NormalizedInstallDir
    }
).Count
```

Expected count: `1`.

## 6. Verify pre-init failure is honest

The isolated acceptance homes do not have a config yet.

macOS:

```sh
rein setup check
echo "exit=$?"
```

Windows:

```powershell
rein setup check
"exit=$LASTEXITCODE"
```

Expected exit code: `3`, with `config missing`. Device detection and installed
agent layouts must not report an unsupported platform.

This failure is expected before `init`; pretending it passed would be dumb.

## 7. Create two source sessions on the Mac

From the disposable project on Device A, create one harmless session per agent.
Do not include credentials, proprietary code, or personal data.

Claude Code test prompt:

```text
Reply with exactly: REINSTATE-PHASE1-MAC-CLAUDE-A1
```

Codex test prompt:

```text
Reply with exactly: REINSTATE-PHASE1-MAC-CODEX-A1
```

Exit both agents cleanly. Then list metadata:

```sh
rein list --agent claude
rein list --agent codex
```

Copy the two new session IDs into your private result note:

```text
CLAUDE_SESSION_ID=...
CODEX_SESSION_ID=...
```

If it is unclear which ID is new, compare the list before and after creating a
fresh marker session. Do not open transcript files to identify it.

## 8. Test the Claude Code setup prompt on Device A

Launch a separate Claude Code session from the Mac acceptance project and paste
the complete
[Claude Code setup prompt](../prompts/claude-code-setup.md).

When it asks:

- this is the first device;
- use canonical ID `local/reinstate-phase1-acceptance`;
- use the Mac absolute project path;
- select only `CLAUDE_SESSION_ID`;
- provide the non-secret endpoint and bucket; and
- enter credentials and the encryption passphrase only in Reinstate's hidden
  prompts.

The agent should detect the already-installed exact RC, inspect the public
bootstrap contract, and prepare this human-run command:

```sh
rein init \
  --project "local/reinstate-phase1-acceptance=$PHASE1_PROJECT"
```

Run it privately. Record the printed non-secret `profile_id` as
`PHASE1_PROFILE_ID`.

The Claude setup workflow must then complete:

```sh
rein setup check
rein doctor --self-test
rein push --agent claude --session CLAUDE_SESSION_ID --dry-run
rein push --agent claude --session CLAUDE_SESSION_ID
```

Mandatory results:

- `setup check` says all checks passed;
- `agent.claude` and `agent.codex` are `SUPPORTED`;
- the synthetic self-test passes;
- dry-run uploads nothing;
- the real push reports one snapshot; and
- Claude returns a redacted report without secrets or transcript contents.

If the agent tries to choose `--all`, stop it. The prompt contract says the
human chooses scope.

## 9. Push the Mac Codex session

Still on Device A:

```sh
rein push --agent codex --session CODEX_SESSION_ID --dry-run
rein push --agent codex --session CODEX_SESSION_ID
rein status
```

Enter the same encryption passphrase privately for each command.

Mandatory result: the remote manifest contains exactly the two selected test
sessions, not unrelated local sessions.

## 10. Check ciphertext-only remote storage

In the bucket, find the acceptance prefix:

```text
profiles/PHASE1_PROFILE_ID/
```

It should contain:

```text
manifest.age
snapshots/<opaque-uuid>.age
```

Download one `.age` snapshot through the storage provider's normal UI. Do not
share it. Search the downloaded bytes locally for both marker strings.

Mandatory result:

- neither plaintext marker appears;
- the object is not readable JSON or JSONL; and
- no `auth.json`, token, credential, or `.env` object exists under the profile
  prefix.

The provider seeing ciphertext filenames is expected. Seeing transcript
plaintext is a release blocker.

## 11. Test the Codex setup prompt on Device B

Launch Codex from the Windows acceptance project and paste the complete
[Codex setup prompt](../prompts/codex-setup.md).

Tell it:

- this is an additional device;
- use `PHASE1_PROFILE_ID`;
- use canonical ID `local/reinstate-phase1-acceptance`;
- use the Windows absolute project path;
- select only `CODEX_SESSION_ID`; and
- never receive secrets through chat.

It should prepare this private command:

```powershell
rein init `
  --profile-id PHASE1_PROFILE_ID `
  --project "local/reinstate-phase1-acceptance=$Phase1Project"
```

Run it and enter the same storage coordinates and credentials.

After successful init, pause the agent-assisted workflow for the negative test
in the next section.

## 12. Test wrong-passphrase refusal

On Device B:

```powershell
rein status
```

Enter a deliberately wrong passphrase at the hidden prompt.

Expected:

- non-zero exit;
- decryption/authentication failure;
- no restored session;
- no changed agent file; and
- no new local backup.

Now rerun `rein status` with the correct passphrase. It must report the two
remote sessions.

Resume the Codex setup workflow. It must complete:

```powershell
rein setup check
rein doctor --self-test
rein pull --agent codex --session CODEX_SESSION_ID --dry-run
rein pull --agent codex --session CODEX_SESSION_ID
rein list --agent codex
```

Then pull the Claude session manually:

```powershell
rein pull --agent claude --session CLAUDE_SESSION_ID --dry-run
rein pull --agent claude --session CLAUDE_SESSION_ID
rein list --agent claude
```

Mandatory result: both test session IDs are discoverable on Windows. Dry-runs
must not create agent files or backups.

## 13. Confirm Mac-to-Windows resume

Use the normal resume UI:

```powershell
claude --resume
codex resume
```

Select the two test session IDs and visually confirm their `A1` markers.
Do not copy transcript contents into the result note.

Mandatory result: both same-vendor sessions resume from the mapped Windows
project without path errors.

## 14. Test active-agent overwrite refusal and backup

On Device A, resume the Claude test session, append:

```text
REINSTATE-PHASE1-MAC-CLAUDE-A2
```

Exit Claude and push only that session:

```sh
rein push --agent claude --session CLAUDE_SESSION_ID --dry-run
rein push --agent claude --session CLAUDE_SESSION_ID
```

On Device B, start Claude Code and leave it open. In a separate PowerShell:

```powershell
rein pull --agent claude --session CLAUDE_SESSION_ID
```

Expected exit code: `7`, with a safety refusal telling you to close Claude
Code. The existing session must remain unchanged.

Close every Claude Code process, then run:

```powershell
rein pull --agent claude --session CLAUDE_SESSION_ID --dry-run
rein pull --agent claude --session CLAUDE_SESSION_ID
Get-ChildItem -Recurse `
  (Join-Path $env:REINSTATE_HOME "backups") |
  Select-Object -First 20 FullName
```

Mandatory result:

- dry-run reports the destination and backup root without mutation;
- real pull succeeds after Claude closes;
- a timestamped backup of the previous Windows session exists; and
- the resumed session shows the `A2` marker.

## 15. Test Windows-to-Mac synchronization

On Device B, resume the selected sessions and append:

```text
REINSTATE-PHASE1-WINDOWS-CLAUDE-B1
```

```text
REINSTATE-PHASE1-WINDOWS-CODEX-B1
```

Exit both agents. Push only those IDs:

```powershell
rein push --agent claude --session CLAUDE_SESSION_ID --dry-run
rein push --agent claude --session CLAUDE_SESSION_ID
rein push --agent codex --session CODEX_SESSION_ID --dry-run
rein push --agent codex --session CODEX_SESSION_ID
```

On Device A, close Claude Code and Codex, then:

```sh
rein pull --agent claude --session CLAUDE_SESSION_ID --dry-run
rein pull --agent claude --session CLAUDE_SESSION_ID
rein pull --agent codex --session CODEX_SESSION_ID --dry-run
rein pull --agent codex --session CODEX_SESSION_ID
find "$REINSTATE_HOME/backups" -type f -print
```

Confirm both `B1` markers through:

```sh
claude --resume
codex resume
```

Mandatory result: both sessions resume on the Mac, and existing Mac targets
were backed up before replacement.

## 16. Test unchanged-session no-op

Without changing either restored Mac session:

```sh
rein push --agent claude --session CLAUDE_SESSION_ID
rein push --agent codex --session CODEX_SESSION_ID
```

Expected for each: `pushed 0 snapshot(s), skipped 1 unchanged`.

A no-op must not create a new remote snapshot or revision.

## 17. Test conflict creation and keep-both recovery

Start from the successfully synchronized Claude session.

1. On Windows, resume it and append:

   ```text
   REINSTATE-PHASE1-CONFLICT-WINDOWS
   ```

   Exit Claude, but do not push.

2. On the Mac, resume the same session and append:

   ```text
   REINSTATE-PHASE1-CONFLICT-MAC
   ```

   Exit and push:

   ```sh
   rein push --agent claude --session CLAUDE_SESSION_ID
   ```

3. On Windows, close Claude and pull:

   ```powershell
   rein pull --agent claude --session CLAUDE_SESSION_ID
   "exit=$LASTEXITCODE"
   rein conflicts list
   ```

Expected exit code: `6`, with one recorded conflict. Copy its non-secret
conflict ID.

Inspect metadata, then preserve both branches:

```powershell
rein conflicts show CONFLICT_ID
rein conflicts resolve CONFLICT_ID --keep-both
rein conflicts list
rein list --agent claude
```

Mandatory result:

- the pull does not overwrite the locally diverged session;
- conflict metadata remains until resolution succeeds;
- `--keep-both` preserves the Windows-local session and restores a distinct
  vendor-safe fork of the Mac remote session; and
- the resolved conflict disappears from the active list.

## 18. Automated integrity gates

The pull request that publishes these installers must have green checks for:

- Go tests on Ubuntu, macOS, and Windows;
- native Windows bootstrap execution and PATH behavior;
- POSIX bootstrap behavior and hash-mismatch refusal;
- exact-tag and no-`latest` static contracts;
- website `npm ci`, tests, and production build;
- byte-for-byte inclusion of both scripts in the Astro/Vercel output;
- lint, race, docs, fixture secret scan, and vulnerability checks.

Do not replace a missing Windows check with “the PowerShell looks right.”
PowerShell has humbled better people.

## 19. Final sign-off checklist

Mark every mandatory row.

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| `install.sh` returns 200 and installs RC3 on Mac | | |
| `install.ps1` returns 200 and installs RC3 on Windows | | |
| Both installers are idempotent and PATH-safe | | |
| Pre-init missing-config failure is accurate | | |
| Post-init setup check and self-test pass on both devices | | |
| Claude setup prompt completes on the Mac | | |
| Codex setup prompt completes on Windows | | |
| Only two selected test sessions reach the remote manifest | | |
| Remote manifest/snapshots are ciphertext-only | | |
| Wrong passphrase fails without mutation | | |
| Claude Mac-to-Windows resume succeeds | | |
| Codex Mac-to-Windows resume succeeds | | |
| Active-agent overwrite is refused | | |
| Existing Windows target is backed up before restore | | |
| Claude Windows-to-Mac resume succeeds | | |
| Codex Windows-to-Mac resume succeeds | | |
| Existing Mac targets are backed up before restore | | |
| Unchanged pushes skip without new snapshots | | |
| Divergence records a conflict without overwrite | | |
| `--keep-both` preserves both branches | | |
| All required GitHub checks are green | | |

### Phase 1 passes only when

- every row is `PASS`;
- no evidence contains secrets or transcript contents;
- there is no unexplained warning or non-zero exit; and
- the tested agent versions are reported as `SUPPORTED`.

Otherwise Phase 1 remains open. Fix the defect, cut a new RC if binary behavior
changed, and rerun the failed gate plus every downstream gate.

## 20. Cleanup

Cleanup is optional and must be reviewed before deletion.

- Keep the private result note.
- Keep the profile prefix until failures are diagnosed.
- Do not delete real Claude Code or Codex session directories.
- The isolated Reinstate homes are:
  - macOS: `~/.reinstate-phase1-acceptance`
  - Windows: `%USERPROFILE%\.reinstate-phase1-acceptance`
- The disposable projects are:
  - macOS: `~/Projects/reinstate-phase1-acceptance`
  - Windows: `%USERPROFILE%\Projects\reinstate-phase1-acceptance`
- The exact remote cleanup target is only:
  `profiles/PHASE1_PROFILE_ID/`

Before removing any of them, confirm the path contains the acceptance name or
the exact recorded profile ID. See [Uninstall](../uninstall.md) for binary and
configuration boundaries.
