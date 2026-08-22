#!/usr/bin/env python3
"""Generate four README banner concepts in the Reinstate landing-page style."""

from __future__ import annotations

import base64
import html
import math
from pathlib import Path

from generate_brand_kit import P, ROOT, box, fmt

OUT = ROOT / "assets" / "banner-concepts"
W, H = 1200, 380
CX = math.cos(math.pi / 6)

PAPER = "#e4e7dd"
PAPER2 = "#f4f6ef"
INK = "#131f1a"
INK2 = "#4a5a52"
INK3 = "#718078"
RULE = "#c3c9bc"
PRIM = "#b8ff3c"
PRIM_DEEP = "#4f8a1e"
FLOOR = "#d3d8c9"
FLOORLINE = "#c3c9b8"

DARK_PAPER = "#1d2723"
DARK_PAPER2 = "#26332e"
DARK_INK = "#e9efe3"
DARK_INK2 = "#b5c0b8"
DARK_INK3 = "#8a9a90"
DARK_RULE = "#445048"
DARK_FLOOR = "#16201c"
DARK_FLOORLINE = "#253029"

STREAMS = [
    ("settings", "#fdfef9", "#e2e6dc", "#c3c9bc"),
    ("skills", "#7ecdf5", "#57aad6", "#3d8ab5"),
    ("mcp", "#ffce4a", "#e0ad2c", "#bd8d1b"),
    ("sessions", "#b8ff3c", "#96d92a", "#78b31d"),
]

LIGHT = {
    "paper": PAPER,
    "paper2": PAPER2,
    "ink": INK,
    "ink2": INK2,
    "ink3": INK3,
    "rule": RULE,
    "floor": FLOOR,
    "floorline": FLOORLINE,
    "stroke": INK,
    "frame": PAPER2,
    "night": "#16231f",
    "furn": ("#c1c9ae", "#a6b090", "#8a9673"),
    "dev": ("#3b4c43", "#2a3831", "#1e2925"),
    "seal": ("#2a3831", "#1a2621", "#121b17"),
    "leaf": ("#7fae3f", "#5f8f2c"),
    "pot": ("#c98f5a", "#b07845", "#966235"),
}

DARK = {
    "paper": DARK_PAPER,
    "paper2": DARK_PAPER2,
    "ink": DARK_INK,
    "ink2": DARK_INK2,
    "ink3": DARK_INK3,
    "rule": DARK_RULE,
    "floor": DARK_FLOOR,
    "floorline": DARK_FLOORLINE,
    "stroke": DARK_INK,
    "frame": "#2b3831",
    "night": "#0c1512",
    "furn": ("#44564b", "#35453c", "#29362f"),
    "dev": ("#46584e", "#35453c", "#27332c"),
    "seal": ("#2b3a33", "#1f2b26", "#16201c"),
    "leaf": ("#6f9c36", "#4f7a25"),
    "pot": ("#9d6c40", "#855a33", "#6d4927"),
}


def data_font(relative_path: str) -> str:
    payload = (ROOT / relative_path).read_bytes()
    return base64.b64encode(payload).decode("ascii")


FONT_CSS = f"""
@font-face {{
  font-family: "Questrial";
  src: url(data:font/woff2;base64,{data_font("website/node_modules/@fontsource/questrial/files/questrial-latin-400-normal.woff2")}) format("woff2");
  font-weight: 400;
}}
@font-face {{
  font-family: "Geist";
  src: url(data:font/woff2;base64,{data_font("website/node_modules/@fontsource-variable/geist/files/geist-latin-wght-normal.woff2")}) format("woff2");
  font-weight: 100 900;
}}
@font-face {{
  font-family: "Geist Mono";
  src: url(data:font/woff2;base64,{data_font("website/node_modules/@fontsource-variable/geist-mono/files/geist-mono-latin-wght-normal.woff2")}) format("woff2");
  font-weight: 100 900;
}}
.display {{ font-family: "Questrial", sans-serif; font-weight: 400; }}
.body {{ font-family: "Geist", sans-serif; }}
.mono {{ font-family: "Geist Mono", monospace; }}
"""


def attrs(**values) -> str:
    return " ".join(
        f'{key.replace("_", "-")}="{html.escape(str(value), quote=True)}"'
        for key, value in values.items()
        if value is not None
    )


def text(
    x: float,
    y: float,
    value: str,
    *,
    size: float,
    fill: str,
    css_class: str = "body",
    weight: int = 500,
    anchor: str | None = None,
    tracking: str | None = None,
) -> str:
    return (
        f"<text {attrs(x=x, y=y, font_size=size, fill=fill, font_weight=weight, text_anchor=anchor, letter_spacing=tracking, class_=css_class)}>"
        f"{html.escape(value)}</text>"
    ).replace('class-="', 'class="')


def multi_text(
    x: float,
    y: float,
    lines: list[str],
    *,
    size: float,
    fill: str,
    leading: float,
    css_class: str = "display",
    anchor: str | None = None,
    tracking: str | None = "-0.035em",
) -> str:
    tspans = []
    for index, line in enumerate(lines):
        dy = 0 if index == 0 else leading
        tspans.append(f'<tspan x="{x}" dy="{dy}">{html.escape(line)}</tspan>')
    return (
        f"<text {attrs(x=x, y=y, font_size=size, fill=fill, text_anchor=anchor, letter_spacing=tracking, class_=css_class)}>"
        f"{''.join(tspans)}</text>"
    ).replace('class-="', 'class="')


def bare_mark(x: float, y: float, size: float, theme: dict) -> str:
    scale = size / 64
    stroke = theme["stroke"]
    punch = theme["paper"]
    return f"""
<g transform="translate({x} {y}) scale({scale})" aria-label="Reinstate mark">
  <rect x="14" y="14" width="28" height="28" rx="5.5" fill="none" stroke="{stroke}" stroke-width="3.6"/>
  <rect x="22" y="22" width="28" height="28" rx="5.5" fill="{punch}" stroke="{stroke}" stroke-width="3.6"/>
  <path d="M30.5 33 L34.5 37 L30.5 41" fill="none" stroke="{stroke}" stroke-width="2.7" stroke-linecap="round" stroke-linejoin="round"/>
  <path d="M37 41 H42.5" fill="none" stroke="{stroke}" stroke-width="2.7" stroke-linecap="round"/>
</g>"""


def lockup(x: float, y: float, theme: dict, *, size: float = 38) -> str:
    return "\n".join(
        [
            bare_mark(x, y - size * 0.72, size, theme),
            text(
                x + size + 8,
                y,
                "Reinstate",
                size=size * 0.62,
                fill=theme["ink"],
                css_class="display",
                weight=400,
                tracking="-0.035em",
            ),
        ]
    )


def polygon(points: str, fill: str, stroke: str, sw: float = 3.2) -> str:
    return (
        f'<polygon points="{points}" fill="{fill}" stroke="{stroke}" '
        f'stroke-width="{sw}" stroke-linejoin="round"/>'
    )


def solid(
    x: float,
    y: float,
    z: float,
    w: float,
    d: float,
    h: float,
    tones: tuple[str, str, str],
    theme: dict,
    sw: float = 3.2,
) -> str:
    shape = box(x, y, z, w, d, h)
    top, front, side = tones
    return "\n".join(
        [
            polygon(shape["side"], side, theme["stroke"], sw),
            polygon(shape["front"], front, theme["stroke"], sw),
            polygon(shape["top"], top, theme["stroke"], sw),
        ]
    )


def plane(x: float, y: float, z: float, w: float, d: float) -> str:
    return fmt([P(x, y, z), P(x + w, y, z), P(x + w, y + d, z), P(x, y + d, z)])


def transformed_point(
    tx: float,
    ty: float,
    scale: float,
    x: float,
    y: float,
    z: float,
) -> tuple[float, float]:
    px, py = P(x, y, z)
    return tx + px * scale, ty + py * scale


def vertical_side(x: float, y: float, z: float, d: float, h: float) -> str:
    """A flat inset on the +x face of an isometric solid."""
    return fmt([P(x, y, z + h), P(x, y + d, z + h), P(x, y + d, z), P(x, y, z)])


def vertical_front(x: float, y: float, z: float, w: float, h: float) -> str:
    """A flat inset on the +y face of an isometric solid."""
    return fmt([P(x, y, z + h), P(x + w, y, z + h), P(x + w, y, z), P(x, y, z)])


def flat_polygon(points: str, fill: str, *, stroke: str | None = None, sw: float = 1.8) -> str:
    stroke_attrs = f' stroke="{stroke}" stroke-width="{sw}"' if stroke else ""
    return f'<polygon points="{points}" fill="{fill}"{stroke_attrs} stroke-linejoin="round"/>'


def stream_stack(
    x: float,
    y: float,
    z: float,
    w: float,
    d: float,
    h: float,
    pitch: float,
    theme: dict,
) -> str:
    output = []
    for index, (_, top, front, side) in enumerate(STREAMS):
        output.append(solid(x, y, z + index * pitch, w, d, h, (top, front, side), theme, 2.8))
    return "\n".join(output)


def table(theme: dict, *, laptop: bool, lifted: bool) -> str:
    output = []
    output.append(
        f'<polygon points="{plane(0, 0, -92, 226, 140)}" fill="{theme["floorline"]}" opacity="0.7"/>'
    )
    for lx, ly in [(12, 12), (198, 12), (12, 112), (198, 112)]:
        output.append(solid(lx, ly, -88, 14, 14, 78, theme["furn"], theme, 2.7))
    output.append(solid(0, 0, -10, 226, 140, 10, theme["furn"], theme, 3))

    if laptop:
        output.extend(
            [
                solid(52, 27, 0, 132, 90, 12, theme["dev"], theme, 2.7),
                solid(52, 21, 12, 132, 7, 8, theme["dev"], theme, 2.4),
                solid(53, 16, 12, 130, 11, 87, theme["dev"], theme, 2.7),
                flat_polygon(
                    vertical_front(63, 27.2, 28, 110, 58),
                    theme["night"],
                    stroke=theme["stroke"],
                    sw=1.8,
                ),
                flat_polygon(vertical_front(72, 27.4, 72, 58, 4), PRIM),
                flat_polygon(vertical_front(72, 27.4, 60, 86, 3), theme["frame"]),
                flat_polygon(vertical_front(72, 27.4, 50, 70, 3), theme["frame"]),
                flat_polygon(vertical_front(153, 27.4, 42, 10, 10), PRIM),
                f'<polygon points="{plane(65, 100, 12.3, 108, 10)}" fill="{theme["frame"]}" stroke="{theme["stroke"]}" stroke-width="1.8"/>',
                stream_stack(70, 44, 13, 96, 60, 10, 10 if not lifted else 22, theme),
            ]
        )
    else:
        output.extend(
            [
                solid(12, 55, 0, 42, 42, 8, theme["dev"], theme, 2.5),
                solid(27, 68, 8, 12, 18, 22, theme["dev"], theme, 2.3),
                solid(20, 10, 30, 14, 126, 86, theme["dev"], theme, 2.8),
                flat_polygon(
                    vertical_side(34.2, 21, 44, 104, 58),
                    theme["night"],
                    stroke=theme["stroke"],
                    sw=1.8,
                ),
                flat_polygon(vertical_side(34.4, 31, 88, 56, 4), PRIM),
                flat_polygon(vertical_side(34.4, 31, 73, 78, 3), theme["frame"]),
                flat_polygon(vertical_side(34.4, 31, 61, 65, 3), theme["frame"]),
                flat_polygon(vertical_side(34.4, 105, 49, 10, 10), PRIM),
                solid(72, 38, 0, 118, 78, 12, theme["dev"], theme, 2.8),
                stream_stack(81, 45, 14, 100, 62, 10, 22 if lifted else 10, theme),
                solid(184, 82, -88, 38, 42, 78, theme["dev"], theme, 2.8),
            ]
        )
    return "\n".join(output)


def checkpoint(theme: dict, *, open_top: bool = False) -> str:
    output = [
        f'<polygon points="{plane(-18, -18, -8, 136, 136)}" fill="{theme["floorline"]}" opacity="0.75"/>',
        solid(-8, -8, -4, 116, 116, 8, theme["furn"], theme, 2.8),
        solid(0, 0, 4, 100, 100, 100, theme["seal"], theme, 3.2),
    ]
    if open_top:
        output.append(stream_stack(10, 10, 110, 80, 80, 10, 25, theme))
    else:
        output.append(solid(34, 38, 105, 32, 24, 9, (PRIM, "#96d92a", "#78b31d"), theme, 2.4))
    return "\n".join(output)


def plant(x: float, y: float, scale: float, theme: dict) -> str:
    return f"""
<g transform="translate({x} {y}) scale({scale})">
  <g transform="translate(-1 13)">
    {solid(-28, -28, -58, 56, 56, 58, theme["pot"], theme, 2.8)}
    {flat_polygon(plane(-20, -20, 0.4, 40, 40), theme["seal"][2])}
  </g>
  <g stroke="{theme["stroke"]}" stroke-width="3" stroke-linejoin="round">
    <path d="M0 6 C-24-34 -18-72 4-92 C17-60 14-20 0 6Z" fill="{theme["leaf"][0]}"/>
    <path d="M0 6 C-42-2 -62-32 -64-58 C-36-46 -12-15 0 6Z" fill="{theme["leaf"][1]}"/>
    <path d="M0 6 C34-2 54-30 58-55 C31-42 10-14 0 6Z" fill="{theme["leaf"][1]}"/>
  </g>
</g>"""


def window(x: float, y: float, theme: dict) -> str:
    return f"""
<g transform="translate({x} {y})">
  <rect width="168" height="112" rx="5" fill="{theme["frame"]}" stroke="{theme["stroke"]}" stroke-width="3"/>
  <rect x="8" y="8" width="152" height="96" fill="{theme["night"]}"/>
  <circle cx="129" cy="30" r="11" fill="{PRIM}"/>
  <g fill="{theme["leaf"][0]}">
    <circle cx="38" cy="27" r="2"/><circle cx="72" cy="19" r="1.7"/><circle cx="103" cy="44" r="2"/>
  </g>
  <g fill="{theme["dev"][0]}">
    <rect x="14" y="72" width="20" height="32"/><rect x="39" y="58" width="25" height="46"/>
    <rect x="70" y="78" width="29" height="26"/><rect x="106" y="65" width="18" height="39"/>
    <rect x="130" y="75" width="24" height="29"/>
  </g>
  <line x1="84" y1="8" x2="84" y2="104" stroke="{theme["frame"]}" stroke-width="3"/>
  <line x1="8" y1="56" x2="160" y2="56" stroke="{theme["frame"]}" stroke-width="3"/>
</g>"""


def lamp(x: float, y: float, theme: dict, scale: float = 1) -> str:
    return f"""
<g transform="translate({x} {y}) scale({scale})" stroke="{theme["stroke"]}" stroke-linejoin="round">
  <polygon points="-31,0 0,-17 31,0 0,17" fill="{theme["furn"][0]}" stroke-width="3"/>
  <rect x="-5" y="-116" width="10" height="116" fill="{theme["dev"][0]}" stroke-width="3"/>
  <path d="M-30-123 L-18-165 Q0-180 18-165 L30-123 Q0-110-30-123Z" fill="{theme["dev"][1]}" stroke-width="3"/>
  <ellipse cx="0" cy="-126" rx="23" ry="7" fill="{PRIM}" stroke-width="2.5"/>
</g>"""


def arrow(
    d: str,
    label_x: float,
    label_y: float,
    label: str,
    theme: dict,
    *,
    anchor: str = "middle",
) -> str:
    return "\n".join(
        [
            f'<path d="{d}" fill="none" stroke="{PRIM_DEEP if theme is LIGHT else PRIM}" stroke-width="2.4" stroke-dasharray="9 7" marker-end="url(#arrow)"/>',
            pill(label_x, label_y, label, theme, anchor=anchor),
        ]
    )


def pill(x: float, y: float, label: str, theme: dict, *, anchor: str = "middle") -> str:
    width = max(86, len(label) * 7.2 + 24)
    left = x - width / 2 if anchor == "middle" else x
    tx = x if anchor == "middle" else x + 12
    return "\n".join(
        [
            f'<rect x="{left}" y="{y - 16}" width="{width}" height="27" rx="13.5" fill="{theme["paper2"]}" stroke="{theme["rule"]}"/>',
            text(tx, y + 2, label, size=12, fill=theme["ink2"], css_class="mono", weight=520, anchor=anchor),
        ]
    )


def floor(theme: dict, horizon: float, *, grid: bool = True) -> str:
    output = [
        f'<rect x="0" y="{horizon}" width="{W}" height="{H - horizon}" fill="{theme["floor"]}"/>',
        f'<line x1="0" y1="{horizon}" x2="{W}" y2="{horizon}" stroke="{theme["stroke"]}" stroke-width="2"/>',
    ]
    if grid:
        paths = []
        for base in range(-280, 1500, 95):
            paths.append(f"M{base} {horizon} L{base + 330} {H}")
            paths.append(f"M{base} {horizon} L{base - 330} {H}")
        output.append(
            f'<path d="{" ".join(paths)}" fill="none" stroke="{theme["floorline"]}" stroke-width="1.1"/>'
        )
    return "\n".join(output)


def defs(theme: dict) -> str:
    arrow_fill = PRIM_DEEP if theme is LIGHT else PRIM
    return f"""<defs>
  <style>{FONT_CSS}</style>
  <marker id="arrow" viewBox="0 0 12 12" refX="9" refY="6" markerWidth="6" markerHeight="6" orient="auto">
    <path d="M0 0 L12 6 L0 12Z" fill="{arrow_fill}"/>
  </marker>
  <clipPath id="canvas"><rect width="{W}" height="{H}" rx="12"/></clipPath>
</defs>"""


def document(title: str, description: str, theme: dict, body: str) -> str:
    return f"""<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" aria-labelledby="title desc">
  <title id="title">{html.escape(title)}</title>
  <desc id="desc">{html.escape(description)}</desc>
  {defs(theme)}
  <g clip-path="url(#canvas)">
    <rect width="{W}" height="{H}" fill="{theme["paper"]}"/>
    {body}
  </g>
  <rect x="1" y="1" width="{W - 2}" height="{H - 2}" rx="11" fill="none" stroke="{theme["rule"]}" stroke-width="2"/>
</svg>
"""


def concept_one() -> str:
    theme = LIGHT

    source = (338, 294, 1.02)
    center = (604, 275, 0.78)
    destination = (870, 294, 1.02)

    push_start = transformed_point(*source, 81, 107, 90)
    push_end = transformed_point(*center, 50, 100, 104)
    pull_start = transformed_point(*center, 100, 50, 104)
    pull_end = transformed_point(*destination, 70, 44, 53)

    axis_x = center[0]
    pull_dx = pull_end[0] - pull_start[0]
    pull_c1 = (pull_start[0] + pull_dx * 0.35, pull_start[1] - 87)
    pull_c2 = (pull_end[0] - pull_dx * 0.25, pull_end[1] - 118)
    push_c1 = (2 * axis_x - pull_c2[0], pull_c2[1])
    push_c2 = (2 * axis_x - pull_c1[0], pull_c1[1])

    push_path = (
        f"M{push_start[0]:.0f} {push_start[1]:.0f} "
        f"C{push_c1[0]:.0f} {push_c1[1]:.0f} "
        f"{push_c2[0]:.0f} {push_c2[1]:.0f} "
        f"{push_end[0]:.0f} {push_end[1]:.0f}"
    )
    pull_path = (
        f"M{pull_start[0]:.0f} {pull_start[1]:.0f} "
        f"C{pull_c1[0]:.0f} {pull_c1[1]:.0f} "
        f"{pull_c2[0]:.0f} {pull_c2[1]:.0f} "
        f"{pull_end[0]:.0f} {pull_end[1]:.0f}"
    )

    body = [
        floor(theme, 190),
        plant(70, 292, 0.68, theme),
        lamp(1140, 317, theme, 1.15),
        lockup(34, 44, theme, size=34),
        multi_text(
            600,
            70,
            ["Start on one device.", "Continue on another. Same session."],
            size=39,
            fill=theme["ink"],
            leading=40,
            anchor="middle",
        ),
        text(
            600,
            138,
            "Encrypted Claude Code and Codex sessions, stored in your own bucket.",
            size=14,
            fill=theme["ink2"],
            css_class="body",
            weight=470,
            anchor="middle",
        ),
        f'<g transform="translate({source[0]} {source[1]}) scale({source[2]})">{table(theme, laptop=False, lifted=True)}</g>',
        f'<g transform="translate({center[0]} {center[1]}) scale({center[2]})">{checkpoint(theme)}</g>',
        f'<g transform="translate({destination[0]} {destination[1]}) scale({destination[2]})">{table(theme, laptop=True, lifted=False)}</g>',
        arrow(
            push_path,
            2 * axis_x - (pull_start[0] + pull_end[0]) / 2,
            177,
            "$ rein push",
            theme,
        ),
        arrow(pull_path, (pull_start[0] + pull_end[0]) / 2, 177, "$ rein pull", theme),
    ]
    return document(
        "Reinstate, session continuity across devices",
        "A light isometric workroom showing a desktop session pushed into an encrypted checkpoint and pulled onto a laptop, with a plant and floor lamp.",
        theme,
        "\n".join(body),
    )


def concept_two() -> str:
    theme = LIGHT
    body = [
        floor(theme, 252),
        lockup(50, 48, theme, size=42),
        pill(76, 101, "CAPTURE  ·  ENCRYPT  ·  REINSTATE", theme, anchor="start"),
        multi_text(
            52,
            157,
            ["Pull apart.", "Seal. Put back."],
            size=51,
            fill=theme["ink"],
            leading=52,
        ),
        text(
            54,
            283,
            "Your session leaves as ciphertext",
            size=15,
            fill=theme["ink2"],
            css_class="body",
            weight=500,
        ),
        text(
            54,
            307,
            "and returns ready for native resume.",
            size=15,
            fill=theme["ink2"],
            css_class="body",
            weight=500,
        ),
        pill(54, 348, "age E2EE", theme, anchor="start"),
        pill(162, 348, "path remap", theme, anchor="start"),
        pill(292, 348, "your bucket", theme, anchor="start"),
        f'<g transform="translate(820 265) scale(1.14)">{checkpoint(theme, open_top=True)}</g>',
        f'<g transform="translate(1048 285) scale(0.68)">{table(theme, laptop=True, lifted=False)}</g>',
        arrow("M914 210 C976 190 1008 205 1036 236", 976, 186, "$ rein pull", theme),
        text(820, 75, "LOCAL", size=10, fill=theme["ink3"], css_class="mono", weight=600, anchor="middle", tracking="0.12em"),
        text(820, 94, "ENCRYPTION", size=10, fill=theme["ink3"], css_class="mono", weight=600, anchor="middle", tracking="0.12em"),
        f'<path d="M780 117 H858" stroke="{PRIM_DEEP}" stroke-width="2.5"/>',
        f'<circle cx="820" cy="117" r="6" fill="{PRIM} " stroke="{theme["stroke"]}" stroke-width="2.5"/>',
    ]
    return document(
        "Reinstate banner concept 2, Sealed Mechanism",
        "A light close-up of the four-layer state motif sealing into an encrypted checkpoint and returning to a laptop.",
        theme,
        "\n".join(body),
    )


def concept_three() -> str:
    theme = LIGHT
    body = [
        floor(theme, 126),
        lockup(42, 45, theme, size=38),
        text(
            1160,
            51,
            "Same session. Another device.",
            size=29,
            fill=theme["ink"],
            css_class="display",
            weight=400,
            anchor="end",
            tracking="-0.035em",
        ),
        text(
            1160,
            78,
            "Claude Code · Codex · encrypted locally · remapped on restore",
            size=12,
            fill=theme["ink2"],
            css_class="mono",
            weight=500,
            anchor="end",
        ),
        f'<g transform="translate(150 268) scale(0.72)">{table(theme, laptop=False, lifted=True)}</g>',
        f'<g transform="translate(579 263) scale(0.78)">{checkpoint(theme)}</g>',
        f'<g transform="translate(914 268) scale(0.72)">{table(theme, laptop=True, lifted=False)}</g>',
        arrow("M345 224 C410 178 477 177 544 219", 447, 166, "$ rein push", theme),
        arrow("M663 219 C730 177 800 178 865 224", 764, 166, "$ rein pull", theme),
        text(236, 144, "SOURCE  ·  WINDOWS", size=10, fill=theme["ink3"], css_class="mono", weight=600, anchor="middle", tracking="0.08em"),
        text(620, 144, "YOUR S3 / R2 BUCKET", size=10, fill=theme["ink3"], css_class="mono", weight=600, anchor="middle", tracking="0.08em"),
        text(1010, 144, "DESTINATION  ·  macOS", size=10, fill=theme["ink3"], css_class="mono", weight=600, anchor="middle", tracking="0.08em"),
    ]
    return document(
        "Reinstate banner concept 3, Verified Relay",
        "A light technical relay showing encrypted session push and pull between Windows and macOS.",
        theme,
        "\n".join(body),
    )


def concept_four() -> str:
    theme = DARK
    body = [
        floor(theme, 205),
        window(26, 30, theme),
        plant(80, 360, 0.58, theme),
        lamp(1142, 370, theme),
        lockup(226, 51, theme, size=38),
        multi_text(
            600,
            98,
            ["Pick up any coding task", "exactly where you left it."],
            size=43,
            fill=theme["ink"],
            leading=44,
            anchor="middle",
        ),
        text(
            600,
            177,
            "Encrypted session continuity for Claude Code and Codex.",
            size=14,
            fill=theme["ink2"],
            css_class="body",
            weight=470,
            anchor="middle",
        ),
        f'<g transform="translate(242 305) scale(0.63)">{table(theme, laptop=False, lifted=True)}</g>',
        f'<g transform="translate(604 288) scale(0.67)">{checkpoint(theme)}</g>',
        f'<g transform="translate(936 305) scale(0.63)">{table(theme, laptop=True, lifted=False)}</g>',
        arrow("M426 274 C489 239 529 238 566 257", 490, 235, "$ rein push", theme),
        arrow("M695 257 C754 236 809 240 857 274", 776, 235, "$ rein pull", theme),
        pill(600, 365, "macOS  ↔  Windows   ·   your bucket, your keys", theme),
    ]
    return document(
        "Reinstate banner concept 4, Night Shift",
        "A dark-theme isometric room showing coding-agent session continuity between a desktop and laptop.",
        theme,
        "\n".join(body),
    )


CONCEPTS = {
    "01-room-continuum.svg": concept_one,
    "02-sealed-mechanism.svg": concept_two,
    "03-verified-relay.svg": concept_three,
    "04-night-shift.svg": concept_four,
}


def main() -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    for filename, builder in CONCEPTS.items():
        path = OUT / filename
        path.write_text(builder(), encoding="utf-8")
        print(path.relative_to(ROOT))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
