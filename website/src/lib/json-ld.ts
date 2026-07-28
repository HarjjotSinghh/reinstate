export type JsonLdNode = Record<string, unknown>;

export function serializeJsonLd(
  data: JsonLdNode | JsonLdNode[] = [],
): string | null {
  const graph = (Array.isArray(data) ? data : [data]).filter(
    (entry) => Object.keys(entry).length > 0,
  );
  if (!graph.length) {
    return null;
  }

  return JSON.stringify({
    '@context': 'https://schema.org',
    '@graph': graph,
  })
    .replace(/</g, '\\u003c')
    .replace(/\u2028/g, '\\u2028')
    .replace(/\u2029/g, '\\u2029');
}
