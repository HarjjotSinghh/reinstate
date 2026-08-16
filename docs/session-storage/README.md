# Per-agent session storage pages

Each page records where one coding agent keeps its local conversation state,
on every operating system Reinstate targets, and how confident we are about
each row.

Read [../session-storage-map.md](../session-storage-map.md) first. It owns the
confidence vocabulary, the cross-OS summary, and the rules every reader must
follow. These pages are the per-agent detail.

Nothing on these pages is a support claim. A documented layout does not mean
Reinstate reads it. Support states live in
[../compatibility.md](../compatibility.md), and the capability ladder lives in
[../agent-support-tiers.md](../agent-support-tiers.md).

## Shipped readers

| Agent | Page | Tier |
| ----- | ---- | ---- |
| Claude Code | [../session-storage-map.md](../session-storage-map.md) section 1 | T5 |
| Codex CLI | [../session-storage-map.md](../session-storage-map.md) section 2 | T5 |
| Gemini CLI | [../session-storage-map.md](../session-storage-map.md) section 3 | T2 |
| OpenCode | [../session-storage-map.md](../session-storage-map.md) section 4 | T2 |
| Grok Build | [../session-storage-map.md](../session-storage-map.md) section 5 | T2 |

These five migrate into this directory during Phase 5 platform work, unchanged
in content. Until then the map remains their home.

## Phase 5 candidates

| Agent | Page | Current | Target |
| ----- | ---- | ------- | ------ |
| Kimi Code CLI | [kimi.md](kimi.md) | T0 | T3 |
| Pi | [pi.md](pi.md) | T0 | T3 |
| Qwen Code | [qwen.md](qwen.md) | T0 | T2 |
| Cursor CLI | [cursor.md](cursor.md) | T0 | T1 |
| GitHub Copilot CLI | [copilot.md](copilot.md) | T0 | T1 |
| Aider | [aider.md](aider.md) | T0 | T1 |
| Cline | [cline.md](cline.md) | T0 | T1 |
| Roo Code | [roo.md](roo.md) | T0 | T1 |
| Amp | [amp.md](amp.md) | T0 | T1 or T0 |
| OpenHands | [openhands.md](openhands.md) | T0 | T0 |
| ZCode | [zcode.md](zcode.md) | T0 | T0 |
| MiniMax | [minimax.md](minimax.md) | T0 | T0 |

Every row on every candidate page starts `Unverified`. Promotion requires a
redacted device probe; see [../testing/agent-storage-probe.md](../testing/agent-storage-probe.md).
