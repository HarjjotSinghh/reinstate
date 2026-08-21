# Phase 5 agent roster

**Tier vocabulary:** [../../agent-support-tiers.md](../../agent-support-tiers.md) ·
**Evidence contract:** [../../testing/agent-storage-probe.md](../../testing/agent-storage-probe.md)

One row per agent in scope for `v0.5.1`, with the owning task, the target tier,
and the specific thing that is currently unknown. Every candidate agent starts
at T0 with every storage row `Unverified`.

An agent whose probe finds no readable local history **stays at T0 and the task
is complete**. Recording `server_backed` accurately is a successful outcome, not
a failure. Do not stretch a tier to look productive.

---

## Shipped agents (baseline, must not regress)

| Key | Agent | Family | Tier | Phase 5 change |
| --- | ----- | ------ | ---- | -------------- |
| `claude` | Claude Code | F1 | T5 | Migrated onto the catalog, behavior unchanged |
| `codex` | Codex CLI | F1 | T5 | Migrated onto the catalog, behavior unchanged |
| `gemini` | Gemini CLI | F1 | T2 | Candidate promotion to T3 (task T-050) |
| `opencode` | OpenCode | F2 | T2 | Migrated onto the catalog, behavior unchanged |
| `grok` | Grok Build | F1 | T2 | Migrated onto the catalog, behavior unchanged |

The migration task must not change a single observable behavior for these five.
That is the acceptance criterion for T-002: the existing test suite passes with
no assertion edited.

---

## Wave A — documented terminal agents (F1)

Highest confidence, lowest cost, highest user value. These land first because
they exercise the shared `hometree` scanner and prove the catalog before the
harder families arrive.

| Key | Agent | Binary | Root override | Target | Task | Principal unknown |
| --- | ----- | ------ | ------------- | ------ | ---- | ----------------- |
| `kimi` | Kimi Code CLI | `kimi` | `KIMI_CODE_HOME` | T3 | T-020 | Two vendor mirrors give different roots: `~/.kimi-code/` vs `~/.kimi/` |
| `pi` | Pi | `pi` | `PI_CODING_AGENT_DIR`, `PI_CODING_AGENT_SESSION_DIR` | T3 | T-021 | Default session directory and on-disk format unstated; check for a machine-readable session list first |
| `qwen` | Qwen Code | unconfirmed | unconfirmed | T2 | T-022 | Whether the Gemini CLI fork relationship extends to session recording |

Pages: [kimi](../../session-storage/kimi.md) ·
[pi](../../session-storage/pi.md) · [qwen](../../session-storage/qwen.md)

Wave A notes:

- Kimi is the flagship: it documents a global `session_index.jsonl` that
  enumerates every session across every project, which would be the cheapest
  discovery path in the entire catalog. Confirm it exists before writing a
  directory walk.
- Pi sets `AI_AGENT=pi` and `PI_CODING_AGENT=true` for child processes. Use
  those for process detection instead of binary-name heuristics.
- Pi releases very frequently. Its T3 version-range policy needs a maintainer
  decision, not an executor guess.
- Qwen may be nearly free if the fork hypothesis holds, and may be a full
  reader if it does not. Establish sources before estimating.

---

## Wave B — editor-hosted and project-scoped (F3, F4)

New storage families. Each introduces a scanner the catalog does not yet have,
so these follow Wave A rather than running beside it.

| Key | Agent | Family | Target | Task | Principal unknown |
| --- | ----- | ------ | ------ | ---- | ----------------- |
| `cursor` | Cursor CLI | F3 | T1 | T-030 | Whether the key means the editor agent, the terminal agent, or both |
| `cline` | Cline | F3 | T1 | T-031 | Storage root per editor host; which file is turns vs UI state |
| `roo` | Roo Code | F3 | T1 | T-032 | Same questions as Cline, answered independently |
| `aider` | Aider | F4 | T1 | T-033 | Whether distinct runs are separable at all inside one repository log |

Pages: [cursor](../../session-storage/cursor.md) ·
[cline](../../session-storage/cline.md) · [roo](../../session-storage/roo.md) ·
[aider](../../session-storage/aider.md)

Wave B notes:

- T-031 owns the F3 multi-host root resolution and host attribution. T-032
  reuses it. Do not run them concurrently.
- F3 agents have no `PATH` executable, so no version probe and no resume argv.
  T3 is unreachable for them through the current launch mechanism, and the
  descriptor states that as a property rather than leaving it pending.
- Aider's discovery is scoped to known projects. Walking the filesystem looking
  for history files is forbidden.
- A rendered Markdown log is lossy by construction. Aider T2 is a separate
  decision with an explicit fidelity statement, not an automatic next step.

---

## Wave C — identification and honest T0

These tasks are research first. The expected deliverable for several is a
descriptor at T0 with a correct reason and a completed storage page.

| Key | Agent | Expected | Task | Question that decides it |
| --- | ----- | -------- | ---- | ------------------------ |
| `copilot` | GitHub Copilot CLI | T1 or T0 | T-040 | Local authoritative history, local cache of account state, or nothing local |
| `amp` | Amp | T1 or T0 | T-041 | Whether threads are authoritative server-side |
| `openhands` | OpenHands | T0 | T-042 | Whether any host-side artifact survives a container restart |
| `zcode` | ZCode | T0 | T-043 | Whether the Z.ai desktop application writes a local session tree |
| `minimax` | MiniMax | T0 | T-044 | Which product is meant, and whether it is a harness or only a model |
| `antigravity` | Antigravity CLI | T0 | T-045 | Whether `cache/last_conversations.json` is a session store or, as its name says, a cache |

Pages: [copilot](../../session-storage/copilot.md) ·
[amp](../../session-storage/amp.md) ·
[openhands](../../session-storage/openhands.md) ·
[zcode](../../session-storage/zcode.md) ·
[minimax](../../session-storage/minimax.md) ·
[antigravity](../../session-storage/antigravity.md)

Wave C notes:

- **Antigravity CLI was added mid-phase.** Google retired the individual OAuth
  path for Gemini CLI on 2026-06-18 and named Antigravity CLI the destination,
  so a roster that covers Gemini CLI but not its successor has a hole where
  users will be. It installs into `~/.gemini/antigravity-cli/`, inside the root
  the shipped Gemini descriptor owns, so the Gemini descriptor now excludes
  that subtree. T-045 must capture Gemini CLI evidence **before** installing
  Antigravity, because the installer copies the existing Gemini setup across.
- **The cache trap.** A local cache of server-held state looks identical to a
  local authoritative store. Distinguishing them requires observing the tree
  across a cache clear or re-login, which T-040 and T-041 must actually do
  rather than infer. Indexing a cache produces records that later vanish
  underneath the user.
- **ZCode policy is already decided.** [ADR 0004](../../adr/0004-universal-agent-coverage.md)
  decision 8 restricts the catalog to officially distributed harnesses. The
  npm `zcode-app-cli` package is an unaffiliated re-packaging of the desktop
  runtime and is not a catalog agent. Probe the Z.ai-distributed application.
- **MiniMax may not be an agent at all.** If the coding experience is a MiniMax
  model running inside another vendor's harness, there is nothing to add:
  Reinstate indexes harnesses, not models. That finding closes T-044.
- **A network-backed source would be a first.** Every current source is local
  and offline. If Amp only exposes a server API, that is a maintainer decision,
  not an executor decision. Escalate before implementing.

---

## Wave D — promotion of a shipped agent

| Key | Agent | From | To | Task | Requirement |
| --- | ----- | ---- | -- | ---- | ----------- |
| `gemini` | Gemini CLI | T2 | T3 | T-050 | A `gemini --version` output shape, a fail-closed supported range, and dual-platform physical resume journeys |

Gemini already has a verified read path and a documented `gemini --resume`.
What it lacks is a version probe, which is precisely the T3 gate. This is the
cheapest available demonstration that the ladder works in both directions:
breadth at the bottom, and a real promotion at the top.

---

## Not in scope for `v0.5.1`

- **No new T4 destinations.** Claude Code and Codex CLI remain the only handoff
  targets.
- **No new T5 sync agents.** Claude Code and Codex CLI remain the only synced
  agents.
- **No configuration adapters.** That is Phase 6; never infer configuration
  support from session support.
- **No agent not on this page.** Adding one requires a roster change reviewed
  by the maintainer, because every catalog key is a public interface.

---

## Release gate

`v0.5.1` ships when:

1. the catalog refactor has landed with the five shipped agents at unchanged
   tiers;
2. `rein doctor --agents` ships with the redacted probe contract enforced by a
   test;
3. at least six new agents have reached T1 with macOS and native Windows
   fixtures; and
4. at least three new agents have reached T2.

Agents that honestly land at T0 count toward the roster being complete, but not
toward criteria 3 and 4. If the probes come back worse than hoped and the
counts cannot be met, the release scope shrinks; the tiers do not inflate.
