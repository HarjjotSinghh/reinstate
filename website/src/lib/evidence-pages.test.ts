import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import compatibility from '../data/compatibility.json';
import { staticOgPages } from '../data/og-pages';

const roadmapUrl = new URL('../pages/roadmap.astro', import.meta.url);
const researchUrl = new URL('../pages/research/index.astro', import.meta.url);

describe('roadmap and primary-evidence pages', () => {
  it('separates the current release candidate from every later product phase', async () => {
    const roadmap = await readFile(roadmapUrl, 'utf8');

    expect(roadmap).toContain('v0.1.0-rc.6');
    expect(roadmap).toContain('Release candidate');
    expect(roadmap.match(/<td>Planned<\/td>/g)?.length).toBeGreaterThanOrEqual(4);
    expect(roadmap.match(/<td>Exploring<\/td>/g)?.length).toBeGreaterThanOrEqual(2);
    expect(roadmap).toContain('Native resume and portable handoff stay distinct');
    expect(roadmap).toContain('Raw API keys');
    expect(roadmap).not.toContain('seamless');
  });

  it('publishes inspectable evidence without inventing benchmark results', async () => {
    const research = await readFile(researchUrl, 'utf8');

    for (const evidence of [
      '/compatibility.json',
      'compatibility.fixtureCommit',
      'phase-1-mac-windows-acceptance.md',
      'deterministic and synthetic',
      'No benchmark claim yet',
      'every failure, exclusion, limitation, and denominator',
    ]) {
      expect(research).toContain(evidence);
    }

    expect(compatibility.fixtureCommit).toMatch(/^[0-9a-f]{40}$/);
    expect(research).not.toMatch(/\b(?:9[0-9](?:\.\d+)?|100)% success\b/i);
    expect(research).not.toContain('formally audited');
  });

  it('registers unique branded cards for both indexable routes', () => {
    for (const route of ['/roadmap', '/research']) {
      const matches = staticOgPages.filter((entry) => entry.route === route);
      expect(matches, route).toHaveLength(1);
      expect(matches[0].title.length).toBeGreaterThan(20);
      expect(matches[0].description.length).toBeGreaterThan(70);
    }
  });
});
