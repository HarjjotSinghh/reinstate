# Grok Build T3 journey — native Windows x64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

Counterpart to `2026-08-22-macos-grok-t3.md`. Together they are the evidence for
Grok Build's T3 rung.

## Verdict

- **Windows journey:** `PASS` — 6 of 6 rows.
- **Continuation:** observed, not assumed.
- **Supersedes a wrong finding.** The macOS report recorded that Grok's headless
  mode does not return on this platform. That was a measurement error; section 4
  records what is actually true.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| Vendor | Grok Build `1.0.5` |
| Declared range under test | `1.0.5`–`1.0.5` |
| `GROK_HOME` | a throwaway acceptance root with its own credential |
| Workspace | `C:\accept\roots\ws-grok-journey`, a real Git repository |

The credential was placed in the acceptance root by signing in with `GROK_HOME`
already redirected. The operator's own Grok tree was never read or launched
against, and no credential was copied.

## 2. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `GS1` | `PASS` | The real vendor created a session and answered in it. |
| `GV1` | `PASS` | `agent.version=1.0.5`, `status=supported`. |
| `GR3` | `PASS` | Launch plan `grok --resume 01a02b5c-…` — a UUID, never a title — with `cwd` equal to the recorded workspace. |
| `GR1` | `PASS` | Launching Reinstate's own argv, the vendor recalled the first turn's token. |
| `GR2` | `PASS` | Fork plan `--resume <uuid> --fork-session` created a **new** session (9 → 10). |
| `GR6` | `PASS` | 4385 entries under `GROK_HOME` byte-identical across inspect and dry-run. |

## 3. Correction: headless does return

The macOS report stated that `grok -p` on native Windows "produces no output at
all — empty stdout *and* empty stderr — and never exits", and concluded that
`GR1` and `GR2` could not be collected here. **That conclusion was wrong, and
the observation behind it was an artifact of the measurement.**

What actually happens:

| Cap | Result |
| --- | ------ |
| 120 s (the original probe) | killed; stdout read as empty |
| 600 s | **`stdout: SLOWPROBE`** — the correct answer — and the process still running |

Grok's headless mode **answers correctly and then fails to terminate**. The first
probe killed it before the answer was written, and an empty file was read as "no
output". A vendor defect was reported where a too-short timeout existed.

Two things made the turn slow enough to cross that cap. This host's acceptance
root sets `fork_secondary_model = "grok-4.6"`, and the interactive session
reports `Grok 4.6 (high)` — a high-reasoning model. The macOS acceptance root
carries no model configuration at all and used the fast default, which answered
in 8 seconds. Same binary, same version, different model.

The remaining defect is real but narrower: the process does not exit after
answering. A harness that reads stdout until the answer appears and then stops
the process collects every row, which is how the rows above were gathered.

## 4. What this journey does not establish

- **T4.** No handoff-destination row was collected on this platform.
- **The process-spawn half of resume.** `GR1` executes the argv Reinstate
  planned and `GR3` proves that argv is Reinstate's own, but the launch was not
  performed by `rein resume` itself.
