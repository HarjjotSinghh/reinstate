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
- [ ] Builds and vulnerability scans use the pinned Go 1.25.12 toolchain

### Supported platform boundary

The maintainer-approved `v0.2.0` reconciliation in
`docs/testing/results/2026-08-02-macos-phase2-V020RC2.md` permits stable
publication with verified support limited to Apple Silicon macOS
(`darwin/arm64`) and native Windows x64 (`windows/amd64`). Those remain the
supported mandatory platforms for Phase 3 RC and stable acceptance.

Intel macOS and Linux/WSL2 artifacts are unsupported/unverified optional
evidence: they may be built, checksummed, SBOM-covered, and attested, but their
absence or failure does not block RC1 or stable `v0.3.0`. Never describe them as
physically certified or supported. Issues #97 and #98 track that optional
physical evidence.

### v0.3.0-rc.1 candidate evidence

The committed
[`v0.3.0-rc.1` dispatch](docs/testing/v0.3.0-rc.1-agent-verification-prompts.md)
required two tagged-artifact reports: Apple Silicon macOS and native Windows
x64. macOS passed (32/32); native Windows failed (9 PASS / 23 FAIL) with six
root blockers, including Windows executable trust for extensionless `codex`
and host/tooling gaps. RC1 does **not** authorize stable `v0.3.0`.

### v0.3.0-rc.2 candidate evidence

The next Phase 3 candidate uses
[`v0.3.0-rc.2` dispatch](docs/testing/v0.3.0-rc.2-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). Those two
device reports decide RC2 tagged-artifact acceptance only. Stable promotion
still requires a separate reviewed stable decision and fresh tagged-artifact
validation on the same two supported platforms. Intel macOS and WSL2 remain
unsupported/unverified optional evidence and are never stable blockers.

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
GOTOOLCHAIN=go1.25.12 go mod tidy -diff
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
dispatch only after both live routes install the new exact version. For
`v0.3.0-rc.2`, Apple Silicon macOS and native Windows x64 own the mandatory
two-device matrix; provision Windows per
[windows-acceptance-host.md](docs/testing/windows-acceptance-host.md) before
product rows. Passing those two reports means only that RC2 passed
tagged-artifact acceptance. Stable `v0.3.0` still requires a separate promotion
decision and fresh/reconciled tagged-artifact results from those two supported
platforms. Native macOS amd64 and WSL2 amd64 are unsupported/unverified optional
evidence and do not block RC2 or stable `v0.3.0`.

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
