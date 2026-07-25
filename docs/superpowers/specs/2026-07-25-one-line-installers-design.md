# Reinstate One-Line Installers and Phase 1 Acceptance Design

**Date:** 2026-07-25  
**Status:** Approved  
**Release installed by default:** `v0.1.0-rc.2`

## Goal

Give a new user one memorable command per supported native platform:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

Each command must install the exact approved Reinstate release without
administrator privileges, verify what it downloads, make the CLI discoverable,
and print the explicit next step. Installation remains non-interactive;
configuration begins only when the user runs `rein init`.

The work also delivers a single, strict acceptance guide for validating Phase 1
across a MacBook and a native Windows PC with Claude Code and Codex.

## Constraints

- Keep the Phase 1 CLI behavior and encrypted object format unchanged.
- Never resolve or install a floating `latest` release.
- Never collect secrets, storage credentials, passphrases, or telemetry in the
  website bootstrap.
- Do not launch `rein init` from a piped installer. Its prompts require a real
  interactive terminal.
- Do not require `sudo`, an elevated PowerShell, Homebrew, WinGet, WSL, or a
  package manager.
- Keep `scripts/install.sh` and `scripts/install.ps1` as the canonical release
  installers.
- Serve the public bootstraps as static website files so installation does not
  depend on an Astro server function.
- Preserve exact-version replacement safeguards in the canonical installers.

## Selected Architecture

Add two small bootstrap files:

- `website/public/install.sh`
- `website/public/install.ps1`

The bootstraps are stable public entrypoints, not duplicate installers. Each
bootstrap:

1. hard-codes `v0.1.0-rc.2`;
2. downloads the matching canonical installer from the signed Git tag;
3. validates the canonical installer against a hard-coded SHA-256 digest;
4. executes it with `REINSTATE_VERSION=v0.1.0-rc.2`;
5. lets the canonical installer select the platform artifact;
6. relies on the canonical installer to verify `checksums.txt`, the binary
   digest, and the binary-reported version;
7. configures the user-level executable path idempotently; and
8. prints the command that begins interactive setup.

Updating the default release therefore requires an intentional website change
that updates both the version and the pinned installer digest.

## macOS and POSIX Behavior

The POSIX bootstrap is compatible with `/bin/sh` and uses `set -eu`.

- Default installation directory: `$HOME/.local/bin`.
- The canonical installer creates `reinstate` and the `rein` symlink.
- A downloaded installer is written to a private temporary directory and is
  removed through a trap.
- The bootstrap accepts the existing `INSTALL_DIR` override.
- If the installation directory is already in `PATH`, no shell file changes
  are made.
- Otherwise, the bootstrap idempotently adds one clearly marked export line to
  the current user's shell startup file:
  - zsh: `~/.zshrc`
  - bash: `~/.bashrc`
  - other shells: `~/.profile`
- `REINSTATE_SKIP_PATH_UPDATE=1` disables startup-file changes.
- Because a piped child shell cannot alter its parent's environment, the
  completion message prints the absolute immediate-use command, followed by
  the shorter command available in a new terminal:

  ```text
  Next: /Users/example/.local/bin/rein init
  New terminals: rein init
  ```

The profile update must not overwrite, reorder, or otherwise rewrite an
existing shell configuration file.

## Native Windows Behavior

The Windows bootstrap supports Windows PowerShell 5.1 and modern PowerShell.

- Default installation directory:
  `%LOCALAPPDATA%\Programs\Reinstate\bin`.
- The canonical installer creates `reinstate.exe` and `rein.exe`.
- The downloaded canonical script is hash-verified before execution.
- The bootstrap accepts the existing `INSTALL_DIR` override.
- If the installation directory is absent from the user's PATH, the bootstrap
  appends it idempotently with
  `[Environment]::SetEnvironmentVariable(..., "User")`.
- The current PowerShell process PATH is also updated, so `rein init` works
  immediately after `irm ... | iex` returns.
- `REINSTATE_SKIP_PATH_UPDATE=1` disables user PATH persistence.
- The completion message prints:

  ```text
  Next: rein init
  ```

PATH comparison is case-insensitive and ignores harmless trailing directory
separators to prevent duplicates.

## Failure Behavior

Both bootstraps fail closed with a non-zero exit when:

- the canonical installer cannot be downloaded;
- its SHA-256 does not match the pinned digest;
- a required local verification utility is unavailable;
- the canonical installer fails;
- the platform or architecture is unsupported; or
- the downloaded binary or release checksum fails verification.

PATH configuration happens only after the canonical installer succeeds. An
installer failure must not modify a shell startup file or Windows user PATH.

## Alternatives Considered

### Astro API endpoints

Rejected because the response is static. A server function adds runtime,
caching, and availability failure modes without improving the installation
contract.

### Duplicate complete installers under `website/public`

Rejected because two copies of security-critical download and verification
logic would drift. The website files remain narrow bootstraps around the
canonical version-tagged installers.

### HTTP redirects to raw GitHub files

Rejected because the current canonical installers intentionally require
`REINSTATE_VERSION`, while the public command must be zero-configuration and
exactly pinned.

## Verification Strategy

Tests are written before implementation and cover observable contracts:

- both public files exist at the website root;
- both pin `v0.1.0-rc.2` and never call a `latest` endpoint;
- both download an exact tag path;
- both verify the canonical installer before executing it;
- both pass the exact version into the canonical installer;
- the POSIX bootstrap is valid `/bin/sh`;
- POSIX PATH setup is idempotent and honors its opt-out;
- Windows PATH setup is idempotent, case-insensitive, and honors its opt-out;
- neither bootstrap launches `rein init`;
- website tests and production build pass;
- existing Go, installer, lint, and documentation gates remain green;
- after deployment, both URLs return successful responses containing the pinned
  version and the expected content type;
- an actual clean macOS install from the public URL reports the expected
  version.

Native Windows installation is a required manual acceptance step in the
two-device guide. Automated Windows contract coverage runs in repository CI
where PowerShell and the native environment are available.

## Documentation

Create `docs/testing/phase-1-mac-windows-acceptance.md` as the authoritative
human acceptance runbook. It includes:

- prerequisites and safe test-data preparation;
- public one-line installation on both devices;
- PATH, version, doctor, and setup-check verification;
- first-device and later-device `rein init`;
- Claude Code and Codex agent-prompt validation;
- Mac-to-Windows and Windows-to-Mac round trips;
- status, diff, dry-run, conflict, wrong-passphrase, backup, and recovery
  checks;
- expected results and evidence fields;
- cleanup guidance; and
- strict mandatory pass/fail criteria for declaring Phase 1 functional.

Update the README, canonical getting-started guide, website getting-started
content, and installer comments to use or reference the public commands without
removing the inspect-before-execute alternative.

## Delivery

Work is implemented on `feat/one-line-installers` from `origin/main`.
Completion requires:

1. focused and full local verification;
2. a Conventional Commit;
3. push and pull request;
4. green required checks;
5. merge to `main`;
6. successful Vercel publication; and
7. live smoke tests of both public installer URLs.

No new CLI release tag is required because the bootstraps install the already
published and verified `v0.1.0-rc.2` artifacts.
