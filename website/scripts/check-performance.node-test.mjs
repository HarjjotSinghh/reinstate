import assert from 'node:assert/strict';
import {
  mkdtemp,
  mkdir,
  readFile,
  rm,
  writeFile,
} from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  DEFAULT_ROUTE_DEFINITIONS,
  auditPerformance,
  formatReport,
} from './check-performance.mjs';

const KIB = 1024;

test('covers every materially distinct production page template', () => {
  assert.deepEqual(
    DEFAULT_ROUTE_DEFINITIONS.map(({ path }) => path),
    [
      '/',
      '/docs',
      '/docs/getting-started',
      '/docs/troubleshooting',
      '/integrations/claude-code',
      '/privacy',
      '/guides',
      '/guides/sync-claude-code-sessions-across-devices',
      '/blog',
      '/blog/why-git-does-not-sync-coding-agent-sessions',
      '/compare/reinstate-vs-manual-session-copying',
      '/use-cases/work-and-personal-computers',
      '/compatibility',
      '/404',
    ],
  );
  assert.ok(DEFAULT_ROUTE_DEFINITIONS.every(({ required }) => required));
});

async function writeFixture(root, path, value) {
  const destination = join(root, path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, value);
}

function generousBudget() {
  return {
    htmlRaw: 20 * KIB,
    htmlGzip: 10 * KIB,
    cssCodeRaw: 20 * KIB,
    cssCodeGzip: 10 * KIB,
    executableJsRaw: 20 * KIB,
    executableJsGzip: 10 * KIB,
    mediaRaw: 20 * KIB,
    mediaGzip: 20 * KIB,
    staticTransferRaw: 50 * KIB,
    staticTransferGzip: 30 * KIB,
    blockingStyleCount: 3,
    blockingScriptCount: 1,
    blockingScriptRaw: 5 * KIB,
    fontCount: 2,
    fontRaw: 10 * KIB,
    fontGzip: 10 * KIB,
    largestFontRaw: 10 * KIB,
    fontPreloadCount: 1,
    externalAssetCount: 0,
    localAssetRequestCount: 4,
  };
}

function fixtureRoute(budget = generousBudget()) {
  return [
    {
      path: '/',
      label: 'Fixture homepage',
      required: true,
      budget,
    },
    {
      path: '/optional-hub',
      label: 'Optional hub',
      required: false,
      budget,
    },
  ];
}

async function validFixture() {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-performance-check-'));
  await writeFixture(
    root,
    'index.html',
    `<!doctype html>
<html>
  <head>
    <link rel="stylesheet" href="/assets/site.css">
    <style>body { margin: 0 }</style>
    <script>document.documentElement.dataset.ready = 'true'</script>
    <script type="module">document.body?.classList.add('enhanced')</script>
  </head>
  <body><h1>Fixture</h1><img src="/assets/mark.svg" alt=""></body>
</html>`,
  );
  await writeFixture(
    root,
    'assets/site.css',
    `@font-face { font-family: Fixture; src: url("./fixture.woff2") format("woff2"); }
body { font-family: Fixture, sans-serif; }`,
  );
  await writeFixture(root, 'assets/fixture.woff2', Buffer.alloc(256, 7));
  await writeFixture(
    root,
    'assets/mark.svg',
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"></svg>',
  );
  return root;
}

test('accepts representative routes within explainable static budgets', async (t) => {
  const root = await validFixture();
  t.after(() => rm(root, { recursive: true, force: true }));

  const result = await auditPerformance(root, {
    routes: fixtureRoute(),
  });

  assert.deepEqual(result.errors, []);
  assert.equal(result.routes.length, 1, 'optional absent routes are skipped');
  assert.equal(result.routes[0].measured.fontCount, 1);
  assert.equal(result.routes[0].measured.blockingScriptCount, 1);
  assert.match(
    formatReport(result),
    /Performance validation passed: 1 representative route stayed within its static-build budgets\./,
  );
});

test('returns an actionable missing-build failure', async () => {
  const result = await auditPerformance(
    join(tmpdir(), 'reinstate-performance-directory-that-does-not-exist'),
  );

  assert.equal(result.errors.length, 1);
  assert.equal(result.errors[0].code, 'BUILD_MISSING');
  assert.match(formatReport(result), /Run "npm run build"/);
});

test('detects payload, blocking-resource, font, request, and external regressions', async (t) => {
  const root = await validFixture();
  t.after(() => rm(root, { recursive: true, force: true }));

  const htmlPath = join(root, 'index.html');
  const html = await readFile(htmlPath, 'utf8');
  await writeFile(
    htmlPath,
    html.replace(
      '</head>',
      `<link rel="stylesheet" href="/assets/more.css">
       <link rel="stylesheet" href="https://cdn.example.com/remote.css">
       <script src="/assets/blocking.js"></script>
       <script src="https://cdn.example.com/tracker.js"></script>
       </head>`,
    ),
  );
  await writeFixture(root, 'assets/more.css', 'main { min-height: 100vh }');
  await writeFixture(root, 'assets/blocking.js', 'window.blocking = true');

  const zeroBudget = Object.fromEntries(
    Object.keys(generousBudget()).map((metric) => [metric, 0]),
  );
  const result = await auditPerformance(root, {
    routes: fixtureRoute(zeroBudget).slice(0, 1),
  });
  const codes = new Set(result.errors.map((error) => error.code));

  for (const expected of [
    'BUDGET_HTML_RAW',
    'BUDGET_CSS_CODE_RAW',
    'BUDGET_EXECUTABLE_JS_RAW',
    'BUDGET_MEDIA_RAW',
    'BUDGET_STATIC_TRANSFER_RAW',
    'BUDGET_BLOCKING_STYLE_COUNT',
    'BUDGET_BLOCKING_SCRIPT_COUNT',
    'BUDGET_FONT_COUNT',
    'BUDGET_FONT_GZIP',
    'BUDGET_EXTERNAL_ASSET_COUNT',
    'BUDGET_LOCAL_ASSET_REQUEST_COUNT',
    'EXTERNAL_BLOCKING_STYLESHEET',
    'EXTERNAL_BLOCKING_SCRIPT',
  ]) {
    assert.ok(codes.has(expected), `expected ${expected} in ${[...codes]}`);
  }

  assert.match(formatReport(result), /Fix:/);
});

test('fails when a required representative route or local asset is absent', async (t) => {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-performance-missing-'));
  t.after(() => rm(root, { recursive: true, force: true }));

  await writeFixture(
    root,
    'index.html',
    '<!doctype html><link rel="stylesheet" href="/missing.css"><h1>Fixture</h1>',
  );
  const result = await auditPerformance(root, {
    routes: [
      ...fixtureRoute().slice(0, 1),
      {
        path: '/docs/getting-started',
        label: 'Getting started',
        required: true,
        budget: generousBudget(),
      },
    ],
  });
  const codes = new Set(result.errors.map((error) => error.code));

  assert.ok(codes.has('ASSET_MISSING'));
  assert.ok(codes.has('REPRESENTATIVE_ROUTE_MISSING'));
});
