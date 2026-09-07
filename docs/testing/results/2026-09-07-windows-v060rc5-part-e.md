# `v0.6.0-rc.5` Windows Hop parity — part E (H6, H6b, H7, H8, H8b)

`PHASE5-DEVICE-REPORT-V1` (section D subset)

Executor E's slice of the tagged-artifact acceptance for `v0.6.0-rc.5`,
against [`docs/testing/v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D and [`v0.6.0-rc.5-agent-verification-prompts.md`](../v0.6.0-rc.5-agent-verification-prompts.md),
in particular the corrected `MatrixH:H7` lab-isolation method (issue #424).
Rows `H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`, `H5`, `H9`, `H10`, `H11`, `H12` are
out of this part's scope (executor D's part,
[`2026-09-07-windows-v060rc5-part-d.md`](2026-09-07-windows-v060rc5-part-d.md))
and are not recorded here.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.5` |
| Full commit | `0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9` |
| Windows archive | `reinstate_0.6.0-rc.5_windows_amd64.zip` |
| Archive SHA-256 | `a648f1de65c15bea7926dda2dd0ce48110d512c9bc60a23b5988c3b24ded9f09` (matches `checksums.txt`, re-verified before install) |
| Install source | checksummed archive at the coordinator's verified `rc5-draft` staging directory, per this candidate's dispatch (Executor A alone uses the live bootstrap; every other executor, including this one, installs from the pre-verified archive) |
| Install directory | own fresh directory under `D:\ReinstateAcceptanceProjects\v060-rc5-e\install\` |
| `rein.exe` / `reinstate.exe` byte identity | identical (`sha256 aa74899f68356ea5127fa8129db56729c137276d255f1d8890fa9dcf276dd35f` both) |
| `rein version --json` | `{"commit":"0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9","date":"2026-09-07T12:59:29Z","name":"reinstate","version":"0.6.0-rc.5"}` |
| **Bootstrap deviation** | This part did **not** use the live `reinstate.dev/install.ps1` bootstrap. Per the dispatch, only Executor A installs from the live bootstrap and records the artifact identity from it; every other executor, including this one, installs the same checksummed archive from the coordinator's pre-verified staging directory. No bootstrap script was fetched or run by this part. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Windows 11 Pro 10.0.26200 (Build 26200), x64 |
| Shell | Git Bash (POSIX); PowerShell 5.1 for the H7 elevated round trip |
| Go | `go1.26.1 windows/amd64` (host toolchain; module pins `go1.25.13` via `GOTOOLCHAIN` where invoked) |
| Device name shown by Hop (`whoami`/login) | redacted as `<device-name>` throughout this report (the host's real machine name) |
| Lab root (H6, H6b's live pair, H8) | `D:\ReinstateAcceptanceProjects\v060-rc5-e\hoplab\`, `hopd` `127.0.0.1:8322`, `fakelocker` `127.0.0.1:9322` |
| Additional short-lived lab pairs | a short-TTL pair for H6b's expiry (`127.0.0.1:8332`/`9332`, stopped immediately after); a dedicated pair for H8b's Console-driven revocation (`127.0.0.1:8342`/`9342`, stopped immediately after) |
| H7 fresh Hop lab account | `D:\ReinstateAcceptanceProjects\v060-rc5-e\h7-device\`, same main lab (`127.0.0.1:8322`) |
| Date | 2026-09-07 |

## Environment hygiene

Every shell in this part began by unsetting `REINSTATE_BACKEND` and
`REINSTATE_MEMORY_BACKEND_DIR` (confirmed empty before each row).
`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were never unset —
this part's rows do not create real vendor sessions (H6/H6b/H8/H8b use only
`hoplab homes`' synthetic session fixtures; H7's mechanism is the daemon's
install/status/stop/start round trip and does not touch any agent's store
at all), so the live-home carve-out that matters here is the corrected
`MatrixH:H7` lab-isolation rule's before/after listing-digest proof, below,
not an isolated vendor credential.

## Section D — Hop parity journeys (this part's 5 rows)

| # | Row | Result | Summary |
| - | --- | ------ | ------- |
| H6 | Device B `rein account join`; device A `rein devices approve`; B pulls | PASS | Live pairing via `hoplab pair join`; `pairing_requests` row confirmed `status=consumed, claims=2, version=2`; device B pulled all 3 of device A's sessions after mapping device A's project id to its own local path; `sessions --json` on B showed 6 sessions, no key overlap |
| H6b | An expired pairing request is refused or rolled back, and B's wrap is absent from every generation | PASS | A dedicated short-TTL lab (`HOPD_PAIRING_TTL=8s`); `rein account join` run directly (no approver ever ran) refused client-side, exit 4, message names the expiry; `pairing_requests` row confirmed `status=expired, payload IS NULL, claims=5`; `account status` on the joining device showed `device_in_keyring=false, enrolled_devices=1` |
| H7 | `rein daemon install/status/stop/start/uninstall` round trip through Task Scheduler; the foreground loop pushes after a change (debounced) and pulls on schedule and before a resume | **PARTIAL — operator unavailable** | A fresh Hop lab account (zero pushed sessions) was created and the mandatory before-digest was captured, satisfying every prerequisite the corrected lab-isolation rule (#424) requires; the elevated round trip itself was never started because no maintainer go-ahead arrived. Polled `D:/ReinstateAcceptanceProjects/h7-go.txt` every 60s from `2026-09-07T14:42:41Z` through `2026-09-07T15:22:41Z` (the full 40-minute window this run's own orchestrator allotted, after every other row in this part was already done, per the row's own ordering); it never appeared. No unattended-elevation workaround was attempted. Because the round trip was never started, no write of any kind occurred to `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, or `XDG_DATA_HOME` this run |
| H8 | A revokes B: generation rolls, B's token is refused, B cannot open a later push; the lagging-device attack is refused on push, pull, and verify naming the control plane; the 404-floor proxy reproduces the documented residual | PASS | `keygeneration_crossplane_test.go -tags hopacceptance` against a real `hopd`: both crossplane tests passed (`2.47s`, `1.84s`); 404-floor residual reproduced (asked 4 times, answered 404 every time). Live: device A revoked device B, generation rolled 1→2; B's `whoami`, `push --all`, `pull --all`, and `sync verify` all refused, exit 4 each |
| H8b | A Console-initiated revocation stays pending until the recovery-code command writes a strictly newer generation | PASS | A local, ephemeral self-signed-TLS reverse proxy drove the real Console HTTP routes end to end (email sign-in, CSRF scrape, `POST /console/devices/{id}/revocation`); `device_revocation_requests` row went `pending` (target device stayed functional, confirmed via a real `whoami`) then `confirmed` (`confirmed_generation=2`) after `rein devices revoke` printed "confirmed after the keyring reached generation 2"; target's next push refused, exit 4 |

**4/5 PASS, 1 PARTIAL (`H7`, operator/harness availability — the same
disposition, for a different practical reason, as the `v0.6.0-rc.4` tagged
run's own `H7` `PARTIAL`; see the note below the H7 evidence).**

---

## Evidence

Argv-only commands; no transcript text, prompts, credentials, private paths,
or session ids the device did not create appear below. `<recovery-code>` /
`<pairing-code>` / `<account-key>` / `<device-key>` stand in for the literal
values, per the ground rules. Every real vendor process this part started
(none — this part's rows use only synthetic `hoplab homes` session fixtures
and the daemon's own mechanism, never a live agent invocation) is noted
where relevant.

### H6 — live pairing, protocol v2, B pulls

```
rein login --email <email> --json          # device-a, then device-b, same email
hoplab approve -root <lab> -email <email> -count 1 -timeout 60s   # once per device
hoplab pair init -root <lab> -device device-a -rein <rein.exe>    # account init, recovery code shown once
hoplab pair join -root <lab> -device device-b -approver device-a -rein <rein.exe>
```

Observed: `hoplab pair join` printed `hoplab: device-b joined the account
live, approved by device-a`. A direct `hopd.db` query:

```sql
SELECT id, status, claims, version FROM pairing_requests;
```

showed exactly one row, `status=consumed, claims=2, version=2` — the
pairing-request schema's own `binding`/`payload` columns are stored
separately (confirmed via `.schema pairing_requests`), matching the row's
"separate HMAC and payload keys" mechanism.

Device A pushed its three synthetic fixture sessions (`push --all --json`)
before device B pulled:

```
rein push --all --json      # device-a: {"snapshots":[...3 ids...]}
```

(Step 4 of the verification report — the bucket-isolation check — reported
`FAIL` for the same pre-existing `scripts/testing/fakelocker` `AnyBucket:
true` harness limitation carried from every prior run's H4/H9 evidence, not
a new finding; see Findings, below.)

Device B initially refused to pull with `claude snapshot project
"hoplab-device-a" has no local mapping on this device; configure it with
`rein init --project hoplab-device-a=/absolute/path` ...` — the correct,
documented refusal for an unmapped project. Device B's own project mapping
was extended with that id (`rein init --hop --project
hoplab-device-b=<path> --project hoplab-device-a=<own-different-path>
--force --yes`), then:

```
rein pull --all --json      # device-b: {"pulled":3,"skipped":0}
rein sessions --json        # device-b: 6 "key" entries, no overlap
```

**Harness note:** the first attempt at extending device B's project mapping
(`rein init --hop --force --yes --project ...`) silently reset
`[encryption] type` from `root-key` to `age-scrypt` in the regenerated
`config.toml`, which broke local decryption of the existing account keyring
(`account status` then showed `enrolled_on_this_device:false`,
`pull --all` refused with "this device is not enrolled in the account's
keyring yet"). This is very likely `--yes`'s non-interactive default for
`[encryption]` not preserving a `--hop` profile's own `root-key` default
when nothing on the command line named it explicitly — worth the
coordinator's attention, though outside this row's own scope, since it did
not block H6: restoring `config.toml`/`account.json`/`state.json` from the
timestamped backup `--force` itself wrote
(`reinstate/backups/<ts>-reinitialize/`) and adding the new project mapping
as a plain text edit instead (same project-id/path syntax `init --project`
itself writes) fixed it, and the row's actual pull/pairing mechanism was
otherwise untouched by this detour.

### H6b — expired pairing request refused/rolled back

A dedicated, short-TTL lab pair (`HOPD_PAIRING_TTL=8s`, `127.0.0.1:8332`/
`127.0.0.1:9332`) was started only for this row, then stopped immediately
after.

```
rein login --email <email2> --json          # device-a, then device-b, same email
hoplab approve -root <lab2> -email <email2> -count 1 -timeout 60s
hoplab pair init -root <lab2> -device device-a -rein <rein.exe>
rein init --hop --project hoplab-device-b=<path> --json               # device-b
rein account join --json                                              # device-b; no approver ever runs
```

`rein account join` printed the pairing code and the request's own
`expires_at` (8s later), then, with no approver ever invoked, exited `4`:

```
{"code":"auth_storage","message":"the pairing request expired before it was approved and collected; run rein account join again on the new device"}
```

A direct `hopd.db` query:

```sql
SELECT id, status, claims, version, payload IS NULL FROM pairing_requests;
```

showed `status=expired, claims=5, version=2, payload IS NULL=1` — the
request rolled back, never wrote a payload. `rein account status --json` on
device-b:

```json
{"enrolled_on_this_device":false,"device_in_keyring":false,"enrolled_devices":1,...}
```

confirms device-b's wrap is absent — only device-a (generation 1's sole
member) is enrolled.

### H7 — daemon install/status/stop/start round trip through Task Scheduler

This row's own row-ordering rule was followed: every other row in this part
(H6, H6b, H8, H8b) was completed first, and the go-signal poll started only
after all four were done and recorded.

**Prerequisites completed (no elevation needed for any of these):**

```
rein login --email <email5> --json                          # h7-device, main lab
hoplab approve -root <lab> -email <email5> -count 1 -timeout 60s
rein init --hop --project h7-throwaway=<isolated-path> --json
rein account init --json     # recovery code fed back via REINSTATE_RECOVERY_CODE_FD
                              # (a purpose-built inherited-handle helper, built for
                              #  this run only, never committed)
```

produced a **fresh Hop lab account** — `account status --json` afterward
showed `key_generation:1, enrolled_devices:1, keyring_present:true`, and
because this device's session store is a brand-new isolated fixture root
with nothing in it yet, `rein sessions --json` returned zero entries, so
the account trivially has **zero sessions pushed to it** (nothing existed
to push before the round trip). This satisfies the corrected lab-isolation
rule's own definition — a new Hop account, not a new Windows user account,
with zero prior pushes — without any elevation beyond what `daemon
install`'s own UAC prompt would raise.

Before-listing digests (sorted SHA-256 over relative path + size) of the
three live agent homes, taken before the go-signal wait began:

| Root | Before digest |
| ---- | -------------- |
| `CLAUDE_CONFIG_DIR` (`D:\Projects\hop-10-lab\claude`) | `21c8719656a45d93a4d312485ec8c67a3dc5c4b382b1bd8781214b841c259200` |
| `CODEX_HOME` (`...\orca\codex-runtime-home\home`) | `0ae01f1283799b17526de22136642212cde429930cc2056f0ac7fd85999273e8` |
| `XDG_DATA_HOME` (`D:\Projects\hop-10-lab\xdg`) | `6b9a0c67efb2846b9ea6f27e16ab3076be0d6c8ba6e3ddf74fb486e5d9462e3e` |

An elevated `Start-Process -Verb RunAs` launcher and a companion script
(`rein daemon install/status/stop/start/uninstall` round trip through real
`schtasks`, plus a synthetic Claude session drop to exercise the foreground
loop's debounced push and its scheduled pull) were prepared and staged,
ready to fire the moment a go-ahead appeared, per this row's own
requirement that the elevation itself needs the maintainer present for the
UAC prompt — no unattended-elevation workaround (a self-elevating scheduled
task, a stored credential, etc.) was built or attempted, per the dispatch's
own explicit prohibition.

**The go-signal never arrived.** `D:/ReinstateAcceptanceProjects/h7-go.txt`
was polled every 60 seconds from `2026-09-07T14:42:41Z` — checked once
before the wait began (absent) and continuously afterward — through
`2026-09-07T15:22:41Z`, a full 40 minutes, and confirmed still absent
immediately after. This run's own orchestrator allotted a 40-minute poll
window for this part (shorter than the contract's own 150-minute figure,
because a longer wait had already once ended the orchestrating session
mid-poll on a prior attempt); the shorter window is a harness/scheduling
constraint of this particular run, not a re-interpretation of the
contract's own text. The elevated script was never launched, so the UAC
prompt was never raised and no daemon action of any kind was taken this
run — none of `install`, `status`, `stop`, `start`, or `uninstall` executed,
and no after-digest comparison is meaningful because nothing was attempted.
**No write of any kind occurred this run to `CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, or `XDG_DATA_HOME`** — the same safe (non-)outcome the
`v0.6.0-rc.4` tagged run's own `H7` `PARTIAL` recorded, for a different
practical reason: that run could not even set up its (then-misread) fresh
Windows-account prerequisite; this run correctly set up the fresh Hop
account per the corrected rule and prepared the elevated script, but the
maintainer was not reachable inside this run's own poll window.

### H8 — A revokes B; lagging-device attack; 404-floor proxy

```
REINSTATE_HOPD_BIN=<built hopd.exe> GOTOOLCHAIN=go1.25.13 \
  go test -tags hopacceptance ./internal/cli -run TestKeyGeneration -count=1 -v
```

```
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (2.47s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.84s)
PASS
```

Live, against the H6 pairing's own two-device account (device-a, device-b,
`key_generation:1` at the time): device-a revoked device-b using the
recovery code captured at `hoplab pair init` time, fed through
`REINSTATE_RECOVERY_CODE_FD` via a purpose-built inherited-handle helper
(same mechanism `scripts/testing/hoplab/secretfd_windows.go` documents,
built for this run only, never committed):

```
rein devices revoke <device-b-id> --json
```

printed `revoked device "<device-name>" (<device-b-id>); key generation 2
started with 1 enrolled device(s), and the control plane refuses its
token`. Immediately after, on device-b:

```
rein whoami --json      # exit 4, "this device's token was rejected by the control plane (revoked or stale)"
rein push --all --json  # exit 4, same message
rein pull --all --json  # exit 4, same message
rein sync verify --json # exit 4, same message
```

All four refused identically, naming the control plane's rejection, not a
generic error.

### H8b — Console-initiated revocation stays pending until confirmed

A dedicated lab pair (`127.0.0.1:8342`/`9342`) with its own two-device
account (device-a holding the recovery code, device-b the target), plus a
local, ephemeral self-signed-TLS reverse proxy (`https://127.0.0.1:8443` →
the lab's plain-HTTP `hopd`) — required because Go's own
`net/http/cookiejar`, like a browser, withholds a `Secure`-flagged cookie
from any request whose URL scheme is not `https` (it stores the cookie but
never sends it back over plain HTTP), so the Console's session cookie needs
a real TLS front to be usable across requests, even against a loopback lab.
A purpose-built driver (Go, `net/http` + `cookiejar`, built for this run
only, never committed) drove the real routes:

```
POST https://127.0.0.1:8443/console/sign-in/email        (form: email=<email>)
GET  https://127.0.0.1:8443/console/sign-in/email/<token>  (renders the confirm page)
POST https://127.0.0.1:8443/console/sign-in/email/<token>  (issues the session cookie)
GET  https://127.0.0.1:8443/console/devices                (scrape csrf, confirm target id present)
POST https://127.0.0.1:8443/console/devices/<device-b-id>/revocation   (form: csrf=<scraped>)
```

Every step returned `200` and the session was genuinely established (the
`/console/devices` page rendered with the target device's real id and a
`csrf` token present, not the sign-in page). A direct `hopd.db` query
immediately after:

```sql
SELECT id, target_device_id, status, requested_generation FROM device_revocation_requests;
-- 5d88b2e0-...|<device-b-id>|pending|0
```

confirmed `pending`; `rein whoami --json` on device-b at this point still
succeeded (target device stayed fully functional while pending). Then,
device-a (recovery code fed via the same inherited-handle helper as H8):

```
rein devices revoke <device-b-id> --json
```

printed `Console request 5d88b2e0-... confirmed after the keyring reached
generation 2`, then the same revoke summary H8 showed. The DB row
afterward: `status=confirmed, confirmed_generation=2`. Immediately after:

```
rein push --all --json      # device-b: exit 4, token rejected
```

## Findings (this part)

| Severity | Row(s) touched | Description | Release blocking |
| -------- | --------------- | ------------ | ----------------- |
| Carried, non-blocking (unchanged from prior tagged runs) | H6 (surfaced while pushing device-a's fixtures) | Step 4 (bucket isolation) fails for the pre-existing `scripts/testing/fakelocker` `AnyBucket:true` harness limitation — not a control-plane or product defect. Same disposition as every prior run's H4/H9 evidence. | No |
| MINOR (product observation, surfaced while satisfying H6) | H6 | `rein init --hop --force --yes --project ...` regenerated `config.toml` with `[encryption] type = "age-scrypt"` instead of preserving the profile's existing `root-key` encryption, breaking local keyring decryption on an already-enrolled device until the pre-`--force` backup was restored and the new project mapping added as a plain edit instead. Worth the coordinator's attention: a forced re-init that only intends to add a project mapping should not silently change the profile's own encryption type out from under an already-enrolled device. | No — this row's own mechanism (pairing, pull, no-overlap sessions) was fully exercised and correct once the encryption type was restored; the defect is in the config-regeneration side-effect of a broader `init --force`, not in pairing/pull itself |
| operator/harness availability, re-recorded reason | H7 | This run correctly set up the corrected fresh-Hop-account prerequisite (issue #424's own fix, unlike the `v0.6.0-rc.4` tagged run, which misread it as a fresh Windows account and could not set that up at all) and staged the elevated Task Scheduler script, but the maintainer go-ahead never arrived inside this run's own 40-minute poll window (shorter than the contract's own 150-minute figure, a scheduling constraint of this particular run, not a contract reinterpretation). No product or contract-text defect — the row's own mechanism (`daemon install` through real `schtasks`, the foreground loop's debounced push and scheduled pull) was never reached | Blocks — `PARTIAL` does not pass a required row; no unattended-elevation workaround was attempted, per the dispatch's own explicit prohibition; no live-agent-home write occurred this run |

No release-blocking *product* finding in this part's rows. `H7`'s
`PARTIAL` blocks the required-row count (per the contract's own
`PARTIAL`/`NOT TESTED` rule) but is an operator/harness-availability
disposition, not a product or documented-contract defect — see the row's
own evidence above and the dispatch's carried-dispositions table.

## Cleanup

Every Hop lab this part started (`D:\ReinstateAcceptanceProjects\v060-rc5-e\hoplab`,
the short-TTL H6b pair at `...\hoplab-h6b`, the H8b pair at `...\hoplab-h8b`)
was confirmed via `hoplab ps` to be the ones this part started, then stopped
with `hoplab stop -root <root>` before writing this report; only the main
lab (`...\hoplab`) remained running for the H7 wait, per the row's own
ordering, and was stopped once H7 was recorded. The local self-signed-TLS
proxy process (H8b) was terminated after use. All temporary Go verification
helpers (the recovery/pairing-code inherited-handle feeder, the Console
HTTP driver, the self-signed-TLS proxy) were built and run only under
`tmp-` directories inside the worktree and removed before this commit;
`git status` is clean of anything but this report. Isolated lab and device
directories under `D:\ReinstateAcceptanceProjects\v060-rc5-e\` hold no
committed content and are not part of this repository.
