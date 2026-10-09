# Feature coverage

The reference tools inspire the range of local dithering features. This project implements its own Go pipeline and agent workflows. Equal catalog counts do not imply identical pixels or full feature parity with every reference. Ditherer, in particular, provides a much broader environment for live visual effects.

See [formats and video](formats.md) for the complete support matrix, setup, examples, and tested codec combinations.

The live `dither_catalog` and `dither_palettes` results are the authoritative lists of accepted identifiers. The following table describes the features available in this release.

| Capability | dither-mcp implementation | Boundary |
|---|---|---|
| Agent interface | 14 MCP tools, three resources, three prompts. Matching CLI workflows | Stdio transport. Host controls model interaction |
| MCP Apps studio | Embedded algorithm, palette, and adjustment controls. Apply preview, zoom, inspect swatches, and explicitly save an image | Requires host support for MCP Apps and local stdio. Other hosts receive PNG and structured JSON. Preview is read-only. Save preserves accepted settings with explicit export dimensions and checks source/mask fingerprints |
| Local processing | Go image engine, rooted local files, immutable output artifacts | Preview images and metadata go to the connected client |
| Error diffusion | Floyd–Steinberg, false Floyd–Steinberg, Jarvis–Judice–Ninke, Atkinson, Stucki, Burkes, three Sierra variants, simple 2D, Steven Pigeon, Fan, two Shiau–Fan variants, Stevenson–Arce | Implemented kernels. Not a bit-exact clone of another application |
| Curve diffusion | Riemersma with Hilbert traversal | Fixed documented error-history implementation |
| Ordered dithering | Bayer 2/4/8/16/32, clustered screens, arithmetic screens, checkerboard | Explicit screen sizes |
| Noise | Seeded uniform noise, void-and-cluster blue-noise tile, interleaved gradient noise | Finite repeating blue-noise tile, not a spatially unbounded optimized distribution |
| Artistic screening | Halftone dots, horizontal/vertical/diagonal lines, crosshatch, diamond, spiral, stipple, waves | Original procedural screens, not calibrated physical print simulation |
| Palettes | 256 named palettes in 16 categories, 2–256 custom colors, deterministic extraction | Opaque sRGB hex colors. Original, referenced and approximate origins are explicit |
| Palette discovery | Text/category/color-count filters, descriptions, tags, provenance, stable pagination | Default 32 results. At most 256 per page. Original 22 IDs and ordered colors remain stable |
| Input normalization | Eight EXIF orientations. Supported ICC v2/v4 RGB matrix/TRC profiles and PNG color metadata convert to sRGB | JPEG, PNG, WebP, and TIFF metadata. BMP V5 embedded profiles and V4/V5 sRGB declarations. Unsupported or malformed selected profiles cause errors. No output device profiles or HDR processing |
| Tonal/color controls | Brightness, contrast, gamma, saturation, grayscale, inversion, threshold, strength. sRGB or linear-RGB quantization | No perceptual Lab/OKLab distance |
| Sampling | Crop after orientation, width/height resize, nearest or alpha-correct bilinear resampling, coarse pixel grid, and exact-size enlargement | Crop uses upright source coordinates. Imported DPI does not automatically change pixel dimensions |
| Regional effects | Rectangle, circle, inverted selection, local image mask | No learned subject segmentation, click tracking, or camera UI |
| Retro effects | Scanlines, CRT modulation, seeded noise/glitch bands, thresholded pixel sorting | A focused set, not Ditherer's 300+ filter library |
| Inspection and iteration | Upright dimensions, normalization details, alpha/hash metadata, source preview, in-memory dithered studio preview, and algorithm comparison contact sheet | Studio previews fit within 1,024 pixels per axis and 2 MiB PNG. The studio provides image settings and zoom, without brush editing |
| Recipes and batch | Versioned JSON look recipes, per-item batch results, independent atomic destinations | No multi-output transaction or automatic rollback |
| Still input | PNG, JPEG, GIF, WebP, BMP, TIFF | Animated GIF uses first frame in still render. No SVG/HEIC/RAW input |
| Still output | PNG, JPEG, GIF, SVG, PBM, ASCII | WebP/TIFF/BMP encode not included. JPEG is lossy |
| SVG | Safe generated pixel-run paths preserving alpha | Raster-derived geometry, not traced editable Bézier artwork |
| ASCII | Luminance character-ramp text | No selectable glyph overlay, HTML character art, or copied React components |
| GIF processing | Full source animation with frame composition/disposal, delays, and loop handling | GIF supports binary alpha and at most 256 colors per frame |
| Generated loops | Wave, orbit, pulse, noise, palette cycling | Noise is a finite repeating frame sequence. No temporal feedback or optical flow |
| Sprite sheets | PNG, configurable columns, row-major frames | No bundled game-engine importer or separate frame-sequence export tool |
| Video | Opt-in local FFmpeg: MP4/MOV, WebM/Matroska, AVI. MP4/WebM/GIF/PNG outputs | No audio, live streaming, webcam capture, or unbounded clips |
| Print exports | Explicit PNG DPI, PBM, per-ink PNG masks + manifest in ZIP | No press profile, CMYK separation, PDF imposition, or device control |
| Showcase | Original generated samples, real CLI/MCP demonstrations, static website | Showcase is a presentation of outputs, not an image editor |
| Formal contracts | OpenSpec scenarios. Executable Quint models. Go refinement tests | Bounded simulation is not exhaustive implementation verification |

## Deliberate exclusions

This release does not provide these reference capabilities:

- Camera capture or live microphone/tab-audio modulation.
- Screensavers or VJ performance.
- GPU/WebGL/WASM acceleration or raymarching/3D rendering.
- Learned background removal or subject motion tracking.
- Brush editing, browser keyboard shortcuts, or dynamic pointer repulsion.
- Cloud storage or export of JavaScript/React artwork components.

These capabilities would require separate specifications. Several would also require a different security or runtime model.

Image masks let agents specify regions locally. They do not automatically discover a subject. FFmpeg gives agents finite local clip workflows without a real-time performance claim. All original reference projects retain their own identities and licenses. [Sources and inspiration](sources.md) records the basis for this scope.
