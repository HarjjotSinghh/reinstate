#!/usr/bin/env node

import { readdir, readFile, stat } from 'node:fs/promises';
import { basename, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const SITE_ORIGIN = 'https://reinstate.dev';
const DEFAULT_BUILD_DIR = resolve(process.cwd(), 'dist/client');
const DEFAULT_REDIRECT_CONFIG = resolve(process.cwd(), 'vercel.json');
const CLEAN_REDIRECT_PATH =
  /^\/(?:[a-z0-9]+(?:[/-][a-z0-9]+)*)?$/;

const UNSUPPORTED_AGENTS = [
  ['Aider', /\baider\b/i],
  ['Amazon Q', /\bamazon\s+q\b/i],
  ['Amp', /\bamp\b/i],
  ['Claude Desktop', /\bclaude\s+desktop\b/i],
  ['Cline', /\bcline\b/i],
  ['Cursor', /\bcursor\b/i],
  ['Devin', /\bdevin\b/i],
  ['Gemini CLI', /\bgemini(?:\s+cli)?\b/i],
  ['GitHub Copilot', /\b(?:github\s+)?copilot\b/i],
  ['Goose', /\bgoose\b/i],
  ['Kiro', /\bkiro\b/i],
  ['OpenCode', /\bopen\s*code\b/i],
  ['Roo Code', /\broo\s+code\b/i],
  ['Sourcegraph Cody', /\b(?:sourcegraph\s+)?cody\b/i],
  ['Tabnine', /\btabnine\b/i],
  ['Windsurf', /\bwindsurf\b/i],
  ['Zed AI', /\bzed\s+ai\b/i],
];

const UNSUPPORTED_OPERATING_SYSTEMS = [
  ['Android', /\bandroid\b/i],
  ['ChromeOS', /\bchrome\s*os\b/i],
  ['Debian', /\bdebian\b/i],
  ['Fedora', /\bfedora\b/i],
  ['FreeBSD', /\bfreebsd\b/i],
  ['iOS', /\bios\b/i],
  ['iPadOS', /\bipados\b/i],
  ['Linux', /\blinux\b/i],
  ['Ubuntu', /\bubuntu\b/i],
];

const ALLOWED_ROBOTS_META_DIRECTIVES = new Set([
  'index',
  'noindex',
  'follow',
  'nofollow',
  'noarchive',
  'nosnippet',
  'notranslate',
  'noimageindex',
]);

const SOCIAL_FIELDS = {
  'og:site_name': { attribute: 'property', expected: 'Reinstate' },
  'og:locale': { attribute: 'property', expected: 'en_US' },
  'og:type': { attribute: 'property' },
  'og:title': { attribute: 'property', match: 'title' },
  'og:description': { attribute: 'property', match: 'description' },
  'og:url': { attribute: 'property', match: 'canonical' },
  'og:image': { attribute: 'property', absoluteUrl: true },
  'og:image:secure_url': { attribute: 'property', absoluteUrl: true },
  'og:image:type': { attribute: 'property', expected: 'image/png' },
  'og:image:width': { attribute: 'property', expected: '1200' },
  'og:image:height': { attribute: 'property', expected: '630' },
  'og:image:alt': { attribute: 'property' },
  'twitter:card': { attribute: 'name', expected: 'summary_large_image' },
  'twitter:title': { attribute: 'name', match: 'title' },
  'twitter:description': { attribute: 'name', match: 'description' },
  'twitter:image': { attribute: 'name', absoluteUrl: true },
  'twitter:image:alt': { attribute: 'name' },
};

const REQUIRED_SCHEMA_FIELDS = {
  Answer: ['text'],
  BlogPosting: [
    '@id',
    'headline',
    'description',
    'url',
    'datePublished',
    'dateModified',
    'image',
    'author',
    'publisher',
    'mainEntityOfPage',
    'articleSection',
  ],
  BreadcrumbList: ['itemListElement'],
  FAQPage: ['@id', 'url', 'mainEntity'],
  HowTo: ['@id', 'name', 'description', 'step'],
  HowToStep: ['name', 'text'],
  ImageObject: ['url', 'width', 'height'],
  ListItem: ['position', 'name', 'item'],
  Offer: ['price', 'priceCurrency'],
  Person: ['@id', 'name', 'url'],
  Question: ['name', 'acceptedAnswer'],
  SoftwareApplication: [
    '@id',
    'name',
    'url',
    'description',
    'applicationCategory',
    'operatingSystem',
    'softwareVersion',
    'isAccessibleForFree',
    'offers',
    'author',
    'license',
  ],
  SoftwareSourceCode: [
    '@id',
    'name',
    'description',
    'codeRepository',
    'programmingLanguage',
    'license',
    'author',
  ],
  TechArticle: [
    '@id',
    'headline',
    'description',
    'url',
    'dateModified',
    'image',
    'author',
    'mainEntityOfPage',
  ],
  WebPage: [
    '@id',
    'url',
    'name',
    'description',
    'primaryImageOfPage',
    'isPartOf',
    'inLanguage',
  ],
  WebSite: ['@id', 'url', 'name', 'description', 'publisher', 'inLanguage'],
};

const SCHEMA_URL_FIELDS = new Set([
  '@id',
  'codeRepository',
  'downloadUrl',
  'item',
  'license',
  'mainEntityOfPage',
  'softwareHelp',
  'url',
]);

const SCHEMA_DATE_FIELDS = new Set([
  'dateCreated',
  'dateModified',
  'datePublished',
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

function cleanText(value) {
  return decodeHtml(value).replace(/\s+/g, ' ').trim();
}

function cleanVisibleText(value) {
  return cleanText(value).replace(/\s+([,.;:!?])/g, '$1');
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
    const value = match[2] ?? match[3] ?? match[4] ?? '';
    attributes[name] = decodeHtml(value);
  }

  return attributes;
}

function findTags(markup, tagName) {
  const pattern = new RegExp(`<${tagName}\\b[^>]*>`, 'gi');
  return [...markup.matchAll(pattern)].map((match) => ({
    raw: match[0],
    attributes: parseAttributes(match[0]),
  }));
}

function withoutEmbeddedContent(html) {
  return html
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/<script\b[^>]*>[\s\S]*?<\/script\s*>/gi, '')
    .replace(/<style\b[^>]*>[\s\S]*?<\/style\s*>/gi, '');
}

function metaTagsBy(markup, attribute, value) {
  const expected = value.toLowerCase();
  return findTags(markup, 'meta').filter(
    (tag) => tag.attributes[attribute]?.toLowerCase() === expected,
  );
}

function canonicalLinks(markup) {
  return findTags(markup, 'link').filter((tag) =>
    (tag.attributes.rel ?? '')
      .toLowerCase()
      .split(/\s+/)
      .includes('canonical'),
  );
}

function routeFromHtml(buildDir, filePath) {
  const relativePath = relative(buildDir, filePath).split(sep).join('/');
  if (relativePath === 'index.html') {
    return '/';
  }
  if (relativePath.endsWith('/index.html')) {
    return `/${relativePath.slice(0, -'/index.html'.length)}`;
  }
  return `/${relativePath}`;
}

function normalizedUrlKey(url) {
  const parsed = new URL(url);
  let pathname = parsed.pathname.replace(/\/{2,}/g, '/');
  if (pathname !== '/') {
    pathname = pathname.replace(/\/+$/, '');
  }
  return `${parsed.origin}${pathname}`;
}

function isExcludedSitemapPath(pathname) {
  const normalized =
    pathname !== '/' ? pathname.replace(/\/+$/, '') : pathname;
  return (
    normalized === '/preview' ||
    normalized.startsWith('/preview/') ||
    normalized === '/api' ||
    normalized.startsWith('/api/') ||
    normalized === '/404' ||
    normalized === '/404.html' ||
    normalized.startsWith('/404/') ||
    normalized === '/docs/overview'
  );
}

function isPreviewRoute(route) {
  return route === '/preview' || route.startsWith('/preview/');
}

function validateSiteUrl(value) {
  try {
    const url = new URL(value);
    if (
      url.origin !== SITE_ORIGIN ||
      url.protocol !== 'https:' ||
      url.username ||
      url.password ||
      url.search ||
      url.hash
    ) {
      return null;
    }
    return url;
  } catch {
    return null;
  }
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

function addError(errors, code, file, message, fix) {
  errors.push({ code, file, message, fix });
}

function schemaTypes(value) {
  const type = value?.['@type'];
  return Array.isArray(type) ? type : typeof type === 'string' ? [type] : [];
}

function schemaValuePresent(value) {
  if (value === null || value === undefined) {
    return false;
  }
  if (typeof value === 'string') {
    return value.trim().length > 0;
  }
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  return true;
}

function inspectSchemaContracts(
  value,
  context,
  errors,
  declaredIds,
  location,
) {
  for (const type of schemaTypes(value)) {
    for (const field of REQUIRED_SCHEMA_FIELDS[type] ?? []) {
      if (!schemaValuePresent(value[field])) {
        addError(
          errors,
          'JSONLD_REQUIRED_FIELD',
          context,
          `JSON-LD ${location} declares ${type} without a non-empty ${field} property.`,
          `Add truthful ${field} data that matches visible content, or remove the ${type} node.`,
        );
      }
    }
  }

  if (schemaTypes(value).length && typeof value['@id'] === 'string') {
    const id = value['@id'];
    if (declaredIds.has(id)) {
      addError(
        errors,
        'JSONLD_ID_DUPLICATE',
        context,
        `JSON-LD ${location} redeclares @id "${id}" already used at ${declaredIds.get(id)}.`,
        'Declare each entity @id once and reference it elsewhere with an @id-only object.',
      );
    } else {
      declaredIds.set(id, location);
    }
  }
}

function inspectStructuredData(
  value,
  context,
  errors,
  declaredIds,
  location = '$',
) {
  if (Array.isArray(value)) {
    value.forEach((item, index) =>
      inspectStructuredData(
        item,
        context,
        errors,
        declaredIds,
        `${location}[${index}]`,
      ),
    );
    return;
  }

  if (!value || typeof value !== 'object') {
    return;
  }

  inspectSchemaContracts(value, context, errors, declaredIds, location);

  for (const [key, child] of Object.entries(value)) {
    const childLocation = `${location}.${key}`;
    const normalizedKey = key.toLowerCase();

    if (
      SCHEMA_URL_FIELDS.has(key) &&
      typeof child === 'string' &&
      child.trim()
    ) {
      try {
        const url = new URL(child);
        if (url.protocol !== 'https:') {
          throw new Error('not HTTPS');
        }
      } catch {
        addError(
          errors,
          'JSONLD_URL_INVALID',
          context,
          `JSON-LD ${childLocation} must be an absolute HTTPS URL; found ${JSON.stringify(child)}.`,
          'Use a canonical absolute HTTPS URL for schema entity and relationship fields.',
        );
      }
    }

    if (
      SCHEMA_DATE_FIELDS.has(key) &&
      (typeof child !== 'string' ||
        !Number.isFinite(Date.parse(child)) ||
        !/^\d{4}-\d{2}-\d{2}T/.test(child))
    ) {
      addError(
        errors,
        'JSONLD_DATE_INVALID',
        context,
        `JSON-LD ${childLocation} is not a valid ISO date-time: ${JSON.stringify(child)}.`,
        'Serialize dates as valid ISO 8601 date-time strings.',
      );
    }

    if (normalizedKey === 'aggregaterating' || normalizedKey === 'review') {
      addError(
        errors,
        'JSONLD_UNVERIFIED_REVIEW',
        context,
        `JSON-LD ${childLocation} uses "${key}", but Reinstate has no verified review or aggregate-rating data.`,
        `Remove ${childLocation}; never publish fabricated ratings or reviews.`,
      );
    }

    if (
      normalizedKey === '@type' &&
      (child === 'Review' || child === 'AggregateRating')
    ) {
      addError(
        errors,
        'JSONLD_UNVERIFIED_REVIEW',
        context,
        `JSON-LD ${childLocation} declares the unsupported type "${child}".`,
        'Remove review and aggregate-rating schema until the data is real, public, and verifiable.',
      );
    }

    if (
      normalizedKey === 'operatingsystem' ||
      normalizedKey === 'runtimeplatform'
    ) {
      const platforms = Array.isArray(child) ? child : [child];
      for (const platform of platforms) {
        if (platform !== 'macOS' && platform !== 'Windows') {
          addError(
            errors,
            'JSONLD_UNSUPPORTED_OS',
            context,
            `JSON-LD ${childLocation} contains unsupported platform value ${JSON.stringify(platform)}.`,
            'Use only "macOS" and "Windows" in operatingSystem and runtimePlatform claims.',
          );
        }
      }
    }

    inspectStructuredData(
      child,
      context,
      errors,
      declaredIds,
      childLocation,
    );
  }
}

function collectJsonStrings(value, location = '$', output = []) {
  if (typeof value === 'string') {
    output.push({ location, value });
  } else if (Array.isArray(value)) {
    value.forEach((item, index) =>
      collectJsonStrings(item, `${location}[${index}]`, output),
    );
  } else if (value && typeof value === 'object') {
    for (const [key, child] of Object.entries(value)) {
      collectJsonStrings(child, `${location}.${key}`, output);
    }
  }
  return output;
}

function inspectProductClaims(value, context, errors) {
  const strings = collectJsonStrings(value);
  const reported = new Set();

  for (const { location, value: text } of strings) {
    for (const [name, pattern] of UNSUPPORTED_AGENTS) {
      const key = `agent:${name}`;
      if (pattern.test(text) && !reported.has(key)) {
        reported.add(key);
        addError(
          errors,
          'JSONLD_UNSUPPORTED_AGENT',
          context,
          `JSON-LD ${location} names unsupported coding agent "${name}".`,
          'Keep structured-data compatibility claims limited to Claude Code and Codex; discuss roadmap tools only in clearly qualified page copy.',
        );
      }
    }

    for (const [name, pattern] of UNSUPPORTED_OPERATING_SYSTEMS) {
      const key = `os:${name}`;
      const isPlatformProperty =
        /\.(?:operatingSystem|runtimePlatform)(?:\[|$)/i.test(location);
      if (pattern.test(text) && !isPlatformProperty && !reported.has(key)) {
        reported.add(key);
        addError(
          errors,
          'JSONLD_UNSUPPORTED_OS',
          context,
          `JSON-LD ${location} names unsupported operating system "${name}".`,
          'Keep structured-data platform claims limited to macOS and Windows.',
        );
      }
    }
  }
}

function inspectMetadataClaims(values, context, errors) {
  const text = Object.values(values).filter(Boolean).join(' ');
  for (const [name, pattern] of UNSUPPORTED_AGENTS) {
    if (pattern.test(text)) {
      addError(
        errors,
        'META_UNSUPPORTED_AGENT',
        context,
        `Page metadata names unsupported coding agent "${name}".`,
        'Keep titles, descriptions, and social metadata limited to current Claude Code and Codex support.',
      );
    }
  }
  for (const [name, pattern] of UNSUPPORTED_OPERATING_SYSTEMS) {
    if (pattern.test(text)) {
      addError(
        errors,
        'META_UNSUPPORTED_OS',
        context,
        `Page metadata names unsupported operating system "${name}".`,
        'Keep platform metadata limited to currently documented macOS and Windows targets.',
      );
    }
  }
  if (/\b(?:todo|tbd|lorem ipsum)\b|\{\{[^}]+\}\}/i.test(text)) {
    addError(
      errors,
      'META_PLACEHOLDER',
      context,
      'Page metadata contains a placeholder value.',
      'Replace placeholders with reviewed, page-specific metadata before building.',
    );
  }
}

function inspectSchemaVisibility(
  value,
  visibleText,
  visibleDateTimes,
  context,
  errors,
  location = '$',
) {
  if (Array.isArray(value)) {
    value.forEach((item, index) =>
      inspectSchemaVisibility(
        item,
        visibleText,
        visibleDateTimes,
        context,
        errors,
        `${location}[${index}]`,
      ),
    );
    return;
  }
  if (!value || typeof value !== 'object') {
    return;
  }

  for (const type of schemaTypes(value)) {
    const visibleFields =
      type === 'Question'
        ? ['name']
        : ['BlogPosting', 'TechArticle'].includes(type)
          ? ['headline']
          : ['HowTo', 'ListItem', 'WebPage'].includes(type)
            ? ['name']
            : type === 'HowToStep'
              ? ['name', 'text']
              : [];
    for (const visibleField of visibleFields) {
      const visibleValue = cleanVisibleText(
        String(value[visibleField] ?? ''),
      );
      if (visibleValue && !visibleText.includes(visibleValue)) {
        addError(
          errors,
          'JSONLD_VISIBLE_MISMATCH',
          context,
          `JSON-LD ${location}.${visibleField} for ${type} is not present in visible page text: ${JSON.stringify(visibleValue)}.`,
          'Render the same schema value visibly or remove the mismatched schema node.',
        );
      }
    }

    for (const dateField of ['datePublished', 'dateModified']) {
      const schemaDate = value[dateField];
      if (
        typeof schemaDate === 'string' &&
        /^\d{4}-\d{2}-\d{2}/.test(schemaDate) &&
        !visibleDateTimes.some(
          (dateTime) =>
            dateTime === schemaDate ||
            dateTime.slice(0, 10) === schemaDate.slice(0, 10),
        )
      ) {
        addError(
          errors,
          'JSONLD_VISIBLE_MISMATCH',
          context,
          `JSON-LD ${location}.${dateField} for ${type} has no matching visible <time datetime>: ${JSON.stringify(schemaDate)}.`,
          'Render the publication or modification date visibly, or remove the unsupported schema date.',
        );
      }
    }
  }

  for (const [key, child] of Object.entries(value)) {
    inspectSchemaVisibility(
      child,
      visibleText,
      visibleDateTimes,
      context,
      errors,
      `${location}.${key}`,
    );
  }
}

function inspectJsonLd(html, context, errors) {
  const scriptPattern = /<script\b([^>]*)>([\s\S]*?)<\/script\s*>/gi;
  let scriptNumber = 0;
  const declaredIds = new Map();
  const visibleText = cleanVisibleText(
    withoutEmbeddedContent(html).replace(/<[^>]+>/g, ' '),
  );
  const visibleDateTimes = findTags(html, 'time')
    .map((time) => time.attributes.datetime ?? '')
    .filter(Boolean);

  for (const match of html.matchAll(scriptPattern)) {
    const attributes = parseAttributes(`<script ${match[1]}>`);
    if (attributes.type?.toLowerCase() !== 'application/ld+json') {
      continue;
    }

    scriptNumber += 1;
    const source = match[2].trim();
    if (!source) {
      addError(
        errors,
        'JSONLD_EMPTY',
        context,
        `JSON-LD script ${scriptNumber} is empty.`,
        'Remove the empty script or serialize a complete schema.org object.',
      );
      continue;
    }

    try {
      const data = JSON.parse(source);
      if (data['@context'] !== 'https://schema.org') {
        addError(
          errors,
          'JSONLD_CONTEXT_INVALID',
          context,
          `JSON-LD script ${scriptNumber} has @context ${JSON.stringify(data['@context'])}; expected "https://schema.org".`,
          'Use the canonical Schema.org HTTPS context on each generated graph.',
        );
      }
      inspectStructuredData(data, context, errors, declaredIds);
      inspectProductClaims(data, context, errors);
      inspectSchemaVisibility(
        data,
        visibleText,
        visibleDateTimes,
        context,
        errors,
      );
    } catch (error) {
      addError(
        errors,
        'JSONLD_INVALID',
        context,
        `JSON-LD script ${scriptNumber} is not valid JSON: ${error.message}`,
        'Render the value with JSON.stringify and validate the generated script.',
      );
    }
  }
}

function validateSocialMetadata(markup, context, values, errors) {
  const socialValues = {};

  for (const [field, rule] of Object.entries(SOCIAL_FIELDS)) {
    const matches = metaTagsBy(markup, rule.attribute, field);
    if (matches.length !== 1) {
      addError(
        errors,
        'SOCIAL_META_COUNT',
        context,
        `Expected exactly one ${field} meta tag; found ${matches.length}.`,
        `Add one non-empty <meta ${rule.attribute}="${field}" content="…"> tag.`,
      );
      continue;
    }

    const content = cleanText(matches[0].attributes.content ?? '');
    socialValues[field] = content;
    if (!content) {
      addError(
        errors,
        'SOCIAL_META_EMPTY',
        context,
        `${field} has an empty content value.`,
        `Give ${field} a descriptive value.`,
      );
      continue;
    }

    if (rule.expected && content !== rule.expected) {
      addError(
        errors,
        'SOCIAL_META_VALUE',
        context,
        `${field} is "${content}"; expected "${rule.expected}".`,
        `Set ${field} to "${rule.expected}".`,
      );
    }

    if (rule.match && content !== values[rule.match]) {
      addError(
        errors,
        'SOCIAL_META_MISMATCH',
        context,
        `${field} does not match the page ${rule.match}.`,
        `Render ${field} from the same ${rule.match} value used by the page.`,
      );
    }

    if (rule.absoluteUrl) {
      try {
        const url = new URL(content);
        if (url.protocol !== 'https:') {
          throw new Error('not HTTPS');
        }
      } catch {
        addError(
          errors,
          'SOCIAL_IMAGE_URL',
          context,
          `${field} must be an absolute HTTPS URL; found "${content}".`,
          `Use an absolute image URL such as ${SITE_ORIGIN}/brand/og.png.`,
        );
      }
    }
  }

  if (
    socialValues['og:type'] &&
    !['article', 'website'].includes(socialValues['og:type'])
  ) {
    addError(
      errors,
      'SOCIAL_META_VALUE',
      context,
      `og:type is "${socialValues['og:type']}"; expected "website" or "article".`,
      'Use website for product/hub pages and article only for genuine visible articles.',
    );
  }

  for (const [left, right] of [
    ['og:image', 'og:image:secure_url'],
    ['og:image', 'twitter:image'],
    ['og:image:alt', 'twitter:image:alt'],
  ]) {
    if (
      socialValues[left] &&
      socialValues[right] &&
      socialValues[left] !== socialValues[right]
    ) {
      addError(
        errors,
        'SOCIAL_META_MISMATCH',
        context,
        `${right} does not match ${left}.`,
        `Render ${right} from the same route-specific social-card value as ${left}.`,
      );
    }
  }

  return socialValues;
}

function inspectContentImages(markup, context, errors) {
  const images = findTags(markup, 'img');

  for (const [index, image] of images.entries()) {
    const label = `Content image ${index + 1}`;
    const alt = image.attributes.alt;
    const decorative =
      image.attributes['aria-hidden']?.toLowerCase() === 'true' ||
      ['none', 'presentation'].includes(
        image.attributes.role?.toLowerCase() ?? '',
      );

    if (alt === undefined) {
      addError(
        errors,
        'IMAGE_ALT_MISSING',
        context,
        `${label} has no alt attribute.`,
        'Add concise alternative text, or use alt="" with aria-hidden="true" for a purely decorative image.',
      );
    } else if (!cleanText(alt) && !decorative) {
      addError(
        errors,
        'IMAGE_ALT_EMPTY',
        context,
        `${label} has empty alternative text without being marked decorative.`,
        'Describe the image purpose in alt text, or mark a decorative image aria-hidden="true".',
      );
    }

    for (const dimension of ['width', 'height']) {
      const value = image.attributes[dimension] ?? '';
      if (!/^[1-9]\d*$/.test(value)) {
        addError(
          errors,
          'IMAGE_DIMENSION_MISSING',
          context,
          `${label} has no positive integer ${dimension} attribute.`,
          `Add the intrinsic ${dimension} to prevent layout shift.`,
        );
      }
    }

    if (!['eager', 'lazy'].includes(image.attributes.loading ?? '')) {
      addError(
        errors,
        'IMAGE_LOADING_MISSING',
        context,
        `${label} has no explicit loading="eager" or loading="lazy" policy.`,
        'Use eager for a genuinely above-the-fold image and lazy for below-the-fold media.',
      );
    }
  }
}

function inspectPageMetadata(headMarkup, context, errors) {
  const titleTags = findTags(headMarkup, 'title');
  let title = '';
  if (titleTags.length !== 1) {
    addError(
      errors,
      'TITLE_COUNT',
      context,
      `Expected exactly one <title>; found ${titleTags.length}.`,
      'Render one unique, descriptive <title> in the document head.',
    );
  } else {
    const titleMatch = headMarkup.match(
      /<title\b[^>]*>([\s\S]*?)<\/title\s*>/i,
    );
    title = cleanText(titleMatch?.[1] ?? '');
    if (!title) {
      addError(
        errors,
        'TITLE_EMPTY',
        context,
        'The page title is empty or has no closing </title>.',
        'Add visible text inside the single <title> element.',
      );
    }
  }

  const descriptions = metaTagsBy(headMarkup, 'name', 'description');
  let description = '';
  if (descriptions.length !== 1) {
    addError(
      errors,
      'DESCRIPTION_COUNT',
      context,
      `Expected exactly one meta description; found ${descriptions.length}.`,
      'Render one page-specific <meta name="description" content="…"> tag.',
    );
  } else {
    description = cleanText(descriptions[0].attributes.content ?? '');
    if (!description) {
      addError(
        errors,
        'DESCRIPTION_EMPTY',
        context,
        'The meta description is empty.',
        'Write a concise description that states the page outcome.',
      );
    }
  }

  const canonicals = canonicalLinks(headMarkup);
  let canonical = '';
  if (canonicals.length !== 1) {
    addError(
      errors,
      'CANONICAL_COUNT',
      context,
      `Expected exactly one canonical link; found ${canonicals.length}.`,
      'Render one <link rel="canonical" href="…"> tag.',
    );
  } else {
    canonical = cleanText(canonicals[0].attributes.href ?? '');
    if (!validateSiteUrl(canonical)) {
      addError(
        errors,
        'CANONICAL_URL',
        context,
        `Canonical "${canonical || '(empty)'}" is not a clean absolute ${SITE_ORIGIN} HTTPS URL.`,
        `Use an absolute ${SITE_ORIGIN} URL without a query string or fragment.`,
      );
      canonical = '';
    }
  }

  const socialValues = validateSocialMetadata(
    headMarkup,
    context,
    { title, description, canonical },
    errors,
  );
  inspectMetadataClaims(
    { title, description, ...socialValues },
    context,
    errors,
  );

  return {
    title,
    description,
    canonical,
    ogImage: socialValues['og:image'],
    twitterImage: socialValues['twitter:image'],
  };
}

function inspectHtml(html, context, route, errors) {
  const markup = withoutEmbeddedContent(html);
  const headMarkup =
    markup.match(/<head\b[^>]*>([\s\S]*?)<\/head\s*>/i)?.[1] ?? markup;
  const robotTags = metaTagsBy(headMarkup, 'name', 'robots');
  const robotTokens = robotTags.flatMap((tag) =>
    (tag.attributes.content ?? '')
      .toLowerCase()
      .split(',')
      .map((token) => token.trim().split(/\s+/)[0])
      .filter(Boolean),
  );
  const noindex = robotTokens.includes('noindex');

  if (robotTags.length !== 1) {
    addError(
      errors,
      'ROBOTS_META_COUNT',
      context,
      `Expected exactly one robots meta tag; found ${robotTags.length}.`,
      'Add one explicit robots meta tag so indexing intent is unambiguous.',
    );
  }
  for (const tag of robotTags) {
    for (const rawDirective of (tag.attributes.content ?? '').toLowerCase().split(',')) {
      const directive = rawDirective.trim();
      if (!directive) {
        addError(
          errors,
          'ROBOTS_META_DIRECTIVE',
          context,
          'Robots meta contains an empty directive.',
          'Use a comma-separated set such as "index, follow" or "noindex, nofollow".',
        );
        continue;
      }
      const [name, value] = directive.split(':', 2);
      const supportsValue = ['max-snippet', 'max-image-preview', 'max-video-preview', 'unavailable_after'].includes(name);
      if (
        (!supportsValue && !ALLOWED_ROBOTS_META_DIRECTIVES.has(name)) ||
        (supportsValue && !value?.trim())
      ) {
        addError(
          errors,
          'ROBOTS_META_DIRECTIVE',
          context,
          `Robots meta contains unsupported directive "${directive}".`,
          'Use documented robots directives with required values.',
        );
      }
    }
  }

  if (isPreviewRoute(route) && !noindex) {
    addError(
      errors,
      'PREVIEW_INDEXABLE',
      context,
      `Preview route ${route} is not protected by a noindex directive.`,
      'Render <meta name="robots" content="noindex, nofollow"> on every /preview page.',
    );
  }

  inspectJsonLd(html, context, errors);
  const metadata = inspectPageMetadata(headMarkup, context, errors);

  if (noindex) {
    return { route, context, indexable: false, ...metadata };
  }

  inspectContentImages(markup, context, errors);

  const headings = findTags(markup, 'h1');
  if (headings.length !== 1) {
    addError(
      errors,
      'H1_COUNT',
      context,
      `Expected exactly one <h1>; found ${headings.length}.`,
      'Give the page one primary heading and use lower heading levels for sections.',
    );
  }

  return {
    route,
    context,
    indexable: true,
    ...metadata,
  };
}

function parseRobotsGroups(source) {
  const groups = [];
  let userAgents = [];
  let directives = [];

  const flush = () => {
    if (userAgents.length) {
      groups.push({ userAgents, directives });
    }
    userAgents = [];
    directives = [];
  };

  for (const rawLine of source.split(/\r?\n/)) {
    const line = rawLine.replace(/#.*$/, '').trim();
    if (!line) {
      flush();
      continue;
    }

    const match = line.match(/^([^:]+):\s*(.*)$/);
    if (!match) {
      continue;
    }

    const name = match[1].trim().toLowerCase();
    const value = match[2].trim();
    if (name === 'user-agent') {
      if (directives.length) {
        flush();
      }
      userAgents.push(value);
    } else if (userAgents.length) {
      directives.push({ name, value });
    }
  }

  flush();
  return groups;
}

async function inspectRobots(buildDir, allFiles, errors) {
  const robotsPath = allFiles.find(
    (file) => relative(buildDir, file).split(sep).join('/') === 'robots.txt',
  );
  if (!robotsPath) {
    addError(
      errors,
      'ROBOTS_TXT_MISSING',
      'robots.txt',
      'The production build does not contain robots.txt.',
      'Publish public/robots.txt with crawler groups and a Sitemap directive.',
    );
    return;
  }

  const source = await readFile(robotsPath, 'utf8');
  const groups = parseRobotsGroups(source);
  for (const [lineNumber, rawLine] of source.split(/\r?\n/).entries()) {
    const line = rawLine.replace(/#.*$/, '').trim();
    if (!line) continue;
    const match = line.match(/^([^:]+):\s*(.*)$/);
    const name = match?.[1]?.trim().toLowerCase();
    const value = match?.[2]?.trim() ?? '';
    if (!match || !['user-agent', 'allow', 'disallow', 'sitemap'].includes(name)) {
      addError(
        errors,
        'ROBOTS_DIRECTIVE_INVALID',
        'robots.txt',
        `Line ${lineNumber + 1} contains an unsupported robots directive.`,
        'Use only User-agent, Allow, Disallow, and Sitemap in this policy.',
      );
      continue;
    }
    if (
      (name === 'user-agent' && !value) ||
      (['allow', 'disallow'].includes(name) && value && !value.startsWith('/'))
    ) {
      addError(
        errors,
        'ROBOTS_DIRECTIVE_INVALID',
        'robots.txt',
        `Line ${lineNumber + 1} has an invalid ${name} value.`,
        'Give User-agent a name and start non-empty Allow/Disallow paths with "/".',
      );
    }
  }

  for (const bot of ['OAI-SearchBot', 'PerplexityBot']) {
    const group = groups.find((candidate) =>
      candidate.userAgents.some(
        (userAgent) => userAgent.toLowerCase() === bot.toLowerCase(),
      ),
    );

    if (!group) {
      addError(
        errors,
        'ROBOTS_AI_CRAWLER_MISSING',
        'robots.txt',
        `robots.txt has no explicit ${bot} user-agent group.`,
        `Add "User-agent: ${bot}" followed by "Allow: /".`,
      );
      continue;
    }

    const allowsRoot = group.directives.some(
      (directive) =>
        directive.name === 'allow' && directive.value.trim() === '/',
    );
    const blocksRoot = group.directives.some(
      (directive) =>
        directive.name === 'disallow' && directive.value.trim() === '/',
    );
    if (!allowsRoot || blocksRoot) {
      addError(
        errors,
        'ROBOTS_AI_CRAWLER_BLOCKED',
        'robots.txt',
        `${bot} is not explicitly allowed to crawl the site root.`,
        `Set the ${bot} group to "Allow: /" and do not use "Disallow: /".`,
      );
    }
  }

  const sitemapLines = source
    .split(/\r?\n/)
    .map((line) => line.replace(/#.*$/, '').trim())
    .filter((line) => /^sitemap\s*:/i.test(line));

  if (!sitemapLines.length) {
    addError(
      errors,
      'ROBOTS_SITEMAP_MISSING',
      'robots.txt',
      'robots.txt has no Sitemap directive.',
      `Add "Sitemap: ${SITE_ORIGIN}/sitemap-index.xml".`,
    );
    return;
  }

  for (const line of sitemapLines) {
    const value = line.replace(/^sitemap\s*:\s*/i, '');
    const url = validateSiteUrl(value);
    if (!url) {
      addError(
        errors,
        'ROBOTS_SITEMAP_URL',
        'robots.txt',
        `Sitemap directive is not a clean absolute ${SITE_ORIGIN} URL: "${value}".`,
        `Use "Sitemap: ${SITE_ORIGIN}/sitemap-index.xml".`,
      );
      continue;
    }

    const localName = url.pathname.replace(/^\/+/, '');
    const exists = allFiles.some(
      (file) => relative(buildDir, file).split(sep).join('/') === localName,
    );
    if (!exists) {
      addError(
        errors,
        'ROBOTS_SITEMAP_NOT_FOUND',
        'robots.txt',
        `Sitemap directive points to ${url.pathname}, which is absent from the build.`,
        'Point the directive at a generated sitemap file.',
      );
    }
  }
}

async function inspectSitemaps(buildDir, allFiles, pages, errors) {
  const sitemapFiles = allFiles.filter(
    (file) =>
      basename(file).toLowerCase().startsWith('sitemap') &&
      file.toLowerCase().endsWith('.xml'),
  );

  if (!sitemapFiles.length) {
    addError(
      errors,
      'SITEMAP_MISSING',
      'sitemap',
      'The production build does not contain a sitemap XML file.',
      'Generate a sitemap during the Astro build and publish it in dist/client.',
    );
    return [];
  }

  const sitemapUrls = [];
  for (const file of sitemapFiles) {
    const source = await readFile(file, 'utf8');
    if (!/<urlset\b/i.test(source)) {
      continue;
    }

    for (const match of source.matchAll(/<loc\b[^>]*>([\s\S]*?)<\/loc\s*>/gi)) {
      sitemapUrls.push({
        context: relative(buildDir, file).split(sep).join('/'),
        value: cleanText(match[1]),
      });
    }
  }

  if (!sitemapUrls.length) {
    addError(
      errors,
      'SITEMAP_EMPTY',
      'sitemap',
      'No <urlset> sitemap contains any <loc> entries.',
      'Generate at least one URL sitemap containing the canonical homepage.',
    );
    return [];
  }

  const uniqueUrls = new Map();
  for (const entry of sitemapUrls) {
    const url = validateSiteUrl(entry.value);
    if (!url) {
      addError(
        errors,
        'SITEMAP_URL_INVALID',
        entry.context,
        `Sitemap URL "${entry.value}" is not a clean absolute ${SITE_ORIGIN} HTTPS URL.`,
        `Emit only canonical URLs on ${SITE_ORIGIN}.`,
      );
      continue;
    }

    const key = normalizedUrlKey(url.href);
    if (uniqueUrls.has(key)) {
      addError(
        errors,
        'SITEMAP_URL_DUPLICATE',
        entry.context,
        `Sitemap URL "${entry.value}" duplicates an entry in ${uniqueUrls.get(key)}.`,
        'Emit each canonical route once across all sitemap files.',
      );
    } else {
      uniqueUrls.set(key, entry.context);
    }

    if (isExcludedSitemapPath(url.pathname)) {
      addError(
        errors,
        'SITEMAP_EXCLUDED_ROUTE',
        entry.context,
        `Sitemap includes excluded route "${url.pathname}".`,
        'Filter /preview, /api, /404, and /docs/overview routes from sitemap generation.',
      );
    }
  }

  const canonicalPages = new Map(
    pages
      .filter((page) => page.indexable && page.canonical)
      .map((page) => [normalizedUrlKey(page.canonical), page]),
  );

  for (const [canonical, page] of canonicalPages) {
    if (!uniqueUrls.has(canonical)) {
      addError(
        errors,
        'SITEMAP_CANONICAL_MISSING',
        page.context,
        `Indexable canonical "${page.canonical}" is missing from the sitemap.`,
        'Add the canonical route to the sitemap, or mark the intentionally excluded page noindex.',
      );
    }
  }

  for (const [sitemapUrl, context] of uniqueUrls) {
    if (!isExcludedSitemapPath(new URL(sitemapUrl).pathname) && !canonicalPages.has(sitemapUrl)) {
      addError(
        errors,
        'SITEMAP_ORPHAN_URL',
        context,
        `Sitemap URL "${sitemapUrl}" has no generated indexable HTML page with that canonical.`,
        'Remove the URL or generate an indexable page whose canonical matches it.',
      );
    }
  }

  return sitemapUrls;
}

async function inspectRedirects(
  configPath,
  pages,
  sitemapUrls,
  errors,
) {
  let config;
  try {
    config = JSON.parse(await readFile(configPath, 'utf8'));
  } catch (error) {
    addError(
      errors,
      error instanceof SyntaxError
        ? 'REDIRECT_CONFIG_INVALID'
        : 'REDIRECT_CONFIG_MISSING',
      relative(process.cwd(), configPath) || 'vercel.json',
      error instanceof SyntaxError
        ? `The redirect configuration is not valid JSON: ${error.message}`
        : `The redirect configuration could not be read: ${error.message}`,
      'Keep website/vercel.json present and valid so redirects are checked with every production build.',
    );
    return [];
  }

  const context = relative(process.cwd(), configPath) || 'vercel.json';
  if (!Array.isArray(config.redirects)) {
    addError(
      errors,
      'REDIRECTS_INVALID',
      context,
      'The redirects property must be an array.',
      'Declare redirects as an array of explicit source, destination, and permanent entries.',
    );
    return [];
  }

  const builtRoutes = new Set(pages.map(({ route }) => route));
  const sitemapRoutes = new Set(
    sitemapUrls
      .map(({ value }) => validateSiteUrl(value))
      .filter(Boolean)
      .map(({ pathname }) =>
        pathname === '/' ? pathname : pathname.replace(/\/+$/, ''),
      ),
  );
  const redirectsBySource = new Map();
  const inspected = [];

  for (const [index, redirect] of config.redirects.entries()) {
    const entryContext = `${context}#redirects[${index}]`;
    if (!redirect || typeof redirect !== 'object' || Array.isArray(redirect)) {
      addError(
        errors,
        'REDIRECT_ENTRY_INVALID',
        entryContext,
        'Redirect entries must be objects.',
        'Use an object with clean source and destination paths plus permanent: true.',
      );
      continue;
    }

    const { source, destination, permanent } = redirect;
    const sourceIsClean =
      typeof source === 'string' && CLEAN_REDIRECT_PATH.test(source);
    const destinationIsClean =
      typeof destination === 'string' && CLEAN_REDIRECT_PATH.test(destination);

    if (!sourceIsClean) {
      addError(
        errors,
        'REDIRECT_SOURCE_INVALID',
        entryContext,
        `Redirect source ${JSON.stringify(source)} is not a clean static route.`,
        'Use one lowercase root-relative path without a host, query, fragment, trailing slash, wildcard, or duplicate slash.',
      );
    }
    if (!destinationIsClean) {
      addError(
        errors,
        'REDIRECT_DESTINATION_INVALID',
        entryContext,
        `Redirect destination ${JSON.stringify(destination)} is not a clean static route.`,
        'Use one lowercase root-relative path without a host, query, fragment, trailing slash, wildcard, or duplicate slash.',
      );
    }
    if (permanent !== true) {
      addError(
        errors,
        'REDIRECT_NOT_PERMANENT',
        entryContext,
        'The redirect is not explicitly permanent.',
        'Set permanent to true so Vercel returns a permanent redirect.',
      );
    }

    if (!sourceIsClean || !destinationIsClean) {
      continue;
    }

    inspected.push({ source, destination });

    if (source === destination) {
      addError(
        errors,
        'REDIRECT_SELF_LOOP',
        entryContext,
        `Redirect source and destination are both "${source}".`,
        'Point the source directly to a different canonical destination.',
      );
    }

    if (redirectsBySource.has(source)) {
      addError(
        errors,
        'REDIRECT_SOURCE_DUPLICATE',
        entryContext,
        `Redirect source "${source}" is already declared.`,
        'Keep exactly one destination for each redirect source.',
      );
    } else {
      redirectsBySource.set(source, destination);
    }

    if (!builtRoutes.has(destination)) {
      addError(
        errors,
        'REDIRECT_DESTINATION_MISSING',
        entryContext,
        `Redirect destination "${destination}" has no generated HTML page.`,
        'Point the redirect directly to an existing built route.',
      );
    }

    if (sitemapRoutes.has(source)) {
      addError(
        errors,
        'REDIRECT_SOURCE_IN_SITEMAP',
        entryContext,
        `Redirect source "${source}" is present in the sitemap.`,
        'Remove redirected sources from the sitemap and publish only their canonical destinations.',
      );
    }
  }

  for (const { source, destination } of inspected) {
    if (redirectsBySource.has(destination)) {
      addError(
        errors,
        'REDIRECT_CHAIN',
        context,
        `Redirect "${source}" points to redirect source "${destination}".`,
        `Point "${source}" directly to the final built destination.`,
      );
    }
  }

  const reportedLoops = new Set();
  for (const source of redirectsBySource.keys()) {
    const visitedAt = new Map();
    const path = [];
    let current = source;

    while (redirectsBySource.has(current)) {
      if (visitedAt.has(current)) {
        const cycle = path.slice(visitedAt.get(current));
        const key = [...cycle].sort().join('|');
        if (!reportedLoops.has(key)) {
          reportedLoops.add(key);
          addError(
            errors,
            'REDIRECT_LOOP',
            context,
            `Redirect loop detected: ${[...cycle, current].join(' -> ')}.`,
            'Point every redirect source directly to a final built destination outside the redirect source set.',
          );
        }
        break;
      }
      visitedAt.set(current, path.length);
      path.push(current);
      current = redirectsBySource.get(current);
    }
  }

  return inspected;
}

async function inspectSocialImages(buildDir, allFiles, pages, errors) {
  const filesByPath = new Map(
    allFiles.map((file) => [
      relative(buildDir, file).split(sep).join('/'),
      file,
    ]),
  );
  const imagesByPage = new Map();
  const inspectedFiles = new Map();

  for (const page of pages) {
    for (const [field, imageUrl] of [
      ['og:image', page.ogImage],
      ['twitter:image', page.twitterImage],
    ]) {
      if (!imageUrl) {
        continue;
      }

      const url = validateSiteUrl(imageUrl);
      if (!url) {
        addError(
          errors,
          'SOCIAL_IMAGE_LOCAL_URL',
          page.context,
          `${field} must point to a clean, build-verifiable URL on ${SITE_ORIGIN}; found "${imageUrl}".`,
          `Generate the social image under public assets and use an absolute ${SITE_ORIGIN} URL.`,
        );
        continue;
      }

      let localPath;
      try {
        localPath = decodeURIComponent(url.pathname).replace(/^\/+/, '');
      } catch {
        addError(
          errors,
          'SOCIAL_IMAGE_LOCAL_URL',
          page.context,
          `${field} contains an invalid encoded path: "${imageUrl}".`,
          'Use a normally encoded absolute social-image URL.',
        );
        continue;
      }

      const filePath = filesByPath.get(localPath);
      if (!filePath) {
        addError(
          errors,
          'SOCIAL_IMAGE_MISSING',
          page.context,
          `${field} points to /${localPath}, which is absent from dist/client.`,
          'Generate or copy the referenced social image into the production build.',
        );
        continue;
      }

      let inspection = inspectedFiles.get(filePath);
      if (!inspection) {
        const source = await readFile(filePath);
        const isPng =
          source.length >= 24 &&
          source.subarray(0, 8).equals(
            Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
          ) &&
          source.subarray(12, 16).toString('ascii') === 'IHDR';
        inspection = {
          isPng,
          width: isPng ? source.readUInt32BE(16) : 0,
          height: isPng ? source.readUInt32BE(20) : 0,
        };
        inspectedFiles.set(filePath, inspection);
      }

      if (
        !inspection.isPng ||
        inspection.width !== 1200 ||
        inspection.height !== 630
      ) {
        addError(
          errors,
          'SOCIAL_IMAGE_DIMENSIONS',
          page.context,
          `${field} image /${localPath} is ${
            inspection.isPng
              ? `${inspection.width}x${inspection.height}`
              : 'not a PNG'
          }; expected a 1200x630 PNG.`,
          'Generate the social card as a 1200x630 PNG.',
        );
      }
    }

    if (page.ogImage) {
      const key = page.ogImage.toLowerCase();
      if (imagesByPage.has(key)) {
        addError(
          errors,
          'SOCIAL_IMAGE_DUPLICATE',
          page.context,
          `og:image duplicates the card used by ${imagesByPage.get(key)}.`,
          'Generate a route-specific social card for every indexable page.',
        );
      } else {
        imagesByPage.set(key, page.context);
      }
    }
  }
}

function inspectUniqueness(pages, errors) {
  const titles = new Map();
  const descriptions = new Map();
  const canonicals = new Map();

  for (const page of pages.filter((candidate) => candidate.indexable)) {
    if (page.title) {
      const titleKey = page.title.toLocaleLowerCase('en-US');
      if (titles.has(titleKey)) {
        addError(
          errors,
          'TITLE_DUPLICATE',
          page.context,
          `Title "${page.title}" duplicates the title on ${titles.get(titleKey)}.`,
          'Give every indexable page a distinct title that describes its search intent.',
        );
      } else {
        titles.set(titleKey, page.context);
      }
    }

    if (page.description) {
      const descriptionKey = page.description.toLocaleLowerCase('en-US');
      if (descriptions.has(descriptionKey)) {
        addError(
          errors,
          'DESCRIPTION_DUPLICATE',
          page.context,
          `Description "${page.description}" duplicates the description on ${descriptions.get(descriptionKey)}.`,
          'Give every indexable page a distinct description that answers its specific intent.',
        );
      } else {
        descriptions.set(descriptionKey, page.context);
      }
    }

    if (page.canonical) {
      const canonicalKey = normalizedUrlKey(page.canonical);
      if (canonicals.has(canonicalKey)) {
        addError(
          errors,
          'CANONICAL_DUPLICATE',
          page.context,
          `Canonical "${page.canonical}" duplicates the canonical on ${canonicals.get(canonicalKey)}.`,
          'Give each indexable generated page its own canonical, or mark the duplicate page noindex.',
        );
      } else {
        canonicals.set(canonicalKey, page.context);
      }
    }
  }
}

export async function auditSeo(
  buildDirectory = DEFAULT_BUILD_DIR,
  { redirectConfigPath = null } = {},
) {
  const buildDir = resolve(buildDirectory);
  const errors = [];
  let buildStats;

  try {
    buildStats = await stat(buildDir);
  } catch {
    addError(
      errors,
      'BUILD_MISSING',
      buildDir,
      'The built client directory does not exist.',
      'Run "npm run build" before "npm run check:seo".',
    );
    return { buildDir, errors, pages: [], redirects: [], sitemapUrls: [] };
  }

  if (!buildStats.isDirectory()) {
    addError(
      errors,
      'BUILD_NOT_DIRECTORY',
      buildDir,
      'The SEO check target is not a directory.',
      'Pass the Astro client build directory, normally dist/client.',
    );
    return { buildDir, errors, pages: [], redirects: [], sitemapUrls: [] };
  }

  const allFiles = await walkFiles(buildDir);
  const htmlFiles = allFiles.filter((file) =>
    file.toLowerCase().endsWith('.html'),
  );

  if (!htmlFiles.length) {
    addError(
      errors,
      'HTML_MISSING',
      buildDir,
      'The production build contains no generated HTML pages.',
      'Run the Astro production build and check its output configuration.',
    );
  }

  const pages = [];
  for (const file of htmlFiles.sort()) {
    const context = relative(buildDir, file).split(sep).join('/');
    const route = routeFromHtml(buildDir, file);
    const html = await readFile(file, 'utf8');
    pages.push(inspectHtml(html, context, route, errors));
  }

  inspectUniqueness(pages, errors);
  await inspectRobots(buildDir, allFiles, errors);
  const sitemapUrls = await inspectSitemaps(
    buildDir,
    allFiles,
    pages,
    errors,
  );
  const redirects = redirectConfigPath
    ? await inspectRedirects(
        resolve(redirectConfigPath),
        pages,
        sitemapUrls,
        errors,
      )
    : [];
  await inspectSocialImages(buildDir, allFiles, pages, errors);

  errors.sort(
    (left, right) =>
      left.file.localeCompare(right.file) ||
      left.code.localeCompare(right.code) ||
      left.message.localeCompare(right.message),
  );

  return { buildDir, errors, pages, redirects, sitemapUrls };
}

export function formatReport(result) {
  if (!result.errors.length) {
    const indexableCount = result.pages.filter((page) => page.indexable).length;
    const socialCardCount = result.pages.filter((page) => page.ogImage).length;
    return [
      `SEO validation passed: ${indexableCount} indexable page${
        indexableCount === 1 ? '' : 's'
      }, ${result.pages.length} generated HTML page${
        result.pages.length === 1 ? '' : 's'
      }, ${socialCardCount} route-specific social card${
        socialCardCount === 1 ? '' : 's'
      }, ${result.redirects.length} redirect${
        result.redirects.length === 1 ? '' : 's'
      }, and ${result.sitemapUrls.length} sitemap URL${
        result.sitemapUrls.length === 1 ? '' : 's'
      } checked.`,
    ].join('\n');
  }

  const lines = [
    `SEO validation failed with ${result.errors.length} actionable error${
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
  const result = await auditSeo(buildDirectory, {
    redirectConfigPath: DEFAULT_REDIRECT_CONFIG,
  });
  const report = formatReport(result);

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
