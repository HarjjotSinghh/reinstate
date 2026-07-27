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

## Release-impact checklist

- Add a CHANGELOG entry.
- Document config/schema migration impact.
- Update exact compatibility evidence.
- Run `make verify`.
- Build a clean GoReleaser snapshot.
- Verify archives, checksums, SBOMs, source archive, and installers.
- Publish installer routes only through the signed-tag production deployment
  script and verify both immutable and live URLs byte for byte.
- Leave stable tagging and publication to the authorized maintainer after all
  native Windows, WSL2, and macOS acceptance rows pass.
