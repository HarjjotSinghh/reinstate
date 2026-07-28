# Phase 1 RC7 acceptance — Device B (Windows) report

Milestone verdict: **`WINDOWS-RC7-W1-FAIL`**. W0 and the storage/init/restore
portion of W1 passed. The exact Codex vendor-resume command opened and held the
restored rollout but did not submit or complete its challenge response, and the
available harness had no safe desktop-control surface. Codex resume and the
downstream Claude resume are therefore `BLOCKED` / `NOT TESTED`, never `PASS`.

This is an acceptance-automation blocker, not evidence of a Reinstate product
failure. W2/W3 and tagged runbook §14 were not started.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date/time (UTC) | `2026-07-28T12:28:44Z` |
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

The launcher remains temporarily present because the blocked acceptance-owned
Codex TUI process is still running; removing its parent runner now would make
the eventual exit record less reliable.

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

## 7. Blocking same-vendor resume gate

### Exact command

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
challenge_count_after_80_seconds=0
process_cpu_delta_over_5_seconds=0
desktop_control_available=false
safe_targeted_window_activation=false
exit=NOT_CAPTURED
process_still_running=true
```

The target file becoming exclusively held is consistent with the exact session
opening, but it does not prove challenge-response resume. The harness could not
safely inspect or operate the unknown TUI state. It did not blind-press keys,
take a transcript screenshot, use `codex exec resume --ephemeral`, invoke a
permission bypass, or force-kill any process.

Because exact response evidence is absent, Codex resume is `NOT TESTED`.
Per the stop-on-vendor-resume rule, Claude resume was not attempted and is also
`NOT TESTED`.

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
| 7 | Codex setup prompt completes on Windows | PARTIAL | install/init/status/pull/discovery passed; vendor resume blocked |
| 8 | Only two selected test sessions reach the remote manifest | PASS | Mac report §5.4 and Windows status |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | Mac report §5.5 |
| 10 | Wrong passphrase fails without mutation | PASS | Mac report and §5 above |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | stopped after Codex resume block |
| 12 | Codex Mac-to-Windows resume succeeds | NOT TESTED | §7 — exact response absent |
| 13 | Unrelated running agents do not block a restore | PASS | Mac report §6.1 plus Windows Codex pull with two unrelated processes |
| 14 | A live session is forked, never overwritten | NOT TESTED | Device B §14b not reached |
| 15 | `scoped` policy still refuses, naming that session | PARTIAL | Mac strict-policy evidence only; Windows scoped test outstanding |
| 16 | Existing Windows target is backed up before restore | NOT TESTED | W2 / §14d |
| 17 | Claude Windows-to-Mac resume succeeds | NOT TESTED | later handoff |
| 18 | Codex Windows-to-Mac resume succeeds | NOT TESTED | later handoff |
| 19 | Existing Mac targets are backed up before restore | PASS | Mac report §6.1 |
| 20 | Unchanged pushes skip without new snapshots | PASS | Mac report |
| 21 | Divergence records a conflict without overwrite | NOT TESTED | W3 |
| 22 | `--keep-both` preserves both branches | NOT TESTED | W3 |
| 23 | All required GitHub checks are green | PASS | Mac report §7 |

**Counts: 13 PASS / 2 PARTIAL / 0 FAIL / 8 NOT TESTED.**

All 23 mandatory rows passed: **No.** Phase 1 remains open.

## 10. Findings

Release-blocking Reinstate product findings: **none established on Device B**.

Acceptance blockers:

1. Exact Codex vendor challenge-response could not be completed safely with the
   available harness, leaving both same-vendor resume rows unverified.
2. The first installer output capture exposed one absolute user-local install
   path to private tool output before all-stream redaction was enabled. No
   storage secret, passphrase, private-file value, transcript, or ciphertext
   was exposed.

Highest untested product risk remains tagged runbook §14b: the real-hardware
Windows Restart Manager path has not yet been exercised with the exact target
file held open and asserted to produce one stable fork.

## 11. Repository hygiene

- Branch `test/phase1-rc7-windows-report` starts from the peeled RC7 tag commit.
- `R2.txt` is locally excluded, untracked, and absent from the staged diff.
- The only repository change is this sanitized report.
- No product code, private-file data, transcript, ciphertext, remote object,
  tag, release, deployment, or product branch was committed or pushed.
- The existing draft PR remains unmerged.

## 12. Milestone block

```text
WINDOWS-RC7-W1-FAIL
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
claude_discovery_and_resume=NOT_TESTED
codex_resume=NOT_TESTED
failure_gate=codex_vendor_resume_challenge
failure_exit=NOT_CAPTURED
failure_class=BLOCKED
windows_report_path=docs/testing/results/2026-07-28-windows-phase1-rc7.md
END-WINDOWS-RC7-W1
```
