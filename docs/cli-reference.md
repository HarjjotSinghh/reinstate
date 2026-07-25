# CLI reference

Binary names: `rein` and `reinstate` (identical behavior).

## Exit codes

| Code | Meaning |
| ---- | ------- |
| 0 | success |
| 1 | unexpected runtime failure |
| 2 | usage / invalid arguments |
| 3 | missing/invalid config |
| 4 | authentication or storage failure |
| 5 | agent/layout compatibility failure |
| 6 | sync conflict |
| 7 | safety refusal |

## Commands

```text
rein version [--json]
rein doctor [--json] [--self-test]
rein setup check [--json]
rein init [--endpoint URL] [--bucket NAME] [--region auto] [--prefix ...]
          [--profile-id UUID] [--project ID=/absolute/local/path] [--yes]
rein list [--agent claude|codex|all] [--json]
rein status [--json]
rein diff [--json]
rein push [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein pull [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein conflicts list|show|resolve ...
rein completion bash|zsh|fish|powershell
```

Interactive encryption uses a hidden terminal prompt. Non-interactive
automation must open a secret file/pipe and set `REINSTATE_PASSPHRASE_FD` to
that descriptor number; `REINSTATE_PASSPHRASE` and secret CLI flags are not
accepted.

Interactive `init` stores storage credentials in the native OS keyring.
The explicit non-interactive fallback reads
`REINSTATE_S3_ACCESS_KEY_ID` / `REINSTATE_S3_SECRET_ACCESS_KEY` without
persisting them.
Override home with `REINSTATE_HOME` (absolute path).

Before overwriting an existing target, mutating `pull` and
`conflicts resolve --keep-remote` operations refuse to restore while the
selected Claude Code or Codex process is active. Close the agent and retry.
New-session restores, `--keep-both`, and `pull --dry-run` remain available.
