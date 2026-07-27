import { describe, expect, it } from 'vitest';
import compatibility from './compatibility.json';
import { product } from './product';
import { releaseHistory } from './releases';

describe('release source of truth', () => {
  it('keeps the current product, compatibility, and release history aligned', () => {
    expect(releaseHistory[0].version).toBe(product.currentRelease);
    expect(releaseHistory[0].date).toBe(product.currentReleaseDate);
    expect(compatibility.reinstateVersion).toBe(product.currentRelease);
  });

  it('keeps release history newest-first without duplicate versions', () => {
    expect(new Set(releaseHistory.map(({ version }) => version)).size).toBe(
      releaseHistory.length,
    );
    const timestamps = releaseHistory.map(({ date }) => Date.parse(`${date}T00:00:00Z`));
    expect(timestamps).toEqual([...timestamps].sort((left, right) => right - left));
  });
});
