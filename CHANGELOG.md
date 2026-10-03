# Changes

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

The initial local implementation includes 41 dithering algorithms and 13 MCP tools.
It also includes a CLI, image masks, recipes, animation, local video processing,
and print exports. Original sample artwork and a standalone GitHub Pages showcase
accompany the implementation.
