#!/bin/sh
# Recreate the artwork, image studies, normalization comparison, recipes, and GIF.
set -eu
cd "$(dirname "$0")/.."
go run ./scripts/showcase-source
go run ./scripts/brand-banner
go run ./scripts/showcase-render
go run ./scripts/palette-atlas
go run ./scripts/normalization-showcase
go build -o bin/dither-mcp ./cmd/dither-mcp
rm -f docs/assets/garden-wave.gif
./bin/dither-mcp animate --root . \
  --input docs/assets/source/moon-garden.png \
  --output docs/assets/garden-wave.gif \
  --algorithm atkinson --palette gameboy --width 480 --pixel-scale 2 \
  --effect wave --frames 24 --fps 12

go run ./scripts/showcase-render --first-frame docs/assets/garden-wave.gif
