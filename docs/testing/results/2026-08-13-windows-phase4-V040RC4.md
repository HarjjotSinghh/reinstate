# Phase 4 structured-handoff report — v0.4.0-rc.4 Windows amd64

Copy of the Phase 4 template with this dispatch's substitutions. Cumulative and
sanitized. No transcript text, prompts, responses, capsule bodies, credentials,
private paths, filenames, diffs, or raw child-process output.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `28 PASS / 0 PARTIAL / 16 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `2 PASS / 4 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `8`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. OpenCode rows are `NOT TESTED` because that vendor has no home
override. Gemini and Grok were already installed and were not installed for
acceptance.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-13T09:13:00Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| CPU architecture/native process | AMD64; 64-bit Windows PowerShell 5.1.26100.8328 Desktop; never WSL |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.4` |
| Tested full commit | `08d8b0ee469472e7382f12feb85914c75a9bdee0` |
| Installed binary SHA-256 | `1c9f23909606bd13eccbb6d783662b5ae4272f921defb7e45574758ec8f09663` |
| Installed version JSON | version `0.4.0-rc.4`; commit `08d8b0ee469472e7382f12feb85914c75a9bdee0`; date `2026-08-13T08:44:44Z` |
| Claude Code version/state | before first product command: host `2.1.229` source+destination; host later auto-updated to `2.1.231`; matrix collected with acceptance-tree PATH pin `2.1.229`; after last row pin still `2.1.229` |
| Codex CLI version/state | `0.147.0` source+destination; unchanged after last row |
| Gemini/OpenCode/Grok state | Gemini `0.53.0`; OpenCode `1.18.2` (not isolated); Grok `0.2.101`; all unchanged after last row |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.12 windows/amd64` via `GOTOOLCHAIN=go1.25.12` (host default go1.26.1 unused) |
| Report branch | `test/v0.4.0-rc.4-windows-amd64-report` |
| Device-report commit | `<filled after commit>` |
| Draft report PR | `<filled after PR>` |

Host identity: native Windows 11 x64, computer name HARJOTS-BEAST, user admin.
Ordinary Microsoft Defender real-time protection was enabled. GNU Make 4.4.1
(MSYS2), MinGW-w64 gcc 16.1.0, goreleaser 2.17.0, syft 1.50.0.

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | `git verify-tag` Good git SSH signature, ED25519, allowed_signers identity |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | annotated tag; peeled commit equals TEST_COMMIT; ancestor of origin/main |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub API: draft false, prerelease true, tag `v0.4.0-rc.4`, target main |
| Exact expected release asset set is present | `PASS` | 25 assets including windows_amd64 zip/exe/sbom and checksums.txt |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | zip digest `cefc01b08bd6301ca49c655a4f43691ce490e56fce20dfe6f362fdf538c62364`; exe digest equals installed binary; `gh attestation verify` exit 0 |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | PowerShell `check-release-artifacts.ps1` exit 0 on snapshot dist; published zip SBOM present |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.4` | `PASS` | live `install.ps1` sha256 `ec029d63267a0ec82021691cc5973f4f66268e1bd0876a633160733a889d82e5`; version pin `v0.4.0-rc.4`; no rc.1/rc.2/rc.3 pin |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | live pin and tagged `scripts/install.ps1` both `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; User and Machine PATH unchanged |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.4` | `PASS` | identical sha256; version JSON name reinstate, version 0.4.0-rc.4 |
| Installed binary reports the literal full tested commit | `PASS` | version `--json` commit field is the 40-character TEST_COMMIT; did not hang |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | detached TEST_COMMIT; tidy -diff exit 0, empty diff |
| `make verify` with Go 1.25.12 | `PASS` | exit 0, 116s, `verify ok`; includes fmt-check, vet, lint, test, test-race, vuln, build. RC3 Windows make-verify regression cleared |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `PASS` | `make test-race` covered FAST_PACKAGES with CGO=1 -race; `./internal/crypto` and `./internal/doctest` raced separately exit 0 |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | `scripts/snapshot.ps1` exit 0 (21s); `stage-release-assets.ps1` 0; `check-release-artifacts.ps1` 0; `test-install.ps1` 0 |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | included in `make verify` / unit tests |
| Capsule/projection/CLI goldens are unchanged across repeated G2 dry-runs | `PASS` | five G2 samples all exit 0; unit goldens in verify |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | fixture tests inside verify; installed D1/D2/D4 dry-run exit 0 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | Windows run 11.25 ms; projection 46541 bytes |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new rc.4 acceptance tree; five isolation variables set before product commands |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | tagged synthetic fixtures copied into throwaway homes; no rc.1/rc.2/rc.3 homes |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | only isolated Claude/Codex/Gemini/Grok homes; list JSON had no operator-home session paths |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | `push --dry-run` config-missing exit 3; no capsule sync |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | dummy dest capability file created then removed under isolated Claude home only |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | dry-run / `--no-launch` only; no destination vendor spawn except F8 refuse-before-launch |

OpenCode has no home override: E3/E4 `NOT TESTED`. D7 does not claim an isolated OpenCode result.

## 5. Required matrix — 44 rows

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | Claude → Codex dest acknowledgement requires Windows Terminal; autonomous SSH is not that TTY. Missing required dest-ack is FAIL |
| A2 | `FAIL` | Codex → Claude dest acknowledgement not collected; same TTY rule |
| A3 | `PASS` | throwaway isolated unauthenticated Claude source; dest Codex home empty of sessions; `--dry-run --json` exit 0; no source API |
| A4 | `FAIL` | dedicated incomplete-final-record dest-ack not collected after fixture overlay |
| A5 | `FAIL` | dest restatement not collected; missing Windows Terminal evidence |
| A6 | `FAIL` | completed-marker non-repeat not collected; missing dest-ack |
| A7 | `PASS` | `handoff list --json` exit 0 after stored lineage |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | tagged `TestHandoffExecutedOutputMatchesDryRunByteForByte` exit 0; installed `--dry-run` and `--no-launch` independently exit 0 |
| B2 | `PASS` | installed balanced `--dry-run --json` on frozen long-history fixture exit 0, ~3.0 s |
| B3 | `PASS` | `--policy checkpoint --no-launch --json` exit 0 |
| B4 | `FAIL` | attachment portability not separately inspected |
| B5 | `FAIL` | unknown opaque records not separately inspected |
| B6 | `FAIL` | `fidelity.json` had exact/normalized/omitted/referenced; no `summarized` class on the controlled source. RC3 regression |
| B7 | `PASS` | checkpoint and balanced policies both completed |
| B8 | `FAIL` | truncation and sidecar-reference counts not recorded as dedicated evidence |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | live dirty tree (staged, unstaged, untracked) appeared in `changed_files` as repo tokens |
| C2 | `FAIL` | moved-repository case not collected |
| C3 | `FAIL` | missing-repository case not collected |
| C4 | `PASS` | wrong-repo cwd `--dry-run --json` exit 5 |
| C5 | `PASS` | changed-file entries used `${REPO:<id>}` tokens; no operator-home paths in artifacts |
| C6 | `PASS` | dest capability gap reported on dry-run without leaking values |
| C7 | `PASS` | launch without capability/warning acknowledgement refused (non-TTY exit 7) |
| C8 | `PASS` | PATH shim `2.1.230` fail-closed exit 5; no new lineage from that probe |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | prompt-injection fixture `--dry-run --json` exit 0 |
| D2 | `PASS` | delimiter/fence-breakout fixture `--dry-run --json` exit 0 |
| D3 | `FAIL` | source-system-prompt exclusion not separately scanned |
| D4 | `PASS` | secret-leakage fixture `--dry-run --json` exit 0 |
| D5 | `PASS` | no backend/auth/keyring read required for launch-free rows |
| D6 | `PASS` | handoffs directory present with owner DACL; not world-writable |
| D7 | `PASS` | five isolation variables set; no operator-home sessions in `list`; OpenCode skipped |
| D8 | `PASS` | `push --dry-run` exit 3 config missing; no capsule sync |
| D9 | `PASS` | Claude `--no-redact` accepted as known flag; Grok `--no-redact` refused exit 2, not unknown-flag |
| D10 | `PASS` | launch-free rows did not rewrite vendor sources |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | JSON mode label `structured handoff`; destination_session_mode `new`; no forbidden resume language |
| F2 | `PASS` | `--json` without `--dry-run`/`--no-launch` exit 2 |
| F3 | `FAIL` | interactive picker requires Windows Terminal human evidence |
| F4 | `FAIL` | dedicated launch-free exact-command byte-reproduction row not collected |
| F5 | `PASS` | `handoff export --format json` exit 0 |
| F6 | `PASS` | `handoff inspect` of stored id exit 0 |
| F7 | `FAIL` | stale-warning acknowledgement case not collected |
| F8 | `FAIL` | after acknowledging current warnings, dest launch refused exit 7 with interactive-terminal requirement, but wall time ~6.1 s. RC2 requires refuse before spawn-scale delay |

### Matrix G — performance

Defender real-time protection enabled.

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run | 11.25 ms | 11.25 ms | 11.25 ms | `<2,000 ms`; projection `≤98,304 bytes` | `PASS` | projection 46541 bytes |
| G2 | 1 warmup + 5 installed `--dry-run` | 3036 ms | 3041 ms | 3041 ms | Windows max `12s` | `PASS` | all exit 0; warmup 3026 ms |
| G3 | 1 warmup + 20 `handoff list --limit 100 --json` | 1017 ms | 1020 ms | 1024 ms | Windows p95 `4s` | `PASS` | dedicated home; 100 `--no-launch` exit 0; list n=100 |

G3 used `--no-launch --json` without warning acknowledgement and still persisted 100 lineage rows. Launch without acks still refused (F8/C7).

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes` | `FAIL` | Gemini 0.53.0 present; Gemini→Claude dest row not collected |
| E2 | `yes` | `FAIL` | Gemini→Codex dest row not collected |
| E3 | `yes` | `NOT TESTED` | OpenCode 1.18.2 present; no isolation override |
| E4 | `yes` | `NOT TESTED` | same OpenCode isolation skip |
| E5 | `yes` | `PASS` | Grok-sourced dry-run did not report unsupported agent `"grok"` (exit 5 preflight, not exit 1) |
| E6 | `yes` | `PASS` | `--to grok` usage exit 2 |
| E7 | `yes` | `FAIL` | Grok rewind/compaction source row not collected |
| E8 | `yes` | `FAIL` | second Grok destination-path row not collected |

## R1–R6, RC2, and RC3 regressions (not in the 44-count)

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R1 | `PASS` | throwaway Claude home had `projects/` and no `version` file; doctor `agent.claude: SUPPORTED (layout-projects-jsonl-v1)`; inspect resolved the same slash-prefixed fixture |
| R2 | `PASS` | handoff completed; changed-file paths tokenized `${REPO:<id>}`; remaining fixture absolute strings were prose (R5); no operator-home hits |
| R3 | `PASS` | dirty disposable tree listed four changed files, not none |
| R4 | `FAIL` | hanging `--version` PATH shim classified UNTESTED exit 5 but wall time 43335 ms, over the 25 s harness budget |
| R5 | `PASS` | slash-prefixed first user message reached `--dry-run --json` exit 0 |
| R6 | `PASS` | pinned `2.1.229` in range and dest Plan exit 0; shim `2.1.230` fail-closed exit 5. Host global later read `2.1.231` |
| RC2 C4 | `PASS` | wrong-repo exit 5 |
| RC2 F8 | `FAIL` | see F8 timing |
| RC2 E5/E6 | `PASS` | see E5/E6 |
| RC2 R4 classification | `FAIL` | hang returned UNTESTED/exit 5 (correct class) but not in time |
| RC2 D9 | `PASS` | see D9 |
| RC2 list isolation | `PASS` | `list --json` had no operator-home sessions |
| RC2 B3 | `PASS` | see B3 |
| RC2 B6 | `FAIL` | see B6 |
| RC2 C6/C7 | `PASS` | see C6/C7 |
| RC3 flagship A empty dest | `PASS` | empty Codex dest layout; `--dry-run --json` exit 0; dest not UNTESTED |
| RC3 R4 hang | `FAIL` | see R4 |
| RC3 Windows `make verify` | `PASS` | see automated gates |
| RC3 G3 list | `PASS` | n=100 after 100 `--no-launch`; p95 1020 ms |
| RC3 B6 summarized | `FAIL` | see B6 |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree + verify |
| `make verify` is green on both mandatory platforms | `PASS` | this device; macOS is coordinator-owned |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A3 Plan PASS; A1/A2 dest-ack FAIL |
| Every fidelity class has real report evidence and byte-stable goldens | `FAIL` | B6 summarized missing |
| Injection, secret, and bounded-read security gates leak nothing | `FAIL` | D3 not separately evidenced |
| 200-turn projection is bounded and reported | `PASS` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `PASS` | C5 + R2 |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 unit |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS` | observed 0/2/3/5/7; D7–D8 |

## 8. Findings and repository hygiene

### Release-blocking

1. F8 non-TTY dest launch refuses with the interactive-terminal error and exit 7, but only after ~6.1 s (RC2 spawn-delay bar).
2. R4 / RC3 hanging `--version` returns UNTESTED/exit 5 after 43 s, over the 25 s budget.
3. B6 / RC3 `summarized` portability class was absent from `fidelity.json` on the controlled source.
4. A1/A2/A5/A6 missing required Windows Terminal dest-acknowledgement evidence.
5. Host Claude Code auto-updated from `2.1.229` to `2.1.231` after the pre-run record. In-range Claude-sourced rows were collected only after pinning `2.1.229` on PATH inside the acceptance tree. Unpinned host `2.1.231` fail-closes as UNTESTED, which matches R6's closed ceiling, but the host is no longer inside 2.1.219–2.1.229.
6. C2/C3 workspace cases not collected.
7. F3 interactive picker missing Windows Terminal evidence.
8. Gemini was installed and E1/E2 were not collected (optional-installed rows are FAIL, not NOT TESTED).

### Non-blocking

- GnuWin32 make 3.81 is on the default PATH and is the wrong make; MSYS2 make 4.4.1 was prepended for this run.
- WinGet `goreleaser.exe` / `syft.exe` shims were 0-byte; real package binaries were prepended.
- `doctor` reports config-missing (expected without `init`) while agents are SUPPORTED.
- Empty Codex `sessions` directory missing caused an early `list` scan error; creating the dest layout after the empty-dest Plan row unblocked list isolation.
- F8 without warning acknowledgement first fails on unacknowledged warnings; the interactive-terminal refuse appears only after those IDs are allowed.

### Test-harness deviations and supersessions

- First matrix pass treated PowerShell hashtable exit codes as null and classified many product-correct rows FAIL. Superseded by matrix2/matrix3 using `PSCustomObject` + `Start-Process`.
- First Claude-sourced handoffs were UNTESTED because the host binary had become `2.1.231`. Superseded by acceptance-tree pin of `2.1.229`. Host global install was not downgraded.
- OpenCode skipped (`NOT TESTED: no isolation override`).
- Real TTY dest-ack and picker rows were not invented over SSH.

Report-only diff: this file. Merge-base with TEST_COMMIT: this branch is created from `08d8b0ee469472e7382f12feb85914c75a9bdee0`. No product files changed.

## 9. Required terminated device block

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.4
test_commit=08d8b0ee469472e7382f12feb85914c75a9bdee0
installed_binary_sha256=1c9f23909606bd13eccbb6d783662b5ae4272f921defb7e45574758ec8f09663
required_pass=28
required_partial=0
required_fail=16
required_not_tested=0
optional_pass=2
optional_fail=4
optional_not_tested=2
artifact_chain=PASS
isolation_privacy=PASS
flagship_directions=FAIL
fidelity_policy=FAIL
workspace_capability=FAIL
security=FAIL
cli_contract=FAIL
performance=PASS
phase1_phase2_phase3_regression=PASS
release_blocking_findings=8
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
