# Third-party discoverability skill review

Status: **reviewed, not installed, not approved**  
Last reviewed: **2026-07-27**

Reinstate uses the nine repository-owned skills under `.agents/skills` and
`.claude/skills`. Third-party skills remain reference material until a
maintainer approves an exact immutable revision after script, network,
dependency, permission, and output review.

An HTTP 200 response from a marketplace page is not proof that a named skill
still exists upstream, and an upstream `SKILL.md` is not authoritative search
documentation. Product truth and current first-party platform guidance always
override a third-party instruction.

## Revisions inspected

| Repository | Revision inspected | Approval |
| --- | --- | --- |
| `coreyhaines31/marketingskills` | [`c21a984a56da10fb6085e6334f6f60929220a4da`](https://github.com/coreyhaines31/marketingskills/tree/c21a984a56da10fb6085e6334f6f60929220a4da) | None |
| `addyosmani/web-quality-skills` | [`95d6e255afe1596b557d7a8498517884438f5b3a`](https://github.com/addyosmani/web-quality-skills/tree/95d6e255afe1596b557d7a8498517884438f5b3a) | None |
| `sanity-io/agent-toolkit` | [`af54474c21b00aee8e2fa2855b8ff6ef8a0cf41c`](https://github.com/sanity-io/agent-toolkit/tree/af54474c21b00aee8e2fa2855b8ff6ef8a0cf41c) | None |
| `alirezarezvani/claude-skills` | [`aa8d778811a557a2c28ccadda4cf3d0bd028a4cc`](https://github.com/alirezarezvani/claude-skills/tree/aa8d778811a557a2c28ccadda4cf3d0bd028a4cc) | None |

These are review snapshots, not approved install pins. Re-review the immutable
tree and all transitive behavior before any future installation.

## Decisions

| Skill | Current upstream name | Decision | Reason |
| --- | --- | --- | --- |
| SEO audit | `seo-audit` | Defer | Script-free at the inspected revision, but substantially overlaps `reinstate-site-audit`; no extra capability justifies another instruction source yet. |
| Technical SEO | `seo` | Reject unmodified | Uses invented fixed ranking-factor weights. Reinstate will not encode unsupported weights as fact. |
| Core Web Vitals | `core-web-vitals` | Defer | Potentially useful after a pinned review, but browser measurement should be implemented directly and interpreted against current official web.dev guidance. |
| Schema markup | `schema` | Defer | Former name `schema-markup` is stale. Generic FAQ/HowTo and rich-result guidance needs current Google feature filtering. |
| Content strategy | `content-strategy` | Defer | Potentially useful, but must remain subordinate to Reinstate product truth, evidence, and the no-fan-out policy. |
| Product marketing | `product-marketing` | Do not install now | Former name `product-marketing-context` is stale. It writes a parallel product-context file and would compete with Reinstate's canonical truth source. |
| SEO/AEO baseline | `seo-aeo-best-practices` | Reject as authority | Framework-specific guidance conflates training controls with search/citation discovery. |
| AEO | `aeo` | Reject | Bundled scripts fetch URLs and write files, may insert placeholder JSON-LD/citation markers, and contain unsupported claims about citation density, training cycles, and bot effects. |
| Programmatic SEO | `programmatic-seo` | Later only | Do not enable template fan-out before Reinstate has proprietary, verified structured data and a page-level usefulness threshold. |
| Performance | `performance` | Defer | Optional after a pinned review; project-specific browser evidence and budgets take precedence over generic thresholds. |
| Free tools | `free-tools` | Later only | Former name `free-tool-strategy` is stale. A public tool needs a validated user job, threat model, maintenance owner, and non-thin canonical page first. |

## Approval procedure

Before changing any decision to approved:

1. Select an immutable commit or signed release.
2. Inspect `SKILL.md` and every bundled file at that exact revision.
3. Record scripts, network calls, package installation, shell commands, file
   writes, environment reads, requested permissions, and output locations.
4. Compare factual claims against current official Google, OpenAI, Perplexity,
   Astro, Schema.org, web.dev, Codex, and Claude Code documentation.
5. Reject instructions that promise rankings/citations, fabricate weights,
   create placeholder facts/schema, or confuse training and search controls.
6. Test in a disposable worktree with no production secrets.
7. Review all generated diffs and network activity.
8. Run the repository's complete verification gates.
9. Add the exact approved revision and owner to this document.
10. Install only the minimal skill that adds a demonstrated capability.

There is intentionally no third-party skill installation command in CI or the
production repository today.
