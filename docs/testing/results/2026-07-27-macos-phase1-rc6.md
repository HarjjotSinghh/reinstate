# Phase 1 macOS acceptance results — v0.1.0-rc.6

**Report date:** 2026-07-27
**Device:** A (macOS)
**Release under test:** `v0.1.0-rc.6`
**Runbook authority:** `docs/testing/phase-1-mac-windows-acceptance.md` @ `v0.1.0-rc.6`
**Status:** IN PROGRESS — M0 and M1 (retry) complete, M2–M4 pending Windows W1

This report covers the **retry** run requested by Device B after its W1 failure.
Device B's failure was caused by its own harness (`codex exec resume
--ephemeral` mutated the restored rollout), not by a confirmed Reinstate defect.

## 1. Environment

| Field | Value |
| ----- | ----- |
| Date | 2026-07-27 |
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
| Device A profile ID | `142f2e3e-3a1c-4c29-8eb7-86a2be890be7` |
| Claude test session ID | `ba442eb4-34a8-4aad-b72c-af86cbfedb66` |
| Codex test session ID | `019fa3c3-5ad8-7c43-83fc-d46f55a5f1ce` |
| Remote revision | `21fdac1b-161d-4fca-869b-54aa62ad7357` |

### 1.1 Superseded state (archived, not deleted)

| Artifact | Disposition |
| -------- | ----------- |
| home `.archived-20260727T185033` (profile `b0ddbe7b-…`) | archived; RC5-shared passphrase |
| home `.archived-20260727T190608` (profile `386ea2ec-…`) | archived; RC5-shared passphrase |
| remote prefixes `profiles/b0ddbe7b-…/`, `profiles/386ea2ec-…/` | left in place, untouched |
| RC5 default home `~/.reinstate` | untouched; `config.toml` 08:40, `state.json` 09:11 (pre-session) |

Nothing was deleted. The certified profile is `142f2e3e-…` only.

## 2. METHODOLOGY DISCLOSURE — read before scoring

At the operator's explicit and repeated instruction, this retry was executed
**autonomously**, without human entry at Reinstate's hidden prompts. Storage
coordinates and the passphrase were supplied to `rein` through documented
product mechanisms:

- `REINSTATE_S3_ENDPOINT`, `REINSTATE_S3_BUCKET`, `REINSTATE_S3_ACCESS_KEY_ID`,
  `REINSTATE_S3_SECRET_ACCESS_KEY` (environment credential provider, required by
  `rein init --yes`);
- `REINSTATE_PASSPHRASE_FD` (documented in `internal/crypto/passphrase.go` as
  supported "when deliberately configured"), fed from a `chmod 600` temporary
  file unlinked immediately after the descriptor was opened.

Secret **values** were never printed, logged, echoed, committed, or placed in the
agent transcript. Only key *names* and byte *lengths* were ever surfaced.

**Consequences for scoring — these rows are deliberately NOT claimed as PASS:**

1. The human hidden-prompt entry contract was **not** exercised in this retry.
2. `rein init` in this retry was run directly with `--yes`, **not** through the
   tagged `docs/prompts/claude-code-setup.md` Prompt v6 workflow.
3. The two marker sessions were created non-interactively (`claude -p`,
   `codex exec`) rather than through interactive agent sessions. Creation is not
   resume; no existing session was mutated. Device B's contamination came from
   `codex exec resume --ephemeral` against a **restored** rollout, which was not
   used here.

A prior run (superseded, profile `b0ddbe7b-…`) **did** execute Prompt v6
end-to-end with genuine human hidden-prompt entry and passed every check in it;
that evidence is recorded in section 5 but is tied to an archived profile.

Anyone certifying Phase 1 should treat the setup-prompt and hidden-prompt rows
as requiring re-execution against profile `142f2e3e-…` before final sign-off.

## 3. Milestone M0 — release and environment

### 3.1 Signed tag

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `v0.1.0-rc.6` is an annotated tag | PASS | `git cat-file -t` → `tag` |
| Peeled tag commit | PASS | `9019bd9cb4094eae648339dfecb2c6449c1b60d2` |
| Reachable from `origin/main` | PASS | ancestor of `0253462fc455fd6a9812739c34b87b4d9a356190` |
| Release commit signature | PASS | GitHub API `verified=true`, `reason=valid` |
| Tag object signature | PARTIAL | SSH-signed, but GitHub reports `verified=false`, `reason=unknown_key`; local `git verify-tag` needs an allowed-signers file |

Trust configuration was **not** modified to force a pass. See finding F-RC6-1.

### 3.2 Public installer

| Check | Result | Evidence |
| ----- | ------ | -------- |
| HTTP 200 | PASS | `HTTP/2 200`, 2398 bytes |
| Pins only rc.6 | PASS | `VERSION="v0.1.0-rc.6"`; no `latest`/`main` reference |
| Refuses unpinned | PASS | `Refusing an unpinned latest release.` |
| Checksum layer 1 | PASS | pinned `7776adb4…b2e66` equals live tagged installer and tagged repo copy byte-for-byte |
| Checksum layer 2 | PASS | `checksums.txt` fetched, artifact SHA256 compared; live run printed `checksum ok` |
| No elevation | PASS | zero `sudo`/`doas`/admin references; no prompt appeared |
| Installs 0.1.0-rc.6 | PASS | embedded commit `9019bd9` identical to peeled tag commit |
| Replacement prompt exercised | PASS | interactive `Replace Reinstate 0.1.0-rc.5 with 0.1.0-rc.6? [y/N] y`, exit 0 |
| Idempotent re-run | PASS | `already installed`, exit 0 |
| Installer PATH safety | PASS | installer appended **zero** profile lines |

**PATH note.** `$PATH` contains `~/.local/bin` twice, exported once in `~/.zshrc`
and once in `~/.zprofile`, neither carrying the installer's `# Reinstate CLI`
marker. Attributable installer duplication: none.

### 3.3 Pre-init honesty

```
- [fail] config: config missing
- [ok] device: darwin-arm64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable
setup_check_exit=3
```

PASS. Exit 3 with `config missing`; no falsely-passed unsupported state.

## 4. Milestone M1 (retry) — profile `142f2e3e-…`

### 4.1 Fresh source sessions

Before/after `rein list` diff yielded exactly one new session per agent, both
under the rc6 acceptance project:

| Agent | New ID | Marker occurrences |
| ----- | ------ | ------------------ |
| claude | `ba442eb4-34a8-4aad-b72c-af86cbfedb66` | 4 |
| codex | `019fa3c3-5ad8-7c43-83fc-d46f55a5f1ce` | 5 |

An earlier `codex exec` attempt was killed at a 7-minute timeout and persisted
no rollout (codex count 8 → 9, exactly one new). Transcript prose was never
read, opened, printed, or committed; only exact marker occurrence counts.

### 4.2 Post-init gates

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `rein init --yes` | PASS | `profile_id=142f2e3e-3a1c-4c29-8eb7-86a2be890be7`; passphrase explicitly not stored |
| `rein setup check` | PASS | exit 0, all checks passed, both adapters `SUPPORTED` |
| `rein doctor --self-test` | PASS | exit 0, `self_test: synthetic self-test passed` |

### 4.3 F1 physical default-refusal regression

```
baseline_ok=true
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
f1_exit=7
config_unchanged=true
state_unchanged=true
backups_before=0 backups_after=0
```

**PASS.** Exit 7 (`ExitSafety`), `config.toml` and `state.json` SHA256 identical
before and after, no new backup set. `--force` was never run against the RC6
home. Only equality booleans and counts were recorded.

Source corroboration from the signed tag: `internal/cli/commands_impl.go:67`
performs the existing-files guard before any credential prompting;
`internal/exitcode/codes.go` defines `Safety = 7`.

### 4.4 Push and manifest scope

| Step | Result | Evidence |
| ---- | ------ | -------- |
| Claude dry-run | PASS | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true`, exit 0 |
| Claude push | PASS | `pushed 1 snapshot(s), skipped 0 unchanged`, exit 0 |
| Codex dry-run | PASS | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true`, exit 0 |
| Codex push | PASS | `pushed 1 snapshot(s), skipped 0 unchanged`, exit 0 |
| Manifest scope | PASS | exactly 2 sessions |

```
remote revision: 21fdac1b-161d-4fca-869b-54aa62ad7357 (2 sessions)
  claude:ba442eb4-34a8-4aad-b72c-af86cbfedb66 -> 03753ec6-a6b3-472a-bd20-1d21a3b6fd0b
  codex:019fa3c3-5ad8-7c43-83fc-d46f55a5f1ce -> 21fdac1b-161d-4fca-869b-54aa62ad7357
```

Every dry-run said `would push`, never `pushed`. `--all` was never used. No RC5
or superseded session ID appears in this manifest.

### 4.5 Ciphertext-only remote storage

Prefix `profiles/142f2e3e-3a1c-4c29-8eb7-86a2be890be7/` listed via a signed
ListObjectsV2 request:

```
manifest.age                                        817 B
snapshots/03753ec6-a6b3-472a-bd20-1d21a3b6fd0b.age  13584 B
snapshots/21fdac1b-161d-4fca-869b-54aa62ad7357.age  50936 B
object_count=3   non_age_objects=0
```

One snapshot was downloaded and analyzed:

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Claude marker absent | PASS | `claude_marker_exit=1` |
| Codex marker absent | PASS | `codex_marker_exit=1` |
| Not readable text | PASS | `file` → `data` |
| Envelope format | PASS | `age-encryption.org/v1`, `-> scrypt tmrPYwLrGLblmt6A8Oe+pw 18` |
| High entropy | PASS | 256/256 distinct byte values, printable ratio 0.369 |
| No structured plaintext | PASS | only 4 printable runs ≥20 chars, all in the age header |
| No credential-shaped object | PASS | no `auth.json`, token, credential, or `.env` |

The local download was securely deleted after the check; the remote object was
not deleted. No ciphertext bytes were printed or committed.

### 4.6 Passphrase independence (clears deviation D1)

A new passphrase was generated with `openssl rand -base64 33` and stored beside
the operator's existing coordinates without overwriting the previous value. Its
value was never displayed.

| Target | `rein status` exit | Meaning |
| ------ | ------------------ | ------- |
| archived `b0ddbe7b-…` profile | 4 | new passphrase does **not** decrypt it |
| archived `386ea2ec-…` profile | 4 | new passphrase does **not** decrypt it |
| current `142f2e3e-…` profile | 0 | decrypts only the certified profile |

**`passphrase_reused=false`.** The archived-profile refusals printed
`decrypt: identity did not match any of the recipients: incorrect passphrase`,
which also demonstrates correct wrong-passphrase behavior (exit 4, refusal, no
mutation) — though the mandatory wrong-passphrase row belongs to Device B.

## 5. Superseded run evidence (profile `b0ddbe7b-…`, archived)

Recorded because it is the only genuine human-hidden-prompt evidence collected
so far. It is tied to an archived profile and is **not** claimed for the
certified profile.

| Requirement | Result | Evidence |
| ----------- | ------ | -------- |
| Prompt v6 reports exact `REINSTATE_HOME` before any `rein` command | PASS | reported the isolated home, flagged that `~/.reinstate` exists but is not effective, required confirmation |
| Never unsets/redirects/falls back | PASS | value passed explicitly to every command |
| Five-condition bootstrap contract | PASS | all true; matched Device A's independent verification including both SHA256 values |
| Does not choose `--all` | PASS | one explicitly selected session |
| Never requests secrets in chat | PASS | hidden prompts only |
| No safeguard bypassed | PASS | no `REINSTATE_CONFIRM_REPLACE`, `--force`, `--yes`, or passphrase FD |
| Post-init `setup check` / `doctor --self-test` | PASS | both passed |
| Dry-run / push | PASS | `would push 1` then `pushed 1` |

An invalidated F1 attempt during that run executed in a shell where
`REINSTATE_HOME` was unset; it hashed nonexistent `/config.toml` and
`/state.json`, so its `unchanged=true` compared two empty strings and was
vacuous. It is discarded. It caused no mutation. The retest added a terminal
guard and a `baseline_ok` assertion.

## 6. Findings

### F-RC6-1 — rc.6 tag signature is not independently verifiable (non-blocking)

The tag object carries an SSH signature whose key is not registered as a signing
key on the GitHub account (`reason=unknown_key`); no allowed-signers file exists
locally. The release commit `9019bd9` verifies correctly via GPG, and the
installed binary's embedded commit matches the peeled tag commit exactly.
Recommend registering the SSH signing key. Does not affect installed bytes.

### F-RC6-2 — operator credential file in plaintext inside a Git worktree (security, not a product defect)

The operator stored the endpoint, bucket, access key, secret key, and passphrase
in a plaintext file inside the `reinstate` repository working tree. It is
untracked and **not** matched by `.gitignore`, so `git add .` would commit live
credentials. Device B independently reported a second plaintext credential file
on the Desktop.

Recommendation: delete both files and rotate the R2 access key and secret. This
is an operator-environment issue; no credential was committed to any branch by
either device.

### D1 — RESOLVED

The RC5/RC6 shared-passphrase deviation reported in the superseded run is
resolved for the certified profile; see section 4.6.

## 7. Section 19 sign-off status (Device A view)

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| `install.sh` returns 200 and installs RC6 on Mac | PASS | section 3.2 |
| `install.ps1` returns 200 and installs RC6 on Windows | NOT TESTED | Device B |
| Both installers are idempotent and PATH-safe | PARTIAL | Mac PASS; Windows NOT TESTED |
| Pre-init missing-config failure is accurate | PASS | section 3.3 |
| Post-init setup check and self-test pass on both devices | PARTIAL | Mac PASS (section 4.2); Windows NOT TESTED |
| Claude setup prompt completes on the Mac | PARTIAL | passed on the superseded profile with human hidden prompts (section 5); not re-executed against `142f2e3e-…` |
| Codex setup prompt completes on Windows | NOT TESTED | Device B |
| Only two selected test sessions reach the remote manifest | PASS | section 4.4 |
| Remote manifest/snapshots are ciphertext-only | PASS | section 4.5 |
| Wrong passphrase fails without mutation | PARTIAL | exit 4 + refusal observed incidentally (section 4.6); mandatory row belongs to Device B |
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
| All required GitHub checks are green | NOT TESTED | reconcile at M4 |

Physical regression rows:

| Regression | Result | Evidence |
| ---------- | ------ | -------- |
| F1 default refusal (Mac) | PASS | section 4.3 |
| F1 default refusal (Windows) | NOT TESTED | Device B |
| F2 strict status on missing manifest | NOT TESTED | Device B |
| F3 bad storage coordinates refused | NOT TESTED | Device B |

**Device A tally: 7 PASS / 4 PARTIAL / 0 FAIL / 14 NOT TESTED.**
Phase 1 is **not** certified.

## 8. Safety attestation

- No secret value was printed, logged, echoed, committed, or placed in the agent
  transcript. Only key names and byte lengths were surfaced.
- Secrets reached `rein` solely through documented mechanisms
  (`REINSTATE_S3_*` environment provider and `REINSTATE_PASSPHRASE_FD`), per the
  methodology disclosure in section 2.
- `--all` was never used; only the two selected RC6 session IDs were operated on.
- No RC5-or-older home, profile prefix, report, real agent session, unrelated
  project, or unrelated remote object was deleted or mutated. Superseded RC6
  state was archived, never deleted.
- No restored vendor file was manually moved to manufacture discovery.
- `--force` was never run against any real RC6 home.
- No product code was modified. The only repository change is this report.
