# v0.6.0 — Reinstate Hop: cloud continuity, plus the CLI experience

Ship the hosted-tier client ("Hop") and the interactive CLI as one stable
release, certified on native Windows x64 now and on Apple Silicon macOS as soon
as that hardware is back.

**Status:** planned 2026-09-05; execution in progress on
`release/v0.6.0-rc.1`.
**Baseline:** stable `v0.5.1` (2026-08-21). `v0.5.2-rc.1` (2026-08-23) was
tagged but never certified on either platform; its content ships here and no
stable `v0.5.2` is cut.
**Source trees:** public `hop/main` (Hop client, 131 commits past `main`) and
public `main` (`v0.5.2-rc.1`). The private control plane
(`reinstate-hosted`, `hopd`) is a lab dependency only; nothing in it is
released by this plan.
**Roadmap mapping:** this is Phase 6C (cloud continuity) shipping ahead of
Phase 6A (universal configuration), which moves to `v0.7.0`.
**Acceptance:** native Windows x64 only, under the waiver in
[ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md).
macOS rows are deferred, tracked, and never claimed.
**Method:** planner–executor. The coordinator writes this plan and reviews;
Sonnet executors implement and verify per
[work-breakdown.md](work-breakdown.md).

---

## Read in this order

| Document | Purpose |
| -------- | ------- |
| [ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md) | The five decisions that constrain everything below |
| [work-breakdown.md](work-breakdown.md) | Nine workstreams, the dependency graph, who runs what |
| [file-ownership.md](file-ownership.md) | Who may edit what. Read before your first commit |
| [review-gates.md](review-gates.md) | What the coordinator checks before merging any branch |
| [task-cards/](task-cards/) | The tasks themselves, one file per workstream |
| [Windows acceptance contract](../../testing/v0.6.0-windows-acceptance.md) | What the candidate and the stable must prove on Windows |
| [clarifications.md](clarifications.md) | Questions for the maintainer, with the assumption each task proceeds under |
| [docs/hop.md](../../hop.md) | What the Hop client does, as shipped |
| [v0.5.2 plan](../v0.5.2-cli-experience/README.md) | The interactive CLI this release also carries |

---

## 1. What ships

### 1.1 The Hop client (from `hop/main`)

Every hosted journey as ordinary `rein` commands, no build tag, no feature
flag (hosted ADR 0007): `rein login` / `rein whoami`, `rein init --hop`,
`rein hop status` / `rein hop credentials`, `rein account init` / `recover` /
`join` / `status`, `rein devices` / `approve` / `revoke`, `rein sync verify`,
`rein sync migrate --to byo`, and `rein daemon` (launchd on macOS, Task
Scheduler on Windows). Underneath: the key-provider seam, refreshing S3
credentials, the root key and keyring (format 5, signed key generations,
recovery wrap inside the signature), the control-plane key-generation floor,
pairing protocol v2, and Console-initiated revocation completion.

Agent coverage moves with it: **OpenCode reaches T5** (encrypted sync) and
**Kimi Code CLI reaches T2** (handoff source).

### 1.2 The CLI experience (from `v0.5.2-rc.1`)

Bare `rein` opens an interactive switcher with a readiness verdict per row; a
warning checklist replaces retyped `--allow-environment-warning` identifiers;
a handoff studio previews every policy; `rein init` becomes a wizard with
`--link` / `--paste` pairing codes; `ctrl+k` opens a palette; `--plain` and
`REINSTATE_NO_TUI` freeze the old output. Every `--json` document and every
non-TTY byte stream stays identical to `v0.5.1`. OpenCode and Grok Build gain
verified resume (T3) and become handoff destinations (T4); Qwen Code becomes a
destination (T4).

### 1.3 Release engineering

- One merge of `main` into the Hop tree, one consolidated changelog section.
- Verified vendor ranges widened on native Windows evidence: Claude Code
  through `2.1.261`, OpenCode through `1.18.27` (see clarifications Q3).
- `RELEASING.md` gains the `v0.6.0-rc.1` candidate gate and the Windows-first
  waiver; `ROADMAP.md` records Phase 6C as shipped and moves 6A to `v0.7.0`.
- Website release truth, compatibility JSON, citation, and bootstrap contract
  move together in the release commit, exactly as `v0.5.2-rc.1` did.

### 1.4 Fixes carried

- `internal/preflight` `TestVerifyHonorsParentCancellationAndSharedDeadline`
  exceeds its 500 ms bound under a parallel `go test ./...` on this host
  (1.19 s observed; 0.4 s alone). Same class as the `TestHugeTreeFinishes`
  widening in 2bc0367f.
- `rein login` against a control plane that does not resolve must say so in
  one sentence and point at `docs/hop.md`, because the default
  `https://hop.reinstate.dev` is not yet a running service (ADR 0005, D1).
- Stale sentences in `docs/hop.md` ("The daemon follows", "Run a daemon" under
  what it does not do yet) and any other claim the doc gate or the truth pass
  finds.
- Whatever the Windows acceptance run finds. Each finding gets a regression
  test or an explicit disposition before the next candidate.

---

## 2. What this release deliberately does not do

| Not in scope | Where it lives |
| ------------ | -------------- |
| Deploying `hopd` to production or staging (hosted #6) | Founder procurement; `reinstate-hosted/LAUNCH-PLAN.md` |
| Billing, pricing page, Terms, privacy notice for the hosted tier (hosted #17) | Launch work after `v0.6.0`; hosted ADR 0009 |
| macOS acceptance of anything | Deferred to hardware return; tracked by a public issue opened by W8 |
| Universal configuration (Phase 6A) | `v0.7.0` |
| Hop Plus plan, new agents, new tiers beyond OpenCode T5 and Kimi T2 | Later |
| Cross-agent transcript translation | Never (AGENTS.md non-negotiable 3) |

---

## 3. The six invariants

Break any of these and the work is wrong regardless of whether it compiles.

1. **Frozen output stays frozen.** `--json` documents and non-TTY streams are
   byte-identical to `v0.5.1`. Row 22 of the CLI matrix measures it.
2. **Encryption on, credentials never synced, native resume same-vendor.**
   AGENTS.md non-negotiables 2 and 3. The Hop locker holds ciphertext plus the
   one documented plaintext object (`keyring.v1.json`), nothing else.
3. **Evidence before claims, per platform.** A verified range, a tier, a
   PASS row, or a "supported" sentence names the platform it was observed on.
   Windows evidence never becomes a macOS claim.
4. **No real transcripts, prompts, credentials, or private paths** in
   fixtures, reports, commits, or this plan. The secret scanner and the
   redaction tests gate every commit.
5. **Never point a test or a bench at the developer's real agent trees or the
   shared checkout's state.** Use `scripts/tuisandbox`, isolated homes,
   `CLAUDE_CONFIG_DIR` / `CODEX_HOME` / `XDG_DATA_HOME` overrides, and your own
   worktree.
6. **Do not edit an existing assertion to make a change pass.** If a refactor
   needs a different expectation, the behaviour changed; send it back.

---

## 4. Definition of done

### Candidate `v0.6.0-rc.1`

1. `release/v0.6.0-rc.1` contains the merge, every workstream's merged branch,
   and the release commit; `make verify` and `go mod tidy -diff` are clean on
   this host; `scripts/snapshot.ps1`, `stage-release-assets.ps1`,
   `check-release-artifacts.ps1`, and `test-install.ps1` pass on the exact
   commit.
2. The pre-tag Windows matrix passes on a snapshot build of that commit: every
   required row in the [Windows acceptance contract](../../testing/v0.6.0-windows-acceptance.md)
   (Phase 5 generated matrix, the 22-row CLI matrix, the Hop parity journeys),
   recorded under `docs/testing/results/`.
3. A PR from `release/v0.6.0-rc.1` to `main` is open and CI is green.
4. The founder signs and pushes `v0.6.0-rc.1` (the signing key is not on the
   Windows host; clarifications Q5). The release workflow publishes the draft
   with checksums, SBOMs, and attestations.

### Stable `v0.6.0`

5. The Windows tagged-artifact run passes every required row against the
   published `v0.6.0-rc.N` artifacts installed from the live bootstrap, per
   the candidate dispatch in
   `docs/testing/v0.6.0-rc.N-agent-verification-prompts.md`.
6. The stable promotion decision is recorded in `RELEASING.md` under the
   waiver, naming the macOS rows still owed and the issue that tracks them.
7. Release commit, signed stable tag, GitHub release, package promotion, and
   the signed `website-v` deployment tag follow `RELEASING.md` steps 1–4.

---

## 5. Release train

```text
hop/main ──┐
           ├─ release/v0.6.0-rc.1 ── W1..W6 branches merged ── release commit ── PR → main
main ──────┘                                                          │
                                                                      ├─ founder signs v0.6.0-rc.1
                                                                      ├─ Windows tagged-artifact run
                                                                      ├─ fixes → v0.6.0-rc.2 … (repeat)
                                                                      └─ stable v0.6.0 (Windows waiver)
```

Branch conventions:

- Integration and release branch: `release/v0.6.0-rc.1` (created from
  `hop/main` at 2bc0367f with `main` at 8caa943a merged in).
- Executor branches: `v060/<workstream>-<slug>` from the release branch, each
  in its own worktree under `D:\Projects\reinstate-worktrees\v060-<slug>`.
  Never work in the shared checkout; never stash in it.
- The coordinator merges executor branches into the release branch after the
  gates in [review-gates.md](review-gates.md); executors never merge.
- `hop/main` is fast-forwarded to the release branch at the end and then
  retired.

---

## 6. Milestones

| M | Deliverable | Gate |
| - | ----------- | ---- |
| M0 | Integration branch exists; suite green after the merge | W0 report |
| M1 | Docs, changelog, fixes, website truth, lab harness merged | Gates 0–3 per branch |
| M2 | Ranges widened on Windows resume evidence; Hop parity journeys recorded PASS on Windows | Results docs committed |
| M3 | Release commit; snapshot gates; pre-tag Windows matrix PASS | Results doc + PR open |
| M4 | Founder signs `v0.6.0-rc.1`; draft published | Release workflow green |
| M5 | Windows tagged-artifact run PASS | Results doc committed to `main` |
| M6 | Stable `v0.6.0` promoted, published, packages and website deployed | `RELEASING.md` record |

M4 and the stable tag in M6 need the founder. Everything else runs
autonomously, and every founder step is written down as an exact command in
[clarifications.md](clarifications.md).

---

## 7. Risks

| Risk | Mitigation |
| ---- | ---------- |
| ConPTY on the Windows host broke on 2026-08-23 (#367); 11+ CLI rows and Grok GD8 need an interactive terminal | W4 probes ConPTY first thing; if it is still broken, W4 builds a Go ConPTY driver (`CreatePseudoConsole`) that can inject keystrokes, and the plan records the reboot as a founder item |
| Claude Code auto-updates past whatever ceiling we set | Record the installed version immediately before the release commit; the ceiling is what was verified, not what is installed |
| The 1,045-line `[Unreleased]` section resists consolidation and the changelog guard (`#364`) bites | W1 reshapes headings only; every bullet survives; the guard test is run, not reasoned about |
| Semantic collisions from the merge (OpenCode tier declared twice, doc-gate claims) | W0 runs the full suite and the conformance suite before anything else branches |
| Hop parity finds Windows-only defects late | W6 starts as soon as W4's harness lands, before the CLI matrix |
| Shipping `rein login` against a service that does not exist confuses users | D1: docs and the command both say the hosted service is not open yet; the URL is configurable for labs and self-hosters |
| macOS regressions land unnoticed for two weeks | Nothing claims macOS; the issue W8 opens lists every deferred row; the first macOS run re-certifies the same tag |
