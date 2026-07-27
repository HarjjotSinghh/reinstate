export const analyticsEvents = [
  'install_command_copy',
  'github_click',
  'docs_getting_started',
  'integration_view',
  'storage_guide_view',
  'waitlist_submit',
  'download_click',
  'changelog_subscribe',
  'issue_report_click',
  'contribute_click',
  'security_doc_view',
  'command_copy',
] as const;

export type AnalyticsEvent = (typeof analyticsEvents)[number];

export const commandCopyLanguages = [
  'bash',
  'console',
  'powershell',
  'ps1',
  'sh',
  'shell',
  'zsh',
] as const;

export const analyticsEventTargets: Readonly<
  Record<AnalyticsEvent, readonly string[]>
> = {
  install_command_copy: ['homepage-hero'],
  github_click: ['header', 'mobile-header', 'footer', 'homepage-hero', 'homepage-explore'],
  docs_getting_started: ['header', 'footer', 'homepage-hero'],
  integration_view: ['claude-code', 'codex'],
  storage_guide_view: ['aws-s3', 'cloudflare-r2'],
  waitlist_submit: ['waitlist'],
  download_click: ['github-release-asset'],
  changelog_subscribe: ['rss-feed'],
  issue_report_click: ['github-issue-form'],
  contribute_click: ['github-contributing'],
  security_doc_view: ['security-overview', 'security-model'],
  command_copy: commandCopyLanguages,
} as const;

export interface AnalyticsEventMatch {
  event: AnalyticsEvent;
  target: string;
}

export interface AnalyticsLinkRule extends AnalyticsEventMatch {
  includes: string;
}

export type AiReferralChannel =
  | 'chatgpt'
  | 'perplexity'
  | 'microsoft-copilot'
  | 'google-gemini'
  | 'google-ai-features';

export interface AiReferralRule {
  channel: AiReferralChannel;
  hostnames: readonly string[];
  utmTokens: readonly string[];
}

export const aiReferralRules: readonly AiReferralRule[] = [
  {
    channel: 'chatgpt',
    hostnames: ['chatgpt.com', 'chat.openai.com', 'openai.com'],
    utmTokens: ['chatgpt', 'openai', 'oai-search'],
  },
  {
    channel: 'perplexity',
    hostnames: ['perplexity.ai'],
    utmTokens: ['perplexity'],
  },
  {
    channel: 'microsoft-copilot',
    hostnames: ['copilot.microsoft.com'],
    utmTokens: ['microsoft-copilot', 'ms-copilot', 'copilot'],
  },
  {
    channel: 'google-gemini',
    hostnames: ['gemini.google.com', 'bard.google.com'],
    utmTokens: ['google-gemini', 'gemini'],
  },
  {
    channel: 'google-ai-features',
    hostnames: [],
    utmTokens: ['google-ai', 'ai-overview', 'ai-overviews', 'ai-mode'],
  },
] as const;

export const analyticsLinkRules: readonly AnalyticsLinkRule[] = [
  {
    includes: '/releases/download/',
    event: 'download_click',
    target: 'github-release-asset',
  },
  {
    includes: '/issues/new',
    event: 'issue_report_click',
    target: 'github-issue-form',
  },
  {
    includes: '/CONTRIBUTING.md',
    event: 'contribute_click',
    target: 'github-contributing',
  },
] as const;

const storageGuidePaths = new Set([
  '/guides/use-s3-for-coding-agent-session-storage',
  '/guides/use-cloudflare-r2-for-coding-agent-session-storage',
]);

function normalizedPath(pathname: string): string {
  if (!pathname.startsWith('/')) {
    return pathname;
  }
  return pathname === '/' ? pathname : pathname.replace(/\/+$/, '');
}

function hostnameMatches(hostname: string, expected: string): boolean {
  return hostname === expected || hostname.endsWith(`.${expected}`);
}

function normalizedTrackingTokens(url: URL): string[] {
  return ['utm_source', 'utm_medium', 'utm_campaign']
    .map((key) => url.searchParams.get(key)?.trim().toLowerCase())
    .filter((value): value is string => Boolean(value));
}

export function analyticsAiReferralChannel({
  referrer,
  currentUrl,
}: {
  referrer?: string;
  currentUrl: string;
}): AiReferralChannel | null {
  let destination: URL;
  try {
    destination = new URL(currentUrl, 'https://reinstate.dev');
  } catch {
    return null;
  }

  const trackingTokens = normalizedTrackingTokens(destination);
  for (const rule of aiReferralRules) {
    if (
      rule.utmTokens.some((expected) =>
        trackingTokens.some((value) => value === expected),
      )
    ) {
      return rule.channel;
    }
  }

  if (!referrer) return null;
  let source: URL;
  try {
    source = new URL(referrer);
  } catch {
    return null;
  }
  const hostname = source.hostname.toLowerCase().replace(/\.$/, '');
  const matched = aiReferralRules.find((rule) =>
    rule.hostnames.some((expected) => hostnameMatches(hostname, expected)),
  );
  return matched?.channel ?? null;
}

export function analyticsPageEvent(pathname: string): AnalyticsEventMatch | null {
  const path = normalizedPath(pathname);

  if (path === '/integrations/claude-code') {
    return { event: 'integration_view', target: 'claude-code' };
  }
  if (path === '/integrations/codex') {
    return { event: 'integration_view', target: 'codex' };
  }
  if (storageGuidePaths.has(path)) {
    return {
      event: 'storage_guide_view',
      target: path.includes('cloudflare-r2') ? 'cloudflare-r2' : 'aws-s3',
    };
  }
  if (path === '/security') {
    return { event: 'security_doc_view', target: 'security-overview' };
  }
  if (path === '/docs/security-model') {
    return { event: 'security_doc_view', target: 'security-model' };
  }

  return null;
}

export function analyticsLinkEvent(href: string): AnalyticsEventMatch | null {
  if (href === '/rss.xml' || href === 'https://reinstate.dev/rss.xml') {
    return { event: 'changelog_subscribe', target: 'rss-feed' };
  }

  const rule = analyticsLinkRules.find(({ includes }) => href.includes(includes));
  return rule ? { event: rule.event, target: rule.target } : null;
}

export function isAllowedAnalyticsEvent(
  event: string,
  target: string,
): event is AnalyticsEvent {
  if (!analyticsEvents.includes(event as AnalyticsEvent)) return false;
  return analyticsEventTargets[event as AnalyticsEvent].includes(target);
}
