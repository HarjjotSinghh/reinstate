#!/usr/bin/env node

import { chmod, mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_PATH = fileURLToPath(import.meta.url);

export const PRODUCTION_ORIGIN = 'https://reinstate.dev';
export const DEFAULT_LAUNCH_PATHS = [
  '/',
  '/docs',
  '/docs/getting-started',
  '/guides',
  '/guides/move-a-coding-agent-session-from-mac-to-windows',
  '/guides/sync-claude-code-sessions-across-devices',
  '/guides/sync-codex-sessions-across-devices',
  '/guides/use-cloudflare-r2-for-coding-agent-session-storage',
  '/guides/use-s3-for-coding-agent-session-storage',
  '/blog',
  '/blog/why-git-does-not-sync-coding-agent-sessions',
  '/integrations',
  '/integrations/claude-code',
  '/integrations/codex',
  '/compatibility',
  '/security',
  '/about/reinstate',
  '/open-source',
  '/roadmap',
  '/research',
  '/changelog',
  '/privacy',
  '/use-cases',
  '/use-cases/desktop-and-laptop',
  '/use-cases/encrypted-session-backup',
  '/use-cases/work-and-personal-computers',
  '/use-cases/macos-and-windows',
  '/compare',
];

const CRAWLER_USER_AGENTS = ['OAI-SearchBot', 'PerplexityBot'];
const CRAWLER_PATHS = ['/', '/docs', '/robots.txt', '/sitemap-index.xml'];
const DISCOVERY_ASSETS = [
  {
    path: '/rss.xml',
    contentTypes: ['application/rss+xml', 'application/xml', 'text/xml'],
    marker: /<rss\b/i,
  },
  {
    path: '/llms.txt',
    contentTypes: ['text/plain'],
    marker: /\bReinstate\b/,
  },
];
const MISSING_PATH = '/.well-known/reinstate-discovery-smoke-missing';
const DEFAULT_TIMEOUT_MS = 10_000;
const DEFAULT_CONCURRENCY = 4;
const DEFAULT_MAX_ATTEMPTS = 2;
const MAX_TIMEOUT_MS = 30_000;
const MAX_CONCURRENCY = 8;
const MAX_ATTEMPTS = 3;
const MAX_RETRY_DELAY_MS = 2_000;
const MAX_HTML_BYTES = 2 * 1024 * 1024;
const MAX_TEXT_BYTES = 1024 * 1024;
const MAX_IMAGE_BYTES = 5 * 1024 * 1024;
const MAX_SITEMAP_DOCUMENTS = 10;
const MAX_SITEMAP_URLS = 100;
const STANDARD_USER_AGENT =
  'Reinstate-Discoverability-Smoke/1.0 (+https://reinstate.dev/)';

function decodeHtml(value) {
  const named = {
    amp: '&',
    apos: "'",
    gt: '>',
    lt: '<',
    nbsp: '\u00a0',
    quot: '"',
  };
  return value.replace(
    /&(?:#(\d+)|#x([\da-f]+)|([a-z]+));/gi,
    (entity, decimal, hexadecimal, name) => {
      if (decimal) return String.fromCodePoint(Number.parseInt(decimal, 10));
      if (hexadecimal) {
        return String.fromCodePoint(Number.parseInt(hexadecimal, 16));
      }
      return named[name.toLowerCase()] ?? entity;
    },
  );
}

function parseAttributes(tag) {
  const attributes = {};
  const body = tag.replace(/^<[^\s>]+/i, '').replace(/\/?>\s*$/i, '');
  const pattern =
    /([^\s"'=<>`]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g;
  for (const match of body.matchAll(pattern)) {
    attributes[match[1].toLowerCase()] = decodeHtml(
      match[2] ?? match[3] ?? match[4] ?? '',
    );
  }
  return attributes;
}

function findTags(markup, tagName) {
  return [...markup.matchAll(new RegExp(`<${tagName}\\b[^>]*>`, 'gi'))].map(
    (match) => parseAttributes(match[0]),
  );
}

function metaValues(markup, attribute, name) {
  return findTags(markup, 'meta')
    .filter(
      (attributes) =>
        attributes[attribute]?.toLowerCase() === name.toLowerCase(),
    )
    .map((attributes) => attributes.content ?? '');
}

export function parsePageMetadata(html) {
  const canonicals = findTags(html, 'link')
    .filter((attributes) =>
      (attributes.rel ?? '').toLowerCase().split(/\s+/).includes('canonical'),
    )
    .map((attributes) => attributes.href ?? '');
  return {
    canonicals,
    robots: metaValues(html, 'name', 'robots'),
    ogImage: metaValues(html, 'property', 'og:image'),
    ogImageType: metaValues(html, 'property', 'og:image:type'),
    ogImageWidth: metaValues(html, 'property', 'og:image:width'),
    ogImageHeight: metaValues(html, 'property', 'og:image:height'),
  };
}

function xmlDecode(value) {
  return decodeHtml(value.trim());
}

function xmlTagValues(xml, tag) {
  return [
    ...xml.matchAll(
      new RegExp(`<${tag}(?:\\s[^>]*)?>([\\s\\S]*?)<\\/${tag}>`, 'gi'),
    ),
  ].map((match) => xmlDecode(match[1]));
}

export function parseSitemapDocument(xml) {
  if (/<sitemapindex(?:\s|>)/i.test(xml)) {
    return { kind: 'index', locations: xmlTagValues(xml, 'loc') };
  }
  if (/<urlset(?:\s|>)/i.test(xml)) {
    return { kind: 'urls', locations: xmlTagValues(xml, 'loc') };
  }
  throw new Error('not-sitemap');
}

export function pngDimensions(bytes) {
  const buffer = Buffer.from(bytes);
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  if (
    buffer.length < 24 ||
    !buffer.subarray(0, 8).equals(signature) ||
    buffer.toString('ascii', 12, 16) !== 'IHDR'
  ) {
    return null;
  }
  return {
    width: buffer.readUInt32BE(16),
    height: buffer.readUInt32BE(20),
  };
}

function isPrivateHostname(hostname) {
  const normalized = hostname.toLowerCase().replace(/^\[|\]$/g, '');
  const privateIpv6 =
    normalized.includes(':') &&
    (normalized === '::1' ||
      normalized === '::' ||
      normalized.startsWith('fc') ||
      normalized.startsWith('fd') ||
      /^fe[89ab]/.test(normalized));
  if (
    normalized === 'localhost' ||
    normalized === '0.0.0.0' ||
    privateIpv6 ||
    normalized.endsWith('.localhost') ||
    /^127\./.test(normalized) ||
    /^10\./.test(normalized) ||
    /^169\.254\./.test(normalized) ||
    /^192\.168\./.test(normalized)
  ) {
    return true;
  }
  const match = normalized.match(/^172\.(\d+)\./);
  if (match && Number(match[1]) >= 16 && Number(match[1]) <= 31) return true;
  const carrierGradeNat = normalized.match(/^100\.(\d+)\./);
  if (
    carrierGradeNat &&
    Number(carrierGradeNat[1]) >= 64 &&
    Number(carrierGradeNat[1]) <= 127
  ) {
    return true;
  }
  const firstOctet = Number(normalized.split('.')[0]);
  return Number.isInteger(firstOctet) && firstOctet >= 224;
}

export function normalizeBaseUrl(
  input = PRODUCTION_ORIGIN,
  { allowNonProduction = false } = {},
) {
  const url = new URL(input);
  if (
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    (url.pathname !== '/' && url.pathname !== '')
  ) {
    throw new Error(
      'Base URL must be an origin without credentials, a path, query, or fragment.',
    );
  }
  if (url.origin !== PRODUCTION_ORIGIN) {
    if (!allowNonProduction) {
      throw new Error(
        'Non-production targets require the explicit --allow-non-production acknowledgement.',
      );
    }
    if (url.protocol !== 'https:' || isPrivateHostname(url.hostname)) {
      throw new Error(
        'Non-production targets must use public HTTPS; private and localhost targets are refused.',
      );
    }
  }
  if (url.protocol !== 'https:') {
    throw new Error('Production discovery checks require HTTPS.');
  }
  return url.origin;
}

function normalizeCanonical(value, expectedOrigin = PRODUCTION_ORIGIN) {
  try {
    const url = new URL(value);
    if (
      url.origin !== expectedOrigin ||
      url.protocol !== 'https:' ||
      url.username ||
      url.password ||
      url.search ||
      url.hash
    ) {
      return null;
    }
    const pathname =
      url.pathname === '/' ? '/' : url.pathname.replace(/\/+$/, '');
    return new URL(pathname, expectedOrigin).toString();
  } catch {
    return null;
  }
}

function normalizedPath(pathname) {
  if (!pathname.startsWith('/') || pathname.includes('\\')) return null;
  const url = new URL(pathname, PRODUCTION_ORIGIN);
  if (url.origin !== PRODUCTION_ORIGIN || url.search || url.hash) return null;
  return url.pathname === '/' ? '/' : url.pathname.replace(/\/+$/, '');
}

function canonicalForPath(pathname) {
  return new URL(pathname, PRODUCTION_ORIGIN).toString();
}

function mediaType(value) {
  return (value ?? '').split(';', 1)[0].trim().toLowerCase();
}

function retryDelay(response, attempt) {
  const retryAfter = response?.headers.get('retry-after');
  if (retryAfter) {
    const seconds = Number(retryAfter);
    if (Number.isFinite(seconds) && seconds >= 0) {
      return Math.min(seconds * 1_000, MAX_RETRY_DELAY_MS);
    }
    const date = Date.parse(retryAfter);
    if (Number.isFinite(date)) {
      return Math.min(
        Math.max(0, date - Date.now()),
        MAX_RETRY_DELAY_MS,
      );
    }
  }
  return Math.min(250 * 2 ** (attempt - 1), MAX_RETRY_DELAY_MS);
}

async function readBodyLimited(response, maximumBytes) {
  const declaredLength = Number(response.headers.get('content-length'));
  if (Number.isFinite(declaredLength) && declaredLength > maximumBytes) {
    await response.body?.cancel();
    throw new Error('body-too-large');
  }
  if (!response.body) return Buffer.alloc(0);

  const reader = response.body.getReader();
  const chunks = [];
  let total = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.byteLength;
    if (total > maximumBytes) {
      await reader.cancel();
      throw new Error('body-too-large');
    }
    chunks.push(Buffer.from(value));
  }
  return Buffer.concat(chunks, total);
}

async function readPrefix(response, length) {
  if (!response.body) return Buffer.alloc(0);
  const reader = response.body.getReader();
  const chunks = [];
  let total = 0;
  while (total < length) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(Buffer.from(value));
    total += value.byteLength;
  }
  await reader.cancel();
  return Buffer.concat(chunks, total).subarray(0, length);
}

async function mapLimit(values, limit, operation) {
  const results = new Array(values.length);
  let nextIndex = 0;
  async function worker() {
    while (nextIndex < values.length) {
      const index = nextIndex;
      nextIndex += 1;
      results[index] = await operation(values[index], index);
    }
  }
  await Promise.all(
    Array.from({ length: Math.min(limit, values.length) }, () => worker()),
  );
  return results;
}

function createRequestClient({
  baseUrl,
  checks,
  fetchImpl,
  maxAttempts,
  sleep,
  timeoutMs,
}) {
  return async function request({
    category,
    headers = {},
    id,
    method = 'GET',
    path,
    userAgent = STANDARD_USER_AGENT,
  }) {
    const started = Date.now();
    const attemptStatuses = [];
    let response;
    let requestError = false;

    for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
      requestError = false;
      try {
        response = await fetchImpl(new URL(path, `${baseUrl}/`), {
          cache: 'no-store',
          headers: {
            accept: '*/*',
            'user-agent': userAgent,
            ...headers,
          },
          method,
          redirect: 'manual',
          signal: AbortSignal.timeout(timeoutMs),
        });
        attemptStatuses.push(response.status);
      } catch {
        response = undefined;
        requestError = true;
        attemptStatuses.push(null);
      }

      const retryable =
        requestError ||
        response?.status === 429 ||
        (response?.status !== undefined && response.status >= 500);
      if (!retryable || attempt === maxAttempts) break;
      await response?.body?.cancel();
      await sleep(retryDelay(response, attempt));
    }

    const check = {
      id,
      category,
      method,
      path,
      userAgent,
      attempts: attemptStatuses.length,
      attemptStatuses,
      status: response?.status ?? null,
      contentType: response
        ? mediaType(response.headers.get('content-type'))
        : null,
      durationMs: Date.now() - started,
      issues: [],
    };
    if (requestError) {
      check.issues.push('REQUEST_FAILED');
    }
    checks.push(check);
    return { check, response };
  };
}

function addFinding(findings, check, code, message) {
  if (check && !check.issues.includes(code)) check.issues.push(code);
  findings.push({
    code,
    checkId: check?.id ?? null,
    path: check?.path ?? null,
    message,
  });
}

function requireStatus(findings, check, response, expected) {
  if (!response || !expected.includes(response.status)) {
    addFinding(
      findings,
      check,
      'HTTP_STATUS',
      `Expected HTTP ${expected.join(' or ')} for ${check.path}.`,
    );
    return false;
  }
  return true;
}

function requireType(findings, check, response, expected) {
  if (!response || !expected.includes(mediaType(response.headers.get('content-type')))) {
    addFinding(
      findings,
      check,
      'CONTENT_TYPE',
      `Expected ${expected.join(' or ')} for ${check.path}.`,
    );
    return false;
  }
  return true;
}

function excludedSitemapPath(pathname) {
  return (
    pathname === '/api' ||
    pathname.startsWith('/api/') ||
    pathname === '/preview' ||
    pathname.startsWith('/preview/') ||
    pathname === '/404' ||
    pathname === '/404.html'
  );
}

async function discardResponse(response) {
  try {
    await response?.body?.cancel();
  } catch {
    // A response that is already closed needs no further cleanup.
  }
}

function parseRobotsGroups(source) {
  const groups = [];
  let group = null;
  for (const rawLine of source.split(/\r?\n/)) {
    const line = rawLine.replace(/#.*$/, '').trim();
    if (!line) continue;
    const match = line.match(/^([^:]+):\s*(.*)$/);
    if (!match) continue;
    const directive = match[1].trim().toLowerCase();
    const value = match[2].trim();
    if (directive === 'user-agent') {
      if (!group || group.rules.length > 0) {
        group = { agents: [], rules: [] };
        groups.push(group);
      }
      group.agents.push(value.toLowerCase());
    } else if (
      group &&
      (directive === 'allow' || directive === 'disallow')
    ) {
      group.rules.push({ directive, value });
    }
  }
  return groups;
}

function robotsAllows(groups, userAgent, pathname) {
  const normalizedAgent = userAgent.toLowerCase();
  const exact = groups.filter((group) =>
    group.agents.some(
      (agent) => agent !== '*' && normalizedAgent.includes(agent),
    ),
  );
  const candidates =
    exact.length > 0
      ? exact
      : groups.filter((group) => group.agents.includes('*'));
  const rules = candidates
    .flatMap((group) => group.rules)
    .filter((rule) => rule.value && pathname.startsWith(rule.value))
    .sort((left, right) => {
      const length = right.value.length - left.value.length;
      if (length !== 0) return length;
      return left.directive === 'allow' ? -1 : 1;
    });
  return rules[0]?.directive !== 'disallow';
}

function pageValidation({
  check,
  expectedCanonical,
  findings,
  html,
  response,
}) {
  const metadata = parsePageMetadata(html);
  if (
    metadata.canonicals.length !== 1 ||
    normalizeCanonical(metadata.canonicals[0]) !== expectedCanonical
  ) {
    addFinding(
      findings,
      check,
      'CANONICAL',
      `Expected one production canonical matching ${check.path}.`,
    );
  }
  if (
    metadata.robots.length !== 1 ||
    /\bnoindex\b/i.test(metadata.robots[0])
  ) {
    addFinding(
      findings,
      check,
      'ROBOTS_META',
      `Expected one indexable robots meta tag for ${check.path}.`,
    );
  }
  if (/\bnoindex\b/i.test(response.headers.get('x-robots-tag') ?? '')) {
    addFinding(
      findings,
      check,
      'X_ROBOTS_TAG',
      `Unexpected noindex X-Robots-Tag for ${check.path}.`,
    );
  }
  if (
    metadata.ogImage.length !== 1 ||
    metadata.ogImageType.length !== 1 ||
    metadata.ogImageType[0] !== 'image/png' ||
    metadata.ogImageWidth.length !== 1 ||
    metadata.ogImageWidth[0] !== '1200' ||
    metadata.ogImageHeight.length !== 1 ||
    metadata.ogImageHeight[0] !== '630'
  ) {
    addFinding(
      findings,
      check,
      'OG_METADATA',
      `Expected one declared 1200x630 PNG Open Graph image for ${check.path}.`,
    );
  }
  const image = normalizeCanonical(metadata.ogImage[0] ?? '');
  if (!image) {
    addFinding(
      findings,
      check,
      'OG_IMAGE_URL',
      `Expected a canonical production Open Graph image URL for ${check.path}.`,
    );
    return null;
  }
  const imagePath = new URL(image).pathname;
  if (excludedSitemapPath(imagePath) || !imagePath.endsWith('.png')) {
    addFinding(
      findings,
      check,
      'OG_IMAGE_PATH',
      `Open Graph image must be a public PNG asset for ${check.path}.`,
    );
    return null;
  }
  return image;
}

async function responseText(response, maximumBytes) {
  return (await readBodyLimited(response, maximumBytes)).toString('utf8');
}

async function fetchSitemapInventory({
  concurrency,
  findings,
  request,
}) {
  const queue = ['/sitemap-index.xml'];
  const seenDocuments = new Set();
  const sitemapUrls = [];

  while (queue.length > 0) {
    const batch = queue.splice(0, concurrency);
    await mapLimit(batch, concurrency, async (path) => {
      if (seenDocuments.has(path)) {
        addFinding(
          findings,
          null,
          'SITEMAP_RECURSION',
          `Sitemap document is duplicated or recursive: ${path}.`,
        );
        return;
      }
      seenDocuments.add(path);
      if (seenDocuments.size > MAX_SITEMAP_DOCUMENTS) {
        addFinding(
          findings,
          null,
          'SITEMAP_LIMIT',
          `Sitemap exceeds ${MAX_SITEMAP_DOCUMENTS} documents.`,
        );
        return;
      }

      const { check, response } = await request({
        category: 'sitemap',
        id: `sitemap:get:${path}`,
        path,
      });
      if (
        !requireStatus(findings, check, response, [200]) ||
        !requireType(findings, check, response, [
          'application/xml',
          'text/xml',
        ])
      ) {
        await discardResponse(response);
        return;
      }

      let parsed;
      try {
        parsed = parseSitemapDocument(
          await responseText(response, MAX_TEXT_BYTES),
        );
      } catch {
        addFinding(
          findings,
          check,
          'SITEMAP_PARSE',
          `Could not parse sitemap document ${path}.`,
        );
        return;
      }
      if (parsed.locations.length === 0) {
        addFinding(
          findings,
          check,
          'SITEMAP_EMPTY',
          `Sitemap document has no locations: ${path}.`,
        );
      }

      for (const location of parsed.locations) {
        const normalized = normalizeCanonical(location);
        if (!normalized) {
          addFinding(
            findings,
            check,
            'SITEMAP_URL',
            `Sitemap contains a non-canonical production URL in ${path}.`,
          );
          continue;
        }
        const pathname = new URL(normalized).pathname;
        if (parsed.kind === 'index') {
          if (excludedSitemapPath(pathname)) {
            addFinding(
              findings,
              check,
              'SITEMAP_LEAKAGE',
              `Sitemap index includes excluded path ${pathname}.`,
            );
          } else {
            queue.push(pathname);
          }
        } else {
          sitemapUrls.push(normalized);
        }
      }
    });
    if (
      seenDocuments.size > MAX_SITEMAP_DOCUMENTS ||
      sitemapUrls.length > MAX_SITEMAP_URLS
    ) {
      break;
    }
  }

  if (sitemapUrls.length > MAX_SITEMAP_URLS) {
    addFinding(
      findings,
      null,
      'SITEMAP_LIMIT',
      `Sitemap exceeds ${MAX_SITEMAP_URLS} canonical URLs.`,
    );
  }
  return sitemapUrls.slice(0, MAX_SITEMAP_URLS);
}

export async function runProductionDiscoverySmoke({
  allowNonProduction = false,
  baseUrl = PRODUCTION_ORIGIN,
  concurrency = DEFAULT_CONCURRENCY,
  fetchImpl = globalThis.fetch,
  launchPaths = DEFAULT_LAUNCH_PATHS,
  maxAttempts = DEFAULT_MAX_ATTEMPTS,
  sleep = (milliseconds) =>
    new Promise((resolveSleep) => setTimeout(resolveSleep, milliseconds)),
  timeoutMs = DEFAULT_TIMEOUT_MS,
} = {}) {
  const safeBaseUrl = normalizeBaseUrl(baseUrl, { allowNonProduction });
  if (
    !Number.isInteger(concurrency) ||
    concurrency < 1 ||
    concurrency > MAX_CONCURRENCY
  ) {
    throw new Error(`Concurrency must be between 1 and ${MAX_CONCURRENCY}.`);
  }
  if (
    !Number.isInteger(maxAttempts) ||
    maxAttempts < 1 ||
    maxAttempts > MAX_ATTEMPTS
  ) {
    throw new Error(`Max attempts must be between 1 and ${MAX_ATTEMPTS}.`);
  }
  if (
    !Number.isInteger(timeoutMs) ||
    timeoutMs < 100 ||
    timeoutMs > MAX_TIMEOUT_MS
  ) {
    throw new Error(`Timeout must be between 100 and ${MAX_TIMEOUT_MS} ms.`);
  }

  const safeLaunchPaths = launchPaths.map((path) => {
    const normalized = normalizedPath(path);
    if (!normalized) throw new Error(`Unsafe launch path: ${path}`);
    return normalized;
  });
  const startedAt = new Date();
  const checks = [];
  const findings = [];
  const request = createRequestClient({
    baseUrl: safeBaseUrl,
    checks,
    fetchImpl,
    maxAttempts,
    sleep,
    timeoutMs,
  });

  const robotsResult = await request({
    category: 'robots',
    id: 'robots:get',
    path: '/robots.txt',
  });
  let robots = '';
  if (
    requireStatus(findings, robotsResult.check, robotsResult.response, [200]) &&
    requireType(findings, robotsResult.check, robotsResult.response, [
      'text/plain',
    ])
  ) {
    try {
      robots = await responseText(robotsResult.response, MAX_TEXT_BYTES);
    } catch {
      addFinding(
        findings,
        robotsResult.check,
        'BODY_LIMIT',
        'robots.txt exceeds the response-size limit.',
      );
    }
  } else {
    await discardResponse(robotsResult.response);
  }
  if (robots) {
    if (
      !new RegExp(
        `^\\s*Sitemap:\\s*${PRODUCTION_ORIGIN.replaceAll('.', '\\.')}/sitemap-index\\.xml\\s*$`,
        'im',
      ).test(robots)
    ) {
      addFinding(
        findings,
        robotsResult.check,
        'ROBOTS_SITEMAP',
        'robots.txt must declare the production sitemap index.',
      );
    }
    const groups = parseRobotsGroups(robots);
    for (const crawler of CRAWLER_USER_AGENTS) {
      if (
        !groups.some((group) =>
          group.agents.includes(crawler.toLowerCase()),
        )
      ) {
        addFinding(
          findings,
          robotsResult.check,
          'ROBOTS_CRAWLER_GROUP',
          `robots.txt must declare an explicit ${crawler} group.`,
        );
      }
      if (!robotsAllows(groups, crawler, '/')) {
        addFinding(
          findings,
          robotsResult.check,
          'ROBOTS_CRAWLER_BLOCKED',
          `robots.txt blocks ${crawler} from canonical pages.`,
        );
      }
    }
    if (
      robotsAllows(groups, STANDARD_USER_AGENT, '/api/') ||
      robotsAllows(groups, STANDARD_USER_AGENT, '/preview/') ||
      robotsAllows(groups, STANDARD_USER_AGENT, '/drafts/')
    ) {
      addFinding(
        findings,
        robotsResult.check,
        'ROBOTS_PRIVATE_PATHS',
        'Wildcard robots policy must disallow /api/, /preview/, and /drafts/.',
      );
    }
  }

  if (safeBaseUrl === PRODUCTION_ORIGIN) {
    const wwwResult = await request({
      category: 'canonical-host',
      id: 'canonical-host:www',
      method: 'HEAD',
      path: 'https://www.reinstate.dev/',
    });
    if (
      requireStatus(
        findings,
        wwwResult.check,
        wwwResult.response,
        [301, 308],
      )
    ) {
      let redirectTarget = null;
      try {
        redirectTarget = new URL(
          wwwResult.response.headers.get('location') ?? '',
          'https://www.reinstate.dev/',
        ).toString();
      } catch {
        // Report the canonical-host finding below.
      }
      if (redirectTarget !== `${PRODUCTION_ORIGIN}/`) {
        addFinding(
          findings,
          wwwResult.check,
          'CANONICAL_HOST_REDIRECT',
          `Expected www.reinstate.dev to redirect permanently to ${PRODUCTION_ORIGIN}/.`,
        );
      }
    }
    await discardResponse(wwwResult.response);
  }

  const sitemapUrls = await fetchSitemapInventory({
    concurrency,
    findings,
    request,
  });
  const sitemapSet = new Set(sitemapUrls);
  if (sitemapSet.size !== sitemapUrls.length) {
    addFinding(
      findings,
      null,
      'SITEMAP_DUPLICATE',
      'Sitemap contains duplicate canonical URLs.',
    );
  }
  for (const url of sitemapSet) {
    const pathname = new URL(url).pathname;
    if (excludedSitemapPath(pathname)) {
      addFinding(
        findings,
        null,
        'SITEMAP_LEAKAGE',
        `Sitemap includes excluded preview, API, or error path ${pathname}.`,
      );
    }
  }
  for (const path of safeLaunchPaths) {
    if (!sitemapSet.has(canonicalForPath(path))) {
      addFinding(
        findings,
        null,
        'LAUNCH_URL_MISSING',
        `Launch canonical is missing from the sitemap: ${path}.`,
      );
    }
  }

  for (const path of ['/robots.txt', '/sitemap-index.xml']) {
    const { check, response } = await request({
      category: 'discovery-head',
      id: `discovery:head:${path}`,
      method: 'HEAD',
      path,
    });
    requireStatus(findings, check, response, [200]);
    requireType(
      findings,
      check,
      response,
      path === '/robots.txt'
        ? ['text/plain']
        : ['application/xml', 'text/xml'],
    );
  }

  await mapLimit(DISCOVERY_ASSETS, concurrency, async (asset) => {
    const { check, response } = await request({
      category: 'discovery-asset',
      id: `discovery:get:${asset.path}`,
      path: asset.path,
    });
    if (
      !requireStatus(findings, check, response, [200]) ||
      !requireType(findings, check, response, asset.contentTypes)
    ) {
      await discardResponse(response);
      return;
    }
    let body = '';
    try {
      body = await responseText(response, MAX_TEXT_BYTES);
    } catch {
      addFinding(
        findings,
        check,
        'BODY_LIMIT',
        `Discovery asset exceeds the response-size limit: ${asset.path}.`,
      );
      return;
    }
    if (!asset.marker.test(body)) {
      addFinding(
        findings,
        check,
        'DISCOVERY_ASSET_CONTENT',
        `Discovery asset is missing its expected marker: ${asset.path}.`,
      );
    }
  });

  await mapLimit(safeLaunchPaths, concurrency, async (path) => {
    const { check, response } = await request({
      category: 'launch-head',
      id: `launch:head:${path}`,
      method: 'HEAD',
      path,
    });
    requireStatus(findings, check, response, [200]);
    requireType(findings, check, response, ['text/html']);
    if (/\bnoindex\b/i.test(response?.headers.get('x-robots-tag') ?? '')) {
      addFinding(
        findings,
        check,
        'X_ROBOTS_TAG',
        `Unexpected noindex X-Robots-Tag for ${path}.`,
      );
    }
  });

  const imageReferences = new Map();
  const safeSitemapUrls = [...sitemapSet].filter(
    (url) => !excludedSitemapPath(new URL(url).pathname),
  );
  await mapLimit(safeSitemapUrls, concurrency, async (canonical) => {
    const path = new URL(canonical).pathname;
    const { check, response } = await request({
      category: 'canonical-page',
      id: `page:get:${path}`,
      path,
    });
    if (
      !requireStatus(findings, check, response, [200]) ||
      !requireType(findings, check, response, ['text/html'])
    ) {
      await discardResponse(response);
      return;
    }
    let html;
    try {
      html = await responseText(response, MAX_HTML_BYTES);
    } catch {
      addFinding(
        findings,
        check,
        'BODY_LIMIT',
        `HTML exceeds the response-size limit for ${path}.`,
      );
      return;
    }
    const image = pageValidation({
      check,
      expectedCanonical: canonical,
      findings,
      html,
      response,
    });
    if (image) {
      const imagePath = new URL(image).pathname;
      if (!imageReferences.has(imagePath)) imageReferences.set(imagePath, []);
      imageReferences.get(imagePath).push(path);
    }
  });

  await mapLimit([...imageReferences.keys()], concurrency, async (path) => {
    const head = await request({
      category: 'open-graph-image',
      id: `og:head:${path}`,
      method: 'HEAD',
      path,
    });
    requireStatus(findings, head.check, head.response, [200]);
    requireType(findings, head.check, head.response, ['image/png']);
    const contentLength = Number(head.response?.headers.get('content-length'));
    if (Number.isFinite(contentLength) && contentLength > MAX_IMAGE_BYTES) {
      addFinding(
        findings,
        head.check,
        'OG_IMAGE_SIZE',
        `Open Graph image exceeds ${MAX_IMAGE_BYTES} bytes: ${path}.`,
      );
    }

    const get = await request({
      category: 'open-graph-image',
      headers: { range: 'bytes=0-23' },
      id: `og:get:${path}`,
      path,
    });
    if (
      !requireStatus(findings, get.check, get.response, [200, 206]) ||
      !requireType(findings, get.check, get.response, ['image/png'])
    ) {
      await discardResponse(get.response);
      return;
    }
    const dimensions = pngDimensions(await readPrefix(get.response, 24));
    if (!dimensions || dimensions.width !== 1200 || dimensions.height !== 630) {
      addFinding(
        findings,
        get.check,
        'OG_IMAGE_DIMENSIONS',
        `Expected an actual 1200x630 PNG Open Graph image at ${path}.`,
      );
    }
  });

  for (const method of ['GET', 'HEAD']) {
    const { check, response } = await request({
      category: 'missing-route',
      id: `missing:${method.toLowerCase()}`,
      method,
      path: MISSING_PATH,
    });
    requireStatus(findings, check, response, [404]);
    await discardResponse(response);
  }

  await mapLimit(
    CRAWLER_USER_AGENTS.flatMap((userAgent) =>
      CRAWLER_PATHS.map((path) => ({ path, userAgent })),
    ),
    concurrency,
    async ({ path, userAgent }) => {
      const { check, response } = await request({
        category: 'crawler-user-agent',
        id: `crawler:${userAgent}:${path}`,
        path,
        userAgent,
      });
      if (!requireStatus(findings, check, response, [200])) {
        await discardResponse(response);
        return;
      }
      const expectedType =
        path === '/robots.txt'
          ? ['text/plain']
          : path === '/sitemap-index.xml'
            ? ['application/xml', 'text/xml']
            : ['text/html'];
      if (!requireType(findings, check, response, expectedType)) {
        await discardResponse(response);
        return;
      }
      if (expectedType[0] === 'text/html') {
        let html;
        try {
          html = await responseText(response, MAX_HTML_BYTES);
        } catch {
          addFinding(
            findings,
            check,
            'BODY_LIMIT',
            `Crawler response exceeds the HTML limit for ${path}.`,
          );
          return;
        }
        const metadata = parsePageMetadata(html);
        if (
          metadata.canonicals.length !== 1 ||
          normalizeCanonical(metadata.canonicals[0]) !== canonicalForPath(path)
        ) {
          addFinding(
            findings,
            check,
            'CRAWLER_CHALLENGE',
            `Crawler response for ${path} is not the canonical HTML page.`,
          );
        }
      } else {
        await discardResponse(response);
      }
    },
  );

  checks.sort((left, right) => left.id.localeCompare(right.id));
  for (const check of checks) check.ok = check.issues.length === 0;
  const completedAt = new Date();
  const report = {
    schemaVersion: 1,
    startedAt: startedAt.toISOString(),
    completedAt: completedAt.toISOString(),
    durationMs: completedAt.getTime() - startedAt.getTime(),
    baseUrl: safeBaseUrl,
    expectedCanonicalOrigin: PRODUCTION_ORIGIN,
    configuration: {
      concurrency,
      maxAttempts,
      timeoutMs,
      maximumSitemapDocuments: MAX_SITEMAP_DOCUMENTS,
      maximumSitemapUrls: MAX_SITEMAP_URLS,
    },
    summary: {
      ok: findings.length === 0,
      requestChecks: checks.length,
      passedChecks: checks.filter((check) => check.ok).length,
      failedChecks: checks.filter((check) => !check.ok).length,
      findings: findings.length,
      sitemapUrls: sitemapSet.size,
      openGraphImages: imageReferences.size,
    },
    findings,
    checks,
  };
  return report;
}

export function formatProductionDiscoverySummary(report, outputPath) {
  const summary = report.summary;
  return [
    `Production discoverability smoke: ${summary.ok ? 'PASS' : 'FAIL'}`,
    `Base: ${report.baseUrl}`,
    `Requests: ${summary.requestChecks} (${summary.passedChecks} passed, ${summary.failedChecks} failed)`,
    `Coverage: ${summary.sitemapUrls} sitemap URLs, ${summary.openGraphImages} Open Graph images`,
    `Findings: ${summary.findings}`,
    `Evidence: ${outputPath}`,
  ].join('\n');
}

function integerOption(value, option, minimum, maximum) {
  const number = Number(value);
  if (!Number.isInteger(number) || number < minimum || number > maximum) {
    throw new Error(`${option} must be between ${minimum} and ${maximum}.`);
  }
  return number;
}

function parseArguments(argv) {
  const options = {};
  const valueOptions = new Set([
    '--base-url',
    '--concurrency',
    '--max-attempts',
    '--output',
    '--timeout-ms',
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === '--allow-non-production') {
      options.allowNonProduction = true;
      continue;
    }
    if (argument === '--help' || argument === '-h') {
      options.help = true;
      continue;
    }
    if (!valueOptions.has(argument)) {
      throw new Error(`Unknown production discovery option: ${argument}`);
    }
    const value = argv[index + 1];
    if (!value || value.startsWith('--')) {
      throw new Error(`${argument} requires a value.`);
    }
    index += 1;
    options[
      argument.slice(2).replace(/-([a-z])/g, (_match, letter) =>
        letter.toUpperCase(),
      )
    ] = value;
  }
  return options;
}

function defaultOutputPath(startedAt = new Date()) {
  const timestamp = startedAt
    .toISOString()
    .replaceAll(':', '')
    .replace(/\.\d{3}Z$/, 'Z');
  return resolve(
    process.cwd(),
    'artifacts/discovery',
    `production-discovery-${timestamp}.json`,
  );
}

async function main() {
  const options = parseArguments(process.argv.slice(2));
  if (options.help) {
    console.log(`Read-only post-deployment discoverability smoke test

Usage:
  node scripts/check-production-discovery.mjs
  node scripts/check-production-discovery.mjs --base-url <https-origin> \\
    --allow-non-production [--output <evidence.json>]

Options:
  --base-url <origin>           Request target; defaults to https://reinstate.dev
  --allow-non-production       Required acknowledgement for another HTTPS origin
  --concurrency <1-${MAX_CONCURRENCY}>       Default ${DEFAULT_CONCURRENCY}
  --max-attempts <1-${MAX_ATTEMPTS}>        Default ${DEFAULT_MAX_ATTEMPTS}
  --timeout-ms <100-${MAX_TIMEOUT_MS}>  Default ${DEFAULT_TIMEOUT_MS}
  --output <path>              JSON evidence path

The command uses GET and HEAD only. It never authenticates, submits URLs, or
mutates the target.`);
    return;
  }

  const baseUrl = normalizeBaseUrl(options.baseUrl, {
    allowNonProduction: options.allowNonProduction,
  });
  const report = await runProductionDiscoverySmoke({
    baseUrl,
    allowNonProduction: options.allowNonProduction,
    concurrency: integerOption(
      options.concurrency ?? DEFAULT_CONCURRENCY,
      '--concurrency',
      1,
      MAX_CONCURRENCY,
    ),
    maxAttempts: integerOption(
      options.maxAttempts ?? DEFAULT_MAX_ATTEMPTS,
      '--max-attempts',
      1,
      MAX_ATTEMPTS,
    ),
    timeoutMs: integerOption(
      options.timeoutMs ?? DEFAULT_TIMEOUT_MS,
      '--timeout-ms',
      100,
      MAX_TIMEOUT_MS,
    ),
  });
  const outputPath = resolve(options.output ?? defaultOutputPath());
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, `${JSON.stringify(report, null, 2)}\n`, {
    mode: 0o600,
  });
  await chmod(outputPath, 0o600);
  console.log(formatProductionDiscoverySummary(report, outputPath));
  if (!report.summary.ok) process.exitCode = 1;
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(SCRIPT_PATH)) {
  main().catch((error) => {
    console.error(
      `Production discovery configuration error: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
    process.exitCode = 1;
  });
}
