# Hop parity journeys, native Windows — tagged-artifact executor E (H6–H8b), 2026-09-06

Physical, single-host journeys for hosted ticket #16 rows `H6`, `H6b`, `H7`,
`H8`, and `H8b` against a local, disposable lab: a real `hopd` (private
control plane) plus `scripts/testing/fakelocker` standing in for the bucket,
both on loopback with fake storage and a log-only email sender, per
[windows-acceptance-host.md](../windows-acceptance-host.md)'s Hop lab
section and `scripts/testing/hoplab/README.md`. Run against the tagged
`v0.6.0-rc.1` artifact (not a pre-tag snapshot) — this is the run that can
authorize stable `v0.6.0`'s Hop rows once the companion tagged reports for
sections A–C and H1–H5/H9–H12 are assembled alongside it.

Contract: [`docs/testing/v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D. Candidate dispatch:
[`docs/testing/v0.6.0-rc.1-agent-verification-prompts.md`](../v0.6.0-rc.1-agent-verification-prompts.md).
Prior evidence and methodology reused throughout:
[`2026-09-06-windows-hop-parity-v060-b.md`](2026-09-06-windows-hop-parity-v060-b.md)
(pre-tag run, same five rows, same executor slot).

Lab root `D:\ReinstateAcceptanceProjects\v060-w7b-e\lab*` (never inside a Git
checkout); `hopd` `127.0.0.1:8322`, `fakelocker` `127.0.0.1:9322` for the
main lab, plus one fresh root/port pair (`127.0.0.1:8324`/`9324`, and a
delaying proxy at `127.0.0.1:8402`) for the H6b race, stopped and deleted
before this report was written. This session ran concurrently with another
tagged-artifact executor's own lab on the same host (`v060-w7b-d`, ports
`8321`/`9321`) — confirmed via `hoplab ps` never to share a port or a lab
root.

Every command below is real; the recovery codes and pairing codes are
redacted (`<recovery-code>`, `<pairing-code>`) except where a code's exact
value is itself the evidence a race actually happened (H6b), in which case
it is shown because it is synthetic, single-use, and already expired by the
time this report was written. Lab-root paths under
`D:\ReinstateAcceptanceProjects\v060-w7b-e\` are synthetic and not private,
shown in full, matching the shape of every prior Hop lab report in this
directory.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.1` (published GitHub prerelease, signed, verified against `.github/allowed_signers`; release workflow run `34029733057`) |
| Full commit | `63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0` |
| Archive | `reinstate_0.6.0-rc.1_windows_amd64.zip` |
| Archive sha256 | `3d1ebf243c6d6f234ffe95955b9504502c340bfda16f42f776db97893bc26542` — matched `checksums.txt` in the coordinator's verified `rc1-draft` directory before install |
| Install source | Coordinator-verified release assets (checksums matched, `gh attestation verify` passed, `scripts/verify-release.ps1`/`scripts/test-install.ps1` both exit 0) |
| Install dir | `D:\ReinstateAcceptanceProjects\v060-w7b-e\install\` (this executor's own fresh directory) |
| `rein.exe`/`reinstate.exe` | Byte-identical (`cmp` exit 0; identical sha256 `d13d683eda6e1d59082be08c0326be196ffaa856cb8b83a35ef181d3ec624e2c`) |
| `rein version --json` | `{"commit":"63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0","date":"2026-09-06T11:17:45Z","name":"reinstate","version":"0.6.0-rc.1"}` |
| **Bootstrap deviation** | The dispatch asks for an install from the live bootstrap (reinstate.dev) after proving it pins `v0.6.0-rc.1`; at run time the live bootstrap still pins `v0.5.2-rc.1` because the guarded website deploy's test gate fails on three Windows-only test files, so the install came from the published release assets whose checksums and attestation were verified instead. |
| Worktree | `D:\Projects\reinstate-worktrees\v060-w7b-tagged`, branch `v060/w7b-tagged` at `63ac5a5b` (matches the release commit) |
| `hopd` source | Built from `D:\Projects\reinstate-hosted` (private control-plane checkout; lab dependency only, never committed to this repository) — a lab component, not the artifact under test |
| Go toolchain | `GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0`, host default `go1.26.1 windows/amd64` |
| Host | Windows 11 Pro `10.0.26200`, native `windows-amd64` (not WSL) |
| Host env guard | `REINSTATE_BACKEND` and `REINSTATE_MEMORY_BACKEND_DIR` explicitly `unset` at the top of every shell in this session before any `rein`/`hoplab`/`go test` invocation; `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` left untouched at the host's live values, isolation achieved per-row with each device's own home variables instead (see §1 harness note F1) |
| UTC date | 2026-09-06, 11:31Z–11:47Z |

## Verdict

- **Required rows run:** 5 of 5 assigned (`H6`, `H6b`, `H7`, `H8`, `H8b`).
- **PASS:** `H6`, `H6b`, `H7`, `H8`, `H8b` — **5 of 5**.
- **Product defects found:** none in `internal/**`. No fix commit accompanies
  this report.
- **Non-blocking harness findings:** two, F1 (`fakelocker`'s `AnyBucket`
  fails the post-push isolation check on this lab, matching the pre-tag
  report's own F7) and F2 (a Task-Scheduler-launched daemon resolves
  per-agent home roots from the scheduled task's own process environment,
  not from this session's isolated overrides, because Windows Task
  Scheduler XML carries no per-task environment block — confirmed in-code,
  `internal/daemon/service.go`'s `schtasksArguments`; not a defect, and no
  content was pushed while this was true, but worth a maintainer's
  attention for future acceptance labs). Neither is release-blocking.

`PARTIAL` and `NOT TESTED` do not pass a required row (evidence policy);
none of the five rows below needed that disposition this run — every
mechanism the contract names was exercised live, including the one that was
`PARTIAL` at pre-tag (`H7`'s Task Scheduler round trip, blocked there by no
path to an elevated shell; this session had the maintainer at the keyboard
to accept the UAC prompt).

## 1. Section D — Hop parity journeys (H6–H8b)

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| H6 | Device B `rein account join`; device A `rein devices approve`; B pulls | `PASS` | §2.1 |
| H6b | An expired pairing request is refused or rolled back; B's wrap is absent from every generation | `PASS` | §2.2 |
| H7 | `rein daemon install/status/stop/start/uninstall` round trip through real Task Scheduler; foreground loop debounced push-on-change and scheduled pull | `PASS` | §2.3 |
| H8 | A revokes B: generation rolls, B's token refused, B cannot open a later push; lagging-device attack and 404-floor proxy variant (automated, real `hopd`) | `PASS` | §2.4 |
| H8b | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | `PASS` | §2.5 |

## 2. Journeys in detail

### 2.0 A harness fact behind every first push below (F1)

Every device's *first* `push --all` in this lab prints a
`WARNING: the verification after the first push FAILED` block —
`internal/cli/verify.go`'s post-push isolation check (step 4: prove this
account's credentials are refused by a bucket that is not its own) fails
every time, because `scripts/testing/fakelocker`'s `s3test.Fake{AnyBucket:
true}` answers *any* bucket name with the same in-memory store instead of
refusing one that is not this account's own — a fake-locker limitation, not
a product defect (H8's own crossplane acceptance test, §2.4, exercises the
identical product code against the real `hopd` binary with no such gap).
Independently reproduced here exactly as the pre-tag report's F7 and
executor A's F3 for the same run already found; not re-litigated per row
below, shown once:

```text
$ rein push --all
WARNING: the verification after the first push FAILED. Full report:
VERIFICATION REPORT
Generated 2026-09-06T11:32:31Z by reinstate 0.6.0-rc.1 (63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0 2026-09-06T11:17:45Z); storage: Reinstate Hop locker.
Checked: lk-f7vbcffpkdr0sgg4md5wknv57w at http://127.0.0.1:9322.

Step 1: List the locker with this device's credentials — Result: PASS
Step 2: Fetch an object and check it is ciphertext — Result: PASS
Step 3: Decrypt the object locally — Result: PASS
Step 4: Prove this account's credentials are refused from another bucket
  What was seen:  Listing the reference locker SUCCEEDED and returned 0 object(s).
                  Could not run: reading the probe object neither succeeded nor was
                  refused as access denied: the storage endpoint says there is no
                  such object (backend: not found).
  Result:         FAIL

OUTCOME: FAIL. This account's credentials reached a bucket that is not its own...
pushed 3 snapshot(s), skipped 0 unchanged, dry_run=false
```

Steps 1–3 (list, fetch ciphertext, decrypt locally) `PASS` every time; only
step 4's per-bucket credential scoping fails, and only because of the fake
locker's `AnyBucket=true`. Every push transcript below should be read with
this exact block understood to appear before the final `pushed N
snapshot(s)` line on that account's first upload.

### 2.1 H6 — live pairing, protocol v2, B pulls

Lab root `v060-w7b-e\lab`. Both devices signed in under the same lab email
(`w7b-e-hoplab@example.com`), approved via `hoplab approve` watching
`hopd.log`, then paired live through `hoplab pair`, which drives the real
compiled `rein` binary under test:

```text
$ hoplab.exe pair init -root <lab> -device device-a -rein rein.exe
hoplab: device-a initialized the account; recovery code saved to <lab>\hoplab-state.json
<recovery-code>

$ hoplab.exe pair join -root <lab> -device device-b -approver device-a -rein rein.exe
hoplab: device-b joined the account live, approved by device-a
```

`rein account status --json` on device-a afterward: `"key_generation": 1,
"enrolled_devices": 2`.

**Pairing protocol v2, confirmed at the wire, not assumed.** `hopd.db`'s
`pairing_requests` row for this join:

```
$ sqlite3 hopd.db "SELECT id, device_id, version, status FROM pairing_requests;"
d3268cf5-e08a-481f-958e-38083cab0ea7|226da4bc-c2ce-411f-8d28-2b9d511804b2|2|consumed
```

`version=2` is what `internal/hop/pairing.go`'s `PublishPairing` sends
(`"version": PairingVersion2`) in the real POST `/v1/pairing` this join
made; `status=consumed` is the request's normal successful ending.

**B pulls.** Device A's own first `push --all` produced the §2.0/F1
`WARNING` block (elided here), ending `pushed 3 snapshot(s), skipped 0
unchanged`. Device B's first `pull --all` refused the Claude snapshot's
project (`hoplab-device-a`) with no local mapping — the correct, documented
path-remap refusal, not a pairing failure:

```text
$ rein pull --all         # device-b, before any mapping
claude snapshot project "hoplab-device-a" has no local mapping on this device;
configure it with rein init --project hoplab-device-a=/absolute/path ...
```

Adding a `[[projects]]` mapping to device-b's own `config.toml` for
`hoplab-device-a` pointing at a real local directory under its own lab home
(the same kind of local mapping `rein init --project` would write) let the
retry complete cleanly:

```text
$ rein pull --all         # device-b, after the mapping
pulled 3 snapshot(s), skipped 0 already synced, dry_run=false
  claude:session-syn-001-a -> ...\device-b\device-a-remap\session-syn-001-a.jsonl
  codex:rollout-syn-001-a -> ...\.codex\sessions\rollout-syn-001-a.jsonl
  opencode:ses_fixture001a -> ...\xdgdata\opencode\opencode.db
```

`rein sessions --json` on device-b afterward listed all six sessions (its
own three plus device-a's three), each under the correct device's own
workspace path, with no overlap — verified with an explicit
`REINSTATE_HOME`-prefix guard on device-b's own shell before the check.

**Verdict: PASS.**

### 2.2 H6b — forcing the real race, and watching `rollBackPairingWrap` fire

**What the row actually requires.** Letting a pairing request merely expire
before anyone tries to approve it refuses at `internal/cli/pairing.go`'s
very first check (`PendingPairings` no longer lists it), before
`approvePairingRequest` or `rollBackPairingWrap` ever run — that would make
"B's wrap is absent from every generation" trivially true because nothing
was ever appended, not evidence the rollback mechanism works. What follows
instead forces the real race the row (and `internal/cli/pairing.go`'s own
comment on `rollBackPairingWrap`) names: a request still open on the
*approving* device's own clock when it lists and answers it, but expired —
checked against `hopd`'s own clock — by the time the relay call actually
lands.

**The tool that forces it.** `pairdelay` (a small standalone Go reverse
proxy, not part of the repository, deleted after use) sits in front of the
real `hopd`: every request is forwarded immediately except `POST
/v1/pairing/{id}/approve`, which it sleeps on first. A fresh lab
(`127.0.0.1:8324`/`9324`, `HOPD_PAIRING_TTL=20s`) with the approving device
(device-a) signed in and initialized with `REINSTATE_HOP_URL` pointed at the
proxy (`http://127.0.0.1:8402`) **from the very first command** — login,
`init --hop`, `account init` — not only the final approve, because the
device token's stored `ControlPlaneURL` (not `REINSTATE_HOP_URL` at call
time) is what a later hop call actually uses
(`internal/cli/hop_backend.go`'s `hostedSession`). The joining device
(device-c) ran `rein account join` straight against the real `hopd` — only
the *approving* side needed the proxy. The recovery code and pairing code
(both generated mid-run, unknowable in advance) were fed back live through
a small Windows-handle-passing helper (`liverun`, built the same way
`scripts/testing/hoplab/secretfd_windows.go`'s live/fixed secret-FD helpers
are — `AdditionalInheritedHandles`, not shell FD redirection, which does
not carry a real inheritable handle to a native child from Git Bash).

**Timing that actually worked.** The client's own HTTP timeout
(`internal/hop/client.go`: `http.Client{Timeout: 30 * time.Second}`) bounds
how long a proxy delay can be before the approving client gives up with its
own `context deadline exceeded` instead of ever reaching `hopd` — two
earlier attempts in this session (`TTL=25s`/`delay=27s`,
`TTL=45s`/`delay=50s`) each failed for exactly the wrong reason (the first
too slow to react before the request's own local `PendingPairings` view
expired — the vacuous case again; the second past the client's own 30s
timeout, so the delayed call never completed at all). The combination that
forced the intended race: `HOPD_PAIRING_TTL=20s`, proxy delay `25s`
(`< 30s` client timeout), reacting to the printed pairing code within
seconds of it appearing:

```text
$ rein account join                                    # device-c, direct to the real hopd, 2026-09-06T11:40:03Z
Pairing code for this device (never sent to the control plane):
    <pairing-code>
The request expires at 2026-09-06T11:40:23Z (Ctrl-C to cancel).

$ REINSTATE_PAIRING_CODE_FD=<fd> rein devices approve   # device-a, through the proxy, called 2026-09-06T11:40:09Z
Device "Harjots-Beast" (windows-amd64) asked to join this account (request opened 2026-09-06T11:40:03Z).
the pairing request was already approved, collected, or cancelled; every wrap appended for device 7d80541b-634f-4340-b639-4345bd5dd41f was removed again, nothing was approved
exit=4
```

Request opened `11:40:03Z`, expired `11:40:23Z` (20s TTL); the approving
call started `11:40:09Z` (6s after creation — still locally "open," well
inside the 20s window) and returned at `11:40:34Z` (25s later, matching the
proxy delay), so the network call landed at `hopd` around `11:40:34Z` —
11 seconds *after* expiry. The printed text — "already approved,
collected, or cancelled...every wrap appended...was removed again, nothing
was approved" — is `rollBackPairingWrap`'s own success message
(`internal/cli/pairing.go`) and only prints when a wrap really was appended
moments earlier inside `approvePairingRequest` and the subsequent
`k.UnenrolAppended` call itself returned no error.

**Confirmed from every independent angle.** `hopd.db` afterward:

```
$ sqlite3 hopd.db "SELECT id, device_id, version, status, payload IS NULL FROM pairing_requests;"
153ce989-c276-4229-b8ac-bbba922c65a7|7d80541b-634f-4340-b639-4345bd5dd41f|2|expired|1
```

`status=expired`, `payload IS NULL`. Device-a's own `rein devices` and
`account status --json` immediately after:

```
9095c63c-...  holds a root-key wrap (key generation 1)     <- device-a itself
7d80541b-...  no root-key wrap yet                          <- device-c
```
```json
{ "key_generation": 1, "enrolled_devices": 1, ... }
```

`enrolled_devices` is 1 (device-a only) — not because nothing was ever
written for device-c, but because whatever was written was taken back
again. Device-c's own `account join` process, independently, printed "the
pairing request expired before it was approved and collected" — the
client-side view, alongside the approving device's own rollback, not
instead of it.

**Verdict: PASS**, on the mechanism the contract names, not on a vacuously
true absence.

### 2.3 H7 — daemon: foreground loop and the real Task Scheduler round trip

Lab root `v060-w7b-e\lab`, device-a (from §2.1) signed in and root-key
initialized.

**Debounced push-on-change (live, real fsnotify watcher, real hop
storage).** `rein daemon run --debounce 3s --pull-every 10s --verbose` in
the foreground; three appends to the seeded Claude fixture, one second
apart, then quiet. `daemon.log`:

```text
11:41:13 daemon started pid=50804 backend=hop watch=fsnotify roots=3
11:41:18 pull: pulled 0 snapshot(s), skipped 3 already synced
11:41:22 push: pushed 0 snapshot(s), skipped 3 unchanged      (startup sync)
11:41:34 pull: pulled 0 snapshot(s), skipped 3 already synced
11:41:34 change: ...session-syn-001-a.jsonl
11:41:34 change: ...session-syn-001-a.jsonl
11:41:34 change: ...session-syn-001-a.jsonl
11:41:37 push: pushed 1 snapshot(s), skipped 2 unchanged      (the coalesced push)
11:41:49 pull: pulled 0 snapshot(s), skipped 3 already synced
11:42:05 pull: pulled 0 snapshot(s), skipped 3 already synced
```

Three real filesystem changes one second apart produced exactly **one**
meaningful push (`pushed 1 snapshot`), not three — the debounce coalesced
them. Four pulls fired on their own repeating schedule (`~15-16s` apart
against a configured `--pull-every 10s`), proving the schedule fires
repeatedly, not once at start-up. Daemon stopped (`taskkill`) after this
observation.

**Task Scheduler round trip: run for real, elevated.** Per this
assignment's ground rule, an elevated PowerShell was launched with
`Start-Process -Verb RunAs` running a script that writes its own output to
a file; the maintainer was at the keyboard and accepted the UAC prompt
(`whoami /groups` inside the elevated process shows `BUILTIN\Administrators`
as an enabled, not deny-only, group — unlike the non-elevated session this
report's other rows ran in). The elevated script ran the full round trip
against the real `schtasks`:

```text
=== rein daemon install ===
installed schtasks com.reinstate.daemon.2c0abc73 (\com.reinstate.daemon.2c0abc73)
the daemon starts at login and is starting now; ...
install exit=0

=== rein daemon status ===
login:    schtasks com.reinstate.daemon.2c0abc73, installed (Running)
daemon:   stopped (last heartbeat just now)
status exit=0

=== schtasks /Query (reinstate) ===
  TaskName:      \com.reinstate.daemon.2c0abc73
  Status:        Running

=== rein daemon stop ===
stopped com.reinstate.daemon.2c0abc73
stop exit=0

=== rein daemon start ===
started com.reinstate.daemon.2c0abc73
start exit=0

=== rein daemon status (after start) ===
login:    schtasks com.reinstate.daemon.2c0abc73, installed (Running)
daemon:   running (pid 49816) since 2026-09-06T17:12:54+05:30, fsnotify watch, hop
status exit=0

=== rein daemon uninstall ===
uninstalled schtasks com.reinstate.daemon.2c0abc73; the log under ... is kept
uninstall exit=0

=== rein daemon status (after uninstall) ===
login:    schtasks com.reinstate.daemon.2c0abc73, not installed
daemon:   never ran for D:\ReinstateAcceptanceProjects\v060-w7b-e\lab\device-a\reinstate
status exit=0
```

`install`, `status`, `stop`, `start`, and `uninstall` all completed through
the real `schtasks`, confirmed independently by `schtasks /Query` showing
the task `Running` mid-run and a post-uninstall `schtasks /Query` (run from
a fresh, non-elevated shell) finding no `reinstate` task at all.

**Finding F2 — non-blocking, harness note, not a product defect.** The
elevated PowerShell process that launched `rein daemon install`/`start` had
this session's isolated device-a environment (`CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, `XDG_DATA_HOME`, `HOME`, `USERPROFILE` all pointed at
`v060-w7b-e\lab\device-a\home\...`) set only in *that* PowerShell's own
process; once `rein daemon install`'s scheduled task actually started the
daemon as its own fresh process (visible after `stop`/`start`, distinct pid
`49816`), its `status` output showed `watching:
D:\Projects\hop-10-lab\claude\projects` and the two other real host agent
roots — this host's live agent homes, not the isolated lab paths — because
Windows Task Scheduler's XML task definition carries no per-task
environment block at all (confirmed in code, not guessed:
`internal/daemon/service.go`'s `schtasksArguments` comment: "the home
travels as an argument (`--home`) because a scheduled task has no per-task
environment block"). `REINSTATE_HOME` survives via `--home` baked into the
task's own arguments (correctly still resolving to the isolated lab's
config/state), but the per-agent `*_HOME`/`*_DIR` overrides this lab
depends on for agent-root isolation do not, and the launched daemon fell
back to whatever the Task Scheduler launch context's own ambient
environment provided. **No content was pushed while this was true** —
`daemon.log` shows `push: not yet` throughout the brief window before this
session ran `daemon stop`/`uninstall`, and `hopd.db`'s `events` table has no
`push`-related event newer than this journey's own deliberate first push at
`11:32:31Z` (only `device_enrolled`/`pairing_*`/`first_push`/`verify_reported`
rows exist, all from before `11:41Z`) — confirmed by direct query before
this report was written, not assumed. Flagged for the maintainer's
attention for future acceptance labs that install the daemon via a real
scheduled task rather than running it in the foreground; this is inherent
to the Windows Task Scheduler API (no product code change implied) and does
not affect an ordinary end-user install, whose real agent homes and real
`REINSTATE_HOME` are exactly what the daemon is supposed to watch.

**Verdict: PASS** — every named mechanism (install/status/stop/start/
uninstall through real `schtasks`; debounced push-on-change; scheduled
pull) was exercised live.

### 2.4 H8 — revocation, the lagging device, the 404 floor

**Automated, against this executor's own real `hopd` binary
(`-tags hopacceptance`, `REINSTATE_HOPD_BIN`):**

```text
$ REINSTATE_HOPD_BIN=<own hopd.exe> GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0 \
  go test -tags hopacceptance ./internal/cli/ -run 'TestKeyGeneration' -v -count=1
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (1.86s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    the floor route was asked for 4 times and answered 404 every time
    documented residual reproduced: on a control plane that carries no floor,
    a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.35s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	3.312s
```

Both the lagging-device attack (refused on push, pull, and account status,
naming the control plane) and the 404-floor proxy variant (the documented
residual reproduced, not papered over) ran against the real binary this
session built from `reinstate-hosted`, not a fake.

**Live, two real `rein` processes, main lab.** Device-b (from §2.1) held
key generation 1. Device-a revoked it (recovery code fed through
`liverun`'s fixed-code mode):

```text
$ rein devices revoke 226da4bc-c2ce-411f-8d28-2b9d511804b2       # device-a, recovery code via liverun
Revoking "Harjots-Beast" (windows-amd64, 226da4bc-...). ...
revoked device "Harjots-Beast" (226da4bc-...); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
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

### 2.5 H8b — Console-initiated revocation stays pending until confirmed

Fresh devices in the main lab, device-d (`da6abe6a...`) and device-e
(`742e535b...`) paired live via `hoplab pair`, key generation 1. Device-e's
own baseline first push (the §2.0/F1 `WARNING` shown once already) ran
before the Console request, so its later push below shows `skipped 3
unchanged`, not a first-push verification block.

**The Console route, driven for real.** hopd's Console email sign-in
(`/console/sign-in/email` → `hopd.log`'s log-sender block →
`/console/sign-in/email/{token}`) is the same log-sender mechanism
`hoplab approve` already uses for ordinary sign-in, just a different route
and email subject ("Sign in to the Reinstate Console"). A small standalone
`net/http` program (`consoledrive`, not part of the repository, deleted
after use) performed the same requests a browser's one click makes — POST
the email form, GET then POST the confirm link, capture the session
cookie — then computed the CSRF token the same way
`internal/server/console.go`'s `consoleCSRF` does
(`sha256("reinstate/console/csrf/v1\x00" + sessionToken)`, base64
URL-encoded — read from source, not guessed) and POSTed the revocation
request:

```text
$ consoledrive -action signin-start -email w7b-e-h8b@example.com
status=200 ("Check your email")

$ consoledrive -action signin-confirm    # reads the magic link from hopd.log
status=303 location=/console            # <session-cookie> printed to stdout

$ consoledrive -action request-revocation -device 742e535b-5289-4111-92ef-21721618ba62 -cookie <session-cookie>
status=303 location=/console/devices
```

**Confirmed pending, and confirmed to do nothing yet.** Device-d's `rein
devices` immediately named the pending request:

```
pending revocation: Harjots-Beast (windows-amd64), Console request 09eab0b6-c597-4672-b530-602484aa8ba8; run rein devices revoke 742e535b-5289-4111-92ef-21721618ba62 on another enrolled device
```

and, the specific claim the row makes, device-e was **not yet affected**:

```text
$ rein push --all          # device-e, right after the Console request, still "pending"
pushed 0 snapshot(s), skipped 3 unchanged, dry_run=false
$ rein whoami
Account: w7b-e-h8b@example.com
...                          # succeeds normally
```

**Completed only by the recovery-code command, writing a strictly newer
generation.** `rein devices revoke <device-e-id>` on device-d (recovery
code via `liverun` again) both rolled the keyring and confirmed the Console
request in the same call:

```text
Console request 09eab0b6-c597-4672-b530-602484aa8ba8 confirmed after the keyring reached generation 2
revoked device "Harjots-Beast" (742e535b-...); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

`hopd.db` afterward: `device_revocation_requests` row
`09eab0b6-...|742e535b-...|confirmed|2` — `status=confirmed`,
`confirmed_generation=2`, strictly newer than the account's prior
generation (1). Device-e's next `push --all` was then refused (`this
device's token was rejected by the control plane`, exit 4) — the same
refusal H8 exercised, reached here through the Console path instead of a
device-initiated `rein devices revoke` with no pending request.

**Verdict: PASS.**

## 3. Cleanup

Both labs (`v060-w7b-e\lab`, `v060-w7b-e\lab-h6b`) `hoplab stop`ped; the
`pairdelay` proxy and every `hoplab approve` watcher this session started
were killed; a post-run `hoplab ps` from this session showed only the
concurrent `v060-w7b-d` lab (owned by a different executor slot) still
running, never this session's own. The standalone helper programs
(`pairdelay`, `liverun`, `consoledrive`) live only under this executor's own
scratch tools directory, were never added to the repository, and are not
referenced by anything committed here. No transcript text, prompts,
credentials, private paths, or session ids belonging to anyone but this
session's own synthetic lab fixtures appear anywhere in this report.

## 4. Deferred: Apple Silicon macOS

Copied verbatim from `docs/testing/v0.6.0-windows-acceptance.md` section E,
the Hop-relevant row. `DEFERRED` in this report — none of it is claimed
here, and none of it is native-Windows evidence.

| Group | Rows | Contract | Result |
| ----- | ---- | -------- | ------ |
| Hop cross-device journeys | pairing macOS↔Windows; path remap Windows↔macOS both directions; daemon launchd round trip; first push on macOS | hosted #16 | DEFERRED |
