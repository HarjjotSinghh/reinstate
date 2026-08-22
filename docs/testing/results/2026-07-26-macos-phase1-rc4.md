# Phase 1 acceptance — Device A (macOS) — v0.1.0-rc.4

Runbook followed: `docs/testing/phase-1-mac-windows-acceptance.md` as published at
tag `v0.1.0-rc.4` (commit `f0b96006df6dee24c6d5c8d8fea5c34a655c4aff`).

No secrets, no transcript contents, no downloaded ciphertext bytes, and no
credential material appear in this report. Hidden inputs were typed only into
Reinstate's own hidden prompts by the operator.

## Status

**Run in progress.** Milestones M0 and part of M1 executed. Rows not yet
executed are marked `NOT TESTED` and are never marked `PASS`.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date/time (start) | 2026-07-26, Asia/Kolkata |
| Mac model and macOS version | macOS 26.5.2 (build 25F84) |
| Mac architecture | `arm64` |
| Windows edition/build | Device B — recorded by Device B owner |
| Claude Code version | `2.1.220` |
| Codex CLI version | `codex-cli 0.145.0` |
| Git version | `git version 2.51.0` |
| Reinstate version | `0.1.0-rc.4` (commit `f0b9600`, built `2026-07-26T05:59:11Z`) |
| Release under test | `v0.1.0-rc.4` |
| Tag commit | `f0b96006df6dee24c6d5c8d8fea5c34a655c4aff` |
| Device A profile ID | `893b7056-c694-4f36-ac60-9c167e586834` |
| Claude test session ID | `0f32db9b-f869-4dd9-8736-725c19edfea3` |
| Codex test session ID | `019f9d25-4592-7921-a779-1843271d2da4` |
| Isolated Reinstate home | `$HOME/.reinstate-phase1-acceptance-rc4` |
| Disposable project | `$HOME/Projects/reinstate-phase1-acceptance-rc4` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc4` |

Both recognized agent version ranges are satisfied: Claude Code
`2.1.219`–`2.1.220` and Codex CLI `0.133.0`–`0.145.0`.

## 2. M0 — release and environment

### 2.1 Signed tag verification — PASS

| Check | Command | Result |
| ----- | ------- | ------ |
| Tag object is annotated | `git cat-file -t v0.1.0-rc.4` | `tag` |
| Tag carries a signature | `git cat-file tag v0.1.0-rc.4` | signature block present |
| Signature verifies | `git verify-tag v0.1.0-rc.4` | exit `0`, `Good "git" signature` (ED25519 key `SHA256:iaAqTbU4yrAkdlYemHMrVRHP58unAfv3r6N3Y6xKv/U`) |
| Tag commit reachable from `origin/main` | `git merge-base --is-ancestor v0.1.0-rc.4^{commit} origin/main` | exit `0` |
| Tag commit | `git rev-list -n1 v0.1.0-rc.4` | `f0b96006df6dee24c6d5c8d8fea5c34a655c4aff` |
| Runbook present at tag | `git cat-file -e v0.1.0-rc.4:docs/testing/phase-1-mac-windows-acceptance.md` | exit `0` |
| Claude setup prompt version at tag | `docs/prompts/claude-code-setup.md` | `**Prompt version:** 4` |

No trust configuration, allowed-signers file, or GPG/SSH setting was changed to
obtain this result.

### 2.2 Public `install.sh` — PASS (with deviation D1)

| Check | Evidence |
| ----- | -------- |
| HTTP status | `HTTP/2 200` for `HEAD https://reinstate.dev/install.sh` (`server: Vercel`, `content-type: text/plain`, `content-length: 2398`) |
| Version pin | bootstrap contains `VERSION="v0.1.0-rc.4"` |
| No `latest` reference | `grep -nE 'latest'` over the bootstrap returned no match |
| Checksum layer 1 | bootstrap pins `PINNED_INSTALLER_SHA256` and compares it against the downloaded second-stage installer; run printed `installer checksum ok` |
| Checksum layer 2 | second-stage installer downloads the exact release asset plus `checksums`, and the run printed `checksum ok` |
| Asset selected | `reinstate_0.1.0-rc.4_darwin_arm64.tar.gz` from the `v0.1.0-rc.4` GitHub release |
| Install result | `Installed reinstate v0.1.0-rc.4 → ~/.local/bin/reinstate`; `Installed rein alias → ~/.local/bin/rein` |
| Exit code | `0` |
| Binary resolution | `command -v rein` → `~/.local/bin/rein`; `command -v reinstate` → `~/.local/bin/reinstate`; `rein` is a symlink to `reinstate` |
| Version JSON | `{"commit":"f0b9600","date":"2026-07-26T05:59:11Z","name":"reinstate","version":"0.1.0-rc.4"}` — matches the tag commit |
| PATH safety (run 1) | `.local/bin` line counts in `~/.zshrc`, `~/.zprofile`, `~/.profile`, `~/.bashrc` were `4 / 2 / 2 / 1` before and after — unchanged, no duplicate entry added |

### 2.3 Installer idempotency and PATH safety — PASS

Second run of the same public one-liner (runbook §5), same deviation D1 applies:

| Check | Evidence |
| ----- | -------- |
| Exit code | `0` |
| Checksum layer 1 re-verified | `installer checksum ok` |
| Checksum layer 2 re-verified | `Downloading checksums...` then `checksum ok` |
| Idempotent outcome | `Reinstate v0.1.0-rc.4 is already installed at ~/.local/bin/reinstate` — no reinstall, no downgrade |
| Binary unchanged | `sha256` of `~/.local/bin/reinstate` identical before and after run 2 |
| No new PATH line | `.local/bin` line counts in `~/.zshrc`, `~/.zprofile`, `~/.profile`, `~/.bashrc` were `4 / 2 / 2 / 1` before run 1, after run 1, and after run 2 — the installer added zero PATH lines across both runs |
| Version after run 2 | `0.1.0-rc.4`, commit `f0b9600` |

Honest note: the operator's live `PATH` contains `~/.local/bin` four times. That
duplication is pre-existing operator shell configuration — the count was
identical before the first RC4 install and did not change across either run, so
the installer contributed no PATH entries. The runbook's macOS requirement
("must not duplicate its PATH entry") is met; the `count = 1` assertion in §5
applies to the Windows user PATH, not to this pre-existing macOS shell config.

### 2.4 Pre-init honest failure — PASS

Command, in the fresh isolated home with no config:

```sh
REINSTATE_HOME="$HOME/.reinstate-phase1-acceptance-rc4" rein setup check
```

The isolated home did not exist before this command.

Exit code: `3`.

Reported checks:

| Check | Status |
| ----- | ------ |
| `config` | `fail` — `config missing` |
| `device` | `ok` — `darwin-arm64` |
| `agent.claude` | `ok` — `SUPPORTED (2.1.220)` |
| `agent.codex` | `ok` — `SUPPORTED (0.145.0)` |
| `keyring` | `ok` — OS keyring provider reachable |

Summary line: `1 check(s) failed`. Nothing falsely passed, and no adapter or
platform was misreported as unsupported.

The tool printed the home path as `${REINSTATE_HOME}` rather than expanding the
local username — no local path leakage in this output.

## 3. M1 — fresh source sessions

### 3.1 Disposable mapped project — PASS

`$HOME/Projects/reinstate-phase1-acceptance-rc4` created, writable, `git init`
completed, `README.md` written. Canonical ID for both devices:
`local/reinstate-phase1-acceptance-rc4`. The path differs from the Windows
path by construction (POSIX vs. `%USERPROFILE%\Projects\...`).

### 3.2 Codex source session — PASS

Prompt used (harmless, no personal data):

```text
Reply with exactly: REINSTATE-PHASE1-RC4-MAC-CODEX-A1
```

Before/after metadata diff of `rein list --agent codex` (465 → 466 rows)
produced exactly one new row:

```text
codex	019f9d25-4592-7921-a779-1843271d2da4	${HOME}/Projects/reinstate-phase1-acceptance-rc4
```

Exactly one fresh rollout was persisted, in the RC4 project, so no marker-count
disambiguation was required and no transcript content was inspected. No older
acceptance session was reused; the RC3 rows under
`~/Projects/reinstate-phase1-acceptance` were left untouched.

`CODEX_SESSION_ID=019f9d25-4592-7921-a779-1843271d2da4`

### 3.3 Claude source session — PASS (sibling candidates disambiguated)

Created by the operator in an authenticated interactive Claude Code session
launched from the RC4 project with the isolated `REINSTATE_HOME` exported.
Prompt used:

```text
Reply with exactly: REINSTATE-PHASE1-RC4-MAC-CLAUDE-A1
```

`rein list --agent claude` went from 173 to 175 rows, so Claude wrote **two**
sibling candidates for the one invocation — exactly the case runbook §7
anticipates:

```text
claude	0ce2b28a-1eea-489a-8cef-1b68abee04a2	-Users-harjjotsinghh-Projects-reinstate-phase1-acceptance-rc4
claude	0f32db9b-f869-4dd9-8736-725c19edfea3	-Users-harjjotsinghh-Projects-reinstate-phase1-acceptance-rc4
```

Selection was made by counting **only** exact occurrences of the marker string
and by reading per-line structural fields (`type`, `role`). No transcript prose
was printed, quoted, opened for reading, or copied anywhere.

| Candidate | Exact marker occurrences | Assistant-role line carries the marker | Last modified |
| --------- | ------------------------ | -------------------------------------- | ------------- |
| `0ce2b28a-1eea-489a-8cef-1b68abee04a2` | 3 | **no** — its single assistant line contains 0 occurrences | 12:08:05 |
| `0f32db9b-f869-4dd9-8736-725c19edfea3` | 4 | **yes** — assistant-role line contains the exact marker | 12:12:44 |

Only `0f32db9b…` contains the completed reply, so it is the selected session.
`0ce2b28a…` carries the marker only in user/summary records and was left in
place untouched.

`CLAUDE_SESSION_ID=0f32db9b-f869-4dd9-8736-725c19edfea3`

Deviation D2 is therefore closed: the automation path could not create the
session, but the operator-run path (which is what the runbook prescribes) did.

### 3.4 Claude Code setup prompt v4 — PASS (with deviation D5)

Runbook §8 was executed literally: the tag-exact text of
`docs/prompts/claude-code-setup.md` (**Prompt version: 4**, extracted with
`git show v0.1.0-rc.4:…` because the working-tree copy has since diverged — see
D4) was pasted into a **separate** interactive Claude Code session launched from
the RC4 acceptance project. The operator supplied only non-secret answers in
chat and entered credentials and the passphrase into Reinstate's hidden prompts.

Agent-reported results, each of which the evidence owner then re-verified
independently (§3.5):

| Setup-prompt step | Reported outcome |
| ----------------- | ---------------- |
| 1–2 detect + select bootstrap | macOS 26.5.2 / darwin-arm64 / zsh / native (not WSL) / Claude Code 2.1.220; selected `https://reinstate.dev/install.sh` |
| 3 inspect contract | all five conditions TRUE: pins `v0.1.0-rc.4`; canonical installer fetched from that exact tag; checksum verified before execution with exit 1 on mismatch; no `latest`/`main` resolution; assets only from `…/releases/download/v0.1.0-rc.4/`, with `REINSTATE_RELEASE_BASE_URL` blanked so no override can redirect it |
| 4 execute | installed to `~/.local/bin`, no elevation, no `/usr/local/bin`, no PATH change needed |
| 5 version + pre-init check | `0.1.0-rc.4`; pre-init failure was only `config missing` |
| 6 collect inputs | first device; one explicitly selected Claude session; `--all` never proposed |
| 7 init | `rein init --project local/reinstate-phase1-acceptance-rc4=<mac path>`, run by the operator in a private terminal |
| 8 post-init | `rein setup check` exit `0`, `rein doctor --self-test` exit `0` |
| 9 push | dry-run printed `would push 1 snapshot(s)`; real push printed `pushed 1 snapshot(s), skipped 0, dry_run=false` |
| 10–11 | correctly declined as Device B work |
| 12 | redacted report returned with no secrets and no transcript content |

The agent never proposed `--all`, and it stopped and reported rather than
proceeding when step 8 failed on the misdirected first init (see D5).

`PHASE1_PROFILE_ID=893b7056-c694-4f36-ac60-9c167e586834`

### 3.5 Independent re-verification of the post-init state — PASS

Performed by the evidence owner directly, not accepted from the setup agent's
report:

| Check | Command | Result |
| ----- | ------- | ------ |
| Profile ID is recorded in the **RC4** home | `grep '^profile_id' $REINSTATE_HOME/config.toml` | `893b7056-c694-4f36-ac60-9c167e586834` (device_id `af02cfe9-9545-488a-b118-05b42aac3b1e`) |
| Profile is fresh, not an RC3 reuse | compared against every other local home | distinct from `47e43f49…` (`~/.reinstate-phase1-acceptance`), `72733dd3…` (`~/.reinstate-phase1-acceptance-rc3`), and `4bbe6b81…` (`~/.reinstate-phase1-acceptance-rc3-r2`) |
| RC4 home layout | `ls $REINSTATE_HOME` | `config.toml`, `state.json`, `backups/`, `cache/`, `conflicts/`, `locks/`, `logs/` |
| Post-init setup check | `rein setup check` | exit `0`, `summary: all checks passed`; `config valid`, `device darwin-arm64`, `agent.claude SUPPORTED (2.1.220)`, `agent.codex SUPPORTED (0.145.0)`, keyring reachable |
| Synthetic self-test | `rein doctor --self-test` | exit `0`, `self_test: synthetic self-test passed` |
| Secret handling under a non-interactive stdin | `rein status < /dev/null` | exit `2`, `secret input requires an interactive terminal or REINSTATE_PASSPHRASE_FD` — refuses cleanly, does not hang, and does not fall back to any insecure input path |

Both `agent.claude` and `agent.codex` report `SUPPORTED`, satisfying runbook §1.

### 3.6 Codex push and remote manifest scope — PASS

Runbook §9, run by the operator in a private interactive terminal with
`REINSTATE_HOME` confirmed to end in `-rc4` before any command. `--all` was
never used; only the one selected Codex ID was passed.

| Command | Output |
| ------- | ------ |
| `rein push --agent codex --session 019f9d25-… --dry-run` | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true` |
| `rein push --agent codex --session 019f9d25-…` | `pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false` |

The human dry-run line begins with `would push`, not `pushed`, and reports
`dry_run=true`. The mutating run reports `dry_run=false`.

`rein status`:

```text
remote revision: 83f677fd-05fd-40f0-bcbb-f2caa858a109 (2 sessions)
  codex:019f9d25-4592-7921-a779-1843271d2da4 -> 83f677fd-05fd-40f0-bcbb-f2caa858a109
  claude:0f32db9b-f869-4dd9-8736-725c19edfea3 -> 42f4317b-b1f4-4387-a256-9d993756e43e
```

Remote session count: **exactly 2**, and both are the selected acceptance IDs.
No unrelated local session reached the remote manifest, and none of the RC3
acceptance sessions under `~/Projects/reinstate-phase1-acceptance` appear.

Recorded non-secret identifiers:

| Item | Value |
| ---- | ----- |
| Remote revision after Codex push | `83f677fd-05fd-40f0-bcbb-f2caa858a109` |
| Codex snapshot | `83f677fd-05fd-40f0-bcbb-f2caa858a109` |
| Claude snapshot (from the setup-prompt push) | `42f4317b-b1f4-4387-a256-9d993756e43e` |

Observation, not a defect claim: the revision identifier printed on the summary
line is the same UUID as the Codex snapshot identifier. Worth confirming this is
intended naming rather than a collision of two distinct concepts.

The passphrase was entered three times, each time at a hidden `Encryption
passphrase:` prompt while the process was visibly waiting. No secret was echoed,
logged, or passed as an argument or environment variable.

### 3.7 Ciphertext-only remote storage — PASS

Runbook §10, executed by the operator against **only** the fresh RC4 profile
prefix `profiles/893b7056-c694-4f36-ac60-9c167e586834/`. No RC3 prefix was
opened, listed, or touched.

| Check | Result |
| ----- | ------ |
| Prefix contains `manifest.age` and `snapshots/<opaque-uuid>.age` | confirmed |
| No `auth.json`, `.env`, token, key, or other credential-shaped object anywhere under the prefix | confirmed absent |
| `grep -aFq 'REINSTATE-PHASE1-RC4-MAC-CLAUDE-A1'` on the downloaded snapshot | exit `1` — marker **absent** |
| `grep -aFq 'REINSTATE-PHASE1-RC4-MAC-CODEX-A1'` on the downloaded snapshot | exit `1` — marker **absent** |
| `file` on the downloaded snapshot | `data` — not readable JSON, JSONL, or text |

`ciphertext_marker_absence=PASS`

The object inspected was one `.age` snapshot from the RC4 prefix. Only the two
grep exit codes and the `file` type string were recorded; no byte of the object,
no filesystem path, and no object name were transmitted to or read by the
evidence owner. The local download is the operator's to delete; the remote
object was left in place per runbook §10.

## 4. Deviations

| ID | Deviation | Classification |
| -- | --------- | -------------- |
| D1 | The runbook's literal `curl -fsSL https://reinstate.dev/install.sh \| sh` blocks forever in a non-TTY context. The second-stage installer's `confirm_replace()` prints `Replace Reinstate <old> with <new>? [y/N]` to `/dev/tty` and reads from `/dev/tty` with no timeout; with a readable but unattended `/dev/tty` the process waits indefinitely. The install was completed with the documented escape hatch `REINSTATE_CONFIRM_REPLACE=1`. | Test-harness / operator condition, not a product defect. The confirmation gate is deliberate safe-by-default behavior and the environment variable is documented in the installer itself. Worth noting for CI and agent-driven installs: there is no `--yes` flag and no TTY-availability timeout. |
| D2 | `claude -p` invoked from a non-interactive subprocess exited `1` with `Not logged in · Please run /login`, so the Claude Code source session could not be created from the automation context. Closed: the operator created it interactively instead. | Vendor/environment condition, not a Reinstate defect. The session must be created by the operator in their own authenticated terminal, which is what the runbook prescribes anyway. |
| D3 | Candidate disambiguation required touching the real `~/.claude/projects/<rc4-project>/` directory, which `CLAUDE.md` otherwise forbids for contributors. Access was limited to marker occurrence counts and per-line `type`/`role` fields for the two RC4 candidates; no prose was read or emitted, no file was moved, renamed, or modified. | Runbook-mandated (§7). Accepted, scoped to the two RC4 candidate files. |
| D5 | The operator's terminal still had `REINSTATE_HOME` exported to `~/.reinstate-phase1-acceptance-rc3-r2` (a leftover RC3 home) when the first `rein init` was run, so that init initialized the **RC3 home** instead of the RC4 home. It minted profile `4bbe6b81-b635-43e4-8d9d-7b5a859d4f73` there and **overwrote that home's existing `config.toml` and `state.json`** (both mtimes moved to `12:24:58`), destroying the RC3-r2 profile identity on disk. The setup agent detected the mismatch when step 8 failed with `config: config missing`, stopped, and reported rather than proceeding; a re-init against the correct RC4 home then produced the profile used for this run. The stale RC3-r2 home was left in place and not deleted. | Operator/test-state defect against this run's own isolation rule ("never mutate an RC3 home"). It did **not** contaminate the RC4 evidence: the RC4 profile is a distinct fresh UUID in the RC4 home, and profile `4bbe6b81…` is obsolete and unused. The underlying overwrite behavior is separately recorded as product finding F1. |
| D4 | The working tree of `docs/prompts/claude-code-setup.md` and `docs/testing/phase-1-mac-windows-acceptance.md` both differ from the `v0.1.0-rc.4` tag. All execution used the tag content, extracted with `git show v0.1.0-rc.4:<path>`. | Test-state condition. Recorded so nobody later reconciles this report against the newer working-tree copies. |

## 5. Findings

### Release-blocking

None so far.

### Release-blocking

- **F2 — `rein status` reports exit `0` and an empty revision when the remote
  manifest object is absent.** Full analysis, with the exact source lines, in
  §8.2. This is the mechanism that turned Device B's misconfiguration into a
  silent success instead of an actionable error, and it blocked the entire
  Windows W1 gate.

- **F3 — `rein init --profile-id` does not verify the named remote profile
  exists at the supplied coordinates**, so a mistyped endpoint produces a device
  that reports `config valid` while joined to nothing (§8.5).

- **F1 — `rein init` silently overwrites an already-initialized home's
  `config.toml` and `state.json`, with no confirmation and no backup.**
  Classification: product defect, data-loss class. Originally recorded here as
  outside the §19 rows; **reclassified to release-blocking** after Device B
  correctly pointed out that runbook §1 lists silent overwrite as an immediate
  stop condition, and a top-level stop condition is not cancelled by the absence
  of a matching checklist row (§8.4).

  Evidence, observed on disk rather than inferred: running `rein init` against
  `~/.reinstate-phase1-acceptance-rc3-r2`, a home that already contained a valid
  `config.toml` from an earlier run, replaced both `config.toml` and
  `state.json` in place (both mtimes moved to `12:24:58`) and minted a new
  `profile_id`. That home's `backups/` directory mtime stayed at `08:12:00`, so
  **no backup of the replaced config was written** — the previous profile
  identity is unrecoverable from disk. `rein init --help` exposes no `--force`,
  `--overwrite`, or equivalent flag, so there is no guard to opt out of and no
  guard to bypass; the destructive path is the default path.

  Suggested fix: refuse when a valid `config.toml` already exists unless an
  explicit flag is passed, and write a timestamped backup under `backups/`
  before replacing either file — matching the safety contract Reinstate already
  honors for agent session restores.

### Non-blocking

- N1 (D1): the public POSIX installer has no non-interactive confirmation flag
  and no timeout when `/dev/tty` is readable but unattended, so replacing an
  existing install hangs indefinitely in automation unless
  `REINSTATE_CONFIRM_REPLACE=1` is set. Consider documenting this on the
  install page or adding a TTY-availability timeout.

### Positive security observations

- `rein status` with stdin closed exits `2` with `secret input requires an
  interactive terminal or REINSTATE_PASSPHRASE_FD`. It neither hangs nor falls
  back to reading a secret from a non-interactive stream, so no passphrase can
  be captured by an automation harness that lacks a TTY.
- `rein setup check` and `rein doctor` print the home as the literal
  `${REINSTATE_HOME}` instead of expanding it, so diagnostic output does not
  leak the local username.

## 6. Section 19 sign-off checklist

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| `install.sh` returns 200 and installs RC4 on Mac | PASS | §2.2 |
| `install.ps1` returns 200 and installs RC4 on Windows | PASS | Device B report §3.2 |
| Both installers are idempotent and PATH-safe | PASS | §2.2, §2.3 (macOS); Device B report §3.2 — persisted user PATH held exactly one normalized entry |
| Pre-init missing-config failure is accurate | PASS | §2.4 (macOS); Device B report §3.3 |
| Post-init setup check and self-test pass on both devices | NOT TESTED — macOS PASS (§3.5), Windows never reached these commands | §3.5; Device B report §4.3 |
| Claude setup prompt completes on the Mac | PASS | §3.4 |
| Codex setup prompt completes on Windows | **FAIL** — init succeeded but the workflow could not complete past the status gate | Device B report §4.1–4.3 |
| Only two selected test sessions reach the remote manifest | PASS | §3.6 |
| Remote manifest/snapshots are ciphertext-only | PASS | §3.7 |
| Wrong passphrase fails without mutation | NOT TESTED | Device B stopped before the negative test; Device B report §4.3 |
| Claude Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| Codex Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| Active-agent overwrite is refused | NOT TESTED | Device B |
| Existing Windows target is backed up before restore | NOT TESTED | Device B |
| Claude Windows-to-Mac resume succeeds | NOT TESTED | |
| Codex Windows-to-Mac resume succeeds | NOT TESTED | |
| Existing Mac targets are backed up before restore | NOT TESTED | |
| Unchanged pushes skip without new snapshots | NOT TESTED | |
| Divergence records a conflict without overwrite | NOT TESTED | |
| `--keep-both` preserves both branches | NOT TESTED | |
| All required GitHub checks are green | NOT TESTED | |

## 7. Milestone handoffs

### MAC-RC4-M1 — emitted, awaiting Windows W1

```text
MAC-RC4-M1
release=v0.1.0-rc.4
tag_commit=f0b96006df6dee24c6d5c8d8fea5c34a655c4aff
profile_id=893b7056-c694-4f36-ac60-9c167e586834
canonical_project_id=local/reinstate-phase1-acceptance-rc4
claude_session_id=0f32db9b-f869-4dd9-8736-725c19edfea3
codex_session_id=019f9d25-4592-7921-a779-1843271d2da4
remote_session_count=2
ciphertext_marker_absence=PASS
mac_report_path=docs/testing/results/2026-07-26-macos-phase1-rc4.md
END-MAC-RC4-M1
```

Contains no endpoint, bucket, credential, passphrase, transcript text, local
username, or downloaded-object path.

## 8. Windows W1 failure — Device A analysis

Device B reported `WINDOWS-RC4-W1` **FAIL**. Its blocking row: correct-passphrase
`rein status` exited `0` but printed `remote revision:  (0 sessions)` — an empty
revision and zero sessions — where exactly two were required. Device B correctly
stopped and marked W2/W3 `NOT TESTED`, and `MAC-RC4-M2` was therefore **not**
started; the M2 gate requires both A1 restores and resumes to pass.

### 8.1 Mechanism, read from the tagged source

This is not speculation — the code path is deterministic at `v0.1.0-rc.4`:

- `internal/cli/commands_impl.go:317-320` — `status` calls
  `eng.FetchManifest(...)` and converts **any** error into
  `ExitError(ExitAuthStorage, …)`. Exit `0` therefore proves `FetchManifest`
  returned **no error**.
- `internal/sync/push.go:453-456` — `FetchManifest` is a thin wrapper over
  `loadManifest`.
- `internal/sync/push.go:217-220` — `loadManifest` does
  `Backend.Get(ctx, e.key("manifest.age"))` and, on
  `errors.Is(err, backend.ErrNotFound)`, returns
  `schema.NewManifest("")` with a **nil error**.

So the observed Windows output means the backend returned `ErrNotFound` for the
key `<storage.prefix>/manifest.age`. The passphrase was never applied: a wrong
passphrase would have failed inside `envelopeCodec().DecryptReader`, which
returns an error and exits `ExitAuthStorage`, not `0`. **Device B never found a
manifest object at the key it computed**, rather than failing to decrypt one.

### 8.2 Consequent product defect — F2

**F2 — `rein status` reports success when the remote manifest is absent.**
Classification: product defect, honest-failure class. Release-blocking for
Phase 1, because encrypted multi-device sync is the stated wedge and this is its
silent-failure mode.

Treating `ErrNotFound` as an empty manifest is correct for the first-ever push
on a first device, where no manifest can exist yet. It is wrong for `status` and
`pull` on a device that was just initialized with an explicit `--profile-id`,
because that flag is an assertion that the profile already exists remotely. In
that case a missing `manifest.age` means the device is misconfigured or pointed
at the wrong bucket/endpoint/prefix, and the tool reports it as a healthy,
empty profile with exit `0`.

Runbook §6 sets the standard this violates: an inaccurate pass is unacceptable,
and §1 requires stopping on an unexplained exit code. An operator following the
runbook literally sees `exit 0` and no error text, with nothing pointing at the
real problem.

Suggested fix: when the config carries a `profile_id` that was adopted from
another device (or generally, whenever `status`/`pull` runs against a
non-empty configured prefix), treat a missing `manifest.age` as a distinct
non-zero condition — e.g. `remote profile not found at <prefix>` — instead of
synthesizing an empty manifest. Keep the empty-manifest fallback confined to the
first push path.

### 8.3 Root cause of the Windows miss — under investigation

F2 explains why the failure was **silent**; it does not by itself explain why
the key was missing. The Mac side is verified correct:

| Device A fact | Value |
| ------------- | ----- |
| `storage.type` | `s3` |
| `storage.prefix` | exactly `profiles/893b7056-c694-4f36-ac60-9c167e586834` (45 chars) |
| Manifest key Device A reads and wrote | `profiles/893b7056-c694-4f36-ac60-9c167e586834/manifest.age` |
| Endpoint identity | SHA-256/16 `ab1ccfe9fd580859` |
| Bucket identity | SHA-256/16 `a230cd97c4d552f5` |
| Region identity | SHA-256/16 `929260ad9b9ea9fe` |

Coordinate values are recorded only as truncated hashes so the two devices can
be compared without either report disclosing an endpoint or bucket name.

Device B returned the same sanitized comparison:

| Field | Device A | Device B | Verdict |
| ----- | -------- | -------- | ------- |
| `storage.prefix` | `profiles/893b7056-…` (45 chars) | matches, 45 chars | identical |
| `storage.type` | `s3` | `s3` | identical |
| bucket hash | `a230cd97c4d552f5` | `a230cd97c4d552f5` | identical |
| region hash | `929260ad9b9ea9fe` | `929260ad9b9ea9fe` | identical |
| endpoint hash | `ab1ccfe9fd580859` | `2f02a7dfdc382692` | **differs** |

The prefix is byte-identical, so `rein init --profile-id` adopted the Device A
profile correctly. This is **not** a `--profile-id` defect.

The endpoint difference was then identified without either device disclosing its
endpoint, by hashing candidate normalizations of the Device A endpoint on
Device A and comparing against Device B's hash:

```text
endpoint + "/" + bucket   -> 2f02a7dfdc382692   MATCH
endpoint as-is            -> ab1ccfe9fd580859   no
endpoint + "/"            -> a716886c1382f4ba   no
endpoint + "/" + bucket + "/" -> 8bf0983a05eb1589 no
host without scheme, http://, virtual-host style, :443, lowercased,
double slash, whitespace-padded -> all no
```

**Root cause, established:** Device B's `storage.endpoint` is the Device A
endpoint with `/<bucket>` appended — the operator pasted the bucket-inclusive
URL into the endpoint prompt during Device B's `init`. With that endpoint plus
bucket `<bucket>`, the effective object key became
`<bucket>/profiles/893b7056-…/manifest.age`, which does not exist, so the
backend returned a legitimate 404. That is consistent with every observed
symptom: a reachable endpoint (no connection error, which would have exited
non-zero), `ErrNotFound`, and therefore exit `0` with an empty manifest.

Classification: **operator-input error at init, unmasked by product defect F2
and permitted by product defect F3.** The Windows W1 failure is therefore
recoverable test state, not a Reinstate binary defect in the sync path. Device A
evidence is unaffected.

### 8.5 Companion product defect — F3

**F3 — `rein init --profile-id <uuid>` does not verify that the named remote
profile is reachable at the supplied storage coordinates.** Classification:
product defect, silent-misconfiguration class. Release-blocking for Phase 1
alongside F2, because together they let an additional device believe it has
joined an existing profile when it has not.

`--profile-id` is an explicit assertion that the profile already exists
remotely. Device B's init exited `0` and wrote a complete, "valid" config while
pointing at a location where that profile's `manifest.age` does not exist.
Nothing in `init`, and nothing in the subsequent `rein setup check`
(`config: config valid`), contradicted the operator. The first signal available
was `status` printing `0 sessions` with exit `0` — which F2 makes
indistinguishable from a genuinely empty profile.

Suggested fixes, either of which would have caught this in seconds:

- on `init --profile-id`, probe for `<prefix>/manifest.age` and fail with a
  clear "profile not found at these coordinates" error when it is absent; and
- reject or normalize an endpoint whose trailing path segment equals the
  configured bucket name, which is a common paste error, or at minimum warn.

### 8.6 Outstanding diagnostic

Device A `rein status` has not yet been re-run to prove the manifest Device A
wrote is still present with 2 sessions. This is expected to pass — `status`
performs no write, and Device B's misconfigured coordinates addressed a
different key entirely, so no Device B command could have touched the Device A
objects — but it is recorded as `NOT TESTED` until observed rather than assumed.

### 8.4 Cross-device agreement on F1

Device B independently escalated the Mac-reported `rein init` overwrite (F1) to
release-blocking, on the grounds that runbook §1 lists **silent overwrite** as
an immediate stop condition and that falling outside a §19 row does not cancel a
top-level stop condition. That reading is correct and this report adopts it: F1
is reclassified from "outside the mandatory rows" to **release-blocking**.

## 9. Verdict

**Phase 1 FAILS. Do not release `v0.1.0-rc.4`.**

Device A M1 is complete and green. Device B W1 failed at its mandatory
correct-passphrase status gate, so M2, M3, and M4 were never started — the M2
gate requires both A1 restores and resumes to pass first.

Section 19 tally: **8 PASS, 1 FAIL, 12 NOT TESTED** of 21 mandatory rows.
All 21 passed: **no**.

Release-blocking findings: **3**

- F2 — `rein status` exits `0` with an empty revision when the remote manifest
  is absent, converting a Device B misconfiguration into a silent success
  (§8.2).
- F3 — `rein init --profile-id` does not verify the named remote profile exists
  at the supplied coordinates, so a mistyped endpoint yields a device that
  reports `config valid` while joined to nothing (§8.5).
- F1 — `rein init` silently overwrites an initialized home's `config.toml` and
  `state.json` with no confirmation and no backup (§5, §8.4).

The Windows W1 gate failure itself is **recoverable test state**, not a sync-path
binary defect: its root cause is an endpoint paste error at Device B's init
(§8.4). Device B can re-init with the corrected endpoint and rerun W1 onward.
F1, F2, and F3 block the release regardless of that rerun.

Non-blocking findings: **1** (N1, installer waits indefinitely on an unattended
readable `/dev/tty`), plus deviations D1–D5 and the revision/snapshot UUID
naming observation in §3.6.

Unresolved: the root cause of the Device B manifest miss is undetermined between
an operator storage-coordinate mismatch at init and a `--profile-id` handling
defect (§8.3). Two sanitized diagnostics are outstanding. Phase 1 cannot be
reassessed until they land and the failed gate plus every downstream gate is
rerun on a new RC if binary behavior changes.
