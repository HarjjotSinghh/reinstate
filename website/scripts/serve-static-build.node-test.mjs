import assert from 'node:assert/strict';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { request } from 'node:http';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  createStaticPreviewServer,
  resolveStaticRequest,
} from './serve-static-build.mjs';

async function writeFixture(root, path, value) {
  const destination = join(root, path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, value);
}

async function buildFixture() {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-preview-'));
  await writeFixture(root, 'index.html', '<h1>Home</h1>');
  await writeFixture(root, 'docs/index.html', '<h1>Docs</h1>');
  await writeFixture(root, '_astro/site.css', 'body{}');
  await writeFixture(root, '404.html', '<h1>Missing</h1>');
  return root;
}

function listen(server) {
  return new Promise((resolveListen, rejectListen) => {
    server.once('error', rejectListen);
    server.listen(0, '127.0.0.1', () => resolveListen(server.address()));
  });
}

function close(server) {
  return new Promise((resolveClose, rejectClose) => {
    server.close((error) => error ? rejectClose(error) : resolveClose());
  });
}

function get(port, path, method = 'GET') {
  return new Promise((resolveRequest, rejectRequest) => {
    const outgoing = request(
      {
        headers: { connection: 'close' },
        host: '127.0.0.1',
        method,
        path,
        port,
      },
      (response) => {
        let body = '';
        response.setEncoding('utf8');
        response.on('data', (chunk) => {
          body += chunk;
        });
        response.on('end', () =>
          resolveRequest({
            body,
            headers: response.headers,
            status: response.statusCode,
          }),
        );
      },
    );
    outgoing.on('error', rejectRequest);
    outgoing.end();
  });
}

test('resolves clean routes and assets inside the generated build only', async (t) => {
  const root = await buildFixture();
  t.after(() => rm(root, { force: true, recursive: true }));

  assert.equal((await resolveStaticRequest(root, '/')).status, 200);
  assert.equal((await resolveStaticRequest(root, '/docs')).status, 200);
  assert.equal((await resolveStaticRequest(root, '/docs/')).status, 200);
  assert.equal((await resolveStaticRequest(root, '/_astro/site.css')).status, 200);
  assert.equal((await resolveStaticRequest(root, '/missing')).status, 404);
  assert.equal((await resolveStaticRequest(root, '/../package.json')).status, 400);
  assert.equal(
    (await resolveStaticRequest(root, '/%2e%2e%2fpackage.json')).status,
    400,
  );
  assert.equal((await resolveStaticRequest(root, '/bad%ZZ')).status, 400);
});

test('serves GET and HEAD requests with useful QA headers and a real 404', async (t) => {
  const root = await buildFixture();
  const server = createStaticPreviewServer({ buildRoot: root });
  t.after(async () => {
    await close(server);
    await rm(root, { force: true, recursive: true });
  });
  const address = await listen(server);
  assert.equal(typeof address, 'object');

  const page = await get(address.port, '/docs');
  assert.equal(page.status, 200);
  assert.equal(page.body, '<h1>Docs</h1>');
  assert.equal(page.headers['content-type'], 'text/html; charset=utf-8');
  assert.equal(page.headers['cache-control'], 'no-store');

  const head = await get(address.port, '/_astro/site.css', 'HEAD');
  assert.equal(head.status, 200);
  assert.equal(head.body, '');
  assert.equal(head.headers['content-type'], 'text/css; charset=utf-8');

  const missing = await get(address.port, '/nope');
  assert.equal(missing.status, 404);
  assert.equal(missing.body, '<h1>Missing</h1>');
});
