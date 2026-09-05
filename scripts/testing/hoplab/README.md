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
export CLAUDE_CONFIG_DIR="<root>\device-a\home\.claude"
export CODEX_HOME="<root>\device-a\home\.codex"
export XDG_DATA_HOME="<root>\device-a\home\xdgdata"
```

`REINSTATE_HOME` is a first-class override the product itself honours
(`internal/config.Home`) — config, state, and device id all follow it, the
same way `scripts/tuisandbox`'s single-home bench and the in-process CLI
journeys (`hop_first_push_test.go`'s `hopDevice`, which sets it per call)
already isolate device identity.

#### What this does *not* isolate: the OS keyring device token

Two real, simultaneously signed-in `rein` processes on **one Windows
account** collide in the OS keyring: `credentials.KeyringStore` (which W4
does not own — `internal/credentials/**`) uses one fixed service name
(`"reinstate"`) and one fixed entry name (`"hop/device-token"`) regardless
of `REINSTATE_HOME`. There is no `-service`/`-namespace` flag or file-backed
alternative in the product code to route around this per device.

Two ways forward, depending on what the scenario needs:

- **Truly simultaneous devices** (pairing, revocation, the lagging device,
  the cross-plane key-generation floor): use the in-process pattern the CLI
  journeys already run — `internal/cli`'s `hopDevice` in
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
  snapshot), for a clean-slate `rein login`.

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
