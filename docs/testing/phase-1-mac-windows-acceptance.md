# Phase 1 MacBook + Windows Acceptance Runbook

Use this runbook to decide whether Reinstate Phase 1 is actually functional.
It tests the public installers, Claude Code and Codex setup prompts, encrypted
two-device synchronization, restore safety, and failure behavior.

**Release under test:** `v0.1.0-rc.8`
**Device A:** macOS  
**Device B:** native 64-bit Windows  
**Scope:** Claude Code and Codex CLI sessions (plus OpenCode at T5, §17b, for later release candidates)

Passing a build or watching one happy-path demo is not enough. Every mandatory
row in the final checklist must pass before Phase 1 is complete.

## 1. Rules and stop conditions

- Use a disposable project created for this run.
- Use an isolated Reinstate home on both devices.
- Create a new RC8 profile and remote prefix. Do not reuse an RC5-or-older
  profile, passphrase, snapshot, state file, or acceptance session.
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

- Claude Code `2.1.219`–`2.1.229`
- Codex CLI `0.133.0`–`0.147.0`

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
local/reinstate-phase1-acceptance-rc8
```

### Device A — macOS

Run in a new terminal:

```sh
export REINSTATE_HOME="$HOME/.reinstate-phase1-acceptance-rc8"
export PHASE1_PROJECT="$HOME/Projects/reinstate-phase1-acceptance-rc8"
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
$env:REINSTATE_HOME = Join-Path $HOME ".reinstate-phase1-acceptance-rc8"
$Phase1Project = Join-Path $HOME "Projects\reinstate-phase1-acceptance-rc8"
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
export REINSTATE_HOME="$HOME/.reinstate-phase1-acceptance-rc8"
export PHASE1_PROJECT="$HOME/Projects/reinstate-phase1-acceptance-rc8"
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
- JSON contains `"version": "0.1.0-rc.8"`.

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
- JSON contains `"version": "0.1.0-rc.8"`; and
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
Reply with exactly: REINSTATE-PHASE1-RC8-MAC-CLAUDE-A1
```

Codex test prompt:

```text
Reply with exactly: REINSTATE-PHASE1-RC8-MAC-CODEX-A1
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

If Claude creates more than one candidate file for one invocation, compare the
metadata list before and after, then count occurrences of only the exact marker
string in each candidate. Select the candidate containing the completed reply.
Do not print, open, quote, or otherwise inspect transcript content. If Codex
does not persist a brand-new rollout, stop and report that vendor failure; do
not silently reuse an older acceptance session.

## 8. Test the Claude Code setup prompt on Device A

Launch a separate Claude Code session from the Mac acceptance project and paste
the complete
[Claude Code setup prompt](../prompts/claude-code-setup.md).

Before any Reinstate command, Prompt version 6 must detect and report the exact
already-exported `REINSTATE_HOME`. Confirm it explicitly. If the agent unsets,
changes, or falls back from that value, stop the run; evidence from the default
home is not RC8 acceptance evidence.

When it asks:

- this is the first device;
- use canonical ID `local/reinstate-phase1-acceptance-rc8`;
- use the Mac absolute project path;
- select only `CLAUDE_SESSION_ID`;
- provide the non-secret endpoint and bucket; and
- enter credentials and the encryption passphrase only in Reinstate's hidden
  prompts.

The agent should detect the already-installed exact RC, inspect the public
bootstrap contract, and prepare this human-run command:

```sh
rein init \
  --project "local/reinstate-phase1-acceptance-rc8=$PHASE1_PROJECT"
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
- dry-run begins with `would push 1 snapshot(s)` and uploads nothing;
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
share it. Search the downloaded bytes locally for both exact RC8 marker
strings without printing any matching bytes:

```sh
SNAPSHOT_FILE="/absolute/path/to/downloaded-snapshot.age"
LC_ALL=C grep -aFq 'REINSTATE-PHASE1-RC8-MAC-CLAUDE-A1' "$SNAPSHOT_FILE"; echo "claude_marker_exit=$?"
LC_ALL=C grep -aFq 'REINSTATE-PHASE1-RC8-MAC-CODEX-A1' "$SNAPSHOT_FILE"; echo "codex_marker_exit=$?"
file "$SNAPSHOT_FILE"
```

Both `grep` commands must exit `1` (marker absent). Delete only the downloaded
local copy after recording the booleans; do not delete the remote object.

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

Before any Reinstate command, Prompt version 6 must detect and report the exact
already-set `$env:REINSTATE_HOME`. Confirm it explicitly. Stop if the agent
unsets, changes, or falls back from that value.

Tell it:

- this is an additional device;
- use `PHASE1_PROFILE_ID`;
- use canonical ID `local/reinstate-phase1-acceptance-rc8`;
- use the Windows absolute project path;
- select only `CODEX_SESSION_ID`; and
- never receive secrets through chat.

It should prepare this private command:

```powershell
rein init `
  --profile-id PHASE1_PROFILE_ID `
  --project "local/reinstate-phase1-acceptance-rc8=$Phase1Project"
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

Wait until Reinstate is visibly waiting at the hidden prompt before typing it.
After Reinstate exits, do not type any passphrase into PowerShell. If the
process has already returned to the shell prompt, stop and rerun the command.

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

Mandatory result: both test session IDs are discoverable on Windows. Each
dry-run must say `would pull`, never `pulled`, and must not create agent files
or backups.

## 13. Confirm Mac-to-Windows resume

Use the normal resume UI:

```powershell
claude --resume CLAUDE_SESSION_ID
codex resume CODEX_SESSION_ID
```

Select the two test session IDs and visually confirm their `A1` markers.
Do not copy transcript contents into the result note.

Mandatory result: both same-vendor sessions resume from the mapped Windows
project without path errors.

## 14. Test live-session restore safety and backup

RC8 changed this behavior deliberately. Restore liveness is now scoped to the
exact session file being replaced, and the default `restore.active_agent_policy`
is `fork`. Unrelated agents running in other projects must never block a
restore, and the session under test must never be overwritten while it is open.

On Device A, resume the Claude test session, append:

```text
REINSTATE-PHASE1-RC8-MAC-CLAUDE-A2
```

Exit Claude and push only that session:

```sh
rein push --agent claude --session CLAUDE_SESSION_ID --dry-run
rein push --agent claude --session CLAUDE_SESSION_ID
```

### 14a. Unrelated agents must not block a restore

On Device B, open Claude Code in a **different** project and leave it running.
Then, in a separate PowerShell:

```powershell
rein pull --agent claude --session CLAUDE_SESSION_ID --dry-run
```

Mandatory result: the dry-run plans normally. An agent that is not holding the
target session file must not produce a refusal. Under RC6 this same state
refused, so a refusal here is a regression.

### 14b. The live session is preserved, not overwritten

This gate failed in RC7 and is the reason RC8 exists. Read the note below before
running it.

Now open Claude Code **on the session under test** and leave it running:

```powershell
rein pull --agent claude --session CLAUDE_SESSION_ID
"exit=$LASTEXITCODE"
rein list --agent claude
```

Mandatory result:

- exit code `0`;
- the output reports that the session was in use and names a new
  distinct session, named in the output and identified by a UUID;
- the original session file is byte-for-byte unchanged; and
- repeating the same pull does not create a second fork.

**Do not accept an open file handle as the mechanism under test.** Claude Code
appends to its session file and closes it again, so a live Claude Code session
normally holds no handle at all, and the file can be opened exclusively by
another process while Claude is running. RC7 detected liveness from handles
alone, found none, attempted an ordinary in-place restore, and was stopped only
by the divergence guard with exit `6` and a recorded conflict. Exit `6` here is a
failure of this gate, not a pass.

Confirm the harder condition explicitly before drawing a conclusion: while
Claude is open on the session, verify that the session file can still be opened
exclusively, then confirm the pull is nevertheless treated as in use. That
combination is what RC7 could not satisfy.

Delete the fork before continuing so later gates start from a clean state.

**Ordering requirement.** Run section 14d *before* any step that resumes this
session. Resuming a Claude session appends to it, so a session used for a resume
or liveness check no longer matches its last pulled revision and an in-place
restore will correctly report divergence with exit `6` instead of replacing the
target. If that has already happened, reach the same evidence through the
conflict route: a real pull records one conflict, and
`rein conflicts resolve <id> --keep-remote` restores the remote copy after
backing up the existing target. The same ordering applies to sections 15 and 16,
where confirming a marker by resuming the session invalidates the unchanged
no-op that section 16 then expects.

### 14c. The refusal still works when it is requested

Set `restore.active_agent_policy = "scoped"` in `config.toml`, keep Claude Code
open on the session under test, and rerun:

```powershell
rein pull --agent claude --session CLAUDE_SESSION_ID
"exit=$LASTEXITCODE"
```

Expected exit code: `7`, naming that session rather than the whole agent. The
existing session must remain unchanged and no backup may be created.

### 14d. Restore and backup

Close every Claude Code process, restore `active_agent_policy` to `fork`, then:

```powershell
rein pull --agent claude --session CLAUDE_SESSION_ID --dry-run
rein pull --agent claude --session CLAUDE_SESSION_ID
Get-ChildItem -Recurse `
  (Join-Path $env:REINSTATE_HOME "backups") |
  Select-Object -First 20 FullName
```

Mandatory result:

- dry-run says `would pull` and reports the destination and backup root without
  mutation;
- the real pull replaces the session in place once nothing holds it;
- a timestamped backup of the previous Windows session exists; and
- the resumed session shows the `A2` marker.

## 15. Test Windows-to-Mac synchronization

On Device B, resume the selected sessions and append:

```text
REINSTATE-PHASE1-RC8-WINDOWS-CLAUDE-B1
```

```text
REINSTATE-PHASE1-RC8-WINDOWS-CODEX-B1
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
claude --resume CLAUDE_SESSION_ID
codex resume CODEX_SESSION_ID
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
   REINSTATE-PHASE1-RC8-CONFLICT-WINDOWS
   ```

   Exit Claude, but do not push.

2. On the Mac, resume the same session and append:

   ```text
   REINSTATE-PHASE1-RC8-CONFLICT-MAC
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

## 17b. OpenCode T5 sync (release candidates after OpenCode reached T5)

OpenCode syncs at T5 alongside Claude Code and Codex, so a release candidate
must exercise it by procedure, not only by the one-off journeys in
`results/2026-08-23-macos-opencode-t5-journey.md` and
`results/2026-08-23-windows-opencode-t5.md`. Use a throwaway
`XDG_DATA_HOME` on both devices and a store the **vendor** initialised
(`opencode import` of a throwaway session); never Reinstate's own schema.

On Device A (macOS), create a session with the vendor (`opencode run` inside
the mapped project, marker `REINSTATE-PHASE1-OPENCODE-A1`), then:

```sh
rein push --agent opencode --session OPENCODE_SESSION_ID --dry-run
rein push --agent opencode --session OPENCODE_SESSION_ID
```

On Device B (Windows), with OpenCode closed:

```powershell
rein pull --agent opencode --session OPENCODE_SESSION_ID --dry-run
rein pull --agent opencode --session OPENCODE_SESSION_ID
opencode --pure session list
opencode --pure export OPENCODE_SESSION_ID
```

Mandatory result: the pre-existing Windows session and the pulled session are
both listed; the export shows the `A1` marker and the assistant message's
`parentID` pointing at the user message; `message.data` rows read straight from
`opencode.db` are compact JSON (no newline inside a blob); the session's
`directory` column is the Device B mapped path; the pre-restore backup under
`$REINSTATE_HOME/backups` contains `opencode.db` and any `-wal`/`-shm` present.
Then `opencode --session OPENCODE_SESSION_ID` resumes it. Repeat the leg
Windows-to-Mac with marker `B1`. Record both rows in the sign-off checklist.

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
| `install.sh` returns 200 and installs RC8 on Mac | | |
| `install.ps1` returns 200 and installs RC8 on Windows | | |
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
| Unrelated running agents do not block a restore | | |
| A live session is forked, never overwritten | | |
| `scoped` policy still refuses, naming that session | | |
| Existing Windows target is backed up before restore | | |
| Claude Windows-to-Mac resume succeeds | | |
| Codex Windows-to-Mac resume succeeds | | |
| OpenCode Mac-to-Windows pull, vendor export, and resume succeed (§17b) | | |
| OpenCode Windows-to-Mac pull, vendor export, and resume succeed (§17b) | | |
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
  - macOS: `~/.reinstate-phase1-acceptance-rc8`
  - Windows: `%USERPROFILE%\.reinstate-phase1-acceptance-rc8`
- The disposable projects are:
  - macOS: `~/Projects/reinstate-phase1-acceptance-rc8`
  - Windows: `%USERPROFILE%\Projects\reinstate-phase1-acceptance-rc8`
- The exact remote cleanup target is only:
  `profiles/PHASE1_PROFILE_ID/`

Before removing any of them, confirm the path contains the acceptance name or
the exact recorded profile ID. See [Uninstall](../uninstall.md) for binary and
configuration boundaries.
