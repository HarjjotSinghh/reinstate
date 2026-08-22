# Grok Build — native resume and handoff-destination acceptance

**Status:** specification only. No row below has been collected.
**Owner:** release coordinator (both devices).
**Applies to:** the catalog promotion of Grok Build from T2 to T3, and from T3
to T4.

This page exists because the tier ladder's evidence gate is physical. The code
for both rungs is merged and unit-tested, but
[agent-support-tiers.md](../agent-support-tiers.md) requires, at T3, "a physical
device journey on macOS **and** native Windows: real agent, real session,
resumed, and the continuation observed", and at T4 bidirectional destination
journeys. Neither has been run. Until the rows below are recorded in a device
report under `results/`, Grok's declared tier is an implementation claim
awaiting confirmation, and that is stated on every surface that names it.

Do not install Grok Build solely to run this. If the device already has it,
run the rows; if it does not, record the row as `UNCOLLECTED`, not `PASS`.

## Preconditions

1. Grok Build **1.0.5** on both devices. The 2026-08-17 native Windows probe
   recorded `0.2.101`; the catalog's verified range is `1.0.5`–`1.0.5`, so an
   un-upgraded Windows host is expected to report `UNTESTED` and exit `5`.
   That outcome is itself a valid row (`GV1`), but it does not satisfy `GR1`.
2. A throwaway `GROK_HOME` for every command that could write. Never point a
   probe or a launch at the operator's own tree.
3. A real console. `IsTerminal` must be true; autonomous SSH without a TTY is
   not an excuse for a missing dest-ack row.

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
| `GD4` | Re-running the same handoff refuses with the session-id collision error and exit `5`, because the vendor requires `--session-id` not already exist. |
| `GD5` | Redaction ran unconditionally on the Grok path and `--no-redact` is refused with exit `2` **in this direction**, and the human output prints the repository-upload warning naming the destination, per [session-storage/grok.md](../session-storage/grok.md). |
| `GD6` | Grok → Claude and Grok → Codex still pass, so promoting Grok to a destination did not regress it as a source. |
| `GD7` | Nothing was written under the Grok root by the handoff: no session files, and no directory-trust record. Compare a recursive listing before and after. If the destination TUI blocks on a directory-trust prompt, record that as a finding — Reinstate deliberately does not pre-accept trust for a vendor whose trust file shape it has not measured. |

## Recording

One device report per platform under `results/`, using the Phase 5 template.
Device reports are immutable once written; if a row is wrong, add a new report
rather than editing an old one.
