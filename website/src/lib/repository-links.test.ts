import { readdir, readFile, stat } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';

const websiteSource = new URL('../', import.meta.url);
const repositoryRoot = new URL('../../../', import.meta.url);

/**
 * Every link on the site into this repository, with the ref it names.
 *
 * check-links.mjs cannot see these. It audits the built pages but skips any
 * link whose origin is not the site's own, counting it as externalSkipped — so
 * a link to a branch that no longer exists passes every check the site has.
 */
const BLOB_LINK = /github\.com\/HarjjotSinghh\/reinstate\/blob\/([^/"')<\s]+)\/([^"')<\s]+)/g;

/** A released tag. These are permalinks and are supposed to be pinned. */
const VERSION_TAG = /^v\d+\.\d+\.\d+(?:-rc\.\d+)?$/;

/** Astro/TS interpolation, resolved at build time from release data. */
const INTERPOLATED = /^\$\{.*\}$/;

const SOURCE_EXTENSIONS = new Set(['.astro', '.json', '.md', '.mdx', '.ts', '.tsx']);

type Link = { file: string; ref: string; path: string };

async function collectLinks(directory: URL): Promise<Link[]> {
  const found: Link[] = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const child = new URL(`${entry.name}${entry.isDirectory() ? '/' : ''}`, directory);
    if (entry.isDirectory()) {
      found.push(...(await collectLinks(child)));
      continue;
    }
    const dot = entry.name.lastIndexOf('.');
    if (dot < 0 || !SOURCE_EXTENSIONS.has(entry.name.slice(dot))) continue;

    const body = await readFile(child, 'utf8');
    for (const match of body.matchAll(BLOB_LINK)) {
      found.push({
        file: child.pathname.slice(repositoryRoot.pathname.length),
        ref: match[1],
        path: match[2].split('#')[0],
      });
    }
  }
  return found;
}

describe('links into this repository', () => {
  it('names a ref that will still exist tomorrow', async () => {
    const links = await collectLinks(websiteSource);
    expect(links.length).toBeGreaterThan(0);

    // A branch is not a durable target for a published page. Nine integration
    // pages and sixteen compatibility entries pointed at
    // feat/universal-agent-coverage, a stale integration branch that PR #274
    // also targeted, long after v0.5.0 shipped from main.
    const unstable = links.filter(
      (link) =>
        link.ref !== 'main' && !VERSION_TAG.test(link.ref) && !INTERPOLATED.test(link.ref),
    );

    expect(
      unstable.map((link) => `${link.file} -> blob/${link.ref}/${link.path}`),
      'published pages must link to main or a released tag, never a branch',
    ).toEqual([]);
  });

  it('names a path that exists', async () => {
    const links = await collectLinks(websiteSource);

    // Repointing a dead branch at main only helps if the file is actually
    // there. Tags are skipped: they legitimately name paths as they were at
    // that release, which may since have moved.
    const missing: string[] = [];
    for (const link of links.filter((candidate) => candidate.ref === 'main')) {
      try {
        await stat(new URL(link.path, repositoryRoot));
      } catch {
        missing.push(`${link.file} -> ${link.path}`);
      }
    }

    expect(missing, 'a link to main must name a path that exists on main').toEqual([]);
  });
});
