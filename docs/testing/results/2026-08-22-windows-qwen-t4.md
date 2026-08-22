# Qwen Code T4 journey — native Windows x64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

**This is not a `PHASE5-DEVICE-REPORT-V1`.** It records the native Windows
evidence for Qwen Code's T4 claim — structured handoff **destination** — and is
the counterpart to `2026-08-22-macos-qwen-t4.md`.

## Verdict

- **Windows journey:** `PASS` after a defect this platform found.
- **Defect found:** every Qwen handoff failed on native Windows before this run.
  It could not be seen on macOS, where the guard responsible is inert.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| Vendor | Qwen Code `0.21.13` |
| `QWEN_HOME` | `C:\accept\roots\qwen-t4` (throwaway) |
| `CODEX_HOME` | `C:\accept\roots\codex-src` (throwaway, synthetic source) |
| `REINSTATE_HOME` | `C:\accept\roots\reinhome-qwen-t4` (throwaway) |

## 2. The defect

`rein handoff codex:<id> --to qwen --dry-run` refused:

```
{ "code": "runtime",
  "message": "handoff: Qwen bootstrap is not safe to pass as argv on this platform" }
```

A briefing is multi-line by construction, so this was not an edge case: **every**
Qwen handoff failed on native Windows, while the same code passed on macOS.

The underlying hazard is real and already known here. Windows `CreateProcess`
truncates an argv element at an embedded CR/LF, and `rc.9` caught Codex
receiving only the first line of its briefing because of it. The fix then was a
fallback to a short, file-backed projection argv, and `planDestination` applies
that fallback for **every** destination.

`QwenTarget.Plan` refused before it could run. `planDestination` returns on a
`Plan` error, so the fallback that exists for exactly this case was unreachable.
The guard turned a handled condition into a hard failure.

Removing the premature refusal lets the established fallback apply. Verified on
this host, the planned argv is now:

| # | Length | Contains CR/LF | Content |
| - | ------ | -------------- | ------- |
| 0 | 12 | no | `--session-id` |
| 1 | 36 | no | the pinned UUID |
| 2 | 20 | no | `--prompt-interactive` |
| 3 | 441 | **no** | the short projection, naming the briefing file under `REINSTATE_HOME` |

No element carries a newline, so nothing can be truncated.

## 3. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `QT1` | `PASS` | The synthetic Codex source was indexed. |
| `QD1` | `PASS` | The plan carries `--session-id <uuid>` and `--prompt-interactive`, and no argv element contains CR or LF. |
| `QD3` | `PASS` | Re-planning the same handoff produced the identical pinned id, so the id is derived from the capsule rather than generated per run. |
| `QD5` | `PASS` | 27 entries under `QWEN_HOME` byte-identical across planning. Reinstate wrote nothing under the vendor root. |

## 4. What this journey does not establish

- **Executed launch and destination acknowledgement.** The rows above cover
  planning. The macOS journey executed the handoff and observed the vendor
  create the session at the pinned id with the briefing as its first record;
  this run did not repeat that on Windows.
- **A vendor-created source.** The Codex source was synthesized. The subject
  here is what Qwen does as a **destination**, and driving Codex would have
  required its own credential.
- **Encrypted sync.** Qwen is not a synced agent and this makes no such claim.
