import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import type { APIContext } from 'astro';
import { resetWaitlistClient } from './waitlist-db';
import { API_ERROR_CODES, apiError, apiErrorBody, apiNotFound, methodNotAllowed } from './api-errors';
import { ALL as waitlistFallback, GET as waitlistGet, POST as waitlistPost, WAITLIST_METHODS } from '../pages/api/waitlist';
import { ALL as apiCatchAll } from '../pages/api/[...path]';

function context(request: Request): APIContext {
  return { request, url: new URL(request.url) } as unknown as APIContext;
}

async function json(response: Response): Promise<Record<string, unknown>> {
  expect(response.headers.get('Content-Type')).toBe('application/json; charset=utf-8');
  return (await response.json()) as Record<string, unknown>;
}

describe('api error helpers', () => {
  it('produce the documented error shape', async () => {
    const body = apiErrorBody(400, 'invalid_email', 'Bad.', 'Fix it.');
    expect(body).toEqual({
      ok: false,
      status: 400,
      code: 'invalid_email',
      error: 'Bad.',
      hint: 'Fix it.',
      docs: 'https://reinstate.dev/openapi.json',
    });
    const response = apiError(503, 'storage_unavailable', 'Down.', 'Retry.', { headers: { 'Retry-After': '60' } });
    expect(response.status).toBe(503);
    expect(response.headers.get('Retry-After')).toBe('60');
    expect(response.headers.get('Cache-Control')).toBe('no-store');
    expect((await json(response)).code).toBe('storage_unavailable');
    expect(API_ERROR_CODES).toContain('not_found');
  });

  it('405 carries an Allow header', async () => {
    const response = methodNotAllowed('PUT', ['GET', 'POST'], '/api/waitlist');
    expect(response.status).toBe(405);
    expect(response.headers.get('Allow')).toBe('GET, POST');
    const body = await json(response);
    expect(body.code).toBe('method_not_allowed');
    expect(body.error).toContain('PUT');
    expect(body.hint).toContain('GET, POST');
  });

  it('404 points back at the OpenAPI document', async () => {
    const body = await json(apiNotFound('/api/nope'));
    expect(body.code).toBe('not_found');
    expect(body.error).toContain('/api/nope');
    expect(body.hint).toContain('openapi.json');
  });
});

describe('/api/{path} catch-all', () => {
  it('returns JSON 404 for any method', async () => {
    for (const method of ['GET', 'POST', 'DELETE']) {
      const response = await apiCatchAll(context(new Request('https://reinstate.dev/api/does-not-exist', { method })));
      expect(response.status).toBe(404);
      const body = await json(response);
      expect(body.ok).toBe(false);
      expect(body.code).toBe('not_found');
      expect(body.error).toContain('/api/does-not-exist');
    }
  });
});

describe('/api/waitlist', () => {
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

  it('GET describes the service', async () => {
    const body = await json(await waitlistGet(context(new Request('https://reinstate.dev/api/waitlist'))));
    expect(body).toEqual({
      ok: true,
      service: 'reinstate-waitlist',
      accepts: 'POST { "email": "you@example.com" }',
    });
  });

  it('POST rejects a non-JSON body with invalid_json', async () => {
    const response = await waitlistPost(
      context(new Request('https://reinstate.dev/api/waitlist', { method: 'POST', body: 'email=x' })),
    );
    expect(response.status).toBe(400);
    const body = await json(response);
    expect(body).toMatchObject({ ok: false, code: 'invalid_json', status: 400 });
    expect(body.error).toBe('Expected JSON body with an email field.');
    expect(typeof body.hint).toBe('string');
  });

  it('POST rejects a bad address with invalid_email and keeps the legacy error string', async () => {
    const response = await waitlistPost(
      context(
        new Request('https://reinstate.dev/api/waitlist', {
          method: 'POST',
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({ email: 'not-an-email' }),
        }),
      ),
    );
    expect(response.status).toBe(400);
    const body = await json(response);
    expect(body.code).toBe('invalid_email');
    expect(typeof body.error).toBe('string');
    expect(body.hint).toContain('email');
  });

  it('POST stores a valid address and reports duplicates', async () => {
    const post = () =>
      waitlistPost(
        context(
          new Request('https://reinstate.dev/api/waitlist', {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({ email: 'Dev@Example.com' }),
          }),
        ),
      );
    const first = await json(await post());
    expect(first).toMatchObject({ ok: true, status: 'created', email: 'dev@example.com' });
    const second = await json(await post());
    expect(second).toMatchObject({ ok: true, status: 'duplicate', email: 'dev@example.com' });
  });

  it('other methods get a JSON 405 with Allow', async () => {
    const response = await waitlistFallback(
      context(new Request('https://reinstate.dev/api/waitlist', { method: 'PUT' })),
    );
    expect(response.status).toBe(405);
    expect(response.headers.get('Allow')).toBe(WAITLIST_METHODS.join(', '));
    expect((await json(response)).code).toBe('method_not_allowed');
  });
});
