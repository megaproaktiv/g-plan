#!/usr/bin/env python3
"""Compose the icon of the .dmg file: the pt-plan app icon on the top face of a
brushed-metal drive (dmg-drive.png, the drive artwork without any logo).

usage: compose-dmg-icon.py <app-icon-1024.png> <dmg-drive.png> <out.png>
"""
import sys
from PIL import Image

app_icon, drive, out = sys.argv[1:4]
base = Image.open(drive).convert("RGBA")
icon = Image.open(app_icon).convert("RGBA")

# The drive's top face is centred at about (50 %, 38 %) of the canvas. The app
# icon (824 of 1024 px is artwork) is scaled so it covers about half the face.
size = round(base.width * 0.558)
icon = icon.resize((size, size), Image.LANCZOS)
cx, cy = round(base.width * 0.501), round(base.height * 0.381)
base.alpha_composite(icon, (cx - size // 2, cy - size // 2))
base.save(out)
print(f"wrote {out} {base.size}")
