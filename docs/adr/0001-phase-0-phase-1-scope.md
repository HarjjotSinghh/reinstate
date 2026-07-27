# ADR 0001: Phase 0 and Phase 1 scope

## Status

Accepted

## Context

Reinstate needs a trustworthy first public release that can install on macOS and
Windows, encrypt session data, and resume same-vendor Claude Code and Codex
sessions across devices. Earlier marketing language mixed multi-agent
MCP/skills scope with the unfinished scaffold, inventing release history
(`v0.0.0`) and phase numbers that did not match implementation reality.

## Decision

1. **Phase 0** is the verified foundation: contracts, CLI routing, config/state,
   diagnostics, fixtures, installers, CI/release trust, and documentation.
2. **Phase 1** is the complete **sessions-only** product for **Claude Code** and
   **OpenAI Codex** over R2/S3-compatible storage with age encryption and
   cross-OS path mapping.
3. The first stable public release is **`v0.1.0`**.
4. MCP servers, skills, hooks/loops, plugins, marketplaces, settings, and
   additional agents are **post-Phase-1**. Their eventual cross-harness
   configuration layer is documented separately and does not expand `v0.1`.
5. Authentication files and credentials are never synced.
6. Same-vendor resume only — no cross-agent transcript translation.

## Consequences

- Roadmap, changelog, README, citation metadata, and support docs must match
  this authority.
- Adapter compatibility is explicit (`SUPPORTED` / `UNTESTED` / `UNSUPPORTED` /
  `NOT_INSTALLED`).
- SemVer tags are created only when installable, documented, and tested gates
  pass — not for intermediate scaffold commits.
