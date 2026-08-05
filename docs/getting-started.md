# Getting Started

Reinstate is the continuity layer for coding-agent work. Stable `v0.2.0`
finds, searches, inspects, resumes, and forks local sessions without
configuration or cloud access. It also synchronizes Claude Code and Codex CLI
sessions across machines through
client-side-encrypted, user-owned object storage.

The current development source additionally implements Phase 3 verified
resume: a privacy-safe environment report and launch gate for same-vendor
Claude/Codex continuation. It has not yet been published as `v0.3.0-rc.1` or
certified as stable.

> **Release status:** the public installers pin stable `v0.2.0`. Exact signed
> artifacts passed the complete physical matrix on Apple Silicon macOS and
> native Windows x64. Intel macOS and Linux/WSL2 packages are available as
> preview, unverified builds pending [#97](https://github.com/HarjjotSinghh/reinstate/issues/97)
> and [#98](https://github.com/HarjjotSinghh/reinstate/issues/98). Evidence is in
> [docs/testing/results](testing/results/).

## Prerequisites

For local Phase 2 commands:

- macOS, native 64-bit Windows, Linux, or WSL2;
- Go 1.25.12 or newer when building the current source; and
- Claude Code and/or Codex CLI. Gemini CLI and OpenCode are read-only Phase 2
  sources when installed.

For optional encrypted multi-device sync, also provide:

- macOS, native 64-bit Windows, Linux, or WSL2
- an S3-compatible bucket you control
- the endpoint, bucket name, access-key ID, and secret access key
- one long encryption passphrase that you can enter privately on every device

Cloudflare R2 is the recommended Phase 1 backend. Local indexing does not need
a backend, and Reinstate never needs your Anthropic, OpenAI, Google, or OpenCode
account credentials.

## Install

### Build the current source

From this checkout:

```sh
make build
./bin/rein version --json
```

Both `./bin/rein` and `./bin/reinstate` are the same binary. Use an isolated
absolute home when evaluating a source build:

```sh
export REINSTATE_HOME="$HOME/.reinstate-phase3-local"
```

Native Windows PowerShell:

```powershell
New-Item -ItemType Directory -Force .\bin | Out-Null
go build -o .\bin\rein.exe .\cmd\reinstate
$env:REINSTATE_HOME = Join-Path $HOME ".reinstate-phase3-local"
.\bin\rein.exe version --json
```

Do not use the stable public installer as evidence for an untagged commit. The
installer proves only the signed release it pins.

### Install v0.2.0 with Homebrew on Apple Silicon macOS

```sh
brew install HarjjotSinghh/tap/reinstate
```

The tap's stable formula passed install, both-alias identity, formula test,
no-op upgrade, and uninstall checks on Apple Silicon. Intel macOS and
Linuxbrew remain unverified for `v0.2.0`.

### Install v0.2.0 on macOS, Linux, or WSL2

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

The default installation directory is `~/.local/bin`. The bootstrap prints an
absolute `rein init` command that works immediately and adds the directory to
the appropriate shell startup file for new terminals.

### Install v0.2.0 on native Windows PowerShell

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

The default installation directory is
`%LOCALAPPDATA%\Programs\Reinstate\bin`. The bootstrap adds it to the user PATH
and the current PowerShell process.

Both public bootstraps:

1. pin `v0.2.0`;
2. download the canonical installer from that exact signed Git tag;
3. verify the canonical installer SHA-256;
4. download only the matching GitHub Release asset and `checksums.txt`;
5. verify the binary checksum and reported version; and
6. preserve an existing different version until you approve replacement.

The Reinstate POSIX installer bounds replacement prompts to 30 seconds.
`REINSTATE_CONFIRM_TIMEOUT_SECONDS` may be set to an
integer from 1 through 300. It refuses immediately when the active shell cannot
perform a timed TTY read. Timeout, unsupported-shell, and invalid-value paths
all preserve the installed binary. After reviewing the requested version
change, deliberate automation may set `REINSTATE_CONFIRM_REPLACE=1`.

They install the CLI only. Interactive configuration starts when you run
`rein init`.

### Inspect before executing

macOS/Linux:

```sh
curl -fsSL https://reinstate.dev/install.sh -o reinstate-install.sh
less reinstate-install.sh
sh reinstate-install.sh
```

Windows PowerShell:

```powershell
$Installer = Join-Path $env:TEMP "reinstate-install.ps1"
Invoke-WebRequest https://reinstate.dev/install.ps1 -OutFile $Installer
Get-Content $Installer
& ([ScriptBlock]::Create([IO.File]::ReadAllText($Installer)))
```

## Verify the binary

```sh
rein version --json
rein sessions --json
rein setup check
```

`rein sessions --json` is configless and should refresh the local derived
index. Before `init`, `setup check` should identify only that the sync config
is missing. That is expected for local-only use; platform, keyring, or
installed-agent compatibility failures need to be resolved before encrypted
synchronization.

## Use the configless local index

No S3/R2 values, credentials, keyring entry, encryption passphrase, or
`config.toml` are required:

```sh
rein sessions
rein search "stripe webhook retry"
rein search "webhook retry" --agent claude --branch main
rein inspect claude:SESSION_ID
rein last --dry-run
rein resume codex:SESSION_ID --dry-run
rein fork claude:SESSION_ID --dry-run
```

Session references are canonical composite identities:

```text
claude:<native-session-id>
codex:<native-session-id>
gemini:<native-session-id>
opencode:<native-session-id>
```

A bare native ID works only when exactly one indexed agent owns it. An
ambiguous ID fails and asks for the composite form.

Search is literal and case-insensitive. Multiple terms are ANDed. Narrow with
`--agent`, `--project`, `--branch`, `--file`, and `--limit`. `sessions` and
`search` return metadata, not transcript passages. `inspect` may show a
whitespace-collapsed preview from a user-authored prompt capped at 160 Unicode
code points; it never exposes a full-transcript mode in Phase 2.

Review launch plans before starting a vendor:

```sh
rein last --dry-run --json
rein resume claude:SESSION_ID --dry-run --json
rein fork codex:SESSION_ID --dry-run --json
```

Reinstate uses an executable/argument array and the recorded working directory,
not a shell command string. Remove `--dry-run` to inherit the current terminal
and launch the same vendor. Gemini and OpenCode remain read-only and refuse
resume/fork.

Current Phase 3 source includes a deterministic `environment` report in
`inspect` and native dry-run output. It checks the fresh selected source,
workspace/repository, installed same-vendor agent, name-only capabilities, and
recognized Node/Go runtime declarations. The verifier is local-only: it does
not fetch, install, checkout, repair, run project scripts, or contact a network
service.

The first real launch for an existing session reports
`baseline.unavailable`. That is deliberate: current state is not session-start
truth. Review all warnings. On a real terminal, answer the prompt with exactly
`yes`; the default is `no`. In automation, acknowledge every exact current
warning ID for that invocation:

```sh
rein resume claude:SESSION_ID \
  --allow-environment-warning baseline.unavailable
```

Repeat `--allow-environment-warning` when the report contains multiple
warnings. Unknown, duplicate, wildcard, informational, stale, and blocker IDs
are rejected, and a partial set does not launch. Missing workspaces or
executables, unverified agent versions/layouts, known repository replacement,
stale source metadata, and verifier failures cannot be bypassed.

Only after the authorized same-vendor child exits successfully does Reinstate
store the prelaunch observation as a private
`reinstate_prelaunch_observed` baseline. A failed, declined, cancelled, or
blocked launch does not establish or update it. See
[Verified resume](verified-resume.md) for the complete contract.

On a TTY, bare `rein` opens the numbered switcher:

```text
/text       filter
i NUMBER    inspect
f NUMBER    fork
NUMBER      resume
q           cancel
```

For scripts, use `rein sessions --json`. Bare `rein` on a non-TTY exits
promptly with that hint instead of waiting for input or selecting a session.

The derived index lives at
`$REINSTATE_HOME/cache/session-index-v2.sqlite`, with owner-only sibling
`.lock` and `.write.lock` files. The first protects database lifetime/rebuild;
the second serializes ordinary writers across processes. None enters encrypted
sync. Vendor session rows can be rebuilt; the v2 store also retains private
successful prelaunch observations used by verified resume. It contains bounded user-authored
prompt search text and metadata—not assistant reasoning/messages, tool output,
environment dumps, credentials, or auth stores.

That v2 path describes the current Phase 3 source. Stable `v0.2.0` uses
`session-index-v1.sqlite` and stores no prelaunch baselines. The separate
versioned path prevents an older binary from silently rebuilding away Phase 3
comparison history.

## Configure the first sync device

Use the same canonical project ID on every device but map it to each device's
real absolute path:

```sh
rein init \
  --project local/my-project=/absolute/path/to/my-project
```

`init` prompts for the S3/R2 endpoint, bucket, and credentials. Enter the
service endpoint only; do not append the bucket name to the endpoint URL. The
bucket has its own prompt. Credential input is hidden and stored in the native
OS keyring. Reinstate probes storage before writing local configuration.

Reinstate refuses to overwrite an existing `config.toml` or `state.json` with safety
exit code `7`. Its explicit `rein init --force` path writes the previous config
and state into one
timestamped directory under `backups/` before replacing them.

Save the printed `profile_id`. It is not secret, and every later device in the
same sync set must reuse it.

Then verify:

```sh
rein setup check
rein doctor --self-test
rein list --agent all
```

The encryption passphrase is not stored. You enter it through a hidden prompt
when `status`, `push`, or `pull` needs to decrypt remote state.

## Push one session

Create or resume a harmless session in your mapped project, then find its ID:

```sh
rein list --agent claude
rein list --agent codex
```

Use an explicit agent and session ID until you intentionally decide to sync
everything:

```sh
rein push --agent claude --session SESSION_ID --dry-run
rein push --agent claude --session SESSION_ID
```

`--all` is available, but neither Reinstate nor an AI setup agent should choose
it for you.

## Configure an additional device

Run the platform installer, then reuse the first device's profile UUID and the
same canonical project ID:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path
```

Enter the same endpoint, bucket, credentials, and encryption passphrase. Keep
the bucket name out of the endpoint URL. Reinstate verifies that
`init --profile-id` can find the existing encrypted `manifest.age` before
saving local configuration; a missing profile fails without initializing the
device.

Validate without mutation:

```sh
rein setup check
rein doctor --self-test
rein status

# Claude Code
rein pull --agent claude --session SESSION_ID --dry-run

# Codex
rein pull --agent codex --session SESSION_ID --dry-run
```

You do not need to close unrelated agents. A pull only cares about the exact
session it is replacing, and if that one session is open it is restored beside
the live copy instead of being blocked. Restore with:

```sh
# Claude Code
rein pull --agent claude --session SESSION_ID
rein list --agent claude
claude --resume SESSION_ID

# Codex
rein pull --agent codex --session SESSION_ID
rein list --agent codex
codex resume SESSION_ID
```

Reinstate never needs to print transcript contents to prove success.

## Agent-assisted setup

The copy-paste prompts are version-pinned and use the same public bootstrap:

- [Claude Code setup prompt](prompts/claude-code-setup.md)
- [Codex setup prompt](prompts/codex-setup.md)

The agent may inspect, install, and run redacted checks. You must enter storage
credentials and the encryption passphrase privately, never in agent chat.

## Phase 1 sign-off

Use the
[MacBook + Windows acceptance runbook](testing/phase-1-mac-windows-acceptance.md)
to test both installers, both agent prompts, bidirectional sync, backups,
conflicts, wrong-passphrase refusal, and ciphertext-only remote storage.

## Phase 2 sign-off

The release-neutral
[local-index acceptance runbook](testing/phase-2-local-index-acceptance.md)
and [parallel operator prompts](testing/phase-2-agent-verification-prompts.md)
cover configless behavior, search/preview privacy, native resume/fork, the
interactive switcher, read-only adapters, and Phase 1 regression. Automated
implementation evidence may be recorded now; physical native macOS and Windows
reports are still pending.

## Safety defaults

- remote manifests and snapshots are encrypted;
- the local index is private derived state and never enters sync;
- local search excludes assistant messages/reasoning, tool output, environment
  dumps, and auth stores;
- auth and credential files are hard-excluded;
- pull validates before mutation and backs up existing targets;
- divergent sessions create conflict records instead of silent overwrite;
- a mutating pull never replaces a session an agent is actively using: the check
  is scoped to that exact session file, so unrelated agents in other projects
  are ignored, and a session that really is in use is restored alongside the
  live one rather than blocking (`restore.active_agent_policy`); and
- no plaintext passphrase is accepted through a normal argument or environment
  variable.

## Next steps

- [CLI reference](cli-reference.md)
- [Configuration](configuration.md)
- [Backup and recovery](backup-and-recovery.md)
- [Security model](security-model.md)
- [Troubleshooting](troubleshooting.md)
