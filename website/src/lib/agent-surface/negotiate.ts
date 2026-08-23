/**
 * Runtime half of Markdown content negotiation. Prerendered pages are served
 * by Vercel's static layer, so the `/agent-surface/markdown` endpoint receives the
 * requests that list `text/markdown`, applies the full Accept algorithm, and
 * serves whichever representation wins by fetching the prebuilt static twin
 * (`/{page}.md`) or the HTML page from the same deployment.
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

export interface NegotiationInput {
  request: Request;
  /** Origin of the deployment that serves the static files (scheme + host). */
  origin: string;
  /** Raw `path` query value set by the Vercel rewrite, or the request pathname in dev. */
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
  };
  if (bypassSecret) headers['x-vercel-protection-bypass'] = bypassSecret;
  return headers;
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

/**
 * Full negotiation for an existing or missing page path.
 *
 * - Markdown preferred → 200 with the static twin, or a Markdown 404.
 * - HTML preferred → the HTML page from the static layer, with `Vary: Accept`.
 * - Nothing acceptable → 406 with a plain-text list of representations.
 */
export async function negotiatePage(input: NegotiationInput): Promise<Response> {
  const { request, origin } = input;
  const fetcher = input.fetcher ?? ((url, init) => fetch(url, init));
  const isHead = request.method === 'HEAD';
  const accept = request.headers.get('accept');
  const pagePath = normalizePagePath(input.path);

  if (!pagePath) {
    const response = notFoundMarkdownResponse(input.path ?? null);
    return isHead ? new Response(null, response) : response;
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
    if (twin.status === 404) {
      const response = notFoundMarkdownResponse(pagePath);
      return isHead ? new Response(null, response) : response;
    }
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
  if (page.status === 404 && !acceptsExplicitly(accept, 'text/html')) {
    const response = notFoundMarkdownResponse(pagePath);
    return isHead ? new Response(null, response) : response;
  }
  const headers = new Headers();
  for (const name of ['content-type', 'cache-control', 'etag', 'last-modified', 'location']) {
    const value = page.headers.get(name);
    if (value) headers.set(name, value);
  }
  appendVaryAccept(headers);
  return new Response(isHead ? null : page.body, { status: page.status, headers });
}

/** Always a Markdown 404; used for page paths that matched neither a static file nor a runtime route. */
export function notFoundPage(input: Pick<NegotiationInput, 'request' | 'path'>): Response {
  const response = notFoundMarkdownResponse(normalizePagePath(input.path) ?? input.path ?? null);
  return input.request.method === 'HEAD' ? new Response(null, response) : response;
}
