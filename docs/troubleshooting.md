# Troubleshooting

## Install / binary not found

```bash
which rein reinstate
echo "$PATH"
# rebuild from source
make build && ./bin/rein version
```

## `pull` does not make `claude --resume` see sessions

Usually a **path remap** issue:

1. Run `rein version --json` and require `0.1.0-rc.7` or newer.
2. Confirm the same canonical project ID maps to this device's absolute
   `local_root` in `config.toml`.
3. Run a scoped `rein pull --agent claude --session SESSION_ID --dry-run` and
   verify the planned destination is under this device's Claude project
   directory, not the source device's directory key.
4. Close Claude Code, run the scoped pull, then require both
   `rein list --agent claude --json` and
   `claude --resume SESSION_ID` to find the exact restored session.

Do not manually move the session file. RC7 rejects legacy snapshots whose
Claude project identity cannot be mapped safely; reinstall RC7 on the source
device and push that selected session again to a fresh RC7 profile.

Open an issue with OS pair (e.g. Windows 11 → macOS 15), agent version, and
**redacted** paths.

## Passphrase verification failed on second device

You must use the **exact same passphrase** as device 1. There is no recovery
from a wrong phrase against existing ciphertext.

Wait until Reinstate is visibly showing its hidden prompt before typing the
passphrase. If the process has already exited, rerun the command; otherwise the
secret can become a shell-history entry instead of input to Reinstate.

## Remote profile manifest not found

Reinstate could reach the configured storage location but did not find the
profile's encrypted `manifest.age`. Check all three non-secret coordinates:

1. `profile_id` and `storage.prefix` match the first device.
2. `storage.bucket` is the same bucket used by the first device.
3. `storage.endpoint` is the service endpoint only and does not end in the
   bucket name.

Do not work around this by creating an empty manifest or using a new profile
ID. Correct the setup inputs and rerun `init --profile-id` in a disposable or
intentionally reinitialized home. If reusing an initialized home, review it
first. RC7 provides `init --force`, which backs up config and state together
before replacing them.

## Conflicts after using both machines the same day

Expected if both sides modified the same session. Reinstate should create a
`.conflict` fork rather than overwrite. Pick the winner manually; delete or
archive the other.

## Huge Codex history / slow sync

Large `~/.codex/sessions` trees need delta/CAS sync (roadmap). Until then:

- Select a specific session instead of `--all`; retention controls are planned
- Exclude very old rollouts via config globs

## Accidentally almost synced credentials

Defaults block common credential paths. If you overrode excludes:

1. Rotate any exposed keys immediately
2. Remove objects from the remote bucket if uploaded
3. Restore excludes to secure defaults

## Agent was running during pull

Close every process for the selected agent, then re-pull. Before overwriting
an existing target, Reinstate blocks mutating pulls and `--keep-remote`
conflict resolution while Claude Code or Codex may still be writing its
session file. New-session restores, `--keep-both`, and `--dry-run` remain
available.

## Still stuck?

- [FAQ](faq.md)
- [SUPPORT.md](../SUPPORT.md)
- [GitHub Issues](https://github.com/HarjjotSinghh/reinstate/issues)
