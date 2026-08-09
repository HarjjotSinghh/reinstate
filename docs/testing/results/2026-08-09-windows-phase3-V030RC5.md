# Reinstate v0.3.0-rc.5 Windows Phase 3 verified-resume triage report

This additive report supersedes the device verdict in
`2026-08-08-windows-phase3-V030RC5.md`. The original report remains unchanged
at commit `af46d5d182978292338c922652b0eea2bfa26ac4`, with SHA-256
`e2622efe59a97015cc2e25b0a7b569dc4648e66245bfad99204e5243d8202e89`.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `27 PASS / 0 PARTIAL / 5 FAIL / 0 NOT TESTED`
- **Optional physical counts:** `0 PASS / 2 NOT TESTED`
- **Release-blocking findings:** `4`

Missing required human evidence remains `FAIL`. Triage fixed host provisioning
and completed the automated matrix, but it does not manufacture physical TTY,
real-vendor fork, picker, or privileged symlink evidence.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC completion time | `2026-08-09T03:57:41Z` |
| Device | `windows-amd64` |
| OS/version/build | `Microsoft Windows 11 Pro; build 26200` |
| CPU architecture/native process | `x64 OS; native x64 Windows PowerShell 5.1` |
| Hardware/filesystem | `24 logical processors; 63.7 GiB RAM; NTFS` |
| Host protection | `Microsoft Defender and ordinary host protections remained enabled` |
| Tested tag | `v0.3.0-rc.5` |
| Tested full commit | `2cbf0bba06497a4dad039a764ee1ea0d2199cda7` |
| Installed aliases | `rein.exe` and `reinstate.exe`; byte-identical |
| Installed binary SHA-256 | `551f3156f5e06dfc8d2df4e133c2667e9c0e3d0bed6f4f9baf43d3cd7fe26cd4` |
| Installed version identity | `version=0.3.0-rc.5; commit=2cbf0bba06497a4dad039a764ee1ea0d2199cda7` |
| Claude Code | `2.1.220; isolated throwaway home; direct create and continue passed` |
| Codex CLI | `0.146.0; isolated throwaway home; direct create and continue passed` |
| Go/Git/GoReleaser | `go1.25.12 windows/amd64; Git 2.52.0.windows.1; GoReleaser 2.17.0` |
| Report branch | `test/v0.3.0-rc.5-windows-amd64-report` |
| Draft report PR | `#132; remains draft and unmerged` |

No RC1, Phase 2, development, WSL, or other candidate corpus was reused as
RC5 evidence. RC1 aggregates were read only for the dispatch-required
same-host performance regression comparison.

## 2. Signed artifact, installer, and automated gates

The original report's successful immutable chain remains applicable because
triage used the same verified installed bytes and full source commit.

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated SSH-signed tag and `origin/main` ancestry | `PASS` | Tag object verified against `.github/allowed_signers`; peeled 40-character commit is an ancestor of `origin/main`. |
| Published exact prerelease and 25-asset inventory | `PASS` | Non-draft prerelease; one checksum manifest plus exactly 24 checksummed assets in the frozen category counts. |
| Checksums, API digests, and attestations | `PASS` | All 24 manifest checksums, all 25 API digests, and all 25 attestations matched the exact repository, tag ref, workflow, and full commit. |
| Archive/SBOM/source safety | `PASS` | Safe relative membership checked without extraction; five SPDX 2.3 SBOMs and the tagged source archive passed. |
| Live bootstrap and canonical installer identity | `PASS` | Live bootstrap equaled tagged website bytes, contained exactly one RC5 pin, and pinned the exact tagged installer SHA-256. |
| Fresh process-scoped installation and aliases | `PASS` | Brand-new installation; no existing user binary replaced; persistent PATH unchanged; both aliases and attested raw binary were byte-identical. |
| Full commit in installed version JSON | `PASS` | Both aliases reported RC5 and the literal full commit; no shortened commit. |
| `make verify` and complete race suite | `PASS` | Fresh coherent toolchain; pinned Go; complete corrected race run passed with full private output retained. |
| Four cross-builds and five fuzz surfaces | `PASS` | Required targets and all five fuzz-smoke surfaces passed. |
| Snapshot, PowerShell artifact gates, installer smoke | `PASS` | GoReleaser snapshot, native staging/checking, SBOM/source inspection, and installer test passed. |
| Phase 1/2 regression | `PASS` | `CGO_ENABLED=0 go test ./... -count=1` exited `0`. |

## 3. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh vendor sanity before Reinstate | `PASS` | Claude and Codex each created and directly continued one controlled session in throwaway homes before product invocation. |
| Fresh Reinstate homes and disposable repositories | `PASS` | Triage used new RC5-only homes and native-Windows repositories; excluded attempts were preserved and not reused. |
| No init/storage/credential/keyring dependency | `PASS` | No `rein init`, Reinstate storage, passphrase, credential, or keyring write. |
| Owner-only Windows derived state | `PASS` | Cache directory, v2 database, lifetime lock, and write lock had protected allow-only DACLs limited to current user, SYSTEM, and Administrators. |
| Forbidden content absent | `PASS` | No transcript, prompt/response, secret/config value, MCP value, dirty filename/diff, raw child error, or private absolute path appears in this report. |
| Product files unchanged | `PASS` | Base-to-tip product diff is empty; only report files exist above `TEST_COMMIT`. |

## 4. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | Exact tag, full commit, installed hash, signature, checksum, attestation, archive, SBOM, and installer provenance | `PASS` | Complete signed 25-asset and installed-artifact chain passed. |
| 2 | Full verification, race, cross-builds, snapshot/artifact gates, and Phase 1/2 regression | `PASS` | Every required corrected host/product gate exited `0`. |
| 3 | Fresh configless home with no secret/storage dependency | `PASS` | Both aliases returned valid local-session JSON without config or storage state. |
| 4 | Fresh controlled Claude and Codex sessions | `PASS` | Standalone vendor create and continue passed for Claude 2.1.220 and Codex 0.146.0 before Reinstate. |
| 5 | First inspect is `baseline.unavailable` | `PASS` | Fresh first inspect reported the warning honestly for both vendors. |
| 6 | Successful verified launch records only a prelaunch-observed baseline | `PASS` | Real Windows Terminal Claude launch passed and later matched its recorded baseline. |
| 7 | Repeat unchanged report matches repository, branch, HEAD, and tree digest | `PASS` | Repeated installed inspect remained deterministic and ready/matching. |
| 8 | Different repository at the same path blocks with exit `7` | `PASS` | Reversible installed replacement fixture blocked with safety exit `7`; original restored. |
| 9 | Missing/non-directory workspace blocks with exit `5` | `PASS` | Both installed mutations blocked with compatibility exit `5`; original restored. |
| 10 | Branch, detached HEAD, and unborn states remain distinct | `PASS` | Installed reports distinguished all three states. |
| 11 | Equal/ahead/behind/diverged/unavailable HEAD relations are truthful and offline | `PASS` | All five local-only relations were reported exactly. |
| 12 | Dirty-tree states warn without filenames or diffs | `PASS` | Staged, unstaged, untracked, conflict, and submodule-uncertain states passed; no name/diff leakage. |
| 13 | Credential-bearing remotes normalize without leakage | `PASS` | HTTPS, SSH, and SCP forms normalized to the same opaque identity with no credential leakage. |
| 14 | Worktree, symlink, Unicode, case, and native paths are safe | `FAIL` | Linked worktree, junction, Unicode, mixed-case, and native paths passed; required true symbolic-link creation was denied by the non-elevated host before Reinstate. |
| 15 | Claude executable/version/layout is fail-closed | `PASS` | Trusted extensionless lookup resolved Claude 2.1.220; incompatible fixture blocked. |
| 16 | Codex executable/version/layout is fail-closed | `PASS` | Trusted extensionless lookup resolved Codex 0.146.0; missing/incompatible fixtures blocked. |
| 17 | Instruction presence/change is bounded and content-free | `PASS` | Claude/Codex presence, missing, and changed states were name-only and bounded. |
| 18 | Skill presence/change is bounded, content-free, and link-safe | `PASS` | Presence/missing/change passed; escaping junction was excluded and never traversed. |
| 19 | MCP reporting is logical-name/transport-only and value-free | `PASS` | Missing/change and allowlisted JSON fields passed; tagged privacy suite covered command, argument, URL, header, and environment values. |
| 20 | Recognized Node/Go declarations and installed versions are safe and truthful | `PASS` | Go and fresh no-BOM Node fixtures produced match, changed-warning, and unknown-warning states; workspace-owned executables were not run. |
| 21 | Inspect human/JSON output agrees and never prompts/launches | `PASS` | Deterministic JSON, alias parity, decision agreement, and unchanged sources passed for both vendors. |
| 22 | Native dry-run preserves plan, adds report, and never mutates | `PASS` | Resume/fork for both vendors and aliases retained environment/executable/argument structure, exited `0`, and preserved sources. |
| 23 | TTY warning no/EOF/Ctrl-C refuses; yes launches once | `FAIL` | Physical Windows Terminal attempts could not select the isolated Claude record in the operator shell; required response-path evidence is absent. No ConPTY result was invented. |
| 24 | Non-TTY launch requires every exact current warning ID | `PASS` | Missing/partial sets exited `7`; exact sets launched once through each alias. |
| 25 | Unknown/stale/duplicate/wildcard/info/blocker IDs cannot bypass | `PASS` | Invalid warning-ID categories exited usage `2`; blocker precedence remained compatibility `5`. |
| 26 | Hard blockers never prompt and exit precedence is stable | `PASS` | Runtime plus compatibility exited `1`; safety plus compatibility exited `7`; neither launched; fixture restored. |
| 27 | Real same-vendor Claude resume and fork preserve the source | `FAIL` | One real Claude resume passed, but complete resume/fork coverage through both aliases and independent fork resumption is missing. |
| 28 | Real same-vendor Codex resume and fork preserve the source | `FAIL` | Real Codex launch evidence is partial; complete resume/fork coverage through both aliases and independent fork resumption is missing. |
| 29 | Picker paths and both aliases apply identical policy | `FAIL` | Required physical picker inspect/resume/fork/invalid/cancel/interrupt evidence is absent. |
| 30 | Gemini/OpenCode stay read-only with exit `5` | `PASS` | Metadata-only Gemini discovery and a harmless documented OpenCode list fixture were each discovered; resume/fork through both aliases returned `5`; fixtures unchanged. |
| 31 | Hostile, timeout, cancellation, stale, race, concurrency, and privacy gates pass | `PASS` | Exact-tag adversarial packages passed 8/8; installed timeout/replacement/privacy guards passed; 24/24 concurrent processes exited `0` with valid JSON and a healthy final index. |
| 32 | Normal/large, cold/warm latency stays inside RC5 ceilings | `PASS` | Exact `phase3perf-v1` completed all frozen samples, validations, parity, fingerprints, and ceilings with zero timeouts. |

## 5. Performance evidence

The exact tagged harness ran once from `TEST_COMMIT` with canonical digest
`4bf0b653ce76dcc3f7dd93916399bfdea8b658e1fbe41a9423608f2e7a6f8a76`,
installed aliases, a fresh private root, and six canonical physical PATH
directories resolving trusted Git, Claude, Codex, and Node while omitting
OpenCode. Environment digest was
`2478ae98a5cc99418c139aeba2afc69543ca537a8af0eb35343643ce61b17be8`.

- Timeout: `30000ms`; warmup: `1`; warm samples: `20`; cold samples: `3`.
- Method: nearest-rank `ceil(0.95*n)`, one-indexed.
- Alias parity, output validation, controlled anchors, frozen Git HEAD, and
  before/after source fingerprints all passed.
- Zero timeouts, no 20-30 second regression, and no source mutation.
- Worst comparable same-host RC1 warm-p95 increase was approximately `6.8%`,
  below the `25%` blocker; every other comparable increase was also below it.

| Corpus | Mode | Logical command | Samples | Median | p95 | Maximum | Result |
| ------ | ---- | --------------- | ------- | ------ | --- | ------- | ------ |
| Startup | Cold | `version --json` | 3 | `31.908ms` | `32.634ms` | `32.634ms` | `PASS` |
| Startup | Warm | `version --json` | 20 | `29.925ms` | `31.928ms` | `34.559ms` | `PASS` |
| Startup | Cold | `--help` | 3 | `30.962ms` | `31.022ms` | `31.022ms` | `PASS` |
| Startup | Warm | `--help` | 20 | `30.026ms` | `31.297ms` | `31.606ms` | `PASS` |
| Normal | Cold | `sessions` | 3 | `62.535ms` | `62.632ms` | `62.632ms` | `PASS` |
| Normal | Warm | `sessions` | 20 | `35.281ms` | `39.488ms` | `41.276ms` | `PASS` |
| Normal | Warm | `search` | 20 | `35.760ms` | `37.978ms` | `39.898ms` | `PASS` |
| Normal | Warm | `inspect_claude` | 20 | `625.482ms` | `672.565ms` | `684.110ms` | `PASS` |
| Normal | Warm | `resume_claude_dry_run` | 20 | `622.245ms` | `702.968ms` | `735.639ms` | `PASS` |
| Normal | Warm | `resume_codex_dry_run` | 20 | `566.043ms` | `578.633ms` | `594.250ms` | `PASS` |
| Normal | Warm | `fork_claude_dry_run` | 20 | `628.380ms` | `687.799ms` | `744.268ms` | `PASS` |
| Normal | Warm | `fork_codex_dry_run` | 20 | `564.839ms` | `579.898ms` | `609.033ms` | `PASS` |
| Large | Cold | `sessions` | 3 | `197.634ms` | `200.387ms` | `200.387ms` | `PASS` |
| Large | Warm | `sessions` | 20 | `140.612ms` | `148.037ms` | `149.126ms` | `PASS` |
| Large | Warm | `search` | 20 | `141.322ms` | `145.667ms` | `145.776ms` | `PASS` |
| Large | Warm | `inspect_claude` | 20 | `671.627ms` | `769.594ms` | `772.447ms` | `PASS` |
| Large | Warm | `resume_claude_dry_run` | 20 | `676.503ms` | `713.542ms` | `724.099ms` | `PASS` |
| Large | Warm | `resume_codex_dry_run` | 20 | `612.972ms` | `624.053ms` | `686.321ms` | `PASS` |
| Large | Warm | `fork_claude_dry_run` | 20 | `665.144ms` | `683.224ms` | `683.959ms` | `PASS` |
| Large | Warm | `fork_codex_dry_run` | 20 | `618.274ms` | `716.118ms` | `728.760ms` | `PASS` |

Normal corpus was exactly `8` records and `16` capability names with limit
`100`; large was exactly `1000` records and `256` capability names with limit
`1000`. Both used frozen clean remote-free `main` workspaces at
`697ed29583a03045783557c3e8aeec92d9f7f01c`. Materialized digests were
`e0d44770113caed3c7f9adaf7d36accc197fb76ed5a99a3b019895d06ded36f5`
and `4284e4c813cc7c314c80edd8dacff8968b3f51617818f6fbb29dd5e2126b8dc2`.

## 6. RC3 regression items

| Item | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| 1. Human-output privacy | `PASS` | Human/JSON inspect and dry-run outputs remained path-redacted; tagged privacy/adversarial coverage passed. |
| 2. PowerShell 5.1 artifact gates | `PASS` | Native stager/checker/installer gates passed, including metadata casing and native `tar.exe` behavior. |
| 3. Race diagnostics | `PASS` | Full diagnostics were retained; initial host-PATH failure was classified, failed package rerun once, and corrected complete race passed. |
| 4. Executable trust | `PASS` | Extensionless Claude/Codex resolved trusted `.cmd`/`.exe` outside workspaces; verified version ranges were not widened. |
| 5. Human Windows Terminal rows | `FAIL` | Real-TTY warning and picker matrices remain incomplete; missing evidence was not simulated. |

## 7. Release-blocking findings

1. Required true Windows symbolic-link coverage is missing because the native
   host denied link creation without elevated privilege or Developer Mode.
2. Required physical TTY warning default/no/EOF/Ctrl-C/yes evidence is missing;
   the operator shell could not discover the already-proven isolated Claude
   records, so no prompt result was invented.
3. Real same-vendor Claude and Codex resume/fork coverage through both aliases,
   including independent resumption of each distinct fork, is incomplete.
4. Required physical picker inspect/resume/fork/invalid/cancel/interrupt
   coverage through both aliases is missing.

These are host/human evidence gaps, not reproduced Reinstate defects. No actual
Reinstate failure reproduced after standalone vendor sanity passed.

### Sanitized failed commands and excluded harness attempts

- Non-elevated native symbolic-link creation failed before Reinstate; the
  controlled target remained intact.
- Early capability fixtures with a wrapped path and UTF-8 BOM contamination
  were preserved and excluded. Fresh corrected fixtures passed.
- Three physical row-23 helper/selector approaches could not expose the
  isolated Claude record in the operator shell. Empty-reference invocations
  exited usage `2`; they are host/harness failures and not product evidence.
- Row-30 fixture discovery and a PowerShell reserved-variable wrapper attempt
  were preserved and excluded; the corrected installed matrix passed 8/8.
- The first concurrency wrapper observed valid outputs but blank process exit
  properties; the corrected run explicitly refreshed processes and passed
  24/24.

The original report was not amended, replaced, or deleted. This additive file
is the only new path in the superseding commit. Generated binaries, private
logs, vendor homes, repositories, corpora, and retained fixtures remain outside
the report worktree. The PR remains draft and unmerged; this report does not
claim stable readiness.

PHASE3-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.3.0-rc.5
test_commit=2cbf0bba06497a4dad039a764ee1ea0d2199cda7
installed_binary_sha256=551f3156f5e06dfc8d2df4e133c2667e9c0e3d0bed6f4f9baf43d3cd7fe26cd4
required_pass=27
required_partial=0
required_fail=5
required_not_tested=0
optional_physical_pass=0
optional_physical_not_tested=2
baseline_provenance=PASS
workspace_git=FAIL
agent_compatibility=PASS
capability_privacy=PASS
resume_fork=FAIL
picker=FAIL
performance=PASS
phase1_phase2_regression=PASS
release_blocking_findings=4
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
