import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import { describe, expect, it, vi } from 'vitest';

import { copyInstallCommand } from './install-copy';

describe('homepage install command copy', () => {
  it('tracks the conversion only after the clipboard write succeeds', async () => {
    const calls: string[] = [];
    const writeText = vi.fn(async () => {
      calls.push('clipboard');
    });
    const track = vi.fn(() => {
      calls.push('analytics');
      return true;
    });

    await copyInstallCommand('rein install', writeText, track);

    expect(writeText).toHaveBeenCalledWith('rein install');
    expect(track).toHaveBeenCalledWith(
      'install_command_copy',
      'homepage-hero',
    );
    expect(calls).toEqual(['clipboard', 'analytics']);
  });

  it('does not track a rejected clipboard write', async () => {
    const failure = new Error('clipboard permission denied');
    const track = vi.fn();

    await expect(
      copyInstallCommand(
        'rein install',
        async () => {
          throw failure;
        },
        track,
      ),
    ).rejects.toBe(failure);

    expect(track).not.toHaveBeenCalled();
  });

  it('keeps the button outside generic declarative click tracking', async () => {
    const componentPath = fileURLToPath(
      new URL('../components/landing/HeroExploded.astro', import.meta.url),
    );
    const source = await readFile(componentPath, 'utf8');
    const button = source.match(
      /<button\b[\s\S]*?\bid="copy-install"[\s\S]*?<\/button>/,
    )?.[0];

    expect(button).toBeDefined();
    expect(button).not.toContain('data-analytics-event');
    expect(button).not.toContain('data-analytics-target');
    expect(source).toContain('await copyInstallCommand(');
  });
});
