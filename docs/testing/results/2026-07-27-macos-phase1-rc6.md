# Phase 1 macOS acceptance results — v0.1.0-rc.6

**Report date:** 2026-07-27
**Device:** A (macOS)
**Release under test:** `v0.1.0-rc.6`
**Runbook authority:** `docs/testing/phase-1-mac-windows-acceptance.md` @ `v0.1.0-rc.6`
**Status:** IN PROGRESS — M0 and M1 complete, M2–M4 pending Windows coordination

## 1. Environment

| Field | Value |
| ----- | ----- |
| Date | 2026-07-27 (UTC report date) |
| macOS version | 26.5.2 (build 25F84) |
| Architecture | arm64 |
| Native shell | `/bin/zsh` |
| Claude Code version | 2.1.220 (recognized range 2.1.219–2.1.220) |
| Codex CLI version | 0.145.0 (recognized range 0.133.0–0.145.0) |
| Git version | 2.55.0 |
| `rein version --json` | `0.1.0-rc.6`, commit `9019bd9`, built `2026-07-27T09:09:11Z` |
| Isolated home | `~/.reinstate-phase1-acceptance-rc6` |
| Disposable project | `~/Projects/reinstate-phase1-acceptance-rc6` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc6` |
| Device A profile ID | `b0ddbe7b-b4f6-43e4-8194-6043baa6dd61` |
| Claude test session ID | `499bfb6f-8f84-41c4-bcf3-c1b61d9ad1f3` |
| Codex test session ID | `019fa308-6f70-7290-8742-80cc89b39f3c` |

Both isolated paths were confirmed absent before the run.

## 2. Milestone M0 — release and environment

### 2.1 Signed tag

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `v0.1.0-rc.6` is an annotated tag | PASS | `git cat-file -t v0.1.0-rc.6` → `tag` |
| Peeled tag commit | PASS | `9019bd9cb4094eae648339dfecb2c6449c1b60d2` |
| Reachable from `origin/main` | PASS | ancestor of `origin/main` = `0253462fc455fd6a9812739c34b87b4d9a356190` |
| Release commit signature | PASS | GitHub API `verification.verified=true`, `reason=valid` |
| Tag object signature | PARTIAL | Tag carries an SSH signature, but GitHub reports `verified=false`, `reason=unknown_key`; local `git verify-tag` cannot complete without an allowed-signers file |

Trust configuration was **not** modified to force a pass. `gpg.format` and
`gpg.ssh.allowedSignersFile` were unset locally before and after the check.

The signed release commit verifies independently. The tag object's own SSH
signing key is not registered as a signing key on the GitHub account, so no
available authority can verify the tag itself. Recorded as PARTIAL rather than
PASS. Non-blocking for installer behavior; see findings.

### 2.2 Public installer (`https://reinstate.dev/install.sh`)

| Check | Result | Evidence |
| ----- | ------ | -------- |
| HTTP 200 | PASS | `HTTP/2 200`, `content-type: text/plain`, 2398 bytes |
| Pins only rc.6 | PASS | `VERSION="v0.1.0-rc.6"`; no `latest`/`main`/`refs/heads` reference |
| No unpinned resolution | PASS | canonical installer aborts with `Refusing an unpinned latest release.` |
| Checksum layer 1 | PASS | pinned `7776adb4ace8aa333745cd3f3e42b3a10d1400b9394c612d065c20a739db2e66` equals the live tagged installer and the tagged repo copy byte-for-byte |
| Checksum layer 2 | PASS | canonical installer fetches `checksums.txt`, compares artifact SHA256, aborts on mismatch; live run printed `checksum ok` |
| No elevation | PASS | zero `sudo`/`doas`/admin references; no elevation prompt appeared |
| Installs 0.1.0-rc.6 | PASS | `rein version --json` → `0.1.0-rc.6`, commit `9019bd9` (identical to the tag commit) |
| Replacement prompt exercised | PASS | interactive: `Replace Reinstate 0.1.0-rc.5 with 0.1.0-rc.6? [y/N] y` → `install1_exit=0` |
| Idempotent re-run | PASS | second run: `Reinstate v0.1.0-rc.6 is already installed`, `install2_exit=0` |
| Binaries resolve under `~/.local/bin` | PASS | `rein` and `reinstate` both resolve there; `rein` is a symlink to `reinstate` |
| Installer PATH safety | PASS | installer appended **zero** profile lines (`# Reinstate CLI` marker count = 0) |

**PATH note.** `echo $PATH` contains `~/.local/bin` twice. This is a
pre-existing user-environment condition: the directory is exported once in
`~/.zshrc` and once in `~/.zprofile`, neither carrying the installer's
`# Reinstate CLI` marker. The installer created no duplicate and modified no
profile. Attributable installer duplication: none.

### 2.3 Pre-init honesty

`rein setup check` against the uninitialized isolated home:

```
summary: 1 check(s) failed
- [fail] config: config missing
- [ok] device: darwin-arm64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable
setup_check_exit=3
```

PASS. Exit 3 with `config missing`. Device detection and both adapters
correctly report supported states rather than falsely passing or falsely
failing.

## 3. Milestone M1 — sessions, init safety, push, ciphertext

### 3.1 Source session creation

Before/after metadata diff via `rein list`:

| Agent | Before | After | New entries |
| ----- | ------ | ----- | ----------- |
| claude | 7 | 8 | 1 |
| codex | 6 | 7 | 1 |

Exactly one new session per agent, both under the rc6 acceptance project. No
sibling-candidate ambiguity (1 candidate file in the rc6 project directory), so
marker-count disambiguation was not required; occurrence counts were still
checked and are non-zero for both. Codex persisted a brand-new rollout, so no
stop condition applied. The pre-existing RC5 codex session count was verified
unchanged before and after.

Transcript prose was never read, opened, printed, or committed. Only exact
marker occurrence counts were computed.

### 3.2 Setup prompt (Prompt version 6)

The tagged `docs/prompts/claude-code-setup.md` (Prompt version 6) was executed
by a separate Claude Code session launched from the acceptance terminal, per
runbook section 8.

| Requirement | Result | Evidence |
| ----------- | ------ | -------- |
| Detects and reports exact `REINSTATE_HOME` before any `rein` command | PASS | reported `~/.reinstate-phase1-acceptance-rc6`, flagged that `~/.reinstate` exists but is not the effective home, and required explicit confirmation |
| Never unsets, redirects, or falls back | PASS | passed the value explicitly to every command; `~/.reinstate` timestamps unchanged |
| Reports the five-condition bootstrap contract | PASS | all five true; findings matched Device A's independent verification exactly, including both SHA256 values |
| Does not choose `--all` | PASS | one explicitly selected session |
| Never requests secrets in chat | PASS | credentials and passphrase entered only at hidden prompts |
| No safeguard bypassed | PASS | no `REINSTATE_CONFIRM_REPLACE`, `--force`, `--yes`, or passphrase FD |
| Post-init `rein setup check` | PASS | all checks passed |
| `rein doctor --self-test` | PASS | synthetic storage round-trip passed |
| Claude dry-run | PASS | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true` |
| Claude push | PASS | `pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false` |

Observed side effect (non-blocking, outside repo and isolated home): the setup
session wrote 3 memory files and `MEMORY.md` into that project's agent memory
directory.

### 3.3 F1 physical default-refusal regression

`rein init --project local/reinstate-phase1-acceptance-rc6=<project>` re-run
without `--force` against the initialized RC6 home:

```
home=~/.reinstate-phase1-acceptance-rc6
baseline_ok=true
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
f1_exit=7
config_unchanged=true
state_unchanged=true
backups_before=0 backups_after=0
```

**PASS.** Exit 7 (`ExitSafety`). `config.toml` and `state.json` SHA256 values
identical before and after; no new backup set. `--force` was never run against
the RC6 home.

Source corroboration from the signed tag: `internal/cli/commands_impl.go:67`
performs the existing-files guard *before* any credential prompting, and
`internal/exitcode/codes.go` defines `Safety = 7`.

Only equality booleans and counts were recorded; no hash values left the
private terminal.

**Invalidated first attempt (disclosed).** An earlier execution of this test ran
in the wrong shell, where `REINSTATE_HOME` and `PHASE1_PROJECT` were unset. It
hashed nonexistent `/config.toml` and `/state.json`, so its `unchanged=true`
result compared two empty strings and was vacuous. That attempt is discarded and
is not counted as evidence. It caused no mutation: `init` refused with exit 7,
both pushes matched nothing, `rein status` is read-only, and the RC5 home's
`config.toml` (08:40) and `state.json` (09:11) timestamps remained pre-session.
The retest above added an explicit terminal guard and a `baseline_ok` assertion
that refuses to report a comparison against empty hashes.

### 3.4 Push and manifest scope

| Step | Result | Evidence |
| ---- | ------ | -------- |
| Codex dry-run | PASS | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true`, exit 0 |
| Codex push | PASS | `pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false`, exit 0 |
| Manifest scope | PASS | `rein status` → remote revision `9a1ae885-33a4-4672-b8ce-6679281a4207`, exactly 2 sessions |

Manifest contents:

```
claude:499bfb6f-8f84-41c4-bcf3-c1b61d9ad1f3 -> bfeb1241-6abc-4887-a2f5-f0a390074584
codex:019fa308-6f70-7290-8742-80cc89b39f3c -> 9a1ae885-33a4-4672-b8ce-6679281a4207
```

Both are the selected RC6 IDs. No RC5 session ID appears in this profile's
manifest, and no unrelated local session was uploaded. `--all` was never used.

### 3.5 Ciphertext-only remote storage

Inspected only the fresh RC6 prefix `profiles/b0ddbe7b-b4f6-43e4-8194-6043baa6dd61/`.

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Expected layout only | PASS | `manifest.age` (817 B) plus `snapshots/` containing exactly 2 objects |
| No credential-shaped object | PASS | no `auth.json`, token, credential, or `.env` object present |
| Object content type | PASS | all objects `application/octet-stream` |
| Claude marker absent | PASS | `claude_marker_exit=1` |
| Codex marker absent | PASS | `codex_marker_exit=1` |
| Not readable text | PASS | `file` → `data` (not JSON/JSONL) |

Only the two marker-absence booleans and the non-text boolean were recorded. No
ciphertext bytes were read, printed, or committed. The local download was
deleted after the check; the remote object was not deleted.

## 4. Deviations and findings

### D1 — RC6 passphrase reused from RC5 (deviation, accepted by operator)

Prompt 1 requires a brand-new RC6 profile **and passphrase**. The profile ID,
isolated home, remote prefix, and both marker sessions are genuinely fresh, but
the encryption passphrase was reused from the RC5 run. This was detected when a
misdirected `rein status` decrypted the RC5 manifest, and confirmed with the
operator, who elected to continue and record it rather than rebuild.

Mitigating technical facts, verified in the signed tag:

- encryption is `age` + scrypt (`internal/crypto/envelope.go`), which uses a
  random salt per object, so RC6 object keys are distinct from RC5's despite the
  shared secret;
- the passphrase is never cached — the keyring
  (`internal/credentials/keyring.go`) stores storage credentials only, and a
  hidden prompt appeared on every single command.

Residual impact: RC5 and RC6 are not cryptographically compartmentalized, so a
compromise of one secret exposes both. This does not affect any product
behavior under test and does not mask a Reinstate defect, but it is a real
departure from the isolation rule and is disclosed here rather than silently
passed.

### F-RC6-1 — rc.6 tag signature is not independently verifiable (non-blocking)

The `v0.1.0-rc.6` tag object carries an SSH signature whose key is not
registered as a signing key on the GitHub account (`reason=unknown_key`), and no
allowed-signers file exists locally. The release commit `9019bd9` verifies
correctly via GPG. Recommend registering the SSH signing key so tag signatures
verify for downstream consumers. Does not affect installed bytes: the installed
binary's embedded commit matches the peeled tag commit exactly, and both
checksum layers passed.

## 5. Section 19 sign-off status (Device A view)

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| `install.sh` returns 200 and installs RC6 on Mac | PASS | section 2.2 |
| `install.ps1` returns 200 and installs RC6 on Windows | NOT TESTED | Device B |
| Both installers are idempotent and PATH-safe | PARTIAL | Mac PASS (section 2.2); Windows NOT TESTED |
| Pre-init missing-config failure is accurate | PASS | section 2.3 |
| Post-init setup check and self-test pass on both devices | PARTIAL | Mac PASS (section 3.2); Windows NOT TESTED |
| Claude setup prompt completes on the Mac | PASS | section 3.2 |
| Codex setup prompt completes on Windows | NOT TESTED | Device B |
| Only two selected test sessions reach the remote manifest | PASS | section 3.4 |
| Remote manifest/snapshots are ciphertext-only | PASS | section 3.5 |
| Wrong passphrase fails without mutation | NOT TESTED | Device B |
| Claude Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| Codex Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| Active-agent overwrite is refused | NOT TESTED | M2/W2 |
| Existing Windows target is backed up before restore | NOT TESTED | W2 |
| Claude Windows-to-Mac resume succeeds | NOT TESTED | M3 |
| Codex Windows-to-Mac resume succeeds | NOT TESTED | M3 |
| Existing Mac targets are backed up before restore | NOT TESTED | M3 |
| Unchanged pushes skip without new snapshots | NOT TESTED | M3 |
| Divergence records a conflict without overwrite | NOT TESTED | M4/W3 |
| `--keep-both` preserves both branches | NOT TESTED | W3 |
| All required GitHub checks are green | NOT TESTED | to reconcile at M4 |

Additional physical regression rows tracked by the RC6 prompts:

| Regression | Result | Evidence |
| ---------- | ------ | -------- |
| F1 default refusal (Mac) | PASS | section 3.3 |
| F1 default refusal (Windows) | NOT TESTED | Device B |
| F2 strict status on missing manifest | NOT TESTED | Device B |
| F3 bad storage coordinates refused | NOT TESTED | Device B |

## 6. Safety attestation

- No endpoint, bucket, access key, secret key, passphrase, keyring value, agent
  auth file, transcript content, downloaded `.age` object, or ciphertext byte
  was requested, read, echoed, logged, hashed for disclosure, or committed by
  the coordinating agent.
- `--all` was never used; only the two selected RC6 session IDs were operated on.
- No RC5-or-older home, profile prefix, report, real agent session, unrelated
  project, or unrelated remote object was deleted or mutated.
- No restored vendor file was manually moved to manufacture discovery.
- No product code was modified. The only repository change is this report.
