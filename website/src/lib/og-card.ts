import { createRequire } from 'node:module';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import satori from 'satori';
import sharp from 'sharp';
import { product } from '../data/product.ts';
import type { OgPage } from '../data/og-pages.ts';
import {
  ogArtVariantForRoute,
  type OgArtVariant,
} from './og-art.ts';

const require = createRequire(import.meta.url);
let fonts:
  | {
      questrial: Buffer;
      geist: Buffer;
      geistMedium: Buffer;
    }
  | undefined;

function loadFonts() {
  fonts ??= {
    questrial: readFileSync(
      require.resolve('@fontsource/questrial/files/questrial-latin-400-normal.woff'),
    ),
    geist: readFileSync(
      require.resolve('@fontsource/geist/files/geist-latin-400-normal.woff'),
    ),
    geistMedium: readFileSync(
      require.resolve('@fontsource/geist/files/geist-latin-600-normal.woff'),
    ),
  };
  return fonts;
}

interface SatoriNode {
  type: string;
  props: Record<string, unknown>;
}

type NodeChild = string | number | SatoriNode;

function node(
  type: string,
  props: Record<string, unknown> = {},
  ...children: (NodeChild | NodeChild[] | undefined)[]
): SatoriNode {
  const style =
    type === 'div'
      ? {
          display: 'flex',
          ...((props.style as Record<string, unknown> | undefined) ?? {}),
        }
      : props.style;
  return {
    type,
    props: {
      ...props,
      ...(style ? { style } : {}),
      children: children.flat().filter((child) => child !== undefined),
    },
  };
}

function svgDataUri(svg: string): string {
  return `data:image/svg+xml;base64,${Buffer.from(svg).toString('base64')}`;
}

function ogArtDataUri(filename: string): string {
  const candidates = [
    resolve(process.cwd(), 'src/assets/og-art', filename),
    resolve(process.cwd(), 'website/src/assets/og-art', filename),
    resolve(import.meta.dirname, '../assets/og-art', filename),
  ];
  const source = candidates.find((candidate) => existsSync(candidate));
  if (!source) {
    throw new Error(`Open Graph artwork source not found: ${filename}`);
  }
  return `data:image/png;base64,${readFileSync(source).toString('base64')}`;
}

const logo = svgDataUri(`
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
    <rect x="10" y="10" width="32" height="32" rx="6" fill="none" stroke="#131f1a" stroke-width="4"/>
    <rect x="20" y="20" width="32" height="32" rx="6" fill="#e4e7dd" stroke="#131f1a" stroke-width="4"/>
    <path d="M28.5 31 L34 36.5 L28.5 42" fill="none" stroke="#131f1a" stroke-width="3.2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M37.5 42 H45" fill="none" stroke="#131f1a" stroke-width="3.2" stroke-linecap="round"/>
  </svg>
`);

const ogArtByVariant: Record<OgArtVariant, string> = {
  'session-stack': ogArtDataUri('session-stack.png'),
  'stranded-workstation': ogArtDataUri('stranded-workstation.png'),
  'device-handoff': ogArtDataUri('device-handoff.png'),
  'local-encryption': ogArtDataUri('local-encryption.png'),
  'owned-storage': ogArtDataUri('owned-storage.png'),
};

const ogArtPlacement: Record<
  OgArtVariant,
  { width: number; height: number; right: number; bottom: number }
> = {
  'session-stack': { width: 350, height: 212, right: 20, bottom: 128 },
  'stranded-workstation': { width: 410, height: 206, right: 12, bottom: 112 },
  'device-handoff': { width: 460, height: 141, right: 12, bottom: 120 },
  'local-encryption': { width: 300, height: 322, right: 28, bottom: 102 },
  'owned-storage': { width: 300, height: 322, right: 28, bottom: 102 },
};

export const repositorySocialPreview = {
  width: 1280,
  height: 640,
  format: 'png',
  maxBytes: 1_000_000,
} as const;

function satoriFonts(loadedFonts: ReturnType<typeof loadFonts>) {
  return [
    {
      name: 'Questrial',
      data: loadedFonts.questrial,
      weight: 400 as const,
      style: 'normal' as const,
    },
    {
      name: 'Geist',
      data: loadedFonts.geist,
      weight: 400 as const,
      style: 'normal' as const,
    },
    {
      name: 'Geist',
      data: loadedFonts.geistMedium,
      weight: 600 as const,
      style: 'normal' as const,
    },
  ];
}

async function renderPng(
  element: SatoriNode,
  width: number,
  height: number,
): Promise<Buffer> {
  const svg = await satori(element as never, {
    width,
    height,
    fonts: satoriFonts(loadFonts()),
  });

  return sharp(Buffer.from(svg)).png({ compressionLevel: 9 }).toBuffer();
}

function routeLabel(route: string): string {
  if (route === '/') return 'reinstate.dev';
  return `reinstate.dev${route}`;
}

function titleSize(title: string): number {
  if (title.length > 68) return 54;
  if (title.length > 48) return 62;
  return 70;
}

function estimatedLineCount(
  value: string,
  maxWidth: number,
  fontSize: number,
): number {
  const averageGlyphWidth = fontSize * 0.44;
  return Math.max(1, Math.ceil((value.length * averageGlyphWidth) / maxWidth));
}

function contentTop(page: OgPage): number {
  const headingSize = titleSize(page.title);
  const headingHeight =
    estimatedLineCount(page.title, 760, headingSize) * headingSize * 0.98;
  const descriptionHeight =
    estimatedLineCount(page.description, 690, 23) * 23 * 1.35;
  const blockHeight = 34 + 18 + headingHeight + 19 + descriptionHeight;

  return Math.max(178, Math.min(222, Math.round(455 - blockHeight)));
}

export interface RenderOgCardOptions {
  artVariant?: OgArtVariant;
}

export async function renderOgCard(
  page: OgPage,
  options: RenderOgCardOptions = {},
): Promise<Buffer> {
  const artVariant = options.artVariant ?? ogArtVariantForRoute(page.route);
  const artPlacement = ogArtPlacement[artVariant];
  const element = node(
    'div',
    {
      style: {
        width: '1200px',
        height: '630px',
        display: 'flex',
        position: 'relative',
        overflow: 'hidden',
        background: '#e4e7dd',
        color: '#131f1a',
        fontFamily: 'Geist',
      },
    },
    node('div', {
      style: {
        position: 'absolute',
        left: '-110px',
        bottom: '-285px',
        width: '920px',
        height: '520px',
        background: '#d3d8c9',
        transform: 'rotate(30deg)',
      },
    }),
    node('div', {
      style: {
        position: 'absolute',
        top: '-110px',
        right: '-100px',
        width: '460px',
        height: '260px',
        background: '#b8ff3c',
        border: '4px solid #131f1a',
        transform: 'rotate(-30deg)',
      },
    }),
    node(
      'div',
      {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          position: 'relative',
        },
      },
      node(
        'div',
        {
          style: {
            display: 'flex',
            position: 'absolute',
            top: '54px',
            left: '62px',
            alignItems: 'center',
            fontFamily: 'Questrial',
            fontSize: '32px',
            letterSpacing: '-1px',
          },
        },
        node('img', {
          src: logo,
          width: 54,
          height: 54,
          style: { marginRight: '12px' },
        }),
        'Reinstate',
      ),
      node(
        'div',
        {
          style: {
            width: '760px',
            display: 'flex',
            position: 'absolute',
            top: `${contentTop(page)}px`,
            left: '62px',
            flexDirection: 'column',
          },
        },
        node(
          'div',
          {
            style: {
              display: 'flex',
              alignSelf: 'flex-start',
              padding: '7px 12px',
              border: '2px solid #131f1a',
              borderRadius: '7px',
              background: '#f4f6ef',
              color: '#3e5148',
              fontSize: '15px',
              fontWeight: 600,
              letterSpacing: '1.5px',
              textTransform: 'uppercase',
            },
          },
          page.kind,
        ),
        node(
          'div',
          {
            style: {
              display: 'flex',
              marginTop: '18px',
              fontFamily: 'Questrial',
              fontSize: `${titleSize(page.title)}px`,
              lineHeight: 0.98,
              letterSpacing: '-2.2px',
              maxWidth: '760px',
            },
          },
          page.title,
        ),
        node(
          'div',
          {
            style: {
              display: 'flex',
              marginTop: '19px',
              maxWidth: '690px',
              color: '#415049',
              fontSize: '23px',
              lineHeight: 1.35,
            },
          },
          page.description,
        ),
      ),
      node(
        'div',
        {
          style: {
            display: 'flex',
            position: 'absolute',
            right: '62px',
            bottom: '46px',
            left: '62px',
            alignItems: 'center',
            justifyContent: 'space-between',
            color: '#53635b',
            fontSize: '16px',
          },
        },
        node(
          'div',
          {
            style: {
              display: 'flex',
              minWidth: 0,
              maxWidth: '760px',
              flexShrink: 1,
            },
          },
          routeLabel(page.route),
        ),
        node(
          'div',
          {
            style: {
              display: 'flex',
              flexShrink: 0,
              marginLeft: '20px',
              padding: '7px 11px',
              border: '2px solid #131f1a',
              borderRadius: '7px',
              background: '#f4f6ef',
              color: '#131f1a',
              fontWeight: 600,
            },
          },
          `${product.currentRelease} · Apache-2.0`,
        ),
      ),
    ),
    node('img', {
      src: ogArtByVariant[artVariant],
      width: artPlacement.width,
      height: artPlacement.height,
      style: {
        position: 'absolute',
        right: `${artPlacement.right}px`,
        bottom: `${artPlacement.bottom}px`,
      },
    }),
  );

  return renderPng(element, 1200, 630);
}

export async function renderRepositorySocialPreview(): Promise<Buffer> {
  const { width, height } = repositorySocialPreview;
  const element = node(
    'div',
    {
      style: {
        width: `${width}px`,
        height: `${height}px`,
        display: 'flex',
        position: 'relative',
        overflow: 'hidden',
        background: '#e4e7dd',
        color: '#131f1a',
        fontFamily: 'Geist',
      },
    },
    node('div', {
      style: {
        position: 'absolute',
        left: '-100px',
        bottom: '-300px',
        width: '980px',
        height: '540px',
        background: '#d3d8c9',
        transform: 'rotate(30deg)',
      },
    }),
    node('div', {
      style: {
        position: 'absolute',
        top: '-114px',
        right: '-104px',
        width: '480px',
        height: '272px',
        background: '#b8ff3c',
        border: '4px solid #131f1a',
        transform: 'rotate(-30deg)',
      },
    }),
    node(
      'div',
      {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: '52px 64px 46px',
          position: 'relative',
        },
      },
      node(
        'div',
        {
          style: {
            display: 'flex',
            alignItems: 'center',
            fontFamily: 'Questrial',
            fontSize: '34px',
            letterSpacing: '-1.1px',
          },
        },
        node('img', {
          src: logo,
          width: 56,
          height: 56,
          style: { marginRight: '13px' },
        }),
        'Reinstate',
      ),
      node(
        'div',
        {
          style: {
            width: '770px',
            display: 'flex',
            flexDirection: 'column',
            marginTop: '-2px',
          },
        },
        node(
          'div',
          {
            style: {
              display: 'flex',
              alignSelf: 'flex-start',
              padding: '7px 12px',
              border: '2px solid #131f1a',
              borderRadius: '7px',
              background: '#f4f6ef',
              color: '#3e5148',
              fontSize: '15px',
              fontWeight: 600,
              letterSpacing: '1.5px',
              textTransform: 'uppercase',
            },
          },
          'Open source · Apache-2.0',
        ),
        node(
          'div',
          {
            style: {
              display: 'flex',
              marginTop: '19px',
              maxWidth: '760px',
              fontFamily: 'Questrial',
              fontSize: '70px',
              lineHeight: 0.98,
              letterSpacing: '-2.2px',
            },
          },
          'Continuity layer for coding-agent work',
        ),
        node(
          'div',
          {
            style: {
              display: 'flex',
              marginTop: '20px',
              maxWidth: '700px',
              color: '#415049',
              fontSize: '24px',
              lineHeight: 1.34,
            },
          },
          'Sync encrypted Claude Code and Codex sessions across devices through storage you control.',
        ),
      ),
      node(
        'div',
        {
          style: {
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            color: '#53635b',
            fontSize: '17px',
          },
        },
        node('div', { style: { display: 'flex' } }, 'reinstate.dev'),
        node(
          'div',
          {
            style: {
              display: 'flex',
              padding: '7px 11px',
              border: '2px solid #131f1a',
              borderRadius: '7px',
              background: '#f4f6ef',
              color: '#131f1a',
              fontWeight: 600,
            },
          },
          `${product.currentRelease} · pre-1.0`,
        ),
      ),
    ),
    node('img', {
      src: ogArtByVariant['device-handoff'],
      width: 500,
      height: 154,
      style: {
        position: 'absolute',
        right: '16px',
        bottom: '106px',
      },
    }),
  );

  return renderPng(element, width, height);
}
