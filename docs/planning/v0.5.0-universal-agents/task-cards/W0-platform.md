# W0 — Platform cards

**One executor, tasks in order, no parallelism inside this workstream.**
Everything else in Phase 5 blocks on these six tasks, so correctness matters
more than speed here.

Specification: [../../../adapters/agent-catalog-sdk.md](../../../adapters/agent-catalog-sdk.md).
Tier vocabulary: [../../../agent-support-tiers.md](../../../agent-support-tiers.md).

---

## T-001 — Catalog core

**Depends on:** nothing · **Owns:** `internal/agents/*.go`,
`internal/agents/scan/hometree/`, `internal/agents/scan/cliquery/`

**Goal.** Create the package that holds one descriptor per agent, with a
registry, tier and family types, and the two shared scanners the existing
agents already need.

**Steps.**

1. Define `Descriptor`, `Tier`, `StorageFamily`, `T0Reason`, `StorageSpec`,
   `NativeSpec`, `VersionSpec`, `ProcessSpec`, `Evidence`, and `Env` exactly as
   specified in the SDK document.
2. Implement `MustRegister`, `All`, `Get`, `Keys`, `AtLeast`, `Capable`.
   `All` returns a deterministic order sorted by key.
3. `MustRegister` panics on an empty key, a duplicate key, a `T0Reason` present
   at a tier above T0 or absent at T0, or a capability constructor above the
   declared tier.
4. Extract the shared `hometree` scanner from the common shape in the existing
   Claude, Codex, Gemini, and Grok index sources: root resolution with an
   environment override, marker check, bounded glob walk, JSONL reading to the
   last complete line, and mod-time plus size change detection.
5. Extract `cliquery` from the OpenCode source: bounded subprocess with a
   timeout and an output ceiling, JSON decode, non-zero-exit handling.
6. Reuse the existing ceilings — `MaxJSONLineBytes`, `MaxSearchTextBytes`,
   `MaxFileReferences`. Do not introduce new ones.

**Done when.** The package compiles, is fully unit tested, and no other package
imports it yet. Nothing outside `internal/agents` changes in this PR.

**Escalate if.** The existing five sources do not in fact share the shape the
scanners assume. That is a real finding and changes T-002's estimate.

---

## T-002 — Migrate the five shipped agents

**Depends on:** T-001 · **Owns:** `internal/agents/catalog/{claude,codex,gemini,opencode,grok}.go`,
`internal/agents/sources/`

**Goal.** Express the five existing agents as descriptors, with their current
tiers, without changing any behavior.

**Steps.**

1. Write one descriptor per agent, in its own file. Tiers: claude T5, codex T5,
   gemini T2, opencode T2, grok T2.
2. Move the version ranges from the adapter packages into the descriptors:
   Claude Code `2.1.219`–`2.1.229`, Codex CLI `0.133.0`–`0.147.0`. Keep the
   existing `SupportedVersion` functions working, reading from the catalog.
3. Port each index source to construct through the shared scanner, keeping its
   record mapping intact. The Grok workspace-key encoding, the Gemini
   `$rewindTo` handling, the Codex filename-wins identity rule, and the Claude
   subagent exclusion all stay exactly as they are.
4. Populate `Evidence` with the existing fixture paths and, for claude and
   codex, the existing Phase 3 and Phase 4 device reports.
5. Add the descriptors' `Excluded` sets from the existing `Exclusions()`
   implementations.

**Done when.** Every existing test passes **with no assertion edited**, and
`go test ./... -count=1` is green.

**Escalate if.** Any test needs an assertion change. That means behavior moved,
which this task forbids. Stop and report what moved.

---

## T-003 — Convert registration sites to derived consumers

**Depends on:** T-002 · **Owns:** the agent-list portions of
`internal/cli/commands_impl.go`, `internal/cli/sessions.go`,
`internal/cli/handoff.go`, `internal/sessionindex/model.go`,
`internal/sessionindex/launch.go`, `internal/agentcheck/agent.go`,
`internal/processcheck/process.go`, `internal/preflight/verify.go`,
`internal/schema/config.go`

**Goal.** Delete every hard-coded agent list and read the catalog instead.

**Steps.**

1. `defaultRegistry()` iterates `Capable(CapabilitySync)`.
2. `defaultLocalSources()` iterates `Capable(CapabilityIndex)`.
3. `agentcheck.definitions` is built from descriptors with a `VersionSpec`.
4. Transcript readers and handoff targets register from the catalog rather than
   from per-file `init()`.
5. `validateLocalAgent` accepts `AtLeast(TierDiscover)` keys plus `all`;
   `validateNativeAgent` accepts `AtLeast(TierResume)` keys plus `all`.
6. `processcheck` builds its matchers from `ProcessSpec`, preferring vendor
   self-identification environment variables where a descriptor declares them.
7. Command help text listing agent choices is generated, not written.
8. `internal/schema/config.go` default enablement comes from the catalog.
9. Keep the exported agent constants as thin aliases so external references do
   not break.

**Done when.** No production file outside `internal/agents` contains a literal
list of agent names, the full suite passes with no assertion edited, and Gate 6
in [../review-gates.md](../review-gates.md) passes by hand.

**Escalate if.** A site needs agent-specific behavior that the descriptor
cannot express. Propose the descriptor field rather than leaving a switch.

---

## T-004 — Conformance suite

**Depends on:** T-003 · **Owns:** `internal/agents/conformance/`

**Goal.** Make the tier claim enforceable by a test instead of by review.

**Steps.**

1. Implement `conformance.Run(t, descriptor, fixtures)` asserting all nine
   checks in the SDK document: structure, capability agreement, evidence
   presence, determinism, isolation, corruption, privacy, fail-closed version,
   and read-only reason.
2. Implement isolation with a wrapped filesystem that records every open and
   fails the test on any write, rename, truncate, or lock, and on any open
   outside the fixture root. Do not implement it as a convention.
3. Corruption cases: truncated final record, invalid UTF-8, empty file, empty
   directory, absent root, unknown layout version.
4. Wire the five migrated agents into the suite and make them pass.
5. Add a negative test proving the suite fails a descriptor with a deliberately
   broken evidence path.

**Done when.** All five shipped agents pass, and the negative test fails the
suite as expected.

---

## T-005 — Probe engine

**Depends on:** T-003 · **Owns:** `internal/agents/probe/`

**Goal.** Produce the `AGENT-PROBE-V1` evidence artifact, with redaction
enforced by tests rather than by care.

**Steps.**

1. Implement the emitter per
   [../../../testing/agent-storage-probe.md](../../../testing/agent-storage-probe.md):
   candidate roots with existence and marker flags, resolved root, executable
   presence, raw version string, tree shape with counts and median sizes, name
   shapes, and first-line JSON **keys**.
2. Implement shape normalization: UUID, hex hash, path slug, and numeric
   sequence each collapse to a token.
3. Emit roots as `{relative_to, suffix}` pairs. Never emit an absolute path.
4. Enforce every redaction rule with a test that feeds a synthetic tree
   containing planted usernames, absolute paths, secrets, and repository names,
   then asserts none appear in the output.
5. Bound everything: sample counts, files opened, bytes read per file, total
   runtime. A probe against a huge tree must finish.
6. Never open anything in the descriptor's `Excluded` set.

**Done when.** The planted-secret test passes, output validates against the
schema, and a probe on a machine with no agents installed produces a valid,
empty-but-complete artifact.

---

## T-006 — `rein doctor --agents` and wrapper scripts

**Depends on:** T-005 · **Owns:** the agent surface of `internal/cli/doctor.go`,
`scripts/testing/agent-storage-probe.sh`, `scripts/testing/agent-storage-probe.ps1`

**Goal.** Expose the inventory and the probe as user-facing commands.

**Steps.**

1. `rein doctor --agents` prints a human-readable inventory: every catalog
   agent, its tier, whether it is installed, its session count where known, and
   for T0 agents the reason.
2. `rein doctor --agents --json` emits the probe artifact.
3. `rein doctor --agents --acceptance-matrix` emits the generated Phase 5 row
   count and per-agent row list, which the device reports cite.
4. Write both wrapper scripts. They must produce byte-identical output to the
   binary; add a test asserting that.
5. Update `docs/cli-reference.md` in the same PR, because `internal/doctest`
   binds them.

**Done when.** All three modes work on macOS and Windows, the wrapper parity
test passes, and the doctest suite is green.

**Escalate if.** The acceptance-matrix generator and the written contract in
[../../../testing/phase-5-universal-agent-coverage-acceptance.md](../../../testing/phase-5-universal-agent-coverage-acceptance.md)
disagree on row counts. One of them is wrong and the coordinator decides which.
