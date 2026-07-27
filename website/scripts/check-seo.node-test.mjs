import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { auditSeo, formatReport } from './check-seo.mjs';

const SITE = 'https://reinstate.dev';

function indexableHtml({
  canonical = `${SITE}/`,
  description = 'Sync coding-agent sessions across devices.',
  extraJsonLd = '',
  image = `${SITE}/social/home.png`,
  title = 'Reinstate session sync',
} = {}) {
  return `<!doctype html>
<html lang="en">
  <head>
    <title>${title}</title>
    <meta name="description" content="${description}">
    <meta name="robots" content="index, follow">
    <link rel="canonical" href="${canonical}">
    <meta property="og:site_name" content="Reinstate">
    <meta property="og:locale" content="en_US">
    <meta property="og:type" content="website">
    <meta property="og:title" content="${title}">
    <meta property="og:description" content="${description}">
    <meta property="og:url" content="${canonical}">
    <meta property="og:image" content="${image}">
    <meta property="og:image:secure_url" content="${image}">
    <meta property="og:image:type" content="image/png">
    <meta property="og:image:width" content="1200">
    <meta property="og:image:height" content="630">
    <meta property="og:image:alt" content="Reinstate session continuity">
    <meta name="twitter:card" content="summary_large_image">
    <meta name="twitter:title" content="${title}">
    <meta name="twitter:description" content="${description}">
    <meta name="twitter:image" content="${image}">
    <meta name="twitter:image:alt" content="Reinstate session continuity">
    <script type="application/ld+json">{"@context":"https://schema.org","@type":"SoftwareApplication","@id":"${SITE}/#software","name":"Reinstate","url":"${SITE}/","description":"Encrypted coding-agent session sync.","applicationCategory":"DeveloperApplication","operatingSystem":["macOS","Windows"],"softwareVersion":"v0.1.0-rc.6","isAccessibleForFree":true,"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@id":"${SITE}/#maintainer"},"license":"https://www.apache.org/licenses/LICENSE-2.0"}</script>
    ${extraJsonLd}
  </head>
  <body><h1>Continue coding-agent work anywhere</h1><h2>What is <code>rein</code> vs <code>reinstate</code>?</h2><svg><title>Decorative continuity diagram</title></svg><img src="/diagram.png" alt="Session continuity diagram" width="1200" height="630" loading="lazy"></body>
</html>`;
}

function pngHeader(width = 1200, height = 630) {
  const source = Buffer.alloc(24);
  Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]).copy(source, 0);
  source.writeUInt32BE(13, 8);
  source.write('IHDR', 12, 'ascii');
  source.writeUInt32BE(width, 16);
  source.writeUInt32BE(height, 20);
  return source;
}

async function writeFixture(root, path, value) {
  const destination = join(root, path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, value);
}

async function validFixture() {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-seo-check-'));
  await writeFixture(root, 'index.html', indexableHtml());
  await writeFixture(
    root,
    'preview/index.html',
    indexableHtml({
      canonical: `${SITE}/preview`,
      image: `${SITE}/social/preview.png`,
      title: 'Preview direction',
    }).replace(
      '<meta name="robots" content="index, follow">',
      '<meta name="robots" content="noindex, nofollow">',
    ),
  );
  await writeFixture(root, 'social/home.png', pngHeader());
  await writeFixture(root, 'social/preview.png', pngHeader());
  await writeFixture(root, 'diagram.png', pngHeader());
  await writeFixture(
    root,
    'robots.txt',
    `User-agent: *
Allow: /
Disallow: /preview/

User-agent: OAI-SearchBot
Allow: /

User-agent: PerplexityBot
Allow: /

Sitemap: ${SITE}/sitemap.xml
`,
  );
  await writeFixture(
    root,
    'sitemap.xml',
    `<?xml version="1.0"?><urlset><url><loc>${SITE}/</loc></url></urlset>`,
  );
  return root;
}

test('accepts a complete build and reports what it checked', async (t) => {
  const root = await validFixture();
  t.after(() => rm(root, { recursive: true, force: true }));

  await writeFixture(
    root,
    'faq/index.html',
    indexableHtml({
      canonical: `${SITE}/faq`,
      extraJsonLd:
        `<script type="application/ld+json">{"@context":"https://schema.org","@type":"FAQPage","@id":"${SITE}/faq#faq","url":"${SITE}/faq","mainEntity":[{"@type":"Question","name":"What is rein vs reinstate?","acceptedAnswer":{"@type":"Answer","text":"They are aliases for the same binary."}}]}</script>`,
      image: `${SITE}/social/faq.png`,
      title: 'Frequently asked questions',
      description: 'Direct answers about Reinstate session continuity.',
    }),
  );
  await writeFixture(root, 'social/faq.png', pngHeader());
  await writeFixture(
    root,
    'sitemap.xml',
    `<?xml version="1.0"?><urlset><url><loc>${SITE}/</loc></url><url><loc>${SITE}/faq</loc></url></urlset>`,
  );

  const result = await auditSeo(root);

  assert.deepEqual(result.errors, []);
  assert.match(
    formatReport(result),
    /SEO validation passed: 2 indexable pages, 3 generated HTML pages, 3 route-specific social cards, 0 redirects, and 2 sitemap URLs checked\./,
  );
});

test('returns an actionable missing-build failure', async () => {
  const result = await auditSeo(
    join(tmpdir(), 'reinstate-seo-directory-that-does-not-exist'),
  );

  assert.equal(result.errors.length, 1);
  assert.equal(result.errors[0].code, 'BUILD_MISSING');
  assert.match(formatReport(result), /Run "npm run build"/);
});

test('rejects duplicate descriptions across indexable pages', async (t) => {
  const root = await validFixture();
  t.after(() => rm(root, { recursive: true, force: true }));
  await writeFixture(
    root,
    'copy/index.html',
    indexableHtml({
      canonical: `${SITE}/copy`,
      image: `${SITE}/social/copy.png`,
      title: 'A distinct copy page',
    }),
  );
  await writeFixture(root, 'social/copy.png', pngHeader());
  await writeFixture(
    root,
    'sitemap.xml',
    `<?xml version="1.0"?><urlset><url><loc>${SITE}/</loc></url><url><loc>${SITE}/copy</loc></url></urlset>`,
  );

  const result = await auditSeo(root);
  assert.ok(result.errors.some(({ code }) => code === 'DESCRIPTION_DUPLICATE'));
});

test('accepts direct permanent redirects to built canonical destinations', async (t) => {
  const root = await validFixture();
  t.after(() => rm(root, { recursive: true, force: true }));
  await writeFixture(
    root,
    'docs/index.html',
    indexableHtml({
      canonical: `${SITE}/docs`,
      description: 'Read the Reinstate documentation and begin a safe session transfer.',
      image: `${SITE}/social/docs.png`,
      title: 'Reinstate documentation',
    }),
  );
  await writeFixture(root, 'social/docs.png', pngHeader());
  await writeFixture(
    root,
    'sitemap.xml',
    `<?xml version="1.0"?><urlset><url><loc>${SITE}/</loc></url><url><loc>${SITE}/docs</loc></url></urlset>`,
  );
  const configPath = join(root, 'vercel.json');
  await writeFixture(
    root,
    'vercel.json',
    JSON.stringify({
      redirects: [
        {
          source: '/docs/overview',
          destination: '/docs',
          permanent: true,
        },
      ],
    }),
  );

  const result = await auditSeo(root, { redirectConfigPath: configPath });

  assert.deepEqual(result.errors, []);
  assert.deepEqual(result.redirects, [
    { source: '/docs/overview', destination: '/docs' },
  ]);
});

test('rejects unsafe redirect routes, missing destinations, chains, loops, and sitemap sources', async (t) => {
  const root = await validFixture();
  t.after(() => rm(root, { recursive: true, force: true }));
  await writeFixture(
    root,
    'docs/index.html',
    indexableHtml({
      canonical: `${SITE}/docs`,
      description: 'Read the Reinstate documentation and begin a safe session transfer.',
      image: `${SITE}/social/docs.png`,
      title: 'Reinstate documentation',
    }),
  );
  await writeFixture(root, 'social/docs.png', pngHeader());
  await writeFixture(
    root,
    'sitemap.xml',
    `<?xml version="1.0"?><urlset><url><loc>${SITE}/</loc></url><url><loc>${SITE}/docs</loc></url><url><loc>${SITE}/old</loc></url></urlset>`,
  );
  const configPath = join(root, 'vercel.json');
  await writeFixture(
    root,
    'vercel.json',
    JSON.stringify({
      redirects: [
        { source: '/old', destination: '/middle', permanent: false },
        { source: '/middle', destination: '/docs', permanent: true },
        { source: '/loop-a', destination: '/loop-b', permanent: true },
        { source: '/loop-b', destination: '/loop-a', permanent: true },
        { source: '/self', destination: '/self', permanent: true },
        { source: '/docs//legacy', destination: '/docs?from=old', permanent: true },
      ],
    }),
  );

  const result = await auditSeo(root, { redirectConfigPath: configPath });
  const codes = new Set(result.errors.map(({ code }) => code));

  for (const expected of [
    'REDIRECT_CHAIN',
    'REDIRECT_DESTINATION_INVALID',
    'REDIRECT_DESTINATION_MISSING',
    'REDIRECT_LOOP',
    'REDIRECT_NOT_PERMANENT',
    'REDIRECT_SELF_LOOP',
    'REDIRECT_SOURCE_INVALID',
    'REDIRECT_SOURCE_IN_SITEMAP',
  ]) {
    assert.ok(codes.has(expected), `expected ${expected} in ${[...codes]}`);
  }
});

test('detects metadata, structured-data, crawler, sitemap, and image regressions', async (t) => {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-seo-check-bad-'));
  t.after(() => rm(root, { recursive: true, force: true }));

  const invalidJsonLd =
    '<script type="application/ld+json">{"@type":"Thing",}</script>';
  const unsupportedJsonLd =
    '<script type="application/ld+json">{"@type":"SoftwareApplication","operatingSystem":"Linux","description":"Supports Gemini CLI","review":[]}</script>';
  const semanticJsonLd =
    `<script type="application/ld+json">{"@context":"https://schema.org","@graph":[` +
    `{"@type":"TechArticle","@id":"relative-id","headline":"Invisible schema headline","description":"Broken fixture","url":"/relative","dateModified":"not-a-date","image":{"@type":"ImageObject","url":"/image.png"},"author":{"@id":"${SITE}/#maintainer"},"mainEntityOfPage":"/relative"},` +
    `{"@type":"Thing","@id":"relative-id"}]}</script>`;

  await writeFixture(
    root,
    'index.html',
    indexableHtml({
      extraJsonLd: `${invalidJsonLd}${unsupportedJsonLd}${semanticJsonLd}`,
    })
      .replace(
        '<meta name="description" content="Sync coding-agent sessions across devices.">',
        '',
      )
      .replace(
        '<img src="/diagram.png" alt="Session continuity diagram" width="1200" height="630" loading="lazy">',
        '<img src="/diagram.png">',
      ),
  );
  await writeFixture(
    root,
    'docs/index.html',
    indexableHtml({
      canonical: `${SITE}/`,
      image: `${SITE}/social/home.png`,
      description: 'Sync coding-agent sessions across devices.',
    }).replace(
      '<meta property="og:description" content="Sync coding-agent sessions across devices.">',
      '<meta property="og:description" content="Supports Gemini CLI on Linux.">',
    ),
  );
  await writeFixture(
    root,
    'preview/index.html',
    '<!doctype html><html><head><title>Preview</title><meta name="robots" content="index, follow"></head><body><h1>Preview</h1></body></html>',
  );
  await writeFixture(root, 'social/home.png', pngHeader(600, 315));
  await writeFixture(
    root,
    'robots.txt',
    `User-agent: *
Allow: /
Crawl-delay: 10
Sitemap: ${SITE}/sitemap.xml
`,
  );
  await writeFixture(
    root,
    'sitemap.xml',
    `<?xml version="1.0"?><urlset>
      <url><loc>${SITE}/</loc></url>
      <url><loc>${SITE}/preview</loc></url>
      <url><loc>${SITE}/preview/</loc></url>
      <url><loc>${SITE}/docs/overview</loc></url>
    </urlset>`,
  );

  const result = await auditSeo(root);
  const codes = new Set(result.errors.map((error) => error.code));

  for (const expected of [
    'CANONICAL_DUPLICATE',
    'DESCRIPTION_COUNT',
    'IMAGE_ALT_MISSING',
    'IMAGE_DIMENSION_MISSING',
    'IMAGE_LOADING_MISSING',
    'JSONLD_INVALID',
    'JSONLD_DATE_INVALID',
    'JSONLD_ID_DUPLICATE',
    'JSONLD_REQUIRED_FIELD',
    'JSONLD_UNSUPPORTED_AGENT',
    'JSONLD_UNSUPPORTED_OS',
    'JSONLD_URL_INVALID',
    'JSONLD_UNVERIFIED_REVIEW',
    'JSONLD_VISIBLE_MISMATCH',
    'PREVIEW_INDEXABLE',
    'ROBOTS_AI_CRAWLER_MISSING',
    'ROBOTS_DIRECTIVE_INVALID',
    'SITEMAP_EXCLUDED_ROUTE',
    'SITEMAP_URL_DUPLICATE',
    'SOCIAL_IMAGE_DIMENSIONS',
    'SOCIAL_IMAGE_DUPLICATE',
    'TITLE_DUPLICATE',
    'META_UNSUPPORTED_AGENT',
    'META_UNSUPPORTED_OS',
  ]) {
    assert.ok(codes.has(expected), `expected ${expected} in ${[...codes]}`);
  }

  assert.match(formatReport(result), /Fix:/);
});
