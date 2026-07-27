import { readdir, readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import { blogPostingSchema, techArticleSchema } from './schema';

const guidesDir = new URL('../content/guides/', import.meta.url);
const blogDir = new URL('../content/blog/', import.meta.url);

function field(frontmatter: string, name: string): string | undefined {
  return frontmatter.match(new RegExp(`^${name}:\\s*(.+)$`, 'm'))?.[1]?.trim();
}

async function markdownFiles(directory: URL) {
  return (await readdir(directory)).filter((file) => /\.mdx?$/.test(file));
}

describe('editorial content foundation', () => {
  it('keeps guide and blog metadata explicit and current', async () => {
    for (const directory of [guidesDir, blogDir]) {
      const files = await markdownFiles(directory);
      expect(files.length).toBeGreaterThan(0);

      for (const file of files) {
        const source = await readFile(new URL(file, directory), 'utf8');
        const match = source.match(/^---\n([\s\S]+?)\n---\n/);
        expect(match, `${file} must start with frontmatter`).not.toBeNull();

        const frontmatter = match?.[1] ?? '';
        const prose = source.replace(/^---\n[\s\S]+?\n---\n/, '').replace(/```[\s\S]*?```/g, '');
        const description = field(frontmatter, 'description')?.replace(/^["']|["']$/g, '');

        expect(field(frontmatter, 'title'), `${file} title`).toBeTruthy();
        expect(description?.length, `${file} description length`).toBeGreaterThanOrEqual(70);
        expect(description?.length, `${file} description length`).toBeLessThanOrEqual(180);
        expect(field(frontmatter, 'publishedAt'), `${file} publishedAt`).toMatch(
          /^\d{4}-\d{2}-\d{2}$/,
        );
        expect(field(frontmatter, 'updatedAt'), `${file} updatedAt`).toMatch(
          /^\d{4}-\d{2}-\d{2}$/,
        );
        expect(field(frontmatter, 'reviewedAt'), `${file} reviewedAt`).toMatch(
          /^\d{4}-\d{2}-\d{2}$/,
        );
        expect(field(frontmatter, 'draft'), `${file} draft`).toMatch(/^(true|false)$/);
        expect(field(frontmatter, 'noindex'), `${file} noindex`).toMatch(/^(true|false)$/);
        expect(prose, `${file} must let EditorialLayout own the single H1`).not.toMatch(/^#\s/m);
      }
    }
  });

  it('documents the current same-vendor session workflow without planned commands', async () => {
    const guideNames = await markdownFiles(guidesDir);
    const guides = await Promise.all(
      guideNames.map((file) => readFile(new URL(file, guidesDir), 'utf8')),
    );

    for (const guide of guides) {
      expect(guide).toContain('rein setup check');
      expect(guide).toContain('--session SESSION_ID --dry-run');
      expect(guide).toContain('same-vendor');
      expect(guide).not.toContain('rein resume');
      expect(guide).not.toContain('cross-agent translation');
    }

    expect(guides.join('\n')).toContain('claude --resume SESSION_ID');
    expect(guides.join('\n')).toContain('codex resume SESSION_ID');
  });

  it('grounds the Git article in implementation and product boundaries', async () => {
    const article = await readFile(
      new URL('why-git-does-not-sync-coding-agent-sessions.md', blogDir),
      'utf8',
    );

    for (const evidence of [
      'internal/adapter',
      'internal/pathmap',
      'internal/crypto',
      'internal/sync',
      'Git remains source truth',
      'does not replace Git',
    ]) {
      expect(article).toContain(evidence);
    }
  });

  it('emits published and modified dates for guides and blog posts', () => {
    const dates = {
      publishedAt: new Date('2026-07-26T00:00:00Z'),
      updatedAt: new Date('2026-07-27T00:00:00Z'),
    };
    const guide = techArticleSchema({
      path: '/guides/example',
      title: 'Example guide',
      description: 'Example description.',
      ...dates,
    });
    const post = blogPostingSchema({
      path: '/blog/example',
      title: 'Example article',
      description: 'Example description.',
      ...dates,
    });

    expect(guide['@type']).toBe('TechArticle');
    expect(guide.datePublished).toBe(dates.publishedAt.toISOString());
    expect(guide.dateModified).toBe(dates.updatedAt.toISOString());
    expect(post['@type']).toBe('BlogPosting');
    expect(post.datePublished).toBe(dates.publishedAt.toISOString());
    expect(post.dateModified).toBe(dates.updatedAt.toISOString());
  });
});
