#!/usr/bin/env bash
#
# render-icon.sh — build the pt-plan application icon from its sources and
# install it into the (git-ignored) Wails build folder src/app/build.
#
# Sources (this folder):
#   appicon.svg        full artwork (rounded square) -> appicon.png -> icons.icns / icon.ico
#   layer.svg          foreground only, for the macOS Icon Composer bundle -> Assets.car
#   icon.json          Icon Composer settings (background fill, shadow, ...)
#   dmg-drive.png      metal drive artwork (no logo) for the .dmg file icon
#   compose-dmg-icon.py puts the app icon on the drive
#
# Needs macOS with: rsvg-convert (brew install librsvg), python3 + Pillow,
# sips, iconutil, and wails3 (which calls actool to build Assets.car).
#
# Usage: ./render-icon.sh        (then rebuild the app: see documentation/ops/release.md)
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD="$HERE/../../src/app/build"
cd "$HERE"

for tool in rsvg-convert python3 sips iconutil wails3; do
  command -v "$tool" >/dev/null 2>&1 || { echo "error: '$tool' not found" >&2; exit 1; }
done
python3 -c "import PIL" 2>/dev/null || { echo "error: python3 needs Pillow (pip3 install pillow)" >&2; exit 1; }
[ -d "$BUILD" ] || { echo "error: $BUILD does not exist (run 'wails3 task common:update:build-assets' first)" >&2; exit 1; }

echo "1/4 app icon png"
rsvg-convert -w 1024 -h 1024 appicon.svg -o appicon.png
cp appicon.png "$BUILD/appicon.png"

echo "2/4 Icon Composer bundle (build/appicon.icon)"
mkdir -p "$BUILD/appicon.icon/Assets"
rm -f "$BUILD/appicon.icon/Assets/"*.svg
cp layer.svg "$BUILD/appicon.icon/Assets/ptplan_plan.svg"
cp icon.json "$BUILD/appicon.icon/icon.json"

echo "3/4 .dmg file icon"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
python3 compose-dmg-icon.py appicon.png dmg-drive.png "$TMP/dmg-file-icon.png"
cp "$TMP/dmg-file-icon.png" "$BUILD/darwin/dmg-file-icon.png"
SET="$TMP/dmg-file-icon.iconset"; mkdir -p "$SET"
for s in 16 32 128 256 512; do
  sips -z $s $s "$TMP/dmg-file-icon.png" --out "$SET/icon_${s}x${s}.png" >/dev/null
  sips -z $((s*2)) $((s*2)) "$TMP/dmg-file-icon.png" --out "$SET/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$SET" -o "$BUILD/darwin/dmg-file-icon.icns"

echo "4/4 icons.icns, icon.ico and Assets.car (wails3 generate icons)"
( cd "$BUILD" && wails3 generate icons \
    -input appicon.png \
    -macfilename darwin/icons.icns \
    -windowsfilename windows/icon.ico \
    -iconcomposerinput appicon.icon \
    -macassetdir darwin )

# icons.icns serves macOS before 26 and is the fallback elsewhere. When actool is present
# wails3 overwrites it with a REDUCED set (nothing above 256 px), which looks soft in Finder's
# large previews. Always write the complete set (16 ... 512@2x = 1024 px) from appicon.png.
ISET="$TMP/icons.iconset"; mkdir -p "$ISET"
for s in 16 32 128 256 512; do
  sips -z $s $s appicon.png --out "$ISET/icon_${s}x${s}.png" >/dev/null
  sips -z $((s*2)) $((s*2)) appicon.png --out "$ISET/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$ISET" -o "$BUILD/darwin/icons.icns"

# Assets.car (the macOS 26 Icon Composer icon) is compiled by actool, which only
# ships with a full Xcode. Without it wails3 skips the step silently and leaves any
# OLD Assets.car behind, which would put the previous icon back into the .app
# (Info.plist points at it with CFBundleIconName). Never keep a stale one.
if xcrun --find actool >/dev/null 2>&1; then
  [ "$BUILD/darwin/Assets.car" -nt "$HERE/appicon.svg" ] \
    || { echo "error: actool exists but Assets.car was not rebuilt" >&2; exit 1; }
  echo "Assets.car rebuilt with actool"
else
  if [ -f "$BUILD/darwin/Assets.car" ]; then
    mv "$BUILD/darwin/Assets.car" "$BUILD/darwin/Assets.car.stale"
    echo "warning: no Xcode (actool): removed the stale Assets.car (kept as Assets.car.stale)."
  fi
  echo "warning: no Xcode (actool): the app uses icons.icns only. On macOS 26+ that icon is shown"
  echo "         inside a system plate; install Xcode and re-run this script for the full-size icon."
  echo "         Then run 'wails3 task common:update:build-assets' so Info.plist matches the files."
fi

echo "done: icons installed in $BUILD"
