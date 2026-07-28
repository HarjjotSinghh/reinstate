import { readdir, readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';

const briefsRoot = new URL('../../../docs/seo/content-briefs/', import.meta.url);
const guidesRoot = new URL('../content/guides/', import.meta.url);
const blogRoot = new URL('../content/blog/', import.meta.url);

const requiredHeadings = [
  '## Page',
  '## Audience and intent',
  '## Product truth',
  '## Outline',
  '## Acceptance criteria',
] as const;

async function markdownBasenames(directory: URL): Promise<string[]> {
  return (await readdir(directory))
    .filter((file) => /\.mdx?$/.test(file))
    .map((file) => file.replace(/\.mdx?$/, ''));
}

describe('content-brief workflow', () => {
  it('has a reviewed brief for every published guide and blog article', async () => {
    const expected = [
      ...(await markdownBasenames(guidesRoot)),
      ...(await markdownBasenames(blogRoot)),
    ];
    const briefs = new Set(await markdownBasenames(briefsRoot));

    for (const slug of expected) {
      expect(briefs.has(slug), `missing content brief for ${slug}`).toBe(true);
    }
  });

  it.each([
    'cli-reference',
    'move-a-coding-agent-session-from-mac-to-windows',
    'sync-claude-code-sessions-across-devices',
    'sync-codex-sessions-across-devices',
    'use-cloudflare-r2-for-coding-agent-session-storage',
    'use-s3-for-coding-agent-session-storage',
    'why-git-does-not-sync-coding-agent-sessions',
  ])('%s records intent, truth, outline, and acceptance', async (slug) => {
    const source = await readFile(new URL(`${slug}.md`, briefsRoot), 'utf8');

    for (const heading of requiredHeadings) {
      expect(source, `${slug}: ${heading}`).toContain(heading);
    }
    expect(source, `${slug}: reviewed date`).toMatch(
      /Last reviewed: 2026-07-27/,
    );
    expect(source, `${slug}: release`).toContain('v0.1.0-rc.6');
    expect(source, `${slug}: route`).toMatch(/- URL: `\/[^`]+`/);
  });
});
