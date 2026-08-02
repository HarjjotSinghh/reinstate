# Phase 2 acceptance — native-Windows Device B (`v0.2.0-rc.2` tagged artifacts)

**Verdict:** `FAIL`
**Milestone:** `DEVICE_COMPLETE`
**Required counts:** `27 PASS / 0 PARTIAL / 5 FAIL / 0 NOT TESTED`
**Optional physical counts:** `2 PASS / 0 NOT TESTED`

This report covers only the exact disposable sessions and paths created for
this run. No real transcript content or secret was used as evidence. The
installed artifact and every executed non-interactive product gate passed, but
the required real Windows Terminal human-keyboard routes were not executable
under the available UI-automation policy. Missing evidence is `FAIL`.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-02T20:11:32Z` through `2026-08-02T20:56:29Z` |
| Device | native-Windows Device B |
| Tested Git commit | `8dd6073118131dd4ecacfca3eb1cde3f07df5eb6` |
| Signed tag | `v0.2.0-rc.2` (annotated, SSH-signed, verified) |
| Reinstate version JSON | `{"commit":"8dd6073118131dd4ecacfca3eb1cde3f07df5eb6","date":"2026-08-02T19:01:43Z","name":"reinstate","version":"0.2.0-rc.2"}` |
| OS/version/build | Windows 11 Pro `10.0.26200`, build `26200` |
| Architecture | native `windows/amd64`; OS and process both x64 |
| Native shell | Windows PowerShell `5.1.26100.8328` Desktop; never WSL |
| Claude Code version/state | `2.1.220`; installed; fresh controlled read/index path PASS; required human-keyboard resume/fork FAIL |
| Codex CLI version/state | `0.146.0`; installed; fresh controlled read/index path PASS; required human-keyboard resume/fork FAIL |
| Gemini CLI version/state | `0.53.0`; installed; fresh controlled read-only path PASS |
| OpenCode version/state | `1.18.2`; installed; fresh controlled read-only path PASS |
| Git version | `2.52.0.windows.1` |
| Go version | `go1.25.12 windows/amd64` under the required process-scoped `GOTOOLCHAIN` |
| Report branch | `test/v0.2.0-rc.2-windows-report` |
| Draft PR | pending creation after the first report-only commit |

## 1a. Release-candidate artifact record

| Field | Value |
| ----- | ----- |
| Tag object type | annotated (`git cat-file -t` returned `tag`) |
| Signature | verified against `.github/allowed_signers` at `TEST_COMMIT`; raw signer-key fingerprint omitted |
| Ancestry | `TEST_COMMIT` is an ancestor of `origin/main` |
| Release URL | https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.2.0-rc.2 |
| Release state | published, non-draft, prerelease, tied to `v0.2.0-rc.2` |
| Public installer SHA-256 | `29ED0CFE697FE5D367EB7877CFC8899F99FFB63B479E5339F7B6983DFC540899` |
| Public installer identity | live `https://reinstate.dev/install.ps1` was byte-identical to tagged `website/public/install.ps1` |
| Installer pin | exactly one version literal: `v0.2.0-rc.2` |
| Pinned downstream installer SHA-256 | `02C68984964556E7C685A275BDE72DC812162E0B898BE0F26718A0813EFC0DFE` |
| Installed binary SHA-256 | `D66807A27A76199CF05127A76274E83BCDA998777B06BC1F8CC71D4E9C9BFE01` |
| Installed alias pair | isolated `rein.exe` and `reinstate.exe`; byte-identical SHA-256 |
| Windows archive SHA-256 | `BB3CBCCC3ACCC03556ACF16962632675B1DFFC0C97C4652EE733BDEC98568436` |
| Checksums SHA-256 | `6F863279C5479670A87B5EBB88ACC626C61DB02E7BEF52362EFA370F2D9DECAB` |

### Artifact inventory and verification

The release contained exactly 12 named assets: `checksums.txt`, five platform
archives, five matching SBOMs, and the source archive.

| Asset | Checksum/API digest | Attestation | Membership/SBOM |
| ----- | ------------------- | ----------- | --------------- |
| `checksums.txt` | PASS | PASS | 11 non-self checksum entries |
| `reinstate_0.2.0-rc.2_darwin_amd64.tar.gz` | PASS | PASS | 5 safe members |
| `reinstate_0.2.0-rc.2_darwin_amd64.tar.gz.sbom.json` | PASS | PASS | SPDX 2.3; matches archive |
| `reinstate_0.2.0-rc.2_darwin_arm64.tar.gz` | PASS | PASS | 5 safe members |
| `reinstate_0.2.0-rc.2_darwin_arm64.tar.gz.sbom.json` | PASS | PASS | SPDX 2.3; matches archive |
| `reinstate_0.2.0-rc.2_linux_amd64.tar.gz` | PASS | PASS | 5 safe members |
| `reinstate_0.2.0-rc.2_linux_amd64.tar.gz.sbom.json` | PASS | PASS | SPDX 2.3; matches archive |
| `reinstate_0.2.0-rc.2_linux_arm64.tar.gz` | PASS | PASS | 5 safe members |
| `reinstate_0.2.0-rc.2_linux_arm64.tar.gz.sbom.json` | PASS | PASS | SPDX 2.3; matches archive |
| `reinstate_0.2.0-rc.2_source.tar.gz` | PASS | PASS | 922 safe members under one top-level directory |
| `reinstate_0.2.0-rc.2_windows_amd64.zip` | PASS | PASS | 5 safe members; one `reinstate.exe` |
| `reinstate_0.2.0-rc.2_windows_amd64.zip.sbom.json` | PASS | PASS | SPDX 2.3; matches archive |

- `checksums.txt`: `11/11` entries matched downloaded files.
- GitHub API asset digests: `12/12` matched local SHA-256 values.
- GitHub artifact attestations: `12/12` verified against
  `HarjjotSinghh/reinstate`; predicate type was SLSA provenance v1.
- Attestation negative controls were non-vacuous: a distinct-byte unattested
  file and a valid asset checked against the wrong repository both exited `1`.
- All six archives were listed before extraction or execution. No absolute,
  drive-prefixed, or parent-traversal member was present.
- Platform archives contained `CHANGELOG.md`, `LICENSE`, `NOTICE`, `README.md`,
  and one platform binary. The installer created the Windows alias pair.

### Installed-artifact chain of custody

1. The live public bootstrap returned HTTP `200` with zero redirects and was
   byte-identical to the tagged website file.
2. Its sole release literal was `v0.2.0-rc.2`; its downstream installer pin
   matched the live tagged and committed `scripts/install.ps1` three ways.
3. The bootstrap installed into a path that did not exist before the run, with
   persistent PATH updates disabled. A pre-existing user command remained
   outside the isolated directory and had a pre-run modification time.
4. The release ZIP matched `checksums.txt`, GitHub's asset digest, and its
   attestation before membership inspection and extraction.
5. Both installed executable hashes matched the independently extracted
   verified archive binary.
6. Both installed aliases returned exit `0`, identical version JSON, release
   version `0.2.0-rc.2`, and the literal full 40-character `TEST_COMMIT`.
7. A supplemental source build also reported the full commit, but it was not
   used as installed-artifact behavior evidence.

## 2. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches requested tag | PASS | `git rev-parse v0.2.0-rc.2^{}` returned the exact 40-character `TEST_COMMIT` |
| Binary reports tested commit | PASS | installed version JSON returned the literal full `8dd6073118131dd4ecacfca3eb1cde3f07df5eb6` |
| Product tree was clean before testing | PASS | dedicated worktree started clean at `TEST_COMMIT` |
| Report branch starts at tested commit | PASS | branch created directly from `TEST_COMMIT` |
| Report is the only committed change | PASS | enforced by explicit staging and base-to-tip validation before publication |
| No secret/transcript/private path committed | PASS | privacy scan and staged-diff inspection required before push |
| Tag commit CI state | PASS | 12 check runs completed: 11 success, 1 skipped, 0 failed/pending |

## 3. Isolation and local-only proof

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | PASS | canonical user-profile home was absent before first command |
| No `rein init` run | PASS | no initialization command executed |
| No `config.toml` or sync state created | PASS | only `cache/session-index-v1.sqlite` appeared |
| No credential/passphrase/keyring request | PASS | none observed across local-index commands |
| No backend/network dependency | PASS | local product commands used no sync/storage command or profile |
| Only derived index state created | PASS | canonical home contained only `cache/` and its SQLite index |
| Index and parent permissions are owner-only | PASS | current user, SYSTEM, and Administrators only; no broad allow principal |
| Corrupt derived-index rebuild | PASS | invalid replacement rebuilt to SQLite and restored all five controlled records |

An initial candidate home under a broad inherited test-root ACL was retired
before the matrix and contributes zero PASS evidence. It remains preserved as
harness evidence. The canonical fresh home used the runbook-prescribed
user-profile scope and passed the native ACL gate.

## 4. Controlled corpus

| Agent | Composite reference | Disposable project label | Marker found | Capability |
| ----- | ------------------- | ------------------------ | ------------ | ---------- |
| Claude | `claude:580473d0-8090-4f47-9bf4-8248881108f5` | `phase2-alpha` | true | full |
| Claude refresh/new | `claude:c93ec438-6b83-4ee3-bf4a-8e323480ca93` | `phase2-alpha` | true | full |
| Codex | `codex:019fc425-6155-7410-9c3e-713704f76795` | `phase2 unicode-beta` | true | full |
| Gemini | `gemini:44178423-e170-42ae-bcb2-d160e2d37d67` | `phase2-alpha` | true | read-only |
| OpenCode | `opencode:ses_03bd8aa57ffelg6aiJPiRL3Sqn` | `phase2 unicode-beta` | true | read-only |

Every record was created after a before/after vendor metadata snapshot. No
older development or RC1 record was reused. Reports contain no full prompt,
response, transcript excerpt, or absolute private path.

## 5. Automated verification

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| `make fmt-check` | PASS | exit `0` |
| Focused session-index/CLI/adapter tests | PASS | exact independent retry exit `0` |
| Full Go test suite | PASS | exact independent retry exit `0` |
| Required `make verify` | PASS | fresh PowerShell, process-scoped MSYS2 PATH, `GOTOOLCHAIN=go1.25.12`, inherited `CGO_ENABLED` removed; exit `0` |
| Required separate full race suite | PASS | `CGO_ENABLED=1 go test -race ./... -count=1`; exit `0` |
| `make vet` | PASS | exit `0` |
| Required four CGO-disabled cross-builds | PASS | Windows amd64, Darwin arm64/amd64, Linux amd64; all exit `0` |
| Phase 1 regression | PASS | complete suite inside `make verify` and independent full test exited `0` |
| Supplemental source identity | PASS | `v0.2.0-rc.2` plus full `TEST_COMMIT`; not substituted for installed artifact |

### Failed command attempts and exact reruns

The first two attempts below failed inside one batched harness while the
machine had roughly 690 active processes. Both were rerun independently in
fresh PowerShell processes with unchanged test arguments and passed. A later
doctest recheck accidentally omitted the required process-scoped MSYS2 PATH;
its exact targeted test and the full package were then rerun with the required
toolchain and passed.

| Command | First exit | Sanitized output | Independent retry |
| ------- | ---------- | ---------------- | ----------------- |
| `go test ./internal/sessionindex ./internal/cli ./internal/adapter/... -count=1` | `1` | `runtime.preemptM: duplicatehandle failed; errno=6`; Go runtime fatal error | exit `0`; all focused packages PASS |
| `go test ./... -count=1` | `1` | same Go 1.25.12 Windows `DuplicateHandle` runtime failure | exit `0`; all packages PASS |
| `go test ./internal/doctest -count=1` without the required MSYS2 PATH | `1` | `TestProductionDeploymentRejectsInvalidWebsiteTagDate`: unexpected invalid-date failure with empty child output | exact targeted test exit `0`, then full `./internal/doctest` package exit `0`, both with process-scoped MSYS2 PATH |

The mandated `make verify` and separate full race commands passed before this
batched retry incident. The incident is retained as a harness deviation, not
erased or reclassified as a product PASS on exit code alone.

## 6. Configless index and refresh

- Initial human and JSON listing exited `0` and found the exact four initial
  controlled references. The refreshed corpus contained five controlled
  references with no duplicate key.
- Ordering had zero violations under newest update, then agent, then native ID.
- Sequential alias captures differed only for two unrelated actively-written
  Codex records, and only in `size_bytes`/`updated_at`. All 98 stable records
  and every controlled record were equivalent. Controlled no-change refresh
  output was byte-equivalent after normalization.
- `rein list --help` remained the distinct Phase 1 local-agent listing surface;
  it was not relabeled as the canonical `sessions` command.
- A controlled Claude append kept one stable composite identity and became
  searchable after refresh. A later new Claude session became the newest
  controlled resumable record.
- Corrupt-index recovery preserved the original evidence copy, rebuilt a valid
  SQLite index, and rediscovered all five controlled records.
- Hash, size, and timestamp checks proved listing, search, and inspect changed
  zero of four controlled vendor source files; the OpenCode vendor JSON record
  also stayed byte-equivalent.

## 7. Search and inspect

- Prompt fragment: one exact Claude result in human and JSON paths.
- AND terms: three expected prompt-indexed vendor results.
- Agent filters: Claude `2`, Codex `1`, Gemini `1`, OpenCode `1`.
- Project and branch filters: two exact Claude records each.
- Structured file filter: one exact new Claude record with one real Read-tool
  file reference.
- Limit: exactly one newest controlled result.
- Case-insensitive, metacharacter, Unicode+AND, and zero-match checks all
  returned the expected bounded counts; zero match returned an honest empty
  result with exit `0`.
- Human and JSON inspect exited `0` for all four initial vendors. Claude,
  Codex, and Gemini user previews were exactly 160 code points or shorter;
  OpenCode had no prompt preview. No assistant text, reasoning, tool output,
  environment dump, auth content, or transcript body appeared.
- Claude/Codex advertised resume and fork. Gemini/OpenCode advertised neither.

## 8. Last, resume, and fork

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| Claude resume dry-run | PASS | exact `claude`, `--resume`, native ID, recorded cwd; exit `0`; no mutation |
| Codex resume dry-run | PASS | exact `codex`, `resume`, native ID, recorded cwd; exit `0`; no mutation |
| Claude fork dry-run | PASS | exact `claude`, `--resume`, native ID, `--fork-session`; exit `0`; no mutation |
| Codex fork dry-run | PASS | exact `codex`, `fork`, native ID; exit `0`; no mutation |
| `last` global | PASS | plan matched the newest resumable row in the same refreshed SQLite snapshot |
| `last --agent claude` | PASS | selected the newest controlled Claude session |
| `last --project` | PASS | selected the newest controlled project session |
| Unique bare ID | PASS | resolved to the exact controlled Claude composite reference |
| Missing reference | PASS | actionable failure, exit `2`, no launch |
| Missing executable | PASS | process-scoped PATH probe failed before launch, exit `5` |
| Ambiguous ID / missing workspace / child failure | PASS | deterministic injected gates passed in the exact full suite |
| Native JSON separation | PASS | resume, fork, and last without dry-run each refused with exit `2` |
| Claude real installed-binary resume | FAIL | required Windows Terminal human-keyboard route not executed |
| Codex real installed-binary resume | FAIL | required Windows Terminal human-keyboard route not executed |
| Claude real installed-binary fork | FAIL | required Windows Terminal human-keyboard route not executed |
| Codex real installed-binary fork | FAIL | required Windows Terminal human-keyboard route not executed |

The controlled append used Claude's documented non-interactive resume mode to
exercise refresh behavior. It is not substituted for any of the four required
real Windows Terminal human-keyboard rows.

## 9. Interactive switcher

**FAIL.** No real Windows Terminal human-keyboard picker interaction was
executed. The available Windows UI automation policy explicitly prohibited
automating terminal applications and Codex CLI, while byte-only ConPTY,
AppActivate, and SendKeys were forbidden by the dispatch. No prohibited
fallback was used.

The independently executable non-TTY half passed for both binary names:

- piped empty input exited `2` immediately;
- both outputs included the `rein sessions --json` hint; and
- output contained no controlled prompt marker or preview.

Filter, inspect, resume, fork, invalid input, cancel, EOF, and interrupt through
the real interactive picker remain missing evidence and therefore row 26 is
`FAIL`.

## 10. Read-only adapters

### Gemini CLI

- Fresh controlled session created normally with Gemini CLI `0.53.0`.
- Default sessions, agent-filter search, literal full-ID search, and inspect
  found the exact composite reference.
- Inspect advertised read-only capability; resume and fork each exited `5`.
- Refusal probes changed zero controlled vendor files.

### OpenCode

- Fresh controlled session created in the fresh controlled workspace with the
  explicitly listed `opencode/deepseek-v4-flash-free` model.
- Vendor `session list --format json` supplied top-level `updated` and
  `created`; both were non-year-1. Reinstate's whole-second `updated_at`
  matched vendor provenance.
- The record was visible in unfiltered default sessions from the same
  workspace, at default-list index 7, with correct non-year-1 ordering.
- Agent filter, literal full-ID search, and inspect found the exact record.
- Inspect advertised read-only capability; resume and fork each exited `5`.
- The vendor JSON record and timestamps remained unchanged across Reinstate
  read-only operations.

## 11. Mandatory matrix

| # | Gate | macOS | Windows |
| - | ---- | ----- | ------- |
| 1 | Exact tested commit/binary provenance | — | PASS — signed tag, release chain, installed full SHA (§1a–2) |
| 2 | Full local verification and required cross-builds | — | PASS — exact gates and retries (§5) |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | — | PASS (§3) |
| 4 | `rein sessions` discovers exact Claude sessions | — | PASS (§4, §6) |
| 5 | `rein sessions` discovers exact Codex sessions | — | PASS (§4, §6) |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | — | PASS — stable-source parity; live-source deviation recorded (§6) |
| 7 | Derived index path, rebuild, idempotency, and private permissions | — | PASS (§3, §6) |
| 8 | Prompt-fragment literal search | — | PASS (§7) |
| 9 | Agent filter | — | PASS (§7) |
| 10 | Project filter | — | PASS (§7) |
| 11 | Branch filter | — | PASS (§7) |
| 12 | File filter | — | PASS (§7) |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | — | PASS (§7) |
| 14 | `sessions` and `search` do not dump transcript passages | — | PASS (§6–7) |
| 15 | `inspect` metadata/160-code-point user preview policy | — | PASS (§7) |
| 16 | Append/new-session refresh and no-change idempotency | — | PASS (§6) |
| 17 | `last` selects the correct resumable session and filters | — | PASS (§8) |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | — | PASS (§8) |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | — | PASS (§8) |
| 20 | Claude native resume | — | FAIL — required human-keyboard route missing (§8, §12) |
| 21 | Codex native resume | — | FAIL — required human-keyboard route missing (§8, §12) |
| 22 | Claude vendor-native fork, source preserved | — | FAIL — required human-keyboard route missing (§8, §12) |
| 23 | Codex vendor-native fork, source preserved | — | FAIL — required human-keyboard route missing (§8, §12) |
| 24 | Missing/ambiguous reference and missing executor fail safely | — | PASS (§8) |
| 25 | JSON/native-child separation and child failure propagation | — | PASS (§5, §8) |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | — | FAIL — real Windows Terminal input missing (§9, §12) |
| 27 | Non-TTY prompt failure is immediate and actionable | — | PASS (§9) |
| 28 | Gemini read-only physical path, when installed | — | PASS (§10) |
| 29 | OpenCode read-only physical path, when installed | — | PASS (§10) |
| 30 | Read-only resume/fork refusal with exit `5` | — | PASS (§10) |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | — | PASS — exact automated suite (§5) |
| 32 | Phase 1 automated regression remains green | — | PASS (§5) |

## 12. Findings

### Release-blocking

1. **RC2-WIN-PHYSICAL-INPUT-MISSING:** Rows 20–23 and 26 lack the mandated
   real Windows Terminal human-keyboard evidence. The available UI automation
   route prohibited terminal/Codex automation, and the dispatch rejected
   byte-only ConPTY, AppActivate, and SendKeys. Device B tagged-artifact
   certification is therefore `FAIL`; stable promotion remains blocked.

No installed-artifact, configless-index, search/privacy, Gemini, OpenCode, or
automated product defect was observed in the gates that actually ran.

### Non-blocking

None.

### Test-harness deviations

1. A first test home under a broad inherited test-root ACL was retired before
   matrix evidence. The canonical fresh user-profile home passed owner-only
   ACLs; the retired home was preserved.
2. Two non-race Go commands first encountered the Windows Go runtime
   `DuplicateHandle` failure under high machine process load. Unchanged exact
   commands passed in independent fresh-process reruns; both failures remain
   recorded in §5.
3. Whole-store alias snapshots crossed writes from two unrelated live Codex
   sessions. Only their size/timestamp changed; all 98 stable rows and every
   controlled row matched. Controlled no-change refresh was identical.
4. A pre/post global-`last` comparison similarly raced live Codex writes. A
   read-only SQLite query immediately after `last` proved the plan matched the
   newest resumable row in the exact same refreshed index.
5. The Windows source build produced an extensionless PE file; a preserved
   `.exe` evidence copy was used only for supplemental version inspection.
6. Computer-use policy blocked the demanded terminal automation. This
   deviation was not converted into PASS evidence.
7. A later doctest recheck omitted the required process-scoped MSYS2 PATH and
   failed because its shell child returned no usable output. The exact targeted
   test and full doctest package passed in fresh PowerShell with MSYS2 restored;
   the failed attempt remains recorded in section 5.

## 13. Repository hygiene

- report-only branch: `test/v0.2.0-rc.2-windows-report`
- tested base commit: `8dd6073118131dd4ecacfca3eb1cde3f07df5eb6`
- changed files: `docs/testing/results/2026-08-02-windows-phase2-V020RC2.md` only
- private/local artifacts excluded: true
- product code unchanged: true
- secrets/transcripts committed: `false`
- cleanup performed: `false`; all evidence preserved
- merge/tag/deploy/stable claim: `false`

## 14. Device milestone block

```text
PHASE2-DEVICE-REPORT-V1
device=windows
test_commit=8dd6073118131dd4ecacfca3eb1cde3f07df5eb6
test_tag=v0.2.0-rc.2
reinstate_version=0.2.0-rc.2
installed_binary_sha256=D66807A27A76199CF05127A76274E83BCDA998777B06BC1F8CC71D4E9C9BFE01
installer_sha256=29ED0CFE697FE5D367EB7877CFC8899F99FFB63B479E5339F7B6983DFC540899
release_url=https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.2.0-rc.2
artifact_attestations=PASS_12_OF_12
report_path=docs/testing/results/2026-08-02-windows-phase2-V020RC2.md
report_branch=test/v0.2.0-rc.2-windows-report
claude_ref=claude:580473d0-8090-4f47-9bf4-8248881108f5
codex_ref=codex:019fc425-6155-7410-9c3e-713704f76795
gemini_state=PASS
opencode_state=PASS
required_pass=27
required_partial=0
required_fail=5
required_not_tested=0
optional_physical_pass=2
optional_physical_not_tested=0
configless_local_only=PASS
preview_privacy=PASS
claude_resume_fork=FAIL
codex_resume_fork=FAIL
picker=FAIL
phase1_regression=PASS
release_blocking_findings=1
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE2-DEVICE-REPORT-V1
```

Device B does not perform final cross-device reconciliation. This report must
be transferred to the existing macOS-arm64 Claude coordinator. No stable claim
is authorized while this device report is `FAIL`.
