import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import { product } from '../data/product';
import { releaseHistory } from '../data/releases';

describe('RSS discovery', () => {
  it('advertises combined, blog, and changelog feeds in the shared head', async () => {
    const layout = await readFile(
      new URL('../layouts/BaseLayout.astro', import.meta.url),
      'utf8',
    );

    for (const feed of ['/rss.xml', '/blog/rss.xml', '/changelog/rss.xml']) {
      expect(layout, feed).toContain(`href="${feed}"`);
    }
    expect(layout.match(/type="application\/rss\+xml"/g)).toHaveLength(3);
  });

  it('keeps the changelog feed grounded in canonical release history', async () => {
    const source = await readFile(
      new URL('../pages/changelog/rss.xml.ts', import.meta.url),
      'utf8',
    );

    expect(releaseHistory[0].version).toBe(product.currentRelease);
    expect(source).toContain('releaseHistory.map');
    expect(source).toContain('/changelog#');
    expect(source).not.toContain('aggregateRating');
  });
});
