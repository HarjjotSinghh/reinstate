/**
 * Contract test for the `vercel.json` header rules, compiled with the same
 * `@vercel/routing-utils` Vercel uses so the path-to-regexp sources are
 * proven to match the page, twin, and text URLs they are meant for.
 */
import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { getTransformedRoutes } from '@vercel/routing-utils';

const root = resolve(import.meta.dirname, '../../..');
const vercelJson = JSON.parse(readFileSync(resolve(root, 'vercel.json'), 'utf8'));

type Route = { src?: string; headers?: Record<string, string>; continue?: boolean; handle?: string };

function headersFor(pathname: string): Record<string, string> {
  const { routes, error } = getTransformedRoutes(vercelJson);
  expect(error).toBeNull();
  const result: Record<string, string> = {};
  for (const route of (routes ?? []) as Route[]) {
    if (!route.src || !route.headers) continue;
    if (new RegExp(route.src, 'i').test(pathname)) {
      for (const [key, value] of Object.entries(route.headers)) result[key.toLowerCase()] = value;
    }
  }
  return result;
}

describe('vercel.json agent headers', () => {
  it('adds Vary: Accept to every clean page URL', () => {
    for (const path of ['/', '/docs', '/docs/getting-started', '/developers', '/integrations/claude-code']) {
      expect(headersFor(path).vary, path).toBe('Accept');
    }
  });

  it('does not add Vary: Accept to assets, feeds, or JSON', () => {
    for (const path of ['/_astro/index.abc123.css', '/rss.xml', '/compatibility.json', '/og/home.png', '/favicon.svg']) {
      expect(headersFor(path).vary, path).toBeUndefined();
    }
  });

  it('serves Markdown twins as text/markdown with nosniff and Vary: Accept', () => {
    for (const path of ['/index.md', '/docs/faq.md', '/agent-instructions.md', '/compare/reinstate-vs-git.md']) {
      const headers = headersFor(path);
      expect(headers['content-type'], path).toBe('text/markdown; charset=utf-8');
      expect(headers['x-content-type-options'], path).toBe('nosniff');
      expect(headers.vary, path).toBe('Accept');
      expect(headers['cache-control'], path).toBe('public, max-age=0, must-revalidate');
    }
  });

  it('serves the RFC 9727 API catalog as a linkset with its profile', () => {
    const headers = headersFor('/.well-known/api-catalog');
    expect(headers['content-type']).toBe('application/linkset+json; profile="https://www.rfc-editor.org/info/rfc9727"; charset=utf-8');
    expect(headers.link).toContain('rel="service-desc"');
  });

  it('keeps llms-full.txt and the installers as plain text', () => {
    expect(headersFor('/llms-full.txt')['content-type']).toBe('text/plain; charset=utf-8');
    expect(headersFor('/install.sh')['content-type']).toBe('text/plain; charset=utf-8');
  });
});
