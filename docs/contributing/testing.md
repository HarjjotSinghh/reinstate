# Testing and fixture policy

## Synthetic data only

Never copy files from `~/.claude`, `~/.codex`, `%USERPROFILE%\.claude`, or
`%USERPROFILE%\.codex` into this repository. Even an apparently harmless
session can contain credentials, private paths, email addresses, or proprietary
source.

The deterministic corpus lives under:

```text
testdata/adapters/<agent>/<macos|windows|wsl>/
```

`internal/fixture.Generate` defines its source content. The committed files
must match that generator exactly.

## Required checks

```bash
go test ./internal/fixture -count=1
make fixture-scan
```

The scanner rejects credential filenames, secret-like tokens, email addresses,
non-synthetic home paths, and non-example GitHub owners.

## Adapter tests

Cover detection, discovery, export/restore round trips, path rewriting,
unknown-version refusal, malformed records, oversized records, archive path
traversal, backups, and session-identity forks. Tests may use temporary
directories but must not inspect the developer's real agent home.
