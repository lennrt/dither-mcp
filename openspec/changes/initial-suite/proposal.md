# Agentic local dithering

## Why

Dithering tools offer many creative choices. Most workflows require repeated manual adjustment in a graphical editor. Agents need discoverable algorithms, explicit palettes, reproducible settings, and comparison artifacts. They need files they can inspect and refine without uploading images to a rendering service.

## What changes

- Build a Go library, command-line application, and stdio MCP server sharing one image pipeline.
- Expose catalog discovery, local image inspection, palette extraction, single renders, comparisons, and batch recipes.
- Support print exports, GIF processing, generated loops, sprite sheets, and optional local video processing.
- Add typed validation, root-confined relative paths, bounded decoding and output dimensions, cancellation, deterministic seeds, and atomic immutable output publication.
- Define behavioral contracts with OpenSpec and finite executable safety models with Quint before implementing the service.
- Ship original procedural image examples, real terminal recordings, operation guides, and a separate static showcase suitable for GitHub Pages.

## Capabilities

### New capabilities

- `local-workspace`: rooted local IO, resource limits, publication safety, cancellation.
- `image-pipeline`: algorithm registry, color quantization, transformations, masks, effects, and exports.
- `agent-workflows`: MCP discovery, structured tool calls, recipes, comparison, and batch behavior.
- `motion-print`: GIF/video operations, generated loops, sprite sheets, and print separation.
- `delivery-quality`: reproducible validation, documentation, original samples, and an offline static showcase.

### Modified capabilities

None. This is a new local implementation.

## Impact

The runtime is a Go binary with no required external service. FFmpeg is optional for video. Node tooling is development-only for the specifications, and VHS is development-only for recordings. The browser references inspire a vocabulary rather than promise byte-for-byte parity. This release excludes live camera/audio, learned segmentation, GPU 3D rendering, cloud libraries, and an interactive image editor. The feature matrix explicitly records these boundaries.
