import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';
import satori from 'satori';
import sharp from 'sharp';
import { product } from '../data/product';
import type { OgPage } from '../data/og-pages';

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

const logo = svgDataUri(`
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
    <rect x="10" y="10" width="32" height="32" rx="6" fill="none" stroke="#131f1a" stroke-width="4"/>
    <rect x="20" y="20" width="32" height="32" rx="6" fill="#e4e7dd" stroke="#131f1a" stroke-width="4"/>
    <path d="M28.5 31 L34 36.5 L28.5 42" fill="none" stroke="#131f1a" stroke-width="3.2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M37.5 42 H45" fill="none" stroke="#131f1a" stroke-width="3.2" stroke-linecap="round"/>
  </svg>
`);

const continuityArt = svgDataUri(`
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 420 470">
    <g fill="none" stroke="#131f1a" stroke-width="4" stroke-linejoin="round">
      <path d="M40 392 196 302 380 408 224 498Z" fill="#d3d8c9"/>
      <path d="M70 337 194 266 348 354 224 426Z" fill="#26382f"/>
      <path d="M70 337 224 426 224 450 70 361Z" fill="#1d2b25"/>
      <path d="M224 426 348 354 348 378 224 450Z" fill="#131f1a"/>

      <path d="M118 276 203 227 310 289 225 338Z" fill="#f4f6ef"/>
      <path d="M118 276 225 338 225 354 118 292Z" fill="#d8ddd0"/>
      <path d="M225 338 310 289 310 305 225 354Z" fill="#bdc5b6"/>

      <path d="M101 225 203 166 330 239 228 298Z" fill="#7ecdf5"/>
      <path d="M101 225 228 298 228 314 101 241Z" fill="#5fa9d0"/>
      <path d="M228 298 330 239 330 255 228 314Z" fill="#4388ad"/>

      <path d="M85 170 201 103 346 187 230 254Z" fill="#ffce4a"/>
      <path d="M85 170 230 254 230 271 85 187Z" fill="#dba631"/>
      <path d="M230 254 346 187 346 204 230 271Z" fill="#b57e1f"/>

      <path d="M68 110 200 34 365 129 233 206Z" fill="#b8ff3c"/>
      <path d="M68 110 233 206 233 224 68 128Z" fill="#86c92d"/>
      <path d="M233 206 365 129 365 147 233 224Z" fill="#57901f"/>

      <path d="M128 98 200 57 291 109 219 151Z" fill="#1d2723"/>
      <path d="M154 99 178 113 164 121 140 107Z" stroke="#b8ff3c" stroke-width="5" stroke-linecap="round"/>
      <path d="M184 129 226 153" stroke="#b8ff3c" stroke-width="5" stroke-linecap="round"/>
    </g>
    <path d="M28 310 C0 210 34 91 125 44" fill="none" stroke="#4f8a1e" stroke-width="5" stroke-linecap="round" stroke-dasharray="8 12"/>
    <path d="m113 41 20-4-8 19" fill="#4f8a1e"/>
  </svg>
`);

function routeLabel(route: string): string {
  if (route === '/') return 'reinstate.dev';
  return `reinstate.dev${route}`;
}

function titleSize(title: string): number {
  if (title.length > 68) return 54;
  if (title.length > 48) return 62;
  return 70;
}

export async function renderOgCard(page: OgPage): Promise<Buffer> {
  const loadedFonts = loadFonts();
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
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: '54px 62px 46px',
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
            flexDirection: 'column',
            marginTop: '-6px',
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
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            color: '#53635b',
            fontSize: '16px',
          },
        },
        node('div', { style: { display: 'flex' } }, routeLabel(page.route)),
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
          `${product.currentRelease} · Apache-2.0`,
        ),
      ),
    ),
    node('img', {
      src: continuityArt,
      width: 350,
      height: 392,
      style: {
        position: 'absolute',
        right: '24px',
        bottom: '94px',
      },
    }),
  );

  const svg = await satori(element as never, {
    width: 1200,
    height: 630,
    fonts: [
      {
        name: 'Questrial',
        data: loadedFonts.questrial,
        weight: 400,
        style: 'normal',
      },
      {
        name: 'Geist',
        data: loadedFonts.geist,
        weight: 400,
        style: 'normal',
      },
      {
        name: 'Geist',
        data: loadedFonts.geistMedium,
        weight: 600,
        style: 'normal',
      },
    ],
  });

  return sharp(Buffer.from(svg)).png({ compressionLevel: 9 }).toBuffer();
}
