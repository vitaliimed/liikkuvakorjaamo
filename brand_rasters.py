#!/usr/bin/env python3
"""Regenerate website/email raster logos from the canonical two-colour SVG originals.

Requires Pillow and CairoSVG. Intended for local/Cloud Shell release prep, not
for service startup. All assets are overwritten only after rasterization works.
"""
from __future__ import annotations

import hashlib
import io
import sys
from pathlib import Path

try:
    import cairosvg
    from PIL import Image
except ImportError as exc:
    raise SystemExit(
        "Missing rasterizer: install dependencies in a virtual environment: "
        "python3 -m venv /tmp/liikkuva-brand-venv && "
        "/tmp/liikkuva-brand-venv/bin/pip install pillow cairosvg; "
        "then run /tmp/liikkuva-brand-venv/bin/python brand_rasters.py"
    ) from exc

ASSETS = Path(__file__).resolve().parent / "source" / "static" / "img"
GREEN = "#23372b"
ORANGE = "#f97315"
LOGOS = (
    "logo-white", "logo-dark", "logo-green",
    "logo-transparent-light", "logo-transparent-dark",
)


def render(svg: Path, width: int, height: int) -> Image.Image:
    text = svg.read_text(encoding="utf-8").lower()
    if ORANGE not in text:
        raise ValueError(f"Expected exact orange arrow in {svg}")
    return Image.open(
        io.BytesIO(
            cairosvg.svg2png(
                url=str(svg), output_width=width, output_height=height
            )
        )
    ).convert("RGBA")


def save_png(img: Image.Image, path: Path) -> None:
    img.save(path, format="PNG", optimize=True)


def report(path: Path) -> None:
    digest = hashlib.sha256(path.read_bytes()).hexdigest()[:16]
    print(f"{path.name:<32} {path.stat().st_size:>9} bytes sha256:{digest}")


def main() -> None:
    for name in LOGOS:
        svg = ASSETS / f"{name}.svg"
        img = render(svg, 1600, 576)
        png = ASSETS / f"{name}.png"
        save_png(img, png)
        report(png)
        webp = ASSETS / f"{name}.webp"
        if webp.is_file():
            img.save(webp, format="WEBP", lossless=True, method=6)
            report(webp)

    favicon_svg = ASSETS / "favicon.svg"
    if GREEN not in favicon_svg.read_text(encoding="utf-8").lower():
        raise ValueError("favicon.svg does not contain the canonical green")

    # Browser icons, email link previews, mobile home-screen icons.
    square_outputs = {
        "favicon-16x16.png": 16,
        "favicon-32x32.png": 32,
        "apple-touch-icon.png": 180,
        "icon-192.png": 192,
        "icon-192x192.png": 192,
        "icon-512.png": 512,
        "icon-512x512.png": 512,
    }
    for filename, size in square_outputs.items():
        output = ASSETS / filename
        save_png(render(favicon_svg, size, size), output)
        report(output)

    # Keep the entire symbol inside the central maskable safe area.
    mask_size = 512
    maskable = Image.new("RGBA", (mask_size, mask_size), GREEN)
    symbol_size = 364
    symbol = render(favicon_svg, symbol_size, symbol_size)
    maskable.alpha_composite(
        symbol, ((mask_size - symbol_size) // 2,) * 2
    )
    output = ASSETS / "icon-maskable-512.png"
    save_png(maskable, output)
    report(output)

    master = render(favicon_svg, 256, 256)
    ico = ASSETS / "favicon.ico"
    master.save(
        ico, format="ICO", sizes=[(16, 16), (32, 32), (48, 48), (64, 64)]
    )
    report(ico)
    print(f"Raster assets regenerated from SVG with green {GREEN} and orange {ORANGE}")


if __name__ == "__main__":
    main()
