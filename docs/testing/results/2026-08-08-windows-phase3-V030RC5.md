# Reinstate v0.3.0-rc.5 Windows Phase 3 verified-resume report

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `14 PASS / 0 PARTIAL / 18 FAIL / 0 NOT TESTED`
- **Optional physical counts:** `0 PASS / 2 NOT TESTED`
- **Release-blocking findings:** `9`

Missing required evidence is recorded as `FAIL`, never `PASS` or `NOT TESTED`.
No RC1, Phase 2, development, WSL, or prior-candidate corpus, home,
installation, or behavioral evidence was reused.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-08T19:27:30Z` |
| Device | `windows-amd64` |
| OS/version/build | `Microsoft Windows 11 Pro; build 26200` |
| CPU architecture/native process | `x64 OS; native x64 Windows PowerShell 5.1 process` |
| Hardware | `24 logical processors; 63.7 GiB RAM` |
| Filesystem | `NTFS` |
| Host protection | `Microsoft Defender antivirus and real-time protection enabled` |
| Tested tag | `v0.3.0-rc.5` |
| Tested full commit | `2cbf0bba06497a4dad039a764ee1ea0d2199cda7` |
| Installed binary SHA-256 | `551f3156f5e06dfc8d2df4e133c2667e9c0e3d0bed6f4f9baf43d3cd7fe26cd4` |
| Installed version JSON | `version=0.3.0-rc.5; commit=2cbf0bba06497a4dad039a764ee1ea0d2199cda7; date=2026-08-08T16:37:27Z` |
| Claude Code version/state | `2.1.226; executable resolved, but version is outside 2.1.219-2.1.220 and correctly blocked` |
| Codex CLI version/state | `0.146.0; executable resolved from extensionless codex through trusted PATH; compatible` |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.12 windows/amd64` |
| GoReleaser / Syft | `2.17.0 / 1.50.0` |
| Performance corpus preparation | `normal 4 Claude + 4 Codex / 16 capabilities; large 500 + 500 / 256 capabilities; timing aborted before samples` |
| Report branch | `test/v0.3.0-rc.5-windows-amd64-report` |
| Device-report commit | `the immutable commit containing this terminated report; returned in the handoff` |
| Draft report PR | `created after the immutable commit; URL returned in the handoff` |

## 2. Signed artifact and installer chain

No downloaded executable ran until its static identity and provenance passed.

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag and SSH signature against `.github/allowed_signers` | `PASS` | Tag object type `tag`; `git verify-tag` exit `0`. Signer fingerprint is intentionally omitted. |
| Tag peels to full commit on `origin/main` | `PASS` | Exact 40-character commit recorded above; merge-base ancestor check exit `0`. |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub release API: `draft=false`, `prerelease=true`, exact tag, 25 assets. |
| Exact 25-asset set | `PASS` | One checksum manifest, five platform archives, five matching SBOMs, five raw binaries, eight Linux packages, and one source archive. |
| Every checksum and GitHub API digest | `PASS` | 24 checksum entries passed; all 25 API SHA-256 digests matched downloaded bytes. |
| Every GitHub attestation, including `checksums.txt` | `PASS` | 25/25 passed with exact repository, release workflow, tag ref, full source commit, and self-hosted runners denied. |
| Safe archive membership | `PASS` | Five platform archives and source archive listed without extraction; no absolute, drive-rooted, or parent-traversal member. Platform member counts were 5/5/5/5/6. |
| SBOM and source inspection | `PASS` | Five SBOMs parsed as SPDX 2.3; source archive had 1,028 safe members. |
| Raw binaries and Linux packages | `PASS` | Five raw binaries and eight native packages were present, checksummed, API-digest verified, and attested. |
| Live public bootstrap identity and single pin | `PASS` | Live bytes equaled the tagged website bootstrap; length 4,889; SHA-256 `446ef13b9eb0108f0577d30e32aeba2e60e056ecec75a1645c24d4a141ce52ce`; exactly one `v0.3.0-rc.5` pin. |
| Canonical installer digest | `PASS` | Bootstrap pin matched tagged `scripts/install.ps1` SHA-256 `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe`. |
| Fresh isolated bootstrap installation | `PASS` | Brand-new install directory, process-only PATH scope, no existing binary replacement, user PATH unchanged. |
| Alias and release binary identity | `PASS` | `rein.exe`, `reinstate.exe`, and the attested raw Windows binary were byte-identical at the recorded SHA-256. |
| Literal full-commit version identity | `PASS` | Both aliases reported RC5 and the exact full commit; no short commit. |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean worktree and `go mod tidy -diff` | `PASS` | Tidy diff exit `0`; tracked source diff remained empty. |
| `make verify` in a coherent native toolchain | `PASS` | Fresh PowerShell, process-scoped MSYS2, pinned Go; exit `0`; vet, lint, tests, race subset, vulnerability scan, and build passed. |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `PASS` | Initial invalid host-PATH attempts are preserved below; corrected coherent-toolchain full suite exited `0`. |
| Four CGO-disabled cross-builds | `PASS` | `darwin/arm64`, `darwin/amd64`, `windows/amd64`, and `linux/amd64`; object-format smoke passed. |
| Five fuzz-smoke surfaces | `PASS` | Six individual invocations covered porcelain parsing, remote normalization, both configuration-name parsers, policy evaluation, and safe rendering; all passed. |
| GoReleaser snapshot and PowerShell release gates | `PASS` | `snapshot.ps1`, staging, artifact/SBOM/source inspection, and installer smoke all exited `0`; five raw binaries staged. |
| Phase 1 and Phase 2 regression | `PASS` | Explicit `CGO_ENABLED=0 go test ./... -count=1` exit `0`. |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated Reinstate home and disposable repositories | `PASS` | RC5-only configless home and native Git repositories; no old index or corpus reused. |
| Fresh controlled Claude and Codex sessions | `FAIL` | Both fresh vendor attempts used throwaway homes and created discoverable metadata, but each process exited `1`; no successful controlled vendor session was established. |
| No init, storage, credential, passphrase, or keyring dependency | `PASS` | No `rein init`; no Reinstate config/state file; no backend or secret flow. |
| Native owner-only derived-state protection | `PASS` | Cache, v2 database, lifetime lock, and write lock had protected DACLs limited to current user, SYSTEM, and Administrators. |
| No forbidden content recorded | `PASS` | No transcript, prompt/response, secret, configuration value, private absolute path, dirty filename/diff, or raw child error is in this report. Vendor output was discarded without content inspection. |
| Product and operator vendor configuration unchanged | `PASS` | Product worktrees stayed tracked-clean; only throwaway vendor homes and disposable repositories were used. |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | Exact tag, full commit, installed hash, signature, checksum, attestation, archive, SBOM, and installer provenance | `PASS` | Complete 25-asset and installed-artifact chain passed. |
| 2 | Full verification, race, cross-builds, and Phase 1/2 regression | `PASS` | All corrected required automated gates passed; original harness failures are preserved. |
| 3 | Fresh configless home with no secret/storage dependency | `PASS` | Both aliases returned valid local session JSON; no config or state file was created. |
| 4 | Fresh controlled Claude and Codex sessions | `FAIL` | Both vendor creation processes exited `1`; failed metadata was not relabeled as a successful session. |
| 5 | First inspect is `baseline.unavailable` | `PASS` | Fresh Claude and Codex inspect each reported exactly one `baseline.unavailable` warning with `unknown` status. |
| 6 | Successful verified launch records only a prelaunch-observed baseline | `FAIL` | No real native vendor launch exited successfully. |
| 7 | Repeat unchanged report matches repository, branch, HEAD, and tree digest | `FAIL` | Deterministic repeated inspect passed, but no successful launch existed to establish the required comparison baseline. |
| 8 | Different repository at the same path blocks with exit `7` | `FAIL` | No trustworthy prelaunch baseline existed, so the installed repository-replacement gate could not be closed. |
| 9 | Missing/non-directory workspace blocks with exit `5` | `PASS` | Both reversible installed-artifact mutations produced a blocked report and resume exit `5`; the original repo was restored. |
| 10 | Branch, detached HEAD, and unborn states remain distinct | `PASS` | Installed inspect distinguished named branch, detached state, and unborn `main`; original branch restored. |
| 11 | Equal/ahead/behind/diverged/unavailable HEAD relations are truthful and offline | `PASS` | Local-only controlled upstream produced exact equal, ahead, behind, diverged, and unknown/not-knowable results without preflight network access. |
| 12 | Dirty-tree states warn without filenames or diffs | `FAIL` | Installed staged, unstaged, and untracked counts passed with no filename/content leak; conflict fixture setup failed three times before product inspection and submodule evidence was not completed. |
| 13 | Credential-bearing remotes normalize without leakage | `PASS` | Controlled HTTPS and SSH userinfo/password/query/fragment sentinels never appeared in JSON or human output; original remote restored. |
| 14 | Worktree, symlink, Unicode, case, and native paths are safe | `FAIL` | Required installed-artifact linked-worktree/symlink/Unicode/case matrix was not independently completed. |
| 15 | Claude executable/version/layout is fail-closed | `PASS` | Extensionless trusted lookup found Claude; installed 2.1.226 produced blocker `agent.version` and dry-run/launch exit `5`. |
| 16 | Codex executable/version/layout is fail-closed | `PASS` | Extensionless `codex` resolved the trusted 0.146.0 executable; resume/fork dry-runs passed through both aliases. |
| 17 | Instruction presence/change is bounded and content-free | `FAIL` | Required installed instruction mutation matrix was not independently completed. |
| 18 | Skill presence/change is bounded, content-free, and does not follow escaping links | `FAIL` | Required installed skill mutation/link-escape matrix was not independently completed. |
| 19 | MCP reporting is logical-name/transport-only and value-free | `FAIL` | Required installed MCP mutation matrix was not independently completed. |
| 20 | Recognized Node/Go declarations and installed versions are safe and truthful | `FAIL` | Required installed runtime-declaration mutation matrix was not independently completed. |
| 21 | Inspect human/JSON output agrees and never prompts/launches | `PASS` | Repeated JSON was deterministic, aliases matched, decision agreed, human output contained no absolute Windows path, and vendor source fingerprints stayed unchanged. |
| 22 | Native dry-run preserves plan, adds report, and never mutates | `PASS` | Codex resume/fork plans retained the required top-level keys and environment report; Claude blocked truthfully; aliases matched and sources were unchanged. |
| 23 | TTY warning no/EOF/Ctrl-C refuses; yes launches once | `FAIL` | Human Windows Terminal evidence was not supplied; no ConPTY input was invented. |
| 24 | Non-TTY launch requires every exact current warning ID | `PASS` | Missing acknowledgment exited `7` without source mutation; exact `baseline.unavailable` acknowledgment reached the vendor, whose child then exited `1`. |
| 25 | Unknown/stale/duplicate/wildcard/info/blocker IDs cannot bypass | `PASS` | Unknown/stale, wildcard, informational, and duplicate IDs exited `2`; Claude blocker still exited `5`. |
| 26 | Hard blockers never prompt and exit precedence is stable | `FAIL` | Installed Claude blocker was non-interactive and exit `5`; the full installed runtime/safety/compatibility precedence combination matrix was not completed. |
| 27 | Real same-vendor Claude resume and fork preserve the source | `FAIL` | Claude 2.1.226 was outside the verified range and both actions were blocked before launch. |
| 28 | Real same-vendor Codex resume and fork preserve the source | `FAIL` | Both correctly authorized real actions reached Codex but child execution exited `1`; no distinct successful fork could independently resume. |
| 29 | Picker paths and both aliases apply identical policy | `FAIL` | Non-interactive alias parity passed, but required human picker/real-TTY evidence was absent. |
| 30 | Gemini/OpenCode stay read-only with exit `5` | `FAIL` | Both optional vendors were installed, but fresh controlled physical read-only-session evidence was not completed; source fixture coverage is supplemental only. |
| 31 | Hostile, timeout, cancellation, stale, race, concurrency, and privacy gates pass | `FAIL` | Automated adversarial suites passed, but the required complete installed stale/path-race/concurrency/privacy matrix was not closed. |
| 32 | Normal/large, cold/warm latency stays inside RC5 ceilings | `FAIL` | Exact harness exited `1` before samples because normal-corpus preflight was blocked; no `results.json`, aggregates, relative comparison, or ceiling pass exists. |

## 6. Performance evidence

The exact tagged `phase3perf-v1` harness and canonical specification digest
`4bf0b653ce76dcc3f7dd93916399bfdea8b658e1fbe41a9423608f2e7a6f8a76`
were used with the attested installed aliases, exact full commit, exact RC5
version, a new evidence root, and a curated physical PATH containing trusted
Git, Claude, Codex, and installed Reinstate directories while omitting
OpenCode. Ordinary host protection remained enabled.

The harness prepared the frozen normal and large corpus shapes but stopped at
normal-corpus measurement with sanitized diagnostic `preflight decision is not
a valid warning/ready result`. Independent installed inspect showed Claude
2.1.226 blocked on `agent.version`; this is the evidence-backed cause of the
invalid preflight. The harness wrote no `results.json`, so all startup,
normal/large, cold/warm medians, p95 values, maxima, timeout counts, and
same-host regression comparisons are **unavailable**. No partial timing is
promoted into an aggregate.

| Performance item | Result | Sanitized evidence |
| ---------------- | ------ | ------------------ |
| Exact tagged harness, commit, version, installed aliases, and canonical digest | `PASS` | Invocation passed initial identity/environment validation and created fresh fixed corpora. |
| Frozen normal and large corpus preparation | `PASS` | Normal `4+4/16`; large `500+500/256`; separate isolated roots created. |
| Untimed complete alias parity | `FAIL` | Harness stopped before completing all required untimed corpus preconditions. |
| Startup cold/warm aggregates | `FAIL` | No final results file; no defensible aggregates. |
| Normal cold/warm aggregates | `FAIL` | No samples accepted. |
| Large cold/warm aggregates | `FAIL` | No samples accepted. |
| Source fingerprints and six cold-family preservations | `FAIL` | Full before/after and cold-reset sequence did not complete. |
| Absolute ceilings and same-host regression | `FAIL` | Not measurable; a missing comparison is not a pass. |

## 7. RC3 regression items

| Item | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| 1. Human-output privacy | `PASS` | Installed inspect and dry-run human output contained no absolute Windows path; focused privacy tests passed for outside-home and allowlisted rendering. |
| 2. PowerShell 5.1 artifact gates | `PASS` | Tagged stager parsed and staged five binaries; metadata-without-extra plus `extra.ID`/`extra.id` targeted tests passed; artifact inspection passed while MSYS2 was on PATH and the script selected native System32 `tar.exe`. |
| 3. Race diagnostics | `PASS` | Complete output was retained privately; the failed package was rerun once; missing process-scoped `sh` was proven as host setup error; corrected coherent full race passed. |
| 4. Executable trust | `PASS` | Extensionless Claude and Codex resolved trusted `.cmd`/`.exe` outside the workspace; quoted-PATH/PATHEXT targeted tests passed. The published compatibility ranges were not widened; Claude 2.1.226 remained blocked. |
| 5. Human Windows Terminal rows | `FAIL` | Required real-TTY picker/warning/repository-swap evidence was not supplied. |

## 8. Findings and repository hygiene

### Release-blocking

1. Claude Code `2.1.226` is outside the verified `2.1.219`-`2.1.220` range;
   installed Claude actions correctly block, and the fixed performance harness
   cannot reach a warning/ready normal-corpus preflight.
2. Both fresh vendor creation commands exited `1`; real Codex resume and fork
   also exited `1`, so neither vendor produced a successful native continuation.
3. Without one successful verified launch, prelaunch-observed baseline,
   unchanged-baseline comparison, and installed repository-replacement safety
   evidence cannot pass.
4. Required human Windows Terminal picker, warning, and prompt-open repository
   swap evidence was not supplied.
5. Installed conflict/submodule dirty-state and worktree/symlink/Unicode/case
   matrices were incomplete.
6. Installed instruction, skill, MCP, and runtime mutation matrices were
   incomplete.
7. The full installed blocker-precedence matrix was incomplete.
8. Gemini/OpenCode physical read-only and complete installed
   stale/path-race/concurrency/privacy matrices were incomplete.
9. The fixed performance harness stopped before samples and produced no
   aggregates or results file.

### Sanitized failed commands and harness deviations

- Initial complete race invocation and its one failed-package rerun lacked the
  required process-scoped MSYS2 PATH, so `sh` was absent and one doctest failed.
  Both logs remain private; the corrected full race passed.
- Three PowerShell-fed Git index conflict-fixture attempts failed before
  product inspection (first malformed input, then two exit `128` results).
  Per the three-fix stop rule, no fourth workaround was attempted; the repo
  remained clean and row 12 stayed `FAIL`.
- Fresh Claude and Codex session-creation commands exited `1`; their stdout and
  stderr were discarded without inspection.
- Installed acknowledged Codex resume and fork each exited `1`; Claude resume
  and fork were blocked with exit `5`.
- Sanitized performance command shape:
  `go run ./scripts/testing/phase3perf run --root <new> --rein <installed> --reinstate <installed> --source-root <tagged> --expected-commit <full> --expected-version 0.3.0-rc.5 --path <curated>`;
  exit `1` before accepted samples.

The report branch starts exactly at `TEST_COMMIT`. The intended base-to-tip
diff is exactly this report file. Generated binaries, snapshot artifacts,
private logs, vendor homes, repositories, corpus data, and retained reversible
fixtures remain outside the report worktree. No product file is modified.

PHASE3-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.3.0-rc.5
test_commit=2cbf0bba06497a4dad039a764ee1ea0d2199cda7
installed_binary_sha256=551f3156f5e06dfc8d2df4e133c2667e9c0e3d0bed6f4f9baf43d3cd7fe26cd4
required_pass=14
required_partial=0
required_fail=18
required_not_tested=0
optional_physical_pass=0
optional_physical_not_tested=2
baseline_provenance=FAIL
workspace_git=FAIL
agent_compatibility=FAIL
capability_privacy=FAIL
resume_fork=FAIL
picker=FAIL
performance=FAIL
phase1_phase2_regression=PASS
release_blocking_findings=9
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
