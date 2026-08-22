# OpenCode T4 journey — native Windows x64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

Counterpart to `2026-08-22-macos-opencode-t4-journey.md`. It records the native
Windows evidence for OpenCode as a structured handoff **destination**.

## Verdict

- **Windows journey:** `PASS` for planning and the no-write invariant.
- **Not collected:** the executed launch and its lineage reconciliation. See
  section 4 — this is a harness limit, not a product one, and the behaviour it
  would have shown is covered by a test.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| Vendor | OpenCode `1.18.21` |
| `XDG_DATA_HOME` | `C:\accept\roots\ocdata` (throwaway, own credential) |
| `CODEX_HOME` | `C:\accept\roots\codex-src-t4` (throwaway, synthetic source) |
| `REINSTATE_HOME` | `C:\accept\roots\reinhome-oc-t4` (throwaway) |

## 2. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `OT0` | `PASS` | The vendor created its own store under the redirected root. |
| `OT1` | `PASS` | The synthetic Codex source was indexed. |
| `OD1` | `PASS` | The plan carries OpenCode's own flag with the briefing as its value: `--prompt "Reinstate structured handoff … Read C:\…\handoffs\4c608c7f…"`. The flag survives the pipeline's substitution. |
| `OD5` | `PASS` | Vendor directory entries unchanged across planning: `auth.json`, `log`, `mcp-auth.json`, `opencode.db`, `opencode.db-shm`, `opencode.db-wal`, `repos`, `snapshot`. |
| `OD2` | `PASS` | The handoff planned and wrote its files, exit `0`. |

At reconciliation time the store was `opencode.db` **4096 bytes** beside an
`opencode.db-wal` of **836392 bytes** — the state that used to make this
destination's lineage unresolvable, present on this platform too.

## 3. Why the flag matters

OpenCode reads a bare positional as a project path. A substitution that replaced
the whole argv with the briefing would plan a launch into a directory named
after the entire briefing. `OD1` pins that the planned argv carries the
bootstrap as the **value of the flag**, which is what the pipeline's positional
substitution finds and replaces.

## 4. What this journey does not establish

**The executed launch and its lineage reconciliation.** `rein handoff --to
opencode` without `--no-launch` starts OpenCode's interactive session, and this
run had no operator at the console. With `--no-launch` the plan carries no
destination state, because nothing was launched for a state to describe.

That is the row that would show reconciliation resolving a session the vendor
had only just created. What replaces it here is a test rather than a claim:
`TestOpenCodeVerifyResolvesAnUncheckpointedSession` stages exactly the vendor's
on-disk state — WAL journalling, automatic checkpointing off, the session rows
present only in the log — and asserts both halves that matter, that Verify
resolves the session **and** that the vendor's directory is byte-identical
afterwards.

Also not established: a vendor-created source (the Codex source was
synthesized, because driving Codex needs its own credential), and encrypted
sync, which OpenCode does not claim.

## 5. Harness defects corrected before any verdict was recorded

- The first run deleted `opencode.db` to start clean. That file **is** the
  agent probe's root marker, so OpenCode read as not installed and the handoff
  refused with `destination agent "opencode" is UNTESTED`. A destination host
  that has never run OpenCode genuinely has no store; the journey now lets the
  vendor create one first, as a real host would.
- `--json` requires `--dry-run` or `--no-launch`; the first attempt passed
  neither and exited `2`.
- An em dash in the harness script was re-encoded on the way to the host into
  bytes containing a quote, which broke the script rather than the run.
