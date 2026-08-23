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

Autonomous agents must not invent ConPTY input. These remain **human QA** with
evidence pasted into the device report:

1. Interactive `rein` picker in Windows Terminal (real TTY)
2. Warning acknowledgment / refusal behavior when stdin is a real console
3. Repository-swap refusal while a confirmation prompt is open

If human QA is unavailable, record those rows as **FAIL** (missing required
evidence), never as PASS or NOT TESTED for required rows.

## What CI does and does not prove

GitHub `windows-latest` runs `CGO_ENABLED=0 go test ./...` and packaging jobs.
It does **not** replace physical Windows acceptance: no real Claude/Codex
sessions, no Windows Terminal, and historically no Windows race job. A green
PR is necessary and insufficient for Windows certification.
