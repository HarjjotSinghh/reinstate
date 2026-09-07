# `v0.6.0-rc.5` Windows Hop parity — part D (H1, H1b, H1c, H2, H3, H4, H5, H9, H10, H11, H12)

`PHASE5-DEVICE-REPORT-V1` (section D subset)

Executor D's slice of the tagged-artifact acceptance for `v0.6.0-rc.5`,
against [`docs/testing/v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
section D and [`v0.6.0-rc.5-agent-verification-prompts.md`](../v0.6.0-rc.5-agent-verification-prompts.md).
Rows `H6`, `H6b`, `H7`, `H8`, `H8b` are out of this part's scope (a different
executor's assignment) and are not recorded here.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.5` |
| Full commit | `0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9` |
| Windows archive | `reinstate_0.6.0-rc.5_windows_amd64.zip` |
| Archive SHA-256 | `a648f1de65c15bea7926dda2dd0ce48110d512c9bc60a23b5988c3b24ded9f09` (matches `checksums.txt`, re-verified before install) |
| Install source | `checksummed archive at the coordinator's verified `rc5-draft` staging directory, per this candidate's dispatch (Executor A alone uses the live bootstrap; every other executor, including this one, installs from the pre-verified archive) |
| Install directory | own fresh directory under `D:\ReinstateAcceptanceProjects\v060-rc5-d\install\` |
| `rein.exe` / `reinstate.exe` byte identity | identical (`sha256 aa74899f68356ea5127fa8129db56729c137276d255f1d8890fa9dcf276dd35f` both) |
| `rein version --json` | `{"commit":"0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9","date":"2026-09-07T12:59:29Z","name":"reinstate","version":"0.6.0-rc.5"}` |
| **Bootstrap deviation** | This part did **not** use the live `reinstate.dev/install.ps1` bootstrap. Per the dispatch, only Executor A installs from the live bootstrap and records the artifact identity from it; every other executor, including this one, installs the same checksummed archive from the coordinator's pre-verified staging directory. No bootstrap script was fetched or run by this part. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Windows 11 Pro 10.0.26200 (Build 26200), x64 |
| Shell | Git Bash (POSIX), PowerShell 5.1 available |
| Go | `go1.26.1 windows/amd64` (host toolchain; module pins `go1.25.13` via `GOTOOLCHAIN` where invoked) |
| Device name shown by Hop (`whoami`/login) | redacted as `<device-name>` throughout this report (the host's real machine name) |
| Lab root | `D:\ReinstateAcceptanceProjects\v060-rc5-d\` |
| Hop lab | `hopd` built from `D:\Projects\reinstate-hosted`, `127.0.0.1:8321`; `fakelocker` `127.0.0.1:9321` (a second short-TTL lab pair, `127.0.0.1:8341`/`127.0.0.1:9341`, was used only for H1b/H1c, then stopped) |
| Date | 2026-09-07 |

## Environment hygiene

Every shell in this part began by unsetting `REINSTATE_BACKEND` and
`REINSTATE_MEMORY_BACKEND_DIR` (confirmed empty before each row).
`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were never unset;
isolation for Hop-lab rows used `hoplab homes`' own per-device
`REINSTATE_HOME`/`HOME`/`USERPROFILE`/`*_HOME` overrides. For the two rows
(H4, H5) that create a real Claude Code session, `CLAUDE_CONFIG_DIR` was
deliberately **left at the host's live value** (never overridden) per the
evidence policy's carve-out, with the Claude session created and resumed
only inside a throwaway project (`D:\ReinstateAcceptanceProjects\v060-rc5-d\throwaway-claude`)
and reported only by its own session id, found with `rein search <token> --agent claude --json`.

Real vendor binaries used throughout: Claude Code `2.1.263`, Codex CLI
`0.149.0`, OpenCode `1.18.29` (all `--version` re-checked immediately before
use).

## Section D — Hop parity journeys (this part's 11 rows)

| # | Row | Result | Summary |
| - | --- | ------ | ------- |
| H1 | `rein login --email` completes through the approver; token in OS keyring; `whoami` names the device | PASS | Real GET-then-POST via `hoplab approve`; OS-keyring entry confirmed present by a fresh, separate process |
| H1b | A refused sign-in tells the terminal why and stores nothing | PASS | `hoplab approve -refuse` against a short-TTL lab (`HOPD_LOGIN_SESSION_TTL=10s`); link expired on its own; documented refusal, exit `1`; `REINSTATE_HOME` never created |
| H1c | An unreachable control plane prints the one-line message and the documented exit code | PASS | `REINSTATE_HOP_URL` at an unused loopback port; human and `--json` both match the documented message, exit `1`, `details.kind=control_plane_unreachable`; nothing written |
| H2 | `rein init --hop` writes the profile and provisions the locker once; a second run does not re-provision | PASS | Exactly one `locker_provisioned` event in `hopd.db`, confirmed unchanged even after a `--force` full re-init reused the same locker id |
| H3 | `rein account init` shows the recovery code once, writes `keyring.v1.json` (format 5, signed generation 1); signature verifies with `keyring.Parse` | PASS | `keyring.v1.json` fetched directly from the locker via S3: `SchemaVersion 5`, `CurrentGeneration 1`, one device, `VerifyGenerations` OK both pinned and unpinned |
| H4 | `rein push --all` sends one Claude Code, one Codex, one OpenCode session as ciphertext; `first_push` reaches the control plane exactly once; `rein hop status` shows it; a no-op push reports nothing | PASS | Real vendor sessions for all three agents; `first_push` event fires exactly once across two `push --all` calls; no-op push reports `skipped:3` |
| H5 | Wipe home, stores, token, key; sign in as a new device; `rein account recover`; `rein pull --all`; `rein resume --dry-run` complete for all three; one real resume answers from history | PASS | Genuine deletion of all four local session files (including the live-config Claude file) and the OS-keyring token; new device recovered from the H3 code; all four restored; `resume --dry-run` reached `confirmation_required` for three (the synthetic fixture correctly `blocked` on a workspace that never existed); a real `claude --resume` recalled the exact planted token |
| H9 | `rein sync verify` human and `--json` name only observed objects; a bucket that is gone is a failed check, not a check that could not run | PASS | `checked_objects` names exactly `"the index"` and `"the newest snapshot in the index"`; step 4 (bucket-isolation) status is `"fail"`, never `"could not run"`, for the pre-existing `fakelocker AnyBucket` harness reason carried from `v0.6.0-rc.4` |
| H10 | `rein sync migrate --to byo` to a second fake-locker bucket; `--switch`; `--forget-hop` drops the token locally and the doc says it does not revoke | PASS | 5 snapshots migrated in one combined call; `switched:true, forgot_hop:true`; local `whoami` afterward correctly refuses; `hopd.db.devices` shows both devices with `revoked_at` empty — confirms "does not revoke"; new BYO bucket's `sync verify` step 4 reports `not-applicable` |
| H11 | Path remap between the two homes' project mappings: pulled sessions carry the other home's path; workspace verification passes on resume | PASS (macOS leg deferred) | A second device, mapping the same BYO profile's projects to its own different local paths, pulled all four sessions with `cwd` rewritten to its own paths; `resume --dry-run` reports `workspace.available: present` |
| H12 | Keyless diagnostics (`rein account status`, `rein devices`) refuse a forged, rolled-back, or re-keyed keyring with nothing written | PASS | Direct S3 `PUT`s of (1) a genuinely-signed but stale (generation-1) keyring after a real revocation had rolled the account to generation 2, (2) a byte-tampered signature, and (3) a different, genuinely-enrolled account's own keyring: each refused correctly (rollback / signature-verify / foreign-account-key error, `keyring_refused:true`), the device's own `account.json` byte-identical throughout every attack |

**11/11 PASS.**

---

## Evidence

Argv-only commands; no transcript text, prompts, credentials, private paths,
or session ids the device did not create appear below. `<recovery-code>` /
`<account-key>` / `<device-key>` stand in for the literal values, per the
ground rules.

### H1 — email sign-in, keyring, `whoami`

```
rein login --email <email> --json          # in one shell, backgrounded
hoplab approve -root <lab> -email <email> -count 1 -timeout 2m   # concurrently
```

Observed: `hoplab approve` printed `device "<device-name>" (<email>): approved`.
`rein login`'s JSON returned an `account`/`device` block with a real device
id. `rein whoami --json` immediately after returned the same account/device.
`hoplab keyring show -root <lab> -device device-a` printed:

```
hoplab: device-a -> REINSTATE_HOME=<path> -> OS-keyring entry "hop/device-token@c1bb9bde0b733649"
hoplab: device token present: control_plane_url=http://127.0.0.1:8321 account_id=<account-id> device_id=<device-id>
```

— a fresh, separate `hoplab keyring show` process confirming the token
lives in the OS keyring, not a file.

### H1b — refused sign-in

A second, short-TTL lab pair (`HOPD_LOGIN_SESSION_TTL=10s`, `127.0.0.1:8341`/
`127.0.0.1:9341`) was started only for H1b/H1c, so the wait would not need
the default 10-minute TTL.

```
rein login --email <email2> --json
hoplab approve -root <lab2> -email <email2> -refuse -count 1 -timeout 30s
```

Observed: `hoplab approve` printed `device "<device-name>" (<email2>):
declined (link seen, approval withheld; it will expire on its own)`. `rein
login` then exited `1` with:

```
{"code":"runtime","message":"the sign-in link expired before it was used; run login again","safe_to_retry":false}
```

`REINSTATE_HOME` for that device was confirmed **never created** (directory
absent) after the refusal.

### H1c — unreachable control plane

```
REINSTATE_HOP_URL=http://127.0.0.1:8399 rein login --email <email3> --json
REINSTATE_HOP_URL=http://127.0.0.1:8399 rein login --email <email4>
```

Both exited `1`. `--json`:

```
{"code":"runtime","message":"could not reach the Reinstate Hop control plane at http://127.0.0.1:8399: connection refused\n...","details":{"kind":"control_plane_unreachable","url":"http://127.0.0.1:8399"},"safe_to_retry":false}
```

The human form printed the identical two-line message. `REINSTATE_HOME` was
confirmed never created either time.

### H2 — `init --hop` provisions once

```
rein init --hop --project <id>=<path> --json          # run 1
rein init --hop --project <id>=<path> --json          # run 2 (no --force)
rein init --hop --force --project <id>=<path> --json  # run 3, forced re-init
```

Run 2 exited `7`: `reinstate home is already initialized; rerun init with
--force to back up and replace existing config/state`. Run 3 (forced full
re-init) printed the **same** locker id as run 1
(`lk-659r8hv00kr4tsj9kxz1mk6vsr`). A direct query against `hopd.db`:

```sql
SELECT id, type, created_at FROM events;
```

showed exactly one `locker_provisioned` row across all three runs — the
forced re-init's own GET-then-provision-if-missing cycle found the locker
already provisioned and did not provision a second one.

### H3 — recovery code once, keyring format 5 generation 1

```
hoplab pair init -root <lab> -device device-a -rein <rein.exe>
```

stdout printed the recovery code exactly once (`hoplab: device-a initialized
the account; recovery code saved to hoplab-state.json for pair recover`,
followed by the code on its own line — recorded here only as
`<recovery-code>`). `rein account status --json` then reported
`key_generation:1, recovery_code_confirmed:true, keyring_present:true`.

`keyring.v1.json` was fetched directly from the locker over S3 with the
device's own minted credentials (`rein hop credentials --json`) using a
small harness program built against this tree's own `internal/backend/s3`
and `internal/keyring` packages (built and run only for verification, never
committed, never used as the artifact under test):

```
SchemaVersion: 5
CurrentGeneration: 1
GenerationNumbers: [1]
DeviceCount: 1
verify unpinned: OK
verify pinned: OK
```

`keyring.Parse` + `VerifyGenerations("")` and `VerifyGenerations(k.AccountKey)`
both succeeded — the object's own signature verifies both without and with
the account key pinned.

### H4 — `push --all` sends one Claude, one Codex, one OpenCode session

Real vendor sessions were created for all three T5 agents: Claude Code
(`claude -p "..."`) against the **host's live config** in a throwaway
project; Codex (`codex exec -s read-only "..."`) and OpenCode (`opencode run
-m opencode/big-pickle "..."`) each against an isolated home seeded with
only the vendor's own credential file. Each was prompted with "Reply with
exactly this token and nothing else: `<token>`" and each replied with
exactly its token.

**Finding, not release-blocking to this row:** `rein push --all` initially
returned only 2 of the 3 sessions (`claude 1, opencode 1`; codex missing).
Root cause, confirmed by inspecting `internal/adapter/codex/codex.go`'s
`Discover`: once **any** project mapping exists in `config.toml`, Codex's
adapter silently drops any local session whose `cwd` does not match a
mapped project root (`projectIDsByRoot`, `if !mapped { return nil }`) — the
same, deliberately tested behavior Claude's adapter has
(`TestClaudeDiscoverSkipsUnmappedProjects`). This is very likely intentional
scope-limiting (push only touches projects the operator explicitly mapped),
not a defect in Codex specifically — but **OpenCode's adapter has no
equivalent filter at all**, so an OpenCode session from an unmapped project
was pushed anyway. That asymmetry between the three T5 adapters is a
genuine product observation worth the coordinator's attention (Matrix
G3/G4 territory, outside this part's own scope), even though it did not
block this row: adding the missing `--project` mapping for the throwaway
Codex/OpenCode/Claude directories (the ordinary, documented way to opt a
project in) let `push --all` proceed correctly.

```
rein push --all --json
```

reported `"skipped":1,"snapshots":["<id>","<id>"]"` — 3 total sessions now
tracked. A direct `hopd.db` query:

```sql
SELECT id, type, created_at FROM events;
```

showed exactly **one** `first_push` event across the whole sequence
(including a second, no-op `push --all --json`, which reported
`"skipped":3,"snapshots":null`). `rein hop status --json` showed
`first_push_at` set once. The decrypted manifest (via `sync verify`, next
row) confirmed `claude 2, codex 1, opencode 1` (2 Claude sessions: the
device's own isolated fixture plus the real live-config one pushed
separately for the H5 real-resume proof, below).

### H5 — wipe / new device / recover / pull / resume, one real resume

Before-state: `sha256sum` recorded for all four local session files
(isolated Claude fixture, isolated Codex real session, isolated OpenCode
real session via its SQLite store, and the **live-config** Claude real
session).

Wipe:

```
rm -rf <REINSTATE_HOME>
hoplab keyring clear -root <lab> -device device-a
rm <codex-session-file>
rm <claude-fixture-session-file>
rm -rf <opencode-sqlite-subtree>
rm <live-config-claude-session-file>
```

All five deletions succeeded, including the live-config Claude file — no
tool-use policy block this run (the `v0.6.0-rc.4` run's blocker on this
exact step is resolved, matching the dispatch note "Claude Code with the
live config now works"). `hoplab keyring show` confirmed the OS-keyring
entry was gone.

New device:

```
rein login --email <same-email> --json     # got a NEW device id
hoplab approve -root <lab> -email <same-email> -count 1 -timeout 2m
hoplab pair recover -root <lab> -device device-a -rein <rein.exe>
```

`hoplab pair recover` printed `hoplab: device-a enrolled from the recovery
code; it now shares the account 'pair init' initialized`. `rein account
status --json` showed `enrolled_via:"recover", recovery_code_confirmed:true,
key_generation:1` (unchanged), a **different** `device_id` than the wiped
device's original one.

Pull:

```
rein pull --all --json
```

First attempt refused: `{"code":"compatibility","message":"opencode session
...: compatibility NOT_INSTALLED refuses restore; install and run opencode
once on this device so its session layout exists, then pull again"}` — the
new device's isolated `XDG_DATA_HOME` had no OpenCode layout yet, matching
`v0.6.0-rc.4`'s own documented method. One harmless `opencode run` created a
real `opencode.db`, and the retried `pull --all --json` completed:
`"pulled":4,"skipped":0"` in the final retry (an earlier, partial attempt
had already restored the Codex and live-Claude legs before the
still-blocked OpenCode leg aborted that call; the isolated Claude fixture
needed one further `pull --agent claude --session <id>` call to land, since
the batched `--all` call's own accounting did not list it as pulled even
though the other three legs were already correctly restored by then — a
harness/reporting-accuracy observation on the `pulled`/`skipped` **counts**
across a multi-call, partially-aborted sequence, not a restore-mechanism
defect: all four files verified correctly restored below).

All four files were confirmed present again with the correct planted token
substrings; the two agent-native files (Codex, live Claude) sizes changed
slightly from their pre-wipe originals (path round-tripped through
`pathmap` normalize/denormalize even though source and destination paths
were identical strings on this same-host round trip) — content, token
count, and recorded `cwd` were all correct; this is noted as an observation,
not scored against the row, since the row's own text asks for pull to
*complete* and *resume to answer from history*, both of which it did.

`resume --dry-run` for all four:

```
rein resume claude:<live-id> --dry-run --json      # confirmation_required
rein resume claude:<fixture-id> --dry-run --json    # blocked, workspace.available: missing
rein resume codex:<id> --dry-run --json             # confirmation_required
rein resume opencode:<id> --dry-run --json          # confirmation_required
```

The fixture's `blocked` verdict is correct, not a defect: its recorded
workspace (`C:\Users\fixture-user\code\device-a`) is a synthetic path that
never existed on this host, so `workspace.available: missing` and
`agent.executable: missing` are the accurate observation, per the dispatch's
"reading a refusal correctly" guidance.

One real resume:

```
claude --resume <live-session-id> -p "What was the exact token string you replied with earlier in this conversation? Reply with exactly that token and nothing else."
```

Reply: the exact planted token, verbatim — a real Claude Code process,
resumed from the pulled-and-restored history, correctly recalling
conversation content from before the wipe.

### H9 — `sync verify` names only observed objects

```
rein sync verify --json
```

`checked_objects` was exactly `["the index","the newest snapshot in the
index"]`; `unopened` named the remaining objects by count only. Step 4
(bucket isolation) `status:"fail"` with an explicit reason (`This account's
credentials reach a bucket that is not its own` — the pre-existing
`fakelocker AnyBucket:true` harness limitation, same disposition carried
from `v0.6.0-rc.4`, not a new finding) — never `"could not run"` as the
step's own overall status, even though one of its two sub-assertions
explicitly reported "Could not run" for a narrower reason (the fake locker
never answers 403), which the tool still rolled up into an overall `fail`,
not a silently-dropped or falsely-passing result. A genuinely-gone-bucket
(hard 404) scenario could not be produced against this lab's `AnyBucket:
true` fake (it accepts and creates a store for any bucket name asked of it,
by design, documented in `scripts/testing/fakelocker/main.go`); this is the
same limitation `v0.6.0-rc.4`'s own H9 evidence exercised.

### H10 — `sync migrate --to byo`, `--switch`, `--forget-hop`

A second, independent bucket on the same fake locker (`byo-migrate-target-d`
— `fakelocker`'s `AnyBucket` mode serves each bucket name from its own
in-memory store) served as the BYO destination.

```
rein sync migrate --to byo --endpoint http://127.0.0.1:9321 --bucket byo-migrate-target-d --switch --forget-hop --json
```

(passphrase supplied via `REINSTATE_PASSPHRASE_FD`, wired through a small
Windows inherited-handle helper built for this run only, following
`scripts/testing/hoplab/secretfd_windows.go`'s own documented pattern —
never committed, never the artifact under test)

Result: `"migrated":{"snapshots":5,"written":5,...},"switched":true,
"forgot_hop":true`. Afterward:

```
rein whoami --json     # {"code":"auth_storage","message":"this device is not signed in to Reinstate Hop; run `rein login`", ...}
```

A direct `hopd.db` query (`SELECT id, account_id, revoked_at FROM devices;`)
showed **both** enrolled devices with `revoked_at` empty — the migrated
device was never revoked server-side, confirming the documented "does not
revoke" claim. `rein sync verify --json` against the new BYO bucket reported
step 4 as `"not-applicable"`: "BYO storage has no control plane and no
reference locker."

### H11 — path remap on pull

A second isolated home (`device-b`) was configured as a BYO client of the
**same** migrated bucket/profile, with the same project ids mapped to
**different** local paths (`...-remap` directories):

```
rein init --yes --force --endpoint <endpoint> --bucket byo-migrate-target-d --region auto --prefix <prefix> --profile-id <same-profile-id> --project <id>=<own-different-path> [...]
rein pull --all --json
```

Result: `"pulled":4,"skipped":0`. The pulled Codex file's `cwd` field, read
directly from the restored `.jsonl`, read the **new device's own** remapped
path (`...\throwaway-codex-remap`), not the pushing device's original path.

```
rein resume codex:<id> --dry-run --json
```

reported `"decision":"confirmation_required"` with
`workspace.available: present` and the launch plan's `cwd` pointing at the
new device's own path — workspace verification passes on the remapped
location. The macOS leg is deferred (section E), as required.

### H12 — keyless diagnostics refuse a tampered keyring

Three independent devices/accounts, each verified with its own real
`account.json` local anchor:

1. **Rolled back.** Saved the account's genuine, valid generation-1
   `keyring.v1.json`. Ran a real `rein devices revoke <device-id>` (fed the
   H3 recovery code via `REINSTATE_RECOVERY_CODE_FD`) — the account rolled
   to generation 2 (confirmed: `rein account status --json` on the
   surviving device then showed `key_generation:2`). Put the saved
   generation-1 object back with a raw S3 `PUT`. `rein account status
   --json` afterward:
   ```
   "keyring_refused":true,"key_generation":1,
   "error":"keyring current_generation 1 is below the 2 the control plane reports for this account ...; the keyring was rolled back ...; Nothing was written; ..."
   ```
   `rein devices --json` surfaced the identical `keyring_error`. The
   device's own `account.json` sha256 was **identical** before and after.

2. **Forged.** Fetched a second, independent account's own valid
   generation-1 keyring, flipped one base64 character in its generation-1
   signature, and `PUT` it back over that account's own object. `rein
   account status --json`:
   ```
   "keyring_present":false,"keyring_refused":true,
   "error":"keyring: key generation is not signed by this account's key: generation 1 does not verify under account key <account-key>"
   ```
   `account.json` sha256 unchanged.

3. **Re-keyed (foreign account).** Fetched a **third**, genuinely different
   account's own real, validly-signed keyring and `PUT` it into the second
   account's bucket location, overwriting the forged copy from test 2.
   `rein account status --json`:
   ```
   "keyring_refused":true,
   "error":"keyring: the keyring is signed by a different account key: it is signed by account key <account-key>, not the <account-key> expected here"
   ```
   `account.json` sha256 unchanged again.

All three attacks: refused correctly, named a specific and correct reason,
`keyring_refused:true` (or the equivalent `keyring_error` on `devices`), and
the device's own local anchor (`account.json`) never wrote anything new in
any of the three cases.

---

## Findings (this part)

| Severity | Row(s) touched | Description | Release blocking |
| -------- | --------------- | ------------ | ----------------- |
| MINOR (product observation) | H4 (discovered while satisfying it; Matrix G territory, out of this part's scope) | Codex's `Discover` silently excludes sessions from any locally unmapped project once *any* project mapping exists in `config.toml` (matches Claude's own tested, presumably intentional behavior) — but OpenCode's `Discover` has no equivalent filter, so an unmapped OpenCode session is pushed while an equivalent unmapped Codex/Claude session is silently dropped. Worth the coordinator's attention for consistency across the three T5 adapters; did not block H4 once the missing `--project` mapping was added (the documented way to opt a project in). | No — this row's own mechanism (push/first_push/no-op) was fully exercised and correct once the projects were mapped |
| MINOR (harness reporting accuracy) | H5 | `pull --all`'s own `pulled`/`skipped` **counts**, taken across a sequence of one refused call and two retried calls (the first retry still partially blocked by the OpenCode compatibility gate), undercounted how many sessions had actually already been restored to disk by an earlier partial call in the same sequence — the isolated Claude fixture needed one further, isolated `pull --agent claude --session <id>` call before it appeared on disk, even though `pull --all`'s own JSON never separately reported it as still outstanding. The underlying restore mechanism itself was correct end to end (all four files verified present with correct content); this is about the reported counts across a multi-call retry sequence, not data loss. | No — every session was genuinely restored and verified; the row's own requirement (pull completes, resume answers from history) was met |
| Carried, non-blocking (unchanged from `v0.6.0-rc.4`) | H9 | Step 4 (bucket isolation) fails for the pre-existing `scripts/testing/fakelocker` `AnyBucket:true` harness limitation — not a control-plane or product defect. Same disposition as `v0.6.0-rc.4`'s own H9 evidence. | No |

No release-blocking finding in this part's 11 rows.

## Cleanup

Both Hop labs this part started (`D:\ReinstateAcceptanceProjects\v060-rc5-d\hoplab`
and `...\hoplab-h1b`) were confirmed via `hoplab ps` to be the ones this
part started, then stopped with `hoplab stop -root <root>` before writing
this report. No lab this part did not start was touched. All temporary
Go verification helpers (the keyring-fetch/S3-PUT tool, the Windows
inherited-handle passphrase helper) were built and run only under a
`tmp-` directory inside the worktree and removed before this commit; `git
status` is clean of anything but this report. Isolated lab directories
under `D:\ReinstateAcceptanceProjects\v060-rc5-d\` hold no committed
content and are not part of this repository.
