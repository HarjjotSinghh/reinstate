# Synthetic handoff fixtures

All content here is **synthetic**. No real user transcripts, tokens,
credentials, or private paths. `make fixture-scan` fails the build if
secret-like patterns appear.

## Tree (§10)

```text
claude/{long-history,compaction,parallel-tools,subagents,attachments,
        absolute-paths,partial-final-record,unknown-records,
        os-roots/{macos,windows,wsl}}/
codex/{long-history,forks,parallel-tools,reasoning-items,absolute-paths,
       partial-final-record,unknown-records,os-roots/{macos,windows,wsl}}/
gemini/{rewind,legacy-json,jsonl}/
opencode/{storage,metadata-only}/
grok/{basic,compacted}/
adversarial/{prompt-injection,secret-leakage,fence-breakout,oversized}/  # WP-24
golden/{capsule,projection}/  # capsule goldens: WP-25
```

## Fixtures must resemble real installations

Two rules exist because rc.1 shipped defects that the suite could not see:

- **No `<root>/version` file under a Claude fixture.** Claude Code never writes
  one. A fixture that invents it made every source probe pass while every real
  installation was refused. Version handling is exercised where it actually
  happens, by resolving an installed executable
  (`internal/transcript/compat_test.go`).
- **Tool inputs carry absolute paths.** Real transcripts record `file_path`,
  `workdir`, and argv entries as absolute paths; `absolute-paths/` keeps that
  shape for both Claude and Codex so reader-boundary path tokenization stays
  covered. Keep those paths low-entropy — a temp-directory path is masked by
  secret redaction, which is another way to hide the defect.

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
