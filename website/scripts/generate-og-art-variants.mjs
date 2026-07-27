import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import process from 'node:process';
import puppeteer from 'puppeteer-core';
import sharp from 'sharp';

const root = resolve(import.meta.dirname, '..');
const outputDirectory = resolve(root, 'src/assets/og-art');
const check = process.argv.includes('--check');

function argument(name, fallback) {
  const index = process.argv.indexOf(name);
  return index === -1 ? fallback : process.argv[index + 1];
}

const baseUrl = argument(
  '--base-url',
  process.env.REINSTATE_OG_BASE_URL ?? 'http://127.0.0.1:4321',
);
const executablePath =
  argument('--chrome-path', process.env.REINSTATE_CHROME_PATH) ??
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

const variants = [
  {
    name: 'session-stack',
    selector: '.problem-section .panel.good .cardart svg',
    crop: { left: 0, top: 0, width: 1, height: 1 },
    width: 900,
    source: 'ProblemExploded.astro — live “With Reinstate” portable-state card',
  },
  {
    name: 'stranded-workstation',
    selector: '.problem-section > .scene > svg',
    crop: { left: 0, top: 0, width: 1, height: 1 },
    width: 1_200,
    safeInset: 48,
    source: 'ProblemExploded.astro — planning corner and failed-workaround scene',
  },
  {
    name: 'device-handoff',
    selector: '.handoff-art',
    crop: { left: 0, top: 0, width: 1, height: 1 },
    width: 1_200,
    source: 'TerminalProof.astro — Windows to encrypted checkpoint to MacBook',
  },
  {
    name: 'local-encryption',
    selector: '.sec-vault-art',
    crop: { left: 0.23, top: 0.02, width: 0.29, height: 0.94 },
    width: 760,
    stageIndex: 1,
    source: 'SecurityVaultArt.astro — local encryption stage',
  },
  {
    name: 'owned-storage',
    selector: '.sec-vault-art',
    crop: { left: 0.48, top: 0.02, width: 0.29, height: 0.94 },
    width: 760,
    stageIndex: 2,
    source: 'SecurityVaultArt.astro — user-owned S3-compatible storage stage',
  },
];

function cropPixels(metadata, crop) {
  const width = metadata.width ?? 0;
  const height = metadata.height ?? 0;
  return {
    left: Math.round(width * crop.left),
    top: Math.round(height * crop.top),
    width: Math.min(Math.round(width * crop.width), width - Math.round(width * crop.left)),
    height: Math.min(
      Math.round(height * crop.height),
      height - Math.round(height * crop.top),
    ),
  };
}

async function capture(page, variant) {
  const element = await page.$(variant.selector);
  if (!element) {
    throw new Error(`Could not find landing-page art: ${variant.selector}`);
  }

  if (variant.stageIndex !== undefined) {
    await element.evaluate((svg, stageIndex) => {
      const stages = [...svg.querySelectorAll('.stage')];
      const labels = [...svg.querySelectorAll('.lab')];
      stages.forEach((stage, index) => {
        stage.style.visibility = index === stageIndex ? 'visible' : 'hidden';
      });
      labels.forEach((label, index) => {
        label.style.visibility =
          Math.floor(index / 2) === stageIndex ? 'visible' : 'hidden';
      });
    }, variant.stageIndex);
  }

  let screenshot;
  try {
    screenshot = await element.screenshot({
      type: 'png',
      omitBackground: true,
    });
  } finally {
    if (variant.stageIndex !== undefined) {
      await element.evaluate((svg) => {
        for (const element of svg.querySelectorAll('.stage, .lab')) {
          element.style.removeProperty('visibility');
        }
      });
    }
  }
  const metadata = await sharp(screenshot).metadata();
  const extracted = await sharp(screenshot)
    .extract(cropPixels(metadata, variant.crop))
    .png()
    .toBuffer();
  const inset = variant.safeInset ?? 0;

  return sharp(extracted)
    .extend({
      top: inset,
      right: inset,
      bottom: inset,
      left: inset,
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    })
    .resize({ width: variant.width, withoutEnlargement: false })
    .png({ compressionLevel: 9, palette: true })
    .toBuffer();
}

async function visuallyMatches(generated, committed) {
  if (generated.equals(committed)) return true;

  const [next, current] = await Promise.all([
    sharp(generated)
      .flatten({ background: '#e4e7dd' })
      .raw()
      .toBuffer({ resolveWithObject: true }),
    sharp(committed)
      .flatten({ background: '#e4e7dd' })
      .raw()
      .toBuffer({ resolveWithObject: true }),
  ]);
  if (
    next.info.width !== current.info.width ||
    next.info.height !== current.info.height ||
    next.info.channels !== current.info.channels
  ) {
    return false;
  }

  const channels = next.info.channels;
  let materiallyChangedPixels = 0;
  for (let index = 0; index < next.data.length; index += channels) {
    let largestChannelDelta = 0;
    for (let channel = 0; channel < channels; channel += 1) {
      largestChannelDelta = Math.max(
        largestChannelDelta,
        Math.abs(next.data[index + channel] - current.data[index + channel]),
      );
    }
    if (largestChannelDelta > 32) materiallyChangedPixels += 1;
  }

  const totalPixels = next.info.width * next.info.height;
  return materiallyChangedPixels / totalPixels <= 0.0005;
}

const browser = await puppeteer.launch({
  executablePath,
  headless: true,
});

try {
  const page = await browser.newPage();
  await page.setViewport({ width: 1_600, height: 1_000, deviceScaleFactor: 2 });
  await page.emulateMediaFeatures([
    { name: 'prefers-reduced-motion', value: 'reduce' },
    { name: 'prefers-color-scheme', value: 'light' },
  ]);
  await page.goto(baseUrl, { waitUntil: 'networkidle0' });
  await page.addStyleTag({
    content: `
      *,
      *::before,
      *::after {
        animation: none !important;
        transition: none !important;
      }
      .nav,
      .hero-block > .lede,
      .problem-section > .lede,
      .floor-grid,
      .sec-vault-art .routes,
      .sec-vault-art .route-label {
        visibility: hidden !important;
      }
      html,
      body,
      .homepage,
      .problem-block,
      .problem-section,
      .problem-section .panel,
      .problem-section .cardart,
      .terminal-proof,
      .handoff-stage,
      .handoff-illustration,
      .security-section,
      .security-scene,
      .sec-vault,
      .sec-vault-scene {
        background: transparent !important;
        background-image: none !important;
        box-shadow: none !important;
      }
    `,
  });
  await page.evaluate(async () => {
    document.documentElement.style.background = 'transparent';
    document.body.style.background = 'transparent';
    await document.fonts.ready;
  });

  for (const variant of variants) {
    const output = resolve(outputDirectory, `${variant.name}.png`);
    const generated = await capture(page, variant);

    if (check) {
      const committed = await readFile(output);
      if (!(await visuallyMatches(generated, committed))) {
        throw new Error(`${variant.name}.png is stale; run npm run generate:og-art`);
      }
      continue;
    }

    await mkdir(dirname(output), { recursive: true });
    await writeFile(output, generated);
    console.log(`generated ${variant.name}.png from ${variant.source}`);
  }
} finally {
  await browser.close();
}
