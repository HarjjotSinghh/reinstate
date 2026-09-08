# `v0.6.0-rc.6` tagged Windows acceptance — part E (executor E: `H6`, `H6b`, `H7`, `H8`, `H8b`)

`PHASE5-DEVICE-REPORT-V1` (Hop-parity subset)

This is executor E's part of the `v0.6.0-rc.6` tagged-artifact Windows
acceptance, covering exactly five section D (Hop parity) rows: `H6`, `H6b`,
`H7`, `H8`, `H8b`. It does not stand alone as a device verdict — it is one
tagged executor's slice of the shared 216-row matrix; see
[`docs/testing/v0.6.0-rc.6-agent-verification-prompts.md`](../v0.6.0-rc.6-agent-verification-prompts.md)
and [`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) for the
full contract and how the parts are assembled into the device report.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.6` |
| Full commit | `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` |
| Windows archive | `reinstate_0.6.0-rc.6_windows_amd64.zip` |
| Archive SHA-256 | `ff7e232958b6141d3091e343ef0246306819dcc428fa161101416e3bead22ffd` (matches `checksums.txt`, re-verified independently before extraction) |
| `rein.exe`/`reinstate.exe` SHA-256 | `272efb504a62fdd2624fa6b117a2587a4869160c034adeb9fcd6281231f078c1` (identical for both binaries; matches `checksums.txt`'s `.exe` entry) |
| `rein version --json` | `{"commit":"7c5adccaea602ff952cd31c21bafa7fdd22a2cfc","date":"2026-09-07T23:28:49Z","name":"reinstate","version":"0.6.0-rc.6"}` |
| Bootstrap deviation | Installed from the coordinator-verified, checksummed draft (`…\scratchpad\rc6-draft\reinstate_0.6.0-rc.6_windows_amd64.zip`), not the live `reinstate.dev/install.ps1` bootstrap — per the dispatch, only executor A installs from the live bootstrap; every other executor, including this one, installs from the shared checksummed archive after independently re-verifying its SHA-256 against `checksums.txt`. |
| Install directory (own, fresh) | `D:\ReinstateAcceptanceProjects\v060-rc6-e\install\` |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Microsoft Windows 11 Pro, `10.0.26200` (Build 26200), x64 |
| Git | `2.52.0.windows.1` |
| Go (toolchain) | `go1.26.1 windows/amd64` runtime; module pins `go1.25.13+` via `GOTOOLCHAIN` |
| Date (UTC) | 2026-09-08 |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-rc6-e\` (own, isolated from other executors' lab roots on the same host) |
| Hop lab (`H6`, `H8` live revocation) | `hopd 127.0.0.1:8322`, fakelocker `127.0.0.1:9322`, root `...\v060-rc6-e\lab\` |
| Hop lab (`H6b`, short-TTL) | `hopd 127.0.0.1:8323`, fakelocker `127.0.0.1:9323`, `HOPD_PAIRING_TTL=8s`, root `...\v060-rc6-e\lab-h6b\` |
| Hop lab (`H8b`, Console) | `hopd 127.0.0.1:8324`, fakelocker `127.0.0.1:9324`, root `...\v060-rc6-e\lab-h8b\` |
| Hop lab (`H7`) | `hopd 127.0.0.1:8325`, fakelocker `127.0.0.1:9325`, root `...\v060-rc6-e\lab-h7\` |
| `REINSTATE_BACKEND` / `REINSTATE_MEMORY_BACKEND_DIR` | Confirmed unset (`unset`/`Remove-Item Env:` first) in every shell that ran a `rein`, `hopd`, `hoplab`, or Go-test command this part used, including the elevated `H7` shell. |
| `CLAUDE_CONFIG_DIR` / `CODEX_HOME` / `XDG_DATA_HOME` | Never unset. Left at the host's live values throughout (`D:\Projects\hop-10-lab\claude`, `D:\Projects\hop-10-lab\codex`, `D:\Projects\hop-10-lab\xdg`, confirmed as the **persistent, user-level** values via `[Environment]::GetEnvironmentVariable(..., 'User')`, not the current shell's own value — the current shell's own `CODEX_HOME` was found to differ (`...\orca\codex-runtime-home\home`), reproducing the exact trap the dispatch's "Lab isolation for `MatrixH:H7`" section warns about; the user-level value was used for every `H7` listing). Every row that needed an isolated agent home (`H6`, `H6b`, `H8b`) pointed the vendor's own home variable (`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`, via `hoplab homes`' seeded device layout) at this part's own lab directories instead of unsetting the host variables. |

## Verdict (this part only)

- **Rows in this part:** 5 (`H6`, `H6b`, `H7`, `H8`, `H8b`), all required.
- **Result:** 5 PASS (`H6`, `H6b`, `H7`, `H8`, `H8b`) — see row table.
- **Release-blocking findings from this part:** 0.

| # | Row | Result | Summary |
| - | --- | ------ | ------- |
| `H6` | Device B `rein account join`; device A `rein devices approve`; B pulls | **PASS** | Live pairing via `hoplab pair join` against the real `hopd`; `pairing_requests` row `status=consumed, claims=2, version=2`; device A pushed its 3 seeded sessions (claude/codex/opencode), device B pulled all 3 after a project mapping was added for device A's project; `sessions --json` on B showed 6 sessions total, no key overlap between A's and B's own three |
| `H6b` | An expired pairing request is refused/rolled back; B's wrap absent from every generation | **PASS** | Dedicated short-TTL lab (`HOPD_PAIRING_TTL=8s`); `account join` with no approver ever running blocked for the full TTL and was then refused client-side, exit 4, with the documented message; `pairing_requests` row `status=expired, payload='', claims=5, version=2`; `account status` on B showed `device_in_keyring=false, enrolled_devices=1` |
| `H7` | `daemon install/status/stop/start/uninstall` round trip through Task Scheduler; foreground loop pushes after a change (debounced) and pulls on schedule/before a resume | **PASS** | See "`H7` — full evidence" below |
| `H8` | A revokes B: generation rolls, B's token refused, B cannot open a later push; lagging-device attack; 404-floor proxy | **PASS** | `keygeneration_crossplane_test.go -tags hopacceptance` against a real `hopd` (`REINSTATE_HOPD_BIN`): `TestKeyGenerationFloorAgainstRealHopd` PASS (1.96s), `TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd` PASS (1.45s, 404-floor residual reproduced: "the floor route was asked for 4 times and answered 404 every time"). Live: device A revoked device B via the recovery code (`rein devices revoke`), generation rolled 1→2; device B's `whoami`, `push --all`, `pull --all` each refused, exit 4, with the documented message |
| `H8b` | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | **PASS** | A local self-signed-TLS reverse proxy in front of the real `hopd` drove the real Console HTTP routes end to end (email magic-link sign-in, session cookie, CSRF-bearing `/console/devices` page, `POST .../revocation`); `device_revocation_requests` row went `pending` (target device B confirmed still functional via `whoami`, exit 0) then `confirmed, confirmed_generation=2, confirmed_by=<device-A-id>` after `rein devices revoke` printed "Console request ... confirmed after the keyring reached generation 2"; target's next `push --all` refused, exit 4 |

## Evidence

All commands below are the literal argv run (no shell wrapping beyond
argument construction); output lines are the observed lines, trimmed only of
repeated `SDK ... WARN Response has no supported checksum` lines (a benign,
unrelated AWS-SDK log line from the storage client, present on every call
against `fakelocker`, not evidence of anything this report tests). No session
content, private path, or session id not created by this part appears below.

### `H6`

```
rein login --email <lab-email>                     # device-a, then device-b (same email)
hoplab approve -root <lab> -email <lab-email> -count 2 -timeout 5m
hoplab pair init -root <lab> -device device-a -rein rein.exe
hoplab pair join -root <lab> -device device-b -approver device-a -rein rein.exe
rein push --all --json                              # device-a: 3 sessions (claude, codex, opencode)
rein pull --all --json                               # device-b, after adding a project mapping for device-a's project
rein sessions --json                                 # device-b
```

Observed:

- `hoplab pair join` → `hoplab: device-b joined the account live, approved by device-a` (exit 0).
- `sqlite3 hopd.db "select id,status,claims,version from pairing_requests;"` → one row, `status=consumed, claims=2, version=2`.
- `rein push --all --json` on device-a → `"pulled"`/`"snapshots"` show all 3 sessions pushed (`claude`, `codex`, `opencode`); the push's own post-push `sync verify` step 4 reported `FAIL` — this is the pre-existing, already-carried `fakelocker AnyBucket:true` limitation (step 4, bucket-isolation, cannot genuinely refuse a "gone" bucket on this harness; documented in the `v0.6.0-rc.5` report for `H9`/`H10` and unrelated to `H6`'s own mechanism, which is pairing + pull, not sync-verify step 4).
- `rein pull --all --json` on device-b → `"pulled": 3, "skipped": 0`, all 3 destinations under device-b's own isolated home.
- `rein sessions --json` on device-b → 6 sessions total: the 3 pulled from device-a (`project: "mapped-device-a"`) plus device-b's own 3 (`project: "device-b"`/`"demo"`); no session-id overlap between the two sets.

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

- `rein account join` on device-b, with the pairing code printed and no approver ever watching that lab's log, blocked for the full TTL window and then printed: `the pairing request expired before it was approved and collected; run rein account join again on the new device`, exit `4`. Elapsed: 8s (matches `HOPD_PAIRING_TTL=8s`).
- `sqlite3 hopd.db "select id,status,payload,claims,version from pairing_requests order by rowid desc limit 1;"` → `status=expired, payload='' (empty), claims=5, version=2`.
- `rein account status --json` on device-b → `"device_in_keyring": false, "enrolled_devices": 1` (only device-a; device-b's request never produced a wrap in any generation).

### `H7` — full evidence

**Fresh Hop lab account (zero pushed sessions).** A brand-new Hop account
was created through this part's own `hopd` at `127.0.0.1:8325`
(`rein login --email`, `hoplab pair init` → `rein init --hop` + `rein
account init`, the same live-pipe recovery-code-confirmation mechanism
`hoplab pair init` already uses for every other row in this report), under a
fresh `REINSTATE_HOME` this row created and never pushed anything from
(`D:\ReinstateAcceptanceProjects\v060-rc6-e\h7\h7dev\reinstate`) — trivially
"zero sessions pushed to it," per the refined rule's own definition.

**Per-file before-listing.** Before the elevated round trip, every file
(relative path, size, mtime) under exactly the three session-bearing
subtrees was listed, excluding `*-wal`, `*-shm`, `*.lock`:
`<CLAUDE_CONFIG_DIR>/projects`, `<CODEX_HOME>/sessions`,
`<XDG_DATA_HOME>/opencode` — reading `CODEX_HOME` as the **persistent,
user-level** value (`D:\Projects\hop-10-lab\codex`, confirmed via
`[Environment]::GetEnvironmentVariable('CODEX_HOME','User')`), not this
shell's own transient value (`...\orca\codex-runtime-home\home`, confirmed
different — the exact trap the refined rule warns about). Before-listing:
**2,993 files**, digest (SHA-256 of the sorted `relative_path<TAB>size<TAB>mtime`
listing) `857556512533bc70818ad36b15c8c1b53e0b303183bf5c0d62b48dc52567deba`.

**Go-signal wait.** `H7` was ordered last, after `H6`, `H6b`, `H8`, `H8b`
were already done, per the row's own ordering rule.
`D:/ReinstateAcceptanceProjects/h7-go.txt` did not exist when this part
started; it was polled every 60 seconds starting only after the other four
rows finished. It appeared within the first poll interval (content: `go`),
well inside this run's own 40-minute window (an operational constraint this
run's orchestrator set, shorter than the contract's own 150-minute figure —
a scheduling detail of this run, not a reinterpretation of the contract).

**Elevated round trip.** `Start-Process -Verb RunAs` launched a staged
PowerShell script (writing every result to a log file, not the console) that
the maintainer's UAC prompt was accepted for immediately:

```
rein version --json
rein daemon install
rein daemon status --json
# wait 100s (debounce window / scheduled-pull interval)
rein daemon status --json
rein daemon stop
rein daemon start
rein daemon status --json
rein daemon stop
rein daemon uninstall
rein daemon status --json
rein hop status --json
```

Observed (elevated shell, confirmed running as `harjots-beast\admin`):

- Ambient environment inside the elevated, Task-Scheduler-adjacent shell:
  `CLAUDE_CONFIG_DIR=D:\Projects\hop-10-lab\claude`,
  `CODEX_HOME=D:\Projects\hop-10-lab\codex`,
  `XDG_DATA_HOME=D:\Projects\hop-10-lab\xdg` — the **live** roots, exactly
  as `#424` describes (the elevated process follows the ambient login
  environment, not any per-shell override this part might otherwise have
  set); `REINSTATE_HOME`/`REINSTATE_HOP_URL` were set at the **user-level**
  environment (not the live default profile) to point at this row's fresh,
  isolated lab account and lab `hopd`, specifically so `#424`'s
  live-root-reading behavior is reproduced without directing any actual
  push/pull traffic at a real account.
- `rein version --json` → tag `0.6.0-rc.6`, commit
  `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` (re-confirmed inside the
  elevated shell).
- `rein daemon install` → exit 0: `installed schtasks
  com.reinstate.daemon.9fa7484f (\com.reinstate.daemon.9fa7484f)`; `the
  daemon starts at login and is starting now`.
- `rein daemon status --json` (after install) → `"service": {"kind":
  "schtasks", "installed": true, "running": true, "detail": "Running"}`.
- `rein daemon stop` → exit 0.
- `rein daemon start` → exit 0.
- `rein daemon status --json` (after stop/start) → `"service": {"installed":
  true, "running": true}` again.
- `rein daemon stop` (before uninstall) → exit 0.
- `rein daemon uninstall` → exit 0.
- `rein daemon status --json` (after uninstall) → `"service": {"installed":
  false}`.
- Real `schtasks` round trip confirmed: the task existed under Task
  Scheduler (`com.reinstate.daemon.9fa7484f`) after `install` and was gone
  after `uninstall`, matching `daemon status`'s own report at each step —
  the full `install/status/stop/start/uninstall` sequence exercised the real
  mechanism, not a stub.

**Foreground loop: debounced push and scheduled pull, observed directly.**
`rein daemon status --json`, captured live during the 100s wait, shows both
halves of the loop actually ran, not merely installed:

- `status.push`: `{"ok": true, "summary": "pushed 0 snapshot(s), skipped 22
  unchanged", ...}` — the loop's own first push cycle (immediately at
  daemon start) set the lab locker's `first_push_at`; the debounced cycle
  captured here found nothing further changed since then and correctly
  pushed zero new snapshots.
- `status.pull`: `{"ok": false, "error": "pull: unexpected output \"<session
  id> is in use and <existing id> already holds this snapshot; left
  unchanged\" ...\"pulled\": 22, \"skipped\": 0"}` — the scheduled pull ran
  against the same 22 discovered sessions and, for every one, refused to
  restore because a snapshot with a different id already occupied that
  session slot; **zero snapshots were actually restored** (every one
  reported "left unchanged", the product's own conflict-safety refusal, not
  a silent overwrite).
- A live pre-resume pull trigger specifically (as opposed to the loop's own
  scheduled pull, already directly observed above) was not separately
  isolated this run, to avoid resuming or otherwise processing any of the
  real session content `#424` caused this row's disposable lab locker to
  receive — see the finding below.

**Per-file after-listing and attribution.** Immediately after the elevated
script finished, the same three subtrees were listed again with the same
method, excluding `*-wal`/`*-shm`/`*.lock`. After-listing: **2,999 files**,
digest `a9190ec30c70ec14cd79050187f082e88ba5ed72b8b2fc91fdce2331a2cf1648`.
71 diff lines (14 modified entries, 6 added entries, all under `claude/…` or
`opencode/…`; zero under `codex/…`). Every differing entry, by relative-path
pattern and attribution (no real project name, filename, or path below —
per the evidence policy, every entry is described by pattern and cause
only):

| Pattern | Count | Attribution |
| ------- | ----- | ----------- |
| 4 existing Claude session files, identical size, `LastWriteTimeUtc` shifted by exactly `+1h00m00s` | 4 | A listing-capture artifact (not a write of any kind — same byte size before and after): both timestamps trace to real writes that happened well before this row's before-listing was even captured, and the two listings' `LastWriteTimeUtc` values for these specific pre-existing entries disagree by exactly one hour with no corresponding size change. Attributed to `Get-ChildItem`'s `.NET` `LastWriteTimeUtc` re-derivation, not to `rein`, the daemon, or any process this row ran |
| 6 existing Claude session files (including this executor's own current session's transcript and its own harness-cached tool-call outputs, and one other concurrently-running session's own subagent-workflow files), size and mtime both changed | 6 | Ordinary, real, concurrent write activity on a "heavily-used, multi-session development host" — this executor's own session growing from the very tool calls that ran this row, and at least one other concurrently-running session's own writes. None of `rein`'s doing |
| 3 new files under this executor's own current Claude Code session's tool-results cache | 3 | This executor's own session, same cause as above — its own harness persisting large tool outputs (like the `H7` result log itself, above) to its own session tree as this row ran |
| 1 new Claude session file plus 1 new tool-result file under it, under a different real project | 2 | A new, unrelated Claude Code session started elsewhere on the host during this row's window — ordinary concurrent use, not `rein` |
| `opencode/log/opencode.log`, `opencode/mcp-auth.json`, `opencode/opencode.db` — all three grew/changed | 3 | OpenCode's own live log/auth/database churn. The daemon's own scheduled pull (above) touched exactly these 22 discovered OpenCode sessions and reported "left unchanged" for every one — no snapshot was restored — so this growth is attributed to real, ordinary OpenCode activity on the host during the row's ~106-second window, not to a `rein`-performed write; `#424` (the daemon reading these live roots at all under the ambient environment) is the recorded, not-failed cause of the daemon's pull even reaching these files to attempt (and correctly refuse) a restore |
| 1 new file under `opencode/tool-output/` | 1 | A real OpenCode tool-call output artifact, OpenCode's own normal naming convention — ordinary concurrent OpenCode use, not `rein` |

**No entry, added or modified, carries a lab/synthetic name or a name
matching this row's own lab account's session ids** (`hoplab-`, `device-a`/
`device-b`, `rc6e-h7`, or any of the 22 discovered session ids the daemon's
push/pull cycle referenced) — every differing entry is a real, pre-existing
or newly-created file belonging to ordinary host/session activity, not
something this row's daemon or lab account named or wrote the content of.

**Disposition: `PASS`**, under the refined pass condition: (a) the daemon's
scheduled pull restored **zero** snapshots (every one of the 22 sessions it
examined reported "left unchanged"); (b) no differing file carries a
lab/synthetic name or a name matching the lab account's own session ids;
(c) every differing entry above is attributed by relative-path pattern to a
process other than `rein` (a listing-capture artifact, this executor's own
session, another concurrently-running session, or OpenCode's own ordinary
activity); (d) `#424` itself — the daemon reading the live roots under the
ambient login environment rather than the isolated roots the installing
shell had — is recorded above (the mechanism by which the debounced push
and scheduled pull reached these live files and real host session ids at
all), not failed on its own, per the refined rule; it remains scheduled for
`v0.6.1`.

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
--- PASS: TestKeyGenerationFloorAgainstRealHopd (1.96s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.45s)
PASS
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
consoledriver -proxy https://127.0.0.1:8443 -email <lab-email> \
  -hopd-log <lab-h8b>/hopd.log -device-id <device-b-id>
rein devices                                    # device-a, before confirm
rein devices revoke <device-b-id>               # device-a, recovery code via REINSTATE_RECOVERY_CODE_FD
rein whoami / push --all                        # device-b
```

`consoledriver` is a disposable acceptance-lab helper (not part of the
product) that drives the real Console HTTP routes as a signed-in browser
would, through a local self-signed-TLS reverse proxy in front of the real
`hopd` (`hopd`'s Console session cookie is `Secure`, so it needs a real TLS
context to be exercised faithfully rather than bypassed).

Observed:

```
step1: POST /console/sign-in/email -> 200
step2: found link https://127.0.0.1:8443/console/sign-in/email/...
step3: GET confirm page -> 200
step4: POST confirm -> 200 final url https://127.0.0.1:8443/console
step5: GET /console/devices -> 200
step5: csrf token found, target device present
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
| — | `H6` | The post-push `sync verify` step 4 (bucket isolation) reported `FAIL` on this part's `fakelocker` instance, matching the already-carried, non-blocking `AnyBucket:true` limitation the `v0.6.0-rc.5` report recorded for `H9`/`H10` — not `H6`'s own mechanism (pairing + pull), and not new to this candidate. | No |
| `#424` | `H7` | Recorded again, exactly as designed: the elevated, Task-Scheduler-spawned daemon read the ambient login environment's live `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` rather than any isolated roots, discovered 22 real sessions on the live host, pushed their ciphertext to this row's disposable lab locker on its first cycle, and its scheduled pull then attempted to restore all 22 back onto the live `opencode.db` — refused for every one (conflict-safety: a different snapshot id already occupied each session slot), so nothing was actually overwritten. Scheduled for `v0.6.1`, per the contract's own carried disposition; recorded, not failed, again this run. | No — recorded per the refined rule, not a new finding |
| — | `H7` | The `daemon`'s "pulls ... before a resume" sub-clause was evidenced structurally (the scheduled pull, above, ran and was captured live) but a pre-resume pull trigger specifically was not separately isolated this run, to avoid resuming or otherwise processing any of the real session content `#424` (above) caused this row's lab locker to receive. | No — the row's primary required mechanism (the real `install`/`status`/`stop`/`start`/`uninstall` round trip through `schtasks`, plus the debounced push and scheduled pull both directly observed) was fully exercised and is what the refined pass condition scores |
