import { readdir, readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import compatibility from '../data/compatibility.json';
import { agentVersionHistory } from '../data/agent-version-history';
import { staticOgPages } from '../data/og-pages';
import { releaseHistory } from '../data/releases';
import staticPageReviews from '../data/static-page-reviews.json';
import {
  compatibilityAgents,
  findDocRowName,
  overClaims,
  parseCatalogDescriptors,
  parseCompatibilityDocTiers,
  repositoryCatalogDir,
  repositoryCompatibilityDoc,
  tierRank,
} from './agent-catalog';

const repositoryRoot = new URL('../../../', import.meta.url);
const page = (path: string) =>
  readFile(new URL(`../pages/${path}`, import.meta.url), 'utf8');
const repositoryFile = (path: string) =>
  readFile(new URL(path, repositoryRoot), 'utf8');

const routes = [
  '/glossary',
  '/tools/path-mapping-visualizer',
  '/research/encrypted-snapshot-format-v1',
  '/compatibility/agent-version-history',
] as const;

describe('evidence-safe linkable assets', () => {
  it('defines every required term while separating current and planned behavior', async () => {
    const glossary = await page('glossary.astro');

    for (const term of [
      'Profile',
      'Snapshot',
      'Encrypted manifest',
      'Canonical project ID',
      'Structural path',
      'Native resume',
      'Portable handoff',
      'SUPPORTED',
      'UNTESTED',
      'UNSUPPORTED',
      'NOT_INSTALLED',
    ]) {
      expect(glossary, term).toContain(term);
    }

    expect(glossary).toContain('Portable handoffs are available in Reinstate as structured');
    expect(glossary).toContain('same-vendor');
    expect(glossary).not.toContain('cross-agent resume');
  });

  it('keeps the path visualizer fixed, synthetic, non-persistent, and analytics-free', async () => {
    const visualizer = await page('tools/path-mapping-visualizer.astro');

    expect(visualizer).toContain('analytics={false}');
    expect(visualizer).toContain('Synthetic-only control');
    expect(visualizer).toContain('${REPO:github.com/acme/acme-app}');
    expect(visualizer).toContain('session_meta.payload.cwd');
    expect(visualizer).toContain('Unknown keys remain untouched');
    expect(visualizer).toContain(
      "text('#unchanged-prose', selectedDirection.unchangedProse)",
    );
    expect(visualizer).toContain("?? examples['mac-to-windows']");
    expect(visualizer).not.toContain(
      "document.querySelector<HTMLElement>('.portable dd:last-child code')",
    );
    expect(visualizer).not.toMatch(/<input\b/i);
    expect(visualizer).not.toMatch(
      /\b(?:fetch|XMLHttpRequest|WebSocket|sendBeacon|localStorage|sessionStorage)\s*(?:\(|\.|=)/,
    );
    expect(visualizer).not.toContain('data-analytics-event');
  });

  it('documents snapshot v1 from the released schema without claiming a standard', async () => {
    const [specification, envelope, manifest, push] = await Promise.all([
      page('research/encrypted-snapshot-format-v1.astro'),
      repositoryFile('internal/schema/envelope.go'),
      repositoryFile('internal/schema/manifest.go'),
      repositoryFile('internal/sync/push.go'),
    ]);

    expect(envelope).toContain('EnvelopeSchemaVersion = 1');
    expect(envelope).toContain('EnvelopeKind = "reinstate-session-snapshot"');
    expect(manifest).toContain('ManifestSchemaVersion = 1');
    expect(push).toContain('maxManifestBytes   = 4 << 20');
    expect(push).toContain('maxMetadataBytes   = 1 << 20');
    expect(push).toContain('defaultMaxPayload  = int64(32 << 30)');

    for (const contract of [
      'manifest.age',
      'reinstate-session-snapshot',
      'Exactly one',
      'SHA-256',
      '32 GiB',
      '16 MiB',
      'Internal specification, not portability standard',
    ]) {
      expect(specification, contract).toContain(contract);
    }
    expect(specification).toContain('not an open portability standard');
    expect(specification).not.toContain('industry standard');
    expect(specification).not.toContain('vendor-approved');
  });

  it('unifies current compatibility and tagged release evidence', async () => {
    const [tracker, changelog, claudeAdapter, compatibilityDoc] = await Promise.all([
      page('compatibility/agent-version-history.astro'),
      repositoryFile('CHANGELOG.md'),
      repositoryFile('internal/adapter/claude/claude.go'),
      repositoryFile('docs/compatibility.md'),
    ]);

    expect(agentVersionHistory.map(({ version }) => version)).toEqual(
      releaseHistory.map(({ version }) => version),
    );
    expect(agentVersionHistory).toHaveLength(37);
    expect(new Set(agentVersionHistory.map(({ source }) => source)).size).toBe(37);
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.11');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.11')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.10');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.10')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.9');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.7')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.6');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.6')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.5');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.5')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.4');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.4')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.3');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.3')
        ?.rangeChange,
    ).toContain('Unchanged from v0.4.0-rc.2');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.2')
        ?.rangeChange,
    ).toContain('2.1.219–2.1.228 to 2.1.219–2.1.229');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.4.0-rc.1')
        ?.rangeChange,
    ).toContain('2.1.219–2.1.227 to 2.1.219–2.1.228');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.3.0')
        ?.rangeChange,
    ).toContain('2.1.219–2.1.227');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.3.0-rc.6')
        ?.rangeChange,
    ).toBe(
      'Expanded the inclusive Claude Code range from 2.1.219–2.1.220 to 2.1.219–2.1.227 and the Codex CLI range from 0.133.0–0.146.0 to 0.133.0–0.147.0.',
    );
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.3.0-rc.4')
        ?.rangeChange,
    ).toBe('No agent-version range change documented.');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.3.0-rc.3')
        ?.rangeChange,
    ).toBe('No agent-version range change documented.');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.2.0-rc.3')
        ?.rangeChange,
    ).toBe('No agent-version range change documented.');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.2.0-rc.1')
        ?.rangeChange,
    ).toContain('Codex CLI range from 0.133.0–0.145.0 to 0.133.0–0.146.0');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.1.0-rc.3')
        ?.rangeChange,
    ).toContain('Claude Code 2.1.219–2.1.220');
    // The published range can move ahead of the newest tag when a candidate
    // widens it before acceptance, so it is pinned to the shipping adapter
    // constants and the compatibility doc rather than to a released entry.
    expect(claudeAdapter).toContain(
      `minimumVerifiedClaudeVersion = "${compatibility.agents[0].minimumTestedVersion}"`,
    );
    expect(claudeAdapter).toContain(
      `maximumVerifiedClaudeVersion = "${compatibility.agents[0].maximumTestedVersion}"`,
    );
    expect(compatibilityDoc).toContain(
      `| Claude Code | \`${compatibility.agents[0].minimumTestedVersion}\`–\`${compatibility.agents[0].maximumTestedVersion}\` |`,
    );
    expect(compatibilityDoc).toContain(
      `| OpenAI Codex CLI | \`${compatibility.agents[1].minimumTestedVersion}\`–\`${compatibility.agents[1].maximumTestedVersion}\` |`,
    );
    expect(changelog).toContain('Claude Code compatibility range through `2.1.227`');
    expect(changelog).toContain(
      `Claude Code compatibility range through \`${compatibility.agents[0].maximumTestedVersion}\``,
    );
    expect(tracker).toContain('compatibility.agents.map');
    expect(tracker).toContain('No change documented');
    expect(tracker).toContain('source-level gate');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.1.0-rc.5')
        ?.compatibilityChange,
    ).toContain('metadata-only');
    expect(
      agentVersionHistory.find(({ version }) => version === 'v0.1.0-rc.6')
        ?.compatibilityChange,
    ).toContain('does not decrypt');
    // rc.7 evidence points at processcheck, not commands_impl.
    for (const version of ['v0.1.0-rc.5', 'v0.1.0-rc.6']) {
      expect(
        agentVersionHistory.find((entry) => entry.version === version)
          ?.implementationSource,
      ).toBe(
        `https://github.com/HarjjotSinghh/reinstate/blob/${version}/internal/cli/commands_impl.go`,
      );
    }
  });

  it('registers freshness ownership and one unique social card for every route', () => {
    for (const route of routes) {
      expect(
        staticPageReviews.filter((entry) => entry.route === route),
        `${route} freshness`,
      ).toHaveLength(1);
      expect(
        staticOgPages.filter((entry) => entry.route === route),
        `${route} social card`,
      ).toHaveLength(1);
    }
  });

  it('provides a complete content brief for every new indexable page', async () => {
    const briefNames = [
      'reinstate-terminology-glossary.md',
      'synthetic-path-mapping-visualizer.md',
      'encrypted-snapshot-format-v1.md',
      'supported-agent-version-history.md',
    ];
    const briefs = await Promise.all(
      briefNames.map((name) =>
        repositoryFile(`docs/seo/content-briefs/${name}`),
      ),
    );

    for (const [index, brief] of briefs.entries()) {
      for (const heading of [
        '## Page',
        '## Audience and intent',
        '## Product truth',
        '## Outline',
        '## Links',
        '## Structured data',
        '## Media',
        '## Acceptance criteria',
      ]) {
        expect(brief, `${briefNames[index]} ${heading}`).toContain(heading);
      }
      expect(brief).toMatch(/`v0\.(?:1|2)\.0`/);
    }
  });

  it('keeps the visualizer analytics opt-out explicit in both shared layouts', async () => {
    const [baseLayout, publicLayout] = await Promise.all([
      readFile(new URL('../layouts/BaseLayout.astro', import.meta.url), 'utf8'),
      readFile(
        new URL('../layouts/PublicContentLayout.astro', import.meta.url),
        'utf8',
      ),
    ]);

    expect(baseLayout).toContain('analytics = true');
    expect(baseLayout).toContain('{analytics && <Analytics />}');
    expect(publicLayout).toContain('analytics={analytics}');
  });

  it('adds evidence gates for every requested distribution and research plan', async () => {
    const launch = await repositoryFile('docs/seo/launch-distribution.md');

    for (const section of [
      '### Demo video plan',
      '### Awesome-list outreach plan',
      '### GitHub Discussion plan',
      '### Architecture post plan',
      '### Archive inspector',
      '### Migration readiness checker',
      '### Path mapper',
      '### Open portability standard',
      '### Proprietary coding-agent format research',
    ]) {
      expect(launch, section).toContain(section);
    }
    expect(launch).toContain('not a standard and not a Reinstate feature');
    expect(launch).toMatch(
      /never real user, employer,\s+customer, or contributor transcripts/,
    );
  });

  it('agrees with the catalog and docs/compatibility.md on every agent tier', () => {
    const catalog = parseCatalogDescriptors(repositoryCatalogDir());
    const claimed = parseCompatibilityDocTiers(repositoryCompatibilityDoc());

    expect(catalog.map(({ key }) => key).sort()).toEqual(
      compatibilityAgents.map(({ key }) => key).sort(),
    );
    expect(catalog).toHaveLength(18);

    for (const desc of catalog) {
      const row = compatibilityAgents.find((agent) => agent.key === desc.key);
      expect(row, desc.key).toBeDefined();
      expect(row?.tier, desc.key).toBe(desc.tier);
      expect(row?.storageFamily, desc.key).toBe(desc.family);
      expect(row?.t0Reason, desc.key).toBe(desc.t0Reason);

      const docName = findDocRowName(claimed, desc);
      expect(docName, `${desc.key} docs row`).toBeTruthy();
      expect(claimed.get(docName ?? ''), desc.key).toBe(desc.tier);
    }

    for (const [name, tier] of claimed) {
      expect(
        catalog.some((desc) => findDocRowName(new Map([[name, tier]]), desc)),
        `docs extra row ${name}`,
      ).toBe(true);
    }
  });

  it('does not publish an integration page for T0 agents or omit one for T1+', async () => {
    const integrationFiles = new Set(
      (await readdir(new URL('../pages/integrations/', import.meta.url))).filter(
        (name) => name.endsWith('.astro') && name !== 'index.astro',
      ),
    );

    for (const agent of compatibilityAgents) {
      const slug = agent.integrationPath?.replace('/integrations/', '');
      if (tierRank(agent.tier) < 1) {
        expect(agent.integrationPath, agent.key).toBeNull();
        expect(integrationFiles.has(`${agent.id}.astro`), agent.id).toBe(false);
        if (slug) {
          expect(integrationFiles.has(`${slug}.astro`), slug).toBe(false);
        }
        continue;
      }
      expect(agent.integrationPath, agent.key).toMatch(/^\/integrations\/[a-z0-9-]+$/);
      expect(integrationFiles.has(`${slug}.astro`), slug).toBe(true);
    }
  });

  it('does not claim a capability above an agent tier on public surfaces', async () => {
    const roots = [
      new URL('../pages/compatibility.astro', import.meta.url),
      new URL('../pages/compatibility/', import.meta.url),
      new URL('../pages/integrations/', import.meta.url),
      new URL('../pages/roadmap.astro', import.meta.url),
    ];
    const files: URL[] = [];
    for (const root of roots) {
      if (root.pathname.endsWith('/')) {
        for (const name of await readdir(root)) {
          if (name.endsWith('.astro')) {
            files.push(new URL(name, root));
          }
        }
      } else {
        files.push(root);
      }
    }

    const hits: string[] = [];
    for (const path of files) {
      const source = await readFile(path, 'utf8');
      for (const agent of compatibilityAgents) {
        for (const hit of overClaims(source, agent)) {
          hits.push(`${path.pathname.split('/src/').at(-1)} ${agent.key}: ${hit}`);
        }
      }
    }
    expect(hits).toEqual([]);
  });
});
