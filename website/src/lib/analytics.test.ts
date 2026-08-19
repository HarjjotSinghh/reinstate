import { describe, expect, it } from 'vitest';
import {
  aiReferralRules,
  analyticsAiReferralChannel,
  analyticsEventTargets,
  analyticsEvents,
  analyticsLinkEvent,
  analyticsPageEvent,
  isAllowedAnalyticsEvent,
} from './analytics';

describe('analytics event taxonomy', () => {
  it('keeps the complete approved event-name inventory', () => {
    expect(analyticsEvents).toEqual([
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
    ]);
    expect(Object.keys(analyticsEventTargets)).toEqual(analyticsEvents);
  });

  it('rejects undeclared event names and targets', () => {
    expect(isAllowedAnalyticsEvent('github_click', 'header')).toBe(true);
    expect(isAllowedAnalyticsEvent('command_copy', 'powershell')).toBe(true);
    expect(isAllowedAnalyticsEvent('github_click', 'raw-url')).toBe(false);
    expect(isAllowedAnalyticsEvent('invented_event', 'header')).toBe(false);
  });

  it.each([
    ['/integrations/claude-code', 'integration_view', 'claude-code'],
    ['/integrations/codex/', 'integration_view', 'codex'],
    ['/integrations/gemini', 'integration_view', 'gemini'],
    ['/integrations/opencode/', 'integration_view', 'opencode'],
    ['/integrations/grok', 'integration_view', 'grok'],
    ['/integrations/kimi', 'integration_view', 'kimi'],
    ['/integrations/qwen', 'integration_view', 'qwen'],
    ['/integrations/pi', 'integration_view', 'pi'],
    ['/integrations/cursor', 'integration_view', 'cursor'],
    ['/integrations/copilot', 'integration_view', 'copilot'],
    [
      '/guides/use-s3-for-coding-agent-session-storage',
      'storage_guide_view',
      'aws-s3',
    ],
    [
      '/guides/use-cloudflare-r2-for-coding-agent-session-storage/',
      'storage_guide_view',
      'cloudflare-r2',
    ],
    ['/security', 'security_doc_view', 'security-overview'],
    ['/docs/security-model', 'security_doc_view', 'security-model'],
  ])('maps %s to a controlled page event', (path, event, target) => {
    expect(analyticsPageEvent(path)).toEqual({ event, target });
  });

  it('does not emit page events for unrelated routes', () => {
    expect(analyticsPageEvent('/docs/getting-started')).toBeNull();
    expect(analyticsPageEvent('/privacy')).toBeNull();
  });

  it.each([
    ['/rss.xml', 'changelog_subscribe', 'rss-feed'],
    [
      'https://github.com/HarjjotSinghh/reinstate/releases/download/v0.1.0/reinstate.zip',
      'download_click',
      'github-release-asset',
    ],
    [
      'https://github.com/HarjjotSinghh/reinstate/issues/new?template=bug.yml',
      'issue_report_click',
      'github-issue-form',
    ],
    [
      'https://github.com/HarjjotSinghh/reinstate/blob/main/CONTRIBUTING.md',
      'contribute_click',
      'github-contributing',
    ],
  ])('maps %s to a controlled link event', (href, event, target) => {
    expect(analyticsLinkEvent(href)).toEqual({ event, target });
  });

  it('does not classify broad repository or issue-list links as conversions', () => {
    expect(
      analyticsLinkEvent('https://github.com/HarjjotSinghh/reinstate'),
    ).toBeNull();
    expect(
      analyticsLinkEvent('https://github.com/HarjjotSinghh/reinstate/issues'),
    ).toBeNull();
  });

  it.each([
    ['https://chatgpt.com/c/abc', 'https://reinstate.dev/docs', 'chatgpt'],
    [
      '',
      'https://reinstate.dev/docs?utm_source=chatgpt.com',
      'chatgpt',
    ],
    ['https://www.perplexity.ai/search/example', 'https://reinstate.dev/', 'perplexity'],
    ['https://copilot.microsoft.com/', 'https://reinstate.dev/', 'microsoft-copilot'],
    ['https://gemini.google.com/app', 'https://reinstate.dev/', 'google-gemini'],
    [
      '',
      'https://reinstate.dev/docs?utm_source=ai-overview&utm_campaign=launch',
      'google-ai-features',
    ],
    [
      'https://example.com/',
      'https://reinstate.dev/?utm_source=perplexity',
      'perplexity',
    ],
  ])('classifies controlled AI referrals from %s', (referrer, currentUrl, channel) => {
    expect(analyticsAiReferralChannel({ referrer, currentUrl })).toBe(channel);
  });

  it('does not guess an AI channel from generic search or arbitrary substring matches', () => {
    expect(
      analyticsAiReferralChannel({
        referrer: 'https://www.google.com/search?q=reinstate',
        currentUrl: 'https://reinstate.dev/',
      }),
    ).toBeNull();
    expect(
      analyticsAiReferralChannel({
        referrer: 'https://example.com/openai-news',
        currentUrl: 'https://reinstate.dev/?utm_source=my-copilot-review',
      }),
    ).toBeNull();
  });

  it('keeps every classifier output in a reviewed, non-identifying vocabulary', () => {
    expect(aiReferralRules.map(({ channel }) => channel)).toEqual([
      'chatgpt',
      'perplexity',
      'microsoft-copilot',
      'google-gemini',
      'google-ai-features',
    ]);
    expect(
      aiReferralRules.every(
        ({ hostnames, utmTokens }) => hostnames.length + utmTokens.length > 0,
      ),
    ).toBe(true);
  });
});
