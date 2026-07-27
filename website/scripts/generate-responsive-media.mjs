#!/usr/bin/env node

import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import sharp from 'sharp';

const media = [
  {
    source: 'public/brand/01_landscape.png',
    outputs: [
      { path: 'public/brand/01_landscape-768.webp', width: 768 },
      { path: 'public/brand/01_landscape-1536.webp', width: 1536 },
    ],
  },
  {
    source: 'public/brand/05_architecture.png',
    outputs: [
      { path: 'public/brand/05_architecture-768.webp', width: 768 },
      { path: 'public/brand/05_architecture-1536.webp', width: 1536 },
    ],
  },
];

const checkOnly = process.argv.includes('--check');

async function sameBytes(path, expected) {
  try {
    const existing = await readFile(path);
    return existing.equals(expected);
  } catch {
    return false;
  }
}

let stale = false;

for (const asset of media) {
  const sourcePath = resolve(asset.source);

  for (const output of asset.outputs) {
    const outputPath = resolve(output.path);
    const rendered = await sharp(sourcePath)
      .resize({ width: output.width, withoutEnlargement: true })
      .webp({ effort: 6, quality: 82 })
      .toBuffer();

    if (await sameBytes(outputPath, rendered)) {
      continue;
    }

    stale = true;
    if (!checkOnly) {
      await mkdir(dirname(outputPath), { recursive: true });
      await writeFile(outputPath, rendered);
      console.log(`Generated ${output.path}`);
    } else {
      console.error(`Responsive media is missing or stale: ${output.path}`);
    }
  }
}

if (checkOnly && stale) {
  console.error('Run "npm run generate:media" and commit the generated files.');
  process.exitCode = 1;
} else if (checkOnly) {
  console.log(`Responsive media validation passed: ${media.length} source images.`);
}
