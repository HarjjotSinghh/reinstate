import { spawnSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import {
  EXPECTED_VERCEL_PROJECT_LINK,
  validateVercelProjectLink,
} from '../../scripts/check-vercel-project-link.mjs';

const checker = new URL(
  '../../scripts/check-vercel-project-link.mjs',
  import.meta.url,
);
const temporaryDirectories: string[] = [];

afterEach(() => {
  while (temporaryDirectories.length > 0) {
    rmSync(temporaryDirectories.pop()!, { force: true, recursive: true });
  }
});

function fixtureDirectory() {
  const root = mkdtempSync(join(tmpdir(), 'reinstate-vercel-project-'));
  temporaryDirectories.push(root);
  return root;
}

describe('Vercel project link contract', () => {
  it('accepts only the canonical Reinstate project link', () => {
    expect(
      validateVercelProjectLink({ ...EXPECTED_VERCEL_PROJECT_LINK }),
    ).toEqual(EXPECTED_VERCEL_PROJECT_LINK);

    for (const [key, value] of Object.entries(
      EXPECTED_VERCEL_PROJECT_LINK,
    )) {
      expect(() =>
        validateVercelProjectLink({
          ...EXPECTED_VERCEL_PROJECT_LINK,
          [key]: `${value}-wrong`,
        }),
      ).toThrow(key);
    }
  });

  it('rejects missing, additional, and non-object project link data', () => {
    const { orgId: _orgId, ...missing } = EXPECTED_VERCEL_PROJECT_LINK;
    expect(() => validateVercelProjectLink(missing)).toThrow(
      'contain exactly',
    );
    expect(() =>
      validateVercelProjectLink({
        ...EXPECTED_VERCEL_PROJECT_LINK,
        unexpected: true,
      }),
    ).toThrow('contain exactly');
    expect(() => validateVercelProjectLink(null)).toThrow('JSON object');
    expect(() => validateVercelProjectLink([])).toThrow('JSON object');
  });

  it('checks .vercel/project.json by default and supports an explicit path', () => {
    const root = fixtureDirectory();
    const vercelDirectory = join(root, '.vercel');
    mkdirSync(vercelDirectory);
    writeFileSync(
      join(vercelDirectory, 'project.json'),
      JSON.stringify(EXPECTED_VERCEL_PROJECT_LINK),
    );

    const defaultResult = spawnSync(process.execPath, [checker.pathname], {
      cwd: root,
      encoding: 'utf8',
    });
    expect(defaultResult.status).toBe(0);
    expect(defaultResult.stdout).toContain('Vercel project link verified');

    const explicitPath = join(root, 'linked-project.json');
    writeFileSync(explicitPath, JSON.stringify(EXPECTED_VERCEL_PROJECT_LINK));
    const explicitResult = spawnSync(
      process.execPath,
      [checker.pathname, explicitPath],
      { cwd: root, encoding: 'utf8' },
    );
    expect(explicitResult.status).toBe(0);
  });

  it('fails closed for malformed files and unexpected arguments', () => {
    const root = fixtureDirectory();
    const malformedPath = join(root, 'project.json');
    writeFileSync(malformedPath, '{');

    const malformed = spawnSync(
      process.execPath,
      [checker.pathname, malformedPath],
      { encoding: 'utf8' },
    );
    expect(malformed.status).toBe(1);
    expect(malformed.stderr).toContain('is not valid JSON');

    const unexpected = spawnSync(
      process.execPath,
      [checker.pathname, malformedPath, 'extra'],
      { encoding: 'utf8' },
    );
    expect(unexpected.status).toBe(1);
    expect(unexpected.stderr).toContain('usage:');
  });
});
