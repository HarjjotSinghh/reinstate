# Reinstate v0.3.0-rc.2 Windows amd64 Phase 3 Acceptance

## Verdict

**FAIL — exact tagged artifact identity is proven, but the device does not
certify the RC2 product matrix.** The published signed artifacts and live
bootstrap are correct. The release is blocked by reproducible PowerShell
release-gate failures, Codex executable trust resolution, incomplete fresh
vendor-session provisioning, and missing human Windows Terminal evidence.

This report is immutable device evidence for the exact tag below. It does not
authorize merge, stable promotion, or release replacement.

## 1. Immutable test record

| Field | Result |
| --- | --- |
| Device | Native Windows x64, native 64-bit Windows PowerShell |
| PowerShell | Windows PowerShell 5.1.26100.8328; 64-bit process and OS |
| Test tag | `v0.3.0-rc.2` |
| Test commit | `1b0fd801a6a3890a7158d48ebee9ebeeeac296a0` |
| Tag state | Annotated tag; SSH signature verified against the tagged allowed-signers file |
| Main ancestry | Peeled commit is an ancestor of `origin/main` |
| Installed `reinstate.exe` SHA-256 | `36a7ab34ca9aa79811c19a7d5451a939bff19da800d9a8a5dc053bdf1cee6bed` |
| Installed `rein.exe` SHA-256 | Same as `reinstate.exe`; byte-identical aliases |
| Installed version JSON | Version `0.3.0-rc.2`; literal 40-character commit above |
| GOTOOLCHAIN | `go1.25.12` |
| Go | `go1.25.12`; inherited `CGO_ENABLED` removed before non-race verification |
| C/make/sh | Native MSYS2 GCC, make, and sh available through process-scoped PATH |
| GoReleaser | `v2.17.0`, verified official Windows x64 archive, process-scoped |
| Claude Code | `2.1.220`, within the documented `2.1.219`–`2.1.220` range |
| Codex | `0.146.0`, within the documented compatibility range |
| Persistent PATH changes | None; bootstrap scope was process-only |
| Freshness | New isolated install directory, Reinstate home, vendor homes, temp area, and Git repositories |

No WSL process was used. No `rein init` was run. No credentials, passphrases,
keyring data, real transcript content, prompt content, response content, MCP
values, instruction content, skill content, dirty filenames, or diffs were
inspected or committed.

## 2. Signed artifact and installer chain

The GitHub release is a published, non-draft prerelease tied exactly to
`v0.3.0-rc.2`.

| Gate | Result |
| --- | --- |
| Exact release tag/ref | PASS |
| Annotated tag and SSH signature | PASS |
| Full peeled commit and `origin/main` ancestry | PASS |
| Exact asset set | PASS — 25 files: `checksums.txt` plus 24 checksummed assets |
| Asset composition | PASS — 5 platform archives, 5 archive SBOMs, 5 raw binaries, 8 Linux native packages, 1 source archive |
| `checksums.txt` verification | PASS — all 24 entries |
| GitHub API SHA-256 digests | PASS — all 25 downloaded files, including `checksums.txt` |
| GitHub attestations | PASS — all 25 files; exact repository, release workflow, tag ref, and full source commit |
| Archive membership | PASS — safe relative central-directory inspection; no extraction, absolute paths, drive paths, or traversal members |
| Archive SBOMs | PASS — all 5 SPDX 2.3 documents parsed |
| Live bootstrap bytes | PASS — `https://reinstate.dev/install.ps1` byte-identical to tagged `website/public/install.ps1` |
| Bootstrap pin count | PASS — exactly one `v0.3.0-rc.2` release pin |
| Canonical installer digest | PASS — pinned SHA-256 matches exact tagged `scripts/install.ps1` bytes |
| Bootstrap installation | PASS — brand-new isolated directory; no existing binary replaced |
| Alias identity | PASS — `rein.exe` and `reinstate.exe` verified byte-identical |

The live bootstrap downloaded and verified the canonical tagged installer and
Windows archive. The installed identity is therefore the release artifact, not
a source build. A source build was used only as supplemental development
evidence.

## 3. Automated gates

| Gate | Result | Evidence |
| --- | --- | --- |
| `make verify` | PASS | Fresh native PowerShell process; Go 1.25.12; process-scoped MSYS2 toolchain |
| Complete race suite | PASS | `CGO_ENABLED=1 go test ./... -race -count=1`; exit 0; no package rerun required |
| Required CGO-disabled cross-builds | PASS | Darwin arm64, Darwin amd64, Windows amd64, Linux amd64 |
| Five fuzz-smoke surfaces | PASS | All five individual surfaces passed; a combined wrapper failed, then each target passed separately |
| Phase 1/2 regression | PASS | `CGO_ENABLED=0 go test ./... -count=1`; exit 0 |
| `make snapshot` | **FAIL** | Repeated exact native invocation exited 2; the snapshot was incomplete |
| `stage-release-assets.ps1` | **FAIL** | PowerShell-native gate exited 1 against the incomplete snapshot |
| `check-release-artifacts.ps1` | **FAIL** | PowerShell-native gate exited 1 against the incomplete snapshot |
| `test-install.ps1` | **FAIL** | PowerShell-native gate exited 1 against the incomplete snapshot |

The failed snapshot and PowerShell release gates are release-blocking. Their
complete output is retained only in private evidence; this report intentionally
contains no raw child errors or private paths.

## 4. Isolation and privacy

The behavior run used fresh disposable native-Windows Git repositories and
throwaway Claude, Codex, Gemini, process, temporary, and Reinstate homes. The
Reinstate home was empty before invocation. The observed generated state was
limited to expected index/lock data, and the inspected lock/database files had
owner-only DACL evidence. No persistent user or machine PATH entry changed.

Fresh Claude and Codex client invocations were attempted with controlled
non-tool prompts. Both vendor commands exited 1 before a completed controlled
session was established; their private output was retained without inspection.
Consequently, no real same-vendor resume/fork claim is made.

## 5. Required 32-row matrix

Missing required evidence is recorded as `FAIL`, per the RC2 dispatch. No row
is hidden as `NOT TESTED`.

| # | Acceptance row | Result | Sanitized evidence |
| ---: | --- | --- | --- |
| 1 | Exact tag, full commit, installed hash, signature, checksum, attestation, archive, SBOM, and installer provenance | PASS | Complete chain in section 2; installed version reports the full commit |
| 2 | Full verification, race, cross-builds, and Phase 1/2 regression | PASS | All listed suites passed; release snapshot gates are tracked separately |
| 3 | Fresh configless home with no secret/storage dependency | PASS | Fresh home; no init, credentials, passphrase, keyring, or backend dependency |
| 4 | Fresh controlled Claude and Codex sessions | FAIL | Both fresh vendor commands exited 1; no completed controlled sessions |
| 5 | First inspect is `baseline.unavailable` | PASS | First inspect of both discovered fresh records reported unavailable baseline; no path leak |
| 6 | Successful verified launch records only a prelaunch-observed baseline | FAIL | No completed vendor launch was available to prove this row |
| 7 | Repeat unchanged report matches repository, branch, HEAD, and tree digest | FAIL | Required post-launch baseline was unavailable |
| 8 | Different repository at the same path blocks with exit 7 | FAIL | Required fresh successful baseline was unavailable |
| 9 | Missing/non-directory workspace blocks with exit 5 | FAIL | Required physical row evidence was not completed |
| 10 | Branch, detached HEAD, and unborn states remain distinct | FAIL | Required physical state matrix was not completed |
| 11 | Equal/ahead/behind/diverged/unavailable HEAD relations are truthful and offline | FAIL | Required physical relation matrix was not completed |
| 12 | Dirty-tree states warn without filenames or diffs | FAIL | Required staged, unstaged, untracked, conflicted, and submodule matrix was not completed |
| 13 | Credential-bearing remotes normalize without leakage | FAIL | Required remote normalization matrix was not completed |
| 14 | Worktree, symlink, Unicode, case, and native paths are safe | FAIL | Required physical path matrix was not completed |
| 15 | Claude executable/version/layout is fail-closed | PASS | Claude 2.1.220; inspect found no unavailable executable state |
| 16 | Codex executable/version/layout is fail-closed | FAIL | Installed `codex.exe` was on trusted PATH, but extensionless Codex lookup still reported native executable unavailable |
| 17 | Instruction presence/change is bounded and content-free | FAIL | Required physical presence/change matrix was not completed |
| 18 | Skill presence/change is bounded, content-free, and does not follow escaping links | FAIL | Required physical skill matrix was not completed |
| 19 | MCP reporting is logical-name/transport-only and value-free | FAIL | Required physical MCP matrix was not completed |
| 20 | Recognized Node/Go declarations and installed versions are safe and truthful | FAIL | Required declaration/version matrix was not completed |
| 21 | Inspect human/JSON output agrees and never prompts/launches | FAIL | JSON inspect evidence exists; paired human-output/determinism evidence was not completed |
| 22 | Native dry-run preserves plan, adds report, and never mutates | FAIL | Claude dry-run was observed, but the required complete Claude+Codex matrix was blocked by Codex resolution |
| 23 | TTY warning no/EOF/Ctrl-C refuses; yes launches once | FAIL | No human-supplied real TTY evidence; ConPTY input was not invented |
| 24 | Non-TTY launch requires every exact current warning ID | PASS | Both aliases refused without the exact fresh warning ID, exit 7, with no vendor launch |
| 25 | Unknown/stale/duplicate/wildcard/info/blocker IDs cannot bypass | FAIL | Required warning-ID adversarial matrix was not completed |
| 26 | Hard blockers never prompt and exit precedence is stable | FAIL | Only partial blocker evidence exists; the complete precedence matrix was not completed |
| 27 | Real same-vendor Claude resume and fork preserve the source | FAIL | No completed fresh Claude session existed to resume or fork |
| 28 | Real same-vendor Codex resume and fork preserve the source | FAIL | Codex executable resolution failed; resume/fork returned compatibility exit 5 |
| 29 | Picker paths and both aliases apply identical policy | FAIL | No human-supplied real TTY picker evidence |
| 30 | Gemini/OpenCode stay read-only with exit 5 | FAIL | Optional vendors were not installed solely for evidence; required refusal rows were not completed |
| 31 | Hostile, timeout, cancellation, stale, race, concurrency, and privacy gates pass | FAIL | Unit/race evidence passed, but the required installed-artifact physical matrix was not completed |
| 32 | Normal/large, cold/warm latency stays inside the RC2 ceilings | PASS | Final fixed `phase3perf-v1` run passed; aggregates in section 6 |

**Counts:** 7 PASS, 0 PARTIAL, 25 FAIL, 0 NOT TESTED = 32 required rows.

## 6. Performance evidence

The final invocation used the fixed `phase3perf-v1` generator and the canonical
fixture digest
`4bf0b653ce76dcc3f7dd93916399bfdea8b658e1fbe41a9423608f2e7a6f8a76`.
It used 1 warmup, 20 warm samples, 3 cold samples, and a 30-second timeout.
All samples validated and all timeout counts were zero. The installed and
alias binary digests were identical.

### Startup and corpus cold aggregates

| Surface | Mode | Samples | Median | P95 | Maximum | Result |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| `version` | Cold | 3 | 30.775 ms | 33.352 ms | 33.352 ms | PASS |
| `version` | Warm | 20 | 29.841 ms | 31.214 ms | 32.013 ms | PASS |
| `help` | Cold | 3 | 30.301 ms | 31.636 ms | 31.636 ms | PASS |
| `help` | Warm | 20 | 28.748 ms | 31.811 ms | 38.057 ms | PASS |
| Normal corpus startup | Cold | 3 | 60.619 ms | 60.751 ms | 60.751 ms | PASS |
| Large corpus startup | Cold | 3 | 202.824 ms | 204.294 ms | 204.294 ms | PASS |

### Normal corpus

The normal corpus contained 8 records and 16 capability names with limit 100.
Alias parity, clean source state, source fingerprint stability, OpenCode
omission, and ambient-capability absence were all true.

| Command | Samples | Median | P95 | Maximum | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| `sessions` | 20 | 34.907 ms | 38.184 ms | 42.862 ms | PASS |
| `search` | 20 | 34.636 ms | 36.250 ms | 36.838 ms | PASS |
| Claude inspect | 20 | 567.949 ms | 580.167 ms | 580.675 ms | PASS |
| Claude resume dry-run | 20 | 566.581 ms | 580.413 ms | 669.667 ms | PASS |
| Codex resume dry-run | 20 | 701.271 ms | 711.737 ms | 721.533 ms | PASS |
| Claude fork dry-run | 20 | 573.688 ms | 630.183 ms | 682.097 ms | PASS |
| Codex fork dry-run | 20 | 704.967 ms | 715.588 ms | 717.463 ms | PASS |

### Large corpus

The large corpus contained 1,000 records and 256 capability names with limit
1,000. Alias parity, clean source state, source fingerprint stability, OpenCode
omission, and ambient-capability absence were all true.

| Command | Samples | Median | P95 | Maximum | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| `sessions` | 20 | 138.044 ms | 140.331 ms | 140.749 ms | PASS |
| `search` | 20 | 139.151 ms | 145.583 ms | 147.458 ms | PASS |
| Claude inspect | 20 | 615.282 ms | 626.211 ms | 707.138 ms | PASS |
| Claude resume dry-run | 20 | 611.313 ms | 623.838 ms | 628.258 ms | PASS |
| Codex resume dry-run | 20 | 751.599 ms | 820.590 ms | 831.734 ms | PASS |
| Claude fork dry-run | 20 | 617.027 ms | 628.131 ms | 628.958 ms | PASS |
| Codex fork dry-run | 20 | 746.639 ms | 759.036 ms | 785.315 ms | PASS |

Every measured surface stayed below the documented Windows ceilings. No
20–30-second regression was observed. No comparable same-host RC1 per-command
baseline was available in this fresh evidence run, so no percentage regression
claim is made.

## 7. Findings and repository hygiene

### Release-blocking findings

1. `make snapshot` reproducibly exited 2, and all three PowerShell-native
   snapshot/stage/artifact/install gates consequently exited nonzero. The
   host was provisioned honestly, including GoReleaser; this is not an excuse
   to relabel the gate green.
2. The installed artifact did not resolve extensionless `codex` to the trusted
   installed `codex.exe` on PATH. Both aliases exposed the unavailable-native-
   executable state, and Codex resume/fork returned compatibility exit 5.
3. Fresh controlled Claude and Codex client commands both exited 1 before a
   completed session existed. Real same-vendor resume/fork therefore has no
   valid source corpus and remains FAIL.
4. Human Windows Terminal evidence for real TTY picker and warning-policy rows
   was not supplied. Those required rows are FAIL; no synthetic ConPTY input
   was used.
5. The remaining required physical acceptance rows listed as FAIL above lack
   evidence because the prerequisite real session and complete matrix were not
   established. They cannot be promoted by inference from unit tests or the
   performance harness.

### RC1 regression items

1. **Windows executable trust — FAIL.** The required Codex extensionless lookup
   regression remains present; `codex.exe` was available on trusted PATH but
   was reported unavailable.
2. **Race diagnostics — PASS.** The complete race suite exited 0, so no failed
   package rerun was required and no stderr was discarded as a substitute for
   evidence.
3. **Artifact gates on Windows — FAIL.** The PowerShell-native scripts were
   used as first-class gates, and the snapshot/stage/check/install chain exited
   nonzero. Missing GNU helper tools were not used as the reason for failure.
4. **Claude version range — PASS.** Claude Code `2.1.220` was used, within the
   documented `2.1.219`–`2.1.220` range. The report does not widen that range.
5. **Human Windows Terminal rows — FAIL.** Real TTY picker/warning evidence
   was not supplied, and no agent-generated ConPTY input was treated as human
   QA.

### Non-blocking and harness notes

- A combined fuzz-smoke wrapper exited 1 because of wrapper invocation behavior;
  all five required fuzz surfaces passed when run as individual targets. This
  is recorded as a harness deviation, not a product pass substitution.
- Earlier performance invocations rejected pre-created or non-canonical
  curated PATH roots. The final canonical physical-path invocation passed and
  is the only performance result used above.

No product files were changed in this report worktree. The report branch diff
contains only this report file. No secrets or transcripts were committed.

## 8. RC2 report handoff

This branch is created from the exact peeled test commit. The report commit is
intended to remain unamended after publication, and its pull request must stay
draft and unmerged for macOS coordinator reconciliation.

PHASE3-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.3.0-rc.2
test_commit=1b0fd801a6a3890a7158d48ebee9ebeeeac296a0
installed_binary_sha256=36a7ab34ca9aa79811c19a7d5451a939bff19da800d9a8a5dc053bdf1cee6bed
required_pass=7
required_partial=0
required_fail=25
required_not_tested=0
optional_physical_pass=0
optional_physical_not_tested=2
baseline_provenance=FAIL
workspace_git=FAIL
agent_compatibility=FAIL
capability_privacy=FAIL
resume_fork=FAIL
picker=FAIL
performance=PASS
phase1_phase2_regression=PASS
release_blocking_findings=5
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
