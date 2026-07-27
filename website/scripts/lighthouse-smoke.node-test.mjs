import assert from 'node:assert/strict';
import test from 'node:test';

import {
  DEFAULT_LIGHTHOUSE_ROUTES,
  evaluateLighthouseReport,
  formatLighthouseSummary,
} from './lighthouse-smoke.mjs';

function report({
  accessibility = 1,
  bestPractices = 1,
  cls = 0,
  lcp = 1200,
  performance = 1,
  seo = 1,
} = {}) {
  return {
    categories: {
      accessibility: { score: accessibility },
      'best-practices': { score: bestPractices },
      performance: { score: performance },
      seo: { score: seo },
    },
    audits: {
      ...Object.fromEntries(
        [
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
        ].map((id) => [id, { score: 1, title: id }]),
      ),
      'largest-contentful-paint': { numericValue: lcp },
      'cumulative-layout-shift': { numericValue: cls },
    },
  };
}

test('covers representative discovery and conversion templates', () => {
  assert.deepEqual(DEFAULT_LIGHTHOUSE_ROUTES, [
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
  ]);
});

test('passes healthy reports', () => {
  assert.deepEqual(evaluateLighthouseReport(report()), {
    errors: [],
    warnings: [],
  });
});

test('fails score and semantic regressions while warning on lab web vitals', () => {
  const input = report({
    accessibility: 0.9,
    bestPractices: 0.9,
    cls: 0.2,
    lcp: 3000,
    performance: 0.7,
    seo: 0.9,
  });
  input.audits['image-alt'].score = 0;
  const result = evaluateLighthouseReport(input);

  assert.equal(result.errors.length, 5);
  assert.equal(result.warnings.length, 2);
  assert.match(result.errors.join('\n'), /image-alt failed/);
  assert.match(result.warnings.join('\n'), /LCP/);
  assert.match(result.warnings.join('\n'), /CLS/);
});

test('formats a compact evidence summary', () => {
  const output = formatLighthouseSummary([
    {
      route: '/',
      scores: {
        performance: 0.98,
        accessibility: 1,
        'best-practices': 1,
        seo: 1,
      },
      lcp: 1234,
      cls: 0,
      warnings: [],
      errors: [],
    },
  ]);
  assert.match(output, /performance=98/);
  assert.match(output, /LCP=1234ms/);
});
