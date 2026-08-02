import { spawnSync } from 'node:child_process';
import { describe, expect, it } from 'vitest';
import {
  expectedCliReleaseAssets,
  parseArguments,
  validateCliRelease,
} from '../../scripts/check-cli-release.mjs';

const checker = new URL('../../scripts/check-cli-release.mjs', import.meta.url);
const TAG = 'v0.2.0-rc.3';

function release(overrides: Record<string, unknown> = {}) {
  return {
    tagName: TAG,
    isDraft: false,
    isPrerelease: true,
    publishedAt: '2026-07-27T09:14:04Z',
    assets: expectedCliReleaseAssets(TAG).map((name) => ({
      name,
      state: 'uploaded',
    })),
    ...overrides,
  };
}

describe('published GitHub CLI release contract', () => {
  it('requires checksums, five platform archives, their SBOMs, and source', () => {
    const assets = expectedCliReleaseAssets(TAG);
    const archives = assets.filter(
      (name) => name.endsWith('.tar.gz') || name.endsWith('.zip'),
    );
    const platformArchives = archives.filter(
      (name) => !name.endsWith('_source.tar.gz'),
    );

    expect(assets).toHaveLength(12);
    expect(platformArchives).toHaveLength(5);
    expect(assets.filter((name) => name.endsWith('.sbom.json'))).toHaveLength(5);
    expect(assets).toContain('checksums.txt');
    expect(assets).toContain('reinstate_0.2.0-rc.3_source.tar.gz');
    for (const archive of platformArchives) {
      expect(assets).toContain(`${archive}.sbom.json`);
    }

    expect(validateCliRelease(release(), TAG)).toMatchObject({
      tag: TAG,
      publishedAt: '2026-07-27T09:14:04Z',
      requiredAssets: assets,
    });
  });

  it('rejects invalid or mismatched tags', () => {
    for (const tag of [
      '',
      '0.2.0-rc.3',
      'v0.1',
      'v0.2.0-rc.3;echo unsafe',
      'website-v2026.07.28.1',
    ]) {
      expect(() => expectedCliReleaseAssets(tag)).toThrow(
        'invalid CLI release tag',
      );
    }
    expect(() =>
      validateCliRelease(release({ tagName: 'v0.1.0-rc.5' }), TAG),
    ).toThrow('tagName');
  });

  it('rejects draft, unpublished, and malformed releases', () => {
    expect(() => validateCliRelease(release({ isDraft: true }), TAG)).toThrow(
      'must not be a draft',
    );
    for (const publishedAt of [null, '', 'not-a-date']) {
      expect(() =>
        validateCliRelease(release({ publishedAt }), TAG),
      ).toThrow('must be published');
    }
    expect(() => validateCliRelease(null, TAG)).toThrow('JSON object');
    expect(() =>
      validateCliRelease(release({ assets: 'not-an-array' }), TAG),
    ).toThrow('assets must be an array');
  });

  it('reports every missing required asset and rejects duplicate names', () => {
    const required = expectedCliReleaseAssets(TAG);
    const missing = [
      'checksums.txt',
      'reinstate_0.2.0-rc.3_windows_amd64.zip',
      'reinstate_0.2.0-rc.3_linux_arm64.tar.gz.sbom.json',
      'reinstate_0.2.0-rc.3_source.tar.gz',
    ];
    const assets = required
      .filter((name) => !missing.includes(name))
      .map((name) => ({ name }));

    expect(() => validateCliRelease(release({ assets }), TAG)).toThrow(
      missing.join(', '),
    );
    expect(() =>
      validateCliRelease(
        release({
          assets: [
            ...release().assets,
            { name: 'checksums.txt', state: 'uploaded' },
          ],
        }),
        TAG,
      ),
    ).toThrow('duplicate asset names');
  });

  it('requires every release-contract asset to be fully uploaded', () => {
    const assets = release().assets.map((asset) =>
      asset.name === 'checksums.txt'
        ? { ...asset, state: 'new' }
        : asset,
    );
    expect(() => validateCliRelease(release({ assets }), TAG)).toThrow(
      'required assets are not uploaded: checksums.txt',
    );
  });

  it('accepts --tag with release JSON on stdin and fails closed otherwise', () => {
    expect(parseArguments(['--tag', TAG])).toEqual({ tag: TAG });
    expect(() => parseArguments([])).toThrow('usage:');
    expect(() => parseArguments([TAG])).toThrow('usage:');

    const valid = spawnSync(
      process.execPath,
      [checker.pathname, '--tag', TAG],
      {
        encoding: 'utf8',
        input: JSON.stringify(release()),
      },
    );
    expect(valid.status).toBe(0);
    expect(valid.stdout).toContain(`GitHub CLI release ${TAG} verified`);

    const malformed = spawnSync(
      process.execPath,
      [checker.pathname, '--tag', TAG],
      {
        encoding: 'utf8',
        input: '{',
      },
    );
    expect(malformed.status).toBe(1);
    expect(malformed.stderr).toContain('not valid JSON');
  });
});
