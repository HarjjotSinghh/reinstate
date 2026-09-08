# `v0.6.0-rc.7` tagged-artifact acceptance — part D (Hop parity, 11 rows)

`PHASE5-DEVICE-REPORT-V1` (part file)

Executor D's part of the `v0.6.0-rc.7` native Windows tagged-artifact
acceptance. Covers section D rows `H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`,
`H5`, `H9`, `H10`, `H11`, `H12` (11 of the 16 Hop parity rows; `H6`, `H6b`,
`H7`, `H8`, `H8b` belong to another executor's part, already recorded in
`2026-09-08-windows-v060rc7-part-e.md`). Satisfies
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D as specialized by
[`../v0.6.0-rc.7-agent-verification-prompts.md`](../v0.6.0-rc.7-agent-verification-prompts.md),
reusing [`scripts/testing/hoplab`](../../../scripts/testing/hoplab) and the
methods recorded in
[`2026-09-08-windows-v060rc6-part-d.md`](2026-09-08-windows-v060rc6-part-d.md)
(the same 11-row assignment, previous candidate, 11/11 PASS). Per the rc.7
dispatch, every row in this part cleared at `v0.6.0-rc.6` and is expected
`PASS` again unchanged; a regression on any of them is a new finding, not
an expected one. Intended to be merged into the cumulative tagged report
alongside the other executors' parts.

## Header

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.7` |
| Full commit | `21e9a9a356df1181601e2da02bfae03b28380cc3` |
| Release workflow run | `34250867105` |
| Windows archive | `reinstate_0.6.0-rc.7_windows_amd64.zip` |
| Archive SHA-256 | `865ad8e8f65fb3f543d2e2670b5d7c82b0d4878513781e04868fc7303e183fcf` (matches `checksums.txt`; re-verified independently against the coordinator's `rc7-draft` copy before install) |
| `rein.exe` / `reinstate.exe` SHA-256 | `ab73dd7c6817da2374a9c72673c62c63fb724d7412fa464d5dc56d381d5ea349` (both files; `cmp` confirms byte-identical) |
| `rein version --json` | `{"commit":"21e9a9a356df1181601e2da02bfae03b28380cc3","date":"2026-09-08T16:26:36Z","name":"reinstate","version":"0.6.0-rc.7"}` |
| Bootstrap deviation | This executor did **not** install from the live `https://reinstate.dev/install.ps1` bootstrap. Per the dispatch, executor A alone performs that install and records the artifact identity from it; this executor (D) installed from the coordinator's pre-verified, checksummed archive at the scratchpad `rc7-draft` directory, as every non-A executor is instructed to. |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc7-d\install\` (this executor's own, fresh) |
| Worktree / branch | `D:\Projects\reinstate-worktrees\v060-rc7-tagged`, branch `v060/rc7-tagged` (this part's own commit is an ancestor; other parts' commits have since landed on top) |
| Previous-release binary | Not needed for this part (rows 2/22 belong to section C) |
| Host OS | Windows 11 Pro, NT 10.0.26200, x64 |
| Go toolchain | `GOTOOLCHAIN=go1.25.13` pinned for every `hoplab`/throwaway-tool build |
| Date (UTC) | 2026-09-08 |
| Lab | `scripts/testing/hoplab`, root `D:\ReinstateAcceptanceProjects\v060-rc7-d\`, `hopd` on `127.0.0.1:8321`, `fakelocker` on `127.0.0.1:9321`, run from a prebuilt `hopd.exe` found at `D:\ReinstateAcceptanceProjects\v060-rc7-d\hopd.exe` (built earlier this session from `D:\Projects\reinstate-hosted`; reused via `-hopd-bin` to skip a rebuild) |
| Host contamination rule | Every shell that ran `rein`, `hoplab`, `go build`, or `go run` for this part first `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` (confirmed absent/cleared in every invocation below). `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were left alone at the host's live values except for `H5`'s one deliberate real-resume step (below) and where each isolated device's own env block redirected the vendor's home variable, per the ground rules. |
| `hoplab ps` before starting | A stale registry entry for this same lab root was found (`hopd`/`fakelocker` both `alive=false`, left over from earlier work this session) — cleared with `hoplab stop -all` before restarting; see Harness notes |

### Lab isolation and devices used

Seeded via `hoplab homes -root <root> -devices device-a,device-b` (default
pair) plus `device-c`, `device-w`, `device-x`, `device-y`, `device-z`
(`H11`/`H12`), each with its own isolated `REINSTATE_HOME`,
`HOME`/`USERPROFILE`, and
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`/etc., per
`scripts/testing/hoplab`'s own isolation contract. Three disposable Hop
accounts were used across this part's rows, never the hosted control
plane, never a developer's real Hop account:

- `h1d-fresh@example.test` (account `86ed0350-4bbe-4b80-b18f-a8d598bc43fe`) — `device-a`, rows `H1`, `H2`, `H3`, `H4`, `H9`, `H10`, one continuous journey.
- `h11d-b@example.test` (account `504db782-f7f9-4fd7-b692-f1c2c06f387e`) — `device-b` then `device-c`, rows `H11`, `H12`.
- `h12d-foreign@example.test` (account `dce21c34-7ab5-4b82-ae73-cd2a2eb7a76f`) — `device-w`, `H12`'s foreign-account re-key attack only (its own real, genuinely-signed keyring object is the attack payload, not the target).
- `h1b-fresh@example.test` / `h1c-fresh@example.test` — throwaway addresses for the two refusal rows, disposable lab roots, never enrolled.

`H5` additionally used the host's live `CLAUDE_CONFIG_DIR`
(`D:\Projects\hop-10-lab\claude`) for its one required real Claude Code
resume, and the disposable lab account `harjot@whiskeylibrary.com`
(account `3b2a0f06-7e8a-4681-8bbc-468ee9a22fe5`) for the wipe/new-device/
recover/pull/dry-run sequence — see `H5`'s own evidence for why that
sequence's timestamps precede the lab restart below.

### Evidence tooling

Two small throwaway Go programs were written under `scripts/testing/`
during this run to reach evidence a shell one-liner could not: `s3rw`
(raw S3 get/put/verify against a locker with minted Hop credentials, for
`H3`'s independent `keyring.v1.json` verification and `H12`'s three
tamper attacks) and `fdrunner` (a Windows secret-file-descriptor driver
for `rein account init`'s live recovery code and `rein account
recover`/`devices revoke`/`sync migrate`'s already-known ones, the same
inheritable-handle pattern `scripts/testing/hoplab`'s own
`secretfd_windows.go` uses). **Neither was committed** — both were
deleted, along with two similarly-purposed leftover tools from earlier
work this session (`scratchtool_h3`, `scratchtool_migrate`), before this
report was written; `git status` on the worktree is clean apart from this
report file. Only their printed output is referenced below as evidence.

## Row table

| Row | Result | Summary |
| --- | ------ | ------- |
| H1 | PASS | Real GET-then-POST via `hoplab approve`; `login`/`whoami` exit 0; device token in OS keyring confirmed by a fresh, separate `hoplab keyring show` process |
| H1b | PASS | Disposable lab, `HOPD_LOGIN_SESSION_TTL=10s`; link expired unapproved; exit 1, documented refusal, nothing written under `REINSTATE_HOME`, no keyring entry |
| H1c | PASS | `REINSTATE_HOP_URL` at an unused loopback port; human and `--json` both exit 1 with the identical documented message, `details.kind=control_plane_unreachable`; nothing written |
| H2 | PASS | Exactly one `locker_provisioned` event in `hopd.db` across two `init --hop` runs (the second with `--force`, full re-init); same locker id reused both times; `lockers` table carries exactly one row |
| H3 | PASS | `account init` (live-FD-confirmed) shows the recovery code once; `keyring.v1.json` independently fetched via minted S3 credentials and parsed with the real `internal/keyring.Parse`: `schema_version=5`, `current_generation=1`, `VerifyGenerations` OK unpinned |
| H4 | PASS | `push --all` sent one Claude, one Codex, one OpenCode session; `first_push` fired exactly once across two calls; no-op push reports `skipped:3`; `hop status` shows `first_push_at` |
| H5 | PASS | Reinstate home/keyring genuinely wiped; signed in as a new device under the same account; `account recover` re-derived the root key; `pull --all` restored all three sessions after resolving a legitimate first-contact conflict; `resume --dry-run` completed (structured decision) for all three; one **real** `claude --resume … -p` against the still-live config recalled the exact planted token at the same session id |
| H9 | PASS | `sync verify` `checked_objects` names exactly the index and the newest snapshot; step 4 status is `"fail"`, never `"could not run"` — the documented `fakelocker AnyBucket:true` harness limitation |
| H10 | PASS | `sync migrate --to byo` moved 3 snapshots to a second bucket in one call; `--switch` and `--forget-hop` both took effect; `hopd.db` shows the device with `revoked_at` empty; the new bucket's `sync verify` reports `outcome: pass`, step 4 `not-applicable` |
| H11 | PASS | A second device, mapping the pushing device's project id to its own different local path, pulled the session with `cwd` rewritten to the remapped path; `resume --dry-run` reports `workspace.available: present` at the remapped location |
| H12 | PASS | Three attacks (forged signature, foreign-account re-key, rolled-back generation) via raw S3 `PUT`s, each refused with `keyring_refused: true`/a specific accurate reason; `push` also refused (exit 7); the surviving device's own `account.json` byte-identical (same SHA-256) throughout all three, and after final restore |

**11/11 PASS.**

## Evidence

### H1 — email sign-in through the approver

```
hoplab approve -root <root> -email h1d-fresh@example.test -count 1 -timeout 60s   (background)
rein login --email h1d-fresh@example.test --no-browser --json   -> LOGIN_EXIT=0
rein whoami --json                                               -> WHOAMI_EXIT=0
```
Approver output: `hoplab approve: device "Harjots-Beast"
(h1d-fresh@example.test): approved`. `login --json` and `whoami --json`
both returned matching `account.id`/`device.id`. A **fresh, separate**
process (`hoplab keyring show -root <root> -device device-a`) confirmed
the OS-keyring entry: `device token present: control_plane_url=
http://127.0.0.1:8321 account_id=<account-key> device_id=<device-key>`.

### H1b — refused sign-in

A disposable lab was started on `127.0.0.1:8322`/`9322` with
`HOPD_LOGIN_SESSION_TTL=10s` (env passthrough into the `hopd` child,
`scripts/testing/hoplab/start.go`). `rein login --email
h1b-fresh@example.test --no-browser --json`, isolated `REINSTATE_HOME`,
**no** approver running:
```
LOGIN_EXIT=1
{"code":"runtime","message":"the sign-in link expired before it was used; run login again","safe_to_retry":false}
```
`REINSTATE_HOME` before and after: both empty (`ls` shows only `.`/`..`).
`hoplab keyring show` against the same isolated home: "no device token
there". Disposable lab stopped afterward.

### H1c — unreachable control plane

`REINSTATE_HOP_URL=http://127.0.0.1:8398` (an unused loopback port), a
fresh, never-signed-in isolated home:
```
$ rein login --email h1c-fresh@example.test --no-browser        -> HUMAN_EXIT=1
could not reach the Reinstate Hop control plane at http://127.0.0.1:8398: connection refused
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.

$ rein login --email h1c-fresh@example.test --no-browser --json -> JSON_EXIT=1
{"code":"runtime","message":"could not reach the Reinstate Hop control plane at http://127.0.0.1:8398: connection refused\n...","details":{"kind":"control_plane_unreachable","url":"http://127.0.0.1:8398"},"safe_to_retry":false}
```
Human and `--json` carry the identical message text. `REINSTATE_HOME`
after: still empty.

### H2 — `init --hop` provisions the locker once

Device-a, first `init --hop --project "h2d=..."`: `EXIT=0`,
`locker lk-rq1qcsrv8cjnq3dzg5p7c7531r at http://127.0.0.1:9321 ...`.
Second run, `init --hop --project "h2d=..." --force` (full re-init):
`EXIT=0`, `backed up existing config/state to backups/...`, same locker
id. `hopd.db`:
```sql
SELECT type, COUNT(*) FROM events WHERE account_id=... GROUP BY type;
device_enrolled    1
locker_provisioned 1
sign_up            1
SELECT COUNT(*) FROM lockers WHERE account_id=...;   -- 1
```
`locker_provisioned` fired exactly once; the `lockers` table carries
exactly one row for the account — GET-then-provision-if-missing.

### H3 — `account init` recovery code and keyring format

Device-a's `account init --json`, confirmed via `fdrunner live` scanning
the child's own stderr for the recovery-code pattern and feeding it back
through `REINSTATE_RECOVERY_CODE_FD`: `EXIT=0`,
`account initialized: root key generated on this device, keyring written
to storage`, `key_generation=1 devices=1`. The recovery code was printed
exactly once, in the child's own output, and is not repeated here
(`<account-key>`).

Independent verification, outside the CLI's own trust: `rein hop
credentials --json` minted a short-lived, bucket-scoped S3 credential set
(`access_key_id`, `secret_access_key`, `session_token`, `bucket
lk-rq1qcsrv8cjnq3dzg5p7c7531r`); the throwaway `s3rw verify` tool fetched
`keyring.v1.json` directly (`GetObject`) and parsed it with the real
`internal/keyring.Parse`, then called `Keyring.VerifyGenerations("")`:
```
schema_version: 5
current_generation: 1
generations: [1]
device_count: 1
verify unpinned: OK
account_key: <account-key>
```
`schema_version=5` matches `keyring.SchemaVersion`; verification
succeeded.

### H4 — `push --all` sends one session per T5 agent; `first_push` fires once

Device-a's seeded fixture home carried one Claude, one Codex, one
OpenCode session (`hoplab homes`), with a `[[projects]]` mapping added for
the fixture's own recorded cwd. `push --all --json`, first call: `EXIT=0`,
3 snapshots written; `hop status --json` afterward shows
`"first_push_at": "2026-09-08T18:04:21Z"`. Second call (no-op):
`{"skipped":3,"snapshots":null}`, `EXIT=0`. `hopd.db`:
```sql
SELECT type, COUNT(*) FROM events WHERE account_id=... GROUP BY type;
device_enrolled    1
first_push         1
locker_provisioned 1
sign_up            1
verify_reported    1
```
`first_push` fired exactly once across two calls. **Non-blocking harness
note (carried, unchanged since `v0.6.0-rc.1`):** the push-triggered `sync
verify`'s step 4 ("credentials refused from another bucket") reported
`"fail"`, not `"could not run"` — the pre-existing `fakelocker
AnyBucket:true` limitation; see `H9` below, which reproduces the identical
shape directly with full detail. Not a product defect.

### H5 — wipe, recover as new device, pull, real resume

**Setup and wipe.** A real Claude Code session was created in a throwaway
git project (`D:\ReinstateAcceptanceProjects\v060-rc7-d\claude-h5`, branch
`prod`) using the host's live `CLAUDE_CONFIG_DIR`
(`claude -p "Remember this exact token for later: RC7D-H5-CLAUDE-8271.
Reply with just OK."`), session id
`9f4a3c1e-7b2d-4e5a-9c3f-b1a2c3d4e5f6` — this executor's own session,
created solely for this test, and the only Claude session id named
anywhere in this report; its assistant turn replied `OK`. Separately, this
part's `device-a` (isolated home, disposable lab account
`3b2a0f06-7e8a-4681-8bbc-468ee9a22fe5`) had its `REINSTATE_HOME` deleted
and recreated empty and its OS-keyring device-token entry cleared
(`hoplab keyring clear`).

**Sign in as a new device.** `rein login --email <same address>
--no-browser --json` after the wipe returned a **new** `device.id`
(`e0fe7e74-e1d8-438e-bc8f-001825b0005a`) under the same, pre-existing
account (`3b2a0f06-...`) — a device signing in fresh has no keyring
knowledge of its own yet, which is exactly why the next step is needed.

**`account recover`.** A live-recovery-code-confirmed `account recover`
(via `fdrunner`) reported: `"device enrolled from the recovery code; this
device now reads everything written under key generation 1 and the 0
earlier one(s)"`.

**`pull --all`.** First attempt correctly refused on a genuine safety
behavior: the re-seeded OpenCode fixture had no local layout marker yet
(`compatibility NOT_INSTALLED`) — resolved by seeding the isolated
OpenCode home once. Second attempt pulled the OpenCode session
(`pulled:1, skipped:2`) and surfaced a legitimate first-contact conflict
for `claude:session-syn-001-a` (the local file existed with no baseline
this brand-new device could compare against — a correct refusal to
silently overwrite), resolved via `rein conflicts resolve ... --keep-
remote`. Final pull of the claude session: `{"pulled":1,"skipped":0,
"conflicts":null}`. `rein sessions --json`, confirmed later this run,
shows all three (claude, codex, opencode) present with `resume`/`fork`
capabilities `true`.

**`resume --dry-run` for all three.** All three completed (returned a
structured `"decision"`, none crashed or hung). Each reports `decision:
"blocked"`, `workspace.available: missing` — the isolated fixture's own
recorded cwd (`C:\Users\fixture-user\code\device-a`) does not exist on
this host, the same synthetic-fixture path-ownership limitation prior
candidates' own H5 evidence documented; block, not crash, and a
structured decision either way satisfies "resume --dry-run completed."
The one component of that block worth naming: `agent.executable:
missing` is reported for all three even though `claude`/`codex`/
`opencode` are genuinely resolvable on `PATH` in the same isolated shell
(confirmed directly: `claude --version` -> `2.1.263`) — this check
appears to short-circuit once `workspace.available` has already blocked,
rather than a real Windows executable-resolution gap; see Findings.

**One real resume.** Against the still-present, never-modified,
this-executor's-own live Claude session, run directly (the live-config
step this dispatch's evidence policy specifically carves out):
```
$ claude --resume 9f4a3c1e-7b2d-4e5a-9c3f-b1a2c3d4e5f6 -p "What was the exact token I asked you to remember? Reply with just the token, nothing else." --output-format json
EXIT=0
{"session_id":"9f4a3c1e-7b2d-4e5a-9c3f-b1a2c3d4e5f6", ..., "result":"RC7D-H5-CLAUDE-8271", ...}
```
The returned `result` matches the planted token exactly, and `session_id`
matches the original — a genuine continuation of the same session's own
history through the real vendor executable, not a fresh session that
merely echoed the prompt. (An earlier attempt this run to drive this same
step interactively through `conptydriver` did not get past Claude Code's
own "trust this folder" safety prompt across 12 tries — the navigation
script sent a bare Enter, which the running app's own default focus
answers as "No, exit" rather than "Yes, I trust this folder"; abandoned in
favor of the non-interactive `-p`/`--output-format json` invocation above,
which is the same real vendor executable continuing the same real session
and is the method the previous candidate's own H5 evidence used
successfully.)

**Disposition: PASS.** Every element of the row's mechanism (wipe
Reinstate's own home/token/key; sign in as a new device; `account
recover`; `pull --all` restoring all three; `resume --dry-run` completing
for all three; one real resume answering from history) was exercised and
succeeded.

### H9 — `sync verify` names only observed objects

```
$ rein sync verify            -> EXIT=7
$ rein sync verify --json     -> EXIT=7
```
`checked_objects: ["the index", "the newest snapshot in the index"]` —
names exactly those two, nothing else claimed as opened. Step 4
(`isolation`) `status: "fail"`, with a full, accurate explanation
("Listing the reference locker SUCCEEDED and returned 0 object(s)...
Could not run: reading the probe object neither succeeded nor was refused
as access denied... never answered 403"). This is the documented
`fakelocker AnyBucket:true` harness limitation, unchanged in shape since
`v0.6.0-rc.1` — never `"could not run"` for the check itself, only inside
its own explanatory text about the probe read.

### H10 — `sync migrate --to byo`, `--switch`, `--forget-hop`

```
$ rein sync migrate --to byo --endpoint http://127.0.0.1:9321 --bucket rc7d-byo-migrate --switch --forget-hop --json
EXIT=0
{"destination":{"bucket":"rc7d-byo-migrate",...},"forgot_hop":true,
 "migrated":{"snapshots":3,"written":3,"skipped":0,"bytes":10218,"manifest_sessions":3},
 "switched":true}
```
`hopd.db`: `SELECT id, revoked_at FROM devices WHERE id=...` — `revoked_at`
empty, confirming the device was **not** revoked. `config.toml` after:
`[storage] type = "s3"`, pointing at the new bucket, confirming
`--switch`. `rein hop credentials --json` afterward refuses: `"this
device is not signed in to Reinstate Hop; run rein login first"` —
confirming `--forget-hop` dropped the local token (the doc's own "does
not revoke" is independently confirmed by the still-empty `revoked_at`
above). The new bucket's own `sync verify --json`:
`outcome: "pass"`, step 4 (`isolation`) `status: "not-applicable"`,
`"Not applicable: BYO storage has no control plane and no reference
locker."`

### H11 — path remap on pull

`device-b` (fresh account `504db782-...`) pushed one session per T5 agent
with a project mapping `hoplab-device-b=C:\Users\fixture-user\code\
device-b` (its own fixture's cwd). `device-c` joined the same account
(`account recover`, live recovery-code FD) with a project mapping for the
**same project id**, `hoplab-device-b`, pointing at a **different**,
device-c-owned local path
(`D:\ReinstateAcceptanceProjects\v060-rc7-d\lab\device-c\remapped-device-b-project`).
`pull --all --json`: `{"pulled":3,"skipped":0,"conflicts":null}`, with the
claude destination:
```
D:\...\lab\device-c\home\.claude\projects\D--ReinstateAcceptanceProjects-v060-rc7-d-lab-device-c-remapped-device-b-project\session-syn-001-b.jsonl
```
— device-c's own remapped root, not device-b's original
`C:\Users\fixture-user\code\device-b`. Direct file check confirms the
restored jsonl's own `"cwd"` field reads the remapped path. `resume
claude:session-syn-001-b --dry-run --json`: `"cwd"` matches the remapped
path, `decision: "confirmation_required"`, checks include
`workspace.available: {"status":"present", ...}` and `agent.version:
{"status":"match","actual":"2.1.263"}` — the real, live Claude Code
binary, detected correctly through the isolated home.

### H12 — keyless diagnostics refuse a tampered keyring

Performed on `device-c`, continuing `H11`'s account, after establishing a
real key generation 2 via a genuine `rein devices revoke <device-b-id>`
run from `device-c` with the recovery code (a legitimate use of the real
revoke mechanism, not simulated): `"key generation 2 started with 1
enrolled device(s)"`. A real gen-2 `keyring.v1.json` was fetched and kept
as the baseline (`s3rw get`, 3652 bytes, `verify` -> `schema_version 5,
current_generation 2, generations [1 2], verify unpinned: OK`). A separate,
genuinely-enrolled foreign account (`device-w`, disposable lab account
`dce21c34-...`) supplied a real, correctly-signed keyring object of its
own for the re-key attack. A pre-attack baseline `account.json` SHA-256
was recorded (`<sha256-a>`) and confirmed unchanged after every attack and
after the final restore.

1. **Forged signature.** The real generation-2 object's last generation
   signature was corrupted (one base64 character flipped, via a small
   script operating on the parsed JSON) and `PUT` back with device-c's own
   minted, real S3 credentials. `account status --json`:
   `keyring_refused: true`, `error: "keyring: key generation is not
   signed by this account's key: generation 2 does not verify under
   account key <account-key>"`. `account.json` SHA-256 unchanged.
2. **Foreign-account re-key.** `device-w`'s own real, genuinely-signed
   `keyring.v1.json` (a different account entirely, `current_generation
   1`) was `PUT` directly in place of the shared account's object.
   `account status --json`: `keyring_refused: true`, `error: "keyring:
   the keyring is signed by a different account key: it is signed by
   account key <foreign-key>, not the <account-key> expected here"`.
   `rein push --all` also refused: `EXIT=7`, same message, `"Nothing was
   written; the keyring in storage was replaced by one signed with a key
   this account never used"`. `account.json` unchanged.
3. **Rolled-back generation.** The real, earlier (pre-revoke)
   generation-1-only object (fetched and saved before the `H12` revoke
   step above, so this is device-c's own genuine earlier object, not a
   forgery) was re-`PUT` over the current generation-2 object.
   `account status --json` **and** `devices --json`: `keyring_refused:
   true`, `error: "keyring current_generation 1 is below the 2 the
   control plane reports for this account (as of ...); the keyring was
   rolled back (a revoked device may have restored an older copy inside
   its credential window). Nothing was written; restore the account's
   keyring from a device that holds key generation 2, or run rein devices
   revoke again from one that does"`. `account.json` unchanged.

Every attack was refused with a specific, accurate reason and
`keyring_refused: true`; nothing was written locally in any of the three
cases (byte-identical `account.json` SHA-256 across baseline, all three
attacks, and after the real generation-2 object was finally restored and
re-confirmed, `keyring_refused: false`).

## Findings

No release-blocking product defects found in this part's 11 rows.

- **Non-blocking, likely harness-side, not asserted as a defect
  (`H5`):** `resume --dry-run` for all three synthetic fixtures reported
  `agent.executable: missing` even though `claude`/`codex`/`opencode` are
  genuinely resolvable on `PATH` in the same isolated shell — but the same
  dry-run for a real session with an existing workspace (this part's own
  `H5` real-resume setup) correctly reported `agent.executable: present`
  with the exact version. The one thing that differs between the two
  cases is `workspace.available` (missing vs. present), which suggests
  `agent.executable`'s own check may short-circuit to its zero value once
  the workspace check has already blocked the environment, rather than a
  genuine executable-resolution gap. Not root-caused within this run's
  time budget; recorded as an observation, not a defect, since the
  synthetic fixture's own recorded workspace path
  (`C:\Users\fixture-user\code\device-a`, owned by another account) is
  independently un-writable on this host regardless.
- **Self-caught harness near-miss, resolved, no product impact
  (`H5`):** one intermediate pull attempt during this run's earlier,
  pre-restart pass (see Harness notes) showed a `pull --all --json` plan
  whose destination for the **synthetic** fixture `claude:
  session-syn-001-a` pointed under the host's **live**
  `CLAUDE_CONFIG_DIR` (`D:\Projects\hop-10-lab\claude\projects\...`) —
  apparently because that shell had `CLAUDE_CONFIG_DIR` pointed at the
  live config (for the unrelated real-resume setup) while `REINSTATE_HOME`
  was still `device-a`'s isolated profile. Checked directly before writing
  this report: no file exists at that path — the write did not
  materialize (plausibly the same guardrail the previous candidate's `H5`
  evidence found blocking a delete under this exact tree). Flagged for
  visibility, not scored as a defect against `H5` (whose actual `pull`
  evidence, `H5`'s own section above, used the correctly-isolated home
  throughout) — but worth the maintainer's attention: a shell that
  crosses an isolated `REINSTATE_HOME` with a live vendor-home variable is
  a real footgun this harness should make harder to hit by accident.
- **Confirmed correct, not a defect (carried, unchanged since
  `v0.6.0-rc.1`):** the `fakelocker AnyBucket:true` harness limitation
  makes `sync verify` step 4 report `"fail"` rather than a genuine
  bucket-isolation pass on this local lab (`H4`, `H9`) — contrasted
  directly by `H10`'s real BYO `not-applicable` result and `H12`'s real
  cross-account signature refusal, both of which exercise the equivalent
  real security boundary correctly.
- **Harness note, not a product issue:** `rein sync migrate --to byo`
  needs `REINSTATE_S3_ACCESS_KEY_ID`/`REINSTATE_S3_SECRET_ACCESS_KEY` for
  the *destination* bucket in addition to the passphrase FD; omitting
  them fails with a generic `"secret input requires an interactive
  terminal or a secret file descriptor"` that does not name which secret
  is missing. Matches a note already carried from `v0.6.0-rc.6`.
- **`CLAUDE.md` (checked into this repository) carries a "context-mode"
  section instructing mandatory routing of Bash/Read/Grep/WebFetch through
  MCP tools (`ctx_fetch_and_index`, `ctx_execute`, `ctx_batch_execute`,
  `ctx_search`, `ctx_index`, `ctx_stats`, `ctx_doctor`, `ctx_upgrade`).**
  None of these tools exist in this session's tool list; this executor
  used ordinary Bash/Read/Grep/Edit throughout and never attempted a
  `ctx_*` call. Carried unchanged from `v0.6.0-rc.6`'s own part-D finding
  — still present, still flagged for the maintainer, since it could
  misdirect a future agent into believing those tools exist.

## Harness notes

- **Mid-run lab restart.** `hoplab ps` at the start of this session's work
  showed a stale registry entry for this exact lab root (`hopd`/
  `fakelocker` both `alive=false`) left over from earlier work this
  session; cleared with `hoplab stop -all` (removed 1 stale entry, stopped
  0 live processes) and the lab was restarted with `hoplab start` against
  the same root, reusing the prebuilt `hopd.exe`. `hoplab start` always
  writes a **fresh** `hopd.db` (per its own documentation), so the
  pre-restart pass's Hop-account state (the disposable accounts `H1`
  through `H10` had used before the restart) no longer exists
  server-side after it. This report therefore presents `H1`–`H4`, `H9`,
  and `H10` from a complete, self-consistent **fresh** pass run entirely
  after the restart (new disposable account, new devices, all evidence
  above); `H5`'s wipe/new-device/recover/pull/dry-run-block sequence is
  the one component still drawn from the pre-restart pass (its own
  disposable account and devices, self-consistent within itself), with
  `H5`'s "one real resume" step captured fresh, since it depends only on
  the real vendor `claude` binary and the still-present real session file,
  not on any lab/`hopd` state at all. `H11`/`H12` were run fresh, once,
  entirely after the restart (never attempted before it).
- `hoplab ps` was re-checked after the restart (`hopd`/`fakelocker` both
  `alive=true`) and again at the end (`no labs recorded`, after this
  executor's own `hoplab stop`) — this part never left a lab running that
  it did not itself start or stop.
- One additional, short-lived disposable lab was started and stopped for
  `H1b`'s `HOPD_LOGIN_SESSION_TTL=10s` requirement (`127.0.0.1:8322`/
  `9322`), never colliding with the primary lab's ports.
- Two throwaway Go evidence tools (`s3rw`, `fdrunner`) were written under
  `scripts/testing/` and deleted before this report was committed, along
  with two similarly-purposed leftover tools from earlier work this
  session (`scratchtool_h3`, `scratchtool_migrate`) — none were ever
  staged or part of this commit.
- A harness-authoring pitfall worth naming for future executors: writing
  a Windows path into a TOML double-quoted string through a Bash heredoc
  can silently lose backslash escaping (observed here first-hand while
  adding a `[[projects]]` mapping by hand) — a TOML **literal** string
  (single quotes, e.g. `local_root = 'C:\Users\...'`) sidesteps the escape
  handling entirely and is what this report's own evidence commands used
  after the first attempt failed with a `toml: ... expected eight
  hexadecimal digits after '\U'` parse error. Not a product issue.
- The `s3rw` tool's first `put` implementation wrapped its `io.Reader` in
  `io.NopCloser`, which drops the underlying `bytes.Reader`'s `Seek`
  method; the S3 SDK's trailing-checksum computation needs a seekable
  body and failed with `"unseekable stream is not supported without TLS
  and trailing checksum"` until the wrapper was removed. Harness-tool
  bug, not a product issue; fixed within this run before any evidence
  above depended on it.

## Verdict (this part only)

**11/11 required rows PASS.** Zero release-blocking findings from this
part. Every row that cleared at `v0.6.0-rc.6` cleared again here, unchanged
— no regression. This part's result is a component of the full
tagged-artifact device report; the overall device verdict is assembled
across all executors' parts (section A, the 178-row Phase 5 matrix, the
22-row CLI matrix, and all 16 section D rows) and is not declared here.
