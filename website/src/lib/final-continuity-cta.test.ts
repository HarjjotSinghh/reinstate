import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const componentPath = fileURLToPath(
  new URL('../components/landing/FinalContinuityCTA.astro', import.meta.url),
);
const continuityArtPath = fileURLToPath(
  new URL('../components/landing/art/ContinuityFitArt.astro', import.meta.url),
);
const installArtPath = fileURLToPath(
  new URL('../components/landing/art/InstallEdgeArt.astro', import.meta.url),
);
const indexPath = fileURLToPath(new URL('../pages/index.astro', import.meta.url));

const component = readFileSync(componentPath, 'utf8');
const continuityArt = readFileSync(continuityArtPath, 'utf8');
const installArt = readFileSync(installArtPath, 'utf8');
const index = readFileSync(indexPath, 'utf8');

describe('landing-page continuity finale', () => {
  it('closes the landing page after the path identity proof', () => {
    expect(index).toContain(
      "import FinalContinuityCTA from '../components/landing/FinalContinuityCTA.astro'",
    );
    expect(index.indexOf('<WhyNotFileSync />')).toBeLessThan(
      index.indexOf('<FinalContinuityCTA />'),
    );
  });

  it('distinguishes cloud handoff, remote control, and Reinstate without overclaiming', () => {
    expect(component).toContain("import ContinuityFitArt from './art/ContinuityFitArt.astro'");
    expect(component).toContain('Cloud handoff');
    expect(component).toContain('Remote control');
    expect(component).toContain('DURABLE HANDOFF');
    expect(component).toContain('Sessions resume in the same agent');
    expect(component).toContain('Cross-agent work uses explicit handoffs');
  });

  it('uses the landing page axonometric art language in both parts of the finale', () => {
    expect(component).toContain("import InstallEdgeArt from './art/InstallEdgeArt.astro'");
    expect(component).toContain('<ContinuityFitArt />');
    expect(component).toContain('<InstallEdgeArt />');
    expect(component).toContain(
      'font-size: clamp(1.9rem, 3.4vw, 3rem)',
    );
    expect(continuityArt).toContain("import { box, fmt, P } from '../../../lib/iso'");
    expect(installArt).toContain("import { box, fmt, P } from '../../../lib/iso'");
    expect(continuityArt).not.toContain('<feDropShadow');
    expect(installArt).not.toContain('<feDropShadow');
  });

  it('keeps the two closing sections visually continuous and uses the official GitHub mark', () => {
    expect(component).not.toMatch(
      /\.install-section\s*\{[^}]*border-top:/s,
    );
    expect(component).toContain('viewBox="0 0 16 16"');
    expect(component).toContain("flex: 0 0 1.05rem");
  });

  it('uses the shared Reinstate mark in the installer and primary CTA', () => {
    expect(component).toContain("import LogoMark from '../LogoMark.astro'");
    expect(component).toContain('<LogoMark size={22} tiled={false} />');
    expect(component).toContain('<LogoMark size={19} tiled={false} />');
    expect(component).toContain('--logo-mark-surface: var(--sv-night)');
    expect(component).toContain('--logo-mark-surface: var(--sv-prim)');
  });

  it('offers both published install commands in accessible tabs', () => {
    expect(component).toContain('role="tablist"');
    expect(component).toContain('role="tabpanel"');
    expect(component).toContain('curl -fsSL https://reinstate.dev/install.sh | sh');
    expect(component).toContain('irm https://reinstate.dev/install.ps1 | iex');
    expect(component).toContain("event.key !== 'ArrowLeft'");
  });

  it('tracks final-CTA conversions and preserves honest trust signals', () => {
    expect(component).toContain("target: 'homepage-final-cta'");
    expect(component).toContain('data-analytics-target="homepage-final-cta"');
    expect(component).toContain('No Reinstate account');
    expect(component).toContain('Your bucket');
    expect(component).toContain('Your keys');
  });
});
