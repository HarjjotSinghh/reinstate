# Synthetic handoff fixtures

All content here is **synthetic**. No real user transcripts, tokens,
credentials, or private paths. `make fixture-scan` fails the build if
secret-like patterns appear.

## Tree (§10)

```text
claude/{long-history,compaction,parallel-tools,subagents,attachments,
        partial-final-record,unknown-records,os-roots/{macos,windows,wsl}}/
codex/{long-history,forks,parallel-tools,reasoning-items,
       partial-final-record,unknown-records,os-roots/{macos,windows,wsl}}/
gemini/{rewind,legacy-json,jsonl}/
opencode/{storage,metadata-only}/
grok/{basic,compacted}/
adversarial/{prompt-injection,secret-leakage,fence-breakout,oversized}/  # WP-24
golden/{capsule,projection}/  # capsule goldens: WP-25
```

## Regenerable corpus

`fixture.GenerateHandoff` regenerates:

- Claude + Codex `long-history` (200 turns, fixed timestamps/IDs)
- Claude + Codex `os-roots/{macos,windows,wsl}`
- `adversarial/*` and `golden/capsule` directory placeholders

Class fixtures from WP-06..10 stay committed; do not delete them when
regenerating. Update regenerable files with:

```bash
UPDATE_HANDOFF_FIXTURES=1 go test ./internal/fixture -run TestGenerateHandoffMatchesCommitted -count=1
make fixture-scan
```
