# Grok Build T3 journey — macOS arm64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

**This is not a `PHASE5-DEVICE-REPORT-V1`**, and it does **not** complete Grok's
T3 claim. `docs/agent-support-tiers.md` requires a device journey on macOS *and*
native Windows, and the conformance gate added in `#339` enforces it. The
Windows half cannot currently be collected; section 5 says why, with the
measurement.

## Verdict

- **macOS journey:** `PASS` — 6 of 6 rows.
- **Native Windows journey:** **blocked by a vendor defect**, not by Reinstate.
- **Tier claim:** still incomplete. Grok stays at its current tier until the
  Windows rows exist.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `macos-arm64` |
| Vendor | Grok Build `1.0.5` (`grok 1.0.5 (5115b46bc909) [stable]`) |
| Declared range under test | `1.0.5`–`1.0.5` |
| Redirect | `GROK_HOME` → a throwaway acceptance root with its own credential |
| Workspace | a real Git repository outside any agent tree |

The credential was placed in the acceptance root by signing in with `GROK_HOME`
already redirected. The operator's own Grok tree was never read or launched
against, and no credential was copied.

## 2. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `GC1` | `PASS` | A session created by the real vendor was indexed. |
| `GV1` | `PASS` | `agent.version=1.0.5`, `status=supported`. |
| `GR3` | `PASS` | Launch plan `grok --resume <uuid>` — a UUID, never a title — with `cwd` equal to the recorded workspace. |
| `GR1` | `PASS` | Launching Reinstate's own argv, the vendor recalled the first turn's token. Continuation observed. |
| `GR2` | `PASS` | Fork plan `--resume <uuid> --fork-session` created a **new** session (1 → 2); the original session directory was retained. |
| `GR6` | `PASS` | Agent root byte-identical across index, inspect and dry-run. |

## 3. The defect this journey found first

`GV1` passes only because of a parser fix made during this run. The shipped CLI
prints a release channel that the version pattern did not allow:

```
grok 1.0.5 (5115b46bc909) [stable]     macOS
grok 1.0.5 (5115b46bc9) [stable]       native Windows
```

The pattern was anchored immediately after the optional build id, so it matched
**neither platform**. agentcheck reports an unparsed version as `UNTESTED`, so
every `rein resume grok:<id>` and `rein fork` refused with exit `5` on both
platforms: the T3 promotion did not work at all.

The existing parser test passed throughout, because it asserted a transcription
of the output rather than the bytes the CLI writes. This was assumed to be a
Windows-only difference until the macOS host was asked for its own bytes and
gave the same suffix.

## 4. What this journey does not establish

- **T4.** No handoff-destination row was collected.
- **The process-spawn half of resume.** `GR1` executes the argv Reinstate
  planned and `GR3` proves that argv is Reinstate's own, but the launch was not
  performed by `rein resume` itself.

## 5. Why the native Windows half is missing

**Grok's headless mode does not return on native Windows.** `grok -p "<prompt>"`
produces no output at all — empty stdout *and* empty stderr — and never exits:

| Platform | Result |
| -------- | ------ |
| macOS arm64 | exit `0`, `8s`, replied |
| Windows 11 Pro `10.0.26200`, AMD64 | killed at `120s`, no output |

It is not a console problem: retried under a real ConPTY with
`IsInputRedirected=False`, it still did not complete. A session directory *is*
created before it stalls, so the vendor reaches its own store and then hangs on
the turn.

Rows that need only static observation were collected on Windows and passed —
`GV1`, `GR3`, `GR5` and `GR6`. The rows that need a completed turn, `GR1` and
`GR2`, cannot be collected autonomously there.

Interactive `grok --resume <id>` is unaffected as far as this run can tell; only
headless is. A maintainer at the physical machine could therefore observe `GR1`
and `GR2` in the TUI. Until that happens, or until the vendor fixes headless on
Windows, Grok's T3 rung has one platform of evidence and is not a tier claim.
