import type { APIRoute } from 'astro';
import { negotiatePage } from '../../lib/agent-surface/negotiate';

export const prerender = false;

/**
 * Target of the Vercel route that precedes the static filesystem phase: every
 * GET/HEAD for a clean page path whose Accept header lists `text/markdown`
 * lands here with `?path=/original/path`. See `lib/agent-surface/negotiate.ts`.
 */
const handler: APIRoute = ({ request, url }) =>
  negotiatePage({
    request,
    origin: url.origin,
    path: url.searchParams.get('path') ?? url.pathname,
    bypassSecret: process.env.VERCEL_AUTOMATION_BYPASS_SECRET,
  });

export const GET = handler;
export const HEAD = handler;
