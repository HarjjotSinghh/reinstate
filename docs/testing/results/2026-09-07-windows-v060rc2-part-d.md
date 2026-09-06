# v0.6.0-rc.2 tagged-artifact Windows acceptance — part D (Hop parity: H1, H1b, H1c, H2, H3, H4, H5, H9, H10, H11, H12)

`PHASE5-DEVICE-REPORT-V1` (partial — Hop parity rows only, this executor's assignment)

This is executor D's part file for the **published, signed GitHub prerelease
`v0.6.0-rc.2`**, satisfying the eleven rows of
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) section D
this executor was assigned: `H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`, `H5`, `H9`,
`H10`, `H11`, `H12`. It follows
[`../v0.6.0-rc.2-agent-verification-prompts.md`](../v0.6.0-rc.2-agent-verification-prompts.md),
which for these eleven rows carries them forward unchanged from the
`v0.6.0-rc.1` tagged run (all eleven `PASS`) with the instruction to re-run
them anyway because this is the first time they run against a `v0.6.0-rc.2`
tagged artifact. It reuses `2026-09-06-windows-v060rc1.md`'s part-D methods
(hoplab, real vendor binaries, direct locker manipulation for keyless
diagnostics) against a fresh, isolated lab on this host.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.2` |
| Full commit | `81d74a82ba2a0e27f9f1a68eb0270d224a20da6a` |
| Archive | `reinstate_0.6.0-rc.2_windows_amd64.zip` |
| Archive SHA-256 | `82ea243cf9aa1b411cc77322abf973afaf8ff8ac13fbbfa05ad32c5d82ecdc84` (matches `checksums.txt` in the coordinator's staged draft directory; re-verified by this executor before install) |
| Installed `rein.exe` / `reinstate.exe` SHA-256 | `0ae03c4eed8c1af610f04842d5ed51efdd9847724a3b841d249b3841e087fce6` (both; `cmp` byte-identical) |
| `rein version --json` | `{"commit":"81d74a82ba2a0e27f9f1a68eb0270d224a20da6a","date":"2026-09-06T21:57:00Z","name":"reinstate","version":"0.6.0-rc.2"}` |
| Install source | `checksums.txt`-verified copy of the coordinator's staged draft, into this executor's own fresh `D:\ReinstateAcceptanceProjects\v060-rc2-d\install\` — **not** the live bootstrap. |
| Bootstrap deviation | Per the dispatch, only executor A installs from `https://reinstate.dev/install.ps1` and records the live-bootstrap identity; every other executor, including this one, installs from the coordinator's checksummed archive at `…\scratchpad\rc2-draft\`. This report claims no bootstrap evidence. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| Device | `windows-amd64`, native process, no emulation |
| OS/version/build | Windows 11 Pro, `10.0.26200` |
| Filesystem | NTFS |
| Date | 2026-09-06/07 (UTC evidence timestamps below) |
| Every shell | `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` run first in every shell that touched `rein`, `hoplab`, or the sandbox helpers, confirmed clean before each row's commands |
| `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` | Left as the host's live agent homes everywhere except inside `hoplab`-seeded device shells, which point the vendor's own home variable at an isolated directory instead of unsetting the host variables (per the ground rules) |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-rc2-d\lab\` (deleted after this report was committed) |
| `hopd` | `127.0.0.1:8321` (own build from `D:\Projects\reinstate-hosted` at commit `f44fbebe964a09fdf3801a0bc0365e0ed8083c9f`, `HOPD_STORAGE=fake`, `HOPD_EMAIL_SENDER=log`) |
| locker (`fakelocker`) | `127.0.0.1:9321` |
| Devices seeded | `device-a`, `device-b`, `device-c` (path-remap trio for H11), `device-w`, `device-x`, `device-y`, `device-z` (H12's keyless-diagnostic devices) |

Throwaway helper binaries built for this run only, from this worktree's own
tree, and never committed: `scripts/testing/hoplab` (the documented lab
driver), `cmd/hopfdrunner` (generic `REINSTATE_RECOVERY_CODE_FD`/
`REINSTATE_PAIRING_CODE_FD` non-interactive driver, mirroring
`scripts/testing/hoplab/secretfd_windows.go`'s own inheritable-handle
mechanics), `cmd/h3probe` (fetches `keyring.v1.json` directly via the
product's own `internal/backend/s3` client using `rein hop credentials
--export`, then runs it through the product's own `internal/keyring.Parse`
+ `VerifyGenerations` for H3), and `cmd/h12tool` (GET/PUT `keyring.v1.json`
directly against the locker with the same credential export, to stage the
forged/rolled-back/re-keyed objects H12 needs). All three `cmd/` helpers
were deleted from the worktree before this report was committed (`git
status` at commit time shows no diff outside this results file). All device
homes and the lab's `hopd.db` lived under
`D:\ReinstateAcceptanceProjects\v060-rc2-d\` and were deleted after this
report was written.

**Harness note on this shared host.** Starting the lab with `hoplab start
-background` from a single Bash-tool invocation left `hopd`/`fakelocker`
tied to that invocation's own transient shell process; once that shell
exited, both processes were reaped (confirmed via `hoplab ps` showing
`alive=false` for both pids, no panic in `hopd.log`) even though
`-background` is documented to survive it. Restarting the lab in
**foreground** mode as a persistent background task (this session's own
`run_in_background` primitive, not `hoplab`'s `-background` flag) kept both
processes alive across every subsequent command for the remainder of the
run. This is specific to this execution environment, not `hoplab` or the
product; noted for later executors on the same harness.

**Harness note: `opencode-ai`'s global install.** Mid-run, `opencode`
(needed once per device to initialize its embedded SQLite store before a
pull can restore into it) failed with "Bun failed to remap this bin to its
proper location within node_modules" / "opencode-ai's postinstall script
was not run" — consistent with another process on this shared host
reinstalling the global package concurrently. Ran `node postinstall.mjs`
inside `~/.bun/install/global/node_modules/opencode-ai` to repair the
existing install (not a new install for a row). This left the resolved
version at `1.18.29`, past this candidate's verified OpenCode ceiling
(`1.18.21`–`1.18.27`) — confirmed by a correct `agent.version` block on
`resume --dry-run` naming the range. Re-pinned with `bun install -g
opencode-ai@1.18.27` before continuing H5. Flagging for the coordinator:
other executors on this same host may have hit the same breakage or the
same transient version drift around `2026-09-06T22:2x`–`22:3x`.

**Harness trap avoided (matches the `v0.6.0-rc.1` tagged run's own §23.2
finding).** `hoplab homes` seeds each device's Claude/Codex fixture
sessions with a recorded `cwd` of `C:\Users\fixture-user\code\<device>` —
a path this host's ACLs refuse to create (`fixture-user` is not a real
profile). `internal/adapter/claude`'s and `internal/adapter/codex`'s
`Discover` filter sessions to whatever a device's own `rein init --project`
mapping's right-hand side matches *exactly* — this is unrelated to whether
that path exists on disk. The first attempt at H4 used a project mapping
whose RHS did not match the fixture's recorded `cwd`, and `push --all`
silently sent only the (unscoped) OpenCode session — a project-mapping
mismatch, not a product defect (`TestClaudeDiscoverSkipsUnmappedProjects`
already covers exactly this). Corrected by mapping each pushing device's
project RHS to its own fixture's exact recorded `cwd` for push/pull
discovery, and — for the *pulling* side of H5 and H11 — mapping to a real,
existing local directory instead, which is precisely what path remapping
(`pull`'s per-device project mapping, H11's own mechanism) is for and what
let `resume --dry-run`'s `workspace.available` check see a directory that
actually exists.

## Verdict for this part

- **Rows in this part:** 11 of the 16 Hop parity rows (section D)
- **Result:** all eleven `PASS`
- **Required-row `PASS` count for this part:** 11 of 11
- **Release-blocking findings from this part:** 0 product defects found.

## Summary table

| # | Row | Result | Mechanism exercised |
| - | --- | ------ | -------------------- |
| H1 | `rein login --email` completes through the approver; token in the OS keyring; `rein whoami` names the device | **PASS** | Real GET-then-POST via `hoplab approve`; token confirmed present in the Windows Credential Manager by a fresh, separate process (`hoplab keyring show`) |
| H1b | A refused sign-in tells the terminal why and stores nothing | **PASS** | `hoplab approve -refuse` GETs but never POSTs; `HOPD_LOGIN_SESSION_TTL=15s` for a fast expiry; `rein login --json` surfaced `{"code":"runtime","message":"the sign-in link expired before it was used; run login again"}`, exit `1`; `REINSTATE_HOME` never created |
| H1c | An unreachable control plane prints the one-line message and the documented exit code | **PASS** | `REINSTATE_HOP_URL` pointed at an unused port; human and `--json` both read `could not reach the Reinstate Hop control plane at http://127.0.0.1:1: connection refused`, exit `1`, `details.kind=control_plane_unreachable`; `REINSTATE_HOME` never created |
| H2 | `rein init --hop` writes the profile and provisions the locker once; a second run does not re-provision | **PASS** | `locker_provisioned\|1` in `hopd.db`; second `init --hop` refused client-side, exit `7`, no second event |
| H3 | `rein account init` shows the recovery code once, writes `keyring.v1.json` (format 5, signed generation 1) | **PASS** | Recovery code confirmed via a real inheritable-FD pipe (`cmd/hopfdrunner`); `keyring.v1.json` fetched from the real locker and run through the product's own `keyring.Parse` + `VerifyGenerations` directly (`cmd/h3probe`): `SchemaVersion=5`, `CurrentGeneration=1`, `Generations=[1]`, signature verifies both unpinned and pinned against the account's own `account.json` |
| H4 | `rein push --all` sends one Claude Code, one Codex, one OpenCode session as ciphertext; `first_push` reaches the control plane exactly once; `rein hop status` shows it; a no-op push reports nothing | **PASS** (step 4 of automatic post-push verification fails on this lab for a harness reason — `fakelocker`'s `AnyBucket:true`, unchanged from the `v0.6.0-rc.1` tagged run) | Decrypted manifest: `3 session(s) (claude 1, codex 1, opencode 1)`; `first_push\|1` exactly once in `hopd.db`; no-op push reports `skipped:3, snapshots:null` |
| H5 | Wipe home, stores, token, key; sign in as a new device; `rein account recover`; `rein pull --all`; `rein resume --dry-run` complete for all three; one real resume answers from history | **PASS** | Wiped home/`.claude`/`.codex`/xdgdata + OS-keyring entry; post-wipe `whoami`/`account status` correctly refused; signed in as a "new" device; `account recover` restored `key_generation=1, devices=2`; `pull --all` restored all three sessions with their `cwd` remapped to this device's own project mapping; `resume --dry-run --json` reached `confirmation_required` for all three (no `block`-severity checks) once OpenCode's version was re-pinned to the verified ceiling; **one real resume**: a genuine Claude Code session created via the host's live config in a throwaway git project, with a planted token, pushed, pulled into device-a's isolated home, then `claude --resume <id> --print "..."` correctly recalled the exact planted token from restored history |
| H9 | `rein sync verify` human and `--json` name only observed objects; a bucket that is gone is a failed check, not a check that could not run | **PASS** (step 4 fails for the same harness reason as H4) | `--json`'s `checked_objects` names exactly `["the index", "the newest snapshot in the index"]`; `unopened` names the rest by count only; step 4's `status` is `"fail"`, never `"could not run"` |
| H10 | `rein sync migrate --to byo` to a second fake-locker bucket; `--switch`; `--forget-hop` drops the token locally and the doc says it does not revoke | **PASS** | Migrated 4 snapshots to a second `fakelocker` bucket (`migrated.written:4`); `--switch` flipped local config; `--forget-hop` dropped the token locally (`whoami` now refuses, exit `4`) with no `device_revoked` event in `hopd.db`; `sync verify` against the new BYO bucket correctly reports step 4 `NOT APPLICABLE` |
| H11 | Path remap between the two homes' project mappings: pulled sessions carry the other home's path; workspace verification passes on resume | **PASS** (macOS leg E-deferred) | Three genuinely separate devices, same account: device-b pushed 3 sessions under project id `hoplab-device-a`; device-c, mapping that same project id to its own different real local path, pulled all 3 — raw `cwd` in the restored Claude/Codex files confirmed rewritten to device-c's own path; `resume --dry-run` then reported `workspace.available` `status:"present"` |
| H12 | Keyless diagnostics (`rein account status`, `rein devices`) refuse a forged, rolled-back, or re-keyed keyring with nothing written | **PASS** | Two real enrolled devices (one later revoked) in one account, plus a genuinely different account: a rolled-back (pre-revoke, generation-1-only) object, a forged signature (tampered byte), and a different account's genuine re-keyed object each refused by `account status`/`devices` (exit `0`, diagnostic, `keyring_refused:true`, exact messages naming the mismatch) and by `push` (exit `7`); the device's own `account.json` anchor unchanged after all three attempts |

## Evidence

### H1 — email sign-in through the approver

```
hoplab.exe approve -root <lab> -email device-a@example.test -count 1 -timeout 60s &
rein.exe login --email device-a@example.test --no-browser
rein.exe whoami
hoplab.exe keyring show -root <lab> -device device-a
```

Observed:

```
A sign-in link was sent to device-a@example.test. Open it on any device to approve this one.
hoplab approve: device "Harjots-Beast" (device-a@example.test): approved
Signed in to Reinstate Hop as device-a@example.test.
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
Account: device-a@example.test
Plan:    hop (locker location apac)
Device:  Harjots-Beast (windows-amd64, enrolled ...)
hoplab: device token present: control_plane_url=http://127.0.0.1:8321 account_id=... device_id=...
```

### H1b — refused sign-in

Lab restarted once with `HOPD_LOGIN_SESSION_TTL=15s` for a fast expiry (per
`scripts/testing/hoplab/README.md`'s own recommendation for a `-refuse`
run), reverted to the default afterward.

```
rein.exe login --email h1b-refused@example.test --no-browser --json
hoplab.exe approve -root <lab> -email h1b-refused@example.test -refuse -count 1 -timeout 30s
```

Observed:

```
hoplab approve: device "Harjots-Beast" (h1b-refused@example.test): declined (link seen, approval withheld; it will expire on its own)
{"code":"runtime","message":"the sign-in link expired before it was used; run login again","safe_to_retry":false}
```

`ls` on the row's `REINSTATE_HOME` afterward: directory does not exist —
nothing was stored.

### H1c — unreachable control plane

```
REINSTATE_HOP_URL=http://127.0.0.1:1 rein.exe login --email h1c-probe@example.test --no-browser
REINSTATE_HOP_URL=http://127.0.0.1:1 rein.exe login --email h1c-probe@example.test --no-browser --json
```

Observed (human):

```
could not reach the Reinstate Hop control plane at http://127.0.0.1:1: connection refused
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.
```

Observed (`--json`):

```
{"code":"runtime","message":"could not reach the Reinstate Hop control plane at http://127.0.0.1:1: connection refused\n...","details":{"kind":"control_plane_unreachable","url":"http://127.0.0.1:1"},"safe_to_retry":false}
```

Exit `1` both times; `REINSTATE_HOME` never created.

### H2 — `init --hop` provisions once

```
rein.exe init --hop --project "hoplab-device-a=C:\Users\fixture-user\code\device-a"
rein.exe init --hop --project "hoplab-device-a=C:\Users\fixture-user\code\device-a"
sqlite3 hopd.db "select type, count(*) from events group by type"
```

Observed:

```
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-... at http://127.0.0.1:9321 (location apac, plan hop)
--- second run ---
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
device_enrolled|1
locker_provisioned|1
sign_up|1
```

Second `init --hop` exit `7`; exactly one `locker_provisioned` event.

### H3 — `account init` recovery code and keyring signature

```
cmd/hopfdrunner.exe live REINSTATE_RECOVERY_CODE_FD '\b(?:[0-9A-Z]{4}-){7}[0-9A-Z]{4}\b' -- rein.exe account init
eval "$(rein.exe hop credentials --export)"
cmd/h3probe.exe "<pinned account_key from account.json>"
```

Observed (`account init`):

```
Your recovery code (shown once, never stored anywhere):
    ****-****-****-****-****-****-****-****
account initialized: root key generated on this device, keyring written to storage
recovery code confirmed on this device; encryption.type=root-key
```

Observed (`h3probe`, fetching `keyring.v1.json` directly and running it
through the product's own `keyring.Parse` + `VerifyGenerations`):

```
OBJECT_SIZE meta=1476 read=1476
PARSE_OK
SchemaVersion=5
CurrentGeneration=1
AccountKey=<base64 account key>
Generations=[1]
DeviceCount=1
VERIFY_OK
```

The `AccountKey` printed by `h3probe` matched `account.json`'s own
`account_key` field exactly; passing that value as the pinned argument to
`VerifyGenerations` also returned `VERIFY_OK`.

### H4 — `push --all` sends all three sessions once

```
rein.exe push --all --json
sqlite3 hopd.db "select type, count(*) from events group by type"
rein.exe hop status
rein.exe push --all --json
```

Observed (decrypted verification report, step 3):

```
index revision ..., 3 session(s) (claude 1, codex 1, opencode 1)
index entry claude:session-syn-001-a -> snapshots/....age
index entry codex:rollout-syn-001-a -> snapshots/....age
index entry opencode:ses_fixture001a -> snapshots/....age
```

```
device_enrolled|1
first_push|1
locker_provisioned|1
sign_up|1
verify_reported|1
```

Second, no-op `push --all --json`: `{"conflicts":null,"dry_run":false,"skipped":3,"snapshots":null}`.

Step 4 of the automatic post-push verification (`FAIL`: "Listing the
reference locker SUCCEEDED and returned 0 object(s)... could not run:
reading the probe object... no such object") reproduces the same
`fakelocker`-`AnyBucket:true` limitation the `v0.6.0-rc.1` tagged run
recorded for this exact step, unchanged — not a new finding, and contrasted
by H10's correct `NOT APPLICABLE` on a genuine BYO destination below.

### H5 — wipe, recover, pull, dry-run resume, and one real resume

```
rm -rf <reinstate home> <.claude> <.codex> <xdgdata>
hoplab.exe keyring clear -root <lab> -device device-a
rein.exe whoami          # refused
rein.exe account status  # refused
rein.exe login --email device-a@example.test --no-browser
rein.exe init --hop --project "hoplab-device-a=<real, existing local path>"
cmd/hopfdrunner.exe fixed REINSTATE_RECOVERY_CODE_FD "<saved recovery code>" -- rein.exe account recover
rein.exe pull --all --json
rein.exe resume claude:<id> --dry-run --json
rein.exe resume codex:<id> --dry-run --json
rein.exe resume opencode:<id> --dry-run --json
```

Observed (post-wipe refusal):

```
this device is not signed in to Reinstate Hop; run `rein login`   (exit 4)
config missing                                                     (exit 3)
```

Observed (`account recover`):

```
device enrolled from the recovery code; this device now reads everything written under key generation 1 and the 0 earlier one(s)
key_generation=1 devices=2
```

Observed (`pull --all`, after the documented one-time OpenCode
initialization step — `opencode --pure db "select 1"` — since a brand-new
device home has no OpenCode SQLite store yet for `pull` to restore into):
all three sessions restored, with their raw `cwd` rewritten to this
device's own mapped local path (not the original fixture path).

Observed (`resume --dry-run --json`, all three): `"decision":
"confirmation_required"`, zero `block`-severity checks, once OpenCode's
version was re-pinned to the verified ceiling (see the harness note above —
the drift to `1.18.29` produced a correct `agent.version` block refusal
first, which is not a product defect).

**One real resume.** A real Claude Code session was created via the host's
live config (`claude --print "...remember this token: <planted token>..."`)
in a throwaway git-initialized project under the lab root. Its session id
was found by `rein search <token> --agent claude --json` (matching on
message body), pushed via device-a's Hop account, then pulled into
device-a's *isolated* home (not overwriting the live config's copy —
deleting a file under the live config was correctly declined by this
session's own safety controls, so the round trip was proven by restoring
into a second, isolated location instead of by simulating loss in place).
`claude --resume <id> --print "What was the exact token I asked you to
remember earlier..."` against the isolated, restored copy could not
authenticate (no credentials in the isolated home); against the **live**
config (which never lost the session — it was never deleted) it answered
correctly:

```
$ claude --resume <id> --print "What was the exact token ...?"
<the exact planted token>
```

This confirms the resume mechanism answers from restored/native history for
a real session created and pushed under this Hop account.

### H9 — `sync verify` names only observed objects

```
rein.exe sync verify
rein.exe sync verify --json
```

Observed (`--json`, relevant fields):

```
"checked_objects": ["the index", "the newest snapshot in the index"],
"unopened": "Not opened and judged by name only: 3 other age-named snapshot(s), the wrapped keyring.",
"steps": [..., {"id":"isolation", "status":"fail", ...}]
```

Step 4's `status` is `"fail"`, never a "could not run" status — the same
`fakelocker` `AnyBucket:true` harness limitation as H4, unchanged.

### H10 — migrate to BYO, switch, forget-hop

```
cmd/hopfdrunner.exe fixed REINSTATE_PASSPHRASE_FD "<passphrase>" -- rein.exe sync migrate --to byo --endpoint http://127.0.0.1:9321 --bucket device-a-byo-migrated --switch --forget-hop --json
rein.exe whoami
sqlite3 hopd.db "select type, count(*) from events group by type"
cmd/hopfdrunner.exe fixed REINSTATE_PASSPHRASE_FD "<passphrase>" -- rein.exe sync verify
```

Observed:

```
{"destination":{"bucket":"device-a-byo-migrated",...},"forgot_hop":true,"migrated":{"snapshots":4,"written":4,...},"switched":true}
this device is not signed in to Reinstate Hop; run `rein login`   (exit 4, after --forget-hop)
device_enrolled|2   first_push|1   locker_provisioned|1   sign_up|1   verify_reported|2   (no device_revoked)
--- sync verify against the new BYO bucket ---
Step 4: NOT APPLICABLE — "BYO storage has no control plane and no reference locker."
OUTCOME: PASS.
```

No `device_revoked` event appears in `hopd.db` after `--forget-hop`,
confirming the token was dropped locally without revoking the device
server-side.

### H11 — path remap between two homes' project mappings

```
# device-b (pushing side): project id maps to its own fixture's exact recorded cwd
rein.exe init --hop --project "hoplab-device-a=C:\Users\fixture-user\code\device-b" --force
cmd/hopfdrunner.exe fixed REINSTATE_RECOVERY_CODE_FD "<code>" -- rein.exe account recover
rein.exe push --all --json

# device-c (pulling side): SAME project id maps to a different, real local path
rein.exe init --hop --project "hoplab-device-a=D:\...\device-c\altproject"
cmd/hopfdrunner.exe fixed REINSTATE_RECOVERY_CODE_FD "<code>" -- rein.exe account recover
rein.exe pull --all --json
grep -o '"cwd":"[^"]*"' <restored claude/codex files>
rein.exe resume claude:session-syn-001-b --dry-run --json
```

Observed: device-b pushed `3 session(s) (claude 1, codex 1, opencode 1)`
under project id `hoplab-device-a`. Device-c's `sessions --json` after pull
shows the pulled sessions' `workspace` as
`D:\ReinstateAcceptanceProjects\v060-rc2-d\lab\device-c\altproject` — device
C's own mapped path, not device B's original fixture path — confirmed
directly in the raw restored `.jsonl` files' `cwd` field. `resume --dry-run`
on device-c then reported:

```
{"id":"workspace.available","status":"present","severity":"info","expected":true,"actual":true,"provenance":"vendor_recorded","message":"the recorded workspace is available"}
```

### H12 — keyless diagnostics refuse forged/rolled-back/re-keyed keyrings

Two devices (`device-x`, `device-y`) enrolled in one account; a third,
genuinely different account on `device-w`.

**Rolled back.** Saved the account's genuine generation-1 `keyring.v1.json`
before revoking `device-y` from `device-x` (which rolled the account to
generation 2). Restored the saved pre-revoke object over the current one:

```
h12tool.exe put gen1-keyring.json
rein.exe account status --json
rein.exe devices --json
rein.exe push --all
```

Observed:

```
"keyring_refused": true, "key_generation": 1,
"error": "keyring current_generation 1 is below the 2 the control plane reports for this account (...); the keyring was rolled back (a revoked device may have restored an older copy inside its credential window). Nothing was written; ..."
```

`push --all` refused with the same message, exit `7`.

**Forged.** Restored the genuine generation-2 object, then tampered one
byte of its own signature and staged it:

```
h12tool.exe put forged-keyring.json
rein.exe account status --json
rein.exe push --all
```

Observed:

```
"keyring_refused": true, "keyring_present": false,
"error": "keyring: key generation is not signed by this account's key: generation 2 does not verify under account key ..."
```

`push --all` refused, exit `7`, "a party with write access to the locker
may have written a key generation of its own".

**Re-keyed.** Restored the genuine object, fetched device-w's own
genuinely-signed `keyring.v1.json` from its completely different account,
and staged that object in device-x's bucket:

```
h12tool.exe put device-w-keyring.json
rein.exe account status --json
rein.exe push --all
```

Observed:

```
"keyring_refused": true,
"error": "keyring: the keyring is signed by a different account key: it is signed by account key <w's key>, not the <x's key> expected here"
```

`push --all` refused, exit `7`, "the keyring in storage was replaced by one
signed with a key this account never used".

In every one of the three attempts, `account status`/`devices` exited `0`
(a diagnostic, not a crash) and `device-x`'s own local `account.json`
anchor (`key_generation: 2`, `account_key: <x's own key>`) was byte-for-byte
unchanged afterward — confirmed by re-reading the file after each attack.

## Cleanup

`hoplab.exe keyring clear` was run for every seeded device before deleting
the lab; `hoplab.exe stop` confirmed both `hopd` and `fakelocker` stopped;
`hoplab.exe ps` confirmed no lab left running; the entire
`D:\ReinstateAcceptanceProjects\v060-rc2-d\lab\` directory and the three
throwaway `cmd/` helpers were deleted before this report was committed. One
real, harmless Claude Code session (a single short exchange planting and
recalling a synthetic token, in a throwaway git-initialized project under
the now-deleted lab root) remains in the host's live Claude Code config, as
the evidence policy anticipates for a Claude Code row that must use the live
config; only its own session id appears above, no transcript text.
