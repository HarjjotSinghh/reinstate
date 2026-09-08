# `v0.6.0-rc.8` tagged Windows acceptance — part E (`H6`, `H6b`, `H7`, `H8`, `H8b`)

Tagged-run executor E, Hop parity journeys `H6`, `H6b`, `H7`, `H8`, `H8b`
(section D of
[`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)), against
this executor's own disposable lab. This part-file covers 5 of the 16
required Hop parity rows; the remaining 11 (`H1`–`H5`, `H9`–`H12`) and every
other section are reported elsewhere by the other tagged-run executors and
assembled into the full device report separately.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.8` |
| Full commit | `3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f` |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc8-tagged`, branch `v060/rc8-tagged`, at the commit above (also `origin/main`) |
| Windows amd64 archive | `reinstate_0.6.0-rc.8_windows_amd64.zip` |
| Archive SHA-256 | `658dc27e607fdec14d14156dfe3edaf9d685bfa9add1e467d604ab0fa298e28f` (matches `checksums.txt`, re-verified by this executor before install) |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc8-e\install\` (fresh, this executor's own) |
| `rein.exe` / `reinstate.exe` SHA-256 | `482b5de12db1312c1a767bcf40c7999de9f8cef7f47bff5bce802fab960c3b71` (both files, byte-identical to each other and to `checksums.txt`'s `reinstate_0.6.0-rc.8_windows_amd64.exe` entry) |
| `rein version --json` | `{"commit":"3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f","date":"2026-09-08T22:38:21Z","name":"reinstate","version":"0.6.0-rc.8"}` |
| Bootstrap deviation | This executor is not executor A and installed from the coordinator-verified, checksum-matched draft directory (`...\scratchpad\rc8-draft\reinstate_0.6.0-rc.8_windows_amd64.zip`) per the run's ground rules, not from the live `https://reinstate.dev/install.ps1` bootstrap; executor A alone verified the live bootstrap pin for this candidate. |
| Go toolchain | `GOTOOLCHAIN=go1.25.13` pinned for every `go test`/`go run` invocation in this part (local default toolchain on this host is `go1.26.1`) |

## Host (sanitized)

Native Windows 11 Pro x64 (`windows/amd64`, never WSL). PowerShell 5.1
(`5.1.26100.9278`) for the elevated round trip and report shell; Git Bash
for POSIX scripts and `sqlite3` queries. No developer agent trees were
touched: `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were read
(never unset) only where the row's own mechanism required it (`H7`); every
other row ran under this executor's own isolated device homes with those
three variables pointed at lab-owned directories. `REINSTATE_BACKEND` and
`REINSTATE_MEMORY_BACKEND_DIR` were confirmed unset in every shell before
every command in this part.

## Lab

Local, disposable `hopd` + `fakelocker`, never the hosted control plane:

- Main lab: `hopd 127.0.0.1:8322`, locker `127.0.0.1:9322`, root
  `D:\ReinstateAcceptanceProjects\v060-rc8-e\lab\` — used for `H6`, `H8`,
  `H8b`.
- Dedicated short-TTL lab (`H6b` only, an expired-request row needs its own
  `HOPD_PAIRING_TTL`): `hopd 127.0.0.1:8324`, locker `127.0.0.1:9324`, root
  `D:\ReinstateAcceptanceProjects\v060-rc8-e\lab-h6b\` — stopped and not
  reused once `H6b` finished.
- `hopd.exe` for both labs: this executor's own copy of a prebuilt `hopd`
  binary from a prior candidate's own part E
  (`v060-rc7-e\bin\hopd.exe`, unmodified since `reinstate-hosted`'s own
  checkout has not moved since 2026-09-05, well before that binary was
  built) — test infrastructure, not the artifact under test, the same
  reuse the `v0.6.0-rc.7` tagged run's own verification round documented.
- `H7`'s own fresh device: `REINSTATE_HOME`
  `D:\ReinstateAcceptanceProjects\v060-rc8-e\h7\reinstate`, against the
  main lab.

Two disposable Go helper programs were written for this part only, never
committed, and deleted from the worktree before this report was committed:
`scripts/testing/devicerevoke` (drives `rein devices revoke` in-process with
the recovery code read from a local file, mirroring
`scripts/testing/accountinit`'s own non-interactive pattern, to avoid the
Windows raw-handle `REINSTATE_RECOVERY_CODE_FD` plumbing outside a real
subprocess-spawning caller) and `scripts/testing/consoledriver` (a
disposable self-signed-TLS reverse proxy in front of this lab's real
`hopd`, driving the real Console HTTP routes for `H8b` — the same shape
the `v0.6.0-rc.7` tagged run's own `H8b` evidence used, rewritten fresh for
this run since no consoledriver helper is checked into the repository).

## Verdict (this part)

- **Rows in this part:** 5 (`H6`, `H6b`, `H7`, `H8`, `H8b`), all required.
- **Result:** `5 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED`.
- **Release-blocking findings from this part:** none.
- **`#424`** (the Task Scheduler daemon following the ambient/persistent
  login environment rather than the environment `daemon install` ran
  under) fired again on `H7`, exactly as designed to be recorded, not
  failed, per the contract's refined rule (d) — see `H7`'s evidence below.

| # | Row | Result |
| - | --- | ------ |
| H6 | Device B `rein account join`; device A `rein devices approve`; B pulls | PASS |
| H6b | An expired pairing request is refused/rolled back, B's wrap absent everywhere | PASS |
| H7 | `daemon install/status/stop/start/uninstall` through Task Scheduler; debounced push, scheduled pull | PASS |
| H8 | A revokes B; the lagging-device attack refused on push/pull/verify; 404-floor proxy | PASS |
| H8b | A Console-initiated revocation stays pending until the recovery-code command confirms it | PASS |

---

## `H6` — pairing protocol v2, separate HMAC and payload keys

**Mechanism exercised:** live `hoplab pair join` (device B publishes a
pairing request via `rein account join`; device A approves it via `rein
devices approve`, no code ever typed by hand), then a real cross-device
push/pull with a project mapping, against the installed `v0.6.0-rc.8`
binary throughout.

**Evidence:**

```
rein login --email <h6-lab-email> --no-browser --json      # device-a
rein login --email <h6-lab-email> --no-browser --json      # device-b (same email)
hoplab approve -root <lab> -email <h6-lab-email> -count 2 -timeout 3m
hoplab pair init -root <lab> -device device-a -rein <candidate rein.exe>
hoplab pair join -root <lab> -device device-b -approver device-a -rein <candidate rein.exe>
rein push --all --json                                      # device-a: 3 fixture sessions
rein pull --all --json                                      # device-b, before mapping: refused
rein init --project hoplab-device-a=<device-b-local-path>    # config-mapping fallback (init --project alone refused: home already initialized); "or the equivalent config mapping" per the refusal text
rein pull --all --json                                      # device-b, after mapping
rein sessions --json                                         # device-b
```

- `hoplab pair join` → `device-b joined the account live, approved by
  device-a` (exit 0).
- `pairing_requests` (sqlite): `status=consumed, version=2, claims=2`.
- `rein push --all --json` on device-a: `"pulled"`-shaped result showing 3
  snapshots pushed (`claude`, `codex`, `opencode`), `skipped:0`. (The
  push's own automatic post-push isolation check reported `outcome: fail`
  at its step 4 — the same documented `fakelocker AnyBucket:true` harness
  limitation `H9` already records elsewhere in this candidate's report,
  not a defect in pairing itself; step 4 said `FAIL`, never `could not
  run`.)
- First pull attempt on device-b, before any mapping: refused —
  `claude snapshot project "hoplab-device-a" has no local mapping on this
  device; configure it with rein init --project hoplab-device-a=/absolute/path
  (or the equivalent config mapping) before pull`, exit 1.
  `rein init --project ... ` alone refused (`reinstate home is already
  initialized; rerun init with --force to back up and replace existing
  config/state`) since device-b's home already carries its Hop
  enrollment from `pair join`; the mapping was added the refusal's own
  offered alternative way, a `[[projects]]` entry appended to
  device-b's own `config.toml`, matching the schema `rein init --project`
  itself writes.
- Second pull attempt: `{"pulled": 3, "skipped": 0}`. `rein sessions
  --json` on device-b: **6 sessions total** — the 3 pulled
  (`claude:session-syn-001-a`, `codex:rollout-syn-001-a`,
  `opencode:ses_fixture001a`, all now under device-b's own remapped local
  path) plus device-b's own original 3
  (`claude:session-syn-001-b`, `codex:rollout-syn-001-b`,
  `opencode:ses_fixture001b`) — no session-key overlap.

**Disposition: `PASS`.** The pairing protocol v2 round trip (publish,
live approval, consumption) completed against the real control plane, and
the cross-device push/pull it exists to unlock delivered exactly the
sessions device-a pushed, distinguishably alongside device-b's own,
matching the `v0.6.0-rc.7` tagged run's own `H6` result on this mechanism.

---

## `H6b` — `UnenrolEverywhere`: an expired pairing request

**Mechanism exercised:** a dedicated lab with `HOPD_PAIRING_TTL=8s`; device
B publishes a pairing request and no approver ever runs against it.

**Evidence:**

```
HOPD_PAIRING_TTL=8s hoplab start -root <lab-h6b> -hopd-addr 127.0.0.1:8324 -locker-addr 127.0.0.1:9324 -background
rein login --email <h6b-lab-email> --no-browser --json     # device-a, device-b
hoplab pair init -root <lab-h6b> -device device-a -rein <candidate rein.exe>
rein init --hop --project hoplab-device-b=<device-b-home> --json   # device-b
rein account join --json                                    # device-b, no approver ever running
rein account status --json                                  # device-b, after expiry
```

- `rein account join` printed the pairing code, then blocked for the full
  8-second TTL and refused client-side:
  `{"code":"auth_storage","message":"the pairing request expired before it
  was approved and collected; run rein account join again on the new
  device","safe_to_retry":false}`, exit `4`, elapsed `8s`.
- `pairing_requests` (sqlite): `status=expired, payload='' (empty),
  claims=5, version=2`.
- `rein account status --json` on device-b: `"device_in_keyring": false,
  "enrolled_devices": 1` — device-b's request never produced a wrap in
  any generation.

**Disposition: `PASS`.** The expired request rolled back cleanly: no
payload was ever posted, device-b holds no keyring wrap, and the account
still shows exactly the one device that actually enrolled — matching
`UnenrolEverywhere`'s documented guarantee and the `v0.6.0-rc.7` tagged
run's own result.

---

## `H7` — `daemon install/status/stop/start/uninstall` through Task Scheduler

**Ordering note.** `D:/ReinstateAcceptanceProjects/h7-go.txt` already
existed when this part started (content `go`). Per the go-signal rule,
`H7` was run **first**, ahead of `H6`, `H6b`, `H8`, `H8b`.

**Fresh Hop lab account, zero pushed sessions.** A brand-new account was
created through this part's own `hopd` under a fresh `REINSTATE_HOME`
(`D:\ReinstateAcceptanceProjects\v060-rc8-e\h7\reinstate`) this row created
and never pushed anything from:

```
rein login --email <h7-fresh-email> --no-browser --json
rein init --hop --project <fresh-project-mapping> --json
go run ./scripts/testing/accountinit           # rein account init, recovery code redacted by the helper
rein sync verify --json
```

`rein sync verify --json` beforehand: `"outcome": "not-applicable"`,
`"nothing has been pushed from this profile yet"` — genuinely fresh, key
generation 1, `devices=1`.

**Full per-file before-listing.** Every file (relative path, size, mtime)
under exactly the three session-bearing subtrees was listed, excluding
`*-wal`, `*-shm`, `*.lock`: `<CLAUDE_CONFIG_DIR>/projects`,
`<CODEX_HOME>/sessions`, `<XDG_DATA_HOME>/opencode` — `CODEX_HOME` read as
the **persistent, user-level** registry value
(`D:\Projects\hop-10-lab\codex`), confirmed different from this shell's
own transient `CODEX_HOME`
(`C:\Users\admin\AppData\Roaming\orca\codex-runtime-home\home`) — the
exact trap the contract's own "Lab isolation for `MatrixH:H7`" section
warns about. Before-listing: **3,191 files**.

**Elevated round trip.** Launched via PowerShell `Start-Process -Verb
RunAs` running a script that wrote its full output to a file
(`Start-Transcript`); the UAC prompt was accepted by the maintainer within
the launch call (elapsed ~44s wall clock for the whole elevated process,
including its own 40-second in-script wait). Argv-only summary of the
script's own steps and their observed lines:

```
rein daemon install --json
  -> installed schtasks com.reinstate.daemon.a76d4cff; the daemon starts at login and is starting now
rein daemon status --json      (immediately after install)
  -> installed:true, running:true (schtasks), alive:false (loop not fully up yet)
[wait 40s for a debounced push + scheduled pull]
rein daemon status --json      (after the wait)
  -> alive:true, pid set, roots:[D:\Projects\hop-10-lab\claude\projects, \codex\sessions, \xdg\opencode]
     push: {"summary":"pushed 25 snapshot(s), skipped 0 unchanged","ok":true}
     pull: {"summary":"pulled 0 snapshot(s), skipped 0 already synced","ok":true}
rein hop status --json
  -> locker first_push_at now set on this run's own disposable locker; devices:1
rein daemon stop --json    -> stopped com.reinstate.daemon.a76d4cff
rein daemon status --json  -> installed:true, running:false
rein daemon start --json   -> started com.reinstate.daemon.a76d4cff
rein daemon status --json  -> installed:true, running:true
rein daemon stop --json    -> stopped com.reinstate.daemon.a76d4cff
rein daemon uninstall --json -> uninstalled schtasks com.reinstate.daemon.a76d4cff; log kept
rein daemon status --json  (final) -> installed:false, running:false, detail:"not registered"
```

Every step completed against a real `schtasks` task
(`com.reinstate.daemon.a76d4cff`).

**`#424` reproduced again, recorded not failed, per rule (d).** The
scheduled daemon's reported `roots` were the **live** host paths
(`D:\Projects\hop-10-lab\claude\projects`, `\codex\sessions`,
`\xdg\opencode`), not this row's isolated `h7\reinstate` home — the same
Task-Scheduler environment-inheritance gap every prior `H7` attempt on
this program has documented (`--home` is baked into the scheduled task's
own command line and correctly targeted this row's own fresh account/
locker; the three agent-root variables are not, and the scheduled task
inherits the ambient, persistent user environment instead).

**Full per-file after-listing:** **3,193 files** (2 more than before).
Diffed against the before-listing: **exactly 8 differing entries**:

- `claude/projects/D--Projects-reinstate/<this executor's own workflow
  scratchpad id>.jsonl` — grew (this executor's own live session
  transcript, writing throughout this very run).
- 5 files under
  `claude/projects/D--Projects-reinstate/<the same workflow id>/subagents/workflows/wf_161f1355-914/agent-*.jsonl`
  — grew; a sibling subagent under the same orchestrated tagged-run
  workflow, writing its own transcript concurrently during the round
  trip's own ~40-second window — the same "this session's own tool use"/
  sibling-subagent pattern the `v0.6.0-rc.7` tagged run's own `H7`
  verification-round recheck documented.
- 2 **new** files under
  `claude/projects/D--ReinstateAcceptanceProjects-v060-rc8-b-projects-claude-*/`
  — real Claude Code sessions created by a different tagged-run executor
  (executor B, per the acceptance run's own lettered-executor layout)
  running its own, unrelated rows concurrently in its own throwaway
  project on this shared host's live Claude Code config; ordinary vendor
  session-creation activity, not `rein push`/`pull`.

None of the 8 entries carries a lab/synthetic fixture name (the pattern
`v0.6.0-rc.7`'s own recheck flagged, e.g. `session-syn-*`) and none
matches this row's own fresh lab account's session ids (there are none —
zero sessions were ever pushed from this row's own profile). `rein daemon
status`'s own reported pull (`"pulled 0 snapshot(s)"`) matches: nothing
was restored onto this host from the round trip's pull.

**Disposition: `PASS`.** All three conditions of the refined per-file-
listing rule hold: the daemon's pull restored **zero** snapshots; **no**
file under the three subtrees was created or modified carrying a
lab/synthetic name or matching this row's own lab account's session ids;
and the **only** 8 differing entries are attributable, by relative path
and by direct correlation with concurrent Claude Code activity on this
shared multi-executor host (this executor's own session, a sibling
subagent's, and a different tagged-run executor's), to a process other
than `rein`. `#424` fired again and is recorded per rule (d), not failed.

---

## `H8` — key-generation floor: the lagging-device attack and the 404-floor proxy

**Mechanism exercised (Go suite):** `keygeneration_crossplane_test.go`,
`-tags hopacceptance`, against a real `hopd` started by the test itself
(`REINSTATE_HOPD_BIN` pointed at this executor's own `hopd.exe`).

```
REINSTATE_HOPD_BIN=<this executor's own hopd.exe> \
  GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0 go test -tags hopacceptance ./internal/cli \
  -run TestKeyGeneration -count=1 -v
```

```
=== RUN   TestKeyGenerationFloorAgainstRealHopd
--- PASS: TestKeyGenerationFloorAgainstRealHopd (1.66s)
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.33s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	3.095s
```

`TestKeyGenerationFloorAgainstRealHopd` is the lagging-device attack: a
device that last read an earlier key generation is refused on `push`,
`pull`, and `sync verify`, each refusal naming "control plane" in its
message. `TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd` is the
404-floor proxy, reproducing the documented residual (a control plane
that carries no floor route lets a device that has confirmed none read a
restored earlier-generation keyring as current) — both required by this
row, both `PASS`.

**Mechanism exercised (live):** `rein devices revoke` on `H6`'s own
device-a, targeting `H6`'s own device-b, against the installed
`v0.6.0-rc.8` binary and this executor's main lab.

```
rein devices revoke <device-b-id> --json     # device-a; recovery code from a local file, in-process, no hidden prompt
```

```
revoked device "Harjots-Beast" (<device-b-id>); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

Device-b afterward, each `--json`, exit `4`:

```
rein whoami --json    -> {"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run `rein login` again",...}
rein push --all --json -> {"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run rein login again",...}
rein pull --all --json -> {"code":"auth_storage","message":"this device's token was rejected by the control plane (revoked or stale); run rein login again",...}
```

**Disposition: `PASS`.** Both cross-plane Go subtests pass against a real
`hopd`, and the live revocation rolled the generation and refused device-
b's token on every subsequent call this row checks (`whoami`, `push`,
`pull`), matching the `v0.6.0-rc.7` tagged run's own `H8` result.

---

## `H8b` — Console-initiated revocation stays pending until confirmed

**Mechanism exercised:** a fresh device pair (device-c enrolls the
account, device-d joins live) in this executor's main lab; a disposable
self-signed-TLS reverse proxy in front of the real `hopd` drives the real
Console HTTP routes end to end (sign-in, session cookie, CSRF-bearing
devices page, revocation POST); the recovery-code command then confirms
the pending request.

**Setup (argv-only):**

```
rein login --email <h8b-lab-email> --no-browser --json   # device-c, device-d
hoplab pair init -root <lab> -device device-c -rein <candidate rein.exe>
hoplab pair join -root <lab> -device device-d -approver device-c -rein <candidate rein.exe>
```

**Console driver, argv-only run and its own printed steps:**

```
consoledriver -hopd 127.0.0.1:8322 -log <lab>\hopd.log -email <h8b-lab-email> -device <device-d-id> -listen 127.0.0.1:8425
```

```
step1: POST /console/sign-in/email -> 200
step2: found link token (redacted length 43)
step3: GET confirm page -> 200
step4: POST confirm -> 200 final url https://127.0.0.1:8425/console
step5: GET /console/devices -> 200
step5: csrf token found, target device present=true
step6: POST https://127.0.0.1:8425/console/devices/<device-d-id>/revocation -> 200 final url https://127.0.0.1:8425/console/devices
step7: GET /console/devices -> 200
step7: device row now shows Pending revocation
```

`device_revocation_requests` (sqlite) after step 7: `status=pending,
requested_generation=0, confirmed_generation=NULL`. Target device-d's
`whoami --json` immediately after: still succeeds, exit `0` — not yet
enforced, pending only.

**Confirmation:**

```
rein devices revoke <device-d-id> --json     # device-c; recovery code from a local file
```

```
Console request <revocation-request-id> confirmed after the keyring reached generation 2
revoked device "Harjots-Beast" (<device-d-id>); key generation 2 started with 1 enrolled device(s), and the control plane refuses its token
```

`device_revocation_requests` after confirmation: `status=confirmed,
confirmed_generation=2, confirmed_by=<device-c-id>`. Device-d's next
`push --all --json`: refused, `"this device's token was rejected by the
control plane (revoked or stale); run rein login again"`, exit `4`.

**Disposition: `PASS`.** The Console-initiated request stayed genuinely
pending (target still authenticated) until the recovery-code command
wrote a strictly newer generation (1→2) and confirmed that exact request
row, at which point enforcement took effect — matching the `v0.6.0-rc.7`
tagged run's own `H8b` result and request-confirmation design.

---

## Cleanup

Both lab instances (`hopd`/`fakelocker` on `8322`/`9322` and, earlier,
`8324`/`9324`) were stopped via `hoplab stop` after their rows finished.
The two disposable Go helpers
(`scripts/testing/devicerevoke`, `scripts/testing/consoledriver`) were
deleted from the worktree before this report was committed; `git status`
in the worktree shows no changes from this part beyond this results file
(two unrelated untracked directories from a different concurrent
executor's own work in this shared worktree, `cmd/labrunfd/` and
`cmd/labverify/`, were left untouched and are not part of this
commit). No recovery code, root/device private key, keyring signature
bytes, or other key material was printed anywhere in this report; account
and device UUIDs named above were minted solely for this run's own
disposable lab accounts and are not credentials. No transcript text,
prompt, or session content from any Claude Code, Codex, or OpenCode
session — this executor's own, a sibling subagent's, or another
executor's — is quoted anywhere in this report.
