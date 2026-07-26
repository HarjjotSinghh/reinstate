# Phase 1 RC5 blocker-fix wake-up handoff

**Date:** 2026-07-26
**Updated:** 2026-07-27
**Status:** **READY FOR HUMAN REVIEW — NOT RELEASED, NOT PHASE-1 STABLE**
**Base:** `v0.1.0-rc.4` / `f0b96006df6dee24c6d5c8d8fea5c34a655c4aff`
**Branch:** `fix/rc5-acceptance-blockers`
**F1-F3 implementation:** `3f2cbf42aeb75311860f5586b300af609a0be585`
**N1 implementation:** `bd60340`
**N1 regression hardening:** `4a161ad`
**Verification acceleration:** `101192d`
**Execution plan:** `6104589`

No commit was pushed. No tag, release, PR, deployment, published installer,
real acceptance home, real agent session, keychain entry, or remote object was
created, changed, or deleted.

## Executive summary

The three RC4 release blockers are fixed locally with regression coverage:

| Finding | Reviewed behavior |
| --- | --- |
| F1 — `rein init` silently overwrote initialized config/state | Default re-init refuses with safety exit `7`. Explicit `--force` creates one timestamped backup set containing the old `config.toml` and `state.json`, reports its relative `backups/...` location, then replaces them. |
| F2 — missing `manifest.age` looked like a healthy empty profile | A genuinely new first-device profile may report 0 sessions before its first push. Joined profiles and first-device profiles with prior sync state fail with auth/storage exit `4` if the manifest disappears. `status`, `diff`, `pull`, and `push` share this policy. |
| F3 — `init --profile-id` accepted storage coordinates with no profile | Additional-device init performs a metadata-only manifest existence probe before saving config. Missing manifest exits `4` and writes no `config.toml`. Joined configs persist `remote_profile_required = true`. |

The Windows RC4 failure remains correctly classified as an operator endpoint
paste error exposed by F2/F3, not a sync/decryption defect. The supplied
endpoint included `/<bucket>` even though the bucket was also configured
separately.

The remaining N1 installer finding is also fixed locally: replacement
confirmation now has a validated 30-second default timeout, supports a bounded
1-300 second override, and refuses immediately when the active shell cannot
perform timed reads. An idle pseudo-TTY regression test exercises the canonical
installer and proves the existing binary is preserved.

Known actionable local product blockers after these commits: **0**. The live
GitHub issue tracker also returned zero open issues on 2026-07-27. This does not
make Phase 1 stable by itself: RC5 does not exist yet, and the full physical
Mac/Windows acceptance run remains unexecuted.

## Verification evidence

### Verification latency — materially reduced

The original release gate repeated documentation and fixture packages, then
ran production-strength scrypt throughout high-level tests under the race
detector. The optimized gate:

- runs documentation and fixture contracts once through `go test ./...`;
- keeps one production-default crypto round trip in the ordinary full suite;
- uses real age envelopes with work factor 1 only in high-level `_test.go`
  fixtures;
- race-instruments product packages while omitting only the stateless crypto
  wrapper and subprocess/document contracts; and
- provides a cached `make quick` edit-loop gate.

Measured in the same heavily contended session:

| Gate | Before | After |
| --- | ---: | ---: |
| Race gate wall time | `312.31s` | `140.46s` |
| Complete `make verify` | still running after 11 minutes | `162.94s` PASS |
| Warm `make quick` | did not exist | `9.97s` PASS |

The final verify used `36.23s` user CPU and `25.62s` system CPU; the larger wall
time came from a host load average around 37 with 21 logged-in sessions. No user
process was killed to manufacture a prettier benchmark.

`make quick` is not release-equivalent. It intentionally uses Go's cache and
omits production-KDF, race, lint, vulnerability, and subprocess/document work.
`make verify` remains the required merge/release gate.

### Final canonical gate — PASS

Run fresh after all blocker fixes and verification acceleration:

```text
make verify
```

Observed:

- `gofmt` check: PASS
- `go vet ./...`: PASS
- pinned `golangci-lint v2.11.4`: `0 issues`
- `go test ./... -count=1`: PASS
- filtered product-package race gate: PASS
- pinned `govulncheck v1.6.0`: 0 reachable vulnerabilities
- documentation and fixture secret-scan contracts through the full suite: PASS
- final CGO-disabled build: PASS
- wall time: `162.94s`

`govulncheck` reported one vulnerable module in the dependency graph but no
affected symbol reachable from Reinstate.

### Release target cross-builds — PASS

All targets in `.goreleaser.yml` compiled with Go `1.25.12` and `CGO_ENABLED=0`:

```text
darwin/amd64  PASS
darwin/arm64  PASS
windows/amd64 PASS
linux/amd64   PASS
linux/arm64   PASS
```

These are compile gates, not substitutes for physical-device acceptance.

### Native macOS acceptance-style blocker checks — PASS

Executed through the compiled `rein` binary on native `Darwin/arm64` using
temporary isolated homes, the disk-backed memory backend, fake credentials,
and a synthetic passphrase provided through a file descriptor:

```text
F1_default_reinit_exit=7 config_state_unchanged=true
F1_force_exit=0 backup_sets=1 backup_hashes_match=true location_reported=true
F2_new_profile_status_exit=0 sessions=0
F2_established_profile_missing_manifest_exit=4 message_match=true
F3_join_missing_manifest_exit=4 config_written=false message_match=true
temporary_synthetic_state_removed=true
```

The binary was confirmed as a native Mach-O arm64 executable. The test did not
read or mutate `~/.reinstate*`, `~/.claude`, `~/.codex`, Keychain, R2, or any
real session. Temporary synthetic state was removed after the checks.

### POSIX installer idle-TTY blocker check — PASS

The canonical installer ran under a real native pseudo-terminal whose input
pipe remained open but sent no bytes, reproducing N1's unattended readable-TTY
condition:

```text
Bash timed read (configured 1s)       PASS — bounded refusal
Dash without timed-read support       PASS — immediate refusal
invalid timeout: invalid, 0, 301, 1000 PASS — rejected
existing installed binary             PASS — preserved on every refusal
```

The pre-fix regression test exceeded its 3-second deadline while visibly
waiting at the replacement prompt. After the fix it completes within the
configured bound and prints the explicit refusal path. `dash -n`, `bash -n`,
and `zsh -n` accepted the script. `shellcheck` was unavailable locally; the
canonical Go installer contract, lint, and native shell executions are green.

### TDD evidence

Each changed behavior was observed failing before implementation:

- F1 default re-init returned `0`, not `7`.
- F1 `--force` was an unknown flag and returned `2`.
- F2 strict fetch returned no error for a missing manifest.
- F3 additional-device init returned `0` and wrote config.
- new first-device status later exposed by review returned `4`, not `0`.
- `diff` returned generic exit `1`, not auth/storage exit `4`.
- forced init did not report the backup set location.
- the idle pseudo-TTY installer exceeded its deadline before the N1 fix;
- an invalid confirmation timeout was ignored before the N1 fix; and
- the first process-exit-based pseudo-TTY assertion later failed under load,
  correctly exposing a test-harness assumption. The final assertion observes
  bounded behavior while input remains open and survived three consecutive
  runs plus a race-instrumented run.

Focused and full suites passed after each minimal correction.

## Independent Claude Code review

One bounded reviewer ran through Claude Code `2.1.220` with:

- non-interactive `claude -p`;
- high effort and a hard USD budget;
- no session persistence;
- no browser/network access;
- edits, writes, subagents, and permission bypass disallowed; and
- repository reads plus focused `git`/Go test commands only.

The reviewer initially returned `CHANGES_REQUIRED`. Independent triage:

### Accepted and fixed

- Differentiate a genuinely new first-device profile from joined/established
  profiles when the remote manifest is absent.
- Map `diff` manifest failures to stable auth/storage exit `4`.
- Report the relative forced-init backup set location.
- Add coverage for new, joined, and established manifest policy; RC4 config
  compatibility; the exported error sentinel; and last-manifest state.

### Not accepted as defects

- `state.LastManifestRev` storing the pushed snapshot ID is intentional:
  `manifest.Revision` is explicitly advanced to the latest snapshot ID. The
  end-to-end test now locks that invariant.
- Preserving the old keyring credential during forced re-init is necessary for
  recovery: the timestamped backup retains the old `credential_ref`. Deleting
  that credential would make the backup incomplete.
- A metadata-only `Head` probe cannot prove decryption without asking for the
  passphrase during `init`; its contract is to catch wrong
  endpoint/bucket/prefix coordinates before config is saved.

Claude's attempted `git fetch` was denied by the read-only tool policy. Review
used the existing local `origin/main`, which was the RC4 release commit.

## Release status and remaining gates

**Phase 1 still fails release acceptance.** Local code gates and synthetic Mac
checks are green; the real RC5 artifacts and two-device run do not exist yet.

Required before calling Phase 1 stable:

1. Human-review commits `3f2cbf4`, `bd60340`, `4a161ad`, and `101192d`.
2. Integrate the reviewed fixes onto protected `main` and require all remote CI
   checks green.
3. Prepare an explicit `v0.1.0-rc.5` release commit:
   - move `[Unreleased]` entries into `[0.1.0-rc.5]`;
   - update compatibility/release evidence;
   - pin public bootstraps, manual docs, and prompt version 5 to RC5;
   - run the GoReleaser snapshot and installer contract gates in
     `RELEASING.md`.
4. With explicit approval, create a signed RC5 tag and draft release; inspect
   archives, checksums, SBOMs, attestations, and exact installers before
   publication.
5. Run the complete physical-device acceptance on a **fresh RC5** profile,
   isolated homes, disposable projects, and newly selected Claude/Codex
   sessions. Do not reuse RC3/RC4 homes or manifests.
6. Require all 21 mandatory Mac/native-Windows rows to pass, plus the WSL2 and
   release preconditions in `RELEASING.md`. In particular, re-prove:
   - correct and wrong storage coordinates during Device B init;
   - wrong-passphrase refusal without mutation;
   - both same-vendor Mac-to-Windows resumes;
   - active-agent overwrite refusal and backup behavior;
   - Windows-to-Mac updates;
   - unchanged-push skip;
   - divergence/conflict and `--keep-both`;
   - ciphertext-only remote objects; and
   - installer idempotency/PATH safety on both physical devices.

Do not resume RC4 M2/W2/W3 and relabel it RC5. A new candidate requires a new,
clean evidence chain.

## Morning review commands

```bash
cd /Users/harjjotsinghh/Documents/Projects/reinstate-rc5-fixes
git status --short --branch
git log --oneline -8
git show --stat --oneline 3f2cbf4
git show --stat --oneline bd60340
git show --stat --oneline 101192d
git diff origin/main...HEAD
```

Expected final prepared state: branch ahead of `origin/main` by seven local
commits, clean, with nothing pushed.

## Decisions needed from GOAT

1. Approve or reject commits `3f2cbf4`, `bd60340`, `4a161ad`, and `101192d`.
2. If approved, explicitly authorize push/PR/main integration and RC5 release
   preparation. Tagging and publication should remain a separate approval.
