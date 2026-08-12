# Release contribution process

The maintainer runbook is [RELEASING.md](../../RELEASING.md). Contributors may
prepare release fixes and evidence, but only maintainers publish or replace
release state.

## Version policy

Reinstate follows Semantic Versioning and signed `vMAJOR.MINOR.PATCH` tags.
Pre-release tags may add `-alpha.N`, `-beta.N`, or `-rc.N`.

A feature is versioned when its reviewed completion commit lands in the release
history. Do not create speculative tags, move published tags, or call an
untested build stable.

Signed `website-vYYYY.MM.DD.N` tags are a separate maintainer-operated
deployment identity for reviewed website-only changes. They do not version the
CLI, create a GitHub Release, advance the changelog's current release, or close
compatibility and acceptance gates. The deployment script derives the existing
signed `vMAJOR.MINOR.PATCH[-PRE-RELEASE]` identity from both public bootstrap
files and refuses to proceed if their pins differ.

## Release-impact checklist

- Add a CHANGELOG entry.
- Document config/schema migration impact.
- Update exact compatibility evidence.
- Run `make verify`.
- Build a clean GoReleaser snapshot.
- Verify archives, checksums, SBOMs, source archive, and installers.
- Publish installer routes only through the signed-tag production deployment
  script and verify both immutable and live URLs byte for byte.
- For a website-only change, use
  `./scripts/deploy-website-production.sh website-vYYYY.MM.DD.N`; preserve exact
  installer parity with the CLI release derived from both public bootstraps.
- Leave RC and stable tagging/publication to the authorized maintainer. Phase 4
  requires Apple Silicon macOS and native Windows x64 acceptance; Intel macOS
  and Linux/WSL2 are optional, unsupported/unverified evidence and do not block
  RC2 or stable `v0.4.0`.
