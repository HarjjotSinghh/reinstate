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
    <meta property="og:title" content="${title}">
    <meta property="og:description" content="${description}">
    <meta property="og:url" content="${canonical}">
    <meta property="og:image" content="${image}">
    <meta property="og:image:alt" content="Reinstate session continuity">
    <meta name="twitter:title" content="${title}">
    <meta name="twitter:description" content="${description}">
    <meta name="twitter:image" content="${image}">
    <meta name="twitter:image:alt" content="Reinstate session continuity">
    <script type="application/ld+json">{"@context":"https://schema.org","@type":"SoftwareApplication","operatingSystem":["macOS","Windows"]}</script>
    ${extraJsonLd}
  </head>
  <body><h1>Continue coding-agent work anywhere</h1><svg><title>Decorative continuity diagram</title></svg><img src="/diagram.png" alt="Session continuity diagram" width="1200" height="630" loading="lazy"></body>
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

  const result = await auditSeo(root);

  assert.deepEqual(result.errors, []);
  assert.match(
    formatReport(result),
    /SEO validation passed: 1 indexable page, 2 generated HTML pages, 2 route-specific social cards, and 1 sitemap URL checked\./,
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

test('detects metadata, structured-data, crawler, sitemap, and image regressions', async (t) => {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-seo-check-bad-'));
  t.after(() => rm(root, { recursive: true, force: true }));

  const invalidJsonLd =
    '<script type="application/ld+json">{"@type":"Thing",}</script>';
  const unsupportedJsonLd =
    '<script type="application/ld+json">{"@type":"SoftwareApplication","operatingSystem":"Linux","description":"Supports Gemini CLI","review":[]}</script>';

  await writeFixture(
    root,
    'index.html',
    indexableHtml({
      extraJsonLd: `${invalidJsonLd}${unsupportedJsonLd}`,
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
    }),
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
    'JSONLD_UNSUPPORTED_AGENT',
    'JSONLD_UNSUPPORTED_OS',
    'JSONLD_UNVERIFIED_REVIEW',
    'PREVIEW_INDEXABLE',
    'ROBOTS_AI_CRAWLER_MISSING',
    'SITEMAP_EXCLUDED_ROUTE',
    'SITEMAP_URL_DUPLICATE',
    'SOCIAL_IMAGE_DIMENSIONS',
    'SOCIAL_IMAGE_DUPLICATE',
    'TITLE_DUPLICATE',
  ]) {
    assert.ok(codes.has(expected), `expected ${expected} in ${[...codes]}`);
  }

  assert.match(formatReport(result), /Fix:/);
});
