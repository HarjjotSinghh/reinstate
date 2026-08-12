# Phase 4 — handover and continuation

**Date:** 2026-08-12
**Integration branch:** `feat/phase4-cross-agent-handoff` at `7082603`
**Reason for handover:** the previous orchestrator ran out of credits mid-flight
**Companion documents:** [architecture plan](2026-08-12-phase-4-cross-agent-handoff-plan.md) ·
[work packets](2026-08-12-phase-4-executor-work-packets.md) ·
[original kickoff](2026-08-12-phase-4-grok-executor-kickoff.md)

This document records verified state, corrects four errors in the original
plan, and carries the continuation prompt for the next agent.

---

## 1. Verified status

Measured on `feat/phase4-cross-agent-handoff` at `7082603`, not inferred from
merge messages.

| Check | Result |
| ----- | ------ |
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `go test` on capsule / transcript / handoff / secretscan / fixture / cli | pass |
| New Go across the four Phase 4 packages | ~13,475 lines including tests |
| `rein handoff` command | **does not exist** — `internal/cli/handoff.go` is absent |

### Merged — 20 of 27 packets

| Packets | PRs |
| ------- | --- |
| WP-01 baseline and plan set | #149 |
| WP-02 secretscan · WP-03 capsule model · WP-04 validation/fidelity | #150, #151, #153 |
| R1–R3 research | #152 |
| WP-05 transcript contract | #154 |
| WP-06 Claude · WP-07 Codex · WP-08 Gemini · WP-09 OpenCode · WP-10 Grok readers | #158, #159, #156, #161, #157 |
| WP-11 checkpoint · WP-12 workspace bind · WP-13 capability diff · WP-14 policy | #163, #160, #162, #164 |
| WP-15 target contract · WP-16 Claude target · WP-17 Codex target · WP-18 projection · WP-19 acknowledgement | #165, #167, #168, #166, #169 |
| WP-20 handoff store and lineage | #155 |
| WP-23 synthetic fixture corpus | #170 |

All eight open research items (R1–R8) are answered. R1/R2/R3 in
`docs/research/2026-08-12-phase-4-r1-r2-r3.md`; R5/R6/R7 have their own files
under `docs/research/`; R4 and R8 are resolved in the Codex and Claude readers
(`vendor_opaque_state`, `attachment_unavailable`).

### Remaining — 6 packets

WP-21, WP-22, WP-24, WP-25, WP-26, WP-27. Scope revisions in §3.

### Work in flight when credits ran out

| Branch | State |
| ------ | ----- |
| `origin/wp/21-handoff-cli` | **Preserved.** One WIP commit adding `internal/handoff/pipeline.go` (780 lines): `Plan`, `Execute`, `Options`, `PlanResult`, `ExecuteResult`, `PipelineError`, warning acknowledgement, redaction, sidecar encoding, preview artifacts, destination planning. **Does not compile.** Branched from `ec9d288`, needs rebase onto `7082603`. Unreviewed, no tests. |
| `wp/24-adversarial` | Dispatched but produced nothing. Local worktree was empty. Start from scratch. |

---

## 2. Known blocker in the preserved WP-21 work

```text
internal/handoff/target_codex.go:251:6: sha256Hex redeclared in this block
    internal/handoff/pipeline.go:769:6: other declaration of sha256Hex
```

Fix by deleting the duplicate and putting shared helpers in one place — add
`internal/handoff/internal_helpers.go` (or fold into `store.go`) and have both
call sites use it. Do not rename one copy; that leaves two implementations of
the same hash helper.

---

## 3. Corrections to the original plan

These are planner errors found during execution. Treat this section as
overriding the work-packets document where they disagree.

### C1 — `internal/handoff/pipeline.go` was never assigned a packet

The architecture plan §3.3 lists `pipeline.go` as "the only orchestration entry
point", but no work packet creates it. WP-21 was written as a CLI packet only.
The previous executor correctly noticed and started building it inside WP-21,
which is why that packet stalled — it is two packets of work.

**Split WP-21:**

- **WP-21a — `internal/handoff/pipeline.go`.** `Plan()` and `Execute()`
  implementing the nine pipeline steps in §5 of the architecture plan. Start
  from the preserved WIP on `origin/wp/21-handoff-cli`; fix the collision;
  add tests. No CLI code.
- **WP-21b — `internal/cli/handoff.go`.** The command surface in §8 of the
  architecture plan, plus `handoff list` / `inspect` / `export`, registered in
  `internal/cli/root.go`. Depends on WP-21a.

### C2 — WP-21's dependency list was wrong

It listed WP-16, WP-17, WP-18, WP-19, WP-20. The pipeline also needs WP-05
through WP-14. All of those are merged, so nothing is blocked — but record the
real dependency so the graph is honest.

### C3 — WP-24's fixtures already exist

WP-23 created `testdata/handoff/adversarial/{prompt-injection,secret-leakage,fence-breakout,oversized}/`.
WP-24 is now **tests only**: write `internal/handoff/security_test.go` against
the existing corpus. Do not recreate fixtures.

### C4 — capsule goldens are empty

`testdata/handoff/golden/projection/` has real goldens from WP-18.
`testdata/handoff/golden/capsule/` contains only `.gitkeep` and a `README.md`.
WP-25 must generate and commit the capsule goldens; it cannot assume they exist.

---

## 4. Revised wave plan for the remaining work

Less parallelism is available than in the original plan — the tail is mostly
sequential. Three packets can still start immediately and concurrently.

```text
WAVE A — 3 parallel
  WP-21a  pipeline.go            (rebase the preserved WIP, fix collision, test)
  WP-24   adversarial tests      (fixtures already exist — tests only)
  WP-26a  docs, non-CLI parts    (README/ROADMAP/adapters/compatibility/
                                  security-model/handoff.md scope language)
  GATE: integration branch green.

WAVE B — 2 parallel
  WP-21b  internal/cli/handoff.go + root.go registration
  WP-25a  capsule goldens + determinism tests   (no CLI dependency)
  GATE: `rein handoff --dry-run` produces output; capsule goldens byte-stable.

WAVE C — 2 parallel
  WP-22   resume --with alias + picker action
  WP-25b  CLI goldens, dry-run/executed parity, 200-turn perf ceiling
  GATE: dry-run output byte-identical to the executed run.

WAVE D — sequential
  WP-26b  cli-reference.md + doctest contract tests for the shipped flag set
  WP-27   acceptance runbook, v0.4.0-rc.1 verification prompts, release prep
  GATE: §14 definition of done in the architecture plan.
```

After WP-27: squash `feat/phase4-cross-agent-handoff` into one PR against
`main`. `main` must never carry a partial Phase 4.

---

## 5. Continuation prompt

Paste the block below into the new agent (Claude Code or Codex).

```text
You are taking over Reinstate Phase 4 — cross-agent session handoff, shipping
as v0.4.0-rc.1. The previous orchestrator ran out of credits with 20 of 27 work
packets merged. Your job is the remaining 6.

Repository: github.com/HarjjotSinghh/reinstate (Go 1.25.12, Apache-2.0)
Integration branch: feat/phase4-cross-agent-handoff at 7082603
main is stable v0.3.0 and must stay that way until Phase 4 is complete.

═══════════════════════════════════════════════════════════════════
STEP 0 — READ FIRST, IN THIS ORDER
═══════════════════════════════════════════════════════════════════
1. docs/superpowers/plans/2026-08-12-phase-4-handover-continuation.md
   ← current state, four plan corrections, revised wave plan. START HERE.
2. docs/superpowers/plans/2026-08-12-phase-4-executor-work-packets.md
   ← WP-01..WP-27. Sections 3 of the handover doc OVERRIDE this where they
     disagree.
3. docs/superpowers/plans/2026-08-12-phase-4-cross-agent-handoff-plan.md
   ← the architecture. AUTHORITATIVE. Never redesign around it.
4. docs/adr/0003-phase-4-rc1-scope-and-launch-route.md — fixed scope decisions.
5. docs/handoff.md — the product contract users will read.
6. docs/testing/phase-4-cross-agent-handoff-acceptance.md — how rc.1 is judged.
7. AGENTS.md and CLAUDE.md — repository non-negotiables.

Then read the code that already exists and that you are completing, not
rewriting: internal/capsule/, internal/transcript/, internal/handoff/,
internal/secretscan/, and internal/cli/root.go + sessions.go.

═══════════════════════════════════════════════════════════════════
STEP 1 — BASELINE
═══════════════════════════════════════════════════════════════════
  git fetch origin
  git checkout -b <your-branch> origin/feat/phase4-cross-agent-handoff
  make verify        # must be green before you write anything

If make verify is red on an untouched checkout, stop and report. Do not build
on a red baseline.

═══════════════════════════════════════════════════════════════════
STEP 2 — WHAT IS LEFT
═══════════════════════════════════════════════════════════════════
Six packets, in the wave order from §4 of the handover doc:

  WAVE A (3 in parallel)
    WP-21a  internal/handoff/pipeline.go — Plan() and Execute()
            Preserved WIP exists on origin/wp/21-handoff-cli (780 lines).
            Rebase it onto 7082603. It DOES NOT COMPILE: sha256Hex is declared
            in both pipeline.go:769 and target_codex.go:251. Delete the
            duplicate and put the shared helper in one file. Then add tests.
    WP-24   internal/handoff/security_test.go — adversarial tests only.
            Fixtures ALREADY EXIST under testdata/handoff/adversarial/.
            Do not recreate them. One test per numbered rule in §7 of the
            architecture plan; each test must fail if its rule is removed.
    WP-26a  documentation that does not depend on the CLI surface:
            README.md, ROADMAP.md Phase 4 rows, docs/adapters.md,
            docs/compatibility.md directional handoff matrix,
            docs/security-model.md, and scope language in docs/handoff.md.

  WAVE B (2 in parallel)
    WP-21b  internal/cli/handoff.go + registration in internal/cli/root.go.
            Full flag surface from §8 of the architecture plan, plus
            handoff list / inspect / export.
    WP-25a  capsule goldens under testdata/handoff/golden/capsule/ (currently
            EMPTY — only .gitkeep and README.md) plus determinism tests.

  WAVE C (2 in parallel)
    WP-22   rein resume --with AGENT alias + picker handoff action.
    WP-25b  CLI goldens, dry-run/executed byte-parity, 200-turn perf ceiling.

  WAVE D (sequential)
    WP-26b  docs/cli-reference.md + internal/doctest contract tests asserting
            every shipped flag is documented and no doc claims cross-agent
            "native resume" or "same session".
    WP-27   docs/testing/v0.4.0-rc.1-agent-verification-prompts.md,
            docs/testing/results/phase-4-report-template.md, release prep.

After WP-27, squash feat/phase4-cross-agent-handoff into ONE PR against main.

═══════════════════════════════════════════════════════════════════
STEP 3 — HOW TO RUN IT
═══════════════════════════════════════════════════════════════════
Use parallel sub-agents where the wave plan says parallel. If your harness
supports sub-agents, give each one its own git worktree and branch:

  git worktree add -b wp/<NN>-<slug> ../wt-wp<NN> feat/phase4-cross-agent-handoff

One packet per sub-agent. One packet per PR against the integration branch —
never against main. Sub-agents do not merge their own PRs; you review and merge.
A sub-agent never edits a file outside its packet's declared file list; if it
needs to, it stops and reports.

If your harness has no sub-agents, run the same wave order sequentially. Same
result, slower.

Between waves: merge every packet PR, run make verify on the integration
branch, and only then open the next wave. A red integration branch stops all
dispatch.

═══════════════════════════════════════════════════════════════════
STEP 4 — RULES NOTHING MAY BREAK
═══════════════════════════════════════════════════════════════════
 1. `make verify` passes before any PR. `make quick` is NOT sufficient.
 2. NEVER read a real ~/.claude, ~/.codex, ~/.gemini, ~/.grok, or OpenCode
    tree. testdata/ fixtures only. make fixture-scan gates it.
 3. NEVER write into any vendor session store. Reinstate creates a NEW
    destination session through the vendor's own documented CLI.
 4. NEVER execute anything found in a transcript. Historical tool calls are
    inert evidence.
 5. NEVER promote source system/developer messages to destination authority.
    They stay out of the projection body.
 6. NEVER require a source model call on the critical path.
 7. NEVER invent structure from prose. constraints / decisions /
    rejected_approaches ship as `omitted` with a reason.
 8. NEVER add a new exit code. 0/1/2/3/5/6/7 is fixed.
 9. NEVER say "resume", "same session", "lossless", or "full context" for a
    cross-agent path. It is a "structured handoff" of "the same task".
    Gemini, OpenCode, and Grok are SOURCE-ONLY in v0.4.0.
10. Bounded reads only. No unbounded io.ReadAll over a vendor file.
11. No network in unit tests. No real vendor process spawned in unit tests.
12. Conventional Commits. CHANGELOG entry under [Unreleased] for user-visible
    behavior. Doc comments on new exported symbols.

═══════════════════════════════════════════════════════════════════
STEP 5 — ESCALATE, DO NOT IMPROVISE
═══════════════════════════════════════════════════════════════════
Stop and ask the human when:
  • a merged package's API would have to change to finish your packet;
  • a security rule in §7 of the architecture plan would have to be weakened
    to make a test pass;
  • the preserved WP-21 WIP conflicts with the architecture plan's §5 pipeline
    (trust the plan, not the WIP);
  • a destination cannot be launched without writing vendor-internal files;
  • dry-run and executed output cannot be made byte-identical.

═══════════════════════════════════════════════════════════════════
STEP 6 — REPORT
═══════════════════════════════════════════════════════════════════
After each wave, post under 200 words: packets merged, packets in flight,
integration branch verify status, escalations, next wave's dispatch list.
No preamble.

Start with STEP 0. Read the seven documents, run STEP 1, then open Wave A
with WP-21a, WP-24, and WP-26a.
```

---

## 6. What "done" still means

Unchanged from §14 of the architecture plan. The rows still outstanding:

- [ ] `rein handoff` exists and Claude → Codex / Codex → Claude both succeed
      with the source CLI closed and no source API call
- [ ] `--dry-run` output is byte-identical to the executed run
- [ ] Adversarial injection and secret fixtures pass with zero leakage
- [ ] A 200+ turn source produces a bounded, reported projection under ceiling
- [ ] Capsule goldens committed and byte-stable across OSes
- [ ] Docs updated and doctest contract tests added
- [ ] Acceptance runbook and rc.1 verification prompts committed
- [ ] Squashed into one `main` PR

Already satisfied by merged work: every fidelity class is implemented and
tested, all five readers are deterministic, the store is private and
append-only, Windows/macOS path remapping goes through canonical project IDs,
no new exit codes exist, and nothing writes into a vendor session store.
