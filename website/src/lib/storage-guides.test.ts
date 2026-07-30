import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';

const guidesDir = new URL('../content/guides/', import.meta.url);
const guideFiles = {
  s3: 'use-s3-for-coding-agent-session-storage.md',
  r2: 'use-cloudflare-r2-for-coding-agent-session-storage.md',
} as const;

async function readGuide(provider: keyof typeof guideFiles) {
  return readFile(new URL(guideFiles[provider], guidesDir), 'utf8');
}

function squashWhitespace(source: string) {
  return source.replace(/\s+/g, ' ');
}

describe('S3-compatible storage outcome guides', () => {
  it.each(Object.keys(guideFiles) as Array<keyof typeof guideFiles>)(
    'keeps the %s workflow aligned with the implemented Reinstate contract',
    async (provider) => {
      const guide = await readGuide(provider);
      const guideText = squashWhitespace(guide);

      for (const command of [
        'rein init',
        'rein setup check',
        'rein doctor --self-test',
        'rein push --agent AGENT --session SESSION_ID --dry-run',
        'rein status',
      ]) {
        expect(guideText, `${provider} command ${command}`).toContain(command);
      }

      for (const behavior of [
        'two-byte probe',
        'local configuration only after that probe succeeds',
        'first persistent `manifest.age`',
        'profiles/<profile_id>/manifest.age',
        'profiles/<profile_id>/snapshots/<opaque-id>.age',
        'does not provide a general `rein undo`',
        'same-vendor',
      ]) {
        expect(guideText, `${provider} behavior ${behavior}`).toContain(behavior);
      }

      for (const section of [
        '## Key points',
        '## Command placeholders and parameters',
        '## Failure modes and common errors',
        '## Safe rollback and undo',
        '## Verification checklist',
        '## Current limitations',
      ]) {
        expect(guide, `${provider} section ${section}`).toContain(section);
      }

      expect(guide.match(/\*\*Expected result:\*\*/g)).toHaveLength(5);
      expect(guide.match(/^\s+anchor: "([^"]+)"$/gm)).toHaveLength(5);
      expect(guide).not.toContain('rein resume');
    },
  );

  it('documents a private, least-privilege Amazon S3 setup from AWS sources', async () => {
    const guide = await readGuide('s3');
    const guideText = squashWhitespace(guide);

    for (const providerFact of [
      'Block Public Access',
      's3:ListBucket',
      's3:GetObject',
      's3:PutObject',
      's3:DeleteObject',
      '--endpoint https://s3.AWS_REGION.amazonaws.com',
      '--region AWS_REGION',
      'Reinstate accepts an access-key ID and secret access key, but not the session token',
      'does not create or delete S3 buckets',
      'policy that requires request-specific SSE-KMS headers can reject',
      'https://docs.aws.amazon.com/',
    ]) {
      expect(guideText, `S3 fact ${providerFact}`).toContain(providerFact);
    }
  });

  it('documents a private, bucket-scoped Cloudflare R2 setup from Cloudflare sources', async () => {
    const guide = await readGuide('r2');
    const guideText = squashWhitespace(guide);

    for (const providerFact of [
      'buckets are not public by default',
      'Object Read & Write',
      '--endpoint https://ACCOUNT_ID.r2.cloudflarestorage.com',
      '--region auto',
      'JURISDICTION',
      'PutObject',
      'GetObject',
      'HeadObject',
      'DeleteObject',
      'ListObjectsV2',
      'does not create or delete R2 buckets',
      'no additional session-token field',
      'https://developers.cloudflare.com/r2/',
    ]) {
      expect(guideText, `R2 fact ${providerFact}`).toContain(providerFact);
    }
  });

  it('uses placeholders without publishing credentials or passphrases', async () => {
    const combined = `${await readGuide('s3')}\n${await readGuide('r2')}`;

    expect(combined).not.toMatch(/\b(?:AKIA|ASIA)[A-Z0-9]{16}\b/);
    expect(combined).not.toMatch(/REINSTATE_S3_SECRET_ACCESS_KEY\s*=/);
    expect(combined).not.toMatch(/secretAccessKey\s*:/i);
    expect(combined).not.toMatch(
      /(?:secret access key|passphrase)\s*(?:is|=|:)\s*["']?[A-Za-z0-9+/]{16,}/i,
    );
  });
});
