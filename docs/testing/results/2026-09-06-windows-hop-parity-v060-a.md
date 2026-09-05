# Hop parity journeys, native Windows — W6 executor A (H1–H5, H9–H12), 2026-09-06

Physical, single-host journeys for hosted ticket #16 rows H1–H5 and H9–H12
against a local, disposable lab: a real `hopd` (private control plane) plus
`scripts/testing/fakelocker` standing in for the bucket, both on loopback
with fake storage and a log-only email sender, per
[windows-acceptance-host.md](../windows-acceptance-host.md)'s Hop lab
section and `scripts/testing/hoplab/README.md`. Executor B ran H6–H8 in a
separate worktree and lab root concurrently
(`docs/testing/results/2026-09-06-windows-hop-parity-v060-b.md`); the two
labs never shared a port, a `-root`, or a file.

Contract: [`docs/testing/v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D. Task card:
[`docs/planning/v0.6.0-hop/task-cards/W6-hop-parity.md`](../../planning/v0.6.0-hop/task-cards/W6-hop-parity.md).

Every command below is real; only the recovery code, the device token, and
the resume dry-run's project path are redacted (`<recovery-code>`,
`<token>`, `[REDACTED_PATH]`). Lab-root paths under
`D:\ReinstateAcceptanceProjects\v060-hop-a\` are not private and are shown
in full, matching the shape of every prior Hop lab report in this
directory.

## Verdict

- **Required rows run:** 11 of 11 assigned (H1, H1b, H1c, H2, H3, H4, H5,
  H9, H10, H11, H12).
- **PASS:** H1, H1b, H1c, H2, H3, H4, H9, H10, H11, H12 (10 rows).
- **PARTIAL:** H5 — the wipe, new-device sign-in, recovery, pull, and
  dry-run-resume mechanism are fully evidenced and correct; the
  "one real resume through conptydriver, answering from history" evidence
  item could not be produced in this environment (see §5 and §7, finding
  F1). Per the evidence rule, PARTIAL does not pass a required row; H5 is
  recorded PARTIAL rather than claimed PASS.
- **Product defects found:** none. No fix commit accompanies this report.
- **Release-blocking findings:** 0. Three non-blocking findings are
  recorded in §7 for the coordinator's attention (F1 auth gap, F2 range
  widening not yet on this branch, F3 `fakelocker` reference-locker
  isolation).

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-05 evening through 2026-09-06 (IST `Asia/Kolkata`, UTC+05:30; timestamps below are as printed by each tool) |
| Host | Windows 11 Pro `10.0.26200`, native `windows-amd64` (not WSL) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-w6-hop-parity`, branch `v060/w6-hop-parity` |
| Tested commit | `b6ed9dc54acb4ccf2719d3a1aa2ce7ce053ca673` (`release/v0.6.0-rc.1` tip at branch time; `git log -1 --format=%H` on this worktree) |
| Client build | `go build -o bin/rein.exe ./cmd/reinstate` (also `bin/reinstate.exe`), `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13`; `rein version --json` reports `v0.5.2-rc.1-185-gb6ed9dc5`, commit `b6ed9dc5...ca673` |
| Go | `go1.25.13 windows/amd64` (`GOTOOLCHAIN=go1.25.13`) |
| hopd | Prebuilt binary at a scratch path (`REINSTATE_HOPD_BIN`); private `reinstate-hosted` control plane, fake storage (`HOPD_STORAGE=fake`), log email sender |
| fakelocker | `scripts/testing/fakelocker`, in-memory fake S3, `AnyBucket=true`, accepting `FAKEKEY*` access key ids |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-hop-a\`, `hopd` on `127.0.0.1:8301`, locker on `127.0.0.1:9301` (executor B: `8302`/`9302`) |
| Agents on this host | `claude 2.1.261`, `codex-cli 0.149.0` (`codex.exe` from `C:\Users\admin\AppData\Local\Programs\OpenAI\Codex\bin`), `opencode 1.18.27`, all on `PATH` |
| Contaminating env vars | `REINSTATE_BACKEND`, `REINSTATE_MEMORY_BACKEND_DIR`, `XDG_DATA_HOME` (and, observed separately, ambient `CLAUDE_CONFIG_DIR`/`CODEX_HOME` pointed at an unrelated prior lab tree) unset in every shell before running `rein`, `hoplab`, or `go test`, per the task's ground rules |
| Isolation | `hoplab homes`/`hoplab env` gave each simulated device its own `REINSTATE_HOME`, `HOME`, `USERPROFILE`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME`; H12's three extra devices (A/B/C) used hand-built isolated homes with a disk-backed `REINSTATE_BACKEND=memory` locker in place of the HTTP fake, the same stand-in `internal/cli/keygeneration_crossplane_test.go` uses for its cross-plane journeys |
| Secrets | Recovery codes and pairing/recovery confirmations were fed through `REINSTATE_RECOVERY_CODE_FD`/`REINSTATE_PAIRING_CODE_FD` via real Windows-inheritable pipe/file handles (`hoplab pair`'s own mechanism, and two small throwaway Go runners built the same way for the handful of commands `hoplab` does not wrap — deleted after use, never committed; see §6) |
| Cleanup | `hoplab stop -root D:\ReinstateAcceptanceProjects\v060-hop-a` run at the end of this session; `hopd.db`, `hopd.log`, and every device's `reinstate`/`home` tree left under the lab root for inspection |

## 2. Section D — Hop parity journeys (H1–H5, H9–H12)

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| H1 | Email sign-in through the approver; token in OS keyring; `whoami` names the device | `PASS` | §3.1 |
| H1b | Refused sign-in tells the terminal why and stores nothing | `PASS` | §3.2 |
| H1c | Unreachable control plane: one-line message, documented exit code | `PASS` | §3.3 |
| H2 | `init --hop` provisions the locker exactly once; a second run does not re-provision | `PASS` | §3.4 |
| H3 | `account init` shows the recovery code once; `keyring.v1.json` format 5, signed generation 1, verifies with `keyring.Parse` | `PASS` | §3.5 |
| H4 | `push --all` sends one Claude, one Codex, one OpenCode session; `first_push` reaches the control plane exactly once; `hop status` shows it; no-op push reports nothing | `PASS` (see F3 footnote on the post-push auto-verification) | §3.6 |
| H5 | Wipe home/stores/token/key; sign in as a new device; `account recover`; `pull --all`; `resume --dry-run` complete for all three; one real resume through conptydriver | `PARTIAL` — recovery/pull/dry-run mechanism PASS; live-resume evidence item not produced (F1) | §3.7 |
| H9 | `sync verify` human and `--json` name only observed objects; 404-floor proxy journey reproduces the documented residual | `PASS` (step 4 of the human/JSON report fails on this lab for harness reasons, not product ones — F3) | §3.8 |
| H10 | `sync migrate --to byo` to a second `fakelocker` bucket; `--switch`; `--forget-hop` | `PASS` | §3.9 |
| H11 | Path remap between two homes' project mappings (Windows→Windows); pulled session carries the other home's path; workspace verification passes on resume | `PASS` (macOS leg E-deferred, unchanged) | §3.10 |
| H12 | Keyless diagnostics refuse a forged, rolled-back, or re-keyed keyring, nothing written | `PASS` | §3.11 |

`PARTIAL` and `NOT TESTED` do not pass a required row (evidence policy); H5
is recorded `PARTIAL`, not `PASS`, for exactly that reason.

## 3. Journeys in detail

### 3.1 H1 — email sign-in through the approver

```text
$ ./scripts/testing/hoplab/hoplab.sh approve -root D:/ReinstateAcceptanceProjects/v060-hop-a -email device-a@example.test -count 1 -timeout 60s &
$ ./bin/rein login --email device-a@example.test --no-browser
A sign-in link was sent to device-a@example.test. Open it on any device to approve this one.
Waiting for approval (expires 2026-09-05T20:37:04Z; Ctrl-C to cancel)...
Signed in to Reinstate Hop as device-a@example.test.
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
exit=0
hoplab approve: device "Harjots-Beast" (device-a@example.test): approved

$ ./bin/rein whoami
Account: device-a@example.test
Plan:    hop (locker location apac)
Device:  Harjots-Beast (windows-amd64, enrolled 2026-09-05T20:27:04Z)
Hop:     http://127.0.0.1:8301
exit=0
```

`approve` performed the real GET-then-POST a person clicking the emailed
link makes (`hopd.log` shows the mailed magic link and the confirming
POST). The device token is genuinely in the OS keyring, not merely
in-process state: `hoplab keyring show -device device-a` (run later, before
the H5 wipe) reported `OS-keyring entry "hop/device-token@2767d451f0df677d"`
and `device token present: control_plane_url=http://127.0.0.1:8301
account_id=0574a7de-1d57-4843-b221-49554e10bcdd
device_id=f849e6e7-4c85-4968-b2ee-71db322c51a3` — the exact account/device
this login produced, read back out of Windows Credential Manager by a
fresh, separate process (`hoplab`, not `rein`). A plain `cmdkey /list`
independently shows dozens of `reinstate:*` `LegacyGeneric` entries on this
host from earlier, unrelated lab sessions (other workstreams, prior
release cycles); none were touched.

### 3.2 H1b — refused sign-in

The ordinary expired-link refusal, with `HOPD_LOGIN_SESSION_TTL=8s` on a
throwaway restart of the same lab (no accounts existed on it yet, so the
restart cost nothing):

```text
$ HOPD_LOGIN_SESSION_TTL=8s ./scripts/testing/hoplab/hoplab.sh start -root D:/ReinstateAcceptanceProjects/v060-hop-a -hopd-addr 127.0.0.1:8301 -locker-addr 127.0.0.1:9301 -background

$ ./scripts/testing/hoplab/hoplab.sh approve -root D:/ReinstateAcceptanceProjects/v060-hop-a -email h1b-refused@example.test -refuse -count 1 -timeout 30s &
$ ./bin/rein login --email h1b-refused@example.test --no-browser --json
EXIT=1
{
  "code": "runtime",
  "message": "the sign-in link expired before it was used; run login again",
  "safe_to_retry": false
}
hoplab approve: device "Harjots-Beast" (h1b-refused@example.test): declined (link seen, approval withheld; it will expire on its own)
```

`-refuse` GETs the confirm page (the link is genuinely seen) but never
POSTs, so the session expires on its own after the 8 s TTL and `rein
login`'s poll surfaces the expiry rather than waiting out a longer clock.
Nothing was stored: `REINSTATE_HOME` for this attempt
(`D:\ReinstateAcceptanceProjects\v060-hop-a\h1b\reinstate`) is an empty
directory afterward (`ls -la` shows only `.`/`..`), so no `config.toml`,
`state.json`, or `account.json` was ever written, matching a refusal that
stops before any local state is touched.

Lab restarted afterward with the default TTL for every later row.

### 3.3 H1c — unreachable control plane

```text
$ REINSTATE_HOME=... REINSTATE_HOP_URL="http://127.0.0.1:1" ./bin/rein login --email h1c-probe@example.test --no-browser
could not reach the Reinstate Hop control plane at http://127.0.0.1:1: connection refused
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.
EXIT=1

$ ... --json
{
  "code": "runtime",
  "message": "could not reach the Reinstate Hop control plane at http://127.0.0.1:1: connection refused\n...",
  "details": { "kind": "control_plane_unreachable", "url": "http://127.0.0.1:1" },
  "safe_to_retry": false
}
EXIT=1
```

Matches `docs/hop.md`'s "When it cannot be reached" section verbatim
(message text, exit code `1`, `--json details.kind`). Nothing was written
(`REINSTATE_HOME` never existed after the attempt).

### 3.4 H2 — `init --hop` provisions once

```text
$ ./bin/rein init --hop --project "local/device-a=D:\ReinstateAcceptanceProjects\v060-hop-a\device-a\project"
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-wgyh1hw1x347wkaxyggh5e45a4 at http://127.0.0.1:9301 (location apac, plan hop)
profile_id=0574a7de-1d57-4843-b221-49554e10bcdd device_id=f849e6e7-4c85-4968-b2ee-71db322c51a3
exit=0

$ ./bin/rein init --hop --project "local/device-a=D:\ReinstateAcceptanceProjects\v060-hop-a\device-a\project"
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
exit=7
```

`sqlite3 hopd.db "select type, count(*) from events group by type"` after
both calls: `locker_provisioned|1` — one provisioning event no matter how
many times `init --hop` is attempted, because the second attempt is
refused client-side before any request is made.

### 3.5 H3 — `account init`: recovery code once, format 5, signed generation 1

```text
$ ./bin/rein account init
Your recovery code (shown once, never stored anywhere):

    <recovery-code>

account initialized: root key generated on this device, keyring written to storage
profile_id=0574a7de-1d57-4843-b221-49554e10bcdd device_id=f849e6e7-4c85-4968-b2ee-71db322c51a3 key_generation=1 devices=1
```

A small throwaway Go program (`cmd/h3probe`, built and deleted the same
session — see §6) fetched `keyring.v1.json` from the real locker using the
scoped, expiring credentials `rein hop credentials --export` printed, and
ran the bytes through the real `internal/keyring.Parse` +
`(*Keyring).VerifyGenerations`:

```text
schema_version=5 current_generation=1 profile_id=0574a7de-1d57-4843-b221-49554e10bcdd account_key_len=44 devices=1
VerifyGenerations: OK (every generation's signature verifies under the published account key)
```

This is the exact mechanism the acceptance row names: the object parses as
schema version 5 (the only version this build reads), is at generation 1,
and its signature verifies under the published account key — not merely
that the CLI claims so, but that the product's own verification function
agrees when run directly against the real bytes in the real locker.

### 3.6 H4 — first push of one Claude, one Codex, one OpenCode session

Sessions: the three synthetic, vendor-format-authentic fixtures
`hoplab homes` seeds per device from `testdata/adapters/{claude,codex,opencode}/windows`
(one Claude Code `.jsonl`, one Codex rollout `.jsonl`, one OpenCode
`opencode.db` row), under device-a's isolated agent-store roots. This is
the same fixture shape `docs/testing/results/2026-08-24-first-push-windows.md`
used for this exact row ("planted" content, real vendor file formats, not
a live interactive vendor session — see §6 for why a live-created session
was not used here).

```text
$ ./bin/rein push --all --json
... (verification steps 1-3 PASS, step 4 FAIL -- see the footnote below and F3)
{
  "dry_run": false, "skipped": 0,
  "snapshots": ["1bdc2f55-...", "b2f5c51c-...", "7be0025b-..."]
}

$ sqlite3 hopd.db "select type, count(*) from events group by type"
device_enrolled|1
first_push|1
locker_provisioned|1
sign_up|1
verify_reported|1

$ ./bin/rein hop status
...
Created:  2026-09-05T20:27:43Z; first push: 2026-09-05T20:29:56Z

$ ./bin/rein push --all --json
{ "dry_run": false, "skipped": 3, "snapshots": null }
```

`first_push|1` after the push and unchanged after every later command in
this session (H5's post-recovery push included) — exactly once. `hop
status` shows the first-push time. A second, no-op push skips all three
and reports nothing pushed.

**Footnote (harness, not product):** `push --all`'s own post-push
verification printed `OUTCOME: FAIL` because step 4 (the isolation probe)
found that this account's credentials could list the control plane's
reference-locker bucket (returning 0 objects) instead of being refused
`AccessDenied`. `scripts/testing/fakelocker` (`s3test.Fake{AnyBucket:
true}`) serves any bucket name it is asked for without real per-credential
bucket-scoped access control, so this is the fake locker's known
limitation, not a defect in `rein sync verify`'s logic — see F3 and §3.8's
contrast with the same check on a real BYO destination in H10, where it
correctly reports `NOT APPLICABLE` instead of failing.

### 3.7 H5 — wipe, new device, recover, pull, dry-run, real resume

**Wipe.** `REINSTATE_HOME` and the isolated `home` tree (Claude/Codex/
OpenCode stores) removed (`rm -rf`); the device token
(`hop/device-token@2767d451f0df677d`) and the device key
(`reinstate/0574a7de.../device/f849e6e7...`) deleted from Windows
Credential Manager with `cmdkey /delete` (via PowerShell, to avoid Git
Bash's own path-mangling of a leading `/` argument, which is otherwise
silently misinterpreted — see §6). Confirmed:

```text
$ ./bin/rein whoami
this device is not signed in to Reinstate Hop; run `rein login`
exit=4
$ ./bin/rein account status
config missing
exit=3
```

**Sign in as a new device**, same email, approved the same way as H1; a
new `device_id` is issued (`f849e6e7-...` → `2e3ccce8-...`).

**`account recover`** (recovery code fed through
`REINSTATE_RECOVERY_CODE_FD` via a plain, already-known-secret temp file —
the same technique `hoplab pair recover` uses):

```text
device enrolled from the recovery code; this device now reads everything written under key generation 1
profile_id=0574a7de-1d57-4843-b221-49554e10bcdd device_id=2e3ccce8-a13a-4aae-83fd-22929a0cd332 key_generation=1 devices=2
```

`account status --json`: `"enrolled_via": "recover", "recovery_code_confirmed": true, "key_generation": 1, "enrolled_devices": 2`.

**`pull --all`** refused first, naming the one genuinely-missing agent
store and keeping what it had already restored:

```text
opencode session ses_fixture001a: compatibility NOT_INSTALLED refuses restore; install and run opencode once on this device so its session layout exists, then pull again
exit=5
```

Claude and Codex sessions were already restored at that point (their
`.jsonl` files existed under the isolated store); running `opencode --pure
db "select 1"` once (per the product's own documented "install and run the
agent once" step) created OpenCode's real schema, and a second `pull
--all` restored all three with `"pulled": 3, "skipped": 0, "conflicts": null`.

**A path-mapping correction, in passing.** `hoplab homes`' fixture project
path (`C:\Users\fixture-user\code\device-a`) is synthetic and does not
exist as a real directory (permission to create it under the shared
`C:\Users\fixture-user\code\` tree was refused — `BUILTIN\Users` only has
`ReadAndExecute` there), so the resumed sessions' recorded workspace was
initially reported missing. `[[projects]]`'s `local_root` in
`config.toml` was edited to a real, existing directory
(`D:\ReinstateAcceptanceProjects\v060-hop-a\device-a\project`), and the
sessions were re-pulled (deleting the locally-restored copies first, so
the remap applies): the restored `cwd` changed from the fixture path to
the real one. This is the exact mechanism H11 names, exercised here
incidentally and again, formally, with two devices in §3.10.

**`resume --dry-run` for all three**, after the remap:

```text
codex ["resume" "rollout-syn-001-a"] in [REDACTED_PATH]
Environment decision: confirmation_required
Environment check: workspace.available status=present ...
Environment check: agent.executable status=present ...
Environment check: agent.layout status=match ... Actual: "sessions-rollout-jsonl"
Environment check: agent.version status=match ... Actual: "0.149.0"
exit=0
```

Codex (`0.149.0`, inside this build's verified `0.149.0` range) reached
`confirmation_required` cleanly: workspace present, layout recognized,
version in range, executable present, no running instance. Claude
(`2.1.261`) and OpenCode (`1.18.27`) both reached workspace/layout/
executable checks correctly (`present`/`match` throughout) but the overall
decision was `blocked`, exit `5`, solely on:

```text
Environment check: agent.version status=unknown severity=block ... native agent version 2.1.261 is outside the verified range 2.1.219 to 2.1.238 inclusive
Environment check: agent.version status=unknown severity=block ... native agent version 1.18.27 is outside the verified range 1.18.21 to 1.18.21 inclusive
```

`internal/agents/catalog/claude.go` and `opencode.go` on this branch still
carry the pre-widening ranges (`2.1.219`–`2.1.238`, `1.18.21`); ADR 0005 D3
and the `v0.6.0-hop` plan name widening through exactly these installed
versions as W3's row, on evidence from native Windows resume attempts —
this is that evidence, and F2 records it for the coordinator rather than
being fixed here (`internal/agents/catalog/*` version constants are W3's
file per `file-ownership.md`). The dry-run mechanism itself — reading the
restored, remapped session, resolving the vendor executable, recognizing
the layout — is correct and complete for all three; only the range gate,
not yet updated on this branch, blocks two of the three decisions.

**One real resume through conptydriver, answering from history: not
produced.** See finding F1 (§7) for the full account. In summary: every
session pushed and pulled in this journey lives under an isolated
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` (mandatory — an
unisolated agent store would enumerate the operator's real sessions into
`push --all`, which AGENTS.md and this task's ground rules both forbid).
Every one of the three vendor CLIs on this host is authenticated only in
its real, ambient default config location (itself found to be
redirected, on this account, to an unrelated prior lab's tree —
`CLAUDE_CONFIG_DIR=D:\Projects\hop-10-lab\claude`,
`XDG_DATA_HOME=D:\Projects\hop-10-lab\xdg` — which `CLAUDE.md`'s
Claude-Code-specific note separately forbids inspecting or using). A fresh,
isolated config directory for any of the three requires a live OAuth (or
ChatGPT) sign-in this session has no credential or browser session to
complete non-interactively; confirmed directly rather than assumed:
`claude -p "..."` under a brand-new, isolated `CLAUDE_CONFIG_DIR` printed
`Not logged in · Please run /login`, and `codex login status` under a
brand-new, isolated `CODEX_HOME` printed `Not logged in`. No
`ANTHROPIC_API_KEY`/`OPENAI_API_KEY` or equivalent is present in this
environment. This is recorded as PARTIAL rather than invented.

### 3.8 H9 — `sync verify`, human and `--json`; the 404-floor proxy journey

Human and `--json` reports for device-a's Hop locker both name exactly the
real objects present and nothing else:

```text
Step 1: ... 5 object(s): manifest.age; keyring.v1.json; 3 snapshot(s) ...
Step 2: manifest.age (922 bytes): begins with the age v1 header ...
Step 3: index revision 7be0025b-..., 3 session(s) (claude 1, codex 1, opencode 1)
        index entry claude:session-syn-001-a -> snapshots/1bdc2f55-....age
        ...
"unopened": "Not opened and judged by name only: 2 other age-named snapshot(s), the wrapped keyring."
Step 4: Listing the reference locker SUCCEEDED and returned 0 object(s). ... Result: FAIL
OUTCOME: FAIL.
```

Steps 1–3 report only observed objects, by their real opaque ids, with the
keyring correctly named as unopened rather than examined — matching the
row's own wording exactly. Step 4 fails on this lab for the harness reason
recorded as F3: `fakelocker`'s `AnyBucket` does not refuse a foreign
bucket. The same check on a genuine BYO destination (H10, §3.9) correctly
reports `NOT APPLICABLE` rather than `FAIL` when there is no control plane
to name a reference locker at all, which is the contrasting evidence that
this is the lab, not the product.

**404-floor proxy journey**, against the real `hopd` binary
(`-tags hopacceptance`, `REINSTATE_HOPD_BIN`):

```text
$ REINSTATE_HOPD_BIN=<hopd.exe path> go test -tags hopacceptance ./internal/cli -run TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd -count=1 -v
=== RUN   TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.53s)
```

This is the test named in the row and in `docs/hop.md`'s residual
discussion, run against a real `hopd` behind a real reverse proxy that
answers the two floor routes `404` (a control plane predating them): the
lagging device (which never confirmed a floor) accepts the restored
generation-1 keyring, exactly as documented, while the revoking device's
own local anchor still refuses a push against the rolled-back object.

### 3.9 H10 — `sync migrate --to byo`, `--switch`, `--forget-hop`

Migrated device-a's Hop locker to a second `fakelocker` bucket
(`device-a-byo-migrated`, distinct `FAKEKEY9999` credentials at the same
endpoint — a fresh bucket, not the account's own), with the passphrase fed
through `REINSTATE_PASSPHRASE_FD` (see §6):

```text
{
  "destination": {"bucket": "device-a-byo-migrated", "endpoint": "http://127.0.0.1:9301", ...},
  "forgot_hop": true,
  "migrated": {"snapshots": 3, "written": 3, "verified": 0, "skipped": 0, "bytes": 10218, "manifest_sessions": 3},
  "switched": true
}

$ ./bin/rein whoami
this device is not signed in to Reinstate Hop; run `rein login`
exit=4
```

`sqlite3 hopd.db "select type, count(*) from events group by type"` after
`--forget-hop`: `device_enrolled|2, first_push|1, locker_provisioned|1,
sign_up|1, verify_reported|2` — no `device_revoked` event anywhere; the
device is still enrolled at the control plane, matching the documented
"does not revoke" claim exactly.

`sync verify` against the new BYO bucket (passphrase again through the
FD): steps 1–3 PASS with the same three sessions readable under
`age-scrypt` encryption, and step 4 reports:

```text
Step 4: What was done: Nothing. This profile stores to a bucket you configured yourself, so there is no control plane to name a reference locker...
Result: NOT APPLICABLE
OUTCOME: PASS.
```

This is the contrast that pins F3: the same isolation check, on a profile
with no control plane, degrades to `NOT APPLICABLE` exactly as documented,
rather than failing — confirming the Hop-side FAIL in H4/H9 is the fake
locker's limitation, not the check's.

### 3.10 H11 — path remap between two homes' project mappings

A fresh Hop account (`h11-pathremap@example.test`); device-a initialized
first with project mapping `hoplab-device-a=D:\...\device-a\project` and
pushed its three sessions (recorded `project hoplab-device-a`, `cwd`
device-a's real path). Device-b joined the **same account** via the
recovery code, with `config.toml`'s `[[projects]]` entry set to the
**same project id** (`hoplab-device-a`) but a **different** real local
path (`D:\ReinstateAcceptanceProjects\v060-hop-a\device-b\altproject`).

```text
$ ./bin/rein pull --all --json   # as device-b
{
  "plans": [
    {"agent": "claude", "destinations": [".../device-b/home/.claude/projects/D--ReinstateAcceptanceProjects-v060-hop-a-device-b-altproject/session-syn-001-a.jsonl"]},
    {"agent": "codex", "destinations": [".../device-b/home/.codex/sessions/rollout-syn-001-a.jsonl"]},
    {"agent": "opencode", ...}
  ],
  "pulled": 3
}

$ grep -o '"cwd":"[^"]*"' .../session-syn-001-a.jsonl   # claude, on device-b
"cwd":"D:\\ReinstateAcceptanceProjects\\v060-hop-a\\device-b\\altproject"
$ grep -o '"cwd":"[^"]*"' .../rollout-syn-001-a.jsonl   # codex, on device-b
"cwd":"D:\\ReinstateAcceptanceProjects\\v060-hop-a\\device-b\\altproject"
```

The session was pushed from device-a's own real path and pulled onto
device-b's distinct real path for the same project id — the pathmap
rewrite, exercised across two genuinely different devices sharing one
account, not merely two directories on one config. `resume --dry-run
codex:rollout-syn-001-a` on device-b then reports
`workspace.available status=present ... Actual: true` and
`Environment decision: confirmation_required`, `exit=0`: workspace
verification passes on the remapped path. The macOS half (Windows↔macOS
path remap) remains E-deferred, unchanged from the contract.

### 3.11 H12 — keyless diagnostics refuse a forged, rolled-back, or re-keyed keyring

A real Hop account (`h12-keyless@example.test`) with three real enrolled
devices (A, B, C), using a disk-backed `REINSTATE_BACKEND=memory` locker
in place of the HTTP fake — the same stand-in
`keygeneration_crossplane_test.go` uses for a bucket a revoked device can
still write to directly, chosen here so the keyring object on disk could
be read, copied, and rewritten exactly like the unit tests' own
`lockerKeyringObject`/`restoreKeyring` helpers do, but against the real
compiled binary rather than the in-process harness. Device A initialized
and pushed (generation 1); B and C joined via the recovery code; C ran one
ordinary command (`account status`) to record its own anchor at generation
1 before going quiet; A then revoked B for real (`rein devices revoke`,
recovery code via FD), rolling the keyring to generation 2 and raising the
control-plane floor to 2.

**Forged** (the current generation's `signature` field overwritten with
86 `A` characters, matching `internal/keyring/generation_test.go`'s own
`forgeGeneration` shape):

```text
$ ./bin/rein account status --json    # device A
"keyring_present": false, "keyring_refused": true, "key_generation": 2,
"error": "keyring: key generation is not signed by this account's key: generation 1 does not verify under account key plu9oh6JT+gUsrO8X5f000xiL46Xb9puOoeFzM31o1Q="
exit=0
$ ./bin/rein devices --json           # same object, same error in keyring_error
exit=0
```

**Rolled back** (the genuine, correctly-signed generation-1 object,
copied aside before A's revoke, written back over the current
generation-2 object — the exact lagging-device attack
`keygeneration_crossplane_test.go` runs against a real `hopd`, here run
against a device that already held the anchor locally):

```text
$ ./bin/rein account status --json    # device C, which recorded generation 1 and last saw floor 0
"keyring_refused": true, "key_generation": 1,
"error": "keyring current_generation 1 is below the 2 the control plane reports for this account (as of 2026-09-05T20:41:34Z); the keyring was rolled back (a revoked device may have restored an older copy inside its credential window). Nothing was written; restore the account's keyring from a device that holds key generation 2, or run rein devices revoke again from one that does"
exit=0
$ ./bin/rein devices --json           # same keyring_error
exit=0
```

**Re-keyed** (a completely separate real Hop account's own genuine,
validly-signed `keyring.v1.json` substituted in place of the first
account's object):

```text
$ ./bin/rein account status --json    # device A, original account
"keyring_present": true, "keyring_refused": true,
"error": "keyring: the keyring is signed by a different account key: it is signed by account key VFLvwVLPiK2eMUFwqss0caWVYOS+GUSIkGRLddQES24=, not the plu9oh6JT+gUsrO8X5f000xiL46Xb9puOoeFzM31o1Q= expected here"
exit=0
$ ./bin/rein devices --json
exit=0
```

**Nothing was written**, verified directly rather than assumed: device
A's `account.json` after all three attempts still reads
`"key_generation": 2, "account_key": "plu9oh6JT+gUsrO8X5f000xiL46Xb9puOoeFzM31o1Q=", "control_plane_key_generation": 2`
— its genuine anchor, from before any of the three corruptions, untouched
by any of the three refused reads. And a write-path command against the
same re-keyed object refuses too, not only the two diagnostics:

```text
$ ./bin/rein push --all
keyring: the keyring is signed by a different account key: ... Nothing was written; the keyring in storage was replaced by one signed with a key this account never used
exit=7
```

All three scenarios: `account status` and `devices` are diagnostics (exit
`0`, reporting the refusal rather than becoming one); a write-path command
(`push`) exits `7` (`ExitSafety`); nothing was written to local state in
any case.

## 4. Deferred (section E), unchanged

Every row this executor's assignment touches that also has a macOS half
(H11's Windows↔macOS leg) is carried in section E of the acceptance
contract, `DEFERRED`, and is not claimed here. No document, data file, or
message from this run implies macOS evidence.

## 5. Gates

```text
$ gofmt -l .                                          # empty
$ GOTOOLCHAIN=go1.25.13 go vet ./...                   # clean
$ GOTOOLCHAIN=go1.25.13 go mod tidy -diff              # empty
$ CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test ./... -count=1
$ CGO_ENABLED=1 GOTOOLCHAIN=go1.25.13 go test -race ./internal/... -count=1
```

Results recorded in the structured report (`gates` field); this workstream
made no product-code changes, so these are a baseline confirmation on the
branch tip this report names, not a gate on a diff.

## 6. Method notes

- **Session content for H4/H5/H9/H10/H11** is the vendor-format-authentic,
  synthetic fixture `hoplab homes` seeds from `testdata/adapters/*`, the
  same shape the accepted `2026-08-24-first-push-windows.md` precedent
  used for this exact ticket. A live, interactively-created vendor session
  was not used for the push/pull/dry-run content, because doing so would
  need either the operator's real, ambient agent config (forbidden — see
  `CLAUDE.md` and finding F1) or a fresh isolated login this session
  cannot complete. Where the row specifically needed a live vendor
  process, that is stated plainly rather than substituted (F1).
- **Two small throwaway Go programs** (`cmd/h3probe`, and a generalised
  FD-runner used for H10's passphrase, H11's recovery code, and H12's
  account-init/-recover/-revoke confirmations) were built in this worktree
  to drive `REINSTATE_PASSPHRASE_FD`/`REINSTATE_RECOVERY_CODE_FD` across a
  real Windows process boundary for the handful of commands
  `scripts/testing/hoplab`'s own `pair` subcommand does not wrap (`sync
  migrate`, a custom project-mapping `account recover`, and `devices
  revoke`/repeated `account init` inside one lab). Both use the identical
  `AdditionalInheritedHandles` technique `scripts/testing/hoplab/secretfd_windows.go`
  already documents and tests; neither reimplements product logic. Both
  were deleted from the worktree before this commit — `git status` at the
  time of writing this report shows no diff outside this file.
- **`cmdkey /list`/`/delete` must be run from PowerShell, not Git Bash**,
  on this host: Git Bash rewrites a bare leading `/list`/`/delete:...`
  argument as a path before `cmdkey.exe` ever sees it, producing "The
  command line parameters are incorrect" instead of the real listing —
  confirmed directly (identical command, two shells, two outcomes). Every
  Credential Manager check and delete in this report used PowerShell.
- **`opencode`'s SQLite store is not created merely by installing it**;
  `opencode --pure db "select 1"` (or any query) against the target
  `XDG_DATA_HOME` is what materializes `opencode.db` with its real schema,
  matching the product's own "install and run it once" pull-refusal
  message.

## 7. Findings for the coordinator

**F1 — no OAuth-capable auth path for a fresh, isolated agent config
directory.** `AGENTS.md`/this task's ground rules require every pushed
session to live under an isolated `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/
`XDG_DATA_HOME` so `push --all` never enumerates the operator's real agent
history. A brand-new directory for any of Claude Code, Codex CLI, or
OpenCode has no stored credential and needs a live sign-in (OAuth or
ChatGPT); this sandboxed session has no browser session or API key that
can complete one non-interactively, and this host's *ambient default*
config locations are themselves redirected to an unrelated prior lab tree
(`CLAUDE_CONFIG_DIR=D:\Projects\hop-10-lab\claude`,
`CODEX_HOME=C:\...\orca\codex-runtime-home\home`,
`XDG_DATA_HOME=D:\Projects\hop-10-lab\xdg`) that `CLAUDE.md` separately
forbids using. Confirmed rather than assumed (§3.7). **Question:** is a
lab-scoped API key (or a documented human-in-the-loop browser step) meant
to be provided for this specific piece of evidence in an unattended agent
run, and if so, where should it come from? Until then, "one real resume
through conptydriver, answering from history" (H5) cannot be produced by
an autonomous executor in this environment, on this host, without either
violating the isolation rule or using state this task forbids touching.

**F2 — verified ranges not yet widened on this branch.** `internal/agents/catalog/claude.go`
(`2.1.219`–`2.1.238`) and `opencode.go` (`1.18.21`) do not yet include this
host's installed `2.1.261`/`1.18.27`, so `resume --dry-run` correctly
blocks on `agent.version` for those two agents (§3.7) — the exact,
expected ADR 0005 D3 situation, and W3's file per `file-ownership.md`, not
touched here. Recommend re-running H5's dry-run row after W3's range
widening lands on this branch; the workspace/layout/executable mechanics
this row is actually about are already correct for all three agents.

**F3 — `fakelocker`'s `AnyBucket` does not emulate per-bucket credential
scoping.** `rein sync verify` step 4 (and `push`'s own post-push
verification) fails against every Hop-mode locker in this lab because the
fake locker serves the control plane's reference-locker bucket instead of
refusing this account's credentials with `AccessDenied` — confirmed as
the lab, not the product, by the same check correctly reporting `NOT
APPLICABLE` on a BYO destination with no control plane at all (§3.9).
`scripts/testing/fakelocker` is W4-owned; flagged for awareness, not fixed
here.

**F4 — host hygiene, not a defect.** Windows Credential Manager on this
account carries several dozen `reinstate:*` entries from earlier,
unrelated lab sessions across prior release cycles. None were touched
(not mine to clean, and executor B's lab was running concurrently); a
periodic sweep between release cycles may be worth scheduling.

No product defect was found or fixed in this workstream; `product_defects`
is empty in the structured report.
