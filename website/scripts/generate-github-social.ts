import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { renderRepositorySocialPreview } from '../src/lib/og-card.ts';

const websiteDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repositoryDirectory = resolve(websiteDirectory, '..');
const targets = [
  resolve(websiteDirectory, 'public/brand/github-social.png'),
  resolve(repositoryDirectory, 'assets/brand-kit/png/github-social-1280x640.png'),
];
const checkOnly = process.argv.includes('--check');

const rendered = await renderRepositorySocialPreview();

for (const target of targets) {
  if (checkOnly) {
    const current = await readFile(target).catch(() => undefined);
    if (!current?.equals(rendered)) {
      throw new Error(
        `${target} is stale; run "npm run generate:github-social" from website/`,
      );
    }
    continue;
  }

  await mkdir(dirname(target), { recursive: true });
  await writeFile(target, rendered);
}

console.log(
  checkOnly
    ? `GitHub social preview is current (${rendered.byteLength} bytes).`
    : `Generated GitHub social preview (${rendered.byteLength} bytes).`,
);
