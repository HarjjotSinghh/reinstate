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
- [ ] macOS arm64, macOS amd64, native Windows amd64, and WSL2 acceptance rows pass
- [ ] Claude Code and Codex exact versions/layouts are recorded in compatibility docs
- [ ] Wrong-passphrase, tamper, backup, rollback, conflict, and installer tests pass
- [ ] Snapshot archives, source archive, checksums, and SBOMs were inspected
- [ ] Builds and vulnerability scans use the pinned Go 1.25.12 toolchain

## Steps

### 1. Prepare the release commit

```bash
# Update CHANGELOG and compatibility evidence.
make verify
goreleaser release --snapshot --clean
sh scripts/test-install.sh dist

git add CHANGELOG.md docs/compatibility.md
git commit -m "chore(release): vX.Y.Z"
git push origin main
```

### 2. Tag and push

```bash
export REINSTATE_SIGNING_KEY="$HOME/.ssh/id_ed25519"
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

### 4. Submit the WinGet manifest (stable releases only)

Every stable version needs its own pull request against
[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs); WinGet has no
auto-following of GitHub Releases. Skip prereleases.

```powershell
wingetcreate update HarjjotSinghRana.Reinstate `
  --version <VERSION_NO_V> `
  --urls https://github.com/HarjjotSinghh/reinstate/releases/download/v<VERSION_NO_V>/reinstate_<VERSION_NO_V>_windows_amd64.zip `
  --submit
```

Notes:

- `InstallerType: zip` with `NestedInstallerType: portable` is correct and
  intentional; the archive carries `rein.exe` and `reinstate.exe`. Automated
  reviewers sometimes question the extension. Confirm and move on.
- The Microsoft CLA bot requires one comment on the first PR from an account:
  `@microsoft-github-policy-service agree`.
- The PR description checklist is a template and does not gate merge. The
  gating labels are `Azure-Pipeline-Passed`, `Validation-Completed`, and
  `Moderator-Approved`.
- Only publish a version to WinGet after the GitHub Release is published and
  the assets are final. Manifests are immutable once merged.

### 5. Announce (optional)

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
