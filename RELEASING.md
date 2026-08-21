# Releasing

How maintainers cut a **Reinstate** release.

## Versioning

- **Semantic Versioning**: `MAJOR.MINOR.PATCH`
- Pre-1.0: breaking changes allowed in `MINOR` with clear CHANGELOG entries
- Git tags are prefixed: `v0.1.0`

## Preconditions

- [ ] `main` is green on CI
- [ ] CHANGELOG `[Unreleased]` section is accurate
- [ ] No open P0 security issues
- [ ] For a stable release, fresh Apple Silicon macOS and native Windows x64
      acceptance rows pass and the separate stable promotion decision is
      recorded
- [ ] For a release candidate, prior candidate failures are recorded and every
      known release blocker has a regression test or an explicit unresolved
      disposition
- [ ] Claude Code and Codex exact versions/layouts are recorded in compatibility docs
- [ ] Wrong-passphrase, tamper, backup, rollback, conflict, and installer tests pass
- [ ] Snapshot archives, source archive, checksums, and SBOMs were inspected
- [ ] Builds and vulnerability scans use the pinned Go 1.25.13 toolchain

### Supported platform boundary

The maintainer-approved `v0.2.0` reconciliation in
`docs/testing/results/2026-08-02-macos-phase2-V020RC2.md` permits stable
publication with verified support limited to Apple Silicon macOS
(`darwin/arm64`) and native Windows x64 (`windows/amd64`). Those remain the
supported mandatory platforms for Phase 3 and Phase 4 candidate and stable
acceptance.

Intel macOS and Linux/WSL2 artifacts are unsupported/unverified optional
evidence: they may be built, checksummed, SBOM-covered, and attested, but their
absence or failure does not block a candidate or stable promotion. Never
describe them as physically certified or supported. Issues #97 and #98 track
that optional physical evidence.

### v0.3.0-rc.1 candidate evidence

The committed
[`v0.3.0-rc.1` dispatch](docs/testing/v0.3.0-rc.1-agent-verification-prompts.md)
required two tagged-artifact reports: Apple Silicon macOS and native Windows
x64. macOS passed (32/32); native Windows failed (9 PASS / 23 FAIL) with six
root blockers, including Windows executable trust for extensionless `codex`
and host/tooling gaps. RC1 does **not** authorize stable `v0.3.0`.

### v0.3.0-rc.3 candidate evidence

The Phase 3 candidate used
[`v0.3.0-rc.3` dispatch](docs/testing/v0.3.0-rc.3-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). Native
Windows x64 failed on PowerShell 5.1 staging and human-output privacy; do not
promote stable from RC3.

### v0.3.0-rc.4 candidate evidence

The Windows-first corrective candidate used
[`v0.3.0-rc.4` dispatch](docs/testing/v0.3.0-rc.4-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). Its signed
tag workflow failed during Ubuntu PowerShell artifact verification before the
draft was published or attested. No RC4 device report applies; do not promote
stable from RC4.

### v0.3.0-rc.5 candidate evidence

The corrective release-workflow candidate used
[`v0.3.0-rc.5` dispatch](docs/testing/v0.3.0-rc.5-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). Dual-platform
tagged-artifact acceptance failed when primary-host Claude/Codex installs were
outside the fail-closed ranges; do not promote stable from RC5.

### v0.3.0-rc.6 candidate evidence

The compatibility-widen candidate used
[`v0.3.0-rc.6` dispatch](docs/testing/v0.3.0-rc.6-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). RC6 expanded
Claude Code through `2.1.227` and Codex CLI through `0.147.0`. Dual-platform
tagged-artifact acceptance failed (macOS 16 PASS / 12 FAIL / 4 NOT TESTED) on
real-launch baseline, authenticated resume/fork, capability mutation matrix,
and required TTY/picker evidence; do not promote stable from RC6.

### v0.3.0 stable evidence

The post-RC6 harden candidate uses
[`v0.3.0` dispatch](docs/testing/v0.3.0-rc.7-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). RC7 packages
non-TTY fail-closed native launch, Windows Ctrl+C safety at the warning prompt,
capability probe demotion to informational, isolated throwaway agent homes for
capability discovery, and expanded deterministic local Phase 3 smoke. Those two
device reports decide RC7 tagged-artifact acceptance only. Stable promotion still
requires a separate reviewed stable decision and fresh tagged-artifact
validation on the same two supported platforms.

### v0.3.0-rc.2 candidate evidence

[`v0.3.0-rc.2` dispatch](docs/testing/v0.3.0-rc.2-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md) were used for
tagged-artifact acceptance. Native Windows x64 failed again (Codex trust and
snapshot/PowerShell staging gates among the blockers); do not promote stable
from RC2. Corrective product fixes land before `v0.3.0-rc.3`. Stable promotion
still requires a separate reviewed stable decision and fresh tagged-artifact
validation on the same two supported platforms. Intel macOS and WSL2 remain
unsupported/unverified optional evidence and are never stable blockers.

### v0.4.0-rc.1 candidate evidence

The `v0.4.0-rc.1` dual-platform run **failed**. Claude Code was unusable as a
handoff source on every real installation (a source probe required an
`<agent-root>/version` file real installs never create, so Claude-sourced
handoffs exited `5`), reader-emitted absolute paths were rejected by capsule
validation, `changed_files` was never populated, a timed-out version probe was
accepted as an absent agent, and any message beginning with a slash aborted the
handoff. One run also omitted `GROK_HOME` and indexed the operator's real
`~/.grok` tree, so that run was discarded and restarted. RC1 does **not**
authorize stable `v0.4.0`, and the corrective fixes land in `v0.4.0-rc.2`.

### v0.4.0-rc.2 candidate gate

The Phase 4 candidate uses the committed
[`v0.4.0-rc.2` dispatch](docs/testing/v0.4.0-rc.2-agent-verification-prompts.md)
and [Phase 4 acceptance contract](docs/testing/phase-4-cross-agent-handoff-acceptance.md).
Start its two independent device runs only after the signed tag is published,
all release artifacts verify, and both live installer routes pin that exact
candidate. Required Claude ↔ Codex structured handoff, fidelity, workspace,
security, CLI, and performance rows must all pass on Apple Silicon macOS and
native Windows x64. Gemini CLI, OpenCode, and Grok Build remain optional
source-only rows and may be `NOT TESTED` only when genuinely absent; do not
install them solely for acceptance.

The candidates widen the fail-closed Claude Code range through `2.1.229` (was
`2.1.227` at stable `v0.3.0`, then `2.1.228` at `v0.4.0-rc.1`) so both physical
acceptance hosts run an in-range install; the Codex CLI range stays
`0.133.0`–`0.147.0`. Neither patch has physical evidence yet — the dual-platform
run is what supplies it, exactly as `v0.3.0-rc.6` widened first and tested
afterwards.

Expect to repeat this. Claude Code auto-updates, and during the `v0.4.0-rc.1`
window both hosts moved past the ceiling within a day (macOS `2.1.225` ->
`2.1.228`, Windows `2.1.228` -> `2.1.229`). Re-check both hosts' installed
versions immediately before tagging, not only when planning the candidate.

`v0.4.0-rc.2` publication meant ready for tagged-artifact acceptance. The
physical dual-platform run **FAILED**: wrong-repo cwd was not refused, non-TTY
destination launch still spawned, Grok-source busy-check exited with
`unsupported agent "grok"`, and a timed-out version probe classified Runtime.
RC2 does **not** authorize stable `v0.4.0`. Corrective product fixes landed in
`v0.4.0-rc.3`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.3 failures landed in
`v0.4.0-rc.4`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.4 failures landed in
`v0.4.0-rc.5`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.5 failures land in
`v0.4.0-rc.6`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.6 failures land in
`v0.4.0-rc.7`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for the remaining rc.7 product
defect (R1 off-PATH layout scan) and the Go 1.25.13 toolchain pin land in
`v0.4.0-rc.8`. That candidate was published and physical dual-platform
acceptance **FAILED** (macOS 38/44, Windows 38/44). The remaining product
defect is Windows off-PATH `inspect` JSON `agent.status=not_installed` despite
`layout_recognized=true`; dest-ack A1–A7 stayed harness/uncollected. That
inspect mapping lands in `v0.4.0-rc.9`. That candidate was published and physical
dual-platform acceptance **FAILED** (macOS 38/44, Windows 38/44). The remaining
product defect is Windows CreateProcess truncating multi-line dest argv (Codex
dest-ack A5/A7). That dest-argv fix lands in `v0.4.0-rc.10`.

### v0.4.0-rc.9 candidate gate

`v0.4.0-rc.9` was published and physical dual-platform tagged-artifact
acceptance **FAILED** (macOS 38/44, Windows 38/44). Dest-ack A4 PASS;
A1/A2/A3/A5/A6/A7 FAIL. Remaining product defect was Windows dest argv CR/LF
truncation. Historical dispatch:
[`v0.4.0-rc.9`](docs/testing/v0.4.0-rc.9-agent-verification-prompts.md).
Publication was not evidence that the matrix passed.

### v0.4.0-rc.10 candidate gate

`v0.4.0-rc.10` was published and physical dual-platform tagged-artifact
acceptance **FAILED** (macOS 41/44 A1/A3/A7, Windows 40/44 A1/A2/A3/A5). Dest-ack
A4/A6 PASS both OS; Windows dest argv one-line `projection.md` held. Remaining
product defects were dest first-reply missing the five bullets, lineage after
Launch, and dest TUI folder trust. Historical dispatch:
[`v0.4.0-rc.10`](docs/testing/v0.4.0-rc.10-agent-verification-prompts.md).
Publication was not evidence that the matrix passed.

### v0.4.0-rc.11 candidate gate

The Phase 4 candidate uses the committed
[`v0.4.0-rc.11` dispatch](docs/testing/v0.4.0-rc.11-agent-verification-prompts.md)
and [Phase 4 acceptance contract](docs/testing/phase-4-cross-agent-handoff-acceptance.md).
Start its two independent device runs only after the signed tag is published,
all release artifacts verify, and both live installer routes pin that exact
candidate. Required Claude ↔ Codex structured handoff, fidelity, workspace,
security, CLI, and performance rows must all pass on Apple Silicon macOS and
native Windows x64, including RC1 R1–R6, the RC2, RC3, RC5, RC6, RC7, RC8, RC9,
RC10, and RC11 regression sets.
Gemini CLI,
OpenCode, and Grok Build remain optional source-only rows and may be
`NOT TESTED` only when genuinely absent; do not install them solely for
acceptance.

The fail-closed Claude Code range stays `2.1.219`–`2.1.229`; the Codex CLI range
stays `0.133.0`–`0.147.0`. Re-check both hosts' installed versions immediately
before tagging. Claude Code auto-updates. A PATH Claude of `2.1.230` must
fail-closed (exit 5) without `--allow-untested`.

`v0.4.0-rc.11` publication meant ready for tagged-artifact acceptance. That
acceptance later passed 44/44 on both mandatory platforms and authorized
stable `v0.4.0`.

### v0.4.0 stable evidence

Dual-platform tagged-artifact acceptance PASS on candidate `v0.4.0-rc.11`:

- Apple Silicon macOS 44/44 (`docs/testing/results/2026-08-15-macos-phase4-V040RC11.md`)
- native Windows x64 44/44 (`docs/testing/results/2026-08-15-windows-phase4-V040RC11.md`)

This authorizes the signed stable `v0.4.0` tag. Intel macOS and Linux/WSL2
remain optional and unverified. Native resume remains same-vendor. Gemini CLI,
OpenCode, and Grok Build remain handoff sources only.

### v0.5.0-rc.1 candidate gate

The first Phase 5 candidate. Catalog refactor landed. Probe shipped. Six new
T1 agents (Kimi, Qwen, Pi, Cursor CLI, Copilot, Cline) and three T2 handoff
sources (Gemini, OpenCode, Grok). Claude Code and Codex stay the only T4/T5
surfaces. Gemini is **not** promoted to T3. Generated matrix:
`rein doctor --agents --acceptance-matrix` reports **150** required rows.

Dispatch:
[`v0.5.0-rc.1`](docs/testing/v0.5.0-rc.1-agent-verification-prompts.md).
Contract:
[Phase 5 acceptance](docs/testing/phase-5-universal-agent-coverage-acceptance.md).

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.5.0`. Current stable remains `v0.4.0`.

### v0.5.0-rc.5 candidate gate

The fifth Phase 5 candidate. Physical `v0.5.0-rc.4` acceptance found one defect
that made every other reader fix unreachable: an index never re-read a source
after Reinstate itself changed. Both layers of change detection asked only
whether a file had moved, so an upgrade left an existing index frozen and a
reader fix reached nobody until the user's agent happened to write a new
session. On the Windows host an index built before the Gemini workspace fix
served 24 sessions with no workspace indefinitely; it heals on the first run of
this candidate. Note the row-level half of that predates `v0.5.0`: it shipped
in `v0.4.0`.

This candidate also carries Gemini project-path recovery on case-insensitive
filesystems, an agent root override that is honoured when it names a missing
path rather than silently walking the home tree, Cursor CLI's verified root
variable, incremental refresh, and Windows portability of the documentation
tests on a non-system drive.

Both verified vendor ranges move on dual-platform physical resume evidence —
Claude Code through `2.1.238` and Codex CLI through `0.149.0`. On each platform
a session was created with the new version, indexed, resumed through the launch
plan Reinstate produced, and the resumed session returned a token that existed
only in the original session's history.

Pre-tag dual-platform matrix state on the candidate commit, run with the
harnesses rather than the tagged artifact: 150/150 on macOS and 150/150 on
Windows. That is not tagged-artifact acceptance and does not substitute for it.

Dispatch:
[`v0.5.0-rc.5`](docs/testing/v0.5.0-rc.5-agent-verification-prompts.md).
Contract:
[Phase 5 acceptance](docs/testing/phase-5-universal-agent-coverage-acceptance.md).

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.5.0`. Current stable remains `v0.4.0`.

### v0.5.0-rc.6 candidate gate

The sixth Phase 5 candidate. Physical `v0.5.0-rc.5` tagged-artifact acceptance
ran the full matrix on both platforms and **failed**: macOS 149/150,
Windows 150/150. One required row failed, **B4** on macOS, so `v0.5.0-rc.5`
did not authorize stable `v0.5.0`.

B4: the agent probe carried a raw 38-character Git object hash into its
artifact. OpenCode keeps a Git object store under each snapshot, and Git stores
an object as a two-character directory plus a thirty-eight character file. The
shape normalizer recognised only exactly 32, 40 and 64 characters, so a
38-character stem matched nothing and reached the artifact verbatim. Those names
are content hashes of the operator's own repository. This candidate collapses
any hex run long enough to identify content; the established tokens are
unchanged so committed artifacts do not churn.

Everything else passed on both platforms, including the upgrade path the
previous candidate existed to fix: an index built by the previous release and
then opened by the tagged binary re-read what the old reader could not resolve
(0 to 3 resolved workspaces on macOS, 0 to 10 on Windows) without losing a row,
and the previous release could still reopen it. All four E1/E2 journeys ran
against the installed tagged binary on both platforms.

Physical acceptance also found more harness defects than product defects, two of
them false passes. The `v0.5.0-rc.6` dispatch records them so a later run does
not repeat them.

Dispatch:
[`v0.5.0-rc.6`](docs/testing/v0.5.0-rc.6-agent-verification-prompts.md).
Contract:
[Phase 5 acceptance](docs/testing/phase-5-universal-agent-coverage-acceptance.md).

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.5.0`. Current stable remains `v0.4.0`.

### v0.5.0 stable evidence

Dual-platform tagged-artifact acceptance PASS on candidate `v0.5.0-rc.6`:

- Apple Silicon macOS 150/150
  (`docs/testing/results/2026-08-21-macos-phase5-V050RC6.md`)
- native Windows x64 150/150
  (`docs/testing/results/2026-08-21-windows-phase5-V050RC6.md`)

This authorizes the signed stable `v0.5.0` tag. Intel macOS and Linux/WSL2
remain optional and unverified. Native resume and encrypted sync remain
same-vendor and remain limited to Claude Code and Codex CLI. Gemini CLI,
OpenCode, and Grok Build remain handoff sources only; Gemini was **not**
promoted to T3.

Two dispositions are carried into this release rather than resolved by it, and
both are recorded in the device reports. Both are fixed after `v0.5.1`; the
records below describe `v0.5.0` and are left as they were written.

- **E5** — active-session detection is implemented and exercised on the handoff
  and restore paths; `rein resume` applies no such guard.
- **B4 on Windows** — the row passes there without exercising the fix, because
  that host's OpenCode object stores hold only packed objects and OpenCode
  declares no root environment variable to redirect the probe with. The
  end-to-end evidence is on macOS; the Windows evidence is the unit tests that
  cover the same code path in CI.

Five candidates were published and failed physical acceptance before this one:
`v0.5.0-rc.1` through `v0.5.0-rc.5`. The defect that `v0.5.0-rc.5` acceptance
found — an agent probe carrying a raw Git object hash — is why `v0.5.0-rc.6`
exists.

## Steps

### 1. Prepare the release commit

The release commit itself must contain both public bootstrap files pinned to
the exact new CLI tag, with each canonical installer digest recomputed from the
final `scripts/install.*` bytes. It must also contain synchronized changelog,
compatibility, citation, website release truth, and candidate-dispatch updates.
Do not tag a commit whose bootstraps still name the previous release: the
signed website deployment later requires those files to be byte-identical to
the CLI tag, so a post-tag pin-only edit cannot repair it.

```bash
# Update release truth, public bootstrap pins, and compatibility evidence.
git add --all
git commit -m "chore(release): vX.Y.Z"

# Verify the exact clean release commit before pushing its PR branch.
GOTOOLCHAIN=go1.25.13 go mod tidy -diff
make verify
make snapshot
./scripts/stage-release-assets.sh dist
./scripts/check-release-artifacts.sh dist
sh scripts/test-install.sh dist
git diff --exit-code -- go.mod go.sum
test -z "$(git status --porcelain)"

git push -u origin release/vX.Y.Z
# Open a draft PR, pass protected-main CI, review, and merge.
```

GoReleaser's before hook runs `go mod tidy`; the post-snapshot diff and clean
check prove that it did not silently change the committed module graph or any
other tracked release input. `stage-release-assets.sh` is required before the
artifact and installer checks because raw GoReleaser binaries originate inside
target-specific directories and must be staged under their checksummed release
asset names.

### 2. Tag and push

```bash
export REINSTATE_SIGNING_KEY="$HOME/.ssh/reinstate_release_signing"
git -c gpg.format=ssh \
  -c user.signingkey="$REINSTATE_SIGNING_KEY" \
  tag -s vX.Y.Z -m "Reinstate vX.Y.Z"
git push origin vX.Y.Z
```

The tag must point at the reviewed commit on protected `main`. Do not move or
reuse a published tag. The matching public key and maintainer principal must
be present in `.github/allowed_signers` so CI can verify the signature without
depending on machine-local keyring state.

### 3. GitHub Release workflow

The release workflow builds binary and source archives, generates checksums and
per-binary-archive SBOMs, tests installer contracts, publishes a draft release,
and creates GitHub artifact attestations.

Before publishing the draft:

1. Confirm asset names match `reinstate_<version-without-v>_<os>_<arch>`.
2. Run both official installers against the exact draft assets.
3. Verify checksums and `gh attestation verify` for each archive.
4. Confirm archive contents include binary, license, notice, README, and changelog.
5. Mark prerelease tags as pre-release; publish stable only after every release gate.

Publishing the verified GitHub draft triggers **Publish package managers**.
That workflow re-verifies the signed tag, main ancestry, checksums, and GitHub
attestations before any enabled downstream job can run. Registry jobs are
opt-in and protected by the `package-publish` environment. Complete the account,
secret, repository-variable, and first-publication steps in
[Package-manager publishing](docs/package-manager-publishing.md) before enabling
them.

Stable releases may promote to npm, JSR, Homebrew, Chocolatey, Scoop, WinGet,
and AUR. Prereleases promote only to npm (`next`) and JSR. Do not publish a
registry package from a draft release or from locally rebuilt binaries.

For stable `v0.2.0`, complete both the **Stable v0.2.0 publication reminder**
and the **Post-publication documentation reminder** in
[Package-manager publishing](docs/package-manager-publishing.md). Enabled CI is
not evidence that a package is publicly listed, accepted by an external
registry, or verified on its native platform.

The GitHub draft now also contains raw binaries and `.deb`, `.rpm`, `.apk`, and
Arch package files. Confirm those files are covered by `checksums.txt` and
artifact attestations before publishing the draft.

### 4. Publish the public installer routes

Automatic Vercel Git deployments are disabled. The public bootstrap pins were
already committed before the CLI tag in Step 1. After the GitHub release is
published, confirm both tagged bootstrap files pin that same release, update a
clean local `main`, and create a signed, annotated
`website-vYYYY.MM.DD.N` tag at the exact `origin/main` commit. Link the existing
Vercel project if necessary. Push the tag and wait for the
**Validate signed website deployment tag** workflow to pass, then run:

```bash
./scripts/deploy-website-production.sh website-vYYYY.MM.DD.N
```

The script derives the CLI release tag independently from `install.sh` and
`install.ps1`, refuses a mismatch, and verifies that release before it deploys.
It deploys without moving the production alias, verifies both installers
against the derived CLI release at the immutable deployment URL, promotes only
that verified deployment, and verifies both live routes again. Never run
`vercel --prod` directly for a release.

For a release candidate, start its committed candidate-specific acceptance
dispatch only after both live routes install the new exact version. Apple
Silicon macOS and native Windows x64 own the mandatory two-device matrix;
provision Windows per
[windows-acceptance-host.md](docs/testing/windows-acceptance-host.md) before
product rows. A passing candidate matrix certifies only that exact tagged
candidate. Stable promotion remains a separate reviewed decision with fresh
tagged-artifact evidence on both supported platforms. Native macOS amd64 and
WSL2 amd64 are unsupported/unverified optional evidence and do not block a
candidate or stable promotion.

### 5. Publish website-only changes

A website deployment is not a CLI release. Use a signed, annotated
`website-vYYYY.MM.DD.N` deployment tag when reviewed website changes need to
ship without advancing the current Reinstate version. The website tag must
point at the exact current `origin/main` commit; it must not be published as a
GitHub Release or described as a new CLI version.

The public installer identity remains explicit inside both committed bootstrap
files. The script derives their CLI release tag, requires them to agree, and
verifies the corresponding published release. Push the website tag and wait
for the **Validate signed website deployment tag** workflow to pass before
running:

```bash
./scripts/deploy-website-production.sh website-vYYYY.MM.DD.N
```

For example, while Reinstate remains current:

```bash
./scripts/deploy-website-production.sh website-v2026.07.28.1
```

The guarded script verifies the website tag signature, requires it to match
clean local `main` and `origin/main`, verifies the derived signed CLI release,
and requires both committed public installers to match that release before
deploying. It then applies the same immutable-deployment checks, installer byte
comparisons, production discovery smoke tests, promotion, and live-origin
verification. A `website-v...` tag does not satisfy or replace any CLI release,
compatibility, or acceptance gate.

### 6. Announce (optional)

- GitHub Discussions "Show and tell" / announcements
- X/Twitter [@HarjjotSinghh](https://x.com/HarjjotSinghh)
- Relevant community threads (only when the release is useful, not spam)

## Hotfix releases

1. Branch from the release tag if needed: `release/x.y`
2. Cherry-pick the fix
3. Bump **PATCH**, release as above
4. Merge back to `main`

## Rollback

Published releases are immutable. If a defect escapes, issue a new patch or
prerelease. Do not move the tag or replace assets.
