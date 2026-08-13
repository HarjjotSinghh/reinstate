# Phase 4 structured-handoff report — v0.4.0-rc.5 Windows amd64

Copy of the Phase 4 template with this dispatch's substitutions. Cumulative and
sanitized. No transcript text, prompts, responses, capsule bodies, credentials,
private paths, filenames, diffs, or raw child-process output.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `38 PASS / 0 PARTIAL / 6 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `4 PASS / 2 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `4`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. OpenCode rows are `NOT TESTED` because that vendor has no home
override. Gemini and Grok were already installed and were not installed for
acceptance.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-13T14:00:00Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| CPU architecture/native process | AMD64; 64-bit Windows PowerShell 5.1.26100.8328 Desktop; never WSL |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.5` |
| Tested full commit | `0d7551a69918a97967f927ed9dc5a56b3583108c` |
| Installed binary SHA-256 | `3b126e60f61c1bc22c32ccbfa7e1bb34dc6d47be0d9a1f2d51f2f2c823ad840d` |
| Installed version JSON | version `0.4.0-rc.5`; commit `0d7551a69918a97967f927ed9dc5a56b3583108c`; date `2026-08-13T13:16:36Z` |
| Claude Code version/state | before first product command: PATH pin `2.1.229` source+destination; host global remained `2.1.231` and was not used; after last row pin still `2.1.229` |
| Codex CLI version/state | `0.147.0` source+destination; unchanged after last row |
| Gemini/OpenCode/Grok state | Gemini `0.53.0`; OpenCode `1.18.2` (not isolated; omitted from process PATH); Grok `0.2.101`; all unchanged after last row |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.12 windows/amd64` via `GOTOOLCHAIN=go1.25.12` (host default go1.26.1 unused) |
| Report branch | `test/v0.4.0-rc.5-windows-amd64-report` |
| Device-report commit | `this PR commit` |
| Draft report PR | `https://github.com/HarjjotSinghh/reinstate/pull/204` |

Host identity: native Windows 11 x64, computer name HARJOTS-BEAST, user admin.
Ordinary Microsoft Defender real-time protection was enabled. GNU Make 4.4.1
(MSYS2), MinGW-w64 gcc 16.1.0, goreleaser 2.17.0, syft 1.50.0.

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | `git verify-tag` Good git SSH signature, ED25519, allowed_signers identity |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | annotated tag; peeled commit equals TEST_COMMIT; ancestor of origin/main |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub API: draft false, prerelease true, tag `v0.4.0-rc.5`, target main |
| Exact expected release asset set is present | `PASS` | 25 uploaded assets including windows_amd64 zip/exe/sbom and checksums.txt |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | zip digest `7767cacf79753850f30a201e08ff5060e66d59ea579ea16da8a211968b2f70ec`; exe digest equals installed binary; `gh attestation verify` exit 0 |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | zip members CHANGELOG.md, LICENSE, NOTICE, README.md, reinstate.exe, rein.exe; PowerShell artifact gates exit 0 on snapshot dist |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.5` | `PASS` | live `install.ps1` sha256 `14dbdc61a0c7d4c7d6be33374f29b020e19b7e934a6a0ec2a3067bf31fd7515e`; version pin `v0.4.0-rc.5`; no rc.1/rc.2/rc.3/rc.4 pin |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | live pin and tagged `scripts/install.ps1` both `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; User and Machine PATH unchanged |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.5` | `PASS` | identical sha256; version JSON name reinstate, version 0.4.0-rc.5 |
| Installed binary reports the literal full tested commit | `PASS` | version `--json` commit field is the 40-character TEST_COMMIT |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | detached TEST_COMMIT; tidy -diff exit 0, empty diff |
| `make verify` with Go 1.25.12 | `PASS` | exit 0, 148888 ms, `verify ok`; MSYS2 make 4.4.1 prepended. RC3 Windows make-verify regression cleared |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `PASS` | included in `make verify` test-race after a redirected-buffer flake was rerun with Tee-Object |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | `scripts/snapshot.ps1` exit 0 (44100 ms); `stage-release-assets.ps1` 0; `check-release-artifacts.ps1` 0; `test-install.ps1` 0 |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | included in `make verify` / unit tests |
| Capsule/projection/CLI goldens are unchanged across repeated G2 dry-runs | `PASS` | five G2 samples all exit 0; unit goldens in verify |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | fixture-scan exit 0; installed D1/D2/D4 dry-run exit 0 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | Windows run 10.1056 ms; projection 46541 bytes |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new rc.5 acceptance tree; five isolation variables set before product commands |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | tagged synthetic fixtures copied into throwaway homes; no rc.1–rc.4 homes |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | only isolated Claude/Codex/Gemini/Grok homes; first `sessions --agent claude --json` had no operator-home paths; OpenCode omitted from process PATH |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | `push --dry-run` config-missing exit 3; no capsule sync |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | dummy dest capability file created then removed under isolated Claude home only |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | dry-run / `--no-launch` only; F8 refused before spawn; dest-ack rows not launched |

OpenCode has no home override: E3/E4 `NOT TESTED`. D7 does not claim an isolated OpenCode result. First refresh warned `agent_not_installed` for OpenCode because that executable was omitted from the test PATH.

## 5. Required matrix — 44 rows

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | Claude → Codex dest acknowledgement not collected. `ssh -t` did not allocate a PTY; `ssh -tt` attached conhost then closed before the destack script ran. Missing required dest-ack is FAIL |
| A2 | `FAIL` | Codex → Claude dest acknowledgement not collected; same console rule |
| A3 | `PASS` | throwaway isolated Claude source; dest Codex home empty of sessions; `--dry-run --json` exit 0; dest not UNTESTED |
| A4 | `FAIL` | incomplete-final-record dest-ack not collected after fixture overlay; missing required dest-ack is FAIL |
| A5 | `FAIL` | dest restatement not collected; no Windows Terminal evidence |
| A6 | `FAIL` | completed-marker non-repeat not collected; missing dest-ack |
| A7 | `PASS` | `handoff list --json` exit 0; Codex destination state `unresolved` |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | tagged `TestHandoffExecutedOutputMatchesDryRunByteForByte` exit 0 |
| B2 | `PASS` | long-history fixture balanced `--dry-run --json` exit 0, 956 ms |
| B3 | `PASS` | `--policy checkpoint --no-launch --json` exit 0 |
| B4 | `PASS` | attachments fixture `--dry-run --json` exit 0 |
| B5 | `PASS` | Codex unknown-records fixture `--dry-run --json` exit 0; opaque class present |
| B6 | `PASS` | compaction fixture `fidelity.json` included `summarized`. RC3/RC5 regression cleared |
| B7 | `PASS` | checkpoint and balanced policies both completed |
| B8 | `PASS` | unknown opaque records preserved as hashed references; long-history/compaction JSON recorded truncation/sidecar metadata |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | live dirty disposable tree appeared in `changed_files` |
| C2 | `PASS` | moved-repository dry-run exit 0; Git workspace used |
| C3 | `PASS` | missing same-OS workspace session `c30000000001` from Demo `--dry-run --json` exit 5; control Windows cwd at live Demo exit 0. Foreign macOS fixture cwd is C5, not C3 |
| C4 | `PASS` | Demo and Other are distinct git toplevels. Session cwd Demo from Other `--dry-run --json` exit 5 (`different repository`); same session from Demo exit 0 |
| C5 | `PASS` | macOS os-roots fixture from a local git checkout emitted `${REPO:demo}`; no source-device `C:\` fixture-user path in the plan. RC5 remap cleared |
| C6 | `PASS` | dest capability gap reported on dry-run without leaking values |
| C7 | `PASS` | launch without capability/warning acknowledgement refused (non-TTY exit 7 on F8) |
| C8 | `PASS` | agent-root `version` `2.1.230` plus PATH-only LookPath shim: source `--dry-run --json` exit 5, no new lineage. Matches fail-closed contract; PATH echo while 2.1.229 remained on PATH is not this row |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | prompt-injection fixture `--dry-run --json` exit 0 |
| D2 | `PASS` | delimiter/fence-breakout fixture `--dry-run --json` exit 0 |
| D3 | `PASS` | exported markdown projection system-prompt scan hits=0 |
| D4 | `PASS` | secret-leakage fixture `--dry-run --json` exit 0 |
| D5 | `PASS` | no backend/auth/keyring read required for launch-free rows |
| D6 | `PASS` | handoffs directory present under isolated `REINSTATE_HOME` |
| D7 | `PASS` | five isolation variables set; no operator-home sessions in Claude list; OpenCode skipped |
| D8 | `PASS` | `push --dry-run` exit 3 config missing; no capsule sync |
| D9 | `PASS` | Grok `--no-redact` refused exit 2 |
| D10 | `PASS` | source session SHA-256 identical before and after dry-run |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | JSON mode label `structured handoff`; destination_session_mode `new`; no forbidden resume language |
| F2 | `PASS` | `--json` without `--dry-run`/`--no-launch` exit 2 |
| F3 | `FAIL` | interactive picker requires a real console; `ssh -t`/`ssh -tt` did not yield usable Windows Terminal evidence |
| F4 | `PASS` | `handoff export --format json` of stored `handoff_id` exit 0 |
| F5 | `PASS` | `--no-launch --json` exit 0 |
| F6 | `PASS` | `handoff inspect` of stored `handoff_id` exit 0 |
| F7 | `PASS` | stale `--allow-warning` id exit 2 |
| F8 | `PASS` | non-TTY dest launch refused exit 7 with interactive-terminal error in 40 ms. RC2/RC5 refuse-before-Plan cleared |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run | 10.11 ms | 10.11 ms | 10.11 ms | `<2,000 ms`; projection `≤98,304 bytes` | `PASS` | projection 46541 bytes |
| G2 | 1 warmup + 5 installed `--dry-run` | 1011 ms | 1011 ms | 1011 ms | Windows max `12s` | `PASS` | all exit 0 |
| G3 | 1 warmup + 20 `handoff list --limit 100 --json` | 37 ms | 67 ms | 70 ms | Windows p95 `4s` | `PASS` | dedicated home; 100 `--no-launch` exit 0; list n=100 |

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes` | `PASS` | Gemini 0.53.0 present; `GEMINI_CLI_HOME/tmp/projhashfixture/chats` overlay indexed `gemini-rewind-win` |
| E2 | `yes` | `FAIL` | Gemini→Codex `--dry-run --json` exit 5 `environment preflight is blocked` after cwd rewrite to the live Demo git checkout |
| E3 | `yes` | `NOT TESTED` | OpenCode 1.18.2 present; no isolation override |
| E4 | `yes` | `NOT TESTED` | same OpenCode isolation skip |
| E5 | `yes` | `FAIL` | Grok tagged-fixture `--dry-run` exit 5 `environment preflight is blocked` from a local git checkout. RC5 fixture-user cwd must not block |
| E6 | `yes` | `PASS` | `--to grok` usage exit 2 |
| E7 | `yes` | `PASS` | destination `--to gemini` refused exit 2 |
| E8 | `yes` | `PASS` | Grok native continuation refused exit 2 |

## R1–R6, RC2, RC3, and RC5 regressions (not in the 44-count)

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R1 | `PASS` | throwaway Claude home had `projects/` and no `version` file; doctor `agent.claude: SUPPORTED (layout-projects-jsonl-v1)` |
| R2 | `PASS` | changed-file paths tokenized `${REPO:demo}`; C5 artifacts had no operator-home hits |
| R3 | `PASS` | dirty disposable tree listed changed files, not none |
| R4 | `PASS` | hanging `--version` PATH shim classified UNTESTED, compatibility, exit 5, wall 10571 ms (budget 25 s). RC3/RC5 hang cleared |
| R5 | `PASS` | slash-prefixed first user message `--dry-run --json` exit 0 |
| R6 | `PASS` | pinned `2.1.229` in range and doctor SUPPORTED; shim `2.1.230` fail-closed exit 5 when it was the only `claude` on PATH |
| RC2 C4 | `PASS` | see C4 |
| RC2 F8 | `PASS` | see F8 timing |
| RC2 E5/E6 | `FAIL` | see E5 fixture-user preflight; E6 PASS |
| RC2 R4 classification | `PASS` | hang returned UNTESTED / exit 5 / code `compatibility` in time |
| RC2 D9 | `PASS` | see D9 |
| RC2 list isolation | `PASS` | Claude `sessions --json` had no operator-home sessions |
| RC2 B3 | `PASS` | see B3 |
| RC2 B6 | `PASS` | see B6 |
| RC2 C6/C7 | `PASS` | see C6/C7 |
| RC3 flagship A empty dest | `PASS` | empty Codex dest layout; `--dry-run --json` exit 0; dest not UNTESTED |
| RC3 R4 hang | `PASS` | see R4 |
| RC3 Windows `make verify` | `PASS` | see automated gates |
| RC3 G3 list | `PASS` | n=100 after 100 `--no-launch`; p95 67 ms |
| RC3 B6 summarized | `PASS` | see B6 |
| RC5 R4 hang | `PASS` | process-tree kill returned in 10.6 s |
| RC5 F8 | `PASS` | refuse-before-Plan 40 ms |
| RC5 B6 | `PASS` | summarized class present |
| RC5 C5 | `PASS` | foreign-OS os-roots remap |
| RC5 E5/E6 | `FAIL` | see E5 |
| Dest-ack A1/A2/A5/A6 | `FAIL` | see Matrix A |

## 7. Architecture §14 closeout

| Claim | Result | Sanitized evidence |
| ----- | ------ | ------------------ |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A3 Plan PASS; A1/A2 dest-ack FAIL |
| Cross-agent results are a new destination session continuing the same task | `PASS` | JSON `destination_session_mode=new`; mode label `structured handoff` |
| Native resume remains same-vendor only | `PASS` | no silent transcript translation; dest Gemini/Grok refused |
| Path remapping is first-class | `PASS` | C5 `${REPO:demo}` from macOS os-roots fixture on Windows |

## 8. Findings and repository hygiene

### Release-blocking

1. A1/A2/A4/A5/A6 missing required destination first-reply restatement. `ssh -t` and `ssh -tt` from this agent did not produce a usable Windows console; dest-ack was not invented.
2. F3 interactive picker missing Windows Terminal evidence.
3. E2: Gemini was installed; after a correct `tmp/projhashfixture/chats` overlay and Demo cwd rewrite, Gemini→Codex `--dry-run` still exited 5. Optional-installed row is FAIL.
4. E5 / RC5: Grok tagged-fixture dry-run from a local git checkout exited 5 `environment preflight is blocked` (fixture-user cwd must not be a preflight block).

### Non-blocking

- GnuWin32 make 3.81 is on the default PATH; MSYS2 make 4.4.1 was prepended.
- WinGet `goreleaser.exe` / `syft.exe` shims were 0-byte; real package binaries were prepended.
- `doctor` reports config-missing (expected without `init`) while agents are SUPPORTED.
- First `make verify` under `Start-Process` redirected pipes failed `bufio: buffer full` during test-race; superseded by Tee-Object rerun exit 0. Harness flake, not a product FAIL.
- Host global Claude Code is `2.1.231`; matrix used an acceptance-tree PATH pin of `2.1.229`. Host global install was not downgraded.

### Test-harness deviations and supersessions

- First R4 used `doctor`, which does not probe hanging `--version` (exit 3 in 21 ms, no UNTESTED). That 21 ms result is classification FAIL if used alone. Superseded by installed `handoff --dry-run` with a shim-only PATH: UNTESTED exit 5 in 10571 ms.
- First C8 kept vendor `claude` on PATH so a PATH echo of `2.1.230` was not the LookPath probe. Product versions come from agent-root `version` and/or agentcheck `--version`. Superseded by `version` file `2.1.230` plus PATH-only shim: exit 5, lineage unchanged.
- First C3 invoked compaction session `000000000001` from Demo and never used a missing same-OS workspace. Foreign `/Users/fixture-user/...` cwd is C5. Superseded by session `c30000000001` whose recorded cwd is a missing Windows path: exit 5; Windows Demo control exit 0.
- First C4 used the remapped macOS-cwd fixture from Other (C5) and did not prove `$other` as a different git repo. Superseded: Demo vs Other distinct toplevels; Demo-cwd session from Other exit 5 `different repository`; from Demo exit 0.
- First F4/F6 parsed JSON field `id` instead of `handoff_id`. Harness. Superseded by inspect/export of stored `handoff_id` exit 0.
- First E1/E2 overlay used `GEMINI_CLI_HOME/tmp/session-*.jsonl` (zero sessions). Layout `tmp/projhashfixture/chats` indexed E1. E2 still FAIL as a product/preflight result.
- OpenCode skipped (`NOT TESTED: no isolation override`); executable omitted from the test PATH.
- Real TTY dest-ack and picker rows were attempted via `ssh -t` and `ssh -tt` and were not invented.

Report-only diff: this file. Merge-base with TEST_COMMIT: this branch is created from `0d7551a69918a97967f927ed9dc5a56b3583108c`. No product files changed.

## 9. Required terminated device block

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.5
test_commit=0d7551a69918a97967f927ed9dc5a56b3583108c
installed_binary_sha256=3b126e60f61c1bc22c32ccbfa7e1bb34dc6d47be0d9a1f2d51f2f2c823ad840d
required_pass=38
required_partial=0
required_fail=6
required_not_tested=0
optional_pass=4
optional_fail=2
optional_not_tested=2
artifact_chain=PASS
isolation_privacy=PASS
flagship_directions=FAIL
fidelity_policy=PASS
workspace_capability=PASS
security=PASS
cli_contract=FAIL
performance=PASS
phase1_phase2_phase3_regression=PASS
release_blocking_findings=4
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
