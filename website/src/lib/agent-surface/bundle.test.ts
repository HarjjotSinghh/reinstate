import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { createServer, type Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { AGENT_FUNCTION, astroRenderRuntime, bundleAgentFunction } from './bundle';

const ENTRY = resolve(import.meta.dirname, 'function.ts');

describe('bundleAgentFunction', () => {
  let root: string;

  beforeEach(() => {
    root = mkdtempSync(join(tmpdir(), 'rein-agent-bundle-'));
    mkdirSync(join(root, '_render.func'), { recursive: true });
    writeFileSync(join(root, '_render.func', '.vc-config.json'), JSON.stringify({ runtime: 'nodejs24.x', handler: 'x.mjs' }));
  });

  afterEach(() => {
    rmSync(root, { recursive: true, force: true });
  });

  it('reads the runtime the Astro adapter chose', async () => {
    expect(await astroRenderRuntime(root)).toBe('nodejs24.x');
    expect(await astroRenderRuntime(join(root, 'missing'))).toBeNull();
  });

  it('writes a self-contained Node function that negotiates like the library', async () => {
    const bundled = await bundleAgentFunction({ entry: ENTRY, functionsDir: root });
    expect(bundled.dir).toBe(join(root, `${AGENT_FUNCTION}.func`));
    expect(bundled.runtime).toBe('nodejs24.x');
    expect(bundled.bytes).toBeGreaterThan(1000);
    const config = JSON.parse(readFileSync(join(bundled.dir, '.vc-config.json'), 'utf8'));
    expect(config).toEqual({ runtime: 'nodejs24.x', handler: 'index.mjs', launcherType: 'Nodejs', shouldAddHelpers: false });
    const source = readFileSync(join(bundled.dir, 'index.mjs'), 'utf8');
    expect(source).not.toMatch(/from ['"]\.\.?\//);
    expect(existsSync(join(bundled.dir, 'package.json'))).toBe(true);

    // Serve a fake static origin, then drive the bundled handler against it.
    const origin: Server = createServer((req, res) => {
      if (req.url === '/docs/faq.md') {
        res.writeHead(200, { 'content-type': 'text/markdown; charset=utf-8' });
        res.end('# FAQ\n');
        return;
      }
      res.writeHead(404, { 'content-type': 'text/plain' });
      res.end('nope');
    });
    await new Promise<void>((done) => origin.listen(0, '127.0.0.1', done));
    const originHost = `127.0.0.1:${(origin.address() as AddressInfo).port}`;

    const { default: handler } = (await import(pathToFileURL(join(bundled.dir, 'index.mjs')).href)) as {
      default: (req: import('node:http').IncomingMessage, res: import('node:http').ServerResponse) => Promise<void>;
    };
    const fn: Server = createServer(handler);
    await new Promise<void>((done) => fn.listen(0, '127.0.0.1', done));
    const fnBase = `http://127.0.0.1:${(fn.address() as AddressInfo).port}`;
    try {
      const markdown = await fetch(`${fnBase}/docs/faq`, {
        headers: { accept: 'text/markdown', 'x-forwarded-host': originHost, 'x-forwarded-proto': 'http' },
      });
      expect(markdown.status).toBe(200);
      expect(markdown.headers.get('content-type')).toBe('text/markdown; charset=utf-8');
      expect(await markdown.text()).toBe('# FAQ\n');

      const missing = await fetch(`${fnBase}/nope`, { headers: { 'x-forwarded-host': originHost, 'x-forwarded-proto': 'http' } });
      expect(missing.status).toBe(404);
      expect(missing.headers.get('content-type')).toBe('text/markdown; charset=utf-8');
    } finally {
      await new Promise<void>((done) => fn.close(() => done()));
      await new Promise<void>((done) => origin.close(() => done()));
    }
  });

  it('falls back to the default runtime without an Astro function', async () => {
    rmSync(join(root, '_render.func'), { recursive: true, force: true });
    const bundled = await bundleAgentFunction({ entry: ENTRY, functionsDir: root });
    expect(bundled.runtime).toBe('nodejs22.x');
  });
});
