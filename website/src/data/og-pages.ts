import { product } from './product';

export interface OgPage {
  route: string;
  kind: string;
  title: string;
  description: string;
}

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
];
