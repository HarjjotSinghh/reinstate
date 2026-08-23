#!/usr/bin/env node
/**
 * Post-build audit of the agent-facing surface. Dependency-free; reads the
 * generated site and the Vercel Build Output and reports every gap with the
 * file and the fix, in the same spirit as `check-seo.mjs`.
 *
 *   npm run build
 *   npm run check:agent-surface
 */
import { readdir, readFile, stat } from 'node:fs/promises';
import { dirname, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_PATH = fileURLToPath(import.meta.url);

export const REQUIRED_LLMS_SECTIONS = ['## Developer and agent resources', '## Documentation', '## Source'];
export const REQUIRED_LLMS_LINKS = [
  '/developers',
  '/agent-instructions.md',
  '/openapi.json',
  '/llms-full.txt',
  '/compatibility.json',
  '/docs/cli-reference',
];
export const REQUIRED_NOT_FOUND_DISALLOW = ['/api/'];

async function exists(path) {
  try {
    await stat(path);
    return true;
  } catch {
    return false;
  }
}

async function walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) files.push(...(await walk(path)));
    else files.push(path);
  }
  return files;
}

function toPosix(path) {
  return path.split(sep).join('/');
}

/** Page path of a generated HTML file: `docs/faq/index.html` → `/docs/faq`, `docs.html` → `/docs`. */
export function pagePathOfHtml(relativeFile) {
  const posix = toPosix(relativeFile);
  if (posix === 'index.html') return '/';
  if (posix.endsWith('/index.html')) return `/${posix.slice(0, -'/index.html'.length)}`;
  if (posix.endsWith('.html')) return `/${posix.slice(0, -'.html'.length)}`;
  return null;
}

export function twinFileFor(pagePath) {
  return pagePath === '/' ? 'index.md' : `${pagePath.slice(1)}.md`;
}

function llmsLinks(text) {
  return [...text.matchAll(/\]\((https:\/\/reinstate\.dev[^)\s]*)\)/g)].map((match) => match[1]);
}

function routeOfUrl(url) {
  const { pathname } = new URL(url);
  return pathname.length > 1 && pathname.endsWith('/') ? pathname.slice(0, -1) : pathname;
}

export async function checkAgentSurface({ root, buildDir = resolve(root, 'dist/client'), vercelDir = resolve(root, '.vercel/output') } = {}) {
  const errors = [];
  const summary = { pages: 0, twins: 0, llmsFullBytes: 0, openApiOperations: 0 };

  if (!(await exists(buildDir))) {
    return { ok: false, errors: [`${buildDir}: build output missing; run \`npm run build\` first`], summary };
  }

  const files = await walk(buildDir);
  const relativeFiles = new Set(files.map((file) => toPosix(relative(buildDir, file))));
  const pagePaths = [];
  for (const file of relativeFiles) {
    const pagePath = pagePathOfHtml(file);
    if (pagePath && pagePath !== '/404') pagePaths.push(pagePath);
  }
  summary.pages = pagePaths.length;

  for (const pagePath of pagePaths) {
    const twin = twinFileFor(pagePath);
    if (!relativeFiles.has(twin)) {
      errors.push(`${pagePath}: missing Markdown twin ${twin}; the reinstate-agent-surface integration did not run or skipped the page`);
      continue;
    }
    const markdown = await readFile(join(buildDir, twin), 'utf8');
    if (!/^# \S/m.test(markdown)) errors.push(`${twin}: Markdown twin must start with a level-one heading`);
    if (!markdown.includes(`Source: https://reinstate.dev${pagePath === '/' ? '/' : pagePath}`)) {
      errors.push(`${twin}: Markdown twin must end with its canonical source URL`);
    }
    if (/<(?:script|style|svg)\b/i.test(markdown)) errors.push(`${twin}: Markdown twin still contains raw markup`);
    summary.twins += 1;
  }
  if (relativeFiles.has('404.md')) errors.push('404.md: the not-found page must not get a Markdown twin (the runtime serves the Markdown 404)');

  if (await exists(vercelDir)) {
    for (const pagePath of pagePaths) {
      const twin = twinFileFor(pagePath);
      if (!(await exists(join(vercelDir, 'static', twin)))) errors.push(`.vercel/output/static/${twin}: twin missing from the Vercel static output`);
    }
    const configPath = join(vercelDir, 'config.json');
    if (!(await exists(configPath))) {
      errors.push('.vercel/output/config.json: missing');
    } else {
      const config = JSON.parse(await readFile(configPath, 'utf8'));
      const routes = config.routes ?? [];
      const filesystemIndex = routes.findIndex((route) => route.handle === 'filesystem');
      const accepts = (route, list, pattern) => route.dest === 'agent-surface' && Array.isArray(route[list]) && route[list].some((c) => c.key === 'accept' && c.value === pattern);
      const isAgent = (route, kind) => (kind === 'markdown' ? accepts(route, 'has', '.*text/markdown.*') : accepts(route, 'missing', '.*text/html.*'));
      const markdownIndex = routes.findIndex((route) => isAgent(route, 'markdown'));
      const notFoundIndex = routes.findIndex((route) => isAgent(route, 'not-found'));
      const catchAllIndex = routes.findIndex((route) => route.status === 404 && /^\^?\/\.\*\$?$/.test(route.src ?? ''));
      const renderIndexes = routes.map((route, index) => (route.dest === '_render' && /agent-surface/.test(route.src ?? '') ? index : -1)).filter((index) => index !== -1);
      if (markdownIndex === -1) errors.push('.vercel/output/config.json: markdown negotiation route missing');
      else if (filesystemIndex !== -1 && markdownIndex > filesystemIndex) errors.push('.vercel/output/config.json: markdown negotiation route must precede the filesystem handle');
      if (notFoundIndex === -1) errors.push('.vercel/output/config.json: markdown not-found route missing');
      else {
        if (notFoundIndex < filesystemIndex) errors.push('.vercel/output/config.json: markdown not-found route must follow the filesystem handle');
        if (catchAllIndex !== -1 && notFoundIndex > catchAllIndex) errors.push('.vercel/output/config.json: markdown not-found route must precede the 404 catch-all');
      }
      for (const file of ['.vc-config.json', 'index.mjs']) {
        if (!(await exists(join(vercelDir, 'functions', 'agent-surface.func', file)))) errors.push(`.vercel/output/functions/agent-surface.func/${file}: missing; the reinstate-agent-surface integration must bundle the agent function`);
      }
      const duplicates = routes.filter((route) => isAgent(route, 'markdown') || isAgent(route, 'not-found')).length;
      if (duplicates > 2) errors.push(`.vercel/output/config.json: agent routes injected ${duplicates} times; expected 2`);
      const allowedKeys = new Set(['src', 'dest', 'has', 'missing', 'methods', 'check', 'continue', 'headers', 'status', 'handle', 'locale', 'caseSensitive', 'important', 'override', 'middlewarePath', 'middlewareRawSrc', 'transforms', 'mitigate']);
      for (const route of routes) {
        for (const key of Object.keys(route)) {
          if (!allowedKeys.has(key)) errors.push(`.vercel/output/config.json: route ${route.src ?? route.handle} has non-standard key "${key}"; Vercel rejects it as invalid_routes when merging vercel.json`);
        }
      }
    }
  }

  const llmsFull = join(buildDir, 'llms-full.txt');
  if (!(await exists(llmsFull))) {
    errors.push('llms-full.txt: missing from the build output');
  } else {
    const text = await readFile(llmsFull, 'utf8');
    summary.llmsFullBytes = Buffer.byteLength(text, 'utf8');
    if (!text.startsWith('# Reinstate')) errors.push('llms-full.txt: must start with the Reinstate heading');
    for (const required of ['<!-- https://reinstate.dev/docs/getting-started -->', '<!-- https://reinstate.dev/developers -->']) {
      if (!text.includes(required)) errors.push(`llms-full.txt: missing section ${required}`);
    }
    if (text.includes('<!-- https://reinstate.dev/preview')) errors.push('llms-full.txt: design previews must not be included');
  }

  const llmsPath = join(buildDir, 'llms.txt');
  if (!(await exists(llmsPath))) {
    errors.push('llms.txt: missing from the build output');
  } else {
    const text = await readFile(llmsPath, 'utf8');
    const lines = text.split('\n');
    if (!lines[0]?.startsWith('# ')) errors.push('llms.txt: first line must be the H1 project name');
    const blockquote = lines.findIndex((line) => line.startsWith('> '));
    const firstSection = lines.findIndex((line) => line.startsWith('## '));
    if (blockquote === -1 || blockquote > firstSection) errors.push('llms.txt: the blockquote summary must follow the H1 and precede the first section');
    const preamble = lines.slice(0, firstSection === -1 ? lines.length : firstSection).join('\n');
    if (!/When to use Reinstate/i.test(preamble)) errors.push('llms.txt: add a "When to use Reinstate" paragraph before the first H2 section');
    if (!/When not to use Reinstate/i.test(preamble)) errors.push('llms.txt: add a "When not to use Reinstate" paragraph before the first H2 section');
    if (!/How an agent should call it/i.test(preamble)) errors.push('llms.txt: add a "How an agent should call it" paragraph before the first H2 section');
    for (const section of REQUIRED_LLMS_SECTIONS) {
      if (!lines.includes(section)) errors.push(`llms.txt: missing section "${section}"`);
    }
    const links = llmsLinks(text);
    for (const required of REQUIRED_LLMS_LINKS) {
      if (!links.some((url) => routeOfUrl(url) === required)) errors.push(`llms.txt: missing link to ${required}`);
    }
    for (const url of links) {
      const route = routeOfUrl(url);
      if (route.startsWith('/api/')) continue; // runtime routes are functions, not static files
      const candidates = route === '/' ? ['index.html'] : [`${route.slice(1)}/index.html`, `${route.slice(1)}.html`, route.slice(1)];
      if (!candidates.some((candidate) => relativeFiles.has(candidate))) errors.push(`llms.txt: link target ${route} is not in the build output`);
    }
    // Tier claims in integration lines must match the reviewed compatibility data.
    const compatibilityPath = join(buildDir, 'compatibility.json');
    if (await exists(compatibilityPath)) {
      try {
        const compatibility = JSON.parse(await readFile(compatibilityPath, 'utf8'));
        const tiers = new Map((compatibility.agents ?? []).filter((agent) => agent.integrationPath).map((agent) => [agent.integrationPath, agent.tier]));
        for (const line of lines) {
          const match = line.match(/^- \[[^\]]+\]\((https:\/\/reinstate\.dev\/integrations\/[a-z0-9-]+)\):\s*(.*)$/);
          if (!match) continue;
          const route = routeOfUrl(match[1]);
          const claimed = match[2].match(/\bT([0-5])\b/);
          const actual = tiers.get(route);
          if (claimed && actual && `T${claimed[1]}` !== actual) {
            errors.push(`llms.txt: ${route} claims ${`T${claimed[1]}`} but compatibility.json says ${actual}`);
          }
        }
      } catch (error) {
        errors.push(`compatibility.json: ${error.message}`);
      }
    }

    const bodyLines = lines.slice(firstSection === -1 ? 0 : firstSection);
    for (const line of bodyLines) {
      if (line.startsWith('#') && !line.startsWith('## ')) errors.push(`llms.txt: only H2 sections are allowed after the preamble (found "${line}")`);
    }
  }

  const openApiPath = join(buildDir, 'openapi.json');
  if (!(await exists(openApiPath))) {
    errors.push('openapi.json: missing from the build output');
  } else {
    try {
      const document = JSON.parse(await readFile(openApiPath, 'utf8'));
      if (!/^3\.1\./.test(document.openapi ?? '')) errors.push('openapi.json: must declare OpenAPI 3.1');
      const ids = [];
      for (const [path, item] of Object.entries(document.paths ?? {})) {
        for (const [method, operation] of Object.entries(item)) {
          if (!['get', 'post', 'put', 'patch', 'delete', 'head', 'options'].includes(method)) continue;
          summary.openApiOperations += 1;
          if (!operation.operationId) errors.push(`openapi.json: ${method.toUpperCase()} ${path} lacks an operationId`);
          else ids.push(operation.operationId);
          if (!operation.description) errors.push(`openapi.json: ${method.toUpperCase()} ${path} lacks a description`);
          if (!operation.responses || Object.keys(operation.responses).length === 0) errors.push(`openapi.json: ${method.toUpperCase()} ${path} lacks responses`);
        }
      }
      if (new Set(ids).size !== ids.length) errors.push('openapi.json: operationIds must be unique');
      if (!document.servers?.some((server) => server.url === 'https://reinstate.dev')) errors.push('openapi.json: servers must include https://reinstate.dev');
    } catch (error) {
      errors.push(`openapi.json: ${error.message}`);
    }
  }

  if (!relativeFiles.has('agent-instructions.md')) {
    errors.push('agent-instructions.md: missing from the build output');
  } else {
    const text = await readFile(join(buildDir, 'agent-instructions.md'), 'utf8');
    for (const heading of ['## When to use Reinstate', '## When not to use Reinstate', '## How to call it']) {
      if (!text.includes(heading)) errors.push(`agent-instructions.md: missing section "${heading}"`);
    }
  }

  const robotsPath = join(buildDir, 'robots.txt');
  if (await exists(robotsPath)) {
    const robots = await readFile(robotsPath, 'utf8');
    for (const disallow of REQUIRED_NOT_FOUND_DISALLOW) {
      if (!robots.includes(`Disallow: ${disallow}`)) errors.push(`robots.txt: add "Disallow: ${disallow}" so runtime endpoints stay out of search indexes`);
    }
  }

  const vercelJsonPath = resolve(root, 'vercel.json');
  if (await exists(vercelJsonPath)) {
    const config = JSON.parse(await readFile(vercelJsonPath, 'utf8'));
    const headerRules = config.headers ?? [];
    const varyRule = headerRules.find((rule) => rule.headers?.some((header) => header.key.toLowerCase() === 'vary' && header.value === 'Accept'));
    if (!varyRule) errors.push('vercel.json: add a headers rule that sets Vary: Accept on page URLs');
    const markdownRule = headerRules.find((rule) => rule.source?.endsWith('.md'));
    if (!markdownRule?.headers?.some((header) => header.key.toLowerCase() === 'content-type' && header.value.startsWith('text/markdown'))) {
      errors.push('vercel.json: add a headers rule that serves *.md as text/markdown; charset=utf-8');
    }
  }

  return { ok: errors.length === 0, errors, summary };
}

async function main() {
  const root = resolve(dirname(SCRIPT_PATH), '..');
  const result = await checkAgentSurface({ root });
  const { summary } = result;
  console.log(
    `Agent surface: ${summary.pages} pages, ${summary.twins} Markdown twins, llms-full.txt ${Math.round(summary.llmsFullBytes / 1024)} KiB, ${summary.openApiOperations} OpenAPI operations`,
  );
  if (!result.ok) {
    console.error(`\n${result.errors.length} agent-surface problem${result.errors.length === 1 ? '' : 's'}:`);
    for (const error of result.errors) console.error(`- ${error}`);
    process.exitCode = 1;
    return;
  }
  console.log('Agent surface OK.');
}

if (process.argv[1] && resolve(process.argv[1]) === SCRIPT_PATH) {
  main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
}
