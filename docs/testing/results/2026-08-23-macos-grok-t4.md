# Grok Build T4 journey — Apple Silicon macOS, 2026-08-23

`AGENT-TIER-JOURNEY-V1` · single agent, single platform, handoff-destination rung.

Builds on `2026-08-22-macos-grok-t3.md`. This report covers only what T4 adds:
Grok Build as a **handoff destination**.

## Verdict

- **macOS T4 journey:** `PASS` — 7 of 7 rows.
- A handoff into Grok starts a **new** session. It is never a cross-agent
  resume, and nothing reconstructs the source thread.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` |
| Device | `darwin-arm64`, Apple Silicon |
| Vendor | Grok Build `1.0.5` |
| Model that answered | `grok-4.6-build`, reasoning effort `high` |
| Source agent | a synthetic Claude Code session under a throwaway `CLAUDE_CONFIG_DIR` |
| Destination | real Grok Build against a throwaway `GROK_HOME` |
| Workspace | a throwaway Git repository under the scratch root |

`HOME`, `REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `GROK_HOME`, `CODEX_HOME` and the
four `XDG_*` variables all pointed at throwaway roots. `HOME` matters: without
it the index picked up 21 real sessions from the operator's own Gemini, Cursor,
Kimi, Qwen, Cline, Copilot and Pi trees. With it, the index holds exactly the
synthetic source session.

The credential was placed in the acceptance root by signing in with `GROK_HOME`
already redirected. The operator's own Grok tree was never read, copied, or
launched against.

## 2. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `GD1` | `PASS` | Plan is `grok --session-id <uuid> "<briefing>"` with `cwd` equal to the verified workspace. Every planned file is under `REINSTATE_HOME`; none under the Grok root. The workspace path reaches the briefing already redacted to `${REPO:…}`. |
| `GD2` | `PASS` | Executing the handoff created a **new** session whose id is exactly the UUID `GD1` pinned. The first user record is Reinstate's structured briefing, carrying the "not native resume" framing and the acknowledgement contract. The vendor restated all five required bullets and carried the source marker across. |
| `GD3` | `PASS` | Two lineage records: `unresolved` / `launched=false` written **before** launch, then `resolved` / `launched=true` naming the pinned id. Reinstate exited `0`. |
| `GD4` | `PASS` | Both halves of the corrected contract. After the completed handoff the next plan minted a **different** UUID and exited `0`. Seeding the store with a plan's own pinned id refused with the collision error and exit `5`. |
| `GD5` | `PASS` | `--no-redact` refused with exit `2`, and the human output printed the repository-upload warning naming the destination direction. |
| `GD6` | `PASS` | Grok → Claude Code and Grok → Codex CLI both plan cleanly (exit `0`), with the source-direction warning intact. |
| `GD7` | `PASS` | Recursive listing of the Grok root across a dry-run: `537` files before, `537` after, byte-identical. No session files, no directory-trust record. |

## 3. The harness needed a PTY that answers back

Reinstate's launch gate requires a real terminal on **both** stdin and stdout,
and the vendor TUI queries that terminal before it will start. Three harness
shapes failed before any verdict was recorded:

- **Redirecting the command's stdout defeats the gate.** Capturing output with
  `> log` makes stdout a file, and the launch refuses with
  `requires an interactive terminal` — which reads as a product refusal rather
  than a harness mistake.
- **A silent PTY hangs forever.** Under `script(1)` the process stayed alive
  with **zero bytes** captured. The vendor had emitted `ESC]11;?` (background
  colour) and `ESC[6n` (cursor position) and was blocking for answers that a
  passive recorder never sends.
- The fix is a PTY driver that replies: device attributes, cursor position, and
  OSC 10/11 colour. With it the first bytes arrived immediately and the run
  proceeded.

The acknowledgement also arrives **late**: 13 assistant turns were written, and
the five-bullet restatement was turn nine. A watcher that stops at the first
assistant record captures a short intent line and reports a missing
acknowledgement that did in fact arrive.

`yolo = true` was set in the acceptance root so tool calls auto-approve and the
run is deterministic; with the vendor default the loop can block on an approval
prompt no one is present to answer.

## 4. What this journey does not establish

- **T5.** No encrypted-sync row was collected, and Grok declares no sync
  capability.
