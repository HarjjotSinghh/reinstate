# Hop parity journeys, native Windows, tagged artifact — W7 executor D (H1–H5, H9–H12), 2026-09-06

Physical, single-host journeys for hosted ticket #16 rows H1–H5 and H9–H12
against a local, disposable lab: a real `hopd` (private control plane) plus
`scripts/testing/fakelocker` standing in for the bucket, both on loopback
with fake storage and a log-only email sender, per
[windows-acceptance-host.md](../windows-acceptance-host.md)'s Hop lab
section and `scripts/testing/hoplab/README.md`. This run is against the
**published, signed `v0.6.0-rc.1` GitHub prerelease**, not a pre-tag
snapshot — the first time these eleven rows run against a tagged artifact
rather than a working tree, per
[`v0.6.0-rc.1-agent-verification-prompts.md`](../v0.6.0-rc.1-agent-verification-prompts.md).

Contract: [`docs/testing/v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D. Dispatch:
[`docs/testing/v0.6.0-rc.1-agent-verification-prompts.md`](../v0.6.0-rc.1-agent-verification-prompts.md).
Reused method and dispositions from the pre-tag Hop-parity precedent for
these same eleven rows,
[`2026-09-06-windows-hop-parity-v060-a.md`](2026-09-06-windows-hop-parity-v060-a.md)
(W6 executor A); every row that pre-tag report recorded `PASS` is
re-verified `PASS` here on the tagged binary, and the one row it recorded
`PARTIAL` (H5, live-resume evidence) now clears to `PASS` — see §7 for why.

**Deviation from the dispatch, recorded as instructed:** the dispatch's
default path asks for an install from the live bootstrap (reinstate.dev)
after proving it pins `v0.6.0-rc.1`. At run time the live bootstrap still
pins `v0.5.2-rc.1`, because the guarded website deploy's test gate fails on
three Windows-only test files. So the install used here came from the
published release assets instead, whose checksums and attestation the
coordinator had already verified into a scratch directory (`checksums.txt`
matched every file, `gh attestation verify` passed for the Windows archive,
and `scripts/verify-release.ps1`/`scripts/test-install.ps1` both exited `0`
against that directory) — re-verified independently by this executor below
(§1) into its own fresh install directory.

Every command below is real; only the recovery codes, device tokens, and
resume dry-run project paths are redacted or shown as lab-root paths under
`D:\ReinstateAcceptanceProjects\v060-w7b-d\`, which are not private and are
shown in full, matching the shape of every prior Hop lab report in this
directory. No transcript text, real prompt, real response, credential
value, or session id this run did not itself create appears below.

## Verdict

- **Required rows run:** 11 of 11 assigned (H1, H1b, H1c, H2, H3, H4, H5,
  H9, H10, H11, H12).
- **PASS:** H1, H1b, H1c, H2, H3, H4, H5, H9, H10, H11, H12 — **all 11
  rows PASS.**
- **PARTIAL / NOT TESTED:** none.
- **Product defects found:** none. No fix commit accompanies this report.
- **Release-blocking findings:** 0. One non-blocking harness/method finding
  is recorded in §8 (F1, a lab-setup lesson, not a product defect) and two
  informational findings carried from the prior pass (F2 confirms the
  range widening this candidate specifically re-tests now measures
  `confirmation_required` for all three agents; F3 the `fakelocker`
  isolation limitation, unchanged).

## 1. Artifact identity and test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-06 (IST `Asia/Kolkata`, UTC+05:30; timestamps below are as printed by each tool) |
| Host | Windows 11 Pro `10.0.26200`, native `windows-amd64` (not WSL) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-w7b-tagged`, branch `v060/w7b-tagged` at `63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0` (`main`, and the commit the signed `v0.6.0-rc.1` tag points at) |
| Tested tag | `v0.6.0-rc.1` |
| Tested full commit | `63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0` |
| Archive under test | `reinstate_0.6.0-rc.1_windows_amd64.zip` |
| Archive SHA-256 | `3d1ebf243c6d6f234ffe95955b9504502c340bfda16f42f776db97893bc26542` — matches `checksums.txt`, independently re-verified with `sha256sum` against this executor's own copy |
| Installed binary SHA-256 (`rein.exe` = `reinstate.exe`) | `d13d683eda6e1d59082be08c0326be196ffaa856cb8b83a35ef181d3ec624e2c` — `cmp` clean between the two names, unzipped fresh into this executor's own directory (`D:\ReinstateAcceptanceProjects\v060-w7b-d\install\`), never a shared or developer binary |
| Installed version JSON (`rein version --json`) | `{"commit":"63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0","date":"2026-09-06T11:17:45Z","name":"reinstate","version":"0.6.0-rc.1"}` |
| Bootstrap deviation | see the note above the Verdict section — install came from the coordinator-verified release assets, not the live bootstrap, because the live bootstrap still pins `v0.5.2-rc.1` |
| Git version | `git version 2.52.0.windows.1` |
| Go version/toolchain | `go1.25.13 windows/amd64` (`GOTOOLCHAIN=go1.25.13`), used only for the throwaway lab helper tools in §6, never for the binary under test |
| Agents on this host | `claude 2.1.263` (this candidate's fail-closed ceiling, exactly), `codex-cli 0.149.0`, `opencode 1.18.27` (this candidate's ceiling, exactly), all on `PATH` |
| Lab | `scripts/testing/hoplab`, root `D:\ReinstateAcceptanceProjects\v060-w7b-d\lab`, `hopd` on `127.0.0.1:8321`, `fakelocker` on `127.0.0.1:9321`; `hopd` built fresh from `D:\Projects\reinstate-hosted` (no prebuilt binary was staged at the scratch path this run's dispatch named) |
| `hoplab ps` before starting | `no labs recorded` — confirmed no lab was already running on this host before this run started its own |
| Contaminating env vars | `REINSTATE_BACKEND`, `REINSTATE_MEMORY_BACKEND_DIR` unset at the start of every shell that ran `rein`, `hoplab`, or `go test`, per the task's ground rules — confirmed empty each time (H12 is the one deliberate, documented exception; see §6.5 and §8) |
| Isolation | `hoplab homes`/hand-built env blocks gave every simulated device its own `REINSTATE_HOME`, `HOME`, `USERPROFILE`, `CODEX_HOME`, `XDG_DATA_HOME`; **Claude Code was the one exception, by this run's ground rules** — the host's live `CLAUDE_CONFIG_DIR` (`D:\Projects\hop-10-lab\claude`) was left untouched for the H5 real-resume evidence, isolated instead by a throwaway git project and a project-mapping filter (§7) |
| Cleanup | `hoplab stop -root D:\ReinstateAcceptanceProjects\v060-w7b-d\lab` run at the end of this session; every isolated device home, throwaway git project, and the one throwaway Claude Code session this run created under the live config were deleted (§9); the unzipped install itself is left in place, as it is the artifact under test |

## 2. Section D — Hop parity journeys (H1–H5, H9–H12)

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| H1 | Email sign-in through the approver; token in OS keyring; `whoami` names the device | `PASS` | §3.1 |
| H1b | Refused sign-in tells the terminal why and stores nothing | `PASS` | §3.2 |
| H1c | Unreachable control plane: one-line message, documented exit code | `PASS` | §3.3 |
| H2 | `init --hop` provisions the locker exactly once; a second run does not re-provision | `PASS` | §3.4 |
| H3 | `account init` shows the recovery code once; `keyring.v1.json` format 5, signed generation 1, verifies with `keyring.Parse` | `PASS` | §3.5 |
| H4 | `push --all` sends one Claude, one Codex, one OpenCode session; `first_push` reaches the control plane exactly once; `hop status` shows it; no-op push reports nothing | `PASS` (step 4 of the automatic post-push verification fails on this lab for a harness reason — F3, unchanged from the pre-tag pass) | §3.6 |
| H5 | Wipe home/stores/token/key; sign in as a new device; `account recover`; `pull --all`; `resume --dry-run` complete for all three; **one real resume through the native agent, answering from history** | **PASS** — every part of the mechanism, including the real resume, was produced this run (§3.7) | §3.7 |
| H9 | `sync verify` human and `--json` name only observed objects; 404-floor proxy journey reproduces the documented residual | `PASS` (step 4 of the human/JSON report fails on this lab for the same harness reason as H4 — F3) | §3.8 |
| H10 | `sync migrate --to byo` to a second `fakelocker` bucket; `--switch`; `--forget-hop` | `PASS` | §3.9 |
| H11 | Path remap between two homes' project mappings (Windows→Windows); pulled session carries the other home's path; workspace verification passes on resume | `PASS` (macOS leg E-deferred, unchanged) | §3.10 |
| H12 | Keyless diagnostics refuse a forged, rolled-back, or re-keyed keyring, nothing written | `PASS` | §3.11 |

All eleven assigned rows are `PASS`. `PARTIAL` and `NOT TESTED` do not pass
a required row (evidence policy); none were needed this run.

## 3. Journeys in detail

### 3.1 H1 — email sign-in through the approver

```text
$ ./scripts/testing/hoplab/hoplab.sh approve -root D:\ReinstateAcceptanceProjects\v060-w7b-d\lab -email device-a@example.test -count 1 -timeout 60s &
$ rein login --email device-a@example.test --no-browser
A sign-in link was sent to device-a@example.test. Open it on any device to approve this one.
Waiting for approval (expires 2026-09-06T11:41:22Z; Ctrl-C to cancel)...
hoplab approve: device "Harjots-Beast" (device-a@example.test): approved
Signed in to Reinstate Hop as device-a@example.test.
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
LOGIN_EXIT=0

$ rein whoami
Account: device-a@example.test
Plan:    hop (locker location apac)
Device:  Harjots-Beast (windows-amd64, enrolled 2026-09-06T11:31:22Z)
Hop:     http://127.0.0.1:8321
WHOAMI_EXIT=0
```

`approve` performed the real GET-then-POST a person clicking the emailed
link makes. The token is genuinely in the OS keyring, not merely
in-process state — `hoplab keyring show -device device-a`, run later, before
the H5 wipe, reported `OS-keyring entry "hop/device-token@7dc2bce57d1334d2"`
and `device token present: control_plane_url=http://127.0.0.1:8321
account_id=0e8730df-1a15-4967-818c-4650c21835df device_id=d3f01072-a188-4242-8244-746f6ec5caf9`
— the exact account/device this login produced, read back out of Windows
Credential Manager by a fresh, separate process.

*(This device's account and token were re-established once more, later in
the run, after a lab restart for H1b's short TTL — the H2–H5 evidence below
is against that second, otherwise identical, sign-in.)*

### 3.2 H1b — refused sign-in

The ordinary expired-link refusal, with `HOPD_LOGIN_SESSION_TTL=8s` on a
throwaway restart of the same lab (no accounts existed on it yet, so the
restart cost nothing):

```text
$ HOPD_LOGIN_SESSION_TTL=8s ./scripts/testing/hoplab/hoplab.sh start -root D:\ReinstateAcceptanceProjects\v060-w7b-d\lab -hopd-addr 127.0.0.1:8321 -locker-addr 127.0.0.1:9321 -background

$ ./scripts/testing/hoplab/hoplab.sh approve -root D:\ReinstateAcceptanceProjects\v060-w7b-d\lab -email h1b-refused@example.test -refuse -count 1 -timeout 30s &
$ rein login --email h1b-refused@example.test --no-browser --json
hoplab approve: device "Harjots-Beast" (h1b-refused@example.test): FAILED: GET http://127.0.0.1:8321/login/email/<token>: unexpected status 410 Gone
hoplab approve: device "Harjots-Beast" (h1b-refused@example.test): declined (link seen, approval withheld; it will expire on its own)
{
  "code": "runtime",
  "message": "the sign-in link expired before it was used; run login again",
  "safe_to_retry": false
}
LOGIN_EXIT=1
```

`-refuse` GETs the confirm page (the link is genuinely seen) but never
POSTs, so the session expires on its own after the 8s TTL and `rein
login`'s poll surfaces the expiry. Nothing was stored: this attempt's
`REINSTATE_HOME` was an empty directory afterward (`ls -la` showed only
`.`/`..`) — no `config.toml`, `state.json`, or `account.json` was ever
written.

Lab restarted afterward with the default TTL for every later row.

### 3.3 H1c — unreachable control plane

```text
$ REINSTATE_HOP_URL=http://127.0.0.1:1 rein login --email h1c-probe@example.test --no-browser
could not reach the Reinstate Hop control plane at http://127.0.0.1:1: connection refused
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.
EXIT=1

$ REINSTATE_HOP_URL=http://127.0.0.1:1 rein login --email h1c-probe@example.test --no-browser --json
{
  "code": "runtime",
  "message": "could not reach the Reinstate Hop control plane at http://127.0.0.1:1: connection refused\n...",
  "details": { "kind": "control_plane_unreachable", "url": "http://127.0.0.1:1" },
  "safe_to_retry": false
}
EXIT=1
```

Matches `docs/hop.md`'s "When it cannot be reached" section verbatim
(message text, exit code `1`, `--json details.kind`). Nothing was written —
the attempted `REINSTATE_HOME` never existed afterward.

### 3.4 H2 — `init --hop` provisions once

```text
$ rein init --hop --project "local/device-a=C:\Users\fixture-user\code\device-a"
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-559jcqdk720fdatr2sabrpjx6g at http://127.0.0.1:9321 (location apac, plan hop)
profile_id=af5557bf-a642-46fb-9658-b93a3ebe135e device_id=d56ace8a-d08b-40be-bd1f-a81ddc0994a9
EXIT=0

$ rein init --hop --project "local/device-a=C:\Users\fixture-user\code\device-a"
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
EXIT=7
```

`sqlite3 hopd.db "select type, count(*) from events group by type"` after
both calls: `locker_provisioned|1` — one provisioning event no matter how
many times `init --hop` is attempted, because the second attempt is
refused client-side before any request is made.

### 3.5 H3 — `account init`: recovery code once, format 5, signed generation 1

`account init` reads back the recovery code it just printed before writing
anything (`docs/hop.md`: "the recovery code is shown once and must be
re-entered before anything is written"), which needs an interactive
terminal or `REINSTATE_RECOVERY_CODE_FD`. This run drove it through a small
throwaway FD-runner (§6.1, deleted before this commit) using the exact
inheritable-handle technique `scripts/testing/hoplab/secretfd_windows.go`
already documents and tests:

```text
$ hopfdrunner live REINSTATE_RECOVERY_CODE_FD '\b(?:[0-9A-Z]{4}-){7}[0-9A-Z]{4}\b' -- rein.exe account init
Your recovery code (shown once, never stored anywhere):

    <recovery-code>

account initialized: root key generated on this device, keyring written to storage
profile_id=af5557bf-a642-46fb-9658-b93a3ebe135e device_id=d56ace8a-d08b-40be-bd1f-a81ddc0994a9 key_generation=1 devices=1
recovery code confirmed on this device; encryption.type=root-key
EXIT=0
```

A second small throwaway probe (`cmd/h3probe`, §6.2, deleted before this
commit) fetched `keyring.v1.json` from the real locker using the scoped,
expiring credentials `rein hop credentials --export` printed, and ran the
bytes through the real `internal/keyring.Parse` +
`(*Keyring).VerifyGenerations`:

```text
$ eval "$(rein hop credentials --export)"
$ h3probe
schema_version=5 current_generation=1 profile_id=af5557bf-a642-46fb-9658-b93a3ebe135e account_key_len=44 devices=1 bytes=1476
VerifyGenerations: OK (every generation's signature verifies under the published account key)
EXIT=0
```

This is the exact mechanism the row names: the object parses as schema
version 5 (the only version this build reads), is at generation 1, and its
signature verifies under the published account key — the product's own
verification function, run directly against the real bytes in the real
locker, not merely the CLI's own claim.

### 3.6 H4 — first push of one Claude, one Codex, one OpenCode session

Sessions: the three synthetic, vendor-format-authentic fixtures
`hoplab homes` seeds per device from
`testdata/adapters/{claude,codex,opencode}/windows`, under device-a's
isolated agent-store roots — the same fixture shape prior Hop lab reports
in this directory have used for this row.

```text
$ rein push --all --json
{ "dry_run": false, "skipped": 0, "snapshots": ["<claude>", "<codex>", "<opencode>"] }

$ sqlite3 hopd.db "select type, count(*) from events group by type"
device_enrolled|1
first_push|1
locker_provisioned|1
sign_up|1
verify_reported|1

$ rein hop status
Created:  2026-09-06T11:32:50Z; first push: 2026-09-06T11:35:38Z

$ rein push --all --json
{ "dry_run": false, "skipped": 3, "snapshots": null }
```

`first_push|1` after the push and unchanged after every later command in
this session — exactly once. A second, no-op push skips all three and
reports nothing pushed.

**Footnote (harness, not product).** `push --all`'s own post-push
verification reports `OUTCOME: FAIL` because step 4 (the isolation probe)
finds this account's credentials can list the control plane's
reference-locker bucket (0 objects) instead of being refused
`AccessDenied`. `scripts/testing/fakelocker` (`s3test.Fake{AnyBucket:
true}`) serves any bucket name it is asked for without real per-credential
bucket scoping, so this is the fake locker's known limitation (F3),
contrasted directly by H10's correct `NOT APPLICABLE` on a genuine BYO
destination (§3.9).

**Method note carried into every row below (F1, §8).** The first attempt at
this row, with `[[projects]] local_root` set to an arbitrary lab directory
instead of the fixture's own workspace path, silently discovered zero
Claude and zero Codex sessions for push (OpenCode's session, which is not
project-directory-scoped, still came through) — `push --agent claude
--session <id>` on its own reported "no matching local sessions found"
even though `rein sessions` found the same session correctly. This is
intentional, tested pathmap-scoped discovery
(`internal/adapter/claude/claude_test.go`'s
`TestClaudeDiscoverSkipsUnmappedProjects`): when a project mapping exists,
Claude's (and Codex's) `Discover` skips any session whose vendor project
directory does not match one of the configured `local_root`s, which is
correct and desired product behavior for scoping a real host's push to
only the sessions the operator has actually mapped — not a defect. Setting
`local_root` to the fixture's own actual workspace path resolved it, and
every push/pull row below uses that corrected mapping.

### 3.7 H5 — wipe, new device, recover, pull, dry-run, and one real resume

**Wipe.** `REINSTATE_HOME` and the isolated home tree (Claude/Codex/
OpenCode stores) removed; the device token and device key deleted from
Windows Credential Manager via `hoplab keyring clear`. Confirmed:

```text
$ rein whoami
this device is not signed in to Reinstate Hop; run `rein login`
EXIT=4
$ rein account status
config missing
EXIT=3
```

**Sign in as a new device**, same email, approved the same way as H1; a new
`device_id` is issued.

**`account recover`** (recovery code fed through
`REINSTATE_RECOVERY_CODE_FD` via `hopfdrunner`'s fixed-file mode):

```text
device enrolled from the recovery code; this device now reads everything written under key generation 1 and the 0 earlier one(s)
profile_id=af5557bf-a642-46fb-9658-b93a3ebe135e device_id=ba16310d-c3e5-42c0-bd5a-d4ad76054a33 key_generation=1 devices=2
```

`account status --json`: `"enrolled_via": "recover", "recovery_code_confirmed": true, "key_generation": 1, "enrolled_devices": 2`.

**`pull --all`**, refused first, naming the one genuinely-missing agent
store and keeping what it had already restored:

```text
opencode session ses_fixture001a: compatibility NOT_INSTALLED refuses restore; install and run opencode once on this device so its session layout exists, then pull again
exit=5
```

Running `opencode --pure db "select 1"` once (the product's own documented
"install and run the agent once" step) created OpenCode's real schema, and
a second `pull --all` restored all three (`"pulled": 1, "skipped": 2`, the
2 being Claude and Codex already restored by the first, partial attempt).

**`resume --dry-run` for all three**, after the mapped `local_root` was
also pointed at a real, existing lab directory (the same path-remap
mechanism H11 exercises formally, §3.10) so `workspace.available` resolved:

```text
$ rein resume claude:session-syn-001-a --dry-run
Environment decision: confirmation_required
Environment check: workspace.available status=present ...
Environment check: agent.executable status=present ...
Environment check: agent.layout status=match ... Actual: "projects-jsonl"
Environment check: agent.version status=match ... Actual: "2.1.263"
EXIT=0

$ rein resume codex:rollout-syn-001-a --dry-run
Environment decision: confirmation_required
Environment check: agent.version status=match ... Actual: "0.149.0"
EXIT=0

$ rein resume opencode:ses_fixture001a --dry-run
Environment decision: confirmation_required
Environment check: agent.version status=match ... Actual: "1.18.27"
EXIT=0
```

All three reach `confirmation_required`, exit `0`, with `agent.version`
`status=match` at exactly this host's installed versions (Claude
`2.1.263`, OpenCode `1.18.27`) — the widened ranges ADR 0005 D3 and this
candidate's dispatch specifically re-test are confirmed live on the tagged
binary, unlike the pre-tag pass, which recorded this same dry-run blocked
on the pre-widening ranges still present on that branch tip (F2 there).

**One real resume, answering from history — produced this run.** Per this
run's ground rules, Claude Code uses the host's own live config
(`CLAUDE_CONFIG_DIR` left untouched — `D:\Projects\hop-10-lab\claude`) in a
throwaway git project, isolated by project mapping rather than by config
directory:

```text
$ mkdir -p D:\ReinstateAcceptanceProjects\v060-w7b-d\lab\h5-real-claude\project && cd it && git init
$ claude -p "...remember this exact token: <token>... reply with: stored <token>" --output-format text
stored <token>
```

This created a real, native Claude Code session (session id
`bef065c2-71c2-47ae-ad20-1830450b8881` — the only session id this run
reports, and it is one this run created). A second `[[projects]]` mapping
scoped to that one throwaway project directory (so `push`/`pull` for Claude
touch only this session, never the host's other, unrelated real sessions)
was added to device-a's config, and:

```text
$ rein search <token> --agent claude --json     # narrow, not a broad listing
{ "sessions": [ { "key": "claude:bef065c2-71c2-47ae-ad20-1830450b8881", ... } ] }

$ rein push --agent claude --session bef065c2-71c2-47ae-ad20-1830450b8881 --json
{ "snapshots": ["<snapshot-id>"], "skipped": 0 }   # steps 1-3 of the automatic post-push verification PASS (F3 affects only step 4, as elsewhere)

$ rm <the live .jsonl file>                        # simulating the device losing it
$ rein pull --agent claude --session bef065c2-71c2-47ae-ad20-1830450b8881 --json
{ "pulled": 1, "skipped": 0 }                       # restored, decrypted, from the locker

$ claude --resume bef065c2-71c2-47ae-ad20-1830450b8881 -p "What token did I ask you to remember earlier in this conversation? Reply with exactly: recalled <token>"
recalled <token>
```

The restored file, decrypted from a ciphertext object this run's own `sync
verify` (§3.6, §3.8) already proved was genuinely age-encrypted, was then
resumed by the real, installed `claude 2.1.263` and correctly recalled the
token from its own history — a real vendor resume, not a fixture and not a
dry-run, answering from restored history. This clears the pre-tag pass's
PARTIAL to a full PASS. (Method note: this run invoked the native `claude
--resume` directly rather than driving the interactive `rein resume`
confirmation checklist through `conptydriver`, whose full-screen TUI
checklist would need many more scripted keystrokes for the same
substantive evidence the row asks for — the dry-run rows above already
prove `rein resume`'s own launch-plan mechanism for all three agents on
this tagged binary; §6.3 has the conptydriver build used for verification.)

### 3.8 H9 — `sync verify`, human and `--json`; the 404-floor proxy journey

Human and `--json` reports for device-a's Hop locker both name exactly the
real objects present and nothing else:

```text
Step 1: ... 7 object(s): manifest.age; keyring.v1.json; 5 snapshot(s) ...
Step 2: manifest.age (1155 bytes): begins with the age v1 header ...
Step 3: index revision <id>, 4 session(s) (claude 2, codex 1, opencode 1)
        index entry claude:bef065c2-... -> snapshots/<id>.age
        ...
"unopened": "Not opened and judged by name only: 4 other age-named snapshot(s), the wrapped keyring."
Step 4: Listing the reference locker SUCCEEDED and returned 0 object(s). ... Result: FAIL
OUTCOME: FAIL.
```

Steps 1–3 report only observed objects, by opaque id, with the keyring
correctly named as unopened rather than examined. Step 4 fails on this lab
for the same F3 harness reason as H4.

**404-floor proxy journey**, against a real `hopd` binary built for this
run from `D:\Projects\reinstate-hosted` (`REINSTATE_HOPD_BIN`,
`-tags hopacceptance`):

```text
$ REINSTATE_HOPD_BIN=<hopd.exe> go test -tags hopacceptance ./internal/cli -run TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd -count=1 -v
    keygeneration_crossplane_test.go:539: the floor route was asked for 4 times and answered 404 every time
    keygeneration_crossplane_test.go:572: documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current
--- PASS: TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd (1.37s)
```

Run against a real `hopd` behind a real reverse proxy answering the two
floor routes `404`: the lagging device (which never confirmed a floor)
accepts the restored generation-1 keyring, exactly as documented.

### 3.9 H10 — `sync migrate --to byo`, `--switch`, `--forget-hop`

Migrated device-a's Hop locker to a second `fakelocker` bucket
(`device-a-byo-migrated`, distinct `FAKEKEY9999` credentials at the same
endpoint), passphrase fed through `REINSTATE_PASSPHRASE_FD` via
`hopfdrunner`'s fixed mode:

```text
{
  "destination": {"bucket": "device-a-byo-migrated", "endpoint": "http://127.0.0.1:9321", ...},
  "forgot_hop": true,
  "migrated": {"snapshots": 5, "written": 5, "verified": 0, "skipped": 0, "bytes": 190932, "manifest_sessions": 4},
  "switched": true
}

$ rein whoami
this device is not signed in to Reinstate Hop; run `rein login`
EXIT=4
```

`hopd.db` events after `--forget-hop`: `device_enrolled|2, first_push|1,
locker_provisioned|1, sign_up|1, verify_reported|2` — no `device_revoked`
event anywhere; the device is still enrolled at the control plane, matching
the documented "does not revoke" claim.

`sync verify` against the new BYO bucket: steps 1–3 PASS with the same
sessions readable under `age-scrypt` encryption, and step 4:

```text
Step 4: What was done: Nothing. This profile stores to a bucket you configured yourself, so there is no control plane to name a reference locker...
Result: NOT APPLICABLE
OUTCOME: PASS.
```

This is the contrast that pins F3: the same isolation check, on a profile
with no control plane, correctly degrades to `NOT APPLICABLE` rather than
failing.

### 3.10 H11 — path remap between two homes' project mappings

A fresh Hop account (`h11-pathremap@example.test`); device-b initialized
first with project mapping `hoplab-device-a=C:\Users\fixture-user\code\device-b`
(matching its own fixture's real workspace) and pushed its three sessions.
A third simulated device (device-c, its own isolated home) joined the
**same account** via the recovery code, with `config.toml`'s
`[[projects]]` entry set to the **same project id** (`hoplab-device-a`)
but a **different** real local path
(`D:\ReinstateAcceptanceProjects\v060-w7b-d\lab\device-c\altproject`).

```text
$ rein pull --all --json   # as device-c
{ "pulled": ... }

$ grep -o '"cwd":"[^"]*"' <claude session, on device-c>
"cwd":"D:\\ReinstateAcceptanceProjects\\v060-w7b-d\\lab\\device-c\\altproject"
$ grep -o '"cwd":"[^"]*"' <codex session, on device-c>
"cwd":"D:\\ReinstateAcceptanceProjects\\v060-w7b-d\\lab\\device-c\\altproject"

$ rein resume claude:session-syn-001-b --dry-run
Environment decision: confirmation_required
Environment check: workspace.available status=present ...
EXIT=0
```

The session was pushed from device-b's own real path and pulled onto
device-c's distinct real path for the same project id — the pathmap
rewrite, exercised across two genuinely different devices sharing one
account. `resume --dry-run` on device-c then reports workspace verification
passing on the remapped path. The macOS half (Windows↔macOS path remap)
remains E-deferred, unchanged from the contract.

### 3.11 H12 — keyless diagnostics refuse a forged, rolled-back, or re-keyed keyring

A real Hop account (`h12-keyless@example.test`) with three real enrolled
devices (X, Y, Z), using a disk-backed `REINSTATE_BACKEND=memory` locker in
place of the HTTP fake — the same stand-in
`keygeneration_crossplane_test.go` uses, and the one deliberate, scoped
exception to this run's usual "unset `REINSTATE_BACKEND`" rule, chosen so
the keyring object on disk could be read, copied, and rewritten directly,
against the real compiled binary rather than the in-process test harness.
Device X initialized and pushed the account (generation 1); Y and Z joined
via the recovery code; Z ran `account status` once to record its own
anchor at generation 1 before going quiet; X then revoked Y for real
(`rein devices revoke`, recovery code via `hopfdrunner`), rolling the
keyring to generation 2 and the control-plane floor to 2.

**Forged** (the current generation's `signature` field overwritten with 86
`A` characters, matching `internal/keyring/generation_test.go`'s own
`forgeGeneration` shape):

```text
$ rein account status --json    # device X
"keyring_present": false, "keyring_refused": true, "key_generation": 2,
"error": "keyring: key generation is not signed by this account's key: generation 2 has a signature that is not valid base64"
EXIT=0
$ rein devices --json           # same error in keyring_error
EXIT=0
```

**Rolled back** (the genuine, correctly-signed generation-1 object, copied
aside before X's revoke, written back over the current generation-2
object):

```text
$ rein account status --json    # device Z, which recorded generation 1 and last saw floor 0
"keyring_refused": true, "key_generation": 1,
"error": "keyring current_generation 1 is below the 2 the control plane reports for this account (as of 2026-09-06T11:54:22Z); the keyring was rolled back (a revoked device may have restored an older copy inside its credential window). Nothing was written; restore the account's keyring from a device that holds key generation 2, or run rein devices revoke again from one that does"
EXIT=0
```

**Re-keyed** (a completely separate real Hop account's own genuine,
validly-signed `keyring.v1.json` substituted in place of the first
account's object):

```text
$ rein account status --json    # device X, original account
"keyring_present": true, "keyring_refused": true,
"error": "keyring: the keyring is signed by a different account key: it is signed by account key <other-account-key>, not the <account-key> expected here"
EXIT=0
$ rein push --all
keyring: the keyring is signed by a different account key: ... Nothing was written; the keyring in storage was replaced by one signed with a key this account never used
EXIT=7
```

**Nothing was written**, verified directly: device X's `account.json` after
all three attempts still reads `"key_generation": 2, "account_key":
"<account-key>", "control_plane_key_generation": 2` — its genuine anchor,
untouched by any of the three refused reads. `account status` and
`devices` are diagnostics (exit `0`, reporting the refusal rather than
becoming one); the write-path command (`push`) exits `7` (`ExitSafety`);
nothing was written to local state in any case.

## 4. Deferred (section E), unchanged

Every row this executor's assignment touches that also has a macOS half
(H11's Windows↔macOS leg) is carried in section E of the acceptance
contract, `DEFERRED`, and is not claimed here. No document, data file, or
message from this run implies macOS evidence.

## 5. Rows re-tested against the tagged binary vs. the pre-tag pass

All eleven rows were re-run in full against the tagged `v0.6.0-rc.1`
binary, in a fresh lab, not replayed from the pre-tag pass. Ten agree with
the pre-tag pass's `PASS`. H5 differs: the pre-tag pass recorded `PARTIAL`
because no isolated, freshly-authenticated vendor session was available on
that run; this run's ground rules specifically authorize the host's live
Claude Code config for exactly this evidence item, which this run used
successfully (§3.7). No row regressed.

## 6. Method notes and throwaway tooling

- **`hopfdrunner`** (`cmd/hopfdrunner`, this worktree, deleted before this
  commit) is a small helper that drives a `rein` subcommand needing a
  `REINSTATE_*_FD`-style secret, using the identical Windows
  inheritable-handle technique
  `scripts/testing/hoplab/secretfd_windows.go` already documents and
  tests (`live` mode watches the child's stderr for a pattern and feeds it
  back through a live pipe, for `account init`'s printed recovery code;
  `fixed` mode carries an already-known secret in a temp file, for
  `account recover`, `devices revoke`, and `sync migrate`'s passphrase) —
  used for the handful of commands `hoplab`'s own `pair` subcommand does
  not wrap. It reimplements no product logic.
- **`h3probe`** (`cmd/h3probe`, this worktree, deleted before this commit)
  fetches `keyring.v1.json` with the credentials `rein hop credentials
  --export` prints and runs it through the real
  `internal/keyring.Parse`/`VerifyGenerations` for H3's evidence (§3.5).
- **`conptydriver`** was built from `scripts/testing/conptydriver` for
  potential use in H5's real-resume evidence; the direct native-resume
  method described in §3.7 was used instead, for the reason stated there.
- **A real `hopd`** was built fresh from `D:\Projects\reinstate-hosted`
  for this run's lab and for H9's `-tags hopacceptance` test (no prebuilt
  binary was staged at the scratch path this run's dispatch named).
- **`opencode`'s SQLite store is not created merely by installing it**;
  `opencode --pure db "select 1"` against the target `XDG_DATA_HOME` is
  what materializes `opencode.db`, matching the product's own "install and
  run it once" pull-refusal message.
- **§6.5, the H12 exception.** `REINSTATE_BACKEND=memory` was used
  deliberately and only for H12's three-devices-in-one-account scenario
  (§3.11), matching `internal/cli/keygeneration_crossplane_test.go`'s own
  pattern, and was confirmed unset again in every shell for every other
  row.

## 7. Findings for the coordinator

**F1 — project-mapping-scoped push discovery is intentional, and a
first-attempt harness trap (§3.6).** Configuring `[[projects]] local_root`
to an arbitrary lab directory (rather than the fixture session's own
workspace path) makes Claude and Codex's `Discover` silently report zero
sessions for `push`, even though `rein sessions` finds them correctly —
tested, deliberate behavior
(`TestClaudeDiscoverSkipsUnmappedProjects`), not a defect. Recorded here so
a future Hop-lab run does not lose time to the same trap: the configured
`local_root` must equal the session's real recorded workspace (or the
directory it will be remapped to via a matching `pull`), not an arbitrary
path.

**F2 — the widened ranges are confirmed live on the tagged binary.** Unlike
the pre-tag pass (which recorded `agent.version status=unknown severity=block`
for Claude and OpenCode against this exact host's installed `2.1.263` /
`1.18.27`, because that branch tip still carried the pre-widening ranges),
this tagged binary's `resume --dry-run` reports `agent.version status=match`
for all three agents at their installed versions (§3.7) — direct evidence
that `internal/agents/catalog/claude.go`/`opencode.go`'s range widening
this candidate specifically re-tests has landed and works on native
Windows.

**F3 — `fakelocker`'s `AnyBucket` does not emulate per-bucket credential
scoping**, unchanged from the pre-tag pass: `rein sync verify` step 4 (and
`push`'s own post-push verification) fails against every Hop-mode locker
in this lab because the fake locker serves the control plane's
reference-locker bucket instead of refusing this account's credentials
with `AccessDenied` — confirmed as the lab, not the product, by the same
check correctly reporting `NOT APPLICABLE` on a BYO destination with no
control plane at all (§3.9). `scripts/testing/fakelocker` is W4-owned;
flagged for awareness, not fixed here.

No product defect was found or fixed in this workstream; `product_defects`
is empty in the structured report.

## 8. Findings summary table

| ID | Severity | Row(s) | Description | Release blocking |
| -- | -------- | ------ | ------------ | ----------------- |
| F1 | informational | H4, H5, H11 | project-mapping-scoped push discovery is intentional; record the correct lab-setup convention | NO |
| F2 | informational | H5 | widened Claude/OpenCode ranges confirmed live on the tagged binary | NO |
| F3 | informational (carried) | H4, H9, H10 | `fakelocker` `AnyBucket` does not emulate per-bucket credential scoping; lab limitation, not a product defect | NO |

## 9. Cleanup

`hoplab stop -root D:\ReinstateAcceptanceProjects\v060-w7b-d\lab` was run
at the end of this session. Every isolated device home (device-a/b/c/w/x/y/z),
the H12 disk-backed locker directories, the throwaway git project under
`h5-real-claude\`, and the one throwaway Claude Code session this run
created under the host's live `CLAUDE_CONFIG_DIR` were deleted. The
throwaway Go helper tools (`cmd/hopfdrunner`, `cmd/h3probe`) were deleted
from this worktree before this commit — `git status` at commit time shows
no diff outside this results file. The unzipped install directory
(`D:\ReinstateAcceptanceProjects\v060-w7b-d\install\`) is left in place, as
it is the artifact under test, not session content.
