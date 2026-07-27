import type { APIRoute, GetStaticPaths } from 'astro';
import { getCollection } from 'astro:content';
import { staticOgPages, type OgPage } from '../../data/og-pages';
import { renderOgCard } from '../../lib/og-card';

export const prerender = true;

function routeToSlug(route: string): string {
  const normalized = route.replace(/^\/+|\/+$/g, '');
  return normalized || 'home';
}

export const getStaticPaths: GetStaticPaths = async () => {
  const [docs, guides, blogPosts] = await Promise.all([
    getCollection('docs', ({ data }) => !data.draft),
    getCollection('guides', ({ data }) => !data.draft),
    getCollection('blog', ({ data }) => !data.draft),
  ]);
  const docPages: OgPage[] = docs.map((entry) => ({
    route: `/docs/${entry.id.replace(/\.mdx?$/, '')}`,
    kind: 'Documentation',
    title: entry.data.title,
    description: entry.data.description,
  }));
  const guidePages: OgPage[] = guides.map((entry) => ({
    route: `/guides/${entry.id.replace(/\.mdx?$/, '')}`,
    kind: 'Guide',
    title: entry.data.title,
    description: entry.data.description,
  }));
  const blogPages: OgPage[] = blogPosts.map((entry) => ({
    route: `/blog/${entry.id.replace(/\.mdx?$/, '')}`,
    kind: 'Engineering notes',
    title: entry.data.title,
    description: entry.data.description,
  }));

  return [...staticOgPages, ...docPages, ...guidePages, ...blogPages].map((page) => ({
    params: { slug: routeToSlug(page.route) },
    props: { page },
  }));
};

export const GET: APIRoute = async ({ props }) => {
  const page = props.page as OgPage;
  const image = await renderOgCard(page);

  return new Response(new Uint8Array(image), {
    headers: {
      'Content-Type': 'image/png',
      'Cache-Control': 'public, max-age=31536000, immutable',
    },
  });
};
