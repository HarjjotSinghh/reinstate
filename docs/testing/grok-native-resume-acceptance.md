# Grok Build — native resume and handoff-destination acceptance

**Status:** collected, with one row outstanding.
**Owner:** release coordinator (both devices).
**Applies to:** the catalog promotion of Grok Build from T2 to T3, and from T3
to T4. Both promotions have landed.

| Rung | macOS | Native Windows | Report |
| ---- | ----- | -------------- | ------ |
| T3 (`GS*`, `GV*`, `GR*`) | collected 2026-08-22 | collected 2026-08-22 | `2026-08-22-macos-grok-t3.md`, `2026-08-22-windows-grok-t3.md` |
| T4 (`GD1`–`GD7`) | 7 of 7 | 6 of 7 — `GD6`'s Codex leg is unmeasurable there, that host has no Codex CLI | `2026-08-23-macos-grok-t4.md`, `2026-08-23-windows-grok-t4.md` |
| T4 (`GD8`) | **outstanding** | **outstanding** | — |

`GD8` is outstanding on both devices because both T4 runs set `yolo = true` in
the acceptance root to make the vendor's tool loop deterministic. That is
disclosed in both reports, and under the rule below it means the approval prompt
was never exercised. Collecting `GD8` needs an attended run at the vendor's
default setting; it deepens T4 rather than gating it, since the tier landed on
`GD1`–`GD7`.

This page exists because the tier ladder's evidence gate is physical.
[agent-support-tiers.md](../agent-support-tiers.md) requires, at T3, "a physical
device journey on macOS **and** native Windows: real agent, real session,
resumed, and the continuation observed", and at T4 bidirectional destination
journeys.

Do not install Grok Build solely to run this. If the device already has it,
run the rows; if it does not, record the row as `UNCOLLECTED`, not `PASS`.

## Preconditions

1. Grok Build **1.0.5** on both devices. The 2026-08-17 native Windows probe
   recorded `0.2.101`; the catalog's verified range is `1.0.5`–`1.0.5`, so an
   un-upgraded Windows host is expected to report `UNTESTED` and exit `5`.
   That outcome is itself a valid row (`GV1`), but it does not satisfy `GR1`.
2. A throwaway `GROK_HOME` for every command that could write. Never point a
   probe or a launch at the operator's own tree.
3. A real console, on **both** stdin and stdout. `IsTerminal` must be true for
   each; redirecting the command's stdout to a log file makes it false, and the
   launch then refuses with `requires an interactive terminal` — a harness
   mistake that reads like a product refusal. Autonomous SSH without a TTY is
   not an excuse for a missing dest-ack row.
4. A pseudo-terminal that **answers back**. The vendor TUI queries the terminal
   before it starts — device attributes, cursor position (`ESC[6n`), and OSC
   10/11 colour (`ESC]11;?`) — and blocks until each is answered. A passive
   recorder such as `script(1)` leaves the process alive with zero bytes
   captured, which is indistinguishable from a hung vendor. The driver must
   reply.
5. Patience measured in turns, not writes. The acknowledgement is not the first
   assistant record: the vendor opens with a short intent line carrying tool
   calls and restates the five bullets several turns later (measured: turn nine
   of thirteen on macOS). A watcher that stops at the first assistant record, or
   at the first history write, will report an acknowledgement that arrived as
   missing.
6. `REINSTATE_ALLOW_NON_TTY_LAUNCH` is **not** an acceptance shortcut. It lets
   the launch proceed without a terminal, but the vendor then stalls on a
   tool-approval prompt it has no terminal to draw. Any row collected under it
   is void.

## T3 rows — native resume

| Row | What must hold |
| --- | -------------- |
| `GV1` | `rein inspect grok:<id> --json` reports `agent.version` matching `grok --version`. An out-of-range build is `UNTESTED` with the range `1.0.5–1.0.5` named in the message, and exits `5`. |
| `GR1` | `rein resume grok:<id>` launches the real `grok` against that exact session, the prior conversation is present in the resumed TUI, and the continuation is observed. Screenshot or transcript excerpt, redacted. |
| `GR2` | `rein fork grok:<id>` launches `grok --resume <uuid> --fork-session` and the vendor creates a **new** session id, leaving the original session's files unchanged. |
| `GR3` | `rein resume grok:<id> --dry-run --json` prints argv `grok --resume <uuid>` — a UUID, never a title — with `cwd` equal to the session's recorded workspace. |
| `GR4` | With that session already open in another window, `rein resume` reports `agent.active` as a **warning**, not a refusal, and proceeds after `--allow-environment-warning agent.active`. A host that cannot enumerate processes reports `unknown` and still resumes. |
| `GR5` | A session directory whose recorded `info.id` is not UUID-shaped is listed with `can_resume: false` and the reason naming title addressing, and `rein resume` on it exits `5`. Synthesize this under a throwaway `GROK_HOME`; do not rename a real session. |
| `GR6` | Reinstate wrote nothing under the Grok root during any row above. Compare a recursive listing before and after. |

## T4 rows — handoff destination

| Row | What must hold |
| --- | -------------- |
| `GD1` | `rein handoff <claude-or-codex-session> --to grok --dry-run --json` plans argv `grok --session-id <uuid> "<bootstrap>"`, `cwd` = verified workspace, and no files under the Grok root. |
| `GD2` | Executing that handoff starts a **new** Grok session — not a cross-agent resume — whose id is the planned UUID, and the destination acknowledges the briefing on its first reply. |
| `GD3` | Lineage records the destination id with state `resolved`. If the vendor did not create the pinned id, state must be `unresolved`, never a guess. |
| `GD4` | A handoff whose pinned UUID already exists under the destination session store refuses with the session-id collision error and exit `5`, because the vendor requires `--session-id` not already exist. Seed the collision by creating that session directory before planning — **re-running the same handoff does not reproduce it.** A completed handoff advances the source's lineage, so the next run derives a new handoff id and therefore a new UUID, and it correctly succeeds. Measured 2026-08-23: a second run planned a different UUID and exited `0`. |
| `GD5` | Redaction ran unconditionally on the Grok path and `--no-redact` is refused with exit `2` **in this direction**, and the human output prints the repository-upload warning naming the destination, per [session-storage/grok.md](../session-storage/grok.md). |
| `GD6` | Grok → Claude and Grok → Codex still pass, so promoting Grok to a destination did not regress it as a source. |
| `GD7` | Nothing was written under the Grok root by the handoff: no session files, and no directory-trust record. Compare a recursive listing before and after. If the destination TUI blocks on a directory-trust prompt, record that as a finding — Reinstate deliberately does not pre-accept trust for a vendor whose trust file shape it has not measured. |
| `GD8` | The destination's **tool-approval prompt** is exercised, not configured away. Run one handoff with the vendor's default approval setting — for Grok Build that is `yolo = false` — and record three things: that Reinstate's briefing still arrives intact; that the vendor blocks on its own prompt rather than failing; and that after the operator approves, the five-bullet acknowledgement lands as in `GD2`. Reinstate must neither set nor read this setting: it is vendor-internal state, and writing it would violate the same rule `GD7` protects. A journey that sets `yolo = true` has **not** collected this row, and must say so. |

## Recording

One device report per platform under `results/`, using the Phase 5 template.
Device reports are immutable once written; if a row is wrong, add a new report
rather than editing an old one.

Every deviation from vendor defaults in the acceptance root — an approval
setting, a pinned model, a reasoning effort — must be stated in the report, next
to the rows it could have changed. A run that is more deterministic than a real
operator's is only useful if the reader can see where.
