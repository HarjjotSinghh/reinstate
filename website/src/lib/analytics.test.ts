import { describe, expect, it } from 'vitest';
import {
  analyticsEvents,
  analyticsLinkEvent,
  analyticsPageEvent,
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
  });

  it.each([
    ['/integrations/claude-code', 'integration_view', 'claude-code'],
    ['/integrations/codex/', 'integration_view', 'codex'],
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
});
