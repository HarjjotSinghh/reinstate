import { describe, expect, it } from 'vitest';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { API_ERROR_CODES } from './api-errors';
import { OPENAPI_VERSION, openApiDocument, openApiJson } from './openapi';
import { product } from '../data/product';

type Operation = Record<string, any>;

const HTTP_METHODS = ['get', 'put', 'post', 'delete', 'options', 'head', 'patch', 'trace'];

function operations(document: Record<string, any>): Array<{ path: string; method: string; operation: Operation }> {
  const result: Array<{ path: string; method: string; operation: Operation }> = [];
  for (const [path, item] of Object.entries(document.paths as Record<string, Record<string, Operation>>)) {
    for (const [method, operation] of Object.entries(item)) {
      if (HTTP_METHODS.includes(method)) result.push({ path, method, operation });
    }
  }
  return result;
}

function collectRefs(value: unknown, refs: Set<string>): void {
  if (Array.isArray(value)) {
    value.forEach((entry) => collectRefs(entry, refs));
  } else if (value && typeof value === 'object') {
    for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
      if (key === '$ref' && typeof entry === 'string') refs.add(entry);
      else collectRefs(entry, refs);
    }
  }
}

/** Where each documented path is implemented; keeps the spec honest about what the site serves. */
const IMPLEMENTATIONS: Record<string, string[]> = {
  '/api/waitlist': ['src/pages/api/waitlist.ts'],
  '/api/{path}': ['src/pages/api/[...path].ts'],
  '/compatibility.json': ['src/pages/compatibility.json.ts'],
  '/{page}': ['src/lib/agent-surface/function.ts', 'src/middleware.ts'],
  '/{page}.md': ['src/lib/agent-surface/build.ts'],
  '/llms.txt': ['public/llms.txt'],
  '/llms-full.txt': ['src/lib/agent-surface/build.ts'],
  '/agent-instructions.md': ['public/agent-instructions.md'],
  '/openapi.json': ['src/pages/openapi.json.ts'],
  '/sitemap-index.xml': ['astro.config.mjs'],
  '/rss.xml': ['src/pages/rss.xml.ts'],
  '/install.sh': ['public/install.sh'],
  '/install.ps1': ['public/install.ps1'],
};

describe('openApiDocument', () => {
  const document = openApiDocument() as Record<string, any>;
  const ops = operations(document);

  it('is an OpenAPI 3.1 document that names the product and the production server', () => {
    expect(document.openapi).toBe(OPENAPI_VERSION);
    expect(document.info.title).toContain(product.name);
    expect(document.info.description).toContain(product.name);
    expect(document.info.description).toContain('no hosted Reinstate API');
    expect(document.servers).toEqual([{ url: product.siteUrl, description: 'Production' }]);
    expect(document.externalDocs.url).toBe(`${product.siteUrl}/developers`);
    expect(document.info.license.name).toBe(product.licenseName);
  });

  it('gives every operation a unique operationId, summary, description, tags, and responses', () => {
    expect(ops.length).toBeGreaterThanOrEqual(14);
    const ids = ops.map(({ operation }) => operation.operationId);
    expect(new Set(ids).size).toBe(ids.length);
    const declaredTags = new Set((document.tags as Array<{ name: string }>).map((tag) => tag.name));
    for (const { path, method, operation } of ops) {
      const label = `${method.toUpperCase()} ${path}`;
      expect(operation.operationId, label).toMatch(/^[a-z][A-Za-z0-9]+$/);
      expect(operation.summary?.length, label).toBeGreaterThan(5);
      expect(operation.description?.length, label).toBeGreaterThan(20);
      expect(operation.tags?.length, label).toBeGreaterThan(0);
      for (const tag of operation.tags) expect(declaredTags.has(tag), `${label} tag ${tag}`).toBe(true);
      const responses = Object.entries(operation.responses as Record<string, any>);
      expect(responses.length, label).toBeGreaterThan(0);
      for (const [status, response] of responses) {
        expect(status, label).toMatch(/^[1-5]\d\d$/);
        expect(response.description?.length, `${label} ${status}`).toBeGreaterThan(3);
      }
      expect(responses.some(([, response]) => response.content), `${label} has a typed response`).toBe(true);
    }
  });

  it('types every parameter and documents every path parameter', () => {
    for (const { path, operation } of ops) {
      const pathParams = [...path.matchAll(/\{(\w+)\}/g)].map((match) => match[1]);
      const declared = (operation.parameters ?? []) as Array<Record<string, any>>;
      for (const parameter of declared) {
        expect(parameter.schema?.type, `${path} ${parameter.name}`).toBeDefined();
        expect(parameter.description?.length, `${path} ${parameter.name}`).toBeGreaterThan(5);
        expect(['path', 'query', 'header']).toContain(parameter.in);
        if (parameter.in === 'path') expect(parameter.required).toBe(true);
      }
      for (const name of pathParams) {
        expect(declared.some((parameter) => parameter.in === 'path' && parameter.name === name), `${path} declares ${name}`).toBe(true);
      }
    }
  });

  it('resolves every $ref and declares the error codes the API can emit', () => {
    const refs = new Set<string>();
    collectRefs(document.paths, refs);
    collectRefs(document.components, refs);
    for (const ref of refs) {
      const name = ref.replace('#/components/schemas/', '');
      expect(document.components.schemas[name], ref).toBeDefined();
    }
    expect(document.components.schemas.ErrorResponse.properties.code.enum).toEqual([...API_ERROR_CODES]);
    expect(document.components.schemas.ErrorResponse.required).toEqual(['ok', 'status', 'code', 'error', 'hint', 'docs']);
  });

  it('documents only paths that exist in this repository', () => {
    const root = resolve(import.meta.dirname, '../..');
    for (const path of Object.keys(document.paths)) {
      const files = IMPLEMENTATIONS[path];
      expect(files, `${path} has a known implementation`).toBeDefined();
      for (const file of files!) expect(existsSync(resolve(root, file)), file).toBe(true);
    }
    expect(Object.keys(IMPLEMENTATIONS).sort()).toEqual(Object.keys(document.paths).sort());
  });

  it('says the negotiated page endpoint varies on Accept and can return 406', () => {
    const page = document.paths['/{page}'].get;
    expect(page.responses['200'].headers.Vary.schema.const).toBe('Accept');
    expect(Object.keys(page.responses['200'].content)).toEqual(['text/markdown', 'text/html']);
    expect(page.responses['404'].content['text/markdown']).toBeDefined();
    expect(page.responses['406'].content['text/plain']).toBeDefined();
  });

  it('serializes as stable, pretty JSON ending in a newline', () => {
    const text = openApiJson();
    expect(text.endsWith('}\n')).toBe(true);
    expect(JSON.parse(text)).toEqual(document);
  });
});
