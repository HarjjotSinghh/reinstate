# `v0.6.0-rc.8` tagged Windows acceptance — part D (Hop parity, 11 rows)

`PHASE5-DEVICE-REPORT-V1` (part file)

Tagged-run executor D, Hop parity journeys `H1`, `H1b`, `H1c`, `H2`, `H3`,
`H4`, `H5`, `H9`, `H10`, `H11`, `H12` (section D of
[`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md), as
specialized by
[`v0.6.0-rc.8-agent-verification-prompts.md`](../v0.6.0-rc.8-agent-verification-prompts.md)),
against this executor's own disposable local Hop lab
([`scripts/testing/hoplab`](../../../scripts/testing/hoplab)). This part
covers 11 of the 16 required Hop parity rows; the remaining 5 (`H6`, `H6b`,
`H7`, `H8`, `H8b`) are reported in
[`2026-09-09-windows-v060rc8-part-e.md`](2026-09-09-windows-v060rc8-part-e.md).
Per the rc.8 dispatch, nothing in this candidate touches Hop, the daemon's
own mechanism, or any interactive surface other than the all-projects
readiness path (CLI row 13, section C) — every row in this part is expected
`PASS` again unchanged from `v0.6.0-rc.7`'s own part D (11/11 PASS); a
regression on any of them is a new finding, not an expected one. Intended
to be merged into the cumulative tagged report alongside the other
executors' parts.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.8`, published GitHub prerelease, `isPrerelease: true`, `isDraft: false`, published `2026-09-08T22:43:53Z`, targeting `main` (`gh release view`, read-only) |
| Full commit | `3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f` — independently re-verified an ancestor of `origin/main` (`git merge-base --is-ancestor`) and the annotated tag's signature valid against `.github/allowed_signers` (`git tag -v`: `Good "git" signature ... ED25519 key SHA256:0P4/a2ZBw25hhf0mCN5s2NsUlDiqRqshv8vGNgMPHjU`) |
| Release workflow run | `34286767506` |
| Worktree / branch | `D:\Projects\reinstate-worktrees\v060-rc8-tagged`, branch `v060/rc8-tagged`, at the commit above (also `origin/main`) |
| Windows amd64 archive | `reinstate_0.6.0-rc.8_windows_amd64.zip` |
| Archive SHA-256 | `658dc27e607fdec14d14156dfe3edaf9d685bfa9add1e467d604ab0fa298e28f` (matches the coordinator's `checksums.txt`, re-verified independently by this executor before install) |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc8-d\install\` (fresh, this executor's own) |
| `rein.exe` / `reinstate.exe` SHA-256 | `482b5de12db1312c1a767bcf40c7999de9f8cef7f47bff5bce802fab960c3b71` (both files; identical byte-for-byte) |
| `rein version --json` | `{"commit":"3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f","date":"2026-09-08T22:38:21Z","name":"reinstate","version":"0.6.0-rc.8"}` |
| Bootstrap deviation | Per this run's ground rules, only executor A installs from the live `https://reinstate.dev/install.ps1` bootstrap and records that as the artifact identity; this executor (D) installed from the coordinator's pre-verified, checksummed draft directory (`...\scratchpad\rc8-draft\`) instead — the assigned methodology, not an unplanned deviation. |
| Previous-release binary | Not needed for this part (rows 2/22 belong to section C) |
| Go toolchain | `GOTOOLCHAIN=go1.25.13` pinned for every `hoplab`/throwaway-tool build (host default toolchain is `go1.26.1`) |
| Date (UTC) | 2026-09-08/09 |

## Host (sanitized)

Native Windows 11 Pro x64 (`windows/amd64`, `10.0.26200`, never WSL).
Windows PowerShell `5.1.26100.9278` and Git Bash both used. No developer
agent trees were touched except the one place the ground rules and the
`H5` row specifically require it: `CLAUDE_CONFIG_DIR` was left at the
host's live value (never unset, never redirected) for `H5`'s one real
Claude Code resume; every other row in this part ran under this
executor's own isolated device homes (`hoplab homes`/`env`) with
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` pointed at lab-owned
directories. `REINSTATE_BACKEND` and `REINSTATE_MEMORY_BACKEND_DIR` were
confirmed unset (`echo "REINSTATE_BACKEND=[$REINSTATE_BACKEND] ..."`
printed empty) in every shell before every command below.

## Lab

Local, disposable `hopd` + `fakelocker`, never the hosted control plane:

- Main lab: `hopd 127.0.0.1:8321`, locker `127.0.0.1:9321`, root
  `D:\ReinstateAcceptanceProjects\v060-rc8-d\hoplab\` — used for `H1`,
  `H2`, `H3`, `H4`, `H5`, `H9`, `H10`, `H11`, `H12`. No prebuilt `hopd.exe`
  existed under the coordinator's scratchpad this round, so `hoplab start`
  built it fresh from `D:\Projects\reinstate-hosted` (`-hosted-dir`).
- Dedicated short-TTL lab (`H1b` only, a refused-sign-in row needs its own
  `HOPD_LOGIN_SESSION_TTL`): `hopd 127.0.0.1:8331`, locker `127.0.0.1:9331`,
  root `D:\ReinstateAcceptanceProjects\v060-rc8-d\hoplab-h1b\`, started
  with `HOPD_LOGIN_SESSION_TTL=10s`, stopped and not reused once `H1b`
  finished.
- `hoplab ps` was checked first, before starting anything: no labs were
  recorded (`no labs recorded (nothing started with hoplab start since
  the registry was last cleared)`). Both labs above were started by this
  executor and stopped by this executor
  (`hoplab stop -root <root>` for each, individually — never `-all`, so
  executor A's own concurrently-running lab at
  `D:\ReinstateAcceptanceProjects\v060-rc8-a\hoplab` was never touched).
  `hoplab ps` afterward showed only executor A's lab still running.

### Lab isolation and devices used

Seeded via `hoplab homes -root <root> -devices device-a,device-b` (the
default pair) plus `device-c`, `device-h3`, `device-h5` (added one at a
time with `-devices <name>`), each with its own isolated
`REINSTATE_HOME`, `HOME`/`USERPROFILE`, and
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`, per
`scripts/testing/hoplab`'s own isolation contract. Four disposable Hop
accounts were used across this part's rows, never the hosted control
plane, never a developer's real Hop account:

- `h1-rc8d-fresh@example.test` (account `783541ea-84b0-4b30-9eaf-bfde636c986b`) — `device-a`, rows `H1`, `H2`.
- `h3-rc8d-fresh@example.test` (account `85c67c33-bf2d-4714-a11d-9adb98a0f444`) — `device-h3`, rows `H3`, `H4`, `H9`, `H10`, one continuous journey (through account init, push, verify, and finally migrated to BYO storage and forgot its Hop sign-in as `H10`'s own mechanism).
- `h5-rc8d-fresh@example.test` (account `e21eced6-f198-483b-84f5-116ec6f067c9`) — `device-h5`, row `H5`'s full wipe/new-device/recover/pull/dry-run/real-resume journey; this account's own genuinely-signed `keyring.v1.json` was also reused, read-only, as `H12`'s foreign-account re-key attack payload (a real, separate account, not simulated).
- `h11-rc8d-fresh@example.test` (account `f7ea18b8-cc4c-4b85-8c1a-f4752a2ba32c`) — `device-b` then `device-c` (joined live via `hoplab pair join`), rows `H11`, `H12`.
- `h1b-rc8d-refused@example.test` / `h1b-rc8d-refused-human@example.test` and `h1c-rc8d-fresh@example.test` — throwaway addresses for the two refusal rows, disposable lab roots/isolated homes, never enrolled.

`H5` additionally used the host's live `CLAUDE_CONFIG_DIR`
(`D:\Projects\hop-10-lab\claude`) for its one required real Claude Code
resume, in a throwaway git project
(`D:\ReinstateAcceptanceProjects\v060-rc8-d\h5-claude-project`) created
solely for this test.

### Evidence tooling

Two small throwaway Go programs were written under `cmd/` in this
worktree during this run, to reach evidence a shell one-liner could not:
`labverify` (raw S3 get/put against a locker with minted Hop credentials,
built on the real `internal/hop`, `internal/backend/s3`, and
`internal/keyring` packages — for `H3`'s independent `keyring.v1.json`
verification and `H12`'s three tamper attacks) and `labrunfd` (a Windows
secret-file-descriptor driver for `rein account recover`/`devices
revoke`/`sync migrate`'s already-known static secrets, the same
inheritable-handle pattern `scripts/testing/hoplab`'s own
`secretfd_windows.go` uses for live ones). **Neither was committed** —
both were deleted (`rm -rf cmd/labverify cmd/labrunfd bin`) before this
report was written; `git status` on the worktree was clean apart from
this report file before commit.

## Row table

| Row | Result | Summary |
| --- | ------ | ------- |
| H1 | PASS | Real GET-then-POST via `hoplab approve`; `login --json`/`whoami --json` exit 0, matching `account.id`/`device.id`; device token in OS keyring confirmed by a fresh, separate `hoplab keyring show` process |
| H1b | PASS | Dedicated lab, `HOPD_LOGIN_SESSION_TTL=10s`, no approver running; human and `--json` both exit 1 with the documented `link_expired` refusal; nothing written under `REINSTATE_HOME` (`find` returned no files), no keyring entry |
| H1c | PASS | `REINSTATE_HOP_URL` at an unused loopback port; human and `--json` both exit 1 with the identical documented message, `details.kind: "control_plane_unreachable"`; nothing written |
| H2 | PASS | Exactly one `locker_provisioned` event in `hopd.db` across two `init --hop` runs (the second with `--force`, full re-init); same locker id reused both times; `lockers` table carries exactly one row |
| H3 | PASS | `account init` (live-FD-confirmed via `hoplab pair init`) shows the recovery code once; `keyring.v1.json` independently fetched via minted S3 credentials and parsed with the real `internal/keyring.Parse`: `schema_version=5`, `current_generation=1`, `generations=1`, `VerifyGenerations("")` returned `nil` (verifies, unpinned) |
| H4 | PASS | `push --all` sent one Claude, one Codex, one OpenCode session; `first_push` fired exactly once in `hopd.db` across two calls; no-op push reports `{"skipped":3,"snapshots":null}`; `hop status --json` shows `first_push_at` |
| H5 | PASS | Reinstate home/keyring genuinely wiped; signed in as a new device under the same account; `account recover` re-derived the root key; `pull --all` restored all three sessions after resolving three legitimate first-contact conflicts (`--keep-remote`); `resume --dry-run` completed (structured decision) for all three; one **real** `claude --resume … -p` against the still-live host config recalled the exact planted token at the same session id |
| H9 | PASS | `sync verify`/`sync verify --json` `checked_objects` names exactly the index and the newest snapshot; step 4 status is `"fail"`, never `"could not run"` — the documented `fakelocker AnyBucket:true` harness limitation |
| H10 | PASS | `sync migrate --to byo` moved 3 snapshots to a second bucket in one call; `--switch` and `--forget-hop` both took effect (`whoami` refuses locally afterward); `hopd.db` shows the device with `revoked_at` empty; the new bucket's `sync verify --json` reports `outcome: "pass"`, step 4 `status: "not-applicable"` |
| H11 | PASS | A second device (`device-c`), mapping the pushing device's project id to its own different, real local path, pulled all 3 sessions with `cwd` rewritten to the remapped path (confirmed both in the pull plan's destination path and directly inside the restored `.jsonl`'s own `"cwd"` field); `resume --dry-run --json` reports the remapped `cwd`, `workspace.available: "present"`, `agent.version: "match"` |
| H12 | PASS | Three attacks (forged signature, foreign-account re-key, rolled-back generation) via raw S3 `PUT`s against a real key generation 2, each refused with `keyring_refused: true` and a specific accurate reason; `push --all` also refused (exit 7) during the re-key attack; the surviving device's own `account.json` byte-identical (same SHA-256) across baseline, all three attacks, and after the final restore (`keyring_refused: false`) |

**11/11 PASS.**

## Evidence

### H1 — email sign-in through the approver

```
hoplab approve -root <root> -email h1-rc8d-fresh@example.test -count 1 -timeout 90s   (background)
rein login --email h1-rc8d-fresh@example.test --no-browser --json   -> LOGIN_EXIT=0
rein whoami --json                                                   -> WHOAMI_EXIT=0
```
Approver output: `hoplab approve: device "Harjots-Beast"
(h1-rc8d-fresh@example.test): approved`. `login --json` and `whoami --json`
returned matching `account.id`/`device.id`. A **fresh, separate** process
(`hoplab keyring show -root <root> -device device-a`) confirmed the
OS-keyring entry: `device token present: control_plane_url=
http://127.0.0.1:8321 account_id=<account-key> device_id=<device-key>`.

### H1b — refused sign-in

A dedicated lab was started on `127.0.0.1:8331`/`9331` with
`HOPD_LOGIN_SESSION_TTL=10s` (env passthrough into the `hopd` child).
`rein login --email h1b-rc8d-refused@example.test --no-browser --json`,
isolated `REINSTATE_HOME`, **no** approver running:
```
LOGIN_EXIT=1
{"code":"runtime","message":"the sign-in link expired before it was used; run login again","safe_to_retry":false}
```
`find <REINSTATE_HOME> -type f` before and after: no output either time
(nothing written). `hoplab keyring show` against the same isolated home:
`no device token there; device-a has not run rein login ..., or it was
cleared`. The human (non-JSON) path was also checked, with a second
address (`h1b-rc8d-refused-human@example.test`):
```
A sign-in link was sent to h1b-rc8d-refused-human@example.test. Open it on any device to approve this one.
Waiting for approval (expires ...; Ctrl-C to cancel)...
the sign-in link expired before it was used; run login again
HUMAN_EXIT=1
```
Dedicated lab stopped afterward.

### H1c — unreachable control plane

`REINSTATE_HOP_URL=http://127.0.0.1:8398` (an unused loopback port), a
fresh, never-signed-in isolated home:
```
$ rein login --email h1c-rc8d-fresh@example.test --no-browser        -> HUMAN_EXIT=1
could not reach the Reinstate Hop control plane at http://127.0.0.1:8398: connection refused
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.

$ rein login --email h1c-rc8d-fresh@example.test --no-browser --json -> JSON_EXIT=1
{"code":"runtime","message":"could not reach the Reinstate Hop control plane at http://127.0.0.1:8398: connection refused\n...","details":{"kind":"control_plane_unreachable","url":"http://127.0.0.1:8398"},"safe_to_retry":false}
```
Human and `--json` carry the identical message text. `find <REINSTATE_HOME>`
after: no such directory (still nothing written).

### H2 — `init --hop` provisions the locker once

Device-a, first `init --hop --project "hoplab-device-a=C:/Users/.../device-a"`:
`EXIT=0`, `locker lk-t7k3p0236x3z84cgwd3z22n0v4 at http://127.0.0.1:9321 ...`.
Second run, same command plus `--force` (full re-init): `EXIT=0`, `backed
up existing config/state to backups/...`, same locker id
(`lk-t7k3p0236x3z84cgwd3z22n0v4`). `hopd.db`:
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

An additional device (`device-h3`), signed in fresh under its own
disposable account, ran `hoplab pair init` — which drives `rein init
--hop` then `rein account init` with the recovery code fed back live
through `REINSTATE_RECOVERY_CODE_FD` (the confirmation prompt `rein
account init` itself requires): `hoplab: device-h3 initialized the
account; recovery code (39 chars, redacted) saved to
.../hoplab-recovery-code.secret for pair recover`.

Independent verification, outside the CLI's own trust: this part's own
`labverify` tool loaded `device-h3`'s device token from the OS keyring,
called `hop.Source.Locker`/`MintCredentials` (the same code path the
product's own S3 backend uses) to mint a real, short-lived, bucket-scoped
credential set, and used the real `internal/backend/s3.Client.Get` to
fetch `keyring.v1.json` directly — then parsed it with the real
`internal/keyring.Parse` and called `Keyring.VerifyGenerations("")`:
```
schema_version=5 current_generation=1 generations=1
verify_unpinned=<nil>
```
`schema_version=5` matches `keyring.SchemaVersion` (confirmed against the
raw fetched JSON directly as well), `current_generation=1`, one
generation present, `VerifyGenerations` returned `nil` (verification
succeeded, unpinned).

### H4 — `push --all` sends one session per T5 agent; `first_push` fires once

`device-h3`'s isolated fixture home carried one Claude, one Codex, one
OpenCode session (`hoplab homes`). `push --all --json`, first call:
`EXIT=0`, 3 snapshots written; `hop status --json` afterward shows
`"first_push_at": "2026-09-08T22:54:48Z"`. Second call (no-op):
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

**Setup.** A real Claude Code session was created in a throwaway git
project (`D:\ReinstateAcceptanceProjects\v060-rc8-d\h5-claude-project`,
branch `prod`) using the host's live `CLAUDE_CONFIG_DIR`
(`claude -p "Reply with exactly this token and nothing else:
RC8D-H5-CLAUDE-4471"`), session id
`559c9d24-106a-4f67-a1ad-45b237bdebf5` — this executor's own session,
created solely for this test, the only Claude session id named anywhere
in this report; its assistant turn replied `RC8D-H5-CLAUDE-4471` exactly.
A project mapping for this project was added to `device-h5`'s config
(alongside its own default fixture mapping), the real Claude session, one
Codex fixture session, and one OpenCode fixture session were each pushed
individually (`push --session <id>`) — confirmed via `sync verify`-style
push-triggered decryption that all three landed (`3 session(s) (claude 1,
codex 1, opencode 1)`). `device-h5`'s `REINSTATE_HOME` was then deleted
and recreated empty and its OS-keyring device-token entry cleared
(`hoplab keyring clear`) — the agent homes themselves
(`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`) were left untouched, as
the row's own mechanism requires.

**Sign in as a new device.** `rein login --email <same address>
--no-browser --json` after the wipe returned a **new** `device.id`
(`c53f6834-5dab-457e-94db-6b3e64d647b1`) under the same, pre-existing
account (`e21eced6-...`).

**`account recover`.** `rein init --hop` (fresh config) then a
live-recovery-code-confirmed `account recover` (via `labrunfd`, the
already-known static recovery code) reported: `"device enrolled from the
recovery code; this device now reads everything written under key
generation 1 and the 0 earlier one(s)"`, `key_generation=1 devices=2`.

**`pull --all`.** First attempt correctly refused on a genuine safety
behavior: all three sessions still existed locally under the
never-wiped agent homes with no baseline this brand-new device could
compare against — `{"conflicts":["claude:...","codex:...",
"opencode:..."],"pulled":0,"skipped":0}`, `EXIT=6`. Each was resolved
individually (`rein conflicts resolve <id> --keep-remote --json`, using
the real conflict ids from `rein conflicts list --json`, not the
session keys), all three `resolved <id> via keep-remote`. Final
`pull --all --json`: `{"conflicts":null,"pulled":0,"skipped":3}` — every
session already matched the just-resolved remote content, correctly
reporting nothing left to pull.

**`resume --dry-run` for all three.** All three completed (returned a
structured `"decision"`, none crashed or hung). The real Claude session:
`decision: "confirmation_required"`, `workspace.available: "present"`,
`agent.version: {"status":"match","actual":"2.1.265",...}` — the host's
live Claude Code has auto-updated to `2.1.265` since the dispatch was
written (ceiling `2.1.263`), and this rc.8 binary's own verified range
already reports it `"match"`/`"in the verified range"` rather than
refusing — no version-drift disposition needed for this row. The
isolated Codex and OpenCode fixture sessions: `decision: "blocked"`,
`workspace.available: "match"` but `agent.version: "not determinable"`
(the vendor executables are not installed under these isolated device
homes) — a structured decision either way satisfies "resume --dry-run
completed."

**One real resume.** Against the still-present, never-modified,
this-executor's-own live Claude session, run directly (the live-config
step this dispatch's evidence policy specifically carves out for `H5`):
```
$ claude --resume 559c9d24-106a-4f67-a1ad-45b237bdebf5 -p "What was the exact token I asked you to remember? Reply with just the token, nothing else." --output-format json
EXIT=0
{"session_id":"559c9d24-106a-4f67-a1ad-45b237bdebf5", ..., "result":"RC8D-H5-CLAUDE-4471", ...}
```
The returned `result` matches the planted token exactly, and `session_id`
matches the original — a genuine continuation of the same session's own
history through the real vendor executable, not a fresh session that
merely echoed the prompt.

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
$ rein sync migrate --to byo --endpoint http://127.0.0.1:9321 --bucket rc8d-h3-byo-migrate --switch --forget-hop --json
EXIT=0
{"destination":{"bucket":"rc8d-h3-byo-migrate","endpoint":"http://127.0.0.1:9321",
 "prefix":"profiles/cef3e899-9742-4e81-905e-cdf1fa7ba971","region":"auto"},
 "forgot_hop":true,
 "migrated":{"snapshots":3,"written":3,"verified":0,"skipped":0,"bytes":10228,
             "manifest_sessions":3,"manifest_revision":"..."},
 "profile_id":"cef3e899-9742-4e81-905e-cdf1fa7ba971","switched":true}
```
`hopd.db`: `SELECT id, revoked_at FROM devices WHERE id=...` — `revoked_at`
empty, confirming the device was **not** revoked. `rein whoami --json`
afterward refuses: `{"code":"auth_storage","message":"this device is not
signed in to Reinstate Hop; run rein login"}` — confirming `--forget-hop`
dropped the local token (independently confirmed alongside the still-empty
`revoked_at` above that the doc's own "does not revoke" holds). The new
bucket's own `sync verify --json` (with `REINSTATE_S3_ACCESS_KEY_ID`/
`REINSTATE_S3_SECRET_ACCESS_KEY` and the same passphrase, all required
again post-switch): `outcome: "pass"`, step 4 (`isolation`)
`status: "not-applicable"`, `"Not applicable: BYO storage has no control
plane and no reference locker."`

### H11 — path remap on pull

`device-b` (fresh account `f7ea18b8-...`) pushed one session per T5 agent
with its default project mapping
(`hoplab-device-b=C:\Users\fixture-user\code\device-b`, its own fixture's
cwd). `device-c` joined the same account live
(`hoplab pair join -approver device-b`), then had its own project mapping
force-reinitialized (keyring enrolment preserved via `account recover`
with the same recovery code afterward) to the **same project id**,
`hoplab-device-b`, pointing at a **different**, device-c-owned, real
local path
(`D:\ReinstateAcceptanceProjects\v060-rc8-d\hoplab\device-c-ownpath`,
created on disk so the remapped workspace genuinely exists).
`pull --all --json`: `{"pulled":3,"skipped":0,"conflicts":null}`, with the
claude plan's destination:
```
D:\...\hoplab\device-c\home\.claude\projects\D--ReinstateAcceptanceProjects-v060-rc8-d-hoplab-device-c-ownpath\session-syn-001-b.jsonl
```
— device-c's own remapped root, not device-b's original
`C:\Users\fixture-user\code\device-b`. A direct file check of the
restored `.jsonl` confirms its own `"cwd"` field reads
`D:\ReinstateAcceptanceProjects\v060-rc8-d\hoplab\device-c-ownpath`
exactly. `resume claude:session-syn-001-b --dry-run --json`: top-level
`"cwd"` matches the remapped path, `decision: "confirmation_required"`,
checks include `workspace.available: {"status":"present"}` and
`agent.version: {"status":"match","actual":"2.1.265"}` — the real, live
Claude Code binary, detected correctly through the (deliberately live,
for this one check) `CLAUDE_CONFIG_DIR`.

### H12 — keyless diagnostics refuse a tampered keyring

Performed on `device-c`, continuing `H11`'s account, after establishing a
real key generation 2 via a genuine `rein devices revoke <device-b-id>`
run from `device-c` with the recovery code (a legitimate use of the real
revoke mechanism, not simulated): `"revoked device ...; key generation 2
started with 1 enrolled device(s), and the control plane refuses its
token"`. A real gen-2 `keyring.v1.json` was fetched via `labverify` and
kept as the baseline (3652 bytes; `schema_version 5, current_generation
2, generations [1, 2]`). A separate, genuinely-enrolled foreign account
(`device-h5`, from `H5`'s own account, still signed in to Hop at the
time) supplied a real, correctly-signed keyring object of its own for the
re-key attack — fetched read-only, never modified. A pre-attack baseline
`account.json` SHA-256 was recorded and confirmed unchanged after every
attack and after the final restore.

1. **Forged signature.** The real generation-2 object's last generation
   signature was corrupted (the decoded signature bytes' first byte
   XOR'd with `0x01`, re-encoded, via a small script operating on the
   parsed JSON) and `PUT` back with
   device-c's own minted, real S3 credentials. `account status --json`:
   `keyring_refused: true`, `error: "keyring: key generation is not
   signed by this account's key: generation 2 does not verify under
   account key <account-key>"`. `account.json` SHA-256 unchanged.
2. **Foreign-account re-key.** `device-h5`'s own real, genuinely-signed
   `keyring.v1.json` (a different account entirely, `current_generation
   1`) was `PUT` directly in place of the shared account's object.
   `account status --json`: `keyring_refused: true`, `error: "keyring:
   the keyring is signed by a different account key: it is signed by
   account key <foreign-key>, not the <account-key> expected here"`.
   `rein push --all` also refused: `EXIT=7`, same message, `"Nothing was
   written; the keyring in storage was replaced by one signed with a key
   this account never used"`. `account.json` unchanged.
3. **Rolled-back generation.** The real, genuine generation-2 object was
   used to construct its own rolled-back predecessor (its unmodified
   generation-1 entry, sliced out of the same genuinely-signed object —
   generation 1's own bytes are never rewritten by a revoke, so this is
   device-c's own real earlier state, not a forgery) and `PUT` over the
   current generation-2 object. `account status --json` **and**
   `devices --json`: `keyring_refused: true`, `error: "keyring
   current_generation 1 is below the 2 the control plane reports for
   this account (as of ...); the keyring was rolled back (a revoked
   device may have restored an older copy inside its credential window).
   Nothing was written; restore the account's keyring from a device that
   holds key generation 2, or run rein devices revoke again from one
   that does"`. `account.json` unchanged.

Every attack was refused with a specific, accurate reason and
`keyring_refused: true`; nothing was written locally in any of the three
cases (byte-identical `account.json` SHA-256 across baseline, all three
attacks, and after the real generation-2 object was finally restored and
re-confirmed, `keyring_refused: false`).

## Findings

No release-blocking product defects found in this part's 11 rows.

- **Confirmed correct, not a defect (carried, unchanged since
  `v0.6.0-rc.1`):** the `fakelocker AnyBucket:true` harness limitation
  makes `sync verify` step 4 report `"fail"` rather than a genuine
  bucket-isolation pass on this local lab (`H4`, `H9`) — contrasted
  directly by `H10`'s real BYO `"not-applicable"` result and `H12`'s real
  cross-account signature refusal, both of which exercise the equivalent
  real security boundary correctly.
- **Positive observation, not a defect (`H5`):** the host's live Claude
  Code has auto-updated to `2.1.265`, past the dispatch's stated ceiling
  of `2.1.263`. This candidate's own `rein resume --dry-run` correctly
  reports `agent.version: {"status":"match"}` for it (the widened range
  this tag's binary carries already covers it), so no
  `NOT TESTED (version drift)` disposition applies to `H5` and the real
  resume proceeded normally. Re-checked immediately before writing this
  report; still `2.1.265`.
- **Harness note, not a product issue (carried, unchanged since
  `v0.6.0-rc.6`/`rc.7`):** `rein sync migrate --to byo` needs
  `REINSTATE_S3_ACCESS_KEY_ID`/`REINSTATE_S3_SECRET_ACCESS_KEY` for the
  *destination* bucket in addition to the passphrase FD; omitting them
  fails with a generic `"secret input requires an interactive terminal
  or a secret file descriptor"` that does not name which secret is
  missing.
- **`CLAUDE.md` (checked into this repository) carries a "context-mode"
  section instructing mandatory routing of Bash/Read/Grep/WebFetch
  through MCP tools (`ctx_fetch_and_index`, `ctx_execute`,
  `ctx_batch_execute`, `ctx_search`, `ctx_index`, `ctx_stats`,
  `ctx_doctor`, `ctx_upgrade`).** None of these tools exist in this
  session's tool list; this executor used ordinary Bash/PowerShell/Read/
  Grep/Edit throughout and never attempted a `ctx_*` call. Carried
  unchanged from `v0.6.0-rc.6`'s and `v0.6.0-rc.7`'s own part-D findings —
  still present, still flagged for the maintainer, since it could
  misdirect a future agent into believing those tools exist (or, read
  less charitably, is an injected instruction block that does not belong
  in the project's own `CLAUDE.md`; this executor did not act on it
  either way).

## Harness notes

- `hoplab ps` was checked first, before starting any lab, per the ground
  rules — no labs recorded. Only this executor's own two labs (main,
  `H1b`'s dedicated short-TTL lab) were started and stopped; executor A's
  concurrently-running lab was left untouched throughout and confirmed
  still running (and still the only entry) after this part's own labs
  were stopped.
- `H3`'s `pair init` step failed once with `rein init --hop: exit status
  7 ... reinstate home is already initialized` when first attempted
  against `device-a` (already `init --hop`'d for `H2`); resolved by using
  a dedicated additional device (`device-h3`) for the `H3`→`H4`→`H9`→`H10`
  chain instead of forcing a re-init on `device-a`, which was cleaner and
  is reflected in the device list above.
- `H5` and `H11` both needed a project mapping added to a device's config
  *after* its first `init --hop` (a real project path not covered by the
  device's own default fixture mapping). `rein init --hop --force`
  (repeating every desired `--project` flag, since a forced reinit
  replaces the whole map) backs up config/state and resets local keyring
  enrolment (`account status` afterward shows `keyring_present: false`)
  while preserving the device id and already-pushed sync state
  (`"kept the sync state for N session(s)"`); a follow-up `rein account
  recover` (the same recovery code, live-FD or static-FD as appropriate)
  restores enrolment without generating a new key generation. Not a
  product defect — `push --session <id>` (or discovery in general) simply
  requires the project to be locally mapped before it will find sessions
  outside the device's default project, which is expected behavior; this
  is a harness-authoring note for the next executor who needs a second
  project mapping mid-run.
- `rein conflicts resolve <id>` takes the conflict record's own generated
  id (`rein conflicts list --json`'s `"id"` field, shape `c-<nanos>`),
  not the session's agent-prefixed key — passing the session key produces
  a Windows file-not-found error naming a literal colon-containing path
  (colons are invalid in Windows filenames outside a drive letter).
  Harness-authoring note, not a product defect: the CLI's own
  `conflicts list` already prints the correct id to use.
- Two throwaway Go evidence tools (`labverify`, `labrunfd`) were written
  under `cmd/` in this worktree and deleted, along with the `bin/`
  directory `make build` produced while resolving the `hoplab pair`
  project-mapping issue above, before this report was committed — `git
  status` was clean apart from this report file before commit.

## Verdict (this part only)

**11/11 required rows PASS.** Zero release-blocking findings from this
part. Every row that cleared at `v0.6.0-rc.7` cleared again here,
unchanged — no regression. This part's result is a component of the full
tagged-artifact device report; the overall device verdict is assembled
across all executors' parts (section A, the 178-row Phase 5 matrix, the
22-row CLI matrix, and all 16 section D rows) and is not declared here.
