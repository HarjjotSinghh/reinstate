import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { checkAgentSurface, pagePathOfHtml, twinFileFor } from './check-agent-surface.mjs';

const LLMS = `# Reinstate

> Reinstate syncs encrypted coding-agent sessions across devices.

**When to use Reinstate:** continuity jobs. **When not to use Reinstate:** Git jobs. **How an agent should call it:** in the shell.

## Developer and agent resources

- [Developer resources](https://reinstate.dev/developers): hub
- [Agent instructions](https://reinstate.dev/agent-instructions.md): guidance
- [OpenAPI document](https://reinstate.dev/openapi.json): spec
- [Full documentation as Markdown](https://reinstate.dev/llms-full.txt): all docs
- [Compatibility JSON](https://reinstate.dev/compatibility.json): data
- [CLI reference](https://reinstate.dev/docs/cli-reference): commands

## Documentation

- [Getting started](https://reinstate.dev/docs/getting-started): install

## Source

- [GitHub repository](https://github.com/HarjjotSinghh/reinstate)
`;

const OPENAPI = {
  openapi: '3.1.0',
  servers: [{ url: 'https://reinstate.dev' }],
  paths: {
    '/api/waitlist': {
      get: { operationId: 'getWaitlistService', description: 'Describe.', responses: { 200: { description: 'ok' } } },
    },
  },
};

const CONFIG = {
  version: 3,
  routes: [
    { src: '^(/[^.]*)$', has: [{ type: 'header', key: 'accept', value: '.*text/markdown.*' }], methods: ['GET', 'HEAD'], dest: 'agent-surface' },
    { handle: 'filesystem' },
    { src: '^(/[^.]*)$', missing: [{ type: 'header', key: 'accept', value: '.*text/html.*' }], methods: ['GET', 'HEAD'], dest: 'agent-surface' },
    { src: '^/api/(.*?)/?$', dest: '_render' },
    { src: '^/.*$', dest: '/404.html', status: 404 },
  ],
};

const html = (title) => `<!doctype html><html><head><title>${title}</title></head><body><main><h1>${title}</h1></main></body></html>`;
const twin = (path) => `# Page\n\nBody.\n\n---\n\nSource: https://reinstate.dev${path}\n`;

async function fixture({ withTwins = true, withVercel = true, withFunction = true, llms = LLMS, openapi = OPENAPI, config = CONFIG } = {}) {
  const root = await mkdtemp(join(tmpdir(), 'rein-check-agent-'));
  const build = join(root, 'dist', 'client');
  const pages = ['/', '/docs/getting-started', '/docs/cli-reference', '/developers', '/integrations/codex'];
  for (const page of pages) {
    const dir = page === '/' ? build : join(build, page.slice(1));
    await mkdir(dir, { recursive: true });
    await writeFile(join(dir, 'index.html'), html(page));
    if (withTwins) await writeFile(join(build, twinFileFor(page)), twin(page));
  }
  await writeFile(join(build, '404.html'), html('404'));
  await writeFile(join(build, 'llms.txt'), llms);
  await writeFile(join(build, 'llms-full.txt'), `# Reinstate: full documentation\n\n<!-- https://reinstate.dev/docs/getting-started -->\n\n<!-- https://reinstate.dev/developers -->\n`);
  await writeFile(join(build, 'openapi.json'), JSON.stringify(openapi));
  await writeFile(join(build, 'agent-instructions.md'), '# Agents\n\n## When to use Reinstate\n\n## When not to use Reinstate\n\n## How to call it\n');
  await writeFile(join(build, 'compatibility.json'), JSON.stringify({ agents: [{ id: 'codex', integrationPath: '/integrations/codex', tier: 'T5' }] }));
  await writeFile(join(build, 'robots.txt'), 'User-agent: *\nDisallow: /api/\n');
  await writeFile(
    join(root, 'vercel.json'),
    JSON.stringify({
      headers: [
        { source: '/:path([^.]*)', headers: [{ key: 'Vary', value: 'Accept' }] },
        { source: '/:path*.md', headers: [{ key: 'Content-Type', value: 'text/markdown; charset=utf-8' }] },
      ],
    }),
  );
  if (withVercel) {
    const vercel = join(root, '.vercel', 'output');
    await mkdir(join(vercel, 'static'), { recursive: true });
    for (const page of pages) {
      const file = join(vercel, 'static', twinFileFor(page));
      await mkdir(join(file, '..'), { recursive: true });
      if (withTwins) await writeFile(file, twin(page));
    }
    await writeFile(join(vercel, 'config.json'), JSON.stringify(config));
    await mkdir(join(vercel, 'functions', '_render.func'), { recursive: true });
    await writeFile(join(vercel, 'functions', '_render.func', '.vc-config.json'), '{}');
    if (withFunction) {
      await mkdir(join(vercel, 'functions', 'agent-surface.func'), { recursive: true });
      await writeFile(join(vercel, 'functions', 'agent-surface.func', '.vc-config.json'), '{}');
      await writeFile(join(vercel, 'functions', 'agent-surface.func', 'index.mjs'), 'export default () => {};\n');
    }
  }
  return root;
}

test('pagePathOfHtml and twinFileFor agree with the build layout', () => {
  assert.equal(pagePathOfHtml('index.html'), '/');
  assert.equal(pagePathOfHtml('docs/faq/index.html'), '/docs/faq');
  assert.equal(pagePathOfHtml('docs.html'), '/docs');
  assert.equal(pagePathOfHtml('robots.txt'), null);
  assert.equal(twinFileFor('/'), 'index.md');
  assert.equal(twinFileFor('/docs/faq'), 'docs/faq.md');
});

test('a complete build passes', async () => {
  const root = await fixture();
  try {
    const result = await checkAgentSurface({ root });
    assert.deepEqual(result.errors, []);
    assert.equal(result.ok, true);
    assert.equal(result.summary.pages, 5);
    assert.equal(result.summary.twins, 5);
    assert.equal(result.summary.openApiOperations, 1);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('missing twins, routes, and llms.txt guidance are reported with fixes', async () => {
  const badConfig = { version: 3, routes: [{ handle: 'filesystem' }, { src: '^/.*$', dest: '/404.html', status: 404 }] };
  const root = await fixture({ withTwins: false, llms: LLMS.replace('**When to use Reinstate:** continuity jobs. ', ''), config: badConfig });
  try {
    const result = await checkAgentSurface({ root });
    assert.equal(result.ok, false);
    assert.ok(result.errors.some((error) => error.includes('/docs/getting-started: missing Markdown twin docs/getting-started.md')));
    assert.ok(result.errors.some((error) => error.includes('markdown negotiation route missing')));
    assert.ok(result.errors.some((error) => error.includes('markdown not-found route missing')));
    assert.ok(result.errors.some((error) => error.includes('"When to use Reinstate"')));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('misordered Vercel routes and unknown llms.txt targets fail', async () => {
  const misordered = { ...CONFIG, routes: [CONFIG.routes[1], CONFIG.routes[0], CONFIG.routes[3], CONFIG.routes[4], CONFIG.routes[2]] };
  const root = await fixture({ config: misordered, llms: `${LLMS}- [Ghost](https://reinstate.dev/ghost): missing\n` });
  try {
    const result = await checkAgentSurface({ root });
    assert.ok(result.errors.some((error) => error.includes('must precede the filesystem handle')));
    assert.ok(result.errors.some((error) => error.includes('must precede the 404 catch-all')));
    assert.ok(result.errors.some((error) => error.includes('link target /ghost is not in the build output')));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('a missing agent function fails', async () => {
  const root = await fixture({ withFunction: false });
  try {
    const result = await checkAgentSurface({ root });
    assert.ok(result.errors.some((error) => error.includes('agent-surface.func/index.mjs: missing')));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('non-standard route keys fail', async () => {
  const tagged = { ...CONFIG, routes: CONFIG.routes.map((route, index) => (index === 0 ? { ...route, 'x-reinstate-agent-route': 'markdown' } : route)) };
  const root = await fixture({ config: tagged });
  try {
    const result = await checkAgentSurface({ root });
    assert.ok(result.errors.some((error) => error.includes('non-standard key "x-reinstate-agent-route"')));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('a tier claim that disagrees with compatibility.json fails', async () => {
  const root = await fixture({ llms: `${LLMS}\n## Integrations\n\n- [Codex](https://reinstate.dev/integrations/codex): T2 handoff source\n` });
  try {
    const result = await checkAgentSurface({ root });
    assert.ok(result.errors.some((error) => error.includes('/integrations/codex claims T2 but compatibility.json says T5')));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('duplicate operationIds and a stray 404 twin fail', async () => {
  const dupe = {
    ...OPENAPI,
    paths: {
      '/a': { get: { operationId: 'same', description: 'A.', responses: { 200: { description: 'ok' } } } },
      '/b': { get: { operationId: 'same', description: 'B.', responses: { 200: { description: 'ok' } } } },
    },
  };
  const root = await fixture({ openapi: dupe });
  try {
    await writeFile(join(root, 'dist', 'client', '404.md'), '# nope');
    const result = await checkAgentSurface({ root });
    assert.ok(result.errors.some((error) => error.includes('operationIds must be unique')));
    assert.ok(result.errors.some((error) => error.startsWith('404.md:')));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('a missing build directory is reported instead of thrown', async () => {
  const root = await mkdtemp(join(tmpdir(), 'rein-check-agent-empty-'));
  try {
    const result = await checkAgentSurface({ root });
    assert.equal(result.ok, false);
    assert.match(result.errors[0], /build output missing/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
