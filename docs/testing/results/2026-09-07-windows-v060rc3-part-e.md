# v0.6.0-rc.3 tagged-artifact Windows acceptance — part E (Hop parity: H6, H6b, H7, H8, H8b)

`PHASE5-DEVICE-REPORT-V1` (partial — Hop parity rows only, this executor's assignment)

This is executor E's part file for the **published, signed GitHub
prerelease `v0.6.0-rc.3`**, satisfying the five rows of
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) section D
this executor was assigned: `H6`, `H6b`, `H7`, `H8`, `H8b`. Per
[`../v0.6.0-rc.3-agent-verification-prompts.md`](../v0.6.0-rc.3-agent-verification-prompts.md),
this candidate touches only the Cursor CLI store reader; nothing in Hop, the
daemon, or the interactive CLI changed, and none of these five rows is a
carried disposition — all five passed at `v0.6.0-rc.2`'s tagged run except
`H7` (`PARTIAL`, UAC declined, no operator available that run). This run
reuses `2026-09-07-windows-v060rc2-part-e.md`'s methods (`hoplab`, the
crossplane suite, a Console-driving probe) against a fresh, isolated lab on
this host, on the assigned ports.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.3` |
| Full commit | `202157c7877d33105dec700604dd23893b4d8b51` |
| Archive | `reinstate_0.6.0-rc.3_windows_amd64.zip` |
| Archive SHA-256 | `5fc5188ea92706e841d9c022cfe29ab386430a1a54a90139eec16d9baf756cd0` (re-verified against `checksums.txt` in the coordinator's staged draft directory before install) |
| Installed `rein.exe` / `reinstate.exe` SHA-256 | `6517281bc5a5984e59f030238e1525d355201bb849762db00e403a990fcde7d0` (both; `cmp` confirmed byte-identical) |
| `rein version --json` | `{"commit":"202157c7877d33105dec700604dd23893b4d8b51","date":"2026-09-07T02:36:11Z","name":"reinstate","version":"0.6.0-rc.3"}` |
| Install source | `checksums.txt`-verified copy of the coordinator's staged draft (`.../scratchpad/rc3-draft/reinstate_0.6.0-rc.3_windows_amd64.zip`), extracted into this executor's own fresh `D:\ReinstateAcceptanceProjects\v060-rc3-e\install\`. |
| Bootstrap deviation | Per the dispatch, only executor A installs from the live `https://reinstate.dev/install.ps1` and records the live-bootstrap identity. This part file installs from the coordinator's checksummed staged archive instead, not the live bootstrap; the live-bootstrap pin to `v0.6.0-rc.3` is executor A's evidence and is not re-verified here. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| Device | `windows-amd64`, native process, no emulation |
| OS/version/build | Windows 11 Pro, `10.0.26200.9278` |
| Go toolchain | `go1.25.13` via `GOTOOLCHAIN=go1.25.13` (host `go` reports `go1.26.1`) |
| Filesystem | NTFS |
| Date | 2026-09-07 (UTC evidence timestamps below) |
| Every shell | `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` run first in every shell that touched `rein`, `hoplab`, or `go test`; `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` left alone in every shell except where a device's own isolated home explicitly overrode them |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-rc3-e\` |
| `hopd` | `127.0.0.1:8322` (own build from `D:\Projects\reinstate-hosted` at commit `f44fbebe964a09fdf3801a0bc0365e0ed8083c9f`, `HOPD_STORAGE=fake`) |
| locker (`fakelocker`) | `127.0.0.1:9322` |

Throwaway helper files added to this worktree for this run only, never
committed, and removed from the tree in the same commit as this file:
`scripts/testing/hoplab/revoke.go` (drives `rein devices revoke` through
`REINSTATE_RECOVERY_CODE_FD`, the same fixed-secret-FD pattern
`pair.go`'s `pair recover` already uses) and
`scripts/testing/hoplab/consoleprobe.go` (drives the real Console HTTP
routes for H8b over a local `httptest.NewTLSServer` reverse proxy in front
of `hopd`, the same TLS-termination need the product's own
`console_test.go` documents). Both are plain additions to the existing
`scripts/testing/hoplab` package (same build tags, same `main` command
dispatch), not a new tool tree.

## Verdict for this part

- **Rows in this part:** 5 of the 16 Hop parity rows (section D)
- **Result:** `H6 PASS`, `H6b PASS`, `H7 PASS`, `H8 PASS`, `H8b PASS`
- **Required-row `PASS` count for this part:** 5 of 5
- **Release-blocking findings from this part (product):** 0
- **Urgent harness incident from this part (not a product defect, but
  requires immediate maintainer action):** this run's elevated `H7` test
  (`daemon stop` / `daemon start` restarting the scheduled task) picked up
  this host's **real, persistent, live** `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/
  `XDG_DATA_HOME` (`D:\Projects\hop-10-lab\...`) instead of this executor's
  isolated device-a home, because the Task Scheduler task definition
  `rein daemon install` writes only captures `--home`, not the three agent
  root-env variables. The daemon's own pull then wrote synthetic fixture
  session content into the host's real live Claude/Codex/OpenCode
  directories before this executor caught it. See "H7 — incident" below for
  the exact paths, the exact pristine backup this executor located but was
  blocked from restoring, and the remediation the maintainer must perform.
  This executor made **no further attempt** to write to or scan
  `D:\Projects\hop-10-lab` after the block, per the write-block's own
  guidance not to route around a denial.

## Summary table

| # | Row | Result | Mechanism exercised |
| - | --- | ------ | -------------------- |
| H6 | Device B `rein account join`; device A `rein devices approve`; B pulls | **PASS** | Live pairing via `hoplab pair join` driving the real installed binary; `pairing_requests` row confirmed `version=2, status=consumed, claims=2` at the wire; device B pulled all 3 of device A's sessions after adding a project mapping and resolving 3 sync conflicts (`--keep-remote`, from an earlier bucket-recovery step — see below); device B's `sessions --json` showed 6 sessions total across `device-b`, `pulled-from-a`, and `demo` projects, no key overlap |
| H6b | An expired pairing request is refused or rolled back; B's wrap is absent from every generation | **PASS** | Real expiry forced with a `hopd` instance restarted at the same ports/db with `HOPD_PAIRING_TTL=8s`: `rein account join` on device-c, deliberate 15s wait past expiry, then `rein devices approve` on device-a. Both sides refused client-side; `pairing_requests` row confirmed `status=expired, payload IS NULL`; `account status` on device-c showed `enrolled_devices=2`, `device_in_keyring=false` |
| H7 | `rein daemon install/status/stop/start/uninstall` round trip through Task Scheduler; the foreground loop pushes after a change (debounced), pulls on schedule, and pulls before a resume | **PASS** | Non-elevated `rein daemon install` independently confirmed elevation is genuinely required (`ERROR: Access is denied.`). The maintainer's go-signal file was already present; the elevated script ran to completion on the first correctly-scripted attempt (one earlier attempt failed on this executor's own PowerShell scripting bug, not elevation — see evidence). Full round trip: `install` → `status` (installed, running, independently confirmed by `schtasks /Query`) → `stop` → `status` (not running) → `start` → `status` (running again) → `uninstall` → `status` (`installed: false`) → `schtasks /Query` (task not found). Separately, the foreground loop: a real file change produced a debounced push (`push: pushed 1 snapshot(s)`); periodic pulls fired on the configured schedule; a real (non-dry-run) `rein resume` triggered its own pull exactly at the call's own timestamp once the daemon's last pull exceeded the 15s freshness window, confirmed by a new credential mint at that same second in `hopd.db`. **See "H7 — incident" below: the `stop`/`start` half of this round trip caused a real, unintended write to this host's live agent directories, remediated as far as this executor's write permissions allowed.** |
| H8 | A revokes B: generation rolls, B's token refused, B cannot open a later push; lagging-device attack and 404-floor proxy | **PASS** | `internal/cli/keygeneration_crossplane_test.go -tags hopacceptance` against this run's own built `hopd`: both `TestKeyGenerationFloorAgainstRealHopd` and `TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd` passed, 404-floor residual reproduced. Live: device A revoked device B via `rein devices revoke` (recovery-code FD), generation rolled 1→2; device B's subsequent `whoami` and `push --all` both refused, exit `4` |
| H8b | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | **PASS** | A purpose-built probe (`hoplab console-revoke`) drove the real Console HTTP routes end to end over a local TLS-terminating reverse proxy in front of `hopd` (the Console's session cookie is `Secure`-flagged and this lab's `hopd` speaks plain HTTP, so a spec-compliant client needs TLS to carry it — the same reason the product's own `console_test.go` wraps its harness in `httptest.NewTLSServer`): real email sign-in, CSRF-token scrape from the signed-in Console body, and a real `POST /console/devices/{id}/revocation`. `device_revocation_requests` row confirmed `status=pending`; the target device (device-d) stayed fully functional while pending (`whoami` exit 0); `rein devices revoke` on device A then printed "Console request … confirmed after the keyring reached generation 3"; the row's own DB state moved to `status=confirmed, confirmed_by=<device-a>, confirmed_generation=3`; device-d's next push refused, exit `4` |

## Evidence

Every command below ran with `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`
unset in the shell first. Session/device ids below are this executor's own
synthetic lab data (`exec-e@rc3.test`, a disposable email against the local
`hopd` only, never a real address), not real user data — the one exception
(the H7 incident) is called out explicitly with its own remediation section.

### H6 — live pairing, device B joins and pulls

```
hoplab.exe homes -root D:\ReinstateAcceptanceProjects\v060-rc3-e
hoplab.exe env -root <lab> -device device-a -shell sh   # (also device-b)
rein.exe login --email exec-e@rc3.test                  # device-a, device-b, approved via hoplab approve
hoplab.exe pair init -root <lab> -device device-a -rein rein.exe
hoplab.exe pair join -root <lab> -device device-b -approver device-a -rein rein.exe
```

Observed:

```
hoplab: device-b joined the account live, approved by device-a
```

Wire-level check (`hopd.db`):

```sql
select id, status, version, claims from pairing_requests order by created_at desc limit 1;
4f751d5e-24e2-45ed-85a9-a1bdcd5cc7c2|consumed|2|2
```

Device A pushed 3 sessions (`push --all`: `pushed 3 snapshot(s)`; step 4 of
the automatic post-push verification failed for the pre-existing
`fakelocker AnyBucket:true` harness limitation this lab has carried since
`v0.6.0-rc.1` — not a new finding, not scored against this row). Device B
added a project mapping for device-a's project (`hoplab-device-a`, since
`rein init --hop` on an already-initialized home correctly refuses without
`--force`; the mapping was added directly in `config.toml`, the documented
"equivalent config mapping" the product's own refusal message names) and
then pulled:

```
rein.exe pull --all --json
{"pulled": 3, "skipped": 0, ...}
```

`sessions --json` on device B afterward: 6 sessions total, `{'claude': 2,
'codex': 2, 'opencode': 2}`, across projects `{device-b, pulled-from-a,
demo}` — no key overlap.

**Note on this run's method (harness, not product):** partway through
building this row this executor restarted `hopd` (for the `H6b` short-TTL
test, below) without also restarting `fakelocker`; `HOPD_STORAGE=fake`'s
locker-provisioning bookkeeping is in-process memory only
(`storage.Fake`, `reinstate-hosted/internal/storage/fake.go`), so the
restart orphaned the account's existing bucket (`MintCredentials` then
correctly refused with `ErrBucketNotFound` → `storage_unavailable`, a
correct server response to a genuinely broken lab, not a bug). Recovered by
deleting the account's `lockers` row so `ProvisionLocker`'s
GET-then-provision-if-missing path (`H2`'s own documented mechanism) issued
a fresh bucket, then re-ran `account init`/`pair join`/`push --all` clean
against it — the `H6` result above is from that clean re-run, fully
wire-confirmed. Flagged for the maintainer as a lab-fragility note: `hoplab`
does not warn that restarting `hopd` alone (not the whole lab) silently
orphans every existing `HOPD_STORAGE=fake` locker.

### H6b — expired pairing request refused/rolled back

Restarted the lab's `hopd` at the same address/db with `HOPD_PAIRING_TTL=8s`
set (all other env identical to `start`'s own launch — `HOPD_STORAGE=fake`,
`HOPD_EMAIL_SENDER=log`, same `HOPD_DB_PATH`/`HOPD_S3_ENDPOINT`), stdout and
stderr merged to one log file so `hoplab approve`'s log-tailing could see
the printed sign-in link. A fresh device (`device-c`) signed in under the
same email, then:

```
rein.exe account join    # device-c, backgrounded; prints the pairing code and blocks
```

```
Pairing code for this device (never sent to the control plane):
    EN8V-S96B-BMVS-DSYW
The request expires at 2026-09-07T03:08:07Z (Ctrl-C to cancel).
```

Waited 15s past expiry, then:

```
rein.exe devices approve   # device-a, fed the (now-expired) code
```

Observed on device-a:

```
{"code":"usage","message":"no pending pairing requests; run rein account join on the new device first"}
```

Observed on device-c (the joiner itself, already unblocked by then):

```
the pairing request expired before it was approved and collected; run rein account join again on the new device
```
`[exited with code 4]`

Wire-level check:

```sql
select id, status, version, claims, payload is null from pairing_requests order by created_at desc limit 1;
e2829666-2aca-425d-9bc1-0e14e154495b|expired|2|5|1
```

`account status --json` on device-c afterward:

```
"enrolled_devices": 2, "device_in_keyring": false, "enrolled_on_this_device": false, "key_generation": 1
```

Device-c's wrap is absent from the only generation that exists at that
point (generation 1), confirming nothing was left behind for the expired
request.

### H7 — Task Scheduler round trip

Non-elevated confirmation first:

```
rein.exe daemon install --json
{"code":"runtime","message":"install with schtasks: schtasks /Create /TN com.reinstate.daemon.aa2a899e /XML ...: exit status 1: ERROR: Access is denied."}
```
`[exit=1]`

This independently confirms `schtasks /Create` genuinely needs elevation on
this host, not a harness misconfiguration.

Every other row in this part was completed first, then this executor polled
for `D:/ReinstateAcceptanceProjects/h7-go.txt` — it was already present on
the first check. Launched:

```
Start-Process -FilePath powershell.exe -ArgumentList '-NoProfile','-ExecutionPolicy','Bypass','-File','<lab>\h7-elevated.ps1' -Verb RunAs -PassThru -Wait
```

The UAC prompt was accepted (the maintainer was available). The elevated
script's **first** run had a bug of this executor's own making — the
PowerShell helper function's parameter was named `$args`, which collides
with PowerShell's automatic per-scope `$args` variable, so `@args` splatted
the wrong value and every step ran bare `rein` (printing top-level help)
instead of the intended subcommand. No elevation problem — `whoami` inside
the elevated shell correctly showed the admin account, and the run's own
`schtasks /Query` line still surfaced a real (if unhelpful) result. Fixed
the parameter name and re-ran the same elevated script once more; the
UAC prompt was accepted again immediately (the go-signal file was still
present) and the corrected script produced the full evidence below.

```
whoami (elevated shell): harjots-beast\admin

=== daemon install ===
installed schtasks com.reinstate.daemon.aa2a899e (\com.reinstate.daemon.aa2a899e)
the daemon starts at login and is starting now; rein daemon status shows it
exit=0

=== daemon status (after install) ===
"service": {"installed": true, "kind": "schtasks", "running": true, "detail": "Running"}
exit=0

=== schtasks /Query (independent confirmation) ===
TaskName:      \com.reinstate.daemon.aa2a899e
Status:        Running
Task To Run:   ...\rein.exe daemon run --home D:\ReinstateAcceptanceProjects\v060-rc3-e\device-a\reinstate
Schedule Type: At logon time

=== daemon stop ===
stopped com.reinstate.daemon.aa2a899e
exit=0

=== daemon status (after stop) ===
"service": {"installed": true, "running": false, "detail": "Ready"}
exit=0

=== daemon start ===
started com.reinstate.daemon.aa2a899e
exit=0

=== daemon status (after start) ===
"service": {"installed": true, "running": true, "detail": "Running"}
exit=0

=== daemon uninstall ===
uninstalled schtasks com.reinstate.daemon.aa2a899e; the log ... is kept
exit=0

=== daemon status (after uninstall) ===
"service": {"installed": false, "detail": "not registered"}
exit=0

=== schtasks /Query after uninstall (expect not found) ===
ERROR: The system cannot find the file specified.
```

Independently confirmed afterward, non-elevated: `schtasks /Query /TN
com.reinstate.daemon.aa2a899e` → "cannot find the file specified"; `Get-
Process -Name rein,reinstate` → none running. The full install → status →
stop → status → start → status → uninstall → status round trip completed,
every step `exit=0`, with `schtasks /Query` independently agreeing at both
the installed and the removed ends.

**Foreground-loop mechanisms**, exercised separately (non-elevated,
`rein daemon run`) against device-a's isolated home:

- **Debounced push after a change:** appended a line to device-a's seeded
  Claude session file; `daemon.log`:
  `change: ...session-syn-001-a.jsonl` followed 5s later by
  `push: pushed 1 snapshot(s), skipped 2 unchanged`.
- **Pulls on schedule:** `daemon.log` showed periodic
  `pull: pulled 0 snapshot(s), skipped 3 already synced` lines at the
  configured `--pull-every` cadence across three separate daemon runs.
- **Pulls before a resume:** started the daemon with `--pull-every 10m` (far
  longer than the 15s freshness window `pullBeforeResume` checks), waited
  20s past its own startup pull, then ran a real (non-dry-run)
  `rein resume claude:session-syn-001-a` (stdin closed). `hopd.db`'s
  `credential_mints` table recorded a new mint for device-a's device id at
  the exact same second the resume command started — no scheduled pull was
  due for another ~9 minutes, so this mint can only be the resume's own
  pre-resume pull. The resume itself then correctly refused at the
  environment preflight (`workspace.available: missing`, `agent.executable:
  missing`) because this isolated fixture's recorded `cwd`
  (`C:\Users\fixture-user\code\device-a`) does not exist on this host and
  the isolated `CLAUDE_CONFIG_DIR` has no real vendor binary metadata to
  read a version from — a correct refusal given the fixture, not part of
  what this row tests, and not scored against it (a `--dry-run` call earlier
  in this same investigation, which does **not** trigger `pullBeforeResume`
  by design, confirmed the launch plan itself was otherwise correct).

**H7 — incident (harness, urgent, not a product defect in the mechanism
under test).** The `daemon stop` / `daemon start` half of the elevated round
trip above launched the scheduled task's process without this executor's
per-shell `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` overrides — the
scheduled task definition (`Task To Run`, above) only captures `--home`
(`REINSTATE_HOME`), not the three agent root-env variables, and Task
Scheduler's own launched process instead inherited this host's real,
persistent user-level values. `daemon status` after `start` confirmed this
directly (`"roots": ["D:\\Projects\\hop-10-lab\\claude\\projects",
"D:\\Projects\\hop-10-lab\\codex\\sessions",
"D:\\Projects\\hop-10-lab\\xdg\\opencode"]` — the host's real, live agent
directories, not device-a's isolated fixture paths), and its very next pull
(`pulled 3 snapshot(s)`) wrote this executor's synthetic fixture session
content into them, confirmed by `daemon.log`'s own `change:` lines naming
those exact real paths:

- `D:\Projects\hop-10-lab\claude\projects\C--Users-fixture-user-code-device-a\session-syn-001-a.jsonl`
  — **new file**, did not exist before; safe to delete outright.
- `D:\Projects\hop-10-lab\codex\sessions\rollout-syn-001-a.jsonl` — **new
  file**, did not exist before; safe to delete outright.
- `D:\Projects\hop-10-lab\xdg\opencode\opencode.db` — **existing real file,
  modified in place.** `reinstate`'s own restore mechanism made a pre-write
  backup before merging (the same `backup_root` mechanism every pull in
  this report already relies on), at
  `D:\ReinstateAcceptanceProjects\v060-rc3-e\device-a\reinstate\backups\20260907T032147.889088900Z-opencode.db-store\opencode.db`.
  This executor verified by SHA-256 that this backup (
  `e2e8b1ce2e9aa2d784c84ea2faacc64ff81ca15b6820ec925e8468c8e631f99e`) differs
  from the current live file (
  `f62ad7855c935b9f2a91cccb1d1ae05e75cacceed66400ca4b5b309b24bc8deb`) —
  confirming the live file really was changed, and that this backup is the
  exact pristine pre-incident copy — but every attempt to copy it back over
  the live file, delete the two new files above, or even list the affected
  directory for a complete-change inventory, was **blocked by this session's
  own write-permission classifier** (a `D:\Projects\hop-10-lab\...` path
  guard). Per that block's own instruction not to route around a denial,
  this executor stopped and is reporting this instead of attempting a
  workaround.

  **Remediation the maintainer needs to perform** (three steps, all outside
  this executor's granted permissions):
  1. Copy `D:\ReinstateAcceptanceProjects\v060-rc3-e\device-a\reinstate\backups\20260907T032147.889088900Z-opencode.db-store\opencode.db`
     over `D:\Projects\hop-10-lab\xdg\opencode\opencode.db`, replacing the
     live file (verify the restored file's SHA-256 is
     `e2e8b1ce2e9aa2d784c84ea2faacc64ff81ca15b6820ec925e8468c8e631f99e`
     afterward).
  2. Delete `D:\Projects\hop-10-lab\claude\projects\C--Users-fixture-user-code-device-a\session-syn-001-a.jsonl`
     and its now-empty parent directory.
  3. Delete `D:\Projects\hop-10-lab\codex\sessions\rollout-syn-001-a.jsonl`.

  No `-wal`/`-shm`/`-journal` sidecar files were left behind for
  `opencode.db` at the time this executor checked (SQLite had already
  checkpointed), and no `.reinstate-opencode-restore-*` staging directory
  remained. This executor confirmed, before hitting the write block, that
  the scheduled task was fully uninstalled and no `rein`/`reinstate` process
  was still running, so nothing further should write to this path on its
  own — but this executor did **not** get to independently re-verify the
  live directory's final state after that point, since even a read-only
  recursive scan of it was blocked.

  **This is a real, disclosed defect in this executor's own H7 harness
  script** (an elevated helper that assumed the scheduled task's captured
  `--home` argument was sufficient to keep every subsequent `daemon start`
  isolated, when the agent-root environment variables also need pinning),
  compounded by a genuine product-documentation gap worth the maintainer's
  attention separately from this incident: `rein daemon install`'s Task
  Scheduler definition pins `--home` explicitly but does not pin
  `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`, so a later `daemon stop`
  / `daemon start` (or an ordinary login-triggered launch) can silently
  watch and push a **different** set of agent roots than whatever was true
  at `daemon install` time, with no warning printed anywhere. For a single
  real end-user this is arguably the intended behavior (always follow the
  live default), but it has no documented callout, and it is exactly what
  turned a should-have-been-isolated test into a real host-data incident
  here.

### H8 — revoke, lagging-device attack, 404-floor proxy

Crossplane suite against this run's own built `hopd`:

```
REINSTATE_HOPD_BIN=<lab>\hopd.exe GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0 \
  go test -tags hopacceptance ./internal/cli -run "TestKeyGenerationFloorAgainstRealHopd|TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd" -count=1 -v
```

```
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (1.72s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.24s)
PASS
```

Live revoke, this part's own two-device account (device A `b413aa2f-…`,
device B `27c6ea0a-…`, `key_generation: 1` before):

```
hoplab.exe revoke -root <lab> -device device-a -target 27c6ea0a-5192-42e0-ab43-6c053d4f8b0d -code "<device-a's recovery code>" -rein rein.exe
```

```
revoked device "Harjots-Beast" (27c6ea0a-…); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

Device B refused on both commands afterward:

```
rein.exe whoami     -> {"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run \`rein login\` again"}   (exit 4)
rein.exe push --all -> {"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run rein login again"}        (exit 4)
```

### H8b — Console-initiated revocation stays pending until the recovery-code command confirms it

Same account, a fourth device (device-d, `f3eae591-…`) paired live as the
target, `key_generation: 2` at the time of the Console request.

```
hoplab.exe console-revoke -hopd http://127.0.0.1:8322 -log <hopd combined log> -email exec-e@rc3.test -target f3eae591-13e6-46d9-a0cf-48a4ff2be259 -action revoke
```

```
consoleprobe: signed in to Console as exec-e@rc3.test, csrf=197PKI0q...
consoleprobe: POST /console/devices/f3eae591-13e6-46d9-a0cf-48a4ff2be259/revocation -> 200
ok
```

Wire-level check:

```sql
select id, status, target_device_id, requested_generation, confirmed_by, confirmed_generation from device_revocation_requests;
05cb64f3-a097-4394-a2d8-319333420825|pending|f3eae591-...|2||
```

Device-d still fully functional while pending:

```
rein.exe whoami --json   -> 200, device object returned   (exit 0)
```

Confirmation via the recovery-code command:

```
hoplab.exe revoke -root <lab> -device device-a -target f3eae591-13e6-46d9-a0cf-48a4ff2be259 -code "<device-a's recovery code>" -rein rein.exe
```

```
Console request 05cb64f3-a097-4394-a2d8-319333420825 confirmed after the keyring reached generation 3
revoked device "Harjots-Beast" (f3eae591-…); key generation 3 started with 1 enrolled device(s), and the control plane refuses its token
```

Wire-level check afterward:

```sql
select id, status, target_device_id, requested_generation, confirmed_by, confirmed_generation from device_revocation_requests;
05cb64f3-a097-4394-a2d8-319333420825|confirmed|f3eae591-...|2|b413aa2f-...|3
```

Device-d's next push refused:

```
rein.exe push --all --json -> {"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run rein login again"}   (exit 4)
```

## Findings from this part

| ID | Severity | Row(s) | Description | Release blocking |
| -- | -------- | ------ | ------------ | ----------------- |
| — | urgent, harness (this executor's own scripting mistake) | `H7` | An elevated `daemon stop`/`daemon start` picked up this host's real, persistent `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` instead of the isolated device-a home, because the Task Scheduler task definition only pins `--home`; the daemon's own pull then wrote synthetic session content into the host's **real** live Claude/Codex/OpenCode directories. A pristine pre-incident backup of the one modified real file was located and SHA-256-verified, but this executor's write access to that path was blocked before it could restore it; see "H7 — incident" above for the exact three-step remediation the maintainer must perform by hand. | Not a product defect in the row's own mechanism (which passed); blocking in the sense that real host data needs manual restoration before this host is trusted for further real-vendor rows |
| — | note, harness (lab fragility) | `H6` | Restarting only `hopd` (not the whole lab) with `HOPD_STORAGE=fake` silently orphans every account's existing locker bucket in memory, producing a `storage_unavailable` (502) the first time any device tries to read or write it; recovered by deleting the account's `lockers` row so `ProvisionLocker`'s own idempotent GET-then-provision-if-missing path (H2's documented mechanism) issued a fresh bucket. `hoplab`'s own docs do not warn about this. | No — recovered fully, `H6`'s reported result is from the clean re-run |

## Deletion of throwaway files

`scripts/testing/hoplab/revoke.go` and `scripts/testing/hoplab/consoleprobe.go`
are removed from the worktree in the same commit as this file, matching the
"never committed" convention the `v0.6.0-rc.2` part-e file (and `v0.6.0-rc.1`
before it) already established for this kind of helper.
