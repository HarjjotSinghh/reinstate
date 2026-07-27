import assert from 'node:assert/strict';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { auditLinks, formatLinkReport } from './check-links.mjs';

const SITE = 'https://reinstate.dev';

async function writeFixture(root, path, value) {
  const destination = join(root, path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, value);
}

async function validFixture() {
  const project = await mkdtemp(join(tmpdir(), 'reinstate-link-check-'));
  const build = join(project, 'dist', 'client');

  await writeFixture(
    build,
    'index.html',
    `<!doctype html>
<html>
  <head>
    <link rel="stylesheet" href="/_astro/site.css">
    <link rel="canonical" href="${SITE}/">
    <meta property="og:image" content="${SITE}/og/home.png">
  </head>
  <body id="top">
    <a href="/docs#getting-started">Read the docs</a>
    <a href="${SITE}/docs#legacy">Legacy anchor</a>
    <a href="/files/release%20notes.pdf" download>Release notes</a>
    <a href="/rss.xml">RSS</a>
    <a href="mailto:hello@example.com">Email</a>
    <a href="https://example.com/research">Research</a>
    <form action="/api/waitlist" method="post"></form>
    <img src="/images/card.png" srcset="/images/card.png 1x, /images/card@2x.png 2x" alt="">
    <script src="/_astro/site.js"></script>
  </body>
</html>`,
  );
  await writeFixture(
    build,
    'docs/index.html',
    `<!doctype html><html><body>
      <h1 id="getting-started">Docs</h1>
      <a name="legacy"></a>
      <a href="#getting-started">Start</a>
    </body></html>`,
  );
  await writeFixture(build, '_astro/site.css', 'body {}');
  await writeFixture(build, '_astro/site.js', 'export {};');
  await writeFixture(build, 'og/home.png', 'png');
  await writeFixture(build, 'images/card.png', 'png');
  await writeFixture(build, 'images/card@2x.png', 'png');
  await writeFixture(build, 'files/release notes.pdf', 'pdf');
  await writeFixture(
    project,
    '.vercel/output/config.json',
    JSON.stringify({
      routes: [{ dest: '_render', src: '^/rss\\.xml$' }],
      version: 3,
    }),
  );

  return { build, project };
}

test('accepts resolvable pages, assets, encoded paths, and anchors', async (t) => {
  const fixture = await validFixture();
  t.after(() => rm(fixture.project, { force: true, recursive: true }));

  const result = await auditLinks(fixture.build);

  assert.deepEqual(result.errors, []);
  assert.equal(result.counts.htmlPages, 2);
  assert.equal(result.counts.runtimeSkipped, 2);
  assert.match(
    formatLinkReport(result),
    /Link validation passed: 2 HTML pages, 4 internal links, 6 asset references, and 3 fragment references checked; 2 runtime endpoints skipped\./,
  );
});

test('returns an actionable missing-build failure', async () => {
  const result = await auditLinks(
    join(tmpdir(), 'reinstate-link-directory-that-does-not-exist'),
  );

  assert.equal(result.errors.length, 1);
  assert.equal(result.errors[0].code, 'BUILD_MISSING');
  assert.match(formatLinkReport(result), /Run the Astro production build/);
});

test('resolves a root-absolute fragment from a nested page against the homepage', async (t) => {
  const project = await mkdtemp(join(tmpdir(), 'reinstate-link-check-root-'));
  const build = join(project, 'dist', 'client');
  t.after(() => rm(project, { force: true, recursive: true }));

  await writeFixture(
    build,
    'index.html',
    '<!doctype html><html><body><section id="root-anchor">Root target</section></body></html>',
  );
  await writeFixture(
    build,
    'docs/nested/index.html',
    '<!doctype html><html><body><a href="/#root-anchor">Root target</a></body></html>',
  );

  const result = await auditLinks(build);

  assert.deepEqual(result.errors, []);
  assert.equal(result.counts.fragments, 1);
  assert.equal(result.counts.internalLinks, 1);
});

test('detects broken links, assets, encoding, fragments, redirects, and HTTP', async (t) => {
  const project = await mkdtemp(join(tmpdir(), 'reinstate-link-check-bad-'));
  const build = join(project, 'dist', 'client');
  t.after(() => rm(project, { force: true, recursive: true }));

  await writeFixture(
    project,
    'vercel.json',
    JSON.stringify({
      redirects: [
        {
          destination: '/docs',
          permanent: true,
          source: '/old-docs',
        },
      ],
    }),
  );
  await writeFixture(
    build,
    'index.html',
    `<!doctype html><html><body>
      <a href="/missing">Missing page</a>
      <a href="/docs#absent">Missing fragment</a>
      <a href="/bad%ZZ">Bad encoding</a>
      <a href="/old-docs">Redirect</a>
      <a href="http://reinstate.dev/docs">Insecure</a>
      <a href="/legacy">Generated redirect</a>
      <img src="/images/missing.png" alt="">
      <script src=""></script>
    </body></html>`,
  );
  await writeFixture(
    build,
    'docs/index.html',
    '<!doctype html><html><body><h1 id="present">Docs</h1></body></html>',
  );
  await writeFixture(
    build,
    'legacy/index.html',
    '<!doctype html><html><head><meta http-equiv="refresh" content="0; url=/docs"></head><body></body></html>',
  );

  const result = await auditLinks(build);
  const codes = new Set(result.errors.map((error) => error.code));

  for (const expected of [
    'ASSET_TARGET_MISSING',
    'ASSET_URL_EMPTY',
    'FRAGMENT_TARGET_MISSING',
    'INTERNAL_URL_INSECURE',
    'LINK_TARGET_MISSING',
    'LINK_TO_REDIRECT',
    'URL_ENCODING_INVALID',
  ]) {
    assert.ok(codes.has(expected), `expected ${expected} in ${[...codes]}`);
  }
  assert.equal(
    result.errors.filter((error) => error.code === 'LINK_TO_REDIRECT').length,
    2,
  );
  assert.match(formatLinkReport(result), /Fix:/);
  assert.match(formatLinkReport(result), /index\.html:\d+/);
  assert.match(formatLinkReport(result), /docs\/index\.html" \(route "\/docs"\)/);
});

test('ignores external and runtime endpoint references without hiding local files', async (t) => {
  const project = await mkdtemp(join(tmpdir(), 'reinstate-link-check-skip-'));
  const build = join(project, 'dist', 'client');
  t.after(() => rm(project, { force: true, recursive: true }));

  await writeFixture(
    build,
    'index.html',
    `<!doctype html><html><body>
      <a href="/api/status">Runtime API</a>
      <a href="tel:+10000000000">Telephone</a>
      <img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP" alt="">
      <a href="//example.com/path">External</a>
      <a href="/install.sh">Installer</a>
    </body></html>`,
  );
  await writeFixture(build, 'install.sh', '#!/bin/sh');

  const result = await auditLinks(build);

  assert.deepEqual(result.errors, []);
  assert.equal(result.counts.internalLinks, 1);
  assert.equal(result.counts.runtimeSkipped, 1);
  assert.equal(result.counts.externalSkipped, 3);
});
