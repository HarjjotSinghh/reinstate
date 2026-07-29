import { product } from './product';

export interface OgPage {
  route: string;
  kind: string;
  title: string;
  description: string;
}

const previewTitles = [
  ['/preview', 'Hero directions'],
  ['/preview/carrier', 'Carrier landing-page direction'],
  ['/preview/console', 'Console landing-page direction'],
  ['/preview/datasheet', 'Datasheet landing-page direction'],
  ['/preview/diptych', 'Diptych landing-page direction'],
  ['/preview/drench', 'Drench landing-page direction'],
  ['/preview/exploded', 'Exploded landing-page direction'],
  ['/preview/fonts', 'Heading font options'],
  ['/preview/manifest', 'Manifest landing-page direction'],
  ['/preview/nightshift', 'Night shift landing-page direction'],
  ['/preview/relay', 'Relay landing-page direction'],
  ['/preview/route', 'Route landing-page direction'],
  ['/preview/spread', 'Spread landing-page direction'],
  ['/preview/vault', 'Vault landing-page direction'],
] as const;

const previewOgPages: OgPage[] = previewTitles.map(([route, title]) => ({
  route,
  kind: 'Design preview',
  title,
  description:
    'A noindex design preview for the Reinstate cross-device coding-agent continuity website.',
}));

export const staticOgPages: OgPage[] = [
  {
    route: '/',
    kind: 'Product',
    title: product.defaultTitle,
    description: product.defaultDescription,
  },
  {
    route: '/404',
    kind: 'Navigation',
    title: 'Page not found',
    description:
      'Continue to Reinstate documentation, compatibility information, or the open-source repository.',
  },
  {
    route: '/docs',
    kind: 'Documentation',
    title: 'Reinstate documentation',
    description:
      'Install, configure, sync, restore, secure, and troubleshoot Reinstate across supported devices.',
  },
  {
    route: '/guides',
    kind: 'Guides',
    title: 'Coding-agent session sync guides',
    description:
      'Verified workflows for encrypted session transfer, project path mapping, safe restore, and native resume.',
  },
  {
    route: '/blog',
    kind: 'Engineering notes',
    title: 'Coding-agent continuity blog',
    description:
      'Evidence-backed engineering and product articles about secure coding-agent work continuity.',
  },
  {
    route: '/integrations',
    kind: 'Integrations',
    title: 'Coding-agent integrations',
    description:
      'Review Reinstate support for same-vendor Claude Code and Codex session sync.',
  },
  {
    route: '/integrations/claude-code',
    kind: 'Integration',
    title: 'Sync Claude Code sessions across devices',
    description:
      'Encrypt, transfer, restore, and natively resume a Claude Code session on another computer.',
  },
  {
    route: '/integrations/codex',
    kind: 'Integration',
    title: 'Sync Codex sessions across devices',
    description:
      'Move an encrypted Codex session between supported macOS and Windows environments.',
  },
  {
    route: '/compatibility',
    kind: 'Product facts',
    title: 'Reinstate compatibility',
    description:
      'Current coding agents, operating systems, versions, storage backends, and release limitations.',
  },
  {
    route: '/compatibility/agent-version-history',
    kind: 'Compatibility history',
    title: 'Claude Code and Codex version support history',
    description:
      'Track evidence-backed agent-version ranges and fail-closed compatibility changes across Reinstate release candidates.',
  },
  {
    route: '/security',
    kind: 'Security',
    title: 'How Reinstate protects session data',
    description:
      'Local age encryption, user-owned storage, credential exclusions, restore safety, and threat boundaries.',
  },
  {
    route: '/about/reinstate',
    kind: 'Product facts',
    title: 'What is Reinstate?',
    description:
      'Verified facts about the open-source continuity layer for coding-agent work.',
  },
  {
    route: '/open-source',
    kind: 'Open source',
    title: 'Open-source coding-agent session sync',
    description:
      'Inspect Reinstate source, security policy, roadmap, governance, releases, and Apache-2.0 license.',
  },
  {
    route: '/roadmap',
    kind: 'Product roadmap',
    title: 'From encrypted sync to work continuity',
    description:
      'Separate Reinstate RC8 capabilities and acceptance gates from planned search, handoff, and configuration phases.',
  },
  {
    route: '/research',
    kind: 'Primary-source evidence',
    title: 'Reinstate research and compatibility evidence',
    description:
      'Inspect adapter tests, compatibility data, acceptance methodology, synthetic fixtures, and benchmark evidence rules.',
  },
  {
    route: '/research/encrypted-snapshot-format-v1',
    kind: 'Implementation specification',
    title: 'Reinstate encrypted session snapshot format v1',
    description:
      'Inspect the current encrypted manifest, metadata envelope, TAR payload, validation, limits, and pre-1.0 evolution boundary.',
  },
  {
    route: '/glossary',
    kind: 'Terminology',
    title: 'Reinstate session sync and continuity glossary',
    description:
      'Define profiles, snapshots, encrypted manifests, canonical project IDs, native resume, handoffs, and compatibility states.',
  },
  {
    route: '/tools/path-mapping-visualizer',
    kind: 'Synthetic technical explainer',
    title: 'macOS and Windows path-mapping visualizer',
    description:
      'Trace fixed synthetic Claude Code and Codex structural paths through one portable canonical project mapping.',
  },
  {
    route: '/changelog',
    kind: 'Releases',
    title: 'Reinstate changelog',
    description:
      'Current pre-1.0 release status, verified changes, compatibility evidence, and full release history.',
  },
  {
    route: '/privacy',
    kind: 'Privacy',
    title: 'Website privacy notice',
    description:
      'How the Reinstate website handles waitlist email addresses, optional analytics, and operational logs.',
  },
  {
    route: '/use-cases',
    kind: 'Use cases',
    title: 'Cross-device coding-agent continuity',
    description:
      'Practical workflows for continuing coding-agent sessions across computers and operating systems.',
  },
  {
    route: '/use-cases/work-and-personal-computers',
    kind: 'Use case',
    title: 'Continue coding-agent work on a personal computer',
    description:
      'Move an encrypted same-vendor session between work and personal devices without syncing credentials.',
  },
  {
    route: '/use-cases/macos-and-windows',
    kind: 'Use case',
    title: 'Move coding-agent sessions between macOS and Windows',
    description:
      'Use canonical project mappings to restore Claude Code or Codex sessions when local paths differ.',
  },
  {
    route: '/use-cases/desktop-and-laptop',
    kind: 'Use case',
    title: 'Continue coding-agent sessions from desktop to laptop',
    description:
      'Move one encrypted same-vendor session between configured computers with path mapping and safe restore.',
  },
  {
    route: '/use-cases/encrypted-session-backup',
    kind: 'Use case',
    title: 'Create an encrypted backup of coding-agent sessions',
    description:
      'Store selected Claude Code or Codex snapshots as client-encrypted objects in your own bucket.',
  },
  {
    route: '/compare',
    kind: 'Factual comparisons',
    title: 'Compare Reinstate with session-continuity workflows',
    description:
      'Sourced comparisons with manual session copying, remote desktop, and Git.',
  },
  {
    route: '/compare/reinstate-vs-manual-session-copying',
    kind: 'Workflow comparison',
    title: 'Reinstate vs. manual session copying',
    description:
      'Compare discovery, path handling, encryption, credentials, and safe restore behavior.',
  },
  {
    route: '/compare/reinstate-vs-remote-desktop',
    kind: 'Workflow comparison',
    title: 'Reinstate vs. remote desktop',
    description:
      'Compare local session transfer with controlling the computer where work already lives.',
  },
  {
    route: '/compare/reinstate-vs-git',
    kind: 'Workflow comparison',
    title: 'Reinstate vs. Git',
    description:
      'Git tracks repository history; Reinstate transfers supported coding-agent session state.',
  },
  ...previewOgPages,
];
