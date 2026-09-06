# v0.6.0-rc.2 tagged-artifact Windows acceptance — part E (Hop parity: H6, H6b, H7, H8, H8b)

`PHASE5-DEVICE-REPORT-V1` (partial — Hop parity rows only, this executor's assignment)

This is executor E's part file for the **published, signed GitHub prerelease
`v0.6.0-rc.2`**, satisfying the five rows of
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) section D
this executor was assigned: `H6`, `H6b`, `H7`, `H8`, `H8b`. It follows
[`../v0.6.0-rc.2-agent-verification-prompts.md`](../v0.6.0-rc.2-agent-verification-prompts.md),
which for these five rows carries them forward unchanged from the
`v0.6.0-rc.1` tagged run (all five `PASS`) with the instruction to re-run
them anyway because this is the first time they run against a `v0.6.0-rc.2`
tagged artifact. It reuses `2026-09-06-windows-v060rc1.md`'s part-E methods
(hoplab, the crossplane suite, a Console-driving probe) against a fresh,
isolated lab on this host.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.2` |
| Full commit | `81d74a82ba2a0e27f9f1a68eb0270d224a20da6a` |
| Archive | `reinstate_0.6.0-rc.2_windows_amd64.zip` |
| Archive SHA-256 | `82ea243cf9aa1b411cc77322abf973afaf8ff8ac13fbbfa05ad32c5d82ecdc84` (matches `checksums.txt` in the coordinator's staged draft directory) |
| Installed `rein.exe` / `reinstate.exe` SHA-256 | `0ae03c4eed8c1af610f04842d5ed51efdd9847724a3b841d249b3841e087fce6` (both; `cmp` byte-identical) |
| `rein version --json` | `{"commit":"81d74a82ba2a0e27f9f1a68eb0270d224a20da6a","date":"2026-09-06T21:57:00Z","name":"reinstate","version":"0.6.0-rc.2"}` |
| Install source | `checksums.txt`-verified copy of the coordinator's staged draft, into this executor's own fresh `D:\ReinstateAcceptanceProjects\v060-rc2-e\install\` — **not** the live bootstrap. Per the dispatch, only executor A installs from `https://reinstate.dev/install.ps1` and records the live-bootstrap identity; every other executor, including this one, installs from the coordinator's checksummed archive. This report claims no bootstrap evidence. |
| Bootstrap deviation | This part file installs from the checksummed staged archive, not the live bootstrap; the live-bootstrap pin to `v0.6.0-rc.2` is executor A's evidence, not re-verified here. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| Device | `windows-amd64`, native process, no emulation |
| OS/version/build | Windows 11 Pro, `10.0.26200` |
| Go toolchain | `go1.25.13` via `GOTOOLCHAIN=go1.25.13` (host `go` reports `go1.26.1`) |
| Filesystem | NTFS |
| Date | 2026-09-06 (UTC evidence timestamps below) — report committed 2026-09-07 |
| Every shell | `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` run first in every shell that touched `rein`, `hoplab`, or `go test`, confirmed by re-checking `env` before each row's commands |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-rc2-e\` |
| `hopd` | `127.0.0.1:8322` (own build from `D:\Projects\reinstate-hosted` at commit `f44fbebe964a09fdf3801a0bc0365e0ed8083c9f`, `HOPD_STORAGE=fake`) |
| locker (`fakelocker`) | `127.0.0.1:9322` |

Throwaway helper binaries built for this run only, from this worktree's own
tree, and never committed: `scripts/testing/hoplab` (the documented lab
driver), `cmd/hopfdrunner` (already present untracked in this worktree,
carried over from the `v0.6.0-rc.1` tagged run per its own findings §19 —
drives `rein` through `REINSTATE_RECOVERY_CODE_FD`/`REINSTATE_PAIRING_CODE_FD`
non-interactively), and `cmd/consoleprobe` (new this run — drives the real
Console HTTP routes for H8b; see its evidence section below). All device
homes, the lab's `hopd.db`, and these helper binaries live under
`D:\ReinstateAcceptanceProjects\v060-rc2-e\` and are deleted after this
report is committed. `cmd/consoleprobe` is deleted from the worktree in the
same commit as this file, matching `cmd/hopfdrunner`'s own "never committed"
convention.

## Verdict for this part

- **Rows in this part:** 5 of the 16 Hop parity rows (section D)
- **Result:** `H6 PASS`, `H6b PASS`, `H7 PARTIAL`, `H8 PASS`, `H8b PASS`
- **Required-row `PASS` count for this part:** 4 of 5 (`H7` does not pass a
  required row; recorded `PARTIAL` per the dispatch's own explicit fallback
  instruction for a declined/timed-out UAC prompt)
- **Release-blocking findings from this part:** 0 product defects. `H7`'s
  `PARTIAL` is a harness/host limitation (no interactive operator accepted
  the UAC elevation prompt this run), not a product defect — the daemon's
  own non-elevated behavior (`ERROR: Access is denied.` from `schtasks
  /Create`) is the documented, correct Windows Task Scheduler requirement,
  and `v0.6.0-rc.1`'s tagged run already exercised the full elevated round
  trip successfully once on this same host under the same mechanism.

## Summary table

| # | Row | Result | Mechanism exercised |
| - | --- | ------ | -------------------- |
| H6 | Device B `rein account join`; device A `rein devices approve`; B pulls | **PASS** | Live pairing via `hoplab pair join` driving the real installed binary; `pairing_requests` row confirmed `version=2, status=consumed` at the wire; device B pulled all 3 of device A's sessions after adding a project mapping; device B's `sessions --json` showed 6 sessions total under two distinct, non-overlapping workspaces |
| H6b | An expired pairing request is refused or rolled back; B's wrap is absent from every generation | **PASS** | Real race forced with `HOPD_PAIRING_TTL=8s` (no proxy needed — direct timing control): `rein account join` on B, deliberate 12s wait past expiry, then `rein devices approve` on A. Both sides refused client-side; `pairing_requests` row confirmed `status=expired, payload IS NULL`; `account status` on B showed `enrolled_devices=1`, `device_in_keyring=false` |
| H7 | `rein daemon install/status/stop/start/uninstall` round trip through Task Scheduler; foreground loop pushes after a change (debounced) and pulls on schedule | **PARTIAL** | UAC elevation declined twice (`Start-Process -Verb RunAs` → "The operation was canceled by the user."); non-elevated `rein daemon install` confirmed `schtasks /Create` genuinely needs elevation on this host (`ERROR: Access is denied.`). No further mechanism exercised this run — see evidence section for the exact reason and fallback per the dispatch |
| H8 | A revokes B: generation rolls, B's token refused, B cannot open a later push; lagging-device attack and 404-floor proxy | **PASS** | `internal/cli/keygeneration_crossplane_test.go -tags hopacceptance` against this run's own built `hopd`: both `TestKeyGenerationFloorAgainstRealHopd` and `TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd` passed, 404-floor residual reproduced. Live: device A revoked device B via `rein devices revoke` (recovery-code FD), generation rolled 1→2; device B's subsequent `whoami` and `push --all` both refused, exit `4` |
| H8b | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | **PASS** | A purpose-built probe (`cmd/consoleprobe`) drove the real Console HTTP routes end to end over a local TLS-terminating reverse proxy in front of `hopd` (the Console's session cookie is `Secure`-flagged and hopd itself speaks plain HTTP in this lab, so a spec-compliant client needs TLS to carry it — the same reason the product's own `console_test.go` wraps its harness in `httptest.NewTLSServer`): real email sign-in, CSRF-token scrape from `/console/devices`, and a real `POST /console/devices/{id}/revocation`. `device_revocation_requests` row confirmed `status=pending`; device B fully functional while pending (`whoami` exit 0, `push --all` succeeded); `rein devices revoke` on device A then printed "Console request … confirmed after the keyring reached generation 2"; the row's own `status=confirmed, confirmed_generation=2`; device B's next push refused, exit `4` |

## Evidence

### H6 — live pairing, device B joins and pulls

```
hoplab.exe homes -root D:\ReinstateAcceptanceProjects\v060-rc2-e\lab
hoplab.exe env -root <lab> -device device-a -shell sh   # (also device-b)
rein.exe login --email h6e@example.com                  # device-a, device-b
hoplab.exe pair init -root <lab> -device device-a -rein rein.exe
hoplab.exe pair join -root <lab> -device device-b -approver device-a -rein rein.exe -timeout 60s
```

Observed:

```
hoplab: device-b joined the account live, approved by device-a
```

Wire-level check (`hopd.db`):

```
sqlite3 hopd.db "select id, version, status from pairing_requests;"
c9abc620-87c3-4c5d-907b-4abc645fc80c|2|consumed
```

Device A pushed 3 sessions (`push --all`: `pushed 3 snapshot(s)`). Device B
added a project mapping for device-a's project (`hoplab-device-a`, since
`rein init --hop` on an already-initialized home correctly refuses without
`--force`; the mapping was added directly in `config.toml`, the documented
"equivalent config mapping" the product's own refusal message names), then:

```
rein.exe pull --all
pulled 3 snapshot(s), skipped 0 already synced, dry_run=false
  claude:session-syn-001-a -> ...\device-b\home\.claude\projects\...\session-syn-001-a.jsonl
  codex:rollout-syn-001-a -> ...\device-b\home\.codex\sessions\rollout-syn-001-a.jsonl
  opencode:ses_fixture001a -> ...\device-b\home\xdgdata\opencode\opencode.db
```

`sessions --json` on device B afterward: 6 sessions total — 3 under
`device-a-on-b` (the pulled project mapping) and 3 under `device-b` (its
own), no key overlap.

### H6b — expired pairing request refused/rolled back

Restarted the lab's `hopd` with `HOPD_PAIRING_TTL=8s` (direct timing control
instead of the rc.1 report's delaying reverse proxy — simpler and equally
real, since this executor controls the wall-clock gap directly rather than
racing a live approver). Fresh account, fresh devices (old keyring entries
cleared with `hoplab keyring clear`).

```
rein.exe account join     # device-b, backgrounded; prints the pairing code and blocks
# waited 12s (TTL 8s already elapsed)
rein.exe devices approve  # device-a, fed the code via stdin
```

Observed on device-a:

```
no pending pairing requests; run rein account join on the new device first
```

Observed on device-b (the joiner itself, moments later):

```
the pairing request expired before it was approved and collected; run rein account join again on the new device
```

Wire-level check:

```
sqlite3 hopd.db "select id, version, status, payload is null from pairing_requests;"
6351c417-211b-499e-a129-4cd9a7c00654|2|expired|1
```

`account status --json` on device B afterward:

```
"enrolled_devices": 1, "device_in_keyring": false, "enrolled_on_this_device": false, "key_generation": 1
```

B's wrap is absent from the only generation that exists (generation 1),
confirming `UnenrolEverywhere` left nothing behind for the expired request.

### H7 — Task Scheduler round trip (PARTIAL)

Per the dispatch: "launch it with PowerShell `Start-Process -Verb RunAs`
running a script that writes its output to a file; the maintainer is at the
keyboard and will accept the UAC prompt; if the prompt is refused or times
out, record PARTIAL with the exact reason."

```
Start-Process -FilePath powershell.exe -ArgumentList @("-NoProfile","-ExecutionPolicy","Bypass","-File","<lab>\h7-elevated.ps1") -Verb RunAs -PassThru
```

First attempt — the launcher itself failed before the elevated script ever
ran:

```
Start-Process : This command cannot be run due to the error: The operation was canceled by the user.
```

Retried once, same result:

```
ELEVATION FAILED: This command cannot be run due to the error: The operation was canceled by the user.
```

The elevated script's own output file (`h7-output.txt`) was never created
either time, confirming the elevated process never started — the decline
happened at the UAC prompt itself, before any daemon/`schtasks` command ran.

To rule out a product-side elevation requirement being avoidable, `rein
daemon install --debounce 1s --pull-every 5s` was also run **non-elevated**
directly:

```
install with schtasks: schtasks /Create /TN com.reinstate.daemon.6f181478 /XML <path>.xml /F: exit status 1: ERROR: Access is denied.
```

This confirms `schtasks /Create` genuinely requires elevation on this host
(not a harness misconfiguration), matching this same host's own
`v0.6.0-rc.1` tagged-run note that Task Scheduler installation needs an
elevated shell. No install/status/stop/start/uninstall round trip or
fsnotify-debounce/scheduled-pull mechanism was exercised this run.
**Recorded `PARTIAL`, reason: the UAC elevation prompt was declined (or the
elevation request itself was refused by the host) on both attempts; exact
message "The operation was canceled by the user."** This is not a carried
rc.1 disposition (rc.1's tagged run passed `H7` in full on this same
mechanism) — it is a new, host/operator-availability-dependent outcome for
this run, recorded per the dispatch's explicit fallback instruction rather
than left unattempted.

### H8 — revoke, lagging-device attack, 404-floor proxy

Crossplane suite against this run's own built `hopd`:

```
REINSTATE_HOPD_BIN=<lab>\..\bin\hopd.exe GOTOOLCHAIN=go1.25.13 \
  go test -tags hopacceptance ./internal/cli -run TestKeyGeneration -count=1 -v
```

```
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (1.82s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.64s)
PASS
```

Live revoke, fresh two-device account (device A `bdc5383f-…`, device B
`3b2c039e-…`, `key_generation: 1`):

```
hopfdrunner.exe fixed REINSTATE_RECOVERY_CODE_FD "<recovery code>" -- rein.exe devices revoke 3b2c039e-b1e9-49ad-8d66-268f82e31070
```

```
revoked device "Harjots-Beast" (3b2c039e-…); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

`account status --json` on device A afterward: `"key_generation": 2,
"enrolled_devices": 1`. Device B refused on both commands:

```
rein.exe whoami     -> this device's token was rejected by the control plane (revoked or stale); run `rein login` again   (exit 4)
rein.exe push --all -> this device's token was rejected by the control plane (revoked or stale); run rein login again      (exit 4)
```

### H8b — Console-initiated revocation stays pending until the recovery-code command confirms it

Fresh dedicated account/email (`h8bconsole@example.com`, never used
elsewhere this run, to stay under `HOPD_LOGIN_RATE_PER_EMAIL_PER_HOUR`'s
default cap of 5/hour) — device A `afa5110d-…`, device B (target)
`ccafe1b5-cd7f-4c92-9b03-de17f9f955f6`, `key_generation: 1`.

`cmd/consoleprobe` (new, throwaway, deleted with this commit) drives the
real Console over a local TLS-terminating reverse proxy in front of `hopd`
(needed because the Console's session cookie is `Secure`-flagged and
`hopd` speaks plain HTTP in this lab — the same reason
`internal/server/console_test.go`'s own harness wraps itself in
`httptest.NewTLSServer` rather than hitting hopd directly):

```
consoleprobe.exe flow http://127.0.0.1:8322 <lab>\hopd.log h8bconsole@example.com ccafe1b5-cd7f-4c92-9b03-de17f9f955f6
```

```
{"step":"signin","ok":true,"link":"http://127.0.0.1:8322/console/sign-in/email/…"}
{"step":"devices_before","ok":true,"csrf_found":true,"device_listed":true}
{"step":"revoke","ok":true,"status_code":200,"final_path":"/console/devices"}
```

Wire-level check:

```
sqlite3 hopd.db "select id, status, requested_generation, confirmed_generation, target_device_id from device_revocation_requests;"
93185673-f87c-439c-b869-ccb9faa94dc7|pending|0||ccafe1b5-cd7f-4c92-9b03-de17f9f955f6
```

While pending, device B (the target) is fully functional:

```
rein.exe whoami      -> Account: h8bconsole@example.com ... (exit 0)
rein.exe push --all  -> pushed 4 snapshot(s), skipped 0 unchanged, dry_run=false (exit 0)
```

Device A then confirms the pending request via the recovery-code command
(`hopfdrunner.exe fixed REINSTATE_RECOVERY_CODE_FD "<code>" -- rein.exe
devices revoke ccafe1b5-cd7f-4c92-9b03-de17f9f955f6`):

```
Console request 93185673-f87c-439c-b869-ccb9faa94dc7 confirmed after the keyring reached generation 2
revoked device "Harjots-Beast" (ccafe1b5-…); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

```
sqlite3 hopd.db "select id, status, requested_generation, confirmed_generation, target_device_id from device_revocation_requests;"
93185673-f87c-439c-b869-ccb9faa94dc7|confirmed|0|2|ccafe1b5-cd7f-4c92-9b03-de17f9f955f6
```

Device B's next push refused:

```
rein.exe push --all -> this device's token was rejected by the control plane (revoked or stale); run rein login again   (exit 4)
```

## Non-blocking findings

- **F-E1 (host/harness, new this run):** the elevated `Start-Process -Verb
  RunAs` request for `H7` was declined ("The operation was canceled by the
  user.") on both attempts; the exact same mechanism passed in full on this
  same host during the `v0.6.0-rc.1` tagged run. Nothing about the product
  changed between the two candidates in this area (the dispatch states
  Hop/daemon code is untouched this candidate); this is availability of an
  interactive UAC acceptor at the time of this run, not a regression. Not
  release-blocking; `H7` is not a carried rc.1 disposition and should be
  re-attempted before this candidate's overall device verdict is finalized
  if an elevated shell becomes available.
- **fakelocker `AnyBucket` (carried, unchanged):** step 4 of `push --all`'s
  automatic post-push verification reports `FAIL` against this lab's
  `fakelocker` (same-credentials-reach-another-bucket check) because
  `fakelocker`'s `AnyBucket:true` does not emulate per-bucket credential
  scoping — the same lab limitation every prior Hop-parity part file on
  this program has recorded (contrasted directly by `H8`'s real-`hopd`
  crossplane test exercising the identical product code path correctly).
  Not release-blocking, not new.
- Two untracked, uncommitted Go helper directories already present in this
  worktree before this run started (`cmd/h3probe`, `cmd/hopfdrunner`,
  carried from the `v0.6.0-rc.1` tagged run per its own findings §19) were
  left untouched; this run added and then removes its own
  `cmd/consoleprobe` in the same commit as this report.

## Cleanup

`D:\ReinstateAcceptanceProjects\v060-rc2-e\` (install, lab, device homes,
helper binaries) is deleted after this report is committed. `hopd` and
`fakelocker` processes for this lab are stopped
(`hoplab.exe stop -root D:\ReinstateAcceptanceProjects\v060-rc2-e\lab`).
`cmd/consoleprobe` is removed from the worktree in this same commit. No
transcript text, prompts, credentials, or private paths appear above beyond
this executor's own throwaway fixture paths and synthetic email addresses.
