#!/bin/sh
# Copy local assets to the static Pages site.
set -eu
cd "$(dirname "$0")/.."
destination=${1:-../dither-mcp-site}
if [ ! -f "$destination/index.html" ]; then
  printf '%s\n' 'The destination must be an existing dither-mcp showcase site.' >&2
  exit 1
fi
mkdir -p "$destination/assets"
cp docs/assets/source/moon-garden.png "$destination/assets/moon-garden.png"
cp docs/assets/banner.svg docs/assets/banner.png "$destination/assets/"
cp docs/assets/normalization.png "$destination/assets/"
cp docs/assets/mcp-app-workflow.svg "$destination/assets/"
cp docs/assets/mcp-app-studio.png docs/assets/mcp-app-studio-mobile.png "$destination/assets/"
for name in garden-clay garden-gameboy garden-cga mineral-ember study-atkinson study-floyd-steinberg study-bayer-8 study-halftone study-blue-noise; do
  cp "docs/assets/$name.png" "$destination/assets/$name.png"
done
cp docs/assets/garden-wave.gif docs/assets/garden-wave-still.png "$destination/assets/"
cp docs/assets/palettes.json docs/assets/palette-atlas.png docs/assets/palette-treatments.png docs/assets/GO-FONTS-LICENSE.txt "$destination/assets/"
for name in copperplate midnight-orchid alpine-morning tidal-glass malachite autumn-orchard ultraviolet-city pistachio-rose black-sesame nebula-rose brutalist-sun oat-and-ink; do
  cp "docs/assets/palette-$name.png" "$destination/assets/"
done
for name in render compare mcp palettes; do
  cp "docs/assets/$name.mp4" "$destination/assets/$name.mp4"
  cp "docs/assets/demo-$name-poster.png" "$destination/assets/demo-$name-poster.png"
done
printf 'Updated %s/assets\n' "$destination"
