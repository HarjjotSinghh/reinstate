#!/usr/bin/env node

import assert from 'node:assert/strict';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { launch } from 'chrome-launcher';
import puppeteer from 'puppeteer-core';

import { createStaticPreviewServer } from './serve-static-build.mjs';

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const BUILD_ROOT = resolve(process.cwd(), 'dist/client');
const ROUTE = '/tools/path-mapping-visualizer';

const scenarios = [
  {
    direction: 'mac-to-windows',
    agent: 'claude',
    sourceRoot: '/Users/alex/Code/acme-app',
    destinationRoot: 'C:\\src\\acme-app',
    destinationValue: 'C:\\src\\acme-app\\src\\main.go',
    recognizedField: 'cwd',
    unchangedProse: 'Open /Users/alex/notes.txt',
  },
  {
    direction: 'mac-to-windows',
    agent: 'codex',
    sourceRoot: '/Users/alex/Code/acme-app',
    destinationRoot: 'C:\\src\\acme-app',
    destinationValue: 'C:\\src\\acme-app\\src\\main.go',
    recognizedField: 'session_meta.payload.cwd',
    unchangedProse: 'Open /Users/alex/notes.txt',
  },
  {
    direction: 'windows-to-mac',
    agent: 'claude',
    sourceRoot: 'C:\\src\\acme-app',
    destinationRoot: '/Users/alex/Code/acme-app',
    destinationValue: '/Users/alex/Code/acme-app/src/main.go',
    recognizedField: 'cwd',
    unchangedProse: 'Open C:\\Users\\alex\\notes.txt',
  },
  {
    direction: 'windows-to-mac',
    agent: 'codex',
    sourceRoot: 'C:\\src\\acme-app',
    destinationRoot: '/Users/alex/Code/acme-app',
    destinationValue: '/Users/alex/Code/acme-app/src/main.go',
    recognizedField: 'session_meta.payload.cwd',
    unchangedProse: 'Open C:\\Users\\alex\\notes.txt',
  },
];

function listen(server) {
  return new Promise((resolveListen, rejectListen) => {
    server.once('error', rejectListen);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      if (!address || typeof address === 'string') {
        rejectListen(new Error('Static preview did not expose a TCP port.'));
        return;
      }
      resolveListen(address.port);
    });
  });
}

function closeServer(server) {
  return new Promise((resolveClose, rejectClose) => {
    server.close((error) => (error ? rejectClose(error) : resolveClose()));
  });
}

async function browserState(page) {
  const storage = await page.evaluate(() => ({
    local: Object.fromEntries(
      Array.from({ length: localStorage.length }, (_, index) => {
        const key = localStorage.key(index) ?? '';
        return [key, localStorage.getItem(key)];
      }),
    ),
    session: Object.fromEntries(
      Array.from({ length: sessionStorage.length }, (_, index) => {
        const key = sessionStorage.key(index) ?? '';
        return [key, sessionStorage.getItem(key)];
      }),
    ),
  }));
  const cookies = await page.cookies();
  return { storage, cookies };
}

async function visibleOutput(page) {
  return page.evaluate(() => {
    const text = (selector) =>
      document.querySelector(selector)?.textContent?.trim() ?? '';
    return {
      sourceRoot: text('#source-root'),
      destinationRoot: text('#destination-root'),
      destinationValue: text('#destination-value'),
      recognizedField: text('#recognized-field'),
      unchangedProse: text('#unchanged-prose'),
      portableValue: text('#portable-value'),
      recordBefore: text('#record-before'),
      recordPortable: text('#record-portable'),
      recordAfter: text('#record-after'),
      nativeLocator: text('#native-locator'),
    };
  });
}

export async function checkPathVisualizer({
  buildRoot = BUILD_ROOT,
  viewport = { width: 390, height: 844 },
} = {}) {
  const server = createStaticPreviewServer({ buildRoot });
  let chrome;
  let browser;

  try {
    const port = await listen(server);
    chrome = await launch({
      chromeFlags: [
        '--headless=new',
        '--disable-dev-shm-usage',
        '--disable-gpu',
        '--no-sandbox',
      ],
    });
    browser = await puppeteer.connect({
      browserURL: `http://127.0.0.1:${chrome.port}`,
    });
    const page = await browser.newPage();
    await page.setViewport(viewport);

    const response = await page.goto(`http://127.0.0.1:${port}${ROUTE}`, {
      waitUntil: 'networkidle0',
    });
    assert.equal(response?.status(), 200, `${ROUTE} must return 200`);
    assert.equal(
      await page.$$eval('input, textarea, [contenteditable="true"]', (nodes) => nodes.length),
      0,
      'The visualizer must not expose free-text input.',
    );
    assert.equal(
      await page.evaluate(
        () =>
          typeof window.reinstateAnalytics === 'undefined' &&
          typeof window.plausible === 'undefined' &&
          document.querySelector('script[data-domain]') === null,
      ),
      true,
      'The visualizer route must not render analytics.',
    );

    const before = await browserState(page);
    const interactionRequests = [];
    let trackingInteraction = false;
    page.on('request', (request) => {
      if (trackingInteraction) interactionRequests.push(request.url());
    });

    trackingInteraction = true;
    for (const scenario of scenarios) {
      await page.select('#path-direction', scenario.direction);
      await page.select('#path-agent', scenario.agent);
      await page.evaluate(
        () => new Promise((resolveFrame) => requestAnimationFrame(() => resolveFrame())),
      );

      const output = await visibleOutput(page);
      assert.equal(output.sourceRoot, scenario.sourceRoot);
      assert.equal(output.destinationRoot, scenario.destinationRoot);
      assert.equal(output.destinationValue, scenario.destinationValue);
      assert.equal(output.recognizedField, scenario.recognizedField);
      assert.equal(output.unchangedProse, scenario.unchangedProse);
      assert.equal(
        output.portableValue,
        '${REPO:github.com/acme/acme-app}/src/main.go',
      );
      assert.match(output.recordBefore, /"cwd"/);
      assert.match(output.recordPortable, /\$\{REPO:github\.com\/acme\/acme-app\}/);
      assert.match(output.recordAfter, /"cwd"/);
      assert.match(
        output.nativeLocator,
        scenario.agent === 'claude' ? /Claude Code restore/ : /Codex keeps the rollout/,
      );
    }
    trackingInteraction = false;

    const after = await browserState(page);
    assert.deepEqual(
      interactionRequests,
      [],
      `Changing fixed controls must make no requests; saw ${interactionRequests.join(', ')}`,
    );
    assert.deepEqual(after, before, 'Changing fixed controls must not persist browser state.');
    assert.equal(new URL(page.url()).pathname, ROUTE);

    return {
      route: ROUTE,
      scenarios: scenarios.length,
      interactionRequests: interactionRequests.length,
      persistenceChanges: 0,
    };
  } finally {
    if (browser) await browser.disconnect();
    if (chrome) await chrome.kill();
    if (server.listening) await closeServer(server);
  }
}

export function formatResult(result) {
  return `Path visualizer browser check passed: ${result.scenarios} fixed scenarios, ${result.interactionRequests} interaction requests, and ${result.persistenceChanges} persistence changes.`;
}

async function main() {
  const result = await checkPathVisualizer();
  process.stdout.write(`${formatResult(result)}\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(SCRIPT_PATH)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
