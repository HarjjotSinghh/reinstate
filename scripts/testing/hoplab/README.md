# hoplab

Runs a disposable Hop lab on this host: a real `hopd` (the private control
plane) plus `scripts/testing/fakelocker` standing in for the bucket, both on
loopback with fake storage and a log-only email sender, and an approver that
clicks the sign-in links hopd's log sender prints — the shape the
2026-08-24 lab used
(`docs/testing/results/2026-08-24-first-push-acceptance-lab.md`).

It never reads, lists, or commits anything from the private control-plane
repository into this one; that repository is referred to only by path,
through `REINSTATE_HOSTED_DIR` / `REINSTATE_HOPD_BIN`.

## Quick start

```powershell
# from the repository root, PowerShell 5.1+
.\scripts\testing\hoplab\hoplab.ps1 start -root D:\ReinstateAcceptanceProjects\hoplab
```

```bash
# Git Bash, from the repository root
./scripts/testing/hoplab/hoplab.sh start -root /d/ReinstateAcceptanceProjects/hoplab
```

`-root` must be outside any Git checkout (same rule as `scripts/tuisandbox`
— hopd.db, logs, and keyring snapshots have no business inside a checkout
`git add -A` could pick up).

This builds (or locates) `hopd`, builds `fakelocker`, starts both, waits for
`hopd`'s `/healthz`, writes `<root>/hoplab-state.json`, and prints:

```
export REINSTATE_HOP_URL="http://127.0.0.1:8082"
```

to stdout (so `eval "$(...)"` / `Invoke-Expression` picks it up on its own)
and a status line to stderr. It then blocks, tailing nothing further, until
Ctrl+C or a `hoplab stop -root <same root>` from another terminal — either
stops both `hopd` and `fakelocker` and removes the state file. Pass
`-background` to have `start` return immediately instead, leaving both
processes running until a later `hoplab stop`.

## Commands

| Command | Purpose |
| ------- | ------- |
| `start` | build/locate hopd and fakelocker, run them, print the env block |
| `stop` | stop a lab (`-root <dir>`, or `-all` for every lab this registry still lists as running) |
| `ps` | list every hopd/fakelocker pair `start` has recorded on this host, and whether each is still running |
| `approve` | approve — or, with `-refuse`, decline — sign-in emails as they appear in the log |
| `homes` | seed isolated device homes under `-root` |
| `env` | print the env block for one seeded device |
| `keyring show/clear` | optional diagnostic: report or clear a device's OS-keyring entry (no pairing flow needs this) |
| `pair init/join/recover` | pair two (or more) seeded devices into one Hop account, driving real `rein` non-interactively |

Run `hoplab -h`, or any subcommand with no required flags, for the full
flag list; the essentials are below.

### `start`

| Flag | Default | Meaning |
| ---- | ------- | ------- |
| `-root` | *(required)* | lab root |
| `-hopd-addr` | `127.0.0.1:8082` | hopd listen address |
| `-locker-addr` | `127.0.0.1:9002` | fakelocker listen address |
| `-hopd-bin` | `$REINSTATE_HOPD_BIN` | a prebuilt hopd binary; skips the build |
| `-hosted-dir` | `$REINSTATE_HOSTED_DIR` or `D:\Projects\reinstate-hosted` | source checkout to build hopd from, when `-hopd-bin` is empty |
| `-background` | off | return immediately; stop later with `hoplab stop` |

hopd runs with `HOPD_STORAGE=fake`, `HOPD_EMAIL_SENDER=log`,
`HOPD_BASE_URL=http://<hopd-addr>`, `HOPD_S3_ENDPOINT=http://<locker-addr>`,
and a fresh `<root>/hopd.db` — the 2026-08-24 lab's own configuration
(`docs/testing/results/2026-08-24-first-push-acceptance-lab.md`). Its
combined stdout+stderr goes to `<root>/hopd.log`.

`start` refuses to proceed if `-hopd-addr` or `-locker-addr` already has a
listener (naming the owning pid where it can identify one), rather than
building and starting a second pair anyway: an earlier, uncleanly-stopped
lab's `hopd` would otherwise still answer `/healthz` for the new lab's
`waitHealthy` check, and everything after `start` would silently talk to
the wrong control plane and locker (see
`docs/testing/windows-acceptance-host.md`'s Hop lab section for the dated
repro). Every successful `start` also records its two pids and addresses
in a small registry file **outside `-root`**
(`os.UserCacheDir()/reinstate-hoplab/labs/`, one file per `-root`), which
`ps` and `stop -all` read:

```bash
./scripts/testing/hoplab/hoplab.sh ps
# D:\ReinstateAcceptanceProjects\hoplab
#   hopd    pid 66824    127.0.0.1:8082  alive=true
#   locker  pid 69152    127.0.0.1:9002  alive=true
#   started 2026-09-06T00:48:58+05:30, status: running

./scripts/testing/hoplab/hoplab.sh stop -all
# hoplab: stopped lab at D:\ReinstateAcceptanceProjects\hoplab (hopd pid 66824, fakelocker pid 69152)
# hoplab: stopped 1 lab(s); removed 0 stale registry entry(ies) whose processes were already gone
```

### `approve`

Watches `<root>/hopd.log` (or `-log <path>` directly) for new
`hopd: email to ... — Sign in to Reinstate Hop` blocks, and for each one
whose address matches `-email` (any address, if omitted): GETs the confirm
page, then POSTs it to approve — the same two requests a person clicking
the emailed link makes (`GET /login/email/{token}` renders the form,
`POST /login/email/{token}` submits it; hopd's confirm page carries no
other field). Stops after `-count` sign-ins are decided or `-timeout`
elapses.

```bash
./scripts/testing/hoplab/hoplab.sh approve -root <root> -email you@example.com -count 2 -timeout 3m
```

Run this **before or alongside** a `rein login --email` (or a
`-tags hopacceptance` test that runs one), in another terminal or process —
it is the "browser" half of email sign-in, watching for the link and
clicking it, not something `rein login` itself waits to be told about.

`-refuse` GETs the confirm page (so the link is genuinely seen, exercising
the same code path a person opening it would) but never POSTs — nothing in
hopd's current sign-in API lets a caller mark a link explicitly declined;
its confirm page has exactly one button. The refused path this exercises is
the ordinary one instead: an unapproved link expires on its own
(`HOPD_LOGIN_SESSION_TTL`, default 10m — set it short, e.g.
`HOPD_LOGIN_SESSION_TTL=10s`, for a fast `-refuse` lab run), and the CLI's
`WaitForApproval` surfaces a `RefusedError` with code `link_expired`.

### `homes` and `env` — two devices on one host

```bash
./scripts/testing/hoplab/hoplab.sh homes -root <root>              # seeds device-a and device-b (the default)
./scripts/testing/hoplab/hoplab.sh env -root <root> -device device-a -shell sh
```

`homes` plants one Claude, one Codex, and one OpenCode session per device
under `<root>/<device>/home/...`, each pointing at a project path — and
carrying a session id — no other device's session set uses, seeded from
`testdata/adapters/{claude,codex,opencode}/windows` and rewritten just
enough to be distinguishable (see `homes.go`'s `rewriteFixture`). `env`
prints the isolated env block for one seeded device:

```
export REINSTATE_HOP_URL="http://127.0.0.1:8082"
export REINSTATE_HOME="<root>\device-a\reinstate"
export HOME="<root>\device-a\home"
export USERPROFILE="<root>\device-a\home"
export CLAUDE_CONFIG_DIR="<root>\device-a\home\.claude"
export CODEX_HOME="<root>\device-a\home\.codex"
export XDG_DATA_HOME="<root>\device-a\home\xdgdata"
export GROK_HOME="<root>\device-a\home\.grok"
export GEMINI_CLI_HOME="<root>\device-a\home\.gemini"
export KIMI_CODE_HOME="<root>\device-a\home\.kimi-code"
export QWEN_HOME="<root>\device-a\home\.qwen"
export CLINE_DATA_DIR="<root>\device-a\home\.cline\data"
export COPILOT_HOME="<root>\device-a\home\.copilot"
export CURSOR_CONFIG_DIR="<root>\device-a\home\.cursor"
export PI_CODING_AGENT_DIR="<root>\device-a\home\.pi\agent"
unset REINSTATE_BACKEND
unset REINSTATE_MEMORY_BACKEND_DIR
unset REINSTATE_S3_ACCESS_KEY_ID
unset REINSTATE_S3_SECRET_ACCESS_KEY
unset REINSTATE_S3_ENDPOINT
unset REINSTATE_S3_BUCKET
unset REINSTATE_S3_REGION
```

The trailing `unset` lines (`Remove-Item Env:<name> -ErrorAction
SilentlyContinue` for `-shell powershell`) matter as much as the exports
above them: `REINSTATE_BACKEND`, `REINSTATE_MEMORY_BACKEND_DIR`, and the
four `REINSTATE_S3_*` names are the product's own BYO-storage escape
hatches (`internal/cli/commands_impl.go`'s `openBackend`,
`internal/credentials.Resolve`), and if any is already set in the
operator's shell — from earlier, unrelated local testing, the ordinary way
to run this project's own local e2e tests by hand — it silently routes a
hop-mode `rein login`/`init --hop`/`account init` around the lab's real
`hopd`/`fakelocker` entirely. See
`docs/testing/windows-acceptance-host.md`'s Hop lab section for the exact,
dated repro this produced on this host (a brand-new account whose keyring
already "had" two devices, because one such variable, left set at the
Windows user level from unrelated earlier work, pointed every hop-mode
account at one shared on-disk object). `hoplab pair` clears the same seven
before launching any `rein` subprocess itself
(`pair.go`'s `reinEnviron`/`stripEnv`), so this is defended twice: once for
a human following this env block, once for `hoplab pair` driving `rein`
directly.

`REINSTATE_HOME` is a first-class override the product itself honours
(`internal/config.Home`) — config, state, and device id all follow it, the
same way `scripts/tuisandbox`'s single-home bench and the in-process CLI
journeys (`hop_first_push_test.go`'s `hopDevice`, which sets it per call)
already isolate device identity.

`HOME`/`USERPROFILE` and the eight `*_HOME`/`*_DIR` variables after them are
just as load-bearing as the four above, not decoration: of the 11 agents in
`internal/agents/catalog`, only Claude/Codex/OpenCode have this package set
their `RootEnv` directly (`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`);
the other eight with an index source (Grok, Gemini, Kimi, Qwen, Cline,
Copilot, Cursor, Pi) fall back to a path under the *real* process home when
their own `RootEnv` is unset
(`internal/agents/scan/hometree.ResolveRoot`: `RootEnv` first, then
`Candidates` built from `agents.Env.HomeDir`, which is `os.UserHomeDir` —
`USERPROFILE` on Windows — when unset). Leaving `HOME`/`USERPROFILE` at the
real host account's value, as an earlier version of this package did,
leaked the real host account's real sessions into every "isolated" device
identically for all eight of those agents; see
`docs/testing/windows-acceptance-host.md`'s Hop lab section for the exact,
dated repro (48 sessions instead of 3) and
`homes_isolation_test.go`'s `TestDeviceHomesDoNotLeakTheHostAccount` for
the regression test. Setting `HOME`/`USERPROFILE` to the device's own
isolated home is the same fix `scripts/tuisandbox`'s `sandboxEnv` already
applies for its single-home bench; the eight explicit `*_HOME`/`*_DIR`
overrides on top are extra insurance against an operator's own shell
already exporting one of them (an explicit env var always wins over a
`HOME`-derived fallback in `hometree.ResolveRoot`, regardless of `HOME`).

### `pair init`/`pair join`/`pair recover` — put the two homes in one account

Every device's own `REINSTATE_HOME` (from `homes`/`env`, above) already
gets its own OS-keyring device-token entry
(`internal/credentials.DeviceTokenEntry`, keyed off `REINSTATE_HOME`), so
`pair` acts directly as whichever `-device` it is given — no `keyring
load`/save swap needed before it, sequential or simultaneous.

**Live pairing (`pair join`), no recovery code — prefer this whenever a
second device is available:**

```bash
./scripts/testing/hoplab/hoplab.sh pair init -root <root> -device device-a -rein bin/rein.exe
# prints the recovery code (also saved to <root>/hoplab-state.json, for `pair recover` if ever needed)

./scripts/testing/hoplab/hoplab.sh pair join -root <root> -device device-b -approver device-a -rein bin/rein.exe
# device-b runs `rein account join` (publishes a pairing request, prints a code, waits);
# device-a runs `rein devices approve`, fed that code -- no code ever typed or copied by hand
```

`pair join` is `internal/cli/pairing_test.go`'s own two-device journey
(`startJoin`/`approveWhilePrompting`) driven against the real compiled
`rein` binary instead of the in-process test harness: `-device` runs `rein
init --hop --project hoplab-<device>=<its project>` then `rein account
join`; once it has published its request and shown a pairing code,
`-approver` runs `rein devices approve` fed that code through
`REINSTATE_PAIRING_CODE_FD`. Both devices must have signed in under **the
same email** first (`rein login` + `hoplab approve`, above) — hopd ties one
account to one email.

**Recovery-code pairing (`pair recover`) — only when no second device can
approve live:**

```bash
./scripts/testing/hoplab/hoplab.sh pair recover -root <root> -device device-c -rein bin/rein.exe
# reads the recovery code back from hoplab-state.json; pass -code to override
```

`pair recover` runs `rein init --hop --project ...` then `rein account
recover` with the first device's recovery code — the exact sequence
`internal/cli/keygeneration_crossplane_test.go` (`-tags hopacceptance`)
proves works against a real `hopd`. `account init` refuses a second device
under a keyring that already exists ("enrol this device with `rein account
recover` instead"), which is exactly what makes `account recover` the right
command when no live approver is available.

Both flows avoid a hidden terminal prompt entirely: `rein account
init`/`account recover` read their secret from
`REINSTATE_RECOVERY_CODE_FD`, and `rein devices approve` from
`REINSTATE_PAIRING_CODE_FD` (`internal/crypto/passphrase.go`'s
`ReadSecretFD` — the product's own documented non-interactive path:
"automation sets `REINSTATE_RECOVERY_CODE_FD`"/`REINSTATE_PAIRING_CODE_FD`),
because `hoplab` drives the real compiled `rein` binary as a separate
process, not the in-process test harness (`hop_first_push_test.go`'s
`hopDevice`) that has a prompt-callback seam to hook — and because a caller
with no real terminal (an agent running this through a piped shell) cannot
answer a hidden prompt at all. `pair init` and `pair join`'s joining device
wire their descriptor to a live pipe whose read end the child process
inherits and feeds the freshly generated code back the moment it appears in
the child's own stderr (the code cannot be known before the command prints
it); `pair recover` and `pair join`'s approving device wire theirs to a
plain temp file carrying the already-known code. See `secretfd_windows.go`
for the Windows handle-passing mechanics — in particular, why marking a
handle inheritable is not sufficient by itself on a modern Go toolchain
(`PROC_THREAD_ATTRIBUTE_HANDLE_LIST` restricts inheritance to an explicit
list once any handle is in it) — and `secretfd_windows_test.go`, which
proves the whole mechanism against the real `crypto.ReadSecretFD` function
across a real process boundary.

`-rein` (or `REINSTATE_REIN_BIN`) names the binary; it defaults to
`bin/rein.exe`/`bin/reinstate.exe` under the repository root (`make
build`'s own output).

### `keyring show`/`clear` — an optional diagnostic, not a pairing step

No pairing flow above needs `keyring` any more: `pair`/`env` already give
each device its own `REINSTATE_HOME`, and
`internal/credentials.DeviceTokenEntry` (commit `2521485f`) already gives
each `REINSTATE_HOME` its own OS-keyring entry, so `device-a` and
`device-b` hold separate device tokens with no swap step. `keyring show`
and `keyring clear` remain only as a read-only-by-default diagnostic for
when something looks wrong:

```bash
./scripts/testing/hoplab/hoplab.sh keyring show -root <root> -device device-a
# hoplab: device-a -> REINSTATE_HOME=<root>\device-a\reinstate -> OS-keyring entry "hop/device-token@<hash>"
# hoplab: device token present: control_plane_url=http://127.0.0.1:8082 account_id=... device_id=...

./scripts/testing/hoplab/hoplab.sh keyring clear -root <root> -device device-a
# removes device-a's device token, for a clean-slate `rein login` as it
```

`show` never prints the token itself, only where it lives and whether it
is there. Both import `internal/credentials` for the service/entry-name
rule (`DeviceTokenEntry`) rather than duplicating it as a literal — W4 does
not own `internal/credentials/**`.

Two real, simultaneously signed-in `rein` **processes** still act
independently only by real OS process, one per device — `hoplab pair`
already drives them that way. A scenario that specifically needs the two
devices' `rein` code running inside *one* process (revocation, the lagging
device, the cross-plane key-generation floor) should use the in-process
pattern the CLI journeys already run instead: `internal/cli`'s `hopDevice`
in `hop_first_push_test.go` gives each device its own
`credentials.MemoryDeviceTokenStore` and its own `REINSTATE_HOME`, switched
per call, all inside one Go test process.
`hop_first_push_acceptance_test.go` (`-tags hopacceptance`) and
`keygeneration_crossplane_test.go` run this exact pattern against a real
`hopd` — two and three devices at once.

## `-tags hopacceptance` and the cross-plane suite

Both tagged suites read the environment `hoplab start` prints, plus their
own sign-in inputs:

```bash
./scripts/testing/hoplab/hoplab.sh start -root <root> -background
eval "$(./scripts/testing/hoplab/hoplab.sh env -root <root> -device device-a)"
./scripts/testing/hoplab/hoplab.sh approve -root <root> -email you@example.com -count 2 -timeout 3m &

HOP_STAGING_URL="$REINSTATE_HOP_URL" HOP_LOGIN_EMAIL=you@example.com HOP_LOGIN_TIMEOUT=2m \
  go test -tags hopacceptance ./internal/cli -run TestHopFirstPushJourneyStaging -count=1 -v

REINSTATE_HOPD_BIN=<root>/hopd.exe \
  go test -tags hopacceptance ./internal/cli -run TestKeyGeneration -count=1 -v

./scripts/testing/hoplab/hoplab.sh stop -root <root>
```

`TestHopFirstPushJourneyStaging` signs in twice (day one, then the wiped
device), hence `-count 2` above. `keygeneration_crossplane_test.go` starts
its own `hopd` per test (`REINSTATE_HOPD_BIN`, no `hoplab start` needed for
that suite) but a prebuilt binary from `hoplab start`'s build step is a
fine source for `REINSTATE_HOPD_BIN` either way.

Verified against a `hoplab`-started control plane on 2026-09-06:
`TestHopFirstPushJourneyStaging` and both `TestKeyGeneration*` subtests
pass (`GOTOOLCHAIN=go1.25.13 CGO_ENABLED=0 go test -tags hopacceptance
./internal/cli -run <name> -count=1 -v`), the former exercising a real
sign-in wait/approval, a real first push, and both a resumable (`codex`)
and a blocked-only-because-not-installed (`claude`/`opencode`) resume
check.

## What `-hopd-bin` / `REINSTATE_HOPD_BIN` and `-hosted-dir` /
`REINSTATE_HOSTED_DIR` mean

The private control plane (`reinstate-hosted`, `cmd/hopd`) is a lab
dependency, not something this repository ships or ever commits. Point
`hoplab` at a prebuilt binary (`REINSTATE_HOPD_BIN`, fastest, and the only
option without access to that checkout) or at the checkout to build fresh
from (`REINSTATE_HOSTED_DIR`, default `D:\Projects\reinstate-hosted`).
Neither name nor path is ever written into this repository by `hoplab`
itself.
