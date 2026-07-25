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

1. Confirm project root mapping in config (`path_map`)
2. Check that munged Claude project dirs were rewritten for this OS
3. Run `rein status` and `rein diff`
4. Verify session files landed under the expected `~/.claude/projects/...`

Open an issue with OS pair (e.g. Windows 11 → macOS 15), agent version, and
**redacted** paths.

## Passphrase verification failed on second device

You must use the **exact same passphrase** as device 1. There is no recovery
from a wrong phrase against existing ciphertext.

## Conflicts after using both machines the same day

Expected if both sides modified the same session. Reinstate should create a
`.conflict` fork rather than overwrite. Pick the winner manually; delete or
archive the other.

## Huge Codex history / slow sync

Large `~/.codex/sessions` trees need delta/CAS sync (roadmap). Until then:

- Use `--scope` filters / retention policies when available
- Exclude very old rollouts via config globs

## Accidentally almost synced credentials

Defaults block common credential paths. If you overrode excludes:

1. Rotate any exposed keys immediately
2. Remove objects from the remote bucket if uploaded
3. Restore excludes to secure defaults

## Agent was running during pull

Close the agent (or wait for idle), then re-pull. Mid-write interleaving can
corrupt append-only JSONL.

## Still stuck?

- [FAQ](faq.md)
- [SUPPORT.md](https://github.com/HarjjotSinghh/reinstate/blob/main/SUPPORT.md)
- [GitHub Issues](https://github.com/HarjjotSinghh/reinstate/issues)
