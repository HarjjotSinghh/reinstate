/**
 * Injects the two agent routes into the Vercel Build Output `config.json`
 * that `@astrojs/vercel` writes.
 *
 * Why routes are injected here instead of `vercel.json`: prerendered pages
 * are answered by Vercel's filesystem phase before any `vercel.json` rewrite
 * runs, so same-URL `Accept: text/markdown` negotiation needs a route that
 * precedes `{ "handle": "filesystem" }`. Only the Build Output config can
 * express that. `vercel build` merges these routes with `vercel.json`
 * (redirects and headers first, then the filesystem handle, then rewrites),
 * so both files keep their jobs: headers live in `vercel.json`, phase-sensitive
 * rewrites live here.
 */
import { AGENT_MARKDOWN_ENDPOINT, AGENT_NOT_FOUND_ENDPOINT } from './paths';

/**
 * Clean page paths only: no dots (assets, feeds, JSON, Markdown twins) and no
 * runtime namespace (`/api/`, `/agent-surface/`, Astro internals, social cards).
 * Keep in sync with {@link isPagePath} in `paths.ts`.
 */
export const PAGE_PATH_SOURCE =
  '^(/(?!api(?:/|$)|agent-surface(?:/|$)|_astro/|_image|_server-islands|og/)[^.]*)$';

export const ACCEPT_MARKDOWN_PATTERN = '.*text/markdown.*';
export const ACCEPT_HTML_PATTERN = '.*text/html.*';

export interface VercelRoute {
  src?: string;
  dest?: string;
  handle?: string;
  status?: number;
  methods?: string[];
  check?: boolean;
  continue?: boolean;
  headers?: Record<string, string>;
  has?: Array<{ type: string; key: string; value?: string }>;
  missing?: Array<{ type: string; key: string; value?: string }>;
  [key: string]: unknown;
}

export interface VercelOutputConfig {
  version: number;
  routes?: VercelRoute[];
  [key: string]: unknown;
}

export const AGENT_ROUTE_MARKER = 'x-reinstate-agent-route';

/** Requests that list `text/markdown` are negotiated by the markdown endpoint before static files are considered. */
export function markdownNegotiationRoute(): VercelRoute {
  return {
    src: PAGE_PATH_SOURCE,
    has: [{ type: 'header', key: 'accept', value: ACCEPT_MARKDOWN_PATTERN }],
    methods: ['GET', 'HEAD'],
    dest: `${AGENT_MARKDOWN_ENDPOINT}?path=$1`,
    check: true,
    [AGENT_ROUTE_MARKER]: 'markdown',
  };
}

/** Unmatched page paths from clients that do not ask for HTML get a Markdown 404 instead of the HTML shell. */
export function markdownNotFoundRoute(): VercelRoute {
  return {
    src: PAGE_PATH_SOURCE,
    missing: [{ type: 'header', key: 'accept', value: ACCEPT_HTML_PATTERN }],
    methods: ['GET', 'HEAD'],
    dest: `${AGENT_NOT_FOUND_ENDPOINT}?path=$1`,
    check: true,
    [AGENT_ROUTE_MARKER]: 'not-found',
  };
}

function isFilesystemHandle(route: VercelRoute): boolean {
  return route.handle === 'filesystem';
}

/**
 * Returns a new config with the agent routes in place. Idempotent: existing
 * injected routes are replaced, so re-running a build never duplicates them.
 */
export function injectAgentRoutes(config: VercelOutputConfig): VercelOutputConfig {
  const routes = (config.routes ?? []).filter((route) => !(AGENT_ROUTE_MARKER in route));
  const filesystemIndex = routes.findIndex(isFilesystemHandle);

  if (filesystemIndex === -1) {
    return {
      ...config,
      routes: [
        markdownNegotiationRoute(),
        { handle: 'filesystem' },
        markdownNotFoundRoute(),
        ...routes,
      ],
    };
  }

  const before = routes.slice(0, filesystemIndex);
  const after = routes.slice(filesystemIndex + 1);
  return {
    ...config,
    routes: [
      ...before,
      markdownNegotiationRoute(),
      { handle: 'filesystem' },
      markdownNotFoundRoute(),
      ...after,
    ],
  };
}

/** Validates an output config: both routes present, in the right phases, exactly once. */
export function verifyAgentRoutes(config: VercelOutputConfig): string[] {
  const errors: string[] = [];
  const routes = config.routes ?? [];
  const filesystemIndex = routes.findIndex(isFilesystemHandle);
  const markdownIndexes = routes
    .map((route, index) => (route[AGENT_ROUTE_MARKER] === 'markdown' ? index : -1))
    .filter((index) => index !== -1);
  const notFoundIndexes = routes
    .map((route, index) => (route[AGENT_ROUTE_MARKER] === 'not-found' ? index : -1))
    .filter((index) => index !== -1);
  const catchAllIndex = routes.findIndex(
    (route) => route.status === 404 && typeof route.src === 'string' && /^\^?\/\.\*\$?$/.test(route.src),
  );

  if (filesystemIndex === -1) errors.push('config.json: missing { "handle": "filesystem" } route');
  if (markdownIndexes.length !== 1) {
    errors.push(`config.json: expected exactly one markdown negotiation route, found ${markdownIndexes.length}`);
  } else if (filesystemIndex !== -1 && markdownIndexes[0]! > filesystemIndex) {
    errors.push('config.json: markdown negotiation route must precede the filesystem handle');
  }
  if (notFoundIndexes.length !== 1) {
    errors.push(`config.json: expected exactly one markdown not-found route, found ${notFoundIndexes.length}`);
  } else {
    if (filesystemIndex !== -1 && notFoundIndexes[0]! < filesystemIndex) {
      errors.push('config.json: markdown not-found route must follow the filesystem handle');
    }
    if (catchAllIndex !== -1 && notFoundIndexes[0]! > catchAllIndex) {
      errors.push('config.json: markdown not-found route must precede the 404 catch-all');
    }
  }
  return errors;
}
