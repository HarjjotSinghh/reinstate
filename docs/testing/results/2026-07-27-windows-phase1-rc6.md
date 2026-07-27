# Reinstate Phase 1 RC6 — Windows Device B acceptance

**Report date:** 2026-07-27
**Evidence owner:** native Windows Device B
**Release:** `v0.1.0-rc.6`
**Peeled tag commit:** `9019bd9cb4094eae648339dfecb2c6449c1b60d2`
**Report branch:** `test/phase1-rc6-windows-report`
**Verdict:** **FAIL — stopped during W1**

Windows release, installer, initialization, refusal, strict-status, scoped
restore, path-remapping, and local-discovery behavior passed. Acceptance
stopped before exact interactive Codex resume could be proven: an autonomous
`codex exec resume --ephemeral` probe loaded the exact restored session and
returned successfully, but still changed the local rollout file. That created
local divergence from Reinstate's recorded post-pull baseline before Device A
could publish A2.

This is an acceptance-harness contamination, not a confirmed Reinstate product
defect. The signed runbook nevertheless requires an immediate stop because W2
depends on an unchanged Windows baseline.

W2 and W3 are `NOT TESTED`, never `PASS`.

No Reinstate product code was modified. This report is the only repository
change.

## 1. Evidence boundary and deviations

The final Windows run used native 64-bit Windows PowerShell and the stock
published RC6 binary. It operated only on:

- profile `b0ddbe7b-b4f6-43e4-8194-6043baa6dd61`;
- Claude session `499bfb6f-8f84-41c4-bcf3-c1b61d9ad1f3`;
- Codex session `019fa308-6f70-7290-8742-80cc89b39f3c`; and
- canonical project `local/reinstate-phase1-acceptance-rc6`.

`--all` and `--force` were never used.

The operator explicitly replaced the original hidden-prompt-only execution
contract with autonomous execution backed by a local Desktop credential file.
Storage credentials were provided to child processes through Reinstate's
documented non-interactive environment provider. The encryption passphrase was
provided through `REINSTATE_PASSPHRASE_FD` using an inherited file handle, not
an argument or ordinary environment variable.

Only field-presence booleans, exit codes, counts, IDs, revisions, redacted
paths, and sanitized error classifications were recorded. No credential or
passphrase value appears in this report, repository history, command output, or
draft PR.

This automation is a real deviation:

- the physical F3 prompt path was not executed;
- the Windows setup prompt did not use Reinstate's human hidden prompts;
- the coordinating Codex process and unrelated Claude Desktop processes were
  not closed before first restore; and
- the two A1 transcript markers were not visually inspected.

Two earlier operator-driven F3 attempts entered valid coordinates instead of
the required malformed coordinates and initialized the home. Their homes and
projects were preserved under timestamped evidence archive names. They are
discarded from the final-run gate evidence. Endpoint and bucket values appeared
in chat during those discarded attempts; no access key, secret key, or
passphrase did. The final run began after the exact home and project paths were
again absent.

## 2. Environment

| Field | Value |
| ----- | ----- |
| Windows product | Windows 10 Pro (reported compatibility name) |
| Windows version/build | 2009 / 26200 |
| OS architecture | 64-bit |
| PowerShell | Desktop 5.1.26100.8328 |
| PowerShell executable | native System32 Windows PowerShell |
| PowerShell process architecture | 64-bit |
| Claude Code | 2.1.220 (`SUPPORTED`) |
| Codex CLI | 0.145.0 (`SUPPORTED`) |
| Git | 2.52.0.windows.1 |
| Reinstate | 0.1.0-rc.6, commit `9019bd9`, built 2026-07-27T09:09:11Z |
| Effective isolated home | `%USERPROFILE%\.reinstate-phase1-acceptance-rc6` |
| Disposable project | `%USERPROFILE%\Projects\reinstate-phase1-acceptance-rc6` |

The final disposable project contained only `.git` and `README.md` before
initialization.

## 3. W0 — release, installer, and pre-init

### 3.1 Release provenance

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Annotated tag | PASS | `git cat-file -t v0.1.0-rc.6` returned `tag` |
| Peeled commit | PASS | Exact commit `9019bd9cb4094eae648339dfecb2c6449c1b60d2` |
| Reachable from `origin/main` | PASS | `git merge-base --is-ancestor` exited 0 |
| Tag-matching allowed-signers blob | PASS | Working-tree and tagged blobs matched |
| Local tag verification | PASS | Command-scoped `git tag -v` exited 0 |
| GitHub release-commit verification | PASS | `verified=true`, `reason=valid` |
| GitHub tag-object verification | PARTIAL | `verified=false`, `reason=unknown_key` |

The local verification used only the allowed-signers file from the exact tag;
no global trust configuration was changed. The GitHub tag-object limitation
matches the open Device A finding.

### 3.2 Public Windows installer

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `https://reinstate.dev/install.ps1` | PASS | HTTP 200 |
| Exact pin | PASS | Only `v0.1.0-rc.6`; no other RC |
| No `latest` resolution | PASS | No `latest` token in the bootstrap |
| Canonical-installer checksum | PASS | Live canonical file matched the bootstrap pin and tagged file |
| Release-archive checksum | PASS | Installer emitted `checksum ok` |
| Release URL restriction | PASS | Canonical installer uses the exact `$Version` release base |
| Installed version | PASS | Both aliases reported 0.1.0-rc.6 / `9019bd9` |
| No elevation | PASS | Non-elevated native PowerShell |
| Idempotent rerun | PASS | Reported exact RC6 already installed; exit 0 |
| PATH safety | PASS | Normalized user PATH contained the install directory exactly once |

Both commands resolved under:

```text
%LOCALAPPDATA%\Programs\Reinstate\bin
```

### 3.3 Pre-init honesty

`rein setup check` before initialization:

```text
exit=3
config_missing=true
device_windows_amd64=true
claude_supported=true
codex_supported=true
keyring_reachable=true
```

Result: **PASS**.

## 4. W1 — regression and restore evidence

### 4.1 F3 malformed-coordinate refusal

The final run intentionally constructed an endpoint containing the bucket
suffix while also supplying the bucket separately. Values are omitted.

```text
execution_mode=documented non-interactive credential provider
exit=4
remote_profile_manifest_not_found=true
config_exists=false
state_exists=false
backup_count=0
credential_values_in_output=false
```

Behavioral result: **PASS**. Physical hidden-prompt method: **PARTIAL** due the
operator-authorized autonomous execution deviation.

### 4.2 Correct initialization and post-init health

```text
init_exit=0
config_created=true
state_created=true
setup_check_exit=0
setup_all_passed=true
claude_supported=true
codex_supported=true
doctor_self_test_exit=0
doctor_self_test_passed=true
```

Result: **PASS**.

### 4.3 F1 default refusal

The same initialization was repeated without `--force`. Config/state hashes
were compared privately; only equality booleans were recorded.

```text
exit=7
safety_refusal=true
config_unchanged=true
state_unchanged=true
backups_before=0
backups_after=0
force_used=false
```

Result: **PASS**.

### 4.4 Wrong-passphrase refusal

A synthetic incorrect passphrase was supplied through an inherited secret
file descriptor.

```text
exit=4
decryption_refused=true
config_unchanged=true
state_unchanged=true
claude_targets_before=0
claude_targets_after=0
codex_targets_before=0
codex_targets_after=0
backups_before=0
backups_after=0
```

Result: **PASS**.

### 4.5 Correct status

```text
exit=0
revision_nonempty=true
remote_session_count=2
exact_expected_two_sessions=true
config_unchanged=true
state_unchanged=true
```

The two remote keys were exactly:

```text
claude:499bfb6f-8f84-41c4-bcf3-c1b61d9ad1f3
codex:019fa308-6f70-7290-8742-80cc89b39f3c
```

Result: **PASS**.

### 4.6 F2 missing-manifest strict status

The real home was copied to the required
`-missing-manifest-probe` path. Exactly one profile-prefix occurrence was
rewritten to a nonexistent prefix; config contents were never printed.

```text
probe_exit=4
remote_profile_manifest_not_found=true
probe_config_state_unchanged=true
probe_backups_before=0
probe_backups_after=0
probe_kept=true
real_status_before_exit=0
real_status_after_exit=0
real_revision_unchanged_nonempty=true
real_exact_two_sessions_before=true
real_exact_two_sessions_after=true
real_config_state_unchanged=true
```

Result: **PASS**.

### 4.7 Scoped dry-runs

| Agent | Exit | Required wording | Mutation |
| ----- | ----: | ---------------- | -------- |
| Codex | 0 | `would pull 1 snapshot(s)`; never `pulled` | none |
| Claude | 0 | `would pull 1 snapshot(s)`; never `pulled` | none |

Targets and backups remained at zero after both dry-runs.

Result: **PASS**.

### 4.8 Scoped pulls, mapping, and discovery

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Codex pull | PASS | Exit 0; pulled exactly one snapshot |
| Claude pull | PASS | Exit 0; pulled exactly one snapshot |
| New-target backups | PASS | Zero before and after, as no prior target existed |
| Codex destination | PASS | One date-partitioned `sessions/2026/07/27/...jsonl` target |
| Claude destination | PASS | One target under the Windows project-directory key |
| Mac source slug absent | PASS | Destination key equals the mapped Windows project key |
| Codex local discovery | PASS | Exact ID found once; canonical project ID matched |
| Claude local discovery | PASS | Exact ID found once; canonical project ID matched |

The Claude destination was:

```text
projects/C--Users-admin-Projects-reinstate-phase1-acceptance-rc6/<CLAUDE_SESSION_ID>.jsonl
```

No restored vendor file was manually moved.

### 4.9 Exact resume — acceptance stop

An initial Windows Terminal harness intended to launch the exact interactive:

```text
codex resume 019fa308-6f70-7290-8742-80cc89b39f3c
```

did not activate or capture a result. No extra Codex process remained and the
session file was unchanged. That attempt is `NOT TESTED`, not a pass.

The autonomous fallback was:

```text
codex exec resume --ephemeral --json 019fa308-6f70-7290-8742-80cc89b39f3c <SANITIZED_PROBE>
```

Observed:

```text
exit=0
json_event_count=5
exact_session_id_observed=true
sanitized_probe_reply_observed=true
session_file_unchanged=false
reinstate_baseline_still_matches=false
conflict_records=0
backups=0
```

The exact restored ID loaded successfully, but `--ephemeral` still changed the
source rollout. This invalidated the unchanged Windows baseline needed for the
next A2 pull. The human visual A1 confirmation was not performed.

Result: **FAIL** for required Codex Mac-to-Windows resume evidence. Claude exact
resume was not attempted after the stop and is **NOT TESTED**.

This failure belongs to the autonomous acceptance harness. It does not prove a
Reinstate restore or adapter defect.

## 5. W2 and W3

### W2 — NOT TESTED

The following were not executed:

- physical active-Claude overwrite refusal after Mac A2;
- successful A2 replacement and timestamped Windows backup;
- visual A2 confirmation;
- Windows B1 updates and exact-ID pushes; and
- Windows-to-Mac resume and backup evidence.

`WINDOWS-RC6-W2-READY` was not emitted.

### W3 — NOT TESTED

No intended Windows/Mac divergence, conflict exit 6, conflict record,
metadata-only inspection, or `--keep-both` resolution was executed.

## 6. GitHub integrity evidence

Live GitHub check-run evidence for release commit `9019bd9`:

```text
check_runs=11
success=10
skipped=1
```

Successful jobs included Windows, macOS, and Ubuntu tests; lint; security;
website; CodeQL; workflow policy/pin review; dependency review; and secret
scan. `Build & release` was skipped.

The mandatory row is scored `PARTIAL`: the required test/security jobs were
green, but the report does not relabel a skipped check as green.

## 7. Mandatory section 19 reconciliation

| # | Gate | Result | Evidence |
| ---: | --- | --- | --- |
| 1 | `install.sh` returns 200 and installs RC6 on Mac | PASS | Device A report section 2.2 |
| 2 | `install.ps1` returns 200 and installs RC6 on Windows | PASS | This report section 3.2 |
| 3 | Both installers are idempotent and PATH-safe | PASS | Mac installer added no duplicate; Windows normalized count 1 |
| 4 | Pre-init missing-config failure is accurate | PASS | Exit 3 with supported device/adapters on both devices |
| 5 | Post-init setup check and self-test pass on both devices | PASS | Device A section 3.2; Windows section 4.2 |
| 6 | Claude setup prompt completes on the Mac | PASS | Device A section 3.2 |
| 7 | Codex setup prompt completes on Windows | PARTIAL | Behavioral workflow passed; human hidden-prompt contract replaced by autonomous provider |
| 8 | Only two selected test sessions reach the remote manifest | PASS | Exact two remote keys |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | Device A section 3.5 |
| 10 | Wrong passphrase fails without mutation | PASS | Windows section 4.4 |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | Pull/discovery passed; exact vendor resume stopped before execution |
| 12 | Codex Mac-to-Windows resume succeeds | FAIL | Interactive command unproven; autonomous fallback changed local baseline |
| 13 | Active-agent overwrite is refused | NOT TESTED | W2 blocked |
| 14 | Existing Windows target is backed up before restore | NOT TESTED | W2 blocked |
| 15 | Claude Windows-to-Mac resume succeeds | NOT TESTED | W2 blocked |
| 16 | Codex Windows-to-Mac resume succeeds | NOT TESTED | W2 blocked |
| 17 | Existing Mac targets are backed up before restore | NOT TESTED | W2 blocked |
| 18 | Unchanged pushes skip without new snapshots | NOT TESTED | W2/M3 blocked |
| 19 | Divergence records a conflict without overwrite | NOT TESTED | W3 blocked |
| 20 | `--keep-both` preserves both branches | NOT TESTED | W3 blocked |
| 21 | All required GitHub checks are green | PARTIAL | 10 success; `Build & release` skipped |

Totals:

```text
PASS=9
PARTIAL=2
FAIL=1
NOT_TESTED=9
ALL_21_PASSED=false
```

## 8. Findings

### A-RC6-1 — autonomous Codex resume probe contaminated the W1 baseline

Severity: **acceptance blocking; not a confirmed product defect**

The exact restored Codex ID was discoverable and loadable. The non-interactive
`codex exec resume --ephemeral` fallback nevertheless changed the local rollout
file. Reinstate correctly retained its original baseline metadata, so the
local hash no longer matched the recorded post-pull revision.

Continuing to Mac A2 would test a pre-diverged target rather than the required
unchanged baseline. No manual state edit, manual file restoration, push,
conflict resolution, or safeguard bypass was used to manufacture a pass.

A clean rerun must prove the exact interactive resume without sending a
message, close the agent, and preserve the target before M2.

### F-RC6-1 — tag signature is not independently verified by GitHub

Severity: **non-blocking**

GitHub reports the tag object's SSH signature as `unknown_key`. The release
commit verifies as valid, local tag verification succeeds with the exact
tagged allowed-signers file, and the installed binary commit matches the peeled
tag commit.

Registering the SSH signing key on GitHub would remove this ambiguity.

### D1 — RC6 passphrase reused from RC5

Device A recorded this operator-accepted deviation. RC6 uses a fresh profile,
remote prefix, home, sessions, and per-object age+scrypt salts, but RC5 and RC6
are not passphrase-compartmentalized.

### D2 — autonomous local credential file

The operator explicitly authorized a local plaintext credential file to
replace human prompts. It was never added to the repository. Its contents were
not printed, copied into the report, or included in any Git command.

The operator should remove or secure that file after evidence review.

## 9. Sanitized stopping handoff

```text
WINDOWS-RC6-W1-FAIL
release=v0.1.0-rc.6
tag_commit=9019bd9cb4094eae648339dfecb2c6449c1b60d2
profile_id=b0ddbe7b-b4f6-43e4-8194-6043baa6dd61
claude_session_id=499bfb6f-8f84-41c4-bcf3-c1b61d9ad1f3
codex_session_id=019fa308-6f70-7290-8742-80cc89b39f3c
f3_bad_coordinates_refused=PASS
f1_default_refusal=PASS
f2_missing_manifest_refused=PASS
wrong_passphrase_refused=PASS
remote_session_count=2
claude_pull_and_discovery=PASS
claude_resume=NOT_TESTED
codex_pull_and_discovery=PASS
codex_resume=FAIL
failed_command=codex exec resume --ephemeral --json <CODEX_SESSION_ID> <SANITIZED_PROBE>
failed_command_exit=0
failed_observation=exact session loaded but local rollout changed
w2=NOT_TESTED
w3=NOT_TESTED
overall_verdict=FAIL
pass_count=9
partial_count=2
fail_count=1
not_tested_count=9
all_21_passed=false
windows_report_path=docs/testing/results/2026-07-27-windows-phase1-rc6.md
END-WINDOWS-RC6-W1-FAIL
```

Phase 1 RC6 acceptance is **FAIL**. All 21 mandatory rows did not pass.
