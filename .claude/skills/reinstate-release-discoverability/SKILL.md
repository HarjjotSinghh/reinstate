---
name: reinstate-release-discoverability
description: Convert a verified Reinstate release into synchronized changelog, documentation, compatibility, metadata, sitemap, IndexNow, GitHub, and launch updates without overstating capabilities.
---

# Reinstate Release Discoverability

## Inputs

- release tag
- commit range or diff
- test results
- compatibility results
- migration notes
- security notes
- known limitations

Use `reinstate-product-truth`.

## Workflow

1. Read the release diff and tests.
2. list verified user-visible changes.
3. list compatibility changes.
4. list breaking changes.
5. identify stale pages.
6. update product truth only for released behavior.
7. create:
   - changelog entry
   - technical release notes
   - compatibility updates
   - docs updates
   - metadata changes
   - schema changes
   - GitHub release summary
   - launch post draft
8. identify new or changed canonical URLs.
9. update sitemap automatically through the build.
10. submit changed URLs through IndexNow if configured.
11. add fixed AI-query tests for new capabilities.
12. report evidence for every claim.

## Output rules

- distinguish added, changed, fixed, deprecated, removed, and security
- state agent and operating-system versions when relevant
- state migration steps
- state limitations
- do not use "seamless" unless failure modes are genuinely absent
- do not call a beta feature stable
