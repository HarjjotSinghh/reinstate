import assert from 'node:assert/strict';
import test from 'node:test';

import {
  DEFAULT_LAUNCH_PATHS,
  formatProductionDiscoverySummary,
  normalizeBaseUrl,
  parsePageMetadata,
  parseSitemapDocument,
  pngDimensions,
  runProductionDiscoverySmoke,
} from './check-production-discovery.mjs';

const PRODUCTION = 'https://reinstate.dev';
const DEPLOYMENT = 'https://reinstate-preview.vercel.app';
const PNG = (() => {
  const bytes = Buffer.alloc(24);
  Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]).copy(bytes);
  bytes.writeUInt32BE(13, 8);
  bytes.write('IHDR', 12, 'ascii');
  bytes.writeUInt32BE(1200, 16);
  bytes.writeUInt32BE(630, 20);
  return bytes;
})();

function response(body, contentType, status = 200, headers = {}) {
  return new Response(body, {
    status,
    headers: {
      'content-type': contentType,
      ...headers,
    },
  });
}

function page(path, imagePath = path === '/' ? '/og/home.png' : `/og${path}.png`) {
  const canonical = new URL(path, `${PRODUCTION}/`).toString();
  return `<!doctype html>
<html>
  <head>
    <meta name="robots" content="index, follow">
    <link rel="canonical" href="${canonical}">
    <meta property="og:image" content="${PRODUCTION}${imagePath}">
    <meta property="og:image:type" content="image/png">
    <meta property="og:image:width" content="1200">
    <meta property="og:image:height" content="630">
  </head>
  <body><h1>Fixture</h1></body>
</html>`;
}

function healthyFetch({
  previewNoindex = false,
  transientRobotsFailure = false,
} = {}) {
  const calls = [];
  const previewRobotsTag =
    previewNoindex === true ? 'noindex' : previewNoindex;
  let robotsAttempts = 0;
  const robots = `User-agent: *
Allow: /
Disallow: /api/
Disallow: /preview/
Disallow: /drafts/

User-agent: OAI-SearchBot
Allow: /

User-agent: PerplexityBot
Allow: /

Sitemap: ${PRODUCTION}/sitemap-index.xml
`;
  const sitemapIndex =
    `<?xml version="1.0"?><sitemapindex><sitemap><loc>` +
    `${PRODUCTION}/sitemap-0.xml</loc></sitemap></sitemapindex>`;
  const sitemap =
    `<?xml version="1.0"?><urlset>` +
    `<url><loc>${PRODUCTION}/</loc></url>` +
    `<url><loc>${PRODUCTION}/docs</loc></url></urlset>`;

  const fetchImpl = async (input, init = {}) => {
    const url = new URL(input);
    const method = init.method ?? 'GET';
    const userAgent = init.headers?.['user-agent'];
    calls.push({
      method,
      path: url.pathname,
      range: init.headers?.range ?? null,
      userAgent,
    });

    if (url.origin === 'https://www.reinstate.dev') {
      return response(null, 'text/plain', 308, {
        location: `${PRODUCTION}/`,
      });
    }
    if (
      transientRobotsFailure &&
      url.pathname === '/robots.txt' &&
      method === 'GET' &&
      userAgent.startsWith('Reinstate-') &&
      robotsAttempts++ === 0
    ) {
      return response('', 'text/plain', 503, { 'retry-after': '0' });
    }
    if (url.pathname === '/robots.txt') {
      return response(method === 'HEAD' ? null : robots, 'text/plain');
    }
    if (url.pathname === '/sitemap-index.xml') {
      return response(method === 'HEAD' ? null : sitemapIndex, 'application/xml');
    }
    if (url.pathname === '/sitemap-0.xml') {
      return response(sitemap, 'application/xml');
    }
    if (
      ['/rss.xml', '/blog/rss.xml', '/changelog/rss.xml'].includes(
        url.pathname,
      )
    ) {
      return response('<rss><channel><title>Reinstate</title></channel></rss>', 'application/rss+xml');
    }
    if (url.pathname === '/llms.txt') {
      return response('# Reinstate\n', 'text/plain');
    }
    if (url.pathname === '/.well-known/reinstate-discovery-smoke-missing') {
      return response(null, 'text/html', 404);
    }
    if (url.pathname === '/og/home.png' || url.pathname === '/og/docs.png') {
      return response(method === 'HEAD' ? null : PNG, 'image/png', 200, {
        'content-length': String(PNG.length),
      });
    }
    if (url.pathname === '/' || url.pathname === '/docs') {
      return response(
        method === 'HEAD' ? null : page(url.pathname),
        'text/html',
        200,
        previewRobotsTag ? { 'x-robots-tag': previewRobotsTag } : {},
      );
    }
    return response(null, 'text/plain', 404);
  };
  return { calls, fetchImpl };
}

test('parses page, sitemap, and PNG discovery primitives', () => {
  const metadata = parsePageMetadata(page('/docs'));
  assert.deepEqual(metadata.canonicals, [`${PRODUCTION}/docs`]);
  assert.deepEqual(metadata.robots, ['index, follow']);
  assert.equal(parseSitemapDocument('<urlset><url><loc>/</loc></url></urlset>').kind, 'urls');
  assert.deepEqual(pngDimensions(PNG), { width: 1200, height: 630 });
  assert.equal(pngDimensions(Buffer.from('not-png')), null);
});

test('covers canonical reference assets in the default launch contract', () => {
  for (const path of [
    '/glossary',
    '/tools/path-mapping-visualizer',
    '/research/encrypted-snapshot-format-v1',
    '/compatibility/agent-version-history',
  ]) {
    assert.ok(DEFAULT_LAUNCH_PATHS.includes(path), `${path} must be covered`);
  }
});

test('requires explicit acknowledgement for safe non-production HTTPS origins', () => {
  assert.equal(normalizeBaseUrl(), PRODUCTION);
  assert.equal(
    normalizeBaseUrl(DEPLOYMENT, { allowNonProduction: true }),
    DEPLOYMENT,
  );
  assert.throws(
    () => normalizeBaseUrl(DEPLOYMENT),
    /--allow-non-production/,
  );
  assert.throws(
    () =>
      normalizeBaseUrl('http://127.0.0.1:4321', {
        allowNonProduction: true,
      }),
    /public HTTPS/,
  );
  assert.throws(
    () =>
      normalizeBaseUrl('https://169.254.169.254', {
        allowNonProduction: true,
      }),
    /public HTTPS/,
  );
  assert.throws(
    () =>
      normalizeBaseUrl('https://user:password@example.com', {
        allowNonProduction: true,
      }),
    /without credentials/,
  );
});

test('allows Vercel preview noindex only with a narrow explicit acknowledgement', async () => {
  const blockedFixture = healthyFetch({ previewNoindex: true });
  const blocked = await runProductionDiscoverySmoke({
    allowNonProduction: true,
    baseUrl: DEPLOYMENT,
    concurrency: 2,
    fetchImpl: blockedFixture.fetchImpl,
    launchPaths: ['/', '/docs'],
    maxAttempts: 1,
    timeoutMs: 1_000,
  });
  assert.equal(blocked.summary.ok, false);
  assert.ok(
    blocked.findings.some(({ code }) => code === 'X_ROBOTS_TAG'),
  );

  const allowedFixture = healthyFetch({ previewNoindex: true });
  const allowed = await runProductionDiscoverySmoke({
    allowNonProduction: true,
    allowVercelPreviewNoindex: true,
    baseUrl: DEPLOYMENT,
    concurrency: 2,
    fetchImpl: allowedFixture.fetchImpl,
    launchPaths: ['/', '/docs'],
    maxAttempts: 1,
    timeoutMs: 1_000,
  });
  assert.equal(allowed.summary.ok, true);
  assert.equal(allowed.configuration.allowVercelPreviewNoindex, true);
  assert.equal(allowed.summary.vercelPreviewNoindexResponses, 4);

  for (const baseUrl of [PRODUCTION, 'https://example.com']) {
    await assert.rejects(
      runProductionDiscoverySmoke({
        allowNonProduction: true,
        allowVercelPreviewNoindex: true,
        baseUrl,
        fetchImpl: async () =>
          assert.fail('invalid preview exemption must not fetch'),
      }),
      /\*.vercel\.app origin/,
    );
  }
});

test('rejects broader preview X-Robots-Tag directives before promotion', async () => {
  for (const previewNoindex of [
    'noindex, nofollow',
    'googlebot: noindex',
    'noindex, noindex',
  ]) {
    const fixture = healthyFetch({ previewNoindex });
    const report = await runProductionDiscoverySmoke({
      allowNonProduction: true,
      allowVercelPreviewNoindex: true,
      baseUrl: DEPLOYMENT,
      concurrency: 2,
      fetchImpl: fixture.fetchImpl,
      launchPaths: ['/', '/docs'],
      maxAttempts: 1,
      timeoutMs: 1_000,
    });

    assert.equal(report.summary.ok, false, previewNoindex);
    assert.equal(report.summary.vercelPreviewNoindexResponses, 0);
    assert.ok(
      report.findings.some(
        ({ checkId, code }) =>
          code === 'X_ROBOTS_TAG' && checkId === 'launch:head:/',
      ),
      `launch HEAD must reject ${previewNoindex}`,
    );
    assert.ok(
      report.findings.some(
        ({ checkId, code }) =>
          code === 'X_ROBOTS_TAG' && checkId === 'page:get:/',
      ),
      `canonical GET must reject ${previewNoindex}`,
    );
  }
});

test('passes a healthy deployment, retries transient reads, and records redacted evidence', async () => {
  const fixture = healthyFetch({ transientRobotsFailure: true });
  const report = await runProductionDiscoverySmoke({
    allowNonProduction: true,
    baseUrl: DEPLOYMENT,
    concurrency: 2,
    fetchImpl: fixture.fetchImpl,
    launchPaths: ['/', '/docs'],
    maxAttempts: 2,
    sleep: async () => {},
    timeoutMs: 1_000,
  });

  assert.equal(report.summary.ok, true);
  assert.equal(report.summary.sitemapUrls, 2);
  assert.equal(report.summary.openGraphImages, 2);
  assert.equal(report.findings.length, 0);
  assert.ok(report.checks.every((check) => check.ok));
  assert.deepEqual(
    report.checks.find(({ id }) => id === 'robots:get').attemptStatuses,
    [503, 200],
  );
  assert.ok(
    fixture.calls.some(
      ({ method, path }) => method === 'HEAD' && path === '/docs',
    ),
  );
  assert.ok(
    fixture.calls.some(
      ({ method, path, range }) =>
        method === 'GET' && path === '/og/home.png' && range === 'bytes=0-23',
    ),
  );
  for (const call of fixture.calls) {
    assert.ok(call.method === 'GET' || call.method === 'HEAD');
  }
  for (const crawler of ['OAI-SearchBot', 'PerplexityBot']) {
    assert.equal(
      fixture.calls.filter(({ userAgent }) => userAgent === crawler).length,
      4,
    );
  }
  for (const feed of ['/rss.xml', '/blog/rss.xml', '/changelog/rss.xml']) {
    assert.ok(
      fixture.calls.some(
        ({ method, path }) => method === 'GET' && path === feed,
      ),
      `${feed} must have a dedicated discovery check`,
    );
  }
  assert.ok(!JSON.stringify(report).includes('retry-after'));
  assert.match(
    formatProductionDiscoverySummary(report, '/safe/evidence.json'),
    /PASS[\s\S]*Evidence: \/safe\/evidence\.json/,
  );
});

test('requires the production www host to redirect permanently to the canonical apex', async () => {
  const healthy = healthyFetch();
  const passing = await runProductionDiscoverySmoke({
    baseUrl: PRODUCTION,
    concurrency: 2,
    fetchImpl: healthy.fetchImpl,
    launchPaths: ['/', '/docs'],
    maxAttempts: 1,
    timeoutMs: 1_000,
  });

  assert.equal(passing.summary.ok, true);
  assert.ok(
    passing.checks.some(
      ({ id, status }) => id === 'canonical-host:www' && status === 308,
    ),
  );

  const broken = healthyFetch();
  const brokenFetch = async (input, init) => {
    const url = new URL(input);
    if (url.origin === 'https://www.reinstate.dev') {
      return response(null, 'text/html', 200);
    }
    return broken.fetchImpl(input, init);
  };
  const failing = await runProductionDiscoverySmoke({
    baseUrl: PRODUCTION,
    concurrency: 2,
    fetchImpl: brokenFetch,
    launchPaths: ['/', '/docs'],
    maxAttempts: 1,
    timeoutMs: 1_000,
  });
  assert.ok(
    failing.findings.some(
      ({ checkId, code }) =>
        checkId === 'canonical-host:www' && code === 'HTTP_STATUS',
    ),
  );
});

test('reports unsafe sitemap, canonical, robots, image, missing-route, and crawler responses without bodies', async () => {
  const secretBody = 'SUPER_SECRET_RESPONSE_BODY';
  const requestedPaths = [];
  const fetchImpl = async (input, init = {}) => {
    const url = new URL(input);
    const method = init.method ?? 'GET';
    const userAgent = init.headers?.['user-agent'];
    requestedPaths.push(url.pathname);
    if (
      userAgent === 'PerplexityBot' &&
      url.pathname === '/' &&
      method === 'GET'
    ) {
      return response(secretBody, 'text/html', 403);
    }
    if (url.pathname === '/robots.txt') {
      return response(
        method === 'HEAD'
          ? null
          : `User-agent: *\nAllow: /\nSitemap: ${PRODUCTION}/wrong.xml\n`,
        'text/plain',
      );
    }
    if (url.pathname === '/sitemap-index.xml') {
      return response(
        method === 'HEAD'
          ? null
          : `<urlset><url><loc>${PRODUCTION}/</loc></url><url><loc>${PRODUCTION}/api/waitlist</loc></url></urlset>`,
        'application/xml',
      );
    }
    if (
      ['/rss.xml', '/blog/rss.xml', '/changelog/rss.xml', '/llms.txt'].includes(
        url.pathname,
      )
    ) {
      return response(secretBody, 'text/plain', 404);
    }
    if (url.pathname === '/.well-known/reinstate-discovery-smoke-missing') {
      return response(secretBody, 'text/html', 200);
    }
    if (url.pathname.startsWith('/og/')) {
      const wrong = Buffer.from(PNG);
      wrong.writeUInt32BE(600, 16);
      return response(
        method === 'HEAD' ? null : wrong,
        url.pathname === '/og/home.png' ? 'image/png' : 'image/jpeg',
        200,
        { 'content-length': String(wrong.length) },
      );
    }
    if (url.pathname === '/' || url.pathname === '/api/waitlist') {
      const html = page(url.pathname)
        .replace(`${PRODUCTION}${url.pathname}`, `${PRODUCTION}/wrong`)
        .replace('index, follow', 'noindex, nofollow')
        .replace('width" content="1200', 'width" content="600');
      return response(method === 'HEAD' ? null : html, 'text/html');
    }
    if (url.pathname === '/docs') {
      return response(method === 'HEAD' ? null : page('/docs'), 'text/html');
    }
    return response(secretBody, 'text/plain', 404);
  };

  const report = await runProductionDiscoverySmoke({
    allowNonProduction: true,
    baseUrl: DEPLOYMENT,
    concurrency: 2,
    fetchImpl,
    launchPaths: ['/', '/docs'],
    maxAttempts: 1,
    timeoutMs: 1_000,
  });
  const codes = new Set(report.findings.map(({ code }) => code));
  for (const code of [
    'CANONICAL',
    'CRAWLER_CHALLENGE',
    'HTTP_STATUS',
    'LAUNCH_URL_MISSING',
    'OG_IMAGE_DIMENSIONS',
    'OG_METADATA',
    'ROBOTS_META',
    'ROBOTS_PRIVATE_PATHS',
    'ROBOTS_SITEMAP',
    'SITEMAP_LEAKAGE',
  ]) {
    assert.ok(codes.has(code), `expected ${code} in ${[...codes]}`);
  }
  assert.equal(report.summary.ok, false);
  assert.equal(
    requestedPaths.filter((path) => path === '/api/waitlist').length,
    0,
    'excluded sitemap paths must be reported without being fetched',
  );
  assert.ok(!JSON.stringify(report).includes(secretBody));
});

test('rejects unsafe execution bounds before making a request', async () => {
  await assert.rejects(
    runProductionDiscoverySmoke({
      concurrency: 9,
      fetchImpl: async () => assert.fail('invalid configuration must not fetch'),
    }),
    /Concurrency must be between 1 and 8/,
  );
  await assert.rejects(
    runProductionDiscoverySmoke({
      fetchImpl: async () => assert.fail('invalid configuration must not fetch'),
      maxAttempts: 4,
    }),
    /Max attempts must be between 1 and 3/,
  );
  await assert.rejects(
    runProductionDiscoverySmoke({
      fetchImpl: async () => assert.fail('invalid configuration must not fetch'),
      timeoutMs: 30_001,
    }),
    /Timeout must be between 100 and 30000/,
  );
});
