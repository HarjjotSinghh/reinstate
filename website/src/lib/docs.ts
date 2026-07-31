import { getCollection, type CollectionEntry } from 'astro:content';

export type DocEntry = CollectionEntry<'docs'>;

export function docId(entry: DocEntry): string {
  // glob loader ids are usually filename without extension
  return entry.id.replace(/\.mdx?$/, '');
}

export function docSlug(entry: DocEntry): string {
  return docId(entry);
}

export function docTitle(entry: DocEntry): string {
  return entry.data.title;
}

export function docNavTitle(entry: DocEntry): string {
  return entry.data.navTitle;
}

export function docDescription(entry: DocEntry): string {
  return entry.data.description;
}

export async function getOrderedDocs(): Promise<DocEntry[]> {
  const published = await getCollection('docs', ({ data }) => !data.draft);
  return published.sort(
    (a, b) => a.data.order - b.data.order || docId(a).localeCompare(docId(b)),
  );
}

export async function getDocBySlug(slug: string): Promise<DocEntry | undefined> {
  const published = await getOrderedDocs();
  return published.find((entry) => docSlug(entry) === slug);
}
