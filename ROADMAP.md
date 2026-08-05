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
2. One agent → another (portable handoff)
3. One project environment → another (verified resume)
4. One device → another (encrypted sync)
5. Eventually one developer → another (team continuity)

### Product layers

| Layer | Who it serves | What |
| ----- | ------------- | ---- |
| **1. Session recovery** | Everyone | discover, search, preview, resume, fork, export |
| **2. Agent portability** | Multi-agent users | handoffs, checkpoints, capability compare |
| **3. Environment continuity** | Serious users | MCP/skills/hooks/runtime/repo validation and repair |
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

## Phase 0 — Foundation ✅

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
| Native Mac/Windows release acceptance | ✅ |

---

## Phase 1 — Encrypted session sync (Claude + Codex) ✅

**Gate:** install, `init`, and same-vendor Claude Code (then Codex) resume across
OS with encryption on and credentials never synced.

*Closed 2026-07-30 in stable `v0.1.0`. The RC8 physical run passed all 23
mandatory rows on native macOS arm64 and Windows 11 amd64, including both
same-vendor directions, encrypted remote storage, live-session safety,
backups, no-op behavior, and keep-both recovery. The stable release then fixed
the non-blocking fork-identity/repeat-write findings and re-verified the
affected behavior before publication.*

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
| Exact-version native Mac/Windows resume matrix | ✅ |
| Complete CLI surface for sync + human docs | ✅ |
| Public `v0.1.0` release gates (signed artifacts, full OS matrix CI) | ✅ |

---

## Phase 2 — Local universal session index ✅

**No cloud dependency.** A single-device developer gets value in under five minutes.

**Gate:** `rein` / `reinstate` with no remote config still finds and resumes
local Claude (then Codex) sessions.

*Closed 2026-08-05 in stable `v0.2.0`. The complete tagged-artifact matrix
passed on Apple Silicon macOS and native Windows x64, including real
same-vendor resume/fork, the TTY picker, and the Windows-only Gemini/OpenCode
physical paths. Intel macOS and Linux/WSL2 artifacts remain explicitly
preview/unverified under a v0.2.0-only waiver; issues #97 and #98 track their
deferred physical acceptance.*

| Item | Status |
| ---- | ------ |
| Private derived local session index | ✅ |
| `rein` / `reinstate` interactive numbered switcher | ✅ |
| `rein sessions` · `rein search` · `rein inspect` · `rein last` | ✅ |
| `rein resume` (same-vendor native launch) | ✅ |
| `rein fork` (same-vendor native fork) | ✅ |
| Literal search by prompt, file, branch, project, agent | ✅ |
| Bounded metadata preview; no transcript dumps | ✅ |
| Claude Code + Codex full local capabilities | ✅ |
| Gemini CLI + OpenCode read-only discovery | ✅ |
| Fixture, corruption, privacy, execution, and Phase 1 regression gates | ✅ |
| Development acceptance on physical native macOS + Windows | ✅ |
| Multi-registry release automation and protected credentials | ✅ |
| Stable package listings and post-publication install docs | 🚧 |
| Signed `v0.2.0` primary-platform acceptance and stable publication | ✅ |

Example:

```bash
rein search "stripe webhook retry"
rein last --dry-run
rein resume claude:<session-id>
```

---

## Phase 3 — Verified resume (signature capability) 🚧

**Gate:** before launch, Reinstate reports environment truth and refuses silent
bad continuation.

*Implemented in the current development source. Local verification and review
are in progress; `v0.3.0-rc.1` has not been published or physically certified.
The stable `v0.2.0` installers therefore do not include this behavior yet.*

| Item | Status |
| ---- | ------ |
| Workspace fingerprint (repo, branch, HEAD, working tree) | ✅ |
| Agent version / layout compatibility (existing states) | ✅ |
| Skills / MCP / instruction-file presence checks | ✅ |
| Runtime checks (recognized Node and Go declarations) | ✅ |
| Exact warning acknowledgement vs non-overridable blocker policy | ✅ |
| `rein inspect` environment report (JSON + human) | ✅ |
| Adversarial, performance, review, and `v0.3.0-rc.1` release gates | 🚧 |

Example output:

```text
Repository       correct
Branch           correct
HEAD             2 commits ahead
Working tree     modified
GitHub MCP       missing
Node version     mismatch

This session can be resumed, but GitHub MCP is unavailable.
TTY: Continue with these environment warnings? Type yes or no [no]
Automation: --allow-environment-warning <exact-check-id>
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
- Configure MCP servers once across harnesses and devices
- Install the same skills, loops, plugins, and marketplaces across harnesses
- Recover sessions after crashes or reinstalls
- Hand work to another developer

Opt-in local signals (never transcripts): searches, old sessions resumed,
handoffs attempted, config mismatches, remote resumes.

---

## Stable release policy

- **Pre-1.0:** minor versions may include breaking CLI/config changes (CHANGELOG)
- **`v0.1.0`:** Phase 1 public and complete — Claude + Codex session sync, path
  remap, security model enforced, and every required native cross-device
  resume row verified on one exact release candidate
- **Later minors:** Phase 2+ land behind flags or clear SemVer notes
- Releases: signed GitHub tags, checksums, SBOMs, source archive, and artifact
  attestations; see [RELEASING.md](RELEASING.md)

## How to propose roadmap items

1. Open a feature request issue
2. Tag with `roadmap`
3. Prefer expansions that reuse session discovery, index, checkpoint, and
   executor primitives — not unrelated product surface

Stars and issues both help prioritize — thank you for using Reinstate.
