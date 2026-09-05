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
- **Non-blocking findings:** six, recorded in §6 for the coordinator's
  attention (F1 shared auth gap, F2 conptydriver TTY detection, F3 range
  widening not yet on this branch, F4 Task Scheduler elevation, F5
  extensionless build output, F6 PowerShell `ScheduledTasks` hangs on
  secure-desktop consent). None are release-blocking.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-05 evening through 2026-09-06 (IST `Asia/Kolkata`, UTC+05:30; timestamps below are as printed by each tool) |
| Host | Windows 11 Pro `10.0.26200`, native `windows-amd64` (not WSL) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-w6-hop-parity`, branch `v060/w6-hop-parity` |
| Tested commit | `b6ed9dc54acb4ccf2719d3a1aa2ce7ce053ca673` (`release/v0.6.0-rc.1` tip at branch time; confirmed with `git rev-parse HEAD`) |
| Client build | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go build -o bin/rein.exe ./cmd/reinstate` (and `bin/reinstate.exe`, byte-identical); `rein version` reports `v0.5.2-rc.1-185-gb6ed9dc5 (b6ed9dc54acb4ccf2719d3a1aa2ce7ce053ca673 2026-09-05T20:14:34Z)` |
| Go | `go1.25.13 windows/amd64` (`GOTOOLCHAIN=go1.25.13`) |
| hopd | Prebuilt binary at a scratch path (`REINSTATE_HOPD_BIN`); private `reinstate-hosted` control plane, fake storage (`HOPD_STORAGE=fake`), log email sender |
| fakelocker | `scripts/testing/fakelocker`, in-memory fake S3, accepting `FAKEKEY*` access key ids, any bucket (see F3) |
| Lab roots | One fresh root per journey group, all `hopd` `127.0.0.1:8302` / locker `127.0.0.1:9302` (started, exercised, then `hoplab stop`ped before the next): `v060-hop-b` (H6), `v060-hop-b-h6b` (H6b, `HOPD_PAIRING_TTL=5s`), `v060-hop-b-h7` (H7), `v060-hop-b-h8` (H8), `v060-hop-b-h8b` (H8b) |
| Agents on this host | `claude 2.1.261`, `codex-cli 0.149.0` (`codex` on `PATH`, resolves under `nvm4w`'s node shim dir), `opencode 1.18.27`, all real, installed, on `PATH` |
| Contaminating env vars | `REINSTATE_BACKEND=memory`, `REINSTATE_MEMORY_BACKEND_DIR`, `XDG_DATA_HOME`, `REINSTATE_S3_ACCESS_KEY_ID`, `REINSTATE_S3_SECRET_ACCESS_KEY` (all present at session start from earlier lab work) unset in every shell before running `rein`, `hoplab`, or `go test` |
| Isolation | `hoplab homes`/`hoplab env` gave each simulated device its own `REINSTATE_HOME`, `HOME`, `USERPROFILE`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME`, and the other catalog `RootEnv`s; every env block was materialized to a file once and sourced with an explicit guard (`case "$REINSTATE_HOME" in *<root>*<device>*) ... esac`) before any `rein` invocation, after a transient shell fork failure showed what an unguarded, silently-empty `eval` can do (see §5) |
| Secrets | Recovery codes fed through `REINSTATE_RECOVERY_CODE_FD` via `hoplab pair` for pairing, and via a small standalone Windows-handle-passing runner (`secretrun`, built the same way `scripts/testing/hoplab/secretfd_windows.go` does — `AdditionalInheritedHandles`, not shell fd redirection, which does not carry a real inheritable handle to a native child from Git Bash) for the two `rein devices revoke` calls hoplab's own subcommands do not wrap; deleted after use, never committed |
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

**B pulls.** Device A pushed its three seeded fixtures (`push --all` ->
"pushed 3 snapshot(s)"); device B's first `pull --all` refused the Claude
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

### 3.2 H6b — an expired pairing request, and B's absent wrap

Fresh lab root `v060-hop-b-h6b`, started with `HOPD_PAIRING_TTL=5s` set in
the shell before `hoplab start` (inherited into `hopd`'s process
environment by `hoplab`'s own `append(os.Environ(), ...)`, which does not
override this name). Device A initialized the account
(`pair init`); device B signed in, ran `rein init --hop`, then `rein
account join` directly (not through `hoplab pair join`, so it could be left
unapproved) and let the request expire on its own:

```text
$ rein account join                       # device-b
Pairing code for this device (never sent to the control plane):
    <pairing-code>
...The request expires at 2026-09-05T20:22:49Z (Ctrl-C to cancel).
the pairing request expired before it was approved and collected; run rein account join again on the new device
exit=4
```

`hopd.db` confirms the rollback, not just the client-side timeout:

```
$ sqlite3 <root>\hopd.db "SELECT id, device_id, version, status, payload IS NULL FROM pairing_requests;"
83c2699f-cf33-4834-9459-4385cca42c6c|<device-b-id>|2|expired|1
```

`status=expired`, `payload IS NULL` — the wrapped payload is dropped on
expiry (`internal/store/pairing.go`'s claim path:
`UPDATE pairing_requests SET status = ?, payload = NULL WHERE id = ?`), not
merely marked stale.

A late approve attempt with the same (now-expired) code, from device-a, is
refused outright:

```text
$ REINSTATE_PAIRING_CODE_FD=<fd> rein devices approve
no pending pairing requests; run rein account join on the new device first
exit=2
```

And device-a's own `rein devices` afterward shows B enrolled with **no
wrap in the account's only generation**:

```
<device-a-id>  ... holds a root-key wrap (key generation 1)
<device-b-id>  ... no root-key wrap yet
```

Since this account never rolled past generation 1, "absent from every
generation" is the whole (one-generation) keyring here; the payload-null
and the explicit late-approve refusal are the two mechanisms doing the
actual work, and both were exercised directly, not inferred.

**Verdict: PASS.**

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
device-b pushes successfully (`pushed 3 snapshot(s)`). Device-a revokes it
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
