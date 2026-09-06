# Hop parity journeys, native Windows — W6 executor B (H6–H8b), 2026-09-06

Physical, single-host journeys for hosted ticket #16 rows H6, H6b, H7, H8,
and H8b against a local, disposable lab: a real `hopd` (private control
plane) plus `scripts/testing/fakelocker` standing in for the bucket, both on
loopback with fake storage and a log-only email sender, per
[windows-acceptance-host.md](../windows-acceptance-host.md)'s Hop lab
section and `scripts/testing/hoplab/README.md`. Executor A ran H1–H5 and
H9–H12 in the same worktree and a separate lab root concurrently
(`docs/testing/results/2026-09-06-windows-hop-parity-v060-a.md`); the two
labs never shared a port, a `-root`, or a file — executor A used
`127.0.0.1:8301`/`9301` and `D:\ReinstateAcceptanceProjects\v060-hop-a\`,
this report used `127.0.0.1:8302`/`9302` and
`D:\ReinstateAcceptanceProjects\v060-hop-b*\`.

Contract: [`docs/testing/v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D. Task card:
[`docs/planning/v0.6.0-hop/task-cards/W6-hop-parity.md`](../../planning/v0.6.0-hop/task-cards/W6-hop-parity.md).

Every command below is real; the recovery code, the console session cookie
and CSRF token, and the device token are redacted (`<recovery-code>`,
`<token>`). Lab-root paths under `D:\ReinstateAcceptanceProjects\v060-hop-b*\`
are synthetic and not private, and are shown in full, matching the shape of
every prior Hop lab report in this directory (and executor A's report for
this same run).

**Revised 2026-09-06, same day, in response to review.** Two things below
were corrected after a verifier's review of the first version of this
report, both re-run live rather than argued in the abstract:

1. **H6b (§3.2) is completely rewritten.** The first version let a pairing
   request expire before any approve attempt ran, which refuses at
   `internal/cli/pairing.go`'s `PendingPairings`/"no pending pairing
   requests" check and never reaches `pairingStillOpen`,
   `approvePairingRequest`, or `rollBackPairingWrap` at all — so "B's wrap
   is absent from every generation" was trivially true (nothing was ever
   appended) rather than evidence the rollback mechanism
   (`rollBackPairingWrap`/`UnenrolAppended`) works. §3.2 now forces the
   real race the contract names — a request still open on the approving
   device's own clock when it is read and answered, but expired by the
   time the relay call reaches `hopd` — using a small delaying proxy in
   front of the real `hopd` (no product code touched), and shows the
   rollback actually firing: the exact append, the exact refusal, the
   exact `rollBackPairingWrap` confirmation text, and the keyring
   afterward with no net trace of the joining device.
2. **The post-push verification `WARNING` this lab reliably produces on
   every account's first push is no longer trimmed out of the record.**
   The first version of this report showed only `push --all`'s final
   `pushed N snapshot(s)` line for H6, H8, and H8b's first pushes,
   omitting `internal/cli/verify.go`'s own post-push isolation check
   (step 4) failing against this lab's fake locker every time — a lab
   limitation (new finding F7, §6), not a product defect, and already
   independently found and disclosed by executor A as their F3 for the
   same run. The `(see F3)` pointer this report's own §1 carried for it
   was wrong (this report's F3 is unrelated, about agent version ranges)
   — a broken cross-reference the review correctly flagged. §3.0 below
   now shows the full, real, untrimmed block once, and §3.1/§3.4/§3.5
   point to it instead of silently trimming it.

## Verdict

- **Required rows run:** 5 of 5 assigned (H6, H6b, H7, H8, H8b).
- **PASS:** H6, H6b, H8, H8b (4 rows).
- **PARTIAL:** H7 — the foreground loop's debounced push-on-change and
  scheduled pull are fully evidenced live; the Task Scheduler
  install/status/stop/start/uninstall round trip could not be run through
  the real `schtasks` because this session has no path to an elevated
  shell (exact reason in §4 and finding F4); the `svccheck` fallback was
  run per the task card and failed identically without elevation. Per the
  evidence rule, PARTIAL does not pass a required row.
- **Product defects found:** none in `internal/**`. No fix commit
  accompanies this report.
- **Non-blocking findings:** seven, recorded in §6 for the coordinator's
  attention (F1 shared auth gap, F2 conptydriver TTY detection, F3 range
  widening not yet on this branch, F4 Task Scheduler elevation, F5
  extensionless build output, F6 PowerShell `ScheduledTasks` hangs on
  secure-desktop consent, F7 `fakelocker`'s `AnyBucket` fails the
  post-push isolation check on this lab). None are release-blocking.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-05 evening through 2026-09-06 (IST `Asia/Kolkata`, UTC+05:30; timestamps below are as printed by each tool) |
| Host | Windows 11 Pro `10.0.26200`, native `windows-amd64` (not WSL) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-w6-hop-parity`, branch `v060/w6-hop-parity` |
| Tested commit | `b6ed9dc54acb4ccf2719d3a1aa2ce7ce053ca673` (`release/v0.6.0-rc.1` tip at branch time; confirmed with `git rev-parse HEAD`). The §3.2/§3.0 fix round below was re-run at this branch's then-current tip, `456a9296506f5761b8e5707ab8dd61669afd8c6f` — `git log b6ed9dc5..456a9296 -- internal/ cmd/` is empty, so this is the identical product code, not a moving target |
| Client build | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go build -o bin/rein.exe ./cmd/reinstate` (and `bin/reinstate.exe`, byte-identical); `rein version` reports `v0.5.2-rc.1-185-gb6ed9dc5 (b6ed9dc54acb4ccf2719d3a1aa2ce7ce053ca673 2026-09-05T20:14:34Z)` for the original journeys, `v0.5.2-rc.1-188-g456a9296 (456a9296506f5761b8e5707ab8dd61669afd8c6f 2026-09-05T21:42:02Z)` for the §3.0/§3.2 fix round |
| Go | `go1.25.13 windows/amd64` (`GOTOOLCHAIN=go1.25.13`) |
| hopd | Prebuilt binary at a scratch path (`REINSTATE_HOPD_BIN`); private `reinstate-hosted` control plane, fake storage (`HOPD_STORAGE=fake`), log email sender |
| fakelocker | `scripts/testing/fakelocker`, in-memory fake S3, accepting `FAKEKEY*` access key ids, any bucket (see F7) |
| Lab roots | One fresh root per journey group, all `hopd` `127.0.0.1:8302` / locker `127.0.0.1:9302` (started, exercised, then `hoplab stop`ped before the next): `v060-hop-b` (H6), `v060-hop-b-h6b` (H6b, original attempt, `HOPD_PAIRING_TTL=5s`), `v060-hop-b-h7` (H7), `v060-hop-b-h8` (H8), `v060-hop-b-h8b` (H8b); after review, `v060-hop-b\h6b-fix3` (H6b redone, `HOPD_PAIRING_TTL=25s`, same two addresses, three earlier same-address attempts on this root — `h6b-fix`, `h6b-fix2` — stopped and discarded once each showed why the technique needed adjusting, see §3.2) |
| Agents on this host | `claude 2.1.261`, `codex-cli 0.149.0` (`codex` on `PATH`, resolves under `nvm4w`'s node shim dir), `opencode 1.18.27`, all real, installed, on `PATH` |
| Contaminating env vars | `REINSTATE_BACKEND=memory`, `REINSTATE_MEMORY_BACKEND_DIR`, `XDG_DATA_HOME`, `REINSTATE_S3_ACCESS_KEY_ID`, `REINSTATE_S3_SECRET_ACCESS_KEY` (all present at session start from earlier lab work) unset in every shell before running `rein`, `hoplab`, or `go test` |
| Isolation | `hoplab homes`/`hoplab env` gave each simulated device its own `REINSTATE_HOME`, `HOME`, `USERPROFILE`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME`, and the other catalog `RootEnv`s; every env block was materialized to a file once and sourced with an explicit guard (`case "$REINSTATE_HOME" in *<root>*<device>*) ... esac`) before any `rein` invocation, after a transient shell fork failure showed what an unguarded, silently-empty `eval` can do (see §5) |
| Secrets | Recovery codes and pairing codes fed through `REINSTATE_RECOVERY_CODE_FD`/`REINSTATE_PAIRING_CODE_FD` via `hoplab pair` for ordinary pairing, and via two small standalone Windows-handle-passing runners for everything hoplab's own subcommands do not wrap: `secretrun` (built the same way `scripts/testing/hoplab/secretfd_windows.go`'s `fixedSecretFD` does — `AdditionalInheritedHandles`, not shell fd redirection, which does not carry a real inheritable handle to a native child from Git Bash) for a secret already known in advance (the two `rein devices revoke` calls, and feeding an already-captured pairing code to `rein devices approve` in §3.2's race), and `liverun` (the same technique as `secretfd_windows.go`'s `liveSecretFD`/`runAccountInit`: a real pipe the child blocks reading until fed) for `rein account init` when the code cannot be known before the child prints it and `hoplab pair init` cannot be used because it always points `REINSTATE_HOP_URL` at the lab's real `hopd` itself (`scripts/testing/hoplab/env.go`'s `hopLabEnv`), which §3.2's race needs to not happen for device A; all three helpers (`secretrun`, `liverun`, and §3.2's delaying proxy `pairdelay`) deleted after use, never committed |
| Console flow | A second standalone Go program (`consoledrive`, plain `net/http`, no product code touched) drove hopd's `/console/sign-in/email` and `/console/devices/{id}/revocation` routes the way a browser does — POST the email form, read the magic link from `hopd.log` (log email sender), POST the confirm form, then GET the devices page for the CSRF token and POST the revocation-request form; deleted after use, never committed |
| Cleanup | Every lab `hoplab stop`ped at the end of its group; `hoplab ps` empty at the end of this session |

## 2. Section D — Hop parity journeys (H6–H8b)

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| H6 | Device B `rein account join`; device A `rein devices approve`; B pulls | `PASS` | §3.1 |
| H6b | An expired pairing request is refused or rolled back; B's wrap is absent from every generation | `PASS` | §3.2 |
| H7 | `rein daemon install/status/stop/start/uninstall` round trip through real Task Scheduler; foreground loop debounced push-on-change and scheduled pull | `PARTIAL` — loop mechanics `PASS`; Task Scheduler round trip not run (elevation unavailable, F4) | §3.3 |
| H8 | A revokes B: generation rolls, B's token refused, B cannot open a later push; lagging-device attack and 404-floor proxy variant (automated, real `hopd`) | `PASS` | §3.4 |
| H8b | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | `PASS` | §3.5 |

`PARTIAL` and `NOT TESTED` do not pass a required row (evidence policy); H7
is recorded `PARTIAL`, not `PASS`, for exactly that reason.

## 3. Journeys in detail

### 3.0 A harness fact behind every first push below (F7)

Every device's *first* `push --all` in this lab prints a
`WARNING: the verification after the first push FAILED` block —
`internal/cli/verify.go`'s post-push isolation check (its step 4: prove
this account's credentials are refused by a bucket that is not its own)
fails every time, because `scripts/testing/fakelocker`'s
`s3test.Fake{AnyBucket: true}` answers *any* bucket name with the same
in-memory store instead of refusing one that is not this account's own —
a fake-locker limitation (F7, §6), not a product defect: the identical
check correctly reports `NOT APPLICABLE` against a BYO destination with no
control plane at all, and H8's own crossplane acceptance test (§3.4)
exercises the real product code the same way against the real `hopd`
binary with no such gap. This is independently the same behavior executor
A's report for this same run found and disclosed as their F3.

The first version of this report trimmed every shown `push --all`
transcript (§3.1, §3.4, §3.5) down to its final `pushed N snapshot(s)`
summary line, silently dropping this block — which a review of that
version correctly flagged: the trimming made every first push look
cleaner than this harness actually produces, and this report's own §1
pointed at the wrong footnote (`F3`, actually about agent version ranges)
for it. Rather than re-run three labs to re-capture byte-identical
behavior, one full, real, untrimmed capture is shown once here — the same
`push --all` code path every other push transcript in this report used,
run fresh in this fix round against the same `fakelocker` and `hopd`
binaries, on a first push for a freshly initialized account:

```text
$ rein push --all
WARNING: the verification after the first push FAILED. Full report:
VERIFICATION REPORT
Generated 2026-09-05T21:52:07Z by reinstate v0.5.2-rc.1-188-g456a9296 (...); storage: Reinstate Hop locker.
Checked: lk-a3xk1enghshkz8t9073tf0f3z4 at http://127.0.0.1:9302.

Step 1: List the locker with this device's credentials
  Result:         PASS
Step 2: Fetch an object and check it is ciphertext
  Result:         PASS
Step 3: Decrypt the object locally
  Result:         PASS
Step 4: Prove this account's credentials are refused from another bucket
  What was done:  Asked the control plane for its reference locker (a bucket the operator owns, holding one probe object), checked that it names a different bucket at the same storage endpoint step 1 listed ..., then tried to list it and read the probe with the same credentials that just listed this account's locker ...
  What was seen:  Listing the reference locker SUCCEEDED and returned 0 object(s). This account's credentials reach a bucket that is not its own. Could not run: reading the probe object neither succeeded nor was refused as access denied: the storage endpoint says there is no such object (backend: not found).
  Result:         FAIL

OUTCOME: FAIL. This account's credentials reached a bucket that is not its own, so they are not scoped to this locker. This is what these checks exist to catch. Keep this report; if the locker is a Hop locker, send it to security@reinstate.dev.
pushed 3 snapshot(s), skipped 0 unchanged, dry_run=false
```

Steps 1–3 (list, fetch ciphertext, decrypt locally) `PASS` every time —
what fails is specifically step 4's per-bucket credential scoping, which
`fakelocker`'s `AnyBucket=true` cannot emulate by design. Every trimmed
`push --all` quote below (§3.1, §3.4, §3.5) should be read with this exact
block understood to have appeared immediately before its own final
`pushed N snapshot(s)` line, on that push's account's first upload —
called out explicitly this time instead of silently absent.

### 3.1 H6 — live pairing, protocol v2, B pulls

Lab root `v060-hop-b`. Both devices signed in under the same lab email,
approved via `hoplab approve` watching `hopd.log`, then paired live:

```text
$ ./scripts/testing/hoplab/hoplab.sh pair init -root <root> -device device-a -rein bin/rein.exe
hoplab: device-a initialized the account; recovery code saved to <root>\hoplab-state.json for `pair recover`
<recovery-code>

$ ./scripts/testing/hoplab/hoplab.sh pair join -root <root> -device device-b -approver device-a -rein bin/rein.exe
hoplab: device-b joined the account live, approved by device-a
```

`rein account status --json` on device-a afterward: `enrolled_devices: 2`,
`key_generation: 1`, `device_in_keyring: true` for both device ids.

**Pairing protocol v2, confirmed at the wire, not assumed.** `hopd.db`'s
`pairing_requests` row for this join:

```
$ sqlite3 <root>\hopd.db "SELECT id, device_id, version, status FROM pairing_requests;"
b9d47bd2-387f-4a84-bf60-50cd060ec347|<device-b-id>|2|consumed
```

`version=2` is what `internal/hop/pairing.go`'s `PublishPairing` sends
(`"version": PairingVersion2`) in the real POST `/v1/pairing` this join
made; `status=consumed` is the request's normal successful ending.
Separate HMAC and payload keys are `internal/keyring/pairing.go`'s
`payloadKey`, domain-separated from the binding HMAC key by the Argon2id
master (code-read, not re-derived by hand here; the wire-level `version=2`
above is the part only a real client-against-real-`hopd` run can show).

**B pulls.** Device A pushed its three seeded fixtures — its own first
push, so the full output includes the §3.0/F7 `WARNING` block, trimmed
here to the final line (`push --all` -> "pushed 3 snapshot(s)"); device
B's first `pull --all` refused the Claude
snapshot's project (`hoplab-device-a`) with no local mapping — the correct,
documented path-remap refusal, not a pairing failure. Adding a
`[[projects]]` mapping to device-b's own `config.toml` for
`hoplab-device-a` pointing at a real local directory under its own lab
home (the same kind of local mapping `rein init --project` would write, and
what H11 exercises directly) let the retry complete cleanly:

```text
$ rein pull --all         # on device-b, after the mapping
pulled 3 snapshot(s), skipped 0 already synced, dry_run=false
  claude:session-syn-001-a -> ...\device-a-remap\session-syn-001-a.jsonl
  codex:rollout-syn-001-a -> ...\.codex\sessions\rollout-syn-001-a.jsonl
  opencode:ses_fixture001a -> ...\xdgdata\opencode\opencode.db
```

`rein sessions --json` on device-b afterward listed all six sessions (its
own three plus device-a's three), each under the correct device's own
workspace path, zero overlap, verified with an explicit
`REINSTATE_HOME`-prefix guard before the check (see §5).

**Verdict: PASS.**

### 3.2 H6b — forcing the real race, and watching `rollBackPairingWrap` fire

**Why this section was completely rewritten.** The first version of this
report let device B's pairing request expire on its own, unapproached, then
had device A try `rein devices approve` afterward. That refuses at
`internal/cli/pairing.go`'s very first check —
`PendingPairings` no longer lists an expired request, so `runDevicesApprove`
prints "no pending pairing requests" (line 416) and returns before
`pairingStillOpen`, `approvePairingRequest`'s keyring update, or
`rollBackPairingWrap` ever run. No wrap was ever appended for device B in
that run, so "B's wrap is absent from every generation" was true only
because there was nothing to roll back — not evidence the rollback
mechanism works. A review of this report caught exactly that gap. What
follows instead forces the actual race the row (and
`internal/cli/pairing.go`'s own comment on `rollBackPairingWrap`, lines
578–594) names: a request still open on the *approving* device's own clock
when it lists and answers it, but expired — checked against `hopd`'s own
clock, independent of anything the client believes
(`store.ApprovePairing`'s `WHERE ... AND expires_at > ?`) — by the time the
relay call (`POST /v1/pairing/{id}/approve`) actually lands.

**The tool that forces it, and why it is needed.** Nothing in `hopd` or
`rein` needed to change; the race only needed the approve call's arrival to
be delayed independently of the earlier listing call. `pairdelay` (a
~70-line standalone Go program, not part of the repository, deleted after
use) is a reverse proxy that sits in front of the real `hopd`: every
request is forwarded immediately except `POST /v1/pairing/{id}/approve`,
which it sleeps on first, so a client pointed at the proxy still lists
pending requests at full speed but has its own approval relayed late. Device
A's stored device token carries the control-plane URL it logged in against
(`internal/cli/hop_backend.go`'s `hostedSession` builds the client from
`tok.ControlPlaneURL`, not from `REINSTATE_HOP_URL` at call time — confirmed
the hard way, see the aside below), so device A signed in, ran
`rein init --hop`, and ran `rein account init` with `REINSTATE_HOP_URL`
pointed at the proxy (`http://127.0.0.1:8402`) from the very first command,
not only for the final approve; the recovery-code confirmation `account
init` needs (a code the process itself only reveals mid-run) was fed back
live through a small pipe-based helper (`liverun`, built the same way
`scripts/testing/hoplab/secretfd_windows.go`'s `liveSecretFD` is) since
`hoplab pair init` cannot be used here — it always points
`REINSTATE_HOP_URL` at the lab's real `hopd` itself regardless of the
caller's own environment (`scripts/testing/hoplab/env.go`'s `hopLabEnv`).
Device C (the joining device for this specific race; a fresh third device,
since the first two race attempts below ended in an ordinary successful
join and left device B already enrolled) logged in and ran
`rein init --hop`/`rein account join` normally, straight against the real
`hopd` — only the *approving* side needed to sit behind the proxy.

**Aside: two earlier attempts on this same lab root, and what each showed.**
The first attempt used `HOPD_PAIRING_TTL=6s` and a 10s proxy delay, but the
few seconds of unavoidable overhead between capturing device B's printed
code and invoking the approve command (real, not simulated: separate tool
calls in this agent session) ate the whole 6s window, so `PendingPairings`
already excluded the request — the same "no pending pairing requests"
outcome the original H6b run hit, for the same reason, confirming that a
short TTL alone reproduces nothing without controlling the delay
independently. The second attempt raised the TTL to 25s but left device
A's `REINSTATE_HOP_URL` pointed at the real `hopd` (device A had logged in
before the proxy idea), so the approve call reached `hopd` directly and
device B was approved normally in under a second — this is what exposed
the `tok.ControlPlaneURL` fact above. Both are disclosed rather than
deleted from the record because each pinned down a real fact about this
codebase (`PendingPairings` genuinely excludes anything not `status=pending`
at read time; `hostedSession` genuinely ignores `REINSTATE_HOP_URL` once a
device has a stored token) that the third attempt's design depends on.

**The race, forced and confirmed at the wire.** Lab root
`v060-hop-b\h6b-fix3`, `HOPD_PAIRING_TTL=25s`, `pairdelay` delaying the
approve route by 27s. Baseline immediately before: device A
(`05a7073e…`) and device B (`416c92e2…`, ordinarily enrolled from the
second attempt above) hold root-key wraps under generation 1; device C
(`95ff8ded…`) is registered (signed in, `init --hop` run) but carries no
wrap. Device C's `rein account join` and device A's `rein devices approve`
(fed the code the instant it was captured, both within the same shell
invocation to remove inter-call latency) then ran:

```text
$ rein account join                                    # device-c, direct to the real hopd
Pairing code for this device (never sent to the control plane):
    <pairing-code>
...The request expires at 2026-09-05T21:51:04Z (Ctrl-C to cancel).

$ REINSTATE_PAIRING_CODE_FD=<fd> REINSTATE_HOP_URL=http://127.0.0.1:8402 rein devices approve   # device-a, through pairdelay
Device "Harjots-Beast" (windows-amd64) asked to join this account (request opened 2026-09-05T21:50:39Z).
Approve only if that machine is yours and is showing a pairing code right now.
the pairing request was already approved, collected, or cancelled; every wrap appended for device 95ff8ded-2339-4961-b734-e2ca4f632156 was removed again, nothing was approved
exit=4
```

`pairdelay`'s own log shows exactly when the approve call actually reached
`hopd`:

```
03:20:40 pairdelay: POST /v1/pairing/79a31e39-c022-483d-9267-43740efdfebe/approve matches -match; sleeping 27s before forwarding
```

Request opened 21:50:39Z, expired 21:51:04Z; the delayed POST landed at
`hopd` at ~21:51:07Z (03:20:40 IST + 27s ≈ 03:21:07 IST = 21:51:07Z) — after
expiry, not before. Device A's own local `pairingStillOpen` check ran at
~21:50:39.9Z, roughly 24 seconds of slack still on the clock it trusted; it
is the printed error text itself that proves the call reached the network
at all — a *local* expiry refusal reads "pairing request … expired at …;
nothing was approved" (`pairingStillOpen`'s own message), not "already
approved, collected, or cancelled" (`hop.ErrPairingDecided`'s message,
`internal/hop/pairing.go` line 81, reached here because device C's own
`rein account join` was itself continuously polling `ClaimPairing` while it
waited (`internal/hop/pairing.go`'s `WaitForPairing`), and that poll loop's
own store call is what flips a pending row to `status=expired` the moment
its own clock crosses `expires_at` (`internal/store/pairing.go`'s
`ClaimPairing`, private control-plane repo): by the time device A's delayed
approve landed, device C's own polling had already made that write, so
`store.ApprovePairing`'s fallback query no longer sees
`status = PairingPending` and falls through to the generic `ErrPairingState`
case instead of the plain `status == pending` `ErrExpired` one it would hit
if nothing had polled first — both still map, in `internal/cli/pairing.go`
line 553, to the identical `rollBackPairingWrap` branch, exactly as that
line's own condition names: `errors.Is(err, hop.ErrPairingExpired) ||
errors.Is(err, hop.ErrPairingDecided)`). The printed message —
"every wrap appended for device %s was removed again, nothing was
approved" — is `rollBackPairingWrap`'s own success text
(`internal/cli/pairing.go` line 557) and cannot be produced any other way:
it only prints when `appended` was non-empty (a wrap for device C really
was written to the keyring by this same call, moments earlier, inside
`approvePairingRequest`'s own `keyring.Update`) and the subsequent
`rollBackPairingWrap` call (`k.UnenrolAppended`) itself returned no error.

**Confirmed from every independent angle, not just the printed line.**
`hopd.db` afterward:

```
$ sqlite3 <root>\hopd.db "SELECT id, device_id, version, status, payload IS NULL FROM pairing_requests;"
a4cfa0d5-a897-4d98-8fda-dde0456394f2|416c92e2-...|2|consumed|1
79a31e39-c022-483d-9267-43740efdfebe|95ff8ded-...|2|expired|1
```

Device C's request (`79a31e39…`) is `status=expired`, `payload IS NULL`.
Device A's own `rein devices` and `rein account status --json` immediately
after:

```
05a7073e-...  ... holds a root-key wrap (key generation 1)
416c92e2-...  ... holds a root-key wrap (key generation 1)
95ff8ded-...  ... no root-key wrap yet
```
```json
{ "key_generation": 1, "enrolled_devices": 2, ... }
```

`enrolled_devices` is still 2 (A and B only) and device C's own line reads
identically to its state *before* the race — not because nothing was ever
written for it, but because whatever was written was taken back again, the
same as the pre-race state by coincidence of arithmetic, not by omission.
Device C's own `account join` process, independently, also timed out and
printed "the pairing request expired before it was approved and
collected" — the client-side view the original report's single-device
version already showed, now alongside the approving device's own
rollback, not instead of it. Since this account never rolled past
generation 1, "absent from every generation" is the whole (one-generation)
keyring here, exactly as the original report said — the difference is that
this time a wrap genuinely existed for one generation, briefly, and was
then genuinely removed, which is what the row and `UnenrolAppended` /
`rollBackPairingWrap` are actually for.

**Verdict: PASS**, on the mechanism the contract names, not on a
vacuously-true absence.

### 3.3 H7 — daemon: foreground loop `PASS`, Task Scheduler `PARTIAL`

Lab root `v060-hop-b-h7`, device-a signed in and root-key initialized.

**Debounced push-on-change (live, real fsnotify watcher, real hop
storage).** `rein daemon run --debounce 3s --pull-every 10s --verbose` in
the foreground; three appends to the seeded Claude fixture, one second
apart, then quiet. `daemon.log`:

```
20:30:20 daemon started pid=30828 backend=hop watch=fsnotify roots=3
20:30:21 pull: pulled 0 snapshot(s), skipped 0 already synced
20:30:25 push: pushed 3 snapshot(s), skipped 0 unchanged      (startup sync)
20:30:29 change: ...session-syn-001-a.jsonl
20:30:30 change: ...session-syn-001-a.jsonl
20:30:33 pull: pulled 0 snapshot(s), skipped 3 already synced
20:30:33 push: pushed 1 snapshot(s), skipped 2 unchanged      (the coalesced push)
20:30:33 change: ...session-syn-001-a.jsonl
20:30:36 push: pushed 0 snapshot(s), skipped 3 unchanged      (debounce fired again, nothing new)
```

Three real filesystem changes one second apart produced exactly **one**
meaningful push (`pushed 1 snapshot`), not three — the debounce coalesced
them, as the row requires.

**Scheduled pull (live, four consecutive intervals).** The same log,
continued:

```
20:30:21 pull ...
20:30:33 pull ...
20:30:45 pull ...
20:30:57 pull ...
```

Four pulls at ~12s apart against a configured `--pull-every 10s`, proving
the schedule fires repeatedly on its own, not once at start-up.

**Pull-before-resume hook.** `internal/cli/daemon.go`'s `resumePull` only
prints a note when the daemon's last pull was stale (`resumePullFresh =
15s`) *and* something new was actually pulled; with the daemon's own
10s schedule keeping every check inside that window throughout this run,
every `rein last --dry-run` and `rein resume` invocation during it
correctly said nothing extra — the daemon was already keeping it fresh, the
documented "nothing happens" branch (`pullBeforeResume`'s own comment: "the
daemon's own schedule keeps it fresh instead"). This account then hit its
lab-plan credential-mint rate limit (60/hour;
`docs/hop.md`'s locker limits) before a second device could be added to
force the "stale daemon, then something new arrives" branch that prints
`pulled N newer snapshot(s) before resuming` — that specific message text
is confirmed by code reading here (`internal/cli/daemon.go:435-446`) and by
the 2026-08-24 macOS daemon record's live demonstration of the identical
code path, not reproduced live again in this run. Recorded honestly rather
than reused as if it were.

**Task Scheduler round trip: not run, elevation unavailable.** This
session's shell is `HARJOTS-BEAST\admin`, a member of `BUILTIN\Administrators`
but running with the UAC-filtered (non-elevated) token
(`whoami /groups` shows the Administrators row "Group used for deny only").
`rein daemon install` failed exactly the way the 2026-08-24 record already
documented for this class of account:

```text
$ rein daemon install
install with schtasks: schtasks /Create /TN com.reinstate.daemon.5acebdcb /XML ...\com.reinstate.daemon.5acebdcb.xml /F: exit status 1: ERROR: Access is denied.
exit=1
```

Checked, not assumed, that elevation genuinely has no path in this
session: `HKLM:\...\Policies\System` reports `EnableLUA=1`,
`ConsentPromptBehaviorAdmin=5`, `PromptOnSecureDesktop=1` — any elevation
request needs an interactive consent click on the secure desktop, which
this automated session cannot supply. A live probe
(`Register-ScheduledTask ... -RunLevel Limited`) confirmed this is not
merely theoretical: it hung indefinitely (no error, no timeout) rather than
failing fast, until the harness's own tool timeout forced it to the
background, at which point it was killed and confirmed to have registered
nothing (`schtasks /Query /TN reinstate-w6b-probe` → "cannot find the file
specified"). See finding F6.

Per the task card, the fallback was run: `scripts/testing/svccheck`, which
shares the exact same `daemon.Manager`/`schtasks` code path, renders a
well-formed Task Scheduler 1.2 XML (logon trigger, named principal,
`RunLevel=LeastPrivilege`) and fails identically without elevation:

```text
manager: schtasks
install: FAILED: schtasks /Create /TN com.reinstate.daemon.svccheck.w6b /XML ...: exit status 1: ERROR: Access is denied.
status after install: installed=false running=false detail="not registered"
uninstall: ok
```

The XML itself is well-formed and matches the 2026-08-24 record's
already-accepted shape; what could not be produced in this session is the
OS actually accepting a `/Create` from this exact host/account without an
elevated shell. **Recorded `PARTIAL` for this sub-mechanism, not claimed
`PASS`.**

**Verdict: H7 overall `PARTIAL`** (foreground-loop mechanics `PASS`; Task
Scheduler round trip not run, elevation unavailable, F4).

### 3.4 H8 — revocation, the lagging device, the 404 floor

**Automated, against the real `hopd` binary (`-tags hopacceptance`,
`REINSTATE_HOPD_BIN`):**

```text
$ REINSTATE_HOPD_BIN=<hopd.exe> GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0 \
  go test -tags hopacceptance ./internal/cli/ -run 'TestKeyGeneration' -v -count=1
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (2.48s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    the floor route was asked for 4 times and answered 404 every time
    documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (2.10s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	4.696s
```

Both the lagging-device attack (refused on push, pull, and account status,
naming the control plane) and the 404-floor proxy variant (the documented
residual reproduced, not papered over) ran against the real binary, not a
fake.

**Live, two real `rein` processes, lab root `v060-hop-b-h8`.** Baseline:
device-b pushes successfully — its own first push, so the §3.0/F7
`WARNING` block appeared here too, trimmed to the final line
(`pushed 3 snapshot(s)`). Device-a revokes it
(recovery code fed through the `secretrun` handle-passing helper, §1):

```text
$ rein devices revoke <device-b-id>       # device-a, recovery code via REINSTATE_RECOVERY_CODE_FD
revoked device "Harjots-Beast" (<device-b-id>); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
...
```

Generation rolled 1 → 2. Device-b immediately after:

```text
$ rein push --all
this device's token was rejected by the control plane (revoked or stale); run rein login again
exit=4

$ rein whoami
this device's token was rejected by the control plane (revoked or stale); run `rein login` again
exit=4
```

B's token refused and B cannot open a later push — both the specific
requirement and the general mechanism.

**Verdict: PASS.**

### 3.5 H8b — Console-initiated revocation stays pending until confirmed

Fresh lab root `v060-hop-b-h8b`, devices A (`234ed1f8…`) and B
(`37e07ccb…`) paired, key generation 1.

**The Console route, driven for real, no browser needed but no shortcut
either.** hopd's Console email sign-in (`/console/sign-in/email` →
`hopd.log`'s log-sender block → `/console/sign-in/email/{token}`) is the
same log-sender mechanism `hoplab approve` already uses for ordinary
sign-in, just a different route and email subject ("Sign in to the
Reinstate Console"); a small standalone `net/http` program (`consoledrive`,
§1) performed the same two requests a browser's one click makes, then GET
`/console/devices` for the session's CSRF token and POST
`/console/devices/{id}/revocation`:

```text
$ consoledrive -action signin ...
<token>                                    # console session cookie

$ consoledrive -action request-revocation -device 37e07ccb-852a-496f-89cd-cbde426f4573 -cookie <token>
status=303 location=/console/devices
```

**Confirmed pending, and confirmed to do nothing yet.** Device-a's `rein
devices` immediately named the pending request:

```
pending revocation: Harjots-Beast (windows-amd64), Console request a8c97819-f254-4f26-bd7f-55f567e66d43; run rein devices revoke 37e07ccb-852a-496f-89cd-cbde426f4573 on another enrolled device
```

and, the specific claim the row makes, device-b was **not yet affected**:

```text
$ rein push --all          # device-b, right after the Console request, still "pending"
pushed 0 snapshot(s), skipped 3 unchanged, dry_run=false
$ rein whoami
Account: w6b-h8b@example.com
...                          # succeeds normally
```

(`skipped 3 unchanged` means this specific call is not device-b's first
push in this lab — its actual first push, during setup before the Console
request and not itself shown above, is where the §3.0/F7 `WARNING` block
would have appeared for this lab; this call, having nothing new to upload,
triggers no verification at all — `internal/cli/verify.go`'s check runs
only once per device, after a push that uploads something.)

**Completed only by the recovery-code command, writing a strictly newer
generation.** `rein devices revoke <device-b-id>` on device-a (recovery
code via `secretrun` again) both rolled the keyring and confirmed the
Console request in the same call:

```text
Console request a8c97819-f254-4f26-bd7f-55f567e66d43 confirmed after the keyring reached generation 2
revoked device "Harjots-Beast" (37e07ccb...); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

`hopd.db` afterward: `device_revocation_requests` row
`a8c97819-...|37e07ccb-...|confirmed|0|2` — `status=confirmed`,
`confirmed_generation=2`, strictly newer than the account's prior
generation (1). Device-b's next `push --all` was then refused
(`this device's token was rejected by the control plane`, exit 4) — the
same refusal H8 exercised, now reached through the Console path instead of
a device-initiated `rein devices revoke` with no pending request.

**Verdict: PASS.**

## 4. Bench facts worth recording for the next run

- `HOPD_PAIRING_TTL` (and, by the same mechanism, any other
  `HOPD_*`/`REINSTATE_*` name `hoplab start` does not itself override) is
  set by exporting it in the shell before `hoplab start` — `hoplab`'s
  `hopdCmd.Env = append(os.Environ(), <its own overrides>)` carries it
  through unchanged. No `hoplab` change was needed or made for H6b.
- Elevation genuinely has no path in this session (§3.3); a
  `Register-ScheduledTask ... -RunLevel Limited` probe hangs on the secure
  desktop rather than failing, so nothing short of a human clicking
  "Yes" (or a saved-credential `runas`, not available here) can register a
  Task Scheduler task from this account today.
- `make build`'s default output (`bin/reinstate`, `bin/rein`, no `.exe`)
  is not directly runnable as `hoplab pair -rein <path>`'s child process on
  this host: `hoplab`'s own `resolveReinBin` finds and `os.Stat`s the
  extensionless path successfully, but the subsequent `exec.CommandContext`
  spawn refuses it (`executable file not found in %PATH%`) even though the
  exact path was given. Building explicitly with `-o bin/rein.exe` (and a
  `bin/reinstate.exe` copy) works, and is what every command above used.
  `scripts/testing/hoplab` is W4-owned; not fixed here, flagged for
  awareness (F5).

## 5. A near-miss worth disclosing plainly

Partway through H6, a transient Git-Bash/MSYS resource failure
(`cygheap read copy failed`, `Resource temporarily unavailable` from a
`go run` subprocess spawn) caused one `eval "$(go run ... env ...)"` call to
silently produce no output. The next `rein sessions --json` in that same
shell then ran with none of the isolation variables set, fell back to this
host's ambient environment, and printed real session data belonging to
this machine's actual account into this session's own tool output —
nothing was pushed, written, or sent anywhere; it was a single read-only
listing. No content from that output appears anywhere in this report, this
worktree, or any command below it. Every subsequent `rein` invocation in
this session first materialized each device's env block to a file once
(`hoplab env ... > device.sh`) and sourced it behind an explicit guard
(`case "$REINSTATE_HOME" in *<root>*<device>*) ... ; *) exit 99 ;; esac`)
that aborts the command outright if the isolation variables are not
exactly what they should be, rather than trusting a bare `eval` to have
worked. This is a fact about this run's own tooling discipline, not a
product finding; recorded here because the ground rules ask for exactly
this kind of transparency.

## 6. Findings for the coordinator

**F1 — no OAuth-capable auth path for a fresh, isolated agent config
directory (shared with executor A's F1).** Same root cause, hit
independently in this workstream: a brand-new `CLAUDE_CONFIG_DIR`/
`CODEX_HOME`/`XDG_DATA_HOME` has no stored vendor credential, and this
sandboxed session has no browser or API key to complete one
non-interactively without touching state this task forbids. Codex CLI
`0.149.0` happened to be within its verified range on this branch (unlike
Claude/OpenCode, F3), so its `--dry-run` preflight and workspace/executable
checks were fully exercised for real (§3.1's H6 evidence, plus a codex
preflight `decision: confirmation_required` with a real, existing mapped
workspace) — but a genuinely live, answering-from-history resume was not
completed in this workstream either.

**F2 — `scripts/testing/conptydriver`'s child process does not appear
console-backed for `term.IsTerminal` in this sandboxed session.** A minimal
probe binary (`term.IsTerminal` on stdin/stdout/stderr, `golang.org/x/term`,
the same library `internal/sessionindex/launch.go` uses) run under
`conptydriver` — from both Git Bash and native PowerShell — reported
`false` for all three, and its own rendered VT frame and `-raw` byte
capture showed only conhost's startup escape sequences, never the probe's
own printed text; `rein resume`'s `RequireInteractiveTerminal` gate
therefore refused even through a real ConPTY here
(`native agent resume/fork requires an interactive terminal`). This is a
distinct finding from F1: codex's version and workspace checks passed
cleanly, so this session reached the TTY gate specifically, and it did not
open. `REINSTATE_ALLOW_NON_TTY_LAUNCH` was deliberately not used to force
past this — established project precedent
(`docs/testing/v0.4.0-rc.*-agent-verification-prompts.md`,
`docs/testing/grok-native-resume-acceptance.md`) says plainly it "is not an
acceptance shortcut" and "must not be set" for acceptance rows, and using
it here would make the resume evidence vacuous rather than real.
`scripts/testing/conptydriver` is W4-owned; not touched here. Recommend a
follow-up on this host specifically (a probe binary and repro steps are
available on request) before the next run relies on it for a live resume.

**F3 — verified ranges not yet widened on this branch (shared with
executor A's F2).** `internal/agents/catalog/claude.go`
(`2.1.219`–`2.1.238`) and `opencode.go` (`1.18.21`) do not yet include this
host's installed `2.1.261`/`1.18.27`; `codex.go`'s range already covers
`0.149.0`. W3's file per `file-ownership.md`, not touched here.

**F4 — Task Scheduler round trip needs elevation this session cannot
provide.** See §3.3 in full: `EnableLUA=1`, `ConsentPromptBehaviorAdmin=5`,
`PromptOnSecureDesktop=1`; no interactive user is present to approve a UAC
consent prompt, and an elevation attempt hangs rather than failing fast
(F6). H7's Task Scheduler sub-row is recorded `PARTIAL`, not `PASS`, for
this reason; the `daemon.Manager`/`schtasks` XML itself is unchanged from
the already-accepted 2026-08-24 shape and rendered correctly by `svccheck`.

**F5 — `make build`'s extensionless output is not directly runnable as
`hoplab pair -rein <path>`'s child process on this host.** See §4. Not a
product defect (`internal/**` is untouched by this); `scripts/testing/hoplab`
is W4-owned. Worth reconciling the Makefile's actual output name against
`scripts/testing/hoplab/README.md`'s stated default
(`bin/rein.exe`/`bin/reinstate.exe`, "make build's own output").

**F6 — a non-elevated `Register-ScheduledTask -RunLevel Limited` probe
hangs indefinitely on the secure desktop rather than failing.** Recorded
for any future automation that might reach for PowerShell's
`ScheduledTasks` module as an alternative to `schtasks.exe`: it does not
fail fast on this class of host either, it blocks on a consent prompt with
no visible error and no timeout of its own.

**F7 — `fakelocker`'s `AnyBucket` fails the post-push isolation check on
every first push in this lab (shared with executor A's F3).** See §3.0 in
full: `internal/cli/verify.go`'s step 4 (prove this account's credentials
are refused by a bucket that is not its own) fails every time in this lab
because `scripts/testing/fakelocker`'s `s3test.Fake{AnyBucket: true}`
serves any bucket name from the same in-memory store, so the reference
locker the check probes answers instead of refusing — a lab limitation,
not a product defect: the identical check correctly reports
`NOT APPLICABLE` against a BYO destination with no control plane at all,
and H8's own crossplane acceptance test (§3.4) exercises the real check
logic against the real `hopd` binary with no such gap.
`scripts/testing/fakelocker` is W4-owned; flagged for awareness, not fixed
here. (This report's first version trimmed this `WARNING` out of every
shown `push --all` transcript without saying so, and pointed a footnote at
the wrong finding for it — both corrected after review; see the note under
the title and §3.0.)

No product defect was found or fixed in this workstream; `product_defects`
is empty in the structured report.

## 7. Gates run on this worktree (no product code changed)

```text
gofmt -l .                                                          # empty
GOTOOLCHAIN=go1.25.13 go vet ./...                                  # clean
GOTOOLCHAIN=go1.25.13 go mod tidy -diff                             # empty
CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test ./... -count=1          # ok, all packages
CGO_ENABLED=1 GOTOOLCHAIN=go1.25.13 go test -race ./internal/... -count=1
```

The full-parallel `-race ./internal/...` run on this host produced several
`FAIL`s tied to `ThreadSanitizer failed to allocate ... (error code: 1455)`
(Windows `ERROR_COMMIT_LIMIT`) and one `open ...\importcfg: The system
cannot find the path specified` build error — resource contention from
running many TSan-instrumented packages in parallel on this host during a
session that had also been running several `hopd`/`fakelocker`/daemon
processes concurrently, not product races. Every package the parallel run
flagged (`internal/capability`, `internal/config`, `internal/crypto`,
`internal/doctor`, `internal/executabletrust`, `internal/sessionindex`'s
`TestExecLaunchRunnerClassifiesPreflightFailures`,
`internal/doctest`'s `TestProductionDeploymentRejectsInvalidWebsiteTagDate`,
and `internal/cli` itself, which the parallel run also failed with a
`ThreadSanitizer` allocation error) was re-run individually (`-p 1`) and
passed cleanly — `internal/cli` at `221.244s` alone versus failing outright
inside the parallel run. No test assertion was edited to reach these
results.
