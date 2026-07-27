# Immutable release, compatibility, and discoverability evidence

Use this template to create a new record for one exact release candidate or
stable tag. Store the completed record under a dated repository path such as
`docs/testing/results/`, not in this template.

A release claim is `PASS` only when the exact tagged commit, artifact, platform,
agent version, direction, and evidence are recorded. “Passed on main,” a local
build, a prior release, a screenshot without version context, or a successful
website build is not evidence for a released binary or physical compatibility
cell.

## Immutability and correction policy

1. Create the evidence record before final release sign-off.
2. Record the exact tag commit and SHA-256 of every attached evidence artifact.
3. Reference repository paths or durable release/workflow URLs; do not paste
   secrets, transcript contents, credentials, or unredacted user paths.
4. Once the tag is published and the record is signed off, do not rewrite its
   results to match later behavior.
5. Correct an error with a new, dated correction record that names the original
   record, exact incorrect field, reason, replacement evidence, reviewer, and
   effect on release claims.
6. A later release gets a new evidence record. It does not inherit a `PASS`
   merely because an earlier tag passed.

Git history alone is not a reason to silently edit published evidence. The
record and its digest registry are an auditable statement about one immutable
release.

## Evidence-state semantics

Use only:

- `PASS` — the exact gate ran against the exact subject and its immutable
  evidence is linked;
- `FAIL` — the exact gate ran and did not meet its acceptance criterion;
- `BLOCKED — <reason>` — the gate could not run because a prerequisite or
  authorized access is missing;
- `UNAVAILABLE — <reason>` — the evidence source cannot be accessed or does
  not exist;
- `NOT RUN` — no attempt was made; and
- `NOT APPLICABLE — <reason>` — the release does not target this gate.

Blank means not reviewed. `UNAVAILABLE`, `BLOCKED`, and `NOT RUN` never count as
`PASS`. A mandatory gate with any non-`PASS` state keeps that release gate
open. Do not use a fabricated zero, guessed version, or “looks good.”

## 1. Release identity

| Field | Value/evidence |
| --- | --- |
| Exact tag |  |
| SemVer without leading `v` |  |
| Release status: candidate/stable |  |
| Tag commit, full SHA |  |
| Prior release tag |  |
| Tag creation time, UTC |  |
| Tag signer identity |  |
| Tag-signature verification evidence |  |
| Main-branch ancestry evidence |  |
| Changelog heading/path |  |
| GitHub workflow run |  |
| GitHub release URL and draft/published state |  |
| Evidence author |  |
| Independent reviewer |  |
| Evidence record repository path |  |
| Evidence record SHA-256 after sign-off |  |

### Release-scope statement

State only behavior supported by the tagged diff and acceptance evidence:

> [Write the evidence-backed release scope.]

### Explicitly unclaimed

List open compatibility cells, incomplete release gates, roadmap features,
platforms, security claims, and product outcomes that this release does not
prove:

- [List each explicitly unclaimed cell, feature, or outcome.]

## 2. External-access disclosure

| Evidence/action | Required access | Authorized owner/operator | State | Date/time | Durable URL or sanitized repository path | Limitation |
| --- | --- | --- | --- | --- | --- | --- |
| Tag/release publication | GitHub release permission |  |  |  |  |  |
| GitHub Actions run and attestations | Repository Actions access |  |  |  |  |  |
| macOS physical acceptance | Physical device and agent accounts |  |  |  |  |  |
| Windows physical acceptance | Physical device and agent accounts |  |  |  |  |  |
| WSL2 acceptance, if in release scope | Physical Windows/WSL2 environment |  |  |  |  |  |
| S3/R2 acceptance prefix | Dedicated private bucket access |  |  |  |  |  |
| Website production deployment | Hosting access |  |  |  |  |  |
| Search Console inspection/submission | Verified domain property |  |  |  |  |  |
| Bing inspection/submission | Verified site |  |  |  |  |  |
| IndexNow production submission | Provisioned proof/key and network |  |  |  |  |  |
| CDN/WAF and verified crawler logs | Production edge/log access |  |  |  |  |  |
| Analytics/referral observation | Privacy-approved analytics access |  |  |  |  |  |

Repository code can prepare, validate, and document an external action. It does
not prove that the account-level action or physical-device test occurred.

## 3. Tagged-source and workflow provenance

| Gate | Subject | Command/workflow step | Acceptance criterion | Result | Evidence | Owner/reviewer |
| --- | --- | --- | --- | --- | --- | --- |
| SemVer tag | Exact tag | Release workflow tag validation | Matches the repository SemVer rule |  |  |  |
| Signed tag | Exact tag | `git verify-tag` with approved signers | Signature valid |  |  |  |
| Tag/HEAD identity | Tag commit | Release workflow | Tag commit equals checked-out release commit |  |  |  |
| Main ancestry | Tag commit | `git merge-base --is-ancestor` | Tagged commit belongs to approved main history |  |  |  |
| Clean source | Tagged checkout | `git diff --exit-code` | No release-time source mutation |  |  |  |
| Changelog | Exact version | Tagged `CHANGELOG.md` | Exact version heading exists and claims match diff |  |  |  |
| Go toolchain | Workflow | Pinned repository toolchain | Exact required toolchain used |  |  |  |
| Unit/integration tests | Tagged source | Release/CI workflow | Required tests pass |  |  |  |
| Race/lint/vulnerability/docs/fixture gates | Tagged source | Required CI jobs | Every mandatory job passes |  |  |  |
| Website tests/build | Tagged source | Website CI | Tests and production build pass |  |  |  |
| Installer contracts | Tagged source/artifacts | POSIX and PowerShell tests | Exact tag, checksum, mismatch refusal, PATH behavior pass |  |  |  |
| Artifact validation | Release distribution | `scripts/check-release-artifacts.sh` | Expected count, contents, checksums, source exclusions, SBOMs pass |  |  |  |
| Attestation | Released artifacts | GitHub artifact attestation step | Every required archive/SBOM/checksum subject attested |  |  |  |

Attach the workflow definition commit or digest. A green rerun after the tag
was moved is not acceptable; tags used for release evidence must not move.

## 4. Artifact registry

Fill one row for every expected and actual artifact. The target matrix is
defined by the tagged GoReleaser configuration; do not copy names from a
different release.

| Expected target/type | Exact filename | Bytes | SHA-256 from `checksums.txt` | Independently recomputed SHA-256 | SBOM filename/digest | Attestation/provenance URL | Version output tested | Result |
| --- | --- | ---: | --- | --- | --- | --- | --- | --- |
| macOS/darwin amd64 archive |  |  |  |  |  |  |  |  |
| macOS/darwin arm64 archive |  |  |  |  |  |  |  |  |
| Linux amd64 archive |  |  |  |  |  |  |  |  |
| Linux arm64 archive |  |  |  |  |  |  |  |  |
| Windows amd64 archive |  |  |  |  |  |  |  |  |
| Source archive |  |  |  |  | N/A or source SBOM policy |  | N/A |  |
| `checksums.txt` |  |  | N/A |  | N/A |  | N/A |  |

For each binary archive, verify the expected executable plus `LICENSE`,
`NOTICE`, `README.md`, and `CHANGELOG.md`. Verify the source archive contains
the module source and excludes `.git`, generated binaries, distribution
directories, and secret files.

An artifact absent from the release is `FAIL` when the tagged configuration
requires it. An unexpected artifact requires review; do not silently add it to
the matrix.

## 5. Installer and bootstrap evidence

| Gate | Platform/environment | Exact public URL/tag | Result | Exit code | Acceptance evidence | Failure-path evidence | Reviewer |
| --- | --- | --- | --- | ---: | --- | --- | --- |
| Installer URL returns ordinary `200` | macOS/POSIX |  |  |  |  |  |  |
| Installs exact tagged binary | macOS |  |  |  |  |  |  |
| Exact canonical installer checksum | macOS |  |  |  |  |  |  |
| Release archive checksum | macOS |  |  |  |  |  |  |
| Idempotent/PATH-safe install | macOS |  |  |  |  |  |  |
| Existing-version replacement refusal/approval | macOS |  |  |  |  |  |  |
| Checksum mismatch stops without replacement | macOS |  |  |  |  |  |  |
| Installer URL returns ordinary `200` | Native Windows PowerShell |  |  |  |  |  |  |
| Installs exact tagged binary | Native Windows amd64 |  |  |  |  |  |  |
| Exact canonical installer checksum | Windows |  |  |  |  |  |  |
| Release archive checksum | Windows |  |  |  |  |  |  |
| Idempotent/PATH-safe install | Windows |  |  |  |  |  |  |
| Existing-version replacement refusal/approval | Windows |  |  |  |  |  |  |
| Checksum mismatch stops without replacement | Windows |  |  |  |  |  |  |

The public URL must be tested after deployment. Repository inclusion of the
script is necessary but does not prove CDN status, redirects, WAF behavior, or
public bytes.

## 6. Compatibility evidence matrix

Create one row for every claimed agent × stable agent version × source
environment × destination environment × direction. Do not collapse both
directions into one result.

| Cell ID | Agent/vendor | Agent version | Source OS/build/arch | Destination OS/build/arch | Direction | Reinstate tag/commit | Adapter state on both devices | Native resume command | Result | Immutable evidence path/digest | Executor/date | Reviewer |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  |  |  |  |  |

Rules:

- `SUPPORTED` is required on both devices for a passing mutating acceptance
  cell.
- `UNTESTED`, `UNSUPPORTED`, `NOT INSTALLED`, `BLOCKED`, or missing physical
  evidence is not a passing cell.
- Claude Code resumes only in Claude Code; Codex resumes only in Codex.
- Discovery of an ID is necessary but not sufficient. Evidence must include
  the exact planned destination and the vendor's native resume result.
- Native Windows and WSL2 are separate environments and device identities.
- macOS arm64 evidence does not prove macOS amd64; local evidence does not prove
  a physical two-device direction.
- Synthetic fixtures prove deterministic contracts, not vendor-native
  end-to-end compatibility by themselves.

### Version-range decision

| Agent | Proposed inclusive stable range | Cells tested across entire proposed range | Out-of-range/prerelease behavior verified | Decision | Evidence |
| --- | --- | --- | --- | --- | --- |
| Claude Code |  |  |  |  |  |
| Codex CLI |  |  |  |  |  |

Do not expand a public range from one sampled version without the documented
compatibility policy and evidence.

## 7. Phase 1 two-device acceptance ledger

Run the tagged release's acceptance runbook with a disposable project, isolated
Reinstate homes, fresh profile/prefix, selected synthetic-safe sessions, and no
transcript evidence.

| Mandatory gate | Result | Exact subject and exit code | Immutable evidence path/digest | Owner/reviewer | Blocker/correction |
| --- | --- | --- | --- | --- | --- |
| Public POSIX installer installs the exact release on Mac |  |  |  |  |  |
| Public PowerShell installer installs the exact release on Windows |  |  |  |  |  |
| Both installers are idempotent and PATH-safe |  |  |  |  |  |
| Pre-init missing-config failure is accurate |  |  |  |  |  |
| Post-init setup check and synthetic self-test pass on both devices |  |  |  |  |  |
| Claude setup prompt completes on the designated device |  |  |  |  |  |
| Codex setup prompt completes on the designated device |  |  |  |  |  |
| Only explicitly selected disposable sessions reach the manifest |  |  |  |  |  |
| Remote manifest and snapshots are ciphertext-only |  |  |  |  |  |
| Wrong passphrase fails without mutation |  |  |  |  |  |
| Additional-device init refuses a missing established manifest |  |  |  |  |  |
| Claude source-to-destination native resume succeeds |  |  |  |  |  |
| Codex source-to-destination native resume succeeds |  |  |  |  |  |
| Active-agent overwrite is refused |  |  |  |  |  |
| Existing destination is backed up before restore |  |  |  |  |  |
| Claude reverse-direction native resume succeeds |  |  |  |  |  |
| Codex reverse-direction native resume succeeds |  |  |  |  |  |
| Existing reverse-direction targets are backed up |  |  |  |  |  |
| Unchanged pushes skip without new snapshots |  |  |  |  |  |
| Divergence records a conflict without silent overwrite |  |  |  |  |  |
| `--keep-both` preserves both branches with distinct identity |  |  |  |  |  |
| Hard-excluded credential artifacts are refused |  |  |  |  |  |
| Every required repository and release workflow check is green |  |  |  |  |  |

Stop on checksum mismatch, plaintext remote session data, silent overwrite,
missing backup, credential inclusion, or unexplained non-zero exit. A product
binary change after a failed gate requires a new RC and rerun of the failed and
downstream gates.

## 8. Security and privacy evidence

| Claim/gate | Test subject and method | Result | Evidence | Limitation/reviewer |
| --- | --- | --- | --- | --- |
| Remote manifest is ciphertext |  |  |  |  |
| Remote snapshots are ciphertext |  |  |  |  |
| Passphrase is absent from config/state/logs/report |  |  |  |  |
| Storage credentials are absent from config/state |  |  |  |  |
| Known credential artifacts are hard-excluded |  |  |  |  |
| Wrong passphrase causes authenticated refusal without mutation |  |  |  |  |
| Restore validates before mutation and creates required backup |  |  |  |  |
| Atomic restore/conflict behavior passes |  |  |  |  |
| Fixture/evidence secret scan passes |  |  |  |  |
| Formal security audit status is stated accurately |  |  |  |  |

Never inspect or publish a real transcript to prove encryption. Use ciphertext
checks, counts, hashes, exit codes, synthetic fixtures, and redacted paths.
“Open source” and an SBOM do not mean “formally audited.”

## 9. Product-truth and documentation synchronization

| Surface/fact | Expected released value | Authoritative evidence | Updated commit/URL | Reviewer | Result |
| --- | --- | --- | --- | --- | --- |
| Current version/date/status |  | Tag/changelog/release |  |  |  |
| Supported agents and tested ranges |  | Compatibility matrix/evidence |  |  |  |
| Supported/target operating systems and open gates |  | Compatibility evidence |  |  |  |
| Same-vendor native resume boundary |  | Adapter/acceptance evidence |  |  |  |
| Storage backends and permissions |  | Released code/tests |  |  |  |
| Encryption and credential exclusions |  | Released code/security evidence |  |  |  |
| License, maintainer, account requirement |  | Repository/project facts |  |  |  |
| README and product docs |  | Tagged source |  |  |  |
| Website compatibility/security/limitations |  | Production deployment |  |  |  |
| Changelog and GitHub release |  | Tag/release |  |  |  |
| JSON-LD, Open Graph cards, RSS, sitemap, `llms.txt` |  | Built and production site |  |  |  |
| Installers and setup prompts |  | Exact tagged bytes |  |  |  |
| External GitHub/social/directory profiles |  | Authorized manual review |  |  |  |

Planned features remain labeled planned. Never add ratings, reviews,
benchmarks, awards, customer counts, or platform support without evidence.

## 10. Discoverability and external release actions

| Action | Triggered canonical(s) | Repository preparation evidence | Authorized external operator | Result/state | Response/evidence | Due/follow-up acceptance |
| --- | --- | --- | --- | --- | --- | --- |
| Technical release note |  |  |  |  |  |  |
| Changelog/RSS update |  |  |  |  |  |  |
| GitHub release summary |  |  |  |  |  |  |
| Compatibility page/data update |  |  |  |  |  |  |
| Stale docs corrected |  |  |  |  |  |  |
| Structured-data/social-card update |  |  |  |  |  |  |
| Sitemap diff reviewed |  |  |  |  |  |  |
| IndexNow changed-URL plan |  |  |  |  |  |  |
| IndexNow production proof/submission |  |  |  |  |  |  |
| Search Console launch-critical inspection |  |  |  |  |  |  |
| Bing launch-critical inspection |  |  |  |  |  |  |
| Production crawler/WAF smoke check |  |  |  |  |  |  |
| External profile/community correction |  |  |  |  |  |  |

IndexNow acceptance is not evidence of crawling, indexing, ranking, or AI
citation. Do not ping unchanged URLs. Keep the IndexNow key private; its public
ownership proof is public only by protocol and must not expose the secret in
logs or client bundles.

## 11. Release-window observations and guardrails

Use a predeclared window and comparable baseline. These observations never
change the immutable compatibility result.

| Measurement | Current | Baseline | Numerator | Denominator | Source/access owner | State | Interpretation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Indexing change for release URLs |  |  | Indexed release canonicals | Eligible release canonicals | Search Console/Bing |  |  |
| Crawler success rate |  |  | Successful verified bot responses | Verified in-scope bot requests | Production logs |  |  |
| Bot `403`/`429`/`5xx` rate |  |  | Responses by status class | Verified in-scope bot requests | Production logs |  |  |
| Organic release-page qualified actions |  |  | Approved actions from organic release-page sessions | Eligible organic release-page sessions | Analytics |  |  |
| AI referral sessions/actions |  |  | Sessions/actions matching versioned AI channel rules | N/A — count; use sessions for an action rate | Analytics |  |  |
| Fixed-query mention/citation frequency |  |  | Mentioned/cited observations | Completed query-provider observations | Manual fixed-query run |  |  |
| Install success |  |  | Verified successful installs | Install attempts with observed outcome | Acceptance/approved telemetry |  |  |
| Restore success |  |  | Verified native resumes | Restore attempts with observed outcome | Acceptance/approved telemetry |  |  |
| Support/documentation failures |  |  | Verified in-scope reports | N/A — count | Support/issue evidence |  |  |
| Compatibility regression rate |  |  | Previously passing cells now failing | Previously passing cells retested | Compatibility evidence |  |  |

Mark unavailable sources explicitly. A download, command copy, or installation
page view is not install success. A pull completion is not native resume
success. Do not attribute a search or citation change to the release without
comparable evidence and stated uncertainty.

## 12. Evidence digest registry

Register every file needed to reproduce the decision. Compute digests after
redaction and finalization.

| Evidence ID | Repository path or durable URL | Subject | Collected by/date | Redactions applied | SHA-256 or immutable workflow/artifact ID | Reviewer | Status |
| --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |

If evidence must remain private, record the custodian, access class, immutable
identifier/digest, and sanitized summary. Do not commit the private payload.

## 13. Findings and release-blocker register

| ID | Severity | Failed/open gate | Exact corrective task | Owner | Due | Acceptance criteria | Required retest/downstream gates | Evidence expected | Status |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  | Open |

Every blocker needs a named owner, due date, testable acceptance criterion, and
retest scope. “Known issue,” “follow up,” and “works locally” do not close a
release gate.

## 14. Final release decision

### Mandatory sign-off

- [ ] Exact tag identity and signature pass.
- [ ] Required CI and release workflow gates pass on the tag.
- [ ] Expected artifacts, checksums, SBOMs, and attestations pass.
- [ ] Public installers pass on their claimed physical targets.
- [ ] Every mandatory two-device acceptance row is `PASS`, or the corresponding
      release/platform claim remains explicitly open and unclaimed.
- [ ] No evidence contains secrets or real transcript content.
- [ ] Product, compatibility, security, docs, changelog, schema, feed, and
      release claims agree.
- [ ] External actions are performed only by authorized owners and their state
      is disclosed.
- [ ] Every evidence item has an immutable path/URL and digest/identifier.
- [ ] Every non-pass mandatory gate is represented as a blocker.

| Decision field | Value |
| --- | --- |
| Decision | Release / Release candidate with explicit open gates / No-go |
| Decision time, UTC |  |
| Mandatory gates passed |  |
| Mandatory gates not passed |  |
| Public compatibility cells approved |  |
| Explicitly unclaimed cells/features |  |
| Open security/privacy findings |  |
| Open Critical/High findings |  |
| Release approver |  |
| Compatibility approver |  |
| Security reviewer |  |
| Documentation/discoverability reviewer |  |
| Next evidence record or correction due |  |

## Repository references

- [Phase 1 physical acceptance runbook](../testing/phase-1-mac-windows-acceptance.md)
- [Weekly report template](weekly-report-template.md)
- [Monthly audit template](monthly-audit-template.md)
- [Quarterly review template](quarterly-review-template.md)
- [Operations and external-access runbook](operations.md)
- [SEO, AEO, and ASEO playbook](seo-aeo-aseo-playbook.md)
