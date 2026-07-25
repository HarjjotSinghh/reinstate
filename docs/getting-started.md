# Getting Started

**Reinstate** syncs AI coding agent sessions (and later: MCP configs, skills,
settings) across your machines — encrypted, vendor-neutral, bring-your-own
storage.

> **Status:** repository and docs are live; the v0.1 CLI is pre-release.
> Commands below describe the implemented Phase 1 sessions workflow. Follow
> [ROADMAP.md](../ROADMAP.md) for ship status.

## Prerequisites

- Two (or more) machines you develop on (e.g. Windows desktop + MacBook)
- At least one supported agent installed (Claude Code and/or Codex first)
- An S3-compatible object-storage backend you control
  - **Recommended:** Cloudflare R2 (10GB free tier is plenty for sessions)

## Install (target)

### Release binary (recommended)

```bash
# Pin the exact release; do not install an unverified moving branch.
VERSION=vX.Y.Z  # replace with an exact published tag
curl -fsSLO \
  "https://raw.githubusercontent.com/HarjjotSinghh/reinstate/$VERSION/scripts/install.sh"
REINSTATE_VERSION="$VERSION" sh ./install.sh

# Or download from GitHub Releases
# https://github.com/HarjjotSinghh/reinstate/releases
```

### From source

```bash
git clone https://github.com/HarjjotSinghh/reinstate.git
cd reinstate
make build
sudo cp bin/reinstate /usr/local/bin/   # optional
reinstate version
```

### Go install

```bash
go install github.com/HarjjotSinghh/reinstate/cmd/reinstate@vX.Y.Z
```

## First device (desktop)

```bash
# Interactive setup stores S3/R2 credentials in the OS keyring.
reinstate init \
  --project github.com/acme/app=/absolute/path/to/app

# Save the printed profile_id somewhere non-secret for Device B.
reinstate doctor --self-test
reinstate push --all --dry-run
reinstate push --all

# Optional: inspect remote vs local
reinstate status
```

### What `init` asks

1. **Backend** — R2 / S3-compatible endpoint and bucket
2. **Credentials** — stored in the OS keyring (never uploaded)
3. **Profile** — generated on Device A; reused on every later device
4. **Path map** — the same canonical project ID mapped to each local root
   (`C:\Users\you\work` ↔ `/Users/you/Projects`)

## Second device (laptop)

```bash
reinstate init \
  --profile-id <PROFILE_UUID_FROM_DEVICE_A> \
  --project github.com/acme/app=/different/local/path

reinstate doctor --self-test
reinstate status                    # hidden passphrase prompt validates remote access
reinstate pull --all --dry-run      # validates/decrypts; does not mutate
reinstate pull --all                # backs up, atomically restores, verifies discovery
```

Then resume in your agent as usual:

```bash
claude --resume
# or
codex resume
```

## The magic moment

```
Windows desktop (8h of sessions)  →  reinstate push
MacBook on the couch              →  reinstate pull
                                    →  claude --resume  # same thread
```

Path remapping rewrites embedded `cwd` / project slugs so resume finds the
session even when absolute paths differ across OSes.

## Daily workflow

```bash
# Manual
reinstate push --all
reinstate pull --all

# Automated (shell hooks — roadmap)
# pull on shell start, push on exit
reinstate hooks install
```

Phase 1 syncs Claude Code and Codex session files only. MCP, skills, agent
configuration, and background hooks are later phases.

## Safety defaults

- Encryption is **on** — no plaintext cloud copies
- Credential files (`auth.json`, tokens) are **never** synced
- Pull never silent-overwrites: conflicts fork; backups are created first
- First pull on a new device defaults to dry-run confirmation

## Next steps

- [Architecture](architecture.md)
- [Security model](security-model.md)
- [Supported adapters](adapters.md)
- [Troubleshooting](troubleshooting.md)
- [Comparison](comparison.md)

## Need help?

See [SUPPORT.md](../SUPPORT.md).
