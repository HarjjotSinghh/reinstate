import { getCollection, type CollectionEntry } from 'astro:content';

export type GuideEntry = CollectionEntry<'guides'>;
export type BlogEntry = CollectionEntry<'blog'>;
export type EditorialEntry = GuideEntry | BlogEntry;

export function editorialSlug(entry: EditorialEntry): string {
  return entry.id.replace(/\.mdx?$/, '');
}

function newestFirst<T extends EditorialEntry>(entries: T[]): T[] {
  return entries.sort(
    (a, b) =>
      b.data.publishedAt.getTime() - a.data.publishedAt.getTime() ||
      editorialSlug(a).localeCompare(editorialSlug(b)),
  );
}

export async function getPublishedGuides(): Promise<GuideEntry[]> {
  const entries = await getCollection('guides', ({ data }) => !data.draft);
  return newestFirst(entries);
}

export async function getIndexableGuides(): Promise<GuideEntry[]> {
  return (await getPublishedGuides()).filter(({ data }) => !data.noindex);
}

export async function getPublishedBlogPosts(): Promise<BlogEntry[]> {
  const entries = await getCollection('blog', ({ data }) => !data.draft);
  return newestFirst(entries);
}

export async function getIndexableBlogPosts(): Promise<BlogEntry[]> {
  return (await getPublishedBlogPosts()).filter(({ data }) => !data.noindex);
}
