/**
 * Accept negotiation for pages Astro renders at request time.
 *
 * Prerendered routes never see request headers: on Vercel they are static
 * files negotiated by the injected `/agent-surface/*` routes, and Astro strips
 * headers from prerendered routes at build time and in `astro dev` to mirror
 * that. So this middleware (1) serves the `/{page}.md` twin URLs in dev by
 * converting the rendered HTML with the build-time converter, and (2) applies
 * the full contract (`Vary: Accept`, Markdown for `Accept: text/markdown`,
 * 406, Markdown 404 for clients that never asked for HTML) to on-demand
 * HTML routes.
 */
import { defineMiddleware } from 'astro:middleware';
import {
  HTML_CONTENT_TYPE,
  MARKDOWN_CONTENT_TYPE,
  acceptsExplicitly,
  appendVaryAccept,
  notAcceptableResponse,
  preferredRepresentation,
} from './lib/agent-surface/accept';
import { notFoundMarkdownResponse } from './lib/agent-surface/not-found';
import { isPagePath, pagePathForMarkdown } from './lib/agent-surface/paths';

function isHtml(response: Response): boolean {
  return (response.headers.get('content-type') ?? '').toLowerCase().startsWith('text/html');
}

export const onRequest = defineMiddleware(async (context, next) => {
  const { method } = context.request;
  if (method !== 'GET' && method !== 'HEAD') return next();

  const pathname = context.url.pathname;

  // Dev-only: `/docs/faq.md` renders the page and converts it, mirroring the build-time twin.
  if (import.meta.env.DEV) {
    const twinOf = pagePathForMarkdown(pathname);
    if (twinOf) {
      const rendered = await next(twinOf);
      if (!isHtml(rendered) || rendered.status !== 200) return rendered;
      const { htmlToMarkdown } = await import('./lib/agent-surface/html-to-markdown');
      const page = htmlToMarkdown(await rendered.text(), { url: new URL(twinOf, context.url).toString() });
      return new Response(method === 'HEAD' ? null : page.markdown, {
        status: 200,
        headers: { 'Content-Type': MARKDOWN_CONTENT_TYPE, 'X-Content-Type-Options': 'nosniff', Vary: 'Accept' },
      });
    }
  }

  // Prerendered routes have no request headers (Astro strips them at build time
  // and in dev); their HTML must be emitted exactly as authored.
  if (context.isPrerendered || !isPagePath(pathname)) return next();

  const accept = context.request.headers.get('accept');
  const preferred = preferredRepresentation(accept);
  if (preferred === null) return notAcceptableResponse(accept);

  const response = await next();
  if (!isHtml(response)) return response;

  if (response.status === 404 && !acceptsExplicitly(accept, 'text/html')) {
    const markdown = notFoundMarkdownResponse(pathname);
    return method === 'HEAD' ? new Response(null, markdown) : markdown;
  }

  if (preferred === 'text/markdown') {
    const { htmlToMarkdown } = await import('./lib/agent-surface/html-to-markdown');
    const page = htmlToMarkdown(await response.text(), { url: context.url.toString() });
    return new Response(method === 'HEAD' ? null : page.markdown, {
      status: response.status,
      headers: { 'Content-Type': MARKDOWN_CONTENT_TYPE, 'X-Content-Type-Options': 'nosniff', Vary: 'Accept' },
    });
  }

  try {
    appendVaryAccept(response.headers);
    if (!response.headers.has('content-type')) response.headers.set('Content-Type', HTML_CONTENT_TYPE);
    return response;
  } catch {
    // Immutable headers (e.g. a static-asset response in dev): re-wrap instead of failing the request.
    const headers = new Headers(response.headers);
    appendVaryAccept(headers);
    return new Response(response.body, { status: response.status, statusText: response.statusText, headers });
  }
});
