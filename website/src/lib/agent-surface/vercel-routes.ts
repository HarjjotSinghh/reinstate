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
 *
 * Both routes target the dedicated `agent-surface` function that `bundle.ts`
 * writes next to Astro's `_render` function. Vercel always invokes a function
 * with the original request path, and Astro's router refuses prerendered
 * static routes at request time, so agent traffic must bypass Astro entirely.
 * Routes carry only schema-standard keys: Vercel rejects unknown properties
 * (`invalid_routes`) when it merges `vercel.json`.
 */
import { AGENT_FUNCTION } from './bundle';

/**
 * Clean page paths only: no dots (assets, feeds, JSON, Markdown twins) and no
 * runtime namespace (`/api/`, Astro internals, social cards).
 * Keep in sync with {@link isPagePath} in `paths.ts`.
 */
export const PAGE_PATH_SOURCE = '^(/(?!api(?:/|$)|_astro/|_image|_server-islands|og/)[^.]*)$';

export const ACCEPT_MARKDOWN_PATTERN = '.*text/markdown.*';
export const ACCEPT_HTML_PATTERN = '.*text/html.*';

/** Keys Vercel accepts on a Build Output route. */
export const STANDARD_ROUTE_KEYS = new Set([
  'src', 'dest', 'handle', 'status', 'methods', 'check', 'continue', 'headers', 'has', 'missing',
  'locale', 'caseSensitive', 'important', 'override', 'middlewarePath', 'middlewareRawSrc', 'transforms', 'mitigate',
]);

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

export type AgentRouteKind = 'markdown' | 'not-found';

/** Identifies an injected route by its shape; no custom keys are allowed on routes. */
export function agentRouteKind(route: VercelRoute): AgentRouteKind | null {
  if (route.src !== PAGE_PATH_SOURCE || route.dest !== AGENT_FUNCTION) return null;
  if (route.has?.some((condition) => condition.key === 'accept' && condition.value === ACCEPT_MARKDOWN_PATTERN)) return 'markdown';
  if (route.missing?.some((condition) => condition.key === 'accept' && condition.value === ACCEPT_HTML_PATTERN)) return 'not-found';
  return null;
}

/** Requests that list `text/markdown` reach the agent function before static files are considered. */
export function markdownNegotiationRoute(): VercelRoute {
  return {
    src: PAGE_PATH_SOURCE,
    has: [{ type: 'header', key: 'accept', value: ACCEPT_MARKDOWN_PATTERN }],
    methods: ['GET', 'HEAD'],
    dest: AGENT_FUNCTION,
  };
}

/** Unmatched page paths from clients that do not ask for HTML get the Markdown 404 instead of the HTML shell. */
export function markdownNotFoundRoute(): VercelRoute {
  return {
    src: PAGE_PATH_SOURCE,
    missing: [{ type: 'header', key: 'accept', value: ACCEPT_HTML_PATTERN }],
    methods: ['GET', 'HEAD'],
    dest: AGENT_FUNCTION,
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
  const routes = (config.routes ?? []).filter((route) => agentRouteKind(route) === null);
  const filesystemIndex = routes.findIndex(isFilesystemHandle);

  if (filesystemIndex === -1) {
    return { ...config, routes: [markdownNegotiationRoute(), { handle: 'filesystem' }, markdownNotFoundRoute(), ...routes] };
  }

  const before = routes.slice(0, filesystemIndex);
  const after = routes.slice(filesystemIndex + 1);
  return {
    ...config,
    routes: [...before, markdownNegotiationRoute(), { handle: 'filesystem' }, markdownNotFoundRoute(), ...after],
  };
}

/** Validates an output config: both routes present, in the right phases, exactly once, standard keys only. */
export function verifyAgentRoutes(config: VercelOutputConfig): string[] {
  const errors: string[] = [];
  const routes = config.routes ?? [];
  const filesystemIndex = routes.findIndex(isFilesystemHandle);
  const markdownIndexes = routes.map((route, index) => (agentRouteKind(route) === 'markdown' ? index : -1)).filter((i) => i !== -1);
  const notFoundIndexes = routes.map((route, index) => (agentRouteKind(route) === 'not-found' ? index : -1)).filter((i) => i !== -1);
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
  for (const route of routes) {
    for (const key of Object.keys(route)) {
      if (!STANDARD_ROUTE_KEYS.has(key)) errors.push(`config.json: route ${route.src ?? route.handle} has non-standard key "${key}"`);
    }
  }
  return errors;
}
