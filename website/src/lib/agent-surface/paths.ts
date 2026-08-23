/**
 * Path rules shared by the middleware, the on-demand catch-all page, the build
 * step that writes Markdown twins, and the Vercel route injector.
 *
 * A "page path" is a clean URL without a file extension: `/`, `/docs`,
 * `/docs/getting-started`. Everything with a dot (assets, feeds, JSON, the
 * Markdown twins themselves) and every runtime namespace (`/api/`, Astro
 * internals, generated social cards) is excluded from negotiation.
 */

/** Namespaces that must never be rewritten into the agent endpoints. */
export const EXCLUDED_PATH_PREFIXES = [
  '/api/',
  '/_astro/',
  '/_image',
  '/_server-islands',
  '/og/',
] as const;

const PAGE_PATH_PATTERN = /^\/(?:[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)*)?$/;

/** Exact-match page path validation (no dots, no query, no traversal, no double slashes). */
export function isPagePath(path: string): boolean {
  if (path === '/api') return false;
  if (EXCLUDED_PATH_PREFIXES.some((prefix) => path.startsWith(prefix))) return false;
  return PAGE_PATH_PATTERN.test(path);
}

/** Strips a trailing slash (except for the root) and rejects anything that is not a page path. */
export function normalizePagePath(raw: string | null | undefined): string | null {
  if (typeof raw !== 'string' || raw.length === 0) return null;
  let path = raw;
  if (!path.startsWith('/')) path = `/${path}`;
  if (path.length > 1 && path.endsWith('/')) path = path.slice(0, -1);
  return isPagePath(path) ? path : null;
}

/** `/docs/getting-started` → `/docs/getting-started.md`; `/` → `/index.md`. */
export function markdownPathFor(pagePath: string): string {
  return pagePath === '/' ? '/index.md' : `${pagePath}.md`;
}

/** Inverse of {@link markdownPathFor}; returns `null` for anything that is not a twin URL. */
export function pagePathForMarkdown(markdownPath: string): string | null {
  if (!markdownPath.endsWith('.md')) return null;
  const stripped = markdownPath.slice(0, -3);
  if (stripped === '/index') return '/';
  return isPagePath(stripped) ? stripped : null;
}

/** Relative file path (POSIX separators) of a twin inside the static output directory. */
export function markdownFileFor(pagePath: string): string {
  return markdownPathFor(pagePath).slice(1);
}

/**
 * Converts the `pathname` Astro reports for a prerendered page into a page
 * path. Astro reports `""` or `"/"` for the homepage and `docs/faq` (without
 * leading slash, sometimes with a trailing one) for nested pages.
 */
export function pagePathFromBuildPathname(pathname: string): string {
  const trimmed = pathname.replace(/^\/+|\/+$/g, '');
  return trimmed ? `/${trimmed}` : '/';
}
