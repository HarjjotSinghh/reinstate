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

`rein setup check` returns compatibility exit code `5` when an installed agent
is `UNTESTED` or `UNSUPPORTED`; its summary never says all checks passed while
that agent is blocked from push/pull. `rein conflicts list` and
`rein conflicts show` require a valid config, so a missing config cannot look
like an empty conflict set.

Before overwriting an existing target, mutating `pull` and
`conflicts resolve --keep-remote` operations refuse to restore while that exact
session is in use by Claude Code or Codex. The check is scoped to the session
file being replaced, so unrelated agents running in other projects do not block
a restore. Close that one session and retry, or pass `--allow-active-agents`.
New-session restores, `--keep-both`, and `pull --dry-run` remain available.

See [Configuration](configuration.md) for `restore.active_agent_policy`
(`scoped` by default, `strict`, or `off`).

## Planned universal configuration commands

The following is roadmap direction, **not current CLI syntax**:

```text
rein mcp add|list|remove …
rein skill install|list|remove …
rein loop install|list|remove …
rein plugin install|list|remove …
rein marketplace add|list|remove …
rein config import|list|diff|apply|status|sync …
rein auth status …
```

The design goal is one non-secret desired-state profile rendered by verified
adapters into each target harness. Exact names and flags require an RFC before
implementation. See
[universal-configuration.md](universal-configuration.md).
