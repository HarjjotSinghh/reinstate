import rss from '@astrojs/rss';
import { product } from '../../data/product';
import {
  editorialSlug,
  getIndexableBlogPosts,
} from '../../lib/editorial';

export const prerender = true;

export async function GET(context: { site?: URL }) {
  const posts = await getIndexableBlogPosts();

  return rss({
    title: 'Reinstate blog',
    description:
      'Engineering explanations for encrypted coding-agent session continuity.',
    site: context.site ?? new URL(product.siteUrl),
    items: posts.map((entry) => ({
      title: entry.data.title,
      description: entry.data.description,
      pubDate: entry.data.publishedAt,
      link: `/blog/${editorialSlug(entry)}`,
    })),
    customData: '<language>en</language>',
  });
}
