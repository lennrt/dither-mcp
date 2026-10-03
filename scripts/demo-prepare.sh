#!/bin/sh
# Prepare source copies in the temporary recording workspace.
set -eu
cd "$(dirname "$0")/.."
mkdir -p demos/work
find demos/work -maxdepth 1 -type f -delete
cp docs/assets/source/moon-garden.png demos/work/source.png
cp docs/assets/source/mineral-nocturne.png demos/work/mineral.png
