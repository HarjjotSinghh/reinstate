# Releasing

How maintainers cut a **Reinstate** release.

## Versioning

- **Semantic Versioning**: `MAJOR.MINOR.PATCH`
- Pre-1.0: breaking changes allowed in `MINOR` with clear CHANGELOG entries
- Git tags are prefixed: `v0.1.0`

## Preconditions

- [ ] `main` is green on CI
- [ ] CHANGELOG `[Unreleased]` section is accurate
- [ ] Version bumped in relevant files (`go.mod` module path stable; version in
      `cmd/reinstate` / `internal/version`)
- [ ] No open P0 security issues

## Steps

### 1. Prepare the release commit

```bash
# Update CHANGELOG: move Unreleased → ## [x.y.z] - YYYY-MM-DD
# Bump version constants if needed
git add -A
git commit -m "chore(release): vX.Y.Z"
git push origin main
```

### 2. Tag and push

```bash
git tag -a vX.Y.Z -m "Reinstate vX.Y.Z"
git push origin vX.Y.Z
```

### 3. GitHub Release

The `release` workflow (when enabled) builds multi-arch binaries and attaches
checksums. Otherwise create a Release manually from the tag:

1. GitHub → Releases → Draft a new release
2. Target tag `vX.Y.Z`
3. Title: `vX.Y.Z`
4. Body: paste the CHANGELOG section for this version
5. Attach artifacts / let the workflow upload them
6. Mark as **Latest release** (or pre-release if `alpha`/`beta`/`rc`)

### 4. Announce (optional)

- GitHub Discussions "Show and tell" / announcements
- X/Twitter [@HarjjotSinghh](https://x.com/HarjjotSinghh)
- Relevant community threads (only when the release is useful, not spam)

## Hotfix releases

1. Branch from the release tag if needed: `release/x.y`
2. Cherry-pick the fix
3. Bump **PATCH**, release as above
4. Merge back to `main`

## Rollback

GitHub Releases cannot unpublish downloads easily — prefer a new patch that
fixes the issue. If a tag must be moved (rare, pre-1.0 only), coordinate in
maintainers chat and document in CHANGELOG.

## npm / package registries

If/when npm packages are published (`@harjjotsinghh/reinstate` or similar):

```bash
# after git tag
npm publish --access public
```

Binary-first distribution remains the primary path (static Go binaries + curl
install script).
