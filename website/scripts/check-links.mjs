#!/usr/bin/env node

import { readFile, readdir, stat } from 'node:fs/promises';
import { dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const SITE_ORIGIN = 'https://reinstate.dev';
const DEFAULT_BUILD_DIR = resolve(process.cwd(), 'dist/client');
const RUNTIME_PATH_PREFIXES = ['/api'];
const IGNORED_SCHEMES = new Set([
  'blob:',
  'data:',
  'javascript:',
  'mailto:',
  'sms:',
  'tel:',
]);
const ASSET_LINK_RELS = new Set([
  'apple-touch-icon',
  'icon',
  'manifest',
  'mask-icon',
  'modulepreload',
  'preload',
  'stylesheet',
]);

function decodeHtml(value) {
  const named = {
    amp: '&',
    apos: "'",
    gt: '>',
    lt: '<',
    nbsp: '\u00a0',
    quot: '"',
  };

  return value.replace(
    /&(?:#(\d+)|#x([\da-f]+)|([a-z]+));/gi,
    (entity, decimal, hexadecimal, name) => {
      if (decimal) {
        return String.fromCodePoint(Number.parseInt(decimal, 10));
      }
      if (hexadecimal) {
        return String.fromCodePoint(Number.parseInt(hexadecimal, 16));
      }
      return named[name.toLowerCase()] ?? entity;
    },
  );
}

function parseAttributes(tag) {
  const attributes = {};
  const body = tag
    .replace(/^<[^\s>]+/i, '')
    .replace(/\/?>\s*$/i, '');
  const pattern =
    /([^\s"'=<>`]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g;

  for (const match of body.matchAll(pattern)) {
    const name = match[1].toLowerCase();
    attributes[name] = decodeHtml(match[2] ?? match[3] ?? match[4] ?? '');
  }

  return attributes;
}

function markupForTagScan(html) {
  return html
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(
      /(<script\b[^>]*>)[\s\S]*?<\/script\s*>/gi,
      '$1</script>',
    )
    .replace(/(<style\b[^>]*>)[\s\S]*?<\/style\s*>/gi, '$1</style>');
}

function lineNumberAt(markup, index) {
  return markup.slice(0, index).split('\n').length;
}

function splitSrcset(value) {
  if (!value || value.trim().toLowerCase().startsWith('data:')) {
    return value ? [value.trim()] : [];
  }

  return value
    .split(',')
    .map((candidate) => candidate.trim().split(/\s+/, 1)[0])
    .filter(Boolean);
}

function addReference(references, tag, attribute, value, kind, line) {
  if (value === undefined) {
    return;
  }
  references.push({ attribute, kind, line, tag, value: value.trim() });
}

function collectReferences(html) {
  const references = [];
  const markup = markupForTagScan(html);
  const tagPattern = /<([a-z][\w:-]*)\b[^>]*>/gi;

  for (const match of markup.matchAll(tagPattern)) {
    const tag = match[1].toLowerCase();
    const attributes = parseAttributes(match[0]);
    const line = lineNumberAt(markup, match.index);

    if (tag === 'a' || tag === 'area') {
      addReference(references, tag, 'href', attributes.href, 'link', line);
      continue;
    }

    if (tag === 'form') {
      addReference(
        references,
        tag,
        'action',
        attributes.action,
        'endpoint',
        line,
      );
      continue;
    }

    if (tag === 'link' && attributes.href !== undefined) {
      const rels = (attributes.rel ?? '')
        .toLowerCase()
        .split(/\s+/)
        .filter(Boolean);
      if (rels.some((rel) => ASSET_LINK_RELS.has(rel))) {
        addReference(
          references,
          tag,
          'href',
          attributes.href,
          'asset',
          line,
        );
      }
      continue;
    }

    if (tag === 'meta') {
      const field = (
        attributes.property ??
        attributes.name ??
        ''
      ).toLowerCase();
      if (field === 'og:image' || field === 'twitter:image') {
        addReference(
          references,
          tag,
          'content',
          attributes.content,
          'asset',
          line,
        );
      }
      continue;
    }

    const sourceAttribute =
      tag === 'object' ? 'data' : tag === 'video' ? 'src' : 'src';
    const sourceTags = new Set([
      'audio',
      'embed',
      'iframe',
      'img',
      'input',
      'script',
      'source',
      'track',
      'video',
    ]);
    if (sourceTags.has(tag)) {
      if (tag !== 'input' || attributes.type?.toLowerCase() === 'image') {
        addReference(
          references,
          tag,
          sourceAttribute,
          attributes[sourceAttribute],
          tag === 'iframe' ? 'link' : 'asset',
          line,
        );
      }
    } else if (tag === 'object') {
      addReference(
        references,
        tag,
        sourceAttribute,
        attributes[sourceAttribute],
        'asset',
        line,
      );
    }

    if (tag === 'video') {
      addReference(
        references,
        tag,
        'poster',
        attributes.poster,
        'asset',
        line,
      );
    }

    if (tag === 'img' || tag === 'source') {
      for (const source of splitSrcset(attributes.srcset ?? '')) {
        addReference(references, tag, 'srcset', source, 'asset', line);
      }
    }
  }

  return references;
}

function collectAnchors(html) {
  const anchors = new Set();
  const markup = markupForTagScan(html);
  const tagPattern = /<([a-z][\w:-]*)\b[^>]*>/gi;

  for (const match of markup.matchAll(tagPattern)) {
    const tag = match[1].toLowerCase();
    const attributes = parseAttributes(match[0]);
    if (attributes.id) {
      anchors.add(attributes.id);
    }
    if (tag === 'a' && attributes.name) {
      anchors.add(attributes.name);
    }
  }

  return anchors;
}

function routeFromHtml(buildDir, filePath) {
  const path = relative(buildDir, filePath).split(sep).join('/');
  if (path === 'index.html') {
    return '/';
  }
  if (path.endsWith('/index.html')) {
    return `/${path.slice(0, -'/index.html'.length)}`;
  }
  return `/${path}`;
}

function routeAliases(route, file) {
  const aliases = new Set([route]);
  if (route === '/') {
    aliases.add('/index.html');
  } else if (file.endsWith('/index.html')) {
    aliases.add(`${route}/`);
    aliases.add(`${route}/index.html`);
  } else if (route.endsWith('.html')) {
    aliases.add(route.slice(0, -'.html'.length));
  }
  return aliases;
}

async function walkFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await walkFiles(path)));
    } else if (entry.isFile()) {
      files.push(path);
    }
  }

  return files;
}

function addError(errors, code, file, line, message, fix) {
  errors.push({
    code,
    file: line ? `${file}:${line}` : file,
    fix,
    message,
  });
}

function hasValidPercentEncoding(value) {
  return !/%(?![\da-f]{2})/i.test(value);
}

function isRuntimePath(pathname) {
  return RUNTIME_PATH_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
  );
}

function normalizePathname(pathname) {
  const decoded = decodeURIComponent(pathname);
  const normalized = decoded.replace(/\/{2,}/g, '/');
  return normalized || '/';
}

function resolveTarget(pathname, filePaths, pagesByRoute) {
  const normalized = normalizePathname(pathname);
  const relativePath = normalized.replace(/^\/+/, '');
  const candidates = new Set();

  if (normalized === '/') {
    candidates.add('index.html');
  } else {
    candidates.add(relativePath);
    if (normalized.endsWith('/')) {
      candidates.add(`${relativePath}index.html`);
    } else {
      candidates.add(`${relativePath}/index.html`);
      if (!relativePath.includes('.')) {
        candidates.add(`${relativePath}.html`);
      }
    }
  }

  for (const candidate of candidates) {
    if (!filePaths.has(candidate)) {
      continue;
    }
    return {
      file: candidate,
      page:
        pagesByRoute.get(normalized) ??
        pagesByRoute.get(`/${candidate}`) ??
        null,
    };
  }

  return null;
}

function redirectPattern(source) {
  const escaped = source
    .replace(/[.+?^${}()|[\]\\]/g, '\\$&')
    .replace(/\\:([a-z][\w-]*)\\\*/gi, '.*')
    .replace(/:([a-z][\w-]*)\*/gi, '.*')
    .replace(/:([a-z][\w-]*)/gi, '[^/]+');
  return new RegExp(`^${escaped}/?$`);
}

async function existingFile(path) {
  try {
    const details = await stat(path);
    return details.isFile();
  } catch {
    return false;
  }
}

async function loadConfiguredRedirects(buildDir, errors) {
  const redirects = [];
  const configCandidates = [
    resolve(buildDir, 'vercel.json'),
    resolve(buildDir, '..', 'vercel.json'),
    resolve(buildDir, '..', '..', 'vercel.json'),
  ];

  for (const candidate of configCandidates) {
    if (!(await existingFile(candidate))) {
      continue;
    }

    try {
      const config = JSON.parse(await readFile(candidate, 'utf8'));
      for (const redirect of config.redirects ?? []) {
        if (
          typeof redirect.source === 'string' &&
          typeof redirect.destination === 'string'
        ) {
          redirects.push({
            destination: redirect.destination,
            matches: redirectPattern(redirect.source),
            source: redirect.source,
          });
        }
      }
    } catch (error) {
      addError(
        errors,
        'REDIRECT_CONFIG_INVALID',
        relative(buildDir, candidate).split(sep).join('/'),
        null,
        `Could not parse redirect configuration: ${error.message}`,
        'Repair vercel.json so internal redirect targets can be audited.',
      );
    }
    break;
  }

  const redirectsFile = resolve(buildDir, '_redirects');
  if (await existingFile(redirectsFile)) {
    const contents = await readFile(redirectsFile, 'utf8');
    for (const line of contents.split(/\r?\n/)) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) {
        continue;
      }
      const [source, destination, status = '301'] = trimmed.split(/\s+/);
      if (/^3\d\d!?$/.test(status)) {
        redirects.push({
          destination,
          matches: redirectPattern(source),
          source,
        });
      }
    }
  }

  return redirects;
}

async function loadRuntimeRoutes(buildDir) {
  const routeMatchers = [];
  const configCandidates = [
    resolve(buildDir, '.vercel', 'output', 'config.json'),
    resolve(buildDir, '..', '.vercel', 'output', 'config.json'),
    resolve(buildDir, '..', '..', '.vercel', 'output', 'config.json'),
  ];

  for (const candidate of configCandidates) {
    if (!(await existingFile(candidate))) {
      continue;
    }

    try {
      const config = JSON.parse(await readFile(candidate, 'utf8'));
      for (const route of config.routes ?? []) {
        if (route.dest !== '_render' || typeof route.src !== 'string') {
          continue;
        }
        try {
          routeMatchers.push(new RegExp(route.src));
        } catch {
          // Vercel validated this generated config. Ignore a matcher if the
          // current JavaScript runtime cannot compile its routing expression.
        }
      }
    } catch {
      // Redirect configuration is author-maintained and reported separately.
      // This file is generated build output, so a missing runtime exemption is
      // preferable to duplicating an opaque build-tool parse error.
    }
    break;
  }

  return routeMatchers;
}

function detectGeneratedRedirect(page) {
  const markup = markupForTagScan(page.html);
  const metaPattern = /<meta\b[^>]*>/gi;

  for (const match of markup.matchAll(metaPattern)) {
    const attributes = parseAttributes(match[0]);
    if (attributes['http-equiv']?.toLowerCase() !== 'refresh') {
      continue;
    }
    const destination = attributes.content?.match(
      /^\s*\d+\s*;\s*url\s*=\s*(.+?)\s*$/i,
    )?.[1];
    return destination
      ? { destination, matches: redirectPattern(page.route), source: page.route }
      : null;
  }

  return null;
}

function referenceLocation(page, reference) {
  return `${reference.tag}[${reference.attribute}]="${reference.value}"`;
}

export async function auditLinks(buildDirectory = DEFAULT_BUILD_DIR) {
  const buildDir = resolve(buildDirectory);
  const errors = [];
  const counts = {
    assets: 0,
    externalSkipped: 0,
    fragments: 0,
    htmlPages: 0,
    internalLinks: 0,
    runtimeSkipped: 0,
  };

  let buildStats;
  try {
    buildStats = await stat(buildDir);
  } catch {
    addError(
      errors,
      'BUILD_MISSING',
      buildDir,
      null,
      'The built client directory does not exist.',
      'Run the Astro production build before the link-integrity check.',
    );
    return { buildDir, counts, errors };
  }

  if (!buildStats.isDirectory()) {
    addError(
      errors,
      'BUILD_NOT_DIRECTORY',
      buildDir,
      null,
      'The link check target is not a directory.',
      'Pass the Astro client build directory, normally dist/client.',
    );
    return { buildDir, counts, errors };
  }

  const allFiles = await walkFiles(buildDir);
  const filePaths = new Set(
    allFiles.map((file) => relative(buildDir, file).split(sep).join('/')),
  );
  const htmlFiles = allFiles
    .filter((file) => file.toLowerCase().endsWith('.html'))
    .sort();

  if (!htmlFiles.length) {
    addError(
      errors,
      'HTML_MISSING',
      buildDir,
      null,
      'The production build contains no generated HTML pages.',
      'Run the Astro production build and check its output configuration.',
    );
  }

  const pages = [];
  const pagesByRoute = new Map();
  for (const file of htmlFiles) {
    const fileName = relative(buildDir, file).split(sep).join('/');
    const route = routeFromHtml(buildDir, file);
    const html = await readFile(file, 'utf8');
    const page = {
      anchors: collectAnchors(html),
      file: fileName,
      html,
      references: collectReferences(html),
      route,
    };
    pages.push(page);
    for (const alias of routeAliases(route, fileName)) {
      pagesByRoute.set(alias, page);
    }
  }
  counts.htmlPages = pages.length;

  const redirects = await loadConfiguredRedirects(buildDir, errors);
  const runtimeRoutes = await loadRuntimeRoutes(buildDir);
  for (const page of pages) {
    const redirect = detectGeneratedRedirect(page);
    if (redirect) {
      redirects.push(redirect);
    }
  }

  for (const page of pages) {
    const pageUrl = new URL(page.route, `${SITE_ORIGIN}/`);

    for (const reference of page.references) {
      const location = referenceLocation(page, reference);
      if (!reference.value) {
        if (reference.kind === 'asset') {
          addError(
            errors,
            'ASSET_URL_EMPTY',
            page.file,
            reference.line,
            `${location} is empty and would request the current document as an asset.`,
            `Remove the empty ${reference.attribute} or point it to a generated public asset.`,
          );
        }
        continue;
      }

      const scheme = reference.value.match(/^([a-z][a-z\d+.-]*:)/i)?.[1];
      if (scheme && IGNORED_SCHEMES.has(scheme.toLowerCase())) {
        counts.externalSkipped += 1;
        continue;
      }

      if (!hasValidPercentEncoding(reference.value)) {
        addError(
          errors,
          'URL_ENCODING_INVALID',
          page.file,
          reference.line,
          `${location} contains a percent sign that is not followed by two hexadecimal digits.`,
          'Percent-encode the URL correctly (for example, a space is %20) or use an unencoded safe character.',
        );
        continue;
      }

      let targetUrl;
      try {
        targetUrl = new URL(reference.value, pageUrl);
      } catch (error) {
        addError(
          errors,
          'URL_INVALID',
          page.file,
          reference.line,
          `${location} is not a valid URL: ${error.message}`,
          'Replace it with a valid site-relative or absolute URL.',
        );
        continue;
      }

      if (
        targetUrl.hostname === 'reinstate.dev' &&
        targetUrl.protocol === 'http:'
      ) {
        addError(
          errors,
          'INTERNAL_URL_INSECURE',
          page.file,
          reference.line,
          `${location} points to Reinstate over HTTP.`,
          `Use the HTTPS URL: https://reinstate.dev${targetUrl.pathname}${targetUrl.search}${targetUrl.hash}`,
        );
        continue;
      }

      if (targetUrl.origin !== SITE_ORIGIN) {
        counts.externalSkipped += 1;
        continue;
      }

      let pathname;
      let fragment;
      try {
        pathname = normalizePathname(targetUrl.pathname);
        fragment = targetUrl.hash
          ? decodeURIComponent(targetUrl.hash.slice(1))
          : '';
      } catch {
        addError(
          errors,
          'URL_ENCODING_INVALID',
          page.file,
          reference.line,
          `${location} contains malformed percent-encoded path or fragment data.`,
          'Correct the percent-encoding and rebuild the site.',
        );
        continue;
      }

      if (
        isRuntimePath(pathname) ||
        runtimeRoutes.some((matcher) => matcher.test(pathname))
      ) {
        counts.runtimeSkipped += 1;
        continue;
      }

      if (reference.kind === 'asset') {
        counts.assets += 1;
      } else if (reference.kind === 'link') {
        counts.internalLinks += 1;
      } else {
        counts.runtimeSkipped += 1;
        continue;
      }

      const redirect = redirects.find((entry) => entry.matches.test(pathname));
      if (redirect) {
        addError(
          errors,
          'LINK_TO_REDIRECT',
          page.file,
          reference.line,
          `${location} points to redirect source "${redirect.source}" → "${redirect.destination}".`,
          `Link directly to "${redirect.destination}" to avoid an unnecessary redirect.`,
        );
        continue;
      }

      const target = resolveTarget(pathname, filePaths, pagesByRoute);
      if (!target) {
        addError(
          errors,
          reference.kind === 'asset'
            ? 'ASSET_TARGET_MISSING'
            : 'LINK_TARGET_MISSING',
          page.file,
          reference.line,
          `${location} resolves to "${pathname}", which is absent from the production build.`,
          reference.kind === 'asset'
            ? 'Add the public asset, correct the generated asset URL, or remove the reference.'
            : 'Create the destination page/file or update the link to an existing canonical route.',
        );
        continue;
      }

      if (fragment) {
        counts.fragments += 1;
        if (target.page && !target.page.anchors.has(fragment)) {
          addError(
            errors,
            'FRAGMENT_TARGET_MISSING',
            page.file,
            reference.line,
            `${location} targets "#${fragment}", but "${target.page.file}" has no matching id or named anchor.`,
            `Add id="${fragment}" to the destination or update the fragment to an existing anchor.`,
          );
        }
      }
    }
  }

  errors.sort(
    (left, right) =>
      left.file.localeCompare(right.file) ||
      left.code.localeCompare(right.code) ||
      left.message.localeCompare(right.message),
  );

  return { buildDir, counts, errors };
}

export function formatLinkReport(result) {
  if (!result.errors.length) {
    const { assets, fragments, htmlPages, internalLinks, runtimeSkipped } =
      result.counts;
    return `Link validation passed: ${htmlPages} HTML page${
      htmlPages === 1 ? '' : 's'
    }, ${internalLinks} internal link${
      internalLinks === 1 ? '' : 's'
    }, ${assets} asset reference${
      assets === 1 ? '' : 's'
    }, and ${fragments} fragment reference${
      fragments === 1 ? '' : 's'
    } checked; ${runtimeSkipped} runtime endpoint${
      runtimeSkipped === 1 ? '' : 's'
    } skipped.`;
  }

  const lines = [
    `Link validation failed with ${result.errors.length} actionable error${
      result.errors.length === 1 ? '' : 's'
    }:`,
    '',
  ];

  result.errors.forEach((error, index) => {
    lines.push(
      `${index + 1}. [${error.code}] ${error.file}`,
      `   ${error.message}`,
      `   Fix: ${error.fix}`,
      '',
    );
  });

  return lines.join('\n').trimEnd();
}

async function main() {
  const buildDirectory = process.argv[2] ?? DEFAULT_BUILD_DIR;
  const result = await auditLinks(buildDirectory);
  const report = formatLinkReport(result);

  if (result.errors.length) {
    console.error(report);
    process.exitCode = 1;
  } else {
    console.log(report);
  }
}

if (
  process.argv[1] &&
  resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))
) {
  await main();
}
