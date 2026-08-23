import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import type { APIContext } from 'astro';
import { resetWaitlistClient } from './waitlist-db';
import {
  API_ERROR_CODES,
  API_ERROR_TITLES,
  API_LINK_HEADER,
  PROBLEM_CONTENT_TYPE,
  apiError,
  apiNotFound,
  methodNotAllowed,
  problemDetails,
} from './api-errors';
import { WAITLIST_LEGACY_DEPRECATED_AT, WAITLIST_METHODS, deprecationHeaders } from './waitlist-api';
import { ALL as v1Fallback, GET as v1Get, POST as v1Post } from '../pages/api/v1/waitlist';
import { ALL as legacyFallback, GET as legacyGet, POST as legacyPost } from '../pages/api/waitlist';
import { ALL as apiCatchAll } from '../pages/api/[...path]';

function context(request: Request): APIContext {
  return { request, url: new URL(request.url) } as unknown as APIContext;
}

async function json(response: Response, type = 'application/json; charset=utf-8'): Promise<Record<string, unknown>> {
  expect(response.headers.get('Content-Type')).toBe(type);
  return (await response.json()) as Record<string, unknown>;
}

const problem = (response: Response) => json(response, PROBLEM_CONTENT_TYPE);

describe('api error helpers', () => {
  it('produce RFC 9457 problem details with the legacy members', async () => {
    const body = problemDetails(400, 'invalid_email', 'Bad.', 'Fix it.', { instance: '/api/v1/waitlist' });
    expect(body).toEqual({
      type: 'https://reinstate.dev/developers#error-invalid-email',
      title: API_ERROR_TITLES.invalid_email,
      status: 400,
      detail: 'Bad.',
      instance: '/api/v1/waitlist',
      code: 'invalid_email',
      hint: 'Fix it.',
      docs: 'https://reinstate.dev/developers#errors',
      ok: false,
      error: 'Bad.',
    });
    const response = apiError(503, 'storage_unavailable', 'Down.', 'Retry.', { headers: { 'Retry-After': '60' } });
    expect(response.status).toBe(503);
    expect(response.headers.get('Retry-After')).toBe('60');
    expect(response.headers.get('Cache-Control')).toBe('no-store');
    expect(response.headers.get('Link')).toBe(API_LINK_HEADER);
    expect((await problem(response)).code).toBe('storage_unavailable');
    expect(API_ERROR_CODES).toContain('rate_limited');
    for (const code of API_ERROR_CODES) expect(API_ERROR_TITLES[code].length).toBeGreaterThan(5);
  });

  it('advertises the OpenAPI description, API catalog, docs, and deprecation policy in Link', () => {
    expect(API_LINK_HEADER).toContain('<https://reinstate.dev/openapi.json>; rel="service-desc"; type="application/vnd.oai.openapi+json"');
    expect(API_LINK_HEADER).toContain('<https://reinstate.dev/.well-known/api-catalog>; rel="api-catalog"');
    expect(API_LINK_HEADER).toContain('<https://reinstate.dev/developers>; rel="service-doc"');
    expect(API_LINK_HEADER).toContain('rel="deprecation"');
  });

  it('405 carries an Allow header', async () => {
    const response = methodNotAllowed('PUT', ['GET', 'POST'], '/api/v1/waitlist');
    expect(response.status).toBe(405);
    expect(response.headers.get('Allow')).toBe('GET, POST');
    const body = await problem(response);
    expect(body.code).toBe('method_not_allowed');
    expect(body.detail).toContain('PUT');
    expect(body.hint).toContain('GET, POST');
    expect(body.instance).toBe('/api/v1/waitlist');
  });

  it('404 points back at the OpenAPI document and the v1 path', async () => {
    const body = await problem(apiNotFound('/api/nope'));
    expect(body.code).toBe('not_found');
    expect(body.detail).toContain('/api/nope');
    expect(body.hint).toContain('openapi.json');
    expect(body.hint).toContain('/api/v1/waitlist');
  });
});

describe('/api/{path} catch-all', () => {
  it('returns a JSON problem 404 for any method, including unknown v1 paths', async () => {
    for (const [method, path] of [['GET', '/api/does-not-exist'], ['POST', '/api/v1/nope'], ['DELETE', '/api/v2/waitlist']]) {
      const response = await apiCatchAll(context(new Request(`https://reinstate.dev${path}`, { method })));
      expect(response.status).toBe(404);
      const body = await problem(response);
      expect(body.ok).toBe(false);
      expect(body.code).toBe('not_found');
      expect(body.instance).toBe(path);
    }
  });
});

describe('waitlist API (v1 and deprecated alias)', () => {
  let dir: string;

  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'rein-waitlist-api-'));
    process.env.TURSO_DATABASE_URL = `file:${join(dir, 'waitlist.db')}`;
    delete process.env.TURSO_AUTH_TOKEN;
    delete process.env.RESEND_API_KEY;
    resetWaitlistClient();
  });

  afterEach(() => {
    resetWaitlistClient();
    delete process.env.TURSO_DATABASE_URL;
    rmSync(dir, { recursive: true, force: true });
  });

  const post = (handler: typeof v1Post, path: string, body: string) =>
    handler(context(new Request(`https://reinstate.dev${path}`, { method: 'POST', headers: { 'content-type': 'application/json' }, body })));

  it('v1 GET describes the service without deprecation signals', async () => {
    const response = await v1Get(context(new Request('https://reinstate.dev/api/v1/waitlist')));
    expect(response.headers.get('Deprecation')).toBeNull();
    expect(response.headers.get('Link')).toBe(API_LINK_HEADER);
    expect(await json(response)).toEqual({
      ok: true,
      service: 'reinstate-waitlist',
      version: 1,
      accepts: 'POST { "email": "you@example.com" }',
    });
  });

  it('the unversioned alias answers identically plus RFC 9745 deprecation headers', async () => {
    const response = await legacyGet(context(new Request('https://reinstate.dev/api/waitlist')));
    const expectedSeconds = Math.floor(Date.parse(WAITLIST_LEGACY_DEPRECATED_AT) / 1000);
    expect(deprecationHeaders().Deprecation).toBe(`@${expectedSeconds}`);
    expect(response.headers.get('Deprecation')).toBe(`@${expectedSeconds}`);
    expect(response.headers.get('Sunset')).toBeNull();
    expect(response.headers.get('Link')).toContain('<https://reinstate.dev/api/v1/waitlist>; rel="successor-version"');
    expect(response.headers.get('Link')).toContain('rel="service-desc"');
    const body = await json(response);
    expect(body).toMatchObject({ ok: true, service: 'reinstate-waitlist', version: 1, deprecated: true, successor: 'https://reinstate.dev/api/v1/waitlist' });
  });

  it('POST rejects a non-JSON body with invalid_json', async () => {
    const response = await v1Post(context(new Request('https://reinstate.dev/api/v1/waitlist', { method: 'POST', body: 'email=x' })));
    expect(response.status).toBe(400);
    const body = await problem(response);
    expect(body).toMatchObject({ ok: false, code: 'invalid_json', status: 400, instance: '/api/v1/waitlist', title: API_ERROR_TITLES.invalid_json });
    expect(body.error).toBe('Expected JSON body with an email field.');
    expect(body.detail).toBe(body.error);
  });

  it('POST rejects a bad address with invalid_email on both paths', async () => {
    for (const [handler, path] of [[v1Post, '/api/v1/waitlist'], [legacyPost, '/api/waitlist']] as const) {
      const response = await post(handler, path, JSON.stringify({ email: 'not-an-email' }));
      expect(response.status).toBe(400);
      const body = await problem(response);
      expect(body.code).toBe('invalid_email');
      expect(typeof body.error).toBe('string');
      expect(body.hint).toContain('email');
      expect(response.headers.get('Deprecation') !== null).toBe(path === '/api/waitlist');
    }
  });

  it('POST stores a valid address and reports duplicates', async () => {
    const first = await json(await post(v1Post, '/api/v1/waitlist', JSON.stringify({ email: 'Dev@Example.com' })));
    expect(first).toMatchObject({ ok: true, status: 'created', email: 'dev@example.com' });
    const second = await json(await post(legacyPost, '/api/waitlist', JSON.stringify({ email: 'dev@example.com' })));
    expect(second).toMatchObject({ ok: true, status: 'duplicate', email: 'dev@example.com' });
  });

  it('other methods get a JSON problem 405 with Allow on both paths', async () => {
    for (const [handler, path] of [[v1Fallback, '/api/v1/waitlist'], [legacyFallback, '/api/waitlist']] as const) {
      const response = await handler(context(new Request(`https://reinstate.dev${path}`, { method: 'PUT' })));
      expect(response.status).toBe(405);
      expect(response.headers.get('Allow')).toBe(WAITLIST_METHODS.join(', '));
      expect((await problem(response)).code).toBe('method_not_allowed');
    }
  });
});
