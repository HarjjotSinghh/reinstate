# Phase 1 acceptance — Device A (macOS) — v0.1.0-rc.5

Sanitized result note for
[`docs/testing/phase-1-mac-windows-acceptance.md`](../phase-1-mac-windows-acceptance.md)
as published at tag `v0.1.0-rc.5`.

This note deliberately contains **no** endpoint, bucket, access key, secret key,
passphrase, keyring value, agent auth material, transcript text, object name,
username, or absolute local path. Only non-secret IDs, counts, booleans,
versions, exit codes, redacted paths, and sanitized output appear here.

**Status: CLOSED. Overall `v0.1.0-rc.5` verdict: FAIL.**

Device A (macOS) completed M0 and M1 with real-device evidence. Device B
returned a release-blocking `WINDOWS-RC5-W1-FAIL`, so milestones M2, M3, and M4
were **not executed** and every gate depending on them is recorded as
`NOT TESTED`, never as `PASS`.

**Tally: 6 PASS / 2 PARTIAL / 2 FAIL / 11 NOT TESTED (21 mandatory rows).**
All 21 mandatory rows passed: **no**.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date | 2026-07-27 |
| Device role | Device A (macOS), evidence owner |
| macOS version | 26.5.2 (build 25F84) |
| Architecture | `arm64` |
| Native shell | `/bin/zsh` |
| Claude Code version | 2.1.220 (recognized range 2.1.219–2.1.220) |
| Codex CLI version | 0.145.0 (recognized range 0.133.0–0.145.0) |
| Git version | 2.55.0 |
| Reinstate version | 0.1.0-rc.5 |
| Release under test | `v0.1.0-rc.5` |
| Tag commit (peeled) | `b4ebd8dcf8b47e7dcbbc0fc40c4ef9adf9ea5065` |
| Reinstate home | `${HOME}/.reinstate` — see deviation **D-1** |
| Project | `${PHASE1_PROJECT}` (isolated RC5 acceptance project) |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc5` |
| Device A profile ID | `451177b7-9eda-4e49-b74d-239773916f77` |
| Claude test session ID | `0ccc78dd-b7cb-44c0-b069-15047089d330` |
| Codex test session ID | `019fa185-78d9-7bb2-8731-7697894de1ba` |

Isolation preconditions verified before any work: neither the isolated home nor
the isolated project existed beforehand, and no RC3/RC4 home, profile prefix,
passphrase, report, or session was read, reused, or mutated. This device has no
RC3 or RC4 Reinstate home at all, so no prior-RC state could be reused.

### Deviation D-1 — acceptance ran against the default home

The runbook and the Device A test plan require an isolated
`~/.reinstate-phase1-acceptance-rc5` home. The acceptance profile was instead
created in the **default** `~/.reinstate` home, and the run continued there by
the operator's explicit decision after the deviation was surfaced.

Cause, established from filesystem metadata rather than assumption: the isolated
home was correctly exported in the terminal used for session creation, but the
Claude Code setup-prompt agent classified `REINSTATE_HOME` as a stray variable,
suppressed it with `env -u REINSTATE_HOME`, and the operator's `rein init` then
targeted the default home.

Blast radius, verified before continuing:

| Question | Answer | How established |
| -------- | ------ | --------------- |
| Did `~/.reinstate` pre-exist? | **No** | directory birth time is inside this run's window |
| Was any RC3/RC4 home reused or mutated? | **No** | no other Reinstate home exists on this device |
| Was any pre-existing config or state overwritten? | **No** | `init` creates; the F1 refusal below proves it will not replace |
| Backups / conflicts present in that home? | 0 / 0 | directory entry counts |
| Isolated home contents | `cache/selftest` only — never initialized | directory walk |

Nothing was destroyed. The deviation is recorded as **DEVIATED**, not PASS, for
the isolation precondition, and Device B remains on its own isolated home, so
the two devices are asymmetric for the remainder of the run.

## 2. M0 — release and environment

### 2.1 Signed tag provenance

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `v0.1.0-rc.5` is an annotated tag | PASS | `git cat-file -t v0.1.0-rc.5` → `tag` |
| Tag object SHA | — | `edc178bdd0e1e4a23757241615fc687d4ca8b055` |
| Peeled commit | — | `b4ebd8dcf8b47e7dcbbc0fc40c4ef9adf9ea5065` |
| Reachable from `origin/main` | PASS | `git merge-base --is-ancestor v0.1.0-rc.5^{commit} origin/main` → exit 0 |
| Remote agrees on tag and peeled commit | PASS | `git ls-remote origin refs/tags/v0.1.0-rc.5{,^{}}` matches local |
| Tag signature verifies | PASS | `Good "git" signature ... ED25519 key SHA256:uXE5rh19MaNxQO/0pgR8n2K73xR+ck0O7fwf351W030` → exit 0 |

Trust configuration was **not** altered to force a pass. The developer's global
Git config has no `gpg.format` and no `gpg.ssh.allowedSignersFile` (both
`git config --get` probes exited 1), so `git verify-tag` initially failed with
`gpg.ssh.allowedSignersFile needs to be configured and exist for ssh signature
verification`. Verification was then performed against the **project's own
committed trust anchor**, `.github/allowed_signers` as it exists *inside the tag
being verified*, supplied transiently via `git -c` with no config written:

```sh
git -c gpg.format=ssh \
    -c gpg.ssh.allowedSignersFile=<tagged .github/allowed_signers> \
    verify-tag v0.1.0-rc.5
```

The signing key was **not** taken from the signature itself, which would have
been circular.

**Non-blocking finding F-A1 (provenance hygiene).** The tag's tagger identity
and the principal listed in `.github/allowed_signers` are not the same string
(they differ by one character in the local part). Git's SSH backend resolves the
principal by *key*, not by tagger e-mail, so a good signature does not assert
that the tagger address is an allow-listed principal. The signature is
genuinely good and the key is genuinely project-allow-listed; only the identity
binding is weaker than it appears. Recommend aligning the release tagger e-mail
with the allow-listed principal, or allow-listing both addresses.

### 2.2 Public installer (`install.sh`)

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `https://reinstate.dev/install.sh` returns 200 | PASS | `curl -fsSI` → `HTTP/2 200`, `content-length: 2398` |
| Served bytes match the tagged website asset | PASS | served file is byte-identical to `website/public/install.sh` at the tag |
| Pins exactly one release, no `latest` | PASS | bootstrap sets `VERSION="v0.1.0-rc.5"`; layer 2 requires exact v-SemVer and prints `Refusing an unpinned latest release.` when unset |
| Layer 1 checksum verified | PASS | `installer checksum ok` |
| Layer 1 pin actually matches the tagged installer | PASS | `PINNED_INSTALLER_SHA256` equals SHA-256 of `scripts/install.sh` at the tag (`7776adb4…db2e66`) |
| Layer 2 checksum verified | PASS | `checksum ok` (asset verified against release `checksums.txt`) |
| Installs 0.1.0-rc.5 | PASS | `rein version --json` → `"version": "0.1.0-rc.5"` |
| Installed build matches the tag | PASS | `rein version --json` → `"commit": "b4ebd8d"`, equal to the peeled tag commit |
| No elevation | PASS | no `sudo`/`doas`/`osascript` in either layer; installs under a user-owned directory; no elevation prompt appeared |
| Idempotent second run | PASS | run 2 → `Reinstate v0.1.0-rc.5 is already installed at <redacted>`, exit 0 |
| Exactly one PATH entry | PASS | interactive and `zsh -l` login shell each report exactly `1` matching PATH entry |
| Shell profile untouched | PASS | `~/.zshrc` SHA-256 and line count identical before and after both runs |

Binaries resolve under the user-local install directory; `rein` is a relative
symlink to `reinstate`.

Third integrity layer observed beyond the two required: the installer refuses
to publish a binary whose own `version --json` does not equal the pinned asset
version.

**Replacement prompt: NOT TESTED (not applicable).** No previous Reinstate
binary existed anywhere on this device before the run, so the interactive
replacement path could not fire, and a same-version re-run returns
`already installed` and exits 0 *before* reaching the confirmation branch. The
prompt was not manufactured by planting an older binary. Static reading of the
tagged installer shows the branch is default-deny (`[y/N]`, bounded 30 s read,
refuses on timeout, on unreadable TTY, and on shells lacking bounded reads).
This is recorded honestly as untested on Device A rather than claimed as PASS.

### 2.3 Pre-init honesty check

```
$ rein setup check          # isolated RC5 home, no config yet
reinstate doctor
version: 0.1.0-rc.5
platform: darwin-arm64
home: ${REINSTATE_HOME}
summary: 1 check(s) failed
- [fail] config: config missing
- [ok] device: darwin-arm64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable

1 check(s) failed
exit=3
```

| Check | Result |
| ----- | ------ |
| Exit code is `3` | PASS |
| Failure reason is `config missing` | PASS |
| Device detection does not report an unsupported platform | PASS (`darwin-arm64`) |
| `agent.claude` not falsely passing an unsupported state | PASS (`SUPPORTED (2.1.220)`, inside recognized range) |
| `agent.codex` not falsely passing an unsupported state | PASS (`SUPPORTED (0.145.0)`, inside recognized range) |
| Isolated home not created as a side effect of a read-only check | PASS |

Neither adapter reported `UNTESTED` or `UNSUPPORTED`, so mutating acceptance is
permitted to proceed.

## 3. M1 — source sessions, init safety, push, ciphertext

Baseline local session metadata captured before creating any acceptance
session, via `rein list --agent all --json`:

| Metric | Value |
| ------ | ----- |
| Local sessions before, total | 6 |
| Local `claude` sessions before | 3 |
| Local `codex` sessions before | 3 |
| Sessions before under the acceptance project | 0 |

Session titles and paths were never printed or recorded; only identifiers and
counts were retained, in a temporary location outside this repository.

### 3.1 Fresh session selection

After creating exactly one harmless session per agent with the mandated
markers, the same metadata listing showed 8 sessions, a delta of exactly two:

| Agent | Fresh session ID | In acceptance project | Own marker occurrences | Foreign marker occurrences |
| ----- | ---------------- | --------------------- | ---------------------- | -------------------------- |
| claude | `0ccc78dd-b7cb-44c0-b069-15047089d330` | yes | 3 | 0 |
| codex | `019fa185-78d9-7bb2-8731-7697894de1ba` | yes | 5 | 0 |

Claude produced **no** sibling candidate for the invocation, so marker-count
disambiguation was not needed; the counts above are recorded only as proof that
each selected session carries its own marker and not the other agent's. No
transcript prose was read, printed, or recorded. Codex **did** persist a
brand-new rollout, so no older acceptance session was reused.

### 3.2 Post-initialization checks

Both run first-hand on Device A:

| Command | Exit | Result |
| ------- | ---- | ------ |
| `rein setup check` | `0` | `all checks passed`; `config valid`, `device darwin-arm64`, `agent.claude SUPPORTED (2.1.220)`, `agent.codex SUPPORTED (0.145.0)`, keyring reachable |
| `rein doctor --self-test` | `0` | `all checks passed`, `self_test: synthetic self-test passed`, `self_test: ok` |

Both selected session IDs remain discoverable after initialization.

### 3.3 F1 default-refusal regression (physical)

`rein init` was re-run with the identical `--project` mapping and **no**
`--force`, against the real RC5 home. Hashes and backup-set counts were
computed in the operator's private terminal; only comparison booleans were
reported back.

```
$ rein init --project "local/reinstate-phase1-acceptance-rc5=<redacted>"
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
F1_exit=7
config_unchanged=true
state_unchanged=true
backups_before=0 backups_after=0
```

| Requirement | Result |
| ----------- | ------ |
| Safety exit code `7` | PASS |
| `config.toml` byte-identical before and after | PASS |
| `state.json` byte-identical before and after | PASS |
| No new backup set created | PASS (0 → 0) |
| Refusal is immediate, with no prompt for any secret | PASS |
| `--force` never run against the real RC5 home | PASS (not executed) |

**F1 default-refusal: PASS.**

### 3.4 Project-mapping asymmetry between adapters

After `init` registered `local/reinstate-phase1-acceptance-rc5`, local metadata
resolves the canonical project ID for one adapter but not the other, for
sessions living in the *same* mapped directory:

| Agent | Sessions in the mapped project | Resolved project ID |
| ----- | ------------------------------ | ------------------- |
| claude | 2 | `local/reinstate-phase1-acceptance-rc5` |
| codex | 1 | raw absolute macOS path, unmapped |

Listing scope also differs: after initialization the Claude listing is scoped to
mapped projects (4 → 2 sessions), while the Codex listing still returns every
discovered session across unrelated directories (4 → 5). See finding **F-A3**.

### 3.5 Dry-run, push, and remote manifest

The Claude session was pushed through the setup-prompt workflow; the Codex
session was pushed directly per runbook section 9. Only exact session IDs were
used, and `--all` was never run.

```
$ rein push --agent codex --session <codex id> --dry-run
would push 1 snapshot(s), would skip 0 unchanged, dry_run=true

$ rein push --agent codex --session <codex id>
pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false

$ rein status
remote revision: 4770adee-1adc-426e-bc18-405c2a112d1b (2 sessions)
  codex:019fa185-78d9-7bb2-8731-7697894de1ba -> 4770adee-1adc-426e-bc18-405c2a112d1b
  claude:0ccc78dd-b7cb-44c0-b069-15047089d330 -> 4d8ed162-08f9-48dd-a24f-42c71c2aede6
```

| Requirement | Result |
| ----------- | ------ |
| Dry-run says `would push`, never `pushed` | PASS |
| Dry-run uploads nothing | PASS (`dry_run=true`, no revision change) |
| Real push reports exactly one snapshot | PASS |
| `rein status` shows exactly two sessions | PASS |
| Both selected IDs present, no unrelated session | PASS |

| Artifact | Non-secret ID |
| -------- | ------------- |
| Remote revision after M1 | `4770adee-1adc-426e-bc18-405c2a112d1b` |
| Claude snapshot | `4d8ed162-08f9-48dd-a24f-42c71c2aede6` |
| Codex snapshot | `4770adee-1adc-426e-bc18-405c2a112d1b` |

`rein status` reports no project ID for either session, so it neither confirms
nor refutes finding **F-A3**; that remains open until the Windows restore in M3.

### 3.6 Ciphertext-only remote storage

The operator inspected only the fresh RC5 profile prefix through the storage
provider's normal UI and reported counts and booleans only. Object names are
deliberately excluded from this report.

| Check | Result |
| ----- | ------ |
| `manifest.age` present | yes |
| `snapshots/*.age` object count | 2 |
| Any object that is not `.age` | none |
| Any `auth`, token, credential, `.env`, or plaintext-shaped object | none |

One `.age` snapshot was downloaded privately, tested without printing any
matching bytes, and deleted:

| Check | Result |
| ----- | ------ |
| `REINSTATE-PHASE1-RC5-MAC-CLAUDE-A1` present in ciphertext | **absent** (`grep` exit 1) |
| `REINSTATE-PHASE1-RC5-MAC-CODEX-A1` present in ciphertext | **absent** (`grep` exit 1) |
| `file` verdict | `data` — not readable text |
| Parses as JSON | `false` |
| age encryption header present | `true` |
| Local download deleted afterwards | `true` |

The encrypted object is larger than the corresponding plaintext session, which
is consistent with authenticated encryption rather than a copied file.

**Ciphertext marker absence: PASS.**

### 3.7 M1 handoff issued

```text
MAC-RC5-M1
release=v0.1.0-rc.5
tag_commit=b4ebd8dcf8b47e7dcbbc0fc40c4ef9adf9ea5065
profile_id=451177b7-9eda-4e49-b74d-239773916f77
canonical_project_id=local/reinstate-phase1-acceptance-rc5
claude_session_id=0ccc78dd-b7cb-44c0-b069-15047089d330
codex_session_id=019fa185-78d9-7bb2-8731-7697894de1ba
remote_session_count=2
f1_default_refusal=PASS
ciphertext_marker_absence=PASS
mac_report_path=docs/testing/results/2026-07-27-macos-phase1-rc5.md
END-MAC-RC5-M1
```

## 4. Device B outcome and stop decision

Device B never reached a usable additional-device initialization on stock RC5.
Its sanitized handoff, reproduced verbatim and unaltered:

```text
WINDOWS-RC5-W1-FAIL
release=v0.1.0-rc.5
tag_commit=b4ebd8dcf8b47e7dcbbc0fc40c4ef9adf9ea5065
device=windows-amd64
canonical_project_id=local/reinstate-phase1-acceptance-rc5
correct_coordinate_additional_device_init=FAIL
failed_operation=HeadObject
failed_http_status=400
failed_error=BadRequest
actual_exit_code=NOT_CAPTURED
config_initialized=false
w1_remainder=NOT_TESTED
w2=NOT_TESTED
w3=NOT_TESTED
post_rc5_pr_26_validation=EXCLUDED
overall_verdict=FAIL
pass_count=1
partial_count=1
fail_count=1
not_tested_count=18
all_21_passed=false
report_path=docs/testing/results/2026-07-27-windows-phase1-rc5.md
END-WINDOWS-RC5-W1-FAIL
```

Windows draft PR: <https://github.com/HarjjotSinghh/reinstate/pull/28>
(open, draft, unmerged; single file — the Windows report).

Notes on how this is scored here:

- `actual_exit_code=NOT_CAPTURED` is reproduced as reported. It is **not**
  inferred, reconstructed, or replaced with a plausible value.
- The Device B counts (`pass_count=1`, `not_tested_count=18`, …) describe Device
  B's own view. They are **not** adopted wholesale. Section 7 reconciles every
  mandatory row individually across both devices, which is why the totals here
  differ.
- Device B reports that a later successful restore used post-RC5 development
  code from PR #26. That work is **excluded** from this acceptance verdict; it
  is not RC5 behaviour and no RC5 gate is credited from it.

Because a same-vendor restore never happened on stock RC5, milestones M2, M3,
and M4 were not executed on Device A either.

## 5. M2, M3, M4 — not executed

| Milestone | Device A work | Status |
| --------- | ------------- | ------ |
| M2 | resume Claude, append `…-MAC-CLAUDE-A2`, dry-run, push that ID | NOT EXECUTED |
| M3 | pull both IDs, prove backups, resume `B1` markers, no-op push | NOT EXECUTED |
| M4 | conflict divergence per runbook section 17, keep-both reconciliation | NOT EXECUTED |

No Mac session was modified after M1, no further push or pull was issued, and
the remote revision remains `4770adee-1adc-426e-bc18-405c2a112d1b` with two
sessions. The `A2`, `B1`, and conflict markers were never created on any device.

## 6. Why a successful `pull` line would not have been enough

Recorded for the RC6 rerun. Phase 1 cannot be called complete on the strength of
a `rein pull` success line. The gates that remain unproven for RC5 are exactly
the ones that distinguish a real continuity layer from a file copier:

- exact-ID vendor discovery at the destination device;
- same-vendor resume through the normal `claude --resume` / `codex resume` UI;
- destination path mapping onto the *other* operating system's project path;
- timestamped backup of an existing target before replacement;
- active-agent overwrite refusal;
- unchanged-push no-op that creates no new remote revision; and
- conflict detection with non-destructive keep-both recovery.

None of these were exercised on RC5.

## 7. Section 19 mandatory sign-off

An unexecuted row is `NOT TESTED` and never `PASS`. Rows owned by Device B are
reconciled from the Windows report in M4.

| # | Gate | Owner | Result | Evidence |
| - | ---- | ----- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC5 on Mac | A | **PASS** | §2.2 |
| 2 | `install.ps1` returns 200 and installs RC5 on Windows | B | **FAIL** | **F-B1** — returns 200, installs 0.1.0-rc.4 |
| 3 | Both installers are idempotent and PATH-safe | A+B | **PARTIAL** | Mac PASS §2.2; Windows public route never installed rc.5, PATH-duplication check NOT TESTED |
| 4 | Pre-init missing-config failure is accurate | A+B | **PASS** | Mac §2.3; Windows exit 3, `config missing`, `windows-amd64`, both adapters `SUPPORTED` |
| 5 | Post-init setup check and self-test pass on both devices | A+B | **PARTIAL** | Mac PASS §3.2; Windows NOT TESTED — `config_initialized=false` |
| 6 | Claude setup prompt completes on the Mac | A | **PASS**, caveat **F-A2** | §3.2, §3.5 |
| 7 | Codex setup prompt completes on Windows | B | **FAIL** | additional-device init failed, HeadObject 400 |
| 8 | Only two selected test sessions reach the remote manifest | A | **PASS** | §3.5 |
| 9 | Remote manifest/snapshots are ciphertext-only | A | **PASS** | §3.6 |
| 10 | Wrong passphrase fails without mutation | B | **NOT TESTED** | blocked by row 7 |
| 11 | Claude Mac-to-Windows resume succeeds | B | **NOT TESTED** | blocked by row 7 |
| 12 | Codex Mac-to-Windows resume succeeds | B | **NOT TESTED** | blocked by row 7 |
| 13 | Active-agent overwrite is refused | B | **NOT TESTED** | blocked by row 7 |
| 14 | Existing Windows target is backed up before restore | B | **NOT TESTED** | blocked by row 7 |
| 15 | Claude Windows-to-Mac resume succeeds | A | **NOT TESTED** | M3 not executed |
| 16 | Codex Windows-to-Mac resume succeeds | A | **NOT TESTED** | M3 not executed |
| 17 | Existing Mac targets are backed up before restore | A | **NOT TESTED** | M3 not executed |
| 18 | Unchanged pushes skip without new snapshots | A | **NOT TESTED** | M3 not executed |
| 19 | Divergence records a conflict without overwrite | A+B | **NOT TESTED** | M4 not executed |
| 20 | `--keep-both` preserves both branches | B | **NOT TESTED** | M4 not executed |
| 21 | All required GitHub checks are green | A | **PASS**, caveat **F-B4** | §7.1 |

**Totals: 6 PASS / 2 PARTIAL / 2 FAIL / 11 NOT TESTED.**
Every mandatory row passed: **no**. Phase 1 therefore remains **open** for
`v0.1.0-rc.5`.

Beyond the 21 mandatory rows, the F1 default-refusal regression was executed
physically on a real RC5 home and **passed** (§3.3).

### 7.1 Required GitHub checks

All check runs on the peeled tag commit
`b4ebd8dcf8b47e7dcbbc0fc40c4ef9adf9ea5065`:

| Conclusion | Checks |
| ---------- | ------ |
| success | Build & release, CodeQL, Lint, Secret scan, Security, Test (ubuntu-latest), Test (macos-latest), Test (windows-latest), Website, Workflow permission and pin review |
| skipped | Dependency review |

10 successful, 1 skipped, 0 failing, so the row passes as written. See **F-B4**
for why green checks did not prevent **F-B1**.

## 8. Findings

| ID | Severity | Summary |
| -- | -------- | ------- |
| F-A1 | Non-blocking | Release tag's tagger e-mail is not the allow-listed signing principal; SSH verification binds by key, not identity (§2.1) |
| F-A2 | Non-blocking, docs/prompt | Setup prompt v5 silently redirected a configured `REINSTATE_HOME` |
| F-A3 | Open, severity pending M3 | Codex sessions are not resolved to the canonical project ID, and adapter listing scope is asymmetric (§3.4) |
| **F-B1** | **RELEASE BLOCKING** | Live public-route mismatch: during acceptance `https://reinstate.dev/install.ps1` served an rc.4 bootstrap, so Windows silently installed 0.1.0-rc.4 while reporting success |
| F-B2 | Non-blocking, Windows-only | The PowerShell replacement prompt omits the target version, so the user approves a replacement without being told what they are getting |
| F-B3 | Minor | `rein status` reports a missing config as a raw OS error containing the absolute config path, instead of the redacted `config missing` that `setup check` produces |
| F-B4 | Process gap | CI verifies the installer assets in the **build output** but nothing verifies the **deployed route**, which is how F-B1 shipped with 11 green checks |
| F-B5 | **RELEASE BLOCKING** (Device B) | Stock RC5 additional-device init fails at the `HeadObject` manifest probe with HTTP 400 against R2 |

### F-B1 — public Windows route did not serve the RC5 asset during acceptance

A live deployment regression / public-route mismatch **observed during
acceptance**. Discovered from Device A by fetching and diffing the live routes,
and reproduced on Device B, where the documented public one-liner installed
0.1.0-rc.4.

This finding records **what was observed**, not why or when it happened.

| Artifact | `$Version` |
| -------- | ---------- |
| `website/public/install.ps1` **at tag `v0.1.0-rc.5`** | `v0.1.0-rc.5` |
| `https://reinstate.dev/install.ps1` **as served during acceptance** | `v0.1.0-rc.4` |
| `website/public/install.sh` at tag vs served | byte-identical, correct |

Byte-level observations, all taken at acceptance time:

- The repository is **correct** at the tag. The POSIX route matched its tagged
  asset exactly, so this is a published-artifact mismatch on the PowerShell
  route, not a source defect.
- The served PowerShell body differed from the RC5-tag asset by **exactly one
  line**, line 5, the `$Version` assignment.
- The served body was **byte-identical to the complete RC4-tag asset**
  (`cmp` clean; SHA-256 `13d9271c…`).
- The RC4-tag and RC5-tag bootstraps themselves differ **only** on that same
  line 5.
- The pinned installer digest `ce46d3a2…` was identical in the served body, the
  RC4-tag asset, and the RC5-tag asset, because `scripts/install.ps1` is
  byte-identical between the two tags.
- It was **not** a CDN caching artifact. A cache-busting query string and an
  explicit `Cache-Control: no-cache` request both returned the same body.

**Cause and timing are deliberately not asserted.** Because the two tags'
bootstraps differ only in line 5, a stale RC4 asset and an RC5 asset whose
version line had been reverted are *the same bytes*. Content alone therefore
cannot distinguish them, and the pinned digest cannot discriminate either, since
it is common to both tags. Establishing whether the route ever served the RC5
asset, and what changed it, requires **deployment-history evidence** that was
not available to this run. Any claim in either direction would be speculation.

Why this is release blocking rather than cosmetic: the rc.4 bootstrap is
*internally consistent*, so both checksum layers legitimately print
`installer checksum ok` and `checksum ok` while installing the wrong release.
A Windows user following the documented public instructions receives rc.4 and is
told the install succeeded. There is no signal that anything is wrong. This is
precisely the failure mode runbook section 18 warns about when it says not to
substitute "the PowerShell looks right" for a real native Windows check.

Fix: republish the website so the `install.ps1` route serves the tagged rc.5
asset, then re-run mandatory row 2 and the Windows half of row 3. No new RC is
required, because no binary behaviour changed — only the published bootstrap.

Device B was unblocked for the remaining gates using the installer's own
documented exact-tag audit path with the layer-1 digest verified by hand. That
workaround does **not** convert row 2 to PASS: the gate tests the public route,
and the public route installs the wrong release.

### F-B2 — PowerShell replacement prompt hides the target version

Exercised on Device B, where an existing rc.4 binary made the confirmation
branch reachable. Observed prompt:

```
Replace Reinstate 0.1.0-rc.4 with  [y/N]:
```

The target version is missing, and so is the literal `?`. Source line 49 of
`scripts/install.ps1` reads:

```powershell
$answer = Read-Host "Replace Reinstate $ExistingVersion with $AssetVersion? [y/N]"
```

Because `?` is a legal character in an unbraced PowerShell variable name, the
token parses as the undefined variable `$AssetVersion?` and expands to the empty
string, consuming the question mark. Line 27 of the same file uses the braced
form `${AssetVersion}` and is correct; line 49 is the only unbraced use followed
by `?`. The POSIX installer builds the same message with `printf '%s'` and is
unaffected, so this is Windows-only.

Fix: brace the expansion, `${AssetVersion}?`.

The safeguard itself is intact — an empty answer was correctly refused with
`refusing to replace existing Reinstate 0.1.0-rc.4`, confirming default-deny.
Rated non-blocking for that reason, but it interacts badly with **F-B1**: a
Windows user already on rc.5 who runs the documented public one-liner is asked
`Replace Reinstate 0.1.0-rc.5 with  [y/N]`, and answering `y` silently
**downgrades** them to rc.4 with no version shown at any point. Both should be
fixed together.

### Replacement-prompt coverage

| Installer | Replacement prompt | Result |
| --------- | ------------------ | ------ |
| POSIX `scripts/install.sh` (Device A) | NOT TESTED | no older binary existed; not manufactured |
| PowerShell `scripts/install.ps1` (Device B) | TESTED, both branches | see below; message defective, see F-B2 |

Device B exercised the confirmation branch twice against a genuine pre-existing
rc.4 binary:

| Answer | Outcome |
| ------ | ------- |
| empty (Enter) | refused: `refusing to replace existing Reinstate 0.1.0-rc.4`, nothing replaced |
| `y` | replaced; `rein version --json` then reported `0.1.0-rc.5` / commit `b4ebd8d` |

Default-deny and explicit-approve both behave correctly. Device B therefore runs
the release under test for all remaining gates, via the exact-tag audit path,
not via the public route affected by F-B1.

### F-A2 — setup prompt v5 does not preserve a configured home

`docs/prompts/claude-code-setup.md` (Prompt version 5, pinned to `v0.1.0-rc.5`)
instructs the agent to detect OS, architecture, shell, WSL, Claude Code version,
and existing binaries, and to report the install path. It never mentions
`REINSTATE_HOME`. Step 5 tells the agent to stop if *the home* is already
initialized, but gives it no way to know **which** home the user intends.

In this run a compliant agent therefore treated an intentionally exported
`REINSTATE_HOME` as environmental noise and removed it with `env -u`, silently
relocating the user's initialization target to the default home. For a product
whose isolation story depends on that variable, a shipped setup prompt that
discards it is a real contract gap.

Suggested fix: have the prompt echo `REINSTATE_HOME`, report which home will be
initialized, require confirmation before proceeding, and forbid unsetting it.

### F-A3 — Codex project mapping and listing scope

Evidence in §3.4. Severity is deliberately left open: the local metadata gap
only becomes release-blocking if the unmapped raw path travels to the remote and
breaks Windows destination mapping. That is decided by M3, not by inspection, so
it is not scored here.

### F-B4 — CI validates the build, not the deployment

The `Website` job in `.github/workflows/ci.yml` at the tag does verify installer
parity, and does it correctly:

```yaml
test -s dist/client/install.sh
test -s dist/client/install.ps1
cmp public/install.sh dist/client/install.sh
cmp public/install.ps1 dist/client/install.ps1
grep -F 'v0.1.0-rc.5' dist/client/install.sh
grep -F 'v0.1.0-rc.5' dist/client/install.ps1
```

That is the runbook section 18 gate for byte-for-byte inclusion of both scripts
in the Astro output, and it passed. Yet during acceptance the live route served the rc.4
PowerShell bootstrap. The check asserts things about `dist/client/` in the CI
workspace; **nothing asserts anything about what `https://reinstate.dev`
actually serves**. So the release could be simultaneously green and broken, and
was.

This is the process gap behind F-B1: the source was right and CI was right, yet
the artifact actually served by the public route was not, and no gate is
positioned to notice that class of divergence — whenever or however it arises.

Suggested fix: add a post-deploy smoke check that fetches both public routes and
asserts the pinned version, for example
`curl -fsSL https://reinstate.dev/install.ps1 | grep -F 'v0.1.0-rc.5'`, and
require it before a release is considered published. Without it, F-B1 can recur
on RC6.

### F-B5 — stock RC5 additional-device init fails against R2

Reported by Device B; not independently reproducible from Device A, which is a
first device and never performs the additional-device manifest probe.

| Field | Value |
| ----- | ----- |
| Operation | `HeadObject` |
| HTTP status | 400 |
| Error | `BadRequest` |
| Exit code | `NOT_CAPTURED` — reported as such, not inferred |
| Result | `config_initialized=false` |

This is the blocker that ended the run: without a second initialized device,
every cross-device gate is unreachable. Device A's own push path against the
same bucket and profile worked, so the failure is specific to the
additional-device manifest probe rather than to storage reachability or
credentials in general.

### Release-blocking summary

| ID | Blocker |
| -- | ------- |
| F-B5 | Stock RC5 cannot initialize an additional device against R2 |
| F-B1 | The public Windows route did not serve the RC5 asset during acceptance; it installs rc.4 |

Non-blocking: F-A1, F-A2, F-A3 (unresolved, could not be decided), F-B2, F-B3.
Process: F-B4.

## 9. Recommendation

`v0.1.0-rc.5` does not pass Phase 1 acceptance. The next candidate should be
RC6 containing the F-B5 fix, and it must ship with:

1. the public Windows route confirmed to serve the tagged RC6 bootstrap,
   verified against the live route rather than the repository;
2. a post-deploy route check so F-B4 cannot hide a repeat of F-B1;
3. `${AssetVersion}` braced in the PowerShell replacement prompt (F-B2); and
4. `REINSTATE_HOME` preserved and echoed by both setup prompts (F-A2).

The full runbook must then be rerun from section 5, because no cross-device gate
has ever been satisfied on any RC5 build. F-A3 in particular is still undecided
and needs the Windows Codex restore to settle it.
