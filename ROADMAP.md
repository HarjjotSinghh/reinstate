# Roadmap

> Status legend: ✅ done · 🚧 in progress · 📋 planned · 💭 exploring · ❌ won't do (for now)

Last updated: **2026-08-05** · Maintainer: [Harjot Singh Rana](https://github.com/HarjjotSinghh)

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

Across vendors, “pick up where you left off” means the **same task in a new,
linked destination session**, with an explicit fidelity report. It does not mean
that unavailable hidden reasoning, vendor system state, credentials, or live
processes become portable.

**Multi-device encrypted sync is the entry wedge, not the entire product.**
Single-device users still live with fragmented agents, sessions, projects,
worktrees, and environments. Same primitives serve both.

Longer term, continuity includes the **agent development environment** around a
session. Reinstate should let a developer declare MCP servers, skills,
instructions, hooks/loops, plugins, marketplaces, and safe settings once, then
reconcile that desired state across supported harnesses and devices. This is a
configuration layer around existing tools, not a new harness or a reason to
copy credentials.

### Value ladder

1. One session → another (find and resume)
2. One agent → another (quota/outage/tool-switch continuation via portable handoff)
3. One project environment → another (verified resume)
4. One device → another (encrypted sync)
5. Eventually one developer → another (team continuity)

### Product layers

| Layer | Who it serves | What |
| ----- | ------------- | ---- |
| **1. Session recovery** | Everyone | discover, search, preview, resume, fork, export |
| **2. Agent portability** | Multi-agent users | continuity capsules, conversation projection, capability compare, lineage |
| **3. Environment continuity** | Serious users | MCP/skills/hooks/runtime/repo validation and repair |
| **4. Cloud continuity** | Multi-device users | encrypted sync, backup, device handoff |
| **5. Team continuity** | Teams (later) | shared checkpoints, onboarding, audit |

We do **not** aim to be:

- Another full agentic development environment (Orca / Conductor / T3 Code class)
- A custom code editor, terminal emulator, or multi-agent scheduler
- Dropbox for raw agent trees without path/environment intelligence
- A vendor-locked cloud for a single agent
- An unqualified cross-agent “same exact session” translator (native resume
  stays same-vendor; cross-agent work uses explicit portable handoffs with
  measured fidelity)

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
| Session lineage metadata for later cross-agent handoffs | 📋 |
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

## Phase 4 — Cross-agent continuation (portable handoffs) 📋

This is a **core product phase**, not a convenience export. The flagship gate:
a developer reaches the Claude Code usage limit mid-task, closes Claude, and
continues in Codex without re-explaining the work or making another Claude API
call. The reverse direction must also pass. Gemini CLI and OpenCode follow.

Cross-agent continuation creates a new destination-native session linked to the
source. It preserves every portable visible element, verifies current workspace
truth, and labels every normalization, summary, reference, redaction, and
omission. It never presents format conversion as invisible native resume.

| Item | Status |
| ---- | ------ |
| Versioned continuity-capsule schema (task, normalized events, workspace, capabilities, security, fidelity, lineage) | 📋 |
| Immutable raw-source hash + canonical record + exact destination projection | 📋 |
| Version-gated transcript readers for visible messages, tool relationships, attachments, compaction, and unknown records | 📋 |
| Deterministic no-source-model fallback for quota/outage handoff | 📋 |
| `checkpoint` / `balanced` / `full` context policies with size/token preview | 📋 |
| `rein handoff` / `rein resume --with <agent>` + dry-run and inspect/export UX | 📋 |
| Workspace truth and destination capability diff before launch | 📋 |
| Claude Code ↔ Codex structured handoff, both directions | 📋 |
| Destination acknowledgement before continuing mutations | 📋 |
| Gemini CLI + OpenCode source/target support | 📋 |
| Grok Build, Copilot CLI, Cursor/agent CLI, Orca, and others based on adapter evidence | 💭 |
| Experimental target-native reconstruction only for exact supported pairs/versions | 📋 |
| Directional compatibility matrix and synthetic/adversarial acceptance suite | 📋 |
| SessionExecutor / ACP launch integration without owning the agent loop | 📋 |

Fidelity model:

| Mode | Meaning |
| ---- | ------- |
| **Native resume** | Claude → Claude Code; vendor session semantics retained (highest fidelity) |
| **Structured handoff** | Claude → capsule → Codex; task state + selected verbatim history + evidence (default cross-agent path) |
| **Reconstructed conversation** | Portable visible history projected into a new native session (experimental, pair/version-specific) |

Cross-agent records classify components as `exact`, `normalized`, `summarized`,
`referenced`, or `omitted` with a reason. User messages and visible assistant
replies should survive subject to explicit redaction. Source system/developer
instructions are audit history, not destination authority; old tool calls are
evidence and are never re-executed. Credentials, approvals, hidden reasoning,
and live process state do not transfer.

Detailed product, architecture, security, test, and delivery plan:
[docs/cross-agent-continuation.md](docs/cross-agent-continuation.md).

---

## Phase 5 — Universal configuration + automatic cross-device sync 📋

Original multi-device superpower, now extended from sessions to the safe,
portable parts of an AI development environment.

**Gate:** define an MCP server such as Mobbin once, preview and apply the
correct native configuration to at least Claude Code, Codex, Grok, and
OpenCode, then reproduce the non-secret desired state on a second device.
Unsupported mappings and missing authentication must be explicit.

### 5A. Universal agent configuration

| Item | Status |
| ---- | ------ |
| Canonical, versioned desired-state profile (“master config”) | 📋 |
| Configuration-adapter contract: import, normalize, diff, render, validate | 📋 |
| Target selection by harness, device, project, and profile | 📋 |
| MCP server declarations and `rein mcp add/list/remove` workflow | 📋 |
| Skills and instruction files | 📋 |
| Hooks, reusable commands, and agent loops/workflows | 📋 |
| Plugins/extensions and marketplace/registry declarations | 📋 |
| Extensible capability schema for future harness features | 📋 |
| `rein config import/diff/apply/status` with dry-run, backup, atomic write, rollback | 📋 |
| Drift detection without overwriting unrelated native settings | 📋 |
| Capability matrix with explicit unsupported/lossy mappings | 📋 |
| Supply-chain policy: source/version pinning, digests, permissions, confirmation | 📋 |
| Claude Code, Codex, Grok, OpenCode, and Gemini CLI config targets | 📋 |

Harnesses use different schemas and install mechanisms. Reinstate will
normalize portable intent and let adapters render each harness's native format;
it will not copy Claude Code JSON wholesale into Codex or silently discard
unsupported fields.

### 5B. Authentication coordination

| Item | Status |
| ---- | ------ |
| Portable secret references (never raw secret values in the profile) | 📋 |
| Per-device/per-harness auth status without revealing tokens | 📋 |
| OS-keychain or explicit secret-provider resolution | 📋 |
| Guided official login flows for targets that require separate auth | 📋 |
| Safe token reuse only where the protocol/provider/harness explicitly supports it | 💭 |

The goal is **configure once, authenticate as few times as safely possible**.
Raw API keys, OAuth tokens, cookies, and vendor credential stores remain
excluded from sync.

### 5C. Cloud continuity

| Item | Status |
| ---- | ------ |
| Hardened push/pull habit (hooks: pull on start / push on exit) | 📋 |
| Device registry + revocation | 📋 |
| Key rotation helpers | 📋 |
| Machine migration UX | 📋 |
| Additional backends (WebDAV, GCS) | 📋 |
| Encrypted sync scopes for sessions and non-secret desired-state profiles | 📋 |
| Cross-device configuration reconciliation and drift reports | 📋 |
| Append-aware delta / CAS for large histories | 📋 |

Detailed design direction:
[docs/universal-configuration.md](docs/universal-configuration.md).

---

## Phase 6 — Reinstate Console (thin client, not a harness) 💭

Optional UI that **selects and prepares** sessions; agents still **execute**.

| Item | Status |
| ---- | ------ |
| Unified transcript / task browser | 💭 |
| Agent selector + capability viewer | 💭 |
| Workspace status + mismatch warnings | 💭 |
| ACP client for compatible agents (`session/load` / `session/resume`) | 💭 |
| Local web or lightweight desktop shell | 💭 |

Reinstate owns continuity **before and after** execution. During execution,
Claude Code / Codex / Gemini / OpenCode own the agent loop.
ACP can reduce client/executor integration work, but resuming a session owned by
one ACP agent does not itself define cross-vendor transcript import.

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
| Unqualified silent Claude↔Codex “same exact session” translation | Formats, tools, policies, and hidden state differ; use explicit handoffs with component-level fidelity |
| Copying source system prompts, approvals, or tool calls into destination authority | Imported history is untrusted evidence; destination policy and authorization win |
| Multi-tenant real-time CRDT collab | Sequential dual-machine (and dual-agent) use first |
| Replacing git | Git remains source truth; Reinstate is context truth |
| Shipping vendor API keys or copying vendor auth stores | Use local secret references and supported login flows; credentials never synced |
| Reinstate-owned plugin runtime or agent marketplace | Coordinate native harness mechanisms; do not become an execution ecosystem |

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

Adapters implement discovery/transform; continuation adapters build capsules
and destination projections; executors implement launch/resume.
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
- switch agents when a subscription usage window, outage, or task fit demands it
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
- Continue in another agent when the current agent hits a usage limit
- Back up sessions automatically
- Configure MCP servers once across harnesses and devices
- Install the same skills, loops, plugins, and marketplaces across harnesses
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
- **Cross-agent GA:** no source-model dependency, Claude ↔ Codex bidirectional
  acceptance, fidelity/security gates, and explicit experimental labels for any
  reconstructed native history
- Releases: signed GitHub tags, checksums, SBOMs, source archive, and artifact
  attestations; see [RELEASING.md](RELEASING.md)

## How to propose roadmap items

1. Open a feature request issue
2. Tag with `roadmap`
3. Prefer expansions that reuse session discovery, index, checkpoint, and
   executor primitives — not unrelated product surface

Stars and issues both help prioritize — thank you for using Reinstate.
