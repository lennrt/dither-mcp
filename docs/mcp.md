# MCP interface

Run `dither-mcp mcp --root /absolute/path/to/images`. The root must exist.
Go [mcp-go](https://github.com/mark3labs/mcp-go) v1.1.1 provides protocol negotiation, stdio transport, tool schemas, resources, and prompts. The server does not start an HTTP listener.

All tool arguments and results are JSON objects. Tools advertise input and output schemas. The server strictly rejects unknown argument fields. Read the exact [generated tool descriptors](tool-schemas.json), or use `tools/list` on a live connection. Regenerate descriptors with `go run ./scripts/schema`.

See [formats and video](formats.md) for the complete support matrix, setup, examples, and tested codec combinations.

The server provides **14 tools**, **3 resources**, and **3 prompts**. The read-only `dither_studio` tool adds an optional [MCP Apps](https://modelcontextprotocol.io/extensions/apps/overview) view. A host that supports MCP Apps and local stdio servers can display its image controls inside the conversation. Other hosts receive the same PNG preview and structured JSON. The server keeps its local Go runtime and stdio transport.

## Tools

| Tool | Inputs | Result |
|---|---|---|
| `dither_catalog` | `{}` | Algorithms, formats, defaults, effects and limits |
| `dither_palettes` | `query`, `category`, `min_colors`, `max_colors`, `limit`, `offset` (all optional) | Searchable palette pages with exact colors, descriptions, tags, provenance and category counts |
| `dither_inspect` | `input` | Upright dimensions, normalization details, first-frame alpha, frame count, size, and SHA256 |
| `dither_preview` | `input`, optional `width` | Structured metadata plus native MCP PNG image content |
| `dither_studio` | `input`, optional `palette` or `colors`, `options`, and `mask_input` | In-memory dithered PNG preview, resolved recipe, and optional interactive studio |
| `dither_palette_extract` | `input`, `count` (default 8) | Deterministic array of up to the requested 2–256 colors |
| `dither_render` | Render request | Artifact and recipe |
| `dither_compare` | Render request, `algorithms`, optional `columns` | Contact sheet plus indexed cell rectangles |
| `dither_batch` | `items`: 1–32 render requests | Each item's artifact or error, in input order |
| `dither_animate` | Render request, `effect`, `frames`, `fps`, `amplitude`, `columns` | GIF or PNG sprite sheet |
| `dither_video` | Render request, `frames`, `fps`, `start` | MP4, WebM, GIF or PNG sprite sheet |
| `dither_separate` | Render request | ZIP of spot-color PNG plates and manifest |
| `dither_recipe_save` | `output`, versioned `recipe` | JSON artifact |
| `dither_recipe_load` | `input` | Validated versioned recipe |

Render requests use `input`, `output`, either `palette` or `colors`, and nested
`options`. Optional `recipe` replaces all inline options and palette selection.

`mask_input` supplies a local image mask. `format` overrides the filename extension. Otherwise, the service infers the format from the extension. `dpi` embeds PNG pHYs metadata from 36 through 2400. Zero omits this metadata. An explicit format does not rename the extension.

```json
{
  "input": "photo.png",
  "output": "studies/atkinson.png",
  "colors": ["#251d32", "#ab4e3b", "#e6ad62", "#f0e7d3"],
  "options": {
    "algorithm": "atkinson",
    "width": 960,
    "pixel_scale": 2,
    "contrast": 1.1,
    "gamma": 1.15,
    "seed": 42,
    "serpentine": true
  }
}
```

The result records a relative path, actual dimensions, encoded byte count, and SHA256. It also records the still-image recipe, engine version, operation name, and complete submitted parameters. The result separates motion and comparison parameters from the reusable still recipe. Replay requires the same input files and engine version. A seed does not guarantee byte compatibility across future releases or FFmpeg versions.

A comparison's `cells` supply labels and exact rectangles. The contact sheet contains no embedded typography. `dither_inspect` can hash source files when a caller needs to keep a complete input manifest.

Solid-color sources cannot produce two distinct extracted colors. Extraction returns an actionable error instead of an unusable one-color render palette.

## Guard a render against changed inputs

Render requests accept optional `expected_source_sha256` and `expected_mask_sha256` fields. Each nonempty value must contain 64 hexadecimal digits. The studio supplies these values automatically from its accepted preview. Other callers can copy `source_sha256` and `mask_sha256` from the studio result. These fields belong to requests, so saved recipes remain portable.

The service verifies the exact bytes it will decode. A mismatch returns `isError: true` with `structuredContent.error.code` set to `source_changed` or `mask_changed`, plus `message`, `path`, `expected_sha256`, and `actual_sha256`. If a guarded file cannot be read, the same error code is returned with an empty `actual_sha256`. No artifact is published. Callers must obtain a fresh preview before retrying with new expectations. An empty or omitted expectation keeps the existing unguarded behavior. `expected_mask_sha256` requires `mask_input`.

The same guards apply to `dither_compare`, each `dither_batch` item, `dither_animate`, `dither_video`, and `dither_separate` because these tools share render requests. Video verifies its source buffer before staging that exact buffer for local decoding. A batch reports a changed input in the affected item's `error` string; earlier successful items remain published. The text fallback for a guarded failure starts with `source_changed:` or `mask_changed:`. SHA256 expectations accept upper- or lowercase hexadecimal digits; studio fingerprints use lowercase.

## MCP Apps studio

Ask your agent to open `dither_studio` for a local image:

```json
{
  "input": "photo.png",
  "palette": "oat-and-ink",
  "options": {
    "algorithm": "atkinson",
    "pixel_scale": 2,
    "contrast": 1.1,
    "seed": 42
  }
}
```

The studio uses all 41 algorithms, 256 palettes, custom colors, and the shared engine options. The embedded controls let you select an algorithm and palette, adjust the image, apply a preview, zoom, and inspect swatches.

`dither_studio` reads the source and renders in memory. It accepts no output path and creates no file. When dimensions are omitted, the preview fits within 512 × 512 pixels without enlarging the source. Explicit or aspect-derived dimensions must fit within 1,024 pixels per axis. Encoded PNG data must fit within 2 MiB. Studio seeds must be safe JavaScript integers, from −9,007,199,254,740,991 through 9,007,199,254,740,991.

Select **Save image** and supply a new relative output path to write a file. The filename extension selects the still-image output format. The app calls `dither_render` with the accepted recipe, source and mask fingerprints, and selected preview, source, or custom export dimensions. Only width and height change for a different export size. Source size includes EXIF orientation and crop. A different resolution can change the dither pattern. Changing a control does not change that saved recipe until **Apply preview** succeeds. Existing destinations remain protected by the service's atomic publication rules.

The tool descriptor associates this view with `ui://dither/studio.html` through `_meta.ui.resourceUri`. The resource has MIME type `text/html;profile=mcp-app`. The original tools keep their normal tool results. Read the [MCP Apps guide](mcp-apps.md) for the host contract, build workflow, and verified behavior.

## Input normalization and inspection

Still-image tools normalize input orientation and supported color metadata automatically. JPEG, PNG, WebP, and TIFF can supply EXIF orientation and embedded RGB ICC profiles. BMP V5 can supply an embedded profile from the same ICC subset. Image masks use the same loader. The service applies orientation and converts supported colors to sRGB before cropping or resizing.

`dither_inspect` returns upright `width` and `height`. The `normalization` object supplies these fields:

| Field | Meaning |
|---|---|
| `stored_width`, `stored_height` | Dimensions before EXIF orientation |
| `exif_orientation` | EXIF value 1–8. Missing orientation uses 1. |
| `orientation_applied` | Whether an orientation transform ran |
| `color_source` | Identifier for the selected source color information |
| `working_space` | `srgb` |
| `color_converted` | Whether color conversion ran |
| `icc_sha256` | Digest of the selected embedded ICC profile. Absent when other color metadata takes precedence. |

The top-level `sha256` identifies the original input file. Inspect an image before selecting crop coordinates. The accepted profile subset, PNG precedence, metadata limits, and output behavior appear in [the format guide](formats.md#orientation-and-input-color).

## Palette discovery

The [palette library](palettes.md) contains 256 named palettes across 16 categories.
Search metadata and hex colors with case-insensitive whitespace-separated AND terms. Combine a query with a category and color-count range. Use a returned ID directly in a render recipe. The service normalizes category slugs to lowercase.

```json
{"query":"game boy","category":"retro","max_colors":4,"limit":8}
```

An empty argument object returns the first 32 palettes in ascending ID order.
Choose `limit` from 1 through 256. Zero selects the default. Follow `next_offset`
for the next page.

The response includes `total` (entire library), `matched`
(filtered total), `count` (page size), `offset`, `limit`, and global category counts.
The final page omits `next_offset`. Any nonnegative offset is valid. An offset beyond the results returns an empty array.

Query text permits up to 256 UTF-8
bytes. Color bounds use 2 and 256 when omitted or zero. The minimum must be at most the maximum. Unknown categories and invalid bounds return actionable tool errors.

All original 22 palette IDs preserve their colors. New original palettes carry `origin: "original"`. Referenced and approximate palettes state their provenance and source links. `colors` always contains normalized, unique opaque hex colors.
The binary embeds the catalog. Discovery uses local data.

## Image options

| Option | Meaning / range |
|---|---|
| `algorithm` | Exact ID from catalog. Default: `floyd-steinberg`. |
| `width`, `height` | Zero or omitted: preserve source dimension/aspect. One supplied: preserve aspect. Two supplied: resize exactly. |
| `pixel_scale` | 0/1: native grid. 2–256: reduce, dither, and enlarge to requested dimensions. |
| `resize_filter` | `nearest` or `bilinear` |
| `brightness` | Offset −1..1, default 0 |
| `contrast`, `saturation` | 0..4, default 1 |
| `gamma` | 0.1..8, default 1 |
| `threshold` | 0..1, default 0.5 |
| `strength` | 0..2, default 1 |
| `seed` | Signed 64-bit integer. Default: 0. Raw MCP JSON preserves integer precision. Studio requests use the safe JavaScript integer range. |
| `serpentine` | Alternate diffusion rows. Default: false. |
| `invert`, `grayscale` | Boolean controls |
| `color_space` | `srgb` or `linear-rgb` |
| `crop` | `{x,y,width,height}` wholly inside the upright source after EXIF orientation |
| `mask` | `rectangle`, `circle`, or `image`. Optional: `invert`. |
| `effects` | `scanlines`, `crt`, `noise` (0..1), `glitch` (0..1024), `pixel_sort`, `pixel_sort_threshold` (0..1), `scanline_spacing` (0..256) |

Optional scalar adjustments preserve explicit zero. The service accepts `contrast:0` but rejects `gamma:0`. All numbers must be finite.

Mask coordinates refer to the
reduced processing grid, before pixel-scale enlargement. Rectangle masks use x/y as the top-left corner. Circle masks use x/y as the center, with `radius`.

An image mask uses luminance × alpha ≥ 0.5 after nearest scaling. Supply its path in `mask_input`. Recipes store `shape:"image"`. Supply the external file on each render.

Masks preserve adjusted source colors outside the selection. Diffusion does not
carry error across that boundary. Post-effects may introduce non-palette colors.

## Motion and video

`dither_animate` defaults to a wave for stills and `source` for GIF input. Effects
are `wave`, `orbit`, `pulse`, `noise`, `palette-cycle`, and `source`. Wave, orbit, and pulse are periodic. Noise changes the deterministic seed each frame. The GIF repeats, but its endpoint is not a smooth temporal transition. Palette cycle permutes the
rendered ink indices.

The defaults are 24 frames, 12 fps, and amplitude 0.15. The default width is 480 when both dimensions are omitted or zero.
Explicit amplitude zero currently selects that default. A source GIF preserves
its original frame count, centisecond delays, loop count and composited disposal
behavior. `frames` and `fps` do not override source-GIF timing.

GIF timing distributes centisecond rounding error: 24 frames at 12 fps lasts
exactly two seconds. GIF has binary transparency. JPEG, PBM, and ASCII composite against white. PNG and SVG preserve partial alpha.

A sprite sheet is PNG and defaults to six columns. It places frames in row-major order and has transparent unused cells. Its artifact width and height describe the whole sheet. Frame count describes its contents.

Install FFmpeg/ffprobe on PATH. To enable video, start the service with `--allow-video`.

The input must be MP4/MOV with an ftyp header, WebM/Matroska, or AVI.
The service rejects playlist inputs and URL sources. `start` is in seconds (0–86400). The defaults are 48 frames and 12 fps.

Video emits no audio. Encoded MP4/WebM pad odd
width/height with one pixel for yuv420p. Native decoded frames and rendered frames
must each fit the aggregate budget. Codec versions can change video bytes.

## Print

`dither_separate` permits at most 16 inks, and rejects masks and post-effects to
keep palette membership exact. It writes `ink-NN-RRGGBB.png` plates and a JSON
manifest in one ZIP. Black means that ink is present. White means unprinted. Transparent pixels remain unprinted. Nonzero source alpha marks ink coverage.

These are binary spot-color plates, not color-managed CMYK separations. PNG DPI
changes physical resolution metadata, not pixel dimensions. Input ICC normalization does not
provide output device profiles, press calibration, paper simulation, or laser-device control.

## Limits and errors

The service enforces these limits:

| Resource | Limit |
|---|---|
| Input file or encoded artifact | 32 MiB each |
| Expanded ICC profile or EXIF payload | 4 MiB each |
| ICC tags or tone-curve samples | 256 tags and 65,536 samples per curve |
| EXIF IFD0 entries | 4,096 |
| Image | 16,384 pixels per axis and 16,777,216 pixels |
| Motion | 120 frames and 67,108,864 aggregate pixels |
| Comparison | 12 candidates |
| Batch | 32 items |
| Palette | 2–256 unique opaque colors |
| Source preview | 1,024 pixels per axis and 2 MiB PNG. Default width: 512. |
| Studio preview | 1,024 pixels per axis and 2 MiB PNG. Default: fit within 512 × 512 without enlargement. |
| Tool arguments | 1 MiB |
| MCP line | 2 MiB |

The service checks GIF limits before allocating all frames. Sprite sheets also obey still-image bounds. Malformed recognized container metadata and unsupported selected profiles return tool errors before publication.

The service admits one operation at a time. Each call has a two-minute timeout,
including queue time. Engine loops and subprocesses observe cancellation. Standard-library codecs have byte and dimension bounds, but are not preemptible.

Cancellation observed before publication leaves no artifact. A cancellation
racing after the final check can return a completed artifact. No unobserved
background job continues after a tool handler returns.

Tool failures are MCP `isError` results, allowing the agent to correct the
request. The protocol session survives ordinary tool errors. Oversized MCP
messages terminate the session.

A batch commits per item and reports errors. A timeout can end the returned list early. Successful earlier items remain.
There are no overwrite, delete, upload, or network-import tools.

## Resources and prompts

- `dither://capabilities`: the live machine-readable catalog.
- `dither://workflow`: agent workflow, privacy and parameter guidance.
- `ui://dither/studio.html`: self-contained MCP Apps studio with MIME type `text/html;profile=mcp-app`.
- `art_director`: inspect, compare, preview, render, save a recipe.
- `prepare_print`: choose inks, preview, generate plates.
- `make_loop`: check budgets, animate and report frame dimensions.

Run `make demo-smoke` to exercise an initialized real stdio subprocess with no
model provider. See [security](security.md) before choosing a workspace root.

## MCP implementation and references

The server uses `mcp-go` v1.1.1 for protocol version handling, stdio framing, dispatch, resources, and prompts. The application supplies typed tool schemas and runtime validation. It does not start an HTTP listener.

Successful tools return structured JSON and a serialized JSON text block. Preview tools also return native PNG image content. Protocol tests validate structured results against each advertised output schema. These choices follow the [MCP tools guidance](https://modelcontextprotocol.io/specification/2026-07-28/server/tools).

The studio adds the [MCP Apps extension](https://modelcontextprotocol.io/extensions/apps/overview). Its HTML bundles CSS and the official `@modelcontextprotocol/ext-apps` SDK. The host mediates app tool calls and controls the embedded view. The app loads no external scripts or assets and needs no runtime Node.js process.

Tool descriptions identify defaults, bounds, required dependencies, and output behavior. Annotations distinguish reads from file writes and describe local, non-destructive behavior. Clients must treat annotations as untrusted unless they trust the server. Annotations do not replace authorization or input validation. See [tool annotations](https://modelcontextprotocol.io/specification/2026-07-28/server/tools#tool).

The client launches the local binary as a subprocess. The server reserves stdout for newline-delimited protocol messages and sends diagnostics to stderr. This follows the [MCP stdio transport rules](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio). The SDK handles protocol discovery and version compatibility. Tests cover the legacy 2025-11-25 `initialize` path and the 2026-07-28 `server/discover` path. Modern requests carry mandatory metadata, and modern results include `resultType`.

The MCP host controls consent to execute the binary and exposes tool calls to the user. Use a trusted binary and a narrow workspace root. The service enforces rooted paths, strict arguments, resource limits, and immutable destinations. These controls address local execution risks described in the [MCP security guidance](https://modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices). See [the security model](security.md) for the exact boundary and its limitations.

The project tests specific protocol paths and tool behavior. It does not claim external MCP certification or prove every client implementation compatible.
