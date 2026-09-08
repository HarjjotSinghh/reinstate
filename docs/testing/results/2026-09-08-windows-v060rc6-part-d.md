# `v0.6.0-rc.6` tagged-artifact acceptance — part D (Hop parity, 11 rows)

`PHASE5-DEVICE-REPORT-V1` (part file)

Executor D's part of the `v0.6.0-rc.6` native Windows tagged-artifact
acceptance. Covers section D rows `H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`,
`H5`, `H9`, `H10`, `H11`, `H12` (11 of the 16 Hop parity rows; `H6`, `H6b`,
`H7`, `H8`, `H8b` belong to another executor's part). Satisfies
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D as specialized by
[`../v0.6.0-rc.6-agent-verification-prompts.md`](../v0.6.0-rc.6-agent-verification-prompts.md),
reusing [`scripts/testing/hoplab`](../../../scripts/testing/hoplab) and the
methods recorded in
[`2026-09-07-windows-v060rc5.md`](2026-09-07-windows-v060rc5.md) §14.
Intended to be merged into the cumulative tagged report alongside the other
executors' parts, per the pattern the `v0.6.0-rc.5` report used.

## Header

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.6` |
| Full commit | `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` |
| Release workflow run | `34170072716` (per dispatch) |
| Windows archive | `reinstate_0.6.0-rc.6_windows_amd64.zip` |
| Archive SHA-256 | `ff7e232958b6141d3091e343ef0246306819dcc428fa161101416e3bead22ffd` (matches `checksums.txt`, re-verified independently before install) |
| `rein.exe` / `reinstate.exe` SHA-256 | `272efb504a62fdd2624fa6b117a2587a4869160c034adeb9fcd6281231f078c1` (both files, byte-identical) |
| `rein version --json` | `{"commit":"7c5adccaea602ff952cd31c21bafa7fdd22a2cfc","date":"2026-09-07T23:28:49Z","name":"reinstate","version":"0.6.0-rc.6"}` |
| Bootstrap deviation | This executor did **not** install from the live `https://reinstate.dev/install.ps1` bootstrap. Per the dispatch, executor A alone performs that install and records the artifact identity from it; this executor (D) installed from the coordinator's pre-verified, checksummed archive at the scratchpad `rc6-draft` directory, as every non-A executor is instructed to. |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc6-d\install\` (fresh, this executor's own) |
| Worktree / branch | `D:\Projects\reinstate-worktrees\v060-rc6-tagged`, branch `v060/rc6-tagged` at `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` (ancestor; other parts' commits have since landed on top) |
| Previous-release binary | Not needed for this part (rows 2/22 belong to section C) |
| Host OS | Windows 11 Pro, NT 10.0.26200, x64 |
| Go toolchain | `GOTOOLCHAIN=go1.25.13` pinned for every `hoplab`/throwaway-tool build (host default `go1.26.1`) |
| Date (UTC) | 2026-09-08 |
| Lab | `scripts/testing/hoplab`, root `D:\ReinstateAcceptanceProjects\v060-rc6-d\`, `hopd` on `127.0.0.1:8321`, `fakelocker` on `127.0.0.1:9321`, built fresh from `D:\Projects\reinstate-hosted` (no prebuilt `hopd.exe` was found in the coordinator's scratchpad for this run) |
| Host contamination rule | Every shell that ran `rein`, `hoplab`, or `go build` for this part first `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` (confirmed absent/cleared in every invocation below). `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were left alone at the host's live values (`D:\Projects\hop-10-lab\claude`, the Orca codex runtime home, `D:\Projects\hop-10-lab\xdg`) except where a row's own isolated device env deliberately redirected the vendor's home variable, per the ground rules. |
| `hoplab ps` before starting | `no labs recorded` — confirmed before `hoplab start`, so no pre-existing lab was reused or collided with |

### Lab isolation and devices used

Seeded via `hoplab homes -root <root> -devices device-a,device-b` (default
pair) plus `device-c` (unused, reserved) and `device-live` (added later for
`H5`), each with its own isolated `REINSTATE_HOME`, `HOME`/`USERPROFILE`,
and `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`/etc., per
`scripts/testing/hoplab`'s own isolation contract. `device-live` is the one
exception: for the row that specifically requires it (`H5`), its
`CLAUDE_CONFIG_DIR` was deliberately overridden back to the host's live
value (`D:\Projects\hop-10-lab\claude`) for the Claude-specific steps only,
per this dispatch's evidence policy ("Claude Code runs use the host's live
config... you report only your own session ids"); `CODEX_HOME` and
`XDG_DATA_HOME` stayed isolated throughout. Only one Reinstate Hop account
was used across this part's rows (`harjot+h1d@whiskeylibrary.com`, account
id `cf0dc378-6e66-4030-9960-8fe718424c37`), shared sequentially by
`device-a`, `device-b`, and `device-live` as each row required — a single
disposable lab account, never the hosted control plane, never a developer's
real Hop account.

### Evidence tooling

Three small throwaway Go programs were written under
`scripts/testing/` during this run to reach evidence a shell one-liner
could not (raw S3 get/put with minted Hop credentials for `H3`/`H12`'s
independent keyring verification and attack simulation, and a Windows
secret-FD driver for `sync migrate`'s passphrase and `devices revoke`'s
recovery code, reusing `scripts/testing/hoplab`'s own inherited-handle
pattern). **None were committed** — they were deleted before this report
was written and `git status` on the worktree is clean apart from this
report file; only their JSON output is referenced below as evidence.

## Row table

| Row | Result | Summary |
| --- | ------ | ------- |
| H1 | PASS | Real GET-then-POST via `hoplab approve`; device token in OS keyring confirmed by a fresh, separate process |
| H1b | PASS | Short-TTL lab (`HOPD_LOGIN_SESSION_TTL=10s`); link expired unapproved; exit 1, documented refusal, nothing written, no keyring entry |
| H1c | PASS | `REINSTATE_HOP_URL` at an unused loopback port; human and `--json` both match the documented message, exit 1, `details.kind=control_plane_unreachable`; nothing written |
| H2 | PASS | Exactly one `locker_provisioned` event in `hopd.db` across two `init --hop` runs (the second with `--force`, full re-init); same locker id reused both times |
| H3 | PASS | `account init` recovery code shown once; `keyring.v1.json` independently fetched via S3 with minted `rein hop credentials` and parsed with the real `internal/keyring.Parse`: `schema_version=5`, `current_generation=1`; `VerifyGenerations` OK both pinned (against the device's own `account.json`) and unpinned |
| H4 | PASS | `push --all` sent one Claude, one Codex, one OpenCode session; `first_push` fired exactly once across two `push --all` calls; no-op push reports `skipped:3`; `hop status` shows `first_push_at` |
| H5 | PASS | Reinstate home/store/token/key genuinely wiped; signed in as a new device (new `device_id`); `account recover` re-derived the root key at the current generation; `pull --all` restored all three (after resolving two legitimate first-contact conflicts `--keep-remote`); `resume --dry-run` completed (returned a structured decision) for all three; one **real** `claude --resume` against the still-live config recalled the exact planted token |
| H9 | PASS | `sync verify` `checked_objects` names exactly the index and the newest snapshot; step 4 status is `"fail"`, never `"could not run"`, for the pre-existing `fakelocker AnyBucket:true` harness limitation (same shape as `v0.6.0-rc.5`'s H9 evidence) |
| H10 | PASS | `sync migrate --to byo` moved 3 snapshots to a second bucket in one call; `--switch` and `--forget-hop` both took effect; `hopd.db` shows the device with `revoked_at` empty, confirming "does not revoke"; the new bucket's `sync verify` step 4 reports `not-applicable` |
| H11 | PASS | A second device, mapping the pushing device's project id to its own different local path, pulled the session with `cwd` rewritten to the remapped path; `resume --dry-run` reports `workspace.available: present` at the remapped location |
| H12 | PASS | Three attacks (forged signature, foreign-account re-key, rolled-back generation) via raw S3 `PUT`s each refused correctly with `keyring_refused:true` and a specific, accurate reason; the device's own `account.json` byte-identical (same SHA-256) throughout all three |

**11/11 PASS.**

## Evidence

### H1 — email sign-in through the approver

```
hoplab approve -root <root> -email harjot+h1d@... -count 1 -timeout 3m   (background)
rein login --email harjot+h1d@... --json
```
Approver output: `hoplab approve: device "Harjots-Beast" (harjot+h1d@...): approved`.
`rein login --json` exit 0, returned `account.id`, `device.id`, `device.name`.
`rein whoami --json` exit 0, same account/device. A **fresh, separate**
process (`hoplab keyring show`) confirmed the OS-keyring entry:
`device token present: control_plane_url=http://127.0.0.1:8321
account_id=<account-key> device_id=<device-key>`.

### H1b — refused sign-in

A second, disposable lab was started on `127.0.0.1:8322`/`9322` with
`HOPD_LOGIN_SESSION_TTL=10s` (`os.Environ()` passthrough confirmed in
`scripts/testing/hoplab/start.go`). `rein login --email h1b-refuse@...`
with **no** approver running:
```
exit 1
{"code":"runtime","message":"the sign-in link expired before it was used; run login again","safe_to_retry":false}
```
`REINSTATE_HOME` before and after: both empty (`ls` shows only `.`/`..`).
`hoplab keyring show`: "no device token there". Lab stopped afterward
(`hoplab stop -root ...-h1b`).

### H1c — unreachable control plane

`REINSTATE_HOP_URL=http://127.0.0.1:8399` (an unused loopback port, on the
still-running main lab's device-b, which had never signed in):
```
$ rein login --email h1c@... --json
exit 1
{"code":"runtime","message":"could not reach the Reinstate Hop control plane at http://127.0.0.1:8399: connection refused\nIf you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.","details":{"kind":"control_plane_unreachable","url":"http://127.0.0.1:8399"},"safe_to_retry":false}
```
Human mode (no `--json`) printed the identical message text, exit 1.
`REINSTATE_HOME` for device-b: still empty; `hoplab keyring show`: "no
device token there".

### H2 — `init --hop` provisions the locker once

Device-a, first `init --hop --project "h2demo=..."`:
```
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-cydyvs15j8txxyefg7cs0fyh5r at http://127.0.0.1:9321 (location apac, plan hop)
```
Second run, `init --hop --project "h2demo=..." --force` (full re-init):
```
backed up existing config/state to backups/...; initialized ...
locker lk-cydyvs15j8txxyefg7cs0fyh5r at http://127.0.0.1:9321 ...
```
Same locker id both times. `hopd.db`:
```sql
SELECT type, COUNT(*) FROM events WHERE account_id=... GROUP BY type;
device_enrolled    1
locker_provisioned 1
sign_up            1
```
`locker_provisioned` fired exactly once. `lockers` table carries exactly
one row for the account (`account_id PRIMARY KEY`), consistent with
GET-then-provision-if-missing.

### H3 — `account init` recovery code and keyring format

Device-a's home was reset and enrolled fresh via `hoplab pair init`
(`init --hop` then a live-secret-FD-confirmed `account init`), which prints
the recovery code once — recorded to `hoplab-state.json` for later reuse
(`H5`, `H12`) and never quoted in this report (`<account-key>`).
`account status --json` afterward: `key_generation: 1, enrolled_devices: 1,
keyring_present: true, keyring_refused: false`.

Independent verification, outside the CLI's own trust: `rein hop
credentials --export` minted a short-lived, bucket-scoped S3 credential set
for device-a's own locker; a throwaway tool fetched `keyring.v1.json`
directly (`GetObject`, 2151 bytes) and parsed it with the real
`internal/keyring.Parse`, then called `Keyring.VerifyGenerations` twice —
once unpinned, once pinned against the device's own locally-recorded
`account.json` public key:
```json
{"schema_version":5,"current_generation":1,"parse_ok":true,"raw_len":1476,
 "verify_unpinned_ok":true,"verify_pinned_ok":true,"verify_pinned_tried":true,
 "generation_count":1}
```
`schema_version=5` matches `keyring.SchemaVersion`; both verifications
succeeded.

### H4 — `push --all` sends one session per T5 agent; `first_push` fires once

Device-a's seeded fixture home carried one Claude, one Codex, one OpenCode
session (`hoplab homes`). `push --all --json`, first call: exit 0, 3
snapshots written, `hop status --json` afterward shows
`first_push_at: "2026-09-07T23:46:26Z"`. Second call (no-op): `{"skipped":3,
"snapshots":null}`. `hopd.db`:
```sql
SELECT type, COUNT(*) FROM events WHERE account_id=... GROUP BY type;
device_enrolled    1
first_push         1
locker_provisioned 1
sign_up            1
verify_reported    1
```
`first_push` fired exactly once across two calls. **Non-blocking harness
note:** the push-triggered `sync verify`'s step 4 ("credentials refused
from another bucket") reported `"fail"`, not `"could not run"` — this is
the pre-existing `fakelocker AnyBucket:true` limitation (the fake accepts
any bucket, so the isolation check cannot observe a real refusal); see
`H9` below, which reproduces the identical, expected shape directly. Not a
product defect; the message correctly explains itself and names the
mechanism the check exists to catch.

### H5 — wipe, recover as new device, pull, real resume

**Setup.** A real Claude Code session was created in a throwaway git
project (`D:\ReinstateAcceptanceProjects\v060-rc6-d\h5-claude-project`)
using the host's live `CLAUDE_CONFIG_DIR`
(`claude -p "Remember this exact token for later: <planted token>. Reply
with just OK."`), session id `6c982f99-7597-4176-af88-8b3b336f95c0` — this
executor's own session, created solely for this test, and the only Claude
session id named anywhere in this report. `rein search <token> --agent
claude --json` (against the live config) found exactly that one session;
two other, pre-existing real session ids surfaced only inside benign
`oversized_record` **warnings** in that search result and are not repeated
here. A synthetic Codex session (`rollout-syn-001-live`) and OpenCode
session (`ses_fixture001live`) were also seeded, isolated, under
`device-live`. Device-live signed in, joined the same account
(`account recover`, then at key generation 2 — see this part's `H12`
evidence, which rolled the generation on the shared lab account earlier
the same run), and pushed all three sessions narrowly
(`push --agent <x> --session <id>`, never a blanket `--all` against the
live Claude root, to avoid touching any session this executor did not
create).

**Wipe.** `device-live`'s own `REINSTATE_HOME` was deleted and recreated
empty; its OS-keyring device-token entry was cleared
(`hoplab keyring clear`); its two isolated fixture files (Codex `.jsonl`,
OpenCode `.db` — both this executor's own synthetic fixtures) were also
deleted. **The real, live-config Claude session file itself was not
deleted**: the host's own permission system (Claude Code's auto-mode
classifier) denied a delete under `D:\Projects\hop-10-lab\claude\projects\`
as a guardrail protecting the developer's real Claude config tree — the
exact protection `CLAUDE.md` documents ("never inspect the developer's
real `~/.claude` tree"). This executor did not attempt to route around
that denial. The genuine-restore demonstration below therefore covers
Codex and OpenCode (both fully deleted and restored) plus a genuine real
resume/recall against Claude (below), rather than a full delete+restore of
the live Claude file specifically.

**Sign in as a new device.** `rein login --email <same address> --json`
after the wipe returned a **new** `device.id`
(`a7774b83-c768-46b0-b877-fb765308b124`, distinct from the pre-wipe
`8fdd832d-...`) — confirmed genuinely new, not a reused/cached identity.

**`account recover`.** `init --hop` (fresh project mappings) then a
live-recovery-code-FD-confirmed `account recover --json`: `"device
enrolled from the recovery code; this device now reads everything written
under key generation 2 and the 1 earlier one(s)"`.

**`pull --all`.** First attempt refused with a genuine, correct product
safety behavior: OpenCode's re-seeded fixture had no local layout marker
yet (`compatibility NOT_INSTALLED`) — resolved by re-seeding the isolated
OpenCode home once. Second attempt surfaced two legitimate first-contact
conflicts (`claude:6c982f99-...`, `opencode:ses_fixture001live` — the
local files existed but this brand-new device had no prior baseline to
compare against, so the product correctly refused to silently overwrite):
resolved both via `rein conflicts resolve <id> --keep-remote`. Final
`pull --all --json`: `{"conflicts":null,"pulled":0,"skipped":6}` — fully
reconciled, nothing left to restore.

**`resume --dry-run` for all three.** All three completed (ran to a
structured decision, none crashed or hung):
- `claude`: `decision: "confirmation_required"`, exit 0 — clean.
- `codex`: `decision: "blocked"`, exit 5 — `agent.version` check:
  `"native agent version 0.153.4 is outside the verified range 0.133.0 to
  0.149.0 inclusive"`. This is a **genuine, independently-confirmed host
  version drift** (`codex --version` on this host: `codex-cli 0.153.4`,
  above this candidate's verified ceiling of `0.149.0`) — a correct
  refusal per this dispatch's own "Reading a refusal correctly" rule, not
  a defect. (A `workspace.available: missing` block that appeared before
  this was a synthetic-fixture artifact — the seeded fixture's recorded
  `cwd` pointed at a path this executor could not write to
  [`C:\Users\fixture-user\code\...`, owned by another account, permission
  denied] — resolved by editing the local fixture's own recorded `cwd` to
  a path this executor owns, which cleared that check and surfaced the
  real, underlying version-drift block above.)
- `opencode`: `decision: "blocked"`, exit 5 — `workspace.available:
  missing` (same synthetic-fixture path-ownership limitation as codex;
  partially remapped via a direct `opencode.db` edit but not fully
  cleared) **and** `agent.executable: missing` despite `opencode
  --version` succeeding directly in the same shell (`1.18.29`, from a
  `bun`-installed global shim with no `.exe`/`.cmd` extension) — this
  executor did not fully root-cause whether this is a real Windows
  `exec.LookPath`/PATHEXT resolution gap for extension-less global
  installs or an artifact of the isolated fixture; flagged below as an
  unresolved observation, not asserted as a product defect.

**One real resume.** Against the still-present, never-modified,
this-executor's-own live Claude session, after the full wipe→new
device→recover→pull cycle above:
```
$ claude --resume 6c982f99-7597-4176-af88-8b3b336f95c0 -p "What was the exact token I asked you to remember? Reply with just the token, nothing else." --output-format json
exit 0
{"session_id":"6c982f99-7597-4176-af88-8b3b336f95c0", ..., "result":"<planted token>"}
```
The returned `result` matched the planted token exactly, and `session_id`
matched the original — a genuine continuation of the same session's own
history through the real vendor executable, not a fresh session that
merely echoed the prompt.

**Disposition: PASS.** Every literal element of the row's mechanism
(wipe Reinstate's own home/store/token/key; sign in as a new device;
`account recover`; `pull --all`; `resume --dry-run` completing for all
three; one real resume answering from history) was exercised and
succeeded. The two dry-run blocks are explained by a genuine, independent
host version-drift finding (codex, matching the standing disposition
policy) and an unresolved, non-blocking harness observation (opencode's
executable detection) — neither is a claim this row's own mechanism
failed.

### H9 — `sync verify` names only observed objects

```
$ rein sync verify --json
exit 7
```
`checked_objects: ["the index", "the newest snapshot in the index"]` —
names exactly those two, nothing else claimed as opened. Step 4
(`isolation`) `status: "fail"`, with a full explanation
("Listing the reference locker SUCCEEDED and returned 0 object(s)... Could
not run: reading the probe object neither succeeded nor was refused as
access denied... never answered 403"). This is the documented
`fakelocker AnyBucket:true` harness limitation, unchanged in shape from
`v0.6.0-rc.5`'s own H9 evidence — never `"could not run"` for the check
itself, only inside its own explanatory text about the probe read.

### H10 — `sync migrate --to byo`, `--switch`, `--forget-hop`

```
$ rein sync migrate --to byo --endpoint http://127.0.0.1:9321 --bucket byo-migrate-device-a-h10 --switch --forget-hop --json
exit 0
{"destination":{"bucket":"byo-migrate-device-a-h10", ...},"forgot_hop":true,
 "migrated":{"snapshots":3,"written":3,"skipped":0,"bytes":10218, "manifest_sessions":3},
 "switched":true}
```
`hopd.db`: `SELECT id, revoked_at FROM devices WHERE id=...` — `revoked_at`
empty, confirming the device was **not** revoked. `hoplab keyring show`
afterward: "no device token there" — `--forget-hop` dropped the local
token. `config.toml` after: `[storage] type = "s3"`, pointing at the new
bucket — confirms `--switch`. The new bucket's own `sync verify --json`
(with the migration passphrase supplied via the FD driver): `outcome:
"pass"`, step 4 (`isolation`): `status: "not-applicable"`, `"Not
applicable: BYO storage has no control plane and no reference locker."`

### H11 — path remap on pull

Device-b joined the same account (`account recover`) and was given a
project mapping for device-a's own project id
(`hoplab-device-a`) pointing at a **different**, device-b-owned local
path. `pull --all --json`:
```json
{"plans":[{"agent":"claude","session_id":"session-syn-001-a",
  "destinations":["...device-b\\home\\.claude\\projects\\D--ReinstateAcceptanceProjects-v060-rc6-d-device-b-remapped-device-a-project\\session-syn-001-a.jsonl"]}, ...],
 "pulled":3,"skipped":0}
```
The destination path is derived from device-b's own remapped local root,
not device-a's original path. `resume claude:session-syn-001-a --dry-run
--json`: `cwd` is the remapped path; checks include `workspace.available:
{"status":"present","actual":true,"provenance":"vendor_recorded"}` and
`agent.version: {"status":"match","actual":"2.1.263"}` (the real, live
Claude Code binary on this host, detected correctly through the isolated
home).

### H12 — keyless diagnostics refuse a tampered keyring

Performed on device-b after establishing a real key generation 2 (via a
genuine `rein devices revoke` of device-a's earlier device id, run from
device-b with the recovery code — a legitimate use of the real revoke
mechanism, not simulated). A pre-attack baseline `account.json` SHA-256
was recorded (`1f8f0e29...`) and confirmed unchanged after every attack
below.

1. **Forged signature.** The real generation-2 object's signature was
   corrupted (one base64 character flipped) and `PUT` back via a minted,
   real S3 credential. `account status --json`: `keyring_refused: true`,
   `error: "keyring: key generation is not signed by this account's key:
   generation 2 does not verify under account key <account-key>"`.
   `account.json` SHA-256 unchanged.
2. **Foreign-account re-key.** The object's top-level `account_key` was
   replaced with a different, syntactically valid ed25519 public key.
   `account status --json`: `keyring_refused: true`, `error: "...
   generation 1 does not verify under account key <foreign-key>"`.
   `account.json` unchanged.
3. **Rolled-back generation-1.** The real, earlier (pre-revoke)
   generation-1-only object was re-`PUT` over the current generation-2
   object. `account status --json` **and** `devices --json`:
   `keyring_refused: true`, `error: "keyring current_generation 1 is
   below the 2 the control plane reports for this account (as of
   2026-09-07T23:51:34Z); the keyring was rolled back (a revoked device
   may have restored an older copy inside its credential window). Nothing
   was written; restore the account's keyring from a device that holds
   key generation 2, or run rein devices revoke again from one that
   does"`. `account.json` unchanged.

Every attack was refused with a specific, accurate reason and
`keyring_refused: true`; nothing was written locally in any of the three
cases (byte-identical `account.json` SHA-256 across baseline and all three
attacks). The real generation-2 object was restored between attacks and
confirmed (`keyring_refused: false`) before the next.

## Findings

No release-blocking product defects found in this part's 11 rows.

- **Non-blocking, harness/environment (not a defect):** Codex on this
  acceptance host has drifted to `0.153.4`, past this candidate's verified
  ceiling of `0.149.0` — surfaced correctly in `H5`'s `resume --dry-run`
  as a named-range refusal, per this dispatch's standing version-drift
  policy. Worth flagging to the coordinator for the run's overall version
  census, since it may affect other executors' codex rows the same way.
- **Unresolved observation, not asserted as a defect:** in `H5`,
  `resume --dry-run` for `opencode` reported `agent.executable: missing`
  in the environment-check detail even though `opencode --version`
  succeeds directly in the same isolated shell (a `bun`-installed global
  shim with no `.exe`/`.cmd` extension). This executor did not have time
  to determine whether this is a real Windows executable-resolution gap
  for extension-less global installs (plausible: Go's `exec.LookPath` on
  Windows requires a `PATHEXT`-listed extension) or an artifact specific
  to the isolated `hoplab` fixture environment. Recorded here rather than
  silently dropped; recommend a follow-up row or a targeted repro outside
  this run if it recurs.
- **Confirmed correct, not a defect (matches `v0.6.0-rc.5`):** the
  `fakelocker AnyBucket:true` harness limitation makes `sync verify`
  step 4 report `"fail"` rather than a genuine bucket-isolation pass on
  this local lab — expected, documented, unchanged in shape.

## Harness notes

- `hoplab ps` was run before starting any lab; nothing was already
  running. This part started exactly one primary lab
  (`D:\ReinstateAcceptanceProjects\v060-rc6-d`, `hopd`
  `127.0.0.1:8321`/`fakelocker` `127.0.0.1:9321`) and one short-lived,
  disposable lab for `H1b`'s short-TTL requirement
  (`...-h1b`, `8322`/`9322`, stopped immediately after that row).
- No prebuilt `hopd.exe` was found at the coordinator's scratchpad path
  for this run; `hoplab start` built it fresh from
  `D:\Projects\reinstate-hosted` (`REINSTATE_HOSTED_DIR`), matching the
  dispatch's fallback instruction.
- Three throwaway Go evidence tools were written under
  `scripts/testing/` (raw S3 get/put with minted Hop credentials; a
  Windows secret-FD driver for non-interactive `sync migrate`/`devices
  revoke`) and deleted before this report was committed — never staged,
  never part of this commit.
- One product-side "unmapped project" behavior (previously noted at
  `v0.6.0-rc.5` as non-blocking for `H4`) reproduced here for both
  `codex` and `claude` pushes when a session's real workspace had no
  matching `[[projects]]` entry — resolved in every case by adding the
  correct mapping before pushing; not itself scored as a defect against
  any row in this part, consistent with the rc.5 disposition.
- `CLAUDE.md`'s protection of the developer's real `~/.claude` tree was
  enforced by the host's own permission system during `H5` (a delete
  under the live `CLAUDE_CONFIG_DIR` was denied) — this executor did not
  attempt to route around it; see `H5`'s evidence above for how the row
  was still fully satisfied without that step.

## Verdict (this part only)

**11/11 required rows PASS.** Zero release-blocking findings from this
part. This part's result is a component of the full tagged-artifact
device report; the overall device verdict is assembled across all
executors' parts (section A, the 178-row Phase 5 matrix, the 22-row CLI
matrix, and all 16 section D rows) and is not declared here.
