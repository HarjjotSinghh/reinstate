---
title: "Troubleshoot Reinstate session sync"
description: "Fix installation, path remapping, passphrase, remote manifest, conflict, large Codex history, credential, and active-agent errors without exposing secrets."
order: 8
updatedAt: 2026-07-27
tags: ["troubleshooting", "session-sync", "path-remapping", "passphrase", "codex"]
targetQuery: "fix Reinstate session sync"
searchIntent: "troubleshooting"
draft: false
noindex: false
---

## Why is the `rein` binary not found after installation?

The shell cannot find `rein` when the installation directory is missing from
the current process's `PATH` or the binary has not been built or installed.

```bash
which rein reinstate
echo "$PATH"
# rebuild from source
make build && ./bin/rein version
```

## Why does `claude --resume` not see a pulled session?

Claude Code usually misses a pulled session when the destination project path
was not remapped to the exact Claude directory key expected on this device.

1. Run `rein version --json` and require `0.1.0-rc.6` or newer.
2. Confirm the same canonical project ID maps to this device's absolute
   `local_root` in `config.toml`.
3. Run a scoped `rein pull --agent claude --session SESSION_ID --dry-run` and
   verify the planned destination is under this device's Claude project
   directory, not the source device's directory key.
4. Close Claude Code, run the scoped pull, then require both
   `rein list --agent claude --json` and `claude --resume SESSION_ID` to find
   the exact restored session.

Do not manually move the session file. RC6 rejects legacy snapshots whose
Claude project identity cannot be mapped safely; reinstall RC6 on the source
device and push that selected session again to a fresh RC6 profile.

Open an issue with OS pair (e.g. Windows 11 → macOS 15), agent version, and
**redacted** paths.

## Why does passphrase verification fail on a second device?

Passphrase verification fails when the destination does not receive the exact
passphrase that encrypted the existing remote manifest. There is no recovery
from a different passphrase against that ciphertext.

Wait until Reinstate visibly shows its hidden prompt before typing. If the
process has exited, rerun the command; otherwise the secret can become a shell
history entry instead of input to Reinstate.

## Why does Reinstate report a remote profile manifest is missing?

Reinstate reports a missing remote manifest when it can reach storage but
cannot find `manifest.age` at the configured profile coordinates.

1. Confirm `profile_id` and `storage.prefix` match the first device.
2. Confirm `storage.bucket` is the same bucket.
3. Keep the bucket name out of the service endpoint URL.

Do not create an empty manifest to bypass this check. Correct the inputs and
rerun `init --profile-id`; RC6 verifies the existing encrypted manifest before
saving the additional device's configuration.

## Why does Reinstate create a session conflict?

Reinstate creates a conflict when local and remote versions of the same session
diverge. It records the conflict instead of silently overwriting one side; use
the conflict commands to inspect the candidates and choose a resolution.

## Why is a large Codex session slow to sync?

Phase 1 transfers full immutable snapshots, so a large Codex rollout takes
longer than a future append-aware delta transfer. Select an explicit session
instead of `--all`; retention and delta controls remain roadmap work.

## Can Reinstate upload credentials from a transcript?

Adapters hard-exclude known credential artifacts, but a secret printed inside a
session transcript is part of the session payload and will be encrypted and
uploaded. Rotate an exposed credential immediately, remove affected remote
snapshots according to your storage policy, and avoid pasting secrets into
agent chats.

## Why does a pull fail while the coding agent is running?

A mutating pull refuses to replace an existing session while Claude Code or
Codex may still be writing it. Close every process for the selected agent and
retry. New-session restores, `--keep-both`, and `--dry-run` remain available.

## Still stuck?

- [FAQ](/docs/faq)
- [SUPPORT.md](https://github.com/HarjjotSinghh/reinstate/blob/main/SUPPORT.md)
- [GitHub Issues](https://github.com/HarjjotSinghh/reinstate/issues)
