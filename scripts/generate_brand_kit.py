#!/usr/bin/env python3
"""
Reinstate brand kit generator.

DESIGN.md language: sharp flat-vector isometric solids, bold outlines,
paper/ink neutrals, chartreuse accent, four stream layers.

Outputs SVG + PNG under assets/brand-kit/ and key copies to website/public/brand/.
"""

from __future__ import annotations

import math
import subprocess
import xml.sax.saxutils as xu
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "assets" / "brand-kit"
SVG_DIR = OUT / "svg"
PNG_DIR = OUT / "png"
WEB_BRAND = ROOT / "website" / "public" / "brand"

PAPER = "#e4e7dd"
PAPER2 = "#f4f6ef"
INK = "#131f1a"
INK2 = "#4a5a52"
RULE = "#c3c9bc"
PRIM = "#b8ff3c"
FLOOR = "#d3d8c9"
FLOORLINE = "#c3c9b8"
SEAL1, SEAL2, SEAL3 = "#2a3831", "#1a2621", "#121b17"
DARK_PAPER = "#1d2723"
DARK_INK = "#e9efe3"
DARK_MUTED = "#8a9a90"

STREAMS = [
    ("settings", "#fdfef9", "#e2e6dc", "#c3c9bc"),
    ("skills", "#7ecdf5", "#57aad6", "#3d8ab5"),
    ("mcp", "#ffce4a", "#e0ad2c", "#bd8d1b"),
    ("sessions", "#b8ff3c", "#96d92a", "#78b31d"),
]

CX = math.cos(math.pi / 6)


def P(x: float, y: float, z: float) -> tuple[float, float]:
    return ((x - y) * CX, (x + y) * 0.5 - z)


def fmt(pts: list[tuple[float, float]]) -> str:
    return " ".join(f"{a:.1f},{b:.1f}" for a, b in pts)


def box(x: float, y: float, z: float, w: float, d: float, h: float):
    A, B = P(x, y, z + h), P(x + w, y, z + h)
    C, D = P(x + w, y + d, z + h), P(x, y + d, z + h)
    B0, C0, D0 = P(x + w, y, z), P(x + w, y + d, z), P(x, y + d, z)
    return {
        "top": fmt([A, B, C, D]),
        "front": fmt([D, C, C0, D0]),
        "side": fmt([B, C, C0, B0]),
    }


def poly(points: str, fill: str, stroke: str = INK, sw: float = 3.2) -> str:
    return (
        f'<polygon points="{points}" fill="{fill}" stroke="{stroke}" '
        f'stroke-width="{sw}" stroke-linejoin="round"/>'
    )


def solid_box(x, y, z, w, d, h, top, front, side, stroke=INK, sw=3.2) -> str:
    b = box(x, y, z, w, d, h)
    return "\n".join(
        [
            poly(b["side"], side, stroke, sw),
            poly(b["front"], front, stroke, sw),
            poly(b["top"], top, stroke, sw),
        ]
    )


def logo_mark(cx: float, cy: float, size: float, dark: bool = True) -> str:
    s = size / 64.0
    ink = "#f4f6ef" if dark else INK
    bg = INK if dark else PAPER
    tx, ty = cx - size / 2, cy - size / 2

    def t(x, y):
        return x * s + tx, y * s + ty

    def rect(x, y, w, h, rx, fill=None, stroke=None, sw=3.6):
        X, Y = t(x, y)
        a = [
            f'x="{X:.2f}"',
            f'y="{Y:.2f}"',
            f'width="{w * s:.2f}"',
            f'height="{h * s:.2f}"',
            f'rx="{rx * s:.2f}"',
            f'fill="{fill}"' if fill else 'fill="none"',
        ]
        if stroke:
            a += [f'stroke="{stroke}"', f'stroke-width="{sw * s:.2f}"']
        return f"<rect {' '.join(a)}/>"

    def path(cmds, stroke, sw=2.7):
        # Tokenize SVG path: commands may be glued to numbers (M30.5).
        import re

        tokens = re.findall(r"[MLHVmlhv]|-?\d*\.?\d+", cmds.replace(",", " "))
        out = []
        i = 0
        while i < len(tokens):
            c = tokens[i]
            if c in ("M", "L", "m", "l"):
                X, Y = t(float(tokens[i + 1]), float(tokens[i + 2]))
                out.append(f"{c.upper()}{X:.2f},{Y:.2f}")
                i += 3
            elif c in ("H", "h"):
                X, _ = t(float(tokens[i + 1]), 0)
                out.append(f"H{X:.2f}")
                i += 2
            else:
                i += 1
        return (
            f'<path d="{" ".join(out)}" fill="none" stroke="{stroke}" '
            f'stroke-width="{sw * s:.2f}" stroke-linecap="round" stroke-linejoin="round"/>'
        )

    return "\n".join(
        [
            '<g aria-label="Reinstate logo">',
            rect(0, 0, 64, 64, 14, fill=bg),
            rect(14, 14, 28, 28, 5.5, stroke=ink, sw=3.6),
            rect(22, 22, 28, 28, 5.5, fill=bg, stroke=ink, sw=3.6),
            path("M30.5 33 L34.5 37 L30.5 41", ink, 2.7),
            path("M37 41 H42.5", ink, 2.7),
            "</g>",
        ]
    )


def floor_grid(ox, oy, cols, rows, cell, stroke, sw=1.5) -> str:
    lines = []
    for i in range(cols + 1):
        a, b = P(ox + i * cell, oy, 0), P(ox + i * cell, oy + rows * cell, 0)
        lines.append(
            f'<line x1="{a[0]:.1f}" y1="{a[1]:.1f}" x2="{b[0]:.1f}" y2="{b[1]:.1f}" '
            f'stroke="{stroke}" stroke-width="{sw}"/>'
        )
    for j in range(rows + 1):
        a, b = P(ox, oy + j * cell, 0), P(ox + cols * cell, oy + j * cell, 0)
        lines.append(
            f'<line x1="{a[0]:.1f}" y1="{a[1]:.1f}" x2="{b[0]:.1f}" y2="{b[1]:.1f}" '
            f'stroke="{stroke}" stroke-width="{sw}"/>'
        )
    return "\n".join(lines)


def stream_stack(x, y, z, w, d, h, pitch, lifted) -> str:
    parts = []
    for i, (_, top, front, side) in enumerate(STREAMS):
        zi = z + i * (pitch if lifted else h)
        parts.append(solid_box(x, y, zi, w, d, h, top, front, side, INK, 2.8))
    return "\n".join(parts)


def sealed_cube(x, y, z, s) -> str:
    return solid_box(x, y, z, s, s, s, SEAL1, SEAL2, SEAL3, INK, 3.0)


def text(
    x,
    y,
    content,
    *,
    size=18,
    fill=INK,
    weight="500",
    family="ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif",
    anchor=None,
    mono=False,
    tracking=None,
    lines=None,
):
    if mono:
        family = "ui-monospace, SFMono-Regular, Menlo, monospace"
    attrs = [
        f'x="{x}"',
        f'y="{y}"',
        f'font-family="{family}"',
        f'font-size="{size}"',
        f'font-weight="{weight}"',
        f'fill="{fill}"',
    ]
    if anchor:
        attrs.append(f'text-anchor="{anchor}"')
    if tracking:
        attrs.append(f'letter-spacing="{tracking}"')
    if lines:
        tspans = []
        for i, line in enumerate(lines):
            if i == 0:
                tspans.append(f'<tspan x="{x}" y="{y}">{xu.escape(line)}</tspan>')
            else:
                tspans.append(
                    f'<tspan x="{x}" dy="{size * 1.15:.1f}">{xu.escape(line)}</tspan>'
                )
        return f"<text {' '.join(attrs)}>{''.join(tspans)}</text>"
    return f"<text {' '.join(attrs)}>{xu.escape(content)}</text>"


def svg_doc(w: int, h: int, body: str, bg: str = PAPER) -> str:
    return f'''<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}" role="img">
  <title>Reinstate</title>
  <rect width="{w}" height="{h}" fill="{bg}"/>
  {body}
</svg>
'''


def g_translate(ox, oy, scale, inner) -> str:
    return f'<g transform="translate({ox},{oy}) scale({scale})" stroke-linejoin="round">\n{inner}\n</g>'


# ---- assets ----

def pfp(size=800, dark=True) -> str:
    bg = DARK_PAPER if dark else PAPER
    return svg_doc(size, size, logo_mark(size / 2, size / 2, size * 0.72, dark=dark), bg)


def banner_wide(w, h, title=None, subtitle=None, dark=False) -> str:
    bg = DARK_PAPER if dark else PAPER
    ink = DARK_INK if dark else INK
    ink2 = DARK_MUTED if dark else INK2
    floor_c = "#16201c" if dark else FLOOR
    floorline = "#253029" if dark else FLOORLINE
    rule = "#2b3831" if dark else RULE
    stroke = DARK_INK if dark else INK

    parts = [
        f'<rect x="0" y="{h * 0.55:.0f}" width="{w}" height="{h * 0.45:.0f}" fill="{floor_c}"/>',
        f'<line x1="0" y1="{h * 0.55:.0f}" x2="{w}" y2="{h * 0.55:.0f}" stroke="{rule}" stroke-width="2"/>',
        f'<rect x="0" y="0" width="8" height="{h}" fill="{PRIM}"/>',
    ]

    sc = min(w, h) / 480
    iso_inner = "\n".join(
        [
            floor_grid(-40, -40, 8, 6, 40, floorline, 1.8),
            sealed_cube(120, 20, 0, 70),
            solid_box(145, 45, 70, 22, 14, 10, PRIM, "#96d92a", "#78b31d", stroke, 2.4),
            stream_stack(-20, 10, 0, 90, 58, 14, 26, True),
            stream_stack(40, 90, 0, 70, 46, 12, 12, False),
        ]
    )
    parts.append(g_translate(w * 0.58, h * 0.62, sc, iso_inner))

    lx = 48
    parts.append(logo_mark(lx + 44, h * 0.28, min(100, h * 0.34), dark=True))
    parts.append(
        text(
            lx + 110,
            h * 0.26,
            "reinstate",
            size=min(52, h * 0.13),
            fill=ink,
            weight="800",
            tracking="-0.04em",
        )
    )
    if title:
        parts.append(
            text(lx + 110, h * 0.42, title, size=min(26, h * 0.065), fill=ink2, weight="600")
        )
    if subtitle:
        parts.append(
            text(lx + 110, h * 0.52, subtitle, size=min(16, h * 0.04), fill=ink2, mono=True)
        )
    else:
        parts.append(
            text(
                lx + 110,
                h * 0.48,
                "Pick up any coding task exactly where you left it.",
                size=min(18, h * 0.045),
                fill=ink2,
            )
        )
    parts.append(
        text(lx + 110, h * 0.78, "reinstate.dev  ·  rein", size=14, fill=ink2, mono=True)
    )
    return svg_doc(w, h, "\n".join(parts), bg)


def youtube_banner() -> str:
    w, h = 2560, 1440
    parts = [
        f'<rect x="0" y="0" width="{w}" height="280" fill="{PAPER2}"/>',
        f'<rect x="0" y="{h - 280}" width="{w}" height="280" fill="{FLOOR}"/>',
        f'<line x1="0" y1="280" x2="{w}" y2="280" stroke="{RULE}" stroke-width="3"/>',
        f'<line x1="0" y1="{h - 280}" x2="{w}" y2="{h - 280}" stroke="{RULE}" stroke-width="3"/>',
        f'<rect x="0" y="0" width="16" height="{h}" fill="{PRIM}"/>',
        logo_mark(w / 2 - 420, h / 2, 200, dark=True),
        text(w / 2 - 280, h / 2 - 10, "reinstate", size=96, fill=INK, weight="800", tracking="-0.04em"),
        text(
            w / 2 - 280,
            h / 2 + 50,
            "Pick up any coding task exactly where you left it.",
            size=28,
            fill=INK2,
        ),
        text(w / 2 - 280, h / 2 + 100, "rein  ·  reinstate.dev", size=20, fill=INK2, mono=True),
        g_translate(
            w / 2 + 380,
            h / 2 + 80,
            1.4,
            stream_stack(0, 0, 0, 100, 64, 16, 28, True)
            + "\n"
            + sealed_cube(140, 20, 0, 80),
        ),
    ]
    return svg_doc(w, h, "\n".join(parts), PAPER)


def post_template(w, h, lines, sub, dark=False) -> str:
    bg = DARK_PAPER if dark else PAPER
    ink = DARK_INK if dark else INK
    ink2 = DARK_MUTED if dark else INK2
    parts = [
        f'<rect x="0" y="0" width="10" height="{h}" fill="{PRIM}"/>',
        logo_mark(72, 72, 96, dark=True),
        text(140, 82, "reinstate", size=34, fill=ink, weight="800", tracking="-0.04em"),
        text(48, h * 0.40, "", size=42, fill=ink, weight="800", tracking="-0.03em", lines=lines),
        text(48, h * 0.40 + 42 * 1.15 * len(lines) + 28, sub, size=22, fill=ink2),
        text(48, h - 48, "reinstate.dev", size=16, fill=ink2, mono=True),
        g_translate(w - 200, h - 90, 0.9, stream_stack(0, 0, 0, 70, 44, 12, 20, True)),
    ]
    return svg_doc(w, h, "\n".join(parts), bg)


def poster(w=1080, h=1350) -> str:
    parts = [
        f'<rect x="0" y="0" width="12" height="{h}" fill="{PRIM}"/>',
        logo_mark(w / 2, 200, 180, dark=True),
        text(w / 2, 380, "reinstate", size=64, fill=INK, weight="800", tracking="-0.04em", anchor="middle"),
        text(
            w / 2,
            440,
            "Continuity layer for coding-agent work",
            size=24,
            fill=INK2,
            anchor="middle",
        ),
        g_translate(
            w / 2 - 20,
            820,
            1.7,
            stream_stack(-80, 0, 0, 90, 56, 14, 26, True) + "\n" + sealed_cube(60, 30, 0, 72),
        ),
        text(w / 2, h - 60, "reinstate.dev", size=18, fill=INK2, mono=True, anchor="middle"),
    ]
    return svg_doc(w, h, "\n".join(parts), PAPER)


def linkedin_company_cover() -> str:
    # LinkedIn company page cover 1128×191
    return banner_wide(1128, 191, title="Continuity for coding agents", subtitle="reinstate.dev", dark=False)


ASSETS = {
    "pfp-800": lambda: pfp(800, True),
    "pfp-400": lambda: pfp(400, True),
    "pfp-light-800": lambda: pfp(800, False),
    "og-1200x630": lambda: banner_wide(
        1200,
        630,
        title="Continuity layer for coding-agent work",
        subtitle="search · resume · handoff · encrypted sync",
    ),
    "og-dark-1200x630": lambda: banner_wide(
        1200,
        630,
        title="Continuity layer for coding-agent work",
        subtitle="search · resume · handoff · encrypted sync",
        dark=True,
    ),
    "x-header-1500x500": lambda: banner_wide(
        1500, 500, title="Continuity for every agent session", subtitle="encrypted · portable · open source"
    ),
    "linkedin-banner-1584x396": lambda: banner_wide(
        1584, 396, title="Restore the workspace. Not just the chat.", subtitle="reinstate.dev"
    ),
    "linkedin-company-1128x191": linkedin_company_cover,
    "youtube-banner-2560x1440": youtube_banner,
    "x-post-1200x675": lambda: post_template(
        1200,
        675,
        ["Your session is not stuck", "on one machine."],
        "Search, resume, and hand off coding-agent work, verified and encrypted.",
    ),
    "linkedin-post-1200x627": lambda: post_template(
        1200,
        627,
        ["Reinstate is not another", "place to code."],
        "It makes every place you code continuous.",
    ),
    "ig-square-1080": lambda: post_template(
        1080,
        1080,
        ["Pick up any coding task", "exactly where you left it."],
        "Continuity for Claude Code, Codex, and more.",
    ),
    "poster-1080x1350": poster,
    "discord-512": lambda: pfp(512, True),
    "slack-512": lambda: pfp(512, True),
}


def export_png(svg_path: Path, png_path: Path, width: int | None = None) -> None:
    cmd = ["rsvg-convert", str(svg_path), "-o", str(png_path)]
    if width:
        cmd = ["rsvg-convert", "-w", str(width), str(svg_path), "-o", str(png_path)]
    subprocess.run(cmd, check=True)


def main() -> int:
    SVG_DIR.mkdir(parents=True, exist_ok=True)
    PNG_DIR.mkdir(parents=True, exist_ok=True)
    WEB_BRAND.mkdir(parents=True, exist_ok=True)

    print(f"Generating brand kit → {OUT}")
    for name, builder in ASSETS.items():
        content = builder()
        svg_path = SVG_DIR / f"{name}.svg"
        svg_path.write_text(content, encoding="utf-8")
        width = None
        for part in name.split("-"):
            if "x" in part and part[0].isdigit():
                width = int(part.split("x")[0])
                break
            if part.isdigit() and int(part) >= 400:
                width = int(part)
        png_path = PNG_DIR / f"{name}.png"
        try:
            export_png(svg_path, png_path, width)
            print(f"  ✓ {name}")
        except Exception as e:
            print(f"  ✓ {name}.svg only ({e})")

    # Website public brand copies
    (WEB_BRAND / "banner.svg").write_text(
        (SVG_DIR / "og-1200x630.svg").read_text(encoding="utf-8"), encoding="utf-8"
    )
    print("  → public/brand/banner.svg")
    for src, dst in [
        (PNG_DIR / "og-1200x630.png", WEB_BRAND / "og.png"),
        (SVG_DIR / "og-1200x630.svg", WEB_BRAND / "og.svg"),
        (PNG_DIR / "x-header-1500x500.png", WEB_BRAND / "x-header.png"),
        (PNG_DIR / "linkedin-banner-1584x396.png", WEB_BRAND / "linkedin-banner.png"),
        (PNG_DIR / "youtube-banner-2560x1440.png", WEB_BRAND / "youtube-banner.png"),
        (PNG_DIR / "pfp-800.png", WEB_BRAND / "avatar.png"),
        (PNG_DIR / "poster-1080x1350.png", WEB_BRAND / "poster.png"),
        (PNG_DIR / "x-post-1200x675.png", WEB_BRAND / "x-post-template.png"),
        (PNG_DIR / "linkedin-post-1200x627.png", WEB_BRAND / "linkedin-post-template.png"),
    ]:
        if src.exists():
            dst.write_bytes(src.read_bytes())
            print(f"  → public/brand/{dst.name}")

    # Copy final logo masters into brand kit
    logo_src = ROOT / "assets" / "logo" / "final"
    logo_dst = OUT / "logo"
    logo_dst.mkdir(exist_ok=True)
    for f in logo_src.glob("*"):
        if f.is_file():
            (logo_dst / f.name).write_bytes(f.read_bytes())

    print("  → GitHub social preview: run `cd website && npm run generate:github-social`")
    print("Done.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
