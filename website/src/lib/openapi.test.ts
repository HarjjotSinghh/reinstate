import { describe, expect, it } from 'vitest';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { API_ERROR_CODES } from './api-errors';
import { API_RATE_LIMIT } from './rate-limit';
import { API_VERSION, OPENAPI_VERSION, openApiDocument, openApiJson } from './openapi';
import { product } from '../data/product';

type Operation = Record<string, any>;

const HTTP_METHODS = ['get', 'put', 'post', 'delete', 'options', 'head', 'patch', 'trace'];
const JSON_MEDIA_TYPES = ['application/json', 'application/problem+json', 'application/linkset+json'];

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
  '/api/v1/waitlist': ['src/pages/api/v1/waitlist.ts', 'src/lib/waitlist-api.ts'],
  '/api/waitlist': ['src/pages/api/waitlist.ts', 'src/lib/waitlist-api.ts'],
  '/api/{path}': ['src/pages/api/[...path].ts'],
  '/compatibility.json': ['src/pages/compatibility.json.ts'],
  '/openapi.json': ['src/pages/openapi.json.ts'],
  '/.well-known/api-catalog': ['public/.well-known/api-catalog'],
};

describe('openApiDocument', () => {
  const document = openApiDocument() as Record<string, any>;
  const ops = operations(document);

  it('is an OpenAPI 3.1 document that names the product, the API version, and the production server', () => {
    expect(document.openapi).toBe(OPENAPI_VERSION);
    expect(document.info.version).toBe(API_VERSION);
    expect(document.info['x-reinstate-release']).toBe(product.currentRelease);
    expect(document.info.title).toContain(product.name);
    expect(document.info.description).toContain(product.name);
    expect(document.info.description).toContain('no hosted Reinstate API');
    expect(document.info.contact.url).toBe(`${product.siteUrl}/contact`);
    expect(document.servers).toEqual([{ url: product.siteUrl, description: 'Production' }]);
    expect(document.externalDocs.url).toBe(`${product.siteUrl}/developers`);
    expect(document.info.license.name).toBe(product.licenseName);
  });

  it('declares the versioning, deprecation, and rate-limit policies agents can rely on', () => {
    expect(document['x-api-lifecycle']).toMatchObject({
      deprecationPolicy: `${product.siteUrl}/developers#versioning-and-deprecation`,
      minimumNoticeDays: 90,
      deprecated: [{ path: '/api/waitlist', successor: '/api/v1/waitlist', sunset: null }],
    });
    expect(document['x-api-lifecycle'].versioning).toContain('/api/v1/');
    expect(document['x-rate-limit-policy']).toMatchObject({
      quota: API_RATE_LIMIT.quota,
      windowSeconds: API_RATE_LIMIT.windowSeconds,
      documentation: `${product.siteUrl}/developers#rate-limits`,
    });
    expect(Object.keys(document.paths).some((path) => path.startsWith('/api/v1/'))).toBe(true);
    for (const op of ['get', 'post']) {
      expect(document.paths['/api/waitlist'][op].deprecated).toBe(true);
      expect(document.paths['/api/waitlist'][op].responses['200'].headers.Deprecation).toBeDefined();
      expect(document.paths['/api/v1/waitlist'][op].deprecated).toBeUndefined();
    }
  });

  it('gives every operation a unique operationId, summary, description, tags, and responses', () => {
    expect(ops.length).toBeGreaterThanOrEqual(8);
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
      expect(Object.keys(operation.responses).length, label).toBeGreaterThan(0);
    }
  });

  it('types every response as JSON with an object schema and describes every header', () => {
    for (const { path, method, operation } of ops) {
      for (const [status, response] of Object.entries(operation.responses as Record<string, any>)) {
        const label = `${method.toUpperCase()} ${path} ${status}`;
        expect(status, label).toMatch(/^[1-5]\d\d$/);
        expect(response.description?.length, label).toBeGreaterThan(3);
        const mediaTypes = Object.keys(response.content ?? {});
        expect(mediaTypes.length, `${label} has content`).toBe(1);
        expect(JSON_MEDIA_TYPES, label).toContain(mediaTypes[0]);
        const schema = response.content[mediaTypes[0]!].schema;
        const name = schema.$ref?.replace('#/components/schemas/', '');
        expect(name, `${label} uses a named schema`).toBeDefined();
        expect(document.components.schemas[name].type, `${label} schema ${name}`).toBe('object');
        for (const [headerName, header] of Object.entries(response.headers ?? {})) {
          const resolved = (header as any).$ref ? document.components.headers[(header as any).$ref.replace('#/components/headers/', '')] : header;
          expect(resolved?.description?.length, `${label} header ${headerName}`).toBeGreaterThan(5);
          expect(resolved?.schema?.type, `${label} header ${headerName}`).toBeDefined();
        }
      }
      for (const parameter of (operation.parameters ?? []) as Array<Record<string, any>>) {
        expect(parameter.schema?.type, `${path} ${parameter.name}`).toBeDefined();
        expect(parameter.description?.length, `${path} ${parameter.name}`).toBeGreaterThan(5);
        if (parameter.in === 'path') expect(parameter.required).toBe(true);
      }
      for (const name of [...path.matchAll(/\{(\w+)\}/g)].map((match) => match[1])) {
        expect((operation.parameters ?? []).some((parameter: any) => parameter.in === 'path' && parameter.name === name), `${path} declares ${name}`).toBe(true);
      }
    }
  });

  it('documents the rate-limit headers on every dynamic operation and 429 with Retry-After', () => {
    for (const { path, operation } of ops.filter((op) => op.path.startsWith('/api/'))) {
      for (const [status, response] of Object.entries(operation.responses as Record<string, any>)) {
        for (const header of ['RateLimit-Policy', 'RateLimit', 'RateLimit-Limit', 'RateLimit-Remaining', 'RateLimit-Reset', 'Link']) {
          expect(response.headers?.[header], `${path} ${status} ${header}`).toBeDefined();
        }
      }
      expect(operation.responses['429'], `${path} documents 429`).toBeDefined();
      expect(operation.responses['429'].headers['Retry-After']).toBeDefined();
    }
  });

  it('resolves every $ref and declares the error codes the API can emit', () => {
    const refs = new Set<string>();
    collectRefs(document.paths, refs);
    collectRefs(document.components, refs);
    for (const ref of refs) {
      const [, kind, name] = ref.match(/^#\/components\/(schemas|headers)\/(.+)$/)!;
      expect(document.components[kind!][name!], ref).toBeDefined();
    }
    expect(document.components.schemas.ProblemDetails.properties.code.enum).toEqual([...API_ERROR_CODES]);
    expect(document.components.schemas.ProblemDetails.required).toEqual(['type', 'title', 'status', 'detail', 'code', 'hint', 'docs', 'ok', 'error']);
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

  it('serializes as stable, pretty JSON ending in a newline', () => {
    const text = openApiJson();
    expect(text.endsWith('}\n')).toBe(true);
    expect(JSON.parse(text)).toEqual(document);
  });
});
