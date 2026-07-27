import assert from 'node:assert/strict';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  buildIndexNowPlan,
  loadChangesFile,
  loadSitemap,
  parseArguments,
  submitIndexNowPlan,
  validateIndexNowPlan,
} from './indexnow.mjs';

const SITE = 'https://reinstate.dev';
const KEY = 'abcDEF0123456789-dead-beef';

async function writeFixture(root, path, value) {
  const destination = join(root, path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, value);
}

function urlset(entries) {
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${entries
  .map(
    ({ url, lastmod }) =>
      `<url><loc>${url}</loc>${lastmod ? `<lastmod>${lastmod}</lastmod>` : ''}</url>`,
  )
  .join('\n')}
</urlset>`;
}

async function sitemapFixture() {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-indexnow-'));
  const current = join(root, 'current');
  const previous = join(root, 'previous');
  const index = (child) => `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>${SITE}/${child}</loc></sitemap>
</sitemapindex>`;

  await writeFixture(current, 'sitemap-index.xml', index('sitemap-0.xml'));
  await writeFixture(
    current,
    'sitemap-0.xml',
    urlset([
      { url: `${SITE}/` },
      { url: `${SITE}/new` },
      { url: `${SITE}/updated`, lastmod: '2026-07-27' },
      { url: `${SITE}/manual` },
      { url: `${SITE}/new-canonical` },
    ]),
  );
  await writeFixture(previous, 'sitemap-index.xml', index('sitemap-0.xml'));
  await writeFixture(
    previous,
    'sitemap-0.xml',
    urlset([
      { url: `${SITE}/` },
      { url: `${SITE}/old` },
      { url: `${SITE}/updated`, lastmod: '2026-07-26' },
      { url: `${SITE}/manual` },
    ]),
  );
  await writeFixture(
    root,
    'changes.json',
    JSON.stringify({
      updated: ['/manual'],
      deleted: ['/legacy'],
      recanonicalized: [{ from: '/moved', to: '/new-canonical' }],
    }),
  );

  return {
    root,
    current: join(current, 'sitemap-index.xml'),
    previous: join(previous, 'sitemap-index.xml'),
    changes: join(root, 'changes.json'),
  };
}

test('collects added, removed, lastmod, explicit update, deletion, and recanonicalization changes', async (t) => {
  const fixture = await sitemapFixture();
  t.after(() => rm(fixture.root, { force: true, recursive: true }));

  const [current, previous, changes] = await Promise.all([
    loadSitemap(fixture.current),
    loadSitemap(fixture.previous),
    loadChangesFile(fixture.changes),
  ]);
  const plan = buildIndexNowPlan({
    current,
    previous,
    changes,
    generatedAt: '2026-07-27T12:00:00.000Z',
  });

  assert.deepEqual(plan.changes.added, [`${SITE}/new`, `${SITE}/new-canonical`]);
  assert.deepEqual(plan.changes.removed, [`${SITE}/old`]);
  assert.deepEqual(plan.changes.modified, [`${SITE}/updated`]);
  assert.deepEqual(plan.changes.explicitlyUpdated, [`${SITE}/manual`]);
  assert.deepEqual(plan.changes.explicitlyDeleted, [`${SITE}/legacy`]);
  assert.deepEqual(plan.changes.recanonicalized, [
    { from: `${SITE}/moved`, to: `${SITE}/new-canonical` },
  ]);
  assert.deepEqual(plan.urlList, [
    `${SITE}/legacy`,
    `${SITE}/manual`,
    `${SITE}/moved`,
    `${SITE}/new`,
    `${SITE}/new-canonical`,
    `${SITE}/old`,
    `${SITE}/updated`,
  ]);
  assert.equal(plan.entries.find(({ url }) => url === `${SITE}/new-canonical`).reasons.length, 2);
  assert.equal(plan.planDigest.length, 64);
});

test('rejects unsafe change inputs and conflicting deletion declarations', async (t) => {
  const fixture = await sitemapFixture();
  t.after(() => rm(fixture.root, { force: true, recursive: true }));

  await writeFixture(
    fixture.root,
    'unsafe.json',
    JSON.stringify({ updated: ['https://example.com/stolen'] }),
  );
  await assert.rejects(
    loadChangesFile(join(fixture.root, 'unsafe.json')),
    /must use https:\/\/reinstate\.dev/,
  );

  const current = await loadSitemap(fixture.current);
  const previous = await loadSitemap(fixture.previous);
  assert.throws(
    () =>
      buildIndexNowPlan({
        current,
        previous,
        changes: {
          updated: [],
          deleted: [`${SITE}/new`],
          recanonicalized: [],
        },
      }),
    /still in the current sitemap/,
  );
});

test('rejects insecure remote sitemaps and marks newly declared lastmod values changed', async () => {
  await assert.rejects(
    loadSitemap('http://reinstate.dev/sitemap-index.xml', {
      fetchImpl: async () => assert.fail('insecure sitemap must not be fetched'),
    }),
    /must use HTTPS/,
  );
  await assert.rejects(
    loadSitemap('https://example.com/sitemap.xml', {
      fetchImpl: async () => assert.fail('cross-origin sitemap must not be fetched'),
    }),
    /must stay on https:\/\/reinstate\.dev/,
  );

  const url = `${SITE}/existing`;
  const plan = buildIndexNowPlan({
    current: new Map([[url, '2026-07-27']]),
    previous: new Map([[url, undefined]]),
  });
  assert.deepEqual(plan.changes.modified, [url]);
  assert.deepEqual(plan.urlList, [url]);
});

test('never accepts a key as a command-line option', () => {
  assert.throws(
    () => parseArguments(['--key', KEY]),
    /Unknown IndexNow option: --key/,
  );
});

test('detects a reviewed plan that was reordered or tampered with', async (t) => {
  const fixture = await sitemapFixture();
  t.after(() => rm(fixture.root, { force: true, recursive: true }));
  const current = await loadSitemap(fixture.current);
  const previous = await loadSitemap(fixture.previous);
  const plan = buildIndexNowPlan({
    current,
    previous,
    generatedAt: '2026-07-27T12:00:00.000Z',
  });

  assert.doesNotThrow(() =>
    validateIndexNowPlan(plan, {
      now: Date.parse('2026-07-27T13:00:00.000Z'),
      maxAgeMs: 2 * 60 * 60 * 1_000,
    }),
  );
  const tampered = structuredClone(plan);
  tampered.entries[0].reasons.push('invented-change');
  assert.throws(
    () => validateIndexNowPlan(tampered),
    /digest does not match/,
  );
});

test('preflights the public key proof, batches conservatively, and retries transient responses', async () => {
  const entries = Array.from({ length: 5 }, (_value, index) => ({
    url: `${SITE}/page-${index + 1}`,
    reasons: ['explicitly-updated'],
  }));
  const validPlan = buildIndexNowPlan({
    current: new Map(entries.map(({ url }) => [url, undefined])),
    previous: new Map(entries.map(({ url }) => [url, undefined])),
    changes: {
      deleted: [],
      recanonicalized: [],
      updated: entries.map(({ url }) => url),
    },
  });
  const responses = [
    new Response(KEY, { status: 200 }),
    new Response('', { status: 429, headers: { 'retry-after': '0' } }),
    new Response('', { status: 202 }),
    new Response('', { status: 500 }),
    new Response('', { status: 200 }),
    new Response('', { status: 200 }),
  ];
  const requests = [];
  const delays = [];
  const events = [];
  const fetchImpl = async (url, init = {}) => {
    requests.push({ url: url.toString(), init });
    return responses.shift();
  };

  const result = await submitIndexNowPlan(validPlan, {
    key: KEY,
    fetchImpl,
    sleep: async (delay) => delays.push(delay),
    log: (event) => events.push(event),
    batchSize: 2,
    maxAttempts: 3,
    baseDelayMs: 10,
    interBatchDelayMs: 5,
  });

  assert.equal(result.ok, true);
  assert.equal(result.acceptedBatches, 3);
  assert.equal(result.submittedUrls, 5);
  assert.equal(requests[0].url, `${SITE}/${KEY}.txt`);
  assert.equal(requests[0].init.method, undefined);
  assert.equal(requests.filter(({ init }) => init.method === 'POST').length, 5);
  assert.deepEqual(delays, [0, 5, 10, 5]);
  for (const request of requests.filter(({ init }) => init.method === 'POST')) {
    const payload = JSON.parse(request.init.body);
    assert.equal(payload.host, 'reinstate.dev');
    assert.equal(payload.key, KEY);
    assert.ok(payload.urlList.length <= 2);
  }
  assert.ok(events.some(({ status }) => status === 429));
  assert.ok(events.some(({ event }) => event === 'indexnow-complete'));
  assert.ok(!JSON.stringify(events).includes(KEY));
});

test('does not post URLs when the public ownership proof does not match', async () => {
  const plan = buildIndexNowPlan({
    current: new Map([[`${SITE}/new`, undefined]]),
    previous: new Map(),
  });
  const events = [];
  let requestCount = 0;
  const result = await submitIndexNowPlan(plan, {
    key: KEY,
    fetchImpl: async () => {
      requestCount += 1;
      return new Response('different-key', { status: 200 });
    },
    log: (event) => events.push(event),
  });

  assert.equal(result.ok, false);
  assert.equal(result.reason, 'key-proof-failed');
  assert.equal(requestCount, 1);
  assert.ok(events.some(({ event }) => event === 'indexnow-key-proof-failed'));
  assert.ok(!JSON.stringify(events).includes(KEY));
});

test('returns a logged soft failure without retrying permanent errors or exposing the key', async () => {
  const plan = buildIndexNowPlan({
    current: new Map([[`${SITE}/new`, undefined]]),
    previous: new Map(),
  });
  const events = [];
  const requests = [];
  const result = await submitIndexNowPlan(plan, {
    key: KEY,
    fetchImpl: async (url, init = {}) => {
      requests.push({ url: url.toString(), init });
      return requests.length === 1
        ? new Response(KEY, { status: 200 })
        : new Response('key invalid', { status: 403 });
    },
    sleep: async () => assert.fail('permanent error must not retry'),
    log: (event) => events.push(event),
  });

  assert.equal(result.ok, false);
  assert.equal(result.softFailed, true);
  assert.equal(result.reason, 'batch-not-accepted');
  assert.equal(requests.length, 2);
  assert.ok(events.some(({ event }) => event === 'indexnow-soft-failure'));
  assert.ok(!JSON.stringify(events).includes(KEY));
});
