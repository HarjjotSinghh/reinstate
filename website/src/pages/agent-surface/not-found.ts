import type { APIRoute } from 'astro';
import { notFoundPage } from '../../lib/agent-surface/negotiate';

export const prerender = false;

/**
 * Target of the Vercel route that runs after the static filesystem phase for
 * GET/HEAD requests whose Accept header does not list `text/html`: the path
 * matched nothing, so the client gets a short Markdown 404 with recovery
 * links instead of the HTML not-found shell.
 */
const handler: APIRoute = ({ request, url }) =>
  notFoundPage({ request, path: url.searchParams.get('path') ?? url.pathname });

export const GET = handler;
export const HEAD = handler;
