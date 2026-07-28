import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const readSource = (path: string) =>
  readFileSync(new URL(path, import.meta.url), 'utf8');

const header = readSource('../components/Header.astro');
const hero = readSource('../components/landing/HeroExploded.astro');
const problem = readSource('../components/landing/ProblemExploded.astro');

describe('landing-page product claims', () => {
  it('leads with the installable Phase 1 product', () => {
    expect(hero).toContain('Capture. Encrypt. Reinstate.');
    expect(hero).toContain('Start on one device.');
    expect(hero).toContain('Continue on another. Same session.');
    expect(hero).toContain('max-width: min(31ch, 100%)');
    expect(hero).toContain(
      'Move encrypted Claude Code or Codex sessions between macOS and Windows.',
    );
    expect(hero).toContain(
      'Reinstate encrypts the session locally, then stores it in your own S3 or R2 bucket.',
    );
    expect(hero).toContain(
      'curl -fsSL https://reinstate.dev/install.sh | sh',
    );
    expect(hero).toContain('Available now:');
    expect(hero).toContain('Claude Code');
    expect(hero).toContain('Codex');
    expect(header).toContain('<span class="navcta-label">Install</span>');
  });

  it('uses one command-first install hierarchy', () => {
    expect(hero).toContain('aria-label="Copy install command"');
    expect(hero).toContain('Read setup guide');
    expect(hero).toContain('View on GitHub');
    expect(hero).toContain('class="github-mark"');
    expect(hero).toContain('official mark-github Octicon');
    expect(hero).toContain('class="micro-separator"');
    expect(hero.match(/class="micro-separator"/g)).toHaveLength(4);
    expect(hero).toContain(
      'Apache&#8209;2.0<span class="micro-separator" aria-hidden="true">·</span>one Go binary',
    );
    expect(hero).toContain('class="conversion-meta"');
    expect(hero).toContain('max-width: 50rem');
    expect(hero).not.toContain('class="cta primary"');
    expect(hero).not.toContain('class="copy-label"');
  });

  it('protects the process rail and floats the navigation shell', () => {
    expect(hero).toContain('class="step-icon"');
    expect(hero).toContain('class="mechanism"');
    expect(header).toContain('class="nav-frame"');
    expect(header).toContain("const onHome = path === '/'");
    expect(header).toContain("'is-home'");
    expect(header).toContain("'is-scrolled'");
    expect(header).toContain('window.scrollY > 12');
    expect(header).toContain('max-width: 58rem');
  });

  it('does not put the released CLI behind a waitlist', () => {
    expect(hero).not.toContain('WaitlistForm');
    expect(header).not.toContain('Join the waitlist');
    expect(hero).not.toContain('Join the waitlist');
  });

  it('does not present later roadmap work as available', () => {
    const landing = `${hero}\n${problem}`;

    for (const unsupported of [
      'Gemini CLI',
      'OpenCode',
      'every coding agent',
      'MCP servers, skills, and settings',
      "key: 'mcp'",
      "key: 'skills'",
      "key: 'settings'",
      'rein resume',
      'rein sessions',
    ]) {
      expect(landing).not.toContain(unsupported);
    }
  });

  it('preserves the core Git problem statement', () => {
    expect(problem).toContain('Git has the code and its history.');
    expect(problem).toContain('It does not have the conversation.');
  });
});
