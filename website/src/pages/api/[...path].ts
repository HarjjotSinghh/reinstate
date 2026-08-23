import type { APIRoute } from 'astro';
import { apiNotFound } from '../../lib/api-errors';

export const prerender = false;

/** Every undocumented `/api/*` path answers with a JSON 404, never the HTML shell. */
export const ALL: APIRoute = ({ url }) => apiNotFound(url.pathname);
