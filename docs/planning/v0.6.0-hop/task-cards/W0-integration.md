# W0 — Integration

**Executor:** one Sonnet agent, serialized. **Verifier:** the coordinator.
**Branch:** creates `release/v0.6.0-rc.1`. **Blocks:** everything.

## T-000 — Create the release branch and merge `main`

**Inputs.** `hop/main` at 2bc0367f; `main` at 8caa943a. `git merge-tree`
reports one textual conflict, `CHANGELOG.md`.

**Do.**

1. `git checkout -b release/v0.6.0-rc.1` from `hop/main`.
2. `git merge --no-ff --no-commit main`.
3. Resolve `CHANGELOG.md`: keep the Hop tree's whole `[Unreleased]` section;
   insert `main`'s `## [0.5.2-rc.1] - 2026-08-23` section verbatim beneath it
   and above `## [0.5.1]`; everything from `[0.5.1]` down is byte-identical to
   `main`; the link-reference block carries the `0.5.2-rc.1` compare link.
4. Commit the merge. Then `go build ./...`, `go vet ./...`, `gofmt -l .`,
   `go mod tidy -diff`, and the full suite.
5. Semantic collisions (OpenCode declared T3/T4 on `main` and T5 on the Hop
   tree; doc-gate claims in `internal/doctest`; conformance in
   `internal/agents`) are fixed in a separate `fix(merge): …` commit that
   preserves both sides' intent: OpenCode ends at T5 and stays a verified
   resume agent and a handoff destination; Grok Build T4; Qwen T4; Kimi T2.

**Done when.** The report names the branch tip, the merge sha, every
follow-up commit, the changelog section order, and `ok=N FAIL=<list>` for the
full suite, with the preflight flake (if it is the only failure) named as
such.

**Never.** Push. Edit an existing assertion. Touch
`assets/logo/concepts/14-diptych-bridge.svg` (pre-existing local edit).
