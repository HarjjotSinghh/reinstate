import rss from '@astrojs/rss';
import { product } from '../../data/product';
import { releaseHistory } from '../../data/releases';

export const prerender = true;

export async function GET(context: { site?: URL }) {
  return rss({
    title: 'Reinstate changelog',
    description:
      'Versioned Reinstate release notes and coding-agent compatibility changes.',
    site: context.site ?? new URL(product.siteUrl),
    items: releaseHistory.map((release) => ({
      title: `${product.name} ${release.version}`,
      description: release.summary,
      pubDate: new Date(`${release.date}T00:00:00Z`),
      link: `/changelog#${release.version === product.currentRelease ? 'rc6' : 'release-history'}`,
    })),
    customData: '<language>en</language>',
  });
}
