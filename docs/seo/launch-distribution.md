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

## GitHub repository discoverability runbook

This section describes an owner-operated GitHub change. It does not authorize
an agent to edit repository metadata, upload the image, create a release, or
publish a post.

### Go/no-go record

Before changing any public repository surface, create one record containing:

- [ ] the exact reviewed commit and `v0.1.0-rc.6` tag;
- [ ] a passing website build and quality-gate log from that commit;
- [ ] a production deployment URL and discovery smoke-test output;
- [ ] the native macOS, macOS-amd64, native Windows, WSL2, and two-device
      acceptance results, with open results explicitly marked;
- [ ] immutable installer URLs, checksums, and verification output;
- [ ] a review of the release summary and launch post below against the final
      acceptance evidence; and
- [ ] the operator, reviewer, UTC time, and a `GO` or `NO-GO` decision.

Any unchecked acceptance item is a `NO-GO` for the launch post. Repository
metadata may be staged only if it remains accurate without implying stable or
complete acceptance.

### Prepare and verify the social preview

The upload-ready file is
`website/public/brand/github-social.png`. It is generated from the same Satori
renderer as the website's route-specific Open Graph cards and mirrored at
`assets/brand-kit/png/github-social-1280x640.png`.

```bash
cd website
npm run generate:github-social
npm run check:github-social
file public/brand/github-social.png
shasum -a 256 public/brand/github-social.png
```

Expected image contract:

- PNG with a solid paper background;
- exactly `1280×640`;
- less than `1,000,000` bytes;
- the bare header/footer logo plus `Reinstate` wordmark;
- Questrial display type and Geist supporting copy; and
- the landing-page paper, ink, chartreuse, yellow, blue, and isometric
  continuity-stack language.

GitHub's official
[social-preview documentation](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/customizing-your-repositorys-social-media-preview)
accepts PNG, JPG, or GIF under 1 MB, recommends at least `640×320`, and
recommends `1280×640` for the best display.

### Apply About metadata and topics

After an authorized owner records `GO`:

1. Open the repository's main page while signed in with repository-admin
   access.
2. Select the gear icon beside **About**.
3. Set **Description** exactly to:
   `Encrypted cross-device session continuity for Claude Code and Codex.`
4. Set **Website** to `https://reinstate.dev`.
5. Replace the topic set with the exact reviewed list above. Topics use
   lowercase letters and hyphens; do not accept suggested topics merely
   because GitHub offers them.
6. Save changes.
7. Reload the public repository page in a signed-out window and verify the
   description, canonical website link, and all ten topics.
8. Record a screenshot, repository URL, operator, UTC time, and the final topic
   list in the launch evidence.

The topic workflow follows GitHub's official
[repository-topic instructions](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/classifying-your-repository-with-topics).

### Apply the social preview

After the same authorized owner confirms that the generated image passed the
contract:

1. Open the repository's main page and select **Settings**.
2. Find **Social preview**, select **Edit**, then **Upload an image**.
3. Upload `website/public/brand/github-social.png`; do not substitute
   `brand/og.png` or a route-specific `1200×630` card.
4. Verify the preview shown by GitHub is uncropped, readable, and uses the bare
   logo-and-wordmark lockup.
5. Record the asset SHA-256, a screenshot, operator, UTC time, and repository
   URL. Cache propagation is not proof of a launch; recheck shared-link
   rendering after GitHub has refreshed it.

### Rollback

If the website URL, support claims, release status, or preview is wrong, use
the same About editor to restore the last verified metadata and use
**Settings → Social preview → Edit** to upload the last verified asset or
remove the image. Record the reason, UTC time, operator, and before/after
screenshots.

## RC6 GitHub release summary — draft, evidence-gated

> **DRAFT — DO NOT PUBLISH OR REPLACE THE EXISTING RELEASE TEXT BEFORE THE
> GO/NO-GO RECORD ABOVE IS COMPLETE AND REVIEWED.**

Suggested title:

> Reinstate v0.1.0-rc.6 — encrypted same-vendor session continuity

Suggested body:

> Reinstate `v0.1.0-rc.6` is a pre-1.0 release candidate for testing encrypted
> Claude Code and Codex session continuity across configured devices through
> S3-compatible storage you control.
>
> RC6 improves Codex project-path handling by resolving rollout working
> directories to configured canonical project IDs, excluding unmapped
> projects, normalizing exported roots, and rejecting duplicate mappings. It
> also validates an additional device's encrypted remote manifest with a
> readable object request, reports `would pull` accurately during dry runs,
> and keeps missing-config errors redacted.
>
> Native resume is same-vendor: Claude Code sessions resume in Claude Code and
> Codex sessions resume in Codex. Reinstate does not silently translate
> transcripts between agents. Cross-agent work remains an explicit portable
> handoff direction, not an RC6 claim.
>
> Install on macOS:
>
> ```bash
> curl -fsSL https://reinstate.dev/install.sh | sh
> ```
>
> Install on Windows PowerShell:
>
> ```powershell
> irm https://reinstate.dev/install.ps1 | iex
> ```
>
> Both bootstraps pin and verify `v0.1.0-rc.6`. Review the installer before
> running it if your environment requires that workflow.
>
> Start with a synthetic, non-sensitive session and follow the getting-started,
> compatibility, security, and limitations pages at
> <https://reinstate.dev/docs/getting-started/>.
>
> **Acceptance status:** native macOS, macOS-amd64, native Windows, WSL2, and
> complete two-device acceptance evidence must be linked here before this
> draft is published. Until then, do not describe RC6 as stable, generally
> available, production-ready, seamless, universal, or Phase 1 complete.
>
> Reinstate is open source under Apache-2.0:
> <https://github.com/HarjjotSinghh/reinstate>.

Before publication, replace the acceptance-status paragraph with links to the
actual evidence while retaining any failed, skipped, or untested targets.
Never turn a missing result into a compatibility claim.

## RC6 launch post — draft, evidence-gated

> **DRAFT — DO NOT PUBLISH BEFORE THE GO/NO-GO RECORD IS `GO`, THE EXACT
> REVIEWED COMMIT IS LIVE, AND EVERY CLAIM BELOW HAS LINKED EVIDENCE.**

> Coding-agent sessions are useful project context, but they are usually tied
> to one agent's local files on one computer. Moving the repository alone does
> not restore that working state.
>
> Reinstate is an open-source continuity layer for that problem. Its current
> `v0.1.0-rc.6` release candidate can sync encrypted Claude Code and Codex
> sessions across configured macOS and Windows devices through S3-compatible
> storage you control. Path remapping handles the project-root differences
> that show up when the same work moves between machines.
>
> The boundary is deliberate: native resume stays same-vendor. Claude Code
> resumes Claude Code sessions; Codex resumes Codex sessions. Reinstate is not
> an editor, agent scheduler, credential-sync service, or silent cross-agent
> transcript translator.
>
> If the linked acceptance evidence covers your device pair, try it first with
> a synthetic, non-sensitive session:
>
> - setup: <https://reinstate.dev/docs/getting-started/>
> - compatibility evidence: <https://reinstate.dev/compatibility/>
> - security model: <https://reinstate.dev/security/>
> - known limitations: <https://reinstate.dev/docs/limitations/>
> - source and Apache-2.0 license:
>   <https://github.com/HarjjotSinghh/reinstate>
>
> I maintain Reinstate. If you test it, please report the agent, exact version,
> source and destination OS, path mapping, and the failing command or resume
> step—without attaching a real session transcript or credentials.

Before posting, add links to the exact tag, checksums, compatibility record,
and two-device evidence. Remove any OS or agent combination that did not pass.
Do not add download counts, user counts, performance claims, testimonials, or
security absolutes without primary evidence.

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
