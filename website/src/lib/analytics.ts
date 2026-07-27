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

export interface AnalyticsEventMatch {
  event: AnalyticsEvent;
  target: string;
}

export interface AnalyticsLinkRule extends AnalyticsEventMatch {
  includes: string;
}

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
