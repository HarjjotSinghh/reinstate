# Phase 0 + Phase 1 Implementation Supervision

**Status:** Final review of the current implementation head; re-open if a newer commit is pushed  
**Reviewed branch:** `feat/phase-0-phase-1`  
**Baseline:** `main` at `bc1b26ba28cca8c92ab2c22cf519a9b8d7a4c395`  
**First reviewed branch head:** `4c50b3f9219bcefed47d254f6a8f31f163dcd4a3`  
**Final reviewed branch head:** `01a1143721ff80716e92f9196bb83a96ff6c6679`  
**Requirements source:** `docs/superpowers/plans/2026-07-25-reinstate-phase-0-phase-1.md`

This is a read-only supervision log. It intentionally does not modify the implementation worktree. Findings describe the observed snapshot and must be re-checked against the final committed tree before merge.

## Merge verdict

**Current verdict: DO NOT MERGE.**

The branch has substantial implementation work, but the current snapshot does not yet deliver the Phase 1 product contract: a real Claude Code/Codex session push from one device, safe pull and restore on another device, and successful discovery by the target agent.

## What passed

The implementation has a healthy conventional engineering baseline:

- GitHub Actions run
  `https://github.com/HarjjotSinghh/reinstate/actions/runs/30129359274`
  passed lint, security, Ubuntu, macOS, and Windows jobs on
  `01a1143721ff80716e92f9196bb83a96ff6c6679`.
- An independent archive of `230bb0b73c2b3dc4b5cd4c155d4e2fb23bb60edf`
  passed `go test ./... -count=1`, `go test -race ./... -count=1`,
  `go vet ./...`, and a clean CLI build. Later commits changed only CI/lint
  configuration and the Windows test environment.
- GoReleaser completed a local snapshot and generated five platform archives
  plus `checksums.txt`.
- `golangci-lint` v2.1.6 reports zero source issues locally at the final head.

These results prove that the code builds and its current tests pass. They do not
prove the Phase 1 acceptance journeys.

## Decisive black-box failure

An independent synthetic Claude test on the reviewed implementation:

1. created a session under a temporary `~/.claude/projects/...` root;
2. initialized a disk-backed test backend;
3. pushed the session successfully;
4. removed the source session;
5. pulled the session successfully according to CLI output; and
6. inspected both the expected Claude path and Reinstate cache.

Observed result:

```text
{
  "dry_run": false,
  "pulled": 1
}
AGENT_RESTORE=missing
CACHE_FILES:
.../reinstate-home/cache/pull/snap-.../session-target.jsonl
```

The CLI reports success while the agent session remains unrestored. This alone
is sufficient for `DO NOT MERGE`.

## P0 — release blockers

### 1. The user-facing push/pull path bypasses the agent adapters

**Observed in:** `internal/cli/commands_impl.go`

- `push` passes the discovered raw session path directly into the sync engine.
- The command does not call the selected adapter's export/transform pipeline.
- `pull` decrypts into `~/.reinstate/cache/pull/<snapshot>` and does not call adapter restore planning or restore.
- The restored content therefore does not land in Claude Code or Codex's actual session store.

**Impact:** The CLI can move an encrypted file, but it cannot currently complete the product's core promise: resume the session in Claude Code or Codex on the second device.

**Required correction:**

1. Push must use adapter discovery and export.
2. Export must produce a validated, portable artifact plus metadata.
3. Pull must resolve the correct adapter, build a restore plan, show the plan in dry-run mode, and perform the adapter restore.
4. Post-restore verification must prove that the target agent can discover the restored session.

### 2. Conflict resolution does not perform conflict resolution

**Observed in:** `internal/sync/conflict.go` and the conflict CLI handler

- `Resolve` writes a `.resolved` marker/timestamp and removes the conflict record.
- It does not execute `keep-local`, `keep-remote`, or `keep-both`.
- The CLI caller does not perform the missing restore/copy operation.

**Impact:** The command can report a conflict as resolved without changing the conflicting data. That is false-success behavior and risks data loss or user confusion.

**Required correction:** Each strategy must execute a concrete, testable data operation before the conflict record is cleared. Failure must preserve the conflict record and return a non-zero stable exit code.

### 3. Manifest compare-and-swap retry can overwrite concurrent updates

**Observed in:** `internal/sync/push.go`

- On a precondition failure, the code reloads the current ETag but then retries the stale manifest with that new ETag.
- The latest manifest contents are not merged/reconciled before the retry.

**Impact:** Concurrent device pushes can clobber manifest entries. This defeats the purpose of optimistic concurrency.

**Required correction:** On CAS failure, reload the latest manifest, reconcile the intended mutation against it, re-run conflict rules, and retry with the latest ETag. Add a deterministic concurrent-writer test that would lose an entry under the current behavior.

### 4. Secrets are accepted through command-line flags and stored as plaintext

**Observed in:** `internal/cli/commands_impl.go` and `internal/credentials/file.go`

- `init` accepts `--access-key` and `--secret-key`.
- Credentials are written as plaintext JSON under `~/.reinstate/credentials`.
- The approved contract requires OS keyring storage or an explicit documented provider fallback and says not to silently store plaintext credentials.

**Impact:** Secrets can leak through shell history, process inspection, backups, or accidental file disclosure.

**Required correction:**

- Remove secret-bearing CLI flags.
- Prefer the AWS shared credential/provider chain or OS keyring.
- If a plaintext fallback remains, require explicit opt-in with a loud warning and document the exact threat model.
- Add a test ensuring diagnostics, errors, JSON output, and logs never print secret values.

### 5. Passphrase handling does not meet the approved security contract

**Observed in:** `internal/cli/commands_impl.go`

- Normal commands require `REINSTATE_PASSPHRASE`.
- A hidden TTY prompt or file-descriptor-based input path is not implemented in the reviewed snapshot.

**Impact:** Environment variables are commonly inherited by child processes and may be captured by diagnostics or automation. This is weaker than the planned interaction model.

**Required correction:** Implement hidden TTY input for interactive use and a documented non-interactive secret-input mechanism that does not expose the passphrase as a command argument. If environment input is retained, document it as an explicit CI fallback with its risks.

## P1 — correctness and safety gaps

### 6. Codex discovery and restore do not match real rollout layout

**Observed in:** `internal/adapter/codex/codex.go`

- Discovery and its synthetic test fixture assume `cwd` and `id` are top-level
  fields in the first JSONL record.
- That assumed record shape and flat layout are not backed by versioned
  compatibility evidence in the branch.
- Restore writes `<root>/sessions/<id>.jsonl`, ignoring the plan's required
  date-partitioned rollout coverage and any state/index reconstruction required
  for `codex resume`.
- The current official Codex manual documents `CODEX_HOME` as containing
  sessions plus SQLite-backed runtime state, but does not publish the private
  rollout-file layout as a stable integration contract. A raw file copy cannot
  therefore be treated as compatible without a version-specific runtime test.

**Impact:** Real Codex sessions may not be discovered, and restored files may not appear in Codex's resume UI.

**Required correction:** Build fixtures from redacted real-world Codex rollouts, parse the versioned record shape, preserve the expected directory layout, update any required index/state atomically, and prove discovery via an end-to-end test.

### 7. Compatibility detection claims support without proving it

**Observed in:** `internal/adapter/claude/claude.go` and `internal/adapter/codex/codex.go`

- Detection reports `SUPPORTED` when the expected root exists, even when the agent version is `unknown`.
- No fail-closed layout/version compatibility gate is evident.

**Impact:** Reinstate can mutate an unknown or changed vendor format while telling the user it is supported.

**Required correction:** Return explicit supported, unsupported, or unknown states based on version/layout probes. Unknown must block writes by default while still allowing safe diagnostics.

### 8. Path rewriting is not sufficiently schema-aware

**Observed in:** both Claude and Codex adapter rewrite functions

- Any absolute-path-looking string can be rewritten, including strings outside known path-bearing fields.
- This can mutate transcript prose, model output, tool output, commands, or serialized content that merely resembles a path.

**Impact:** Session contents can be silently corrupted.

**Required correction:** Rewrite only explicitly allow-listed fields for each supported record schema. Preserve unknown records and unknown fields byte-for-byte where possible. Add adversarial fixtures containing path-like prose that must not change.

### 9. Sync operations buffer entire sessions in memory

**Observed in:** `internal/sync/push.go`, `internal/adapter/claude/claude.go`, and `internal/adapter/codex/codex.go`

- Session files are loaded with `os.ReadFile`.
- Export/transformation and encryption use in-memory buffers.
- Pull decrypts the whole object into a buffer.
- Codex discovery reads an entire rollout to inspect initial metadata.

**Impact:** Large JSONL rollouts can cause excessive memory use or process failure. This violates the bounded-memory/streaming requirement.

**Required correction:** Stream discovery, archive construction, encryption, upload, download, decryption, hashing, and restore. Add a large synthetic fixture and assert bounded memory behavior or at least prove no whole-file reads in the critical path.

### 10. Pull does not validate the portable artifact before restore

**Observed in:** `internal/sync/push.go`

- The reviewed pull path does not clearly enforce envelope schema version, kind, identity fields, declared size, or SHA-256 before writing output.

**Impact:** Corrupt, truncated, mismatched, or malicious remote objects may be accepted.

**Required correction:** Validate metadata and content hash before restore. Reject path traversal, unknown schema versions, unexpected artifact kinds, oversized records, and identity mismatches.

### 11. Dry-run is not a real preview

**Observed in:** sync pull and CLI command flow

- Pull dry-run returns early without fetching, decrypting, validating, or constructing a restore plan.
- The command can count a pull that performed no meaningful analysis.

**Impact:** Users cannot trust dry-run to tell them what will change or whether the operation will succeed.

**Required correction:** Dry-run must perform every safe read/validation/planning step and omit only mutations. Output should list source revision, destination paths, backups, conflicts, and expected adapter changes.

### 12. Restore paths lack consistent atomicity, backups, and locking

**Observed in:** Claude and Codex adapter restore implementations

- Direct `os.WriteFile` calls are used in the reviewed snapshot.
- Backup creation, atomic replacement, and per-session locking are not consistently integrated.

**Impact:** A crash, concurrent run, disk-full event, or interrupted write can corrupt local agent state.

**Required correction:** Use the shared atomic-write and backup primitives, acquire deterministic locks, fsync as appropriate, and verify rollback behavior with injected failures.

### 13. Profile isolation is incomplete

**Observed in:** `internal/sync/push.go` and the memory/disk backend path

- `Engine.Prefix` is unused in the reviewed implementation.
- S3 may apply a prefix internally, but the generic/local backend keyspace does not demonstrate equivalent profile isolation.

**Impact:** Multiple configured profiles can collide or see each other's manifests and snapshots.

**Required correction:** Make namespace/prefix behavior part of the backend contract and test two profiles against the same backend root.

### 14. Secret-file exclusion is basename-only and incomplete

**Observed in:** `internal/sync/push.go`

- The critical path checks a small basename denylist such as `auth.json` and `.credentials.json`.
- It does not consistently enforce adapter-level allowlists and the full credential/config exclusion contract.

**Impact:** A newly discovered file or unexpected layout can upload credentials.

**Required correction:** Export only explicitly allow-listed session artifacts, add defense-in-depth deny rules, scan the final portable artifact before encryption, and test known plus adversarial credential filenames and embedded secret patterns.

## P1 — verification and documentation gaps

### 15. Doctor reports capabilities it does not actually probe

**Observed in:** `internal/cli/doctor.go`

- Keyring availability is reported without a demonstrated keyring implementation/probe.
- Agent checks primarily observe directory presence, not version/layout compatibility.
- Self-test covers a small encryption/atomic-write exercise but not the complete storage, manifest, adapter, restore, and discovery path.

**Impact:** Doctor can show green while the product's critical path is broken.

**Required correction:** Every green result must be backed by a real probe. The full self-test should exercise a sandboxed push/pull round trip through the configured backend and adapters without touching real agent data.

### 16. Documentation and implementation disagree on cryptography and credentials

**Observed in:** `docs/architecture.md`, `docs/security-model.md`, README/setup documentation, and implementation

- Documentation mentions Argon2 in places while implementation uses age's scrypt recipient.
- Security docs mention credential-sync overrides even though Phase 1's approved contract forbids syncing credentials.
- Setup claims must match the actually implemented prompt/input and credential-storage behavior.

**Impact:** Open-source users cannot make informed security decisions, and contributors may implement against the wrong contract.

**Required correction:** Reconcile all docs with the final implementation and include a single threat-model/source-of-truth page.

### 17. Go compatibility changed without an explicit product decision

**Observed in:** `go.mod` and CI/release workflows

- The final tree declares Go language version `1.24.0` and toolchain
  `go1.25.12`.
- CI and release builds install Go `1.25.12`.
- The implementation plan targeted Go 1.22+ compatibility.

**Impact:** Installation can fail on supported developer/user machines or contradict CI/release documentation.

**Required correction:** Choose the supported minimum intentionally, pin it in `go.mod`, CI, release tooling, and contributor docs, then test that exact minimum version.

### 18. Installer asset names do not match the GoReleaser archive contract

**Observed in:** `scripts/install.sh`, `scripts/install.ps1`, and
`.goreleaser.yml`

- The installers construct archive names such as
  `reinstate_<tag>_<lowercase-os>_<arch>.tar.gz` or `.zip`.
- GoReleaser currently relies on its default archive name template, whose
  version and OS components do not match the installer-constructed form.

**Impact:** A valid published release can still be impossible to install with
the official scripts because the requested asset URL does not exist.

**Required correction:** Define one explicit archive `name_template`, make both
installers consume exactly that contract, and add a release-fixture test that
runs each installer against the names emitted by a snapshot build.

### 19. The copy-paste prompts are not yet end-to-end setup workflows

**Observed in:** `docs/prompts/claude-code-setup.md`,
`docs/prompts/codex-setup.md`, and the `init` command

- Both prompts stop at telling the human to run `rein init`.
- The current `init` command describes itself as interactive but does not
  implement an interactive wizard; it requires endpoint/bucket flags or
  environment values.
- The prompts do not walk the human through non-chat secret entry, path-map
  setup, a real backend probe, or a verified session round trip.

**Impact:** Copy-pasting the flagship quality-of-life prompt cannot currently
produce a configured, verified two-device installation.

**Required correction:** Make the prompt and CLI form one executable contract:
the agent performs safe detection/download/checksum/install steps, pauses for
the human to enter secrets privately through the interactive CLI, then resumes
doctor, backend, adapter, path-map, and sandboxed round-trip verification.

### 20. Release provenance artifacts required by Phase 0 are absent

**Observed in:** `.goreleaser.yml` and `.github/workflows/release.yml`

- SBOM generation was removed/commented out to make local snapshot builds pass.
- The release workflow does not install or invoke an SBOM generator.
- No checksum signature, Sigstore bundle, build provenance attestation, or
  equivalent verification step is present in the reviewed snapshot.

**Impact:** Checksums provide corruption detection but do not independently
authenticate a compromised release channel. The published artifact set would
not satisfy the approved Phase 0 release contract.

**Required correction:** Keep local snapshots ergonomic, but make the tag
release gate generate and publish the required SBOM and provenance/attestation
artifacts. Test that the end-user prompts verify the actual published contract.

## P2 — engineering process and supply-chain findings

### 21. Tool installation bypassed the agreed confirmation gate

**Observed process behavior:**

- The implementation session was launched with `--permission-mode bypassPermissions`.
- It installed external tooling during the run.
- `govulncheck` was installed with `@latest`; other tools were fetched by versioned curl/install commands.

**Impact:** This weakens reproducibility and supply-chain review. `@latest` can change between runs.

**Required correction:** Pin all build/security tooling, verify checksums/signatures where available, record versions in a tool manifest or CI image, and require confirmation before machine-level installation.

### 22. Commit history is too coarse for the approved implementation plan

**Observed:** The initial implementation arrived as a small number of very large commits covering major subsystems.

**Impact:** Review, bisecting, revert safety, and contributor onboarding are harder. This matters for a security-sensitive open-source CLI.

**Required correction:** The final history does not need theatrical commit spam, but each independently testable subsystem should have a focused commit and the feature-completing commit should receive the SemVer tag only after all release gates pass.

### 23. Branch publication occurred before supervision was complete

**Observed:** `origin/feat/phase-0-phase-1` matched the first reviewed head while implementation was still ongoing.

**Impact:** Not a product bug, but it may conflict with the original no-push execution handoff unless the implementer received later authorization.

**Required correction:** Confirm whether branch pushes were authorized. Do not tag, merge, publish a release, or update `main` until independent verification passes.

### 24. Diff hygiene has minor unresolved whitespace errors

**Observed:** `git diff --check origin/main...HEAD`

- The three setup-prompt Markdown files contain trailing whitespace.

**Impact:** Non-blocking, but it makes a supposedly hard formatting gate
incomplete and creates needless review noise.

**Required correction:** Remove the whitespace or configure the documentation
formatter intentionally.

## Exact evidence map at final head

| Finding | Final-head evidence |
|---|---|
| CLI bypasses adapter export | `internal/cli/commands_impl.go:290-320` sends discovered raw paths directly to `sync.PushSession` |
| CLI does not restore into an agent | `internal/cli/commands_impl.go:348-370` writes to `home/cache/pull/<snapshot>` and never calls an adapter restore |
| Pull dry-run performs no validation | `internal/sync/push.go:164-170` returns before backend fetch/decrypt/plan |
| Manifest CAS can clobber updates | `internal/sync/push.go:151-160` retries stale ciphertext with the latest ETag |
| Conflict resolution is bookkeeping only | `internal/sync/conflict.go:80-93` writes an audit marker and deletes the record |
| Secrets accepted as CLI args | `internal/cli/commands_impl.go:119-125` exposes access/secret key flags |
| Credentials stored as plaintext JSON | `internal/cli/commands_impl.go:85-89` and `internal/credentials/file.go:37-52` |
| No hidden passphrase prompt | `internal/cli/commands_impl.go:195-200` requires an environment variable |
| Compatibility is optimistic | `internal/adapter/claude/claude.go:62-70` and `internal/adapter/codex/codex.go:57-61` mark unknown versions supported |
| Whole-file buffering | `internal/sync/push.go:32-75`, `internal/adapter/claude/claude.go:128-150`, and `internal/adapter/codex/codex.go:84-95,124-140` |
| Unsafe broad path rewriting | `internal/adapter/claude/claude.go:233-255` and `internal/adapter/codex/codex.go:212-233` |
| Direct non-atomic adapter restore | `internal/adapter/claude/claude.go:172-210` and `internal/adapter/codex/codex.go:161-191` |
| Doctor overstates keyring support | `internal/doctor/report.go:104-117` |
| Self-test does not test sync | `internal/doctor/selftest.go:13-46` |
| Installer/release asset mismatch | `scripts/install.sh:47-50`, `scripts/install.ps1:18-19`, and `.goreleaser.yml:31-41` |
| SBOM/attestation missing | `.goreleaser.yml:43-48` and `.github/workflows/release.yml:38-45` |

## Required independent acceptance evidence

The branch should not merge until the following evidence exists:

- [ ] Clean worktree and a reviewable final commit range against `main`.
- [ ] Unit, race, lint, vulnerability, and release-snapshot gates pass with pinned tools.
- [ ] macOS and Windows CI pass on the documented minimum Go version.
- [ ] A real Claude Code fixture is pushed, pulled into a clean sandbox, restored, and discovered by the target agent.
- [ ] A real Codex fixture is pushed, pulled into a clean sandbox, restored into the correct rollout layout, and discovered by `codex resume`.
- [ ] Cross-device path mapping works in both directions for representative macOS and Windows paths.
- [ ] A concurrent two-writer test proves manifest updates do not clobber each other.
- [ ] `keep-local`, `keep-remote`, and `keep-both` produce the promised filesystem state.
- [ ] Interrupted restore and disk-full tests leave original data intact and backups usable.
- [ ] Dry-run shows the same plan the mutating run executes.
- [ ] Artifact corruption, wrong passphrase, wrong schema, path traversal, oversized inputs, and backend precondition failures fail safely.
- [ ] A secret scan of source, fixtures, logs, diagnostics, release archives, and portable artifacts passes.
- [ ] The copy-paste setup prompts for Claude Code and Codex succeed from clean supported machines without exposing credentials.
- [ ] Manual setup documentation reproduces the same successful result.
- [ ] The release commit is tagged according to the chosen SemVer starting point only after all above gates pass.

## Re-review notes

At implementation completion:

1. Rebase this document's findings onto the final commit, removing anything Grok fixed.
2. Add exact file/line references for every surviving finding.
3. Run the independent test matrix without modifying the implementation branch.
4. Issue one of: `READY TO MERGE`, `READY WITH DOCUMENTED FOLLOW-UPS`, or `DO NOT MERGE`.
