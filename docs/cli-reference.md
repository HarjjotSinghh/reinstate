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
rein init --endpoint URL --bucket NAME [--region auto] [--prefix ...]
rein list [--agent claude|codex|all] [--json]
rein status [--json]
rein diff [--json]
rein push [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein pull [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein conflicts list|show|resolve ...
rein completion bash|zsh|fish|powershell
```

Non-interactive encryption uses `REINSTATE_PASSPHRASE` (never a CLI flag).
Storage credentials use `REINSTATE_S3_ACCESS_KEY_ID` / `REINSTATE_S3_SECRET_ACCESS_KEY`.
Override home with `REINSTATE_HOME` (absolute path).
