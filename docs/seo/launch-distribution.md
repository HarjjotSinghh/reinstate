# Release discoverability and launch distribution

This is an execution runbook, not evidence that a launch or external submission
has happened. Reinstate is currently `v0.1.0-rc.6`; native-device,
macOS-amd64, WSL2, and complete two-device acceptance gates are still open.
Do not publish a stable-launch narrative until the release evidence closes
those gates.

## Release input contract

Before preparing distribution for a tag, collect:

- exact release tag and immutable commit;
- commit range since the prior public release;
- automated test and installer results;
- physical compatibility and two-device acceptance results;
- migration or storage-format notes;
- security-relevant changes;
- verified limitations and open gates; and
- the list of added, materially updated, removed, or recanonicalized URLs.

Classify each proposed statement as `verified`, `planned`, `ambiguous`, or
`unsupported`. Only verified statements belong in release metadata,
indexable pages, schema, or launch copy. Planned statements must remain visibly
roadmap-qualified.

## Synchronized release checklist

For one reviewed release commit:

- [ ] update `[Unreleased]` and create the tagged changelog section;
- [ ] publish technical release notes with added, changed, fixed, deprecated,
      removed, and security sections as applicable;
- [ ] update `website/src/data/product.ts`;
- [ ] update README, getting-started, integrations, compatibility, limitations,
      security, and relevant guides;
- [ ] update exact tested agent versions, OS evidence, `lastTested`, and source
      links in `website/src/data/compatibility.json`;
- [ ] update metadata, schema, RSS, `llms.txt`, and Open Graph copy only when
      product facts changed;
- [ ] run the freshness audit and record meaningful review dates;
- [ ] build and run every website quality gate;
- [ ] deploy the exact reviewed commit;
- [ ] run production discovery smoke tests;
- [ ] generate and review the IndexNow delta plan;
- [ ] submit the delta only after the new URLs and ownership proof are live;
- [ ] inspect launch-critical URLs in Google and Bing when authorized;
- [ ] run the fixed AI-query baseline manually;
- [ ] update GitHub release text, About text, topics, and social profiles from
      the same verified definition; and
- [ ] attach evidence and owners for every remaining failure.

## Canonical external description

Use this while RC6 remains the current release:

> Reinstate is an open-source continuity layer that synchronizes encrypted
> Claude Code and Codex sessions across configured devices using storage the
> developer controls. Native resume remains same-vendor, and stable Phase 1
> platform acceptance is still in progress.

Suggested GitHub About description:

> Encrypted cross-device session continuity for Claude Code and Codex.

Suggested repository topics, after the website branch is deployed and the
owner reviews them:

```text
coding-agents
claude-code
codex
developer-tools
end-to-end-encryption
golang
local-first
open-source
s3
session-sync
```

Do not add agent or platform topics that are only on the roadmap.

## Channel plan

| Channel | Asset | Release gate | Evidence to retain |
| --- | --- | --- | --- |
| Website | changelog, compatibility, guides, RSS | exact commit deployed | production smoke JSON and deployment URL |
| GitHub release | technical summary, assets, checksums, limitations | tag and artifacts immutable | release URL and asset/checksum log |
| README/About/topics | canonical definition and current support | website/release facts synchronized | before/after capture |
| Mailing list | concise outcome, limitations, links | privacy review and authorized list | sent-message record and audience source |
| Founder/project social | one factual problem/outcome post | public URLs return `200` | post URL and UTC time |
| Show HN | working project, source, setup path, candid limitations | trial is genuinely usable | submission URL and discussion follow-up |
| Relevant communities | tailored answer to a real community need | community rules reviewed | rule URL, post URL, moderator action if any |
| Newsletters/directories | accurate listing and primary evidence | listing category fits | submission and live listing URLs |
| Podcasts/interviews | technical evidence and maintainer availability | no unsupported announcement | episode/pitch record |

Never mass-submit, buy links, automate community posting, conceal affiliation,
or reuse a generic pitch where it does not answer the community's question.

## Launch-post source outline

Draft only after the release gate is satisfied:

1. the cross-device continuity problem in one concrete example;
2. the exact supported agents, OS targets, and current release;
3. the same-vendor resume boundary;
4. how client-side encryption and user-owned S3-compatible storage work;
5. path remapping and why raw directory copying is insufficient;
6. a short reproducible setup or demo;
7. evidence links: compatibility JSON, architecture, security, tests, source;
8. candid limitations and remaining roadmap;
9. the request: try a synthetic/non-sensitive session and report a specific
   class of failure; and
10. maintainer affiliation and Apache-2.0 license.

Avoid “seamless,” “universal,” benchmark claims, customer claims, security
absolutes, or cross-agent transcript translation.

## Linkable assets and evidence gates

| Asset | Current state | Publication gate |
| --- | --- | --- |
| Machine-readable compatibility matrix | Implemented | keep sources and review dates current |
| Session format map | Planned | sanitized primary research and methodology |
| Archive inspector | Planned | safe local-only tool and threat review |
| Migration readiness checker | Planned | diagnostic behavior released and documented |
| Storage validator | Partly available through `rein setup check` | public asset needs scoped UX and no secret collection |
| Cross-device path mapper | Planned | released behavior plus synthetic fixtures |
| Restoration benchmark | Not published | reproducible method, raw synthetic data, hardware/OS versions, failures |
| Threat model | Architecture/security material exists | standalone review before calling it a formal threat model |
| Adapter starter kit | Planned | supported extension contract and tested tutorial |

Each published asset needs one canonical URL, author/owner, methodology,
meaningful update date, limitations, and downloadable evidence when useful.

## Measurement and follow-up

For every distribution action, record:

| UTC time | Channel | Canonical URL | Message/version | Owner | Referral/citation evidence | Follow-up |
| --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |

Review crawl/indexing changes and factual corrections daily during the launch
window, weekly while the release is active, and in the monthly audit. A mention
without a link, an IndexNow `202`, or a search-console discovery is not proof of
indexing, ranking, conversion, or AI citation.

