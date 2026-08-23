/**
 * Runtime half of Markdown content negotiation. Prerendered pages are served
 * by Vercel's static layer, so the dedicated `agent-surface` function receives
 * the requests the injected Vercel routes select: page requests whose Accept
 * header lists `text/markdown`, and unknown paths from clients that never
 * asked for HTML. Vercel invokes the function with the original request path,
 * so negotiation works from `url.pathname` and serves whichever representation
 * wins by fetching the prebuilt static twin (`/{page}.md`) or the HTML page
 * from the same deployment.
 */
import {
  MARKDOWN_CONTENT_TYPE,
  acceptsExplicitly,
  appendVaryAccept,
  notAcceptableResponse,
  preferredRepresentation,
} from './accept';
import { notFoundMarkdownResponse } from './not-found';
import { markdownPathFor, normalizePagePath } from './paths';

export type Fetcher = (input: string, init?: RequestInit) => Promise<Response>;

/** Marks self-requests so a misrouted function never recurses into itself. */
export const SELF_FETCH_HEADER = 'x-reinstate-agent-surface';

export interface NegotiationInput {
  request: Request;
  /** Origin of the deployment that serves the static files (scheme + host). */
  origin: string;
  /** Request pathname, or the `path` query value when a rewrite supplied one. */
  path: string | null | undefined;
  fetcher?: Fetcher;
  /** Vercel "Protection Bypass for Automation" secret for self-requests on protected deployments. */
  bypassSecret?: string | undefined;
}

export const MARKDOWN_CACHE_CONTROL = 'public, max-age=0, s-maxage=300, stale-while-revalidate=86400';

function selfHeaders(accept: string, bypassSecret?: string): HeadersInit {
  const headers: Record<string, string> = {
    accept,
    'user-agent': 'reinstate-agent-surface/1 (+https://reinstate.dev/developers)',
    [SELF_FETCH_HEADER]: '1',
  };
  if (bypassSecret) headers['x-vercel-protection-bypass'] = bypassSecret;
  return headers;
}

export function isSelfFetch(request: Request): boolean {
  return request.headers.get(SELF_FETCH_HEADER) === '1';
}

export function markdownResponse(body: string | null, status = 200, extraHeaders: Record<string, string> = {}): Response {
  return new Response(body, {
    status,
    headers: {
      'Content-Type': MARKDOWN_CONTENT_TYPE,
      'Cache-Control': MARKDOWN_CACHE_CONTROL,
      'X-Content-Type-Options': 'nosniff',
      Vary: 'Accept',
      ...extraHeaders,
    },
  });
}

function unavailable(representation: string, detail: string): Response {
  return new Response(
    `The ${representation} representation is temporarily unavailable (${detail}). Retry shortly or read https://reinstate.dev/llms.txt.\n`,
    {
      status: 503,
      headers: {
        'Content-Type': 'text/plain; charset=utf-8',
        'Cache-Control': 'no-store',
        'Retry-After': '30',
        Vary: 'Accept',
      },
    },
  );
}

function plainNotFound(): Response {
  return new Response('Not found\n', {
    status: 404,
    headers: { 'Content-Type': 'text/plain; charset=utf-8', 'Cache-Control': 'no-store', Vary: 'Accept' },
  });
}

function withoutBody(response: Response, isHead: boolean): Response {
  return isHead ? new Response(null, response) : response;
}

/**
 * Full negotiation for an existing or missing page path.
 *
 * - Markdown preferred → 200 with the static twin, or a Markdown 404.
 * - HTML preferred → the HTML page from the static layer (or the HTML
 *   not-found page) with `Vary: Accept`; a Markdown 404 when the client never
 *   listed `text/html` explicitly.
 * - Nothing acceptable → 406 with a plain-text list of representations.
 */
export async function negotiatePage(input: NegotiationInput): Promise<Response> {
  const { request, origin } = input;
  const fetcher = input.fetcher ?? ((url, init) => fetch(url, init));
  const isHead = request.method === 'HEAD';
  const accept = request.headers.get('accept');
  const pagePath = normalizePagePath(input.path);

  if (isSelfFetch(request)) return plainNotFound();

  const htmlNotFound = async (): Promise<Response> => {
    try {
      const page = await fetcher(`${origin}/404.html`, { headers: selfHeaders('text/html', input.bypassSecret), redirect: 'manual' });
      if (page.ok) {
        const headers = new Headers({ 'Content-Type': page.headers.get('content-type') ?? 'text/html; charset=utf-8', 'Cache-Control': 'no-store' });
        appendVaryAccept(headers);
        return new Response(page.body, { status: 404, headers });
      }
    } catch {
      // fall through to the plain body
    }
    return plainNotFound();
  };

  if (!pagePath) {
    if (acceptsExplicitly(accept, 'text/html')) return withoutBody(await htmlNotFound(), isHead);
    return withoutBody(notFoundMarkdownResponse(input.path ?? null), isHead);
  }

  const preferred = preferredRepresentation(accept);
  if (preferred === null) return notAcceptableResponse(accept);

  if (preferred === 'text/markdown') {
    let twin: Response;
    try {
      twin = await fetcher(`${origin}${markdownPathFor(pagePath)}`, {
        headers: selfHeaders('text/markdown', input.bypassSecret),
        redirect: 'manual',
      });
    } catch (error) {
      return unavailable('Markdown', error instanceof Error ? error.message : 'fetch failed');
    }
    if (twin.status === 404) return withoutBody(notFoundMarkdownResponse(pagePath), isHead);
    if (!twin.ok) return unavailable('Markdown', `upstream status ${twin.status}`);
    const body = isHead ? null : await twin.text();
    return markdownResponse(body, 200, {
      ...(twin.headers.get('etag') ? { ETag: twin.headers.get('etag')! } : {}),
      'Content-Location': markdownPathFor(pagePath),
    });
  }

  let page: Response;
  try {
    page = await fetcher(`${origin}${pagePath}`, {
      headers: selfHeaders('text/html', input.bypassSecret),
      redirect: 'manual',
    });
  } catch (error) {
    return unavailable('HTML', error instanceof Error ? error.message : 'fetch failed');
  }
  if (page.status === 404) {
    if (!acceptsExplicitly(accept, 'text/html')) return withoutBody(notFoundMarkdownResponse(pagePath), isHead);
    if (!(page.headers.get('content-type') ?? '').startsWith('text/html')) return withoutBody(await htmlNotFound(), isHead);
  }
  const headers = new Headers();
  for (const name of ['content-type', 'cache-control', 'etag', 'last-modified', 'location']) {
    const value = page.headers.get(name);
    if (value) headers.set(name, value);
  }
  appendVaryAccept(headers);
  return new Response(isHead ? null : page.body, { status: page.status, headers });
}
