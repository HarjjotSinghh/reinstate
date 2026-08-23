import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { buildAgentSurface, fullTextOrder, htmlFileForPage, isExcludedFromFullText, isExcludedFromTwins } from './build';
import { verifyAgentRoutes } from './vercel-routes';

function html(title: string, body: string): string {
  return `<!doctype html><html><head><title>${title}</title><meta name="description" content="${title} description"></head><body><main><h1>${title}</h1>${body}</main></body></html>`;
}

describe('buildAgentSurface', () => {
  let root: string;
  let clientDir: string;
  let vercelStatic: string;
  let vercelConfig: string;

  beforeEach(() => {
    root = mkdtempSync(join(tmpdir(), 'rein-agent-surface-'));
    clientDir = join(root, 'dist', 'client');
    vercelStatic = join(root, '.vercel', 'output', 'static');
    vercelConfig = join(root, '.vercel', 'output', 'config.json');
    mkdirSync(join(clientDir, 'docs', 'faq'), { recursive: true });
    mkdirSync(join(clientDir, 'preview'), { recursive: true });
    mkdirSync(vercelStatic, { recursive: true });
    writeFileSync(join(clientDir, 'index.html'), html('Reinstate', '<p>Home</p>'));
    writeFileSync(join(clientDir, 'docs', 'faq', 'index.html'), html('FAQ', '<p>Answers</p>'));
    writeFileSync(join(clientDir, 'docs.html'), html('Docs', '<p>Hub</p>'));
    writeFileSync(join(clientDir, 'preview', 'index.html'), html('Preview', '<p>Design</p>'));
    writeFileSync(join(clientDir, '404.html'), html('Not found', '<p>Missing</p>'));
    writeFileSync(
      vercelConfig,
      JSON.stringify({ version: 3, routes: [{ handle: 'filesystem' }, { src: '^/.*$', dest: '/404.html', status: 404 }] }),
    );
  });

  afterEach(() => {
    rmSync(root, { recursive: true, force: true });
  });

  it('writes a twin per page, llms-full.txt, mirrors them, and injects the Vercel routes', async () => {
    const result = await buildAgentSurface({
      clientDir,
      pathnames: ['', 'docs/faq', 'docs', 'preview', '404', 'missing/page'],
      site: 'https://reinstate.dev/',
      mirrorDirs: [vercelStatic, join(root, 'absent')],
      vercelConfigPath: vercelConfig,
    });

    expect(result.twins.map((twin) => twin.file).sort()).toEqual(['docs.md', 'docs/faq.md', 'index.md', 'preview.md']);
    expect(result.skipped).toEqual(['/404', '/missing/page']);
    expect(result.vercelRoutesInjected).toBe(true);

    const faq = readFileSync(join(clientDir, 'docs', 'faq.md'), 'utf8');
    expect(faq.startsWith('# FAQ\n\n> FAQ description\n\nAnswers')).toBe(true);
    expect(faq).toContain('Source: https://reinstate.dev/docs/faq');
    expect(readFileSync(join(vercelStatic, 'docs', 'faq.md'), 'utf8')).toBe(faq);
    expect(readFileSync(join(vercelStatic, 'index.md'), 'utf8')).toContain('# Reinstate');
    expect(existsSync(join(root, 'absent'))).toBe(false);

    const full = readFileSync(join(clientDir, 'llms-full.txt'), 'utf8');
    expect(full.startsWith('# Reinstate: full documentation\n')).toBe(true);
    expect(full).toContain('Pages: 3');
    expect(full.indexOf('<!-- https://reinstate.dev -->')).toBeLessThan(full.indexOf('<!-- https://reinstate.dev/docs -->'));
    expect(full.indexOf('<!-- https://reinstate.dev/docs -->')).toBeLessThan(full.indexOf('<!-- https://reinstate.dev/docs/faq -->'));
    expect(full).not.toContain('/preview');
    expect(full).not.toContain('Missing');
    expect(readFileSync(join(vercelStatic, 'llms-full.txt'), 'utf8')).toBe(full);

    const config = JSON.parse(readFileSync(vercelConfig, 'utf8'));
    expect(verifyAgentRoutes(config)).toEqual([]);
  });

  it('is idempotent', async () => {
    const options = { clientDir, pathnames: ['', 'docs/faq'], site: 'https://reinstate.dev', vercelConfigPath: vercelConfig };
    await buildAgentSurface(options);
    const first = readFileSync(vercelConfig, 'utf8');
    await buildAgentSurface(options);
    expect(readFileSync(vercelConfig, 'utf8')).toBe(first);
    expect(verifyAgentRoutes(JSON.parse(first))).toEqual([]);
  });

  it('locates either HTML layout Astro can emit', async () => {
    expect(await htmlFileForPage(clientDir, '/docs/faq')).toBe(join(clientDir, 'docs', 'faq', 'index.html'));
    expect(await htmlFileForPage(clientDir, '/docs')).toBe(join(clientDir, 'docs.html'));
    expect(await htmlFileForPage(clientDir, '/')).toBe(join(clientDir, 'index.html'));
    expect(await htmlFileForPage(clientDir, '/nope')).toBeNull();
  });
});

describe('ordering and exclusions', () => {
  it('puts product facts and docs first, then everything else alphabetically', () => {
    expect(fullTextOrder(['/zeta', '/docs/faq', '/', '/alpha', '/docs/getting-started', '/docs'])).toEqual([
      '/',
      '/docs',
      '/docs/getting-started',
      '/docs/faq',
      '/alpha',
      '/zeta',
    ]);
  });

  it('excludes 404 from twins and previews from the full text only', () => {
    expect(isExcludedFromTwins('/404')).toBe(true);
    expect(isExcludedFromTwins('/preview/exploded')).toBe(false);
    expect(isExcludedFromFullText('/preview/exploded')).toBe(true);
    expect(isExcludedFromFullText('/preview')).toBe(true);
    expect(isExcludedFromFullText('/docs')).toBe(false);
  });
});
