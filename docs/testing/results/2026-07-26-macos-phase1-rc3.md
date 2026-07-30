# Phase 1 Acceptance — Device A (macOS) — v0.1.0-rc.3

**Report status:** FINAL — verdict FAIL
**Executed:** 2026-07-26, 05:48–06:40 IST (UTC+05:30)
**Device role:** Device A (macOS)
**Runbook authority:** `docs/testing/phase-1-mac-windows-acceptance.md` @ tag `v0.1.0-rc.3`

## 1. Verdict

**FAIL.**

Phase 1 fails on a reproducible synchronization defect found by Device B and
recorded there as **`RC3-WIN-F3`**: restoring a macOS-origin Claude session onto
Windows writes the target beneath the **Mac** project slug instead of deriving
the destination from the canonical project's Windows `local_root`. Reinstate
reports the pull as successful (exit `0`), but Claude Code cannot discover the
session — the mapped-project resume UI shows no conversations, and direct-ID
resume exits `1`.

That is the flagship Windows ↔ macOS path-remapping case failing while
reporting success, which makes it release-blocking regardless of how clean the
rest of the run looks.

**Everything Device A tested passed.** Runbook sections 2 through 9 all pass on
macOS against the published `v0.1.0-rc.3` — public installer, pre-init failure
contract, session discovery, `init`, `setup check`, `doctor --self-test`,
scoped dry-runs and pushes, and a remote manifest holding exactly the two
selected sessions. None of the Mac-side findings are release-blocking, and none
of them caused this verdict.

Also unresolved, and separate from the verdict:

1. **§10 was never completed on any live profile.** Encryption is assumed, not
   demonstrated, on all three profiles.
2. **§14–§17 never ran** — no A2 update, no Windows→Mac restore, no
   unchanged-session no-op, no conflict test. M2 through M4 of this executor's
   assignment are untested.
3. **Six findings are open** across both devices: `RC3-MAC-F1/F2/F3` and
   `RC3-WIN-F1/F2/F4`, none release-blocking on their own.

## 2. Executive summary

Sections 2–6 of the runbook completed and passed on macOS against the published
`v0.1.0-rc.3` release:

- the public installer route is live, pins and verifies two independent
  checksums, installs exactly `0.1.0-rc.3`, is idempotent, and adds no
  duplicate PATH entry;
- the installer's replacement guard refused an unconfirmed in-place upgrade and
  left the previously installed binary untouched;
- pre-init `rein setup check` fails honestly with exit `3` / `config missing`
  while still correctly reporting the platform and both agent adapters as
  `SUPPORTED`.

Section 7 is complete for both agents. Both disposable marker sessions are
identified and confirmed to carry the exact runbook markers, with the Codex
deviation disclosed in §6. Sections 8–10 and all of M2–M5 are not started,
pending credentials.

## 3. Release identity

| Field | Value | Verified |
| ----- | ----- | -------- |
| Release | `v0.1.0-rc.3` | yes |
| Tag object type | annotated/signed (`git cat-file -t` → `tag`) | yes |
| Tag → commit | `94cc1e23f2e67054cd6102180664d83776d2406f` | yes — matches assignment |
| Binary self-reported commit | `94cc1e2` | yes — matches the tag commit |
| Fix PR | #12 | not independently re-verified |
| Release workflow | actions/runs/30179243906 | not independently re-verified |
| Signed-tag verification | **PARTIAL** | see below |

**Signed tag — PARTIAL, deliberately not resolved.** `git verify-tag
v0.1.0-rc.3` returns `error: gpg.ssh.allowedSignersFile needs to be configured
and exist for ssh signature verification`. The tag *is* a real tag object, but
local trust material is not configured. Per instruction, signing trust was
**not** configured to manufacture a green result. This row must be verified on
a host with the allowed-signers file in place, or accepted from GitHub's own
"Verified" badge, which is out of scope for this device report.

## 4. Environment

| Field | Value |
| ----- | ----- |
| macOS version | 26.5.2 (build 25F84) |
| Architecture | `arm64` |
| Claude Code | 2.1.220 (in RC3 supported range 2.1.219–2.1.220) |
| Codex CLI | 0.145.0 (in RC3 supported range 0.133.0–0.145.0) |
| Git | 2.51.0 |
| Reinstate (before) | 0.1.0-rc.2 (`57c06f2`) |
| Reinstate (after) | 0.1.0-rc.3 (`94cc1e2`, 2026-07-25T23:21:41Z) |
| Timezone | IST (UTC+05:30) |

Both agent CLIs are inside the RC3 supported ranges, so the mutating acceptance
flow is **not** version-blocked.

## 5. Isolation

| Item | Value |
| ---- | ----- |
| `REINSTATE_HOME` (active, after rotation) | `$HOME/.reinstate-phase1-acceptance-rc3` |
| `REINSTATE_HOME` (retired, retained) | `$HOME/.reinstate-phase1-acceptance` |
| Project | `$HOME/Projects/reinstate-phase1-acceptance` |
| Canonical project ID | `local/reinstate-phase1-acceptance` |

The run used the canonical home through §9, then rotated to the `-rc3` home
after the Device B passphrase exposure (§9a). The retired home and its remote
prefix are retained, not deleted. The project directory is unchanged across the
rotation, and the canonical project ID and both session IDs are identical in
both profiles.

The canonical paths were inspected before reuse. `$HOME/.reinstate-phase1-acceptance`
contained only three files, all under `cache/selftest/`, left by an earlier
RC2-era run, and **no configuration file**. The pre-init failure test in §6 was
therefore still valid, and the `-rc3` fallback paths were not needed. The
project directory already contained `.git`, `README.md`, `CLAUDE.md` and
`.serena` from the earlier run and was reused without deletion.

Nothing outside these two paths was modified, except the intended upgrade of
`~/.local/bin/reinstate` from rc.2 to rc.3, which the runbook requires.

## 6. Gate results

### §3 Prerequisites

| Gate | Result | Exit | Evidence |
| ---- | ------ | ---- | -------- |
| macOS reports arm64/x86_64 | PASS | 0 | `arm64` |
| Both agent CLIs run | PASS | 0 | 2.1.220 / 0.145.0 |
| Versions inside RC3 ranges | PASS | — | both inside inclusive ranges |

### §5 Live public installer (macOS)

| Gate | Result | Exit | Expected vs actual |
| ---- | ------ | ---- | ------------------ |
| `HEAD https://reinstate.dev/install.sh` | PASS | 0 | expected 200 → `HTTP/2 200`, `content-type: text/plain`, `content-disposition: inline; filename="install.sh"`, served by Vercel |
| Bootstrap pin checksum | PASS | — | `installer checksum ok` |
| Release asset checksum | PASS | — | `checksum ok` for `reinstate_0.1.0-rc.3_darwin_arm64.tar.gz` |
| Installed version is exactly RC3 | PASS | 0 | `{"version":"0.1.0-rc.3","commit":"94cc1e2"}` |
| `rein` resolves | PASS | 0 | `$HOME/.local/bin/rein` (symlink → `reinstate`) |
| `reinstate` resolves | PASS | 0 | `$HOME/.local/bin/reinstate` |
| Second run is idempotent | PASS | 0 | `Reinstate v0.1.0-rc.3 is already installed at …`; no prompt, no re-extraction |
| No duplicate PATH entry | PASS | — | installer wrote **zero** PATH blocks (`# Reinstate CLI` count in `~/.zshrc` = 0 before and after both runs), because `~/.local/bin` was already on PATH — the correct no-op branch |

Both checksum validations required by §5 were observed as two distinct,
separately labelled steps: the bootstrap verifying its pinned copy of the
canonical installer, then the canonical installer verifying the release asset
against `checksums.txt`.

**Replacement-safety observation (positive, not a defect).** Because rc.2 was
already installed, the installer prompted
`Replace Reinstate 0.1.0-rc.2 with 0.1.0-rc.3? [y/N]` on `/dev/tty`. On a first
attempt where no answer was delivered, it printed
`refusing to replace existing Reinstate 0.1.0-rc.2; set REINSTATE_CONFIRM_REPLACE=1
after reviewing the version change`, exited `1`, and its cleanup trap removed
the staged temporary binary from `~/.local/bin`. The previously installed rc.2
binary was left byte-identical and fully functional. This is correct
overwrite-safety behaviour and is recorded as supporting evidence for the
"no silent overwrite" stop condition.

The upgrade was then completed by answering `y` at the genuine prompt.

### §6 Pre-init honesty

| Gate | Result | Exit | Expected vs actual |
| ---- | ------ | ---- | ------------------ |
| `rein setup check` before init | PASS | 3 | expected exit `3` with `config missing` → exact match |
| Config check | PASS | — | `- [fail] config: config missing` |
| Device not reported unsupported | PASS | — | `- [ok] device: darwin-arm64` |
| `agent.claude` | PASS | — | `- [ok] agent.claude: SUPPORTED (2.1.220)` |
| `agent.codex` | PASS | — | `- [ok] agent.codex: SUPPORTED (0.145.0)` |
| Keyring reachable | PASS | — | `- [ok] keyring: OS keyring provider reachable` |
| Summary | PASS | — | `summary: 1 check(s) failed` — only the expected one |

Both adapters report `SUPPORTED`, which satisfies the §8 adapter precondition
in advance and means no adapter-driven block exists.

### §7 Source sessions

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| Claude marker session created | PASS | agent replied exactly `REINSTATE-PHASE1-MAC-CLAUDE-A1`; exited cleanly |
| Claude session discoverable via metadata | PASS | `rein list --agent claude` grew 170 → 172 rows, new rows under the encoded project key for `$HOME/Projects/reinstate-phase1-acceptance` |
| Exactly one new Claude ID isolatable | PASS (after marker-presence disambiguation) | see `RC3-MAC-F1` |
| Codex marker session created | PASS (session reused from earlier same-day run) | see `RC3-MAC-F2` |

Candidate Claude IDs, both under the acceptance project, disambiguated by
**marker-presence count only** — permitted evidence under safety rule 5. No
transcript content was read, quoted, or recorded:

| ID | Marker hits | Size | mtime | Selected |
| -- | ----------- | ---- | ----- | -------- |
| `a36153a6-d70a-43ec-8dcf-7a3c6787ac56` | 5 | 102737 B | 06:22:06 | **yes** |
| `d5af45e6-1534-4778-b789-4b20dea58043` | 3 | 101339 B | 06:21:55 | no |

`CLAUDE_SESSION_ID = a36153a6-d70a-43ec-8dcf-7a3c6787ac56` — the complete
session including the assistant's marker reply. The 3-hit sibling is the
pre-reply file. The identical 5/3 split appears in the earlier same-day run
(`2b3b7185…` = 5, `8ad99d38…` = 3), confirming the pattern.

`CODEX_SESSION_ID = 019f9b4d-d8d2-79e1-9e23-810786676f5a`.

**Disclosure — the Codex session was not created fresh in this run.** Codex CLI
0.145.0 refused to persist any new session on this host (see `RC3-MAC-F2`), so
the marker session created at 03:33 the same day from the same acceptance
project was reused. It qualifies on every substantive requirement: it is
disposable, it is scoped to `$HOME/Projects/reinstate-phase1-acceptance`, it
carries the exact runbook marker `REINSTATE-PHASE1-MAC-CODEX-A1` (5
occurrences), and it is the **only** Codex session Reinstate maps to the
acceptance project. It is not fresh, and that deviation is recorded here rather
than hidden.

### §8 Post-init setup and self-test

`rein init` was run by the operator on 2026-07-26. Storage coordinates and
credentials were entered at Reinstate's hidden prompts; the executor never
received them. Reinstate reported that credentials went to the OS keyring
(`credential_ref=reinstate/<profile-id>/s3`) and that the **passphrase is not
stored** and is re-prompted on every push and pull.

`PHASE1_PROFILE_ID = 47e43f49-35ea-49b1-a269-fb7cd8ee41a8`

The endpoint host and bucket name are deliberately omitted from this report.

| Gate | Result | Exit | Expected vs actual |
| ---- | ------ | ---- | ------------------ |
| `rein init` completes | PASS | 0 | printed non-secret `profile_id`, wrote `config.toml` + `state.json` |
| Passphrase not persisted | PASS | — | `Passphrase is not stored; you will enter it on push/pull` |
| Credentials in OS keyring | PASS | — | `credential_ref=reinstate/<profile-id>/s3` — not written to config in plaintext |
| `rein setup check` | PASS | 0 | `summary: all checks passed`; `config: config valid` |
| `agent.claude` SUPPORTED | PASS | — | `SUPPORTED (2.1.220)` |
| `agent.codex` SUPPORTED | PASS | — | `SUPPORTED (0.145.0)` |
| Keyring reachable | PASS | — | `OS keyring provider reachable` |
| `rein doctor --self-test` | PASS | 0 | `self_test: synthetic self-test passed` |

**Bucket-dedication deviation.** Runbook §3 asks for a bucket dedicated to this
test. The operator elected to use an existing bucket named `reinstate` in an
existing R2 account. The risk was raised before init and accepted. Consequence:
the §20 cleanup instruction to delete `profiles/PHASE1_PROFILE_ID/` must be
applied to that prefix **only**, never to the bucket.

### §8/§9 Push of the two selected sessions

Run by the operator; the passphrase was entered at Reinstate's hidden prompt
each time. Every push and pull re-prompts, and `rein push --help` exposes no
passphrase flag. The only non-interactive path is `REINSTATE_PASSPHRASE_FD`,
which is neither a hidden prompt nor the OS credential store, so it was
excluded under safety rule 2.

| Gate | Result | Exit | Expected vs actual |
| ---- | ------ | ---- | ------------------ |
| Claude dry-run uploads nothing | PASS | 0 | `dry_run=true`; no upload — see reasoning below and `RC3-MAC-F3` |
| Claude real push reports one snapshot | PASS | 0 | `pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false` |
| Codex dry-run uploads nothing | PASS | 0 | `dry_run=true` |
| Codex real push reports one snapshot | PASS | 0 | `pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false` |
| Manifest holds exactly the two selected sessions | PASS | 0 | `rein status` → `remote revision: 26897d25-… (2 sessions)` |
| No unrelated session uploaded | PASS | 0 | only the two selected IDs appear; 170 other Claude and 464 other Codex local sessions were not touched |
| `--all` never used | PASS | — | every command was `--session`-scoped |

Snapshot mapping reported by `rein status`:

| Agent | Session ID | Snapshot ID |
| ----- | ---------- | ----------- |
| claude | `a36153a6-d70a-43ec-8dcf-7a3c6787ac56` | `c0a8c645-733e-4fd1-ab7c-e9174a8368a3` |
| codex | `019f9b4d-d8d2-79e1-9e23-810786676f5a` | `26897d25-4fbc-41fd-bb26-f678afc0e38e` |

Remote revision: `26897d25-4fbc-41fd-bb26-f678afc0e38e`.

**How "dry-run uploaded nothing" was established.** The dry-run output text is
misleading (see `RC3-MAC-F3`), so the gate was verified behaviourally instead:
each dry-run was immediately followed by a real push of the same unchanged
session, and every real push reported `pushed 1 snapshot(s), skipped 0
unchanged`. Had the dry-run actually uploaded, the real push would have
detected identical content and reported `skipped 1 unchanged` — which is
exactly the signature §16 relies on. It did not. This is indirect but
behaviourally conclusive.

### §10 Ciphertext-only remote storage — PARTIAL

Reinstate exposes no remote object-listing command (`rein --help` offers only
`conflicts`, `diff`, `doctor`, `init`, `list`, `pull`, `push`, `setup`,
`status`, `version`), so this gate depends on operator inspection of the bucket
through the R2 dashboard.

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| `manifest.age` exists under the profile prefix | PASS | operator-confirmed in the R2 dashboard |
| `snapshots/` holds opaque `<uuid>.age` objects | PASS | operator-confirmed; two objects, matching the two pushed snapshots |
| No `auth.json`, `.env`, token or credential object under the prefix | PASS | operator-confirmed; nothing credential-shaped present |
| Neither plaintext marker appears in downloaded bytes | **NOT TESTED** | no `.age` object was downloaded |
| Snapshot is not readable JSON/JSONL | **NOT TESTED** | same |

**This gate is not PASS.** The structural half is confirmed; the substantive
half — that the stored bytes are actually ciphertext and contain neither
`REINSTATE-PHASE1-MAC-CLAUDE-A1` nor `REINSTATE-PHASE1-MAC-CODEX-A1` — was not
executed. Filenames ending in `.age` are naming, not proof of encryption. The
runbook is explicit that seeing transcript plaintext would be a release
blocker, so the check that would detect it must actually be run before Phase 1
can be signed off.

Supporting local evidence, gathered by the executor:

- no `.age` artifact is written to the local Reinstate home — encryption and
  upload are streamed, so nothing plaintext or ciphertext is cached on disk;
- `state.json` contains only agent name, session ID, an opaque local revision
  hash, the remote snapshot ID and timestamps — no credential material, no
  passphrase, no transcript content;
- credentials were placed in the OS keyring under
  `credential_ref=reinstate/<profile-id>/s3`, not written into `config.toml`.

**The structural confirmation above applies to the retired prefix
`profiles/47e43f49-…/` only.** After the §9a rotation, the live data lives under
`profiles/72733dd3-ee7c-45bd-89a6-5e448108367f/`, which has **not** been
inspected at all. Both halves of §10 are therefore outstanding for the profile
that actually matters.

**To close this gate:** inspect `profiles/72733dd3-…/` in the R2 dashboard,
confirm `manifest.age`, two opaque `.age` snapshots and no credential-shaped
objects, then download one `snapshots/<uuid>.age` and run, locally, a
marker-presence count plus `file(1)`. Report only booleans. Do not commit the
downloaded object.

### §11–§17 (Windows and cross-device)

**NOT TESTED.** No Windows device in scope for this executor, and every one of
these sections depends on a completed §8.

### §18 Automated integrity gates

**NOT VERIFIED LOCALLY.** The required GitHub check states for PR #12 and the
release workflow were not independently re-fetched by this executor and are not
claimed either way.

## 7. Evidence categories

| Category | Status |
| -------- | ------ |
| Locally verified on Mac by the executor | §3, §5, §6, §7 |
| Run by the operator, output verified by the executor | §8, §9, and the structural half of §10 |
| Reported by Windows handoff | none — no handoff received |
| Not tested | byte-level half of §10, §11–§18, M2–M5 |

**Ciphertext-only evidence:** partial — see §10. Structure confirmed, bytes not
inspected.
**Backup / overwrite-safety evidence:** partial — installer-level overwrite
refusal captured (§5). A `backups/` directory now exists under the acceptance
home but is empty; no restore has been performed, so Reinstate's own backup
behaviour is untested.
**No-op evidence:** none — §16 requires a completed pull first.
**Conflict evidence:** none.

## 8. Blocking findings

### RC3-MAC-B1 — no storage credentials or passphrase available (release-blocking for this run, not a product defect)

- **Severity:** blocker (process), not a product defect
- **Release-blocking:** no — does not indicate a fault in `v0.1.0-rc.3`
- **Where:** runbook §8, `rein init`
- **Expected:** operator runs `rein init --project "local/reinstate-phase1-acceptance=$PHASE1_PROJECT"` and enters the S3/R2 endpoint, bucket, access key ID, secret access key and encryption passphrase at Reinstate's hidden prompts
- **Actual:** not run. This executor holds no bucket coordinates and no
  credentials, and the acceptance safety rules forbid credentials or
  passphrases being passed through the agent channel, placed in command
  arguments, or entered anywhere other than Reinstate's own hidden prompts or
  the OS credential store
- **Consequence:** §8, §9, §10 and all of M2–M5 cannot execute. No profile ID,
  no snapshots, no manifest, no ciphertext check
- **Next step:** a human runs `rein init` in a real terminal with
  `REINSTATE_HOME=$HOME/.reinstate-phase1-acceptance` exported, then reports
  only the non-secret `profile_id`

### RC3-MAC-F2 — Codex CLI 0.145.0 persists no rollout for new sessions

- **Status:** worked around — an equivalent marker session from earlier the same
  day was reused, so §7 is no longer blocked
- **Severity:** medium
- **Release-blocking:** **no** — this is Codex CLI behaviour in this
  environment, not Reinstate behaviour. Reinstate's Codex adapter reports
  `SUPPORTED` and lists 465 pre-existing Codex sessions correctly
- **Affected:** Codex CLI 0.145.0, macOS 26.5.2 arm64
- **Reproduction (sanitized):**
  1. `cd $HOME/Projects/reinstate-phase1-acceptance`
  2. `codex exec "Reply with exactly: REINSTATE-PHASE1-MAC-CODEX-A1"` — no `--ephemeral` flag
  3. `find "$HOME/.codex/sessions" -type f -mmin -3`
- **Expected:** a new `rollout-<ts>-<uuid>.jsonl` under
  `$HOME/.codex/sessions/2026/07/26/`, and `rein list --agent codex` growing by one
- **Actual:** the agent replies correctly with the marker, but **no** rollout
  file is written anywhere under `$HOME/.codex`. `rein list --agent codex`
  stays at 465 rows. Repeated twice with the same result
- **Exit code:** 0 (the command itself succeeds)
- **Also attempted, and also negative:**
  - the interactive `codex` TUI driven through `expect` — reported session
    `019f9bea-a2bd-7993-9b0e-0d9f7512f510` on exit, wrote nothing;
  - a **genuine human-run interactive TUI session** in Terminal.app
    (`codex --dangerously-bypass-approvals-and-sandbox`), which echoed the
    marker correctly and printed
    `To continue this session, run codex resume 019f9bfa-c24e-7dd0-bde1-6386c5a1ee0a`
    — yet still wrote no rollout, and `rein list --agent codex` stayed at 465
- **Where the data is not:** session id `019f9bfa…` does not appear in
  `sessions/`, `history.jsonl` (stale since 2026-07-17), `.codex-global-state.json`
  (stale since 2026-07-22), or any of `goals_1`, `logs_2`, `memories_1`,
  `state_5` SQLite files or their write-ahead logs. Codex reports a resumable
  session id for a session it has not stored anywhere discoverable
- **Evidence that persistence works on this host:** the 03:33 same-day session
  `019f9b4d-…` produced both a rollout
  (`sessions/2026/07/26/rollout-2026-07-26T03-33-18-019f9b4d-….jsonl`, 84603 B)
  **and** entries in `state_5.sqlite-wal` (3 marker hits, mtime 03:33:30), so
  the mechanism is functional and something changed between 03:33 and 06:44
- **Impact on Reinstate:** none observed. Reinstate's Codex adapter reports
  `SUPPORTED (0.145.0)`, lists 465 pre-existing Codex sessions, and correctly
  maps the one acceptance-project session. It cannot index what the agent never
  writes
- **Next investigation:** determine what differs between the 03:33 run and the
  06:44 run (Codex auth/usage-limit state, YOLO mode, or the failed
  `chrome_devtools` MCP startup were all present at 06:44). If Codex 0.145.0
  can silently stop persisting sessions, the RC3 supported-range claim for
  Codex deserves a caveat, because Reinstate cannot sync sessions that are
  never written to disk

## 9. Non-blocking findings

### RC3-MAC-F1 — one `claude -p` invocation produced two session files

- **Severity:** low
- **Release-blocking:** no
- **Affected:** Claude Code 2.1.220, macOS 26.5.2 arm64. Not a Reinstate defect —
  Reinstate lists faithfully what the agent wrote
- **Reproduction (sanitized):**
  1. record `rein list --agent claude | wc -l`
  2. `claude -p "Reply with exactly: REINSTATE-PHASE1-MAC-CLAUDE-A1"` from the acceptance project
  3. re-record the count and diff the listings
- **Expected:** exactly one new session ID, so §7's "copy the two new session
  IDs" instruction is unambiguous
- **Actual:** two new IDs, 11 s apart, 101339 B and 102737 B. The same
  doubling is visible in the earlier RC2-era run in the same directory
- **Exit code:** 0
- **Impact on the runbook:** §7 tells the tester to disambiguate by diffing
  before/after listings and explicitly forbids opening transcripts. When one
  invocation yields two IDs, that guidance is insufficient, and `rein list --json`
  exposes only `ID`, `Agent`, `ProjectID`, `Title`, `UpdatedAt`, `SizeBytes`,
  `Path`, `RelativePath` — no turn count or first-user-message digest that
  would settle it without reading the transcript
- **Resolved in this run by:** counting occurrences of the marker string in each
  candidate file without reading or recording any transcript content — a
  marker-presence confirmation, explicitly permitted by safety rule 5. The full
  session scored 5 hits, the pre-reply sibling 3
- **Recommended next investigation:** either add a non-content disambiguator to
  `rein list --json` (e.g. message count), or amend §7 to tell the tester to
  disambiguate by marker-presence count, which is already permitted evidence and
  is what actually worked here

### RC3-MAC-F3 — `push --dry-run` reports `pushed 1 snapshot(s)` when it uploaded nothing

- **Severity:** low (reporting/UX), high confusion risk during acceptance
- **Release-blocking:** no — behaviour is correct, only the wording is wrong
- **Affected:** Reinstate 0.1.0-rc.3, macOS arm64; expected on all platforms
- **Reproduction (sanitized):**
  1. `rein push --agent claude --session <ID> --dry-run`
- **Expected:** wording that makes clear nothing was uploaded — e.g.
  `would push 1 snapshot(s)` or `planned 1 snapshot(s)`
- **Actual:** `pushed 1 snapshot(s), skipped 0 unchanged, dry_run=true`. The
  verb `pushed` asserts an upload that did not happen; only the trailing
  `dry_run=true` field contradicts it
- **Exit code:** 0
- **Why it matters here:** runbook §8's mandatory result is "dry-run uploads
  nothing". Read literally, this output states the opposite of the gate it is
  supposed to satisfy. A tester following the runbook exactly could reasonably
  record a FAIL, or worse, record a PASS on the assumption that the tool means
  what it says
- **Likely cause:** the dry-run path reuses the success formatter of the real
  push path rather than a plan-specific one
- **Recommended next investigation:** branch the summary string on `dry_run`,
  and consider having §8 of the runbook state the exact expected dry-run text

## 9a. Passphrase exposure incident and profile rotation

During Device B §12, the wrong-passphrase negative test passed correctly:
Reinstate refused, exited non-zero, and mutated nothing. The **intended real
passphrase was then typed after Reinstate had already exited**, so PowerShell
received it as a command line. It was recorded in PowerShell history and
relayed into an agent conversation.

Device B read-only verification established the blast radius:

| Check | Result |
| ----- | ------ |
| Selected session files on Windows | 0 changed / 0 created |
| Isolated backups on Windows | 0 |
| Pull or resume performed | none |
| R2 credentials exposed | no — they stayed in hidden prompts and the OS keyring |

**No Reinstate defect is implicated.** The tool's refusal path behaved exactly
as specified; the exposure happened in the surrounding shell after the process
had exited.

**Consequences, applied:**

- encryption passphrase for profile `47e43f49-35ea-49b1-a269-fb7cd8ee41a8` is
  treated as compromised and is **not** reused or retried;
- that profile and its remote prefix are **retained, not deleted**, so the
  exposure can be reviewed later;
- because the acceptance bucket is shared rather than dedicated (see the §8
  deviation), anyone holding both bucket read access and the exposed passphrase
  could decrypt the two disposable marker sessions stored under that prefix.
  Their contents are the acceptance markers only — no proprietary data — but
  the combination is precisely the risk the dedicated-bucket requirement in
  runbook §3 exists to prevent;
- a fresh profile is created under an isolated home
  `$HOME/.reinstate-phase1-acceptance-rc3` with a new passphrase, reusing the
  same canonical project ID and the same two session IDs;
- Device B retries against `%USERPROFILE%\.reinstate-phase1-acceptance-rc3`.

**Additional containment identified by Device A:** the Windows Codex session in
which the passphrase was relayed now contains that secret in its transcript.
That session must never be pushed through Reinstate, or the compromised
passphrase would be encrypted into a snapshot and synced. It is not one of the
two selected acceptance session IDs, and `--all` is prohibited, so no push path
currently reaches it.

Tracked by Device B as `RC3-WIN-F1`.

### Rotated profile — re-verified on Device A

`PHASE1_PROFILE_ID = 72733dd3-ee7c-45bd-89a6-5e448108367f`
(supersedes `47e43f49-35ea-49b1-a269-fb7cd8ee41a8`)

Same endpoint and bucket, new prefix, new passphrase entered only at hidden
prompts. All §6/§8/§9 gates were re-run against the rotated home and passed
again:

| Gate | Result | Exit | Evidence |
| ---- | ------ | ---- | -------- |
| Pre-init `setup check` in clean `-rc3` home | PASS | 3 | `config missing`; adapters still `SUPPORTED` |
| `rein init --project …` | PASS | 0 | new `profile_id`; `credential_ref=reinstate/<profile-id>/s3` |
| Post-init `setup check` | PASS | 0 | `summary: all checks passed` |
| `doctor --self-test` | PASS | 0 | `self_test: synthetic self-test passed` |
| Claude dry-run / real push | PASS | 0 | `dry_run=true` then `pushed 1 snapshot(s), skipped 0 unchanged` |
| Codex dry-run / real push | PASS | 0 | same pattern |
| Manifest holds exactly 2 sessions | PASS | 0 | `remote revision: 5fb472b0-… (2 sessions)` |
| No secret material in `config.toml` | PASS | — | pattern scan for secrets and long base64 tokens returned 0 matches; credentials live in the OS keyring |

Rotated snapshot mapping:

| Agent | Session ID | Snapshot ID |
| ----- | ---------- | ----------- |
| claude | `a36153a6-d70a-43ec-8dcf-7a3c6787ac56` | `1d681f3f-52bb-4685-8a76-5eecb74d4da9` |
| codex | `019f9b4d-d8d2-79e1-9e23-810786676f5a` | `5fb472b0-9674-49c5-93fe-701c59184d74` |

Remote revision: `5fb472b0-9674-49c5-93fe-701c59184d74`.

The dry-run-uploads-nothing gate was again established behaviourally: each real
push that followed a dry-run of unchanged content reported `skipped 0
unchanged`, which it could not have done had the dry-run already uploaded.

## 9b. Device B handoff

Windows report: `docs/testing/results/2026-07-26-windows-phase1-rc3.md` on
branch `test/phase1-rc3-windows-report`, commit `340b51e`. Device B verdict:
**FAIL**. Device B environment: Windows 11 Pro 25H2, build `10.0.26200.8328`,
native 64-bit PowerShell 5.1 (not WSL), Claude Code `2.1.220` (upgraded from
`2.1.211` before any mutation), Codex CLI `0.145.0`, Git `2.52.0.windows.1`,
Reinstate `0.1.0-rc.3` commit `94cc1e2`.

A third profile was created during Device B recovery. Three profiles now exist,
all retained:

| Role | Profile ID | Mac home |
| ---- | ---------- | -------- |
| Retired (passphrase exposed) | `47e43f49-35ea-49b1-a269-fb7cd8ee41a8` | `$HOME/.reinstate-phase1-acceptance` |
| Replacement (superseded) | `72733dd3-ee7c-45bd-89a6-5e448108367f` | `$HOME/.reinstate-phase1-acceptance-rc3` |
| Controlled fresh (current) | `9e17efb0-400e-4002-b986-b79f3b7b08e5` | `$HOME/.reinstate-phase1-acceptance-rc3-r2` |

Controlled-fresh snapshots: Claude `05c43895-0b73-427b-b30c-2ca1b243690f`,
Codex `93ff151c-6511-4c01-bf6f-55c0a5b7b94e`, remote revision
`93ff151c-6511-4c01-bf6f-55c0a5b7b94e`.

### Device B findings, as reported

| ID | Summary | Release-blocking |
| -- | ------- | ---------------- |
| `RC3-WIN-F1` | acceptance passphrase exposed outside the hidden prompt | no |
| `RC3-WIN-F2` | replacement-profile decryption refused, exit `4` | no |
| `RC3-WIN-F3` | **Claude restore stranded under the Mac project slug** | **yes** |
| `RC3-WIN-F4` | Codex picker UUID search can select the wrong session | no |

### Device A note on `RC3-WIN-F2`

`RC3-WIN-F2` states that decryption was refused with exit `4` "on both Windows
and Mac" for profile `72733dd3-…`. Device A's own evidence for that profile is
the opposite and is recorded in §9a: four scoped pushes and one `rein status`
all returned exit `0`, and `rein status` reported `remote revision: 5fb472b0-…
(2 sessions)`. Encryption, upload and manifest read all demonstrably worked for
that profile on the Mac at push time.

Both records can be true. Encryption succeeded when the passphrase used at push
time was supplied; the later refusals occur when a different string is entered.
Device B's own stated likely cause — "a passphrase or test-state mismatch" — is
consistent with Device A's evidence, and the non-release-blocking
classification is correct. The distinction worth preserving is that this is a
passphrase-state problem, not evidence that RC3 cannot decrypt what it wrote:
Device A proved a successful encrypt-and-read round trip on that exact profile.

## 10. Section 19 sign-off checklist

Copied verbatim from the tagged runbook. `n/a (blocked)` means the gate was
never reached; it is **not** a pass.

| Gate | Result | Evidence |
| ---- | ------ | -------- |
Source of each row: **[A]** verified locally by Device A, **[B]** reported by
the Device B handoff, **[—]** not tested by either.

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| `install.sh` returns 200 and installs RC3 on Mac | PASS [A] | `HTTP/2 200`; `version 0.1.0-rc.3`, commit `94cc1e2`; both checksum layers |
| `install.ps1` returns 200 and installs RC3 on Windows | PASS [B] | HEAD 200; both checksum layers; exact RC3; no elevation prompt |
| Both installers are idempotent and PATH-safe | PASS [A][B] | Mac: re-run reports already installed, 0 PATH blocks written. Windows: two runs, PATH entry count `1` before and after |
| Pre-init missing-config failure is accurate | PASS [A][B] | exit `3`, `config missing`, platform and both adapters still reported correctly on both devices |
| Post-init setup check and self-test pass on both devices | PASS [A][B] | Mac: `setup check` exit 0, `self_test: synthetic self-test passed`, re-verified after profile rotation. Windows: same on the controlled-fresh profile |
| Claude setup prompt completes on the Mac | PARTIAL [A] | the §8 command sequence completed with all required results; the prompt was not executed verbatim inside a separate Claude session |
| Codex setup prompt completes on Windows | PASS [B] | prompt version 3 pinned to RC3; workflow completed |
| Only two selected test sessions reach the remote manifest | PASS [A][B] | Mac `rein status` → 2 sessions on every profile; Windows confirms 2 on the controlled-fresh profile |
| Remote manifest/snapshots are ciphertext-only | **NOT TESTED** [—] | structure only, and only on the retired prefix. No byte-level marker-absence inspection on any profile |
| Wrong passphrase fails without mutation | PASS [B] | exit `4`; selected targets `0/0`; isolated backups `0` before and after |
| Claude Mac-to-Windows resume succeeds | **FAIL** [B] | restored file exists but Claude cannot discover it; mapped-project UI empty, direct-ID resume exit `1` — `RC3-WIN-F3` |
| Codex Mac-to-Windows resume succeeds | PASS [B] | exact-ID resume exit `0`; mapped project opened; A1 marker visually confirmed |
| Active-agent overwrite is refused | NOT TESTED [—] | §14 never executed |
| Existing Windows target is backed up before restore | NOT TESTED [—] | §14 never executed |
| Claude Windows-to-Mac resume succeeds | NOT TESTED [—] | §15 never executed; no B1 markers were created |
| Codex Windows-to-Mac resume succeeds | NOT TESTED [—] | §15 never executed |
| Existing Mac targets are backed up before restore | NOT TESTED [—] | no pull was performed on the Mac; `backups/` is empty |
| Unchanged pushes skip without new snapshots | NOT TESTED [—] | §16 requires a completed pull first |
| Divergence records a conflict without overwrite | NOT TESTED [—] | §17 never executed |
| `--keep-both` preserves both branches | NOT TESTED [—] | §17 never executed |
| All required GitHub checks are green | PASS [B] | PR and release workflow metadata verified by Device B |

Tally: 10 PASS, 1 PARTIAL, 1 **FAIL**, 9 NOT TESTED.

**Phase 1 does not pass.** One mandatory row fails outright, nine were never
exercised, and the ciphertext-only row — the gate protecting the project's
central privacy claim — has no substantive evidence on any of the three
profiles.

## 11. What a human must do to unblock

1. **Fix `RC3-WIN-F3` and cut a new RC.** The Claude restore planner must derive
   the destination from the canonical project's destination-device `local_root`
   instead of replaying the snapshot's archived relative path. Binary behaviour
   changes, so runbook §19 requires a new RC and a rerun of the failed gate plus
   every downstream gate. A regression test should assert that a macOS-origin
   Claude session restored on Windows is discoverable by the normal
   `claude --resume` UI — not merely that a file was written.
2. **Do not let a successful exit code stand in for a successful restore.** The
   pull returned `0` while producing an unusable result. Whatever check is added
   should fail the pull itself, not just the test.
3. **Close §10 on the controlled-fresh profile.** Download one
   `snapshots/<uuid>.age` from `profiles/9e17efb0-…/`, run a marker-presence
   count for both A1 markers plus `file(1)`, record booleans. Until then the
   ciphertext-only claim is untested on every profile.
4. **Run §14–§17 after the fix** — A2 update, Windows→Mac restore with backup
   verification, unchanged-session no-op, and conflict keep-both recovery. None
   have ever been exercised.

Carry-forward caveats: the Codex session is the 03:33 same-day session rather
than a freshly created one (`RC3-MAC-F2`); the bucket is shared rather than
dedicated, so cleanup targets only the three profile prefixes
`47e43f49-…`, `72733dd3-…` and `9e17efb0-…`, never the bucket; and the retired
profile's passphrase remains compromised (`RC3-WIN-F1`).
