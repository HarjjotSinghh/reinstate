# v0.6.0 — Reinstate Hop: cloud continuity, plus the CLI experience

Ship the hosted-tier client ("Hop") and the interactive CLI as one stable
release, certified on native Windows x64 now and on Apple Silicon macOS as soon
as that hardware is back.

**Status:** planned 2026-09-05; M0–M4 done — `v0.6.0-rc.1` was published
2026-09-06 and its tagged-artifact native Windows acceptance ended at
201/216 required rows `PASS` (every Hop and CLI-experience row passed; all 7
failures were in the Phase 5 generated matrix). `v0.6.0-rc.2` (2026-09-07)
fixed those 7 rows plus one fixture gap the same run found, plus one
addition beyond that scope (`search_text` now indexes message body for
Cline, Cursor, and OpenCode, closes #405); its own tagged-artifact native
Windows acceptance
([report](../../testing/results/2026-09-07-windows-v060rc2.md)) ended device
verdict `FAIL`: **203 PASS / 5 PARTIAL / 2 FAIL / 5 NOT TESTED** of **215**
required rows. All 22 CLI rows and 15 of 16 Hop rows passed; the seven rc.1
failures cleared. A post-commit finding (report §0.12) found the real
Cursor CLI `store.db` schema (`blobs`/`meta`) is not what the `v0.6.0-rc.2`
reader read (it guessed `messages`/`message`/`bubbles`, which no real store
has), re-scoring `cursor:C2`/`cursor:C3` `FAIL`; the other nine blocking
rows (`cline:C3`, `pi:C3`, `grok:E1`/`E2`/`E3`, `qwen:E1`/`E2`/`E3`,
`MatrixH:H7`) are host/harness/operator-availability gaps unrelated to
Cursor. `v0.6.0-rc.3` (2026-09-07) fixed exactly the Cursor CLI store reader
and nothing else; its own tagged-artifact native Windows acceptance
([report](../../testing/results/2026-09-07-windows-v060rc3.md)) ended device
verdict `FAIL`: **201 PASS / 3 PARTIAL / 1 FAIL / 10 NOT TESTED** of **215**
required rows. All 22 CLI rows and all 16 Hop rows passed, and the Cursor
fix was confirmed `PASS` on real data. It found a confirmed Pi reader defect
(`pi:C3` — `message.content` parts were never read for real `version:3`
sessions), a Qwen Code version-compatibility block (the host self-updated to
`0.23.0`, above the verified `0.21.13` ceiling, correctly refusing ten
rows), a headless-harness gap for three T3+ agents (`codex:E5`,
`opencode:E5`, `grok:E5`, no PTY available), and a lab incident: the `H7`
daemon started by Task Scheduler ran with the host's login environment and
pulled synthetic sessions into the live agent stores (cleaned up;
[#424](https://github.com/HarjjotSinghh/reinstate/issues/424)).
`v0.6.0-rc.4` (2026-09-07) changed exactly two things beyond rc.3 — the Pi
reader and the verified Qwen Code range (`0.21.12`–`0.23.0`, Windows
evidence); no other agent tier changed, no other compatibility range
widened. Its own tagged-artifact native Windows acceptance
([report](../../testing/results/2026-09-07-windows-v060rc4.md)) ended
device verdict `FAIL`: **208 PASS / 1 PARTIAL / 4 FAIL / 2 NOT TESTED** of
**215** required rows. Both of that candidate's own fixes were confirmed on
real data (every `pi`/`qwen` row `PASS`). It found two new agent-probe
findings (`MatrixB:B4` — real project-name path segments reaching a
committed artifact unshaped; `MatrixB:B7` — an existing-but-empty overridden
agent root indistinguishable from an absent one), a stale sync-completion
contract sentence scored against unchanged shipped behavior (`MatrixG:G4`),
a confirmed interactive-CLI defect (CLI row `13` — the switcher's
all-projects scope showed every visible row as unresumable regardless of
true state), an OpenCode version-compatibility block (host self-updated to
`1.18.29`, above the verified `1.18.27` ceiling), and an
operator/elevation-availability gap on `MatrixH:H7` (the run's shell held no
administrator rights, and that run's misreading of "fresh lab account" as a
Windows account, rather than a Hop account, meant the row's own UAC prompt
was never reached).
`v0.6.0-rc.5` (2026-09-07) changed exactly these things beyond rc.4 —
agent-probe shape normalization at every tree depth with root state
reported (closes `B4`/`B7`), the switcher's all-projects readiness (closes
row `13`), the sync-completion contract text (closes `G4`), and the
verified OpenCode range (`1.18.21`–`1.18.29`, Windows evidence, closes
`opencode:E5`/`E6`); no other agent tier changed, no other compatibility
range widened. Its own tagged-artifact native Windows acceptance
([report](../../testing/results/2026-09-07-windows-v060rc5.md), plus two
same-artifact rechecks, §23–§24) ended device verdict `FAIL`: **211 PASS /
4 PARTIAL / 0 FAIL / 0 NOT TESTED** of **215** required rows — **zero
product defects.** All six required-row gaps that candidate's own fixes
targeted cleared to `PASS`. Four required rows stayed not-`PASS` across the
run and both rechecks: `grok:E1`/`E2`/`E3` (live backend connectivity to
xAI, compounded by the host's live `grok` self-updating from `1.0.5` to
`1.0.13` mid-cycle) and `MatrixH:H7` (the daemon round-trip mechanism
completed on both rechecks for the first time this release, but `#424`
meant it still read live agent-home roots, and the rule's own
digest-equality pass condition proved unmeasurable on this live,
multi-session host).
`v0.6.0-rc.6` (2026-09-08) changed exactly these things beyond rc.5 — the
verified Grok Build range widened to `1.0.13` on the maintainer's own
console evidence (completed-turn `grok:E1`–`E3` evidence executed by the
maintainer at their own console in that candidate's own tagged run), and
`MatrixH:H7`'s live-home check refined from aggregate digest equality to a
per-file before/after listing with per-entry attribution; `#424` itself
remains scheduled for `v0.6.1`, not that candidate. No other agent tier
changed, no other compatibility range widened. Its own tagged-artifact
native Windows acceptance
([report](../../testing/results/2026-09-08-windows-v060rc6.md)) ended
device verdict `FAIL`: **211 PASS / 1 PARTIAL / 0 FAIL / 3 NOT TESTED** of
**215** required rows — **zero product defects.** Both required-row gaps
that candidate's own fixes targeted cleared to `PASS`: `grok:E1`–`E3` via
the maintainer's own console transcript, and `MatrixH:H7` via the refined
per-file-listing rule. Two new, genuinely blocking gaps surfaced, neither
present at `v0.6.0-rc.5`: `MatrixG:G1`'s Codex half (`PARTIAL`,
host/account — the acceptance host's live, authenticated Codex account was
usage-limit exhausted) and `codex:E1`–`E3` (`NOT TESTED (version drift)` —
the host's Codex CLI self-updated to `0.153.4`, past the in-tree `0.149.0`
ceiling).
`v0.6.0-rc.7` (2026-09-08) changed exactly one thing beyond rc.6 — the
verified Codex CLI range widened to `0.153.4` on native Windows evidence,
under the maintainer's standing self-update policy. No other agent tier
changed, no other compatibility range widened. Its own tagged-artifact
native Windows acceptance
([report](../../testing/results/2026-09-08-windows-v060rc7.md)) ended
device verdict `FAIL`: **214 PASS / 0 PARTIAL / 1 FAIL / 0 NOT TESTED** of
**215** required rows. This candidate's own targeted fix was fully
confirmed on real, completed conversational turns, and every row carried
from rc.6 cleared to `PASS`. It found one new, genuinely blocking
regression: CLI row `13` (the all-projects switcher rendered a `Ready` or
`Warn` session as `Blocked` in 11 of 15 independent launches, a defect
class the `v0.6.0-rc.5` fix had only partly closed).
`v0.6.0-rc.8` (2026-09-09) is the current candidate: it changes exactly one
thing beyond rc.7 — the readiness path (`internal/tui/readiness`,
`internal/preflight`, `internal/agentcheck`), fixing exactly that
regression and, in a follow-up, a related gap where a permanently,
deterministically broken agent install also rendered "still checking"
forever instead of settling on `Blocked` with its actual repair message.
No agent tier changes, no compatibility range widens. Under the disposition
rules in
[`docs/testing/v0.6.0-windows-acceptance.md`](../../testing/v0.6.0-windows-acceptance.md#dispositions-that-do-not-block-the-device-verdict),
stable requires **215 of 216** rows `PASS` (`opencode:D4` is
`N/A (definitional)`, excluded from the required count), plus whatever
dispositions are still open on the acceptance host when that run happens.
What remains: merging and tagging `v0.6.0-rc.8`, a tagged-artifact Windows
run against it (with CLI row `13` re-tested under the expanded,
at-least-15-launch method plus a `-stale-claude` settle check), and the
Apple Silicon macOS rows, still deferred until that hardware returns
(#403).
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
| M0 ✅ | Integration branch exists; suite green after the merge | W0 report |
| M1 ✅ | Docs, changelog, fixes, website truth, lab harness merged | Gates 0–3 per branch |
| M2 ✅ | Ranges widened on Windows resume evidence; Hop parity journeys recorded on Windows (14 PASS, H5/H7 PARTIAL) | Results docs committed |
| M3 ✅ | Release commit; snapshot gates; pre-tag Windows matrix recorded (185/200; dispositions in `RELEASING.md`) | Results doc + PR #404 ready |
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
