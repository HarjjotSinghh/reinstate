import rss from '@astrojs/rss';
import { product } from '../data/product';
import {
  editorialSlug,
  getPublishedBlogPosts,
  getPublishedGuides,
} from '../lib/editorial';

export async function GET(context: { site?: URL }) {
  const [guides, blogPosts] = await Promise.all([
    getPublishedGuides(),
    getPublishedBlogPosts(),
  ]);

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
