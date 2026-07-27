# Phase 1 acceptance — Device A (macOS) — v0.1.0-rc.5

Sanitized result note for
[`docs/testing/phase-1-mac-windows-acceptance.md`](../phase-1-mac-windows-acceptance.md)
as published at tag `v0.1.0-rc.5`.

This note deliberately contains **no** endpoint, bucket, access key, secret key,
passphrase, keyring value, agent auth material, transcript text, object name,
username, or absolute local path. Only non-secret IDs, counts, booleans,
versions, exit codes, redacted paths, and sanitized output appear here.

**Status: IN PROGRESS — M0 complete, M1 in progress.**

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

## 4. M2 — Mac update and push

_Pending `WINDOWS-RC5-W1-PASS`._

## 5. M3 — Windows-to-Mac updates

_Pending `WINDOWS-RC5-W2-READY`._

## 6. M4 — divergence and final verdict

_Pending._

## 7. Section 19 mandatory sign-off

An unexecuted row is `NOT TESTED` and never `PASS`. Rows owned by Device B are
reconciled from the Windows report in M4.

| # | Gate | Owner | Result | Evidence |
| - | ---- | ----- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC5 on Mac | A | PASS | §2.2 |
| 2 | `install.ps1` returns 200 and installs RC5 on Windows | B | _pending_ | |
| 3 | Both installers are idempotent and PATH-safe | A+B | _partial: Mac PASS (§2.2)_ | |
| 4 | Pre-init missing-config failure is accurate | A+B | _partial: Mac PASS (§2.3)_ | |
| 5 | Post-init setup check and self-test pass on both devices | A+B | _partial: Mac PASS (§3.2)_ | |
| 6 | Claude setup prompt completes on the Mac | A | PASS, with caveat **F-A2** | §3.2, §3.5 |
| 7 | Codex setup prompt completes on Windows | B | _pending_ | |
| 8 | Only two selected test sessions reach the remote manifest | A | PASS | §3.5 |
| 9 | Remote manifest/snapshots are ciphertext-only | A | PASS | §3.6 |
| 10 | Wrong passphrase fails without mutation | B | _pending_ | |
| 11 | Claude Mac-to-Windows resume succeeds | B | _pending_ | |
| 12 | Codex Mac-to-Windows resume succeeds | B | _pending_ | |
| 13 | Active-agent overwrite is refused | B | _pending_ | |
| 14 | Existing Windows target is backed up before restore | B | _pending_ | |
| 15 | Claude Windows-to-Mac resume succeeds | A | _pending_ | |
| 16 | Codex Windows-to-Mac resume succeeds | A | _pending_ | |
| 17 | Existing Mac targets are backed up before restore | A | _pending_ | |
| 18 | Unchanged pushes skip without new snapshots | A | _pending_ | |
| 19 | Divergence records a conflict without overwrite | A+B | _pending_ | |
| 20 | `--keep-both` preserves both branches | B | _pending_ | |
| 21 | All required GitHub checks are green | A | _pending_ | |

## 8. Findings

| ID | Severity | Summary |
| -- | -------- | ------- |
| F-A1 | Non-blocking | Release tag's tagger e-mail is not the allow-listed signing principal; SSH verification binds by key, not identity (§2.1) |
| F-A2 | Non-blocking, docs/prompt | Setup prompt v5 silently redirected a configured `REINSTATE_HOME` |
| F-A3 | Open, severity pending M3 | Codex sessions are not resolved to the canonical project ID, and adapter listing scope is asymmetric (§3.4) |

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

No release-blocking finding confirmed so far.
