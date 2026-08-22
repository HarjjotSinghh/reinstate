# Qwen Code T3 journey — native Windows x64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

**This is not a `PHASE5-DEVICE-REPORT-V1`.** It covers no release candidate, no
acceptance matrix, and no agent other than Qwen Code. It records the physical
evidence gathered on native Windows for Qwen's T3 claim, and what that claim is
still missing.

It is the counterpart to `2026-08-22-macos-qwen-t3.md`. Neither platform's
evidence is sufficient alone: `docs/agent-support-tiers.md` requires a device
journey on macOS **and** native Windows before T3, and since `#339` the
conformance suite enforces it.

## Verdict

- **Windows journey:** `PASS` — 7 of 7 rows collected, 0 failed.
- **Continuation:** observed, not assumed.
- **Tier claim:** with the macOS journey, Qwen's T3 rung now has evidence on
  both platforms. T4 is **not** covered here.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| Architecture | `AMD64` |
| Vendor | Qwen Code `0.21.13`, official npm `@qwen-code/qwen-code` |
| Declared range under test | `0.21.12`–`0.21.13` |
| Reinstate build | cross-compiled `windows/amd64` from `feat/qwen-t4` at `c778729590a3` |
| Binary SHA-256 | `7D080D184395B0290E9C0C9A1945FE4F…` |
| Reported version | `0.0.0-dev` — a local build, expected |

## 2. Isolation

Every command ran against throwaway roots. No real agent tree was read and no
real credential was used.

| Redirect | Value |
| -------- | ----- |
| `QWEN_HOME` | `C:\accept\roots\qwen` |
| `REINSTATE_HOME` | `C:\accept\roots\reinhome-qwen` |
| Workspace | `C:\accept\roots\ws-qwen`, a real Git repository |
| Model endpoint | `http://127.0.0.1:8931/v1`, loopback only |

The vendor confirmed the redirect itself, warning that `QWEN_HOME` pointed
somewhere with no settings and that existing config remained at the default
root. Model traffic went to a local stub speaking the OpenAI chat-completions
shape, selected with `OPENAI_BASE_URL`. The stub appends every request body to
a log, which is what makes row `QR1` a measurement rather than an impression.

## 3. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `QS1` | `PASS` | The real vendor created a session under the redirected root, id `700f4523-…`, exit `0`. |
| `QC1` | `PASS` | Indexed with real searchable text: `title` = "Say TOKEN-WIN-A1 and nothing else", workspace resolved to the Windows path, `message_count` 4. |
| `QV1` | `PASS` | `inspect` reports `agent.version=0.21.13`, `status=supported`, exit `0`. |
| `QR3` | `PASS` | Launch plan is `qwen --resume 700f4523-…` with `cwd` equal to the recorded workspace. |
| `QR1` | `PASS` | Executing that plan replayed the first turn: the model request carried `TOKEN-WIN-A1`, a token that existed only in the original session's history. |
| `QR2` | `PASS` | Fork plan `--resume <id> --fork-session` created a new `chats/<uuid>.jsonl`; the original file was byte-identical afterwards. |
| `QR6` | `PASS` | 27 entries under `QWEN_HOME` byte-identical across index, inspect and dry-run. |

### Why `QR1` is the row that matters

A resumed TUI showing prior turns proves the screen was populated. It does not
prove the vendor sent that history to the model. The stub logs every request
body, so the check is against what the vendor actually replayed. The token was
introduced in turn one and appeared in the request issued after the resume, so
the continuation is a measurement.

## 4. What this journey does not establish

- **The process-spawn half of resume.** `QR1` executes the argv Reinstate
  planned, and `QR3` proves that argv is Reinstate's own output, but the launch
  was not performed by `rein resume` itself — that starts an interactive TUI,
  and this run had no operator at the console. Planning is evidenced; spawning
  is not.
- **T4.** No handoff-destination row was collected on this platform. Qwen's T4
  claim rests on macOS evidence alone and is not established by this report.
- **A real model.** Every reply came from a loopback stub. This journey says
  nothing about Qwen's answers, only about session identity, storage and replay.
- **Any other agent.** Grok and OpenCode are out of scope here.

## 5. Harness defects corrected before any verdict was recorded

Recorded because each would have produced a false verdict, and because the same
traps will meet the next person to run this on Windows.

- `Start-Process -ArgumentList` joins its array with spaces and does not quote,
  so a multi-word prompt arrived as positional words and the vendor rejected it.
- The `qwen.cmd` shim is not a Win32 image, so `Start-Process qwen` fails
  outright; routing through `cmd.exe` re-splits the arguments. The Node entry
  has to be invoked directly.
- `prompt_preview` does not exist in the `sessions --json` listing; the field
  is `title`. Reading the absent field made a populated record look empty.
