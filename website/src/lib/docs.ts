import { getCollection, type CollectionEntry } from 'astro:content';

export type DocEntry = CollectionEntry<'docs'>;

const TITLE_OVERRIDES: Record<string, string> = {
  'getting-started': 'Getting started',
  architecture: 'Architecture',
  adapters: 'Adapters',
  'security-model': 'Security model',
  comparison: 'Comparison',
  faq: 'FAQ',
  troubleshooting: 'Troubleshooting',
  README: 'Docs overview',
};

const ORDER: string[] = [
  'getting-started',
  'architecture',
  'adapters',
  'security-model',
  'comparison',
  'faq',
  'troubleshooting',
  'README',
];

export function docId(entry: DocEntry): string {
  // glob loader ids are usually filename without extension
  return entry.id.replace(/\.mdx?$/, '');
}

export function docSlug(entry: DocEntry): string {
  const id = docId(entry);
  if (id.toLowerCase() === 'readme') return 'overview';
  return id;
}

export function docTitle(entry: DocEntry): string {
  if (entry.data.title) return entry.data.title;
  const id = docId(entry);
  if (TITLE_OVERRIDES[id]) return TITLE_OVERRIDES[id];
  if (TITLE_OVERRIDES[id.toLowerCase()]) return TITLE_OVERRIDES[id.toLowerCase()];

  const body = entry.body ?? '';
  const h1 = body.match(/^#\s+(.+)$/m);
  if (h1?.[1]) return h1[1].trim();

  return id
    .split(/[-_/]/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export function docDescription(entry: DocEntry): string | undefined {
  if (entry.data.description) return entry.data.description;
  const body = entry.body ?? '';
  const lines = body
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('#') && !l.startsWith('>') && !l.startsWith('|'));
  return lines[0]?.slice(0, 160);
}

export async function getOrderedDocs(): Promise<DocEntry[]> {
  const all = await getCollection('docs');
  const rank = (entry: DocEntry) => {
    const id = docId(entry);
    const idx = ORDER.indexOf(id);
    return idx === -1 ? 1000 + id.localeCompare('') : idx;
  };
  return all.sort((a, b) => rank(a) - rank(b) || docId(a).localeCompare(docId(b)));
}

export async function getDocBySlug(slug: string): Promise<DocEntry | undefined> {
  const all = await getCollection('docs');
  return all.find((entry) => docSlug(entry) === slug);
}
