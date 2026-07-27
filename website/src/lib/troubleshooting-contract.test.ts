import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';

const troubleshootingPath = new URL(
  '../content/docs/troubleshooting.md',
  import.meta.url,
);

const requiredFields = [
  'Symptom',
  'Likely cause',
  'Affected agent(s)',
  'Affected OS',
  'Diagnostic commands',
  'Corrective action',
  'Expected recovery evidence',
  'When to file an issue',
] as const;

describe('troubleshooting entry contract', () => {
  it('gives every troubleshooting question all eight fields in order', async () => {
    const source = await readFile(troubleshootingPath, 'utf8');
    const body = source.replace(/^---\n[\s\S]*?\n---\n/, '');
    const sections = body
      .split(/^##\s+/m)
      .slice(1)
      .filter((section) => !section.startsWith('Still stuck?'));

    expect(sections).toHaveLength(8);

    for (const section of sections) {
      const [question = 'unknown entry'] = section.split('\n');
      const fields = [...section.matchAll(/^###\s+(.+)$/gm)].map(
        (match) => match[1],
      );
      expect(fields, question).toEqual(requiredFields);

      for (const field of requiredFields) {
        const escapedField = field.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const fieldBody = section.match(
          new RegExp(`^### ${escapedField}\\n\\n([\\s\\S]+?)(?=^### |$)`, 'm'),
        )?.[1];
        expect(fieldBody?.trim().length, `${question}: ${field}`).toBeGreaterThan(
          20,
        );
      }
    }
  });

  it('keeps diagnostics scoped and secret-safe', async () => {
    const source = await readFile(troubleshootingPath, 'utf8');

    expect(source).toContain('same vendor');
    expect(source).toContain('0.1.0-rc.6');
    expect(source).toContain(
      'rein pull --agent AGENT --session SESSION_ID --dry-run --json',
    );
    expect(source).toContain('private security report');
    expect(source).not.toContain('REINSTATE_PASSPHRASE=');
    expect(source).not.toContain('Gemini');
    expect(source).not.toContain('cross-agent resume');
  });
});
