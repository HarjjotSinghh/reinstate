# Roadmap

> Status legend: ✅ done · 🚧 in progress · 📋 planned · 💭 exploring · ❌ won't do (for now)

Last updated: **2026-07-25** · Maintainer: [Harjot Singh Rana](https://github.com/HarjjotSinghh)

This roadmap is a living document. Priorities follow real activation signals —
especially **successfully resumed tasks per active user** — and vendor format
churn. Open an issue or discussion to influence it.

Strategic depth (positioning, ICP, validation, product boundaries):
[docs/product-strategy.md](docs/product-strategy.md)

---

## Vision

> **Reinstate is the continuity layer for coding-agent work.**
> Pick up any coding task exactly where you left it — across agents, projects,
> environments, and devices.

Explain it as:

> Search, resume, and hand off coding-agent sessions across agents, projects,
> environments, and devices — with environment verification so continuation is
> correct, not just possible.

**Multi-device encrypted sync is the entry wedge, not the entire product.**
Single-device users still live with fragmented agents, sessions, projects,
worktrees, and environments. Same primitives serve both.

### Value ladder

1. One session → another (find and resume)
2. One agent → another (portable handoff)
3. One project environment → another (verified resume)
4. One device → another (encrypted sync)
5. Eventually one developer → another (team continuity)

### Product layers

| Layer | Who it serves | What |
| ----- | ------------- | ---- |
| **1. Session recovery** | Everyone | discover, search, preview, resume, fork, export |
| **2. Agent portability** | Multi-agent users | handoffs, checkpoints, capability compare |
| **3. Environment continuity** | Serious users | MCP/skills/hooks/runtime/repo validation |
| **4. Cloud continuity** | Multi-device users | encrypted sync, backup, device handoff |
| **5. Team continuity** | Teams (later) | shared checkpoints, onboarding, audit |

We do **not** aim to be:

- Another full agentic development environment (Orca / Conductor / T3 Code class)
- A custom code editor, terminal emulator, or multi-agent scheduler
- Dropbox for raw agent trees without path/environment intelligence
- A vendor-locked cloud for a single agent
- A perfect cross-agent transcript translator (native resume stays same-vendor;
  handoffs use portable checkpoints)

---

## North-star metric

**Number of previously started coding tasks successfully resumed per active user.**

Not “number of devices connected.” That includes one-laptop and ten-device users.

---

## Phase 0 — Foundation 🚧

**Gate:** contracts, diagnostics, installers, fixtures, CI/release trust, and
honest docs.

| Item | Status |
| ---- | ------ |
| Authority docs, ADR, compatibility matrix | ✅ |
| CLI routing, stable exit codes, `version --json` | ✅ |
| Versioned config/state + atomic writes | ✅ |
| Device detection (macOS / Windows / WSL2; WSL1 refused) | ✅ |
| Redacted `doctor` + synthetic self-test | ✅ |
| Synthetic fixtures + secret scanner | ✅ |
| Hard CI/release definitions + local GoReleaser snapshot | ✅ |
| Checksum-verifying installers + tested setup prompts | ✅ |
| Clean native Windows, macOS amd64, and WSL2 acceptance | 🚧 |

---

## Phase 1 — Encrypted session sync (Claude + Codex) 🚧

**Gate:** install, `init`, and same-vendor Claude Code (then Codex) resume across
OS with encryption on and credentials never synced.

*Status note (2026-07-25): implementation and local synthetic/macOS-arm64
verification are green. Native Windows, macOS amd64, WSL2, two-device vendor
resume, remote CI, and public SemVer release certification remain gates.*

| Item | Status |
| ---- | ------ |
| R2/S3-compatible backend + memory/disk test doubles | ✅ |
| Credentials / interactive `init` | ✅ |
| age passphrase envelopes | ✅ |
| Project identity + path mapping | ✅ |
| Manifests, push/pull, conflicts | ✅ |
| Atomic restore + backups + locks | ✅ |
| Claude Code adapter implementation + synthetic fixtures | ✅ |
| Codex adapter implementation + synthetic fixtures | ✅ |
| Exact-version native device resume matrix | 🚧 |
| Complete CLI surface for sync + human docs | ✅ |
| Public `v0.1.0` release gates (signed artifacts, full OS matrix CI) | 🚧 |

---

## Phase 2 — Local universal session index 📋

**No cloud dependency.** A single-device developer gets value in under five minutes.

**Gate:** `rein` / `reinstate` with no remote config still finds and resumes
local Claude (then Codex) sessions.

| Item | Status |
| ---- | ------ |
| Local session index (all supported agents on this machine) | 📋 |
| `rein` / `reinstate` interactive session switcher (TUI/CLI picker) | 📋 |
| `rein sessions` · `rein search` · `rein inspect` · `rein last` | 📋 |
| `rein resume` (native agent launch by default) | 📋 |
| `rein fork` | 📋 |
| Search by prompt fragment, file, branch, project, agent | 📋 |
| Preview metadata (not full secret-leaking transcript dumps by default) | 📋 |
| Claude Code fully indexed first; then Codex | 📋 |
| Gemini CLI + OpenCode discovery (read path) | 📋 |

Example:

```bash
rein search "stripe webhook retry"
rein last
rein resume <session-id>
```

---

## Phase 3 — Verified resume (signature capability) 📋

**Gate:** before launch, Reinstate reports environment truth and refuses silent
bad continuation.

| Item | Status |
| ---- | ------ |
| Workspace fingerprint (repo, branch, HEAD, working tree) | 📋 |
| Agent version / layout compatibility (existing states) | 📋 |
| Skills / MCP / instruction-file presence checks | 📋 |
| Runtime checks (Node/runtime mismatches where known) | 📋 |
| Clear continue-without vs repair-first UX | 📋 |
| `rein inspect` environment report (JSON + human) | 📋 |

Example output:

```text
Repository       correct
Branch           correct
HEAD             2 commits ahead
Working tree     modified
GitHub MCP       missing
Node version     mismatch

This session can be resumed, but GitHub MCP is unavailable.
Continue without it or repair the environment first?
```

---

## Phase 4 — Cross-agent handoff (portable checkpoints) 📋

**Gate:** a task started in Claude can continue in Codex (and later Gemini)
via an explicit portable checkpoint — not silent format magic.

| Item | Status |
| ---- | ------ |
| Portable checkpoint schema (goal, decisions, done/rejected, files, tests, next action) | 📋 |
| `rein handoff` / `rein resume --with <agent>` | 📋 |
| Native resume where same vendor | 📋 |
| Verified handoff summaries everywhere | 📋 |
| Experimental native migration only for supported pairs (labeled) | 📋 |
| Capability diff (what the destination agent cannot do) | 📋 |
| SessionExecutor interface (Claude, Codex, Gemini, OpenCode) | 📋 |

Fidelity model:

| Mode | Meaning |
| ---- | ------- |
| **Native resume** | Claude → Claude Code (highest fidelity) |
| **Portable handoff** | Claude → checkpoint → Codex (explicit, lossy by design) |
| **Reconstructed conversation** | Normalized history (experimental, labeled) |

---

## Phase 5 — Automatic cross-device sync (cloud continuity) 📋

Original multi-device superpower, now layered on a useful local product.

| Item | Status |
| ---- | ------ |
| Hardened push/pull habit (hooks: pull on start / push on exit) | 📋 |
| Device registry + revocation | 📋 |
| Key rotation helpers | 📋 |
| Machine migration UX | 📋 |
| Additional backends (WebDAV, GCS) | 📋 |
| Config scopes: MCP, skills, instruction files (`--scope`) | 📋 |
| Append-aware delta / CAS for large histories | 📋 |

---

## Phase 6 — Reinstate Console (thin client, not a harness) 💭

Optional UI that **selects and prepares** sessions; agents still **execute**.

| Item | Status |
| ---- | ------ |
| Unified transcript / task browser | 💭 |
| Agent selector + capability viewer | 💭 |
| Workspace status + mismatch warnings | 💭 |
| ACP client for compatible agents (`session/resume`) | 💭 |
| Local web or lightweight desktop shell | 💭 |

Reinstate owns continuity **before and after** execution. During execution,
Claude Code / Codex / Gemini / OpenCode own the agent loop.

---

## Phase 7 — Team continuity 💭

| Item | Status |
| ---- | ------ |
| Developer-to-developer handoffs | 💭 |
| Shared session checkpoints | 💭 |
| Onboarding from prior work | 💭 |
| Reviewable agent provenance | 💭 |
| Project policies and audit logs | 💭 |
| Hosted zero-knowledge convenience layer (paid) | 💭 |

---

## Explicit non-goals (near term) ❌

| Item | Why |
| ---- | --- |
| Full agentic IDE / ADE (editor + terminal + multi-agent scheduler) | Compete with Orca, Conductor, T3 Code; dilutes continuity mission |
| Custom code editor or full terminal emulator | Out of spine; integrate instead |
| Browser automation environment | Not continuity infrastructure |
| Worktree orchestration platform | Agents/harnesses own this |
| Multi-agent PR review suite | Separate product |
| Proprietary model router / agent marketplace | Not our layer |
| Perfect silent Claude↔Codex transcript translation | Formats and tools differ; use portable handoffs |
| Multi-tenant real-time CRDT collab | Sequential dual-machine (and dual-agent) use first |
| Replacing git | Git remains source truth; Reinstate is context truth |
| Shipping vendor API keys or auth proxies | Local-only file access; credentials never synced |

### When to revisit a Reinstate-owned harness

Only if **all three** are true from real usage:

1. Users frequently choose cross-agent continuation.
2. Existing agents cannot reliably consume portable handoffs.
3. A Reinstate-owned runtime would materially improve continuity — not merely
   duplicate an existing ADE UI.

Until then: **client, not harness.**

---

## Architecture direction (executors)

Implement an execution-provider boundary (language-agnostic contract):

```text
SessionExecutor
  capabilities()
  canResume(session) → CompatibilityResult
  launch(preparedSession) → ExecutionHandle
```

Adapters implement discovery/transform; executors implement launch/resume.
Claude Code, Codex, Gemini, and OpenCode are first executors. ACP support
can later unify clients that speak the standard.

See [docs/architecture.md](docs/architecture.md) and
[docs/product-strategy.md](docs/product-strategy.md).

---

## First ICP (narrow on purpose)

> Developers who use **multiple terminal coding agents** and regularly keep
> several concurrent sessions or projects.

Multiple **physical devices are optional** for Phase 2–4 value.

Highest-value early users typically:

- use Claude Code plus Codex or Gemini
- work across multiple repositories / worktrees
- keep many terminal sessions open
- switch WSL, containers, SSH, or host environments
- already feel session-management pain

Desktop ↔ laptop remains the flagship demo, not the eligibility filter.

---

## Validation before overbuilding

Landing / docs survey:

> What problem would you use Reinstate to solve?

- Continue sessions across computers
- Find and resume old sessions
- Move sessions between coding agents
- Back up sessions automatically
- Sync MCP servers, skills, and configuration
- Recover sessions after crashes or reinstalls
- Hand work to another developer

Opt-in local signals (never transcripts): searches, old sessions resumed,
handoffs attempted, config mismatches, remote resumes.

---

## Stable release policy

- **Pre-1.0:** minor versions may include breaking CLI/config changes (CHANGELOG)
- **`v0.1.0`:** Phase 1 public — Claude + Codex session sync, path remap,
  security model enforced, and every required native cross-device resume row
  verified on one exact release candidate
- **Later minors:** Phase 2+ land behind flags or clear SemVer notes
- Releases: signed GitHub tags, checksums, SBOMs, source archive, and artifact
  attestations; see [RELEASING.md](RELEASING.md)

## How to propose roadmap items

1. Open a feature request issue
2. Tag with `roadmap`
3. Prefer expansions that reuse session discovery, index, checkpoint, and
   executor primitives — not unrelated product surface

Stars and issues both help prioritize — thank you for using Reinstate.
