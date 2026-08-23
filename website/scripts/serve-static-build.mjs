import { createReadStream } from 'node:fs';
import { access, realpath, stat } from 'node:fs/promises';
import { createServer } from 'node:http';
import { extname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const DEFAULT_BUILD_ROOT = resolve(fileURLToPath(new URL('../dist/client/', import.meta.url)));
const DEFAULT_HOST = '127.0.0.1';
const DEFAULT_PORT = 4321;
const CONTENT_TYPES = new Map([
  ['.avif', 'image/avif'],
  ['.css', 'text/css; charset=utf-8'],
  ['.gif', 'image/gif'],
  ['.html', 'text/html; charset=utf-8'],
  ['.ico', 'image/x-icon'],
  ['.jpeg', 'image/jpeg'],
  ['.jpg', 'image/jpeg'],
  ['.js', 'text/javascript; charset=utf-8'],
  ['.json', 'application/json; charset=utf-8'],
  ['.map', 'application/json; charset=utf-8'],
  ['.md', 'text/markdown; charset=utf-8'],
  ['.png', 'image/png'],
  ['.ps1', 'text/plain; charset=utf-8'],
  ['.sh', 'text/x-shellscript; charset=utf-8'],
  ['.svg', 'image/svg+xml'],
  ['.txt', 'text/plain; charset=utf-8'],
  ['.webp', 'image/webp'],
  ['.woff', 'font/woff'],
  ['.woff2', 'font/woff2'],
  ['.xml', 'application/xml; charset=utf-8'],
]);

function isWithinRoot(root, candidate) {
  return candidate === root || candidate.startsWith(`${root}${sep}`);
}

async function isFile(path) {
  try {
    return (await stat(path)).isFile();
  } catch {
    return false;
  }
}

/**
 * Resolve a URL pathname to prerendered output without ever escaping the
 * generated build directory.
 */
export async function resolveStaticRequest(
  buildRoot,
  requestPathname,
) {
  const root = await realpath(resolve(buildRoot));
  let pathname;
  try {
    pathname = decodeURIComponent(requestPathname);
  } catch {
    return { status: 400 };
  }

  if (
    !pathname.startsWith('/') ||
    pathname.includes('\0') ||
    pathname.includes('\\')
  ) {
    return { status: 400 };
  }

  const relativePath = pathname.replace(/^\/+/, '');
  const requested = resolve(root, relativePath);
  if (!isWithinRoot(root, requested)) {
    return { status: 400 };
  }

  const candidates = pathname === '/'
    ? [resolve(root, 'index.html')]
    : extname(pathname)
      ? [requested]
      : [resolve(requested, 'index.html')];

  for (const candidate of candidates) {
    if (isWithinRoot(root, candidate) && await isFile(candidate)) {
      const realCandidate = await realpath(candidate);
      if (!isWithinRoot(root, realCandidate)) {
        return { status: 400 };
      }
      return { status: 200, filePath: realCandidate };
    }
  }

  const notFound = resolve(root, '404.html');
  if (await isFile(notFound)) {
    const realNotFound = await realpath(notFound);
    if (!isWithinRoot(root, realNotFound)) {
      return { status: 404 };
    }
    return { status: 404, filePath: realNotFound };
  }
  return { status: 404 };
}

function writePlainResponse(response, status, message) {
  response.writeHead(status, {
    'Cache-Control': 'no-store',
    'Content-Type': 'text/plain; charset=utf-8',
    'X-Content-Type-Options': 'nosniff',
  });
  response.end(message);
}

export function createStaticPreviewServer({ buildRoot = DEFAULT_BUILD_ROOT } = {}) {
  return createServer(async (request, response) => {
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      response.writeHead(405, {
        Allow: 'GET, HEAD',
        'Cache-Control': 'no-store',
        'Content-Type': 'text/plain; charset=utf-8',
        'X-Content-Type-Options': 'nosniff',
      });
      response.end('Method not allowed.\n');
      return;
    }

    let pathname;
    try {
      pathname = new URL(request.url ?? '/', 'http://preview.invalid').pathname;
    } catch {
      writePlainResponse(response, 400, 'Bad request.\n');
      return;
    }

    const resolved = await resolveStaticRequest(buildRoot, pathname);
    if (!resolved.filePath) {
      writePlainResponse(
        response,
        resolved.status,
        resolved.status === 400 ? 'Bad request.\n' : 'Not found.\n',
      );
      return;
    }

    const details = await stat(resolved.filePath);
    response.writeHead(resolved.status, {
      'Cache-Control': 'no-store',
      'Content-Length': details.size,
      'Content-Type':
        CONTENT_TYPES.get(extname(resolved.filePath).toLowerCase()) ??
        'application/octet-stream',
      'X-Content-Type-Options': 'nosniff',
    });
    if (request.method === 'HEAD') {
      response.end();
      return;
    }
    createReadStream(resolved.filePath).pipe(response);
  });
}

function parseArguments(argv) {
  const options = {
    buildRoot: DEFAULT_BUILD_ROOT,
    host: process.env.PREVIEW_HOST || DEFAULT_HOST,
    port: Number(process.env.PREVIEW_PORT || DEFAULT_PORT),
  };
  const valueOptions = new Set(['--build-root', '--host', '--port']);

  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === '--help' || argument === '-h') {
      options.help = true;
      continue;
    }
    if (!valueOptions.has(argument)) {
      throw new Error(`Unknown preview option: ${argument}`);
    }
    const value = argv[index + 1];
    if (!value || value.startsWith('--')) {
      throw new Error(`${argument} requires a value.`);
    }
    index += 1;
    if (argument === '--build-root') options.buildRoot = resolve(value);
    if (argument === '--host') options.host = value;
    if (argument === '--port') options.port = Number(value);
  }

  if (
    !Number.isInteger(options.port) ||
    options.port < 1 ||
    options.port > 65_535
  ) {
    throw new Error('Preview port must be an integer between 1 and 65535.');
  }
  return options;
}

async function main() {
  const options = parseArguments(process.argv.slice(2));
  if (options.help) {
    console.log(`Serve Reinstate's prerendered Astro output

Usage:
  node scripts/serve-static-build.mjs [--host 127.0.0.1] [--port 4321]
    [--build-root dist/client]

This static QA server does not execute server routes such as /api/waitlist,
/rss.xml, or the IndexNow key-proof endpoint.`);
    return;
  }

  await access(options.buildRoot);
  const server = createStaticPreviewServer({ buildRoot: options.buildRoot });
  await new Promise((resolveListen, rejectListen) => {
    server.once('error', rejectListen);
    server.listen(options.port, options.host, resolveListen);
  });
  console.log(
    `Static build available at http://${options.host}:${options.port} (Ctrl+C to stop)`,
  );
  const close = () => server.close();
  process.once('SIGINT', close);
  process.once('SIGTERM', close);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(SCRIPT_PATH)) {
  main().catch((error) => {
    console.error(
      `Static preview error: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
    process.exitCode = 1;
  });
}
