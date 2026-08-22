# Kimi Code CLI T2 journey — native Windows x64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

**This is not a `PHASE5-DEVICE-REPORT-V1`.** It covers no release candidate, no
acceptance matrix, and no agent other than Kimi Code CLI. It records the
evidence gathered on native Windows for Kimi's T2 claim — handoff **source**
only — and is explicit about the one thing it could not establish.

T2 requires no device journey under `docs/agent-support-tiers.md`; the
conformance gate added in `#339` applies from T3 upward. This report exists
because the reader is new and because Windows path handling is where a reader
diverges from its macOS behaviour.

## Verdict

- **Windows journey:** `PASS` — 4 of 4 rows collected, 0 failed.
- **Scope limit:** the session tree was **synthesized**, not created by the
  vendor. See section 4. This verifies the reader on Windows; it does not
  independently re-verify the record shape there.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| Architecture | `AMD64` |
| Vendor | Kimi Code CLI `0.36.1`, wire protocol `1.5` |
| Reinstate build | `feat/kimi-t2-wire-reader` `77feb77a9ee9` merged with `fix/kimi-sessionindex-fixture-shape` `e30871721c2f` |
| Why merged | The reader lands in the first; the index source's handling of `context.append_loop_event` lands in the second. Either alone would have tested half the path. |

## 2. Isolation

| Redirect | Value |
| -------- | ----- |
| `KIMI_CODE_HOME` | `C:\accept\roots\kimi-t2` |
| `REINSTATE_HOME` | `C:\accept\roots\reinhome-kimi` |
| Workspace | `C:\accept\roots\ws-kimi-t2`, a real Git repository |

No real agent tree was read and no real credential was used.

## 3. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `KC1` | `PASS` | Indexed with a real title and the Windows workspace path resolved to `C:\accept\roots\ws-kimi-t2`. |
| `KC2` | `PASS` | `resume` refused with exit `5`. Kimi declares no native resume, and the ladder above T2 is not claimed by accident. |
| `KD1` | `PASS` | Handoff projection to Claude produced a **7-event** capsule, so the wire reader parses `context.append_loop_event` end to end on Windows. |
| `KR6` | `PASS` | 7 entries under `KIMI_CODE_HOME` byte-identical across index, resume-refusal and projection. |

`KD1` reports the source token as absent from the dry-run output. That is
correct rather than a miss: a `--dry-run` capsule is a preview and carries no
verbatim bodies.

## 4. What this journey does not establish

**The session tree was synthesized.** This is the honest limit of this report
and the reason it is stated before anything else.

Kimi refuses to run without a configured model and writes **no** session tree
offline. Measured directly on this host: `kimi -p "…"` under a throwaway
`KIMI_CODE_HOME` exited `1` with `failed to run prompt: No model configured`,
and the root afterwards held only `cache/`, `logs/`, `updates/`, `device_id`
and `migrations-effort.json` — no `sessions/` directory at all.

A throwaway root therefore has no credentials, and the non-interactive route
that would avoid them, `kimi provider add <url>`, imports from a custom
registry whose `api.json` shape the CLI does not document. Guessing at it would
have produced a fixture testing the guess.

So the tree here was written by hand in the native `1.5` shape that `#341` and
`#344` pinned against the shipped bundle's op registry on macOS. What this
report establishes on Windows is that the reader, the projection and the
no-write property hold **on Windows paths**. What it does not establish is the
record shape on Windows; that still rests on the macOS work against the real
binary.

A future run with a Kimi account whose root can be handed over deliberately, or
with the registry shape documented, would close this.

## 5. Not covered

- Any tier above T2. Kimi ships no `NativeSpec` and no `VersionSpec`, and this
  report makes no resume, fork, sync or handoff-**destination** claim.
- Any other agent.
