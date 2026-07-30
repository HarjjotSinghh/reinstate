# Phase 1 RC7 acceptance — Device B (Windows) report

Milestone verdict: **`WINDOWS-RC7-W2-FAIL`**. W0 and W1 remain passed. Device A
M2 commit `22207c43…` and its `MAC-RC7-M2-READY` block validated completely,
but the first mandatory W2 gate (§14b) did not create the required live-session
fork. With Claude open on the exact session under the default `fork` policy,
the pull exited `6`, recorded a divergence conflict, and created no
`-active-<short>` session.

The canonical target, config, state, and existing backup count were unchanged,
so no unsafe overwrite occurred. Per the operator's explicit stop rule, §14c,
§14d, both B1 appends/pushes, and W3 were not attempted.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date/time (UTC) | `2026-07-28T12:28:44Z` |
| W1 retry completed (UTC) | `2026-07-28T13:03:57Z` |
| W2 §14b attempt (UTC) | `2026-07-29T05:39:27Z` |
| Release under test | `v0.1.0-rc.7` |
| Tag commit | `66211599dd7cfb74f1436d2221b983050e8b1bc2` |
| Windows edition/build | Windows Pro, display version `25H2`, build `26200.8328` |
| Architecture | native 64-bit Windows and 64-bit process |
| Shell | Windows PowerShell `5.1.26100.8328` |
| Claude Code version | `2.1.220` (`SUPPORTED`) |
| Codex CLI version | `0.145.0` (`SUPPORTED`) |
| Git version | `2.52.0.windows.1` |
| Reinstate version | `0.1.0-rc.7` (commit `6621159`) |
| Device A profile ID | `2949d464-03f4-4de1-b326-2b3072bcb2a5` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc7` |
| Claude test session ID | `1cf4ab6d-3e36-424d-8f30-4f41858b7f20` |
| Codex test session ID | `019fa82a-2b87-71d2-947d-a8146d3049fd` |
| Remote revision | `4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574` |

No RC6 home, project, profile, passphrase, marker session, probe, or report was
used as RC7 behavioral evidence. The RC6 checkout remains clean and untouched.

## 2. Private-file retry

The first supplied home file was valid UTF-8 with five non-empty `key=value`
lines, but zero names matched the RC7 contract. That attempt stopped before W0
and was recorded in the first revision of this report.

The replacement placed on Desktop was validated without emitting field names or
values:

```text
regular_file=true
reparse_point=false
utf8_valid=true
line_count=5
required_key_count=5
unknown_key_count=0
duplicate_key_count=0
empty_value_count=0
endpoint_https=true
endpoint_bucket_separate=true
```

The rejected home copy was recoverably archived. The validated replacement was
copied to the exact required home location; the source was preserved, the two
files are byte-identical, and both use owner-only ACLs. One rejected-file
archive remains owner-only. No value was printed, exported to the parent,
placed in argv, committed, or uploaded.

The ephemeral launcher:

- parsed the private file locally at runtime;
- supplied storage coordinates and credentials only in each Reinstate child's
  environment;
- supplied the passphrase only through an inheritable anonymous pipe handle
  named by `REINSTATE_PASSPHRASE_FD`;
- generated the wrong passphrase in memory;
- redacted all five private values, user-local paths, usernames, and remote
  object shapes before returning child output; and
- left all storage/passphrase variables absent from the parent environment.

The launcher remains temporarily present because the acceptance-owned Codex
TUI process is still running after recording its exact response and
`task_complete`; removing its parent runner now would make the eventual exit
record less reliable.

## 3. Release provenance and installer (W0)

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Tag object is annotated | PASS | `git cat-file -t` → `tag` |
| Signature verifies | PASS | good SSH signature against tag-matching committed allowed-signers blob |
| Tag commit reachable from `origin/main` | PASS | ancestor check exit `0` |
| Live `install.ps1` reachable | PASS | HTTP `200`, 4889 bytes |
| Live bootstrap matches signed tag | PASS | exact Git blob match |
| Pins only RC7 | PASS | one RC7 token, zero other RC tokens, zero `latest` tokens |
| Canonical stage-2 URL | PASS | exact-tag raw GitHub installer URL |
| Stage-2 checksum layer | PASS | live stage-2 matches tag and pinned SHA-256 |
| Release-asset checksum layer | PASS | installer printed `checksum ok` |
| Exact release asset base | PASS | RC7 GitHub release path, bootstrap clears overrides |
| Unelevated install | PASS | process was not elevated; no elevation prompt |
| Installed aliases | PASS | both aliases resolve under the expected user-local install directory |
| Exact installed version | PASS | both aliases report RC7 commit `6621159` |
| Idempotent rerun | PASS | reports RC7 already installed |
| User PATH | PASS | exactly one normalized install-directory entry |

### Evidence-capture deviation

The first installer execution used a sanitizer that captured stdout/stderr but
not PowerShell's information stream. The installer therefore emitted the
user-local install path into the private tool output before the sanitizer could
redact it. No endpoint, bucket, access key, secret key, passphrase, private-file
value, transcript, ciphertext, or remote object was exposed. The idempotency
run captured all streams and returned only a redacted path. The report and Git
history contain no absolute local path or username.

## 4. Honest pre-init and successful additional-device init

Pre-init `rein setup check`:

```text
summary: 1 check(s) failed
- [fail] config: config missing
- [ok] device: windows-amd64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable
exit=3
```

Physical F3 used the endpoint with the bucket suffix while retaining the
separate bucket:

```text
remote profile manifest not found at configured storage coordinates
exit=4
```

Exact isolated-home checks after F3: no `config.toml`, no `state.json`, and
zero backup files.

Correct endpoint-only init then passed with the exact Mac profile ID and
canonical Windows mapping:

| Check | Result |
| ----- | ------ |
| Correct init exit | `0` |
| Config/state created | yes |
| Profile ID | exact match |
| Canonical project ID | exact match |
| Local root | exact Windows project path after TOML decoding |
| Default restore policy | `fork` |
| Post-init setup check | exit `0`, all checks passed |
| `doctor --self-test` | exit `0`, synthetic self-test passed |

## 5. F1, wrong-passphrase, and F2 safety

### Physical F1

```text
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
exit=7
```

`config.toml` and `state.json` were byte-identical before/after; backup count
remained `0 → 0`.

### Wrong passphrase

```text
decrypt: identity did not match any of the recipients: incorrect identity for recipient block: incorrect passphrase
exit=4
```

Config/state were unchanged, exact restore-target count stayed zero, and backup
count remained `0 → 0`. Correct-passphrase status immediately afterward showed
exactly the two Mac sessions and remote revision `4ffdb7ea-...`.

### Physical F2

A fresh copied home had exactly one profile-prefix line changed to a
missing-manifest probe suffix:

```text
sync: remote profile manifest not found at configured storage coordinates
exit=4
```

The real home config/state remained byte-identical. Correct status afterward
returned the same two-session remote revision. Exact targets and backups were
still zero before restore.

## 6. Scoped Mac-to-Windows restore

Two unrelated Codex processes were alive and were not terminated. Neither exact
target existed locally, so neither process could hold a target file.

| Command | Result | No-mutation/target evidence |
| ------- | ------ | --------------------------- |
| Codex exact-ID dry-run | PASS — `would pull 1`, exit `0` | zero targets, zero backups |
| Claude exact-ID dry-run | PASS — `would pull 1`, exit `0` | zero targets, zero backups |
| Codex exact-ID pull | PASS — `pulled 1`, exit `0` | one date-partitioned target |
| Claude exact-ID pull | PASS — `pulled 1`, exit `0` | one target under exact Windows project key |

No timestamped backup was expected or created because both targets were new.

Exact-marker-only checks, with no surrounding transcript read or returned:

| Target | A1 marker occurrences |
| ------ | --------------------- |
| Codex | `5` |
| Claude | `4` |

`rein list --json` discovers exactly one matching ID per agent. Both report the
canonical project ID; Codex reports a date-partitioned rollout path; Claude's
vendor directory equals the adapter's exact Windows key and is not the Mac
source key.

## 7. Same-vendor resume gates

### Codex: PASS

Exact command:

```text
codex resume --no-alt-screen 019fa82a-2b87-71d2-947d-a8146d3049fd "Reply with exactly: REINSTATE-PHASE1-RC7-WINDOWS-CODEX-RESUME-OK"
```

Sanitized observations:

```text
baseline_A1_count=5
baseline_challenge_count=0
terminal_window_opened=true
exact_resume_process_count=1
exact_rollout_size_changed=true
exact_rollout_exclusively_held=true
matching_challenge_record_count=5
assistant_role_record_count=1
assistant_response_exact_match=true
task_complete_record_present=true
exit=NOT_CAPTURED
process_still_running=true
```

The five exact-marker records are one user message, one user-message event, one
agent-message event, one assistant-role response, and one task-complete event.
The assistant response contains one text item exactly equal to the 44-byte
challenge marker and no additional text. No surrounding transcript prose was
read or returned.

The response appeared after the terminal/runtime restart. Computer Use still
had no native control pipe, but no UI action was needed to establish the
result. The harness did not blind-press keys, take a transcript screenshot, use
`codex exec resume --ephemeral`, invoke a permission bypass, or force-kill any
process.

### Claude: PASS after clean retry

The first exact command failed before an assistant response:

```text
claude --resume 1cf4ab6d-3e36-424d-8f30-4f41858b7f20 --print "Reply with exactly: REINSTATE-PHASE1-RC7-WINDOWS-CLAUDE-RESUME-OK"
```

Sanitized result:

```text
exit=1
error_class=authentication_failed
error=Failed to authenticate: OAuth session expired and could not be refreshed
exact_target_count=1
A1_marker_count_before=4
A1_marker_count_after=4
challenge_marker_record_count=3
assistant_challenge_response_count=0
```

The exact restored session was discovered and updated with the requested
challenge, but the vendor could not produce an assistant response because its
local authentication had expired. The three challenge occurrences are
queue-operation, user-message, and last-prompt records; none is an assistant
response. This attempt is not used as successful same-vendor resume evidence.

After the operator completed `/login`, `claude auth status` exited `0`, reported
an authenticated session, and contained no expired-token error. The failed
challenge had locally diverged the restored target, so the retry did not append
another prompt to that contaminated file.

Clean-retry recovery:

```text
dry_run_pull_exit=6
dry_run_target_unchanged=true
dry_run_backup_count=0
mutating_pull_exit=6
conflict_record_count=1
conflict_agent_session_project_exact=true
conflict_local_revision_matches_target=true
conflict_remote_snapshot_matches_Mac=true
pre_resolution_target_unchanged=true
pre_resolution_state_unchanged=true
pre_resolution_backup_count=0
keep_remote_exit=0
restored_target_A1_count=4
restored_target_failed_challenge_count=0
backup_count=1
backup_A1_count=4
backup_failed_challenge_count=3
active_conflict_count=0
resolved_audit_count=1
```

The signed RC7 implementation deliberately detects divergence during dry-run
without recording a conflict. The mutating pull then recorded one exact
retry-only conflict without touching target, state, or backups.
`conflicts resolve --keep-remote` backed up the failed-auth target and restored
the pristine remote snapshot. This recovery is retry hygiene only; it is not
claimed as the later W3 marker/`--keep-both` gate or tagged runbook §14d.

Successful exact command:

```text
claude --resume 1cf4ab6d-3e36-424d-8f30-4f41858b7f20 --print "Reply with exactly: REINSTATE-PHASE1-RC7-WINDOWS-CLAUDE-RESUME-RETRY-OK"
```

Sanitized result:

```text
exit=0
stdout_line_count=1
stdout_exact_response_count=1
target_A1_marker_count=4
old_failed_challenge_count=0
retry_marker_record_count=5
retry_user_role_record_count=1
retry_assistant_role_record_count=1
exact_assistant_response_count=1
assistant_api_error_count=0
```

No surrounding transcript prose was read or returned. With both exact vendor
resume challenges proven, W1 passes. No dependent W2/W3 or §14 gate was
attempted.

## 8. Discarded evidence attempts

The following bad assertions were excluded and replaced with valid checks:

1. The first F3 post-check collided with PowerShell's read-only `$HOME`
   variable and inspected the wrong location. The F3 command evidence was kept;
   exact isolated-home checks were rerun successfully.
2. The first wrong-passphrase target counter had an array-parentheses bug.
   Config/state/backup evidence was kept; exact metadata counts were rerun.
3. The first Claude project-key predicate rejected any key containing `Users`.
   It was replaced with the adapter's exact non-alphanumeric-to-dash transform.
4. The first `rein list --json` assertions used snake_case property names.
   Discovery was rerun against the actual PascalCase fields.
5. The first retry-conflict predicate compared the launcher-redacted canonical
   project ID and therefore found zero matches. The owner-local conflict JSON
   was then checked directly and matched the exact agent, session, canonical
   project, local hash, and remote snapshot. No secret value was returned.

None of these discarded assertions is used as PASS evidence.

## 9. Mandatory sign-off checklist (all 23 rows)

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC7 on Mac | PASS | Mac report §3 |
| 2 | `install.ps1` returns 200 and installs RC7 on Windows | PASS | §3 |
| 3 | Both installers are idempotent and PATH-safe | PASS | Mac report §3 and §3 above |
| 4 | Pre-init missing-config failure is accurate | PASS | Mac report §4 and §4 above |
| 5 | Post-init setup check and self-test pass on both devices | PASS | Mac report §5.2 and §4 above |
| 6 | Claude setup prompt completes on the Mac | PASS | Mac report §5.2 |
| 7 | Codex setup prompt completes on Windows | PASS | install/init/status/pull/discovery and exact Codex resume passed |
| 8 | Only two selected test sessions reach the remote manifest | PASS | Mac report §5.4 and Windows status |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | Mac report §5.5 |
| 10 | Wrong passphrase fails without mutation | PASS | Mac report and §5 above |
| 11 | Claude Mac-to-Windows resume succeeds | PASS | §7 — clean retry produced one exact assistant response with A1 preserved |
| 12 | Codex Mac-to-Windows resume succeeds | PASS | §7 — exact assistant response plus task-complete evidence |
| 13 | Unrelated running agents do not block a restore | PASS | Mac report §6.1 plus Windows Codex pull with two unrelated processes |
| 14 | A live session is forked, never overwritten | FAIL | §13 — exact Claude process open; pull exited 6 with no fork and no overwrite |
| 15 | `scoped` policy still refuses, naming that session | PARTIAL | Mac strict-policy evidence only; Windows scoped test outstanding |
| 16 | Existing Windows target is backed up before restore | NOT TESTED | W2 / §14d; retry-hygiene backup is explicitly excluded |
| 17 | Claude Windows-to-Mac resume succeeds | NOT TESTED | later handoff |
| 18 | Codex Windows-to-Mac resume succeeds | NOT TESTED | later handoff |
| 19 | Existing Mac targets are backed up before restore | PASS | Mac report §6.1 |
| 20 | Unchanged pushes skip without new snapshots | PASS | Mac report |
| 21 | Divergence records a conflict without overwrite | NOT TESTED | W3 |
| 22 | `--keep-both` preserves both branches | NOT TESTED | W3 |
| 23 | All required GitHub checks are green | PASS | Mac report §7 |

**Counts: 16 PASS / 1 PARTIAL / 1 FAIL / 5 NOT TESTED.**

All 23 mandatory rows passed: **No.** Phase 1 remains open.

## 10. Findings

Release-blocking acceptance finding: **§14b did not create the required
live-session fork on real Windows hardware.**

Remaining acceptance work:

1. W1 is complete, but W2 stopped at §14b. §14c, §14d, B1 pushes, later
   Mac/Windows handoffs, and W3 remain unexecuted.
2. The first installer output capture exposed one absolute user-local install
   path to private tool output before all-stream redaction was enabled. No
   storage secret, passphrase, private-file value, transcript, or ciphertext
   was exposed.

The data-safety guard behaved correctly: local divergence produced exit `6`
and one conflict record without modifying the canonical target, config, state,
or backups. A product data-loss defect is not established. The required
Restart Manager fork behavior remains unproven because the live Claude process
did not retain an open handle to the target in this launch surface.

## 11. Repository hygiene

- Branch `test/phase1-rc7-windows-report` starts from the peeled RC7 tag commit.
- `R2.txt` is locally excluded, untracked, and absent from the staged diff.
- The only repository change is this sanitized report.
- No product code, private-file data, transcript, ciphertext, remote object,
  tag, release, deployment, or product branch was committed or pushed.
- The existing draft PR remains unmerged.

## 12. Milestone block

```text
WINDOWS-RC7-W1-PASS
release=v0.1.0-rc.7
tag_commit=66211599dd7cfb74f1436d2221b983050e8b1bc2
profile_id=2949d464-03f4-4de1-b326-2b3072bcb2a5
claude_session_id=1cf4ab6d-3e36-424d-8f30-4f41858b7f20
codex_session_id=019fa82a-2b87-71d2-947d-a8146d3049fd
f3_bad_coordinates_refused=PASS
f1_default_refusal=PASS
f2_missing_manifest_refused=PASS
wrong_passphrase_refused=PASS
remote_session_count=2
claude_discovery_and_resume=PASS
codex_resume=PASS
windows_report_path=docs/testing/results/2026-07-28-windows-phase1-rc7.md
END-WINDOWS-RC7-W1
```

## 13. W2 attempt — §14b live-session fork failure

### 13.1 Mac M2 handoff validation

Origin was fetched and commit
`22207c43c53f421dbc8c9c4b4d9ea1518fb81bfc` was both the requested commit and
the exact tip of `origin/test/phase1-rc7-macos-report`. Section 11 and the
complete `MAC-RC7-M2-READY` block validated:

| Check | Result |
| ----- | ------ |
| Release/profile/exact Claude ID | PASS |
| Windows W1 commit `82c324ce…` validated by Device A | PASS |
| Mac A1/A2 occurrence counts | `4 / 4` |
| New Claude snapshot and remote revision | `3e789e6e-b00b-492e-97a5-f0836d115dab` |
| Codex snapshot unchanged | `4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574` |
| Remote sessions / objects | `2 / 4` |
| Ciphertext marker absence | PASS |
| Forbidden private-field assignments in Mac report | `0` |

Windows status independently returned the same M2 remote revision and exactly
two sessions before §14b.

### 13.2 Exact live-session setup

An acceptance-owned hidden native PowerShell console launched:

```text
claude --resume 1cf4ab6d-3e36-424d-8f30-4f41858b7f20
```

Before pull:

```text
exact_claude_process_count=1
active_agent_policy=fork
canonical_A1_count=4
canonical_A2_count=0
existing_backup_count=1
active_conflict_count=0
active_fork_count=0
```

The process stayed alive and its command line named only the exact session.
No transcript text or screenshot was inspected. A direct exclusive-open probe
against the canonical JSONL succeeded, showing that this launch surface did
not retain a continuous target-file handle before the product test.

### 13.3 Exact §14b command and failure

```text
rein pull --agent claude --session 1cf4ab6d-3e36-424d-8f30-4f41858b7f20
```

Sanitized result:

```text
local session diverged; conflict recorded
exit=6
```

Required fork result:

```text
pulled_1=false
reported_in_use=false
reported_fork_id=NONE
fork_count_before=0
fork_count_after=0
```

No-mutation proof:

```text
canonical_hash_unchanged=true
canonical_A1_count=4
canonical_A2_count=0
config_hash_unchanged=true
state_hash_unchanged=true
backup_count=1 -> 1
exact_claude_process_count_after=1
```

The pull created exactly one conflict,
`c-1785303426934481400`. Its owner-local metadata matches the exact Claude
agent, session ID, canonical project ID, current local target hash, and M2
remote snapshot. The conflict remains active and the live acceptance-owned
Claude process remains open to preserve failure evidence.

The signed RC7 implementation treats an available Restart Manager result with
zero holding PIDs as scoped and inactive; it then reaches divergence
protection. The observed exit `6`, zero fork, successful pre-pull exclusive
open, and exact live process are consistent with that boundary. This does not
establish a Restart Manager API failure, but it does fail the required §14b
outcome.

No fork existed to delete. Per the operator's instruction, no attempt was made
to manufacture a handle, resolve the conflict, switch to scoped policy,
perform §14d, append either B1 marker, or push either session.

```text
WINDOWS-RC7-W2-FAIL
release=v0.1.0-rc.7
profile_id=2949d464-03f4-4de1-b326-2b3072bcb2a5
claude_session_id=1cf4ab6d-3e36-424d-8f30-4f41858b7f20
mac_m2_commit=22207c43c53f421dbc8c9c4b4d9ea1518fb81bfc
mac_m2_remote_revision=3e789e6e-b00b-492e-97a5-f0836d115dab
failure_gate=section_14b_live_session_fork
failure_exit=6
failure_output=local session diverged; conflict recorded
active_agent_policy=fork
exact_claude_process_count=1
pre_pull_exclusive_open_succeeded=true
restart_manager_fork_created=false
canonical_target_unchanged=true
backup_count_before=1
backup_count_after=1
active_conflict_count=1
failure_class=LIVE_TARGET_HANDLE_NOT_ENUMERATED
windows_report_path=docs/testing/results/2026-07-28-windows-phase1-rc7.md
END-WINDOWS-RC7-W2-FAIL
```
