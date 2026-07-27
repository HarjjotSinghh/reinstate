import { existsSync, readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const readSource = (path: string) =>
  existsSync(new URL(path, import.meta.url))
    ? readFileSync(new URL(path, import.meta.url), 'utf8')
    : '';

const index = readSource('../pages/index.astro');
const security = readSource('../components/landing/SecurityExploded.astro');
const vaultArt = readSource('../components/landing/art/SecurityVaultArt.astro');
const header = readSource('../components/Header.astro');
const footer = readSource('../components/Footer.astro');

describe('landing-page security section', () => {
  it('is wired after the terminal proof on the continuous floor', () => {
    expect(index).toContain(
      "import SecurityExploded from '../components/landing/SecurityExploded.astro';",
    );
    expect(index).toContain('<SecurityExploded />');
    expect(index.indexOf('<TerminalProof />')).toBeLessThan(
      index.indexOf('<SecurityExploded />'),
    );
  });

  it('leads with local encryption and ownership claims', () => {
    expect(security).toContain('id="security"');
    expect(security).toContain('Sessions leave your device encrypted.');
    expect(security).toContain('Your bucket never sees plaintext.');
    expect(security).toContain('class="h2-line"');
    expect(security).toContain('Local encryption before anything leaves your device');
    expect(security).toContain('Your encryption secret is never uploaded');
    expect(security).toContain('Your bucket, your controls');
    expect(security).toContain('No Reinstate account required');
    expect(security).toContain('Backed up before restore');
    expect(security).toContain('Lose it and remote snapshots cannot be recovered.');
    expect(security).toContain('class="proof-matrix"');
    expect(security).toContain('class="proof-icon"');
    expect(security).toContain('class="slink-icon"');
  });

  it('keeps the trust proofs compressed and chronologically numbered', () => {
    const numbers = [...security.matchAll(/k: '(\d{2})'/g)].map((match) => match[1]);
    expect(numbers).toEqual(['01', '02', '03', '04', '05']);
    expect(security).not.toContain('No auth tokens');
    expect(security).not.toContain('No Reinstate servers');
    expect(security).not.toContain('Open-source client');
  });

  it('links to model, encryption source, and private reporting', () => {
    expect(security).toContain('/docs/security-model');
    expect(security).toContain('internal/crypto');
    expect(security).toContain('security/advisories/new');
  });

  it('uses the vault trust-room art (not the TerminalProof pipeline)', () => {
    expect(security).toContain("from './art/SecurityVaultArt.astro'");
    expect(security).toContain('<SecurityVaultArt />');
    expect(vaultArt).toContain("from '../../../lib/iso'");
    expect(vaultArt).toContain('role="img"');
    expect(vaultArt).toContain('Capture locally');
    expect(vaultArt).toContain('Encrypt locally');
    expect(vaultArt).toContain('Store in your bucket');
    expect(vaultArt).toContain('Restore anywhere');
    expect(vaultArt).toContain('HTTPS');
    expect(vaultArt).toContain('Passphrase stays local');
    expect(vaultArt).toContain('Snapshot sealed with age');
    expect(vaultArt).toContain('S3-compatible object storage');
    expect(vaultArt).toContain('RESTORED');
    expect(vaultArt).toContain('makeOpen');
    expect(vaultArt).toContain('makeSealed');
    expect(vaultArt).toContain('#b7ff38');
    expect(vaultArt).toContain('marker-end');
    expect(vaultArt).toContain('vault-arrow');
    expect(vaultArt).toContain('--sv-stroke');
    expect(vaultArt).toContain('html.dark');
  });

  it('keeps nav and footer pointing at the section', () => {
    expect(header).toContain('href="/#security"');
    expect(footer).toContain('href="/#security"');
  });

  it('contains no em dash characters', () => {
    expect(security).not.toMatch(/[—–]/);
  });
});
