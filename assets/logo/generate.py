#!/usr/bin/env python3
"""Generates every derived Trux logo file from the approved mark.

The source of truth is the mark geometry below (MARK_PATHS and STROKE). This
script writes the mark in both colors, the wordmark, the app icon, and the
copies used by the docs site, the README, and the VS Code extension.

Requirements: Python 3 with cairosvg, Pillow, and fontTools, plus the Inter
variable font for the wordmark (pass its path with --inter if it is not in the
default location). Run from anywhere:

    python3 assets/logo/generate.py
"""
import argparse
import io
import json
import pathlib

import cairosvg
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer
from PIL import Image, ImageDraw

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent.parent
SITE = ROOT / "website"
VSCODE = ROOT / "tooling" / "vscode-trux"

# Brand colors.
BLUE = "#1E3A8A"        # primary: the mark on light backgrounds
LIGHT_BLUE = "#93C5FD"  # the mark on dark backgrounds
INK = "#1F2937"         # wordmark text on light backgrounds
WHITE = "#FFFFFF"

# The mark ("variant C", the compact t with an x crossbar), in 512x512 units.
MARK_PATHS = [
    "M256 208 V352 Q256 440 344 440",
    "M200 222 L312 334",
    "M312 222 L200 334",
]
STROKE = 40
# Unshifted, the stroked mark spans x 180..364 and y 188..460. The shift
# centers it in the 512x512 box of trux-logo.svg.
SHIFT_X = -16
SHIFT_Y = -68
# Bounds of the stroked mark after the shift: x 164..348, y 120..392.
MARK_TOP = 208 - STROKE / 2 + SHIFT_Y      # top of the stem's round cap
MARK_BASE = 440 + STROKE / 2 + SHIFT_Y     # bottom edge of the foot
MARK_BOX = (164, 120, 184, 272)
# A square around the mark for small icons, centered on it.
SQUARE_BOX = (116, 116, 280, 280)


def mark_group(stroke, extra=""):
    paths = "".join(f'<path d="{d}"/>' for d in MARK_PATHS)
    # var() only works in a style attribute, not in a presentation attribute.
    color = f'style="stroke:{stroke}"' if stroke.startswith("var(") else f'stroke="{stroke}"'
    return (
        f'<g transform="translate({SHIFT_X} {SHIFT_Y})" fill="none" {color} '
        f'stroke-width="{STROKE}" stroke-linecap="round" stroke-linejoin="round"{extra}>'
        f"{paths}</g>"
    )


def mark_svg(stroke, box=(0, 0, 512, 512), size=None, title="Trux", extra=""):
    x, y, w, h = box
    width, height = size or (w, h)
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{x} {y} {w} {h}" '
        f'width="{width}" height="{height}">\n  <title>{title}</title>\n  {mark_group(stroke, extra)}\n</svg>\n'
    )


def wordmark_path(inter_path, cap_height):
    """Returns (path data, advance width) for "Trux" in Inter SemiBold, with
    the baseline at y=0 and the given cap height in output units."""
    font = TTFont(inter_path)
    font = instancer.instantiateVariableFont(font, {"wght": 600, "opsz": 32})
    scale = cap_height / font["OS/2"].sCapHeight
    glyphs = font.getGlyphSet()
    cmap = font.getBestCmap()
    pen = SVGPathPen(glyphs)
    x = 0.0
    for ch in "Trux":
        name = cmap[ord(ch)]
        glyphs[name].draw(TransformPen(pen, (scale, 0, 0, -scale, x, 0)))
        x += glyphs[name].width * scale
    import re
    return re.sub(r"-?\d+\.\d+", lambda m: f"{float(m.group()):.1f}".rstrip("0").rstrip("."), pen.getCommands()), x


def logo_svg(mark_stroke, text_fill, inter_path, title="Trux"):
    """Mark plus "Trux" wordmark. The text's cap height equals the mark's
    height, so the mark's top lines up with the top of the T and the bottom of
    its foot with the baseline."""
    mx, my, mw, mh = MARK_BOX
    baseline = MARK_BASE
    cap = MARK_BASE - MARK_TOP
    gap = 60
    text, advance = wordmark_path(inter_path, cap)
    tx = mx + mw + gap
    width = round(tx + advance + 10 - mx)
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{mx} {my} {width} {mh}" '
        f'role="img" aria-label="{title}">\n  <title>{title}</title>\n  {mark_group(mark_stroke)}\n'
        f'  <path transform="translate({tx:.1f} {baseline:.1f})" fill="{text_fill}" d="{text}"/>\n</svg>\n'
    )


def icon_svg():
    """App icon: the white mark on a rounded dark blue square."""
    inner = mark_group(WHITE)
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">\n'
        "  <title>Trux</title>\n"
        f'  <rect width="512" height="512" rx="112" fill="{BLUE}"/>\n'
        f'  <g transform="translate(256 256) scale(1.12) translate(-256 -256)">{inner}</g>\n'
        "</svg>\n"
    )


def viewbox_size(svg):
    _, _, w, h = (float(v) for v in svg.split('viewBox="')[1].split('"')[0].split())
    return w, h


def png_height(svg, height):
    w, h = viewbox_size(svg)
    return png(svg, round(height * w / h), height)


def png(svg, width, height=None):
    data = cairosvg.svg2png(bytestring=svg.encode(), output_width=width, output_height=height or width)
    return Image.open(io.BytesIO(data)).convert("RGBA")


def write(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    if isinstance(data, Image.Image):
        data.save(path, optimize=True)
    else:
        path.write_text(data)
    print("wrote", path.relative_to(ROOT))


def og_image(inter_path, base_only=False):
    """1200x630 social card: white, a thin brand bar, the logo centered.
    With base_only, an empty card used behind the per-page titles."""
    card = Image.new("RGBA", (1200, 630), WHITE)
    draw = ImageDraw.Draw(card)
    draw.rectangle((0, 618, 1200, 630), fill=BLUE)
    if base_only:
        ghost = png_height(mark_svg("#EEF2FF", box=MARK_BOX), 420)
        card.alpha_composite(ghost, (1200 - ghost.width - 60, 150))
        return card.convert("RGB")
    img = png_height(logo_svg(BLUE, INK, inter_path), 170)
    card.alpha_composite(img, ((1200 - img.width) // 2, (618 - img.height) // 2))
    return card.convert("RGB")


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--inter", default="/usr/share/fonts/truetype/sand-box/google/Inter/Inter-VariableFont_opsz,wght.ttf")
    inter = parser.parse_args().inter

    # Shared assets (repository root).
    write(HERE / "trux-logo.svg", mark_svg(BLUE))
    for size in (1024, 512, 256, 64, 32):
        write(HERE / f"trux-logo-{size}.png", png(mark_svg(BLUE), size))
    write(HERE / "trux-logo-light.svg", mark_svg(LIGHT_BLUE))
    for size in (1024, 512, 256, 64, 32):
        write(HERE / f"trux-logo-light-{size}.png", png(mark_svg(LIGHT_BLUE), size))
    write(HERE / "trux-wordmark.svg", logo_svg(BLUE, INK, inter))
    write(HERE / "trux-wordmark-light.svg", logo_svg(LIGHT_BLUE, WHITE, inter))
    write(HERE / "trux-icon.svg", icon_svg())
    write(HERE / "trux-icon-512.png", png(icon_svg(), 512))
    write(HERE / "trux-og.png", og_image(inter))

    # Docs site. The header logos are inlined into the page, so their colors
    # come from CSS variables (set in assets/docs/scss/custom/colors/_trux.scss)
    # with the light-mode colors as fallbacks.
    css_mark = "var(--trux-logo-mark, %s)" % BLUE
    write(SITE / "assets/images/logos/mark.svg",
          mark_svg(css_mark, box=MARK_BOX, size=(None, None)).replace(' width="None" height="None"', ""))
    write(SITE / "assets/images/logos/logo.svg",
          logo_svg(css_mark, "currentColor", inter))
    # Used by Lotus Docs on the generated per-page social cards.
    # Lotus Docs scales it to 60px high; the transparent space below the logo
    # keeps the name about as large as with the old, taller mark.
    card_logo = png_height(logo_svg(BLUE, INK, inter), 126)
    padded = Image.new("RGBA", (card_logo.width, 180), (0, 0, 0, 0))
    padded.alpha_composite(card_logo, (0, 0))
    write(SITE / "assets/images/logos/logo.png", padded)
    write(SITE / "assets/opengraph/card-base-2.png", og_image(inter, base_only=True))
    write(SITE / "static/og-image.png", og_image(inter))

    # Favicons. favicon.svg follows the browser's color scheme; the raster
    # icons use the app icon so they stay visible on light and dark tabs.
    favicon = mark_svg(BLUE, box=SQUARE_BOX, size=(32, 32)).replace(
        "<title>Trux</title>",
        f"<title>Trux</title>\n  <style>g{{stroke:{BLUE}}}@media (prefers-color-scheme:dark){{g{{stroke:{LIGHT_BLUE}}}}}</style>")
    write(SITE / "static/favicon.svg", favicon)
    icon = icon_svg()
    write(SITE / "static/favicon-16x16.png", png(icon, 16))
    write(SITE / "static/favicon-32x32.png", png(icon, 32))
    write(SITE / "static/apple-touch-icon.png", png(icon, 180))
    write(SITE / "static/android-chrome-192x192.png", png(icon, 192))
    write(SITE / "static/android-chrome-512x512.png", png(icon, 512))
    ico = SITE / "static/favicon.ico"
    png(icon, 256).save(ico, sizes=[(16, 16), (32, 32), (48, 48)])
    print("wrote", ico.relative_to(ROOT))
    manifest = {
        "name": "Trux",
        "short_name": "Trux",
        "icons": [
            {"src": "android-chrome-192x192.png", "sizes": "192x192", "type": "image/png"},
            {"src": "android-chrome-512x512.png", "sizes": "512x512", "type": "image/png"},
        ],
        "theme_color": BLUE,
        "background_color": WHITE,
        "display": "standalone",
    }
    write(SITE / "static/site.webmanifest", json.dumps(manifest, indent=2) + "\n")

    # VS Code extension: marketplace icon and the .tx file icon.
    write(VSCODE / "images/icon.png", png(icon, 128))
    write(VSCODE / "images/file-icon-light.svg", mark_svg(BLUE, box=SQUARE_BOX, size=(16, 16)))
    write(VSCODE / "images/file-icon-dark.svg", mark_svg(LIGHT_BLUE, box=SQUARE_BOX, size=(16, 16)))


if __name__ == "__main__":
    main()
