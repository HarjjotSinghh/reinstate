import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
  auditFreshness,
  formatFreshnessReport,
} from './check-freshness.mjs';

async function fixture(reviewDate = '2026-07-27') {
  const root = await mkdtemp(join(tmpdir(), 'reinstate-freshness-'));
  await mkdir(join(root, 'src/content/docs'), { recursive: true });
  await mkdir(join(root, 'src/content/guides'), { recursive: true });
  await mkdir(join(root, 'src/content/blog'), { recursive: true });
  await mkdir(join(root, 'src/data'), { recursive: true });
  await writeFile(
    join(root, 'src/content/docs/example.md'),
    `---\ntitle: Example\nupdatedAt: ${reviewDate}\n---\n`,
  );
  await writeFile(
    join(root, 'src/content/guides/example.md'),
    `---\ntitle: Example\nreviewedAt: ${reviewDate}\n---\n`,
  );
  await writeFile(
    join(root, 'src/content/blog/example.md'),
    `---\ntitle: Example\nreviewedAt: ${reviewDate}\n---\n`,
  );
  await writeFile(
    join(root, 'src/data/compatibility.json'),
    JSON.stringify({
      lastReviewed: reviewDate,
      agents: [
        {
          id: 'supported-agent',
          status: 'primary-release-target',
          lastTested: reviewDate,
          source:
            'https://github.com/HarjjotSinghh/reinstate/blob/commit/adapter_test.go',
        },
      ],
      environments: [
        {
          id: 'open-gate',
          status: 'release-gate-open',
          lastTested: null,
          source:
            'https://github.com/HarjjotSinghh/reinstate/blob/main/acceptance.md',
        },
      ],
    }),
  );
  await writeFile(
    join(root, 'src/data/product.ts'),
    `export const product = { lastVerified: '${reviewDate}' };\n`,
  );
  return root;
}

test('passes current records and preserves explicit open-gate nulls', async (t) => {
  const root = await fixture();
  t.after(() => rm(root, { recursive: true, force: true }));

  const result = await auditFreshness({
    root,
    asOf: new Date('2026-07-27T12:00:00Z'),
  });

  assert.deepEqual(result.warnings, []);
  assert.deepEqual(result.errors, []);
  assert.match(formatFreshnessReport(result), /6 reviewed records, 0 warnings, 0 errors/);
});

test('warns before the hard stale threshold and fails after it', async (t) => {
  const root = await fixture('2026-01-01');
  t.after(() => rm(root, { recursive: true, force: true }));

  const warning = await auditFreshness({
    root,
    asOf: new Date('2026-03-15T00:00:00Z'),
  });
  assert.equal(warning.errors.length, 0);
  assert.equal(warning.warnings.length, 6);

  const failure = await auditFreshness({
    root,
    asOf: new Date('2026-06-01T00:00:00Z'),
  });
  assert.equal(failure.warnings.length, 0);
  assert.equal(failure.errors.length, 6);
});

test('rejects missing evidence, missing review fields, and future dates', async (t) => {
  const root = await fixture('2026-07-28');
  t.after(() => rm(root, { recursive: true, force: true }));

  await writeFile(
    join(root, 'src/content/docs/example.md'),
    '---\ntitle: Example\n---\n',
  );
  await writeFile(
    join(root, 'src/data/compatibility.json'),
    JSON.stringify({
      lastReviewed: 'not-a-date',
      agents: [
        {
          id: 'missing-proof',
          status: 'primary-release-target',
          lastTested: null,
          source: 'https://example.com/untrusted',
        },
      ],
      environments: [],
    }),
  );

  const result = await auditFreshness({
    root,
    asOf: new Date('2026-07-27T00:00:00Z'),
  });
  assert.ok(result.errors.some((error) => error.includes('missing updatedAt')));
  assert.ok(result.errors.some((error) => error.includes('missing lastTested')));
  assert.ok(result.errors.some((error) => error.includes('canonical evidence')));
  assert.ok(result.errors.some((error) => error.includes('invalid review date')));
  assert.ok(result.errors.some((error) => error.includes('is in the future')));
});
