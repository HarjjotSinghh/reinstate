import rss from '@astrojs/rss';
import { product } from '../data/product';
import { releaseHistory } from '../data/releases';
import {
  editorialSlug,
  getIndexableBlogPosts,
  getIndexableGuides,
} from '../lib/editorial';

export async function GET(context: { site?: URL }) {
  const [guides, blogPosts] = await Promise.all([
    getIndexableGuides(),
    getIndexableBlogPosts(),
  ]);

  return rss({
    title: 'Reinstate updates',
    description:
      'Release and documentation updates for encrypted coding-agent session sync.',
    site: context.site ?? new URL(product.siteUrl),
    items: [
      ...releaseHistory.map((release) => ({
        title: `${product.name} ${release.version}`,
        description: release.summary,
        pubDate: new Date(`${release.date}T00:00:00Z`),
        link: `${product.repositoryUrl}/releases/tag/${release.version}`,
      })),
      ...blogPosts.map((entry) => ({
        title: entry.data.title,
        description: entry.data.description,
        pubDate: entry.data.publishedAt,
        link: `/blog/${editorialSlug(entry)}`,
      })),
      ...guides.map((entry) => ({
        title: entry.data.title,
        description: entry.data.description,
        pubDate: entry.data.publishedAt,
        link: `/guides/${editorialSlug(entry)}`,
      })),
    ],
    customData: '<language>en</language>',
  });
}
