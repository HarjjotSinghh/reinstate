# Agent-assisted setup

Use the versioned prompt matching the coding agent already installed:

- [Claude Code prompt](../prompts/claude-code-setup.md)
- [Codex prompt](../prompts/codex-setup.md)

The prompts pin `v0.1.0` and use the public one-line bootstrap for the
detected platform. No placeholder replacement is required. The agent may
detect, download, inspect, verify, install, and run redacted checks. The human
must enter storage credentials and the encryption passphrase privately in
Reinstate's hidden terminal prompts—never in agent chat.

The workflow is incomplete until init, doctor self-test, sync dry-run, the
selected push/pull, and post-restore vendor discovery pass.
