# Reinstate Phase 1 RC3 native-Windows acceptance report

- **Recorded at:** 2026-07-26T05:53:07+05:30 (Asia/Calcutta)
- **Last updated:** 2026-07-26T09:08:59+05:30
- **Execution state:** W1 complete; Codex restore/resume passed and Claude mapped-project restore/resume failed
- **Verdict:** **FAIL**

## Executive summary

The native-Windows environment is 64-bit Windows 11 Pro running native Windows
PowerShell 5.1, not WSL. Codex CLI `0.145.0` and Claude Code `2.1.220` are both
within the RC3-supported ranges. The initial Claude Code `2.1.211` prerequisite
blocker was cleared by a human-approved, exact-version npm upgrade; npm package
ownership, `claude --version`, and `claude doctor` verified the result.

The disposable project was created without initializing Reinstate. The live
public PowerShell bootstrap returned HTTP 200 and was executed twice without
elevation. Both installer checksum layers succeeded, `rein` and `reinstate`
resolved under the expected user-local directory, the runtime reported exact
RC3, and the normalized user PATH count remained one after the second run.
Before init, `rein setup check` returned the mandatory exit code 3 with
`config missing`; platform, both agent adapters, and the OS keyring passed.

PR #12, its CI/security checks, the signed merge commit, and the RC3 release
workflow were verified through GitHub metadata. Mac report PR #13 records
sections 2-9 passing for the original profile. Two later profiles were retired
after an operator passphrase exposure and a cross-device decryption mismatch;
their homes and remote prefixes remain untouched. A controlled third profile
then passed Mac setup/self-test, scoped dry-runs and pushes, and a status
roundtrip proving exactly the two selected sessions.

On Windows, the controlled-fresh init returned exit `0`. The required wrong
passphrase returned exit `4` with explicit decryption refusal and no target or
backup mutation; the correct passphrase then returned exit `0` with the exact
two-session manifest. Setup check and synthetic self-test passed. Both scoped
pull dry-runs created no agent file or backup, and both real pulls returned exit
`0` with exact target files present.

Codex exact-ID native resume returned exit `0` in the mapped Windows project,
and its A1 marker was visually confirmed by the human operator. Claude did not:
the restored file was written below a directory retaining the Mac source
project slug, direct-ID resume returned exit `1`, and neither mapped-project nor
all-project vendor discovery found the selected ID. This reproducible
cross-platform Claude sync/resume defect is release-blocking
`RC3-WIN-F3`; therefore the W1 verdict is **FAIL**. No section 14 action was
performed.

## Release identity

| Field | Value | Evidence source |
| --- | --- | --- |
| Release | `v0.1.0-rc.3` | Git tag, release API, tagged runbook |
| Tag commit | `94cc1e23f2e67054cd6102180664d83776d2406f` | Fetched tag |
| Latest `origin/main` at final report base | `c31bc82b9737b701b22851aa4e56e39ed58e25b1` | Final `git fetch` and branch rebase |
| Signed merge commit | `94cc1e23f2e67054cd6102180664d83776d2406f` | GitHub verification: `verified=true`, `reason=valid` |
| Fix PR | [#12](https://github.com/HarjjotSinghh/reinstate/pull/12), merged | GitHub PR API |
| PR head | `8832244f812b3fa8c0f6f0033872a9f1adcbbf11` | GitHub PR workflow metadata |
| Release workflow | [30179243906](https://github.com/HarjjotSinghh/reinstate/actions/runs/30179243906), success | GitHub Actions API |
| Release publication | 2026-07-25T23:26:01Z, prerelease | GitHub release API |

The local GPG client could not independently verify the commit because the
signer's public key was unavailable, and local tag verification lacked an SSH
allowed-signers configuration. This is recorded honestly rather than treated as
a local PASS. GitHub reported the merge commit signature valid, and the release
workflow's `Validate signed tag, main ancestry, and changelog` step passed.

## Sanitized Windows environment

| Field | Actual | Gate |
| --- | --- | --- |
| Windows edition | Microsoft Windows 11 Pro, 25H2 | PASS |
| OS version/build | `10.0.26200.8328` | PASS |
| OS architecture | 64-bit | PASS |
| Process architecture | AMD64, 64-bit process | PASS |
| Shell | Native Windows PowerShell `5.1.26100.8328` Desktop | PASS |
| PowerShell executable | `%WINDIR%\System32\WindowsPowerShell\v1.0\powershell.exe` | PASS |
| WSL indicator | `WSL_DISTRO_NAME` not set | PASS |
| Claude Code | `2.1.220` | PASS: expected `2.1.219`-`2.1.220` |
| Codex CLI | `0.145.0` | PASS: expected `0.133.0`-`0.145.0` |
| Git | `2.52.0.windows.1` | PASS |
| Reinstate | `0.1.0-rc.3`, commit `94cc1e2` | PASS |

`Get-ComputerInfo` returned the legacy registry product label `Windows 10 Pro`;
`Win32_OperatingSystem` reported `Microsoft Windows 11 Pro`, and the version,
build, architecture, display version, and edition ID otherwise agreed.

## Isolated paths and report workspace

| Purpose | Sanitized path | Final W1 state |
| --- | --- | --- |
| Preferred `REINSTATE_HOME` | `%USERPROFILE%\.reinstate-phase1-acceptance` | Retired after incident; retained without deletion |
| RC3 fallback `REINSTATE_HOME` | `%USERPROFILE%\.reinstate-phase1-acceptance-rc3` | Corrected W1 config/state initialized |
| Controlled fresh `REINSTATE_HOME` | `%USERPROFILE%\.reinstate-phase1-acceptance-rc3-r2` | Initialized with controlled fresh profile; config/state present |
| Preferred disposable project | `%USERPROFILE%\Projects\reinstate-phase1-acceptance` | Created, writable Git repository |
| RC3 fallback disposable project | `%USERPROFILE%\Projects\reinstate-phase1-acceptance-rc3` | Absent; not created |
| Canonical project ID | `local/reinstate-phase1-acceptance` | Unchanged for corrected W1 |
| Expected install directory | `%LOCALAPPDATA%\Programs\Reinstate\bin` | Created by public installer |
| Report repository | `D:\Projects\reinstate` | Only this Windows report changed |
| Report branch | `test/phase1-rc3-windows-report` | Created from latest `origin/main` |

Neither preferred path had stale initialized state. The preferred project was
created and initialized with Git; its README is the only disposable-project
file and remains uncommitted in that disposable repository. The first two
Reinstate homes were retained after test-state incidents. The controlled-fresh
home was initialized and used for the completed W1 flow without deleting or
overwriting either prior home.

## Profile and session metadata

| Field | Value |
| --- | --- |
| Retired Phase 1 profile ID | `47e43f49-35ea-49b1-a269-fb7cd8ee41a8` |
| Replacement Phase 1 profile ID | `72733dd3-ee7c-45bd-89a6-5e448108367f` |
| Controlled fresh Phase 1 profile ID | `9e17efb0-400e-4002-b986-b79f3b7b08e5` |
| Canonical project ID | `local/reinstate-phase1-acceptance` |
| Claude session ID | `a36153a6-d70a-43ec-8dcf-7a3c6787ac56` |
| Codex session ID | `019f9b4d-d8d2-79e1-9e23-810786676f5a` |
| Original handoff Claude snapshot ID | `c0a8c645-733e-4fd1-ab7c-e9174a8368a3` |
| Original handoff Codex snapshot / remote revision | `26897d25-4fbc-41fd-bb26-f678afc0e38e` |
| Controlled fresh Claude snapshot ID | `05c43895-0b73-427b-b30c-2ca1b243690f` |
| Controlled fresh Codex snapshot ID | `93ff151c-6511-4c01-bf6f-55c0a5b7b94e` |
| Controlled fresh remote revision | `93ff151c-6511-4c01-bf6f-55c0a5b7b94e` |
| Selected remote sessions | `2`, proven by controlled fresh Mac status |
| Mac setup/self-test evidence | PASS, provided by Mac handoff |
| Controlled-fresh remote ciphertext evidence | Not tested; byte-level marker-absence inspection was not supplied |

No transcript or agent-file contents are retained, quoted, or used as report
evidence. Resume evidence is limited to human-confirmed marker booleans and
sanitized vendor errors.

The marker sentence accompanying the handoff was malformed. Resume validation
therefore used the two canonical A1 expectations from tagged runbook §7 and
recorded only human-confirmed presence booleans.

## Command and gate evidence

`N/E` means not executed. It is not a PASS.

| Command or gate | Result | Exit | Expected | Sanitized actual evidence |
| --- | --- | ---: | --- | --- |
| Read complete tagged runbook | PASS | 0 | RC3 tagged document read before acceptance commands | 18,384 UTF-8 bytes read from the tagged raw URL |
| Read complete tagged Codex setup prompt | PASS | 0 | Prompt version 3 pinned to RC3 | 4,310 UTF-8 bytes read from the tagged raw URL |
| Native PowerShell / Windows / architecture inventory | PASS | 0 | Native 64-bit Windows, not WSL | PowerShell 5.1 Desktop; Windows 11 Pro build 26200.8328; OS and process 64-bit; AMD64; no WSL indicator |
| Initial `claude --version` | BLOCKED, resolved | 0 | `2.1.219`-`2.1.220` | Initially `2.1.211`; no acceptance mutation followed until corrected |
| Approved pinned Claude upgrade | PASS | 0 | Exact supported version | `npm install -g @anthropic-ai/claude-code@2.1.220`; two packages changed |
| Final Claude package/version/doctor | PASS | 0 | `2.1.220`, healthy npm-global install | npm owns `2.1.220`; CLI reports `2.1.220`; doctor reports win32-x64 and no installation issues |
| `codex --version` | PASS | 0 | `0.133.0`-`0.145.0` | `0.145.0` |
| `git --version` | PASS | 0 | Git installed | `2.52.0.windows.1` |
| Inspect preferred/fallback isolation paths | PASS | 0 | Do not overwrite previous state | All four paths absent; no config found |
| Create disposable project and `git init` | PASS | 0 | New writable Git project | Preferred project created; write probe passed; `.git` and README exist |
| `HEAD https://reinstate.dev/install.ps1` | PASS | 0 | HTTP 200 | HTTP 200 |
| Inspect public bootstrap pin/contract | PASS | 0 | Exact RC3 tag, canonical tagged installer, checksum before execution, no latest resolver | RC3 pin present; tagged raw URL present; pinned SHA present; release base cleared by bootstrap |
| Public bootstrap to canonical installer SHA-256 | PASS | 0 | Hashes equal | Public pin and downloaded canonical installer both `4ac266d4f59ff60f70d8da463d33751546cab8121e12769c0b00d0004c5d6050` |
| RC3 Windows release archive checksum | PASS | 0 | Downloaded archive equals `checksums.txt` entry | `reinstate_0.1.0-rc.3_windows_amd64.zip`, 6,149,978 bytes; expected and actual `fa423ef4acbd9a57d740e62b305c54ee3d641f0e95c683cf65adfff844aa114e` |
| First `irm https://reinstate.dev/install.ps1 \| iex` | PASS | 0 | No elevation; both checksums; install exact RC3 | Non-elevated process; no elevation prompt; `installer checksum ok`; `checksum ok`; both aliases installed |
| Run public installer a second time | PASS | 0 | Same version; idempotent | Both checksums passed; reported RC3 already installed |
| `(Get-Command rein).Source` | PASS | 0 | Under expected install directory | `%LOCALAPPDATA%\Programs\Reinstate\bin\rein.exe` |
| `(Get-Command reinstate).Source` | PASS | 0 | Under expected install directory | `%LOCALAPPDATA%\Programs\Reinstate\bin\reinstate.exe` |
| `rein version --json` | PASS | 0 | `"version":"0.1.0-rc.3"` | Version `0.1.0-rc.3`, commit `94cc1e2`, build date `2026-07-25T23:21:41Z` |
| Normalized user PATH count | PASS | 0 | `1` after two installs | `0` before first run, `1` after first run, `1` before and after second run |
| Pre-init `rein setup check` | PASS | 3 | Exit 3 and `config missing` | Exact exit 3; config missing; device, Claude, Codex, and keyring checks OK |
| Fetch `origin/main` and tags | PASS | 0 | Latest report base and RC3 tag available | RC3 tag resolves to signed merge SHA; report branch rebased to later `origin/main` commit `c31bc82` |
| PR #12 checks | PASS | 0 | Required PR checks green | `gh pr checks 12` returned exit 0; all listed checks passed |
| RC3 release workflow | PASS | 0 | Required release gates green | Completed with conclusion `success` on the signed merge SHA |
| Local PowerShell docs check | PASS | 0 | Repository doctests pass | `scripts/check-docs.ps1`: `internal/doctest` passed in fresh W0 verification |
| Final report PowerShell docs check | PASS | 0 | Repository doctests pass on final report branch base | `scripts/check-docs.ps1`: `internal/doctest` passed after rebasing to latest `origin/main` |
| Validate `MAC-RC3-M1` identifiers | PASS | N/A | RC3, profile, canonical project, two session IDs, two selected sessions, Mac setup/self-test | Required non-secret identifiers and counts supplied |
| Fresh W1 Windows/isolation inventory | PASS | 0 | Native 64-bit Windows; supported adapters; no initialized isolated home or selected local sessions | Windows 11 Pro build 26200; Claude `2.1.220`; Codex `0.145.0`; selected local-file counts `0/0`; isolated backups `0` |
| Fresh W1 installer executions | PASS | 0 | HEAD 200; both checksums; exact RC3; no elevation; idempotent PATH | Two live runs reported both checksums and RC3 already installed; non-elevated; PATH `1` before and after |
| Fresh W1 pre-init `rein setup check` | PASS | 3 | Exit 3 and only config missing | Device, both adapters, and keyring OK; isolated home still absent |
| Private additional-device `rein init` | PASS | 0 | Exact Mac profile ID and canonical Windows project mapping | Operator entered storage credentials privately; expected profile ID returned; config and state files created |
| Wrong-passphrase mutation baseline | PASS | 0 | No selected agent target and no isolated backup before negative test | Selected Claude/Codex file counts `0/0`; isolated backup-file count `0`; no file contents read |
| Wrong-passphrase `rein status` | PASS | 4 | Non-zero decryption/authentication refusal | Incorrect-identity/passphrase refusal; no selected target created and no backup created |
| Post-refusal no-mutation check | PASS | 0 | Selected agent targets and isolated backups unchanged | Selected Claude/Codex file counts remained `0/0`; isolated backup-file count remained `0` |
| Retired-profile intended-passphrase status | RETIRED | 4 | Two remote sessions | Decryption refusal repeated; intended value was subsequently exposed outside a hidden prompt, so the profile was retired |
| Replacement profile/home validation | PASS | 0 | Retired home retained; fresh RC3 home unused; same two targets absent | Old config retained; new home/config absent; selected Claude/Codex file counts `0/0` |
| Replacement-profile `rein init` | PASS | 0 | Replacement profile and unchanged canonical project | Expected profile ID returned; new config/state created; retired home retained |
| Replacement wrong-passphrase baseline | PASS | 0 | No selected targets or new-home backups | Selected Claude/Codex file counts `0/0`; new-home backup-file count `0`; no contents read |
| Replacement wrong-passphrase `rein status` | PASS | 4 | Non-zero decryption/authentication refusal | Two deliberate attempts both refused; final recorded exit `4` |
| Replacement post-refusal no-mutation check | PASS | 0 | Selected targets and replacement-home backups unchanged | Selected Claude/Codex file counts remained `0/0`; replacement-home backup-file count remained `0` |
| Replacement intended-passphrase `rein status` | RETIRED | 4 | Exactly two remote sessions | Operator-confirmed intended replacement phrase was refused; Windows config profile ID was correct; selected targets/backups remained `0/0/0` |
| Mac replacement-profile `rein status` proof | RETIRED | 4 | Exit 0 and exactly two selected sessions | Two Mac attempts with the intended replacement phrase produced the same decryption refusal; profile superseded by controlled-fresh success |
| Mac replacement-home profile check | PASS | 0 | Mac home points to replacement profile | Config reports `72733dd3-ee7c-45bd-89a6-5e448108367f`; wrong-home hypothesis eliminated |
| Controlled fresh Mac `rein init` | PASS | 0 | New unused home and new profile; canonical project unchanged | `%USERPROFILE%`-independent Mac home suffix `-rc3-r2`; profile `9e17efb0-400e-4002-b986-b79f3b7b08e5`; prior homes/prefixes retained |
| Controlled fresh Mac `rein setup check` | PASS | 0 | Config/device/keyring OK; both adapters supported | Darwin arm64; Claude `2.1.220` and Codex `0.145.0` SUPPORTED |
| Controlled fresh Mac `rein doctor --self-test` | PASS | 0 | Synthetic self-test passes | All checks passed; `self_test: ok` |
| Controlled fresh Mac scoped Claude push dry-run | PASS | 0 | Selected Claude ID only; no remote mutation | Reported `1` planned snapshot and `dry_run=true`; wording caveat `RC3-MAC-F3` retained |
| Controlled fresh Mac scoped Claude push | PASS | 0 | Exactly one selected Claude snapshot uploaded | Reported `1` pushed, `0` skipped, and `dry_run=false`; snapshot UUID not printed by command |
| Controlled fresh Mac status after Claude push | PASS | 0 | New passphrase decrypts the new manifest; exactly one expected remote session | Revision `05c43895-0b73-427b-b30c-2ca1b243690f`; Claude ID matched; Codex absent as expected before its push |
| Controlled fresh Mac scoped Codex push dry-run | PASS | 0 | Selected Codex ID only; no remote mutation | Reported `1` planned snapshot and `dry_run=true`; wording caveat `RC3-MAC-F3` retained |
| Controlled fresh Mac scoped Codex push | PASS | 0 | Exactly one selected Codex snapshot uploaded | Reported `1` pushed, `0` skipped, and `dry_run=false`; snapshot UUID not printed by command |
| Controlled fresh Mac final `rein status` | PASS | 0 | New passphrase decrypts manifest containing exactly the two selected IDs | Revision `93ff151c-6511-4c01-bf6f-55c0a5b7b94e`; expected Claude and Codex IDs only |
| Controlled fresh Windows-home inspection | PASS | 0 | New unused home; existing disposable project retained without disturbance | `%USERPROFILE%\.reinstate-phase1-acceptance-rc3-r2` absent; project Git repository exists with its prior uncommitted README untouched |
| Controlled fresh Windows `rein init` | PASS | 0 | Exact controlled-fresh profile ID and unchanged canonical project mapping | Profile `9e17efb0-400e-4002-b986-b79f3b7b08e5`; config/state created; credentials entered privately and stored by reference |
| Controlled fresh Windows wrong-passphrase baseline | PASS | 0 | No selected targets and no new-home backups | Selected Claude/Codex filename counts `0/0`; new-home backup-file count `0`; no contents read |
| Controlled fresh Windows wrong-passphrase `rein status` | PASS | 4 | Non-zero decryption/authentication refusal | Explicit incorrect-identity/passphrase refusal |
| Controlled fresh Windows post-refusal no-mutation check | PASS | 0 | Selected targets and new-home backups unchanged | Selected Claude/Codex filename counts remained `0/0`; new-home backup-file count remained `0` |
| Controlled fresh Windows correct-passphrase `rein status` | PASS | 0 | Exact remote revision and exactly the two selected sessions | Revision `93ff151c-6511-4c01-bf6f-55c0a5b7b94e`; expected Claude and Codex IDs only |
| Controlled fresh Windows `rein setup check` | PASS | 0 | Config/device/keyring OK; both adapters supported | Windows amd64; Claude `2.1.220` and Codex `0.145.0` SUPPORTED |
| Controlled fresh Windows `rein doctor --self-test` | PASS | 0 | Synthetic self-test passes | All checks passed; `self_test: ok` |
| Pre-Codex-dry-run mutation baseline | PASS | 0 | No selected targets and no new-home backups | Selected Claude/Codex filename counts `0/0`; backup-file count `0`; Codex processes were still open, so no pull was attempted |
| Codex-closure guard before dry-run | NOT EXECUTED | N/E | Process count `0` before invoking pull | Process count was `13`; local guard printed STOP and did not invoke `rein pull` |
| Codex-closure guard retry before dry-run | NOT EXECUTED | N/E | Process count `0` before invoking pull | Process count was `1`; repeated local guards printed STOP and did not invoke `rein pull` |
| Controlled fresh Windows scoped Codex pull dry-run | PASS | 0 | Selected Codex ID only; destination/backup root reported; no mutation | Pull executed only after refreshed process count reached `0`; reported one planned snapshot and `dry_run=true` |
| Post-Codex-dry-run no-mutation check | PASS | 0 | No selected target file and no backup created | Exact reported target absent; selected Codex filename count `0`; reported backup root contains `0` files |
| Controlled fresh Windows scoped Codex pull | PASS | 0 | All Codex processes closed; exactly one selected snapshot restored | Process count `0`; reported one pulled snapshot and `dry_run=false` |
| Post-Codex-pull metadata check | PASS | 0 | Exact target exists; first restore creates no backup file | Exact reported target exists; backup-file count remains `0`; transcript not opened |
| Pre-Claude-dry-run mutation baseline | PASS | 0 | No selected Claude target; no backup file; all Claude processes closed | Selected Claude filename count `0`; backup-file count `0`; Claude process count `0` |
| Controlled fresh Windows scoped Claude pull dry-run | PASS | 0 | Selected Claude ID only; destination/backup root reported; no mutation | Claude process count `0`; reported one planned snapshot and `dry_run=true`; destination retained the sanitized Mac source-project slug |
| Post-Claude-dry-run no-mutation check | PASS | 0 | No selected target file and no backup created | Exact reported target absent; selected Claude filename count `0`; backup-file count `0` |
| Controlled fresh Windows scoped Claude pull | PASS | 0 | All Claude processes closed; exactly one selected snapshot restored | Claude process count `0`; reported one pulled snapshot and `dry_run=false` |
| Post-Claude-pull metadata check | PASS | 0 | Both exact selected targets exist; first restores create no backup file | Exact Claude and Codex target paths exist; backup-file count remains `0`; transcripts not opened |
| `rein list --agent claude/codex` | NOT EXECUTED | N/E | Both selected IDs discoverable without unrelated-session access | RC3 `rein list` has no `--session` filter; explicit safety scope prohibited unscoped enumeration, so exact target metadata and exact-ID vendor resume were used instead |
| Non-runbook Claude direct-ID resume diagnostic | FAIL | 1 | Diagnostic only; tagged §13 requires interactive `claude --resume` | Direct-ID form reported no conversation for the selected ID; restored file still exists under the sanitized Mac source-project slug; no Claude process remained |
| Tagged §13 Claude interactive resume from mapped project | FAIL | N/A | Selected session appears and resumes from the mapped Windows project | UI reported no conversations in the current mapped project; screenshot contains no transcript content |
| Claude all-project selected-ID search | FAIL | N/A | Selected restored ID remains discoverable through the vendor UI | Exact selected ID returned no match; screenshot contains no transcript content; A1 not testable |
| Tagged §13 Codex interactive resume from mapped project | FAIL | N/A | Selected session appears and resumes from the mapped Windows project | With picker filter `Cwd`, exact selected ID returned no results; screenshot contains no transcript content |
| Codex all-session UUID search | INCONCLUSIVE | N/A | Exact selected restored ID is globally discoverable | Search returned a different session containing the query text; no transcript content retained |
| Codex picker false-positive resume | NOT A GATE | 0 | N/A | Vendor command exited `0`, but human confirmed it was not the selected restore; superseded by exact-ID resume |
| Codex exact-ID native resume | PASS | 0 | Exact selected session resumes in the mapped Windows project and A1 is visually confirmed | `codex resume SESSION_ID` opened the mapped Windows project; marker visually confirmed by human; no transcript content retained |

## Installer, idempotence, and PATH evidence

Locally verified read-only:

- Public bootstrap HEAD status is 200.
- Public bootstrap SHA-256 is
  `f6b881f0ac58fdb2e02752bc7cafa2eff72e29f4a00fdb43cd7fd18f287f5d39`.
- Its pinned canonical-installer SHA-256 matches the downloaded exact-tag
  installer.
- The downloaded RC3 Windows archive matches both `checksums.txt` and the
  GitHub release asset digest.
- The expected install directory did not exist, neither CLI alias resolved, and
  the normalized pre-install user PATH count was zero before execution.

Locally verified through live execution:

- The first bootstrap ran non-elevated with no elevation prompt and printed both
  checksum-success messages.
- Both aliases were installed under the expected directory and are
  byte-identical.
- Runtime version is exactly `0.1.0-rc.3`.
- The second bootstrap printed both checksum-success messages and reported the
  same version already installed.
- The normalized user PATH count was exactly one after both runs.

## Wrong-passphrase and no-mutation evidence

The authoritative controlled-fresh negative test passed: `rein status` returned
exit `4` with an explicit decryption refusal. Before and after the refusal,
selected Claude/Codex target counts were `0/0`, and the fresh-home backup-file
count was `0`. The next separately invoked status command accepted the fresh
correct passphrase, returned exit `0`, and reported exactly the two selected
remote IDs at revision `93ff151c-6511-4c01-bf6f-55c0a5b7b94e`.

The earlier retired profile also refused a wrong phrase without mutation, but
its intended phrase was later exposed outside Reinstate's hidden prompt and is
excluded from this report. A replacement profile then refused the
operator-intended phrase on both devices. The controlled third profile passed
an immediate Mac encrypt/decrypt roundtrip and the Windows negative/positive
pair, resolving that incident as test-state or passphrase mismatch rather than
a demonstrated RC3 encryption defect. Both prior homes and prefixes remain
untouched.

## Resume evidence

Claude Code's direct-ID diagnostic returned exit `1` with no conversation found.
The authoritative interactive `claude --resume` UI reported no conversations in
the mapped Windows project. Its all-project search returned no match for the
exact selected ID. The restored file exists, but its parent directory retains
the sanitized Mac source-project slug rather than the mapped Windows project
context. No transcript content was inspected. Claude A1 is not testable; the
Codex mapped-project picker returned no selected-session result. Its all-session
UUID search produced a false-positive different session. The supported exact-ID
form, `codex resume SESSION_ID`, then resumed the selected session with exit
`0` in the mapped Windows project, and the human visually confirmed Codex A1.
No transcript content from either session is retained or reported.

## Active-agent refusal evidence

Not tested. This is a W2 gate.

## Backup evidence

Both W1 dry-runs reported the backup root but produced zero backup files and no
agent target. Both initial real restores wrote previously absent targets, so no
pre-existing target required backup; the backup-file count remained zero. The
mandatory timestamped overwrite backup is a section 14/W2 gate and was not
tested.

## Conflict and keep-both evidence

Not tested. This is a W3/W4 gate.

## Automated GitHub integrity gates

These are GitHub-provided results, not locally rerun Windows acceptance:

| Automated gate | Result | Evidence |
| --- | --- | --- |
| Ubuntu Go test | PASS | PR CI run `30178945983`, `Test (ubuntu-latest)`, test step success |
| macOS Go test | PASS | PR CI run `30178945983`, `Test (macos-latest)`, test step success |
| Windows Go test | PASS | PR CI run `30178945983`, `Test (windows-latest)`, test step success |
| Native Windows bootstrap/PATH contract | PASS | Windows `go test ./... -count=1` step success includes `TestWindowsPublicBootstrapContract` and hash-mismatch refusal |
| POSIX bootstrap/hash-mismatch contracts | PASS | Ubuntu/macOS test steps success include the POSIX contract tests |
| Exact-tag/no-`latest` static contracts | PASS | Cross-platform doctest suite passed; live bootstrap also read-only inspected |
| Website `npm ci`, tests, production build | PASS | PR Website job steps succeeded |
| Installer bytes in Astro/Vercel output | PASS | `Verify public installer assets` step succeeded |
| Lint | PASS | PR Lint job succeeded |
| Race | PASS | Ubuntu `Race` step succeeded |
| Docs POSIX | PASS | Ubuntu/macOS docs steps succeeded |
| Docs PowerShell | PASS | Windows docs step succeeded |
| Fixture secret scan | PASS | Matrix fixture-scan steps succeeded |
| Vulnerability scan | PASS | `govulncheck` step succeeded |
| CodeQL / dependency / secret / workflow security | PASS | Security run `30178945993` and external listed checks succeeded |
| Release artifact validation | PASS | Release run `30179243906`, `Validate release artifacts` succeeded |
| POSIX release installer verification | PASS | Release run `30179243906`, POSIX installer step succeeded |
| PowerShell release verification | PASS | Release run `30179243906`, PowerShell verification step succeeded |
| Signed tag, main ancestry, changelog | PASS | Release workflow validation step succeeded |

## Evidence provenance

### Locally verified on native Windows

- tagged runbook and Codex prompt retrieval;
- OS, process architecture, native shell, and tool versions;
- absence of preferred/fallback isolated state;
- exact supported Claude upgrade and post-upgrade diagnostics;
- disposable-project creation and Git initialization;
- public bootstrap HTTP response, static pinning contract, and two live runs;
- canonical-installer and release-archive SHA-256 checks;
- alias locations, runtime version, PATH idempotence, and pre-init exit 3;
- pre-install command/install-directory/PATH state;
- controlled-fresh init, wrong/correct passphrase status, setup check, and
  self-test;
- scoped Claude/Codex pull dry-runs, no-mutation checks, and real restores;
- exact selected target-file and backup-file presence counts;
- Claude vendor discovery failure and Codex exact-ID resume success; and
- fetched Git/tag identities and report-branch base.

### Provided by GitHub

- PR, commit-signature, release, CI, security, and release-workflow metadata.

### Provided by Mac handoff and Mac report PR #13

- Mac sections 2-9 installer/setup evidence from PR #13.
- Controlled-fresh profile ID, canonical project ID, both selected session IDs,
  scoped dry-run/push results, both snapshot IDs, remote revision,
  selected-session count `2`, and PASS status for Mac setup check and self-test.
- Mac-side finding `RC3-MAC-F2` is carried as a provided caveat: Codex CLI
  `0.145.0` reportedly stopped persisting newly created sessions on that host.
  Cross-platform relevance is not assumed and will be reported only if
  reproduced on Windows.
- Mac-side finding `RC3-MAC-F3` is carried as a provided caveat: dry-run wording
  may say `pushed N snapshot(s)` while also saying dry-run/no mutation.
  Windows dry-runs will be judged by actual file/backup mutation evidence.

### Not tested

- byte-level ciphertext marker-absence inspection for the controlled-fresh
  remote prefix;
- Windows-to-Mac synchronization, active-agent refusal, overwrite backups,
  unchanged pushes, divergence, conflict, and keep-both recovery (sections
  14-17); and
- Phase 1 final sign-off beyond the assigned W1 stopping point.

## Section 19 final sign-off checklist

| Gate | Result | Evidence |
| --- | --- | --- |
| `install.sh` returns 200 and installs RC3 on Mac | PASS | Mac report PR #13, sections 2-9 |
| `install.ps1` returns 200 and installs RC3 on Windows | PASS | HEAD 200; two checksum layers passed; exact RC3 installed |
| Both installers are idempotent and PATH-safe | PASS | Mac report PR #13 plus Windows two-run/PATH-count evidence |
| Pre-init missing-config failure is accurate | PASS | Exit 3; `config missing`; all other checks OK |
| Post-init setup check and self-test pass on both devices | PASS | Controlled-fresh Mac and native-Windows checks exited 0; both adapters supported |
| Claude setup prompt completes on the Mac | PASS | Mac report PR #13 and controlled-fresh scoped push evidence |
| Codex setup prompt completes on Windows | PASS | Tagged prompt followed; controlled-fresh init/setup/self-test and scoped Codex restore completed |
| Only two selected test sessions reach the remote manifest | PASS | Controlled-fresh Mac status: exact two IDs at revision `93ff151c-6511-4c01-bf6f-55c0a5b7b94e` |
| Remote manifest/snapshots are ciphertext-only | NOT TESTED | Controlled-fresh prefix did not receive byte-level marker-absence inspection |
| Wrong passphrase fails without mutation | PASS | Exit 4; selected targets `0/0`; isolated backups `0` before and after |
| Claude Mac-to-Windows resume succeeds | FAIL | Restored file exists but is undiscoverable by Claude; `RC3-WIN-F3` |
| Codex Mac-to-Windows resume succeeds | PASS | Exact-ID resume exit 0; mapped project opened; A1 visually confirmed |
| Active-agent overwrite is refused | NOT TESTED | Section 14/W2 is outside this assignment |
| Existing Windows target is backed up before restore | NOT TESTED | Section 14/W2 is outside this assignment |
| Claude Windows-to-Mac resume succeeds | NOT TESTED | Section 15/W2 is outside this assignment |
| Codex Windows-to-Mac resume succeeds | NOT TESTED | Section 15/W2 is outside this assignment |
| Existing Mac targets are backed up before restore | NOT TESTED | Section 15/W2 is outside this assignment |
| Unchanged pushes skip without new snapshots | NOT TESTED | Section 16 is outside this assignment |
| Divergence records a conflict without overwrite | NOT TESTED | Section 17/W3-W4 is outside this assignment |
| `--keep-both` preserves both branches | NOT TESTED | Section 17/W4 is outside this assignment |
| All required GitHub checks are green | PASS | PR and release workflow metadata verified |

## Findings

### RC3-WIN-F1 — acceptance passphrase exposed outside the hidden prompt

- **Severity:** High for acceptance-test validity
- **Release-blocking:** No; no Reinstate defect has been established
- **Acceptance-blocking:** No; affected profile was retired and the full gate
  was repeated with a controlled-fresh profile
- **Sanitized reproduction:** After `rein status` returned a decryption refusal,
  the intended real passphrase was entered at the ordinary PowerShell prompt.
  PowerShell attempted to execute it as a command, and the terminal excerpt was
  subsequently included in chat.
- **Expected:** The passphrase is entered only while Reinstate's hidden prompt
  is active and is never placed in shell history, logs, or chat.
- **Actual:** The value left the hidden-prompt boundary and must be considered
  compromised. The value is intentionally omitted.
- **Exit code:** `rein status` returned `4`; the subsequent PowerShell command
  produced `CommandNotFoundException`.
- **Affected versions:** This RC3 acceptance run and profile
  `47e43f49-35ea-49b1-a269-fb7cd8ee41a8`.
- **Evidence:** Sanitized operator terminal output and post-failure metadata
  checks showing no selected-session or backup mutation.
- **Likely cause:** Input was sent after the Reinstate process had already
  exited, so the shell—not the hidden prompt—received it.
- **Recommended next investigation:** Keep the retired local/remote state for
  review, never reuse the exposed value, and retain the mandatory prompt-focus
  protocol used by the controlled-fresh rerun.

### RC3-WIN-F2 — replacement-profile decryption refusal on both devices

- **Severity:** Medium test-state mismatch
- **Release-blocking:** No; the controlled-fresh profile passed immediate
  encrypt/decrypt roundtrips on both devices
- **Acceptance-blocking:** No; W1 was repeated with a controlled-fresh profile
- **Sanitized reproduction:** Initialize the fresh Windows RC3 home with
  replacement profile `72733dd3-ee7c-45bd-89a6-5e448108367f`, run
  `rein status`, wait for the hidden prompt, and enter the operator-confirmed
  intended replacement passphrase.
- **Expected:** Exit 0 and exactly two selected remote sessions.
- **Actual:** Decryption refused with incorrect-identity/passphrase error on
  both Windows and Mac.
- **Exit code:** `4` on Windows; `4` on both recorded Mac attempts.
- **Affected versions:** Reinstate `0.1.0-rc.3` on native Windows,
  Claude Code `2.1.220`, and Codex CLI `0.145.0`.
- **Evidence:** Windows config points to the replacement profile; selected
  target counts remain `0/0`; replacement-home backup count remains `0`.
- **Likely cause:** A passphrase or test-state mismatch specific to this retired
  profile. The controlled-fresh success disproves a general RC3 or
  Windows-only decryption failure in this run.
- **Recommended next investigation:** Retain the profile/home/prefix without
  mutation for forensic review. No release action should depend on this
  superseded profile.

### RC3-WIN-F3 — Claude restore is stranded under the Mac project slug

- **Severity:** Critical Phase 1 sync/resume failure
- **Release-blocking:** Yes
- **Acceptance-blocking:** Yes
- **Sanitized reproduction:** From the correctly configured mapped Windows
  project, with all Claude Code processes closed, dry-run and pull only selected
  Claude session `a36153a6-d70a-43ec-8dcf-7a3c6787ac56`, then run the tagged
  `claude --resume` interface.
- **Expected:** The selected session is discoverable and resumes from
  `%USERPROFILE%\Projects\reinstate-phase1-acceptance`.
- **Actual:** Reinstate reports one successful restore, but writes the target
  below a directory retaining the sanitized Mac source-project slug. Claude's
  mapped-project resume UI reports no conversations. Direct-ID resume reports
  no conversation found, and the all-project UI returns no match for the exact
  selected ID.
- **Exit code:** Reinstate pull `0`; Claude direct-ID resume `1`; interactive UI
  has no command exit while open.
- **Affected versions:** Reinstate `0.1.0-rc.3`, Claude Code `2.1.220`, native
  Windows amd64 restoring a macOS-origin Claude session.
- **Evidence:** Pull output, exact-target existence check, correct Windows
  canonical-project mapping in `config.toml`, direct-ID refusal, and
  transcript-free interactive UI screenshot.
- **Likely cause:** The Claude restore planner uses the snapshot's archived
  relative path as the destination path, preserving the source-device project
  directory slug instead of deriving the destination from the canonical
  project's Windows `local_root`.
- **Recommended next investigation:** Add a cross-platform Claude restore test
  that maps the snapshot's canonical project ID to the destination vendor
  project directory and proves the normal Windows resume UI discovers the
  restored ID. Do not manually move this acceptance artifact.

### RC3-WIN-F4 — Codex picker UUID search can select the wrong session

- **Severity:** Low test-procedure ambiguity
- **Release-blocking:** No
- **Acceptance-blocking:** No
- **Sanitized reproduction:** Launch `codex resume` and enter a UUID in the
  picker search field when another local session contains that UUID as text.
- **Expected:** Acceptance operators can unambiguously select the exact session
  ID requested by the runbook.
- **Actual:** Picker search returned a different session containing the query
  text. The supported exact-ID form `codex resume SESSION_ID` subsequently
  resumed the correct selected session in the mapped Windows project.
- **Exit code:** False-positive resume `0`; exact-ID resume `0`.
- **Affected versions:** Codex CLI `0.145.0`; RC3 acceptance procedure.
- **Evidence:** Human mismatch confirmation followed by exact-ID resume and
  human A1 confirmation. No transcript content is retained or reported.
- **Likely cause:** The picker search is content-oriented rather than a strict
  session-ID lookup.
- **Recommended next investigation:** Update the acceptance procedure to use
  `codex resume SESSION_ID` when the non-secret UUID is already known.

The initial environment prerequisite mismatch (Claude Code `2.1.211`) was
resolved before acceptance mutation through a human-approved exact upgrade to
`2.1.220`. It is retained in the command evidence for auditability but is not a
Reinstate defect or an active finding.

## Current sign-off

Windows W0 and the assigned Windows W1 scope are complete. W1 is **FAIL**
because Claude's selected macOS-origin session cannot be discovered or resumed
from native Windows after a successful reported pull (`RC3-WIN-F3`). Codex
exact-ID restore/resume and A1 confirmation passed. Phase 1 sign-off remains
**OPEN**. No section 14 action was performed.
