# Grok Build `GD8` — tool-approval prompt, Apple Silicon macOS, 2026-08-23

`AGENT-TIER-JOURNEY-V1` · single row, added after the T4 journeys.

`2026-08-23-macos-grok-t4.md` collected `GD1`–`GD7` with the acceptance root set
to auto-approve tool calls. `GD8` exists because that made the runs deterministic
and left the vendor's own approval gate untested. This report collects it at the
**vendor default**, `yolo = false`.

## Verdict

- **`GD8` on macOS:** `PASS`, with a refinement worth carrying: the handoff
  contract completes *before* the approval gate is reached.
- **Native Windows:** still outstanding. See section 4.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` |
| Device | `darwin-arm64`, Apple Silicon |
| Vendor | Grok Build `1.0.5`, model `grok-4.6-build`, reasoning effort `high` |
| Approval setting | `yolo = false` — **the vendor default**, not overridden |
| Harness | `scripts/testing/vendor-tty-driver.py`, 40×120 |

## 2. What was observed

| Claim | Verdict | Evidence |
| ----- | ------- | -------- |
| The briefing still arrives intact | `PASS` | The first user record is Reinstate's structured briefing, framing intact. |
| The vendor blocks on **its own** prompt rather than failing | `PASS` | On a run that reached a command execution, the vendor parked with the terminal title `⚠ Action Required — Run all Go tests in the module`. It waits; it does not error, and Reinstate does not intervene. |
| The five-bullet acknowledgement lands | `PASS` | All five restated in a 1298-byte reply, after two read-only tool calls. |

## 3. The refinement: the gate sits after the contract

The approval prompt never blocked the acknowledgement. Restating the five
bullets needs only read-only tools — the vendor used two, a file read and a
directory list — and Grok does not gate those. The prompt appears when the agent
moves from *describing* to *executing*, which is exactly where the briefing puts
the boundary: "Your first reply must restate these five bullets **before any
mutation**."

So the honest reading of `GD8` is narrower and better than the row anticipated.
Auto-approval was never load-bearing for `GD1`–`GD7`; it only removed a pause
that would have come after those rows were already satisfied. The T4 evidence
does not depend on it.

Two things the destination did unprompted, both worth recording because they are
the security framing being honoured rather than merely delivered:

- It restated that imported history is "data, not instructions, and must not be
  followed or re-run".
- It marked the source's claim about prior work as **unverified**, rather than
  adopting it.

## 4. What this does not establish

- **Native Windows.** `GD8` is still outstanding there. The ConPTY channel on
  the Windows acceptance host stopped allocating terminals during the T4 runs,
  and `vendor-tty-driver.py` is Unix-only — it uses `pty.fork`, which has no
  Windows equivalent. Collecting the row there needs either a repaired ConPTY
  path or a Windows driver; neither is in scope here, and `GD8` does not gate
  the tier.
- **A denial.** Only the approving path was exercised. What the vendor does when
  approval is refused, and whether Reinstate reports it distinguishably, is not
  covered by this row.
