import { mkdir, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import sharp from 'sharp';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const assetRoot = path.join(repoRoot, 'assets');
const outputRoot = path.join(assetRoot, 'rebuilt-diagrams');
const websiteBrandRoot = path.join(repoRoot, 'website/public/brand');

const c = {
  bg: '#ffffff',
  paper: '#ffffff',
  ink: '#17231e',
  muted: '#5f6d67',
  rule: '#cfd7cf',
  ruleStrong: '#91a09a',
  lime: '#b8ff3c',
  limeDeep: '#4d8f18',
  purple: '#6657e8',
  violet: '#8a5cf1',
  cyan: '#28a9df',
  green: '#28bd94',
  orange: '#f3a11a',
  red: '#ef4d4d',
  slate: '#728399',
  dark: '#24323d',
};

function escapeXml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

function text(x, y, value, className = 'body', anchor = 'start', lineHeight = 28, extra = '') {
  const lines = Array.isArray(value) ? value : [value];
  const spans = lines
    .map(
      (line, index) =>
        `<tspan x="${x}" dy="${index === 0 ? 0 : lineHeight}">${escapeXml(line)}</tspan>`,
    )
    .join('');
  return `<text x="${x}" y="${y}" class="${className}" text-anchor="${anchor}" ${extra}>${spans}</text>`;
}

function line(x1, y1, x2, y2, options = {}) {
  const { dashed = false, arrow = false, color = c.ruleStrong, width = 2 } = options;
  return `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${color}" stroke-width="${width}"${dashed ? ' stroke-dasharray="9 8"' : ''}${arrow ? ' marker-end="url(#arrow)"' : ''} />`;
}

function circle(x, y, radius, fill, stroke = c.paper, strokeWidth = 3) {
  return `<circle cx="${x}" cy="${y}" r="${radius}" fill="${fill}" stroke="${stroke}" stroke-width="${strokeWidth}" />`;
}

function roundedNode(x, y, width, height, fill, title, details = [], options = {}) {
  const { titleClass = 'nodeTitle', detailClass = 'nodeDetail', border = c.paper } = options;
  const centerX = x + width / 2;
  const allLines = [title, ...details];
  const lineHeight = 27;
  const startY = y + height / 2 - ((allLines.length - 1) * lineHeight) / 2 + 7;
  return [
    `<rect x="${x}" y="${y}" width="${width}" height="${height}" rx="24" fill="${fill}" stroke="${border}" stroke-width="3" />`,
    text(centerX, startY, allLines, titleClass, 'middle', lineHeight),
  ].join('');
}

function svgDocument(width, height, titleValue, description, body) {
  return `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-labelledby="title description">
  <title id="title">${escapeXml(titleValue)}</title>
  <desc id="description">${escapeXml(description)}</desc>
  <defs>
    <marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="5" orient="auto" markerUnits="strokeWidth">
      <path d="M0,0 L10,5 L0,10 Z" fill="${c.ruleStrong}" />
    </marker>
  </defs>
  <style>
    text { font-family: Geist, Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif; fill: ${c.ink}; }
    .title { font-size: 42px; font-weight: 720; letter-spacing: -0.7px; }
    .subtitle { font-size: 27px; fill: ${c.muted}; }
    .body { font-size: 24px; }
    .small { font-size: 20px; fill: ${c.muted}; }
    .tiny { font-size: 17px; fill: ${c.muted}; }
    .axis { font-size: 23px; font-weight: 680; }
    .label { font-size: 23px; fill: ${c.muted}; }
    .strong { font-size: 24px; font-weight: 700; }
    .accent { font-size: 25px; font-weight: 740; fill: ${c.limeDeep}; }
    .nodeTitle, .nodeDetail { font-size: 23px; font-weight: 700; fill: #fff; }
    .nodeDetail { font-size: 21px; font-weight: 570; }
  </style>
  <rect width="${width}" height="${height}" fill="${c.bg}" />
  ${body}
</svg>`;
}

function buildLandscape() {
  const width = 2320;
  const height = 1320;
  const plot = { left: 140, right: 2240, top: 120, bottom: 1195 };
  const points = [
    [370, 215, 14, c.purple, 'Claude Code on web', 396, 223],
    [520, 315, 14, c.green, 'Codex CLI + ChatGPT sync', 547, 323],
    [790, 380, 14, c.violet, 'claude-sync', 817, 388],
    [645, 475, 14, c.cyan, 'antigravity-storage-manager', 672, 483],
    [750, 590, 14, c.purple, 'Anthropic Remote Control', 612, 560],
    [938, 700, 14, c.orange, 'Happy (mobile relay)', 826, 670],
    [1378, 455, 14, c.violet, 'coding-agent-sync', 1405, 463],
    [1190, 840, 14, '#a6b3c5', 'Git / Syncthing / NAS DIY', 1055, 808],
    [1462, 960, 14, '#a6b3c5', 'mcp-sync / MCP Manager', 1489, 968],
    [1673, 1045, 14, c.orange, 'Vibe Kanban / Conductor / Nimbalyst', 1476, 1014],
    [580, 1014, 14, c.green, 'OpenCode', 607, 1022],
    [392, 1067, 14, c.purple, 'Claude Code (local CLI)', 420, 1075],
    [476, 1120, 14, c.cyan, 'Gemini CLI', 504, 1128],
    [685, 1078, 14, '#4b5868', 'Grok Build', 713, 1086],
  ];

  let body = text(width / 2, 62, 'Where everyone sits: agent scope vs. state portability', 'title', 'middle');
  body += line(plot.left, plot.top, plot.left, plot.bottom, { color: c.ruleStrong });
  body += line(plot.left, plot.bottom, plot.right, plot.bottom, { color: c.ruleStrong });
  body += line(1190, plot.top, 1190, plot.bottom, { dashed: true, color: c.rule });
  body += line(plot.left, 660, plot.right, 660, { dashed: true, color: c.rule });
  for (const y of [335, 550, 770, 985]) body += line(plot.left - 10, y, plot.left, y, { color: c.ruleStrong });
  for (const x of [560, 980, 1400, 1820]) body += line(x, plot.bottom, x, plot.bottom + 10, { color: c.ruleStrong });

  body += text(180, 168, ['Native cloud sessions', '(vendor-locked)'], 'subtitle', 'start', 25, 'font-style="italic"');
  body += text(180, 1140, ['Local CLI agents', '(default today)'], 'subtitle', 'start', 25, 'font-style="italic"');
  body += text(1964, 1140, ['Cross-tool config sync', '(same machine only)'], 'subtitle', 'start', 25, 'font-style="italic"');
  body += text(1880, 175, ['THE GAP: universal,', 'portable agent state'], 'accent', 'start', 27);

  for (const [x, y, radius, fill, label, labelX, labelY] of points) {
    body += circle(x, y, radius, fill);
    body += text(labelX, labelY, label, 'label');
  }

  body += circle(1960, 245, 23, c.lime, c.ink, 4);
  body += text(1922, 253, 'Reinstate (this project)', 'accent', 'end');
  body += text(width / 2, 1275, 'Agent coverage  →  (single agent → universal)', 'axis', 'middle');
  body += `<text x="52" y="700" class="axis" text-anchor="middle" transform="rotate(-90 52 700)">State portability  →  (local-only → cloud-synced)</text>`;

  return svgDocument(
    width,
    height,
    'Agent scope versus state portability',
    'A positioning chart comparing coding-agent tools by agent coverage and state portability, with Reinstate highlighted in the universal portable-state gap.',
    body,
  );
}

function buildDemandTimeline() {
  const width = 2480;
  const height = 1120;
  const left = 300;
  const right = 2380;
  const top = 170;
  const bottom = 980;
  const months = ['Oct 2025', 'Nov 2025', 'Dec 2025', 'Jan 2026', 'Feb 2026', 'Mar 2026', 'Apr 2026', 'May 2026', 'Jun 2026', 'Jul 2026', 'Aug 2026'];
  const xAt = (monthIndex, fraction = 0) => left + ((monthIndex + fraction) / 11) * (right - left);
  const rows = [305, 575, 845];
  let body = text(width / 2, 62, "Users keep asking for this — on every vendor's own tracker", 'title', 'middle');
  body += text(
    width / 2,
    108,
    'Selected feature requests asking for cross-device session / state sync (Oct 2025 – Jul 2026)',
    'subtitle',
    'middle',
  );

  for (let i = 0; i <= 11; i += 1) body += line(xAt(i), top, xAt(i), bottom, { color: '#e4e9e4', width: 1.5 });
  for (const y of [top, 440, 710, bottom]) body += line(left, y, right, y, { color: '#e4e9e4', width: 1.5 });
  body += line(left, top, left, bottom, { color: c.ruleStrong });
  body += text(285, 295, ['Gemini + Antigravity', '(Google)'], 'axis', 'end', 32);
  body += text(285, 565, ['Codex', '(OpenAI)'], 'axis', 'end', 32);
  body += text(285, 835, ['Claude Code', '(Anthropic)'], 'axis', 'end', 32);
  months.forEach((month, index) => {
    body += text(xAt(index + 0.5), 1018, month, 'body', 'middle');
  });

  const events = [
    [xAt(1, 0.18), rows[0], c.cyan, ['#11758: auto-save', '/resume demand'], -5, -76],
    [xAt(5, 0.72), rows[0], c.cyan, ['Antigravity:', "'push to cloud'"], 0, -76],
    [xAt(7, 0.45), rows[1], c.green, ['#20757'], 0, -47],
    [xAt(7, 0.54), rows[1], c.green, ['#21079'], 0, 62],
    [xAt(10, 0.18), rows[1], c.green, ['discussion', '#14067'], 0, -75],
    [xAt(6, 0.28), rows[2], c.purple, ['#35906'], -12, -45],
    [xAt(6, 0.38), rows[2], c.purple, ['#37130'], 0, 58],
    [xAt(6, 0.72), rows[2], c.purple, ['#42219'], 0, -45],
    [xAt(6, 0.94), rows[2], c.purple, ['#45358'], 0, 58],
    [xAt(8, 0.2), rows[2], c.purple, ['#61967'], 0, -45],
    [xAt(8, 0.43), rows[2], c.purple, ['#64081'], 0, 58],
  ];
  for (const [x, y, fill, labels, dx, dy] of events) {
    body += circle(x, y, 17, fill);
    body += text(x + dx, y + dy, labels, 'label', 'middle', 28);
  }

  const legendY = 1076;
  const legend = [
    [760, c.purple, 'Claude Code (Anthropic)'],
    [1105, c.green, 'Codex (OpenAI)'],
    [1365, c.cyan, 'Gemini + Antigravity (Google)'],
  ];
  for (const [x, fill, label] of legend) {
    body += circle(x, legendY - 7, 13, fill);
    body += text(x + 34, legendY, label, 'body');
  }

  return svgDocument(
    width,
    height,
    'Cross-device session sync demand timeline',
    'A timeline of selected feature requests across Claude Code, Codex, Gemini, and Antigravity vendor trackers from October 2025 through July 2026.',
    body,
  );
}

function buildTraction() {
  const width = 2320;
  const height = 1120;
  const left = 500;
  const right = 2150;
  const top = 150;
  const bottom = 1040;
  const minLog = Math.log10(10);
  const maxLog = Math.log10(300000);
  const xFor = (value) => left + ((Math.log10(value) - minLog) / (maxLog - minLog)) * (right - left);
  const rows = [198, 297, 396, 495, 594, 693, 792, 891, 990];
  const bars = [
    ['Gemini CLI (agent CLI)', 106152, c.cyan],
    ['Codex CLI (agent CLI)', 101218, c.green],
    ['Vibe Kanban (orchestrator)', 27500, c.orange],
    ['Happy (mobile relay)', 22832, c.orange],
    ['Grok Build (agent CLI)', 22303, c.cyan],
    ['OpenCode (agent CLI)', 13527, c.cyan],
    ['grok-cli community (agent CLI)', 3335, c.cyan],
    ['claude-sync (sync tool)', 229, c.violet],
    ['mcp-sync (sync tool)', 46, c.violet],
  ];
  let body = text(width / 2, 62, 'Agent CLIs are massive — the sync layer is still up for grabs', 'title', 'middle');
  body += text(width / 2, 108, 'GitHub stars, fetched 2026-07-25 (log scale)', 'subtitle', 'middle');
  for (const tick of [10, 100, 1000, 10000, 100000, 300000]) {
    const x = xFor(tick);
    body += line(x, top, x, bottom, { color: '#e4e9e4', width: 1.5 });
    body += text(x, 1080, tick.toLocaleString('en-US'), 'body', 'middle');
  }
  body += line(left, top, left, bottom, { color: c.ruleStrong });
  bars.forEach(([label, value, fill], index) => {
    const y = rows[index];
    const barEnd = xFor(value);
    body += text(left - 18, y + 8, label, 'body', 'end');
    body += `<rect x="${left}" y="${y - 21}" width="${Math.max(8, barEnd - left)}" height="44" rx="8" fill="${fill}" />`;
    body += text(barEnd + 12, y + 8, value.toLocaleString('en-US'), 'strong');
  });

  return svgDocument(
    width,
    height,
    'Agent CLI and sync-tool GitHub traction',
    'A logarithmic horizontal bar chart comparing GitHub star counts for major agent CLIs, orchestrators, relays, and session sync tools as fetched on July 25, 2026.',
    body,
  );
}

function buildMarket() {
  const width = 2320;
  const height = 1160;
  const left = 140;
  const right = 2220;
  const top = 160;
  const bottom = 1018;
  const yFor = (value) => bottom - (value / 30) * (bottom - top);
  const xs = [348, 764, 1180, 1596, 2012];
  const anth = [[xs[0], 1], [xs[3], 14], [xs[4], 30]];
  const claude = [[xs[1], 0.1], [xs[2], 0.5], [xs[3], 2.5]];
  const cursor = [[xs[0], 0.2], [xs[3], 2]];
  const polyline = (points, color) =>
    `<polyline points="${points.map(([x, value]) => `${x},${yFor(value)}`).join(' ')}" fill="none" stroke="${color}" stroke-width="6" stroke-linejoin="round" stroke-linecap="round" />`;
  const pointSeries = (points, color, labels, offsets = []) =>
    points
      .map(([x, value], index) => {
        const y = yFor(value);
        const [dx = 0, dy = -25, anchor = 'middle'] = offsets[index] ?? [];
        return `${circle(x, y, 9, c.bg, color, 4)}${text(x + dx, y + dy, labels[index], 'strong', anchor)}`;
      })
      .join('');

  let body = text(width / 2, 62, 'The surface area is exploding: money and usage behind agent CLIs', 'title', 'middle');
  body += text(width / 2, 108, 'Annualized revenue run-rate, $B (reported figures)', 'subtitle', 'middle');
  body += text(58, 126, '$B annualized', 'axis');
  for (const tick of [0, 5, 10, 15, 20, 25, 30]) {
    const y = yFor(tick);
    body += line(left, y, right, y, { color: '#e4e9e4', width: 1.5 });
    body += text(left - 16, y + 7, tick, 'body', 'end');
  }
  body += line(left, bottom, right, bottom, { color: c.ruleStrong });
  ['End 2024', 'May 2025', 'Oct 2025', 'Feb 2026', 'Apr 2026'].forEach((label, index) => {
    body += text(xs[index], 1055, label, 'body', 'middle');
  });
  // The source chart intentionally has no connecting segment from the isolated
  // End 2024 Anthropic point to the later reported run-rate series.
  body += polyline(anth.slice(1), c.purple) + pointSeries(anth, c.purple, ['1', '14', '30']);
  body +=
    polyline(claude, c.orange) +
    pointSeries(claude, c.orange, ['0.1', '0.5', '2.5'], [
      [0, 42, 'middle'],
      [0, 42, 'middle'],
      [0, 42, 'middle'],
    ]);
  body +=
    pointSeries(cursor, '#9aa8bb', ['0.2', '2'], [
      [0, 42, 'middle'],
      [0, 42, 'middle'],
    ]);

  const legendY = 1122;
  const legend = [
    [710, c.purple, 'Anthropic (total, all products)'],
    [1120, c.orange, 'Claude Code only'],
    [1420, '#9aa8bb', 'Cursor (AI IDE)'],
  ];
  for (const [x, color, label] of legend) {
    body += line(x, legendY - 8, x + 42, legendY - 8, { color, width: 4 });
    body += circle(x + 21, legendY - 8, 10, c.bg, color, 4);
    body += text(x + 54, legendY, label, 'body');
  }

  return svgDocument(
    width,
    height,
    'Agent CLI market context',
    'A line chart of reported annualized revenue run-rate figures for Anthropic, Claude Code, and Cursor between the end of 2024 and April 2026.',
    body,
  );
}

function buildArchitecture() {
  const width = 2560;
  const height = 1360;
  let body = text(width / 2, 66, 'Reinstate MVP: one pipeline, every agent, any backend', 'title', 'middle');

  // Connections render first so they stay behind nodes, matching the original diagram.
  for (const [fromY, toX, toY] of [
    [250, 625, 575],
    [420, 625, 595],
    [590, 625, 615],
    [760, 625, 635],
    [930, 625, 655],
  ]) {
    body += line(405, fromY, toX, toY, { arrow: true, color: c.rule, width: 4 });
  }
  body += line(755, 320, 755, 535, { arrow: true, color: c.rule, width: 4 });
  body += line(970, 615, 1110, 615, { arrow: true, color: c.rule, width: 4 });
  body += line(1500, 615, 1620, 615, { arrow: true, color: c.rule, width: 4 });
  body += line(1815, 535, 1815, 390, { arrow: true, color: c.rule, width: 4 });
  body += line(2035, 295, 2100, 295, { arrow: true, color: c.rule, width: 4 });
  body += line(2295, 390, 2295, 690, { arrow: true, color: c.rule, width: 4 });
  body += line(2010, 650, 2110, 730, { arrow: true, color: c.rule, width: 4 });
  body += line(2210, 850, 1800, 1040, { arrow: true, color: c.rule, width: 4 });

  body += roundedNode(60, 195, 345, 110, c.purple, 'Claude Code', ['$HOME/.claude/projects/*.jsonl']);
  body += roundedNode(60, 365, 345, 110, c.green, 'Codex CLI', ['$HOME/.codex/sessions/.../rollout-*.jsonl']);
  body += roundedNode(60, 535, 345, 110, c.cyan, 'Gemini CLI', ['$HOME/.gemini/tmp/<hash>/chats/*.json']);
  body += roundedNode(60, 705, 345, 110, '#22b6a7', 'OpenCode', ['$HOME/.local/share/opencode/']);
  body += roundedNode(60, 875, 345, 110, c.dark, 'Grok Build', ['$HOME/.grok/ sessions']);
  body += roundedNode(520, 165, 470, 155, c.orange, 'REINSTATE CONFIG', [
    'MCP servers · skills · agents',
    'rules · settings',
  ]);
  body += roundedNode(625, 535, 345, 160, c.violet, '1 · WATCHER', [
    'fs events, debounce,',
    'append-aware diffs',
  ]);
  body += roundedNode(1110, 535, 390, 160, c.violet, '2 · NORMALIZER', [
    'path tokens (${HOME},',
    '${WORK}) · ignore rules',
    '· secret redaction',
  ]);
  body += roundedNode(1620, 535, 390, 160, c.red, '3 · ENCRYPT', [
    'age / AES-256-GCM,',
    'passphrase-derived keys',
  ]);
  body += roundedNode(1615, 230, 420, 160, c.orange, '4 · SYNC ENGINE', [
    'SQLite manifest · delta sync',
    '· conflict detection',
  ]);
  body += roundedNode(2100, 230, 400, 160, c.slate, 'CLOUD BACKENDS (BYO)', [
    'R2 · S3 · GCS · WebDAV · Gist',
    'ciphertext only',
  ]);
  body += roundedNode(2110, 690, 390, 160, c.purple, '5 · RESTORE (device B)', [
    'pull · decrypt · path-rewrite',
    '· atomic swap + backup',
  ]);
  body += roundedNode(1390, 1040, 590, 160, '#22b6a7', 'resume anywhere:', [
    'claude --resume · codex resume ·',
    'gemini --resume · opencode · grok -r',
  ]);

  body += text(72, 1260, 'Device A (write path)', 'subtitle');
  body += line(335, 1252, 1170, 1252, { arrow: true, color: c.ruleStrong, width: 2 });
  body += text(1200, 1260, 'cloud', 'subtitle');
  body += line(1270, 1252, 1435, 1252, { arrow: true, color: c.ruleStrong, width: 2 });
  body += text(1465, 1260, 'Device B (read path)', 'subtitle');
  body += text(72, 1310, 'Encryption happens before anything leaves the machine — backends never see plaintext.', 'subtitle');

  return svgDocument(
    width,
    height,
    'Reinstate architecture',
    'The Reinstate MVP pipeline watches agent state, normalizes paths, encrypts locally, syncs ciphertext to a user-selected backend, restores on another device, and resumes with the native agent.',
    body,
  );
}

const diagrams = [
  ['01_landscape', buildLandscape()],
  ['02_demand_timeline', buildDemandTimeline()],
  ['03_traction', buildTraction()],
  ['04_market', buildMarket()],
  ['05_architecture', buildArchitecture()],
];

async function comparisonSheet(name, originalPath, rebuiltPath) {
  const panelWidth = 760;
  const panelHeight = 470;
  const original = await sharp(originalPath)
    .resize(panelWidth, panelHeight, { fit: 'contain', background: '#ffffff' })
    .png()
    .toBuffer();
  const rebuilt = await sharp(rebuiltPath)
    .resize(panelWidth, panelHeight, { fit: 'contain', background: c.bg })
    .png()
    .toBuffer();
  const labels = Buffer.from(`
    <svg xmlns="http://www.w3.org/2000/svg" width="1600" height="540">
      <style>text { font-family: Arial, sans-serif; fill: ${c.ink}; font-size: 24px; font-weight: 700; }</style>
      <text x="20" y="36">Original PNG</text>
      <text x="820" y="36">Rebuilt SVG render</text>
    </svg>
  `);
  await sharp({ create: { width: 1600, height: 540, channels: 3, background: c.paper } })
    .composite([
      { input: original, left: 20, top: 50 },
      { input: rebuilt, left: 820, top: 50 },
      { input: labels, left: 0, top: 0 },
    ])
    .png()
    .toFile(path.join(outputRoot, `compare_${name}.png`));
}

async function main() {
  await mkdir(outputRoot, { recursive: true });
  await mkdir(websiteBrandRoot, { recursive: true });
  for (const [name, svg] of diagrams) {
    const svgPath = path.join(outputRoot, `${name}.svg`);
    const pngPath = path.join(outputRoot, `${name}.png`);
    await writeFile(svgPath, svg, 'utf8');
    await sharp(Buffer.from(svg)).png().toFile(pngPath);
    await writeFile(path.join(assetRoot, `${name}.svg`), svg, 'utf8');
    await writeFile(path.join(websiteBrandRoot, `${name}.svg`), svg, 'utf8');
    await sharp(Buffer.from(svg)).png().toFile(path.join(websiteBrandRoot, `${name}.png`));
    await comparisonSheet(name, path.join(assetRoot, `${name}.png`), pngPath);
  }

  const sections = diagrams
    .map(
      ([name]) => `
        <section>
          <h2>${name}</h2>
          <div class="comparison">
            <figure><img src="../${name}.png" alt="Original ${name} PNG"><figcaption>Original PNG</figcaption></figure>
            <figure><img src="${name}.png" alt="Rebuilt ${name} SVG render"><figcaption>Rebuilt SVG render</figcaption></figure>
          </div>
          <p><a href="${name}.svg">Open editable SVG</a> · <a href="compare_${name}.png">Open comparison sheet</a></p>
        </section>`,
    )
    .join('');
  const reviewHtml = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Reinstate diagram rebuild review</title>
<style>
  :root{font-family:Arial,sans-serif;color:${c.ink};background:${c.bg}}body{max-width:1500px;margin:0 auto;padding:32px}
  h1{font-size:36px}section{padding:28px 0;border-top:1px solid ${c.rule}}.comparison{display:grid;grid-template-columns:1fr 1fr;gap:24px}
  figure{margin:0}img{display:block;width:100%;height:auto;border:1px solid ${c.rule};background:#fff}figcaption{margin-top:8px;color:${c.muted}}
  a{color:${c.limeDeep};font-weight:700}@media(max-width:800px){.comparison{grid-template-columns:1fr}}
</style></head><body><h1>Reinstate diagram rebuild review</h1><p>The rebuilt SVGs are now the canonical documentation assets. The original PNGs remain on the left for comparison.</p>${sections}</body></html>`;
  await writeFile(path.join(outputRoot, 'review.html'), reviewHtml, 'utf8');
}

await main();
