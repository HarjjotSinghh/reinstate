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

## Install in a repository

```bash
mkdir -p .agents/skills
cp -R reinstate-seo-agent-skills/* .agents/skills/
```

For a Claude Code setup that reads `.claude/skills`:

```bash
mkdir -p .claude/skills
cp -R reinstate-seo-agent-skills/* .claude/skills/
```

Use one directory as the canonical source. Copy or symlink intentionally so the versions do not drift.

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
