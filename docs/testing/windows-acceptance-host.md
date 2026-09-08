# Native Windows acceptance host

Native Windows x64 is a **mandatory** Phase 3 platform. Treat it as a
pre-release development environment, not a post-tag discovery host.

RC1 and RC2 both failed native Windows certification. RC2 still saw Codex
extensionless trust failures and incomplete snapshot/PowerShell release gates
even when GoReleaser was present. This document pins the host so later
candidates fail on product defects only.

## Required host shape

- **OS:** Windows 11 native x64 (never WSL for the Windows column)
- **Shell for the report:** 64-bit Windows PowerShell 5.1 or PowerShell 7+
- **Git:** 2.x with SSH tag verification support
- **Go:** toolchain `go1.25.13` via `$env:GOTOOLCHAIN='go1.25.13'` (GnuWin32 `make` cannot apply Unix `GOTOOLCHAIN=` prefixes)
- **C compiler for race:** MinGW-w64 or MSYS2 `gcc` on `PATH` so
  `CGO_ENABLED=1 go test -race` can link
- **Make / sh:** MSYS2 or Git-for-Windows userland so `make verify` works
- **GoReleaser:** same major line CI uses for snapshots (`goreleaser` on `PATH`)
- **Claude Code + Codex CLI:** installed for real same-vendor rows; versions
  recorded against `docs/compatibility.md`
- **TTY for dest-ack:** Windows Terminal or `ssh -t` so destination launch sees
  an interactive console. Autonomous SSH without a TTY cannot collect A1/A2/A5/A6.
- **Optional only:** Gemini CLI, OpenCode — never install solely for optional
  evidence

## PowerShell-native gates (preferred on Windows)

When GNU tools are missing, use these checked-in scripts. They are first-class
acceptance gates, not fallbacks:

| Gate | PowerShell entrypoint | POSIX twin |
| ---- | --------------------- | ---------- |
| GoReleaser snapshot | `scripts/snapshot.ps1` | `make snapshot` |
| Artifact / SBOM / source inspection | `scripts/check-release-artifacts.ps1` | `scripts/check-release-artifacts.sh` |
| Stage raw GoReleaser binaries | `scripts/stage-release-assets.ps1` | `scripts/stage-release-assets.sh` |
| Host archive identity | `scripts/check-release-binary-identity.ps1` | `scripts/check-release-binary-identity.sh` |
| Installer smoke on staged dist | `scripts/test-install.ps1` | `scripts/test-install.sh` |

Do **not** FAIL a Windows row solely because `sha256sum`, `unzip`, or `jq` are
absent if the matching PowerShell gate passed with exit 0.

Prefer `scripts/snapshot.ps1` on native Windows. It sets the same
`GORELEASER_*_TAG` env vars as `make snapshot`, fails with explicit exit codes
(not make’s masked exit 2), and requires tags (`git fetch --tags`) plus
`goreleaser` on PATH. Install [Syft](https://github.com/anchore/syft) when
archive SBOMs are required by the gate.

`stage-release-assets.ps1` resolves GoReleaser `path` fields as **repository
root-relative** (`dist/...`), matching `stage-release-assets.sh`. Error strings
use `${variable}` form so Windows PowerShell 5.1 does not treat `$name:` as a
drive-scoped variable.

## Pre-flight checklist (before product matrix)

Run in a fresh PowerShell process from the repository root:

```powershell
$env:GOTOOLCHAIN = "go1.25.13"
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
git fetch --tags

go version
gcc --version
goreleaser --version
syft version   # optional early check; required for full SBOM snapshot
make verify

# Race: retain full package output in private evidence if this fails.
$env:CGO_ENABLED = "1"
go test ./... -race -count=1 -timeout=20m *>&1 |
  Tee-Object -FilePath $PrivateEvidence\race-full.txt
if ($LASTEXITCODE -ne 0) {
  # Re-run the failed package once with -count=1 and keep complete stderr.
  # Classify product race vs host/toolchain flake in the report; never discard diagnostics.
}

Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
# Prefer the PowerShell snapshot entrypoint on Windows:
powershell -NoProfile -File .\scripts\snapshot.ps1
powershell -NoProfile -File .\scripts\stage-release-assets.ps1 -DistDir dist
powershell -NoProfile -File .\scripts\check-release-artifacts.ps1 -DistDir dist
powershell -NoProfile -File .\scripts\test-install.ps1 -DistDir dist
```

## Hop lab

`scripts/testing/hoplab` runs a disposable Hop lab on this host: a real
`hopd` (the private control plane, built from `REINSTATE_HOSTED_DIR` or
named by `REINSTATE_HOPD_BIN`) plus `scripts/testing/fakelocker` standing in
for the bucket, both on loopback with fake storage and a log-only email
sender — the shape
`docs/testing/results/2026-08-24-first-push-acceptance-lab.md` used. Full
usage is in `scripts/testing/hoplab/README.md`; the essentials:

```powershell
.\scripts\testing\hoplab\hoplab.ps1 start -root D:\ReinstateAcceptanceProjects\hoplab -background
.\scripts\testing\hoplab\hoplab.ps1 homes -root D:\ReinstateAcceptanceProjects\hoplab
.\scripts\testing\hoplab\hoplab.ps1 approve -root D:\ReinstateAcceptanceProjects\hoplab -email you@example.com -count 1 -timeout 2m
.\scripts\testing\hoplab\hoplab.ps1 stop -root D:\ReinstateAcceptanceProjects\hoplab
```

`start` prints `export REINSTATE_HOP_URL="http://127.0.0.1:8082"` (D1: the
control-plane URL is always configurable, never hardcoded) and writes
`<root>/hoplab-state.json` for `stop`/`approve`/`env` to find later, from
another terminal or process. `approve` tails `<root>/hopd.log` for
`hopd: email to ... — Sign in to Reinstate Hop` blocks and performs the same
GET-then-POST a person clicking the emailed link does; `-refuse` performs
only the GET, exercising the ordinary expired-link refusal path instead
(hopd's confirm page has one button; there is no explicit decline route to
call). `homes` seeds two (or more) isolated device identities under `-root`,
each with its own `REINSTATE_HOME`/`HOME`/`USERPROFILE`/`CLAUDE_CONFIG_DIR`/
`CODEX_HOME`/`XDG_DATA_HOME` plus the `RootEnv` of every other catalog agent
with an index source (`GROK_HOME`, `GEMINI_CLI_HOME`, `KIMI_CODE_HOME`,
`QWEN_HOME`, `CLINE_DATA_DIR`, `COPILOT_HOME`, `CURSOR_CONFIG_DIR`,
`PI_CODING_AGENT_DIR`), and a project path no other device's sessions use.

**This was not always true — a real isolation gap shipped in an earlier
version of this branch, reproduced and rejected by the verifier on
2026-09-05, and fixed the same day.** `hopLabEnv` originally set only
`REINSTATE_HOME`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`, leaving
`HOME`/`USERPROFILE` pointed at the real host account. Of the 11 agents in
`internal/agents/catalog`, only Claude, Codex, and OpenCode have an
explicit `RootEnv` this package set; the other eight with an index source
(Grok, Gemini, Kimi, Qwen, Cline, Copilot, Cursor, Pi) fall back to a path
under the *real* process home when their own `RootEnv` is unset
(`internal/agents/scan/hometree.ResolveRoot`) — so `rein sessions --json`
run as either "isolated" device returned **48 sessions each, 45 of them the
real host account's real sessions and byte-identical between the two
"different" devices**; only the 3 seeded fixtures differed. `hopLabEnv` now
sets `HOME`/`USERPROFILE` to the device's own isolated home (the same fix
`scripts/tuisandbox`'s `sandboxEnv` already applies for its single-home
bench) plus all eight of those `RootEnv` names explicitly, as defense
against an operator's own shell already exporting one of them. Verified
end to end on 2026-09-05, after the fix, with the *real* `rein.exe` binary
(not the in-process test harness): `rein sessions --json` run as
`device-a` and as `device-b` each returned exactly its own **3** synthetic
sessions (`workspace` correctly `...\device-a` vs `...\device-b`, zero
overlap with the other device or the real host account) — see
`scripts/testing/hoplab/homes_isolation_test.go`'s
`TestDeviceHomesDoNotLeakTheHostAccount` for the regression test that pins
this, run against the real `internal/agents/catalog` sources the way `rein
sessions` itself gathers them.

A real `rein login --email` against the lab's `hopd`, approved by `hoplab
approve` running concurrently, signed in and stored a real device token in
the OS keyring — its own entry, isolated per `REINSTATE_HOME` (see the next
section), not the one host-wide slot an earlier round of this branch
assumed.

### The round-2 blocker: ambient environment variables, not the OS keyring

The adversarial verifier rejected the first round of this branch on
`rein account init` failing with "a keyring already exists for this
profile" on what was reported as a first-ever device, and `rein account
status --json` showing `keyring_present:true`, `enrolled_devices:2` for it.
The coordinator's diagnosis (confirmed correct, and already fixed on this
branch as `internal/credentials` commit `2521485f`, "give each Reinstate
home its own device-token entry") was that the client's OS-keyring device
token lived in one fixed entry per host regardless of `REINSTATE_HOME`, so
a "fresh" home could pick up a stale token from an orphaned earlier lab.

Reproducing the exact blocker cold on this host, with that fix already on
the branch tip, still failed the same way — for a different reason. This
host's Windows user account carries two **persistent, User-scoped**
environment variables from earlier, unrelated local development on this
machine:

```text
REINSTATE_BACKEND=memory
REINSTATE_MEMORY_BACKEND_DIR=D:\Projects\hop-10-lab\locker
```

Any new shell on this account inherits them — including a fresh `hoplab`/
`rein` run, and (since these are `[Environment]::GetEnvironmentVariable`
User values, refreshed into every new process) very likely the verifier's
own shell during the first round. `internal/cli/commands_impl.go`'s
`openBackend` checks `REINSTATE_BACKEND == "memory"` **before** it even
looks at `cfg.Storage.Type`, so every command — `login`, `init --hop`,
`account init`, `account status` — silently used a local disk store rooted
at that one fixed, long-lived directory instead of ever talking to the
lab's `hopd`/`fakelocker` over the network (confirmed: with these two
variables set, `hopd.log` shows zero HTTP requests from any `rein`
subprocess for the whole run). Because a hop-mode config's storage prefix
is always empty (`openBackend` scopes hop storage through the bucket the
control plane names, not a shared prefix), *every* hop-mode account on this
host — any email, any lab, any port — read and wrote the exact same flat
`keyring.v1.json` object in that one directory, left over from unrelated
earlier work (`hop-10-lab`) that had already enrolled two devices in it.
Reproduced by loading a brand-new account's `rein account status --json`
twice, with two different emails against two different fresh `hopd`
instances: both reported `keyring_present:true, enrolled_devices:2`,
disappearing the moment `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`
were unset by hand.

The fix is in `hoplab` itself (`scripts/testing/hoplab/env.go`'s
`ambientOverrideEnv`, `stripEnv`, `printEnvClear`), not the product: every
environment `hoplab` builds — the block `hoplab env` prints (now followed
by `unset`/`Remove-Item Env:` lines) and every subprocess `hoplab pair`
launches itself (`reinEnviron`) — clears `REINSTATE_BACKEND`,
`REINSTATE_MEMORY_BACKEND_DIR`, and the four `REINSTATE_S3_*` BYO-credential
overrides unconditionally, regardless of what the operator's shell already
carries. `scripts/testing/hoplab/pair_test.go`'s
`TestReinEnvironClearsAmbientOverridesFromTheOperatorShell` pins the
regression. Verified end to end on this host on 2026-09-06 with both
variables still set in the driving shell (unchanged, not worked around):
the full pairing flow below completed cold with no manual `unset` anywhere.

### Pairing the two homes

`hoplab pair init` / `hoplab pair join` / `hoplab pair recover` drive the
real `rein` binary through account pairing, non-interactively, so a caller
with no real terminal (an agent driving this through a piped shell, exactly
what rejected the previous round of this branch) can still complete it:

- **`pair init`** — the first device: `rein init --hop`, then `rein
  account init`. Saves the recovery code to a sibling file,
  `<root>/hoplab-recovery-code.secret`, for a later `pair recover`: written
  mode 0600 on every OS, and on Windows additionally locked to the current
  user only by replacing the file's DACL outright (mode bits alone do not
  restrict access there --
  `scripts/testing/hoplab/secretacl_windows.go`). It prints only a
  redacted, length-only acknowledgement -- never the code itself, and
  never into `hoplab-state.json` (shared, world-readable, meant to be read
  and copied freely for pids/addresses/log paths). See "Updated
  2026-09-09" below.
- **`pair join`** — the live path, preferred whenever a second device is
  available: the joining device runs `rein init --hop` then `rein account
  join` (which publishes a pairing request, prints a short code, and
  blocks); `-approver` names an already-enrolled device, which runs `rein
  devices approve` fed that code. This is
  `internal/cli/pairing_test.go`'s own two-device journey
  (`startJoin`/`approveWhilePrompting`), driven against the real compiled
  binaries instead of the in-process test harness. No recovery code is
  needed or shown.
- **`pair recover`** — the recovery-code path (what `pair join` was called
  in the round-1 branch), for when no second device is available to
  approve live: `rein init --hop` then `rein account recover` with the
  code `pair init` saved (or `-code` to override).

Both `account init`/`account recover` and `devices approve` read their
secret from an FD, never a hidden terminal prompt:
`REINSTATE_RECOVERY_CODE_FD` and `REINSTATE_PAIRING_CODE_FD`
(`crypto.ReadSecretFD`, `internal/crypto/passphrase.go` — the product's own
documented automation path, not something this branch invented). `hoplab
pair` supplies each across a real Windows process boundary by adding the
handle to `syscall.SysProcAttr.AdditionalInheritedHandles`, not just
marking it inheritable — Go's `CreateProcess` call restricts inheritance to
that explicit list once any entry is present
(`PROC_THREAD_ATTRIBUTE_HANDLE_LIST`,
`$GOROOT/src/syscall/exec_windows.go`), so marking a handle inheritable
alone (the first version of this file) silently inherited nothing; see
`scripts/testing/hoplab/secretfd_windows.go` and its test.

Verified end to end on this host on 2026-09-06, cold, against a fresh
`hopd`/`fakelocker` pair, with the two ambient variables above still set in
the driving shell — `pair init` for `device-a`, then `pair join` for
`device-b` (`-approver device-a`), then `rein account status --json` as
each device:

```bash
$ ./scripts/testing/hoplab/hoplab.sh pair init -root <root> -device device-a -rein bin/rein.exe
hoplab: device-a initialized the account; recovery code saved to <root>\hoplab-state.json for `pair recover`
<recovery-code>

$ ./scripts/testing/hoplab/hoplab.sh pair join -root <root> -device device-b -approver device-a -rein bin/rein.exe
hoplab: device-b joined the account live, approved by device-a
```

**Updated 2026-09-09:** `pair init` no longer prints the raw recovery code
to stdout, and no longer persists it into `hoplab-state.json` (0644, meant
to be read and copied around freely). It now writes the code only to
`<root>/hoplab-recovery-code.secret` (mode 0600) and prints a redacted,
length-only acknowledgement in its place; `pair recover` reads the code
back from that file transparently, so the flow verified below is otherwise
unchanged. A repeat of this transcript today prints, with no
`<recovery-code>` line at all:

```
hoplab: device-a initialized the account; recovery code (39 chars, redacted) saved to <root>\hoplab-recovery-code.secret for `pair recover`
```

The transcript above is kept unedited as the historical record of the
2026-09-06 verification run.

**Updated 2026-09-09 (later the same day):** the previous update above
described `<root>/hoplab-recovery-code.secret` as a "mode-0600" file and
called that owner-only. On Windows it was not: `os.WriteFile`'s mode bits
are a no-op there, so the file actually inherited its parent directory's
ACL, not an owner-only one. `saveRecoveryCode` (`state.go`) now calls
`restrictSecretFileACL` (`secretacl_windows.go`) after every write, which
replaces the file's DACL with a single, non-inherited ACE naming only the
current user, verified by `TestSaveRecoveryCodeAppliesAnOwnerOnlyACL`
(`secretacl_windows_test.go`) reading the DACL back with
`GetNamedSecurityInfo`/`GetAce` rather than trusting the mode bits. Every
other OS keeps the original 0600-only behavior, which was already correct
there.

```json
{"profile_id": "b9442b52-a15e-4668-81b8-78111f84ea6c", "device_id": "452f6282-8b5a-4936-8cca-8da75b3f8aa5", "enrolled_via": "init", "recovery_code_confirmed": true, "enrolled_devices": 2, "device_in_keyring": true, ...}
{"profile_id": "b9442b52-a15e-4668-81b8-78111f84ea6c", "device_id": "cccea436-4504-48e4-94bb-5e2b40076f83", "enrolled_via": "join", "recovery_code_confirmed": false, "enrolled_devices": 2, "device_in_keyring": true, ...}
```

— the same `profile_id`, two distinct `device_id`s, both reporting
`enrolled_devices: 2` and `device_in_keyring: true`, `device-a` via `init`
and `device-b` via the live `join` with no recovery code anywhere in that
device's run: the card's "Done when" bar (signed in twice, paired the two
homes) completed from the docs alone, non-interactively, cold.

`pair recover` was verified the same day on a third seeded device
(`device-c`, same account): `pair init` for `device-a`, `pair recover` for
`device-c` with the saved code, `rein account status --json` on `device-c`
reported `"enrolled_via": "recover"`, `"recovery_code_confirmed": true`,
`"enrolled_devices": 2`.

One harmless, self-recovering wrinkle worth expecting: `hoplab approve`
re-reads the whole `hopd.log` on every poll and starts a fresh "already
decided" set each invocation, so approving a second device under the same
email as a first can print one
`hoplab approve: device "…": FAILED: … unexpected status 410 Gone` line for
the first device's already-consumed link before it finds and approves the
second device's genuinely new one. It does not stop the run.

### What the OS keyring device token isolates now

`internal/credentials.DeviceTokenEntry()` (commit `2521485f`) derives the
OS-keyring entry from `REINSTATE_HOME`: the default home keeps the legacy
`hop/device-token` entry, and any other `REINSTATE_HOME` gets its own
entry name. `hoplab`'s per-device environment (`hopLabEnv`, `reinEnviron`)
already sets a distinct `REINSTATE_HOME` per device, so `device-a` and
`device-b` hold separate device tokens with no swap required — the
`hoplab keyring save`/`load`/`clear` workaround an earlier round of this
branch needed (and this document used to describe under "What this does
not isolate") no longer applies and has been removed.

`hoplab keyring show`/`clear` remain as an **optional, read-only-by-default
diagnostic** — never a step any pairing flow needs — for when something
looks wrong: `show` reports the OS-keyring entry a device's `REINSTATE_HOME`
maps to and whether a token is present there (never the token itself);
`clear` removes it, for a clean-slate `rein login`. Both import
`internal/credentials` for the entry-name rule rather than duplicating it.
Truly simultaneous real-binary devices (revocation, the lagging device, the
cross-plane key-generation floor) still need the in-process pattern
`internal/cli`'s `hopDevice` already uses (`hop_first_push_test.go`,
`hop_first_push_acceptance_test.go`, `keygeneration_crossplane_test.go`),
which gives each device its own `credentials.MemoryDeviceTokenStore` inside
one Go test process — `hoplab pair`'s real-subprocess devices act
sequentially or concurrently by real OS process, not by an in-process seam.

### Orphan processes: registry, `ps`, and `stop -all`

An earlier `hoplab start` that was not stopped cleanly (a killed terminal,
a crashed shell) used to leave `hopd`/`fakelocker` listening with nothing
to point at it: a later "fresh" lab's `waitHealthy` check only waits for
`/healthz` to answer, which an orphan answers just as well as a genuine new
process, so the caller silently talked to the wrong control plane and
locker. `hoplab start` now refuses a port that already has a listener,
naming the owning pid:

```text
hoplab start: hopd address 127.0.0.1:8499 is already in use by pid 66824; `hoplab ps` lists hoplab-started processes, `hoplab stop -all` stops every one still running, or stop it yourself (Windows: taskkill /PID 66824 /F)
```

Every `hoplab start` also records its two pids and addresses in a registry
file **outside any lab root** (`os.UserCacheDir()/reinstate-hoplab/labs/`,
one file per `-root`, named by a hash of its path — never under `-root`
itself, so it survives that directory being deleted), so it can be found
and stopped from any terminal without remembering every `-root` ever used:

```bash
$ ./scripts/testing/hoplab/hoplab.sh ps
D:\ReinstateAcceptanceProjects\cold-run
  hopd    pid 66824    127.0.0.1:8499  alive=true
  locker  pid 69152    127.0.0.1:9499  alive=true
  started 2026-09-06T00:48:58+05:30, status: running

$ ./scripts/testing/hoplab/hoplab.sh stop -all
hoplab: stopped lab at D:\ReinstateAcceptanceProjects\cold-run (hopd pid 66824, fakelocker pid 69152)
hoplab: stopped 1 lab(s); removed 0 stale registry entry(ies) whose processes were already gone
```

Liveness (`hoplab ps`, and what `stop -all` kills) is checked with
`OpenProcess`/`GetExitCodeProcess` directly
(`scripts/testing/hoplab/registry_windows.go`), not by shelling out to
`tasklist`: a bare `tasklist` on this development host failed outright
(`ERROR: Critical error`, a WMI/performance-counter dependency that can be
unavailable in a restricted or sandboxed Windows environment), while
`netstat` — used only for the best-effort pid name in the port-refusal
message above, never for the liveness check itself — worked throughout.

## ConPTY driver

`scripts/testing/conptydriver` is the Windows twin of
`scripts/testing/vendor-tty-driver.py`: runs a command under a real Windows
pseudo console (`golang.org/x/sys/windows`'s `CreatePseudoConsole`; no new
module dependency), drives it with a small step script, and renders what
appeared through a real VT screen model, not a regex strip. Full usage,
the step-script grammar, and two traps worth knowing (conhost rewriting
unchanged runs as cursor-forward moves instead of literal spaces; Bubble
Tea's OSC 11 / CSI 6n startup queries, which it answers) are in
`scripts/testing/conptydriver/README.md`.

```powershell
.\scripts\testing\conptydriver\conptydriver.ps1 -cols 80 -rows 24 -script steps.txt -- .\bin\rein.exe
```

## Product regressions Windows must cover

- Extensionless vendor lookup for `codex` / `claude` resolving `*.exe` and
  `*.cmd` via PATHEXT outside the workspace trust boundary, including quoted
  PATH entries and EvalSymlinks fallbacks (`internal/executabletrust`)
- Owner-only DACLs on derived index/lock files
- Real Claude and Codex same-vendor resume/fork with installed artifacts
- Real OpenCode `rein push` / `rein pull` into a vendor-initialised
  `opencode.db`, then `opencode --pure export <id>` showing compact message
  bodies and the workspace path remapped onto the device (the T5 procedure in
  `phase-1-mac-windows-acceptance.md` §17b)
- Path remapping and non-TTY warning policy under PowerShell redirection

## Human-owned Windows Terminal rows

Autonomous agents must not invent ConPTY input. These remain human QA with
evidence pasted into the device report.

This section predates `scripts/testing/conptydriver` (added 2026-09-05,
T-403), which since then has driven a real ConPTY session autonomously and
end to end (see Results, below) -- so the sentence above is a standing
policy for these three specific rows, not a claim that autonomous input
injection is technically impossible in general. Rows the CLI matrix (W7)
chooses to run through `conptydriver` instead are that matrix's call, made
against its own contract, not a rewrite of this list:

1. Interactive `rein` picker in Windows Terminal (real TTY)
2. Warning acknowledgment / refusal behavior when stdin is a real console
3. Repository-swap refusal while a confirmation prompt is open

If human QA is unavailable, record those rows as **FAIL** (missing required
evidence), never as PASS or NOT TESTED for required rows.

## Results

### ConPTY probe (#367), 2026-09-05

**PASS.** #367 reported the acceptance host's pseudo-console subsystem
broke on 2026-08-23 (Q6, `docs/planning/v0.6.0-hop/clarifications.md`);
before building `conptydriver`, W4 probed it directly with
`CreatePseudoConsole` → `NewProcThreadAttributeList` → `CreateProcess` under
`EXTENDED_STARTUPINFO_PRESENT`, the same sequence Microsoft's own sample
uses. On this host, today: `CreatePseudoConsole` allocates a handle,
`CreateProcess` starts `cmd.exe /c "echo conpty-probe-ok & exit"` attached
to it, the child exits 0, and the pseudo console's output pipe carries the
child's real, correctly VT-wrapped output — a console-init sequence, the
echoed text, and a title-bar OSC. Pseudo-console allocation on this host is
not broken; whatever produced #367 on 2026-08-23 is not reproducing on
2026-09-05 (a reboot, mentioned as untried in Q6, may have been all it
needed, or the cause was otherwise transient). No document, issue, or
report should still call this host's ConPTY broken without a fresh
contradicting probe.

### A trap in how `conptydriver` itself must be launched

Driving `scripts/tuisandbox`'s bare `rein` to its switcher initially failed
every attempt with `interactive session picker requires a terminal`, and a
minimal `golang.org/x/term` `IsTerminal` probe run the same way reported
**not a terminal** for a ConPTY-attached child — even though the earlier
probe above proves pseudo-console allocation itself works. The cause was
not `conptydriver`: it was that `conptydriver.exe` was being started by a
tool that redirects its own stdout/stderr to a pipe (to capture output for
the caller). A process launched that way has no console of its own
(`GetConsoleMode` on its `GetStdHandle`-derived handles returns
`ERROR_INVALID_HANDLE`, confirmed independent of ConPTY entirely — a plain
`cmd.exe` shows the same thing under such a launch), and
`CreatePseudoConsole`'s automatic "attach the child to a new console"
behaviour for `conptydriver`'s own children inherits that: the pseudo
console it allocates is real (openable and correctly VT-moded through
`CONOUT$`/`CONIN$` from inside the child) but `GetStdHandle` in the
grandchild does not resolve to it.

**Fix: launch `conptydriver.exe` itself from something that does not
redirect its stdio** — a real interactive PowerShell/Windows Terminal
session (the normal case `scripts/testing/conptydriver/conptydriver.ps1`
documents), or, from an automated context, PowerShell's `Start-Process`
*without* `-RedirectStandardOutput`/`-RedirectStandardError` (a hidden
window is fine: `-WindowStyle Hidden` does not redirect stdio, it only
skips showing the window). Once `conptydriver.exe` has a real console of
its own, everything downstream resolves correctly — confirmed by rerunning
the same `GetConsoleMode` probe as a `conptydriver` grandchild launched this
way: `mode=7` (`ENABLE_VIRTUAL_TERMINAL_PROCESSING` set), `err=<nil>`, both
stdin and stdout.

### Full proof, 2026-09-05: the switcher, and Claude Code `--resume` to completion

With that launch fix, both of T-403's proof requirements pass in full.

**The switcher.** `conptydriver`, launched via `Start-Process` (no stdio
redirection) against `scripts/tuisandbox`'s bare `rein`
(`D:\ReinstateAcceptanceProjects\tuisandbox-w4`), with the step script

```
wait /ctrl\+k commands/ 15s
snapshot switcher-snapshot.txt
kill
```

produced exactly the switcher frame:

```
 rein                                           1 session · 1 agent · reinstate
 ❯ type to filter
YESTERDAY                                      │ opencode · reinstate
▸ ◌ opencode reinstate   Hosted bill… 1d ago   │ Hosted billing landing check
                                               │ 1d ago
                                               │
                                               │ ◌ CHECKING
...
 ↵ resume   tab actions   ctrl+a scope   ctrl+k commands   esc quit
```

**Claude Code `--resume`, to completion, in a throwaway project.** Per the
task's condition (a Claude Code login exists for the host user; never read
or list the real `~/.claude`; run the vendor binary only in a throwaway
project under `D:\ReinstateAcceptanceProjects\`): using
`D:\ReinstateAcceptanceProjects\claude-conpty-w4-proof`, `conptydriver`
drove `cmd.exe /c claude` through the first-run trust prompt (`key down`,
`key enter`), a real message ("Reply with exactly the single word:
banana..."), a clean exit (`key ctrl+c` twice), then `cmd.exe /c claude
--resume`, selecting the just-created session from the real resume picker,
confirming its prior turn was restored (the "banana" prompt reappeared in
context, not a fresh session), sending a second message ("...single word:
kumquat..."), and capturing the completed response:

```
❯ Reply with exactly the single word: banana. Nothing else.

❯ Now reply with exactly the single word: kumquat. Nothing else.

● kumquat

✻ Crunched for 5s · done 10:27 PM
```

This is a real session against the host's own Claude Max account (a live
`claude.ai/code/session_...` URL appeared during the run). **Cleanup:**
`claude rm <id>` only deletes a `--bg` background session ("Works on
sessions that have already exited"); there is no CLI command to delete a
regular interactive session's transcript, so — per the task's explicit
instruction for exactly this case — the session was left in place rather
than invented a way to remove it. It lives only under
`D:\ReinstateAcceptanceProjects\claude-conpty-w4-proof` (a throwaway
project, never committed) and in the host account's own Claude Code
history; nothing about it was written into this repository.

**Two operational notes for whoever runs this next:** the throwaway
project's `~/.claude`-recorded "trust this folder" decision persists
between runs (expected — it is the real, host-level trust store), so a
second run against the same directory goes straight to the chat prompt and
a script that still `wait`s for the trust screen will fail immediately;
write scripts defensively (`wait` on the chat prompt's own chrome, not on a
screen that may already be behind you). And this session's own
`CLAUDE_CODE_CHILD_SESSION`/`CLAUDE_CODE_MESSAGING_*`/`CLAUDE_CODE_SESSION_ID` /
`CLAUDE_CODE_BRIDGE_SESSION_ID` environment variables (inherited from
whatever agent session is doing the driving) make a nested `claude` think
it is a child session with transcript saving off; clear them before
launching if the point is to prove a normal, resumable session, as here.

## What CI does and does not prove

GitHub `windows-latest` runs `CGO_ENABLED=0 go test ./...` and packaging jobs.
It does **not** replace physical Windows acceptance: no real Claude/Codex
sessions, no Windows Terminal, and historically no Windows race job. A green
PR is necessary and insufficient for Windows certification.
