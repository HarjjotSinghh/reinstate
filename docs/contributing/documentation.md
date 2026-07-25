# Documentation workflow

Documentation is a product surface, not cleanup after the code.

## Rules

- Describe released or verified behavior only.
- Keep manual setup and Claude Code/Codex agent prompts equivalent.
- Use exact versions where compatibility depends on a vendor layout.
- Never add real sessions, credentials, passphrases, bucket names, or private
  paths to examples or screenshots.
- Update CLI/config docs, the compatibility matrix, and CHANGELOG together when
  user-visible behavior changes.
- Use relative links for repository documents.

## Validate

```bash
make docs-check
make fixture-scan
```

The docs check validates local Markdown links, release/support claims, workflow
pinning, and installer contracts. External links should point to primary
project documentation when available.

Changes to `docs/prompts/` must preserve the authority boundary: the agent may
download, verify, install, and run redacted checks; the human enters secrets and
approves mutation.
