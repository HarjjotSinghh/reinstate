import { describe, expect, it } from 'vitest';
import { SELF_FETCH_HEADER, negotiatePage, type Fetcher } from './negotiate';

const ORIGIN = 'https://reinstate.dev';

function fakeFetcher(files: Record<string, { status: number; body: string; type: string }>) {
  const calls: Array<{ url: string; accept: string | undefined }> = [];
  const fetcher: Fetcher = async (url, init) => {
    const headers = new Headers(init?.headers);
    calls.push({ url, accept: headers.get('accept') ?? undefined });
    const file = files[url];
    if (!file) return new Response('Not found', { status: 404, headers: { 'content-type': 'text/plain' } });
    return new Response(file.body, {
      status: file.status,
      headers: { 'content-type': file.type, 'cache-control': 'public, max-age=0, must-revalidate', etag: '"abc"' },
    });
  };
  return { fetcher, calls };
}

const files = {
  [`${ORIGIN}/docs/faq.md`]: { status: 200, body: '# FAQ\n\nAnswers.\n', type: 'text/markdown; charset=utf-8' },
  [`${ORIGIN}/docs/faq`]: { status: 200, body: '<!doctype html><h1>FAQ</h1>', type: 'text/html; charset=utf-8' },
  [`${ORIGIN}/index.md`]: { status: 200, body: '# Reinstate\n', type: 'text/markdown; charset=utf-8' },
};

function request(path: string, accept?: string, method = 'GET'): Request {
  return new Request(`${ORIGIN}${path}`, {
    method,
    headers: accept === undefined ? {} : { accept },
  });
}

describe('negotiatePage', () => {
  it('serves the static Markdown twin when Markdown is preferred', async () => {
    const { fetcher, calls } = fakeFetcher(files);
    const response = await negotiatePage({
      request: request('/docs/faq', 'text/markdown, text/html;q=0.5'),
      origin: ORIGIN,
      path: '/docs/faq',
      fetcher,
      bypassSecret: 'secret',
    });

    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Type')).toBe('text/markdown; charset=utf-8');
    expect(response.headers.get('Vary')).toBe('Accept');
    expect(response.headers.get('Content-Location')).toBe('/docs/faq.md');
    expect(response.headers.get('ETag')).toBe('"abc"');
    expect(await response.text()).toBe('# FAQ\n\nAnswers.\n');
    expect(calls).toEqual([{ url: `${ORIGIN}/docs/faq.md`, accept: 'text/markdown' }]);
  });

  it('marks self-requests and never answers one with another self-request', async () => {
    const { fetcher, calls } = fakeFetcher(files);
    const capture: Fetcher = async (url, init) => {
      expect(new Headers(init?.headers).get(SELF_FETCH_HEADER)).toBe('1');
      return fetcher(url, init);
    };
    await negotiatePage({ request: request('/docs/faq', 'text/markdown'), origin: ORIGIN, path: '/docs/faq', fetcher: capture });
    expect(calls).toHaveLength(1);

    const self = new Request(`${ORIGIN}/docs/nope`, { headers: { accept: 'text/html', [SELF_FETCH_HEADER]: '1' } });
    const response = await negotiatePage({ request: self, origin: ORIGIN, path: '/docs/nope', fetcher });
    expect(response.status).toBe(404);
    expect(response.headers.get('Content-Type')).toBe('text/plain; charset=utf-8');
    expect(calls).toHaveLength(1);
  });

  it('serves the static 404.html to browsers in production without rewriting', async () => {
    const { fetcher, calls } = fakeFetcher({ ...files, [`${ORIGIN}/404.html`]: { status: 200, body: '<h1>Not found</h1>', type: 'text/html; charset=utf-8' } });
    const response = await negotiatePage({ request: request('/docs/nope', 'text/html,*/*;q=0.8'), origin: ORIGIN, path: '/docs/nope', fetcher });
    expect(response.status).toBe(404);
    expect(response.headers.get('Content-Type')).toBe('text/html; charset=utf-8');
    expect(response.headers.get('Vary')).toBe('Accept');
    expect(await response.text()).toContain('<h1>Not found</h1>');
    expect(calls.map((call) => call.url)).toEqual([`${ORIGIN}/docs/nope`, `${ORIGIN}/404.html`]);
  });

  it('maps the homepage to /index.md', async () => {
    const { fetcher, calls } = fakeFetcher(files);
    const response = await negotiatePage({ request: request('/', 'text/markdown'), origin: ORIGIN, path: '/', fetcher });
    expect(response.status).toBe(200);
    expect(await response.text()).toBe('# Reinstate\n');
    expect(calls[0]!.url).toBe(`${ORIGIN}/index.md`);
  });

  it('passes the HTML page through with Vary: Accept when HTML wins', async () => {
    const { fetcher, calls } = fakeFetcher(files);
    const response = await negotiatePage({
      request: request('/docs/faq', 'text/html, text/markdown'),
      origin: ORIGIN,
      path: '/docs/faq',
      fetcher,
    });

    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Type')).toBe('text/html; charset=utf-8');
    expect(response.headers.get('Vary')).toBe('Accept');
    expect(response.headers.get('Cache-Control')).toBe('public, max-age=0, must-revalidate');
    expect(await response.text()).toContain('<h1>FAQ</h1>');
    expect(calls).toEqual([{ url: `${ORIGIN}/docs/faq`, accept: 'text/html' }]);
  });

  it('returns a Markdown 404 for an unknown page', async () => {
    const { fetcher } = fakeFetcher(files);
    const response = await negotiatePage({
      request: request('/does-not-exist', 'text/markdown'),
      origin: ORIGIN,
      path: '/does-not-exist',
      fetcher,
    });
    expect(response.status).toBe(404);
    expect(response.headers.get('Content-Type')).toBe('text/markdown; charset=utf-8');
    expect(await response.text()).toContain('# 404: no page at /does-not-exist');
  });

  it('returns a Markdown 404 for an invalid path without touching the network', async () => {
    const { fetcher, calls } = fakeFetcher(files);
    const response = await negotiatePage({
      request: request('/../etc', 'text/markdown'),
      origin: ORIGIN,
      path: '/../etc',
      fetcher,
    });
    expect(response.status).toBe(404);
    expect(calls).toEqual([]);
  });

  it('returns 406 when nothing is acceptable', async () => {
    const { fetcher, calls } = fakeFetcher(files);
    const response = await negotiatePage({ request: request('/docs/faq', 'application/pdf'), origin: ORIGIN, path: '/docs/faq', fetcher });
    expect(response.status).toBe(406);
    expect(response.headers.get('Content-Type')).toBe('text/plain; charset=utf-8');
    expect(await response.text()).toContain('- text/markdown');
    expect(calls).toEqual([]);
  });

  it('answers HEAD without a body', async () => {
    const { fetcher } = fakeFetcher(files);
    const response = await negotiatePage({
      request: request('/docs/faq', 'text/markdown', 'HEAD'),
      origin: ORIGIN,
      path: '/docs/faq',
      fetcher,
    });
    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Type')).toBe('text/markdown; charset=utf-8');
    expect(await response.text()).toBe('');
  });

  it('fails soft with 503 when the static layer is unreachable', async () => {
    const fetcher: Fetcher = async () => {
      throw new Error('ECONNRESET');
    };
    const response = await negotiatePage({ request: request('/docs/faq', 'text/markdown'), origin: ORIGIN, path: '/docs/faq', fetcher });
    expect(response.status).toBe(503);
    expect(response.headers.get('Retry-After')).toBe('30');
    expect(await response.text()).toContain('ECONNRESET');
  });
});
