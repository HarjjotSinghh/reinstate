import { afterAll, beforeAll, describe, expect, it } from 'vitest';
import { createServer, type Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import { createAgentFunction, requestOrigin } from './function';
import type { Fetcher } from './negotiate';

const ORIGIN = 'https://reinstate.dev';

const files: Record<string, { status: number; body: string; type: string }> = {
  [`${ORIGIN}/docs/faq.md`]: { status: 200, body: '# FAQ\n', type: 'text/markdown; charset=utf-8' },
  [`${ORIGIN}/docs/faq`]: { status: 200, body: '<h1>FAQ</h1>', type: 'text/html; charset=utf-8' },
  [`${ORIGIN}/index.md`]: { status: 200, body: '# Home\n', type: 'text/markdown; charset=utf-8' },
  [`${ORIGIN}/404.html`]: { status: 200, body: '<h1>Not found</h1>', type: 'text/html; charset=utf-8' },
};

const calls: string[] = [];
const fetcher: Fetcher = async (url) => {
  calls.push(url);
  const file = files[url];
  if (!file) return new Response('Not found', { status: 404, headers: { 'content-type': 'text/plain' } });
  return new Response(file.body, { status: file.status, headers: { 'content-type': file.type } });
};

let server: Server;
let base: string;

beforeAll(async () => {
  server = createServer(createAgentFunction({ fetcher, bypassSecret: 'bypass' }));
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
});

afterAll(async () => {
  await new Promise<void>((resolve, reject) => server.close((error) => (error ? reject(error) : resolve())));
});

/** Mimics Vercel: the function sees the original path plus forwarded host headers. */
function call(path: string, init: RequestInit & { accept?: string } = {}) {
  const headers = new Headers(init.headers);
  if (init.accept !== undefined) headers.set('accept', init.accept);
  headers.set('x-forwarded-host', 'reinstate.dev');
  headers.set('x-forwarded-proto', 'https');
  return fetch(`${base}${path}`, { ...init, headers });
}

describe('agent function', () => {
  it('derives the deployment origin from forwarded headers', () => {
    expect(requestOrigin({ 'x-forwarded-host': 'reinstate.dev', 'x-forwarded-proto': 'https' }, 'https://x.test')).toBe('https://reinstate.dev');
    expect(requestOrigin({ host: 'localhost:3000' }, 'https://x.test')).toBe('http://localhost:3000');
    expect(requestOrigin({ 'x-forwarded-host': 'a.vercel.app, b.vercel.app' }, 'https://x.test')).toBe('https://a.vercel.app');
    expect(requestOrigin({}, 'https://x.test')).toBe('https://x.test');
  });

  it('serves the Markdown twin for Accept: text/markdown on a static prerendered page', async () => {
    const response = await call('/', { accept: 'text/markdown' });
    expect(response.status).toBe(200);
    expect(response.headers.get('content-type')).toBe('text/markdown; charset=utf-8');
    expect(response.headers.get('vary')).toBe('Accept');
    expect(await response.text()).toBe('# Home\n');
    expect(calls.at(-1)).toBe(`${ORIGIN}/index.md`);
  });

  it('passes the HTML page through when HTML wins', async () => {
    const response = await call('/docs/faq', { accept: 'text/html, text/markdown' });
    expect(response.status).toBe(200);
    expect(response.headers.get('content-type')).toBe('text/html; charset=utf-8');
    expect(response.headers.get('vary')).toBe('Accept');
    expect(await response.text()).toBe('<h1>FAQ</h1>');
  });

  it('answers unknown paths with the Markdown 404 for non-HTML clients and 404.html for browsers', async () => {
    const agent = await call('/nope');
    expect(agent.status).toBe(404);
    expect(agent.headers.get('content-type')).toBe('text/markdown; charset=utf-8');
    expect(await agent.text()).toContain('# 404: no page at /nope');

    const browser = await call('/nope', { accept: 'text/html,application/xhtml+xml,*/*;q=0.8' });
    expect(browser.status).toBe(404);
    expect(browser.headers.get('content-type')).toBe('text/html; charset=utf-8');
    expect(await browser.text()).toBe('<h1>Not found</h1>');
  });

  it('returns 406 when nothing is acceptable and 405 for other methods', async () => {
    const notAcceptable = await call('/docs/faq', { accept: 'application/pdf' });
    expect(notAcceptable.status).toBe(406);
    const post = await call('/docs/faq', { method: 'POST', accept: 'text/markdown' });
    expect(post.status).toBe(405);
    expect(post.headers.get('allow')).toBe('GET, HEAD');
  });

  it('answers HEAD without a body', async () => {
    const response = await call('/docs/faq', { method: 'HEAD', accept: 'text/markdown' });
    expect(response.status).toBe(200);
    expect(response.headers.get('content-type')).toBe('text/markdown; charset=utf-8');
    expect(await response.text()).toBe('');
  });
});
