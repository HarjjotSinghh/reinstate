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
| `stop` | stop a lab (background, or from another terminal) |
| `approve` | approve — or, with `-refuse`, decline — sign-in emails as they appear in the log |
| `homes` | seed isolated device homes under `-root` |
| `env` | print the env block for one seeded device |
| `keyring save/load/clear` | move which device's token is active in the OS keyring |
| `pair init/join` | pair two (or more) seeded devices into one Hop account, driving real `rein` non-interactively |

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
```

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

### `pair init`/`pair join` — put the two homes in one account

```bash
./scripts/testing/hoplab/hoplab.sh keyring load -root <root> -device device-a
./scripts/testing/hoplab/hoplab.sh pair init -root <root> -device device-a -rein bin/rein.exe
# prints the recovery code, and also saves it to <root>/hoplab-state.json

./scripts/testing/hoplab/hoplab.sh keyring load -root <root> -device device-b
./scripts/testing/hoplab/hoplab.sh pair join -root <root> -device device-b -rein bin/rein.exe
# reads the recovery code back from hoplab-state.json; pass -code to override
```

`pair init` runs `rein init --hop --project hoplab-device-a=<device's project>`
then `rein account init` for the first device; `pair join` runs `rein init
--hop --project ...` then `rein account recover` with the first device's
recovery code, for every device after it — the exact sequence
`internal/cli/keygeneration_crossplane_test.go` (`-tags hopacceptance`)
proves works against a real `hopd`. Both devices must have signed in under
**the same email** first (`rein login` + `hoplab approve`, above) — hopd
ties one account to one email, and `account init` refuses a second device
under a keyring that already exists ("enrol this device with `rein account
recover` instead"), which is exactly what makes `account recover` the right
command for every device after the first.

Both `rein account init`'s confirmation step and `rein account recover`
read their secret from `REINSTATE_RECOVERY_CODE_FD`
(`internal/crypto/passphrase.go`'s `ReadSecretFD` — the product's own
documented non-interactive path: "automation sets
`REINSTATE_RECOVERY_CODE_FD`"), never a hidden terminal prompt, because
`hoplab` drives the real compiled `rein` binary as a separate process, not
the in-process test harness (`hop_first_push_test.go`'s `hopDevice`) that
has a prompt-callback seam to hook — and because a caller with no real
terminal (an agent running this through a piped shell) cannot answer a
hidden prompt at all. `pair init` wires that descriptor to a live pipe
whose read end the child process inherits and feeds the freshly generated
code back into the moment it appears in the child's own stderr (the code
cannot be known before `account init` prints it); `pair join` wires it to
a plain temp file carrying the already-known code. See
`secretfd_windows.go` for the Windows handle-passing mechanics — in
particular, why marking a handle inheritable is not sufficient by itself
on a modern Go toolchain (`PROC_THREAD_ATTRIBUTE_HANDLE_LIST` restricts
inheritance to an explicit list once any handle is in it) — and
`secretfd_windows_test.go`, which proves the whole mechanism against the
real `crypto.ReadSecretFD` function across a real process boundary.

`-rein` (or `REINSTATE_REIN_BIN`) names the binary; it defaults to
`bin/rein.exe`/`bin/reinstate.exe` under the repository root (`make
build`'s own output). Like `rein login`, `pair init`/`pair join` act as
whichever device's token is currently active in the OS keyring — `hoplab
keyring load -device <name>` first, every time, same as any other
sequential real-binary use of two devices (below).

#### What this does *not* isolate: the OS keyring device token

Two real, simultaneously signed-in `rein` processes on **one Windows
account** collide in the OS keyring: `credentials.KeyringStore` (which W4
does not own — `internal/credentials/**`) uses one fixed service name
(`"reinstate"`) and one fixed entry name (`"hop/device-token"`) regardless
of `REINSTATE_HOME`. There is no `-service`/`-namespace` flag or file-backed
alternative in the product code to route around this per device.

Two ways forward, depending on what the scenario needs:

- **Truly simultaneous devices** (revocation, the lagging device, the
  cross-plane key-generation floor — and pairing itself, if the scenario
  specifically needs two real `rein` processes signed in at once rather
  than sequentially): use the in-process pattern the CLI journeys already
  run — `internal/cli`'s `hopDevice` in
  `hop_first_push_test.go` gives each device its own
  `credentials.MemoryDeviceTokenStore` and its own `REINSTATE_HOME`,
  switched per call, all inside one Go test process.
  `hop_first_push_acceptance_test.go` (`-tags hopacceptance`) and
  `keygeneration_crossplane_test.go` run this exact pattern against a real
  `hopd` — two and three devices at once — which is what makes them the
  right vehicle for scenarios needing real simultaneity, not the compiled
  `rein` binary run twice.

- **Sequential real-binary use** (drive the actual `rein.exe`, e.g. under
  `conptydriver`, as one device, then the other): `hoplab keyring save`
  and `load` swap which device's token is the active one, using the same
  `github.com/zalando/go-keyring` the product already depends on:

  ```bash
  # device A signs in for real
  rein login --email you@example.com   # + hoplab approve, elsewhere
  ./scripts/testing/hoplab/hoplab.sh keyring save -root <root> -device device-a

  # device B signs in for real (overwrites the OS keyring's one slot)
  rein login --email you@example.com
  ./scripts/testing/hoplab/hoplab.sh keyring save -root <root> -device device-b

  # act as device A again
  ./scripts/testing/hoplab/hoplab.sh keyring load -root <root> -device device-a
  rein whoami   # answers as device A

  # act as device B again
  ./scripts/testing/hoplab/hoplab.sh keyring load -root <root> -device device-b
  ```

  `keyring clear` removes whatever token is currently active (not a saved
  snapshot), for a clean-slate `rein login`. `pair init`/`pair join`
  (above) build directly on this save/load pattern to drive the actual
  account-pairing commands (`rein account init`/`rein account recover`)
  non-interactively, once each device has signed in this way.

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

## What `-hopd-bin` / `REINSTATE_HOPD_BIN` and `-hosted-dir` /
`REINSTATE_HOSTED_DIR` mean

The private control plane (`reinstate-hosted`, `cmd/hopd`) is a lab
dependency, not something this repository ships or ever commits. Point
`hoplab` at a prebuilt binary (`REINSTATE_HOPD_BIN`, fastest, and the only
option without access to that checkout) or at the checkout to build fresh
from (`REINSTATE_HOSTED_DIR`, default `D:\Projects\reinstate-hosted`).
Neither name nor path is ever written into this repository by `hoplab`
itself.
