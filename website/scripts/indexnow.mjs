import { createHash } from 'node:crypto';
import { access, mkdir, readFile, writeFile } from 'node:fs/promises';
import { basename, dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const SITE_ORIGIN = 'https://reinstate.dev';
export const INDEXNOW_ENDPOINT = 'https://api.indexnow.org/indexnow';
export const INDEXNOW_KEY_PATTERN = /^[A-Za-z0-9-]{8,128}$/;
export const DEFAULT_BATCH_SIZE = 100;
export const MAX_BATCH_SIZE = 1_000;
export const DEFAULT_MAX_ATTEMPTS = 3;
export const MAX_RETRY_DELAY_MS = 60_000;
export const PLAN_SCHEMA_VERSION = 1;

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const DEFAULT_CURRENT_SITEMAP = resolve(
  dirname(SCRIPT_PATH),
  '../dist/client/sitemap-index.xml',
);
const ALLOWED_CHANGE_KEYS = new Set([
  'deleted',
  'recanonicalized',
  'updated',
]);

function xmlDecode(value) {
  return value
    .replaceAll('&amp;', '&')
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
    .replaceAll('&quot;', '"')
    .replaceAll('&apos;', "'")
    .replace(/&#(\d+);/g, (_match, codePoint) =>
      String.fromCodePoint(Number(codePoint)),
    )
    .replace(/&#x([0-9a-f]+);/gi, (_match, codePoint) =>
      String.fromCodePoint(Number.parseInt(codePoint, 16)),
    );
}

function tagValue(block, tag) {
  const match = block.match(
    new RegExp(`<${tag}(?:\\s[^>]*)?>([\\s\\S]*?)<\\/${tag}>`, 'i'),
  );
  return match ? xmlDecode(match[1].trim()) : undefined;
}

function tagBlocks(xml, tag) {
  return [
    ...xml.matchAll(
      new RegExp(`<${tag}(?:\\s[^>]*)?>([\\s\\S]*?)<\\/${tag}>`, 'gi'),
    ),
  ].map((match) => match[1]);
}

export function parseSitemapXml(xml) {
  if (/<sitemapindex(?:\s|>)/i.test(xml)) {
    const children = tagBlocks(xml, 'sitemap')
      .map((block) => tagValue(block, 'loc'))
      .filter(Boolean);
    if (children.length === 0) {
      throw new Error('Sitemap index contains no <sitemap><loc> entries.');
    }
    return { kind: 'index', children };
  }

  if (/<urlset(?:\s|>)/i.test(xml)) {
    const entries = tagBlocks(xml, 'url').map((block) => {
      const loc = tagValue(block, 'loc');
      if (!loc) {
        throw new Error('Sitemap URL entry is missing <loc>.');
      }
      return { loc, lastmod: tagValue(block, 'lastmod') };
    });
    return { kind: 'urls', entries };
  }

  throw new Error('Expected a sitemap index or URL set XML document.');
}

function isHttpSource(source) {
  return /^https:\/\//i.test(source);
}

async function fetchText(source, fetchImpl, timeoutMs) {
  const response = await fetchImpl(source, {
    headers: {
      accept: 'application/xml,text/xml,text/plain;q=0.9,*/*;q=0.1',
      'user-agent': 'Reinstate-IndexNow/1.0',
    },
    redirect: 'error',
    signal: AbortSignal.timeout(timeoutMs),
  });
  if (!response.ok) {
    throw new Error(`Sitemap request returned HTTP ${response.status}: ${source}`);
  }
  return response.text();
}

async function localChildSource(parentSource, childLocation) {
  let filename;
  try {
    filename = basename(new URL(childLocation).pathname);
  } catch {
    filename = basename(childLocation);
  }
  if (!filename) {
    throw new Error(`Cannot resolve child sitemap location: ${childLocation}`);
  }
  const candidate = resolve(dirname(parentSource), filename);
  try {
    await access(candidate);
  } catch {
    throw new Error(
      `Local sitemap index references missing sibling file: ${candidate}`,
    );
  }
  return candidate;
}

function mergeSitemapEntries(target, source, sourceName) {
  for (const [url, lastmod] of source) {
    const existing = target.get(url);
    if (
      existing !== undefined &&
      lastmod !== undefined &&
      existing !== lastmod
    ) {
      throw new Error(
        `Sitemap contains conflicting lastmod values for ${url} (${sourceName}).`,
      );
    }
    target.set(url, existing ?? lastmod);
  }
}

export async function loadSitemap(
  source,
  {
    fetchImpl = globalThis.fetch,
    timeoutMs = 15_000,
    maxDepth = 8,
    siteOrigin = SITE_ORIGIN,
  } = {},
) {
  const seen = new Set();

  async function visit(currentSource, depth) {
    if (depth > maxDepth) {
      throw new Error(`Sitemap recursion exceeded ${maxDepth} levels.`);
    }
    if (/^http:\/\//i.test(currentSource)) {
      throw new Error(`Remote sitemap sources must use HTTPS: ${currentSource}`);
    }
    if (isHttpSource(currentSource)) {
      const sourceUrl = new URL(currentSource);
      const allowedOrigin = new URL(siteOrigin).origin;
      if (
        sourceUrl.origin !== allowedOrigin ||
        sourceUrl.username ||
        sourceUrl.password
      ) {
        throw new Error(
          `Remote sitemap sources must stay on ${allowedOrigin}: ${currentSource}`,
        );
      }
    }

    const sourceKey = isHttpSource(currentSource)
      ? new URL(currentSource).toString()
      : resolve(currentSource);
    if (seen.has(sourceKey)) {
      throw new Error(`Recursive or duplicate sitemap reference: ${sourceKey}`);
    }
    seen.add(sourceKey);

    const xml = isHttpSource(currentSource)
      ? await fetchText(currentSource, fetchImpl, timeoutMs)
      : await readFile(currentSource, 'utf8');
    const parsed = parseSitemapXml(xml);

    if (parsed.kind === 'urls') {
      return new Map(
        parsed.entries.map(({ loc, lastmod }) => [
          normalizeCanonicalUrl(loc, siteOrigin),
          lastmod,
        ]),
      );
    }

    const entries = new Map();
    for (const childLocation of parsed.children) {
      const childSource = isHttpSource(currentSource)
        ? new URL(childLocation, currentSource).toString()
        : await localChildSource(currentSource, childLocation);
      const childEntries = await visit(childSource, depth + 1);
      mergeSitemapEntries(entries, childEntries, childSource);
    }
    return entries;
  }

  return visit(source, 0);
}

export async function loadPreviousSitemap(
  source,
  { allowMissing = false, ...options } = {},
) {
  try {
    return await loadSitemap(source, options);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    if (
      allowMissing &&
      isHttpSource(source) &&
      /Sitemap request returned HTTP (?:404|410):/.test(message)
    ) {
      return new Map();
    }
    throw error;
  }
}

export function normalizeCanonicalUrl(input, siteOrigin = SITE_ORIGIN) {
  if (typeof input !== 'string' || input.trim() === '') {
    throw new Error('Canonical URL must be a non-empty string.');
  }

  const base = new URL(siteOrigin);
  const url = new URL(input.trim(), base);
  if (url.origin !== base.origin || url.protocol !== 'https:') {
    throw new Error(`IndexNow URL must use ${base.origin}: ${input}`);
  }
  if (url.username || url.password || url.search || url.hash) {
    throw new Error(
      `IndexNow canonical URLs cannot contain credentials, a query, or a fragment: ${input}`,
    );
  }

  const pathname =
    url.pathname === '/' ? '/' : url.pathname.replace(/\/+$/, '');
  return new URL(pathname, base).toString();
}

function emptyChanges() {
  return { deleted: [], recanonicalized: [], updated: [] };
}

export async function loadChangesFile(path, siteOrigin = SITE_ORIGIN) {
  if (!path) return emptyChanges();
  const raw = JSON.parse(await readFile(path, 'utf8'));
  if (!raw || Array.isArray(raw) || typeof raw !== 'object') {
    throw new Error('IndexNow changes file must contain a JSON object.');
  }

  for (const key of Object.keys(raw)) {
    if (!ALLOWED_CHANGE_KEYS.has(key)) {
      throw new Error(`Unknown IndexNow changes field: ${key}`);
    }
  }

  const deleted = raw.deleted ?? [];
  const updated = raw.updated ?? [];
  const recanonicalized = raw.recanonicalized ?? [];
  if (
    !Array.isArray(deleted) ||
    !Array.isArray(updated) ||
    !Array.isArray(recanonicalized)
  ) {
    throw new Error(
      'deleted, updated, and recanonicalized must each be JSON arrays.',
    );
  }

  return {
    deleted: deleted.map((url) => normalizeCanonicalUrl(url, siteOrigin)),
    updated: updated.map((url) => normalizeCanonicalUrl(url, siteOrigin)),
    recanonicalized: recanonicalized.map((change, index) => {
      if (
        !change ||
        Array.isArray(change) ||
        typeof change !== 'object' ||
        typeof change.from !== 'string' ||
        typeof change.to !== 'string'
      ) {
        throw new Error(
          `recanonicalized[${index}] must contain string "from" and "to" fields.`,
        );
      }
      const unknown = Object.keys(change).filter(
        (key) => key !== 'from' && key !== 'to',
      );
      if (unknown.length > 0) {
        throw new Error(
          `recanonicalized[${index}] has unknown fields: ${unknown.join(', ')}`,
        );
      }
      return {
        from: normalizeCanonicalUrl(change.from, siteOrigin),
        to: normalizeCanonicalUrl(change.to, siteOrigin),
      };
    }),
  };
}

function sorted(values) {
  return [...values].sort((left, right) => left.localeCompare(right));
}

function addReason(reasons, url, reason) {
  if (!reasons.has(url)) reasons.set(url, new Set());
  reasons.get(url).add(reason);
}

function digestPlan(site, entries) {
  return createHash('sha256')
    .update(JSON.stringify({ site, entries }))
    .digest('hex');
}

export function buildIndexNowPlan({
  current,
  previous,
  changes = emptyChanges(),
  generatedAt = new Date().toISOString(),
  siteOrigin = SITE_ORIGIN,
}) {
  if (!(current instanceof Map) || !(previous instanceof Map)) {
    throw new Error('Current and previous sitemap inventories must be Maps.');
  }

  const currentUrls = new Set(current.keys());
  const previousUrls = new Set(previous.keys());
  const added = new Set();
  const removed = new Set();
  const modified = new Set();
  const explicitUpdated = new Set(changes.updated ?? []);
  const explicitDeleted = new Set(changes.deleted ?? []);
  const recanonicalized = changes.recanonicalized ?? [];
  const reasons = new Map();

  for (const url of currentUrls) {
    if (!previousUrls.has(url)) {
      added.add(url);
      addReason(reasons, url, 'sitemap-added');
      continue;
    }
    const currentLastmod = current.get(url);
    const previousLastmod = previous.get(url);
    if (
      (currentLastmod || previousLastmod) &&
      currentLastmod !== previousLastmod
    ) {
      modified.add(url);
      addReason(reasons, url, 'sitemap-lastmod-changed');
    }
  }

  for (const url of previousUrls) {
    if (!currentUrls.has(url)) {
      removed.add(url);
      addReason(reasons, url, 'sitemap-removed');
    }
  }

  for (const url of explicitUpdated) {
    if (!currentUrls.has(url)) {
      throw new Error(`Explicitly updated URL is not in the current sitemap: ${url}`);
    }
    addReason(reasons, url, 'explicitly-updated');
  }

  for (const url of explicitDeleted) {
    if (currentUrls.has(url)) {
      throw new Error(`Explicitly deleted URL is still in the current sitemap: ${url}`);
    }
    addReason(reasons, url, 'explicitly-deleted');
  }

  for (const { from, to } of recanonicalized) {
    if (from === to) {
      throw new Error(`Recanonicalization source and destination are identical: ${from}`);
    }
    if (currentUrls.has(from)) {
      throw new Error(
        `Recanonicalization source is still in the current sitemap: ${from}`,
      );
    }
    if (!currentUrls.has(to)) {
      throw new Error(
        `Recanonicalization destination is not in the current sitemap: ${to}`,
      );
    }
    addReason(reasons, from, 'recanonicalized-from');
    addReason(reasons, to, 'recanonicalized-to');
  }

  const entries = sorted(reasons.keys()).map((url) => ({
    url,
    reasons: sorted(reasons.get(url)),
  }));
  const normalizedSite = new URL(siteOrigin).origin;

  return {
    schemaVersion: PLAN_SCHEMA_VERSION,
    generatedAt,
    site: normalizedSite,
    currentUrlCount: currentUrls.size,
    previousUrlCount: previousUrls.size,
    changes: {
      added: sorted(added),
      removed: sorted(removed),
      modified: sorted(modified),
      explicitlyUpdated: sorted(explicitUpdated),
      explicitlyDeleted: sorted(explicitDeleted),
      recanonicalized: [...recanonicalized].sort((left, right) =>
        left.from.localeCompare(right.from),
      ),
    },
    entries,
    urlList: entries.map(({ url }) => url),
    planDigest: digestPlan(normalizedSite, entries),
  };
}

export function validateIndexNowPlan(
  plan,
  {
    now = Date.now(),
    maxAgeMs,
    siteOrigin = SITE_ORIGIN,
  } = {},
) {
  if (!plan || Array.isArray(plan) || typeof plan !== 'object') {
    throw new Error('IndexNow plan must be a JSON object.');
  }
  if (plan.schemaVersion !== PLAN_SCHEMA_VERSION) {
    throw new Error(
      `Unsupported IndexNow plan schema version: ${plan.schemaVersion}`,
    );
  }
  if (plan.site !== new URL(siteOrigin).origin) {
    throw new Error(`IndexNow plan site must be ${new URL(siteOrigin).origin}.`);
  }
  if (!Array.isArray(plan.entries) || !Array.isArray(plan.urlList)) {
    throw new Error('IndexNow plan must include entries and urlList arrays.');
  }

  const normalizedEntries = plan.entries.map((entry, index) => {
    if (
      !entry ||
      typeof entry !== 'object' ||
      typeof entry.url !== 'string' ||
      !Array.isArray(entry.reasons) ||
      entry.reasons.some((reason) => typeof reason !== 'string')
    ) {
      throw new Error(`Invalid IndexNow plan entry at index ${index}.`);
    }
    return {
      url: normalizeCanonicalUrl(entry.url, siteOrigin),
      reasons: sorted(new Set(entry.reasons)),
    };
  });
  const expectedUrls = normalizedEntries.map(({ url }) => url);
  if (
    expectedUrls.length !== new Set(expectedUrls).size ||
    JSON.stringify(expectedUrls) !== JSON.stringify(plan.urlList)
  ) {
    throw new Error('IndexNow plan URL list is duplicated, reordered, or inconsistent.');
  }
  if (digestPlan(plan.site, normalizedEntries) !== plan.planDigest) {
    throw new Error('IndexNow plan digest does not match its reviewed URL entries.');
  }

  const generatedAt = Date.parse(plan.generatedAt);
  if (!Number.isFinite(generatedAt)) {
    throw new Error('IndexNow plan generatedAt must be a valid timestamp.');
  }
  if (generatedAt > now + 5 * 60_000) {
    throw new Error('IndexNow plan timestamp is unexpectedly in the future.');
  }
  if (maxAgeMs !== undefined && now - generatedAt > maxAgeMs) {
    throw new Error('IndexNow plan is older than the permitted submission window.');
  }

  return plan;
}

function chunks(values, size) {
  const result = [];
  for (let index = 0; index < values.length; index += size) {
    result.push(values.slice(index, index + size));
  }
  return result;
}

function retryDelay(response, attempt, baseDelayMs, now) {
  const retryAfter = response?.headers?.get?.('retry-after');
  if (retryAfter) {
    const seconds = Number(retryAfter);
    if (Number.isFinite(seconds) && seconds >= 0) {
      return Math.min(seconds * 1_000, MAX_RETRY_DELAY_MS);
    }
    const date = Date.parse(retryAfter);
    if (Number.isFinite(date)) {
      return Math.min(Math.max(0, date - now()), MAX_RETRY_DELAY_MS);
    }
  }
  return Math.min(baseDelayMs * 2 ** (attempt - 1), MAX_RETRY_DELAY_MS);
}

function safeLog(log, event) {
  log({
    timestamp: new Date().toISOString(),
    ...event,
  });
}

export async function submitIndexNowPlan(
  plan,
  {
    key,
    fetchImpl = globalThis.fetch,
    sleep = (milliseconds) =>
      new Promise((resolveSleep) => setTimeout(resolveSleep, milliseconds)),
    log = (event) => console.log(JSON.stringify(event)),
    batchSize = DEFAULT_BATCH_SIZE,
    maxAttempts = DEFAULT_MAX_ATTEMPTS,
    baseDelayMs = 1_000,
    interBatchDelayMs = 1_000,
    requestTimeoutMs = 15_000,
    now = Date.now,
    endpoint = INDEXNOW_ENDPOINT,
  } = {},
) {
  if (plan.urlList.length === 0) {
    safeLog(log, {
      event: 'indexnow-noop',
      planDigest: plan.planDigest,
      urlCount: 0,
    });
    return {
      ok: true,
      softFailed: false,
      acceptedBatches: 0,
      submittedUrls: 0,
      totalBatches: 0,
    };
  }
  if (!INDEXNOW_KEY_PATTERN.test(key ?? '')) {
    throw new Error(
      'INDEXNOW_KEY must be 8–128 letters, numbers, or dashes and must come from the environment.',
    );
  }
  if (!Number.isInteger(batchSize) || batchSize < 1 || batchSize > MAX_BATCH_SIZE) {
    throw new Error(`IndexNow batch size must be between 1 and ${MAX_BATCH_SIZE}.`);
  }
  if (
    !Number.isInteger(maxAttempts) ||
    maxAttempts < 1 ||
    maxAttempts > 5
  ) {
    throw new Error('IndexNow max attempts must be between 1 and 5.');
  }
  if (
    baseDelayMs < 0 ||
    baseDelayMs > MAX_RETRY_DELAY_MS ||
    interBatchDelayMs < 0 ||
    interBatchDelayMs > 30_000
  ) {
    throw new Error('IndexNow retry or batch delay is outside the safe limit.');
  }

  const site = new URL(plan.site);
  const keyLocation = new URL(`${encodeURIComponent(key)}.txt`, `${site.origin}/`);
  let proofResponse;
  try {
    proofResponse = await fetchImpl(keyLocation, {
      headers: {
        accept: 'text/plain',
        'user-agent': 'Reinstate-IndexNow/1.0',
      },
      redirect: 'error',
      signal: AbortSignal.timeout(requestTimeoutMs),
    });
    const proof = proofResponse.ok ? (await proofResponse.text()).trim() : '';
    if (proofResponse.status !== 200 || proof !== key) {
      safeLog(log, {
        event: 'indexnow-key-proof-failed',
        planDigest: plan.planDigest,
        status: proofResponse.status,
      });
      return {
        ok: false,
        softFailed: true,
        reason: 'key-proof-failed',
        acceptedBatches: 0,
        submittedUrls: 0,
        totalBatches: 0,
      };
    }
  } catch (error) {
    safeLog(log, {
      event: 'indexnow-key-proof-failed',
      planDigest: plan.planDigest,
      status: null,
      errorType: error ? 'request-error' : 'unknown-error',
    });
    return {
      ok: false,
      softFailed: true,
      reason: 'key-proof-unavailable',
      acceptedBatches: 0,
      submittedUrls: 0,
      totalBatches: 0,
    };
  }

  const batches = chunks(plan.urlList, batchSize);
  let acceptedBatches = 0;
  let submittedUrls = 0;

  for (const [batchIndex, urlList] of batches.entries()) {
    let accepted = false;
    for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
      let response;
      let networkError;
      try {
        response = await fetchImpl(endpoint, {
          method: 'POST',
          headers: {
            accept: 'application/json,text/plain;q=0.9,*/*;q=0.1',
            'content-type': 'application/json; charset=utf-8',
            'user-agent': 'Reinstate-IndexNow/1.0',
          },
          body: JSON.stringify({
            host: site.host,
            key,
            keyLocation: keyLocation.toString(),
            urlList,
          }),
          redirect: 'error',
          signal: AbortSignal.timeout(requestTimeoutMs),
        });
      } catch (error) {
        networkError = error;
      }

      const status = response?.status ?? null;
      accepted = status === 200 || status === 202;
      safeLog(log, {
        event: 'indexnow-batch-response',
        planDigest: plan.planDigest,
        batch: batchIndex + 1,
        totalBatches: batches.length,
        urlCount: urlList.length,
        attempt,
        status,
        accepted,
        errorType: networkError ? 'request-error' : undefined,
      });
      if (accepted) break;

      const retryable =
        networkError !== undefined ||
        status === 429 ||
        (status !== null && status >= 500);
      if (!retryable || attempt === maxAttempts) break;
      await sleep(retryDelay(response, attempt, baseDelayMs, now));
    }

    if (!accepted) {
      safeLog(log, {
        event: 'indexnow-soft-failure',
        planDigest: plan.planDigest,
        failedBatch: batchIndex + 1,
        acceptedBatches,
        totalBatches: batches.length,
      });
      return {
        ok: false,
        softFailed: true,
        reason: 'batch-not-accepted',
        acceptedBatches,
        submittedUrls,
        totalBatches: batches.length,
      };
    }

    acceptedBatches += 1;
    submittedUrls += urlList.length;
    if (batchIndex < batches.length - 1 && interBatchDelayMs > 0) {
      await sleep(interBatchDelayMs);
    }
  }

  safeLog(log, {
    event: 'indexnow-complete',
    planDigest: plan.planDigest,
    acceptedBatches,
    submittedUrls,
    totalBatches: batches.length,
  });
  return {
    ok: true,
    softFailed: false,
    acceptedBatches,
    submittedUrls,
    totalBatches: batches.length,
  };
}

function positiveInteger(value, option) {
  const number = Number(value);
  if (!Number.isInteger(number) || number < 0) {
    throw new Error(`${option} requires a non-negative integer.`);
  }
  return number;
}

export function parseArguments(argv) {
  const options = {
    current: DEFAULT_CURRENT_SITEMAP,
    submit: false,
  };
  const valueOptions = new Set([
    '--batch-delay-ms',
    '--batch-size',
    '--base-delay-ms',
    '--changes',
    '--current',
    '--max-attempts',
    '--max-plan-age-hours',
    '--output',
    '--plan',
    '--previous',
  ]);

  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === '--help' || argument === '-h') {
      options.help = true;
      continue;
    }
    if (argument === '--submit') {
      options.submit = true;
      continue;
    }
    if (argument === '--allow-missing-previous') {
      options.allowMissingPrevious = true;
      continue;
    }
    if (!valueOptions.has(argument)) {
      throw new Error(`Unknown IndexNow option: ${argument}`);
    }
    const value = argv[index + 1];
    if (value === undefined || value.startsWith('--')) {
      throw new Error(`${argument} requires a value.`);
    }
    index += 1;
    const key = argument.slice(2).replace(/-([a-z])/g, (_match, letter) =>
      letter.toUpperCase(),
    );
    options[key] = value;
  }
  return options;
}

function helpText() {
  return `Reinstate IndexNow planner and soft-fail submitter

Plan only (default; never reads a key or submits):
  node scripts/indexnow.mjs --current <sitemap> --previous <sitemap> \\
    [--changes <json>] [--output <plan.json>]

Submit a previously reviewed plan:
  INDEXNOW_KEY=<secret> node scripts/indexnow.mjs --plan <plan.json> --submit

Options:
  --current <path-or-url>       Current generated sitemap index
  --previous <path-or-url>      Previous production or saved sitemap index
  --allow-missing-previous     Treat an explicit remote 404/410 as first-deploy empty state
  --changes <json>              Explicit updated/deleted/recanonicalized URLs
  --output <plan.json>          Save the secret-free reviewed plan
  --plan <plan.json>            Reviewed plan to submit
  --submit                      Opt in to network submission
  --batch-size <1-${MAX_BATCH_SIZE}>       Default ${DEFAULT_BATCH_SIZE}
  --max-attempts <1-5>          Default ${DEFAULT_MAX_ATTEMPTS}
  --base-delay-ms <0-60000>     Default 1000
  --batch-delay-ms <0-30000>    Default 1000
  --max-plan-age-hours <hours>  Default 48

The key is accepted only through INDEXNOW_KEY. It is never accepted as a CLI
argument, written to a plan, or included in structured logs.`;
}

async function writePlan(path, plan) {
  const destination = resolve(path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, `${JSON.stringify(plan, null, 2)}\n`, {
    mode: 0o600,
  });
}

async function main() {
  const options = parseArguments(process.argv.slice(2));
  if (options.help) {
    console.log(helpText());
    return;
  }

  if (options.submit) {
    if (!options.plan) {
      throw new Error('--submit requires a reviewed --plan file.');
    }
    if (
      options.previous ||
      options.changes ||
      options.output ||
      options.allowMissingPrevious ||
      options.current !== DEFAULT_CURRENT_SITEMAP
    ) {
      throw new Error(
        'Submission accepts only --plan and submission-tuning options; generate and review the plan separately.',
      );
    }
    const maxPlanAgeHours = positiveInteger(
      options.maxPlanAgeHours ?? 48,
      '--max-plan-age-hours',
    );
    const plan = validateIndexNowPlan(
      JSON.parse(await readFile(options.plan, 'utf8')),
      { maxAgeMs: maxPlanAgeHours * 60 * 60 * 1_000 },
    );
    const result = await submitIndexNowPlan(plan, {
      key: process.env.INDEXNOW_KEY,
      batchSize: positiveInteger(
        options.batchSize ?? DEFAULT_BATCH_SIZE,
        '--batch-size',
      ),
      maxAttempts: positiveInteger(
        options.maxAttempts ?? DEFAULT_MAX_ATTEMPTS,
        '--max-attempts',
      ),
      baseDelayMs: positiveInteger(
        options.baseDelayMs ?? 1_000,
        '--base-delay-ms',
      ),
      interBatchDelayMs: positiveInteger(
        options.batchDelayMs ?? 1_000,
        '--batch-delay-ms',
      ),
    });
    console.log(
      JSON.stringify({
        mode: 'submit',
        planDigest: plan.planDigest,
        ...result,
      }),
    );
    return;
  }

  if (options.plan) {
    throw new Error('--plan is only valid together with --submit.');
  }
  if (!options.previous) {
    throw new Error(
      'Planning requires --previous so unchanged URLs are never submitted accidentally.',
    );
  }

  const [current, previous, changes] = await Promise.all([
    loadSitemap(options.current),
    loadPreviousSitemap(options.previous, {
      allowMissing: options.allowMissingPrevious,
    }),
    loadChangesFile(options.changes),
  ]);
  const plan = buildIndexNowPlan({ current, previous, changes });
  validateIndexNowPlan(plan);
  if (options.output) await writePlan(options.output, plan);
  console.log(
    JSON.stringify(
      {
        mode: 'dry-run',
        networkSubmissionAttempted: false,
        output: options.output ? resolve(options.output) : null,
        plan,
      },
      null,
      2,
    ),
  );
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(SCRIPT_PATH)) {
  main().catch((error) => {
    console.error(
      `IndexNow configuration error: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
    process.exitCode = 1;
  });
}
