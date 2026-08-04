import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

function read(path: string) {
  return readFileSync(new URL(path, import.meta.url), 'utf8');
}

function squashWhitespace(source: string) {
  return source.replace(/\s+/g, ' ');
}

const guide = read('../content/guides/move-a-coding-agent-session-from-mac-to-windows.md');
const desktop = read('../pages/use-cases/desktop-and-laptop.astro');
const backup = read('../pages/use-cases/encrypted-session-backup.astro');
const ogPages = read('../data/og-pages.ts');

describe('Mac, Windows, and backup outcome content', () => {
  it('publishes a complete visible Mac-to-Windows HowTo contract', () => {
    const text = squashWhitespace(guide);

    for (const section of [
      '## Key points',
      '## Command placeholders and parameters',
      '## Failure modes and common errors',
      '## Safe rollback and undo',
      '## Verification checklist',
      '## Current limitations',
      '## Mac-to-Windows session FAQ',
    ]) {
      expect(guide, section).toContain(section);
    }

    for (const command of [
      'rein init',
      'rein setup check',
      'rein doctor --self-test',
      'rein push --agent AGENT --session SESSION_ID --dry-run',
      'rein pull --agent AGENT --session SESSION_ID --dry-run',
      'claude --resume SESSION_ID',
      'codex resume SESSION_ID',
    ]) {
      expect(text, command).toContain(command);
    }

    expect(guide.match(/\*\*Expected result:\*\*/g)).toHaveLength(5);
    const anchors = [...guide.matchAll(/^\s+anchor: "([^"]+)"$/gm)].map(
      (match) => match[1],
    );
    expect(anchors).toHaveLength(5);
    expect(new Set(anchors).size).toBe(5);
    for (const anchor of anchors) {
      expect(guide).toContain(`<h2 id="${anchor}">`);
    }

    expect(guide).not.toContain('rein resume');
    expect(text).toContain('same-vendor');
    expect(text).toContain('physical two-device acceptance');
    expect(text).toContain('does not provide a general `rein undo`');
  });

  it('qualifies native Windows, WSL, path remapping, and preview boundaries', () => {
    const combined = squashWhitespace(`${guide}\n${desktop}`);

    for (const fact of [
      'native Windows',
      'WSL2 is a separate Linux Reinstate device',
      'WSL1 is unsupported',
      'one canonical project ID',
      '${REPO:PROJECT_ID}',
      'known structural fields',
      'same-vendor',
      'physical acceptance checklist',
      'preview and unverified',
    ]) {
      expect(combined, fact).toContain(fact);
    }

    for (const unsupportedClaim of [
      'Phase 1 is complete',
      'production-ready',
      'fully supported on every platform',
      'seamless cross-agent',
    ]) {
      expect(combined).not.toContain(unsupportedClaim);
    }
  });

  it.each([
    {
      source: desktop,
      route: '/use-cases/desktop-and-laptop',
      evidence: 'id="evidence"',
      relatedRoute: '/use-cases/encrypted-session-backup',
    },
    {
      source: backup,
      route: '/use-cases/encrypted-session-backup',
      evidence: 'id="verification"',
      relatedRoute: '/use-cases/desktop-and-laptop',
    },
  ])(
    'keeps $route substantive, answer-first, recoverable, and internally connected',
    ({ source, route, evidence, relatedRoute }) => {
      expect(source).toContain('<PublicContentLayout');
      expect(source).toContain(`path="${route}"`);
      expect(source).toContain('answer="');
      expect(source).toContain(evidence);
      expect(source).toContain('id="failure-recovery"');
      expect(source).toContain('id="faq"');
      expect(source).toContain('class="pc-related"');
      expect(source).toContain(relatedRoute);
      expect(source).toContain('independent backup');
      expect(source).toContain('same-vendor');
    },
  );

  it('documents encryption, credential exclusions, restore safety, and honest backup scope', () => {
    const text = squashWhitespace(backup);

    for (const fact of [
      'encrypts before upload',
      'age scrypt',
      'passphrase is not stored',
      'provider receives ciphertext',
      'timestamped local backup',
      'conflict exit',
      'active same-vendor process',
      'not a complete backup system',
      'Git repository',
      'vendor authentication',
      'restore drills',
    ]) {
      expect(text, fact).toContain(fact);
    }
  });

  it('registers unique branded OG cards for both static pages and relies on dynamic guide OG', () => {
    for (const route of [
      '/use-cases/desktop-and-laptop',
      '/use-cases/encrypted-session-backup',
    ]) {
      expect(ogPages.match(new RegExp(`route: '${route}'`, 'g'))).toHaveLength(1);
    }

    expect(guide).toContain(
      'title: "Move a Coding-Agent Session from Mac to Windows"',
    );
    expect(guide).toContain('draft: false');
  });

  it('uses placeholders without publishing credentials or transcript data', () => {
    const combined = `${guide}\n${desktop}\n${backup}`;

    expect(combined).not.toMatch(/\b(?:AKIA|ASIA)[A-Z0-9]{16}\b/);
    expect(combined).not.toMatch(/REINSTATE_S3_SECRET_ACCESS_KEY\s*=/);
    expect(combined).not.toMatch(/secretAccessKey\s*:/i);
    expect(combined).not.toMatch(
      /(?:secret access key|passphrase)\s*(?:is|=|:)\s*["']?[A-Za-z0-9+/]{16,}/i,
    );
  });
});
