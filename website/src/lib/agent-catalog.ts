import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import compatibility from '../data/compatibility.json';

export const TIERS = ['T0', 'T1', 'T2', 'T3', 'T4', 'T5'] as const;
export type Tier = (typeof TIERS)[number];

export const TIER_LABELS: Record<Tier, string> = {
  T0: 'Known',
  T1: 'Discover',
  T2: 'Handoff source',
  T3: 'Verified resume',
  T4: 'Handoff destination',
  T5: 'Encrypted sync',
};

export const FAMILY_LABELS: Record<string, string> = {
  F1: 'Home tree',
  F2: 'CLI query',
  F3: 'Embedded DB',
  F4: 'Project file',
  F5: 'Remote',
};

export const T0_REASON_LABELS: Record<string, string> = {
  layout_unverified: 'layout unverified',
  server_backed: 'server-backed history',
  desktop_only: 'desktop-only distribution',
  no_local_history: 'no local history',
  unidentified_product: 'unidentified product',
  unofficial_distribution_only: 'unofficial distribution only',
};

const TIER_FROM_GO: Record<string, Tier> = {
  TierKnown: 'T0',
  TierDiscover: 'T1',
  TierHandoffFrom: 'T2',
  TierResume: 'T3',
  TierHandoffTo: 'T4',
  TierSync: 'T5',
};

const FAMILY_FROM_GO: Record<string, string> = {
  FamilyHomeTree: 'F1',
  FamilyCLIQuery: 'F2',
  FamilyEmbeddedDB: 'F3',
  FamilyProjectFile: 'F4',
  FamilyRemote: 'F5',
};

const SESSIONINDEX_KEYS: Record<string, string> = {
  AgentClaude: 'claude',
  AgentCodex: 'codex',
  AgentGemini: 'gemini',
  AgentOpenCode: 'opencode',
  AgentGrok: 'grok',
  AgentKimi: 'kimi',
  AgentQwen: 'qwen',
};

const T0_FROM_GO: Record<string, string> = {
  T0NoLocalHistory: 'no_local_history',
  T0ServerBacked: 'server_backed',
  T0DesktopOnly: 'desktop_only',
  T0UnidentifiedProduct: 'unidentified_product',
  T0UnofficialDistributionOnly: 'unofficial_distribution_only',
  T0LayoutUnverified: 'layout_unverified',
};

export interface CatalogDescriptor {
  key: string;
  displayName: string;
  tier: Tier;
  family: string;
  t0Reason: string | null;
}

export interface CompatibilityAgent {
  id: string;
  key: string;
  name: string;
  vendor: string;
  tier: Tier;
  storageFamily: string;
  t0Reason: string | null;
  integrationPath: string | null;
  status: string;
  minimumTestedVersion: string | null;
  maximumTestedVersion: string | null;
  resumeMode: string;
  evidenceLevel: string;
  evidence: {
    storagePage: string;
    fixtures: string[];
    deviceReports: string[];
    probeReports: string[];
  };
  lastTested: string | null;
  source: string;
  notes: string;
}

export const compatibilityAgents = compatibility.agents as CompatibilityAgent[];

export function tierRank(tier: string): number {
  return TIERS.indexOf(tier as Tier);
}

export function reachedTier(agentTier: string, column: Tier): boolean {
  return tierRank(agentTier) >= tierRank(column);
}

export function versionedAgents(
  agents: CompatibilityAgent[] = compatibilityAgents,
): CompatibilityAgent[] {
  return agents.filter(
    (agent) => agent.minimumTestedVersion && agent.maximumTestedVersion,
  );
}

export function integrationAgents(
  agents: CompatibilityAgent[] = compatibilityAgents,
): CompatibilityAgent[] {
  return agents.filter((agent) => tierRank(agent.tier) >= 1 && agent.integrationPath);
}

export function agentNamesMatch(cell: string, desc: CatalogDescriptor): boolean {
  const name = stripMarkdown(cell);
  return (
    name.localeCompare(desc.displayName, undefined, { sensitivity: 'accent' }) === 0 ||
    name.localeCompare(`OpenAI ${desc.displayName}`, undefined, {
      sensitivity: 'accent',
    }) === 0 ||
    name.localeCompare(desc.key, undefined, { sensitivity: 'accent' }) === 0
  );
}

export function parseCatalogDescriptors(catalogDir: string): CatalogDescriptor[] {
  const files = readdirSync(catalogDir).filter(
    (name) => name.endsWith('.go') && !name.endsWith('_test.go'),
  );
  const descriptors: CatalogDescriptor[] = [];
  for (const file of files) {
    const source = readFileSync(join(catalogDir, file), 'utf8');
    const displayName = field(source, 'DisplayName');
    const keyRaw = field(source, 'Key');
    const tierRaw = field(source, 'Tier');
    const familyRaw = field(source, 'Family');
    if (!displayName || !keyRaw || !tierRaw || !familyRaw) {
      continue;
    }
    const key = resolveCatalogKey(keyRaw);
    const tier = TIER_FROM_GO[tierRaw.replace(/^agents\./, '')];
    const family = FAMILY_FROM_GO[familyRaw.replace(/^agents\./, '')];
    if (!key || !tier || !family) {
      throw new Error(`${file}: could not parse catalog identity`);
    }
    const t0Raw = field(source, 'T0Reason');
    const t0Token = t0Raw?.replace(/^agents\./, '') ?? '';
    descriptors.push({
      key,
      displayName,
      tier,
      family,
      t0Reason: t0Token ? (T0_FROM_GO[t0Token] ?? null) : null,
    });
  }
  return descriptors.sort((a, b) => a.key.localeCompare(b.key));
}

export function parseCompatibilityDocTiers(markdown: string): Map<string, Tier> {
  const table = tableWithHeader(markdown, 'Tier');
  const claimed = new Map<string, Tier>();
  if (!table) {
    return claimed;
  }
  const agentIndex = table.headers.indexOf('Agent');
  const tierIndex = table.headers.indexOf('Tier');
  for (const row of table.rows) {
    const name = stripMarkdown(row[agentIndex] ?? '');
    const token = (row[tierIndex] ?? '').replace(/\*/g, '').trim().toUpperCase();
    if (name && /^T[0-5]$/.test(token)) {
      claimed.set(name, token as Tier);
    }
  }
  return claimed;
}

export function findDocRowName(
  claimed: Map<string, Tier>,
  desc: CatalogDescriptor,
): string | undefined {
  for (const name of claimed.keys()) {
    if (agentNamesMatch(name, desc)) {
      return name;
    }
  }
  return undefined;
}

const OVER_CLAIM_WORD = /\b(resume|sync|push|pull)\b|handoff to/i;
const OVER_CLAIM_DENY =
  /\b(no|not|never|without|refuse|refuses|refused|fail|fails|failed|read-only|source-only|unsupported|unverified|unavailable|later|planned|exploring|cannot)\b|mutation\/sync|—/;
const OVER_CLAIM_AFFIRM =
  /\b(support|supports|supported|include|includes|included|can|allow|allows|provide|provides|enable|enables)\b/i;
const T0_EXTRA_CLAIM = /\b(index(?:es|ed|ing)?|discover(?:y|s|ed)?|inspect(?:s|ed)?)\b/i;

export function overClaims(body: string, agent: CompatibilityAgent): string[] {
  const hits: string[] = [];
  const sentences = body.split(/(?<=[.!?\n])\s+/);
  for (const sentence of sentences) {
    if (!sentenceIncludesAgent(sentence, agent)) {
      continue;
    }
    const claim = capabilityClaim(sentence, agent);
    if (claim) {
      hits.push(`${claim}: ${squash(sentence)}`);
    }
  }
  return hits;
}

export function repositoryCatalogDir(): string {
  return fileURLToPath(new URL('../../../internal/agents/catalog/', import.meta.url));
}

export function repositoryCompatibilityDoc(): string {
  return readFileSync(
    fileURLToPath(new URL('../../../docs/compatibility.md', import.meta.url)),
    'utf8',
  );
}

function capabilityClaim(sentence: string, agent: CompatibilityAgent): string | null {
  const rank = tierRank(agent.tier);
  const forbidden =
    rank <= 0
      ? new RegExp(`${OVER_CLAIM_WORD.source}|${T0_EXTRA_CLAIM.source}`, 'i')
      : OVER_CLAIM_WORD;
  if (!forbidden.test(sentence)) {
    return null;
  }
  if (OVER_CLAIM_DENY.test(sentence)) {
    return null;
  }
  if (rank >= 5) {
    return null;
  }
  if (rank >= 2 && !/\b(resume|sync|push|pull)\b|handoff to/i.test(sentence)) {
    return null;
  }
  if (!OVER_CLAIM_AFFIRM.test(sentence) && !/\|/.test(sentence)) {
    return null;
  }
  return 'above-tier capability';
}

function sentenceIncludesAgent(sentence: string, agent: CompatibilityAgent): boolean {
  const haystack = sentence.toLowerCase();
  const names = [agent.name, agent.key, `OpenAI ${agent.name}`]
    .filter(Boolean)
    .map((value) => value.toLowerCase());
  return names.some((name) => haystack.includes(name.toLowerCase()));
}

function field(source: string, name: string): string | null {
  const match = source.match(new RegExp(`(?:^|\\n)\\s*${name}:\\s*([^,\\n]+)`));
  if (!match) {
    return null;
  }
  return match[1].trim().replace(/^"|"$/g, '');
}

function resolveCatalogKey(raw: string): string | null {
  if (/^"[a-z][a-z0-9_-]*"$/.test(raw) || /^[a-z][a-z0-9_-]*$/.test(raw)) {
    return raw.replace(/"/g, '');
  }
  const constName = raw.replace(/^sessionindex\./, '');
  return SESSIONINDEX_KEYS[constName] ?? null;
}

function stripMarkdown(value: string): string {
  let next = value.trim();
  if (next.startsWith('[')) {
    const end = next.indexOf(']');
    if (end > 1) {
      next = next.slice(1, end);
    }
  }
  return next.replace(/\*/g, '').trim();
}

function squash(value: string): string {
  return value.replace(/\s+/g, ' ').trim().slice(0, 180);
}

function tableWithHeader(
  markdown: string,
  header: string,
): { headers: string[]; rows: string[][] } | null {
  for (const table of parseMarkdownTables(markdown)) {
    if (table.headers.includes(header) && table.headers.includes('Agent')) {
      return table;
    }
  }
  return null;
}

function parseMarkdownTables(markdown: string): { headers: string[]; rows: string[][] }[] {
  const tables: { headers: string[]; rows: string[][] }[] = [];
  let lines: string[] = [];
  const flush = () => {
    if (lines.length < 2) {
      lines = [];
      return;
    }
    const headers = splitRow(lines[0]);
    const start = isSeparator(lines[1]) ? 2 : 1;
    tables.push({
      headers,
      rows: lines.slice(start).map(splitRow).filter((row) => row.length > 0),
    });
    lines = [];
  };
  for (const line of markdown.split('\n')) {
    if (line.trim().startsWith('|')) {
      lines.push(line.trim());
    } else {
      flush();
    }
  }
  flush();
  return tables;
}

function splitRow(line: string): string[] {
  return line
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map((cell) => cell.trim());
}

function isSeparator(line: string): boolean {
  return splitRow(line).every((cell) => cell === '' || /^:?-+:?$/.test(cell));
}
