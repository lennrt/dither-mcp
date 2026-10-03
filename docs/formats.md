# Formats and video

Choose the operation first. The same PNG extension can mean a still image or a sprite sheet. The following tables define the supported inputs and outputs. `dither_catalog` also exposes media capabilities for agents.

## Still images

Use `dither_render` or the CLI `render` command for one image.

| Input | Supported behavior |
|---|---|
| PNG | Decode one image. APNG uses the default PNG image, which is not necessarily the first animation frame. |
| JPEG | Decode one image. |
| GIF | Render the first frame on its logical canvas. Use `dither_animate` with `effect: "source"` for all frames. |
| WebP | Decode a static image. Animated WebP is unsupported. |
| BMP | Decode one image. |
| TIFF | Decode the first image directory. Multipage TIFF processing is unsupported. |

The service detects still inputs from their content. Decoder support does not imply support for every extension of a format. SVG, HEIC, and camera RAW inputs are unsupported.

The service normalizes supported orientation and color metadata before cropping, resizing, palette extraction, previews, and dithering. Image masks use the same loader.

| Output | Extension or format | Alpha and representation |
|---|---|---|
| PNG | `png` | Preserve partial alpha. Optional PNG DPI metadata does not change pixel dimensions. |
| JPEG | `jpeg` or `jpg` | Composite against white. JPEG is lossy. |
| GIF | `gif` | Use binary alpha and at most 256 palette entries. Alpha below 128 becomes transparent. |
| SVG | `svg` | Preserve partial alpha in generated raster-derived pixel geometry. |
| PBM | `pbm` | Composite against white and convert to monochrome. |
| ASCII | `ascii` or `txt` | Composite against white and convert luminance to a character ramp. |

The service does not encode WebP, BMP, or TIFF. An explicit `format` overrides the output extension without renaming the path. Use a new destination for every output. Existing destinations always cause an error.

```sh
./bin/dither-mcp render --root /absolute/path/to/images \
  --input photo.webp --output output/photo.png \
  --algorithm atkinson --palette gameboy --width 640
```

## Orientation and input color

The service reads orientation and embedded ICC data from these containers:

| Input | Orientation metadata | Embedded ICC profile |
|---|---|---|
| JPEG | EXIF in APP1 | Segments in APP2 |
| PNG | `eXIf` | `iCCP` |
| WebP | `EXIF` | `ICCP` |
| TIFF | First image directory, IFD0 | ICC tag in IFD0 |
| BMP | No EXIF orientation | Embedded profile in V5 headers |

The loader applies all eight EXIF orientation values, including reflection, before image options. Missing orientation means identity, value 1. Crop coordinates and inspection dimensions refer to the upright image.

For example, a JPEG can store 4000 × 3000 pixels with orientation 6. Inspection reports 3000 × 4000. Cropping uses those upright dimensions.

The service accepts **ICC v2 and v4 RGB matrix/TRC profiles**. These profiles define a color matrix and per-channel tone response curves (TRCs). Accepted classes are input (`scnr`), display (`mntr`), and color-space (`spac`). Profiles must use an XYZ profile connection space (PCS). Supported tone curves use `curv` or `para`. Parametric `para` curves accept function types 0–4. This subset includes common matrix/TRC profiles for sRGB, Display P3, and Adobe RGB. The profile’s structure determines support.

The loader converts supported profiles to sRGB with a fixed media-relative colorimetric transform. Converted values outside the sRGB range clip at its boundaries. Conversion does not use perceptual gamut mapping or black point compensation. Normalization preserves 8-bit and 16-bit alpha values. The dithering pipeline uses 8-bit pixels.

Untagged RGB and grayscale inputs use sRGB. The loader rejects CMYK images, grayscale ICC profiles, Lab profiles, device-link profiles, and multidimensional LUT profiles. An RGB ICC profile cannot describe a decoded grayscale image. The loader rejects that combination and GIF ICC extensions.

BMP V4 and V5 headers can declare sRGB or the Windows default color space, which the service treats as sRGB. BMP V5 supports embedded RGB ICC profiles from the accepted subset. Linked BMP profiles, calibrated color spaces, and unknown declarations cause errors. The loader never reads an external profile file.

### PNG color precedence

The loader selects PNG color information in this order:

1. `cICP`, accepting only full-range sRGB values `[1,13,0,1]`.
2. An embedded `iCCP` profile.
3. An `sRGB` chunk.
4. `gAMA` and `cHRM` information.
5. Assumed sRGB when color metadata is absent.

For `gAMA` and `cHRM`, missing gamma uses the sRGB transfer curve. Missing chromaticities use sRGB primaries. Other `cICP` values, including Display P3 or HDR signals, cause an error. A PNG that contains both `iCCP` and `sRGB` also causes an error.

The [W3C PNG specification](https://www.w3.org/TR/png-3/) defines these chunks and their precedence. This implementation supports the subset listed above.

### Inspection, limits, and exports

`dither_inspect` reports upright `width` and `height`. Its `normalization` object records stored dimensions, EXIF orientation, the selected color source, conversion status, and the sRGB working space. The selected ICC profile has a SHA-256 identifier. The field is absent when another source, such as PNG `cICP`, takes precedence. The file’s main `sha256` hashes the original input bytes.

The service applies these metadata limits:

| Resource | Limit |
|---|---:|
| EXIF payload | 4 MiB |
| Expanded ICC profile | 4 MiB |
| ICC tags | 256 |
| Samples per tone curve | 65,536 |
| EXIF IFD0 entries | 4,096 |

Malformed recognized container metadata and malformed or unsupported selected color profiles return tool errors. The service does not silently treat a rejected profile as untagged sRGB.

Exports use sRGB pixel values and omit source EXIF and ICC metadata. PNG and JPEG exports do not embed an output ICC profile. SVG colors use sRGB. Explicit output PNG DPI remains available.

Input normalization does not add printer profiles, calibrated CMYK conversion, or video HDR processing. Video keeps its existing FFmpeg behavior.

See the [ICC profile specification](https://archive.color.org/specification/ICC.1-2022-05.pdf), [ICC curve guidance](https://archive.color.org/files/whitepapers/ICC_White_Paper35-Use_of_the_parametricCurveType.pdf), and [CIPA EXIF specifications](https://www.cipa.jp/e/std/std-sec.html) for the underlying formats.

### Reproduce the comparison

![Stored EXIF orientation 6 beside an upright service preview, and stored linear RGB samples beside their normalized sRGB preview](assets/normalization.png)

The left panels visualize stored pixel samples. The right panels contain actual `dither_preview` results. The color comparison uses an original synthetic linear-sRGB ICC profile. Its neutral sample 128 becomes sRGB 188. Alpha 160 remains unchanged.

```sh
go run ./scripts/normalization-showcase
```

The generator creates an original arrow chart, embeds metadata, and calls the application service used by MCP and the CLI. It retains the source PNGs, embedded profile, and preview PNGs under `docs/assets/`. [Inspection receipts](assets/normalization-inspection.json) record the input hashes, orientation, working space, profile digest, and conversion result.

## GIF animation and sprite sheets

Use `dither_animate` or the CLI `animate` command for these workflows:

| Source | Effect | Result |
|---|---|---|
| Supported still image | `wave`, `orbit`, `pulse`, `noise`, or `palette-cycle` | Generate a finite animation. |
| GIF | `source` | Process all source frames with their composition, disposal, delays, and loop count. |

The output is an animated GIF or a PNG sprite sheet. APNG, animated WebP, video containers, and separate frame files are not animation outputs. Use `dither_video` for MP4 or WebM output from a local video source.

Still sources default to `wave`, 24 frames, and 12 fps. GIF input defaults to `source`. The default width is 480 when both output dimensions are omitted or zero. Generated animation accepts 1–120 frames and 1–50 fps. Source GIF timing overrides the `frames` and `fps` controls.

Source GIFs must contain at most 120 frames and 67,108,864 aggregate logical-canvas pixels. The service checks all source frames against these limits before decoding them.

Sprite sheets use row-major order. They default to six columns and retain transparent unused cells. The animation operation accepts a `columns` option. The artifact dimensions describe the whole sheet.

```sh
./bin/dither-mcp animate --root /absolute/path/to/images \
  --input source.gif --output output/dithered.gif \
  --effect source --palette pico-8 --width 320
./bin/dither-mcp animate --root /absolute/path/to/images \
  --input photo.png --output output/sheet.png \
  --effect wave --frames 24 --fps 12 --columns 6 --width 320
```

## Video requirements and inputs

Video processing requires trusted `ffmpeg` and `ffprobe` executables on PATH. Enable it explicitly with `--allow-video`. Still images and GIF animation do not require these executables.

The service admits these binary container families:

| Container | Required content signature | Locally tested input codec |
|---|---|---|
| MP4 | `ftyp` header | H.264, with AAC audio in an audio-removal test |
| MOV | `ftyp` header | H.264 |
| WebM | EBML header | VP9 |
| Matroska | EBML header | FFV1 |
| AVI | RIFF header with `AVI ` type | MPEG-4 |

The service detects containers from content, including an MP4 file renamed with a `.bin` extension. MOV files without the required `ftyp` header are unsupported. Playlists, remote URLs, network streams, and live camera inputs are unsupported.

Input codecs depend on the decoders in your FFmpeg build. Container admission does not guarantee that your build can decode every codec in that container. The tested combinations above passed the local integration tests. They are not an exhaustive codec matrix.

## Video outputs and limits

| Output | Encoder and behavior |
|---|---|
| MP4 | `libx264`, H.264, CRF 18, `yuv420p`, and faststart metadata. |
| WebM | `libvpx-vp9`, VP9, CRF 24, zero target bitrate, and `yuv420p`. |
| GIF | Animated GIF with binary alpha. |
| PNG | Sprite sheet with six columns. |

The installed FFmpeg build must provide the selected video encoder. MP4 and WebM add one padding pixel to each odd dimension. Both video formats are lossy. Video processing selects only the first video stream and omits audio, subtitles, and data streams. GIF and PNG video exports also contain no audio.

| Control or resource | Default | Limit or condition |
|---|---|---|
| Frames | 48 | 1–120. A short source can produce fewer frames. |
| FPS | 12 | 1–50 |
| Width | 480 | Applies when both width and height are omitted or zero. |
| Start | 0 seconds | 0–86400 seconds |
| Clip window | 4 seconds | At most `frames / fps` seconds from the requested start. |
| Input and encoded output | — | 32 MiB each |
| Image dimensions | — | 16,384 pixels per axis and 16,777,216 pixels |
| Native decoded frames | — | 67,108,864 aggregate pixels |
| Rendered frames | — | 67,108,864 aggregate pixels |
| Operation timeout | — | 120 seconds, including queue time |

The native frame budget applies before resizing. Reducing output width alone cannot make an oversized source request fit. Reduce the requested frame count when necessary. PNG sprite sheets also obey still-image dimension and pixel limits. Video requests do not support DPI metadata.

The service checks output byte size and encoded frame count before publishing MP4 or WebM. Cancellation is cooperative. See [security](security.md) for subprocess controls and deadline limits.

## Run video from the CLI

Build the binary with `make build`. Place a supported clip beneath an existing workspace root. Run this command with a new destination:

```sh
./bin/dither-mcp video --root /absolute/path/to/images --allow-video \
  --input clip.mp4 --output output/dithered.mp4 \
  --algorithm bayer-8 --palette cga --width 320 \
  --frames 24 --fps 12 --start 0
```

Use `.webm`, `.gif`, or `.png` for the other video outputs. The PNG result is a sprite sheet.

## Enable video for MCP

For hosts that use `mcpServers`, configure a local stdio server as follows. Replace both absolute paths with your local paths.

```json
{
  "mcpServers": {
    "dither": {
      "command": "/absolute/path/to/dither-mcp",
      "args": ["mcp", "--root", "/absolute/path/to/images", "--allow-video"]
    }
  }
}
```

Ensure that the MCP host's PATH includes `ffmpeg` and `ffprobe`. Call `dither_catalog` to inspect media capabilities. Call `dither_video` with these arguments:

```json
{
  "input": "clip.mp4",
  "output": "output/dithered.webm",
  "palette": "cga",
  "options": {"algorithm": "bayer-8", "width": 320},
  "frames": 24,
  "fps": 12,
  "start": 0
}
```

File paths remain relative to the configured root. A recipe controls the still-image look. The operation request controls the source, frame count, FPS, and start time.

For visual review, first export a PNG sprite sheet. Pass its path to
`dither_preview`. The preview tool accepts still-image formats.

## Print separation

`dither_separate` exports a ZIP archive with one PNG plate per ink and a JSON manifest. It permits at most 16 inks. It rejects masks and post-effects. Black marks ink coverage, while white marks unprinted areas.

DPI metadata defaults to unspecified. An explicit PNG DPI value must be 36–2400. These binary spot-color plates do not provide calibrated CMYK separation, press profiles, or device control.

## Verification

The local video tests exercised every container and codec combination listed above. They exported GIF, PNG sprite sheets, H.264 MP4, and VP9 WebM. They checked actual codecs, frame counts, dimensions, even padding, and silent output from an input with audio.

Run `make video-test` to repeat the optional integration tests. See [testing](testing.md) for recorded evidence and [the feature matrix](feature-matrix.md) for other supported workflows.
