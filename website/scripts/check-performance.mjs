#!/usr/bin/env node

import { readFile, stat } from 'node:fs/promises';
import { dirname, posix, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { gzipSync } from 'node:zlib';

const DEFAULT_BUILD_DIR = resolve(process.cwd(), 'dist/client');
const SITE_ORIGIN = 'https://reinstate.dev';
const KIB = 1024;

const SHARED_LIMITS = {
  executableJsRaw: 16 * KIB,
  executableJsGzip: 6 * KIB,
  blockingScriptCount: 1,
  blockingScriptRaw: 8 * KIB,
  fontCount: 16,
  fontRaw: 240 * KIB,
  fontGzip: 245 * KIB,
  largestFontRaw: 48 * KIB,
  fontPreloadCount: 2,
  externalAssetCount: 1,
};

const DOC_LIMITS = {
  ...SHARED_LIMITS,
  htmlRaw: 64 * KIB,
  htmlGzip: 14 * KIB,
  cssCodeRaw: 80 * KIB,
  cssCodeGzip: 18 * KIB,
  mediaRaw: 64 * KIB,
  mediaGzip: 60 * KIB,
  staticTransferRaw: 220 * KIB,
  staticTransferGzip: 90 * KIB,
  blockingStyleCount: 3,
  localAssetRequestCount: 6,
};

const PUBLIC_CONTENT_LIMITS = {
  ...SHARED_LIMITS,
  htmlRaw: 72 * KIB,
  htmlGzip: 16 * KIB,
  cssCodeRaw: 100 * KIB,
  cssCodeGzip: 22 * KIB,
  mediaRaw: 64 * KIB,
  mediaGzip: 60 * KIB,
  staticTransferRaw: 250 * KIB,
  staticTransferGzip: 100 * KIB,
  blockingStyleCount: 4,
  localAssetRequestCount: 7,
};

export const DEFAULT_ROUTE_DEFINITIONS = [
  {
    path: '/',
    label: 'Homepage',
    required: true,
    budget: {
      ...SHARED_LIMITS,
      htmlRaw: 200 * KIB,
      htmlGzip: 35 * KIB,
      cssCodeRaw: 140 * KIB,
      cssCodeGzip: 28 * KIB,
      mediaRaw: 128 * KIB,
      mediaGzip: 120 * KIB,
      staticTransferRaw: 460 * KIB,
      staticTransferGzip: 180 * KIB,
      blockingStyleCount: 4,
      localAssetRequestCount: 8,
    },
  },
  {
    path: '/docs',
    label: 'Documentation index',
    required: true,
    budget: PUBLIC_CONTENT_LIMITS,
  },
  {
    path: '/docs/getting-started',
    label: 'Getting started',
    required: true,
    budget: DOC_LIMITS,
  },
  {
    path: '/docs/troubleshooting',
    label: 'Troubleshooting and FAQ',
    required: true,
    budget: {
      ...DOC_LIMITS,
      htmlRaw: 96 * KIB,
      htmlGzip: 22 * KIB,
      staticTransferRaw: 260 * KIB,
      staticTransferGzip: 105 * KIB,
    },
  },
  {
    path: '/integrations/claude-code',
    label: 'Claude Code integration',
    required: true,
    budget: {
      ...SHARED_LIMITS,
      htmlRaw: 48 * KIB,
      htmlGzip: 12 * KIB,
      cssCodeRaw: 90 * KIB,
      cssCodeGzip: 20 * KIB,
      mediaRaw: 64 * KIB,
      mediaGzip: 60 * KIB,
      staticTransferRaw: 230 * KIB,
      staticTransferGzip: 95 * KIB,
      blockingStyleCount: 4,
      localAssetRequestCount: 7,
    },
  },
  {
    path: '/privacy',
    label: 'Privacy',
    required: true,
    budget: {
      ...SHARED_LIMITS,
      htmlRaw: 48 * KIB,
      htmlGzip: 12 * KIB,
      cssCodeRaw: 80 * KIB,
      cssCodeGzip: 18 * KIB,
      mediaRaw: 64 * KIB,
      mediaGzip: 60 * KIB,
      staticTransferRaw: 210 * KIB,
      staticTransferGzip: 85 * KIB,
      blockingStyleCount: 3,
      localAssetRequestCount: 6,
    },
  },
  ...[
    {
      path: '/guides',
      label: 'Guides index',
    },
    {
      path: '/guides/sync-claude-code-sessions-across-devices',
      label: 'Guide article',
    },
    {
      path: '/blog',
      label: 'Blog index',
    },
    {
      path: '/blog/why-git-does-not-sync-coding-agent-sessions',
      label: 'Blog article',
    },
    {
      path: '/compare/reinstate-vs-manual-session-copying',
      label: 'Comparison',
    },
    {
      path: '/use-cases/work-and-personal-computers',
      label: 'Use case',
    },
    {
      path: '/compatibility',
      label: 'Compatibility matrix',
    },
    {
      path: '/compatibility/agent-version-history',
      label: 'Compatibility version history',
    },
    {
      path: '/glossary',
      label: 'Terminology glossary',
    },
    {
      path: '/research/encrypted-snapshot-format-v1',
      label: 'Snapshot format reference',
    },
    {
      path: '/tools/path-mapping-visualizer',
      label: 'Path-mapping visualizer',
    },
  ].map(({ path, label }) => ({
    path,
    label,
    required: true,
    budget:
      [
        '/compare/reinstate-vs-manual-session-copying',
        '/tools/path-mapping-visualizer',
      ].includes(path)
        ? {
            ...PUBLIC_CONTENT_LIMITS,
            blockingStyleCount: 5,
          }
        : PUBLIC_CONTENT_LIMITS,
  })),
  {
    path: '/404',
    label: 'Not-found page',
    required: true,
    budget: {
      ...DOC_LIMITS,
      htmlRaw: 48 * KIB,
      htmlGzip: 12 * KIB,
    },
  },
];

const METRIC_LABELS = {
  htmlRaw: 'HTML (raw)',
  htmlGzip: 'HTML (gzip)',
  cssCodeRaw: 'CSS code (raw)',
  cssCodeGzip: 'CSS code (gzip)',
  executableJsRaw: 'executable JavaScript (raw)',
  executableJsGzip: 'executable JavaScript (gzip)',
  mediaRaw: 'route media (raw)',
  mediaGzip: 'route media (gzip)',
  staticTransferRaw: 'initial static transfer (raw)',
  staticTransferGzip: 'initial static transfer (gzip)',
  blockingScriptRaw: 'render-blocking JavaScript (raw)',
  largestFontRaw: 'largest declared font file',
  fontRaw: 'declared font candidates (raw)',
  fontGzip: 'declared font candidates (gzip)',
};

function parseAttributes(tag) {
  const attributes = {};
  const body = tag.replace(/^<[^\s>]+/i, '').replace(/\/?>\s*$/i, '');
  const pattern =
    /([^\s"'=<>`]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g;

  for (const match of body.matchAll(pattern)) {
    attributes[match[1].toLowerCase()] =
      match[2] ?? match[3] ?? match[4] ?? '';
  }

  return attributes;
}

function tags(markup, tagName) {
  const pattern = new RegExp(`<${tagName}\\b[^>]*>`, 'gi');
  return [...markup.matchAll(pattern)].map((match) => ({
    raw: match[0],
    attributes: parseAttributes(match[0]),
  }));
}

function relIncludes(attributes, value) {
  return (attributes.rel ?? '')
    .toLowerCase()
    .split(/\s+/)
    .includes(value);
}

function isExecutableScript(attributes) {
  const type = (attributes.type ?? '').trim().toLowerCase();
  return (
    !type ||
    type === 'module' ||
    /^(?:application|text)\/(?:java|ecma)script(?:\s*;|$)/.test(type)
  );
}

function isBlockingScript(attributes) {
  const type = (attributes.type ?? '').trim().toLowerCase();
  return (
    isExecutableScript(attributes) &&
    type !== 'module' &&
    !Object.hasOwn(attributes, 'async') &&
    !Object.hasOwn(attributes, 'defer')
  );
}

function isBlockingStyle(attributes) {
  return (
    !Object.hasOwn(attributes, 'disabled') &&
    (attributes.media ?? '').trim().toLowerCase() !== 'print'
  );
}

function routeHtmlPath(buildDir, route) {
  if (route === '/404') {
    return resolve(buildDir, '404.html');
  }
  if (route === '/') {
    return resolve(buildDir, 'index.html');
  }
  return resolve(buildDir, route.replace(/^\/+|\/+$/g, ''), 'index.html');
}

async function exists(filePath) {
  try {
    return (await stat(filePath)).isFile();
  } catch {
    return false;
  }
}

function formatBytes(bytes) {
  if (bytes < KIB) {
    return `${bytes} B`;
  }
  return `${(bytes / KIB).toFixed(1)} KiB`;
}

function gzipBytes(buffer) {
  return gzipSync(buffer, { level: 9 }).length;
}

function sumRaw(buffers) {
  return buffers.reduce((total, buffer) => total + buffer.length, 0);
}

function sumGzip(buffers) {
  return buffers.reduce((total, buffer) => total + gzipBytes(buffer), 0);
}

function publicPathForReference(reference, basePublicPath) {
  const value = reference.trim().replace(/^['"]|['"]$/g, '');
  if (!value || value.startsWith('#') || value.startsWith('data:')) {
    return { kind: 'ignored' };
  }

  let url;
  try {
    url = new URL(
      value,
      new URL(basePublicPath, `${SITE_ORIGIN}/`),
    );
  } catch {
    return { kind: 'invalid', reference: value };
  }

  if (!['http:', 'https:'].includes(url.protocol)) {
    return { kind: 'external', reference: value, origin: url.protocol };
  }
  if (url.origin !== SITE_ORIGIN) {
    return { kind: 'external', reference: value, origin: url.origin };
  }

  let pathname;
  try {
    pathname = decodeURIComponent(url.pathname);
  } catch {
    return { kind: 'invalid', reference: value };
  }
  return { kind: 'local', publicPath: pathname };
}

function diskPathForPublicPath(buildDir, publicPath) {
  const diskPath = resolve(buildDir, publicPath.replace(/^\/+/, ''));
  const relativePath = relative(buildDir, diskPath);
  if (
    relativePath === '..' ||
    relativePath.startsWith(`..${sep}`) ||
    resolve(diskPath) === resolve(buildDir)
  ) {
    return null;
  }
  return diskPath;
}

async function collectLocalAssets({
  buildDir,
  references,
  errors,
  route,
  type,
}) {
  const assets = new Map();
  const external = new Set();

  for (const { reference, basePublicPath } of references) {
    const resolvedReference = publicPathForReference(
      reference,
      basePublicPath,
    );
    if (resolvedReference.kind === 'ignored') {
      continue;
    }
    if (resolvedReference.kind === 'external') {
      external.add(`${resolvedReference.origin} (${reference})`);
      continue;
    }
    if (resolvedReference.kind === 'invalid') {
      errors.push({
        code: 'ASSET_REFERENCE_INVALID',
        route,
        detail: `${type} reference "${reference}" is not a valid URL.`,
        fix: 'Use a valid local or absolute asset URL.',
      });
      continue;
    }

    const diskPath = diskPathForPublicPath(
      buildDir,
      resolvedReference.publicPath,
    );
    if (!diskPath || !(await exists(diskPath))) {
      errors.push({
        code: 'ASSET_MISSING',
        route,
        detail: `${type} asset "${resolvedReference.publicPath}" is missing from the production build.`,
        fix: 'Build the referenced asset or remove the stale reference.',
      });
      continue;
    }

    if (!assets.has(resolvedReference.publicPath)) {
      assets.set(resolvedReference.publicPath, await readFile(diskPath));
    }
  }

  return { assets, external };
}

function cssUrlReferences(css, publicPath) {
  const basePublicPath = posix.join(posix.dirname(publicPath), '/');
  return [...css.matchAll(/url\(\s*([^)]*?)\s*\)/gi)].map((match) => ({
    reference: match[1],
    basePublicPath,
  }));
}

function srcsetReferences(value, basePublicPath) {
  return value
    .split(',')
    .map((candidate) => candidate.trim().split(/\s+/)[0])
    .filter(Boolean)
    .map((reference) => ({ reference, basePublicPath }));
}

function pushBudgetErrors(errors, route, measured, budget) {
  for (const [metric, limit] of Object.entries(budget)) {
    const actual = measured[metric];
    if (actual === undefined || actual <= limit) {
      continue;
    }

    const byteMetric =
      metric.endsWith('Raw') ||
      metric.endsWith('Gzip') ||
      metric === 'largestFontRaw';
    errors.push({
      code: `BUDGET_${metric.replace(/([a-z])([A-Z])/g, '$1_$2').toUpperCase()}`,
      route,
      detail: `${METRIC_LABELS[metric] ?? metric} is ${
        byteMetric ? formatBytes(actual) : actual
      }; the budget is ${byteMetric ? formatBytes(limit) : limit}.`,
      fix: `Reduce ${METRIC_LABELS[metric] ?? metric} or intentionally review and update this route's documented budget.`,
    });
  }
}

async function measureRoute(buildDir, definition, errors) {
  const htmlPath = routeHtmlPath(buildDir, definition.path);
  if (!(await exists(htmlPath))) {
    if (definition.required) {
      errors.push({
        code: 'REPRESENTATIVE_ROUTE_MISSING',
        route: definition.path,
        detail: `${definition.label} was not generated at ${relative(
          buildDir,
          htmlPath,
        )}.`,
        fix: 'Restore the representative route or update the performance gate when the route is intentionally replaced.',
      });
    }
    return null;
  }

  const htmlBuffer = await readFile(htmlPath);
  const html = htmlBuffer.toString('utf8');
  const basePublicPath =
    definition.path === '/'
      ? '/'
      : `${definition.path.replace(/\/+$/, '')}/`;
  const linkTags = tags(html, 'link');
  const styleMatches = [
    ...html.matchAll(/<style\b([^>]*)>([\s\S]*?)<\/style\s*>/gi),
  ];
  const scriptMatches = [
    ...html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script\s*>/gi),
  ];

  const stylesheetTags = linkTags.filter((link) =>
    relIncludes(link.attributes, 'stylesheet'),
  );
  const stylesheetReferences = stylesheetTags
    .filter((link) => link.attributes.href)
    .map((link) => ({
      reference: link.attributes.href,
      basePublicPath,
    }));
  const stylesheetAssets = await collectLocalAssets({
    buildDir,
    references: stylesheetReferences,
    errors,
    route: definition.path,
    type: 'stylesheet',
  });
  const blockingExternalStylesheets = stylesheetTags.filter((style) => {
    if (!style.attributes.href || !isBlockingStyle(style.attributes)) {
      return false;
    }
    return (
      publicPathForReference(style.attributes.href, basePublicPath).kind ===
      'external'
    );
  });
  if (blockingExternalStylesheets.length > 0) {
    errors.push({
      code: 'EXTERNAL_BLOCKING_STYLESHEET',
      route: definition.path,
      detail: `${blockingExternalStylesheets.length} render-blocking external stylesheet reference(s) cannot be measured from the static build.`,
      fix: 'Self-host critical styles or load non-critical external styles without blocking first render.',
    });
  }

  const executableScripts = scriptMatches
    .map((match) => ({
      attributes: parseAttributes(`<script${match[1]}>`),
      body: Buffer.from(match[2]),
    }))
    .filter((script) => isExecutableScript(script.attributes));
  const externalScriptReferences = executableScripts
    .filter((script) => script.attributes.src)
    .map((script) => ({
      reference: script.attributes.src,
      basePublicPath,
    }));
  const scriptAssets = await collectLocalAssets({
    buildDir,
    references: externalScriptReferences,
    errors,
    route: definition.path,
    type: 'script',
  });

  const mediaReferences = [];
  for (const image of tags(html, 'img')) {
    if (image.attributes.src) {
      mediaReferences.push({
        reference: image.attributes.src,
        basePublicPath,
      });
    }
    if (image.attributes.srcset) {
      mediaReferences.push(
        ...srcsetReferences(image.attributes.srcset, basePublicPath),
      );
    }
  }
  for (const source of tags(html, 'source')) {
    if (source.attributes.src) {
      mediaReferences.push({
        reference: source.attributes.src,
        basePublicPath,
      });
    }
    if (source.attributes.srcset) {
      mediaReferences.push(
        ...srcsetReferences(source.attributes.srcset, basePublicPath),
      );
    }
  }
  for (const preload of linkTags.filter(
    (link) =>
      relIncludes(link.attributes, 'preload') &&
      link.attributes.as === 'image' &&
      link.attributes.href,
  )) {
    mediaReferences.push({
      reference: preload.attributes.href,
      basePublicPath,
    });
  }

  const fontReferences = [];
  for (const [publicPath, buffer] of stylesheetAssets.assets) {
    for (const reference of cssUrlReferences(
      buffer.toString('utf8'),
      publicPath,
    )) {
      const normalized = reference.reference
        .trim()
        .replace(/^['"]|['"]$/g, '')
        .split(/[?#]/, 1)[0]
        .toLowerCase();
      if (/\.(?:woff2?|ttf|otf)$/.test(normalized)) {
        fontReferences.push(reference);
      } else {
        mediaReferences.push(reference);
      }
    }
  }

  const mediaAssets = await collectLocalAssets({
    buildDir,
    references: mediaReferences,
    errors,
    route: definition.path,
    type: 'media',
  });
  const fontAssets = await collectLocalAssets({
    buildDir,
    references: fontReferences,
    errors,
    route: definition.path,
    type: 'font',
  });

  const externalAssets = new Set([
    ...stylesheetAssets.external,
    ...scriptAssets.external,
    ...mediaAssets.external,
    ...fontAssets.external,
  ]);
  const blockingExternalScripts = executableScripts.filter((script) => {
    if (!script.attributes.src || !isBlockingScript(script.attributes)) {
      return false;
    }
    return (
      publicPathForReference(script.attributes.src, basePublicPath).kind ===
      'external'
    );
  });
  if (blockingExternalScripts.length > 0) {
    errors.push({
      code: 'EXTERNAL_BLOCKING_SCRIPT',
      route: definition.path,
      detail: `${blockingExternalScripts.length} render-blocking external script reference(s) cannot be measured from the static build.`,
      fix: 'Self-host the script, or load it with defer/async after reviewing the user-visible tradeoff.',
    });
  }

  const inlineStyleBuffers = styleMatches.map((match) =>
    Buffer.from(match[2]),
  );
  const localStylesheetBuffers = [...stylesheetAssets.assets.values()];
  const localScriptBuffers = [...scriptAssets.assets.values()];
  const inlineExecutableScriptBuffers = executableScripts
    .filter((script) => !script.attributes.src)
    .map((script) => script.body);
  const localMediaBuffers = [...mediaAssets.assets.values()];
  const localFontBuffers = [...fontAssets.assets.values()];
  const blockingInlineScripts = executableScripts.filter(
    (script) =>
      !script.attributes.src && isBlockingScript(script.attributes),
  );
  const blockingLocalScriptPaths = executableScripts
    .filter(
      (script) =>
        script.attributes.src && isBlockingScript(script.attributes),
    )
    .map((script) =>
      publicPathForReference(script.attributes.src, basePublicPath),
    )
    .filter((result) => result.kind === 'local')
    .map((result) => result.publicPath);
  const blockingLocalScriptBuffers = blockingLocalScriptPaths
    .map((publicPath) => scriptAssets.assets.get(publicPath))
    .filter(Boolean);

  const measured = {
    htmlRaw: htmlBuffer.length,
    htmlGzip: gzipBytes(htmlBuffer),
    cssCodeRaw:
      sumRaw(localStylesheetBuffers) + sumRaw(inlineStyleBuffers),
    cssCodeGzip:
      sumGzip(localStylesheetBuffers) + sumGzip(inlineStyleBuffers),
    executableJsRaw:
      sumRaw(localScriptBuffers) + sumRaw(inlineExecutableScriptBuffers),
    executableJsGzip:
      sumGzip(localScriptBuffers) + sumGzip(inlineExecutableScriptBuffers),
    mediaRaw: sumRaw(localMediaBuffers),
    mediaGzip: sumGzip(localMediaBuffers),
    staticTransferRaw:
      htmlBuffer.length +
      sumRaw(localStylesheetBuffers) +
      sumRaw(localScriptBuffers) +
      sumRaw(localMediaBuffers),
    staticTransferGzip:
      gzipBytes(htmlBuffer) +
      sumGzip(localStylesheetBuffers) +
      sumGzip(localScriptBuffers) +
      sumGzip(localMediaBuffers),
    blockingStyleCount:
      stylesheetTags.filter((style) => isBlockingStyle(style.attributes))
        .length +
      styleMatches.filter((match) =>
        isBlockingStyle(parseAttributes(`<style${match[1]}>`)),
      ).length,
    blockingScriptCount:
      blockingInlineScripts.length + blockingLocalScriptBuffers.length,
    blockingScriptRaw:
      sumRaw(blockingInlineScripts.map((script) => script.body)) +
      sumRaw(blockingLocalScriptBuffers),
    fontCount: fontAssets.assets.size,
    fontRaw: sumRaw(localFontBuffers),
    fontGzip: sumGzip(localFontBuffers),
    largestFontRaw: Math.max(
      0,
      ...localFontBuffers.map((buffer) => buffer.length),
    ),
    fontPreloadCount: linkTags.filter(
      (link) =>
        relIncludes(link.attributes, 'preload') &&
        link.attributes.as === 'font',
    ).length,
    externalAssetCount: externalAssets.size,
    localAssetRequestCount:
      stylesheetAssets.assets.size +
      scriptAssets.assets.size +
      mediaAssets.assets.size,
  };

  pushBudgetErrors(errors, definition.path, measured, definition.budget);

  return {
    label: definition.label,
    path: definition.path,
    measured,
    externalAssets: [...externalAssets],
  };
}

export async function auditPerformance(
  buildDir = DEFAULT_BUILD_DIR,
  { routes = DEFAULT_ROUTE_DEFINITIONS } = {},
) {
  const resolvedBuildDir = resolve(buildDir);
  const errors = [];

  try {
    if (!(await stat(resolvedBuildDir)).isDirectory()) {
      throw new Error('not a directory');
    }
  } catch {
    return {
      buildDir: resolvedBuildDir,
      routes: [],
      errors: [
        {
          code: 'BUILD_MISSING',
          route: null,
          detail: `Production client build not found at ${resolvedBuildDir}.`,
          fix: 'Run "npm run build" before the performance check.',
        },
      ],
    };
  }

  const measuredRoutes = [];
  for (const definition of routes) {
    const result = await measureRoute(
      resolvedBuildDir,
      definition,
      errors,
    );
    if (result) {
      measuredRoutes.push(result);
    }
  }

  return { buildDir: resolvedBuildDir, routes: measuredRoutes, errors };
}

function metricSummary(measured) {
  return [
    `HTML ${formatBytes(measured.htmlRaw)} / ${formatBytes(measured.htmlGzip)}`,
    `CSS ${formatBytes(measured.cssCodeRaw)} / ${formatBytes(measured.cssCodeGzip)}`,
    `JS ${formatBytes(measured.executableJsRaw)} / ${formatBytes(measured.executableJsGzip)}`,
    `media ${formatBytes(measured.mediaRaw)} / ${formatBytes(measured.mediaGzip)}`,
    `static ${formatBytes(measured.staticTransferRaw)} / ${formatBytes(measured.staticTransferGzip)}`,
    `fonts ${measured.fontCount} (${formatBytes(measured.fontRaw)} / ${formatBytes(measured.fontGzip)} declared; ${measured.fontPreloadCount} preloaded)`,
    `blocking ${measured.blockingStyleCount} style + ${measured.blockingScriptCount} script`,
    `requests ${measured.localAssetRequestCount} local + ${measured.externalAssetCount} external`,
  ].join(' · ');
}

export function formatReport(result) {
  const lines = [
    'Website static performance budget',
    'Values are raw / gzip from dist/client; declared fonts are reported separately from initial static transfer.',
  ];

  for (const route of result.routes) {
    lines.push(`- ${route.label} (${route.path}): ${metricSummary(route.measured)}`);
    if (route.externalAssets.length > 0) {
      lines.push(
        `  External assets (counted, size not measurable here): ${route.externalAssets.join(', ')}`,
      );
    }
  }

  if (result.errors.length === 0) {
    const count = result.routes.length;
    lines.push(
      `Performance validation passed: ${count} representative ${
        count === 1 ? 'route stayed within its' : 'routes stayed within their'
      } static-build budgets.`,
    );
    return lines.join('\n');
  }

  lines.push(
    `Performance validation failed with ${result.errors.length} issue${
      result.errors.length === 1 ? '' : 's'
    }:`,
  );
  for (const error of result.errors) {
    lines.push(
      `- [${error.code}]${error.route ? ` ${error.route}:` : ''} ${error.detail}`,
    );
    lines.push(`  Fix: ${error.fix}`);
  }
  return lines.join('\n');
}

async function main() {
  const requestedBuildDir = process.argv[2]
    ? resolve(process.cwd(), process.argv[2])
    : DEFAULT_BUILD_DIR;
  const result = await auditPerformance(requestedBuildDir);
  console.log(formatReport(result));
  if (result.errors.length > 0) {
    process.exitCode = 1;
  }
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : '';
if (invokedPath === fileURLToPath(import.meta.url)) {
  await main();
}
