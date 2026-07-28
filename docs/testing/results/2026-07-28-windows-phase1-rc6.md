# Reinstate Phase 1 RC6 — Windows Device B acceptance

**Report date:** 2026-07-28
**Evidence owner:** native Windows Device B
**Release:** `v0.1.0-rc.6`
**Peeled tag commit:** `9019bd9cb4094eae648339dfecb2c6449c1b60d2`
**Report branch:** `test/phase1-rc6-windows-report`
**Verdict:** **PARTIAL — W1 executed autonomously; W2/W3 not tested**

Windows release provenance, public installation, initialization, F3/F1/F2
refusals, wrong-passphrase refusal, exact two-session status, scoped restore,
path remapping, local discovery, and process-level exact-ID resume all behaved
correctly.

No Reinstate product release blocker was found on Device B through W1.

This run cannot honestly emit `WINDOWS-RC6-W1-PASS`: the operator explicitly
required fully autonomous execution with no human present. Storage credentials
therefore used Reinstate's documented non-interactive environment provider,
the encryption passphrase used an inherited anonymous-pipe handle, and nobody
visually confirmed the two A1 transcript markers. Those method deviations make
the Windows setup-prompt gate and both Mac-to-Windows resume gates `PARTIAL`.

W2 and W3 are `NOT TESTED`, never `PASS`.

No Reinstate product code was modified. This report is the only cumulative
repository change from the RC6 tag on this branch.

## 1. Scope and safety boundary

The run operated only on:

- profile `fd182697-957a-421f-8ee0-b45c18bf61a7`;
- Claude session `0eb4f696-c513-4bd8-8b80-8d9a8b964718`;
- Codex session `019fa608-ec57-7071-b6be-d8047004bbc9`; and
- canonical project `local/reinstate-phase1-acceptance-rc6`.

`--all`, `--force`, manual restored-file relocation, transcript inspection,
and remote deletion were never used.

The prior contaminated Windows RC6 home, project, and incomplete launcher home
were preserved under timestamped retired/aborted names. Nothing was deleted.
The fresh run began with the exact required home, probe home, and project paths
absent.

The Windows credential file arrived in the warned six-key pre-normalized form,
not the five-key form described by Device A. A local helper:

- validated key names without printing values;
- preserved the original Windows copy;
- selected the RC6-specific passphrase field exactly as directed by Device A;
- rewrote the working copy to one passphrase field; and
- reran a strict five-key/nonempty validator successfully.

No credential or passphrase value was printed, logged, committed, hashed for
disclosure, or placed in a command argument. Captured child-process text was
reduced in memory to result booleans and then discarded. Access key, secret
key, and passphrase value detectors found no match. A bucket-coordinate
detector matched ordinary product/path text by substring; the raw coordinate
and raw child output were never emitted.

## 2. Environment

| Field | Value |
| ----- | ----- |
| Windows product | Microsoft Windows 11 Pro |
| Windows version/build | 10.0.26200 / 26200 |
| OS architecture | 64-bit |
| PowerShell | Desktop 5.1.26100.8328 |
| PowerShell executable | native `System32\WindowsPowerShell\v1.0\powershell.exe` |
| PowerShell process | 64-bit, non-elevated |
| Claude Code | 2.1.220 (`SUPPORTED`) |
| Codex CLI | 0.145.0 (`SUPPORTED`) |
| Git | 2.52.0.windows.1 |
| Reinstate | 0.1.0-rc.6, commit `9019bd9`, built 2026-07-27T09:09:11Z |
| Effective isolated home | `%USERPROFILE%\.reinstate-phase1-acceptance-rc6` |
| Disposable project | `%USERPROFILE%\Projects\reinstate-phase1-acceptance-rc6` |

The disposable project was a fresh Git repository containing only `README.md`
before initialization.

## 3. W0 — release, installer, and pre-init

### 3.1 Release provenance

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Annotated tag | PASS | `git cat-file -t v0.1.0-rc.6` returned `tag` |
| Peeled commit | PASS | Exact commit `9019bd9cb4094eae648339dfecb2c6449c1b60d2` |
| Reachable from `origin/main` | PASS | `git merge-base --is-ancestor` exited 0 |
| Tag-matching allowed-signers blob | PASS | Tagged and working blobs both `5397329e...` |
| Local tag verification | PASS | Command-scoped SSH allowed-signers verification exited 0 |

No global Git trust configuration was changed.

### 3.2 Public Windows installer

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `https://reinstate.dev/install.ps1` | PASS | HTTP 200 |
| Exact pin | PASS | `v0.1.0-rc.6`; zero other RC literals |
| No `latest` resolver | PASS | No `latest` token in live bootstrap |
| Bootstrap equals tagged file | PASS | Git blob IDs matched |
| Canonical URL | PASS | Exact RC6 tagged `scripts/install.ps1` URL |
| Canonical checksum | PASS | Live SHA-256 matched pinned checksum |
| Canonical file equals tag | PASS | Git blob IDs matched |
| Release archive checksum | PASS | Installer emitted both checksum confirmations |
| Installed version | PASS | Both aliases reported 0.1.0-rc.6 / `9019bd9` |
| No elevation | PASS | Native non-elevated PowerShell |
| Idempotent rerun | PASS | Two runs reported exact RC6 already installed |
| PATH safety | PASS | Normalized user PATH count remained exactly 1 |

Both aliases resolved under:

```text
%LOCALAPPDATA%\Programs\Reinstate\bin
```

### 3.3 Pre-init honesty

```text
exit=3
config_missing=true
device_windows_amd64=true
claude_supported=true
codex_supported=true
keyring_reachable=true
home_created=false
```

Result: **PASS**.

## 4. W1 — regressions and Mac-to-Windows restore

### 4.1 F3 malformed-coordinate refusal

The endpoint was programmatically changed to include the bucket suffix while
the bucket remained separately configured. Values were never emitted.

```text
execution_mode=documented non-interactive credential provider
exit=4
remote_profile_manifest_not_found=true
config_exists=false
state_exists=false
backup_count=0
secret_value_in_output=false
```

Behavioral result: **PASS**. Physical hidden-prompt method: **PARTIAL** due the
explicit autonomous-execution instruction.

### 4.2 Correct initialization and health

```text
init_exit=0
expected_profile=true
config_created=true
state_created=true
setup_check_exit=0
setup_all_passed=true
claude_supported=true
codex_supported=true
doctor_self_test_exit=0
doctor_self_test_passed=true
```

Behavioral result: **PASS**.

### 4.3 F1 default refusal

The same correct initialization was repeated without `--force`.

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

A generated incorrect passphrase was sent through a one-use inherited
anonymous-pipe handle.

```text
exit=4
decryption_refused=true
config_unchanged=true
state_unchanged=true
backups_before=0
backups_after=0
secret_value_in_output=false
```

No restore target existed before this test and none was created.

Result: **PASS**.

### 4.5 Correct status

```text
exit=0
revision_nonempty=true
remote_session_count=2
exact_expected_claude_id=true
exact_expected_codex_id=true
secret_value_in_output=false
```

Result: **PASS**.

### 4.6 F2 missing-manifest strict status

The real home was copied to the required
`-missing-manifest-probe` path. Exactly one profile-prefix occurrence was
rewritten to a nonexistent prefix; config contents were never printed.

```text
probe_exit=4
remote_profile_manifest_not_found=true
probe_kept=true
real_config_unchanged=true
real_state_unchanged=true
real_backups_before=0
real_backups_after=0
secret_value_in_output=false
```

Final correct status still returned the same non-empty revision and exact two
session IDs.

Result: **PASS**.

### 4.7 Scoped dry-runs

Before dry-run:

```text
codex_exact_target_count=0
claude_exact_target_count=0
backup_count=0
```

| Agent | Exit | Required wording | Mutation |
| ----- | ---: | ---------------- | -------- |
| Codex | 0 | `would pull 1 snapshot(s)`; never `pulled` | none |
| Claude | 0 | `would pull 1 snapshot(s)`; never `pulled` | none |

After dry-run, both target counts and backup count remained zero.

Result: **PASS**.

### 4.8 Scoped pulls, mapping, and local discovery

The coordinating Codex process and unrelated Claude processes remained open
because the operator required an unattended run and preserving active sessions
was mandatory. RC6 permits new-session restores while vendor processes are
open; both exact targets were new. No existing target was overwritten.

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Codex pull | PASS | Exit 0; pulled exactly one snapshot |
| Claude pull | PASS | Exit 0; pulled exactly one snapshot |
| New-target backups | PASS | Zero before and after |
| Codex destination | PASS | One `sessions/2026/07/28/...<ID>.jsonl` target |
| Claude destination | PASS | One target under the mapped Windows project key |
| Manual relocation | PASS | None |
| Codex local discovery | PASS | Exact ID and canonical project present |
| Claude local discovery | PASS | Exact ID and canonical project present |

The Claude destination key was:

```text
C--Users-admin-Projects-reinstate-phase1-acceptance-rc6/<CLAUDE_SESSION_ID>.jsonl
```

### 4.9 Exact-ID vendor resume

No prompt or marker text was sent to either vendor.

For Codex, the literal command was launched:

```text
codex resume <CODEX_SESSION_ID> --no-alt-screen
```

Observed:

```text
exact_id_argument=true
tui_still_running_after_8_seconds=true
dedicated_process_tree_terminated=true
restored_rollout_unchanged=true
prompt_sent=false
```

For Claude, the literal command was launched through its Windows command shim:

```text
claude --resume <CLAUDE_SESSION_ID>
```

Observed:

```text
exact_id_argument=true
claude_process_descendant_present=true
tui_still_running_after_8_seconds=true
dedicated_process_tree_terminated=true
restored_session_unchanged=true
prompt_sent=false
```

An initial Claude launcher selected the PowerShell script's file association
and opened it in Notepad instead of executing it. That accidental process was
identified by exact PID/command line and closed; it touched no session file.
The retry used the real command shim and produced the evidence above.

These results prove exact-ID command acceptance, correct vendor process launch,
and an unchanged restored baseline. They do not prove the human-only visual A1
marker check.

Results:

- Claude Mac-to-Windows resume: **PARTIAL**.
- Codex Mac-to-Windows resume: **PARTIAL**.

## 5. W2 and W3

### W2 — NOT TESTED

W2 requires `MAC-RC6-M2-READY`, an intentionally active selected Claude
session, A2 visual confirmation, Windows B1 updates, and a new Mac handoff.
Those prerequisites do not exist in an unattended Windows-only run.

`WINDOWS-RC6-W2-READY` was not emitted.

### W3 — NOT TESTED

No intended cross-device divergence, conflict exit 6, conflict record,
metadata-only inspection, or `--keep-both` resolution was executed.

## 6. GitHub and local integrity evidence

Live GitHub check runs for release commit `9019bd9`:

```text
check_runs=11
success=10
skipped=1
failure=0
nonterminal=0
```

`Build & release`, CodeQL, lint, secret scan, security, all three OS test jobs,
website, and workflow permission/pin review succeeded. `Dependency review`
was skipped because this was a push event, matching Device A's documented
expected condition.

Local native-Windows regression:

```text
go vet ./... = exit 0
go test <all packages except internal/doctest> = exit 0 (23 packages)
go test ./... = exit 1
```

The full test failure is isolated to two `internal/doctest`
repository-policy tests invoking `make -n verify` / `make -n quick`; native
Windows has no `make` in PATH. Product packages passed. This is test-harness
portability debt, not an RC6 runtime failure.

## 7. Mandatory sign-off checklist

| # | Gate | Result | Evidence |
| --: | ---- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC6 on Mac | PASS | Device A report §2.2 |
| 2 | `install.ps1` returns 200 and installs RC6 on Windows | PASS | This report §3.2 |
| 3 | Both installers are idempotent and PATH-safe | PASS | Device A §2.2; this report §3.2 |
| 4 | Pre-init missing-config failure is accurate | PASS | Device A §3.1; this report §3.3 |
| 5 | Post-init setup check and self-test pass on both devices | PASS | Device A §3.2; this report §4.2 |
| 6 | Claude setup prompt completes on the Mac | PASS | Device A §3.2 |
| 7 | Codex setup prompt completes on Windows | PARTIAL | Behavioral workflow passed; hidden-prompt method replaced by documented automation provider |
| 8 | Only two selected test sessions reach the remote manifest | PASS | Exact two expected IDs |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | Device A §3.5 |
| 10 | Wrong passphrase fails without mutation | PASS | This report §4.4 |
| 11 | Claude Mac-to-Windows resume succeeds | PARTIAL | Exact TUI/process launch and unchanged file; no visual A1 confirmation |
| 12 | Codex Mac-to-Windows resume succeeds | PARTIAL | Exact TUI launch and unchanged file; no visual A1 confirmation |
| 13 | Active-agent overwrite is refused | NOT TESTED | W2 prerequisite absent |
| 14 | Existing Windows target is backed up before restore | NOT TESTED | W2 prerequisite absent |
| 15 | Claude Windows-to-Mac resume succeeds | NOT TESTED | W2 |
| 16 | Codex Windows-to-Mac resume succeeds | NOT TESTED | W2 |
| 17 | Existing Mac targets are backed up before restore | NOT TESTED | W2 |
| 18 | Unchanged pushes skip without new snapshots | NOT TESTED | W2/M3 |
| 19 | Divergence records a conflict without overwrite | NOT TESTED | W3 |
| 20 | `--keep-both` preserves both branches | NOT TESTED | W3 |
| 21 | All required GitHub checks are green | PASS | Device A §6; this report §6 |

Counts:

```text
PASS=10
PARTIAL=3
FAIL=0
NOT_TESTED=8
ALL_21_PASSED=false
```

Phase 1 remains open.

## 8. Findings and deviations

### A-RC6-1 — human-only W1 evidence unavailable

Severity: **acceptance incomplete; not a confirmed product defect**

Exact-ID vendor processes loaded without early exit and without changing either
restored file. A human was explicitly unavailable, so hidden-prompt execution
and visual transcript-marker confirmation could not occur. Do not relabel
these partial gates as pass.

### T-RC6-1 — native-Windows doctests assume `make`

Severity: **non-blocking test-harness portability debt**

`go test ./...` fails two repository-policy doctests because `make` is not
installed on stock native Windows. The cross-platform CI job and every
non-doctest package passed. A future reviewable test-only change should either
skip these policy tests when `make` is unavailable or test the Makefile parser
without requiring the executable.

### D1 — autonomous credential provider

The operator explicitly required fully autonomous execution. Storage
credentials used Reinstate's documented environment fallback only in child
processes. The encryption passphrase used a one-use inherited anonymous-pipe
handle, never an argument or ordinary environment variable.

### D2 — coordinating vendor processes remained open

One coordinating Codex process and unrelated Claude processes remained open.
They were not killed. The restores created new exact targets, which RC6 allows
while vendor processes are present. This does not cover the W2 existing-target
active-agent refusal.

### D3 — Windows credential file required normalization

The Desktop copy was the warned original six-key file, not Device A's
normalized five-key file. The Windows working copy was normalized locally,
the original was preserved, and the RC6-specific passphrase was selected
exactly as the Device A handoff directed.

## 9. Sanitized stopping handoff

```text
WINDOWS-RC6-W1-PARTIAL
release=v0.1.0-rc.6
tag_commit=9019bd9cb4094eae648339dfecb2c6449c1b60d2
profile_id=fd182697-957a-421f-8ee0-b45c18bf61a7
claude_session_id=0eb4f696-c513-4bd8-8b80-8d9a8b964718
codex_session_id=019fa608-ec57-7071-b6be-d8047004bbc9
f3_bad_coordinates_refused=PASS
f1_default_refusal=PASS
f2_missing_manifest_refused=PASS
wrong_passphrase_refused=PASS
remote_session_count=2
claude_pull_and_discovery=PASS
claude_process_level_resume=PASS
claude_visual_marker_confirmation=NOT_TESTED
codex_pull_and_discovery=PASS
codex_process_level_resume=PASS
codex_visual_marker_confirmation=NOT_TESTED
w2=NOT_TESTED
w3=NOT_TESTED
overall_verdict=PARTIAL
pass_count=10
partial_count=3
fail_count=0
not_tested_count=8
all_21_passed=false
windows_report_path=docs/testing/results/2026-07-28-windows-phase1-rc6.md
END-WINDOWS-RC6-W1-PARTIAL
```
