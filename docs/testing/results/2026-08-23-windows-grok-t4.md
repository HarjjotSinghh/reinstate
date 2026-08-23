# Grok Build T4 journey — native Windows x64, 2026-08-23

`AGENT-TIER-JOURNEY-V1` · single agent, single platform, handoff-destination rung.

Builds on `2026-08-22-windows-grok-t3.md`, which established verified resume on
this device. This report covers only what T4 adds: Grok Build as a **handoff
destination**.

## Verdict

- **Windows T4 journey:** `PASS` — 6 of 7 rows, 1 not measurable on this host.
- A handoff into Grok starts a **new** session. It is never a cross-agent
  resume, and nothing reconstructs the source thread.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| Vendor | Grok Build `1.0.5` |
| Model that answered | `grok-4.6-build`, reasoning effort `high` |
| Source agent | a synthetic Claude Code session under a throwaway `CLAUDE_CONFIG_DIR` |
| Destination | real Grok Build against a throwaway `GROK_HOME` |
| Workspace | `C:\accept\roots\ws-grok-t4`, a real Git repository |

`REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `GROK_HOME` and `USERPROFILE` all pointed
at throwaway roots. The destination root is a copy of a previously
credentialed acceptance root, never the operator's own Grok tree, and its
session store was emptied before the run. No credential was read or written by
this journey.

## 2. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `GD1` | `PASS` | Plan is `grok --session-id <uuid> "<briefing>"` with `cwd` equal to the verified workspace. Every planned file is under `REINSTATE_HOME`; none under the Grok root. |
| `GD2` | `PASS` | Executing the handoff created a **new** session whose id is exactly the UUID the plan pinned. The first user record is Reinstate's structured briefing, carrying the "not native resume" framing and the acknowledgement contract. The vendor restated all five required bullets — goal, constraints, changed files and test state, missing capabilities, next action — and carried the source marker across. |
| `GD3` | `PASS` | Two lineage records: `unresolved` / `launched=false` written **before** launch, then `resolved` / `launched=true` naming the pinned id. |
| `GD4` | `PASS` | An interrupted earlier handoff left its pinned id in the store; the next plan refused with the session-id collision error and exit `5`, without touching the vendor store. |
| `GD5` | `PASS` | `--no-redact` refused with exit `2`, and the human output printed the repository-upload warning naming the destination direction. |
| `GD6` | `PARTIAL` | Grok → Claude Code passes (exit `0`). Grok → Codex CLI is **not measurable on this host**: Codex CLI is not installed, and Reinstate correctly refuses with `destination agent "codex" is NOT_INSTALLED`, exit `5`. That is an environment fact, not a regression. |
| `GD7` | `PASS` | Recursive listing of the Grok root across a dry-run: `3391` files before, `3391` after. No session files, no directory-trust record. |

## 3. What the harness had to get right

Three harness defects produced wrong readings before any verdict was recorded.
Each is noted so the next run does not repeat them.

- **Killing Reinstate along with the vendor loses `GD3`.** Lineage is written
  `unresolved` before launch by design, then rewritten `resolved` after the
  child exits. Stopping `rein` together with `grok` leaves only the pre-launch
  record, which reads as a failure that never happened. Stop the **vendor**
  only, and let Reinstate observe the exit.
- **Bypassing the TTY gate changes vendor behaviour.**
  `REINSTATE_ALLOW_NON_TTY_LAUNCH=1` lets the launch proceed without a
  terminal, but Grok then stalls after its first turn: it issues tool calls and
  waits on an approval prompt it has no terminal to draw. Measured: two
  processes alive and no progress for eight minutes. A T4 journey needs a real
  console; the run above used an interactive scheduled task.
- **The vendor answers in two turns, not one.** Grok's first assistant record
  is a short intent line ("Reading the handoff briefing now, then restating the
  five required bullets") with tool calls attached. The acknowledgement lands in
  a **later** record. A watcher that stops at the first assistant record, or at
  the first history write, captures the opener and reports a missing
  acknowledgement. Wait for an assistant record with substantive content.

`yolo = true` was set in the acceptance root so tool calls auto-approve and the
run is deterministic. With the vendor default the loop can block on an approval
prompt indefinitely.

## 4. What this journey does not establish

- **macOS.** The T4 rung needs a journey on both platforms; the macOS
  destination journey is recorded separately.
- **`GD6`'s Codex leg**, for the host reason above.
- **T5.** No encrypted-sync row was collected, and Grok declares no sync
  capability.
