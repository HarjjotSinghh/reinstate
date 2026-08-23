import { describe, expect, it } from 'vitest';
import { AGENT_FUNCTION } from './bundle';
import {
  ACCEPT_HTML_PATTERN,
  ACCEPT_MARKDOWN_PATTERN,
  STANDARD_ROUTE_KEYS,
  agentRouteKind,
  injectAgentRoutes,
  markdownNegotiationRoute,
  markdownNotFoundRoute,
  verifyAgentRoutes,
  type VercelOutputConfig,
} from './vercel-routes';

/** Shape `@astrojs/vercel` 11 writes for this site. */
function astroConfig(): VercelOutputConfig {
  return {
    version: 3,
    routes: [
      { handle: 'filesystem' },
      { src: '^/_astro/(.*)$', headers: { 'cache-control': 'public, max-age=31536000, immutable' }, continue: true },
      { src: '^/_server-islands/([^/]+?)/?$', dest: '_render' },
      { src: '^/_image/?$', dest: '_render' },
      { src: '^/api/waitlist/?$', dest: '_render' },
      { src: '^/api/(.*?)/?$', dest: '_render' },
      { src: '^/([^/]+?)\\.txt$', dest: '_render' },
      { src: '^/.*$', dest: '/404.html', status: 404 },
    ],
  };
}

describe('injectAgentRoutes', () => {
  it('places negotiation before the filesystem phase and not-found right after it', () => {
    const result = injectAgentRoutes(astroConfig());
    const routes = result.routes!;
    const filesystem = routes.findIndex((route) => route.handle === 'filesystem');

    expect(routes[filesystem - 1]).toEqual(markdownNegotiationRoute());
    expect(routes[filesystem + 1]).toEqual(markdownNotFoundRoute());
    expect(routes.at(-1)).toEqual({ src: '^/.*$', dest: '/404.html', status: 404 });
    expect(routes).toHaveLength(astroConfig().routes!.length + 2);
    expect(verifyAgentRoutes(result)).toEqual([]);
  });

  it('is idempotent across rebuilds', () => {
    const once = injectAgentRoutes(astroConfig());
    const twice = injectAgentRoutes(once);
    expect(twice).toEqual(once);
    expect(verifyAgentRoutes(twice)).toEqual([]);
  });

  it('does not mutate its input', () => {
    const input = astroConfig();
    const snapshot = JSON.stringify(input);
    injectAgentRoutes(input);
    expect(JSON.stringify(input)).toBe(snapshot);
  });

  it('creates the filesystem phase when the adapter emitted none', () => {
    const result = injectAgentRoutes({ version: 3, routes: [] });
    expect(result.routes!.map((route) => route.handle ?? agentRouteKind(route))).toEqual(['markdown', 'filesystem', 'not-found']);
  });
});

describe('agent routes', () => {
  it('carry only schema-standard keys and target the dedicated function directly', () => {
    for (const route of [markdownNegotiationRoute(), markdownNotFoundRoute()]) {
      for (const key of Object.keys(route)) expect(STANDARD_ROUTE_KEYS.has(key), key).toBe(true);
      expect(route.dest).toBe(AGENT_FUNCTION);
      expect(route.check).toBeUndefined();
      expect(route.methods).toEqual(['GET', 'HEAD']);
    }
    expect(agentRouteKind(markdownNegotiationRoute())).toBe('markdown');
    expect(agentRouteKind(markdownNotFoundRoute())).toBe('not-found');
    expect(agentRouteKind({ src: '^/api/waitlist$', dest: '_render' })).toBeNull();
    expect(agentRouteKind({ src: markdownNegotiationRoute().src, dest: '_render' })).toBeNull();
  });

  it('negotiate only page paths that list text/markdown', () => {
    const route = markdownNegotiationRoute();
    expect(route.has).toEqual([{ type: 'header', key: 'accept', value: ACCEPT_MARKDOWN_PATTERN }]);
    expect(new RegExp(route.has![0]!.value!).test('text/markdown, text/html;q=0.9')).toBe(true);
    expect(new RegExp(route.has![0]!.value!).test('text/html')).toBe(false);
  });

  it('send non-HTML clients for unknown paths to the Markdown 404', () => {
    const route = markdownNotFoundRoute();
    expect(route.missing).toEqual([{ type: 'header', key: 'accept', value: ACCEPT_HTML_PATTERN }]);
    const browser = new RegExp(ACCEPT_HTML_PATTERN);
    expect(browser.test('text/html,application/xhtml+xml,*/*;q=0.8')).toBe(true);
    expect(browser.test('*/*')).toBe(false);
    expect(browser.test('text/markdown')).toBe(false);
  });

  it('match page paths only', () => {
    const source = new RegExp(markdownNegotiationRoute().src!);
    for (const path of ['/', '/docs', '/docs/getting-started', '/developers']) expect(source.test(path), path).toBe(true);
    for (const path of ['/docs/faq.md', '/api/waitlist', '/api', '/_astro/a.css', '/og/home.png', '/_image', '/rss.xml']) {
      expect(source.test(path), path).toBe(false);
    }
  });
});

describe('verifyAgentRoutes', () => {
  it('reports missing, misplaced, and non-standard routes', () => {
    expect(verifyAgentRoutes(astroConfig())).toEqual([
      'config.json: expected exactly one markdown negotiation route, found 0',
      'config.json: expected exactly one markdown not-found route, found 0',
    ]);

    const misplaced = injectAgentRoutes(astroConfig());
    const [markdown, filesystem, notFound, ...rest] = misplaced.routes!;
    misplaced.routes = [filesystem!, ...rest, markdown!, notFound!];
    expect(verifyAgentRoutes(misplaced)).toEqual([
      'config.json: markdown negotiation route must precede the filesystem handle',
      'config.json: markdown not-found route must precede the 404 catch-all',
    ]);

    const tagged = injectAgentRoutes(astroConfig());
    tagged.routes![0] = { ...tagged.routes![0]!, 'x-reinstate-agent-route': 'markdown' };
    expect(verifyAgentRoutes(tagged)).toContain(
      `config.json: route ${markdownNegotiationRoute().src} has non-standard key "x-reinstate-agent-route"`,
    );
  });
});
