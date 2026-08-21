import { readdir, readFile } from 'node:fs/promises';
import { extname } from 'node:path';
import { describe, expect, it } from 'vitest';
import compatibility from '../data/compatibility.json';
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

  it('keeps the v0.5.0 catalog line from claiming an unpublished tag or extra T5 agents', () => {
    expect(product.currentRelease).toBe('v0.5.0');
    expect(product.stableRelease).toBe('v0.5.0');
    expect(compatibility.reinstateVersion).toBe('v0.5.0');
    expect(compatibility.catalogLine).toBe('v0.5.0');
    expect(compatibilityAgents.filter((agent) => agent.tier === 'T5').map((agent) => agent.key)).toEqual(
      ['claude', 'codex'],
    );
    expect(
      compatibilityAgents.filter((agent) => agent.tier === 'T2').map((agent) => agent.key).sort(),
    ).toEqual(['gemini', 'grok', 'opencode']);
    expect(compatibilityAgents.filter((agent) => agent.tier === 'T3')).toEqual([]);
    expect(
      compatibilityAgents.filter((agent) => agent.tier === 'T1').map((agent) => agent.key),
    ).toEqual(['kimi', 'pi', 'qwen', 'cursor', 'cline', 'copilot']);
    expect(compatibilityAgents.filter((agent) => agent.tier === 'T0')).toHaveLength(7);
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
