#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { launch } from 'chrome-launcher';
import lighthouse from 'lighthouse';

import { createStaticPreviewServer } from './serve-static-build.mjs';

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const DEFAULT_BUILD_ROOT = resolve(process.cwd(), 'dist/client');
const DEFAULT_OUTPUT_ROOT = resolve(process.cwd(), 'artifacts/lighthouse');

export const DEFAULT_LIGHTHOUSE_ROUTES = [
  '/',
  '/docs',
  '/docs/getting-started',
  '/docs/troubleshooting',
  '/integrations/claude-code',
  '/guides/sync-claude-code-sessions-across-devices',
  '/blog',
  '/blog/why-git-does-not-sync-coding-agent-sessions',
  '/compatibility',
  '/compatibility/agent-version-history',
  '/glossary',
  '/research/encrypted-snapshot-format-v1',
  '/tools/path-mapping-visualizer',
  '/compare/reinstate-vs-manual-session-copying',
  '/use-cases/work-and-personal-computers',
  '/privacy',
];

const REQUIRED_AUDITS = [
  'button-name',
  'canonical',
  'color-contrast',
  'document-title',
  'heading-order',
  'html-has-lang',
  'image-alt',
  'label',
  'link-name',
  'meta-description',
];

function score(value) {
  return typeof value === 'number' ? value : 0;
}

export function evaluateLighthouseReport(report) {
  const errors = [];
  const warnings = [];
  const categoryThresholds = {
    seo: 1,
    accessibility: 0.95,
    'best-practices': 0.95,
    performance: 0.8,
  };

  for (const [category, minimum] of Object.entries(categoryThresholds)) {
    const actual = score(report.categories?.[category]?.score);
    if (actual < minimum) {
      errors.push(
        `${category} score ${(actual * 100).toFixed(0)} is below ${(minimum * 100).toFixed(0)}`,
      );
    }
  }

  for (const auditId of REQUIRED_AUDITS) {
    const audit = report.audits?.[auditId];
    if (audit && audit.score !== null && audit.score < 1) {
      errors.push(`${auditId} failed: ${audit.title ?? auditId}`);
    }
  }

  const lcp = report.audits?.['largest-contentful-paint']?.numericValue;
  if (typeof lcp === 'number' && lcp > 2500) {
    warnings.push(`LCP ${Math.round(lcp)} ms exceeds the 2,500 ms lab target`);
  }

  const cls = report.audits?.['cumulative-layout-shift']?.numericValue;
  if (typeof cls === 'number' && cls > 0.1) {
    warnings.push(`CLS ${cls.toFixed(3)} exceeds the 0.1 lab target`);
  }

  return { errors, warnings };
}

function routeFileName(route) {
  return route === '/'
    ? 'home'
    : route.replace(/^\/+|\/+$/g, '').replaceAll('/', '--');
}

async function listen(server) {
  await new Promise((resolveListen, rejectListen) => {
    server.once('error', rejectListen);
    server.listen(0, '127.0.0.1', resolveListen);
  });
  const address = server.address();
  if (!address || typeof address === 'string') {
    throw new Error('Static preview did not expose a TCP port.');
  }
  return address.port;
}

async function closeServer(server) {
  await new Promise((resolveClose, rejectClose) => {
    server.close((error) => (error ? rejectClose(error) : resolveClose()));
  });
}

export async function runLighthouseSmoke({
  buildRoot = DEFAULT_BUILD_ROOT,
  outputRoot = DEFAULT_OUTPUT_ROOT,
  routes = DEFAULT_LIGHTHOUSE_ROUTES,
} = {}) {
  const server = createStaticPreviewServer({ buildRoot });
  let chrome;
  const results = [];

  try {
    const port = await listen(server);
    chrome = await launch({
      chromeFlags: [
        '--headless=new',
        '--disable-dev-shm-usage',
        '--disable-gpu',
        '--no-sandbox',
      ],
    });
    await mkdir(outputRoot, { recursive: true });

    for (const route of routes) {
      const url = `http://127.0.0.1:${port}${route}`;
      const runner = await lighthouse(url, {
        logLevel: 'error',
        output: 'json',
        onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo'],
        port: chrome.port,
      });
      if (!runner) {
        throw new Error(`Lighthouse returned no report for ${route}.`);
      }

      const report = runner.lhr;
      const evaluation = evaluateLighthouseReport(report);
      const summary = {
        route,
        finalUrl: report.finalDisplayedUrl,
        fetchTime: report.fetchTime,
        scores: Object.fromEntries(
          Object.entries(report.categories).map(([key, value]) => [
            key,
            value.score,
          ]),
        ),
        lcp: report.audits['largest-contentful-paint']?.numericValue ?? null,
        cls: report.audits['cumulative-layout-shift']?.numericValue ?? null,
        ...evaluation,
      };
      results.push(summary);
      await writeFile(
        resolve(outputRoot, `${routeFileName(route)}.json`),
        `${JSON.stringify(report, null, 2)}\n`,
      );
    }

    await writeFile(
      resolve(outputRoot, 'summary.json'),
      `${JSON.stringify(results, null, 2)}\n`,
    );
  } finally {
    if (chrome) {
      await chrome.kill();
    }
    if (server.listening) {
      await closeServer(server);
    }
  }

  return results;
}

export function formatLighthouseSummary(results) {
  const lines = ['Rendered-browser quality audit:'];
  for (const result of results) {
    const values = ['performance', 'accessibility', 'best-practices', 'seo']
      .map((category) => {
        const value = result.scores[category];
        return `${category}=${typeof value === 'number' ? Math.round(value * 100) : 'n/a'}`;
      })
      .join(', ');
    lines.push(
      `- ${result.route}: ${values}; LCP=${Math.round(result.lcp ?? 0)}ms; CLS=${Number(result.cls ?? 0).toFixed(3)}`,
    );
    lines.push(...result.warnings.map((warning) => `  warning: ${warning}`));
    lines.push(...result.errors.map((error) => `  error: ${error}`));
  }
  return lines.join('\n');
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(SCRIPT_PATH)) {
  const results = await runLighthouseSmoke();
  console.log(formatLighthouseSummary(results));
  if (results.some((result) => result.errors.length)) {
    process.exitCode = 1;
  }
}
