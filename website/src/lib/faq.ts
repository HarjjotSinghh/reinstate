import type { FaqEntry } from './schema';

function markdownToText(source: string): string {
  return source
    .replace(/```(?:[^\n]*)\n([\s\S]*?)```/g, '$1')
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/<[^>]+>/g, ' ')
    .replace(/^\s*\|?[\s:|-]+\|?\s*$/gm, ' ')
    .replace(/^\s*[|>*+-]\s*/gm, '')
    .replace(/^\s*\d+\.\s*/gm, '')
    .replace(/[`*_~]/g, '')
    .replace(/\s*\|\s*/g, '; ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function extractFaqEntries(markdown: string): FaqEntry[] {
  const source = markdown.replace(/^---\n[\s\S]*?\n---\n/, '');
  const headings = [...source.matchAll(/^##\s+(.+?)\s*$/gm)];

  return headings
    .map((heading, index) => {
      const start = (heading.index ?? 0) + heading[0].length;
      const end = headings[index + 1]?.index ?? source.length;
      return {
        question: markdownToText(heading[1]),
        answer: markdownToText(source.slice(start, end)),
      };
    })
    .filter(
      (entry) =>
        entry.question.endsWith('?') &&
        entry.question.length > 3 &&
        entry.answer.length > 10,
    );
}
