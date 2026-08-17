# Native Windows probe runbook

This produces the one artifact Phase 5 is blocked on. Every T1 promotion in the
roster requires a macOS **and** a native Windows `AGENT-PROBE-V1`, and since
`probePlatformGap` that requirement is enforced in code rather than trusted.

Budget about 40 minutes, most of it waiting on installers.

## Before you start

**It must be native Windows.** WSL is a different device with a different
filesystem, and the conformance check rejects a `-wsl-` artifact in place of a
`-windows-` one. Use PowerShell on the Windows host itself.

**Do not install Antigravity CLI on this machine.** Its installer copies an
existing Gemini CLI setup into `~/.gemini/antigravity-cli/` and perturbs the
tree we would want to measure later.

Nothing in this runbook needs administrator rights.

## 1. Get the probe binary across (2 min)

A prebuilt `rein.exe` is at `dist/rein.exe` in this repository, cross-compiled
from the current branch. Copy it to the Windows machine by USB, cloud drive, or
`scp` and put it somewhere simple such as `C:\probe\rein.exe`.

It must be **this** build. The stable `v0.4.0` binary predates the per-agent
timeout, the account-name redaction, and the marker-gated root discovery: it
will time out on a machine with several agents installed, and if it does emit
anything it will put your Windows account name in the file.

Alternatively, with Go 1.25+ installed on Windows:

```powershell
git clone https://github.com/HarjjotSinghh/reinstate
cd reinstate
git checkout fix/agent-probe-installed-signal
go build -o rein.exe ./cmd/reinstate
```

Confirm it runs:

```powershell
C:\probe\rein.exe version
```

## 2. Install the agents (10 min)

Node 24 is needed first — `winget install OpenJS.NodeJS` if it is absent.

Install the **same versions** that produced the macOS artifacts, so the two
platforms are comparable:

```powershell
npm install -g @moonshot-ai/kimi-code@0.36.1
npm install -g @github/copilot@1.0.80
npm install -g @qwen-code/qwen-code@0.21.12
```

Priority if you only have time for one: **Kimi**. Its macOS evidence is
complete, so a Windows artifact promotes it to T1 on its own. Copilot is second.
Qwen is last and will likely stall at sign-in for the same regional reason it
did on macOS — install it anyway, since the directory shape is still worth
recording.

## 3. Run real sessions (15 min)

A probe of an installed-but-unused agent measures an empty directory. Each
agent needs a real session that reaches a real answer.

**Kimi, in two different project folders.** Two matters: it is the only way to
show whether the global `session_index.jsonl` stays consistent across projects,
which is the open question that decides whether the shipped source can ever use
the index instead of walking the tree.

```powershell
cd C:\Users\<you>\code\project-one
kimi
```

Ask it something that makes it read a file, for example `read README.md and
tell me what this project does`, let it finish, then exit. Repeat in a second,
different folder.

**Copilot:**

```powershell
cd C:\Users\<you>\code\project-one
copilot
```

Same idea — one prompt that touches a file, let it complete, exit.

**Qwen:** run `qwen` and attempt sign-in. If it refuses, exit and carry on; the
attempt still creates the directory tree.

## 4. Capture the artifact (1 min)

From any directory:

```powershell
C:\probe\rein.exe doctor --agents
C:\probe\rein.exe doctor --agents --json > C:\probe\windows-probe.json
```

The first command prints a human-readable table — glance at it and check that
`kimi` shows `yes` in both the `installed` and `root` columns. If `root` is
`no`, the Windows root differs from macOS's `~/.kimi-code`, which is itself a
finding worth reporting rather than a failure.

If any agent shows `timed_out`, re-run with a longer per-agent budget:

```powershell
C:\probe\rein.exe doctor --agents --json --agent-timeout 30s > C:\probe\windows-probe.json
```

## 5. Check redaction before sending (2 min)

The probe is built to emit shapes rather than paths, but this file gets
committed to a public repository, so verify rather than assume:

```powershell
Select-String -Path C:\probe\windows-probe.json -Pattern $env:USERNAME
Select-String -Path C:\probe\windows-probe.json -SimpleMatch 'C:\Users'
```

Both should return nothing. If either matches, **do not send the file** — send
the matching line only, with your username edited out, and the redaction gets
fixed before anything is committed.

## 6. Send it back

One file: `C:\probe\windows-probe.json`. It holds every agent in a single
artifact; splitting it into the per-agent files under
`docs/testing/results/agent-probes/` happens on the other side.

Worth mentioning alongside it:

- Which agents you actually signed into and ran, versus installed only.
- Anything the human-readable table said that looked wrong.
- For Kimi, the two project folder names, so the bucket count can be checked
  against them.

## Optional: settle the Copilot question (10 min)

Copilot's tier hangs on one thing that no amount of probing a healthy install
can answer: whether `session-state/` is authoritative local storage or a cache
the CLI rebuilds from your account. Rich local files look identical either way.

The difference is only visible across a cache clear:

1. Note a session ID under `~\.copilot\session-state\`.
2. Run `copilot` and `/logout`, or rename `~\.copilot` aside.
3. Sign back in.
4. Check whether that session still exists and still opens.

If it survives, Copilot is a local store and can be indexed. If it is rebuilt
from the server, it is a cache and Copilot stays T0 with reason `server_backed`
no matter how substantial the files look. Report which happened; it decides the
tier on its own.
