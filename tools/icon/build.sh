#!/bin/sh
# Renders the app icon into build/appicon.png, with a soft shadow like macOS icons.
set -e
cd "$(dirname "$0")/../.."
tmp=$(mktemp -d)
python3 tools/icon/icon.py > "$tmp/icon.svg"
magick -background none "$tmp/icon.svg" "$tmp/tile.png"
magick "$tmp/tile.png" \( +clone -background black -shadow 40x16+0+14 \) +swap -background none -layers merge +repage -gravity center -extent 1024x1024 PNG32:build/appicon.png
rm -r "$tmp"
