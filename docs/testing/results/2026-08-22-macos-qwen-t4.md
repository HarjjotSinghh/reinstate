# Qwen Code T4 journey — macOS arm64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

**This is not a `PHASE5-DEVICE-REPORT-V1`.** It records the macOS evidence for
Qwen Code as a handoff **destination**, and nothing else. The T3 journey is a
separate file: [`2026-08-22-macos-qwen-t3.md`](2026-08-22-macos-qwen-t3.md).

## Verdict

- **macOS journey:** `PASS`
- **native Windows journey:** `NOT RUN` — coordinated separately
- **Tier claim complete:** **NO.** T4 inherits T3's dual-platform requirement,
  and only the macOS half exists.

## 1. Isolation

`QWEN_HOME`, `CLAUDE_CONFIG_DIR`, and `REINSTATE_HOME` all pointed at throwaway
directories. The Claude source session was **synthesised** for this journey —
two records, a fixture author and a fixture assistant reply — inside the
throwaway Claude root. No real agent tree was read and no real credential was
used. Model traffic went to a local stub speaking the OpenAI chat-completions
shape.

## 2. Vendor primitive

| Argv | Result |
| ---- | ------ |
| `qwen --session-id <uuid> --prompt-interactive "<briefing>"` | created `chats/<uuid>.jsonl` at exactly that id, with the briefing as the first `type:"user"` record, and stayed interactive |
| `qwen --session-id <existing>` | refused: "Session Id … already exists (active or archived). Delete or unarchive it first." |
| `qwen --session-id not-a-uuid` | rejected as a usage error |

A vendor that refuses a duplicate id itself is a stronger guarantee than Claude
Code's, whose collision behaviour is undocumented (see the R5 note in
`docs/session-storage/CLAUDE.md`). Reinstate still checks its own index first,
so the refusal happens before a process is spawned rather than after.

No workspace-trust prompt appeared on a first launch in a fresh root, so the
target writes no trust record. Inventing one would be a vendor-internal write
for no observed reason.

## 3. Reinstate journey

`rein handoff claude:<synthetic-id> --to qwen`, run from the throwaway
workspace with the environment warnings acknowledged.

| Step | Result |
| ---- | ------ |
| planned argv | `qwen --session-id <uuid> --prompt-interactive "<briefing>"` — the pinned id, the prompt flag, and the rendered briefing |
| dry-run vs execute | identical argv; the id is derived deterministically from the capsule |
| capability diff | `attachment/support` reported `informational` |
| files written under `$QWEN_HOME` by Reinstate | **none** — everything Reinstate wrote went to the handoff directory under `$REINSTATE_HOME` |
| destination session after launch | present at `projects/<sanitized-cwd>/chats/<pinned-uuid>.jsonl` |
| first destination record | `type:"user"`, `provenance:"real_user"`, `sessionId` equal to the pinned id, body = the briefing including the acknowledgement block |
| project bucket | the directory the vendor chose matched `QwenProjectKey(plan.Dir)` byte for byte |
| re-planning the same handoff afterwards | refused with `handoff: Qwen --session-id collided with indexed sessions; refuse rather than reuse` |

The last row is the collision guard working through the real path: the newly
created destination session was indexed, and the next plan for the same capsule
refused rather than reusing the id.

## 4. Two defects this journey found

Both were in shared handoff code, not in the Qwen target, and both are fixed in
the same change.

1. **`rewriteBootstrapArgs` dropped a destination's own flags.** The pipeline
   re-renders the briefing after the target plans, and rebuilt argv from a
   per-agent switch whose default was a single positional element — Codex's
   shape. For Qwen that discarded both `--session-id` and
   `--prompt-interactive`, so the launch would have created an *unpinned*
   session that `Verify` could never resolve. The first execute attempt showed
   exactly that argv. It now swaps the briefing into the element that already
   carried the bootstrap, so a target's own flags survive and the next
   destination does not have to remember to add a case.

2. **Capability gaps were reported as `degraded` for destinations nobody
   enumerated.** `discoverInventory` only covers Claude Code and Codex; every
   other destination arrives with an empty inventory, so each source
   instruction, MCP server, and skill would have been reported as missing from
   the destination. That asserts a gap that was never looked for. Those entries
   are now `informational` when the destination has no capability discovery.

## 5. What this journey does not establish

- **Native Windows.** Nothing here ran on Windows. The project bucket is
  lower-cased before sanitising on Windows and only there, so the destination
  directory name differs between platforms; that rule is encoded from the
  vendor's source, not observed on a Windows host.
- **A real source session.** The Claude source was synthetic.
- **Vendor model behaviour.** A local stub answered every request, so the
  destination agent's *reply* to the briefing is not evidence of anything.
- **Destination acknowledgement.** Whether Qwen restates the five bullets
  before mutating is a model behaviour, not a launch contract, and was not
  tested.
