# Documentation

**CLI:** prefer short alias **`rein`**. Full name **`reinstate`** is equivalent.

| Document | Description |
| -------- | ----------- |
| [Getting started](getting-started.md) | Install and dual-device setup |
| [Agent-assisted install](install/agent-assisted.md) | Version-pinned Claude/Codex setup prompts |
| [macOS install](install/manual-macos.md) | Manual install instructions |
| [Windows install](install/manual-windows.md) | Native PowerShell install instructions |
| [WSL2 install](install/manual-wsl.md) | Linux install instructions under WSL2 |
| [Verify installation](install/verify-installation.md) | Setup and dry-run gates |
| [Phase 1 Mac + Windows acceptance](testing/phase-1-mac-windows-acceptance.md) | Strict two-device release checklist |
| [Backup and recovery](backup-and-recovery.md) | Restore and conflict recovery |
| [Uninstall](uninstall.md) | Binary and data cleanup boundaries |
| [Development workflow](contributing/development.md) | Build, test, and package contributions |
| [Documentation workflow](contributing/documentation.md) | Keep docs, prompts, and claims honest |
| [Testing and fixtures](contributing/testing.md) | Synthetic-data and adapter test policy |
| [Contributing an adapter](adapters/contributing-an-adapter.md) | Fail-closed adapter requirements |
| [Release process](contributing/release-process.md) | Version and release contribution boundaries |
| [Package-manager publishing](package-manager-publishing.md) | Maintainer distribution plan, registry setup, credentials, and promotion workflow |
| [Architecture](architecture.md) | System design + continuity stack |
| [Product strategy](product-strategy.md) | Positioning, ICP, layers, non-goals |
| [Adapters](adapters.md) | Agent support matrix |
| [Features and commands](features.md) | What shipped in v0.1.0 through v0.4.0 |
| [Cross-agent handoff](handoff.md) | Phase 4 structured handoff: continue the same task in a new Claude Code or Codex session |
| [Cross-agent continuation design](cross-agent-continuation.md) | Capsule, fidelity, pipeline, and security design |
| [Local session storage map](session-storage-map.md) | Where each supported agent stores sessions, per OS, with confidence levels |
| [Universal agent configuration](universal-configuration.md) | Planned cross-harness MCP/skills/loops/plugins/settings layer |
| [Compatibility](compatibility.md) | Environments and compatibility states |
| [Security model](security-model.md) | Threat model and defaults |
| [Comparison](comparison.md) | vs alternatives |
| [FAQ](faq.md) | Common questions |
| [Troubleshooting](troubleshooting.md) | Debug guide |
| [Phase 4 handoff acceptance](testing/phase-4-cross-agent-handoff-acceptance.md) | Release-neutral cross-agent acceptance matrix |
| [ADR 0001](adr/0001-phase-0-phase-1-scope.md) | Phase 0 / Phase 1 scope decision |
| [ADR 0002](adr/0002-cross-agent-continuation.md) | Cross-agent continuation is a core explicit handoff |
| [ADR 0003](adr/0003-phase-4-rc1-scope-and-launch-route.md) | Phase 4 `v0.4.0-rc.1` scope, launch route, and capsule storage |

Project-level docs live in the repository root:

- [README](../README.md)
- [ROADMAP](../ROADMAP.md)
- [PRODUCT](../PRODUCT.md)
- [CONTRIBUTING](../CONTRIBUTING.md)
- [SECURITY](../SECURITY.md)
- [SUPPORT](../SUPPORT.md)
- [CHANGELOG](../CHANGELOG.md)
