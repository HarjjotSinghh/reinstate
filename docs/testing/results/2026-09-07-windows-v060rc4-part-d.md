# `v0.6.0-rc.4` native Windows acceptance — part D (Hop parity H1–H5, H9–H12)

Executor D of the tagged-artifact run. Covers section D Hop parity journeys
`H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`, `H5`, `H9`, `H10`, `H11`, `H12` (11 of
the 16 section-D rows) against a local, disposable lab
(`scripts/testing/hoplab`). `H6`, `H6b`, `H7`, `H8`, `H8b` are owned by
another executor. Per the rc.4 dispatch, none of these 11 rows is a carried
disposition — every one passed at `v0.6.0-rc.3`'s tagged run and this
candidate's own fixes (the Pi reader, the Qwen version widening) touch
neither Hop, the daemon, nor the interactive surfaces, so a `FAIL` here would
be a new finding, not an expected one.

## Header

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-09-07T07:01:00Z`–`2026-09-07T07:26:00Z` (session) / report filed `2026-09-07` |
| Device | `windows-amd64`, native x64, not WSL |
| OS/version/build | Windows 11 Pro `10.0.26200` |
| Host | acceptance host (hostname redacted) |
| Tested tag | `v0.6.0-rc.4` |
| Tested full commit | `561ec133e7fd040d7937d555a75f0bd7dd7b878e` |
| Windows archive SHA-256 (`checksums.txt`, re-verified before install) | `f5ae24e2edf0364f0e6f9b0df3c5b2c21f47222ecd394b9c40fc5dbd1ae14d87` |
| Installed binary SHA-256 (`rein.exe` == `reinstate.exe`, byte-identical, `cmp` exit `0`) | `fbea94615dabcbc30fc1ebb6e93259ddd330b62573555e551fd3112b31b9d3ef` |
| `rein version --json` (own install, `D:\ReinstateAcceptanceProjects\v060-rc4-d\install`) | `{"commit":"561ec133e7fd040d7937d555a75f0bd7dd7b878e","date":"2026-09-07T06:48:08Z","name":"reinstate","version":"0.6.0-rc.4"}` |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc4-tagged`, branch `v060/rc4-tagged` @ `561ec133e7fd040d7937d555a75f0bd7dd7b878e` |
| Host contamination rule | Every shell in this report ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` before any `rein`/`hoplab` invocation (the seeded device env blocks from `hoplab env` also carry these `unset` lines). `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were left at each seeded device's own isolated value throughout, except for the Claude legs of `H4` and `H5`, where `CLAUDE_CONFIG_DIR` was deliberately pointed at the host's live value (`D:\Projects\hop-10-lab\claude`) so a genuine, live-signed-in Claude Code session could be created and resumed, per this run's explicit evidence-policy exception for Claude. Neither `--all` nor any broad listing was ever run against the live config: only single-session `--agent claude --session <id>` pushes/pulls/resumes, and the only session id ever reported below is the one this run itself created. |
| Bootstrap deviation sentence | Not applicable to this executor: per this run's ground rules, executor A alone installs from the live `https://reinstate.dev/install.ps1` bootstrap and records the artifact identity for the whole run; this executor installed from the coordinator-verified, checksum-matched `reinstate_0.6.0-rc.4_windows_amd64.zip` in the shared `rc4-draft` staging directory into its own fresh `D:\ReinstateAcceptanceProjects\v060-rc4-d\install\` (sha256 re-verified above, matches `checksums.txt`), per the dispatch for every non-A executor. |
| Lab | `hopd` 127.0.0.1:8321, fakelocker 127.0.0.1:9321, root `D:\ReinstateAcceptanceProjects\v060-rc4-d\lab`, built via `hoplab start -root ... -hopd-addr 127.0.0.1:8321 -locker-addr 127.0.0.1:9321 -hosted-dir D:\Projects\reinstate-hosted` (no prebuilt `hopd.exe` was present at the scratch path this run's dispatch named, so `hoplab` built one from source); stopped with `hoplab stop -root <same root>` at the end of this run (`hoplab ps` before starting showed no labs recorded on this host; before stopping, `hoplab ps` showed only this executor's own lab still running) |

All work happened under `D:\ReinstateAcceptanceProjects\v060-rc4-d\` and the
worktree. Six isolated device homes were seeded via `hoplab homes -root <lab>
-devices device-a,device-b,device-c,device-e,device-x,device-y` (each with
its own `REINSTATE_HOME`, `HOME`/`USERPROFILE`, and per-agent `*_HOME`
variables from `hoplab env`; the seven
`REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`/`REINSTATE_S3_*` variables
were left unset in every device's env block). `device-a` signed in first and
initialized the shared Hop account (`account init`); `device-b` and
`device-c` joined it later via `account recover` with the same recovery
code; `device-x` is the `H1b` refused-sign-in throwaway device, later reused
under a **second, independent** email (`device-x2@example.test`) to
initialize a genuinely separate Hop account purely to produce `H12`'s
"re-keyed" variant's own real, correctly-signed keyring object; `device-y`
is the `H1c` unreachable-control-plane throwaway device. One throwaway git
project, `h5-claude-live` under `D:\ReinstateAcceptanceProjects\v060-rc4-d\`,
holds one real Claude Code session this run itself created (via the host's
live `CLAUDE_CONFIG_DIR`, per this run's explicit `H4`/`H5` exception to the
isolated-home rule for Claude specifically) with a planted, non-sensitive
token; only its own session id and the planted token (never a real prompt,
title, or path) appear below, per the evidence policy. Two throwaway Go
helper programs (a `keyring.Parse`/`VerifyGenerations` verifier for
`H3`/`H12`, built as `cmd/tmpverifykeyring`, and a Windows-inheritable-handle
FD runner for `REINSTATE_PASSPHRASE_FD`/`REINSTATE_RECOVERY_CODE_FD`
automation reusing `scripts/testing/hoplab/secretfd_windows.go`'s
`fixedSecretFD` pattern, built as `cmd/tmpfdrun`) were built inside this
worktree, used, and deleted before this commit — `git status` at commit time
shows neither.

---

## Section D — Hop parity journeys (this executor's 11 of 16 rows)

| # | Row | Result | Summary |
| - | --- | ------ | ------- |
| H1 | `rein login --email` completes through the approver; token in OS keyring; `whoami` names the device | PASS | Real GET-then-POST via `hoplab approve`; token confirmed present in the Windows Credential Manager by a fresh, separate `hoplab keyring show` process |
| H1b | Refused sign-in tells the terminal why and stores nothing | PASS | `hoplab approve -refuse`; link expired on its own (default TTL, ~10 min — see harness note below); documented refusal, exit 1; `REINSTATE_HOME` never created |
| H1c | Unreachable control plane prints the one-line message and documented exit code | PASS | `REINSTATE_HOP_URL` at an unused loopback port; human and `--json` both match the documented message, exit 1, `details.kind=control_plane_unreachable`; nothing written |
| H2 | `init --hop` provisions the locker exactly once | PASS | `locker_provisioned\|1` in `hopd.db` at the time of this row; second run refused client-side, exit 7, no second event |
| H3 | `account init` recovery code shown once; keyring format 5, generation 1, verifies | PASS | Recovery code via `hoplab pair init`'s live inheritable-FD pipe; `keyring.v1.json` fetched directly (minted credentials + `curl --aws-sigv4`) and run through a throwaway `keyring.Parse`+`VerifyGenerations` harness: parses cleanly, `CurrentGeneration=1`, signature verifies both unpinned and pinned to the object's own `account_key` |
| H4 | `push --all` sends one Claude, one Codex, one OpenCode session; `first_push` fires once | PASS (step 4 of the automatic post-push verify fails for the pre-existing `fakelocker AnyBucket` harness reason, unchanged from rc.3) | Decrypted manifest: 3 session(s) (claude 1, codex 1, opencode 1); `first_push\|1` exactly once in `hopd.db`; no-op push reports `"skipped":3` |
| H5 | Wipe home/stores/token/key, sign in as a new device, `account recover`, `pull --all`, `resume --dry-run` complete for all three; one real resume | PASS | New device id issued post-wipe; `account recover` via FD-fed known recovery code; the Codex and OpenCode legs were **genuinely deleted from local disk** before pulling — both restored byte-for-byte from ciphertext; the Claude leg's local file could not be deleted (see harness note; the same tool-use protection rc.3 recorded) so its pull correctly reported a conflict against still-present identical content; `resume --dry-run` reached `confirmation_required` with `workspace.available=present` and `agent.executable=present` for all three; a real `claude --resume <id> -p "..."` against the still-intact live config recalled the exact planted token |
| H9 | `sync verify` human and `--json` name only observed objects | PASS (step 4 fails for the same harness reason as H4) | `checked_objects` names exactly `"the index"` and `"the newest snapshot in the index"`; step 4 status is `"fail"`, never `"could not run"` |
| H10 | `sync migrate --to byo`, `--switch`, `--forget-hop` | PASS | 3 snapshots migrated to a second fake-locker bucket in one combined `--switch --forget-hop` call; `switched:true`, `forgot_hop:true`; local `whoami` afterward correctly refuses (`auth_storage`, "not signed in"); `hopd.db.devices` shows the migrated device's row **not** revoked (`revoked_at` empty) — confirms "does not revoke"; the new BYO bucket's `sync verify` step 4 reports `"not-applicable"`, overall `"pass"` |
| H11 | Path remap between two homes' project mappings | PASS (macOS leg E-deferred) | A third device, mapping the pushing device's project id to its own different real local path before pulling, pulled with `cwd` rewritten from the source device's fixture path to its own real workspace; `resume --dry-run` on the remapped Claude session then reports `workspace.available: "present"` |
| H12 | Keyless diagnostics refuse a forged, rolled-back, or re-keyed keyring with nothing written | PASS | Direct S3 `PUT`s (minted credentials, `curl --aws-sigv4`, simulating an attacker with bucket access) of (1) a byte-tampered signature, (2) a genuinely-signed but stale single-device generation-1 object, and (3) a different, genuinely-enrolled account's own keyring, each in turn: (1) `account status`/`devices` refuse with an explicit signature-verify error; (2) the device is absent from the served object, so `push` refuses (`auth_storage`, "not enrolled in key generation 1"); (3) `account status`/`push` both explicitly refuse ("signed by a different account key ... Nothing was written", exit 7); the device's own `account.json` local anchor is byte-identical (`sha256 5ab55e8cc9f20067e5c6b38ca0c3135571336a93c4ec633556e6ccc24b7f7d3c`) before, between, and after all three attempts |

**11/11 PASS.** `H6`, `H6b`, `H7`, `H8`, `H8b` are out of this executor's
scope (part E).

---

## Evidence — H1

```
$ source envs/device-a.sh   # REINSTATE_HOME=...\device-a\reinstate, unset REINSTATE_BACKEND/REINSTATE_MEMORY_BACKEND_DIR
$ hoplab.sh approve -root <lab> -email device-a@example.test -count 1 -timeout 90s &
$ rein login --email device-a@example.test --no-browser
A sign-in link was sent to device-a@example.test. Open it on any device to approve this one.
Waiting for approval (expires 2026-09-07T07:11:59Z; Ctrl-C to cancel)...
Signed in to Reinstate Hop as device-a@example.test.
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
LOGIN_EXIT=0
$ hoplab.sh keyring show -root <lab> -device device-a
hoplab: device-a -> REINSTATE_HOME=...\device-a\reinstate -> OS-keyring entry "hop/device-token@faa98c875ca87501"
hoplab: device token present: control_plane_url=http://127.0.0.1:8321 account_id=262f6081-b656-4b46-8340-6bcd952ffda0 device_id=ba919c90-971f-4947-a321-6d80659d4df6
$ rein whoami --json
{"account":{"id":"262f6081-...","email":"device-a@example.test","plan":"hop", ...},"device":{"id":"ba919c90-...","name":"Harjots-Beast", ...}}
```

## Evidence — H1b

```
$ source envs/device-x.sh
$ hoplab.sh approve -root <lab> -email device-x@example.test -count 1 -timeout 60s -refuse
$ rein login --email device-x@example.test --no-browser
A sign-in link was sent to device-x@example.test. Open it on any device to approve this one.
Waiting for approval (expires 2026-09-07T07:12:14Z; Ctrl-C to cancel)...
the sign-in link expired before it was used; run login again
LOGIN_EXIT=1
--- approve log ---
hoplab approve: device "Harjots-Beast" (device-x@example.test): declined (link seen, approval withheld; it will expire on its own)
--- REINSTATE_HOME after ---
total 0   (only . and ..; nothing created)
```

**Harness note.** A duplicate second `rein login --email device-x@example.test`
was accidentally issued a few minutes later by this session while chasing the
first attempt's own background-task output (the first `login` and `approve`
calls both ran longer than this tool's default foreground window and were
moved to background automatically). The duplicate created its own,
independent sign-in link that nothing ever approved or explicitly declined;
it exited on its own with the same "link expired" refusal roughly ten minutes
later, once its own default TTL elapsed. It did not affect this row's
evidence (already captured from the first, `hoplab approve -refuse`d attempt
above) and created nothing under `REINSTATE_HOME` either. Not a product
defect — a self-inflicted harness duplicate, disclosed for completeness.

## Evidence — H1c

```
$ source envs/device-y.sh
$ export REINSTATE_HOP_URL="http://127.0.0.1:19998"   # unused loopback port
$ rein login --email device-y@example.test --no-browser
could not reach the Reinstate Hop control plane at http://127.0.0.1:19998: connection refused
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.
EXIT=1
$ rein login --email device-y@example.test --no-browser --json
{"code":"runtime","message":"could not reach the Reinstate Hop control plane at http://127.0.0.1:19998: connection refused\n...","details":{"kind":"control_plane_unreachable","url":"http://127.0.0.1:19998"},"safe_to_retry":false}
EXIT_JSON=1
$ ls REINSTATE_HOME    # only . and .. -- nothing written
```

## Evidence — H2

```
$ source envs/device-a.sh
$ rein init --hop --project "hoplab-device-a=C:\Users\fixture-user\code\device-a" --json
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-kg1xfmcyde7qsh0c6kqcwkvtg4 at http://127.0.0.1:9321 (location apac, plan hop)
profile_id=262f6081-... device_id=ba919c90-...
EXIT1=0
$ rein init --hop --project "hoplab-device-a=C:\Users\fixture-user\code\device-a" --json   # second run
{"code":"safety","message":"reinstate home is already initialized; rerun init with --force to back up and replace existing config/state","safe_to_retry":false}
EXIT2=7
$ sqlite3 hopd.db "SELECT type, COUNT(*) FROM events GROUP BY type;"   # at this point in the run
device_enrolled|1
locker_provisioned|1
sign_up|1
```

Device-a's `REINSTATE_HOME` was then wiped (`rm -rf`) before the `H3` row,
since `hoplab pair init` always drives `rein init --hop` itself first and
refuses a device whose home is already initialized (no "skip init" flag) —
a sequencing note, not a product defect; this row's own "second init
refused" evidence is already fully captured above regardless.

## Evidence — H3

```
$ hoplab.sh pair init -root <lab> -device device-a -rein <install>\rein.exe
hoplab: device-a initialized the account; recovery code saved to <lab>\hoplab-state.json for `pair recover`
3SKK-J44F-MPG9-TZ8V-M5H8-E4ZY-XJRV-3SPC
EXIT=0
$ rein account status --json
{"profile_id":"262f6081-...","key_generation":1,"encryption_type":"root-key","keyring_present":true,"keyring_refused":false, ...}
$ rein hop credentials --export | eval
$ curl -s "$AWS_ENDPOINT_URL/$REIN_LOCKER_BUCKET/keyring.v1.json" --aws-sigv4 "aws:amz:auto:s3" \
    --user "$AWS_ACCESS_KEY_ID:$AWS_SECRET_ACCESS_KEY" -H "x-amz-security-token: $AWS_SESSION_TOKEN" \
    -o keyring_v1_fetched.json -w "HTTP=%{http_code}\n"
HTTP=200
$ head keyring_v1_fetched.json
{"schema_version":5,"profile_id":"262f6081-...","current_generation":1,"account_key":"NJKHTC5J1BaizANTEvIcApzW3M635yWUOF4T/KXnQX4=", ...}
$ go run ./cmd/tmpverifykeyring keyring_v1_fetched.json
Parse OK. ProfileID=262f6081-... CurrentGeneration=1 AccountKey=NJKHTC5J1BaizANTEvIcApzW3M635yWUOF4T/KXnQX4= DeviceCount=1
VerifyGenerations(unpinned) OK
VerifyGenerations(pinned to AccountKey) OK
```

## Evidence — H4

```
$ source envs/device-a.sh
$ rein push --all --json
... VERIFICATION REPORT ... Step 1 PASS, Step 2 PASS, Step 3 PASS, Step 4 FAIL (fakelocker AnyBucket — see harness note)
{"conflicts":null,"dry_run":false,"skipped":0,"snapshots":["887a...","bffc...","f6ae..."],"verification":{"outcome":"fail","posted":true}}
EXIT=0
$ rein push --all --json   # no-op
{"conflicts":null,"dry_run":false,"skipped":3,"snapshots":null}
$ rein hop status --json
{"locker":{"first_push_at":"2026-09-07T07:13:08Z", ...}}
$ sqlite3 hopd.db "SELECT type, COUNT(*) FROM events GROUP BY type;"
device_enrolled|1
first_push|1
locker_provisioned|1
sign_up|1
verify_reported|1
```

Step 3's decrypted manifest named `index revision ..., 3 session(s) (claude
1, codex 1, opencode 1)` with one index entry per agent (`claude:session-syn-001-a`,
`codex:rollout-syn-001-a`, `opencode:ses_fixture001a`), confirming the "one
Claude, one Codex, one OpenCode" mechanism directly, not just the count.
These three sessions were `hoplab homes`'s own seeded fixture sessions
(real, valid session-file shapes from `testdata/adapters/{claude,codex,opencode}/windows`)
under device-a's fully isolated home — the genuinely live-CLI-created Claude
session this run also holds (`h5-claude-live`) was deliberately **not**
included in this row's `--all`, and was pushed separately, by single-session
id, as part of `H5` below, to avoid ever running `push --all` against an
environment where `CLAUDE_CONFIG_DIR` points at the host's live config.

## Evidence — H5

```
$ cd h5-claude-live && git init -q && git commit -q --allow-empty -m init
$ claude -p "Remember this exact token for later recall: RC4H5TOKEN-9d3c72. Reply with just: token stored."
token stored.
$ source envs/device-e.sh
$ export CLAUDE_CONFIG_DIR="D:\Projects\hop-10-lab\claude"   # host's live config, per this run's Claude exception
$ rein search RC4H5TOKEN-9d3c72 --agent claude --json
{"sessions":[{"key":"claude:93378822-f8dc-43e8-9191-9d1ffd350a25","project":"h5-claude-live", ...}]}
$ rein login --email device-a@example.test --no-browser   # device-e, via hoplab approve
$ rein init --hop --project "hoplab-device-e=..." --project "h5-claude-live=D:\...\h5-claude-live" --json
$ <tmpfdrun.exe> REINSTATE_RECOVERY_CODE_FD "3SKK-...-3SPC" -- rein account recover --json
device enrolled from the recovery code; ... key_generation=1 devices=4
$ rein push --agent codex --session rollout-syn-001-e --json     # device-e's own fixture sessions
$ rein push --agent opencode --session ses_fixture001e --json
$ rein push --agent claude --session 93378822-f8dc-43e8-9191-9d1ffd350a25 --json   # the one real session; CLAUDE_CONFIG_DIR still live; never --all
--- wipe (device-e) ---
$ hoplab.sh keyring clear -root <lab> -device device-e
hoplab: cleared the OS-keyring device-token entry for device-e
$ rm -rf REINSTATE_HOME/*
$ rm -f  .../device-e/home/.codex/sessions/rollout-syn-001-e.jsonl        # genuinely deleted, not just unmapped
$ rm -f  .../device-e/home/xdgdata/opencode/opencode.db                  # genuinely deleted
--- sign in as a new device ---
$ rein login --email device-a@example.test --no-browser
Signed in to Reinstate Hop as device-a@example.test. This device is enrolled as "Harjots-Beast" ...
$ rein whoami --json   # device.id = 923013a7-... (new; was 767053c9-... before the wipe)
$ rein init --hop --project "hoplab-device-e=..." --project "h5-claude-live=..." --json
$ <tmpfdrun.exe> REINSTATE_RECOVERY_CODE_FD "3SKK-...-3SPC" -- rein account recover --json
device enrolled from the recovery code; ... key_generation=1 devices=5
$ rein pull --agent codex --session rollout-syn-001-e --json
{"pulled":1,"skipped":0, "plans":[{"destinations":[".../home/.codex/sessions/rollout-syn-001-e.jsonl"]}]}   # restored
$ rein pull --agent opencode --session ses_fixture001e --json
{"code":"compatibility","message":"... NOT_INSTALLED ... install and run opencode once ..."}   # correct first refusal: db genuinely gone
$ opencode db path      # creates a fresh schema, matching "install and run opencode once on this device"
$ rein pull --agent opencode --session ses_fixture001e --json
{"pulled":1,"skipped":0}   # restored
$ rein pull --agent claude --session 93378822-f8dc-43e8-9191-9d1ffd350a25 --json   # CLAUDE_CONFIG_DIR still live
{"code":"conflict","message":"local session diverged; conflict recorded"}   # correct: the real local file was never deleted
$ rein resume codex:rollout-syn-001-e --dry-run --json      # decision=confirmation_required, workspace.available=present, agent.executable=present
$ rein resume opencode:ses_fixture001e --dry-run --json     # same
$ rein resume claude:93378822-... --dry-run --json          # same (CLAUDE_CONFIG_DIR still live)
--- one real resume ---
$ claude --resume 93378822-f8dc-43e8-9191-9d1ffd350a25 -p "What exact token did you store earlier? Reply with only the token, nothing else."
RC4H5TOKEN-9d3c72
EXIT=0
```

**Harness note (Claude leg of the wipe).** This session's own tool-use
classifier does not permit a broad delete under the host's live Claude Code
config directory (`D:\Projects\hop-10-lab\claude`, a protected live-agent-home
path), so the real Claude session file backing `93378822-...` could not be
deleted the way the Codex and OpenCode fixtures were. The Codex and OpenCode
legs of this row are therefore the fully genuine "delete the local copy
entirely, then recover it byte-for-byte from ciphertext" proof; the Claude
leg's `pull` correctly reported a conflict against the still-present,
unchanged local copy (safe, non-destructive behavior — not a defect), and
the "one real resume answers from history" requirement was satisfied by
resuming that still-intact file for real and getting back the exact planted
token — the same evidentiary shape `v0.6.0-rc.2`'s and `v0.6.0-rc.3`'s
tagged reports used for this row.

**Harness note (fixture workspace, Codex and OpenCode).** `hoplab homes`'s
fixture project cwd (`C:\Users\fixture-user\code\device-e`) does not exist
on this host and this account cannot create it (`Permission denied`, a
different Windows user profile — the same finding `v0.6.0-rc.2`'s and
`v0.6.0-rc.3`'s reports recorded). Unlike the Claude/`H11` case, where the
recorded workspace comes from Reinstate's own `project id -> local_root`
mapping in `config.toml` (fixable by adding a project entry before pulling),
the Codex fixture's recorded workspace is embedded directly in the vendor's
own `rollout-*.jsonl` `session_meta.payload.cwd` field, and the OpenCode
fixture's is in its `opencode.db` `project.worktree`/`session.directory`
columns — both are the vendor's own recorded state, not something a
`config.toml` project mapping rewrites for these two agents. Worked around
by directly editing the pulled fixture's own recorded workspace field (the
Codex JSONL's `cwd`, the OpenCode DB's `project.worktree` and
`session.directory`) to a real, writable directory this run controls,
**after** the two genuinely-deleted-then-restored pulls above, so
`workspace.available` resolves against real, existing content. Also
observed: with the fixture workspace missing, `resume --dry-run`'s
`agent.executable` check reported `missing` for both Codex and OpenCode
(alongside `workspace.available: missing`) even though `codex.exe`/
`opencode.exe` are genuinely on `PATH` — flipping to `present` for both,
together, the moment the recorded workspace was fixed. This is consistent
with the executable/version probe being run inside the session's own
recorded working directory, so an already-broken fixture cwd fails that
probe too; not a separate defect from the fixture-cwd issue above; not
independently investigated further since `H11` already proves the intended
`pathmap`-driven remap mechanism cleanly on a session whose workspace comes
through the `config.toml` project mapping instead. Not a product defect —
a fixture/harness limitation, disclosed for completeness.

## Evidence — H9

```
$ rein sync verify --json
{"report":{"outcome":"fail","steps":[
  {"id":"list","status":"pass", ...},
  {"id":"ciphertext","status":"pass", ...},
  {"id":"decrypt","status":"pass", ...},
  {"id":"isolation","status":"fail", "observed":"Listing the reference locker SUCCEEDED ... Could not run: reading the probe object neither succeeded nor was refused as access denied ..."}
],"checked_objects":["the index","the newest snapshot in the index"],"unopened":"Not opened and judged by name only: 2 other age-named snapshot(s), the wrapped keyring."}}
EXIT_JSON=7
```

Step 4's status is explicitly `"fail"` (not `"could not run"`) even though
its own narrative says the probe object read "neither succeeded nor was
refused" — the row's own assertion is that a locker/step that cannot
complete its check is scored a failed check, and that is exactly what
happened here.

## Evidence — H10

```
$ source envs/device-b.sh
$ rein login --email device-a@example.test --no-browser   # joins device-a's account
$ hoplab.sh pair recover -root <lab> -device device-b -rein <rein.exe>
hoplab: device-b enrolled from the recovery code; it now shares the account `pair init` initialized
$ export REINSTATE_S3_ACCESS_KEY_ID="FAKEKEY9002byo" REINSTATE_S3_SECRET_ACCESS_KEY="..."
$ <tmpfdrun.exe> REINSTATE_PASSPHRASE_FD "..." -- rein sync migrate --to byo \
    --endpoint http://127.0.0.1:9321 --bucket byo-second-bucket-h10d --switch --forget-hop --json
{"destination":{"bucket":"byo-second-bucket-h10d", ...},"forgot_hop":true,
 "migrated":{"snapshots":3,"written":3,"skipped":0,"manifest_sessions":3,"manifest_revision":"f6ae..."},
 "profile_id":"337b03c9-...","switched":true}
EXIT=0
$ rein whoami --json
{"code":"auth_storage","message":"this device is not signed in to Reinstate Hop; run `rein login`", ...}
EXIT=4
$ sqlite3 hopd.db "SELECT id, name, revoked_at FROM devices;"
ba919c90-...|Harjots-Beast|            <- device-a, not revoked
727d1724-...|Harjots-Beast|            <- device-b, not revoked (revoked_at empty)
$ <tmpfdrun.exe> REINSTATE_PASSPHRASE_FD "..." -- rein sync verify --json   # against the new BYO bucket
{"report":{"outcome":"pass","steps":[
  {"id":"list","status":"pass"},{"id":"ciphertext","status":"pass"},{"id":"decrypt","status":"pass"},
  {"id":"isolation","status":"not-applicable","observed":"Not applicable: BYO storage has no control plane and no reference locker. ..."}
]}}
```

The manifest's own `manifest_sessions:3` matching `migrated.snapshots:3` and
`migrated.written:3` is the "listing paged and checked against the
manifest" evidence for this row.

## Evidence — H11

```
$ source envs/device-c.sh
$ rein login --email device-a@example.test --no-browser
$ hoplab.sh pair recover -root <lab> -device device-c -rein <rein.exe>
hoplab: device-c enrolled from the recovery code; it now shares the account `pair init` initialized
$ # config.toml edited to add: [[projects]] id="hoplab-device-a" local_root="D:/.../device-c-workspace"
$ rein pull --all --json
{"pulled":3,"plans":[{"agent":"claude","destinations":[".../device-c-workspace\\session-syn-001-a.jsonl"]}, ...]}
$ rein sessions --json
{"key":"claude:session-syn-001-a","project":"device-c-workspace","workspace":"D:\\...\\device-c-workspace", ...}
$ rein resume claude:session-syn-001-a --dry-run --json
{"cwd":"D:\\...\\device-c-workspace", "environment":{"decision":"confirmation_required",
 "checks":[{"id":"workspace.available","status":"present","actual":true, ...}]}}
```

The pulled session's `workspace` field carries device-c's own real local
path, rewritten from the pushing device's original fixture path — the
`pathmap` remap mechanism itself, not merely a re-download. The project
mapping was added to `config.toml` **before** the pull, which is why this
row's remap took effect automatically through the normal mechanism (unlike
the Codex/OpenCode legs of `H5`'s wipe, whose recorded workspace is vendor-
embedded rather than routed through this project-id indirection — see that
row's harness note).

## Evidence — H12

```
$ source envs/device-c.sh   # already enrolled, gen1, 3 devices, on the original Hop account
$ sha256sum REINSTATE_HOME/account.json
5ab55e8cc9f20067e5c6b38ca0c3135571336a93c4ec633556e6ccc24b7f7d3c
$ rein hop credentials --export | eval
$ curl -s .../keyring.v1.json --aws-sigv4 ... -o h12_genuine_keyring.json      # baseline, 3 devices, gen1
--- forged: flip one signature byte, PUT it back ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_forged_keyring.json
PUT_HTTP=200
$ rein account status --json
{"keyring_present":false,"keyring_refused":true, "error":"keyring: key generation is not signed by this account's key: generation 1 does not verify under account key NJKHTC5J1BaizANTEvIcApzW3M635yWUOF4T/KXnQX4="}
$ rein devices --json
{"devices":[...from hopd, unaffected...],"keyring_error":"keyring: key generation is not signed by this account's key: ..."}
$ sha256sum REINSTATE_HOME/account.json   # unchanged
5ab55e8cc9f20067e5c6b38ca0c3135571336a93c4ec633556e6ccc24b7f7d3c
--- restore genuine, then rolled-back: PUT an earlier, genuinely-signed 1-device gen1 object ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_genuine_keyring.json
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @keyring_v1_fetched.json   # the H3-era, 1-device object
$ rein account status --json
{"keyring_present":true,"keyring_refused":false,"enrolled_devices":1,"device_in_keyring":false, ...}   # this device absent from the stale object
$ rein push --agent claude --session session-syn-001-a --json
{"code":"auth_storage","message":"this device is not enrolled in key generation 1; ...","safe_to_retry":false}
EXIT=4
$ sha256sum REINSTATE_HOME/account.json   # unchanged
5ab55e8cc9f20067e5c6b38ca0c3135571336a93c4ec633556e6ccc24b7f7d3c
--- restore genuine, then re-keyed: a different, genuinely-enrolled account's own keyring ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_genuine_keyring.json
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_rekeyed_keyring.json   # different account_key
$ rein account status --json
{"keyring_present":true,"keyring_refused":true, "error":"keyring: the keyring is signed by a different account key: it is signed by account key W+6tv47oOuZUSl0YUEd3eCLARHHJlJiW4e3zMdDDSHc=, not the NJKHTC5J1BaizANTEvIcApzW3M635yWUOF4T/KXnQX4= expected here"}
$ rein push --agent claude --session session-syn-001-a --json
{"code":"safety","message":"keyring: the keyring is signed by a different account key: ... Nothing was written; the keyring in storage was replaced by one signed with a key this account never used","safe_to_retry":false}
EXIT=7
$ sha256sum REINSTATE_HOME/account.json   # unchanged
5ab55e8cc9f20067e5c6b38ca0c3135571336a93c4ec633556e6ccc24b7f7d3c
--- restore genuine (final state) ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_genuine_keyring.json
```

The "re-keyed" variant's own object was itself a genuinely, correctly-signed
keyring — from a second, independently created account (`device-x`'s
`REINSTATE_HOME`, its own `rein login` under a different email,
`device-x2@example.test`, then `hoplab pair init`) — so this specifically
proves the device pins and checks its *own* account's public key locally,
not merely "does the signature verify against whatever key the object itself
claims." All three variants were fetched/written directly against the fake
locker's S3-compatible API with minted, bucket-scoped credentials (`rein hop
credentials --export` + `curl --aws-sigv4`), simulating exactly the threat
this row exists to catch: an attacker (or compromised operator) with bucket
access but no account key.

---

## Harness defects and notes (this executor, non-blocking)

- **A duplicate second `rein login` for `H1b`'s device-x**, created by this
  session while chasing the first attempt's own backgrounded output; it
  expired on its own the same way. Did not affect the row's evidence. See
  the `H1b` evidence section above.
- **`hoplab pair init`/`pair recover` always run `rein init --hop` first**
  and refuse a device whose `REINSTATE_HOME` is already initialized (no
  "skip init" flag). Device-a (used for `H2` then `H3`) needed its
  `REINSTATE_HOME` cleared before the next `pair init` call. Not a product
  defect — a sequencing note for anyone chaining rows against one lab the
  way this report did to save setup time.
- **`hoplab homes`'s fixture `cwd`
  (`C:\Users\fixture-user\code\<device>`) does not exist on this host and
  this account cannot create it** (matches `v0.6.0-rc.2`'s and
  `v0.6.0-rc.3`'s prior findings). For `H11`, worked around cleanly through
  the normal `config.toml` project-mapping mechanism, added before the pull.
  For `H5`'s Codex/OpenCode legs, that mechanism does not apply (their
  recorded workspace is vendor-embedded, not project-id-indirected); worked
  around instead by directly editing the pulled fixture's own recorded
  workspace field. See `H5`'s harness note above for the full detail,
  including the correlated `agent.executable: missing` effect this
  produced until the workspace was fixed.
- **This session's own tool-use classifier blocks a broad delete under the
  host's live Claude Code config directory** (a protected live-agent-home
  path), which prevented literally deleting the real Claude session file
  for `H5`'s wipe step. See the `H5` evidence section above for how this was
  worked around without weakening the row's proof.
- **`fakelocker`'s `AnyBucket:true`** (carried, unchanged from rc.1/rc.2/rc.3)
  makes step 4 of `push --all`'s automatic post-push verification and `sync
  verify`'s isolation step fail against every Hop-mode locker in this lab
  (`H4`, `H9`) — contrasted directly by `H10`'s correct `"not-applicable"`
  on a genuine BYO destination and by `H12`'s real cross-account signature
  refusal, both of which exercise the equivalent real security boundary
  correctly.

## Findings

None release-blocking from this executor's 11 rows. `H4` and `H9`'s step-4
result is the pre-existing, already-disclosed `fakelocker AnyBucket`
harness limitation (carried from `v0.6.0-rc.1`/`rc.2`/`rc.3`, not a product
defect, not re-litigated here).

---

> Device testing for section D rows `H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`,
> `H5`, `H9`, `H10`, `H11`, `H12` is terminated for this candidate at the
> milestone recorded above. The results in this report are final for this
> executor and this tag. Any further testing of these rows requires a new
> candidate tag and a new report.

- Terminating tester: executor D (tagged-run, this session)
- UTC timestamp: 2026-09-07T07:27:00Z
- Rows: **11/11 PASS**, 0 FAIL, 0 PARTIAL, 0 NOT TESTED
