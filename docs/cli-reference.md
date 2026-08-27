# CLI reference

Binary names: `rein` and `reinstate` (identical behavior).

Public installers pin candidate `v0.5.1`, including the Phase 1/2 surface,
Phase 3 verified resume, the Phase 4 structured-handoff surface, and the
Phase 5 catalog/T1 index surface. Dual-platform tagged-artifact acceptance is
pending. Stable remains `v0.4.0`. Intel macOS and Linux/WSL2 remain optional
and unverified.

Stable `v0.3.0` includes the Phase 3 environment report and
`--allow-environment-warning` flag. The command synopsis below additionally
includes the `v0.4.0` structured-handoff surface.

## Exit codes

| Code | Meaning |
| ---- | ------- |
| 0 | success |
| 1 | unexpected runtime failure |
| 2 | usage / invalid arguments |
| 3 | missing/invalid config |
| 4 | authentication or storage failure |
| 5 | agent/layout compatibility failure |
| 6 | ambiguous session reference or sync conflict |
| 7 | safety refusal |

## Commands

```text
rein
rein version [--json]
rein doctor [--json] [--self-test] [--agents] [--acceptance-matrix]
rein setup check [--json]
rein login [--email ADDRESS] [--no-browser] [--json]
rein whoami [--json]
rein sessions [--agent claude|codex|gemini|opencode|grok|kimi|qwen|pi|cursor|copilot|cline|all] [--json]
rein search QUERY [QUERY...] [--agent ...] [--project FRAGMENT]
            [--branch FRAGMENT] [--file FRAGMENT] [--limit N] [--json]
rein inspect AGENT:SESSION_ID [--json]
rein last [--agent claude|codex|grok|all] [--project FRAGMENT] [--dry-run] [--json]
          [--allow-environment-warning CHECK_ID ...]
rein resume AGENT:SESSION_ID [--dry-run] [--json] [--fork]
            [--with claude|codex|grok|opencode|qwen]
            [--allow-environment-warning CHECK_ID ...]
rein fork AGENT:SESSION_ID [--dry-run] [--json]
          [--allow-environment-warning CHECK_ID ...]
rein handoff [AGENT:]SESSION_ID --to claude|codex|grok|opencode|qwen
             [--policy checkpoint|balanced|full] [--dry-run|--no-launch]
             [--json] [--export PATH] [--allow-warning ID ...]
             [--allow-active] [--allow-untested] [--show-redactions]
             [--no-redact]
rein handoff --last [--from claude|codex|gemini|grok|kimi|opencode|qwen]
             --to claude|codex|grok|opencode|qwen [handoff flags]
rein handoff list [--json] [--limit N]
rein handoff inspect HANDOFF_ID [--json]
             [--acknowledged|--not-acknowledged]
rein handoff export HANDOFF_ID --format json|markdown [--out PATH]
rein init [--endpoint URL] [--bucket NAME] [--region auto] [--prefix ...]
          [--profile-id UUID] [--project ID=/absolute/local/path] [--yes]
          [--link] [--paste]
rein account init
rein account join
rein account recover
rein account status [--json]
rein devices [--json]
rein devices approve [--request ID]
rein daemon run [--pull-every DUR] [--debounce DUR] [--poll] [--verbose]
rein daemon install|start|stop|uninstall
rein daemon status [--json]
rein list [--agent claude|codex|all] [--json]
rein status [--json]
rein sync verify [--json] [--post=false]
rein diff [--json]
rein push [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein pull [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein sync migrate --to=byo [--endpoint URL] [--bucket NAME] [--region auto]
                  [--prefix ...] [--switch|--keep-hop-config] [--forget-hop]
                  [--json]
rein conflicts list|show|resolve ...
rein completion bash|zsh|fish|powershell
```

Every command accepts the global `--plain` flag, which forces the frozen
non-interactive output on a terminal that could otherwise show the interactive
UI.

### `rein doctor`

`--json` emits machine-readable diagnostics. `--self-test` runs a synthetic
encryption/storage check (in-memory; it does not prove remote storage).
`--agents` lists every catalog agent and its support tier. `--agents --json`
emits the redacted `AGENT-PROBE-V1` artifact. `--agents --acceptance-matrix`
prints the generated Phase 5 acceptance row count and per-agent row list.

### `rein login` and `rein whoami`

`rein login` signs this device in to Reinstate Hop without a password: GitHub
in the browser by default, or a one-time link with `--email ADDRESS`. The
device token lands in the OS keyring. `rein whoami` prints the account, the
device, and the control plane. `REINSTATE_HOP_URL` or `[hop] url` selects a
non-production control plane. See [docs/hop.md](hop.md).

### `rein hop status`

Shows the signed-in account's locker: endpoint, bucket, location, plan,
measured usage against the plan's storage limit, enrolled devices against
the device limit, the push-rate limit, and when the first push happened.
Before the first push it says the locker is not provisioned yet. `--json`
emits the same. Exit `4` when the device is not signed in or its token was
rejected.

### `rein hop credentials`

Mints one credential set for this account's locker and prints it, so the
first, second and fourth checks of `rein sync verify` can be repeated by
hand with any S3 client ([object format](hop/object-format.md),
"Reproducing the checks by hand"). It prints the bucket, the locker's key
prefix, the endpoint, region and expiry, then `AWS_ACCESS_KEY_ID`,
`AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` and `AWS_ENDPOINT_URL` on
stdout, with a caution on stderr so the values can be redirected without
it. `--json` emits the credential fields as data. `--export` prints shell
`export` statements and nothing else — the four names above plus
`AWS_REGION`, `AWS_DEFAULT_REGION`, `REIN_LOCKER_BUCKET` and
`REIN_LOCKER_PREFIX` — for `eval "$(rein hop credentials --export)"`,
which is how the documented recipe starts: a shell assignment that is
never exported does not reach the `aws` process, which is why the flag
exists. `REIN_LOCKER_PREFIX` is empty on a locker with no prefix and ends
in `/` on one that has a prefix, so the recipe can paste it in front of a
key either way. Reading the prefix costs one extra request to the control
plane, made before the mint so a failed lookup does not spend one. `--json`
and `--export` cannot be combined.

These are the credentials `rein push` already uses: valid for at most an
hour, scoped by the storage provider to this account's bucket and no other
(which is what step 4 of the verification tests). Every session object
they can read is ciphertext; `keyring.v1.json` is not, and is not meant to
be — it is plaintext by design and holds no usable key, but it names the
account's profile id, every enrolled device's id, public key and enrolment
time, and one entry per key generation with the time it started (so a
locker whose key has rolled over also shows which devices stopped being
enrolled, and when), so a credential printed here hands that over too. Each run
mints a fresh set and counts against the push-rate limit `rein hop status`
shows. The third check needs the account's root key, which never leaves
the device and which no command exports. Exit `4` when the device is not
signed in, its token was rejected, or the mint was refused by a quota.

### `rein daemon`

`rein daemon` runs a resident per-device process that pushes after a
session changes, pulls on a schedule, and surfaces devices waiting to join
the account. `rein daemon install` registers it to start at login (launchd
on macOS, systemd `--user` on Linux, Task Scheduler on Windows) and starts
it; `start`, `stop`, and `uninstall` control the registered daemon.
`rein daemon run` is the foreground loop that registration runs (`--pull-every`
defaults to 30s, `--debounce` to 3s). While the daemon runs, `rein resume`,
`rein fork`, and the switcher pull once more before launching when its last
pull is older than 15s; a failed pull is reported and the local copy is
resumed. `rein daemon status [--json]` reports whether the daemon is registered and running, the
last push and pull, the watched roots, and — on Hop — enrolled devices and
pending approvals. It needs the root-key model (`rein account init`, which
works on BYO storage too). See [docs/hop.md](hop.md#the-daemon). Exit `4`
when a hosted daemon's device is not signed in; `3` when the home is not
configured for the root-key model.

### `rein sync verify`

Runs the checks behind the zero-knowledge claim against the configured
storage and prints a verification report a non-expert can read and repeat
step by step: list the store with this device's credentials; fetch an
object and show it is ciphertext (age v1 header, recipient type, no
plaintext field names in the body); decrypt it locally and show what it
contains (the index, a snapshot's envelope and verified payload checksum);
and, on a Hop locker, show that the same credentials are refused (access
denied, not a rejection of the credential itself) from the control
plane's reference locker — a different bucket, at the same storage
endpoint the listing used, over a client that refuses to follow a redirect
elsewhere and sends the credential over plaintext `http` to nothing but a
loopback address (`localhost`, `127.0.0.0/8`, `::1`), where the request
does not leave the machine; every other plaintext endpoint is refused
without a request being made. No Hop locker is at a loopback address; a
locally run control plane in development is. The
outcome sentence names only what was fetched as ciphertext and lists what
was judged by name. Each step prints what
was done, what was seen, and PASS, FAIL, or NOT APPLICABLE, followed by
one of four outcome lines: `OUTCOME: PASS`, `OUTCOME: FAIL`,
`OUTCOME: NOT VERIFIED` (nothing answered, so nothing was checked), or
`OUTCOME: NOT YET VERIFIABLE` (nothing has been pushed yet).

A step that got no answer — from the storage endpoint or from the control
plane — is reported as a check that could not run, not as a check that
failed. That holds on all four steps, not only the fourth. Steps 1 and 2
are NOT APPLICABLE, with a reason beginning "Could not run", when the
storage endpoint answered nothing at all: a request that timed out, a
connection that dropped, a name that did not resolve, a 500. A refusal is
an answer, and a refusal still fails the step. The fourth step is NOT
APPLICABLE whenever the control plane could not be reached or answered an
error, its reference row names this account's own bucket, the reference
bucket has been deleted or would not answer, or the locker credential was
rejected or rotated mid-run; a step 4 that could not run leaves the exit
code unchanged, and the outcome sentence says isolation was not checked. A
step 4 that *failed* exits `7` like any other failed step. A run where the
storage endpoint answered nothing checks nothing: `outcome` is
`not-applicable`, the report ends `OUTCOME: NOT VERIFIED`, and the exit
code is `1`. The
[threat model](hop/threat-model.md) lists every case, and what does fail
the step.

One check that cannot run is still reported as a failure, and it is named
here rather than left to be found: step 3 with no key available on this
device fails. `rein sync verify` resolves a key before it runs, so the CLI
does not reach that state; a caller embedding the `verify` package without
one does.

`--json` emits the report (`report.steps[].{id,name,did,observed,status,
detail}`, `report.outcome`, `report.storage`, `report.locker`,
`report.summary`, `report.checked_objects`, `report.unopened`) plus
`posted`. `report.summary` is the outcome sentence itself, so a script
renders exactly what a person reads instead of inferring it from
`outcome`. On a Hop profile the step results (never object contents,
session ids, or project paths) are posted to the control plane for the
account console; `--post=false` keeps them local. BYO storage runs the
first three steps and reports the fourth as not applicable. Exit `0` when
every step passed or did not apply — including a profile that has pushed
nothing yet, where `outcome` is `not-applicable` and the report ends
`OUTCOME: NOT YET VERIFIABLE` — `7` (safety) when a step failed, `1` when
the control plane or the storage endpoint could not be reached (the report
still prints, saying which checks did not run), and the usual storage or
sign-in codes when the store cannot be opened.

The same checks run automatically, once per device, after the first push
that uploaded something to a Hop locker; `rein push --json` then carries
`verification: {outcome, posted}`. See [hop.md](hop.md) and the
[threat model](hop/threat-model.md).

### `rein init`

`rein init --hop` configures this home for the hosted tier instead of your
own bucket: it needs a prior `rein login`, provisions the account's locker,
and writes `storage.type = "hop"` with the account as the profile and the
enrolled device as the device. It takes `--project` and `--force` but not
`--endpoint`, `--bucket`, `--prefix`, `--profile-id`, or `--paste`. Follow it
with `rein account init` (first device) or `rein account join` (later
devices, approved from an enrolled one with `rein devices approve`; `rein
account recover` is the recovery-code fallback); see [docs/hop.md](hop.md).

On a terminal, and when the coordinates were not already supplied by flags or
environment, `rein init` opens a setup wizard: a provider preset, then the
endpoint, bucket, region, prefix, and whether this device is joining an existing
profile. Each field is validated as it is entered and every step can be revisited,
so a mistake in one does not discard the others.

The wizard collects no secret material. Access keys, secret keys, and the
passphrase are read afterwards, through the same hidden-input path they have
always used, once the full-screen program has released the terminal.

`--yes` and explicitly supplied coordinates skip the wizard entirely.

#### Device pairing

```text
rein init --link      print this profile's pairing code
rein init --paste     start setup from a pairing code
```

A pairing code carries the endpoint, bucket, region, prefix, and profile ID as
one string, so a second device does not need those copied by hand. It carries
**no keys and no passphrase**; the receiving device still asks for both. Treat it
as a convenience, not a credential.

## Local commands

`sessions`, `search`, and `inspect` refresh and read a private derived index
without requiring `init`, config, storage credentials, a passphrase, keyring
access, or a backend. Stable `v0.2.0` used the Phase 2 v1 index. `v0.3.0`
deliberately moves to a separate path so an older binary cannot erase
new baseline metadata:

```text
$REINSTATE_HOME/cache/session-index-v2.sqlite
```

The database and its `.lock` and `.write.lock` files are owner-only and never
synced. The shared/exclusive `.lock` protects database lifetime and rebuild;
the `.write.lock` serializes ordinary writers across processes. Session rows
are safe to rebuild; successful prelaunch observations are private continuity
metadata and therefore live in the versioned v2 store rather than the Phase 2
v1 file. The database contains bounded
user-authored prompt text plus known metadata/file fields. It excludes
assistant messages/reasoning, tool output, environment dumps, credentials, and
auth stores.

Canonical references use:

```text
<agent>:<native-session-id>
```

A bare native ID is accepted only when it resolves to one indexed session.
Ambiguous IDs fail with a request to use the composite reference. Result
ordering is deterministic: newest update first, then agent, then native ID.

Search is literal and case-insensitive. Multiple query terms are ANDed.
`sessions` and `search` identify metadata without printing transcript passages.
`inspect` may show a terminal-safe, whitespace-collapsed first-user-prompt
preview capped at 160 Unicode code points; Phase 2 has no full-transcript dump.

`resume`, `fork`, and `last` build a structured executable/argv/cwd plan and
delegate execution to the source vendor:

| Agent | Resume | Fork |
| ----- | ------ | ---- |
| Claude Code | `claude --resume ID` | `claude --resume ID --fork-session` |
| Codex | `codex resume ID` | `codex fork ID` |
| Gemini CLI | read-only | read-only |
| OpenCode | `opencode --session ID` | `opencode --session ID --fork` |

Review the plan with `--dry-run --json`. Stable `v0.3.0` also includes
the verified-resume `environment` report described below. A real launch
inherits the terminal, waits for the child, and propagates failure. JSON mode
requires `--dry-run` for `resume`, `fork`, and `last`, so native child output
cannot corrupt the JSON document. Read-only agents refuse resume/fork with
compatibility exit `5` before any environment probe or vendor launch.

### The interactive switcher

On a terminal that can host it, bare `rein` opens a full-screen session
switcher. Typing filters the list; the arrow keys move; the selected session is
previewed beside it with a readiness verdict computed from the same preflight
report `rein resume` enforces.

```text
type          filter, using the same matching as `rein search`
up/down       move; ctrl+p and ctrl+n also work
pgup/pgdn     page; home and end jump
enter         resume the selected session
tab           action menu: r resume, f fork, h hand off, i inspect, y copy ref
ctrl+a        switch between this project and every project
ctrl+k        command palette
ctrl+r        rescan every agent now
esc           clear the filter, or quit when it is already empty
```

Letters always filter. Actions live behind `tab` and `ctrl+k` so that no
keystroke means "filter" in one moment and "fork" in the next.

The status column shows how resumable each session is, computed in the
background for the rows on screen:

```text
●  ready to resume
◐  resumable once environment warnings are acknowledged
○  cannot resume: blocked, or a read-only agent
◌  still being checked
```

`h` opens the handoff studio, which measures the projection for each
destination and policy before anything is written, and states the exact
`rein handoff` command for the current selection.

Environment warnings are acknowledged with the spacebar rather than by retyping
identifiers. The screen shows the equivalent
`--allow-environment-warning` command line as it is built, so the scriptable
form is always visible.

### Plain output and the degradation ladder

Plain output is what Reinstate emitted before the switcher existed, byte for
byte. It is selected, in order, by:

```text
1. --json                                   always plain
2. neither stream is a terminal             always plain
3. --plain, REINSTATE_NO_TUI=1, TERM=dumb,
   CI (or GITHUB_ACTIONS, GITLAB_CI,
   BUILDKITE, CIRCLECI, TF_BUILD)           plain
4. a terminal under 40x10                   plain
5. NO_COLOR set                             interactive, no colour
6. width under 80                           interactive, single column
7. otherwise                                interactive, split panes
```

On a terminal that reaches step 3 or 4, bare `rein` falls back to the numbered
switcher, whose contract is unchanged:

```text
/text       filter
i NUMBER    inspect
f NUMBER    fork
h NUMBER    structured handoff; then choose claude or codex
NUMBER      resume
q           cancel
```

On a non-TTY, bare `rein` exits promptly with usage code `2` and a
`rein sessions --json` hint.

`REINSTATE_TUI_COLS` and `REINSTATE_TUI_ROWS` pin the frame size, so acceptance
runs produce the same frame on any console.

`rein list` remains the Phase 1 compatibility command used by sync scripts.
`rein sessions` is the canonical config-independent local listing command.

## Phase 4 structured handoff (`v0.4.0`)

A structured handoff continues the same task in a new Claude Code or Codex
session. It is not native resume: Reinstate does not reconstruct vendor history,
write a vendor-internal session file, or claim that the destination is the same
session. Source parsing and projection are local and require no source model
call. Gemini CLI and Kimi Code CLI are source-only; Claude Code, Codex, Grok
Build, OpenCode, and Qwen Code are destinations. A handoff into Grok always runs
redaction and always prints the repository-upload warning; `--no-redact` is
refused with exit `2` in that direction as well as when Grok is the source.

### `rein handoff`

`rein handoff [SESSION] --to AGENT` accepts:

| Flag | Contract |
| ---- | -------- |
| `--last` | Select the newest matching source instead of `SESSION`. |
| `--from AGENT` | Restrict `--last` to one source agent. |
| `--to AGENT` | Required destination: `claude`, `codex`, or `grok`. |
| `--policy checkpoint\|balanced\|full` | Projection policy; default `balanced`. |
| `--dry-run` | Preview using temporary files only; no durable handoff and no launch. |
| `--json` | Emit machine-readable, launch-free output; requires `--dry-run` or `--no-launch`. |
| `--no-launch` | Store the capsule and print the exact command without spawning the destination. |
| `--export PATH` | Also write the projection to `PATH`; incompatible with `--dry-run`. |
| `--allow-warning ID` | Acknowledge one exact current warning ID; repeat for each warning. |
| `--allow-active` | Freeze the last complete source record while its agent is active. |
| `--allow-untested` | Proceed with an untested source or destination layout. |
| `--show-redactions` | Show redaction categories and counts, never values. |
| `--no-redact` | Skip secret redaction. Refused with exit `2` for Grok sources. |

### `rein handoff list`

Accepts `--json` and `--limit N` (default `100`).

### `rein handoff inspect`

`rein handoff inspect HANDOFF_ID` accepts `--json`, `--acknowledged`, and
`--not-acknowledged`; the two acknowledgement flags are mutually exclusive.

### `rein handoff export`

`rein handoff export HANDOFF_ID` requires `--format json|markdown` and accepts
`--out PATH`; without `--out`, it writes to stdout.

### `rein resume --with` and `--fork`

`rein resume SESSION --with AGENT` is a convenience alias for
`rein handoff SESSION --to AGENT`. It accepts `--dry-run`, `--json`, and
repeatable `--allow-environment-warning ID`, translated to exact handoff
warning acknowledgements. With this alias, `--json` requires `--dry-run`;
`resume --with` has no `--no-launch` mode. The alias prints a one-line
structured-handoff notice. `rein resume SESSION --fork` instead invokes the
source agent's native fork path; `--with` and `--fork` are mutually exclusive.

### Handoff exit codes

Handoff exit codes use the shared table above: `2` for bad flags, unknown
agents, and invalid launch/JSON combinations; `3` for invalid local config; `5`
for an untested or unsupported source/destination layout; `6` for an ambiguous
session reference; `7` for unacknowledged warnings or safety refusal; and `1`
for runtime failure. A planned or completed handoff returns `0`. No Phase 4
handoff path uses authentication/storage code `4`.

## Phase 3 verified resume (`v0.3.0`)

Phase 3 is included in stable `v0.3.0`; tagged-artifact acceptance passed on
Apple Silicon macOS and native Windows x64. Stable `v0.2.0` does not include
it. Before any real Claude or Codex native continuation, Reinstate builds a
deterministic, local-only environment report.
The same report is exposed by `inspect` and native dry-runs and enforced by
`resume`, `fork`, `last`, and picker resume/fork.

The report covers:

- selected-source freshness and the recorded workspace;
- an offline repository fingerprint, branch, HEAD, and privacy-safe working
  tree state/digest;
- installed same-vendor executable, verified version, and recognized layout;
- bounded logical names/states for recognized instruction files, skills, and
  MCP declarations; and
- supported Node and Go runtime declarations and locally installed versions.

It does not fetch, install, repair, checkout, reset, run project scripts, or
contact a network service. It omits dirty filenames, raw remote URLs,
instruction/skill contents, MCP commands/arguments/URLs/headers/environment
values, credentials, and raw environment dumps.

`rein inspect SESSION` always emits the report. Human output prints an
`Environment decision` and deterministic check lines. JSON adds:

```json
{
  "environment": {
    "schema_version": 1,
    "session_ref": "claude:SESSION_ID",
    "decision": "confirmation_required",
    "checks": []
  }
}
```

Inspect does not prompt or launch. A successfully generated blocked report
still exits `0`; automation must read `environment.decision`. Failure to
produce an honest report exits `1`.

Native dry-runs preserve the launch-plan keys and add `environment`. They
never prompt or launch. Ready and warning-only reports exit `0`. A blocked
dry-run emits one report-bearing error and returns its applicable exit code.
A dry-run does not need warning acknowledgements because it cannot launch; if
warning flags are supplied, their IDs are validated, but a partial valid set is
not treated as launch authorization.

### Launch decisions and warning acknowledgement

| Decision | Real launch behavior |
| -------- | -------------------- |
| `ready` | Launches after a final identical refresh and preflight. |
| `confirmation_required` | On a TTY, prompts `yes`/`no` with default `no`; otherwise every warning must be acknowledged by exact ID. |
| `blocked` | Refuses without prompting; no flag can override it. |

For non-interactive use, repeat the flag for every warning in the fresh report:

```sh
rein resume claude:SESSION_ID \
  --allow-environment-warning baseline.unavailable \
  --allow-environment-warning git.branch
```

The flag is invocation-scoped and accepts only an exact current warning ID.
Empty, duplicate, wildcard, unknown, stale, and informational IDs are usage
errors (`2`). Supplying only some current warnings is a safety refusal (`7`).
A blocked report takes its applicable blocker exit before acknowledgements are
considered; naming a blocker can never authorize it. There is no `--force`,
wildcard, persisted approval, environment-variable bypass, or
`--continue-without` alias.

A real terminal prompt accepts exactly `yes` or `no`; empty input and `no`
decline. EOF also declines. Decline/refusal returns safety exit `7` without a
vendor launch.

Resuming a session that the owning agent already has open warns with
`agent.active`. Reinstate does not refuse it: the vendor CLI owns every write to
its own store, and a second window on one session is a legitimate thing to want.
The warning exists because it is more often an accident. A host that cannot
enumerate its own processes reports that it could not tell and still resumes,
rather than claiming the session is free on evidence it never gathered. A
structured handoff does not raise this warning; it enforces its own
`--allow-active` boundary against the same signal.

The first preflight for an existing session warns with
`baseline.unavailable`; inspection never turns the current workspace into
historical truth. After an authorized native child exits successfully,
Reinstate stores the immediately preceding observation with
`reinstate_prelaunch_observed` provenance. Failed, declined, cancelled, blocked,
or child-error launches do not establish or advance it. A subsequent preflight
can compare repository identity, branch, HEAD, working-tree digest,
capabilities, and recognized runtimes with that private baseline.

Known repository replacement and stale selected-source metadata are safety
blockers (`7`). A missing workspace/executable or unverified agent layout/version
is a compatibility blocker (`5`). Probe infrastructure failure is runtime
failure (`1`). When multiple blockers exist, deterministic precedence is
runtime `1`, safety `7`, then compatibility `5`; the report still includes all
checks.

## Phase 1 encrypted sync

Interactive encryption uses a hidden terminal prompt. Non-interactive
automation must open a secret file/pipe and set `REINSTATE_PASSPHRASE_FD` to
that descriptor number; `REINSTATE_PASSPHRASE` and secret CLI flags are not
accepted.

Interactive `init` stores storage credentials in the native OS keyring.
The explicit non-interactive fallback reads
`REINSTATE_S3_ACCESS_KEY_ID` / `REINSTATE_S3_SECRET_ACCESS_KEY` without
persisting them.
Override home with `REINSTATE_HOME` (absolute path).

`rein setup check` returns compatibility exit code `5` when an installed agent
is `UNTESTED` or `UNSUPPORTED`; its summary never says all checks passed while
that agent is blocked from push/pull. `rein conflicts list` and
`rein conflicts show` require a valid config, so a missing config cannot look
like an empty conflict set.

`push --all` and `pull --all` keep going past a session that diverged on
this device: the divergence is recorded once under `rein conflicts`
(records are keyed by the divergence, so repeated runs never add a second
copy), every other session is still pushed or restored, the conflicted
sessions are reported together on stderr, and the exit code is `6`
(`--json` lists them under `conflicts`). An explicit `--session` still
stops at its own conflict.

A mutating `pull` never waits on a human closing an agent. Liveness is scoped to
the exact session file being replaced, so unrelated agents running in other
projects are ignored, and if that one session really is in use the live file is
left untouched and the remote copy is restored beside it as a distinct session.

`conflicts resolve --keep-remote` still refuses while the target session is in
use, because `--keep-both` is the explicit way to preserve both branches there.

New-session restores, `--keep-both`, and `pull --dry-run` remain available.
`--allow-active-agents` skips the liveness check for one run.

See [Configuration](configuration.md) for `restore.active_agent_policy`
(`fork` by default, or `scoped`, `strict`, `off`).

### `rein account`

Hosted key model. Every subcommand needs an initialized home (`rein init`
with the storage coordinates first).

- `rein account init` — generate the root key on this first device, write the
  keyring to storage, and show the recovery code exactly once. The code must
  be re-entered before anything is written; a mismatch aborts with nothing
  written. Prints the recovery policy plainly: losing every device and the
  recovery code makes the locker unrecoverable by anyone; local copies survive.
  Refuses when this device is already enrolled or a keyring already exists.
- `rein account join` — enrol this machine by approval from one that is
  already enrolled. Needs a prior `rein login` whose device matches the
  home's `device_id` (`rein init --hop` guarantees that). Generates this
  device's key, publishes a pairing request, shows a 16-character code once
  on stderr, and waits (Ctrl-C withdraws the request). When an enrolled
  device approves, the root key arrives sealed under the code and to this
  device's key; it is accepted only if the keyring's own wrap for this device
  opens to the same key of the same generation. An expired request exits `4`
  with nothing written; a device already enrolled exits `7`.
- `rein devices` — list the account's devices (from the control plane),
  whether each holds a root-key wrap (from the keyring), and pending pairing
  requests. `rein devices approve` reads the code shown on the joining device
  (hidden prompt, or `REINSTATE_PAIRING_CODE_FD`), checks it against the
  request, appends the new device's wrap with compare-and-swap, and relays
  the sealed root key. A wrong code exits `7` and approves nothing; a typo is
  rejected by the checksum (exit `2`); with several requests pending,
  `--request ID` picks one (exit `2` otherwise). Only an enrolled device can
  approve.
- `rein account recover` — enrol a fresh machine from the recovery code
  (hidden prompt, or `REINSTATE_RECOVERY_CODE_FD` pointing at a pre-opened
  descriptor for automation, like `REINSTATE_PASSPHRASE_FD`). Unwraps the root
  key locally, generates this device's key, and appends a wrap for it with
  compare-and-swap. A wrong code fails closed (exit `4`); a code with a typo
  is rejected by its checksum before any key derivation (exit `2`). An
  existing device key in the OS keyring is never overwritten or deleted: when
  the keyring already lists this device and the stored key matches, `recover`
  only restores the local enrolment record; when the two disagree (key gone,
  or a key the keyring does not list) it refuses with exit `7` and nothing is
  written.
- `rein account status [--json]` — encryption mode, whether this device is
  enrolled and how, whether the recovery code was confirmed here (a local
  boolean only), whether the device key is present in the OS keyring, and the
  keyring's key generation and enrolled device count.

After enrolment `push`, `pull`, `status`, and `diff` use the root key and no
longer prompt for a passphrase. The root key and recovery code are never
accepted as flags or plain environment variables.

### `rein sync migrate --to=byo`

Leave Reinstate Hop with the history intact. Works on a home whose
`storage.type` is `hop` and `encryption.type` is `root-key`; anything else
exits `3` with nothing to migrate.

1. Takes the destination bucket from flags, `REINSTATE_S3_ENDPOINT` /
   `REINSTATE_S3_BUCKET` / `REINSTATE_S3_REGION`, or prompts; its access keys
   from `REINSTATE_S3_ACCESS_KEY_ID` / `REINSTATE_S3_SECRET_ACCESS_KEY` or
   hidden prompts; and a **new BYO passphrase** from `REINSTATE_PASSPHRASE_FD`
   or a hidden prompt entered twice. A fresh profile id is minted and the
   destination prefix defaults to `profiles/<profile id>`.
2. Lists every snapshot in the locker (heads and earlier revisions alike,
   following every listing page) and refuses to write anything unless the
   listing covers every snapshot the locker manifest points at. It also
   refuses, before writing, a destination prefix that already holds a
   keyring object or sessions the locker lacks (exit `6`). Then it opens each with the root key on this device, re-seals it under the
   passphrase, writes it create-only to the destination under the same
   snapshot id, and re-reads it to compare the plaintext digest. The manifest
   is written last with the usual compare-and-swap and re-read the same way.
   Progress (`[n/total] <snapshot> written (bytes so far)`) goes to stderr.
   The root key, the keyring, and the device key are never written to the
   destination; every destination envelope is sealed to the passphrase only.
3. The locker is only read (list, get, head): the command works while the
   account is read-only after a lapse, and it never empties or deletes the
   locker (that is account deletion).
4. Interrupted? Rerun the same command. `migrate-byo.json` in the home
   records the destination, the profile id, and the digests of verified
   snapshots (never a passphrase or credential); verified snapshots are
   skipped (the first one is re-read as a check that the passphrase is the
   same), an object that exists but was not recorded is verified in place,
   and nothing is written twice. Flags that name a different destination
   while a migration is in progress exit `2`. A finished migration keeps the
   record so a rerun reuses the profile instead of making a second copy.
   A destination object that re-reads differently from the source stops the
   run with exit `6` and the object's key; it is left for you to inspect,
   and a rerun hits the same object until it is removed or moved.
5. Offers to switch this device to the destination (`--switch` or
   `--keep-hop-config` decide without asking; `--json` requires one of them).
   Switching backs up `config.toml` and `state.json` under `backups/`, writes
   a BYO profile (`storage.type = "s3"`, `encryption.type = "age-scrypt"`,
   `remote_profile_required = true`, projects and agents carried over),
   stores typed access keys in the OS keyring (keys that came from
   `REINSTATE_S3_*` are not stored, so keep them exported or run `rein init`
   to store them; the command says which applies), and keeps local state
   because snapshot ids were preserved. Then offers to forget this device's
   Hop sign-in (`--forget-hop`): the device token leaves the OS keyring; the
   locker and account are untouched.

   To revert the switch, copy `backups/<timestamp>-migrate-byo/config.toml`
   and `state.json` back over the ones in the home; if `--forget-hop` was
   used, `rein login` again. The destination copy stays where it is.

Other devices join the destination with `rein init --profile-id <printed id>`
and the passphrase. Exit codes: `2` usage, `3` not a Hop profile, `4`
storage or control-plane refusal (the message says how far it got and that a
rerun resumes), `6` a destination in use (keyring object or foreign
sessions), a verification mismatch, or a resume under a different
passphrase.

## Planned universal configuration commands

The following is roadmap direction, **not current CLI syntax**:

```text
rein mcp add|list|remove …
rein skill install|list|remove …
rein loop install|list|remove …
rein plugin install|list|remove …
rein marketplace add|list|remove …
rein config import|list|diff|apply|status|sync …
rein auth status …
```

The design goal is one non-secret desired-state profile rendered by verified
adapters into each target harness. Exact names and flags require an RFC before
implementation. See
[universal-configuration.md](universal-configuration.md).
call. Gemini CLI, OpenCode, Grok Build, and Kimi Code CLI are source-only;
