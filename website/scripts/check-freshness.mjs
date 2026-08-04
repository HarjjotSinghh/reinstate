#!/usr/bin/env node

import { readdir, readFile } from 'node:fs/promises';
import { relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const DAY_MS = 24 * 60 * 60 * 1000;
const DEFAULT_ROOT = resolve(process.cwd());

function parseDate(value) {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return null;
  }
  const parsed = new Date(`${value}T00:00:00Z`);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

function reviewedRecord(id, value, route = null) {
  return { id, value, route };
}

function frontmatterValue(source, key) {
  const frontmatter = source.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!frontmatter) {
    return null;
  }
  const match = frontmatter[1].match(
    new RegExp(`^${key}:\\s*["']?([^"'\\s]+)["']?\\s*$`, 'm'),
  );
  return match?.[1] ?? null;
}

async function markdownFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await markdownFiles(path)));
    } else if (/\.mdx?$/.test(entry.name)) {
      files.push(path);
    }
  }
  return files;
}

function contentRoute(root, collection, path) {
  const relativePath = relative(resolve(root, 'src/content', collection), path)
    .replaceAll('\\', '/')
    .replace(/\.mdx?$/, '')
    .replace(/\/index$/, '');
  return `/${collection}/${relativePath}`.replace(/\/+$/, '');
}

async function sitemapRoutes(root) {
  const outputRoot = resolve(root, 'dist/client');
  let entries;
  try {
    entries = await readdir(outputRoot, { withFileTypes: true });
  } catch {
    return null;
  }
  const routes = new Set();
  for (const entry of entries) {
    if (!entry.isFile() || !/^sitemap.*\.xml$/.test(entry.name)) continue;
    const xml = await readFile(resolve(outputRoot, entry.name), 'utf8');
    for (const match of xml.matchAll(/<loc>([^<]+)<\/loc>/g)) {
      try {
        const url = new URL(match[1]);
        if (url.hostname === 'reinstate.dev' && !url.pathname.endsWith('.xml')) {
          routes.add(url.pathname === '/' ? '/' : url.pathname.replace(/\/+$/, ''));
        }
      } catch {
        // The SEO gate owns malformed sitemap URL reporting.
      }
    }
  }
  return routes;
}

export async function collectFreshnessRecords(root = DEFAULT_ROOT) {
  const records = [];
  const publications = [];
  const errors = [];
  const contentRoot = resolve(root, 'src/content');

  for (const collection of ['docs', 'guides', 'blog']) {
    const collectionRoot = resolve(contentRoot, collection);
    let files = [];
    try {
      files = await markdownFiles(collectionRoot);
    } catch (error) {
      errors.push(`${collectionRoot}: ${error.message}`);
      continue;
    }

    for (const path of files) {
      const source = await readFile(path, 'utf8');
      const key = collection === 'docs' ? 'updatedAt' : 'reviewedAt';
      const value = frontmatterValue(source, key);
      const id = relative(root, path).replaceAll('\\', '/');
      if (!value) {
        errors.push(`${id}: missing ${key} frontmatter`);
      } else {
        const route = contentRoute(root, collection, path);
        records.push(reviewedRecord(id, value, route));
        if (collection !== 'docs') {
          const publishedAt = frontmatterValue(source, 'publishedAt');
          if (!publishedAt) {
            errors.push(`${id}: missing publishedAt frontmatter`);
          } else {
            publications.push(
              reviewedRecord(`${id}#publishedAt`, publishedAt, route),
            );
          }
        }
      }
    }
  }

  try {
    const reviewPath = resolve(root, 'src/data/static-page-reviews.json');
    const reviews = JSON.parse(await readFile(reviewPath, 'utf8'));
    if (!Array.isArray(reviews)) {
      errors.push('src/data/static-page-reviews.json: expected an array');
    } else {
      const seenRoutes = new Set();
      for (const [index, review] of reviews.entries()) {
        const id = `src/data/static-page-reviews.json#${index}`;
        if (
          typeof review.route !== 'string' ||
          !/^\/(?:[a-z0-9]+(?:[/-][a-z0-9]+)*)?$/.test(review.route)
        ) {
          errors.push(`${id}: invalid route`);
          continue;
        }
        if (seenRoutes.has(review.route)) {
          errors.push(`${id}: duplicate route ${review.route}`);
        }
        seenRoutes.add(review.route);
        if (typeof review.owner !== 'string' || review.owner.trim().length < 2) {
          errors.push(`${id}: missing owner`);
        }
        if (
          !Array.isArray(review.sources) ||
          review.sources.length === 0 ||
          review.sources.some(
            (source) =>
              typeof source !== 'string' ||
              (!source.startsWith('src/') &&
                !source.startsWith('docs/') &&
                !source.startsWith('../') &&
                source !== 'PRODUCT.md' &&
                !source.startsWith('https://')),
          )
        ) {
          errors.push(`${id}: sources must contain reviewed local paths or HTTPS URLs`);
        }
        records.push(reviewedRecord(id, review.reviewedAt, review.route));
      }
    }
  } catch (error) {
    errors.push(`src/data/static-page-reviews.json: ${error.message}`);
  }

  try {
    const compatibilityPath = resolve(root, 'src/data/compatibility.json');
    const compatibility = JSON.parse(await readFile(compatibilityPath, 'utf8'));
    records.push(
      reviewedRecord('src/data/compatibility.json#lastReviewed', compatibility.lastReviewed),
    );

    for (const entry of [...(compatibility.agents ?? []), ...(compatibility.environments ?? [])]) {
      if (entry.lastTested) {
        records.push(
          reviewedRecord(
            `src/data/compatibility.json#${entry.id}.lastTested`,
            entry.lastTested,
          ),
        );
      } else if (
        entry.status !== 'release-gate-open' &&
        entry.status !== 'preview-unverified'
      ) {
        errors.push(
          `src/data/compatibility.json#${entry.id}: missing lastTested without an open or preview gate`,
        );
      }
      if (
        typeof entry.source !== 'string' ||
        !entry.source.startsWith('https://github.com/HarjjotSinghh/reinstate/')
      ) {
        errors.push(
          `src/data/compatibility.json#${entry.id}: missing canonical evidence source`,
        );
      }
    }
  } catch (error) {
    errors.push(`src/data/compatibility.json: ${error.message}`);
  }

  try {
    const productPath = resolve(root, 'src/data/product.ts');
    const productSource = await readFile(productPath, 'utf8');
    const match = productSource.match(/lastVerified:\s*['"](\d{4}-\d{2}-\d{2})['"]/);
    if (!match) {
      errors.push('src/data/product.ts: missing product.lastVerified');
    } else {
      records.push(reviewedRecord('src/data/product.ts#lastVerified', match[1]));
    }
  } catch (error) {
    errors.push(`src/data/product.ts: ${error.message}`);
  }

  const indexableRoutes = await sitemapRoutes(root);
  if (indexableRoutes) {
    const reviewedRoutes = new Set(
      records.map(({ route }) => route).filter((route) => typeof route === 'string'),
    );
    for (const route of indexableRoutes) {
      if (!reviewedRoutes.has(route)) {
        errors.push(`sitemap route ${route}: missing freshness owner and source record`);
      }
    }
    for (const route of reviewedRoutes) {
      if (!indexableRoutes.has(route)) {
        errors.push(`freshness route ${route}: not present in the generated sitemap`);
      }
    }
  }

  const routeOwners = new Map();
  for (const record of records.filter(({ route }) => route)) {
    if (routeOwners.has(record.route)) {
      errors.push(
        `${record.id}: duplicate freshness route ${record.route} also owned by ${routeOwners.get(record.route)}`,
      );
    } else {
      routeOwners.set(record.route, record.id);
    }
  }

  return { records, publications, errors };
}

export async function auditFreshness({
  root = DEFAULT_ROOT,
  asOf = new Date(),
  warnAfterDays = 60,
  failAfterDays = 120,
} = {}) {
  const collected = await collectFreshnessRecords(root);
  const warnings = [];
  const errors = [...collected.errors];
  const current = new Date(asOf);

  if (Number.isNaN(current.getTime())) {
    return {
      records: collected.records,
      warnings,
      errors: [...errors, `Invalid as-of date: ${String(asOf)}`],
    };
  }

  for (const record of collected.records) {
    const reviewed = parseDate(record.value);
    if (!reviewed) {
      errors.push(`${record.id}: invalid review date ${JSON.stringify(record.value)}`);
      continue;
    }
    const ageDays = Math.floor((current.getTime() - reviewed.getTime()) / DAY_MS);
    if (ageDays < 0) {
      errors.push(`${record.id}: review date ${record.value} is in the future`);
    } else if (ageDays > failAfterDays) {
      errors.push(
        `${record.id}: ${ageDays} days since review (failure threshold ${failAfterDays})`,
      );
    } else if (ageDays > warnAfterDays) {
      warnings.push(
        `${record.id}: ${ageDays} days since review (warning threshold ${warnAfterDays})`,
      );
    }
  }

  for (const publication of collected.publications) {
    const published = parseDate(publication.value);
    if (!published) {
      errors.push(
        `${publication.id}: invalid publication date ${JSON.stringify(publication.value)}`,
      );
    } else if (published.getTime() > current.getTime()) {
      errors.push(
        `${publication.id}: publication date ${publication.value} is in the future`,
      );
    }
  }

  return { records: collected.records, warnings, errors };
}

export function formatFreshnessReport(result) {
  const lines = [
    `Freshness audit: ${result.records.length} reviewed records, ${result.warnings.length} warnings, ${result.errors.length} errors.`,
  ];
  if (result.warnings.length) {
    lines.push('', 'Warnings:', ...result.warnings.map((item) => `- ${item}`));
  }
  if (result.errors.length) {
    lines.push('', 'Errors:', ...result.errors.map((item) => `- ${item}`));
  }
  return lines.join('\n');
}

function argumentValue(name) {
  const index = process.argv.indexOf(name);
  return index === -1 ? null : process.argv[index + 1] ?? null;
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  const asOfArgument = argumentValue('--as-of');
  const result = await auditFreshness({
    asOf: asOfArgument ? new Date(`${asOfArgument}T00:00:00Z`) : new Date(),
  });
  console.log(formatFreshnessReport(result));
  if (result.errors.length) {
    process.exitCode = 1;
  }
}
