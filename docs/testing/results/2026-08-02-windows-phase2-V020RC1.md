# Phase 2 acceptance — native-Windows Device B report

**Verdict:** `FAIL`
**Milestone:** `DEVICE_COMPLETE`
**Required counts:** `3 PASS / 0 PARTIAL / 2 FAIL / 25 NOT TESTED`
**Optional physical counts:** `0 PASS / 2 NOT TESTED`
**Complete 32-row counts:** `3 PASS / 0 PARTIAL / 2 FAIL / 27 NOT TESTED`

This report covers only the exact release artifacts and fresh disposable paths
created for this run. No real transcript content or secret was used as
evidence. Installed-binary behavior testing stopped at the mandatory W0
provenance gate because the release binary did not report the full tested
commit.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | 2026-08-02T18:09:47Z |
| Device | native-Windows Device B |
| Tested Git commit | `e458e8e80be206ee09dd95e5698a588f60d41a25` |
| Signed tag | `v0.2.0-rc.1` |
| Reinstate version JSON | `{"commit":"e458e8e","date":"2026-08-01T09:05:30Z","name":"reinstate","version":"0.2.0-rc.1"}` |
| OS/version/build | Microsoft Windows 11 Pro 10.0.26200, build 26200 |
| Architecture | AMD64; native 64-bit process |
| Native shell | Windows PowerShell 5.1.26100.8328 Desktop |
| Claude Code version/state | 2.1.220; installed; physical path `NOT TESTED` after W0 stop |
| Codex CLI version/state | 0.146.0; installed; physical path `NOT TESTED` after W0 stop |
| Gemini CLI version/state | 0.53.0; installed; physical path `NOT TESTED` after W0 stop |
| OpenCode version/state | 1.18.2; installed; physical path `NOT TESTED` after W0 stop |
| Git version | 2.52.0.windows.1 |
| Go version | go1.25.12 windows/amd64 |
| Toolchain | GNU Make 4.4.1; MSYS2 GCC 16.1.0; process-scoped PATH; `GOTOOLCHAIN=go1.25.12`; `CGO_ENABLED=1` |
| Report branch | `test/v0.2.0-rc.1-windows-report` |
| Draft PR | `https://github.com/HarjjotSinghh/reinstate/pull/86` (open, draft, unmerged) |

## 2. Tagged release, installer, and installed-artifact evidence

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tag is annotated | PASS | `git cat-file -t refs/tags/v0.2.0-rc.1` returned `tag`. |
| Tag signature is allowed | PASS | Command-scoped SSH verification against the tag commit's `.github/allowed_signers` exited 0 for the expected signer. |
| Tag commit is on `origin/main` | PASS | `git merge-base --is-ancestor TEST_COMMIT origin/main` exited 0 after a fetch without merge. |
| GitHub release state | PASS | Published 2026-08-01T09:12:14Z; non-draft; prerelease; tag `v0.2.0-rc.1`. |
| Required release assets | PASS | 12 uploaded assets: `checksums.txt`, five platform archives, five matching SPDX 2.3 SBOMs, and one source archive. |
| Release checksums | PASS | All 11 entries in `checksums.txt` matched downloaded bytes; all 12 downloaded SHA-256 values also matched GitHub asset digests. |
| Artifact attestations | PASS | `gh attestation verify` passed for each of 12 assets with repository, source tag ref, and full source digest enforced. |
| Archive membership | PASS | Five platform archives each contained only four documents plus the expected binary; the source archive had one expected root; zero absolute or traversal members. No archive member was executed before checksum and attestation verification. |
| Public bootstrap identity | PASS | Live `https://reinstate.dev/install.ps1` returned HTTP 200 without redirect and was byte-identical to `website/public/install.ps1` at `TEST_COMMIT`. |
| Public bootstrap version pin | PASS | Exactly one SemVer pin was present: `v0.2.0-rc.1`. |
| Public bootstrap SHA-256 | PASS | `17E294FE0D9E095125B18A733D9DAC855B8CE7A40015874544C76E87335D24B3`. |
| Canonical installer SHA-256 | PASS | Bootstrap pin and tagged `scripts/install.ps1` both identified `02C68984964556E7C685A275BDE72DC812162E0B898BE0F26718A0813EFC0DFE`. |
| Isolated install | PASS | `<INSTALL_DIR>` was empty before install; only `rein.exe` and `reinstate.exe` were created; user and machine PATH values remained byte-identical. |
| Windows archive SHA-256 | PASS | `22C5A06F2828F7347C67FAA5E39A939B9D16D3E21A177DB2F94F82BBDA8F883E`, matching checksums and GitHub digest. |
| Installed binary SHA-256 | PASS | Both aliases and the independently extracted verified archive binary matched `A63AFAD31B24C5ABB04487E8AE28BE5434F6AC3B572D3776315BF5DA90938D65`. |
| Installed binary architecture | PASS | PE machine `0x8664` (AMD64). |
| Installed version | PASS | Both aliases returned version `0.2.0-rc.1` and identical JSON. |
| Installed full commit | FAIL | Both aliases returned only `e458e8e`, not required full `e458e8e80be206ee09dd95e5698a588f60d41a25`. `.goreleaser.yml` injects `{{.ShortCommit}}`. |
| Release URL | PASS | `https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.2.0-rc.1` |

## 3. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches the requested commit | PASS | Tag dereference and report base both equal `e458e8e80be206ee09dd95e5698a588f60d41a25`. |
| Binary reports the tested commit | FAIL | Installed JSON contains only the seven-character short commit. |
| Product tree was clean before testing | PASS | Porcelain status and diff were empty. |
| Report branch starts at the tested commit | PASS | Branch merge-base equals `TEST_COMMIT`. |
| Report is the only committed change | PASS | PR #86 and base-to-tip diff contain exactly this report file. |
| No secret/transcript/private path was committed | PASS | Final privacy scan reported zero prohibited-path, credential, key, or transcript-pattern hits. |

## 4. Isolation and local-only proof

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | PASS (precondition only) | `<ISOLATED_HOME>` did not exist before W0 and remains absent. |
| Fresh controlled corpus | PASS (precondition only) | `<CORPUS_ROOT>` was fresh and remains empty. |
| No `rein init` run | PASS | No initialization command was run. |
| No `config.toml` or sync state created | PASS | The isolated home remains absent. |
| No credential/passphrase/keyring request | PASS | No local-index command was run after the W0 stop; installer requested none. |
| No backend/network dependency | NOT TESTED | Installed local-index behavior was not entered. |
| Only derived index state created | NOT TESTED | No index was created. |
| Index and parent permissions are owner-only | NOT TESTED | No index exists. |

## 5. Controlled corpus

| Agent | Composite reference | Disposable project label | Marker found | Capability |
| ----- | ------------------- | ------------------------ | ------------ | ---------- |
| Claude | `NOT TESTED` | `NOT CREATED` | `NOT TESTED` | full, not physically exercised |
| Codex | `NOT TESTED` | `NOT CREATED` | `NOT TESTED` | full, not physically exercised |
| Gemini | `NOT TESTED` | `NOT CREATED` | `NOT TESTED` | read-only; installed; not physically exercised |
| OpenCode | `NOT TESTED` | `NOT CREATED` | `NOT TESTED` | read-only; installed; not physically exercised |

The mandatory W0 rule requires a binary-commit mismatch to be recorded as
`FAIL` and product behavior testing to stop. No older development corpus was
reused, and no vendor session was read or modified.

## 6. Automated verification

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Direct format assertion | PASS | `gofmt -l .` exited 0 with zero paths. |
| Focused session-index/CLI/adapter tests | PASS | Exact focused command exited 0 for all requested packages. |
| Full Go test suite | PASS on first direct run | `go test ./... -count=1` exited 0 across all packages. |
| Full race suite | PASS | `CGO_ENABLED=1 go test -race ./... -count=1` exited 0 across all packages. |
| Vet | PASS | `make vet` exited 0. |
| `make verify` | FAIL | Exit 2; its embedded full test crashed in the Go Windows runtime with invalid-handle error. |
| Required cross-builds | PASS | Windows amd64, Darwin arm64, Darwin amd64, and Linux amd64 builds all exited 0 with `CGO_ENABLED=0`. |
| Supplemental exact source build | PASS, not release evidence | Separate source binary reported version `0.2.0-rc.1` plus the full commit; it was not used for any installed-artifact gate. |
| Phase 1 regression | PASS | The first complete direct `go test ./... -count=1` run passed all Phase 1 packages. |
| Commit CI workflows | PASS | CI, Security, Release, and signed website deployment validation completed successfully at `TEST_COMMIT`. |

### Failed commands and failed assertions

1. Installed `rein version --json` and `reinstate version --json` both exited
   0, but failed the mandatory assertion because `commit` was `e458e8e`
   instead of the full `TEST_COMMIT`.
2. `make verify` exited 2. Sanitized output:
   `runtime: setevent failed; errno=6`; `fatal error: runtime.semawakeup`;
   `FAIL .../internal/cli`; `make: *** [Makefile:46: test] Error 1`.
3. Minimal unchanged reproduction `go test ./internal/cli -count=1` exited 1.
   Sanitized output: `runtime.preemptM: duplicatehandle failed; errno=6`;
   `fatal error: runtime.preemptM: duplicatehandle failed`;
   `FAIL .../internal/cli`.
4. Seven initially concurrent `gh attestation verify` processes exited 1 for
   the Darwin amd64 archive, Darwin arm64 archive and SBOM, Linux amd64
   archive, Linux arm64 archive, Windows amd64 archive, and Windows amd64
   SBOM. Sanitized output was either `no valid Sigstore verifiers could be
   initialized` or `public good verifier is not available`. Sequential exact
   same-file retries all exited 0 with one verified attestation; the final
   artifact-attestation result is PASS.

`make fmt-check` exited 0 but printed
`tee: /dev/stderr: No such file or directory`; the independent direct format
assertion passed. This warning is not hidden or treated as clean Makefile
behavior.

## 7. Configless index and refresh

`NOT TESTED`. No installed local-index command was run after the W0 provenance
failure. Therefore sessions discovery, alias parity, ordering, stable identity,
permissions, rebuild, idempotency, append refresh, and new-session refresh are
not claimed.

## 8. Search and inspect

`NOT TESTED` physically. Prompt, agent, project, branch, file, AND-term, limit,
case, Unicode, zero-match, and bounded-preview assertions were not run against
the installed artifact. Fixture-backed tests passed in the successful direct
full suite.

## 9. Last, resume, and fork

`NOT TESTED` physically. No dry-run or native child was launched. No Claude or
Codex session was created, resumed, forked, inspected, or modified.

## 10. Interactive switcher

`NOT TESTED`. Real Windows Terminal human-keyboard picker, resume, and fork
routes were not entered after the W0 stop. No byte-only ConPTY, AppActivate,
SendKeys, or other automated input was substituted.

## 11. Read-only adapters

Gemini CLI 0.53.0 and OpenCode 1.18.2 are installed, but both optional physical
paths are `NOT TESTED` because the installed-binary provenance gate stopped the
run. Consequently, no physical Gemini/OpenCode session was created and no
physical OpenCode top-level `updated`/`created`, non-year-1 ordering,
unfiltered visibility, literal-ID search, inspect, or exit-5 refusal claim is
made.

The injected-record and fake-runner gates did pass in the successful focused
and full test runs. They cover read-only capability metadata, OpenCode and
Gemini discovery, and compatibility refusal. They do not substitute for the
missing optional physical rows.

## 12. Mandatory matrix

| # | Gate | Windows | Evidence |
| - | ---- | ------- | -------- |
| 1 | Exact tested commit/binary provenance | FAIL | Sections 2–3: tag/archive provenance passed, installed JSON omitted the full SHA. |
| 2 | Full local verification and required cross-builds | FAIL | Section 6: cross-builds and full race passed; required `make verify` exited 2 and reproduced a runtime crash. |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | NOT TESTED | Section 4: fresh preconditions passed, installed local behavior was not entered. |
| 4 | `rein sessions` discovers exact Claude sessions | NOT TESTED | No controlled Claude session created. |
| 5 | `rein sessions` discovers exact Codex sessions | NOT TESTED | No controlled Codex session created. |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | NOT TESTED | Installed behavior stopped at W0. |
| 7 | Derived index path, rebuild, idempotency, and private permissions | NOT TESTED | No index created. |
| 8 | Prompt-fragment literal search | NOT TESTED | Installed behavior stopped at W0. |
| 9 | Agent filter | NOT TESTED | Installed behavior stopped at W0. |
| 10 | Project filter | NOT TESTED | Installed behavior stopped at W0. |
| 11 | Branch filter | NOT TESTED | Installed behavior stopped at W0. |
| 12 | File filter | NOT TESTED | Installed behavior stopped at W0. |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | NOT TESTED | Installed behavior stopped at W0. |
| 14 | `sessions` and `search` do not dump transcript passages | NOT TESTED | No physical listing/search output produced. |
| 15 | `inspect` metadata/160-code-point user preview policy | NOT TESTED | No physical inspect output produced. |
| 16 | Append/new-session refresh and no-change idempotency | NOT TESTED | No controlled session created. |
| 17 | `last` selects the correct resumable session and filters | NOT TESTED | Installed behavior stopped at W0. |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | NOT TESTED | Installed behavior stopped at W0. |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | NOT TESTED | Installed behavior stopped at W0. |
| 20 | Claude native resume | NOT TESTED | No native launch. |
| 21 | Codex native resume | NOT TESTED | No native launch. |
| 22 | Claude vendor-native fork, source preserved | NOT TESTED | No native launch. |
| 23 | Codex vendor-native fork, source preserved | NOT TESTED | No native launch. |
| 24 | Missing/ambiguous reference and missing executor fail safely | NOT TESTED | Installed behavior stopped at W0. |
| 25 | JSON/native-child separation and child failure propagation | NOT TESTED | Installed behavior stopped at W0. |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | NOT TESTED | No human-keyboard Windows Terminal run. |
| 27 | Non-TTY prompt failure is immediate and actionable | NOT TESTED | Installed behavior stopped at W0. |
| 28 | Gemini read-only physical path, when installed | NOT TESTED | Gemini installed; physical path not run after W0 stop. |
| 29 | OpenCode read-only physical path, when installed | NOT TESTED | OpenCode installed; required provenance/discovery path not run after W0 stop. |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | PASS | Successful focused/full tests include CLI read-only compatibility-exit and launch-plan refusal gates. |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | PASS | Successful focused/full tests include bounded-line, malformed, incomplete, oversized, source, store, and preview-policy gates. |
| 32 | Phase 1 automated regression remains green | PASS | Successful first complete direct full suite covered all Phase 1 packages. |

## 13. Findings

### Release-blocking

1. The installed attested RC1 binary fails the explicit full-commit identity
   requirement. It reports seven characters because the release configuration
   injects `{{.ShortCommit}}`.
2. The required native-Windows `make verify` gate fails under the mandated Go
   1.25.12/CGO=1 toolchain with reproducible Go runtime invalid-handle crashes.
3. The mandatory installed-artifact physical matrix is incomplete after the W0
   stop: 25 required rows and both installed optional vendor rows remain
   `NOT TESTED`. Missing evidence blocks certification.

### Non-blocking

1. `make fmt-check` emits a native-Windows `/dev/stderr` warning even though
   its exit is 0 and an independent `gofmt -l .` assertion is clean.
2. Concurrent Sigstore verifier initialization was unreliable on this host;
   sequential per-asset verification passed for all 12 assets.

### Test-harness deviations

- The required real Windows Terminal human-keyboard routes were not run after
  the W0 stop. No prohibited automation was accepted as evidence.
- No retry result is used to erase the failed `make verify` command.
- Fresh home, corpus, install directory, report worktree, and artifact paths
  are preserved for coordinator review. No cleanup was performed.

## 14. Repository hygiene

- report-only branch: `test/v0.2.0-rc.1-windows-report`
- tested base commit: `e458e8e80be206ee09dd95e5698a588f60d41a25`
- changed files: `docs/testing/results/2026-08-02-windows-phase2-V020RC1.md` only
- private/local artifacts excluded: `true`
- product code unchanged: `true`
- secrets/transcripts committed: `false`

## 15. Device milestone block

```text
PHASE2-DEVICE-REPORT-V1
device=windows
test_commit=e458e8e80be206ee09dd95e5698a588f60d41a25
reinstate_version=0.2.0-rc.1
report_path=docs/testing/results/2026-08-02-windows-phase2-V020RC1.md
report_branch=test/v0.2.0-rc.1-windows-report
claude_ref=NOT_TESTED
codex_ref=NOT_TESTED
gemini_state=FAIL
opencode_state=FAIL
required_pass=3
required_partial=0
required_fail=2
required_not_tested=25
optional_physical_pass=0
optional_physical_not_tested=2
configless_local_only=FAIL
preview_privacy=FAIL
claude_resume_fork=FAIL
codex_resume_fork=FAIL
picker=FAIL
phase1_regression=PASS
release_blocking_findings=3
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE2-DEVICE-REPORT-V1
```

Device B does not perform final cross-device reconciliation. This report is a
failed Windows tagged-artifact certification and must not be used to promote
stable `v0.2.0`.
