# Changes

## Request reliability

- Add context-aware encoding and palette mapping. Propagate request cancellation
  through GIF training, handwritten encoder loops, and encoded writes.
  Preserve successful export bytes and existing background-context APIs.
- Reject CLI flags that a command would silently ignore, including explicit
  zero and false values. Require `compare --palettes` for `--algorithm`.
- Preserve advanced studio options when a later slider or control changes.
  Retain malformed JSON and explain corrections without silently discarding edits.
  Reject explicit null options instead of resetting the draft on Apply.

## Palette workflows

- Compare 1–12 named palettes with one algorithm through MCP and the CLI.
  Contact sheets identify each cell's palette and retain the submitted settings.
- Extract source colors directly in the studio. Require an explicit preview
  before saving, and discard canceled or stale extraction responses.
  Keep the host connection usable when a canceled request responds late.
- Recover visible colors when a sparse image falls between extraction samples.
  Preserve bounded sampling and existing successful extraction results.
- Fit initial embedded previews within 512 × 512 pixels, including portraits.
  Preserve initial custom colors without adding a conflicting default palette.

## Browser verification follow-up

- Preserve JavaScript dollar sequences when building the embedded HTML. Add a
  regression check that compiles the delivered script.
- Use explicit button actions for preview and save in sandboxes without native
  form permission. Support Enter in input fields.
- Add an optional Playwright check with an isolated Go-backed host and publish
  actual desktop and mobile studio screenshots.

## 0.1.0-dev · MCP Apps studio

- Add `dither_studio`, the fourteenth MCP tool, for read-only dithered previews.
- Add `ui://dither/studio.html`, the third resource, with MIME type
  `text/html;profile=mcp-app` and official MCP Apps tool metadata.
- Provide algorithm, palette, and adjustment controls, Apply preview, zoom,
  palette swatches, and an explicit Save image action in supporting hosts.
- Use all 41 algorithms, 256 palettes, custom colors, and the shared Go engine.
- Fit omitted preview dimensions within 512 × 512 without enlarging the source.
  Bound all studio previews to 1,024 pixels per axis and 2 MiB PNG.
- Save through `dither_render` with a new relative destination and the exact
  recipe from the last successful preview.
- Bundle the official MCP Apps SDK, HTML, CSS, and JavaScript in the Go binary.
  Preserve local stdio operation and PNG plus JSON fallback for other hosts.
- Add studio documentation and feature coverage to the standalone showcase site.

## 0.1.0-dev · Clay visual style revision

- Use warm paper backgrounds, dark ink, serif typography, and clay accents.
- Add an original README banner and a warm clay image treatment.
- Apply the same visual style to the website and terminal recordings.
- Preserve the palette explorer, comparison controls, and supported-format tables.

## 0.1.0-dev · Media discovery revision

- Document supported inputs, outputs, codecs, and format limits in one guide.
- Add format tables to the README and showcase website.
- Add typed media capabilities to `dither_catalog` and `dither://capabilities`.
  The catalog distinguishes still images, animation, video, and print exports.
- Describe source GIF timing separately from generated animation settings.
- Test MP4, MOV, WebM, Matroska, and AVI input containers with local FFmpeg.
  Export checks cover codecs, dimensions, frame counts, padding, and audio removal.
- Test current and legacy MCP discovery, tool calls, metadata, and error behavior.

## 0.1.0-dev · Writing style revision

- Edit documentation, code comments, CLI help, MCP descriptions, and website text.
- Apply the Google developer style guide and STE-flavored writing rules.
- Use consistent terms, explicit conditions, and short, complete sentences.
- Preserve technical requirements, palette values, and runtime behavior.

## 0.1.0-dev · Palette library revision

- Expand the library from 22 to 256 distinct palettes across 16 categories.
- Preserve existing palette IDs and ordered color arrays for recipe replay.
- Add descriptions, tags, categories, and provenance to palette discovery.
- Add search, category filters, color-count filters, and stable pagination.
  The MCP tool `dither_palettes` and CLI command `palettes` use the same filters.
- Return the first 32 palettes for an empty discovery request.
  Responses include category counts and `next_offset` when another page exists.
  Set `limit` to `256` to request the complete library.
- Add an offline palette reference, a visual atlas, and image treatments.
- Add an accessible palette explorer and a recorded palette discovery demo.
- Use positive language in the README and showcase.
- Extend OpenSpec and Quint with palette compatibility and discovery contracts.

The initial local implementation included 41 dithering algorithms and 13 MCP tools.
It also includes a CLI, image masks, recipes, animation, local video processing,
and print exports. Original sample artwork and a standalone GitHub Pages showcase
accompany the implementation.
