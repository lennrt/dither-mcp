# Reproduce the showcase

Local programs generate the artwork, dithering studies, recipes, and terminal recordings. The Go generators and saved inputs reproduce the gallery from this checkout.

## Original artwork

`scripts/showcase-source` is a deterministic Go ray tracer using only the standard library. It renders two scenes:

- **Moon Garden:** a waxy-leaf plant, fluted ceramic pot, textured stones, and a suspended golden sphere. Ellipsoids, explicit leaf veins, procedural surface grain, and multiple shadow rays create the detail.
- **Mineral Nocturne:** layered stones, a small cairn, mineral bands, a textured floor, and a golden sphere.

The repository’s MIT license covers the original images under `docs/assets/source/` and the generator code. Ray-traced geometry, surface functions, and light form this synthetic artwork.

```sh
go run ./scripts/showcase-source
```

The default source dimensions are 960 × 640. Use `--width` to render another resolution. The program fixes the image count, dimensions, camera, geometry, surface functions, and sampling pattern.

## Gallery and recipe files

```sh
./scripts/showcase.sh
```

The script regenerates the sources and processes them with the `engine` package. It also regenerates the normalization comparison described below. It builds the CLI and generates a 24-frame, 12-fps wave GIF through the application. It saves gallery settings as version-1 recipes under `docs/assets/recipes/`.

| Asset | Algorithm | Palette | Processing grid |
|---|---|---|---|
| `garden-clay.png` | Atkinson | `#141413`, `#3d3d3a`, `#d97757`, `#e3dacc`, `#f0eee6`, `#faf9f5` | 960 × 640 output, 2× pixels |
| `garden-gameboy.png` | Atkinson | Game Boy | 960 × 640 output, 2× pixels |
| `garden-cga.png` | Bayer 4×4 | CGA | 960 × 640 output, 2× pixels |
| `mineral-ember.png` | Floyd–Steinberg | `#251d32`, `#ab4e3b`, `#e6ad62`, `#f0e7d3` | 960 × 640 output, 2× pixels |
| `garden-paper.png` | Atkinson | `#242a25`, `#eee9de` | 960 × 640 output, 2× pixels |
| `study-*.png` | Atkinson, Floyd–Steinberg, Bayer 8×8, halftone 8, blue noise, Riemersma | `#242a25`, `#eee9de` | 720 × 540 detail crop, 2× pixels |

The still recipes specify small brightness and contrast adjustments and seed 42. The animation uses the CLI settings in `scripts/showcase.sh` with neutral image adjustments. The script retains its settings alongside the still recipes so you can reproduce both treatments.

Garden recipes use `docs/assets/source/moon-garden.png`. The mineral recipe uses `mineral-nocturne.png`. The texture-study recipes use `moon-garden-detail.png`, a retained 600 × 450 crop of Moon Garden. Recipe files contain portable processing settings. To reproduce a study, use its corresponding source.

`hero.png` compares the Moon Garden source and its six-color clay result. The generator composes `algorithm-study.png` and `palette-study.png` from the processed images. The contact sheets follow the order in the README captions. The assets directory retains the original files alongside their exports.

### Palette atlas and treatments

`scripts/palette-atlas` reads `engine.Palettes()` directly. It generates the complete atlas, the explorer’s catalog, twelve rendered treatments, a contact sheet, and their recipes:

```sh
go run ./scripts/palette-atlas
```

`palette-atlas.png` groups all 256 entries by their sixteen categories. Each entry shows its stable ID and exact color swatches. `palettes.json` contains the engine’s IDs, hex values, descriptions, tags, origins, and source references. It also connects the twelve featured palettes to their image treatments.

The site and MCP tool use the same search rules. The search ignores letter case and requires every whitespace-separated query word to match part of the palette text. Category and color-count filters further limit the results.

`palette-treatments.png` assembles twelve real Floyd–Steinberg renders of Moon Garden. Their palette IDs are `copperplate`, `midnight-orchid`, `alpine-morning`, `tidal-glass`, `malachite`, `autumn-orchard`, `ultraviolet-city`, `pistachio-rose`, `black-sesame`, `nebula-rose`, `brutalist-sun`, and `oat-and-ink`. Each uses a 960 × 640 output, 2× pixels, brightness +0.06, contrast 1.1, and seed 42. The assets directory retains each version-1 recipe and full-resolution image.

The atlas and contact sheet use Source Serif 4 titles and Go font labels. The artwork directory includes the [Source Serif 4 license](assets/fonts/SOURCE-SERIF-LICENSE.md) and [Go font license](assets/GO-FONTS-LICENSE.txt).

`scripts/brand-banner` creates `banner.svg` and `banner.png` with an original ordered-dither crescent. The SVG contains outlined letters, so browsers can show it without a font installation. The warm paper background, ink text, and clay accents continue through the gallery and terminal recordings.

The engine studies are deterministic for the same source bytes and engine implementation. Video encoding may vary with FFmpeg versions and codecs. The formal models do not establish bit-identical video output across installations.

### Normalization comparison

```sh
go run ./scripts/normalization-showcase
```

The generator creates an original arrow chart with EXIF orientation 6 and a synthetic linear-RGB ICC profile. It calls `dither_inspect` and `dither_preview` through the service used by MCP and the CLI. The comparison labels stored samples and actual normalized previews separately.

The generator checks upright dimensions, selected-profile metadata, conversion of neutral 128 to sRGB 188, and preservation of alpha 160. It saves source images, the original profile, preview PNGs, and inspection receipts. See [the format guide](formats.md#reproduce-the-comparison) for the image and results.

## Terminal recordings

Prerequisites:

- Go 1.27
- [VHS](https://github.com/charmbracelet/vhs) v0.12.1
- `ttyd` and `ffmpeg` on `PATH`
- A local Chromium browser that VHS can use

Install VHS to a directory of your choosing:

```sh
go install github.com/charmbracelet/vhs@v0.12.1
```

Record all four demos:

```sh
./scripts/showcase-record.sh
```

To record only palette discovery, run `./scripts/showcase-record.sh palettes`. The optional arguments select individual tapes. By default, the script records all four.

The `.tape` files under `demos/` define the recordings. The preparation script copies source images into a temporary workspace. Each recording starts with these copies and fresh output paths.

The MCP tape runs the scripted Go client against the stdio server. The client discovers tools, calls them, and checks their results.

The render, comparison, and palette tapes show CLI JSON through `dither-summary`. This Go helper comes from `scripts/demo-summary`. It selects result fields for the terminal view. Its `--inspect` mode shows stored and upright dimensions, EXIF orientation, the selected color source, and conversion status. The helper checks required inspection, artifact, or palette fields before displaying them. The recording shells enable `pipefail` so a processing failure also fails the output pipeline.

The script creates GIF and H.264 MP4 recordings at 1200 × 750. It extracts MP4 poster frames locally with FFmpeg. Recording checks cover readable command output, successful calls, nonempty media, and expected video dimensions. The demo clips contain synthetic art and workspace-relative image paths.

## Standalone static site

Refresh the separate site assets after making the images and recordings:

```sh
./scripts/showcase-site.sh ../dither-mcp-site
python3 -m http.server 8080 --directory ../dither-mcp-site
```

The [showcase repository](https://github.com/k-a2a/dither-mcp-site) includes relative assets, scripts, local fonts, and a `.nojekyll` marker for static hosting. Serve the directory over HTTP so the browser can load its local palette JSON. The site supports a keyboard-accessible range slider and native buttons to select algorithms.

The site bundles Source Serif 4, Inter, and JetBrains Mono in `assets/fonts/`. That directory contains the complete SIL Open Font License for each family. Font requests stay within the site’s directory.

The palette explorer shows twelve entries per page. It provides category and color-count filters, exact swatches, and copy controls. If the browser cannot access the clipboard, the site shows a selectable text field. The gallery uses saved image treatments from the Go engine.

The looping image has a pause/play control. It starts as a still image for visitors who prefer reduced motion. Native video controls let visitors play and pause the terminal demos.

These scripts create assets for static hosting. Preserve the site's directory structure in a GitHub Pages publishing directory.

## Recorded verification

On 2026-10-03, the generators rebuilt the gallery, clay brand artwork, normalization comparison, and recordings after the engine and prose reviews. VHS v0.12.1 produced these H.264 clips, each 1200 × 750, with matching GIF recordings:

| Clip | Duration | Checked behavior |
|---|---|---|
| `render.mp4` | 21.60 s | Image inspection with normalization fields, actual Atkinson render, recipe and artifact digest |
| `compare.mp4` | 28.80 s | Three algorithm cells, 24-frame wave GIF, artifact metadata |
| `mcp.mp4` | 21.00 s | Real stdio discovery and tool calls, image preview, saved recipe, digest, collision guard |
| `palettes.mp4` | 30.68 s | Actual 256-entry discovery, botanical color-count filtering, word search, Oat and Ink render and digest |

The wave artwork contains 24 frames with a total duration of exactly 2.00 s. All static page asset references resolve locally. Browser checks covered:

- The desktop layout and a 390 × 844 mobile viewport, with content that fits the viewport.
- Home/End range-slider input, algorithm selection, and animation pause/play.
- Keyboard pagination and search and filter results that match the CLI.
- Exact clipboard values and the selectable-text fallback.
- Empty-result behavior, Escape reset, and featured treatment selection.
- Keyboard disclosure and image loading for the normalization comparison.

All observed gallery images loaded, and the browser console remained clear. The standalone site directory contains desktop, mobile, palette explorer, and supported-format screenshots.

The Go showcase programs compile, shell scripts pass syntax checks, and VHS checks all four tapes. These checks establish the recorded local behavior and media integrity. They do not establish compatibility with every MCP host or browser.
