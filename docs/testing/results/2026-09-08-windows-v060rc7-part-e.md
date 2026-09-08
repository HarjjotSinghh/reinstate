# `v0.6.0-rc.7` tagged Windows acceptance — part E (executor E: `H6`, `H6b`, `H7`, `H8`, `H8b`)

`PHASE5-DEVICE-REPORT-V1` (Hop-parity subset)

This is executor E's part of the `v0.6.0-rc.7` tagged-artifact Windows
acceptance, covering exactly five section D (Hop parity) rows: `H6`, `H6b`,
`H7`, `H8`, `H8b`. It does not stand alone as a device verdict — it is one
tagged executor's slice of the shared 216-row matrix; see
[`docs/testing/v0.6.0-rc.7-agent-verification-prompts.md`](../v0.6.0-rc.7-agent-verification-prompts.md)
and [`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) for the
full contract and how the parts are assembled into the device report. This
part reused the same methods `docs/testing/results/2026-09-08-windows-v060rc6-part-e.md`
recorded for `v0.6.0-rc.6`, re-run fresh against this candidate's own tagged
binary and its own disposable lab.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.7` |
| Full commit | `21e9a9a356df1181601e2da02bfae03b28380cc3` |
| Windows archive | `reinstate_0.6.0-rc.7_windows_amd64.zip` |
| Archive SHA-256 | `865ad8e8f65fb3f543d2e2670b5d7c82b0d4878513781e04868fc7303e183fc` (matches `checksums.txt`, re-verified independently before extraction) |
| `rein.exe`/`reinstate.exe` SHA-256 | `ab73dd7c6817da2374a9c72673c62c63fb724d7412fa464d5dc56d381d5ea34` (identical for both binaries; matches `checksums.txt`'s `.exe` entry) |
| `rein version --json` | `{"commit":"21e9a9a356df1181601e2da02bfae03b28380cc3","date":"2026-09-08T16:26:36Z","name":"reinstate","version":"0.6.0-rc.7"}` |
| Bootstrap deviation | Installed from the coordinator-verified, checksummed draft (`…\scratchpad\rc7-draft\reinstate_0.6.0-rc.7_windows_amd64.zip`), not the live `reinstate.dev/install.ps1` bootstrap — per the dispatch, only executor A installs from the live bootstrap; every other executor, including this one, installs from the shared checksummed archive after independently re-verifying its SHA-256 against `checksums.txt`. |
| Install directory (own, fresh) | `D:\ReinstateAcceptanceProjects\v060-rc7-e\install\` |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Microsoft Windows 11 Pro, `10.0.26200` (Build 26200), x64 |
| Git | `2.52.0.windows.1` |
| Go (toolchain) | `go1.26.1 windows/amd64` runtime; module pins `go1.25.13+` via `GOTOOLCHAIN` |
| Date (UTC) | 2026-09-08 |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-rc7-e\` (own, isolated from other executors' lab roots on the same host) |
| Hop lab (`H6`, `H8` live revocation) | `hopd 127.0.0.1:8322`, fakelocker `127.0.0.1:9322`, root `...\v060-rc7-e\lab\` |
| Hop lab (`H6b`, short-TTL) | `hopd 127.0.0.1:8323`, fakelocker `127.0.0.1:9323`, `HOPD_PAIRING_TTL=8s`, root `...\v060-rc7-e\lab-h6b\` |
| Hop lab (`H8b`, Console) | `hopd 127.0.0.1:8324`, fakelocker `127.0.0.1:9324`, root `...\v060-rc7-e\lab-h8b\` |
| Hop lab (`H7`) | `hopd 127.0.0.1:8325`, fakelocker `127.0.0.1:9325`, root `...\v060-rc7-e\lab-h7\` |
| `REINSTATE_BACKEND` / `REINSTATE_MEMORY_BACKEND_DIR` | Confirmed unset (`unset`/`Remove-Item Env:` first) in every shell that ran a `rein`, `hopd`, `hoplab`, or Go-test command this part used. |
| `CLAUDE_CONFIG_DIR` / `CODEX_HOME` / `XDG_DATA_HOME` | Never unset. Left at the host's live values throughout. The current shell's own transient `CODEX_HOME` (`C:\Users\admin\AppData\Roaming\orca\codex-runtime-home\home`) was found to differ from the **persistent, user-level** value (`D:\Projects\hop-10-lab\codex`, confirmed via `[Environment]::GetEnvironmentVariable('CODEX_HOME','User')`) — the exact trap the dispatch's "Lab isolation for `MatrixH:H7`" section warns about; the persistent user-level value (`CLAUDE_CONFIG_DIR=D:\Projects\hop-10-lab\claude`, `CODEX_HOME=D:\Projects\hop-10-lab\codex`, `XDG_DATA_HOME=D:\Projects\hop-10-lab\xdg`) was the one used for the `H7` before-listing. Every row that needed an isolated agent home (`H6`, `H6b`, `H8b`) pointed the vendor's own home variable (`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`, via `hoplab homes`' seeded device layout) at this part's own lab directories instead of unsetting the host variables. |

## Verdict (this part only)

- **Rows in this part:** 5 (`H6`, `H6b`, `H7`, `H8`, `H8b`), all required.
- **Result:** 4 PASS (`H6`, `H6b`, `H8`, `H8b`), 1 PARTIAL (`H7`, operator unavailable within the run's own 40-minute go-signal wait window).
- **Release-blocking findings from this part:** 0 (`H7`'s `PARTIAL` is a harness/operator-availability constraint, not a product defect — see "`H7`" below).

| # | Row | Result | Summary |
| - | --- | ------ | ------- |
| `H6` | Device B `rein account join`; device A `rein devices approve`; B pulls | **PASS** | Live pairing via `hoplab pair join` against the real `hopd`; `pairing_requests` row `status=consumed, claims=2, version=2`; device A pushed its 3 seeded sessions (claude/codex/opencode), device B pulled all 3 after a project mapping was added for device A's project; `sessions --json` on B showed 6 sessions total, no key overlap between A's and B's own three |
| `H6b` | An expired pairing request is refused/rolled back; B's wrap absent from every generation | **PASS** | Dedicated short-TTL lab (`HOPD_PAIRING_TTL=8s`); `account join` with no approver ever running blocked for the full TTL and was then refused client-side, exit 4, with the documented message; `pairing_requests` row `status=expired, payload='', claims=5, version=2`; `account status` on B showed `device_in_keyring=false, enrolled_devices=1` |
| `H7` | `daemon install/status/stop/start/uninstall` round trip through Task Scheduler; foreground loop pushes after a change (debounced) and pulls on schedule/before a resume | **PARTIAL** — operator/harness availability | See "`H7`" below |
| `H8` | A revokes B: generation rolls, B's token refused, B cannot open a later push; lagging-device attack; 404-floor proxy | **PASS** | `keygeneration_crossplane_test.go -tags hopacceptance` against a real `hopd` (`REINSTATE_HOPD_BIN`): `TestKeyGenerationFloorAgainstRealHopd` PASS (2.01s), `TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd` PASS (2.15s, 404-floor residual reproduced: "the floor route was asked for 4 times and answered 404 every time"). Live: device A revoked device B via the recovery code (`rein devices revoke`), generation rolled 1→2; device B's `whoami`, `push --all`, `pull --all` each refused, exit 4, with the documented message |
| `H8b` | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | **PASS** | A local self-signed-TLS reverse proxy in front of the real `hopd` drove the real Console HTTP routes end to end (email magic-link sign-in, session cookie, CSRF-bearing `/console/devices` page, `POST .../revocation`); `device_revocation_requests` row went `pending` (target device B confirmed still functional via `whoami`, exit 0) then `confirmed, confirmed_generation=2, confirmed_by=<device-A-id>` after `rein devices revoke` printed "Console request ... confirmed after the keyring reached generation 2"; target's next `push --all` refused, exit 4 |

## Evidence

All commands below are the literal argv run (no shell wrapping beyond
argument construction); output lines are the observed lines, trimmed only of
repeated `SDK ... WARN Response has no supported checksum` lines (a benign,
unrelated AWS-SDK log line from the storage client, present on every call
against `fakelocker`, not evidence of anything this report tests). No session
content, private path, or session id not created by this part appears below.
No account key, device key, or keyring/recovery-code value appears below,
per the evidence policy — every place one was used is described by
mechanism only (`REINSTATE_RECOVERY_CODE_FD`, a small non-product Go helper
this part wrote reusing `scripts/testing/hoplab/secretfd_windows.go`'s own
inheritable-handle mechanism to feed it to `rein devices revoke` from a
native Windows process, since a Git Bash `exec N<file` descriptor does not
translate into an inheritable Windows `HANDLE` a native `rein.exe` child can
read).

### `H6`

```
rein login --email <lab-email>                     # device-a, then device-b (same email)
hoplab approve -root <lab> -email <lab-email> -count 2 -timeout 2m
hoplab pair init -root <lab> -device device-a -rein rein.exe
hoplab pair join -root <lab> -device device-b -approver device-a -rein rein.exe
rein push --all --json                              # device-a: 3 sessions (claude, codex, opencode)
rein pull --all --json                               # device-b, after adding a project mapping for device-a's project
rein sessions --json                                 # device-b
```

Observed:

- `hoplab pair join` → `hoplab: device-b joined the account live, approved by device-a` (exit 0).
- `sqlite3 hopd.db "select id,status,claims,version from pairing_requests;"` → one row, `status=consumed, claims=2, version=2`.
- `rein push --all --json` on device-a → `"snapshots"` names all 3 sessions pushed (`claude`, `codex`, `opencode`); the push's own post-push `sync verify` step 4 reported `FAIL` — this is the pre-existing, already-carried `fakelocker AnyBucket:true` limitation (step 4, bucket-isolation, cannot genuinely refuse a "gone" bucket on this harness; documented in the `v0.6.0-rc.5`/`rc.6` reports for `H9`/`H10` and unrelated to `H6`'s own mechanism, which is pairing + pull, not sync-verify step 4).
- `rein pull --all --json` on device-b → after a `[[projects]]` mapping for `hoplab-device-a` was added to device-b's `config.toml` (the first attempt refused: `claude snapshot project "hoplab-device-a" has no local mapping on this device`): `"pulled": 3, "skipped": 0`, all 3 destinations under device-b's own isolated home.
- `rein sessions --json` on device-b → 6 sessions total: the 3 pulled from device-a (`project: "hoplab-device-a"`) plus device-b's own 3 (`project: "device-b"`/`"demo"`); no session-id overlap between the two sets.

### `H6b`

```
HOPD_PAIRING_TTL=8s hoplab start -root <lab-h6b> -hopd-addr 127.0.0.1:8323 -locker-addr 127.0.0.1:9323 -background
rein login --email <lab-email>                       # device-a, device-b
hoplab pair init -root <lab-h6b> -device device-a -rein rein.exe
rein init --hop --project hoplab-device-b=<path>      # device-b
rein account join                                     # device-b, no approver ever running
rein account status --json                            # device-b, after expiry
```

Observed:

- `rein account join` on device-b, with the pairing code printed and no approver ever watching that lab's log, blocked for the full TTL window (8s, matching `HOPD_PAIRING_TTL=8s`) and then printed: `the pairing request expired before it was approved and collected; run rein account join again on the new device`, exit `4`.
- `sqlite3 hopd.db "select id,status,payload,claims,version from pairing_requests order by rowid desc limit 1;"` → `status=expired, payload='' (empty), claims=5, version=2`.
- `rein account status --json` on device-b → `"device_in_keyring": false, "enrolled_devices": 1` (only device-a; device-b's request never produced a wrap in any generation).

### `H7`

**Fresh Hop lab account (zero pushed sessions).** A brand-new Hop account
was created through this part's own `hopd` at `127.0.0.1:8325`
(`rein login --email`, `hoplab pair init` → `rein init --hop` + `rein
account init`), under a fresh `REINSTATE_HOME` this row created and never
pushed anything from (`D:\ReinstateAcceptanceProjects\v060-rc7-e\lab-h7\device-a\reinstate`)
— trivially "zero sessions pushed to it," per the refined rule's own
definition.

**Per-file before-listing.** Every file (relative path, size, mtime) under
exactly the three session-bearing subtrees was listed, excluding `*-wal`,
`*-shm`, `*.lock`: `<CLAUDE_CONFIG_DIR>/projects`, `<CODEX_HOME>/sessions`,
`<XDG_DATA_HOME>/opencode` — reading `CODEX_HOME` as the **persistent,
user-level** value (`D:\Projects\hop-10-lab\codex`), not this shell's own
transient value (`...\orca\codex-runtime-home\home`, confirmed different).
Before-listing: **3,062 files**, digest (SHA-256 of the sorted
`relative_path<TAB>size<TAB>mtime` listing, one root path prefix per line)
`1ba422b730e74c2964085401169dd1555cddf7a6122db7d5c14c012cbd4d777a`.

**Go-signal wait — not satisfied.** `H7` was ordered last, after `H6`,
`H6b`, `H8`, `H8b` were already done, per the row's own ordering rule.
`D:/ReinstateAcceptanceProjects/h7-go.txt` did not exist when this part
started, and was polled every 20–60 seconds, starting only after the other
four rows finished (2026-09-08T16:49:49Z), for a total of **2,408 seconds
(~40.1 minutes)** across five consecutive poll windows (541s, 542s, 542s,
542s, 241s), ending 2026-09-08T17:24:17Z. The file never appeared. This
40-minute window is an operational constraint this run's own orchestrator
set — shorter than the contract's own 150-minute figure — a scheduling
detail of this run, not a reinterpretation of the contract; the same
shorter-window disposition the `v0.6.0-rc.6` `H7` evidence anticipated as a
possible outcome of this run's own orchestrator-set budget.

**Elevated round trip — not attempted.** Because the go-signal file never
appeared inside the wait window, the elevated `Start-Process -Verb RunAs`
script prepared for this row (`rein daemon install` → `status` → a 100s
debounce/scheduled-pull wait → `status` → `stop` → `start` → `status` →
`stop` → `uninstall` → `status` → `hop status`, all writing to a log file,
not the console) was never launched — attempting the elevation without the
maintainer present to accept the UAC prompt would be the exact unattended
workaround the dispatch's "Harness traps" section forbids ("Do not attempt
an unattended elevation workaround ... to route around the missing
operator").

**Cleanup.** The user-level `REINSTATE_HOME`/`REINSTATE_HOP_URL`
environment variables staged for the (unattempted) elevated shell to pick
up under the ambient login environment were removed again after the wait
window closed, so no stray state was left on the host.

**Disposition: `PARTIAL`** (operator/harness availability) — the row's
required mechanism (a real UAC-gated `schtasks` round trip) needs the
maintainer at the keyboard, which this run's own budget did not produce
within its window; this is not a product defect, and matches the dispatch's
own explicitly anticipated outcome for a wait window shorter than 150
minutes ("If the file never appears within that full wait window, record
`PARTIAL` (operator/harness availability) with that reason").

### `H8`

```
GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0 go test -tags hopacceptance ./internal/cli \
  -run TestKeyGeneration -count=1 -v          # REINSTATE_HOPD_BIN=<this part's own hopd.exe>
rein devices revoke <device-b-id>              # device-a, recovery code via REINSTATE_RECOVERY_CODE_FD
rein whoami / push --all / pull --all          # device-b, after revocation
```

Observed:

```
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (2.01s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (2.15s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	4.280s
```

- `rein devices revoke <device-b-id>` on device-a (recovery code fed through
  `REINSTATE_RECOVERY_CODE_FD`, the product's own documented automation
  path) → `revoked device "Harjots-Beast" (...); key generation 2 started
  with 1 enrolled device(s), and the control plane refuses its token`, exit
  0.
- Device-b afterward: `rein whoami` → `this device's token was rejected by
  the control plane (revoked or stale); run rein login again`, exit 4.
  `rein push --all` → same message, exit 4. `rein pull --all` → same
  message, exit 4.

### `H8b`

```
consoledriver -hopd http://127.0.0.1:8324 -proxy-addr 127.0.0.1:8443 -email <lab-email> \
  -hopd-log <lab-h8b>/hopd.log -device-id <device-b-id>
rein devices                                    # device-a, before confirm
rein devices revoke <device-b-id>               # device-a, recovery code via REINSTATE_RECOVERY_CODE_FD
rein whoami / push --all                        # device-b
```

`consoledriver` is a disposable acceptance-lab helper (not part of the
product) this part wrote fresh for this run, that drives the real Console
HTTP routes as a signed-in browser would, through a local self-signed-TLS
reverse proxy in front of the real `hopd` (`hopd`'s Console session cookie
is `Secure`, so it needs a real TLS context to be exercised faithfully
rather than bypassed).

Observed:

```
step1: POST /console/sign-in/email -> 200
step2: found link https://127.0.0.1:8443/console/sign-in/email/...
step3: GET confirm page -> 200
step4: POST confirm -> 200 final url https://127.0.0.1:8443/console
step5: GET /console/devices -> 200
step5: csrf token found, target device present=true
step6: POST https://127.0.0.1:8443/console/devices/<device-b-id>/revocation -> 200 final url https://127.0.0.1:8443/console/devices
step7: GET /console/devices -> 200
step7: device row now shows Pending revocation
```

- `sqlite3 hopd.db "select id,status,target_device_id,requested_generation from device_revocation_requests;"` (after step 6) → `status=pending, target=<device-b-id>, requested_generation=0`.
- `rein whoami` on device-b (after the Console request, before confirmation) → succeeds normally, exit 0 — the revocation stays pending, not yet enforced.
- `rein devices` on device-a → `pending revocation: Harjots-Beast (...), Console request <id>; run rein devices revoke <device-b-id> on another enrolled device`.
- `rein devices revoke <device-b-id>` on device-a (recovery code via `REINSTATE_RECOVERY_CODE_FD`) → `Console request <id> confirmed after the keyring reached generation 2`, then the same `revoked device ...` lines as `H8`, exit 0.
- `sqlite3 hopd.db "select id,status,confirmed_generation,confirmed_by from device_revocation_requests;"` (after confirm) → `status=confirmed, confirmed_generation=2, confirmed_by=<device-a-id>`.
- `rein push --all` on device-b (after confirmation) → `this device's token was rejected by the control plane (revoked or stale); run rein login again`, exit 4.

## Non-blocking findings

| ID | Row(s) | Description | Blocking |
| -- | ------ | ------------ | -------- |
| — | `H6` | The post-push `sync verify` step 4 (bucket isolation) reported `FAIL` on this part's `fakelocker` instance, matching the already-carried, non-blocking `AnyBucket:true` limitation the `v0.6.0-rc.5`/`rc.6` reports recorded for `H9`/`H10` — not `H6`'s own mechanism (pairing + pull), and not new to this candidate. | No |
| — | `H7` | The row could not be scored `PASS`/`FAIL` on its mechanism this run because the maintainer was not available inside the run's own 40-minute go-signal window; the dispatch's own contract anticipates this exact outcome for a wait window shorter than its 150-minute figure and calls for `PARTIAL` (operator/harness availability), not a fresh `#424`-style finding — nothing about the daemon's own mechanism was exercised or found wrong this run, so this is not evidence of a regression from `v0.6.0-rc.6`'s own `PASS` on this row. | Yes — `PARTIAL` does not pass a required row (device verdict impact only; not a product defect) |

## Cleanup

All four disposable lab pairs this part started (`lab`, `lab-h6b`,
`lab-h8b`, `lab-h7`, ports `8322`–`8325`/`9322`–`9325`) were stopped via
`hoplab stop -root <root>` after evidence collection, and the isolated
device-home directories under `D:\ReinstateAcceptanceProjects\v060-rc7-e\`
are scheduled for deletion per the evidence policy; nothing from them is
committed. The user-level `REINSTATE_HOME`/`REINSTATE_HOP_URL` environment
variables staged for `H7`'s (unattempted) elevated shell were removed.
