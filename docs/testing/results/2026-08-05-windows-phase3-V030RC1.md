# Phase 3 verified-resume report — native Windows x64

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `9 PASS / 0 PARTIAL / 23 FAIL / 0 NOT TESTED`
- **Optional physical counts:** `0 PASS / 2 NOT TESTED`
- **Release-blocking findings:** `6`

Required evidence that could not be completed is recorded as `FAIL`, never as
`NOT TESTED`. This is an artifact-certification result for RC1, not a stable
promotion decision.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-05T20:25:25Z` |
| Device | `windows-amd64` |
| OS/version/build | `Windows native x64; Windows PowerShell Desktop 5.1.26100.8328` |
| CPU architecture/native process | `AMD64; Is64BitProcess=True; Is64BitOS=True` |
| Filesystem | `NTFS; private mount/path withheld` |
| Tested tag | `v0.3.0-rc.1` |
| Tested full commit | `f6bdbaae83b8fc2b4ce769787c8fc1bba6c79bc6` |
| Installed binary SHA-256 | `7bf4e4e762f5a3cc8b59f8fb9c2b446f9ceb510a5841b26bd09f834e7a40f398` |
| Installed version JSON | `version=0.3.0-rc.1; commit=f6bdbaae83b8fc2b4ce769787c8fc1bba6c79bc6; date=2026-08-05T04:07:21Z` |
| Claude Code version/state | `2.1.220; executable/layout/version preflight passed in the fresh Claude probe; native session creation failed` |
| Codex CLI version/state | `0.146.0; installed version is in range, but Windows executable/layout/version preflight blocked` |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.12 windows/amd64` |
| Normal-corpus size | `8 records; 16 capability names; limit 100` |
| Large-corpus size | `1000 records; 256 capability names; limit 1000` |
| Report branch | `test/v0.3.0-rc.1-windows-amd64-report` |
| Device-report commit | `returned in the immutable handoff; self-reference is intentionally not rewritten after commit` |
| Draft report PR | `created after the immutable commit; URL returned in the handoff` |

## 2. Signed artifact and installer chain

No downloaded executable ran before the release identity, checksums, API
digests, attestations, archive membership, SBOMs, and installer provenance
passed.

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tag is annotated and SSH signature verifies against `.github/allowed_signers` | `PASS` | Annotated tag object `b4c53ed9a9ee0b2af6dd4c855ba4dfd8437f8230`; native `git verify-tag` with SSH format and the tagged allowlist passed. |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | Peeled commit is `f6bdbaae83b8fc2b4ce769787c8fc1bba6c79bc6`; ancestor check against `origin/main` exited `0`. |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub release is published, `draft=false`, `prerelease=true`, and tied to `v0.3.0-rc.1`. |
| Exact 25-asset release set is present: `checksums.txt` plus 24 checksummed assets | `PASS` | Exact set count `25`: checksums file, five platform archives, five archive SBOMs, five raw binaries, eight Linux packages, and one source archive. |
| Every checksum and GitHub API asset digest matches | `PASS` | `24/24` checksums match; all `25/25` downloaded bytes match the GitHub API SHA-256 digests. `checksums.txt` digest: `1f2e3a6211ede9fcc1956bc013ea09380dc1073023b4a197e5c2819be2b33fb1`. |
| Every release asset, including `checksums.txt`, has valid GitHub attestation provenance for the exact tag and commit | `PASS` | `25/25` attestations verify against the exact repository, `release.yml` signer workflow, `refs/tags/v0.3.0-rc.1`, full source digest, SLSA provenance predicate, and one subject per asset. |
| Five platform archives have safe relative membership and the expected binary/documents | `PASS` | All five archives inspected without extraction; safe relative members only, expected platform binary plus `LICENSE`, `NOTICE`, `README.md`, and `CHANGELOG.md`. |
| Five archive SBOMs and the source archive are present and inspected | `PASS` | All five SBOM JSON files parsed; source archive membership inspected for required source layout. |
| Raw binaries and eight Linux native packages are checksummed and attested | `PASS` | All thirteen assets are in the exact set, checksum-valid, API-digest-valid, and attested. |
| Live public bootstrap is byte-identical to the tested commit and pins only `v0.3.0-rc.1` | `PASS` | Live `install.ps1` equals tagged `website/public/install.ps1`; installer-script SHA-256 is `c34f2d4b1439e84d9c46fd329aca6628ef865c92ea99235982beed04b778f356`; exactly one RC pin. |
| Bootstrap-pinned canonical installer digest matches the tagged installer | `PASS` | Pinned SHA-256 `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` equals the exact tagged `scripts/install.ps1` bytes. |
| Install used a brand-new isolated `INSTALL_DIR` and did not replace a user binary | `PASS` | Live bootstrap used a nonexistent isolated install directory; `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; persistent User/Machine PATH unchanged. |
| Both aliases resolve to identical verified bytes and report version `0.3.0-rc.1` | `PASS` | `rein.exe` and `reinstate.exe` are byte-identical; both version JSON outputs report `0.3.0-rc.1` and the full tested commit. |
| Installed binary reports the literal full 40-character tested commit | `PASS` | Both aliases report literal `f6bdbaae83b8fc2b4ce769787c8fc1bba6c79bc6`; no short commit. |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean worktree and `go mod tidy -diff` | `PASS` | Exact tagged source worktree remained clean before and after automated gates. |
| `make verify` with pinned Go toolchain and Makefile-owned CGO settings | `PASS` | Fresh native PowerShell process, `GOTOOLCHAIN=go1.25.12`, inherited `CGO_ENABLED` removed; exit `0`. |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | Exit `1`; sanitized package-level failure is `github.com/HarjjotSinghh/reinstate/internal/cli`. Original failure preserved; no raw child error or transcript recorded. |
| Required four CGO-disabled cross-builds | `PASS` | Darwin arm64, Darwin amd64, Windows amd64, and Linux amd64 all exited `0` with nonzero binaries. |
| Five required fuzz-smoke surfaces | `PASS` | Six checked-in fuzz targets covering five required surfaces; all exited `0` with `-fuzztime=5s`. |
| Clean GoReleaser snapshot, staged assets, archive/SBOM inspection, and installer smoke | `FAIL` | GoReleaser was absent; `make snapshot` exited `2`; staging exited `1` without `artifacts.json`. Native PowerShell release and installer checks passed independently; the required clean snapshot/staging gate is missing. |
| Phase 1 and Phase 2 regression | `PASS` | `CGO_ENABLED=0 go test ./... -count=1` exited `0`; tracked source remained clean. |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME` and disposable repositories | `PASS` | Fresh native-Windows homes and disposable Git repositories were created for behavior and performance evidence; no Phase 2 or WSL corpus was reused. |
| Fresh controlled Claude and Codex sessions; no older corpus reused | `FAIL` | Codex produced a fresh discoverable record after a nonzero vendor command; Claude’s fresh native command exited `1` and produced no discoverable session. |
| No `rein init`, credential, passphrase, keyring write, or backend dependency | `PASS` | `rein init` was never run; no Reinstate storage/backend or secret input was configured. |
| Only private derived index/baseline state was created with native owner-only protection | `PASS` | Derived cache DACLs for the cache directory, database, and both lock files are protected, non-inherited, allow-only SYSTEM/Administrators/current-user SIDs, and passed the owner-only validator. |
| No transcript, prompt/response, secret, private absolute path, config value, filename, diff, or raw child error was recorded | `PASS` | Controlled sentinel scan found zero output leaks and zero private-index leaks; raw vendor/error material stayed outside the report. |
| Product/vendor configuration was unchanged | `PASS` | No Reinstate-caused vendor mutation was observed; operator-created fixtures were confined to disposable acceptance roots and excluded from the report. |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | Exact tag, full commit, installed hash, signature, checksum, attestation, archive, SBOM, and installer provenance | `PASS` | Complete signed-artifact chain in section 2 passed. |
| 2 | Full verification, race, cross-builds, and Phase 1/2 regression | `FAIL` | Race gate exited `1`; complete row cannot pass despite cross-build, fuzz, and regression passes. |
| 3 | Fresh configless home with no secret/storage dependency | `PASS` | Fresh isolated home; no init, secret, keyring, or backend operation. |
| 4 | Fresh controlled Claude and Codex sessions | `FAIL` | Claude native session creation exited `1`; only Codex yielded a discoverable fresh record. |
| 5 | First inspect is `baseline.unavailable` | `PASS` | First fresh inspect reported `baseline.unavailable` as an unknown warning; no baseline was manufactured. |
| 6 | Successful verified launch records only a prelaunch-observed baseline | `FAIL` | No verified native launch completed, so no valid prelaunch-observed baseline exists. |
| 7 | Repeat unchanged report matches repository, branch, HEAD, and tree digest | `FAIL` | JSON determinism passed, but no successful launch baseline/tree digest was established and working-tree state remained uncertain. |
| 8 | Different repository at the same path blocks with exit `7` | `FAIL` | Same-path replacement inspect detected a changed HEAD, but required safety-exit-`7` launch evidence was unavailable; Codex compatibility blocked first. |
| 9 | Missing/non-directory workspace blocks with exit `5` | `PASS` | Non-directory replacement produced `workspace.available=missing/block`; native resume exited `5`, with no child launch observed. |
| 10 | Branch, detached HEAD, and unborn states remain distinct | `PASS` | Installed inspect distinguished nonempty branch state, detached state, and unborn state (`unknown` branch/HEAD). |
| 11 | Equal/ahead/behind/diverged/unavailable HEAD relations are truthful and offline | `FAIL` | Complete relation matrix was not established; the synthetic remote remained unavailable/unknown. |
| 12 | Dirty-tree states warn without filenames or diffs | `FAIL` | Staged, unstaged, untracked, and conflict fixtures were exercised without leakage, but the installed working-tree probe remained `unknown`; complete state evidence is absent. |
| 13 | Credential-bearing remotes normalize without leakage | `PASS` | Synthetic credential-bearing HTTPS remote was supplied; inspect emitted no raw URL or sentinel and retained only logical unknown repository metadata. |
| 14 | Worktree, symlink, Unicode, case, and native paths are safe | `FAIL` | Native path fixtures were isolated, but linked-worktree/symlink/case coverage was not completed through a successful same-vendor preflight. |
| 15 | Claude executable/version/layout is fail-closed | `PASS` | Fresh Claude probe reported executable, recognized layout, and supported version; `claude 2.1.220`. |
| 16 | Codex executable/version/layout is fail-closed | `FAIL` | Installed `codex 0.146.0` was not resolved by Windows executable trust; inspect reported executable/layout/version blockers and launch exited `5`. |
| 17 | Instruction presence/change is bounded and content-free | `FAIL` | Codex instruction fixture was detected by name-only diagnostics with no sentinel leakage; Claude-side presence/change proof is missing. |
| 18 | Skill presence/change is bounded, content-free, and does not follow escaping links | `FAIL` | Codex skill inventory was bounded to metadata with no sentinel leakage; complete two-vendor escaping-link/change proof is missing. |
| 19 | MCP reporting is logical-name/transport-only and value-free | `FAIL` | Codex MCP diagnostic and sentinel privacy scan passed; complete Claude/Codex logical-name/transport comparison is missing. |
| 20 | Recognized Node/Go declarations and installed versions are safe and truthful | `FAIL` | No complete two-vendor declaration/version matrix was established. |
| 21 | Inspect human/JSON output agrees and never prompts/launches | `PASS` | Repeated Codex JSON was deterministic after message normalization; human and JSON inspect exited `0`, included a decision, and did not launch a vendor. Claude’s corresponding probe also passed. |
| 22 | Native dry-run preserves plan, adds report, and never mutates | `FAIL` | Claude dry-run plan/report keys passed in the fresh probe; Codex resume/fork dry-runs failed closed with exit `5`, so the required both-vendor row fails. |
| 23 | TTY warning no/EOF/Ctrl-C refuses; yes launches once | `FAIL` | No safe native Windows PTY driver was available and no compatible Claude session existed; real TTY input evidence is missing. |
| 24 | Non-TTY launch requires every exact current warning ID | `FAIL` | Codex hard blockers masked warning-only policy paths; exact warning-only non-TTY evidence could not be established. |
| 25 | Unknown/stale/duplicate/wildcard/info/blocker IDs cannot bypass | `FAIL` | All identifier probes remained behind the Codex compatibility blocker; no warning-only authorization path was available. |
| 26 | Hard blockers never prompt and exit precedence is stable | `FAIL` | Codex compatibility blocker and missing-workspace exit `5` were observed without a prompt, but complete safety/runtime precedence evidence is missing. |
| 27 | Real same-vendor Claude resume and fork preserve the source | `FAIL` | Fresh Claude command exited `1`; no real resume/fork launch or independent fork could be certified. |
| 28 | Real same-vendor Codex resume and fork preserve the source | `FAIL` | Codex executable-trust blocker prevented both real resume and fork. |
| 29 | Picker paths and both aliases apply identical policy | `FAIL` | Alias identity and inspect parity passed, but real picker, cancel, interrupt, and TTY paths were not available. |
| 30 | Gemini/OpenCode stay read-only with exit `5` | `FAIL` | Gemini/OpenCode were not installed for optional evidence; ambient optional discovery was not reused as fresh corpus evidence. |
| 31 | Hostile, timeout, cancellation, stale, race, concurrency, and privacy gates pass | `FAIL` | Privacy sentinel scan and DACL checks passed, but complete hostile timeout/cancellation/path-race/concurrency evidence is absent. |
| 32 | Normal/large, cold/warm latency stays inside the RC1 ceilings | `PASS` | Frozen installed-artifact harness completed with validated samples, no timeouts, alias parity, clean workspaces, and unchanged vendor fingerprints; aggregates are in section 6. |

## 6. Performance evidence

The installed `rein` alias was the only timed executable. The frozen harness ran
from the exact tagged source and wrote raw results only to private evidence.

| Fixed precondition | Sanitized evidence | Result |
| ------------------ | ------------------ | ------ |
| Harness came from exact tagged source; generator `phase3perf-v1`; canonical digest and materialized digests recorded | `fixture_canonical_digest=4bf0b653ce76dcc3f7dd93916399bfdea8b658e1fbe41a9423608f2e7a6f8a76`; normal materialized `cfa591fcfeaf4ed07ceaf40b8430c3270d4c6f5e3afd1f793ebfd34f798b17dd`; large materialized `fcaac7d3d81607b06cafb5ac77286a2c344ba745dbdc20fc9f2f55b29f6fd3d3` | `PASS` |
| Normal corpus is exactly 4 Claude + 4 Codex records, 16 capability names, 4 events/2 messages/2 file refs per record, `NORMAL_LIMIT=100`, and both anchors visible | `8` records, `16` capability names, limit `100`; command/result validation passed | `PASS` |
| Large corpus is exactly 500 Claude + 500 Codex records, 256 capability names, 4 events/2 messages/2 file refs per record, `LARGE_LIMIT=1000`, and both anchors visible | `1000` records, `256` capability names, limit `1000`; command/result validation passed | `PASS` |
| Both controlled workspaces are clean remote-free `main` Git repositories at frozen HEAD `697ed29583a03045783557c3e8aeec92d9f7f01c` before and after | Both corpus workspaces reported clean and the frozen HEAD | `PASS` |
| Dedicated functional/baseline performance homes; isolated Reinstate/Claude/Codex/Gemini/process homes, temp, cwd, and cold evidence per corpus | Harness-created isolated roots; no ambient home reuse | `PASS` |
| Curated PATH entries/executables are canonical, physical, trusted, and outside source/evidence; OpenCode omitted; ambient capabilities absent; environment digest recorded | `environment_digest=49ead265c18f9ee964da74e1ecea18173a860acce1e5610d1d547e38e75af9ac`; OpenCode omitted; ambient capabilities absent | `PASS` |
| Capture is bounded and raw output discarded; monotonic process-start-to-exit clock, 30-second timeout, 1 warmup, 20 warm samples, 3 cold samples, nearest-rank p95 | Harness result: timeout `30000ms`, warm samples `20`, cold samples `3`, validated outputs | `PASS` |
| Untimed `rein`/`reinstate` parity for version JSON and normalized help, with exact version/full commit and required help surface | Alias bytes, version JSON, and normalized help parity passed | `PASS` |
| Untimed `rein`/`reinstate` exit and normalized-JSON parity for all seven commands, normal corpus | All seven command preconditions passed in the harness | `PASS` |
| Untimed `rein`/`reinstate` exit and normalized-JSON parity for all seven commands, large corpus | All seven command preconditions passed in the harness | `PASS` |
| Three normal and three large cold resets moved only the exact v2 index/lock/SQLite companion family into preserved private evidence | Three cold samples per corpus; exact family reset procedure validated | `PASS` |
| Controlled vendor fingerprints unchanged across normal timings | Before/after `b842e4d996c2000dd73dfd70ada9b269bd0f0ff80bae7994115aa61a51e98dc0` | `PASS` |
| Controlled vendor fingerprints unchanged across large timings | Before/after `137a46b9dc678229445d2562ad53c35d1e3638e3e3f15f30ea0f44674eaeee6f` | `PASS` |
| Hardware, OS/build, filesystem, agent versions, and generic antivirus state recorded | Native Windows x64, NTFS, Claude `2.1.220`, Codex `0.146.0`; ordinary host protections were not disabled or weakened | `PASS` |

| Corpus | Mode | Logical command | Samples | Median | p95 | Maximum | Timeouts | Validated | Windows RC1 ceiling | Result |
| ------ | ---- | --------------- | ------- | ------ | --- | ------- | -------- | --------- | ------------------- | ------ |
| Startup | Cold | `version --json`, fresh home | 3 | `39.214ms` | `43.367ms` | `43.367ms` | 0 | true | max `8s` | `PASS` |
| Startup | Warm | `version --json` | 20 | `39.729ms` | `44.583ms` | `47.255ms` | 0 | true | p95 `4s` | `PASS` |
| Startup | Cold | `--help`, fresh home | 3 | `42.926ms` | `44.607ms` | `44.607ms` | 0 | true | max `8s` | `PASS` |
| Startup | Warm | `--help` | 20 | `41.225ms` | `45.002ms` | `45.315ms` | 0 | true | p95 `4s` | `PASS` |
| Normal | Cold | `sessions --limit 100 --json` | 3 | `73.530ms` | `75.005ms` | `75.005ms` | 0 | true | max `12s` | `PASS` |
| Normal | Warm | `sessions --limit 100 --json` | 20 | `47.696ms` | `52.774ms` | `54.948ms` | 0 | true | p95 `4s` | `PASS` |
| Normal | Warm | `search`, limit 100 | 20 | `46.060ms` | `50.378ms` | `50.490ms` | 0 | true | p95 `4s` | `PASS` |
| Normal | Warm | `inspect` Claude anchor | 20 | `648.095ms` | `670.069ms` | `704.304ms` | 0 | true | p95 `4s` | `PASS` |
| Normal | Warm | Claude resume dry-run | 20 | `647.036ms` | `666.220ms` | `693.197ms` | 0 | true | p95 `4s` | `PASS` |
| Normal | Warm | Codex resume dry-run | 20 | `815.942ms` | `841.823ms` | `907.181ms` | 0 | true | p95 `4s` | `PASS` |
| Normal | Warm | Claude fork dry-run | 20 | `642.290ms` | `655.269ms` | `686.726ms` | 0 | true | p95 `4s` | `PASS` |
| Normal | Warm | Codex fork dry-run | 20 | `812.273ms` | `838.561ms` | `886.231ms` | 0 | true | p95 `4s` | `PASS` |
| Large | Cold | `sessions --limit 1000 --json` | 3 | `223.008ms` | `230.101ms` | `230.101ms` | 0 | true | max `18s` | `PASS` |
| Large | Warm | `sessions --limit 1000 --json` | 20 | `167.128ms` | `192.391ms` | `210.633ms` | 0 | true | p95 `8s` | `PASS` |
| Large | Warm | `search`, limit 1000 | 20 | `160.340ms` | `167.625ms` | `170.710ms` | 0 | true | p95 `8s` | `PASS` |
| Large | Warm | `inspect` Claude anchor | 20 | `711.612ms` | `720.887ms` | `791.101ms` | 0 | true | p95 `8s` | `PASS` |
| Large | Warm | Claude resume dry-run | 20 | `697.810ms` | `714.243ms` | `715.957ms` | 0 | true | p95 `8s` | `PASS` |
| Large | Warm | Codex resume dry-run | 20 | `872.322ms` | `885.646ms` | `886.390ms` | 0 | true | p95 `8s` | `PASS` |
| Large | Warm | Claude fork dry-run | 20 | `695.848ms` | `718.586ms` | `750.997ms` | 0 | true | p95 `8s` | `PASS` |
| Large | Warm | Codex fork dry-run | 20 | `873.506ms` | `945.422ms` | `955.699ms` | 0 | true | p95 `8s` | `PASS` |

## 7. Findings and repository hygiene

### Release-blocking

1. `CGO_ENABLED=1 go test -race ./... -count=1` exited `1` in `internal/cli`; the complete race gate is not green.
2. GoReleaser was not installed; `make snapshot` exited `2`, so the required clean snapshot and `artifacts.json` staging proof is absent.
3. The required shell release-artifact helper exited `127` because native host tools `unzip`, `sha256sum`, and `jq` were unavailable; independent native PowerShell/archive checks do not replace that required gate.
4. Fresh native Claude session creation exited `1` and yielded no discoverable session; real Claude resume/fork and TTY warning evidence therefore cannot pass.
5. Windows executable trust does not resolve the installed `codex.exe` because it probes the extensionless `codex` name; Codex inspect reports executable/layout/version blockers and resume/fork exit `5`.
6. Mandatory product rows dependent on successful same-vendor sessions and a real native TTY remain failed; missing evidence is a release blocker under the RC1 dispatch.

### Sanitized failed commands

| Command | Exit | Sanitized result |
| ------- | ---- | ---------------- |
| `CGO_ENABLED=1 go test -race ./... -count=1` | `1` | Failed package: `internal/cli`; raw child error intentionally not retained. |
| `make snapshot` | `2` | GoReleaser executable unavailable. |
| `scripts/check-release-artifacts.sh` | `127` | Required host helpers unavailable: `unzip`, `sha256sum`, `jq`. |
| `scripts/stage-release-assets.sh` | `1` | No GoReleaser `artifacts.json` because snapshot could not run. |
| `claude --print --output-format json ...` in a fresh config | `1` | No discoverable fresh Claude session. |
| `codex exec --json ...` in a fresh home | `1` | Fresh Codex record was discoverable, but the controlled vendor command itself failed. |
| `rein resume <fresh-codex-session>` | `5` | Native Codex compatibility blocker; no vendor launch. |

### Non-blocking

- The frozen installed-artifact performance harness passed every timing,
  validation, alias, workspace, and source-fingerprint gate.
- All exact release assets, attestations, installer provenance, installed
  identity, cross-builds, fuzz smoke, `make verify`, and Phase 1/2 regression
  gates passed.
- Gemini and OpenCode were not installed solely for optional evidence. Ambient
  optional discovery was not reused as fresh product evidence.

### Test-harness deviations

- An initial Codex setup attempt used a read-only PowerShell automatic variable
  name and was disqualified. The existing user directory was preserved; the
  corrected retry used a fresh isolated Codex home and no user content was
  inspected.
- No safe native Windows PTY driver was available in the controlled process;
  TTY rows are therefore failed rather than inferred from redirected input.

The report-only diff is exactly this Markdown path. No product files, tags,
releases, deployments, stable promotion, or source changes were made.

## 8. Required terminated device block

PHASE3-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.3.0-rc.1
test_commit=f6bdbaae83b8fc2b4ce769787c8fc1bba6c79bc6
installed_binary_sha256=7bf4e4e762f5a3cc8b59f8fb9c2b446f9ceb510a5841b26bd09f834e7a40f398
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
release_blocking_findings=6
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
