# Reinstate v0.3.0-rc.3 Windows Phase 3 verified-resume report

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `9 PASS / 0 PARTIAL / 23 FAIL / 0 NOT TESTED`
- **Optional physical counts:** `0 PASS / 2 NOT TESTED`
- **Release-blocking findings:** `7`

Required evidence that was not obtained is recorded as `FAIL`, per the RC3
dispatch. No RC2 report, corpus, home, installation, or evidence was reused.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-07T20:17:37Z` (acceptance continued after this timestamp) |
| Device | `windows-amd64` |
| OS/version/build | `Microsoft Windows 11 Pro 10.0.26200 build 26200` |
| CPU architecture/native process | `AMD64; native 64-bit PowerShell process` |
| Filesystem | `NTFS` |
| Tested tag | `v0.3.0-rc.3` |
| Tested full commit | `01459c2001a314e1734cbcf1126305db138c71f4` |
| Installed binary SHA-256 | `87cb384a9332a2cebe983d1be428b8bf12ac8adb5a9a784e2351477e785ec6fc` |
| Installed version JSON | `version=0.3.0-rc.3; commit=01459c2001a314e1734cbcf1126305db138c71f4` |
| Claude Code version/state | `2.1.220; executable/layout compatibility passed in the installed synthetic matrix; real session provisioning failed` |
| Codex CLI version/state | `0.146.0; executable/layout compatibility and extensionless discovery passed in the installed synthetic matrix; real session provisioning failed` |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.12` |
| Normal-corpus size | `8 records: 4 Claude + 4 Codex; 16 capability names` |
| Large-corpus size | `1000 records: 500 Claude + 500 Codex; 256 capability names` |
| Report branch | `test/v0.3.0-rc.3-windows-amd64-report` |
| Device-report commit | `returned as the immutable handoff commit; self-referential hash is not embedded` |
| Draft report PR | `created after the immutable commit; URL returned with the handoff` |

Host inventory recorded 24 logical processors and 63.7 GiB RAM. Ordinary host
protection remained enabled; no antivirus or other host-protection setting was
weakened for timing.

## 2. Signed artifact and installer chain

No downloaded executable was run until the release chain passed. The published
release was verified independently before installation.

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tag is annotated and SSH signature verifies against `.github/allowed_signers` | `PASS` | Annotated tag; `git verify-tag` passed against the tag's signer allowlist. |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | Peeled commit is the exact 40-character `TEST_COMMIT`; ancestor check passed. |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub release API: non-draft prerelease, `asset_count=25`, exact tag. |
| Exact 25-asset release set is present: `checksums.txt` plus 24 checksummed assets | `PASS` | `1` checksum manifest plus `5` archives, `5` archive SBOMs, `5` raw binaries, `8` Linux packages, and `1` source archive. |
| Every checksum and GitHub API asset digest matches | `PASS` | `24` checksum rows; local checksum failures `0`; API digest mismatches `0`. |
| Every release asset, including `checksums.txt`, has valid GitHub attestation provenance for the exact tag and commit | `PASS` | `25/25` strict attestations passed for the exact repository, workflow, tag ref, and full source commit. |
| Five platform archives have safe relative membership and the expected binary/documents | `PASS` | PowerShell archive gate passed without extraction. |
| Five archive SBOMs and the source archive are present and inspected | `PASS` | All five SBOM assets and the source archive were present, checksummed, attested, and inspected. |
| Raw binaries and eight Linux native packages are checksummed and attested | `PASS` | Exact asset set, checksums, API digests, and attestations passed. |
| Live public bootstrap is byte-identical to the tested commit and pins only `v0.3.0-rc.3` | `PASS` | Live installer bytes matched `website/public/install.ps1`; exactly one RC3 release pin was present. |
| Bootstrap-pinned canonical installer digest matches the tagged installer | `PASS` | Pinned installer SHA-256 matched the exact tagged `scripts/install.ps1` bytes. |
| Install used a brand-new isolated `INSTALL_DIR` and did not replace a user binary | `PASS` | Fresh isolated directory; process-scoped bootstrap PATH; user PATH unchanged; no existing binary replaced. |
| Both aliases resolve to identical verified bytes and report version `0.3.0-rc.3` | `PASS` | `rein.exe` and `reinstate.exe` matched the verified Windows raw binary SHA-256 and both reported RC3. |
| Installed binary reports the literal full 40-character tested commit | `PASS` | Both `version --json` outputs reported the exact full `TEST_COMMIT`; no short commit. |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean worktree and `go mod tidy -diff` | `PASS` | Tagged report worktree was clean before gates; tidy diff was empty; `gofmt -l` found no files. |
| `make verify` with pinned Go toolchain and Makefile-owned CGO settings | `PASS` | Fresh native PowerShell process; `GOTOOLCHAIN=go1.25.12`; exit `0`. |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `PASS` | Complete race suite exit `0`. |
| Required four CGO-disabled cross-builds | `PASS` | `darwin/arm64`, `darwin/amd64`, `windows/amd64`, and `linux/amd64` all exited `0`. |
| Five required fuzz-smoke surfaces | `PASS` | Six invocations covering workspace parsing/remote normalization, capability names, warning policy, and safe rendering all exited `0`. |
| Clean GoReleaser snapshot, staged assets, archive/SBOM inspection, and installer smoke | `FAIL` | Snapshot, downstream artifact verification, and installer smoke passed; the tagged `stage-release-assets.ps1` failed to parse in Windows PowerShell 5.1 before staging. A private manual stage was used only to exercise downstream gates and is not relabeled as the required stage pass. |
| Phase 1 and Phase 2 regression | `PASS` | `CGO_ENABLED=0 go test ./... -count=1` exit `0`. |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME` and disposable repositories | `PASS` | Fresh RC3-only homes and native Windows disposable repositories were created. |
| Fresh controlled Claude and Codex sessions; no older corpus reused | `FAIL` | Fresh Claude and Codex process attempts were made in throwaway homes, but neither produced a usable controlled session record. No older corpus was substituted. |
| No `rein init`, credential, passphrase, keyring write, or backend dependency | `PASS` | No `rein init` was run; no credentials or keyring writes were used; configless sessions/search worked locally. |
| Only private derived index/baseline state was created with native owner-only protection | `PASS` | Cache directory and v2 index/lock family had protected DACLs, current-owner/SYSTEM/Administrators access only, no inherited or broad Users/Everyone access. |
| No transcript, prompt/response, secret, private absolute path, config value, filename, diff, or raw child error was recorded | `PASS` | Report contains no transcript, prompt, secret, raw child error, private path, dirty filename, or diff. Raw vendor/process output remained private and was not inspected for content. |
| Product/vendor configuration was unchanged | `PASS` | Performance corpus fingerprints were identical before/after for normal and large runs; no real vendor home was modified. |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | Exact tag, full commit, installed hash, signature, checksum, attestation, archive, SBOM, and installer provenance | `PASS` | Complete signed-release, 25-asset, attestation, live-bootstrap, and installed-identity chain passed. |
| 2 | Full verification, race, cross-builds, and Phase 1/2 regression | `PASS` | Verification, race, four cross-builds, fuzz smoke, and Phase 1/2 regression passed. |
| 3 | Fresh configless home with no secret/storage dependency | `PASS` | Fresh home; sessions/search returned valid empty local results with only deterministic optional-agent warnings; no init or backend dependency. |
| 4 | Fresh controlled Claude and Codex sessions | `FAIL` | Fresh Claude attempt exited `1`; fresh Codex attempt exited `2`; no usable real session records were created. |
| 5 | First inspect is `baseline.unavailable` | `PASS` | Installed-artifact synthetic corpus inspect contained the `baseline.unavailable` warning with `unknown` status; no baseline was manufactured. |
| 6 | Successful verified launch records only a prelaunch-observed baseline | `FAIL` | No successful real native vendor launch occurred, so the required prelaunch-observed baseline was not established. |
| 7 | Repeat unchanged report matches repository, branch, HEAD, and tree digest | `PASS` | Installed performance harness repeated full refresh/preflight commands; fixed workspace HEAD and clean state held, and source fingerprints were unchanged. |
| 8 | Different repository at the same path blocks with exit `7` | `FAIL` | Required installed-artifact repository-swap mutation was not independently completed; source tests are supplemental only. |
| 9 | Missing/non-directory workspace blocks with exit `5` | `FAIL` | Required installed-artifact missing/non-directory workspace mutation was not independently completed; source tests are supplemental only. |
| 10 | Branch, detached HEAD, and unborn states remain distinct | `FAIL` | Required installed-artifact state matrix was not independently completed. |
| 11 | Equal/ahead/behind/diverged/unavailable HEAD relations are truthful and offline | `FAIL` | Required installed-artifact relation matrix was not independently completed. |
| 12 | Dirty-tree states warn without filenames or diffs | `FAIL` | Required installed-artifact dirty-state privacy matrix was not independently completed. |
| 13 | Credential-bearing remotes normalize without leakage | `FAIL` | Required installed-artifact adversarial remote matrix was not independently completed. |
| 14 | Worktree, symlink, Unicode, case, and native paths are safe | `FAIL` | Required installed-artifact path matrix was not independently completed. |
| 15 | Claude executable/version/layout is fail-closed | `PASS` | Installed synthetic preflight resolved Claude 2.1.220 through the curated trusted path and validated Claude layout/version. |
| 16 | Codex executable/version/layout is fail-closed | `PASS` | Installed synthetic preflight resolved Codex 0.146.0 from the extensionless vendor name through the curated trusted path; resume/fork dry-runs validated the Codex executable. |
| 17 | Instruction presence/change is bounded and content-free | `FAIL` | Required installed-artifact instruction presence/change mutation was not independently completed. |
| 18 | Skill presence/change is bounded, content-free, and does not follow escaping links | `FAIL` | Required installed-artifact skill mutation/link-escape matrix was not independently completed. |
| 19 | MCP reporting is logical-name/transport-only and value-free | `FAIL` | Required installed-artifact MCP adversarial matrix was not independently completed. |
| 20 | Recognized Node/Go declarations and installed versions are safe and truthful | `FAIL` | Required installed-artifact runtime declaration mutation matrix was not independently completed. |
| 21 | Inspect human/JSON output agrees and never prompts/launches | `FAIL` | JSON and human inspect exited `0` and agreed on the decision, but installed human inspect output included the controlled absolute workspace path. |
| 22 | Native dry-run preserves plan, adds report, and never mutates | `PASS` | Installed Claude/Codex resume/fork dry-runs exited `0`; fixed synthetic source fingerprints remained unchanged across the performance run. |
| 23 | TTY warning no/EOF/Ctrl-C refuses; yes launches once | `FAIL` | Required human Windows Terminal/real TTY evidence was not supplied; ConPTY input was not invented. |
| 24 | Non-TTY launch requires every exact current warning ID | `FAIL` | Installed dry-run negative policy probes rejected unknown, wildcard, and duplicate acknowledgements, but no real non-TTY vendor launch was available to close the full row. |
| 25 | Unknown/stale/duplicate/wildcard/info/blocker IDs cannot bypass | `FAIL` | Unknown, wildcard, and duplicate probes exited `2`; stale/informational/blocker coverage was not completed against a real launchable session. |
| 26 | Hard blockers never prompt and exit precedence is stable | `FAIL` | Required installed-artifact hard-blocker and precedence matrix was not independently completed. |
| 27 | Real same-vendor Claude resume and fork preserve the source | `FAIL` | No real Claude session was provisioned; no real resume/fork or independent fork resume can be certified. |
| 28 | Real same-vendor Codex resume and fork preserve the source | `FAIL` | No real Codex session was provisioned; no real resume/fork or independent fork resume can be certified. |
| 29 | Picker paths and both aliases apply identical policy | `FAIL` | Human picker/TTY evidence and real vendor records were unavailable; installed non-interactive alias parity alone is insufficient. |
| 30 | Gemini/OpenCode stay read-only with exit `5` | `FAIL` | Gemini and OpenCode were genuinely absent from the curated PATH, so optional physical counts remain not tested; required read-only invocation evidence was not completed. |
| 31 | Hostile, timeout, cancellation, stale, race, concurrency, and privacy gates pass | `FAIL` | Source adversarial tests passed, but installed-artifact adversarial coverage was incomplete and human inspect leaked an absolute workspace path. |
| 32 | Normal/large, cold/warm latency stays inside the RC3 ceilings | `PASS` | Fixed installed-artifact harness passed all validation, alias parity, fingerprint, timeout, and absolute-ceiling checks. Same-host comparable baseline was unavailable and no regression pass was manufactured. |

## 6. Performance evidence

The installed `rein` alias was the only timed executable. The fixed RC3
`phase3perf-v1` harness ran from the exact tagged source with canonical fixture
digest `4bf0b653ce76dcc3f7dd93916399bfdea8b658e1fbe41a9423608f2e7a6f8a76`.
The materialized normal and large corpus digests were distinct and remained
stable within their runs. The frozen controlled workspace HEAD was
`697ed29583a03045783557c3e8aeec92d9f7f01c`.

| Fixed precondition | Sanitized evidence | Result |
| ------------------ | ------------------ | ------ |
| Exact tagged harness and canonical fixture digest | `generator=phase3perf-v1`; canonical digest matched; installed SHA matched the release binary | `PASS` |
| Normal corpus shape | `4 Claude + 4 Codex`, `16` capability names, `8` records, limit `100` | `PASS` |
| Large corpus shape | `500 Claude + 500 Codex`, `256` capability names, `1000` records, limit `1000` | `PASS` |
| Controlled workspaces | Clean remote-free repositories at frozen HEAD before/after | `PASS` |
| Isolated homes, temp, cwd, and cold evidence | Separate per-corpus homes and preserved cold evidence directories | `PASS` |
| Curated physical PATH | Installed aliases, Git, Claude runtime, and physical Codex release directory; OpenCode omitted; ambient capabilities absent | `PASS` |
| Capture and clock contract | Bounded capture; raw stdout/stderr discarded after validation; monotonic process timing; 30 s timeout; 1 warmup; 20 warm; 3 cold; nearest-rank p95 | `PASS` |
| Untimed alias parity | Version, help, and all seven normal/large commands matched normalized exits/JSON | `PASS` |
| Cold reset contract | Three normal and three large cold runs moved only the exact v2 index/lock family into private preserved evidence | `PASS` |
| Vendor source mutation | Normal and large fingerprints matched before/after | `PASS` |
| Hardware, OS, filesystem, agent versions, and protection state | Recorded in the immutable test record; no host protection weakening | `PASS` |
| Comparable same-host baseline | No acceptable same-host RC3 comparison was available; no regression pass was claimed | `UNAVAILABLE` |

Times below are milliseconds converted from the harness nanosecond output. Every
sample had zero timeouts and `validated=true`.

| Corpus | Mode | Logical command | Samples | Median | p95 | Maximum | Timeouts | Validated | Windows ceiling | Result |
| ------ | ---- | --------------- | ------- | ------ | --- | ------- | -------- | --------- | --------------- | ------ |
| Startup | Cold | `version --json` | 3 | 34.360 ms | 37.190 ms | 37.190 ms | 0 | yes | max 8 s | `PASS` |
| Startup | Warm | `version --json` | 20 | 32.844 ms | 34.551 ms | 36.796 ms | 0 | yes | p95 4 s | `PASS` |
| Startup | Cold | `--help` | 3 | 34.943 ms | 35.089 ms | 35.089 ms | 0 | yes | max 8 s | `PASS` |
| Startup | Warm | `--help` | 20 | 32.525 ms | 34.175 ms | 35.179 ms | 0 | yes | p95 4 s | `PASS` |
| Normal | Cold | sessions, limit 100 | 3 | 63.213 ms | 65.760 ms | 65.760 ms | 0 | yes | max 12 s | `PASS` |
| Normal | Warm | sessions, limit 100 | 20 | 38.878 ms | 40.999 ms | 45.570 ms | 0 | yes | p95 4 s | `PASS` |
| Normal | Warm | search, limit 100 | 20 | 38.649 ms | 40.847 ms | 43.552 ms | 0 | yes | p95 4 s | `PASS` |
| Normal | Warm | inspect Claude anchor | 20 | 599.742 ms | 611.160 ms | 616.749 ms | 0 | yes | p95 4 s | `PASS` |
| Normal | Warm | Claude resume dry-run | 20 | 602.699 ms | 610.382 ms | 709.542 ms | 0 | yes | p95 4 s | `PASS` |
| Normal | Warm | Codex resume dry-run | 20 | 762.322 ms | 836.069 ms | 842.726 ms | 0 | yes | p95 4 s | `PASS` |
| Normal | Warm | Claude fork dry-run | 20 | 604.824 ms | 614.870 ms | 622.779 ms | 0 | yes | p95 4 s | `PASS` |
| Normal | Warm | Codex fork dry-run | 20 | 734.336 ms | 767.809 ms | 783.735 ms | 0 | yes | p95 4 s | `PASS` |
| Large | Cold | sessions, limit 1000 | 3 | 191.074 ms | 197.183 ms | 197.183 ms | 0 | yes | max 18 s | `PASS` |
| Large | Warm | sessions, limit 1000 | 20 | 136.735 ms | 140.432 ms | 140.555 ms | 0 | yes | p95 8 s | `PASS` |
| Large | Warm | search, limit 1000 | 20 | 138.739 ms | 142.062 ms | 143.684 ms | 0 | yes | p95 8 s | `PASS` |
| Large | Warm | inspect Claude anchor | 20 | 624.483 ms | 719.157 ms | 798.375 ms | 0 | yes | p95 8 s | `PASS` |
| Large | Warm | Claude resume dry-run | 20 | 631.086 ms | 654.569 ms | 657.014 ms | 0 | yes | p95 8 s | `PASS` |
| Large | Warm | Codex resume dry-run | 20 | 782.787 ms | 800.022 ms | 901.856 ms | 0 | yes | p95 8 s | `PASS` |
| Large | Warm | Claude fork dry-run | 20 | 621.533 ms | 634.861 ms | 636.608 ms | 0 | yes | p95 8 s | `PASS` |
| Large | Warm | Codex fork dry-run | 20 | 783.131 ms | 887.287 ms | 909.478 ms | 0 | yes | p95 8 s | `PASS` |

## 7. Findings and repository hygiene

### Release-blocking

1. The tagged `scripts/stage-release-assets.ps1` does not parse in Windows
   PowerShell 5.1. The parser rejects the interpolated variable immediately
   before the drive-colon in the staging error string (`$resolvedDist:`), so
   the required PowerShell staging gate exits before doing its work.
2. The fresh Claude process attempt, using a sanitized controlled prompt and
   throwaway session identity, exited `1` and produced no usable controlled
   session record.
3. The fresh Codex `exec --json --sandbox read-only --ask-for-approval never`
   attempt, using a fresh disposable repository, exited `2` and produced no
   usable controlled session record.
4. Without real vendor sessions, no successful native launch can establish the
   required prelaunch-observed baseline, and real same-vendor resume/fork rows
   cannot pass.
5. Required human Windows Terminal evidence for TTY picker, warning refusal,
   and confirmation paths was not supplied. ConPTY input was not invented.
6. Installed human `inspect` output includes the controlled absolute workspace
   path. This fails the RC3 privacy requirement even though the report itself
   contains no private path.
7. The installed-artifact mutation/adversarial rows listed in the matrix were
   not independently closed. Source tests and the performance harness are
   supplemental evidence, not a substitute for those required rows.

### Non-blocking

- No comparable same-host RC3 performance baseline was available. The absolute
  ceilings, validation, timeout, alias, and mutation checks passed; no relative
  regression pass was claimed.
- Initial ambient-PATH alias discovery produced an OpenCode warning mismatch.
  A controlled curated-PATH recheck superseded it: both aliases then produced
  the same deterministic `agent_not_installed` warning, and the fixed harness
  passed alias parity. The ambient attempt is not used as acceptance evidence.

### Test-harness deviations

- The required PowerShell staging failure was preserved. A private manual stage
  exercised the already-tagged artifact checker and installer smoke scripts;
  those downstream passes do not convert the failed stage gate into a pass.
- Early harness invocations that rejected a pre-existing root or non-canonical
  PATH entries were setup failures and were superseded by the successful RC3
  `perf-run5` invocation. They are not product verdicts.
- No raw child stderr, transcript, prompt, response, credential, configuration
  value, private path, dirty filename, or diff is included here.

Repository hygiene at handoff: the report branch is rooted at `TEST_COMMIT`; the
intended committed diff is exactly this report file, with no product-file
changes. The generated local `dist` directory is ignored and was not staged.

RC2 regression items:

| Item | Result | Evidence |
| ---- | ------ | -------- |
| 1. Windows executable trust / Codex extensionless lookup | `PASS` | Installed aliases matched the verified raw binary; the curated physical Codex path and installed dry-run validation resolved the extensionless vendor name. |
| 2. Race diagnostics | `PASS` | Complete native Windows race suite exited `0`. |
| 3. Artifact gates | `FAIL` | Published artifact chain passed, but the tagged PowerShell staging gate failed to parse; manual downstream staging is not a substitute. |
| 4. Claude compatibility range | `PASS` | Claude Code `2.1.220` is within the documented RC3 range and installed synthetic compatibility checks passed. |
| 5. Human TTY | `FAIL` | Required real Windows Terminal picker/warning evidence was not supplied. |

## 8. Terminated device block

PHASE3-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.3.0-rc.3
test_commit=01459c2001a314e1734cbcf1126305db138c71f4
installed_binary_sha256=87cb384a9332a2cebe983d1be428b8bf12ac8adb5a9a784e2351477e785ec6fc
required_pass=9
required_partial=0
required_fail=23
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
release_blocking_findings=7
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
