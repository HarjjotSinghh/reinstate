import { readdir, readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';

const docsDir = new URL('../content/docs/', import.meta.url);
const intents = new Set([
  'navigational',
  'problem',
  'solution',
  'how-to',
  'troubleshooting',
  'comparison',
  'evaluation',
]);

function field(frontmatter: string, name: string): string | undefined {
  return frontmatter.match(new RegExp(`^${name}:\\s*(.+)$`, 'm'))?.[1]?.trim();
}

describe('documentation content metadata', () => {
  it('keeps every document explicit, current, and ready for search metadata', async () => {
    const files = (await readdir(docsDir)).filter((file) => /\.mdx?$/.test(file));

    expect(files).not.toContain('README.md');

    for (const file of files) {
      const source = await readFile(new URL(file, docsDir), 'utf8');
      const match = source.match(/^---\n([\s\S]+?)\n---\n/);

      expect(match, `${file} must start with frontmatter`).not.toBeNull();
      const frontmatter = match?.[1] ?? '';
      const description = field(frontmatter, 'description')?.replace(/^["']|["']$/g, '');
      const searchIntent = field(frontmatter, 'searchIntent')?.replace(/^["']|["']$/g, '');
      const prose = source.replace(/```[\s\S]*?```/g, '');

      expect(field(frontmatter, 'title'), `${file} title`).toBeTruthy();
      expect(field(frontmatter, 'author'), `${file} author`).toBeTruthy();
      expect(field(frontmatter, 'status'), `${file} status`).toMatch(
        /^(current|planned|deprecated)$/,
      );
      expect(description?.length, `${file} description length`).toBeGreaterThanOrEqual(70);
      expect(description?.length, `${file} description length`).toBeLessThanOrEqual(180);
      expect(field(frontmatter, 'updatedAt'), `${file} updatedAt`).toMatch(/^\d{4}-\d{2}-\d{2}$/);
      expect(field(frontmatter, 'tags'), `${file} tags`).toMatch(/^\[.+\]$/);
      expect(field(frontmatter, 'targetQuery'), `${file} targetQuery`).toBeTruthy();
      expect(intents.has(searchIntent ?? ''), `${file} searchIntent`).toBe(true);
      expect(field(frontmatter, 'draft'), `${file} draft`).toMatch(/^(true|false)$/);
      expect(field(frontmatter, 'noindex'), `${file} noindex`).toMatch(/^(true|false)$/);
      expect(prose, `${file} must let DocsLayout own the single H1`).not.toMatch(/^#\s/m);
    }
  });
});
