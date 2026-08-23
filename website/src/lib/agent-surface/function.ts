/**
 * Entry point of the dedicated Vercel function that answers agent traffic.
 *
 * The two injected Build Output routes send requests here with the original
 * request path (Vercel never rewrites the path a function sees). Astro's own
 * router cannot serve prerendered static routes at request time, so this
 * handler bypasses it entirely: it negotiates on `req.url` and fetches the
 * prebuilt Markdown twin or the HTML page from the same deployment.
 *
 * Bundled by `bundle.ts` during `astro build` into
 * `.vercel/output/functions/agent-surface.func/index.mjs` (Node launcher).
 */
import type { IncomingHttpHeaders, IncomingMessage, ServerResponse } from 'node:http';
import { negotiatePage, type Fetcher } from './negotiate';

export interface AgentFunctionOptions {
  fetcher?: Fetcher;
  bypassSecret?: string | undefined;
  /** Origin to use when the request carries no host information. */
  fallbackOrigin?: string;
}

export type NodeHandler = (req: IncomingMessage, res: ServerResponse) => Promise<void>;

function firstValue(value: string | string[] | undefined): string | undefined {
  const raw = Array.isArray(value) ? value[0] : value;
  return raw?.split(',')[0]?.trim() || undefined;
}

export function requestOrigin(headers: IncomingHttpHeaders, fallbackOrigin: string): string {
  const host = firstValue(headers['x-forwarded-host']) ?? firstValue(headers.host);
  if (!host) return fallbackOrigin;
  const proto = firstValue(headers['x-forwarded-proto']) ?? (host.startsWith('localhost') || host.startsWith('127.') ? 'http' : 'https');
  return `${proto}://${host}`;
}

function toWebHeaders(headers: IncomingHttpHeaders): Headers {
  const result = new Headers();
  for (const [name, value] of Object.entries(headers)) {
    if (value === undefined) continue;
    for (const entry of Array.isArray(value) ? value : [value]) result.append(name, entry);
  }
  return result;
}

export function createAgentFunction(options: AgentFunctionOptions = {}): NodeHandler {
  const fallbackOrigin = options.fallbackOrigin ?? 'https://reinstate.dev';
  return async (req, res) => {
    const method = req.method ?? 'GET';
    if (method !== 'GET' && method !== 'HEAD') {
      res.statusCode = 405;
      res.setHeader('Allow', 'GET, HEAD');
      res.setHeader('Content-Type', 'text/plain; charset=utf-8');
      res.end('Method not allowed\n');
      return;
    }
    const origin = requestOrigin(req.headers, fallbackOrigin);
    const url = new URL(req.url ?? '/', origin);
    const request = new Request(url, { method, headers: toWebHeaders(req.headers) });
    const response = await negotiatePage({
      request,
      origin,
      path: url.pathname,
      fetcher: options.fetcher,
      bypassSecret: options.bypassSecret,
    });
    res.statusCode = response.status;
    response.headers.forEach((value, name) => res.setHeader(name, value));
    if (method === 'HEAD' || !response.body) {
      res.end();
      return;
    }
    res.end(Buffer.from(await response.arrayBuffer()));
  };
}

export default createAgentFunction({ bypassSecret: process.env.VERCEL_AUTOMATION_BYPASS_SECRET });
