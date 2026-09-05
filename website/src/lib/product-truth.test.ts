import { readdir, readFile } from 'node:fs/promises';
import { extname } from 'node:path';
import { describe, expect, it } from 'vitest';
import compatibility from '../data/compatibility.json';
import releasedTiers from '../data/released-tiers.json';
import { product } from '../data/product';
import { compatibilityAgents } from './agent-catalog';
import { homepageSchema } from './schema';

const sourceRoot = new URL('../', import.meta.url);
const currentRc = `RC${product.currentRelease.match(/-rc\.(\d+)$/)?.[1] ?? ''}`;

async function sourceFiles(directory: URL): Promise<URL[]> {
  const entries = await readdir(directory, { withFileTypes: true });
  const files: URL[] = [];

  for (const entry of entries) {
    const path = new URL(entry.name + (entry.isDirectory() ? '/' : ''), directory);
    if (entry.isDirectory()) {
      files.push(...(await sourceFiles(path)));
    } else if (
      ['.astro', '.json', '.md', '.mdx', '.ts'].includes(extname(entry.name)) &&
      !entry.name.endsWith('.test.ts')
    ) {
      files.push(path);
    }
  }

  return files;
}

describe('central product-truth drift guard', () => {
  it('links the exact current tag and the real installer download', () => {
    expect(product.currentReleaseUrl).toBe(
      `${product.repositoryUrl}/tree/${product.currentRelease}`,
    );
    const software = homepageSchema().find(
      (entry) => entry['@type'] === 'SoftwareApplication',
    );
    expect(software?.downloadUrl).toBe(`${product.siteUrl}/install.sh`);
  });

  it('keeps every document frontmatter version on the canonical release', async () => {
    const contentRoot = new URL('../content/', import.meta.url);
    const files = (await sourceFiles(contentRoot)).filter((path) =>
      ['.md', '.mdx'].includes(extname(path.pathname)),
    );

    for (const path of files) {
      const source = await readFile(path, 'utf8');
      const version = source.match(/^version:\s*["']?([^"'\n]+)["']?\s*$/m)?.[1];
      const status = source.match(/^status:\s*["']?([^"'\n]+)["']?\s*$/m)?.[1];
      if (version) {
        expect(version, path.pathname).toBe(
          status === 'planned' ? 'roadmap' : product.currentRelease,
        );
      }
    }
  });

  it('rejects stale Reinstate release-candidate labels outside canonical data', async () => {
    const files = await sourceFiles(sourceRoot);
    const stale: string[] = [];

    for (const path of files) {
      if (
        path.pathname.endsWith('/data/product.ts') ||
        path.pathname.endsWith('/data/releases.ts') ||
        path.pathname.endsWith('/data/compatibility.json') ||
        path.pathname.endsWith('/data/agent-version-history.ts') ||
        path.pathname.endsWith('/pages/changelog.astro') ||
        path.pathname.endsWith(
          '/pages/compatibility/agent-version-history.astro',
        ) ||
        path.pathname.endsWith('/pages/research/index.astro')
      ) {
        continue;
      }

      const source = await readFile(path, 'utf8');
      const releaseLabels = [
        ...source.matchAll(/\bv?\d+\.\d+\.\d+-rc\.\d+\b|\bRC\d+\b/g),
      ].map((match) => match[0]);

      for (const label of releaseLabels) {
        const normalized = label.toUpperCase().startsWith('RC')
          ? label.toUpperCase()
          : label.replace(/^v/, '');
        const expected = normalized.startsWith('RC')
          ? currentRc
          : product.currentRelease.replace(/^v/, '');
        if (normalized !== expected) {
          stale.push(`${path.pathname}: ${label}`);
        }
      }
    }

    expect(stale).toEqual([]);
  });

  it('keeps the catalog line from claiming an unpublished tag or extra T5 agents', () => {
    expect(product.currentRelease).toBe('v0.5.2-rc.1');
    // Stable deliberately lags the candidate: the interactive surfaces have
    // development verification but no tagged-artifact acceptance yet.
    expect(product.stableRelease).toBe('v0.5.1');
    expect(compatibility.reinstateVersion).toBe(product.currentRelease);
    expect(compatibility.catalogLine).toBe(product.currentRelease);
    expect(compatibilityAgents.filter((agent) => agent.tier === 'T5').map((agent) => agent.key)).toEqual(
      ['claude', 'codex'],
    );
    // T3 is empty, and that is not a gap. Every agent with a verified resume
    // journey has also earned a destination, so nothing currently stops at the
    // rung between them. A new agent landing at T3 must add itself here.
    expect(compatibilityAgents.filter((agent) => agent.tier === 'T3')).toEqual([]);
    expect(compatibilityAgents.filter((agent) => agent.tier === 'T0')).toHaveLength(7);
  });

  // The published matrix describes a *release*, not main. A user or coding
  // agent reading it cannot install main, so a tier that exists only there is a
  // claim they cannot act on.
  //
  // This is what went wrong once already: compatibility.json advertised three
  // agents at T4 while declaring itself reviewed against v0.5.1, a release
  // whose binary had them at T2, T2 and T1. Nothing caught it, because the
  // tiers were correct about main and the version field was maintained by hand
  // and had not moved since the file was created.
  it('pins every published tier to the release the matrix names', () => {
    expect(compatibility.reinstateVersion).toBe(releasedTiers.release);
    const published = Object.fromEntries(
      compatibilityAgents.map((agent) => [agent.key, agent.tier]),
    );
    expect(published).toEqual(releasedTiers.tiers);
  });

  it('keeps the doctor self-test distinct from real remote-storage evidence', async () => {
    const reference = await readFile(
      new URL('../content/docs/cli-reference.md', import.meta.url),
      'utf8',
    );
    const doctorSection =
      reference.match(
        /### `rein doctor`([\s\S]*?)(?=\n### `rein setup check`)/,
      )?.[1] ?? '';

    expect(doctorSection).toContain('in-memory sync');
    expect(doctorSection.toLowerCase()).toMatch(
      /this\s+command does not prove remote storage access/,
    );
    expect(doctorSection).not.toContain('configured storage without');
  });
});
