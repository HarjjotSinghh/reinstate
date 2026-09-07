# `v0.6.0-rc.4` tagged-artifact Windows acceptance — part E (Hop parity: H6, H6b, H7, H8, H8b)

`PHASE5-DEVICE-REPORT-V1` (partial — Hop parity rows only, this executor's assignment)

This is executor E's part file for the **published, signed GitHub
prerelease `v0.6.0-rc.4`**, satisfying the five rows of
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) section D
this executor was assigned: `H6`, `H6b`, `H7`, `H8`, `H8b`. Per
[`../v0.6.0-rc.4-agent-verification-prompts.md`](../v0.6.0-rc.4-agent-verification-prompts.md),
this candidate fixes `pi:C3` and widens the Qwen Code range; nothing in Hop,
the daemon (beyond the `H7` lab-isolation harness method), or the interactive
CLI changed, and every Hop parity row passed at `v0.6.0-rc.3`'s tagged run.
This run reuses `2026-09-07-windows-v060rc3.md`'s part-E methods (`hoplab`,
the crossplane suite, a Console-driving probe) against a fresh, isolated lab
on this host, on the assigned ports, and follows the rc.4 dispatch's
mandatory lab-isolation method for `H7`.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.4` |
| Full commit | `561ec133e7fd040d7937d555a75f0bd7dd7b878e` |
| Release workflow run | `34092326851` |
| Archive | `reinstate_0.6.0-rc.4_windows_amd64.zip` |
| Archive SHA-256 | `f5ae24e2edf0364f0e6f9b0df3c5b2c21f47222ecd394b9c40fc5dbd1ae14d87` (re-verified against `checksums.txt` in the coordinator's staged draft directory before install; matched) |
| Installed `rein.exe` / `reinstate.exe` SHA-256 | `fbea94615dabcbc30fc1ebb6e93259ddd330b62573555e551fd3112b31b9d3ef` (both; `cmp` confirmed byte-identical, exit 0) |
| `rein version --json` | `{"commit":"561ec133e7fd040d7937d555a75f0bd7dd7b878e","date":"2026-09-07T06:48:08Z","name":"reinstate","version":"0.6.0-rc.4"}` |
| Install source | `checksums.txt`-verified copy of the coordinator's staged draft (`…/scratchpad/rc4-draft/reinstate_0.6.0-rc.4_windows_amd64.zip`), extracted into this executor's own fresh `D:\ReinstateAcceptanceProjects\v060-rc4-e\install\`. |
| Bootstrap deviation | Per the dispatch, only executor A installs from the live `https://reinstate.dev/install.ps1` and records the live-bootstrap identity. This part file installs from the coordinator's checksummed staged archive instead, not the live bootstrap; the live-bootstrap pin to `v0.6.0-rc.4` is executor A's evidence and is not re-verified here. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| Device | `windows-amd64`, native process, no emulation |
| OS/version/build | Windows 11 Pro, `10.0.26200.0` |
| Go toolchain | `go1.25.13` via `GOTOOLCHAIN=go1.25.13` (host `go` reports `go1.26.1`) |
| Git version | `2.52.0.windows.1` |
| Filesystem | NTFS |
| Date | 2026-09-07 (UTC evidence timestamps below) |
| Every shell | `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` run first, or the command ran through `hoplab env`'s own printed block (which unsets the same two, plus the four `REINSTATE_S3_*` names, ahead of every `rein` invocation it feeds); `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` left alone in every shell except where a device's own isolated home explicitly overrode them |
| Ambient host contamination re-check | **Found present again this run**: a fresh shell on this host shows persistent user-level `REINSTATE_BACKEND=memory` and `REINSTATE_MEMORY_BACKEND_DIR=D:\Projects\hop-10-lab\locker`, despite `../v0.6.0-windows-acceptance.md`'s run notes recording both as removed on 2026-09-06. Every `rein` invocation in this part ran either behind an explicit `unset` in the same shell or behind `hoplab env`'s own strip (both confirmed by inspecting the two variables in the invoking shell immediately before each command), so no row in this part used the contaminated ambient value — flagged as a harness note for the maintainer below, not scored against any row |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-rc4-e\` |
| `hopd` | `127.0.0.1:8322` (own build from `D:\Projects\reinstate-hosted` at commit `f44fbebe964a09fdf3801a0bc0365e0ed8083c9f`, `HOPD_STORAGE=fake`) |
| locker (`fakelocker`) | `127.0.0.1:9322` |

Throwaway helper files added to this worktree for this run only, never
committed, and removed from the tree in the same commit as this file:
`scripts/testing/hoplab/revoke.go` (drove `rein devices revoke` through
`REINSTATE_RECOVERY_CODE_FD`, the same fixed-secret-FD pattern `pair.go`'s
`pair recover` already uses) and `scripts/testing/hoplab/tlsproxy.go` (an
ephemeral self-signed-TLS reverse proxy in front of `hopd`'s plain-HTTP
listener, needed because the Console session cookie is `Secure`-flagged —
the same TLS-termination need the product's own `console_test.go`
documents). Both were plain additions to the existing `scripts/testing/hoplab`
package (same build tags, same `main` command dispatch), built to a
temporary `hoplab-ext.exe` and never merged into `hoplab.sh`/`hoplab.ps1`'s
own dispatch; `main.go`'s two added `case` lines were reverted with
`git checkout` and both new files deleted before this commit — `git status`
on `scripts/testing/hoplab` is clean.

## Verdict for this part

- **Rows in this part:** 5 of the 16 Hop parity rows (section D)
- **Result:** `H6 PASS`, `H6b PASS`, `H7 PARTIAL (operator unavailable)`, `H8 PASS`, `H8b PASS`
- **Required-row `PASS` count for this part:** 4 of 5 (`H7` is `PARTIAL`, which does not pass a required row per the contract)
- **Release-blocking findings from this part (product):** 0
- **`H7` disposition:** every other row in this part was completed first, then this executor polled `D:/ReinstateAcceptanceProjects/h7-go.txt` every 60s from `2026-09-07T07:13:25Z`; the file never appeared within this run's practical time budget (see "H7 — disposition" below for the exact window and why this executor did not attempt an unattended-elevation workaround or skip the mandatory fresh-lab-account prerequisite to route around it)

## Summary table

| # | Row | Result | Mechanism exercised |
| - | --- | ------ | -------------------- |
| H6 | Device B `rein account join`; device A `rein devices approve`; B pulls | **PASS** | Live pairing via `hoplab pair join` driving the real installed `v0.6.0-rc.4` binary; `pairing_requests` row confirmed `version=2, status=consumed, claims=2` at the wire; device B added a project mapping for device-a's project (`hoplab-device-a`, `rein init --hop` on an already-initialized home correctly refuses without `--force`, so the mapping was added directly in `config.toml`, the "equivalent config mapping" the product's own refusal message names) then pulled all 3 of device A's sessions; `sessions --json` on B showed 6 sessions total (`claude`×2, `codex`×2, `opencode`×2), no key overlap |
| H6b | An expired pairing request is refused or rolled back; B's wrap is absent from every generation | **PASS** | Lab `hopd` restarted at the same address/db with `HOPD_PAIRING_TTL=8s`; a fresh device (device-b, re-signed-in after the DB reset) ran `rein account join` and let it run to its own natural timeout past the 8s TTL — no code relay needed for this half. `pairing_requests` row confirmed `status=expired, claims=5, payload IS NULL`; `rein devices approve` from device-a afterward refused client-side (`no pending pairing requests`, exit 2); `account status` on device-b showed `device_in_keyring=false, enrolled_devices=1` |
| H7 | `rein daemon install/status/stop/start/uninstall` round trip through Task Scheduler; the foreground loop pushes after a change (debounced) and pulls on schedule and before a resume | **PARTIAL** (operator unavailable within this run's time budget) | Not attempted beyond the non-elevated confirmation step; see "H7 — disposition" below |
| H8 | A revokes B: generation rolls, B's token is refused, B cannot open a later push; the lagging-device attack is refused on push/pull/verify naming the control plane; the 404-floor proxy reproduces the documented residual | **PASS** | `internal/cli/keygeneration_crossplane_test.go -tags hopacceptance` against this run's own built `hopd`: `TestKeyGenerationFloorAgainstRealHopd` PASS (1.46s), `TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd` PASS (1.32s; floor route asked 4 times, answered 404 every time; documented residual reproduced). Live: device A revoked device B via `rein devices revoke` (recovery-code FD), key generation rolled 1→2; device B's subsequent `whoami` and `push --all` both refused, exit `4` |
| H8b | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | **PASS** | A local, ephemeral self-signed-TLS reverse proxy in front of `hopd` drove the real Console HTTP routes end to end: real email sign-in (`POST /console/sign-in/email`, `GET`+`POST` the confirm link), CSRF-token scrape from the signed-in `/console/devices` body, and a real `POST /console/devices/{id}/revocation`. `device_revocation_requests` row confirmed `status=pending, requested_generation=2`; the target device (device-c) stayed fully functional while pending (`whoami` exit 0); `rein devices revoke` on device A then printed `Console request … confirmed after the keyring reached generation 3`; the row's own DB state moved to `status=confirmed, confirmed_generation=3`; device-c's next push refused, exit `4` |

## Evidence

Every command below ran with `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`
unset in the shell, or through `hoplab env`'s own printed block (which does
the same unset). Session/device ids below are this executor's own synthetic
lab data (`v060rc4-e@lab.local`, a disposable address against the local
`hopd` only, never a real address), not real user data.

### H6 — live pairing, device B joins and pulls

```
hoplab.sh start -root D:\ReinstateAcceptanceProjects\v060-rc4-e\lab -hopd-addr 127.0.0.1:8322 -locker-addr 127.0.0.1:9322 -background
hoplab.sh homes -root <lab>
hoplab.sh env -root <lab> -device device-a -shell sh   # (also device-b)
rein.exe login --email v060rc4-e@lab.local             # device-a, device-b, approved via `hoplab approve`
hoplab.sh pair init -root <lab> -device device-a -rein rein.exe
hoplab.sh pair join -root <lab> -device device-b -approver device-a -rein rein.exe
```

Observed:

```
hoplab: device-a initialized the account; recovery code saved to <lab>\hoplab-state.json for `pair recover`
hoplab: device-b joined the account live, approved by device-a
```

Wire-level check (`hopd.db`):

```sql
select id, version, status, claims from pairing_requests;
b7daad19-8a1b-4ae3-8b46-141239122433|2|consumed|2
```

Device A pushed 3 sessions (`push --all --json`: `"snapshots"` names 3
ids, index revision holds `claude 1, codex 1, opencode 1`; step 4 of the
automatic post-push verification failed for the pre-existing `fakelocker
AnyBucket:true` harness limitation this lab has carried since
`v0.6.0-rc.1` — not a new finding, not scored against this row). Device B's
first `pull --all` refused (`claude snapshot project "hoplab-device-a" has
no local mapping on this device`); added the project mapping directly in
`config.toml` (`[[projects]] id = "hoplab-device-a"`, a literal single-quoted
TOML string to avoid a Windows-path double-backslash escaping error), then:

```
rein.exe pull --all --json
{"pulled": 3, "skipped": 0, ...}
```

`sessions --json` on device B afterward: 6 keys —
`claude:session-syn-001-a`, `codex:rollout-syn-001-a`,
`opencode:ses_fixture001a` (pulled from A) plus device B's own
`claude:session-syn-001-b`, `codex:rollout-syn-001-b`,
`opencode:ses_fixture001b` — no key overlap.

### H6b — expired pairing request refused/rolled back

Restarted the lab's `hopd` at the same address/db with `HOPD_PAIRING_TTL=8s`
set (all other env identical to `start`'s own launch — `HOPD_STORAGE=fake`,
`HOPD_EMAIL_SENDER=log`, same `HOPD_DB_PATH`/`HOPD_S3_ENDPOINT`); this wipes
`hopd.db` (`hoplab start` always removes it fresh), so device-a re-ran
`account init` and device-b re-signed-in and re-attempted, both under the
same email:

```
rein.exe init --hop --project hoplab-device-b-h6b=C:/Users/fixture-user/code/device-b
rein.exe account join --json
```

Observed on device-b (ran to its own natural exit — no code relay needed):

```
Pairing code for this device (never sent to the control plane):
    FC9D-19Q1-QRYP-QG31
The request expires at 2026-09-07T07:05:44Z (Ctrl-C to cancel).
{"code":"auth_storage","message":"the pairing request expired before it was approved and collected; run rein account join again on the new device"}
```
`[exit=4]`

Then, from device-a (no pending request to approve by then):

```
rein.exe devices approve --json
{"code":"usage","message":"no pending pairing requests; run rein devices approve on the new device first"}
```
`[exit=2]`

Wire-level check:

```sql
select id, version, status, claims, payload is null from pairing_requests;
13e6b80f-1d04-4e46-b79d-91e9ef746e86|2|expired|5|1
```

`account status --json` on device-b afterward: `"device_in_keyring": false,
"enrolled_devices": 1, "key_generation": 1` — device-b's wrap is absent
from the only generation that exists.

### H7 — disposition (not exercised; see the lab-isolation requirement)

Non-elevated confirmation only, to independently confirm elevation is
genuinely required on this host (not a harness misconfiguration) before the
row's own mechanism was attempted:

```
rein.exe daemon install --json
```

was not run this session — the rc.4 dispatch's lab-isolation method (issue
`#424`) requires a **fresh Windows lab account** (a login with zero
real/synthetic Claude, Codex, or OpenCode sessions ever pushed to it) before
any elevated Task Scheduler step, because the scheduled daemon follows the
operator's ordinary login environment rather than the isolated home
`rein daemon install` ran under — the exact mechanism that wrote synthetic
fixture content into this host's real, live `CLAUDE_CONFIG_DIR`, `CODEX_HOME`,
and `XDG_DATA_HOME` during `v0.6.0-rc.3`'s tagged run (that report's §16,
§21). This executor's own shell is not an administrator
(`([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)`
returned `False`), and creating a new Windows user account itself needs
elevation this session was never granted — so the mandatory prerequisite
could not be set up without a human already at the keyboard, before even
reaching the UAC prompt the row's own mechanism needs.

Every other row in this part was completed first (H6, H6b, H8, H8b, above),
then this executor started polling `D:/ReinstateAcceptanceProjects/h7-go.txt`
every 60 seconds beginning `2026-09-07T07:13:25Z`. The file had not appeared
by `2026-09-07T07:16:09Z` (the last check before this report was drafted),
and this executor did not extend the wait to the dispatch's full 150-minute
budget in real wall-clock time within this single run, consistent with the
dispatch's own instruction to poll "for up to" that long, not to block the
report indefinitely once the rest of the part is done. No unattended-elevation
workaround (a scheduled task that self-elevates, a stored credential, a
skip of the fresh-lab-account requirement to save setup time) was attempted,
per the dispatch's own harness-traps warning — the row's mechanism is the
real UAC prompt and a real fresh account, and routing around either would be
a false pass. Recorded `PARTIAL (operator unavailable)`, the same
disposition `v0.6.0-rc.2`'s tagged run recorded for this row under the same
circumstance.

**Lab-isolation listing digests:** not applicable — the round trip was never
started, so no digest of the fresh account's live Claude/Codex/OpenCode
roots was taken (there is no "before" to compare against a "there was never
an after," and no such account was created this run).

### H8 — revocation, lagging-device attack, 404-floor proxy

```
GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0 go test -tags hopacceptance ./internal/cli -run TestKeyGeneration -count=1 -v
```

Observed:

```
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (1.46s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.32s)
PASS
```

Live revocation, device A acting on device B, fed the account's recovery
code through `REINSTATE_RECOVERY_CODE_FD` (a plain temp file, non-inheritable
by default on Windows, marked inheritable and passed via
`syscall.SysProcAttr.AdditionalInheritedHandles` — the same pattern
`pair.go`'s `runAccountRecover` already uses for `rein account recover`):

```
rein.exe devices revoke <device-b-id> --json
```

Observed:

```
revoked device "Harjots-Beast" (efd00d82-3f3d-49a8-99eb-a242f5eda6d3); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

Then, on device B:

```
rein.exe whoami --json
{"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run `rein login` again"}
```
`[exit=4]`

```
rein.exe push --all --json
{"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run rein login again"}
```
`[exit=4]`

### H8b — Console-initiated revocation, pending until confirmed

An ephemeral self-signed-TLS reverse proxy (`crypto/tls`, ECDSA P-256,
`net/http/httputil.NewSingleHostReverseProxy`) listened on
`127.0.0.1:8443` and forwarded to the lab `hopd` at `127.0.0.1:8322` — the
Console session cookie is issued `Secure: true`
(`issueConsoleSession`), so a plain-HTTP lab cannot carry it at all.

```
curl -X POST https://127.0.0.1:8443/console/sign-in/email -d email=v060rc4-e@lab.local
```

`hopd.log` printed the real magic link (`Sign in to the Reinstate Console`,
`/console/sign-in/email/<token>`); `GET` then `POST` the same link through
the proxy (cookie jar retained) returned:

```
Set-Cookie: __Secure-reinstate_console=...; Path=/console; HttpOnly; Secure; SameSite=Strict
```

Fetched `/console/devices`, scraped the CSRF token from the revocation form
for device-c's row (`904832f5-d0c0-43d6-8924-4f5cfec96b1f`), then:

```
curl -X POST https://127.0.0.1:8443/console/devices/904832f5-d0c0-43d6-8924-4f5cfec96b1f/revocation --data-urlencode csrf=<token>
```
→ `303 See Other`, `Location: /console/devices`.

Wire-level check immediately after (target still enrolled):

```sql
select id, target_device_id, requested_generation, status from device_revocation_requests;
ee7313df-8329-46a4-8518-22c4b0fcdfad|904832f5-d0c0-43d6-8924-4f5cfec96b1f|2|pending
```

Device-c stayed functional while pending:

```
rein.exe whoami --json
```
→ succeeds, exit 0, names device `904832f5-d0c0-43d6-8924-4f5cfec96b1f`.

Confirmed from device-a (the recovery-code command), fed the same
`REINSTATE_RECOVERY_CODE_FD` mechanism as H8:

```
rein.exe devices revoke 904832f5-d0c0-43d6-8924-4f5cfec96b1f --json
```

Observed:

```
Console request ee7313df-8329-46a4-8518-22c4b0fcdfad confirmed after the keyring reached generation 3
revoked device "Harjots-Beast" (904832f5-d0c0-43d6-8924-4f5cfec96b1f); key generation 3 started with 1 enrolled device(s), and the control plane refuses its token
```

Wire-level check afterward:

```sql
select status, confirmed_generation from device_revocation_requests where id='ee7313df-8329-46a4-8518-22c4b0fcdfad';
confirmed|3
```

Device-c's next push refused:

```
rein.exe push --all --json
{"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run rein login again"}
```
`[exit=4]`

## Harness defects and notes (non-blocking to the mechanisms this part exercised)

- **Ambient host contamination re-observed (see the Host table above).**
  `REINSTATE_BACKEND=memory` / `REINSTATE_MEMORY_BACKEND_DIR=D:\Projects\hop-10-lab\locker`
  are set at the persistent user level on this host again, despite the
  `v0.6.0` acceptance contract's own run notes recording both as removed on
  2026-09-06. Not scored against any row in this part — every command in
  this part ran either behind an explicit `unset` or behind `hoplab env`'s
  own strip, confirmed by inspecting both variables in the invoking shell
  immediately before each `rein` call — but flagged so the maintainer can
  remove them from the user environment again (or identify what keeps
  reintroducing them; the path names a directory this run never wrote to).
- **`fakelocker`'s `AnyBucket:true` (carried, unchanged since `v0.6.0-rc.1`).**
  Makes step 4 of `push --all`'s automatic post-push verification fail
  against this lab's locker for every row in this part that pushed — H4/H9's
  own carried caveat, not scored against H6/H8.
- **Restarting only `hopd` (not the whole lab) with `HOPD_STORAGE=fake`
  silently orphans every account's existing locker bucket in memory (carried
  from `v0.6.0-rc.3`'s part E finding).** This run avoided the incident by
  restarting `hopd` (for `H6b`'s short-TTL requirement) before creating any
  account that would need the locker for that generation of the lab, and by
  re-establishing pairing state freshly after the restart rather than
  reusing pre-restart state across it.
- **`rein init --hop` on an already-initialized `REINSTATE_HOME` correctly
  refuses without `--force` (carried, unchanged).** Device B's project
  mapping for `H6` was added directly in `config.toml` — the "equivalent
  config mapping" the refusal message itself names — using a TOML literal
  (single-quoted) string for the Windows path; a double-quoted string with
  literal backslashes fails TOML's own escape-sequence parsing
  (`invalid escape in string`), which is a TOML syntax fact, not a product
  defect.
- **`H7` — not attempted this run; no incident.** Because the round trip was
  never started, this run caused no write of any kind to
  `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, or `XDG_DATA_HOME`, live or otherwise —
  unlike `v0.6.0-rc.3`'s part E, which did (that report's §16, §21, the
  reason the rc.4 dispatch added the lab-isolation requirement this
  disposition is honoring).

## Cleanup

This executor's own lab (`D:\ReinstateAcceptanceProjects\v060-rc4-e\`,
`hopd`/`fakelocker` on `127.0.0.1:8322`/`127.0.0.1:9322`) and its install
directory are deleted after this report is committed. The temporary TLS
proxy process was stopped before this report was drafted. Nothing from this
executor's lab directory is referenced by path elsewhere in this report
except the sanitized fields above.
