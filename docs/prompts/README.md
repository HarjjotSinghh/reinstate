# AI-agent setup prompts

Version-pinned prompts for installing Reinstate with a coding agent.

| Prompt | Audience |
| ------ | -------- |
| [claude-code-setup.md](claude-code-setup.md) | End-user install via Claude Code |
| [codex-setup.md](codex-setup.md) | End-user install via Codex CLI |
| [contributor-setup.md](contributor-setup.md) | Contributors cloning the repo |

Rules common to all prompts:

- Never disable sandbox/approval protections
- Never read or sync auth files / credential stores
- Never accept passphrases in chat
- Install only from a verified GitHub Release (checksums)
- Continue through init, doctor, dry-run, sync, and post-restore discovery
- Finish with a redacted report; stopping at “run `rein init`” is incomplete
