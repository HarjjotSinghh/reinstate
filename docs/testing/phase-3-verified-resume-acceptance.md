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

These are the supported mandatory platforms for RC1 and stable `v0.3.0`.
Intel macOS and Linux/WSL2 are unsupported/unverified optional evidence. Do not
install them solely for acceptance, count a missing or failed result as a
release failure, or claim support based on built artifacts. They do not block
RC1 or stable `v0.3.0`. Stable promotion still requires a separate decision and
fresh tagged-artifact validation on both supported platforms under
`RELEASING.md`; an RC1 pass does not automatically authorize stable.

The exact tag-specific instructions are the committed
[`v0.3.0-rc.1` dispatch](v0.3.0-rc.1-agent-verification-prompts.md). Each device
uses the cumulative sanitized
[Phase 3 report template](results/phase-3-report-template.md). The dispatch
fixes report branches, artifact/full-commit checks, corpus sizes, absolute
performance ceilings, and final reconciliation rules before physical testing
starts.

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

The reproducible synthetic development methodology and Phase 3 before/after
results are recorded in [Phase 3 CLI performance](phase-3-cli-performance.md).
Those source-level results do not replace the installed-artifact samples below.

Measure the installed artifact with the checked-in
`scripts/testing/phase3perf` generator and harness from the exact tagged source.
The embedded specification freezes the normal and large corpus schemas, every
generated byte modulo the materialized absolute workspace placeholder, stable
controlled references and search query, result limit, timed executable, exact
commands, sample counts, cold-reset procedure, capture limits, timeout, p95
method, and ceilings before testing. The generator writes a root-independent
canonical digest and a root-bound materialized digest; both must verify before
timing. Device reports may not combine commands into one aggregate or relax the
contract after observing results.

The normal corpus is exactly four Claude plus four Codex records and sixteen
capability names. The large corpus is exactly 500 Claude plus 500 Codex records
and 256 capability names. Every record contains exactly four JSONL events, two
counted messages, and two file references; the embedded specification also
freezes ID/title/message/file/capability byte widths, timestamps, anchor index,
and a tracked 64-byte workspace file. Corpus count is never greater than its
result limit and both anchors must be visible in every full result.

Each corpus has distinct `REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`,
`GEMINI_CLI_HOME`, process home, temporary directory, working directory, and
cold-evidence directory. Its working directory is a clean, remote-free,
controlled SHA-1 Git repository with a frozen branch, commit identity, and
HEAD. The harness builds a fixed allowlisted environment rather than inheriting
ambient agent homes or credentials. Its operator-supplied curated `PATH` must
contain absolute canonical physical directories outside the source checkout
and evidence root, must resolve trusted Git/Claude/Codex executables and the
installed aliases, and must not resolve OpenCode. Any OpenCode visibility,
unexpected capability name/count, dirty workspace, source mutation, PATH
ambiguity, or manifest mismatch fails closed before it can become performance
evidence. No product bypass flag or vendor source mutation is permitted.

Use `rein` as the timed executable. Before timing each corpus, run the same
command set once through `reinstate` and compare exits plus normalized JSON;
alias parity is a required untimed precondition. Substitute each corpus's
literal controlled values into these exact command shapes and preserve this
flag order:

```text
rein sessions --limit LIMIT --json
rein search QUERY --limit LIMIT --json
rein inspect CLAUDE_REF --json
rein resume CLAUDE_REF --dry-run --json
rein resume CODEX_REF --dry-run --json
rein fork CLAUDE_REF --dry-run --json
rein fork CODEX_REF --dry-run --json
```

The references and query must select stable, known controlled records. Dry-runs
must use no warning-acknowledgement flags: warning-only reports already exit
zero, while a blocked dry-run invalidates the sample. Run commands sequentially
with no live corpus writer. Record twenty independent process-start-to-exit
warm samples for **each** of the seven commands, then report per-command median,
p95, maximum, exit/result validation, and timeout count. An average or pooled
"inspect/dry-run" row is not evidence.

Measure installed `rein version --json` and `rein --help` separately from the
corpus commands. For each, take three process-start-to-exit samples with a fresh
dedicated startup home, then one untimed warmup and twenty sequential warm
samples in another dedicated home. Require untimed alias parity, literal RC
version/full-commit validation for version JSON, and semantic required-command
plus normalized alias validation for help. A fresh home does not claim to evict
the OS filesystem or executable page cache.

Take three cold samples per corpus using only the first command. Before each
sample, close every Reinstate process using the isolated `REINSTATE_HOME` and
follow the manual-reset procedure in [troubleshooting](../troubleshooting.md):
move the exact `session-index-v2.sqlite`, `.lock`, `.write.lock`, and any
SQLite `-journal`/`-wal`/`-shm` companions out of `cache/` into a distinct
private evidence directory. Do not delete them, move any other home entry, or
touch a vendor session source. Confirm the v2 family is absent, time one fresh
`rein sessions --limit LIMIT --json` process, and verify its JSON and expected
controlled records. Keep every moved family until reconciliation. The final
cold run may seed the warm samples after one untimed validation refresh.

The harness captures stdout/stderr into bounded memory, validates every sample,
then discards raw output; it uses Go's monotonic `time.Now`/`time.Since` process
start-to-exit duration, a 30-second per-process timeout, one warmup, twenty warm
samples, and nearest-rank `ceil(0.95*n)` p95. Fingerprint the exact controlled
vendor sources before and after each corpus;
any Reinstate-caused mutation is a failure. Raw JSON, raw timings, source
content, transcripts, and private paths stay outside the repository. Commit
only sanitized corpus counts, logical command labels, aggregates, ceilings,
parity/mutation results, and bounded diagnostics. Record hardware, OS,
filesystem, agent versions, and generic antivirus state; do not weaken host
protections solely for testing.

Warm-command p95 targets are one second on Apple Silicon and two seconds on
native Windows. Release-blocking normal-corpus ceilings are two and four
seconds, respectively, and apply to every warm command independently. Startup
version/help warm p95 ceilings are two seconds on macOS and four seconds on
Windows; startup cold maxima are four and eight seconds. The candidate dispatch
must freeze full-refresh and large-corpus ceilings; device reports may not relax them. Any failed
result validation, source mutation, timeout, unbounded command/file count,
20–30 second regression, or more than 25 percent comparable same-host
per-command p95 regression is a blocker. For `v0.3.0-rc.1`, the values and
ceilings are frozen in
[`v0.3.0-rc.1-agent-verification-prompts.md`](v0.3.0-rc.1-agent-verification-prompts.md).

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
