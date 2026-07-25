# Agent-assisted setup

Use the versioned prompt matching the coding agent already installed:

- [Claude Code prompt](../prompts/claude-code-setup.md)
- [Codex prompt](../prompts/codex-setup.md)

Before pasting, replace `<REINSTATE_VERSION>` with an exact published tag. The
agent may detect/download/verify/install and run redacted checks. The human must
enter storage credentials and the encryption passphrase privately in
Reinstate's hidden terminal prompts—never in agent chat.

The workflow is incomplete until init, doctor self-test, sync dry-run, the
selected push/pull, and post-restore vendor discovery pass.
