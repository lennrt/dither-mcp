#!/bin/sh
# Record locally. This script never publishes media.
set -eu
cd "$(dirname "$0")/.."
for tool in vhs ffmpeg ffprobe ttyd; do
  command -v "$tool" >/dev/null 2>&1 || { printf 'Missing required tool: %s\n' "$tool" >&2; exit 1; }
done
go build -o bin/dither-mcp ./cmd/dither-mcp
go build -o bin/dither-summary ./scripts/demo-summary
export PATH="$PWD/bin:$PATH"
vhs validate 'demos/*.tape'
if [ "$#" -eq 0 ]; then
  set -- render compare mcp palettes
fi
for name in "$@"; do
  case "$name" in render|compare|mcp|palettes) ;; *) printf 'Unknown demo: %s\n' "$name" >&2; exit 1 ;; esac
  vhs "demos/$name.tape"
  ffprobe -v error -select_streams v:0 -show_entries stream=width,height,codec_name -of json "docs/assets/$name.mp4"
  ffmpeg -v error -y -sseof -2 -i "docs/assets/$name.mp4" -frames:v 1 "docs/assets/demo-$name-poster.png"
done
printf '%s\n' 'Recordings are ready under docs/assets/.'
