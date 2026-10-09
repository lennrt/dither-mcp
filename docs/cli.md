# Command-line guide

See [formats and video](formats.md) for the complete support matrix, setup, examples, and tested codec combinations.

Build with `make build`. Run `./bin/dither-mcp help`. Flags follow the command.

All filesystem paths except `--root` are relative to that existing root. The CLI writes JSON results to stdout and diagnostics to stderr. Exit status is nonzero for a failed
operation. A batch result includes per-item failures. Inspect `items`.

```sh
./bin/dither-mcp catalog
./bin/dither-mcp palettes --query 'game boy' --max-colors 4
./bin/dither-mcp palettes --category ocean --limit 8
./bin/dither-mcp inspect --root . --input docs/assets/source/moon-garden.png
./bin/dither-mcp render --root . \
  --input docs/assets/source/moon-garden.png --output output/garden.png \
  --algorithm atkinson --palette gameboy --width 640 --pixel-scale 2
./bin/dither-mcp compare --root . \
  --input docs/assets/source/moon-garden.png --output output/compare.png \
  --algorithms floyd-steinberg,atkinson,bayer-8 --palette sepia --width 240
./bin/dither-mcp animate --root . \
  --input docs/assets/source/moon-garden.png --output output/wave.gif \
  --algorithm atkinson --palette gameboy --width 320 \
  --effect wave --frames 24 --fps 12
./bin/dither-mcp separate --root . \
  --input docs/assets/source/moon-garden.png --output output/plates.zip \
  --palette sepia --width 640 --dpi 300
```

Use a new output path on every run. The tool creates output directories automatically.
The tool never silently replaces existing files.

The `render`, `compare`, `animate`, `video` and `separate` commands accept the
common flags documented by `help`. Advanced JSON options merge into the configuration built from flags. JSON fields override flags with the same name. For fully explicit
requests, use `call`, whose arguments exactly match the MCP schemas:

```sh
./bin/dither-mcp call --root . dither_palette_extract \
  '{"input":"docs/assets/source/moon-garden.png","count":5}'
./bin/dither-mcp call --root . dither_render '{
  "input":"docs/assets/source/moon-garden.png",
  "output":"output/masked.png",
  "palette":"gameboy",
  "options":{
    "algorithm":"bayer-4","width":640,"pixel_scale":2,
    "mask":{"shape":"circle","x":160,"y":106,"radius":90},
    "effects":{"scanlines":0.15,"scanline_spacing":3}
  }
}'
```

For still PNG print metadata, use `--dpi 300`. For a sprite sheet, use `.png` output with `animate`. Optionally, specify `--columns 6`. To process a video, enable the local
adapter explicitly:

```sh
./bin/dither-mcp video --root . --allow-video \
  --input clip.mp4 --output output/clip.gif --width 320 \
  --algorithm bayer-8 --palette cga --frames 12 --fps 12 --start 0
```

The native video frame budget can require fewer frames for high-resolution
sources even when output width is small. See [MCP limits](mcp.md#limits-and-errors).

`call ... dither_preview` emits base64 PNG in JSON for direct CLI callers. The MCP
transport instead emits native image content alongside compact structured metadata.

## Studio previews

Use `call` to run the same read-only preview operation as the [MCP Apps studio](mcp-apps.md):

```sh
./bin/dither-mcp call --root . dither_studio '{
  "input":"docs/assets/source/moon-garden.png",
  "palette":"oat-and-ink",
  "options":{"algorithm":"atkinson","pixel_scale":2,"seed":42}
}'
```

This call returns PNG data, preview metadata, and a resolved recipe as JSON. It creates no output file. Omitted dimensions fit within 512 × 512 pixels without enlarging the source. Explicit and aspect-derived dimensions must fit within 1,024 pixels per axis, and the PNG must fit within 2 MiB. The tool accepts `input`, `palette` or `colors`, `options`, and an optional `mask_input`. It accepts neither an output path nor a recipe file.

The interactive controls appear when an MCP host supports MCP Apps and local stdio servers. The CLI returns the data for direct use. To save a matching PNG, pass the successful preview's resolved recipe to `dither_render` with the same input, mask input when needed, and a new relative output path. Add `expected_source_sha256` and any `expected_mask_sha256` from the preview metadata to detect changed inputs. You can set explicit export width and height while preserving the other recipe options. See [studio export behavior](mcp-apps.md#preview-and-save-behavior).

## Motion settings

Source GIF animation preserves source timing when you select `--effect source` or that effect applies by default. Still `render` processes only the first frame.
The CLI and MCP service share frame controls and all limits.

## Explore the palette library

`palettes` accepts `--query`, `--category`, `--min-colors`, `--max-colors`,
`--limit` and `--offset`. The default page size is 32, with a maximum of 256.
The service orders results by ID and includes `next_offset` when more matches exist.

`--limit 256` exports the full built-in collection. A category list and counts
accompany every page, including empty search results. All search behavior matches MCP `dither_palettes`. See the [complete palette reference](palettes.md).
