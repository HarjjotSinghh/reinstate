import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageRoot = new URL('../pages/compare/', import.meta.url);
const ogManifest = readFileSync(
  fileURLToPath(new URL('../data/og-pages.ts', import.meta.url)),
  'utf8',
);

const comparisons = [
  {
    file: 'reinstate-vs-manual-session-copying.astro',
    route: '/compare/reinstate-vs-manual-session-copying',
    requiredSource: '/docs/adapters',
  },
  {
    file: 'reinstate-vs-remote-desktop.astro',
    route: '/compare/reinstate-vs-remote-desktop',
    requiredSource: 'learn.microsoft.com',
  },
  {
    file: 'reinstate-vs-git.astro',
    route: '/compare/reinstate-vs-git',
    requiredSource: 'git-scm.com',
  },
] as const;

function readPage(file: string): string {
  return readFileSync(fileURLToPath(new URL(file, pageRoot)), 'utf8');
}

describe('factual comparison pages', () => {
  it('publishes a WebPage comparison hub with links to every comparison', () => {
    const source = readPage('index.astro');
    expect(source).toContain('article={false}');
    expect(source).toContain('<ComparisonEvidence');

    for (const comparison of comparisons) {
      expect(source).toContain(`href="${comparison.route}"`);
    }
  });

  it.each(comparisons)(
    'keeps $route evidence-backed and outside TechArticle schema',
    ({ file, route, requiredSource }) => {
      const source = readPage(file);
      expect(source).toContain(`path="${route}"`);
      expect(source).toContain('article={false}');
      expect(source).toContain('<ComparisonEvidence');
      expect(source).toContain('limitations={[');
      expect(source).toContain(requiredSource);
      expect(source).not.toContain('aggregateRating');
      expect(source).not.toContain('TechArticle');
    },
  );

  it('registers a unique branded Open Graph image for every route', () => {
    expect(ogManifest).toContain("route: '/compare'");
    for (const comparison of comparisons) {
      expect(ogManifest).toContain(`route: '${comparison.route}'`);
    }
  });

  it('states the same-vendor boundary instead of implying transcript translation', () => {
    const allSource = ['index.astro', ...comparisons.map(({ file }) => file)]
      .map(readPage)
      .join('\n');
    expect(allSource).toContain('same-vendor');
    expect(allSource).not.toContain('seamless cross-agent');
    expect(allSource).not.toContain('all coding agents');
  });
});
