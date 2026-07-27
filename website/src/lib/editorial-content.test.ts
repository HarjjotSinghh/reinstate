import { readdir, readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import { blogPostingSchema, howToSchema, techArticleSchema } from './schema';

const guidesDir = new URL('../content/guides/', import.meta.url);
const blogDir = new URL('../content/blog/', import.meta.url);
const editorialLayout = new URL('../layouts/EditorialLayout.astro', import.meta.url);

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
        if (directory === guidesDir) {
          expect(field(frontmatter, 'estimatedTaskMinutes'), `${file} task duration`).toMatch(
            /^\d+$/,
          );
        }
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

  it('gives every session-sync guide a complete, visible AEO task contract', async () => {
    const guideNames = await markdownFiles(guidesDir);

    for (const file of guideNames) {
      const guide = await readFile(new URL(file, guidesDir), 'utf8');

      for (const requiredSection of [
        '## Key points',
        '## Command placeholders and parameters',
        '## Failure modes and common errors',
        '## Safe rollback and undo',
        '## Verification checklist',
      ]) {
        expect(guide, `${file} must include ${requiredSection}`).toContain(
          requiredSection,
        );
      }

      for (const requiredFact of [
        'Expected result:',
        'Installer-compatible',
        'not a certified Phase 1 agent-resume target',
        'physical two-device acceptance',
        'does not provide a general `rein undo`',
        '`--dry-run`',
        'same-vendor',
      ]) {
        expect(guide, `${file} must include ${requiredFact}`).toContain(
          requiredFact,
        );
      }

      expect(
        guide.match(/\*\*Expected result:\*\*/g)?.length,
        `${file} expected outputs`,
      ).toBe(5);
      expect(guide, `${file} FAQ heading`).toMatch(/^## .+ FAQ$/m);

      const failureSection =
        guide.match(
          /^## Failure modes and common errors\n([\s\S]+?)^## Safe rollback and undo$/m,
        )?.[1] ?? '';
      const failureQuestions = [
        ...failureSection.matchAll(/^### (.+)$/gm),
      ].map((match) => match[1]);
      expect(failureQuestions.length, `${file} question-shaped failures`).toBeGreaterThanOrEqual(8);
      expect(
        failureQuestions.every((heading) => heading.endsWith('?')),
        `${file} failure headings must be questions`,
      ).toBe(true);
      expect(failureSection, `${file} must not hide failures in a table`).not.toMatch(
        /^\| (Symptom|Error)/m,
      );

      const anchors = [...guide.matchAll(/^\s+anchor: "([^"]+)"$/gm)].map(
        (match) => match[1],
      );
      expect(anchors, `${file} structured steps`).toHaveLength(5);
      expect(new Set(anchors).size, `${file} unique structured steps`).toBe(
        anchors.length,
      );
      for (const anchor of anchors) {
        expect(
          guide,
          `${file} must expose #${anchor} in visible content`,
        ).toContain(`<h2 id="${anchor}">`);
      }
    }
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

  it('links broad workflows to the relevant agent integration pages', async () => {
    const files = [
      new URL('move-a-coding-agent-session-from-mac-to-windows.md', guidesDir),
      new URL('use-s3-for-coding-agent-session-storage.md', guidesDir),
      new URL('use-cloudflare-r2-for-coding-agent-session-storage.md', guidesDir),
      new URL('why-git-does-not-sync-coding-agent-sessions.md', blogDir),
    ];

    for (const file of files) {
      const content = await readFile(file, 'utf8');
      expect(content).toContain('path: "/integrations/claude-code"');
      expect(content).toContain('path: "/integrations/codex"');
    }
  });

  it('uses the same category for visible blog metadata, JSON-LD, and Open Graph', async () => {
    const layout = await readFile(editorialLayout, 'utf8');

    expect(layout).toContain('articleSection: category');
    expect(layout).toContain("{kind === 'guide' ? 'Practical guide' : category}");
    expect(layout).toContain("section={kind === 'guide' ? 'Guides' : category}");
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

  it('builds HowTo schema from the same ordered guide steps shown to readers', () => {
    const steps = [
      {
        name: 'Install the source tool',
        text: 'Install the pinned release, verify its version, and stop if compatibility checks do not pass on the source device.',
        anchor: 'install-source',
      },
      {
        name: 'Restore on the destination',
        text: 'Preview the destination plan, restore the exact selected session, and verify it through the vendor-native resume command.',
        anchor: 'restore-destination',
      },
    ];
    const schema = howToSchema({
      path: '/guides/example',
      title: 'Sync one coding-agent session',
      description: 'A visible and testable session-sync procedure.',
      estimatedTaskMinutes: 30,
      steps,
    });

    expect(schema['@type']).toBe('HowTo');
    expect(schema.totalTime).toBe('PT30M');
    expect(schema.step).toEqual(
      steps.map((step, index) => ({
        '@type': 'HowToStep',
        position: index + 1,
        name: step.name,
        text: step.text,
        url: `https://reinstate.dev/guides/example#${step.anchor}`,
      })),
    );
    expect(schema).not.toHaveProperty('review');
    expect(schema).not.toHaveProperty('aggregateRating');
  });
});
