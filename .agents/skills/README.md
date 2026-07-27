# Reinstate SEO Agent Skills

This pack contains repository-local Agent Skills for implementing and maintaining SEO, AEO, and AI Search Optimization for Reinstate.

## Skills

- `reinstate-product-truth`
- `reinstate-technical-seo`
- `reinstate-structured-data`
- `reinstate-content-brief`
- `reinstate-answer-optimization`
- `reinstate-ai-search`
- `reinstate-seo-ci`
- `reinstate-release-discoverability`
- `reinstate-site-audit`

## Repository discovery

The reviewed canonical copies live in `.agents/skills`, the portable Agent
Skills discovery path used by Codex and other compatible agents.

Claude Code discovers the same nine reviewed skills under `.claude/skills`.
Those files are deliberate mirrors rather than symlinks so Windows checkouts
remain reliable. `TestSEOAgentSkillsStayPortableAndInSync` fails if names,
frontmatter, or contents drift between the two locations.

When updating a skill, change the canonical `.agents/skills` copy, mirror the
same file under `.claude/skills`, and run `go test ./internal/doctest`.

## Suggested first task

```text
Use reinstate-product-truth, reinstate-technical-seo,
reinstate-structured-data, and reinstate-seo-ci.

Audit website/ and implement the P0 technical SEO work. Preserve the current
design, use only released product claims, add tests, run the production build,
and report changed files and remaining risks.
```

## Security

Review every skill before installation. Skills can contain instructions and code. Keep production credentials out of agent context and review all generated diffs.
