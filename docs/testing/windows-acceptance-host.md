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
the OS keyring.

### Pairing the two homes

`hoplab pair init`/`hoplab pair join` drive the real `rein` binary through
`rein init --hop` and `rein account init`/`rein account recover` — the
same sequence `internal/cli/keygeneration_crossplane_test.go`
(`-tags hopacceptance`) proves against a real `hopd` — non-interactively,
so a caller with no real terminal (an agent driving this through a piped
shell, exactly what rejected the previous round of this branch) can still
complete it. Both `account init` and `account recover` read their secret
from `REINSTATE_RECOVERY_CODE_FD` when it is set
(`crypto.ReadSecretFD`, `internal/crypto/passphrase.go` — the product's own
documented automation path, not something this branch invented); `hoplab
pair` supplies it across a real Windows process boundary by adding the
handle to `syscall.SysProcAttr.AdditionalInheritedHandles`, not just
marking it inheritable — Go's `CreateProcess` call restricts inheritance to
that explicit list once any entry is present
(`PROC_THREAD_ATTRIBUTE_HANDLE_LIST`,
`$GOROOT/src/syscall/exec_windows.go`), so marking a handle inheritable
alone (the first version of this file) silently inherited nothing; see
`scripts/testing/hoplab/secretfd_windows.go` and its test.

```bash
./scripts/testing/hoplab/hoplab.sh pair init -root <root> -device device-a -rein bin/rein.exe
# hoplab: device-a initialized the account; recovery code saved to <root>/hoplab-state.json for `pair join`
# GZ63-Z90G-XPS0-Y54Z-7YAT-RXDP-28H1-YPQZ

./scripts/testing/hoplab/hoplab.sh pair join -root <root> -device device-b -rein bin/rein.exe
# hoplab: device-b enrolled from the recovery code; it now shares the account pair init initialized
```

Run `hoplab keyring save`/`load` around each device's `rein login` and
`pair init`/`pair join` (the OS keyring's single-slot limitation, below,
applies here exactly as it does to sign-in). Verified end to end on
2026-09-05 against the real `hopd`/`rein.exe`: `pair init` for `device-a`
then `pair join` for `device-b` (same account email), then `rein account
status --json` as each device (`hoplab keyring load` between them) —

```json
{"profile_id": "2873c8c7-b077-43ff-b7d1-8cd8a0de630f", "device_id": "140875dd-108a-4115-b307-69d3d6b14c25", "enrolled_via": "init", "enrolled_devices": 2, ...}
{"profile_id": "2873c8c7-b077-43ff-b7d1-8cd8a0de630f", "device_id": "e5bcad21-06d3-4e0a-a5d3-0d69b5c6cfd6", "enrolled_via": "recover", "enrolled_devices": 2, ...}
```

— the same `profile_id`, two distinct `device_id`s, both reporting
`enrolled_devices: 2`: the card's "Done when" bar (signed in twice, paired
the two homes) completed from the docs alone, non-interactively, with no
hand-typed recovery code anywhere in the run.

### What the OS keyring device token does not isolate

**The OS keyring device token is a real, host-wide, single-slot resource
`hoplab` does not (and, without touching code W4 does not own, cannot)
isolate per device.** `credentials.KeyringStore`
(`internal/credentials/keyring.go`, `devicetoken.go`) uses one fixed service
name and one fixed entry name regardless of `REINSTATE_HOME`. During an
earlier verification pass, a `rein login --email` for the lab above
**overwrote an already-signed-in device token pointing at a non-loopback
control plane** (a private LAN address, port `8081` — evidently another
workstream's, or an earlier session's, real device, not one of this lab's
own loopback addresses) with no way to recover the value it replaced; it
was cleared afterward (`hoplab keyring clear`) rather than left in an
unknown state. Anyone signing this device in for a lab or test purpose on
this shared host should `hoplab keyring save -root <root> -device <name>`
first if the existing sign-in might be needed back — see
`scripts/testing/hoplab/README.md`'s "What this does *not* isolate" section
for the full explanation and the `keyring save`/`load` workaround for
sequential real-binary use of two devices; truly simultaneous devices need
the in-process pattern `internal/cli`'s `hopDevice` already uses
(`hop_first_push_test.go`, `hop_first_push_acceptance_test.go`,
`keygeneration_crossplane_test.go`).

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
