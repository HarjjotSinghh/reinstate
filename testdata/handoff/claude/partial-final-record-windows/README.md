# Claude partial-final-record (native Windows)

Same shape as `../partial-final-record/` — two complete JSONL records plus a
third record truncated mid-line, no closing brace, no trailing newline — but
`cwd` and the `projects/` directory name (`C--Users-fixture-user-code-demo`)
are Windows-shaped.

Pins the truncation boundary offset and the SHA-256 of the bytes before it,
independently recomputed from the fixture's own bytes, at two layers:

- `internal/transcript`'s `TestClaudePartialFinalRecordExcluded/windows`
  exercises `ClaudeReader.Snapshot`/`Parse` directly against this file.
- `internal/handoff`'s `TestPlanPartialFinalRecordByteExactCrossCheck`
  (`partial_final_record_route_test.go`) drives the real `handoff.Plan()`
  pipeline — the function `internal/cli/handoff.go` calls for
  `rein handoff --no-launch --json` — against this fixture from a throwaway
  git repository named `demo`, and cross-checks the resulting capsule's
  `raw_source.byte_offset`/`raw_source.artifact_sha256`.

**Correction (2026-09-07):** the version of this README committed with this
fixture claimed its Windows-shaped `cwd` is what lets the
`handoff --no-launch --json` byte-exact cross-check reach these numbers on
native Windows, where the sibling macOS-shaped fixture supposedly could not.
That is false, and was disproved by direct reproduction: on this native
Windows host, `rein handoff` against the *macOS*-shaped
`../partial-final-record/` fixture already succeeds and produces the
identical cross-check, provided the operator's working directory is a git
repository named `demo`; both fixtures refuse identically
(`working directory is a different repository than the source session`) from
any other directory name, on every OS.

`internal/handoff/workspace.go`'s `shouldRemapWorkspace` triggers on the
literal substring `/fixture-user/` (or `/synthetic-user/`) in the recorded
workspace path, checked identically regardless of `runtime.GOOS` —
`isForeignOSPath` never enters into the decision for any fixture built to
this repository's own fixture-user convention (every fixture under
`testdata/`). What actually gates the remap, and the refusal, is
`sameProjectLeaf`: whether the operator's local git-root directory shares the
recorded workspace's last path segment. See
`TestPlanPartialFinalRecordRefusalIsOSShapeIndependent` for this asserted
directly, for both this fixture and the macOS one, on this host.

What this fixture genuinely adds, and the earlier claim did not need to
overstate to be worth committing: a Windows-shaped recorded workspace and
`projects/` directory-name encoding as input to the reader's own boundary
math, which had no committed Windows-shaped case for `claude` before this
fixture existed.
