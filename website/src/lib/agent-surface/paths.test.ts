import { describe, expect, it } from 'vitest';
import {
  isPagePath,
  markdownFileFor,
  markdownPathFor,
  normalizePagePath,
  pagePathForMarkdown,
  pagePathFromBuildPathname,
} from './paths';
import { PAGE_PATH_SOURCE } from './vercel-routes';

describe('isPagePath', () => {
  it('accepts clean page paths', () => {
    for (const path of ['/', '/docs', '/docs/getting-started', '/integrations/claude-code', '/compare/reinstate-vs-git']) {
      expect(isPagePath(path), path).toBe(true);
    }
  });

  it('rejects assets, runtime namespaces, traversal, and malformed paths', () => {
    for (const path of [
      '/docs/faq.md',
      '/compatibility.json',
      '/rss.xml',
      '/api',
      '/api/waitlist',
      '/_astro/x.css',
      '/_image',
      '/_server-islands/x',
      '/og/home.png',
      '/docs/',
      '/docs//faq',
      '/../etc',
      'docs',
      '/docs?x=1',
      '/docs faq',
      '',
    ]) {
      expect(isPagePath(path), path).toBe(false);
    }
  });

  it('agrees with the Vercel route source pattern', () => {
    const source = new RegExp(PAGE_PATH_SOURCE);
    for (const path of ['/', '/docs', '/docs/getting-started', '/developers']) {
      expect(source.test(path), path).toBe(true);
    }
    for (const path of ['/docs/faq.md', '/api/waitlist', '/api', '/_astro/a.css', '/og/home.png', '/_image', '/rss.xml']) {
      expect(source.test(path), path).toBe(false);
    }
  });
});

describe('normalizePagePath', () => {
  it('adds the leading slash, trims one trailing slash, and rejects the rest', () => {
    expect(normalizePagePath('docs/faq')).toBe('/docs/faq');
    expect(normalizePagePath('/docs/faq/')).toBe('/docs/faq');
    expect(normalizePagePath('/')).toBe('/');
    expect(normalizePagePath('')).toBeNull();
    expect(normalizePagePath(null)).toBeNull();
    expect(normalizePagePath('/docs/faq.md')).toBeNull();
    expect(normalizePagePath('/api/waitlist')).toBeNull();
  });
});

describe('markdown twin mapping', () => {
  it('maps pages to twins and back', () => {
    expect(markdownPathFor('/')).toBe('/index.md');
    expect(markdownPathFor('/docs/faq')).toBe('/docs/faq.md');
    expect(markdownFileFor('/docs/faq')).toBe('docs/faq.md');
    expect(markdownFileFor('/')).toBe('index.md');
    expect(pagePathForMarkdown('/index.md')).toBe('/');
    expect(pagePathForMarkdown('/docs/faq.md')).toBe('/docs/faq');
    expect(pagePathForMarkdown('/docs/faq')).toBeNull();
    expect(pagePathForMarkdown('/api/x.md')).toBeNull();
  });

  it('normalizes Astro build pathnames', () => {
    expect(pagePathFromBuildPathname('')).toBe('/');
    expect(pagePathFromBuildPathname('/')).toBe('/');
    expect(pagePathFromBuildPathname('docs/faq')).toBe('/docs/faq');
    expect(pagePathFromBuildPathname('/docs/faq/')).toBe('/docs/faq');
  });
});
