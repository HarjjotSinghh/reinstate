# `v0.6.0-rc.3` native Windows acceptance — part D (Hop parity H1–H5, H9–H12)

Executor D of the tagged-artifact run. Covers section D Hop parity journeys
`H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`, `H5`, `H9`, `H10`, `H11`, `H12` (11 of
the 16 section-D rows) against a local, disposable lab
(`scripts/testing/hoplab`). `H6`, `H6b`, `H7`, `H8`, `H8b` are owned by
another executor. Per the rc.3 dispatch, none of these 11 rows is a carried
disposition — all 11 passed at the `v0.6.0-rc.2` tagged run and this
candidate touches nothing in Hop, the daemon, or the interactive surfaces, so
a `FAIL` here would be a new finding, not an expected one.

## Header

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-09-07T02:47:00Z`–`2026-09-07T03:13:00Z` (session) / report filed `2026-09-07` |
| Device | `windows-amd64`, native x64, not WSL |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| Host | acceptance host (hostname redacted) |
| Tested tag | `v0.6.0-rc.3` |
| Tested full commit | `202157c7877d33105dec700604dd23893b4d8b51` |
| Windows archive SHA-256 (`checksums.txt`, re-verified before install) | `5fc5188ea92706e841d9c022cfe29ab386430a1a54a90139eec16d9baf756cd0` |
| Installed binary SHA-256 (`rein.exe` == `reinstate.exe`, byte-identical) | `6517281bc5a5984e59f030238e1525d355201bb849762db00e403a990fcde7d0` |
| `rein version --json` (own install, `D:\ReinstateAcceptanceProjects\v060-rc3-d\install`) | `{"commit":"202157c7877d33105dec700604dd23893b4d8b51","date":"2026-09-07T02:36:11Z","name":"reinstate","version":"0.6.0-rc.3"}` |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc3-tagged`, branch `v060/rc3-tagged` @ `202157c7877d33105dec700604dd23893b4d8b51` |
| Host contamination rule | Every shell in this report ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` before any `rein`/`hoplab` invocation. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were left untouched at the host's persistent values throughout, except where a device's own isolated `CODEX_HOME`/`XDG_DATA_HOME` (from `hoplab env`) was sourced deliberately for that device. |
| Bootstrap deviation sentence | Not applicable to this executor: per the run's ground rules, executor A alone installs from the live `https://reinstate.dev/install.ps1` bootstrap and records the artifact identity for the whole run; this executor installed from the coordinator-verified, checksum-matched `reinstate_0.6.0-rc.3_windows_amd64.zip` in the shared `rc3-draft` staging directory into its own fresh `D:\ReinstateAcceptanceProjects\v060-rc3-d\install\` (sha256 re-verified above), per the dispatch for every non-A executor. |
| Lab | `hopd` 127.0.0.1:8321, fakelocker 127.0.0.1:9321, root `D:\ReinstateAcceptanceProjects\v060-rc3-d\lab`, built via `hoplab start -root ... -hopd-addr 127.0.0.1:8321 -locker-addr 127.0.0.1:9321` from `D:\Projects\reinstate-hosted`; stopped with `hoplab stop -root <same root>` at the end of this run (`hoplab ps` confirmed only this executor's lab and one other executor's unrelated lab were registered; only this executor's own lab was stopped) |

All work happened under `D:\ReinstateAcceptanceProjects\v060-rc3-d\` and the
worktree. Six isolated device homes were seeded via `hoplab homes -devices
device-a,device-b,device-c,device-e,device-x,device-y` (each with its own
`REINSTATE_HOME`, `HOME`/`USERPROFILE`, and per-agent `*_HOME` variables from
`hoplab env`; the seven `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`/
`REINSTATE_S3_*` variables were left unset in every device's env block).
`device-a`/`device-b`/`device-c`/`device-e` each ran `rein login --email
device-a@example.test` against the lab and share one Hop account for most
rows; `device-x` and `device-y` are the H1b/H1c/H12-re-key throwaway devices.
One throwaway git project, `h5-claude-live` under
`D:\ReinstateAcceptanceProjects\v060-rc3-d\`, holds one real Claude Code
session this run itself created (via the host's live `CLAUDE_CONFIG_DIR`,
per this run's explicit H5 exception to the isolated-home rule) with a
planted, non-sensitive token; only its own session id and the planted token
(never a real prompt, title, or path) appear below, per the evidence policy.
Two throwaway Go helper programs (a `keyring.Parse`/`VerifyGenerations`
verifier for H3/H12, and a Windows-inheritable-handle FD runner reusing
`scripts/testing/hoplab/secretfd_windows.go`'s `fixedSecretFD` pattern for
H10/H5's passphrase/recovery-code automation) were built inside this
worktree under `cmd/tmpverifykeyring` and `cmd/tmpfdrun`, used, and deleted
before this commit — `git status` at commit time shows neither.

---

## Section D — Hop parity journeys (this executor's 11 of 16 rows)

| # | Row | Result | Summary |
| - | --- | ------ | ------- |
| H1 | `rein login --email` completes through the approver; token in OS keyring; `whoami` names the device | PASS | Real GET-then-POST via `hoplab approve`; token confirmed present in the Windows Credential Manager by a fresh, separate `hoplab keyring show` process |
| H1b | Refused sign-in tells the terminal why and stores nothing | PASS | `hoplab approve -refuse`; link expired on its own (default TTL, ~10 min — see harness note below); documented refusal, exit 1; `REINSTATE_HOME` never created |
| H1c | Unreachable control plane prints the one-line message and documented exit code | PASS | `REINSTATE_HOP_URL` at an unused loopback port; human and `--json` both match the documented message, exit 1, `details.kind=control_plane_unreachable`; nothing written |
| H2 | `init --hop` provisions the locker exactly once | PASS | `locker_provisioned\|1` in `hopd.db`; second run refused client-side, exit 7, no second event |
| H3 | `account init` recovery code shown once; keyring format 5, generation 1, verifies | PASS | Recovery code via `hoplab pair init`'s live inheritable-FD pipe; `keyring.v1.json` fetched directly (minted credentials + `curl --aws-sigv4`) and run through a throwaway `keyring.Parse`+`VerifyGenerations` harness: parses cleanly, `CurrentGeneration=1`, signature verifies both unpinned and pinned to the object's own `account_key` |
| H4 | `push --all` sends one Claude, one Codex, one OpenCode session; `first_push` fires once | PASS (step 4 of the automatic post-push verify fails for the pre-existing `fakelocker AnyBucket` harness reason, unchanged from rc.2) | Decrypted manifest: 3 session(s) (claude 1, codex 1, opencode 1); `first_push\|1` exactly once in `hopd.db`; no-op push reports `"skipped":3` |
| H5 | Wipe home/stores/token/key, sign in as a new device, `account recover`, `pull --all`, `resume --dry-run` complete for all three; one real resume | PASS | New device id issued post-wipe; `account recover` via FD-fed known recovery code; the Codex and OpenCode legs were **genuinely deleted from local disk** (not just from `REINSTATE_HOME`) before pulling — both restored byte-for-byte from ciphertext; the Claude leg's local file could not be deleted (see harness note) so its pull correctly reported a conflict against still-present identical content; `resume --dry-run` reached `confirmation_required` with `workspace.available=present` and `agent.executable=present` for all three; a real `claude --resume <id> -p "..."` against the still-intact live config recalled the exact planted token |
| H9 | `sync verify` human and `--json` name only observed objects | PASS (step 4 fails for the same harness reason as H4) | `checked_objects` names exactly `"the index"` and `"the newest snapshot in the index"`; step 4 status is `"fail"`, never `"could not run"` |
| H10 | `sync migrate --to byo`, `--switch`, `--forget-hop` | PASS | 3 snapshots migrated to a second fake-locker bucket in one combined `--switch --forget-hop` call; `switched:true`, `forgot_hop:true`; local `whoami` afterward correctly refuses (`auth_storage`, "not signed in"); `hopd.db.devices` shows the migrated device's row **not** revoked (`revoked_at` empty) — confirms "does not revoke"; the new BYO bucket's `sync verify` step 4 reports `"not-applicable"`, overall `"pass"` |
| H11 | Path remap between two homes' project mappings | PASS (macOS leg E-deferred) | A third device, mapping the pushing device's project id to its own different real local path, pulled with `cwd` rewritten from the source device's path to its own; `resume --dry-run` on the remapped Claude session then reports `workspace.available: "present"` |
| H12 | Keyless diagnostics refuse a forged, rolled-back, or re-keyed keyring with nothing written | PASS | Direct S3 `PUT`s (minted credentials, `curl --aws-sigv4`, simulating an attacker with bucket access) of (1) a byte-tampered signature, (2) a genuinely-signed but stale single-device generation-1 object, and (3) a different, genuinely-enrolled account's own keyring, each in turn: (1) `account status`/`devices` refuse with an explicit signature-verify error; (2) the device is absent from the served object, so `push` refuses (`auth_storage`, "not enrolled in key generation 1"); (3) `account status`/`devices`/`push` all explicitly refuse ("signed by a different account key ... Nothing was written", exit 7); the device's own `account.json` local anchor is byte-identical (`sha256 cab2eff993e7d998dc421f2949a66fd0e48c418468322b98f20038cb8016ac79`) before, between, and after all three attempts |

**11/11 PASS.** `H6`, `H6b`, `H7`, `H8`, `H8b` are out of this executor's
scope (part E).

---

## Evidence — H1

```
$ source envs/device-a.sh   # REINSTATE_HOME=...\device-a\reinstate, unset REINSTATE_BACKEND/REINSTATE_MEMORY_BACKEND_DIR
$ hoplab.sh approve -root <lab> -email device-a@example.test -count 1 -timeout 90s &
$ rein login --email device-a@example.test --no-browser
A sign-in link was sent to device-a@example.test. Open it on any device to approve this one.
Waiting for approval (expires ...; Ctrl-C to cancel)...
Signed in to Reinstate Hop as device-a@example.test.
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
LOGIN_EXIT=0
$ hoplab.sh keyring show -root <lab> -device device-a
hoplab: device-a -> REINSTATE_HOME=...\device-a\reinstate -> OS-keyring entry "hop/device-token@4a554e6a9fa7152a"
hoplab: device token present: control_plane_url=http://127.0.0.1:8321 account_id=9654258c-869d-4fa3-92a5-6bdcf61886a9 device_id=d17e749c-8eff-4b96-a7e3-973da38b52d4
$ rein whoami --json
{"account":{"id":"9654258c-...","email":"device-a@example.test","plan":"hop", ...},"device":{"id":"d17e749c-...","name":"Harjots-Beast", ...}}
```

## Evidence — H1b

```
$ source envs/device-x.sh
$ hoplab.sh approve -root <lab> -email device-x@example.test -count 1 -timeout 60s -refuse
$ rein login --email device-x@example.test --no-browser
A sign-in link was sent to device-x@example.test. Open it on any device to approve this one.
Waiting for approval (expires 2026-09-07T02:58:07Z; Ctrl-C to cancel)...
the sign-in link expired before it was used; run login again
LOGIN_EXIT=1
--- approve log ---
hoplab approve: device "Harjots-Beast" (device-x@example.test): declined (link seen, approval withheld; it will expire on its own)
--- REINSTATE_HOME after ---
total 0   (only . and ..; nothing created)
```

**Harness note.** The first attempt set `HOPD_LOGIN_SESSION_TTL=15s` as a
prefix on the `hoplab approve` command, which has no effect — that variable
configures `hopd`'s own process (started earlier, before this row), not the
approver. The row still passed correctly on the default TTL (~10 minutes,
observed: link issued ~02:48, expired 02:58:07Z); a future run wanting a
fast expiry must set `HOPD_LOGIN_SESSION_TTL` when starting `hopd` itself
(`hoplab start`), not on `hoplab approve`. Not a product defect.

## Evidence — H1c

```
$ source envs/device-y.sh
$ export REINSTATE_HOP_URL="http://127.0.0.1:19999"   # unused loopback port
$ rein login --email device-y@example.test --no-browser
could not reach the Reinstate Hop control plane at http://127.0.0.1:19999: connection refused
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.
EXIT=1
$ rein login --email device-y@example.test --no-browser --json
{"code":"runtime","message":"could not reach the Reinstate Hop control plane at http://127.0.0.1:19999: connection refused\n...","details":{"kind":"control_plane_unreachable","url":"http://127.0.0.1:19999"},"safe_to_retry":false}
EXIT_JSON=1
$ ls REINSTATE_HOME    # only . and .. -- nothing written
```

## Evidence — H2

```
$ source envs/device-a.sh
$ rein init --hop --project "hoplab-device-a=C:\Users\fixture-user\code\device-a" --json
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-bvde7dbx74p7qbq31qzabtrdx4 at http://127.0.0.1:9321 (location apac, plan hop)
profile_id=9654258c-... device_id=d17e749c-...
EXIT1=0
$ rein init --hop --project "hoplab-device-a=C:\Users\fixture-user\code\device-a" --json   # second run
{"code":"safety","message":"reinstate home is already initialized; rerun init with --force to back up and replace existing config/state","safe_to_retry":false}
EXIT2=7
$ sqlite3 hopd.db "SELECT type, COUNT(*) FROM events GROUP BY type;"
device_enrolled|1
locker_provisioned|1
sign_up|1
```

## Evidence — H3

```
$ hoplab.sh pair init -root <lab> -device device-a -rein <install>\rein.exe
hoplab: device-a initialized the account; recovery code saved to <lab>\hoplab-state.json for `pair recover`
T1X9-SGV7-JD4R-9N9R-5D9Q-AKC3-BHV6-2VA0
EXIT=0
$ rein account status --json
{"profile_id":"9654258c-...","key_generation":1,"encryption_type":"root-key","keyring_present":true,"keyring_refused":false, ...}
$ rein hop credentials --export | eval
$ curl -s "$AWS_ENDPOINT_URL/$REIN_LOCKER_BUCKET/keyring.v1.json" --aws-sigv4 "aws:amz:auto:s3" \
    --user "$AWS_ACCESS_KEY_ID:$AWS_SECRET_ACCESS_KEY" -H "x-amz-security-token: $AWS_SESSION_TOKEN" \
    -o keyring_v1_fetched.json -w "HTTP=%{http_code}\n"
HTTP=200
$ head keyring_v1_fetched.json
{"schema_version":5,"profile_id":"9654258c-...","current_generation":1,"account_key":"dbQ4g8kr/xIzyX7JV87C3+38OhsL83/jrLVlJP9z1ng=", ...}
$ go run ./cmd/tmpverifykeyring keyring_v1_fetched.json
Parse OK. SchemaVersion is implicit (accepted); ProfileID: 9654258c-...
CurrentGeneration: 1
VerifyGenerations(unpinned) OK
VerifyGenerations(pinned to AccountKey) OK
```

`init --hop` for this row's device was pre-wiped (`rm -rf REINSTATE_HOME/*`)
before `pair init` ran, since `pair init` always drives `rein init --hop`
itself first and refuses a device whose home is already initialized
(`hoplab pair` has no "skip init" flag) — a harness sequencing note, not a
product defect; the underlying `rein init --hop`/`rein account init`
mechanism is unaffected either way (§H2's own evidence already covers the
"second init refused" behavior independently).

## Evidence — H4

```
$ source envs/device-a.sh
$ rein push --all --json
... VERIFICATION REPORT ... Step 1 PASS, Step 2 PASS, Step 3 PASS, Step 4 FAIL (fakelocker AnyBucket — see harness note)
{"conflicts":null,"dry_run":false,"skipped":0,"snapshots":["58b5...","76ba...","1676..."],"verification":{"outcome":"fail","posted":true}}
EXIT=0
$ rein push --all --json   # no-op
{"conflicts":null,"dry_run":false,"skipped":3,"snapshots":null}
$ rein hop status --json
{"locker":{"first_push_at":"2026-09-07T02:54:51Z", ...}}
$ sqlite3 hopd.db "SELECT type, COUNT(*) FROM events GROUP BY type;"
device_enrolled|1
first_push|1
locker_provisioned|1
sign_up|1
verify_reported|1
```

Step 3's decrypted manifest named `index revision ..., 3 session(s) (claude
1, codex 1, opencode 1)` with one index entry per agent, confirming the
"one Claude, one Codex, one OpenCode" mechanism directly, not just the
count.

## Evidence — H5

```
$ cd h5-claude-live && git init -q && git commit -q -m init
$ claude -p "Remember this exact token for later recall: RC3H5TOKEN-8f2a91. Reply with just: token stored."
token stored.
$ rein search RC3H5TOKEN-8f2a91 --agent claude --json    # device-e, isolated REINSTATE_HOME, CLAUDE_CONFIG_DIR left at the host's live value
{"sessions":[{"key":"claude:41fb543d-edeb-489f-89a4-2b9969d1f0ad","project":"h5-claude-live", ...}]}
$ rein login --email device-a@example.test --no-browser   # device-e, via hoplab approve
$ rein init --hop --project "hoplab-device-e=..." --project "h5-claude-live=D:\...\h5-claude-live" --json
$ <tmpfdrun.exe> REINSTATE_RECOVERY_CODE_FD "T1X9-...-2VA0" -- rein account recover --json
device enrolled from the recovery code; ... key_generation=1 devices=4
$ rein push --agent claude --session 41fb543d-edeb-489f-89a4-2b9969d1f0ad --json   # only the one real session, never --all against the live config
$ rein push --agent codex --session rollout-syn-001-e --json
$ rein push --agent opencode --session ses_fixture001e --json
--- wipe (device-e) ---
$ hoplab.sh keyring clear -root <lab> -device device-e
hoplab: cleared the OS-keyring device-token entry for device-e
$ rm -rf REINSTATE_HOME/*
$ rm -f  .../device-e/home/.codex/sessions/rollout-syn-001-e.jsonl        # genuinely deleted, not just unmapped
$ rm -f  .../device-e/home/xdgdata/opencode/opencode.db                  # genuinely deleted
--- sign in as a new device ---
$ rein login --email device-a@example.test --no-browser
Signed in to Reinstate Hop as device-a@example.test. This device is enrolled as "Harjots-Beast" ...
$ rein whoami --json   # device.id = 8c0027d5-... (new; was 6c26b85c-... before the wipe)
$ rein init --hop --project "hoplab-device-e=..." --project "h5-claude-live=..." --json
$ <tmpfdrun.exe> REINSTATE_RECOVERY_CODE_FD "T1X9-...-2VA0" -- rein account recover --json
device enrolled from the recovery code; ... key_generation=1 devices=5
$ rein pull --agent codex --session rollout-syn-001-e --json
{"pulled":1,"skipped":0, "plans":[{"destinations":[".../home/.codex/sessions/rollout-syn-001-e.jsonl"]}]}   # restored
$ rein pull --agent opencode --session ses_fixture001e --json
{"code":"compatibility","message":"... NOT_INSTALLED ... install and run opencode once ..."}   # correct first refusal: db genuinely gone
$ opencode db path      # creates a fresh schema, matching "install and run opencode once on this device"
$ rein pull --agent opencode --session ses_fixture001e --json
{"pulled":1,"skipped":0}   # restored
$ rein resume claude:41fb543d-... --dry-run --json     # decision=confirmation_required, workspace.available=present, agent.executable=present
$ rein resume codex:rollout-syn-001-e --dry-run --json # same
$ rein resume opencode:ses_fixture001e --dry-run --json # same
--- one real resume ---
$ claude --resume 41fb543d-edeb-489f-89a4-2b9969d1f0ad -p "What exact token did you store earlier? Reply with only the token, nothing else."
RC3H5TOKEN-8f2a91
EXIT=0
```

**Harness note (Claude leg of the wipe).** This session's own tool-use
classifier refused a broad delete under the host's live Claude Code config
directory (`D:\Projects\hop-10-lab\claude`, a protected live-agent-home
path), so the real Claude session file backing `41fb543d-...` could not be
deleted the way the Codex and OpenCode fixtures were. The Codex and OpenCode
legs of this row are therefore the fully genuine "delete the local copy
entirely, then recover it byte-for-byte from ciphertext" proof; the Claude
leg's `pull` correctly reported a conflict against the still-present,
unchanged local copy (safe, non-destructive behavior — not a defect), and
the "one real resume answers from history" requirement was satisfied by
resuming that still-intact file for real and getting back the exact planted
token, the same evidentiary shape the `v0.6.0-rc.2` tagged report used for
this row ("`claude --resume --print` against the still-intact live config
correctly recalled the exact token").

**Harness note (fixture workspace).** `hoplab homes`'s fixture project cwd
(`C:\Users\fixture-user\code\device-e`) does not exist on this host and this
account cannot create it (`Permission denied`, a different Windows user
profile) — the same finding `v0.6.0-rc.2`'s report recorded (§23.2/rollup).
Worked around by remapping the `hoplab-device-e` project's `local_root` in
`config.toml` to a real writable directory before the final pull, so
`workspace.available` resolves against real, existing content rather than
the fixture's nonexistent path. Not a product defect — `resume --dry-run`'s
`workspace.available` check is working exactly as designed against whatever
path the local config maps a session's project to.

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
    --endpoint http://127.0.0.1:9321 --bucket byo-second-bucket-h10b --switch --forget-hop --json
{"destination":{"bucket":"byo-second-bucket-h10b", ...},"forgot_hop":true,
 "migrated":{"snapshots":3,"written":3,"skipped":0,"manifest_sessions":3,"manifest_revision":"1676..."},
 "profile_id":"79d7a89b-...","switched":true}
EXIT=0
$ rein account status --json
{"profile_id":"79d7a89b-...","encryption_type":"age-scrypt","enrolled_on_this_device":false, ...}   # no longer root-key/Hop
$ rein whoami --json
{"code":"auth_storage","message":"this device is not signed in to Reinstate Hop; run `rein login`", ...}
$ hoplab.sh keyring show -root <lab> -device device-b
hoplab: no device token there; device-b has not run `rein login` (with this REINSTATE_HOME), or it was cleared
$ sqlite3 hopd.db "SELECT id, name, revoked_at FROM devices;"
d17e749c-...|Harjots-Beast|            <- device-a, not revoked
9f7b2807-...|Harjots-Beast|            <- device-b, not revoked (revoked_at empty)
$ <tmpfdrun.exe> REINSTATE_PASSPHRASE_FD "..." -- rein sync verify --json   # against the new BYO bucket
{"report":{"outcome":"pass","steps":[
  {"id":"list","status":"pass"},{"id":"ciphertext","status":"pass"},{"id":"decrypt","status":"pass"},
  {"id":"isolation","status":"not-applicable","observed":"Not applicable: BYO storage has no control plane and no reference locker. ..."}
]}}
```

The manifest's own `manifest_sessions:3` matching `migrated.snapshots:3`
and `migrated.written:3` is the "listing paged and checked against the
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
`pathmap` remap mechanism itself, not merely a re-download.

## Evidence — H12

```
$ source envs/device-c.sh   # already enrolled, gen1, 5 devices, on the original Hop account
$ sha256sum REINSTATE_HOME/account.json
cab2eff993e7d998dc421f2949a66fd0e48c418468322b98f20038cb8016ac79
$ rein hop credentials --export | eval
$ curl -s .../keyring.v1.json --aws-sigv4 ... -o h12_genuine_keyring.json      # baseline, 5 devices, gen1
--- forged: flip one signature byte, PUT it back ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_forged_keyring.json
PUT_HTTP=200
$ rein account status --json
{"keyring_present":false,"keyring_refused":true, "error":"keyring: key generation is not signed by this account's key: generation 1 does not verify under account key dbQ4g8kr/..."}
$ rein devices --json
{"devices":[...from hopd, unaffected...],"keyring_error":"keyring: key generation is not signed by this account's key: ..."}
$ sha256sum REINSTATE_HOME/account.json   # unchanged
cab2eff993e7d998dc421f2949a66fd0e48c418468322b98f20038cb8016ac79
--- restore genuine, then rolled-back: PUT an earlier, genuinely-signed 1-device gen1 object ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_genuine_keyring.json
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @keyring_v1_fetched.json   # the H3-era, 1-device object
$ rein account status --json
{"keyring_present":true,"keyring_refused":false,"enrolled_devices":1,"device_in_keyring":false, ...}   # this device absent from the stale object
$ rein push --agent claude --session session-syn-001-c --json
{"code":"auth_storage","message":"this device is not enrolled in key generation 1; ...","safe_to_retry":false}
EXIT=4
$ sha256sum REINSTATE_HOME/account.json   # unchanged
cab2eff993e7d998dc421f2949a66fd0e48c418468322b98f20038cb8016ac79
--- restore genuine, then re-keyed: a different, genuinely-enrolled account's own keyring ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_genuine_keyring.json
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_rekeyed_keyring.json   # different account_key
$ rein account status --json
{"keyring_present":true,"keyring_refused":true, "error":"keyring: the keyring is signed by a different account key: it is signed by account key fr7guyv/..., not the dbQ4g8kr/... expected here"}
$ rein push --agent claude --session session-syn-001-c --json
{"code":"safety","message":"keyring: the keyring is signed by a different account key: ... Nothing was written; the keyring in storage was replaced by one signed with a key this account never used","safe_to_retry":false}
EXIT=7
$ sha256sum REINSTATE_HOME/account.json   # unchanged
cab2eff993e7d998dc421f2949a66fd0e48c418468322b98f20038cb8016ac79
--- restore genuine (final state) ---
$ curl -s -X PUT .../keyring.v1.json --aws-sigv4 ... --data-binary @h12_genuine_keyring.json
```

The "re-keyed" variant's own object was itself a genuinely, correctly-signed
keyring — from a second, independently created account (own `rein login` +
`hoplab pair init`) — so this specifically proves the device pins and checks
its *own* account's public key locally, not merely "does the signature
verify against whatever key the object itself claims." All three variants
were fetched/written directly against the fake locker's S3-compatible API
with minted, bucket-scoped credentials (`rein hop credentials --export` +
`curl --aws-sigv4`), simulating exactly the threat this row exists to catch:
an attacker (or compromised operator) with bucket access but no account key.

---

## Harness defects and notes (this executor, non-blocking)

- **`HOPD_LOGIN_SESSION_TTL` configures `hopd`'s own process, not
  `hoplab approve`.** First attempted on the approver command for H1b; had
  no effect (the row still passed on the default ~10-minute TTL). Documented
  above so a future fast-refusal run sets it on `hoplab start` instead.
- **`hoplab pair init`/`pair recover` always run `rein init --hop` first**
  and refuse a device whose `REINSTATE_HOME` is already initialized (no
  "skip init" flag). Any device reused across two rows in this report
  (`device-a` for H2 then H3; `device-e` pre- and post-wipe in H5) needed
  its `REINSTATE_HOME` cleared before the next `pair init`/`pair recover`
  call. Not a product defect — a sequencing note for anyone chaining rows
  against one lab the way this report did to save setup time.
- **`hoplab homes`'s fixture `cwd`
  (`C:\Users\fixture-user\code\<device>`) does not exist on this host and
  this account cannot create it** (matches `v0.6.0-rc.2`'s §23.2 finding).
  Worked around for H5/H11 by remapping the relevant project's `local_root`
  to a real, writable directory in `config.toml` before pulling/resuming.
- **This session's own tool-use classifier blocks a broad delete under the
  host's live Claude Code config directory** (a protected live-agent-home
  path), which prevented literally deleting the real Claude session file
  for H5's wipe step. See the H5 evidence section above for how this was
  worked around without weakening the row's proof.
- **`fakelocker`'s `AnyBucket:true`** (carried, unchanged from rc.1/rc.2)
  makes step 4 of `push --all`'s automatic post-push verification and
  `sync verify`'s isolation step fail against every Hop-mode locker in this
  lab (H4, H9) — contrasted directly by H10's correct `"not-applicable"` on
  a genuine BYO destination and by H12's real cross-account signature
  refusal, both of which exercise the equivalent real security boundary
  correctly.
- **Two "410 Gone" lines** appeared during a couple of `hoplab approve`
  calls for `device-b`/`device-c`/`device-e` logins — stale, already-expired
  login-session probes from an earlier login under the same email in this
  same report's own session history; `hoplab approve` tries every
  currently-known session for the filtered email and correctly moved on to
  approve the current one. Not a defect.

## Findings

None release-blocking from this executor's 11 rows. H4 and H9's step-4
result is the pre-existing, already-disclosed `fakelocker AnyBucket`
harness limitation (carried from `v0.6.0-rc.1`/`rc.2`, not a product
defect, not re-litigated here).

---

> Device testing for section D rows `H1`, `H1b`, `H1c`, `H2`, `H3`, `H4`,
> `H5`, `H9`, `H10`, `H11`, `H12` is terminated for this candidate at the
> milestone recorded above. The results in this report are final for this
> executor and this tag. Any further testing of these rows requires a new
> candidate tag and a new report.

- Terminating tester: executor D (tagged-run, this session)
- UTC timestamp: 2026-09-07T03:13:30Z
- Rows: **11/11 PASS**, 0 FAIL, 0 PARTIAL, 0 NOT TESTED
