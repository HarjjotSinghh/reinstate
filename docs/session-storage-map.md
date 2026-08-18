# Local session storage map

**Status:** Phase 4 research input
**Last researched:** 2026-08-12
**Complements:** [cross-agent-continuation.md](cross-agent-continuation.md),
[Phase 4 plan](superpowers/plans/2026-08-12-phase-4-cross-agent-handoff-plan.md)

This is the ground truth Phase 4 parsing depends on: where each supported
coding agent keeps its local conversation state, on every operating system
Reinstate targets, and how confident we are about each row.

Nothing here is a support claim. A layout being documented does not mean
Reinstate parses it, and parsing it does not mean handoff is supported. See
[compatibility.md](compatibility.md) for support states.

Per-agent layout detail lives under [session-storage/](session-storage/).
This file keeps the confidence vocabulary, the cross-OS summary, the reader
rules, and the sources list.

## Confidence vocabulary

Every claim below carries one of these:

| Level | Meaning |
| ----- | ------- |
| **Verified** | Reinstate already reads this layout in shipped code, with fixtures |
| **Documented** | The vendor or its docs state this; Reinstate has not shipped a reader |
| **Observed** | Community/third-party evidence only; no vendor statement |
| **Unverified** | Assumed from a sibling platform; **must** be confirmed before a reader ships |

An `Unverified` row may not be turned into a `SUPPORTED` compatibility state.
It must first be confirmed on a physical device and captured as a synthetic
fixture under `testdata/`.

## Per-agent pages

The five agents with shipped readers, and every Phase 5 candidate, have their
own page under [session-storage/](session-storage/), so that parallel work on
different agents never edits the same file:

- [Index of per-agent pages](session-storage/README.md)
- Shipped readers: [Claude Code](session-storage/claude.md) ·
  [Codex CLI](session-storage/codex.md) ·
  [Gemini CLI](session-storage/gemini.md) ·
  [OpenCode](session-storage/opencode.md) ·
  [Grok Build](session-storage/grok.md)
- Phase 5 candidates: [Kimi Code CLI](session-storage/kimi.md) ·
  [Pi](session-storage/pi.md) · [Qwen Code](session-storage/qwen.md) ·
  [Cursor CLI](session-storage/cursor.md) ·
  [GitHub Copilot CLI](session-storage/copilot.md) ·
  [Aider](session-storage/aider.md) · [Cline](session-storage/cline.md) ·
  [Roo Code](session-storage/roo.md) · [Amp](session-storage/amp.md) ·
  [OpenHands](session-storage/openhands.md) · [ZCode](session-storage/zcode.md) ·
  [MiniMax](session-storage/minimax.md)

Confirming a candidate row requires a redacted device probe; the evidence
contract is [testing/agent-storage-probe.md](testing/agent-storage-probe.md),
and the capability ladder those rows feed is
[agent-support-tiers.md](agent-support-tiers.md).

---

## Cross-OS summary

| Agent | macOS | Windows | WSL2 | Env override |
| ----- | ----- | ------- | ---- | ------------ |
| Claude Code | `~/.claude/projects/` | `%USERPROFILE%\.claude\projects\` | `~/.claude/projects/` | `CLAUDE_CONFIG_DIR` |
| Codex CLI | `~/.codex/sessions/` | `%USERPROFILE%\.codex\sessions\` | `~/.codex/sessions/` | `CODEX_HOME` |
| Gemini CLI | `~/.gemini/tmp/` | `%USERPROFILE%\.gemini\tmp\` | `~/.gemini/tmp/` | `GEMINI_CLI_HOME` |
| OpenCode | `~/.local/share/opencode/storage/` | `%USERPROFILE%\.local\share\opencode\storage\` | `~/.local/share/opencode/storage/` | `XDG_DATA_HOME` |
| Grok Build | `~/.grok/sessions/` | `%USERPROFILE%\.grok\sessions\` | `~/.grok/sessions/` | `GROK_HOME` |

Native Windows and WSL2 are **different devices** with different agent trees.
Reinstate never treats one agent-state directory as shared between them.

## Rules every Phase 4 reader must follow

1. **Never parse a partially appended record.** Take the boundary at the last
   complete line; record the byte offset and the SHA-256 of everything up to it.
2. **Never mutate a source.** Readers open read-only and never write, rename,
   truncate, or lock a vendor file.
3. **Never guess an unknown record.** Preserve it as an opaque, hashed source
   reference with `portability: referenced`.
4. **Fail closed on an unknown layout version.** An unrecognized shape produces
   a compatibility refusal (exit `5`), not a best-effort parse.
5. **Never use a contributor's real agent tree.** Fixtures under `testdata/`
   only; the fixture secret scanner gates every commit.
6. **Bound everything.** Reuse the existing `MaxJSONLineBytes`,
   `MaxSearchTextBytes`, and `MaxFileReferences` ceilings; add explicit capsule
   ceilings rather than inventing new unbounded reads.

## Sources

- [Claude Code sessions](https://code.claude.com/docs/en/sessions) · [headless mode](https://code.claude.com/docs/en/headless)
- [OpenAI Codex CLI](https://github.com/openai/codex) · [Codex session lifecycle](https://codex.danielvaughan.com/2026/06/05/codex-cli-session-lifecycle-archive-resume-fork-compact-management/)
- [Gemini CLI session management](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/session-management.md) · [checkpointing](https://google-gemini.github.io/gemini-cli/docs/cli/checkpointing.html) · [rewind](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/rewind.md) · [`chatRecordingService.ts`](https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/services/chatRecordingService.ts)
- [OpenCode CLI](https://opencode.ai/docs/cli/) · [OpenCode troubleshooting/storage](https://opencode.ai/docs/troubleshooting/) · [storage.ts](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/storage/storage.ts) · [message-v2.ts](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/session/message-v2.ts) · [global.ts](https://github.com/anomalyco/opencode/blob/dev/packages/core/src/global.ts)
- [xAI Grok Build sessions](https://docs.x.ai/build/features/sessions) · [Grok Build changelog](https://x.ai/build/changelog) · [17-sessions.md (source)](https://github.com/xai-org/grok-build/blob/main/crates/codegen/xai-grok-pager/docs/user-guide/17-sessions.md) · [`encode_cwd_dirname`](https://github.com/xai-org/grok-build/blob/main/crates/codegen/xai-grok-config/src/paths.rs)
- [Grok Build CLI wire-level analysis (privacy)](https://gist.github.com/cereblab/dc9a40bc26120f4540e4e09b75ffb547)
- Phase 4 R1–R3 research note: [research/2026-08-12-phase-4-r1-r2-r3.md](research/2026-08-12-phase-4-r1-r2-r3.md)
- Phase 4 R7 context ceilings: [research/2026-08-12-phase-4-r7-context-ceilings.md](research/2026-08-12-phase-4-r7-context-ceilings.md)
