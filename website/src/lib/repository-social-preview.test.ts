import { readFile } from 'node:fs/promises';
import sharp from 'sharp';
import { describe, expect, it } from 'vitest';
import {
  renderRepositorySocialPreview,
  repositorySocialPreview,
} from './og-card';

const publicAsset = new URL('../../public/brand/github-social.png', import.meta.url);
const brandKitAsset = new URL(
  '../../../assets/brand-kit/png/github-social-1280x640.png',
  import.meta.url,
);

describe('GitHub repository social preview', () => {
  it('meets the GitHub image contract', async () => {
    const rendered = await renderRepositorySocialPreview();
    const metadata = await sharp(rendered).metadata();

    expect(metadata.format).toBe(repositorySocialPreview.format);
    expect(metadata.width).toBe(repositorySocialPreview.width);
    expect(metadata.height).toBe(repositorySocialPreview.height);
    expect(rendered.byteLength).toBeLessThan(repositorySocialPreview.maxBytes);
  });

  it('keeps both tracked copies reproducible from the shared renderer', async () => {
    const [rendered, publicCopy, brandKitCopy] = await Promise.all([
      renderRepositorySocialPreview(),
      readFile(publicAsset),
      readFile(brandKitAsset),
    ]);

    expect(publicCopy.equals(brandKitCopy)).toBe(true);

    if (process.platform === 'win32') {
      // The tracked PNGs are produced on Linux by the render-social-preview
      // GitHub Actions workflow; Satori/Sharp render the same markup to
      // different (still valid) bytes on win32, so byte-for-byte parity
      // against a fresh local render is only meaningful where that
      // workflow runs. Still verify what holds on every platform: the
      // fresh render is a valid PNG at the tracked copies' dimensions.
      const [renderedMetadata, trackedMetadata] = await Promise.all([
        sharp(rendered).metadata(),
        sharp(publicCopy).metadata(),
      ]);
      expect(renderedMetadata.format).toBe(repositorySocialPreview.format);
      expect(renderedMetadata.width).toBe(trackedMetadata.width);
      expect(renderedMetadata.height).toBe(trackedMetadata.height);
      return;
    }

    expect(publicCopy.equals(rendered)).toBe(true);
    expect(brandKitCopy.equals(rendered)).toBe(true);
  });
});
