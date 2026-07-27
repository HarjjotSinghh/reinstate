# Reinstate SEO, AEO, and AI Search Optimization Playbook

> Repository copy of the implementation playbook supplied on 2026-07-27.
> The checklist records the desired system; generated-site tests and current
> product documentation remain authoritative when the implementation changes.

**Website:** https://reinstate.dev  
**Repository reviewed:** `HarjjotSinghh/reinstate`  
**Website stack:** Astro, MDX, Tailwind CSS, Vercel  
**Primary audience:** Developers who use coding agents for work across multiple devices  
**Primary product outcome:** Sync coding-agent sessions across devices  
**Current Phase 1 scope:** Claude Code and Codex sessions, macOS and Windows, local encryption, user-owned S3 or Cloudflare R2 storage  
**Status of this document:** Implementation blueprint, content system, measurement plan, and coding-agent workflow  
**Last updated:** 2026-07-27

---

## 1. Executive summary

Reinstate should not try to rank as a vague "AI developer tool." That category is crowded, fuzzy, and mostly useless.

The product should own a much sharper concept:

> **Reinstate is an open-source tool that syncs encrypted coding-agent sessions across devices, so developers can continue the same Claude Code or Codex work on another computer.**

That sentence is the strategic spine of the entire website. Titles, headings, documentation, comparisons, structured data, GitHub copy, release notes, and third-party descriptions should reinforce it without turning every page into keyword soup.

The full discoverability strategy has three connected layers:

1. **SEO** makes Reinstate crawlable, indexable, understandable, fast, and relevant in conventional search.
2. **AEO** makes pages easy for search engines and assistants to extract into direct answers.
3. **AI Search Engine Optimization, or ASEO**, makes Reinstate a clear, verifiable, citation-worthy entity across ChatGPT Search, Google AI features, Perplexity, Bing Copilot, and other retrieval-driven systems.

These are not three unrelated marketing projects. They are one system:

```text
Technical clarity
    +
Useful, intent-matched content
    +
Verifiable product facts
    +
Strong internal and external entity signals
    =
Search visibility, answer inclusion, and AI citations
```

### The highest-priority work

Implement these before publishing a large blog:

1. Add a generated XML sitemap.
2. publish an explicit `robots.txt`.
3. Upgrade the shared SEO layout and social metadata.
4. Add truthful JSON-LD for the website, software, source code, docs, articles, and breadcrumbs.
5. Expand Astro content schemas so every page has deliberate metadata.
6. Establish a stable URL and page taxonomy.
7. Build integration, use-case, comparison, security, and troubleshooting pages.
8. Create answer-first content templates.
9. Verify Google Search Console and Bing Webmaster Tools.
10. Add a search and AI-referral measurement system.
11. Make GitHub, the website, docs, package listings, and social profiles use the same product definition.
12. Publish original technical evidence, not recycled "10 AI coding tips" sludge.

### The blunt reality

No schema tag, `llms.txt` file, prompt trick, or magic crawler incantation guarantees rankings or AI citations. Google explicitly states that its generative search features use normal Search fundamentals and do not require special AI markup or AI text files. ChatGPT Search requires discoverable public pages and access for `OAI-SearchBot`, but placement is not guaranteed.

The winning strategy is boring in the best way: technically clean pages, sharp positioning, primary-source documentation, original evidence, useful answers, consistent entities, and actual distribution. The boring stuff compounds. SEO jugaad does not.

---

# Part I: Strategic foundation

## 2. Definitions and boundaries

### 2.1 SEO

SEO is the process of making Reinstate discoverable and competitive in traditional web search.

For this project, SEO includes:

- crawlability and indexability
- URL architecture
- titles and descriptions
- canonicalization
- internal links
- structured data
- performance and Core Web Vitals
- documentation quality
- topic coverage
- backlinks and third-party mentions
- search performance measurement

### 2.2 AEO

AEO means Answer Engine Optimization.

Its job is to make a page easy to extract into:

- featured snippets
- direct answers
- "People also ask" style responses
- voice or assistant answers
- AI-generated summaries
- cited how-to instructions
- comparison answers
- troubleshooting answers

AEO is mostly a content-structure and factual-clarity discipline.

### 2.3 ASEO

In this playbook, ASEO means AI Search Engine Optimization.

Its job is to maximize the chance that retrieval-driven AI systems:

1. discover Reinstate,
2. understand what it is,
3. distinguish it from adjacent tools,
4. retrieve the correct page for a question,
5. trust the page enough to cite it,
6. present the product accurately,
7. send qualified visitors.

ASEO includes crawler accessibility, entity clarity, citation-ready content, first-party evidence, third-party corroboration, structured pages, freshness, and AI-referral measurement.

### 2.4 What ASEO is not

ASEO is not:

- stuffing pages with phrases such as "best tool according to ChatGPT"
- adding invisible text for language models
- creating hundreds of near-duplicate question pages
- publishing unsupported claims
- inventing ratings, users, benchmarks, or customer quotes
- treating `llms.txt` as a ranking switch
- blocking normal users behind JavaScript while serving bots special content
- generating thousands of pages because an agent can do it cheaply

That last one is how a useful website becomes a content landfill with a domain name.

---

## 3. Product positioning for search

### 3.1 Canonical one-sentence definition

Use this as the default factual definition:

> **Reinstate is an open-source tool that syncs encrypted coding-agent sessions across devices, allowing developers to continue the same Claude Code or Codex session on another computer.**

This definition should appear, with natural variations, on:

- the homepage
- docs overview
- GitHub README
- GitHub repository description
- release announcements
- social profiles
- launch posts
- package descriptions
- integration pages
- directory submissions
- interviews or guest posts
- structured data descriptions

### 3.2 Short positioning line

> **Continue the same coding-agent session on another device.**

### 3.3 Expanded positioning statement

> Reinstate lets developers move an active coding-agent workflow between computers without manually reconstructing the conversation, workspace context, and session state. Phase 1 supports Claude Code and Codex sessions across macOS and Windows, encrypts data locally, and stores encrypted artifacts in the developer's own S3-compatible bucket.

Only use the last sentence where every claim is currently true and documented.

### 3.4 Core problem statement

> Coding-agent sessions are commonly tied to one machine. Developers working across a work computer and a personal computer often lose continuity, duplicate context, or restart the session from scratch.

### 3.5 Category language

Use these category phrases deliberately:

- coding-agent session sync
- AI coding session sync
- cross-device coding-agent sessions
- portable coding-agent sessions
- coding-agent session continuity
- Claude Code session sync
- Codex session sync
- coding-agent session backup and restore
- encrypted developer session sync
- cross-platform AI coding workflow

Do not rotate between dozens of invented category names. Search engines and humans need repetition before a category becomes legible.

### 3.6 Primary differentiators

Current differentiators should be framed as verifiable facts:

- Open source
- Supports more than one coding agent
- Built for cross-device continuation
- Works across macOS and Windows
- Encrypts session data locally
- Supports user-owned S3 or Cloudflare R2 storage
- Does not require a Reinstate-hosted storage service
- Focuses on session portability rather than remote agent execution

Only add future differentiators after release and documentation.

### 3.7 Anti-positioning

Explicitly explain what Reinstate is not:

- not a cloud IDE
- not a remote desktop
- not a hosted coding agent
- not a live terminal mirror
- not credential synchronization
- not Git synchronization
- not a replacement for version control
- not a general backup tool

This helps search engines and AI systems avoid category confusion.

### 3.8 Audience priority

Primary:

- developers using Claude Code or Codex
- developers splitting work between a company device and personal device
- developers switching between desktop and laptop
- developers using macOS and Windows for the same project
- open-source developers who prefer self-controlled infrastructure

Secondary:

- engineering teams evaluating coding-agent continuity
- developer-tool maintainers
- security-conscious developers
- technical founders and solo developers
- agent platform researchers

---

## 4. Search intent model

Every page should satisfy one dominant intent.

| Intent | What the user is trying to do | Reinstate page type |
|---|---|---|
| Navigational | Find Reinstate | Homepage, GitHub, docs |
| Problem-aware | Solve lost coding-agent continuity | Use-case guide |
| Solution-aware | Find session sync software | Product or integration page |
| Agent-specific | Sync Claude Code or Codex | Integration page |
| Platform-specific | Move a session between macOS and Windows | Platform guide |
| How-to | Set up, upload, restore, continue | Documentation |
| Troubleshooting | Fix a concrete error | Troubleshooting article |
| Comparison | Compare Reinstate with manual copying or another tool | Comparison page |
| Security | Understand encryption, storage, threat model | Security page |
| Evaluation | Decide whether to install | Homepage, architecture, limitations |
| Contribution | Build an adapter or contribute code | Contributor docs |
| Freshness | See releases and compatibility | Changelog and release notes |

Do not make a page target five intents at once. A homepage can introduce multiple paths, but every deeper page needs one job.

---

# Part II: Current implementation audit

## 5. What the current site already does well

The current Astro implementation has a useful base:

- `site` is configured as `https://reinstate.dev`.
- The homepage is prerendered.
- The shared layout supports title, description, canonical URL, Open Graph title, Open Graph description, Open Graph URL, Open Graph image, and Twitter card metadata.
- The homepage has a focused title and description.
- Documentation pages are prerendered.
- Documentation content is stored in an Astro content collection.
- Docs have individual titles and descriptions.
- Important product claims are already concrete: supported agents, supported operating systems, local encryption, and user-owned storage.
- The current landing page communicates the problem visually and directly.

This means the site is not starting from zero. The foundation exists. It just needs to become a complete system.

## 6. Current gaps and risks

### Critical

1. No sitemap integration was found in the reviewed website package or Astro config.
2. No `robots.txt` was found in the repository search.
3. No JSON-LD implementation was found in the reviewed shared layout.
4. No explicit crawler policy for AI search bots was found.
5. No search measurement configuration was found in the reviewed files.
6. The content model is too small for a serious docs, guides, and blog system.

### High priority

1. The default shared description references sessions, MCP servers, skills, and settings across every coding agent. That is broader than the current Phase 1 product. A page that forgets to override the default could publish an inaccurate promise.
2. `twitter:image` is missing.
3. `og:image:alt` is missing.
4. `twitter:image:alt` is missing.
5. `og:site_name` is missing.
6. Page-specific robots controls are missing.
7. Docs lack visible breadcrumbs and breadcrumb structured data.
8. Docs metadata can be inferred from the first body line, which is convenient but not editorially reliable.
9. Docs content does not require an updated date, target query, author, or status.
10. No blog, guides, integration, use-case, or comparison collection is established.
11. No explicit draft or `noindex` workflow exists in the content schema.
12. No automated metadata or link validation was found.

### Medium priority

1. No RSS feed was found.
2. No optional `llms.txt` content index was found.
3. No changelog page was found in the inspected website paths.
4. No author or maintainer entity pages were found.
5. No dedicated security landing page outside docs was confirmed.
6. No compatibility matrix page was confirmed.
7. No browser-readable fact sheet was confirmed.
8. No image-specific metadata workflow was found.
9. No page-level Open Graph type distinction was found for articles.
10. No publication and modification timestamps were found in shared article metadata.

### Risk: positioning drift

The project is evolving quickly. Docs, homepage copy, GitHub README, and structured data can easily describe different products.

Create one source of truth, for example:

```text
website/src/data/product.ts
```

That file should contain:

- canonical product name
- canonical one-line definition
- current supported agents
- current supported operating systems
- storage providers
- license
- repository URL
- current release status
- stable social URLs
- default Open Graph image
- organization or maintainer details

Import that data into layouts and schema builders. Do not manually rewrite core facts across twenty components.

---

# Part III: Information architecture

## 7. Recommended URL structure

Use a shallow, predictable structure.

```text
/
├── docs/
│   ├── getting-started/
│   ├── installation/
│   ├── configuration/
│   ├── sync-a-session/
│   ├── restore-a-session/
│   ├── architecture/
│   ├── security-model/
│   ├── storage/
│   ├── adapters/
│   ├── troubleshooting/
│   ├── faq/
│   └── limitations/
├── integrations/
│   ├── claude-code/
│   └── codex/
├── use-cases/
│   ├── work-and-personal-computers/
│   ├── macos-and-windows/
│   ├── desktop-and-laptop/
│   └── encrypted-session-backup/
├── guides/
│   ├── sync-claude-code-sessions-across-devices/
│   ├── sync-codex-sessions-across-devices/
│   ├── move-a-coding-agent-session-from-mac-to-windows/
│   ├── use-s3-for-coding-agent-session-storage/
│   └── use-cloudflare-r2-for-coding-agent-session-storage/
├── compare/
│   ├── reinstate-vs-manual-session-copying/
│   ├── reinstate-vs-remote-desktop/
│   ├── reinstate-vs-git/
│   └── reinstate-vs-[real-competitor]/
├── blog/
│   ├── [post-slug]/
│   └── tags/[tag]/
├── security/
├── open-source/
├── changelog/
├── roadmap/
├── compatibility/
├── about/
├── privacy/
└── terms/
```

### URL rules

- lowercase only
- ASCII hyphens between words
- no dates in evergreen article URLs
- no `.html`
- no query parameters for indexable content
- one canonical URL per concept
- avoid both `/docs/foo` and `/guides/foo` covering the same intent
- use permanent redirects when changing a slug
- pick one trailing-slash policy and enforce it
- keep docs versioning out of the URL until multiple supported versions actually exist

### Docs versus guides

Use docs for product operation:

- installation
- command reference
- config
- adapters
- errors
- architecture

Use guides for user outcomes:

- move a Claude Code session from a MacBook to a Windows desktop
- continue Codex work on another computer
- store encrypted sessions in R2
- maintain coding continuity across work and personal devices

Docs answer "How does Reinstate work?"  
Guides answer "How do I accomplish my task?"

### Integration pages

An integration page must be more than a logo and three paragraphs.

Each should include:

- what is supported
- version or compatibility status
- what session data is handled
- what is not handled
- setup instructions
- sync workflow
- path remapping behavior
- security behavior
- common problems
- current limitations
- changelog links
- related docs
- visible last-updated date

### Comparison pages

Comparison pages should be factual and scoped.

Use a table with dimensions such as:

- workflow type
- supported agents
- cross-device behavior
- operating systems
- storage ownership
- encryption location
- account required
- self-hosting
- open-source license
- live remote control
- credential handling
- setup complexity
- current limitations

Never invent competitor facts. Add a source and "verified on" date for every comparison.

---

## 8. Page-to-query map

This is a qualitative starting map, not a claim about search volume.

| Page | Primary query family | Intent | Priority |
|---|---|---|---|
| Homepage | coding agent session sync | Solution | P0 |
| Claude Code integration | sync Claude Code sessions across devices | Agent-specific | P0 |
| Codex integration | sync Codex sessions across devices | Agent-specific | P0 |
| Work and personal computers | continue AI coding work on another computer | Problem-aware | P0 |
| macOS and Windows | move coding agent session from Mac to Windows | Platform-specific | P0 |
| Security | encrypted coding agent session sync | Security | P0 |
| Getting started | install Reinstate | How-to | P0 |
| Sync a session | how to sync a coding agent session | How-to | P0 |
| Restore a session | restore Claude Code or Codex session | How-to | P0 |
| Troubleshooting | Reinstate errors | Support | P0 |
| Compatibility | Reinstate supported agents and platforms | Evaluation | P0 |
| Architecture | how Reinstate works | Evaluation | P1 |
| Storage guide | use S3 or R2 for session sync | How-to | P1 |
| Open source | open-source coding agent session sync | Evaluation | P1 |
| Comparison with manual copy | copy coding agent sessions between computers | Comparison | P1 |
| FAQ | coding agent session sync questions | Answer | P1 |
| Changelog | Reinstate releases and compatibility | Freshness | P1 |

### Keyword cluster: core category

- coding agent session sync
- AI coding session sync
- sync AI coding sessions across devices
- cross-device coding agent
- coding agent session portability
- continue coding agent session on another computer
- coding agent session backup
- coding agent workspace sync

### Keyword cluster: Claude Code

- sync Claude Code sessions
- move Claude Code session to another computer
- continue Claude Code session on another device
- Claude Code session backup
- Claude Code session restore
- Claude Code macOS Windows sync
- Claude Code cross-device workflow
- where are Claude Code sessions stored

### Keyword cluster: Codex

- sync Codex sessions
- continue Codex session on another computer
- Codex CLI session backup
- Codex session restore
- Codex macOS Windows sync
- Codex cross-device workflow
- where are Codex sessions stored

### Keyword cluster: security and ownership

- encrypted coding agent session sync
- self-hosted coding agent session storage
- S3 coding agent session backup
- Cloudflare R2 session sync
- private Claude Code session backup
- open-source session sync
- client-side encrypted developer tool

### Keyword cluster: problem language

- coding session stuck on another computer
- continue AI coding work from laptop
- move coding agent context between devices
- work computer personal computer coding workflow
- switch computers without losing Claude Code context
- share Codex session between Mac and Windows
- coding agent loses context across devices

### Keyword cluster: education

- what is a coding agent session
- where coding agents store sessions
- how coding agent session state works
- coding agent session portability
- coding agent session versus project files
- why Git does not sync coding agent conversations

---

# Part IV: Technical SEO implementation for Astro

## 9. Install and configure sitemap generation

Install the official Astro integration:

```bash
pnpm --dir website add @astrojs/sitemap
```

Update `website/astro.config.mjs`:

```js
// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import mdx from '@astrojs/mdx';
import vercel from '@astrojs/vercel';
import sitemap from '@astrojs/sitemap';

const ignoreGeneratedDevPath = (filePath) =>
  /\/(?:\.vercel|dist)(?:\/|$)/.test(filePath.replaceAll('\\', '/'));

export default defineConfig({
  site: 'https://reinstate.dev',
  output: 'server',
  adapter: vercel(),
  trailingSlash: 'never',
  integrations: [
    mdx(),
    sitemap({
      filter: (page) =>
        !page.includes('/api/') &&
        !page.includes('/preview/') &&
        !page.includes('/drafts/'),
    }),
  ],
  vite: {
    plugins: [tailwindcss()],
    server: {
      watch: {
        ignored: ignoreGeneratedDevPath,
        usePolling: true,
        interval: 100,
      },
    },
  },
  markdown: {
    shikiConfig: {
      theme: 'github-dark-default',
      wrap: true,
    },
  },
});
```

### Validate

After a production build:

```bash
pnpm --dir website build
find website/dist -maxdepth 2 -iname '*sitemap*' -print
```

Check:

- the sitemap returns `200`
- only canonical indexable URLs are present
- no API or preview routes are present
- dynamic docs routes are present
- redirects are absent
- each URL resolves without a redirect chain

Submit the sitemap to:

- Google Search Console
- Bing Webmaster Tools

Do not manually maintain a sitemap once the site grows. Humans forget. Build systems are less lazy.

---

## 10. Publish `robots.txt`

Use a static file at:

```text
website/public/robots.txt
```

Recommended baseline:

```text
User-agent: *
Allow: /
Disallow: /api/
Disallow: /preview/
Disallow: /drafts/

User-agent: OAI-SearchBot
Allow: /

User-agent: PerplexityBot
Allow: /

Sitemap: https://reinstate.dev/sitemap-index.xml
```

### GPTBot decision

`OAI-SearchBot` affects discoverability in ChatGPT Search. `GPTBot` is a separate control associated with model training. Decide separately.

Allow training:

```text
User-agent: GPTBot
Allow: /
```

Disallow training while retaining ChatGPT Search discovery:

```text
User-agent: GPTBot
Disallow: /
```

Do not assume that allowing training improves ChatGPT Search rankings. OpenAI documents these as separate controls.

### Important crawler notes

- A `robots.txt` rule manages crawling, not reliable deindexing.
- Use `noindex` for public pages that should not appear in search.
- A crawler must access the page to read its `noindex`.
- Check the CDN and WAF. A perfect robots file is useless if Vercel, Cloudflare, or another layer returns `403`.
- Verify bot IP ranges before broad allowlisting.
- Do not copy random "AI bot mega lists" from old blog posts. User-agent names and policies change.
- No stable official Anthropic crawler policy was used in this playbook. Verify Anthropic's current official documentation before adding an Anthropic-specific rule.

### Test

```bash
curl -i https://reinstate.dev/robots.txt
curl -A "OAI-SearchBot" -I https://reinstate.dev/
curl -A "PerplexityBot" -I https://reinstate.dev/
```

Expect a successful response without a JavaScript challenge.

---

## 11. Build a real SEO component

Create:

```text
website/src/components/SeoHead.astro
```

Example:

```astro
---
interface Props {
  title: string;
  description: string;
  canonical?: URL | string;
  image?: URL | string;
  imageAlt?: string;
  type?: 'website' | 'article';
  noindex?: boolean;
  nofollow?: boolean;
  publishedTime?: Date;
  modifiedTime?: Date;
  section?: string;
  tags?: string[];
}

const {
  title,
  description,
  canonical = new URL(Astro.url.pathname, Astro.site),
  image = new URL('/brand/og.png', Astro.site),
  imageAlt = 'Reinstate - encrypted coding-agent session sync',
  type = 'website',
  noindex = false,
  nofollow = false,
  publishedTime,
  modifiedTime,
  section,
  tags = [],
} = Astro.props;

const canonicalUrl = new URL(canonical, Astro.site).toString();
const imageUrl = new URL(image, Astro.site).toString();

const robots = [
  noindex ? 'noindex' : 'index',
  nofollow ? 'nofollow' : 'follow',
  'max-image-preview:large',
  'max-snippet:-1',
  'max-video-preview:-1',
].join(',');
---

<title>{title}</title>
<meta name="description" content={description} />
<meta name="robots" content={robots} />
<link rel="canonical" href={canonicalUrl} />

<meta property="og:site_name" content="Reinstate" />
<meta property="og:locale" content="en_US" />
<meta property="og:type" content={type} />
<meta property="og:title" content={title} />
<meta property="og:description" content={description} />
<meta property="og:url" content={canonicalUrl} />
<meta property="og:image" content={imageUrl} />
<meta property="og:image:alt" content={imageAlt} />

<meta name="twitter:card" content="summary_large_image" />
<meta name="twitter:title" content={title} />
<meta name="twitter:description" content={description} />
<meta name="twitter:image" content={imageUrl} />
<meta name="twitter:image:alt" content={imageAlt} />

{
  type === 'article' && publishedTime && (
    <meta
      property="article:published_time"
      content={publishedTime.toISOString()}
    />
  )
}
{
  type === 'article' && modifiedTime && (
    <meta
      property="article:modified_time"
      content={modifiedTime.toISOString()}
    />
  )
}
{type === 'article' && section && (
  <meta property="article:section" content={section} />
)}
{type === 'article' && tags.map((tag) => (
  <meta property="article:tag" content={tag} />
))}
```

### Rules

- Every indexable page gets a unique title.
- Every indexable page gets an intentional description.
- Do not use the `meta keywords` tag.
- Do not include a canonical override unless the page is genuinely duplicated elsewhere.
- Do not canonicalize unrelated pages to the homepage.
- Open Graph images should be at least 1200 x 630 for predictable sharing.
- Keep the actual image dimensions and alt text in metadata.
- Use `article` only for articles, not every documentation page by default.
- Add `noindex` to previews, drafts, temporary launch pages, internal search pages, and low-value generated pages.

---

## 12. Refactor `BaseLayout.astro`

The shared layout should:

1. import product facts from one source
2. require or strongly validate page title and description
3. render `SeoHead`
4. accept structured-data objects
5. permit article metadata
6. expose `noindex`
7. preserve semantic page structure

Example shape:

```astro
---
import '../styles/global.css';
import Header from '../components/Header.astro';
import Footer from '../components/Footer.astro';
import SeoHead from '../components/SeoHead.astro';
import JsonLd from '../components/JsonLd.astro';
import { product } from '../data/product';

interface Props {
  title?: string;
  description?: string;
  canonical?: string;
  ogImage?: string;
  ogImageAlt?: string;
  type?: 'website' | 'article';
  noindex?: boolean;
  publishedTime?: Date;
  modifiedTime?: Date;
  section?: string;
  tags?: string[];
  jsonLd?: Record<string, unknown> | Record<string, unknown>[];
  bare?: boolean;
}

const {
  title = product.defaultTitle,
  description = product.defaultDescription,
  canonical,
  ogImage = product.defaultOgImage,
  ogImageAlt = product.defaultOgImageAlt,
  type = 'website',
  noindex = false,
  publishedTime,
  modifiedTime,
  section,
  tags = [],
  jsonLd = [],
  bare = false,
} = Astro.props;
---

<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="generator" content={Astro.generator} />

    <SeoHead
      title={title}
      description={description}
      canonical={canonical}
      image={ogImage}
      imageAlt={ogImageAlt}
      type={type}
      noindex={noindex}
      publishedTime={publishedTime}
      modifiedTime={modifiedTime}
      section={section}
      tags={tags}
    />

    <JsonLd data={jsonLd} />

    <!-- theme and favicon tags -->
  </head>
  <body class="min-h-dvh flex flex-col">
    {!bare && <Header />}
    <main id="main-content" class="flex-1">
      <slot />
    </main>
    {!bare && <Footer />}
  </body>
</html>
```

### Fix the default product description

Replace the current broad default with something that cannot accidentally overpromise:

```ts
defaultDescription:
  'Reinstate is an open-source tool for syncing encrypted Claude Code and Codex sessions across macOS and Windows using your own S3-compatible storage.',
```

When Phase 2 adds features, update the source of truth after the feature is released and documented, not the night someone adds it to a roadmap.

---

## 13. Add JSON-LD safely

Create:

```text
website/src/components/JsonLd.astro
```

```astro
---
interface Props {
  data: Record<string, unknown> | Record<string, unknown>[];
}

const { data } = Astro.props;
const graph = Array.isArray(data) ? data : [data];
const payload = {
  '@context': 'https://schema.org',
  '@graph': graph,
};

const safeJson = JSON.stringify(payload).replace(/</g, '\\u003c');
---

<script type="application/ld+json" set:html={safeJson} />
```

### Security note

Do not inject untrusted user input directly into JSON-LD. Serialize it and escape `<` to reduce script-breakout risk.

### Structured-data doctrine

- Mark up visible, truthful content.
- Do not add fields merely because a validator accepts them.
- Do not invent ratings or reviews.
- Do not claim every supported platform until it is actually supported.
- Do not use FAQ schema for hidden or nonexistent answers.
- Structured data helps understanding. It does not force a rich result.
- Validate with both Google's Rich Results Test and Schema.org Validator.
- Revalidate after template changes.

---

## 14. Homepage structured data

Recommended graph:

```ts
const homepageSchema = [
  {
    '@type': 'Organization',
    '@id': 'https://reinstate.dev/#organization',
    name: 'Reinstate',
    url: 'https://reinstate.dev/',
    logo: {
      '@type': 'ImageObject',
      url: 'https://reinstate.dev/brand/logo.png',
    },
    sameAs: [
      'https://github.com/HarjjotSinghh/reinstate',
      // Add only official profiles.
    ],
  },
  {
    '@type': 'WebSite',
    '@id': 'https://reinstate.dev/#website',
    url: 'https://reinstate.dev/',
    name: 'Reinstate',
    description:
      'Open-source encrypted coding-agent session sync across devices.',
    publisher: {
      '@id': 'https://reinstate.dev/#organization',
    },
    inLanguage: 'en',
  },
  {
    '@type': 'SoftwareApplication',
    '@id': 'https://reinstate.dev/#software',
    name: 'Reinstate',
    url: 'https://reinstate.dev/',
    description:
      'An open-source tool that syncs encrypted Claude Code and Codex sessions across macOS and Windows.',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: ['macOS', 'Windows'],
    isAccessibleForFree: true,
    offers: {
      '@type': 'Offer',
      price: '0',
      priceCurrency: 'USD',
    },
    downloadUrl: 'https://github.com/HarjjotSinghh/reinstate',
    softwareHelp: 'https://reinstate.dev/docs/',
    author: {
      '@id': 'https://reinstate.dev/#organization',
    },
  },
  {
    '@type': 'SoftwareSourceCode',
    '@id': 'https://reinstate.dev/#source',
    name: 'Reinstate source code',
    codeRepository: 'https://github.com/HarjjotSinghh/reinstate',
    programmingLanguage: 'Go',
    runtimePlatform: ['macOS', 'Windows'],
    license: 'https://www.apache.org/licenses/LICENSE-2.0',
    targetProduct: {
      '@id': 'https://reinstate.dev/#software',
    },
  },
];
```

### Organization versus Person

Use `Organization` only if Reinstate is presented as a project or organization entity. If it is explicitly a personal open-source project, a `Person` author plus `Organization` project identity may be more accurate.

Do not pretend there is a company if there is not one.

### SearchAction

Do not add `SearchAction` until the site has a real public search endpoint with a working URL template.

---

## 15. Documentation structured data

Use:

- `TechArticle` for substantial conceptual docs
- `HowTo` only when a page is genuinely a step-by-step task and every step is visible
- `BreadcrumbList` for hierarchy
- `FAQPage` only for a visible FAQ page

Example for a doc:

```ts
const docSchema = [
  {
    '@type': 'TechArticle',
    '@id': `${canonical}#article`,
    headline: title,
    description,
    url: canonical,
    dateModified: updatedAt.toISOString(),
    inLanguage: 'en',
    isPartOf: {
      '@id': 'https://reinstate.dev/#website',
    },
    about: {
      '@id': 'https://reinstate.dev/#software',
    },
    author: {
      '@id': 'https://reinstate.dev/#organization',
    },
    mainEntityOfPage: canonical,
  },
  {
    '@type': 'BreadcrumbList',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Home',
        item: 'https://reinstate.dev/',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Docs',
        item: 'https://reinstate.dev/docs/',
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: title,
        item: canonical,
      },
    ],
  },
];
```

Also render visible breadcrumbs:

```astro
<nav aria-label="Breadcrumb">
  <ol>
    <li><a href="/">Home</a></li>
    <li><a href="/docs">Docs</a></li>
    <li aria-current="page">{title}</li>
  </ol>
</nav>
```

Structured data should mirror the visible hierarchy.

---

## 16. Blog structured data

Use `BlogPosting` or `Article`.

Required editorial fields:

- headline
- description
- canonical URL
- author
- date published
- date modified
- image
- image alt
- tags
- section
- publisher
- visible byline
- visible updated date

Never silently rewrite an old article and keep the original date as if nothing changed. Show both publication and update dates.

---

## 17. Expand Astro content collections

Current docs metadata is too permissive.

Recommended shared schema:

```ts
import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

const seoFields = z.object({
  title: z.string().min(10).max(70),
  description: z.string().min(70).max(180),
  canonical: z.string().url().optional(),
  noindex: z.boolean().default(false),
  ogImage: z.string().optional(),
  ogImageAlt: z.string().optional(),
  targetQuery: z.string().optional(),
  searchIntent: z
    .enum([
      'navigational',
      'problem',
      'solution',
      'how-to',
      'troubleshooting',
      'comparison',
      'evaluation',
    ])
    .optional(),
});

const docs = defineCollection({
  loader: glob({
    pattern: '**/*.{md,mdx}',
    base: './src/content/docs',
  }),
  schema: seoFields.extend({
    order: z.number().optional(),
    updatedAt: z.coerce.date(),
    version: z.string().optional(),
    tags: z.array(z.string()).default([]),
    draft: z.boolean().default(false),
  }),
});

const guides = defineCollection({
  loader: glob({
    pattern: '**/*.{md,mdx}',
    base: './src/content/guides',
  }),
  schema: seoFields.extend({
    publishedAt: z.coerce.date(),
    updatedAt: z.coerce.date(),
    author: z.string(),
    tags: z.array(z.string()).default([]),
    relatedDocs: z.array(z.string()).default([]),
    draft: z.boolean().default(false),
  }),
});

const blog = defineCollection({
  loader: glob({
    pattern: '**/*.{md,mdx}',
    base: './src/content/blog',
  }),
  schema: seoFields.extend({
    publishedAt: z.coerce.date(),
    updatedAt: z.coerce.date(),
    author: z.string(),
    section: z.string(),
    tags: z.array(z.string()).default([]),
    draft: z.boolean().default(false),
  }),
});

export const collections = { docs, guides, blog };
```

### Editorial rule

Do not infer important SEO descriptions from the first body paragraph in production. Require them in frontmatter. Inference is fine as a migration fallback, not as a permanent editorial system.

### Example frontmatter

```yaml
---
title: "How to sync Claude Code sessions across devices"
description: "Use Reinstate to encrypt, upload, and restore a Claude Code session between macOS and Windows using your own S3 or R2 bucket."
publishedAt: 2026-08-10
updatedAt: 2026-08-10
author: "Reinstate maintainers"
section: "Claude Code"
tags:
  - claude-code
  - session-sync
  - cross-device
targetQuery: "sync Claude Code sessions across devices"
searchIntent: "how-to"
ogImage: "/og/guides/sync-claude-code.png"
ogImageAlt: "A Claude Code session moving securely between two computers"
draft: false
noindex: false
---
```

---

## 18. Canonicalization and redirects

### Policy

- HTTPS only
- `reinstate.dev` as the only production host
- no `www` unless intentionally selected
- one trailing-slash format
- lowercase paths
- no duplicate pages at old routes
- query parameters excluded from canonical URLs unless they materially change content

### Vercel redirects

Create or update `vercel.json`:

```json
{
  "redirects": [
    {
      "source": "/docs/getting_started",
      "destination": "/docs/getting-started",
      "permanent": true
    }
  ]
}
```

Use real old paths only. Do not add speculative redirects.

### Canonical tests

For each page:

```bash
curl -s https://reinstate.dev/docs/getting-started \
  | grep -o '<link rel="canonical"[^>]*>'
```

Assert:

- exactly one canonical tag
- absolute HTTPS URL
- canonical returns `200`
- no redirect
- no fragment
- correct trailing slash
- self-referential for unique pages

---

## 19. Status codes and error pages

Requirements:

- valid pages return `200`
- permanent moves return `301` or `308`
- missing pages return `404`, not `200`
- deleted content with no replacement may return `410`
- server failures return `5xx`
- preview or auth redirects should not trap crawlers
- soft 404s should be eliminated

Create a useful custom `404.astro` with:

- plain explanation
- search or docs navigation
- links to getting started, integrations, and troubleshooting
- no fake content
- `noindex`

---

## 20. Internal linking system

Every indexable page should have contextual links, not only header and footer links.

### Homepage links to

- getting started
- Claude Code integration
- Codex integration
- security
- architecture
- compatibility
- GitHub
- changelog

### Integration pages link to

- installation
- session sync steps
- agent-specific troubleshooting
- security
- platform use cases
- changelog entries
- relevant comparison pages

### Blog and guides link to

- the product feature that solves the discussed problem
- relevant docs
- related guides
- canonical integration pages
- one logical next action

### Anchor text

Good:

- "sync a Claude Code session across devices"
- "review Reinstate's local encryption model"
- "configure Cloudflare R2 storage"

Weak:

- "click here"
- "learn more"
- "read this"

Do not use the exact same keyword-heavy anchor fifty times. Natural language is fine.

### Orphan-page check

Every indexable page should be reachable through crawlable HTML links within three logical steps from the homepage or a hub.

---

## 21. Page titles

Recommended patterns:

Homepage:

```text
Reinstate: Sync Coding-Agent Sessions Across Devices
```

Integration:

```text
Sync Claude Code Sessions Across Devices | Reinstate
```

Guide:

```text
How to Move a Codex Session to Another Computer | Reinstate
```

Docs:

```text
Configure Cloudflare R2 Storage | Reinstate Docs
```

Comparison:

```text
Reinstate vs Manual Session Copying
```

### Title rules

- describe the page, not the company slogan
- primary concept near the beginning
- unique across the site
- avoid repeating "best," "ultimate," and "complete"
- avoid putting every supported agent in every title
- do not chase an arbitrary character count at the expense of clarity
- inspect actual Search Console truncation and rewrites later

---

## 22. Meta descriptions

Descriptions should explain:

1. what the page helps the user do,
2. what makes Reinstate relevant,
3. the most important constraint.

Example:

> Sync an encrypted Claude Code session between macOS and Windows with Reinstate, using your own Amazon S3 or Cloudflare R2 bucket.

Avoid:

> Reinstate is the ultimate revolutionary AI-powered platform for seamless productivity.

That sentence says nothing and sounds like it was generated during a caffeine shortage.

---

## 23. Heading structure

Every page:

- one visible H1
- H2s for major sections
- H3s for subtopics
- no heading levels selected for visual size
- headings describe the answer below them
- question headings where users ask questions

Good:

```markdown
# How to sync a Claude Code session across devices

## What Reinstate copies

## Before you begin

## Upload the session from your first computer

## Restore the session on your second computer

## What to do when paths differ

## Common errors
```

---

## 24. Images and media

### Required

- descriptive filenames
- `width` and `height`
- meaningful alt text
- responsive `srcset`
- modern formats where practical
- lazy loading below the fold
- eager loading only for the LCP image
- no critical explanation embedded only in screenshots
- captions for diagrams
- text transcript for videos
- compressed Open Graph assets
- dark and light mode contrast checks

Astro example:

```astro
---
import { Image } from 'astro:assets';
import sessionFlow from '../assets/session-flow.png';
---

<figure>
  <Image
    src={sessionFlow}
    alt="Encrypted session data moving from a MacBook to an S3 bucket and then to a Windows desktop"
    widths={[640, 960, 1280]}
    sizes="(max-width: 768px) 100vw, 960px"
    loading="lazy"
  />
  <figcaption>
    Reinstate encrypts session data locally before upload and decrypts it on the destination device.
  </figcaption>
</figure>
```

### Diagram SEO

Architecture diagrams should have a nearby textual explanation including:

- components
- sequence
- trust boundary
- encryption point
- storage point
- restoration point
- failure behavior

AI systems cannot reliably infer every detail from a decorative SVG.

---

## 25. Core Web Vitals and performance

Targets at the 75th percentile:

- LCP: 2.5 seconds or less
- INP: 200 milliseconds or less
- CLS: 0.1 or less

### Astro-specific actions

- keep most content server-rendered or static
- minimize hydrated islands
- remove unused font families
- subset fonts
- preload only the font used above the fold
- use `font-display: swap`
- self-host fonts, which the project already does through Fontsource
- optimize the hero's largest visual
- avoid loading Motion code for elements that do not need runtime animation
- reserve dimensions for visual components
- avoid client-side rendering for documentation text
- use responsive images
- defer nonessential scripts
- keep analytics lightweight
- cache immutable assets
- use Brotli or gzip
- avoid enormous syntax-highlighting bundles
- audit third-party embeds

### Performance budget

Suggested initial budget:

| Resource | Budget |
|---|---:|
| Initial compressed HTML | 75 KB |
| Initial compressed CSS | 100 KB |
| Initial JavaScript on docs pages | 100 KB |
| Initial JavaScript on homepage | 200 KB |
| LCP image compressed | 200 KB |
| Total initial transfer | 1 MB |
| Third-party scripts | 2 maximum before consent |

Budgets are engineering guardrails, not ranking formulas.

### Test commands

```bash
pnpm --dir website build
pnpm --dir website preview
npx lighthouse https://reinstate.dev \
  --only-categories=performance,seo,accessibility,best-practices \
  --output=json \
  --output-path=./lighthouse.json
```

Also use field data from Search Console and the Chrome UX Report when traffic becomes sufficient. Lab tests are a smoke detector, not the whole building inspection.

---

## 26. Accessibility as discoverability infrastructure

Implement:

- skip link
- correct landmarks
- accessible navigation labels
- keyboard operation
- visible focus
- sufficient contrast
- text alternatives
- labelled form controls
- descriptive error messages
- reduced-motion support
- semantic tables
- captions and transcripts
- no content hidden behind hover only
- no critical text rendered through canvas
- stable zoom up to 200 percent

Accessibility is primarily for users. It also makes page structure less ambiguous for machines.

---

## 27. JavaScript and rendering

Astro is a strong choice because important content can ship as HTML.

Rules:

- product definitions, headings, steps, FAQs, links, and comparison data must exist in initial HTML
- do not require a button click to render the only copy search engines need
- do not inject titles or canonicals client-side
- avoid content that appears only after local-storage checks
- use real `<a href>` links for navigation
- use buttons for actions, links for destinations
- ensure error boundaries do not replace the whole page with empty markup
- test with JavaScript disabled
- inspect rendered HTML in Search Console

---

## 28. RSS and release feeds

Create RSS for blog and changelog.

Install:

```bash
pnpm --dir website add @astrojs/rss
```

Example:

```ts
// src/pages/rss.xml.ts
import rss from '@astrojs/rss';
import { getCollection } from 'astro:content';

export async function GET(context) {
  const posts = await getCollection(
    'blog',
    ({ data }) => !data.draft && !data.noindex
  );

  return rss({
    title: 'Reinstate Blog',
    description:
      'Guides and engineering notes about coding-agent session portability.',
    site: context.site,
    items: posts.map((post) => ({
      title: post.data.title,
      description: post.data.description,
      pubDate: post.data.publishedAt,
      link: `/blog/${post.id}/`,
    })),
  });
}
```

Link it in `<head>`:

```html
<link
  rel="alternate"
  type="application/rss+xml"
  title="Reinstate Blog"
  href="https://reinstate.dev/rss.xml"
/>
```

Feeds help subscribers, aggregators, and discovery systems notice fresh content.

---

## 29. Optional `llms.txt`

Google states that special AI text files are not required for its AI search features. Treat `llms.txt` as an optional navigation aid, not a ranking factor.

A minimal version:

```text
# Reinstate

> Reinstate is an open-source tool that syncs encrypted coding-agent sessions across devices. Phase 1 supports Claude Code and Codex across macOS and Windows with user-owned S3-compatible storage.

## Product

- [Homepage](https://reinstate.dev/): Product overview and current capabilities
- [Compatibility](https://reinstate.dev/compatibility/): Supported agents, operating systems, and storage backends
- [Security](https://reinstate.dev/security/): Encryption model, trust boundaries, and limitations
- [Changelog](https://reinstate.dev/changelog/): Releases and compatibility changes

## Documentation

- [Getting started](https://reinstate.dev/docs/getting-started/)
- [Architecture](https://reinstate.dev/docs/architecture/)
- [Adapters](https://reinstate.dev/docs/adapters/)
- [Troubleshooting](https://reinstate.dev/docs/troubleshooting/)
- [FAQ](https://reinstate.dev/docs/faq/)

## Integrations

- [Claude Code](https://reinstate.dev/integrations/claude-code/)
- [Codex](https://reinstate.dev/integrations/codex/)

## Source

- [GitHub repository](https://github.com/HarjjotSinghh/reinstate)
- License: Apache-2.0
```

Maintenance rules:

- only include canonical pages
- update it when routes change
- do not duplicate the whole website
- do not put private or unreleased claims in it
- do not expect indexing benefits by itself

---

# Part V: AEO implementation

## 30. Answer-first writing model

Every important guide should answer the main question immediately after the H1.

Example:

```markdown
# How do you sync a Claude Code session across devices?

Reinstate encrypts the Claude Code session on the source computer, uploads the encrypted archive to your own S3 or Cloudflare R2 bucket, and restores it on the destination computer. Phase 1 supports transfers between macOS and Windows.
```

Then expand.

### Recommended answer block length

Use approximately 40 to 80 words for the first direct answer when possible. Do not mutilate a complex answer to hit a word count. Clarity wins.

### Answer block component

```astro
---
interface Props {
  question: string;
}

const { question } = Astro.props;
---

<section class="answer" aria-labelledby="answer-question">
  <h2 id="answer-question">{question}</h2>
  <div class="answer-body">
    <slot />
  </div>
</section>
```

Do not add fake schema merely because the component is called an answer box.

---

## 31. AEO page template

Use this structure for high-intent guides:

```markdown
# Exact user question or task

Direct answer in one concise paragraph.

## Key points

- Supported agents and platforms
- Required storage
- Security behavior
- Important limitation

## Before you begin

Prerequisites.

## Steps

### 1. Install Reinstate

Commands and expected output.

### 2. Configure storage

Commands and expected output.

### 3. Upload the session

Commands and expected output.

### 4. Restore on the second device

Commands and expected output.

## How it works

Short technical explanation.

## Common errors

Question-shaped headings and direct fixes.

## Security considerations

Visible limitations and threat model.

## Frequently asked questions

Specific questions with direct answers.

## Related documentation

Contextual links.
```

---

## 32. Question inventory

Create pages or sections that directly answer these questions.

### Product

- What is Reinstate?
- What problem does Reinstate solve?
- Is Reinstate open source?
- Is Reinstate free?
- Does Reinstate require an account?
- Does Reinstate run its own storage service?
- Which coding agents does Reinstate support?
- Which operating systems does Reinstate support?

### Workflow

- How do I sync a Claude Code session across devices?
- How do I move a Codex session to another computer?
- How do I continue an AI coding session on a laptop?
- Can I move a session from macOS to Windows?
- What happens when project paths differ?
- Can I restore a session without copying the whole repository?
- Does Reinstate sync Git changes?
- Does Reinstate sync MCP servers or credentials?

### Security

- Where is session data encrypted?
- Who can read a Reinstate archive?
- Where is encrypted data stored?
- Does Reinstate upload API keys?
- What happens if an S3 bucket is compromised?
- What metadata remains visible?
- How are encryption keys managed?
- What is outside the threat model?

### Technical

- Where does Claude Code store sessions?
- Where does Codex store sessions?
- What does a Reinstate adapter do?
- How does path remapping work?
- What happens after an agent changes its session format?
- How is compatibility tested?
- What files are included or excluded?

### Evaluation

- Is Reinstate a remote desktop?
- Is Reinstate a cloud IDE?
- Is Reinstate a backup tool?
- How is Reinstate different from Git?
- How is Reinstate different from manually copying session files?
- When should I not use Reinstate?

These are excellent H2s, FAQ entries, guide topics, and test queries for AI systems.

---

## 33. Definition blocks

Maintain concise definitions for recurring entities.

Example:

```markdown
## What is a coding-agent session?

A coding-agent session is the saved conversation and tool state that lets an AI coding agent resume prior work. It is separate from the project's Git history and may include messages, identifiers, execution context, and agent-specific metadata.
```

Each definition should:

- define one thing
- use plain language
- distinguish it from nearby concepts
- avoid circular wording
- link to deeper technical documentation

---

## 34. How-to content

For every command:

- state the purpose
- show the command
- show expected output
- explain parameters
- state platform differences
- state failure modes
- state how to undo it

Example:

````markdown
### Upload the current session

Run:

```bash
rein push
```

Reinstate locates the configured agent session, encrypts the archive locally, and uploads the encrypted result to the configured storage bucket. A successful run prints the session identifier and remote object location.

This command does not push Git commits or upload your repository unless the documented adapter explicitly includes a file.
````

Never let a command float without context.

---

## 35. Comparison content

Use explicit dimensions.

Bad:

> Reinstate is much better than other tools.

Good:

| Capability | Reinstate | Manual copy |
|---|---|---|
| Cross-OS path handling | Documented adapter behavior | Manual editing |
| Local encryption | Yes, before upload | Depends on user |
| Storage ownership | User-owned S3 or R2 | User-selected |
| Repeatability | Command-based workflow | Manual process |
| Agent support | Claude Code and Codex in Phase 1 | Any format the user understands |
| Maintenance | Adapter updates required | User handles format changes |

Add limitations to both columns. Honest comparison is more citation-worthy than chest-thumping.

---

## 36. Troubleshooting content

Each error should have:

1. exact error text
2. probable cause
3. fastest safe fix
4. diagnostic command
5. platform notes
6. when to file an issue
7. data to include in the issue
8. sensitive data to redact

Example heading:

```markdown
## Why does Reinstate report "session not found"?
```

Opening answer:

> Reinstate reports "session not found" when the selected adapter cannot locate a compatible local session for the current project or session identifier. Confirm the agent, project path, and adapter configuration before retrying.

This format is excellent for both humans and answer engines.

---

## 37. FAQ strategy

Build one canonical FAQ page, but also answer context-specific questions on relevant pages.

Rules:

- visible questions and answers
- one question per heading
- no marketing filler
- no duplicated answer with conflicting details
- link to supporting docs
- include updated date
- remove obsolete questions
- schema must match visible content
- do not expect FAQ rich results for every site

---

# Part VI: AI Search Engine Optimization

## 38. The ASEO model

An AI system is more likely to cite Reinstate when it can retrieve a page that is:

- relevant to the exact question
- publicly accessible
- available as text
- internally well linked
- technically indexable
- explicit about entities and relationships
- current
- factually dense
- backed by first-party evidence
- corroborated elsewhere
- easy to quote without losing context

Think in terms of citation units, not keyword density.

A citation unit is a paragraph, table, definition, procedure, benchmark, or fact block that remains accurate when extracted.

---

## 39. Entity consistency

Use the exact product name "Reinstate."

Maintain one factual profile:

```yaml
name: Reinstate
category: coding-agent session sync
description: Open-source tool that syncs encrypted coding-agent sessions across devices
supported_agents:
  - Claude Code
  - Codex
supported_operating_systems:
  - macOS
  - Windows
storage:
  - Amazon S3
  - Cloudflare R2
license: Apache-2.0
repository: https://github.com/HarjjotSinghh/reinstate
website: https://reinstate.dev
```

Use the same facts on:

- website
- README
- GitHub About
- GitHub topics
- releases
- package metadata
- docs
- social profiles
- launch directories
- interviews

AI systems often reconcile multiple sources. Conflicting descriptions weaken confidence.

---

## 40. Create an authoritative fact page

Recommended route:

```text
/about/reinstate/
```

Include:

- one-sentence definition
- current release status
- current supported agents
- current supported systems
- storage model
- encryption summary
- license
- repository
- maintainers
- initial release date
- latest stable release
- non-goals
- limitations
- official links
- visible "last verified" date

This page is a clean citation target.

Do not hide these facts only in an animated landing page.

---

## 41. Publish primary-source evidence

High-value original material:

1. Agent session format research
2. Cross-platform path mapping design
3. Adapter compatibility matrix
4. Encryption architecture
5. Threat model
6. Reproducible migration tests
7. Session-size distribution from opt-in or synthetic testing
8. Restore success rates from automated fixtures
9. Agent-version compatibility history
10. Failure taxonomy
11. Open benchmark methodology
12. Release regression reports

Primary-source evidence is the moat. A generic article can be summarized away. Original data must be attributed.

### Example benchmark page

```text
/research/cross-device-session-restore-benchmark/
```

Include:

- question
- environment
- hardware
- operating systems
- agent versions
- Reinstate version
- dataset or fixtures
- exact commands
- success criteria
- results
- failures
- limitations
- raw data download
- date
- commit SHA

Never publish benchmark numbers without a reproducible method.

---

## 42. Citation-ready claims

Good:

> Reinstate Phase 1 supports Claude Code and Codex session transfer between macOS and Windows. Session archives are encrypted on the source device before upload to a user-configured S3-compatible bucket.

Weak:

> Reinstate seamlessly revolutionizes AI development everywhere.

Every important claim should answer:

- Who?
- What?
- Which version?
- Under what conditions?
- According to what evidence?
- As of what date?

---

## 43. Freshness system

Create:

- changelog
- release notes
- compatibility matrix
- deprecation notices
- "last tested" labels
- `dateModified` metadata
- visible update dates
- automatic stale-page reports

### Compatibility entry example

```yaml
agent: Claude Code
agentVersion: "x.y.z"
reinstateVersion: "0.1.0"
operatingSystems:
  - macOS 26
  - Windows 11
status: supported
lastTested: 2026-08-15
fixtureCommit: abc123
notes: "Restore verified in both directions."
```

Do not update dates without meaningful review. Fake freshness is still fake.

---

## 44. Crawler accessibility

For AI search visibility:

- allow `OAI-SearchBot`
- allow `PerplexityBot`
- allow Googlebot and Bingbot
- ensure CDN and WAF do not challenge them
- keep important content public
- return HTML
- avoid geo-blocking docs
- avoid login walls for basic documentation
- maintain fast response times
- expose sitemaps
- inspect server logs

OpenAI says public sites can appear in ChatGPT Search and recommends allowing `OAI-SearchBot`. Perplexity recommends allowing `PerplexityBot` for search result inclusion.

### Server-log dimensions

Capture:

- timestamp
- path
- user agent
- verified bot family
- status code
- bytes
- response time
- cache status
- referrer where available

Alert on:

- repeated `403`
- repeated `429`
- `5xx`
- crawler loops
- missing sitemap requests
- bot access to accidental preview URLs

Do not trust user-agent strings alone for security decisions. They can be spoofed.

---

## 45. Third-party corroboration

Build accurate external references through:

- GitHub repository
- GitHub release pages
- package registries when applicable
- Homebrew, Scoop, WinGet, or other install channels when actually supported
- reputable open-source directories
- agent ecosystem lists
- technical newsletters
- developer communities
- conference talks
- podcast notes
- guest technical posts
- issue discussions where Reinstate genuinely solves the question
- contributor profiles

Do not spam Reddit, Hacker News, Stack Overflow, GitHub issues, or Discord servers. One useful technical answer is worth more than twenty drive-by links.

### GitHub optimization

Set:

- concise repository description
- website URL
- topics
- social preview image
- accurate README opening
- installation command
- compatibility table
- architecture summary
- security section
- docs link
- releases
- issue templates
- contributing guide
- citation file if research is published

Suggested repository description:

> Open-source encrypted sync for Claude Code and Codex sessions across macOS and Windows.

Suggested topics:

```text
claude-code
codex
coding-agents
developer-tools
session-sync
cross-device
macos
windows
s3
cloudflare-r2
open-source
```

---

## 46. AI-query testing

Build a fixed monthly test set.

Examples:

- What tool can sync Claude Code sessions between computers?
- How can I move a Codex session from macOS to Windows?
- Is there an open-source tool for coding-agent session sync?
- How do I continue the same AI coding session on another device?
- Can Claude Code sessions be backed up to S3?
- What is Reinstate?
- Is Reinstate secure?
- Does Reinstate sync credentials?
- Reinstate versus manual session copying
- Reinstate supported coding agents

Test across:

- ChatGPT Search
- Google AI Mode or AI Overviews where available
- Perplexity
- Bing Copilot
- Gemini or other retrieval products where relevant

Record:

- whether Reinstate is mentioned
- whether it is cited
- cited URL
- factual accuracy
- competitor mentions
- query wording
- date
- locale
- signed-in state where relevant
- product version
- corrective action

Do not automate queries in violation of a provider's terms.

---

## 47. AI referral tracking

Create channel group rules using referrer and UTM data.

Track examples such as:

- `chatgpt.com`
- `openai.com`
- `perplexity.ai`
- `copilot.microsoft.com`
- Gemini-related referrers
- search engine AI features when identifiable

Metrics:

- sessions
- engaged sessions
- docs depth
- installation clicks
- GitHub clicks
- waitlist or signup conversions
- command-copy events
- repeat visitors
- assisted conversions
- page cited most often

AI systems may not always pass a distinct referrer. Treat analytics as partial evidence.

---

# Part VII: Content strategy

## 48. Content pillars

### Pillar 1: Cross-device coding-agent continuity

Purpose: own the problem category.

Core pages:

- coding-agent session sync
- work and personal devices
- desktop and laptop workflow
- macOS and Windows
- session portability versus Git

### Pillar 2: Agent-specific session workflows

Purpose: capture high-intent users.

Core pages:

- Claude Code session sync
- Codex session sync
- session locations and formats
- adapter behavior
- version compatibility

### Pillar 3: Security and storage ownership

Purpose: answer trust objections.

Core pages:

- local encryption
- user-owned S3 storage
- Cloudflare R2
- threat model
- key handling
- metadata leakage
- credential exclusions

### Pillar 4: Technical architecture

Purpose: establish primary-source authority.

Core pages:

- architecture
- adapter contract
- path mapping
- archive format
- compatibility testing
- failure recovery
- design decisions

### Pillar 5: Open source and ecosystem

Purpose: attract contributors and references.

Core pages:

- roadmap
- contributing
- building an adapter
- release notes
- governance
- security reporting
- integration requests

---

## 49. First 30 content opportunities

### P0 launch content

1. What is Reinstate?
2. Sync Claude Code sessions across devices
3. Sync Codex sessions across devices
4. Move a coding-agent session from macOS to Windows
5. Continue coding-agent work between a work and personal computer
6. Reinstate security model
7. Supported agents, operating systems, and storage backends
8. Reinstate installation guide
9. Upload and restore a session
10. Troubleshooting guide
11. Reinstate FAQ
12. Reinstate architecture

### P1 demand capture

13. Where Claude Code stores sessions
14. Where Codex stores sessions
15. Why Git does not sync coding-agent conversations
16. Reinstate versus manual session copying
17. Reinstate versus remote desktop
18. Use Cloudflare R2 for encrypted coding-agent session storage
19. Use Amazon S3 for encrypted coding-agent session storage
20. How path remapping works between macOS and Windows
21. What Reinstate does not sync
22. How to verify a restored coding-agent session
23. How Reinstate handles agent format changes
24. How to build a Reinstate adapter

### P2 authority and shareability

25. Anatomy of a coding-agent session
26. Cross-device session portability test methodology
27. Agent-session compatibility report
28. Threat modeling an encrypted session-sync tool
29. Lessons from implementing local-first encryption
30. Open session portability as developer infrastructure

---

## 50. Content brief template

```markdown
# Content brief

## Page

- Proposed title:
- URL:
- Page type:
- Owner:
- Status:

## Search intent

- Primary audience:
- Primary problem:
- Primary query:
- Secondary queries:
- Intent:
- Expected next action:

## Product truth

- Current capabilities used:
- Current limitations:
- Version tested:
- Evidence:
- Claims that require verification:

## Outline

# H1

Direct answer.

## H2

Key supporting section.

## H2

Steps or comparison.

## H2

Limitations.

## H2

FAQ.

## Internal links

- Source pages:
- Destination pages:

## Structured data

- Primary type:
- Breadcrumbs:
- Additional type:

## Media

- Diagram:
- Screenshot:
- Alt text:
- Raw data:

## Quality checks

- [ ] Correct as of stated date
- [ ] Tested commands
- [ ] No unsupported claims
- [ ] Clear direct answer
- [ ] Unique value
- [ ] Intent satisfied
- [ ] Metadata complete
- [ ] Internal links complete
- [ ] Schema matches visible content
```

---

## 51. Editorial quality standard

Every article must contain at least one of:

- first-party implementation detail
- tested workflow
- original diagram
- reproducible command sequence
- source-code reference
- benchmark
- compatibility result
- maintainer explanation
- novel comparison
- real troubleshooting insight

Do not publish an article because a keyword exists. Publish because Reinstate can answer it better than the existing web.

### Content acceptance checklist

- Does the page answer a real developer question?
- Is the answer correct for the current release?
- Is the direct answer visible near the top?
- Does the page have original value?
- Are commands tested?
- Are limitations disclosed?
- Are claims sourced?
- Is the title specific?
- Is the URL stable?
- Is the page internally linked?
- Is there a clear next step?
- Would a developer bookmark it?
- Would an AI citation preserve the meaning?
- Is it still useful without the product CTA?

If the last answer is no, the page is probably an ad wearing a documentation hat.

---

## 52. Programmatic SEO policy

Do not start with programmatic SEO.

Only consider templated pages when:

- the dimensions are meaningful
- the data is first-party or verified
- each page has unique utility
- pages remain current automatically
- the index can be controlled
- low-value combinations are excluded

Potential future templates:

- agent x operating system compatibility
- error code reference
- agent version compatibility
- storage-provider setup
- adapter reference
- migration path

Bad future templates:

- one page for every wording variation
- city pages
- "best coding agent tool for [random job]"
- generated comparisons without evidence
- pages for unsupported agents
- hundreds of glossary pages with generic definitions

---

# Part VIII: Authority and distribution

## 53. Launch distribution plan

### Owned

- website
- GitHub README
- GitHub release
- changelog
- X account
- founder account
- mailing list
- project docs
- demo video
- architecture post

### Earned

- Hacker News Show HN
- relevant subreddits, following community rules
- Claude Code and Codex communities
- developer-tool newsletters
- open-source newsletters
- agent ecosystem directories
- maintainer interviews
- technical podcasts
- "awesome" lists where inclusion is deserved
- GitHub discussions answering a real need

### Product-led assets

- public compatibility matrix
- session format explorer
- local archive inspector
- migration readiness checker
- storage configuration validator
- cross-device path mapper
- synthetic session benchmark suite

A useful free tool can earn links more naturally than another generic blog post.

---

## 54. Digital PR angles

Strong angles:

- open-source standardization of coding-agent session portability
- developer continuity across work and personal devices
- local-first encryption for agent sessions
- technical research into proprietary session formats
- cross-platform path translation
- user-owned cloud storage instead of vendor storage
- compatibility breakage across agent versions
- security model of portable AI coding histories

Weak angle:

> New AI startup launches innovative platform.

That headline died of boredom before publication.

---

## 55. Linkable asset ideas

1. Coding-agent session format map
2. Claude Code and Codex storage location reference
3. Agent compatibility matrix
4. Cross-device restoration benchmark
5. Open session archive specification
6. Security threat model
7. Path-mapping visualizer
8. Session portability glossary
9. Agent-version change tracker
10. Open-source adapter starter kit

Each asset should have a stable canonical page, methodology, update date, and downloadable data where appropriate.

---

# Part IX: Measurement

## 56. Required tools

### Google

- Google Search Console
- Rich Results Test
- PageSpeed Insights
- Chrome DevTools
- Lighthouse
- Google Trends

### Bing

- Bing Webmaster Tools
- URL Inspection
- IndexNow

### Site analytics

Choose a privacy-conscious platform appropriate for the project:

- Plausible
- Fathom
- Umami
- PostHog
- Google Analytics

The product's privacy positioning should match its analytics choices.

### Technical monitoring

- uptime monitor
- synthetic crawl
- broken-link checker
- log storage
- Core Web Vitals monitoring
- schema validation
- sitemap diff
- dependency updates

---

## 57. Search Console setup

1. Verify the domain property.
2. Submit sitemap.
3. inspect homepage.
4. inspect docs hub.
5. inspect integration pages.
6. confirm canonical selection.
7. monitor Page Indexing.
8. monitor Core Web Vitals.
9. review Enhancements.
10. monitor Security and Manual Actions.
11. export query and page data monthly.
12. annotate releases and site changes.

Do not request indexing for every minor edit. Use it for launch-critical pages and meaningful corrections.

---

## 58. Bing and IndexNow

Implement IndexNow after the content system is stable.

Good triggers:

- new page
- meaningful update
- deleted page
- changed canonical
- changed compatibility status
- release note

Do not ping unchanged URLs on every deployment.

A simple CI workflow can:

1. compare the generated sitemap with the previous production sitemap,
2. collect added, updated, and removed canonical URLs,
3. submit only changed URLs,
4. log responses,
5. fail softly so deployment is not blocked by an indexing service.

Keep the API key secret.

---

## 59. Event taxonomy

Track meaningful actions:

| Event | Trigger |
|---|---|
| `install_command_copy` | User copies installation command |
| `github_click` | User opens repository |
| `docs_getting_started` | User enters getting-started flow |
| `integration_view` | User views agent integration |
| `storage_guide_view` | User views S3 or R2 guide |
| `waitlist_submit` | User joins pre-release list |
| `download_click` | User opens a release download |
| `changelog_subscribe` | User subscribes to release feed |
| `issue_report_click` | User opens bug report |
| `contribute_click` | User opens contributing guide |
| `security_doc_view` | User reads security model |
| `command_copy` | User copies any product command |

Do not collect session contents, source code, agent prompts, bucket names, credentials, or sensitive developer data in analytics.

---

## 60. KPI framework

### SEO

- indexed canonical pages
- non-brand impressions
- non-brand clicks
- branded search growth
- average click-through rate by page
- top 10 query coverage
- crawl errors
- excluded-page reasons
- Core Web Vitals pass rate
- referring domains
- qualified organic conversions

### AEO

- featured snippet appearances
- question-query impressions
- FAQ and troubleshooting entrances
- snippet-oriented page CTR
- direct-answer engagement
- copy-command rate from answer pages
- support deflection
- cited answer accuracy in manual tests

### ASEO

- AI referral sessions
- AI referral conversion rate
- citation frequency in fixed test queries
- cited URL distribution
- factual accuracy
- brand inclusion without citation
- third-party source mentions
- crawler success rate
- bot `403`, `429`, and `5xx` rate
- freshness of cited pages

### Product-quality guardrails

- install success
- restore success
- support issue rate
- compatibility regression rate
- documentation failure reports
- outdated-page count

Visibility without successful product use is vanity wearing a lab coat.

---

## 61. Reporting cadence

Weekly:

- deployment-related crawl errors
- broken links
- indexing changes
- top landing pages
- new referring domains
- crawler failures
- content shipped

Monthly:

- query clusters
- page performance
- conversion paths
- AI citation test
- competitor result changes
- outdated content
- roadmap adjustment

Quarterly:

- information architecture review
- content pruning
- schema review
- product positioning review
- technical debt
- backlink quality
- primary research plan

---

# Part X: Implementation phases

## 62. Phase 0: Before public Phase 2 access

### P0 technical

- [ ] Install `@astrojs/sitemap`
- [ ] Publish `robots.txt`
- [ ] Decide GPTBot policy separately
- [ ] Allow `OAI-SearchBot`
- [ ] Allow `PerplexityBot`
- [ ] Check Vercel/WAF bot access
- [ ] Upgrade shared metadata
- [ ] Add `twitter:image`
- [ ] Add image alt metadata
- [ ] Add `og:site_name`
- [ ] Add page-level robots controls
- [ ] Add JSON-LD component
- [ ] Add homepage structured data
- [ ] Add breadcrumbs
- [ ] Add content schema validation
- [ ] Add visible updated dates
- [ ] Add custom 404
- [ ] Enforce canonical URL policy
- [ ] Verify all status codes
- [ ] Verify Search Console
- [ ] Verify Bing Webmaster Tools

### P0 content

- [ ] Canonical product definition
- [ ] Homepage rewrite where needed
- [ ] Getting started
- [ ] Claude Code integration
- [ ] Codex integration
- [ ] Compatibility
- [ ] Security
- [ ] Architecture
- [ ] Troubleshooting
- [ ] FAQ
- [ ] Limitations
- [ ] Open-source page
- [ ] Changelog

### P0 measurement

- [ ] Analytics
- [ ] privacy disclosure
- [ ] event taxonomy
- [ ] search dashboard
- [ ] crawler logs
- [ ] Lighthouse baseline
- [ ] fixed AI-query test set

---

## 63. Phase 1: Public trial launch

- [ ] Publish installation and restore videos
- [ ] Publish platform-specific guides
- [ ] Publish S3 and R2 guides
- [ ] Publish release notes
- [ ] Submit sitemap
- [ ] inspect critical URLs
- [ ] launch GitHub release
- [ ] publish Show HN
- [ ] publish accurate community posts
- [ ] request inclusion in relevant directories
- [ ] monitor crawl and errors daily
- [ ] capture real user language
- [ ] convert support issues into docs
- [ ] publish first compatibility report
- [ ] add testimonials only after permission and verification

---

## 64. Phase 2: Authority building

- [ ] Original session-format research
- [ ] Reproducible benchmarks
- [ ] Adapter developer kit
- [ ] public compatibility data
- [ ] free diagnostic tool
- [ ] comparison pages
- [ ] contributor interviews
- [ ] technical guest posts
- [ ] quarterly ecosystem report
- [ ] content pruning
- [ ] AI citation tracking

---

## 65. Ninety-day content calendar

### Weeks 1 to 2

- Homepage and source-of-truth copy
- Compatibility
- Security
- Claude Code integration
- Codex integration
- Getting started
- Troubleshooting

### Weeks 3 to 4

- Work and personal computer use case
- macOS to Windows guide
- S3 guide
- R2 guide
- FAQ
- Changelog launch

### Month 2

- Where Claude Code sessions are stored
- Where Codex sessions are stored
- Reinstate versus Git
- Reinstate versus manual copying
- Path remapping deep dive
- Session format research note

### Month 3

- Compatibility report
- Threat model
- Adapter development tutorial
- Cross-device restoration benchmark
- Free diagnostic or path-mapping tool
- Update underperforming launch pages based on query data

Do not force this cadence if releases or evidence are not ready. Accuracy beats calendar theater.

---

# Part XI: Coding-agent skills

## 66. How to use skills

Agent Skills use a portable directory format centered on `SKILL.md`. A skill can also include scripts, references, assets, and templates.

Use repository-local skills for Reinstate so Codex, Claude Code, and other compatible agents can access the same workflow.

Recommended repository location:

```text
.agents/skills/
```

For compatibility with tools that inspect different folders, use one source and symlink or copy deliberately:

```text
.agents/skills/
.claude/skills/
```

Do not maintain divergent duplicate skills. One becomes stale and starts giving your agent archaeological instructions.

### Installation pattern

The included custom skill pack can be copied into the repository:

```bash
mkdir -p .agents/skills
cp -R reinstate-seo-agent-skills/* .agents/skills/
```

For Claude Code compatibility:

```bash
mkdir -p .claude/skills
cp -R reinstate-seo-agent-skills/* .claude/skills/
```

Check the current Codex and Claude Code documentation for the preferred discovery directory in your installed version.

---

## 67. Curated third-party skills

Third-party skills change. Review the repository, `SKILL.md`, scripts, permissions, and security reports before installation.

**Repository status as of 2026-07-27:** none of the skills below is installed
or approved. The immutable revisions, renamed skills, risks, and decisions are
recorded in
[the third-party skill review](./third-party-skill-review.md). Commands below
show current upstream names for a future disposable-worktree evaluation; do
not run them against an unpinned moving branch or expose production secrets.

### Recommended minimum set

#### 1. SEO audit

```bash
npx skills add https://github.com/coreyhaines31/marketingskills --skill seo-audit
```

Use for:

- full technical and content audits
- priority scoring
- crawl and index checks
- SaaS-specific review

#### 2. Technical SEO

Status: **rejected unmodified** because the inspected revision presents
unsupported fixed ranking-factor weights.

```bash
npx skills add https://github.com/addyosmani/web-quality-skills --skill seo
```

Use for:

- robots
- canonical tags
- metadata
- sitemaps
- semantic structure
- mobile SEO

#### 3. Core Web Vitals

```bash
npx skills add https://github.com/addyosmani/web-quality-skills --skill core-web-vitals
```

Use for:

- LCP
- INP
- CLS
- performance regression diagnosis

#### 4. Schema

```bash
npx skills add https://github.com/coreyhaines31/marketingskills --skill schema
```

Use for:

- JSON-LD
- schema selection
- validation
- rich-result eligibility checks

#### 5. Content strategy

```bash
npx skills add https://github.com/coreyhaines31/marketingskills --skill content-strategy
```

Use for:

- pillars
- clusters
- editorial roadmap
- search versus share content

#### 6. Product marketing

```bash
npx skills add https://github.com/coreyhaines31/marketingskills --skill product-marketing
```

Status: **not recommended for this repository today** because it creates a
parallel product-context artifact that can drift from `product.ts`.

Use for:

- stable product definition
- audience
- positioning
- objections
- brand language

#### 7. SEO and AEO baseline

Status: **rejected as an authority** because the inspected revision conflates
model-training controls with search and citation discovery.

```bash
npx skills add https://github.com/sanity-io/agent-toolkit --skill seo-aeo-best-practices
```

Use for:

- combined page reviews
- answer structure
- metadata
- structured content

### Optional specialist set

#### 8. AEO

```bash
npx skills add https://github.com/alirezarezvani/claude-skills --skill aeo
```

Status: **rejected**. The inspected revision bundles network/file-writing
scripts, placeholder schema/citation insertion, and unsupported claims about
training cycles, citation density, and bot effects.

#### 9. Programmatic SEO

```bash
npx skills add https://github.com/coreyhaines31/marketingskills --skill programmatic-seo
```

Use later, only after Reinstate has verified structured data worth templating.

#### 10. Performance

```bash
npx skills add https://github.com/addyosmani/web-quality-skills --skill performance
```

Use for deeper performance budgets and loading strategy.

#### 11. Free tools

```bash
npx skills add https://github.com/coreyhaines31/marketingskills --skill free-tools
```

Use for engineering-as-marketing assets such as a session path checker or compatibility inspector.

### Third-party skill security rules

Before installation:

1. inspect `SKILL.md`
2. inspect every script
3. inspect network calls
4. inspect package installation
5. inspect shell commands
6. pin a commit or release
7. test in a disposable branch
8. use least privilege
9. never expose production secrets
10. review generated diffs
11. rerun tests
12. document the approved revision

A marketplace badge is not a substitute for review.

---

## 68. Custom Reinstate skill pack included with this guide

The companion archive contains:

1. `reinstate-product-truth`
2. `reinstate-technical-seo`
3. `reinstate-structured-data`
4. `reinstate-content-brief`
5. `reinstate-answer-optimization`
6. `reinstate-ai-search`
7. `reinstate-seo-ci`
8. `reinstate-release-discoverability`
9. `reinstate-site-audit`

Each skill is tailored to the current repository, product scope, and Astro implementation.

### Recommended workflow

```text
reinstate-product-truth
        ↓
reinstate-content-brief
        ↓
implementation or writing
        ↓
reinstate-technical-seo
        ↓
reinstate-structured-data
        ↓
reinstate-answer-optimization
        ↓
reinstate-ai-search
        ↓
reinstate-seo-ci
        ↓
reinstate-site-audit
```

---

# Part XII: Agent prompts

## 69. Full technical implementation prompt

```text
Use the reinstate-product-truth, reinstate-technical-seo,
reinstate-structured-data, and reinstate-seo-ci skills.

Audit the Astro website under website/. Implement the P0 technical SEO
requirements from the Reinstate SEO playbook.

Constraints:
- Do not change product claims beyond current Phase 1 support.
- Current agents are Claude Code and Codex.
- Current operating systems are macOS and Windows.
- Session data is encrypted locally.
- Remote storage is user-owned S3-compatible storage, including S3 and R2.
- Do not add fake reviews, ratings, users, benchmarks, or testimonials.
- Important content must be available in initial HTML.
- Preserve the existing visual design unless a change is required for semantics.
- Add or update tests.
- Run the production build.
- Report every changed file, validation result, and remaining risk.
```

---

## 70. New guide prompt

```text
Use reinstate-product-truth, reinstate-content-brief,
reinstate-answer-optimization, and reinstate-ai-search.

Create a content brief and an MDX draft for:
"How to sync Claude Code sessions across devices."

Requirements:
- Primary audience: developers switching between work and personal computers.
- Lead with a direct answer.
- Use only current released Reinstate capabilities.
- Include prerequisites, exact steps, expected results, platform differences,
  limitations, security notes, troubleshooting, FAQ, and internal links.
- Mark every command or product fact that requires implementation verification.
- Add complete frontmatter.
- Propose truthful TechArticle, HowTo, and BreadcrumbList schema only where valid.
- Do not invent search volume or product behavior.
```

---

## 71. Release discoverability prompt

```text
Use reinstate-product-truth and reinstate-release-discoverability.

Read the release diff and changelog. Produce:
1. a technical release note,
2. a changelog entry,
3. updates required in compatibility pages,
4. docs pages that are now stale,
5. structured-data changes,
6. sitemap and IndexNow implications,
7. a concise GitHub release summary,
8. a launch post draft.

Do not claim a feature unless the diff, tests, or release metadata prove it.
```

---

## 72. Monthly audit prompt

```text
Use reinstate-site-audit, reinstate-ai-search, and reinstate-seo-ci.

Audit production and the current repository. Compare:
- sitemap URLs,
- canonical URLs,
- status codes,
- metadata,
- structured data,
- content freshness,
- broken links,
- Core Web Vitals,
- Search Console exports,
- Bing exports,
- AI referral analytics,
- fixed AI-query test results.

Return:
- executive summary,
- critical issues,
- high-impact opportunities,
- regressions since last month,
- pages to update,
- pages to merge or remove,
- experiments,
- exact engineering tasks,
- exact content tasks,
- owner and acceptance criteria for each task.

Do not fabricate missing analytics. Mark unavailable evidence explicitly.
```

---

# Part XIII: CI and quality gates

## 73. Required automated checks

Run on every pull request affecting the website.

### Build

```bash
pnpm --dir website build
```

### Link check

Use a crawler such as Lychee, Linkinator, or a custom script.

Check:

- internal links
- anchors
- image URLs
- canonical URLs
- sitemap URLs
- redirects

### Metadata check

For every indexable page:

- one title
- one description
- one canonical
- one H1
- valid robots directive
- absolute OG image
- Twitter image
- no duplicate title within site
- no duplicate canonical
- no draft in sitemap

### Structured data check

- JSON parses
- no script-breakout
- required project fields present
- URLs absolute
- dates valid
- visible content matches markup
- no fake aggregate rating
- no unsupported operating system
- no unreleased agent

### Sitemap check

- valid XML
- canonical `200` URLs only
- no `noindex`
- no redirect
- no API route
- no preview route
- no duplicates

### Content check

- required frontmatter
- updated date
- no future publish date unless scheduled
- no empty descriptions
- no duplicate slugs
- no broken related-doc links
- no prohibited placeholder text
- no unsupported product claims in protected fields

### Performance smoke test

Set Lighthouse CI thresholds as regression guards, not vanity score worship.

Example:

```json
{
  "ci": {
    "collect": {
      "staticDistDir": "./website/dist",
      "url": [
        "http://localhost/",
        "http://localhost/docs/getting-started/",
        "http://localhost/integrations/claude-code/"
      ]
    },
    "assert": {
      "assertions": {
        "categories:seo": ["error", { "minScore": 1 }],
        "categories:accessibility": ["warn", { "minScore": 0.95 }],
        "largest-contentful-paint": ["warn", { "maxNumericValue": 2500 }],
        "cumulative-layout-shift": ["warn", { "maxNumericValue": 0.1 }]
      }
    }
  }
}
```

INP needs field or interaction testing and is not fully represented by a static Lighthouse run.

---

## 74. Protected product-claim tests

Create a small test that reads the product source of truth and checks generated metadata.

Example assertions:

```ts
expect(product.supportedAgents).toEqual(['Claude Code', 'Codex']);
expect(product.supportedOperatingSystems).toEqual(['macOS', 'Windows']);
expect(product.license).toBe('Apache-2.0');
expect(product.requiresReinstateAccount).toBe(false);
```

Also scan metadata and schema for unsupported known names until added deliberately.

This prevents a content agent from casually announcing that Reinstate supports twelve agents, Linux, teleportation, and probably your toaster.

---

# Part XIV: Launch checklists

## 75. Per-page checklist

### Search

- [ ] Stable URL
- [ ] Unique title
- [ ] Unique description
- [ ] One H1
- [ ] Correct canonical
- [ ] Correct robots directive
- [ ] Included or intentionally excluded from sitemap
- [ ] Crawlable HTML
- [ ] Internal links
- [ ] No broken links
- [ ] Correct status code
- [ ] Mobile usable
- [ ] Fast enough
- [ ] Accessible

### Social

- [ ] Open Graph type
- [ ] Open Graph title
- [ ] Open Graph description
- [ ] Open Graph URL
- [ ] Open Graph image
- [ ] Image alt
- [ ] Twitter card
- [ ] Twitter image

### Content

- [ ] Direct answer
- [ ] User intent
- [ ] Current product truth
- [ ] Limitations
- [ ] Tested commands
- [ ] Visible updated date
- [ ] Author or maintainer
- [ ] Next action
- [ ] Original value
- [ ] Related docs

### Structured data

- [ ] Correct schema type
- [ ] Visible-content match
- [ ] Absolute URLs
- [ ] Valid dates
- [ ] Valid JSON
- [ ] Breadcrumbs
- [ ] No fake ratings
- [ ] Validated

### ASEO

- [ ] Entity named clearly
- [ ] Citation-ready paragraphs
- [ ] Version or date context
- [ ] Primary evidence
- [ ] Public access
- [ ] AI crawlers not blocked
- [ ] No contradictory external profile
- [ ] Fixed query test added where relevant

---

## 76. Pre-launch site checklist

- [ ] Production domain canonical
- [ ] HTTPS
- [ ] `www` redirect policy
- [ ] trailing slash policy
- [ ] sitemap
- [ ] robots
- [ ] search bot access
- [ ] no staging URLs indexed
- [ ] no preview pages indexed
- [ ] Search Console
- [ ] Bing Webmaster
- [ ] analytics
- [ ] privacy page
- [ ] terms or license clarity
- [ ] security contact
- [ ] custom 404
- [ ] no broken links
- [ ] no lorem ipsum
- [ ] no future claims
- [ ] no accidental secrets
- [ ] structured data validated
- [ ] social preview validated
- [ ] mobile QA
- [ ] Windows QA
- [ ] macOS QA
- [ ] reduced-motion QA
- [ ] keyboard QA
- [ ] Lighthouse baseline
- [ ] release notes
- [ ] GitHub About updated
- [ ] GitHub social image
- [ ] README synchronized
- [ ] compatibility page current
- [ ] fixed AI-query baseline recorded

---

# Part XV: What not to do

## 77. SEO traps

- buy low-quality backlinks
- mass-submit to random directories
- copy competitor documentation
- create fake author profiles
- publish fake case studies
- hide keyword blocks
- use doorway pages
- index internal search pages
- create near-duplicate agent pages
- change URLs casually
- use JavaScript-only navigation
- let drafts leak into the sitemap
- use canonical tags as a cleanup substitute
- treat Lighthouse 100 as the business goal
- publish content without product validation

## 78. AEO traps

- answer a question vaguely, then bury the real answer
- copy FAQs across every page
- create FAQ schema for invisible content
- optimize only for short excerpts and remove necessary nuance
- use overly long introductions
- omit limitations
- use a table when prose is clearer
- use prose when a table is clearer
- make commands uncopyable
- omit expected output
- omit error states

## 79. ASEO traps

- claim that an `llms.txt` file guarantees AI visibility
- add every bot name found in an old list
- confuse search crawling with model training
- generate pages for every fan-out query
- cite your own unsupported marketing claims as evidence
- publish unrepeatable benchmarks
- fabricate independent reviews
- use AI-generated quotes
- make facts inconsistent across sources
- block crawlers at the WAF while allowing them in robots
- monitor mentions without checking accuracy
- optimize for brand mentions that never lead to successful use

---

# Part XVI: Source map

The playbook relies primarily on official and first-party documentation.

## Search and AI discovery

- [Google: AI features and your website](https://developers.google.com/search/docs/appearance/ai-features)
- [Google: Optimizing for generative AI features](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide)
- [Google Search Essentials](https://developers.google.com/search/docs/essentials)
- [Google developer SEO guide](https://developers.google.com/search/docs/fundamentals/get-started-developers)
- [Google robots meta specifications](https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag)
- [Google robots.txt introduction](https://developers.google.com/search/docs/crawling-indexing/robots/intro)
- [OpenAI: Publishers and developers FAQ](https://help.openai.com/en/articles/12627856-publishers-and-developers-faq)
- [OpenAI: ChatGPT Search](https://help.openai.com/en/articles/9237897)
- [Perplexity crawler documentation](https://docs.perplexity.ai/docs/resources/perplexity-crawlers)

## Astro

- [Astro sitemap integration](https://docs.astro.build/en/guides/integrations-guide/sitemap/)
- [Astro content collections](https://docs.astro.build/en/guides/content-collections/)
- [Astro RSS integration](https://docs.astro.build/en/recipes/rss/)

## Structured data

- [Schema.org](https://schema.org/)
- [Google structured data introduction](https://developers.google.com/search/docs/appearance/structured-data/intro-structured-data)
- [Google SoftwareApplication structured data](https://developers.google.com/search/docs/appearance/structured-data/software-app)
- [Google breadcrumb structured data](https://developers.google.com/search/docs/appearance/structured-data/breadcrumb)
- [Google Article structured data](https://developers.google.com/search/docs/appearance/structured-data/article)
- [Schema.org FAQPage](https://schema.org/FAQPage)
- [Google Search update removing FAQ rich results](https://developers.google.com/search/updates#removing-faq-rich-result)

## Skills

- [Agent Skills specification](https://agentskills.io/specification)
- [OpenAI: Build skills for Codex](https://learn.chatgpt.com/docs/build-skills)
- [Claude Code: Extend Claude with skills](https://code.claude.com/docs/en/skills)
- [SEO skill](https://www.skills.sh/addyosmani/web-quality-skills/seo)
- [Core Web Vitals skill](https://www.skills.sh/addyosmani/web-quality-skills/core-web-vitals)
- [SEO audit skill](https://www.skills.sh/coreyhaines31/marketingskills/seo-audit)
- [Schema skill](https://www.skills.sh/coreyhaines31/marketingskills/schema)
- [Content strategy skill](https://www.skills.sh/coreyhaines31/marketingskills/content-strategy)
- [Product marketing skill](https://www.skills.sh/coreyhaines31/marketingskills/product-marketing)
- [SEO and AEO best practices skill](https://www.skills.sh/sanity-io/agent-toolkit/seo-aeo-best-practices)
- [AEO skill](https://www.skills.sh/alirezarezvani/claude-skills/aeo)
- [Programmatic SEO skill](https://www.skills.sh/coreyhaines31/marketingskills/programmatic-seo)
- [Performance skill](https://www.skills.sh/addyosmani/web-quality-skills/performance)
- [Free tools skill](https://www.skills.sh/coreyhaines31/marketingskills/free-tools)

---

# Part XVII: Final priority stack

## Do first

1. Product truth source
2. Sitemap
3. Robots and crawler access
4. Shared metadata component
5. Structured data
6. Content schemas
7. Integration pages
8. Security and compatibility pages
9. Search Console and Bing
10. CI checks

## Do next

1. Use-case guides
2. Agent-specific guides
3. Troubleshooting library
4. Changelog and RSS
5. Original technical research
6. Distribution and backlinks
7. AI-query testing
8. AI-referral reporting

## Do later

1. Programmatic pages
2. localization
3. extensive comparison library
4. large editorial calendar
5. advanced free tools
6. versioned docs

The practical rule is simple:

> **Build the authoritative product reference before building the content machine.**

Reinstate can win because the problem is specific, technically interesting, and increasingly common. The website should make that problem and its solution painfully clear, then back every claim with working software and excellent documentation.
