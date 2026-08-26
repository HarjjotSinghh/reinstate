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
  integration_view: ['claude-code', 'codex', 'gemini', 'opencode', 'grok', 'kimi', 'qwen', 'pi', 'cursor', 'copilot', 'cline'],
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
    utmTokens: ['chatgpt.com', 'chatgpt', 'openai', 'oai-search'],
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

/**
 * Marketing referral channels.
 *
 * Deliberately separate from `aiReferralRules`: AI-search citation and marketing
 * distribution are different questions, they are reported separately, and mixing
 * them would silently change the meaning of the existing AI-referral baseline.
 *
 * Like every other value in this file, these are a fixed reviewed taxonomy. The
 * resolved value is a channel *label* chosen from this list — never the raw
 * referrer, and never the raw query string.
 */
export type MarketingReferralChannel =
  | 'x'
  | 'linkedin'
  | 'devto'
  | 'hackernews'
  | 'github'
  | 'newsletter'
  | 'paid-x'
  | 'paid-meta';

export interface MarketingReferralRule {
  channel: MarketingReferralChannel;
  hostnames: readonly string[];
  utmTokens: readonly string[];
}

export const marketingReferralRules: readonly MarketingReferralRule[] = [
  {
    channel: 'x',
    hostnames: ['x.com', 'twitter.com', 't.co'],
    utmTokens: ['x', 'x.com', 'twitter'],
  },
  {
    channel: 'linkedin',
    hostnames: ['linkedin.com', 'lnkd.in'],
    utmTokens: ['linkedin'],
  },
  {
    channel: 'devto',
    hostnames: ['dev.to'],
    utmTokens: ['devto', 'dev-to', 'dev.to'],
  },
  {
    channel: 'hackernews',
    hostnames: ['news.ycombinator.com'],
    utmTokens: ['hn', 'hackernews'],
  },
  {
    channel: 'github',
    hostnames: ['github.com'],
    utmTokens: ['github'],
  },
  {
    channel: 'newsletter',
    hostnames: [],
    utmTokens: ['newsletter', 'tldr', 'console-dev'],
  },
  {
    channel: 'paid-x',
    hostnames: [],
    utmTokens: ['paid-x'],
  },
  {
    channel: 'paid-meta',
    hostnames: [],
    utmTokens: ['paid-meta'],
  },
] as const;

/**
 * Allowlisted campaign identifiers.
 *
 * A paid test needs to tell its variants apart, but the privacy commitment is
 * that no raw query parameter is ever transmitted. So `utm_content` is not read
 * through — it is matched against this fixed list, and only an exact match is
 * sent. An unrecognised value resolves to null and nothing is transmitted.
 *
 * Adding a campaign is a reviewed edit to this array, not a runtime behaviour.
 */
export const analyticsCampaigns = [
  'hk01-paths',
  'hk02-remote',
  'hk03-outcome',
  'hk04-envelope',
  'hk05-recover',
] as const;

export type AnalyticsCampaign = (typeof analyticsCampaigns)[number];

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

interface ReferralRule<TChannel extends string> {
  channel: TChannel;
  hostnames: readonly string[];
  utmTokens: readonly string[];
}

/**
 * Resolve a referral to a channel label.
 *
 * UTM tokens win over the referrer hostname, because an explicit campaign tag is
 * a stronger statement of origin than whatever the browser happened to send.
 */
function resolveReferralChannel<TChannel extends string>(
  rules: readonly ReferralRule<TChannel>[],
  { referrer, currentUrl }: { referrer?: string; currentUrl: string },
): TChannel | null {
  let destination: URL;
  try {
    destination = new URL(currentUrl, 'https://reinstate.dev');
  } catch {
    return null;
  }

  const trackingTokens = normalizedTrackingTokens(destination);
  for (const rule of rules) {
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
  const matched = rules.find((rule) =>
    rule.hostnames.some((expected) => hostnameMatches(hostname, expected)),
  );
  return matched?.channel ?? null;
}

export function analyticsAiReferralChannel(input: {
  referrer?: string;
  currentUrl: string;
}): AiReferralChannel | null {
  return resolveReferralChannel(aiReferralRules, input);
}

export function analyticsMarketingReferralChannel(input: {
  referrer?: string;
  currentUrl: string;
}): MarketingReferralChannel | null {
  return resolveReferralChannel(marketingReferralRules, input);
}

/**
 * Resolve `utm_content` to an allowlisted campaign id, or null.
 *
 * Anything not on the list is discarded rather than transmitted, so an arbitrary
 * query parameter can never reach the analytics provider through this path.
 */
export function analyticsCampaign(currentUrl: string): AnalyticsCampaign | null {
  let destination: URL;
  try {
    destination = new URL(currentUrl, 'https://reinstate.dev');
  } catch {
    return null;
  }
  const value = destination.searchParams.get('utm_content')?.trim().toLowerCase();
  if (!value) return null;
  return (
    analyticsCampaigns.find((campaign) => campaign === value) ?? null
  );
}

export function analyticsPageEvent(pathname: string): AnalyticsEventMatch | null {
  const path = normalizedPath(pathname);

  if (path === '/integrations/claude-code') {
    return { event: 'integration_view', target: 'claude-code' };
  }
  if (path === '/integrations/codex') {
    return { event: 'integration_view', target: 'codex' };
  }
  if (path === '/integrations/gemini') {
    return { event: 'integration_view', target: 'gemini' };
  }
  if (path === '/integrations/opencode') {
    return { event: 'integration_view', target: 'opencode' };
  }
  if (path === '/integrations/grok') {
    return { event: 'integration_view', target: 'grok' };
  }
  if (path === '/integrations/kimi') {
    return { event: 'integration_view', target: 'kimi' };
  }
  if (path === '/integrations/qwen') {
    return { event: 'integration_view', target: 'qwen' };
  }
  if (path === '/integrations/pi') {
    return { event: 'integration_view', target: 'pi' };
  }
  if (path === '/integrations/cursor') {
    return { event: 'integration_view', target: 'cursor' };
  }
  if (path === '/integrations/copilot') {
    return { event: 'integration_view', target: 'copilot' };
  }
  if (path === '/integrations/cline') {
    return { event: 'integration_view', target: 'cline' };
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
