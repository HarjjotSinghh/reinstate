# Reinstate Phase 1 RC4 — Windows Device B acceptance

**Date:** 2026-07-26
**Evidence owner:** native Windows Device B
**Release:** `v0.1.0-rc.4`
**Verdict:** **FAIL — stopped during W1**

Device B did not receive the two-session remote manifest after the operator
entered the correct encryption passphrase. `rein status` exited `0` but
reported an empty revision and `0 sessions`; the mandatory result is exactly
two sessions. A zero exit is necessary, not sufficient.

Per the tag-exact runbook and the acceptance instructions, execution stopped
before restore, resume, section 14, W2, W3, and the automated-integrity gate.
Unexecuted rows are `NOT TESTED`, never `PASS`.

No Reinstate product code was modified. This report is the only repository
change.

## 1. Sanitized Mac handoff

The formal `MAC-RC4-M1` handoff satisfied the required input gate:

```text
profile_id=893b7056-c694-4f36-ac60-9c167e586834
canonical_project_id=local/reinstate-phase1-acceptance-rc4
claude_session_id=0f32db9b-f869-4dd9-8736-725c19edfea3
codex_session_id=019f9d25-4592-7921-a779-1843271d2da4
remote_session_count=2
ciphertext_marker_absence=PASS
```

The ciphertext check had real byte-level evidence:

- both exact RC4 marker searches exited `1`;
- the downloaded snapshot was reported as `data`, not JSON, JSONL, or text;
- the operator confirmed `manifest.age` plus opaque `.age` snapshot objects;
- credential-shaped objects were confirmed absent; and
- no object bytes, downloaded path, credential, or passphrase were provided
  to the Windows evidence owner.

Mac-reported non-secret remote identifiers:

| Item | Identifier |
| --- | --- |
| Remote revision after the second selected push | `83f677fd-05fd-40f0-bcbb-f2caa858a109` |
| Codex snapshot | `83f677fd-05fd-40f0-bcbb-f2caa858a109` |
| Claude snapshot | `42f4317b-b1f4-4387-a256-9d993756e43e` |

## 2. Release and trust evidence

`origin/main` and tag `v0.1.0-rc.4` were fetched without changing trust
configuration.

| Check | Result | Evidence |
| --- | --- | --- |
| Tag object type | PASS | `git cat-file -t v0.1.0-rc.4` → `tag` |
| Annotated tag | PASS | tag object names RC4 and points to the recorded commit |
| Signature | PASS | `git tag -v` exit `0`; good ED25519 signature for the repository signer |
| Signer policy | PASS | working-tree `.github/allowed_signers` blob `9a0c37f…` matched the tag-exact blob before command-scoped verification |
| Tag on `origin/main` | PASS | tag commit equals current `origin/main` |
| Tag commit | PASS | `f0b96006df6dee24c6d5c8d8fea5c34a655c4aff` |
| Runbook source | PASS | read with `git show v0.1.0-rc.4:docs/testing/phase-1-mac-windows-acceptance.md` |
| Setup prompt source | PASS | prompt version 4 read with `git show v0.1.0-rc.4:docs/prompts/codex-setup.md` |

The first plain `git tag -v` attempt exited `1` because this checkout did not
have an existing `gpg.ssh.allowedSignersFile` configured. Verification was
rerun with command-scoped `-c` options pointing at the repository signer file
only after proving that file was byte-identical to the tag-exact blob. No
global or local Git trust setting was written or weakened.

## 3. W0 — release and environment

### 3.1 Native Windows environment — PASS

| Field | Value |
| --- | --- |
| OS | Microsoft Windows 11 Pro |
| Build | `26200` |
| OS architecture | 64-bit |
| PowerShell | Windows PowerShell `5.1.26100.8328`, Desktop edition |
| PowerShell executable | native `%WINDIR%\System32\WindowsPowerShell\v1.0\powershell.exe` |
| Process architecture | X64 |
| Claude Code | `2.1.220` (`SUPPORTED`) |
| Codex CLI | `0.145.0` (`SUPPORTED`) |
| Git | `2.52.0.windows.1` |
| Reinstate | `0.1.0-rc.4`, commit `f0b9600`, built `2026-07-26T05:59:11Z` |

The shell was native 64-bit Windows PowerShell, not WSL. The installer process
was not elevated.

The execution environment exposed unrestricted filesystem access with approval
prompts disabled. This conflicted with the requested normal-sandbox setting;
the operator explicitly authorized continuing. This is recorded as deviation
`D-W0-1` and is not represented as equivalent to a sandboxed run.

### 3.2 Public Windows installer — PASS

| Check | Result |
| --- | --- |
| `HEAD https://reinstate.dev/install.ps1` | HTTP `200` |
| Live bootstrap SHA-256 | `13d9271ce777586629441e992de586c44919fd68a3f76dbc670338c72d975c6e` |
| Exact release pin | `v0.1.0-rc.4` |
| Canonical installer URL | exact `${Version}` Git tag |
| Pinned canonical-installer SHA-256 | `ce46d3a22d4d9349c7e6847ed65b5e8ff93b51e7f035ad3ff93b7dc19d2f1232` |
| Independent tag-byte SHA-256 | same value; PASS |
| Unpinned resolution | canonical installer refuses missing or non-SemVer `REINSTATE_VERSION` |
| Release asset base | RC4 GitHub release URL; bootstrap blanks the override |
| Bootstrap checksum layer | `installer checksum ok` |
| Binary checksum layer | `checksum ok` |
| Install elevation | none |
| Installed commands | `%LOCALAPPDATA%\Programs\Reinstate\bin\rein.exe` and `reinstate.exe` |
| Alias parity | both executable SHA-256 values identical |
| Idempotent rerun | exit `0`; reports RC4 already installed |
| Normalized user-PATH entries | exactly `1` before and after |

An existing RC3 **binary** was present at the user-local destination. The first
replacement attempt verified both checksums and then safely refused replacement.
The operator had already authorized installing RC4, so the documented
one-process `REINSTATE_CONFIRM_REPLACE=1` acknowledgment was used. The variable
was removed immediately afterward. No RC3 home, profile, prefix, state file, or
session was read, reused, changed, or deleted on Windows.

Separate automation tool calls inherited the parent agent's old process PATH,
so later evidence shells prepended the already-verified install directory.
The persisted user PATH remained normalized at exactly one entry.

### 3.3 Fresh project and pre-init setup check — PASS

Before creation:

```text
RC4 home exists: false
RC4 project exists: false
```

The disposable project was created at the required absolute Windows location
under `%USERPROFILE%\Projects`, initialized as a Git repository, and given the
runbook README. The isolated RC4 home remained absent before `init`.

`rein setup check` produced:

```text
version: 0.1.0-rc.4
platform: windows-amd64
summary: 1 check(s) failed
config: config missing
device: windows-amd64
agent.claude: SUPPORTED (2.1.220)
agent.codex: SUPPORTED (0.145.0)
keyring: OS keyring provider reachable
exit=3
```

Only `config missing` failed. Device, keyring, and both installed adapters
reported their real supported states. The check did not create the home.

## 4. W1 — init and mandatory status gate

### 4.1 Codex setup prompt v4 init — PASS

The additional-device command used:

```powershell
rein init `
  --profile-id 893b7056-c694-4f36-ac60-9c167e586834 `
  --project "local/reinstate-phase1-acceptance-rc4=$Phase1Project"
```

The operator entered storage credentials only at Reinstate's private prompts.
No credential or keyring value was printed. Init exited `0`, created the
expected isolated-home layout, and printed the exact required profile ID.

Post-init, pre-status metadata-only baseline:

```text
expected home entries: backups, cache, config.toml, conflicts, locks, logs, state.json
exact Claude target ID files: 0
exact Codex target ID files: 0
backup files: 0
backup directories: 0
```

No transcript, config content, auth file, keyring value, or `.age` object was
read.

### 4.2 Correct-passphrase status — FAIL

The operator ran `rein status`, entered the correct passphrase at Reinstate's
hidden prompt, and reported:

```text
Encryption passphrase:
remote revision:  (0 sessions)
status_exit=0
```

Mandatory result:

```text
remote revision: <non-secret UUID> (2 sessions)
```

Actual result was an empty revision and `0 sessions`. The operator requested
counting this as a correct-passphrase test; it is recorded as an attempted
correct-passphrase test but cannot be marked `PASS`.

Metadata-only post-command verification:

```text
exact Claude target ID files: 0
exact Codex target ID files: 0
backup files: 0
backup directories: 0
```

No target or backup mutation occurred.

### 4.3 Downstream W1 work — NOT TESTED

Execution stopped at the failed mandatory status gate. The following were not
executed:

- deliberate wrong-passphrase refusal and required exit `4`;
- Windows post-init `rein setup check`;
- Windows `rein doctor --self-test`;
- Codex dry-run and exact-ID pull;
- Claude dry-run and exact-ID pull;
- mapped Windows discovery;
- `codex resume` exact-ID proof;
- Claude normal-discovery proof;
- `claude --resume` exact-ID proof; and
- visual confirmation of both A1 markers.

`WINDOWS-RC4-W1-PASS` was not emitted.

## 5. W2 and W3 — NOT TESTED

The run stopped before section 14 as required. No Mac A2 request was made, no
active-agent overwrite test ran, no backup restore ran, no B1 marker was
created or pushed, and `WINDOWS-RC4-W2-READY` was not emitted.

No divergence was created, no Mac conflict marker was requested, no conflict
pull ran, and `--keep-both` was not exercised.

## 6. Findings and classification

### 6.1 Release-blocking acceptance findings

#### `RB-W1-1` — correct-passphrase status returned zero remote sessions

- **Observed:** exit `0`, empty remote revision, `0 sessions`.
- **Required:** exactly the two selected Mac session IDs.
- **Mutation:** no exact target files and no backups before or after.
- **Classification:** acceptance/test-state failure; root cause undetermined.
  It is not yet proven to be a Reinstate binary defect, storage configuration
  defect, or operator-input defect because the runbook required stopping before
  downstream mutation or ad-hoc repair.
- **Impact:** blocks every restore/resume and downstream W2/W3 gate.

#### `RB-MAC-1` — Mac-reported silent initialized-home overwrite

The supplied Mac evidence reports that `rein init` replaced an existing
home's `config.toml` and `state.json`, minted a new profile ID, and created no
backup or confirmation. The Mac report classifies the initiating stale
`REINSTATE_HOME` as an operator/test-state deviation and the overwrite behavior
as product defect `F1`.

This Windows run did not reproduce or touch that RC3 home. Nevertheless, the
tag-exact runbook explicitly lists **silent overwrite** as an immediate stop
condition, so the reported behavior remains release-blocking until fixed or
disproved. Being outside a section 19 row does not cancel a top-level stop
condition.

### 6.2 Non-blocking findings and deviations

- `D-W0-1`: the operator explicitly authorized proceeding despite this agent
  session having unrestricted filesystem access and approvals disabled.
- `D-W0-2`: an RC3 binary replacement required the installer's documented
  confirmation override. The safety refusal worked, RC4 then installed, and
  the ordinary rerun was idempotent.
- `D-W0-3`: automated child PowerShell processes inherited a stale process
  PATH. Persisted user PATH evidence remained exactly one normalized entry.
- `N-MAC-1`: the Mac report records that the POSIX installer can wait
  indefinitely on an unattended readable `/dev/tty`; not reproduced on
  Windows.

No vendor resume finding exists because neither restored session reached a
vendor resume command.

## 7. Section 19 reconciliation

| # | Gate | Result | Evidence |
| ---: | --- | --- | --- |
| 1 | `install.sh` returns 200 and installs RC4 on Mac | PASS | Mac report |
| 2 | `install.ps1` returns 200 and installs RC4 on Windows | PASS | §3.2 |
| 3 | Both installers are idempotent and PATH-safe | PASS | Mac report and §3.2 |
| 4 | Pre-init missing-config failure is accurate | PASS | Mac report and §3.3 |
| 5 | Post-init setup check and self-test pass on both devices | PARTIAL | Mac PASS; Windows NOT TESTED |
| 6 | Claude setup prompt completes on the Mac | PASS | Mac report |
| 7 | Codex setup prompt completes on Windows | FAIL | Init passed; mandatory two-session status failed |
| 8 | Only two selected test sessions reach the remote manifest | PASS | Mac report recorded exactly the selected two |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | Mac byte-level check |
| 10 | Wrong passphrase fails without mutation | NOT TESTED | stopped at §4.2 |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | stopped at §4.2 |
| 12 | Codex Mac-to-Windows resume succeeds | NOT TESTED | stopped at §4.2 |
| 13 | Active-agent overwrite is refused | NOT TESTED | stopped before section 14 |
| 14 | Existing Windows target is backed up before restore | NOT TESTED | stopped before section 14 |
| 15 | Claude Windows-to-Mac resume succeeds | NOT TESTED | stopped before section 14 |
| 16 | Codex Windows-to-Mac resume succeeds | NOT TESTED | stopped before section 14 |
| 17 | Existing Mac targets are backed up before restore | NOT TESTED | stopped before section 14 |
| 18 | Unchanged pushes skip without new snapshots | NOT TESTED | stopped before section 14 |
| 19 | Divergence records a conflict without overwrite | NOT TESTED | stopped before section 14 |
| 20 | `--keep-both` preserves both branches | NOT TESTED | stopped before section 14 |
| 21 | All required GitHub checks are green | NOT TESTED | automated-integrity section not executed after stop |

Totals:

```text
PASS=7
PARTIAL=1
FAIL=1
NOT_TESTED=12
ALL_21_PASSED=false
```

## 8. Sanitized stopping handoff

```text
WINDOWS-RC4-W1-FAIL
release=v0.1.0-rc.4
tag_commit=f0b96006df6dee24c6d5c8d8fea5c34a655c4aff
profile_id=893b7056-c694-4f36-ac60-9c167e586834
canonical_project_id=local/reinstate-phase1-acceptance-rc4
claude_session_id=0f32db9b-f869-4dd9-8736-725c19edfea3
codex_session_id=019f9d25-4592-7921-a779-1843271d2da4
correct_status_exit=0
correct_status_remote_session_count=0
wrong_passphrase_test=NOT_TESTED
claude_target_count_before=0
claude_target_count_after=0
codex_target_count_before=0
codex_target_count_after=0
backup_file_count_before=0
backup_file_count_after=0
windows_project_is_absolute=true
windows_project_key_is_mapped=false
windows_claude_source_slug_absent=NOT_TESTED
report_path=docs/testing/results/2026-07-26-windows-phase1-rc4.md
END-WINDOWS-RC4-W1-FAIL
```

Phase 1 RC4 acceptance is **FAIL**. All 21 mandatory rows did not pass.
