export const searchIntents = [
  'navigational',
  'problem',
  'solution',
  'how-to',
  'troubleshooting',
  'comparison',
  'evaluation',
  'agent-specific',
  'platform-specific',
  'security',
  'contribution',
  'freshness',
  'answer',
] as const;

export type SearchIntent = (typeof searchIntents)[number];
