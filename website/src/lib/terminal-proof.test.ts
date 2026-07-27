import { existsSync, readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const readSource = (path: string) =>
  existsSync(new URL(path, import.meta.url))
    ? readFileSync(new URL(path, import.meta.url), 'utf8')
    : '';

const index = readSource('../pages/index.astro');
const problem = readSource('../components/landing/ProblemExploded.astro');
const terminal = readSource('../components/landing/TerminalProof.astro');
const commands = readSource('../../../internal/cli/commands_impl.go');
const passphrase = readSource('../../../internal/crypto/passphrase.go');

describe('landing-page terminal proof', () => {
  it('renders as the next section inside the continuous illustrated floor', () => {
    expect(index).toContain(
      "import TerminalProof from '../components/landing/TerminalProof.astro';",
    );
    expect(index).toContain('<ProblemExploded>');
    expect(index).toContain('<TerminalProof />');
    expect(problem).toContain('<slot />');
    expect(terminal).toContain('class="terminal-proof"');
    expect(terminal).toContain('class="handoff-art"');
    expect(terminal).toContain('class="sealed-checkpoint"');
  });

  it('puts terminals first and keeps a compact transfer illustration secondary', () => {
    expect(terminal).toContain('<figure class="handoff-illustration"');
    expect(terminal.indexOf('class="workflow-grid"')).toBeLessThan(
      terminal.indexOf('class="handoff-illustration"'),
    );

    const artStyles = terminal.match(/\.handoff-art\s*\{([^}]*)\}/)?.[1] ?? '';
    expect(artStyles).not.toMatch(/position:\s*absolute/);
    const figureStyles = terminal.match(/\.handoff-illustration\s*\{([^}]*)\}/)?.[1] ?? '';
    expect(figureStyles).toMatch(/max-width/);
  });

  it('shows the released four-command Claude Code handoff', () => {
    for (const command of [
      'rein list --agent claude',
      'rein push --agent claude --session ses_7f3a',
      'rein pull --agent claude --session ses_7f3a',
      'claude --resume ses_7f3a',
    ]) {
      expect(terminal).toContain(command);
    }

    expect(terminal).toContain('pushed 1 snapshot, skipped 0 unchanged');
    expect(terminal).toContain('pulled 1 snapshot');
    expect(terminal).not.toContain('dry_run=false');
    expect(terminal).not.toContain('snapshot(s)');
    expect(terminal).toContain('claude  ses_7f3a  reinstate  2.1 MB  4h ago');
    expect(commands).toContain(
      'pushed %d snapshot(s), skipped %d unchanged, dry_run=%v',
    );
    expect(commands).toContain('pulled %d snapshot(s) dry_run=%v');
  });

  it('keeps secret entry hidden and storage user-owned', () => {
    expect(terminal).toContain('Encryption passphrase:');
    expect(terminal).toContain('••••••••');
    expect(terminal).toContain('age encrypted');
    expect(terminal).toContain('your S3 or R2 bucket');
    expect(terminal).toContain('End-to-end encrypted');
    expect(passphrase).toContain(
      'ReadHiddenSecret(input, promptOut, "Encryption passphrase: ")',
    );
  });

  it('keeps terminal evidence selectable and accessible', () => {
    expect(terminal).toContain('<pre');
    expect(terminal).toContain('<code>');
    expect(terminal).toContain('role="img"');
    expect(terminal).toContain('<title');
    expect(terminal).toContain('<desc');
    expect(terminal).toContain('@media (prefers-reduced-motion: reduce)');
  });

  it('names the Codex equivalent without claiming translation or future scope', () => {
    expect(terminal).toContain('codex resume ses_7f3a');

    for (const unsupported of [
      'rein resume',
      'cross-agent translation',
      'MCP sync',
      'skills sync',
      'Gemini',
      'OpenCode',
      'match by git remote',
    ]) {
      expect(terminal).not.toContain(unsupported);
    }
  });

  it('contains no em dash characters', () => {
    expect(terminal).not.toMatch(/[—–]/);
  });
});
