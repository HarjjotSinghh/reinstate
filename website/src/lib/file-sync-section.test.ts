import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const sectionPath = fileURLToPath(
  new URL('../components/landing/WhyNotFileSync.astro', import.meta.url),
);
const artPath = fileURLToPath(
  new URL('../components/landing/art/PathMismatchEdgeArt.astro', import.meta.url),
);
const indexPath = fileURLToPath(new URL('../pages/index.astro', import.meta.url));

const section = readFileSync(sectionPath, 'utf8');
const art = readFileSync(artPath, 'utf8');
const index = readFileSync(indexPath, 'utf8');

describe('landing-page file-sync objection section', () => {
  it('appears after security as the final substantive landing section', () => {
    expect(index).toContain("import WhyNotFileSync from '../components/landing/WhyNotFileSync.astro'");
    expect(index.indexOf('<SecurityExploded />')).toBeLessThan(
      index.indexOf('<WhyNotFileSync />'),
    );
  });

  it('shows the concrete cross-OS native identity failure', () => {
    expect(section).toContain('Files can sync.');
    expect(section).toContain('Resume can still fail.');
    expect(section).toContain('Same repo, different path, different project key.');
    expect(section).toContain('<code>--resume</code> may find nothing.');
    expect(section).toContain(String.raw`C:\Users\harjot\projects\api`);
    expect(section).toContain('/Users/harjot/dev/api');
    expect(section).toContain('C--Users-harjot-projects-api');
    expect(section).toContain('-Users-harjot-dev-api');
    expect(section).toContain('same repo · api');
    expect(section).toContain('Claude index');
    expect(section).toContain('keys differ');
    expect(section).toContain('<code>--resume</code>: no matching project');
  });

  it('uses a translucent comparison surface without making its content translucent', () => {
    expect(section).toContain(
      'background: color-mix(in oklab, var(--paper-2) 70%, transparent)',
    );
    expect(section).toContain('backdrop-filter: blur(16px)');
    expect(section).not.toContain('opacity: 0.7');
  });

  it('states the adapter and continuity boundaries precisely', () => {
    expect(section).toContain('Reinstate rebuilds identity');
    expect(section).toContain(
      'reconstructs the project reference the agent expects',
    );
    expect(section).toContain('Resume stays in your agent');
    expect(section).toContain('Codex rollouts resume in Codex');
    expect(section).not.toContain('Reinstate owns identity');
    expect(section).not.toContain('Native resume stays native');
  });

  it('uses the product mark and semantic transformation icons in the mapping strip', () => {
    expect(section).toContain("import LogoMark from '../LogoMark.astro'");
    expect(section).toContain('<LogoMark size={27} tiled={false} />');
    expect(section).toContain('map-transition--route');
    expect(section).toContain('map-transition--identity');
    expect(section).not.toContain('class="map-arrow"');
  });

  it('ends with honest current compatibility and documentation links', () => {
    for (const label of [
      'Claude Code',
      'Codex',
      'macOS ↔ Windows',
      'S3',
      'Cloudflare R2',
    ]) {
      expect(section).toContain(label);
    }
    expect(section).toContain('href="/docs/adapters"');
    expect(section).toContain('href="/docs/comparison"');
  });

  it('keeps detailed edge art accessible and peripheral', () => {
    expect(art).toContain('viewBox="0 0 1600 500"');
    expect(art).toContain('The same API repository on Windows and macOS');
    expect(art).toContain('enters from the left');
    expect(art).toContain('macOS laptop enters');
    expect(art).toContain('class="cable-plug"');
    expect(art).toContain('class="cable-port"');
    expect(art).not.toContain('class="path-plate"');
    expect(art).not.toContain('dotGrid');
    expect(art).toContain('position: absolute');
    expect(art).toContain('@media (max-width: 760px)');
  });

  it('uses the published Windows and Apple platform marks', () => {
    expect(section).toContain('Windows 11 mark geometry');
    expect(section).toContain('viewBox="0 0 48.746 48.746"');
    expect(section).toContain('Apple mark geometry');
    expect(section).toContain('viewBox="0 0 814 1000"');
  });
});
