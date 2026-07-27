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

function reviewedRecord(id, value) {
  return { id, value };
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

export async function collectFreshnessRecords(root = DEFAULT_ROOT) {
  const records = [];
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
        records.push(reviewedRecord(id, value));
      }
    }
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
      } else if (entry.status !== 'release-gate-open') {
        errors.push(
          `src/data/compatibility.json#${entry.id}: missing lastTested without an open release gate`,
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

  return { records, errors };
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
