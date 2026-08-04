# Phase 3 verified-resume acceptance

This is the release-neutral acceptance contract for Reinstate Phase 3. It
extends, and does not replace, the Phase 1 encrypted-sync and Phase 2 local
continuity matrices.

The authoritative product behavior is defined in
[Verified resume](../verified-resume.md). A candidate passes only when the
installed tagged artifacts produce truthful, privacy-safe environment reports
and gate every native continuation consistently.

## Evidence policy

- Test one immutable full commit and signed tag on every device.
- Build development evidence from a clean worktree. Test release evidence from
  verified installed artifacts, not an untagged local rebuild.
- Use a fresh isolated Reinstate home and fresh disposable Git repositories.
- Create fresh controlled Claude Code and Codex sessions. Do not reuse Phase 2
  or an older candidate's corpus.
- Never commit a transcript, full prompt, response, secret, credential, MCP
  configuration value, instruction content, skill content, private absolute
  path, or raw environment dump.
- Preserve original failures. A targeted recheck may supersede a row only on
  the identical product commit/artifact and with new complete evidence.
- A missing required physical result is `FAIL`, not `PASS`. An optional vendor
  path may be `NOT TESTED` only when that vendor is genuinely absent and the
  runbook forbids installing it solely for acceptance.
- No test may fetch, checkout, reset, install, repair, or mutate project/vendor
  configuration on the user's behalf.

## Required environments for RC1

Development verification runs on the implementation host and CI. Tagged
release-candidate certification requires:

1. Apple Silicon macOS (`darwin/arm64`); and
2. native Windows x64 (`windows/amd64`, never WSL for the Windows column).

Intel macOS and Linux/WSL2 artifacts remain separately tracked platform
evidence. RC1 may be published for testing without claiming stable support on
those platforms. Stable `v0.3.0` still follows `RELEASING.md`; the v0.2.0
limited-platform waiver does not silently carry forward.

## Automated development gate

Run from the exact clean product commit:

```sh
make verify
CGO_ENABLED=1 GOTOOLCHAIN=go1.25.12 go test -race ./... -count=1
go test ./internal/workspace ./internal/environment ./internal/preflight -count=1
```

Run the repository's pinned lint and vulnerability tools through `make verify`,
then build with `CGO_ENABLED=0` for:

- `darwin/arm64`;
- `darwin/amd64`;
- `windows/amd64`; and
- `linux/amd64`.

The suite must include fuzz smoke for the porcelain parser, remote normalizer,
configuration-name parsers, policy evaluator, and safe renderer. No unit test
may depend on the network.

Before tagging, also run a clean GoReleaser snapshot, inspect every archive and
SBOM, and exercise the installer smoke suite against the snapshot artifacts.

## Mandatory matrix

Rows 1–32 must pass on both required devices unless the row explicitly assigns
an optional vendor path.

| # | Gate |
| - | ---- |
| 1 | Exact tag, full 40-character commit, installed-binary hash, signature, checksum, attestation, archive, SBOM, and installer provenance |
| 2 | Full local verification, complete race suite, required cross-builds, and Phase 1/2 regression |
| 3 | Fresh configless home; no init, credential, passphrase, keyring write, or backend dependency |
| 4 | Fresh controlled Claude and Codex sessions in disposable Git repositories |
| 5 | First inspect reports `baseline.unavailable`, never a manufactured match |
| 6 | Successful verified native launch records a `reinstate_prelaunch_observed` baseline without modifying vendor configuration |
| 7 | Repeat unchanged inspect/preflight matches repository, branch, HEAD, and working-tree digest |
| 8 | Replacement by a different repository at the same path blocks with safety exit `7` and launches no vendor |
| 9 | Missing/non-directory workspace blocks with compatibility exit `5` and launches no vendor |
| 10 | Branch change, detached HEAD, and unborn repository states are reported distinctly |
| 11 | Equal, ahead, behind, diverged, and locally unavailable HEAD relations are truthful and offline |
| 12 | Staged, unstaged, untracked, conflicted, and submodule changes warn without exposing filenames or diffs |
| 13 | Credential-bearing and alternate SSH/HTTPS remotes normalize without leaking raw URLs or secrets |
| 14 | Linked worktree, symlink, Unicode, case, and native path behavior is safe and deterministic |
| 15 | Claude executable, version, and layout compatibility is reported and fail-closed |
| 16 | Codex executable, version, and layout compatibility is reported and fail-closed |
| 17 | Instruction-file presence/missing/change is name-only, bounded, and never content-bearing |
| 18 | Skill presence/missing/change is name-only, bounded, does not follow escaping links, and executes nothing |
| 19 | MCP presence/missing/change is logical-name/transport-only and exposes no command, argument, URL credential, header, or environment value |
| 20 | Recognized Node/Go runtime declarations and installed versions match/warn/unknown correctly without running project scripts |
| 21 | `inspect` human and JSON output agree, remain deterministic, include decision/provenance, and do not prompt or launch |
| 22 | Native dry-run preserves launch-plan keys, adds the environment report, never prompts/launches/mutates, and uses stable exits |
| 23 | TTY warning prompt defaults to no; no/EOF/Ctrl-C exits `7` and launches nothing; yes launches once |
| 24 | Non-TTY warning refuses until every fresh warning has an exact `--allow-environment-warning CHECK_ID` |
| 25 | Unknown, stale, duplicate, wildcard, informational, and blocker warning IDs cannot bypass policy |
| 26 | Hard blockers never prompt; compatibility, safety, and runtime exit precedence is deterministic |
| 27 | Real same-vendor Claude resume and fork pass; fork source is preserved and the distinct fork independently resumes |
| 28 | Real same-vendor Codex resume and fork pass; fork source is preserved and the distinct fork independently resumes |
| 29 | Real picker inspect/resume/fork/invalid/cancel/interrupt paths apply identical preflight policy through both aliases |
| 30 | Gemini/OpenCode remain read-only with exit `5`; their physical discovery paths are optional only when genuinely absent |
| 31 | Malformed, oversized, cyclic, hostile-control, timeout, cancellation, stale-index, path-race, concurrency, and privacy gates pass |
| 32 | Cold/warm small/large-corpus latency evidence stays under the documented device ceilings with no 20–30 second regression |

## Freshness and stale-index proof

Every launch refreshes the selected source and records whether that refresh was
fresh. A failed source scan may leave a stale row available for diagnostic
inspection, but it must block launch with safety exit `7`. The report must not
claim that stale source metadata is current.

Critical workspace, repository, and agent checks run once against the final
selected record immediately before execution. The exact immutable report used
for policy is the report displayed to the user and passed to the launch
decision. Tests must demonstrate that a path-type/repository swap cannot occur
between authorization and the child launch without refusal.

## Privacy/adversarial corpus

Synthetic fixtures must place distinct controlled sentinels in:

- remote user information, password, query, and fragment;
- MCP URL, headers, command arguments, and environment values;
- instruction and skill contents;
- dirty filenames, rename destinations, ANSI/bidirectional control strings,
  and malformed UTF-8; and
- raw child stderr and oversized configuration input.

Assert that no sentinel appears in stdout, stderr, JSON, the private index,
report commits, or committed fixtures. Store only normalized/digested identity,
bounded logical names, state bits/counts, and fixed diagnostics.

## Performance evidence

Measure the installed artifact with three cold and twenty warm samples for a
normal corpus and a synthetic large corpus. Record hardware, OS, filesystem,
agent versions, corpus size, median, p95, and maximum. Antivirus state may be
described generically on Windows; do not weaken it solely for testing.

Warm inspect/dry-run p95 targets are one second on Apple Silicon and two seconds
on native Windows. Release-blocking ceilings are two and four seconds. Cold
full-refresh and large-corpus ceilings must be declared in the candidate
dispatch before testing. Any timeout, unbounded command/file count, 20–30
second regression, or more than 25 percent same-host p95 regression is a
blocker.

CI gates deterministic command/file-count and complexity ratios rather than
flaky wall-clock assertions. Absolute ceilings belong to the physical matrix.

## Tagged-candidate chain

For `v0.3.0-rc.1`:

1. merge the reviewed implementation through protected `main`;
2. create an annotated signed tag at the exact merged commit;
3. wait for release CI to build the draft and attest every asset;
4. independently verify all checksums, API digests, attestations, SBOMs, and
   safe archive membership before executing anything;
5. publish the prerelease only after draft-artifact verification;
6. update and deploy the public installer routes through the signed website-tag
   workflow;
7. install into fresh isolated directories on both required devices; and
8. run this complete matrix against those installed binaries.

RC1 publication means “ready for tagged-artifact acceptance.” It does not mean
stable `v0.3.0`, and no report may make that claim.

## Device report block

Each immutable report ends with exactly one terminated block:

```text
PHASE3-DEVICE-REPORT-V1
device=<macos-arm64|windows-amd64>
test_tag=v0.3.0-rc.1
test_commit=<40-character commit>
installed_binary_sha256=<sha256>
required_pass=<count>
required_partial=<count>
required_fail=<count>
required_not_tested=<count>
optional_physical_pass=<count>
optional_physical_not_tested=<count>
baseline_provenance=PASS|FAIL
workspace_git=PASS|FAIL
agent_compatibility=PASS|FAIL
capability_privacy=PASS|FAIL
resume_fork=PASS|FAIL
picker=PASS|FAIL
performance=PASS|FAIL
phase1_phase2_regression=PASS|FAIL
release_blocking_findings=<count>
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
```

The coordinator validates branch tip, merge-base, report-only diff, arithmetic,
privacy, and every supersession rather than trusting peer labels. Stable
promotion remains a separate decision after both reports reconcile to PASS.
