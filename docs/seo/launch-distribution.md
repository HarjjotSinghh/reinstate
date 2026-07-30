# Release discoverability and launch distribution

This is an execution runbook, not evidence that a launch or external submission
has happened. Reinstate is currently `v0.1.0`; native-device,
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

Use this while Reinstate remains the current release:

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

- [ ] the exact reviewed commit and `v0.1.0` tag;
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

## Reinstate GitHub release summary — draft, evidence-gated

> **DRAFT — DO NOT PUBLISH OR REPLACE THE EXISTING RELEASE TEXT BEFORE THE
> GO/NO-GO RECORD ABOVE IS COMPLETE AND REVIEWED.**

Suggested title:

> Reinstate v0.1.0 — encrypted same-vendor session continuity

Suggested body:

> Reinstate `v0.1.0` is a stable pre-1.0 release for encrypted
> Claude Code and Codex session continuity across configured devices through
> S3-compatible storage you control.
>
> Reinstate improves Codex project-path handling by resolving rollout working
> directories to configured canonical project IDs, excluding unmapped
> projects, normalizing exported roots, and rejecting duplicate mappings. It
> also validates an additional device's encrypted remote manifest with a
> readable object request, reports `would pull` accurately during dry runs,
> and keeps missing-config errors redacted.
>
> Native resume is same-vendor: Claude Code sessions resume in Claude Code and
> Codex sessions resume in Codex. Reinstate does not silently translate
> transcripts between agents. Cross-agent work remains an explicit portable
> handoff direction, not a Reinstate claim.
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
> Both bootstraps pin and verify `v0.1.0`. Review the installer before
> running it if your environment requires that workflow.
>
> Start with a synthetic, non-sensitive session and follow the getting-started,
> compatibility, security, and limitations pages at
> <https://reinstate.dev/docs/getting-started/>.
>
> **Acceptance status:** native macOS, macOS-amd64, native Windows, WSL2, and
> complete two-device acceptance evidence must be linked here before this
> draft is published. Until then, do not describe Reinstate as stable, generally
> available, production-ready, seamless, universal, or Phase 1 complete.
>
> Reinstate is open source under Apache-2.0:
> <https://github.com/HarjjotSinghh/reinstate>.

Before publication, replace the acceptance-status paragraph with links to the
actual evidence while retaining any failed, skipped, or untested targets.
Never turn a missing result into a compatibility claim.

## Reinstate launch post — draft, evidence-gated

> **DRAFT — DO NOT PUBLISH BEFORE THE GO/NO-GO RECORD IS `GO`, THE EXACT
> REVIEWED COMMIT IS LIVE, AND EVERY CLAIM BELOW HAS LINKED EVIDENCE.**

> Coding-agent sessions are useful project context, but they are usually tied
> to one agent's local files on one computer. Moving the repository alone does
> not restore that working state.
>
> Reinstate is an open-source continuity layer for that problem. Its current
> `v0.1.0` stable release can sync encrypted Claude Code and Codex
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
| Terminology glossary | Implemented at `/glossary` | keep definitions synchronized with current code and roadmap |
| Reinstate encrypted snapshot format v1 | Implemented at `/research/encrypted-snapshot-format-v1` | update only from released schema/source evidence; never call it an open standard |
| Agent-version change tracker | Implemented at `/compatibility/agent-version-history` | update compatibility data, adapter evidence, and tagged release notes together |
| Vendor session-format map | Planned | sanitized primary research, method, exact vendor versions, and review of vendor terms |
| Archive inspector | Planned | safe local-only tool and threat review |
| Migration readiness checker | Planned | diagnostic behavior released and documented |
| Storage validator | Partly available through `rein setup check` | public asset needs scoped UX and no secret collection |
| Synthetic cross-device path mapper | Implemented at `/tools/path-mapping-visualizer` | fixed public fixtures only; no user input, persistence, or analytics |
| Local private-path mapper | Planned | client-only proof, no network or persistence, redaction and browser-threat review |
| Restoration benchmark | Not published | reproducible method, raw synthetic data, hardware/OS versions, failures |
| Threat model | Architecture/security material exists | standalone review before calling it a formal threat model |
| Adapter starter kit | Planned | supported extension contract and tested tutorial |

Each published asset needs one canonical URL, author/owner, methodology,
meaningful update date, limitations, and downloadable evidence when useful.

## Evidence-safe launch packets

These are channel-specific plans, not evidence that publication or outreach has
happened. Every packet must link to the exact deployed commit and preserve the
release-candidate, same-vendor, and open-acceptance boundaries.

### Demo video plan

Produce two short recordings only after the exact release and production site
pass their launch gates:

1. **Install and inspect, 90 seconds.** Show checksum-verifying installation,
   `rein version --json`, `rein setup check`, one synthetic session in
   `rein list`, and the compatibility page. Do not enter credentials, a
   passphrase, a real bucket, or a real session on screen.
2. **macOS → Windows restore, 3–5 minutes.** Use a fresh synthetic repository
   and session. Show the canonical project ID, different redacted local roots,
   scoped push/pull dry-runs, ciphertext-only object inspection, exact native
   resume, and the open platform evidence rows.

Video acceptance:

- record the tag, commit, agent versions, OS builds, installer checksums, UTC
  time, and evidence owners;
- use a dedicated synthetic profile and rotate/delete its temporary storage
  credentials after recording;
- review every frame, terminal scrollback, window title, shell history, browser
  autofill, and file path for secrets or personal data;
- caption commands and expected outcomes, with a transcript linked from a
  canonical website page;
- state that a successful demo is one bounded observation, not a benchmark or
  universal compatibility claim; and
- publish only after both the demonstrated path and public installer pass the
  same release's acceptance record.

Retain the original capture privately, the redacted export, transcript, asset
hash, video URL, and reviewer sign-off. Do not use an edited success montage to
erase failed attempts from the underlying acceptance evidence.

### Awesome-list outreach plan

Awesome lists are editorial repositories, not backlink vending machines.
Before opening one pull request:

1. confirm the list accepts open-source developer tools in Reinstate's actual
   category;
2. read the current contribution rules, alphabetical format, license
   requirements, and self-promotion policy;
3. search the list and open pull requests for duplicates;
4. wait until the documented public setup path works for the exact listed
   release;
5. propose one factual line using the canonical external description, source
   URL, Apache-2.0 license, and current release-candidate qualification; and
6. disclose maintainer affiliation in the pull request.

Keep the rule URL, immutable proposed diff, pull-request URL, review outcome,
and final live entry. Do not submit to loosely related lists, automate parallel
pull requests, add stars, pressure maintainers, or rewrite the description to
fit an unsupported category.

### GitHub Discussion plan

Open one repository Discussion only after Discussions are enabled and the
trial path is usable. Prefer a category such as **Announcements** or
**Show and tell** selected by the maintainer.

The post should contain:

- the concrete work/personal or macOS/Windows continuity problem;
- current release, exact supported agent ranges, and open platform gates;
- a 30-second architecture summary with links to the
  [snapshot format](https://reinstate.dev/research/encrypted-snapshot-format-v1),
  [path visualizer](https://reinstate.dev/tools/path-mapping-visualizer),
  security model, and source;
- a synthetic setup path and a request for one specific class of compatibility
  feedback;
- a warning not to post transcripts, credentials, bucket details, or private
  paths; and
- maintainer affiliation and the Apache-2.0 license.

Pin a correction comment when a release or supported range changes. Archive or
edit the title only with a dated explanation; do not silently leave stale
version support in a high-ranking discussion.

### Architecture post plan

The first architecture article should answer:

> How can an encrypted coding-agent session move between macOS and Windows
> without rewriting transcript prose?

Required sections:

1. the continuity problem and same-vendor boundary;
2. adapter discovery and explicit exclusions;
3. canonical project identity and allow-listed structural fields;
4. the manifest and encrypted snapshot v1 framing;
5. age encryption, object-store metadata leakage, and credential separation;
6. conflict detection, backups, atomic writes, and exact native rediscovery;
7. deterministic synthetic fixtures and remaining physical acceptance gates;
8. rejected approaches, including raw config-tree mirroring and global
   path-string replacement; and
9. source links, reproducible tests, limitations, and a correction policy.

Publication requires maintainer technical review against the tagged source,
one clean diagram with accessible text, a content brief, and a passing
generated-site schema/link/metadata run. Do not introduce performance numbers
until the benchmark publication contract has raw evidence.

## Diagnostic and engineering-as-marketing plans

The tools below remain small continuity diagnostics. They must not become a
hosted transcript processor, agent runtime, credential collector, or raw
vendor-config mirror.

### Archive inspector

Proposed job: tell a developer whether one local `.age` object is recognizable
by a selected Reinstate release, without restoring it.

Minimum safe contract:

- local CLI first; no upload, remote URL, telemetry, or hidden network call;
- require an explicit file and hidden passphrase input;
- default output includes only envelope/manifest version, kind, agent,
  redacted IDs, declared size, hash-verification result, and compatibility
  decision;
- never print transcript text, inner JSONL records, credentials, full private
  paths, passphrases, or decrypted TAR bytes;
- preserve input read-only and write no decrypted temp file unless an explicit
  reviewed export mode is later designed;
- reject wrong passphrases, oversized metadata/payloads, unsupported schemas,
  identity mismatches, unsafe paths, multiple TAR entries, credential paths,
  and hash failures;
- publish synthetic fixtures for every pass/fail result; and
- complete a threat review, cross-platform tests, CLI documentation, and
  redaction tests before the page changes from `Planned`.

The inspector may describe the current Reinstate format; it must not claim it
can inspect arbitrary Claude Code, Codex, or third-party archives.

### Migration readiness checker

Proposed job: answer whether a profile and its encrypted heads can move from
one explicit Reinstate release to another before replacing the binary.

Required inputs and behavior:

- source and target Reinstate versions, local config/state schema versions, and
  read-only encrypted manifest/snapshot metadata;
- an explicit compatibility matrix maintained in released code, not a guessed
  SemVer rule;
- statuses `READY`, `ACTION_REQUIRED`, `UNSUPPORTED`, and
  `EVIDENCE_UNAVAILABLE`, each with a reason and non-mutating next step;
- no mutation, migration, deletion, garbage collection, downgrade, or upload
  in check mode;
- backups, interruption behavior, rollback, and partial-migration recovery
  designed before any future mutating command; and
- fixtures spanning supported, unknown, corrupt, wrong-profile, and downgrade
  cases.

There is no public migration checker in Reinstate. Until a compatibility contract and
tests ship, release notes and independent backups are the only supported
pre-upgrade guidance.

### Path mapper

The first safe asset is the implemented
[fixed synthetic visualizer](https://reinstate.dev/tools/path-mapping-visualizer).
It accepts only
two direction choices and the two current adapters, contains no private input,
disables analytics, and explains the recognized-field boundary.

A later local private-path checker must not launch merely by adding a text box.
Before publication it needs:

- client-only processing with a reviewed build that makes no network requests;
- no analytics script, service worker, cookies, browser storage, crash
  reporting, URL/query serialization, clipboard read, or autofill;
- an explicit “synthetic example” default and a clear warning before private
  input;
- only released `${HOME}` and `${REPO:<id>}` behavior; do not expose the
  lower-level `${WORK:<alias>}` primitive until configuration and adapters
  wire it in;
- no filesystem existence claim from string transformation alone;
- paste/type/Unicode/Windows drive/UNC/WSL/adversarial tests and a threat
  review; and
- a local CLI alternative for environments that do not trust browser input.

Do not call either tool a path migration tool: Reinstate remaps recognized
session structure, not repositories, Git state, runtimes, or full environments.

## Gated research and public-relations angles

These angles can become useful primary research only after their evidence
gates close. They are prohibited launch claims today.

### Open portability standard

Potential future angle:

> An open, implementation-neutral continuity checkpoint for handing coding work
> between agents without pretending native transcripts are interchangeable.

Current status: **not a standard and not a Reinstate feature**. The published
Reinstate snapshot v1 page documents one project's internal encrypted storage
format. It is not a proposal for cross-agent interchange.

Before using “open portability standard,” “standard,” “protocol,” or
“interoperable” in outreach, require:

- a separately versioned public problem statement and scope;
- a vendor-neutral data model that keeps native resume distinct from lossy
  handoff;
- security/privacy threat model, extension and version-negotiation rules, test
  vectors, and conformance criteria;
- governance, license, change process, and trademark/namespace decisions;
- at least two independent implementations with documented interoperability
  results;
- public failure cases, downgrade behavior, and capability-diff semantics; and
- review from affected open-source maintainers without implying vendor
  endorsement.

Until then, acceptable language is “Reinstate plans explicit portable
handoffs.” A repository name, JSON schema, or published spec does not by itself
make a standard.

### Proprietary coding-agent format research

Potential future angle:

> What changed in local coding-agent session formats, and which structural
> fields matter for safe same-vendor continuity?

Publication gates:

- identify exact public agent versions, operating systems, dates, and
  collection method;
- use synthetic sessions generated for research—never real user, employer,
  customer, or contributor transcripts;
- inspect and publish only the minimum structural metadata needed for the
  research question, with secrets and user paths redacted;
- review applicable vendor terms, licenses, responsible-disclosure needs, and
  redistribution limits before publishing fixtures or byte excerpts;
- distinguish public observation from official documentation and never imply
  vendor approval, partnership, or a stable proprietary contract;
- publish reproducible scripts, hashes, schema diffs, failures, unknowns, and a
  correction path where legally and safely permitted; and
- coordinate vulnerability findings privately before public release.

Do not describe reverse-engineered behavior as a vendor guarantee. Do not
publish a fan-out page for every vendor version; maintain one dated canonical
research record with meaningful diffs.

## Measurement and follow-up

For every distribution action, record:

| UTC time | Channel | Canonical URL | Message/version | Owner | Referral/citation evidence | Follow-up |
| --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |

Review crawl/indexing changes and factual corrections daily during the launch
window, weekly while the release is active, and in the monthly audit. A mention
without a link, an IndexNow `202`, or a search-console discovery is not proof of
indexing, ranking, conversion, or AI citation.
