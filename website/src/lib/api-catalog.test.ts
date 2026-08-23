import { describe, expect, it } from 'vitest';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { openApiDocument } from './openapi';

const root = resolve(import.meta.dirname, '../..');
const catalog = JSON.parse(readFileSync(resolve(root, 'public/.well-known/api-catalog'), 'utf8')) as {
  linkset: Array<Record<string, unknown>>;
};

/** Where each catalog href is implemented; keeps the catalog honest about what the site serves. */
const IMPLEMENTATIONS: Record<string, string> = {
  '/api/v1/waitlist': 'src/pages/api/v1/waitlist.ts',
  '/openapi.json': 'src/pages/openapi.json.ts',
  '/developers': 'src/pages/developers.astro',
  '/llms.txt': 'public/llms.txt',
  '/llms-full.txt': 'src/lib/agent-surface/build.ts',
  '/agent-instructions.md': 'public/agent-instructions.md',
  '/compatibility.json': 'src/pages/compatibility.json.ts',
  '/compatibility': 'src/pages/compatibility.astro',
  '/sitemap-index.xml': 'astro.config.mjs',
  '/': 'src/pages/index.astro',
};

function hrefs(): string[] {
  const result: string[] = [];
  for (const entry of catalog.linkset) {
    for (const [key, value] of Object.entries(entry)) {
      if (key === 'anchor') result.push(value as string);
      else for (const link of value as Array<{ href: string }>) result.push(link.href);
    }
  }
  return result;
}

describe('RFC 9727 API catalog', () => {
  it('is an RFC 9264 linkset whose links all carry href and type', () => {
    expect(Array.isArray(catalog.linkset)).toBe(true);
    expect(catalog.linkset.length).toBeGreaterThan(0);
    for (const entry of catalog.linkset) {
      expect(typeof entry.anchor).toBe('string');
      for (const [relation, targets] of Object.entries(entry)) {
        if (relation === 'anchor') continue;
        expect(['service-desc', 'service-doc', 'service-meta', 'status']).toContain(relation);
        for (const target of targets as Array<Record<string, unknown>>) {
          expect(typeof target.href).toBe('string');
          expect(typeof target.type).toBe('string');
        }
      }
    }
  });

  it('points the API anchor at the OpenAPI description and the developer docs', () => {
    const api = catalog.linkset.find((entry) => entry.anchor === 'https://reinstate.dev/api/v1/waitlist')!;
    expect(api).toBeDefined();
    expect((api['service-desc'] as Array<{ href: string; type: string }>)[0]).toMatchObject({
      href: 'https://reinstate.dev/openapi.json',
      type: 'application/vnd.oai.openapi+json',
    });
    const docs = (api['service-doc'] as Array<{ href: string }>).map((link) => link.href);
    expect(docs).toContain('https://reinstate.dev/developers');
    expect(docs).toContain('https://reinstate.dev/developers#versioning-and-deprecation');
    expect(docs).toContain('https://reinstate.dev/developers#rate-limits');
  });

  it('only links to URLs this repository serves', () => {
    for (const href of hrefs()) {
      const url = new URL(href);
      expect(url.origin).toBe('https://reinstate.dev');
      const file = IMPLEMENTATIONS[url.pathname];
      expect(file, `${url.pathname} has a known implementation`).toBeDefined();
      expect(existsSync(resolve(root, file!)), file).toBe(true);
    }
  });

  it('agrees with the OpenAPI document about the catalog path', () => {
    const document = openApiDocument() as { paths: Record<string, unknown> };
    expect(document.paths['/.well-known/api-catalog']).toBeDefined();
    expect(document.paths['/api/v1/waitlist']).toBeDefined();
  });
});
