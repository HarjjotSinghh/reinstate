import rss from '@astrojs/rss';
import { product } from '../data/product';

export function GET(context: { site?: URL }) {
  return rss({
    title: 'Reinstate updates',
    description:
      'Release and documentation updates for encrypted coding-agent session sync.',
    site: context.site ?? new URL(product.siteUrl),
    items: [
      {
        title: `${product.name} ${product.currentRelease}`,
        description:
          'The current pre-1.0 release candidate for same-vendor Claude Code and Codex session sync.',
        pubDate: new Date(`${product.currentReleaseDate}T00:00:00Z`),
        link: `${product.repositoryUrl}/releases/tag/${product.currentRelease}`,
      },
    ],
    customData: '<language>en</language>',
  });
}
