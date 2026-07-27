import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';

const docsRoot = new URL('../content/docs/', import.meta.url);
const documents = [
  'installation.md',
  'configuration.md',
  'storage.md',
  'sync-a-session.md',
  'restore-a-session.md',
  'limitations.md',
] as const;

const requiredFrontmatter = [
  'title',
  'description',
  'order',
  'author',
  'status',
  'updatedAt',
  'tags',
  'targetQuery',
  'searchIntent',
  'draft',
  'noindex',
] as const;

const requiredSections = [
  'Prerequisites',
  'Expected evidence',
  'Failure paths',
  'Security boundaries',
  'Related pages',
] as const;

function frontmatterField(frontmatter: string, field: string): string | undefined {
  return frontmatter.match(new RegExp(`^${field}:\\s*(.+)$`, 'm'))?.[1]?.trim();
}

function sectionBody(body: string, heading: string): string {
  const marker = `## ${heading}\n`;
  const start = body.indexOf(marker);
  if (start === -1) return '';

  const remaining = body.slice(start + marker.length);
  const nextHeading = remaining.search(/^##\s+/m);
  return (nextHeading === -1 ? remaining : remaining.slice(0, nextHeading)).trim();
}

describe('operational documentation contract', () => {
  it.each(documents)('%s is current, answer-first, and operationally complete', async (file) => {
    const source = await readFile(new URL(file, docsRoot), 'utf8');
    const match = source.match(/^---\n([\s\S]+?)\n---\n\n([\s\S]+)$/);

    expect(match, `${file} frontmatter and body`).not.toBeNull();
    const frontmatter = match?.[1] ?? '';
    const body = match?.[2] ?? '';

    for (const field of requiredFrontmatter) {
      expect(frontmatterField(frontmatter, field), `${file}: ${field}`).toBeTruthy();
    }

    expect(frontmatterField(frontmatter, 'status')).toBe('current');
    expect(frontmatterField(frontmatter, 'draft')).toBe('false');
    expect(frontmatterField(frontmatter, 'noindex')).toBe('false');
    expect(body).toContain('v0.1.0-rc.6');

    const opening = body.split(/^##\s+/m)[0]?.replace(/^>\s?/gm, '').trim() ?? '';
    expect(opening.length, `${file}: answer-first opening`).toBeGreaterThan(120);

    const headings = [...body.matchAll(/^##\s+(.+)$/gm)].map((heading) => heading[1]);
    for (const section of requiredSections) {
      expect(headings, `${file}: ${section}`).toContain(section);
    }

    const failureSection = sectionBody(body, 'Failure paths');
    expect(failureSection, `${file}: troubleshooting route`).toContain(
      '/docs/troubleshooting',
    );

    const relatedSection = sectionBody(body, 'Related pages');
    const relatedLinks = [
      ...relatedSection.matchAll(/\]\((\/[a-z0-9/#-]+)\)/g),
    ].map((link) => link[1]);
    expect(relatedLinks.length, `${file}: contextual related links`).toBeGreaterThanOrEqual(
      3,
    );
  });

  it('keeps planned product surfaces out of current command examples', async () => {
    const sources = await Promise.all(
      documents.map((file) => readFile(new URL(file, docsRoot), 'utf8')),
    );
    const commandBlocks = sources
      .flatMap((source) => [...source.matchAll(/```(?:sh|bash|powershell|text)\n([\s\S]*?)```/g)])
      .map((match) => match[1])
      .join('\n');

    expect(commandBlocks).not.toMatch(/\brein (?:search|resume|handoff|mcp|skill|plugin)\b/);
    expect(commandBlocks).not.toContain('REINSTATE_PASSPHRASE=');
    expect(sources.join('\n')).not.toContain('cross-agent resume');
  });
});
