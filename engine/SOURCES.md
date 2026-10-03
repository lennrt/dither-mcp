# Image engine contracts and references

The engine uses original Go code. Named algorithms follow the published kernels
and constructions listed below. Procedural screens use original designs.
These references explain the mathematics and conventions. They are not runtime
dependencies. The project bundles no images or source code from the reference
projects.

## Processing contract

`Process(context.Context, image.Image, Config)` returns a new image with
zero-origin bounds. It checks the recipe and all derived dimensions before
allocating image-sized memory. Inputs and outputs can contain at most
16,777,216 pixels. Each dimension can be at most 16,384 pixels. The service
applies additional file and animation limits.

Processing checks cancellation at least once per row. Hilbert traversal checks
every 256 visited pixels. Blue-noise construction checks its bounded iterations.
Cancellation returns an error without a partial result.

The pipeline follows these steps:

1. Crop the source.
2. Resize the image and reduce its pixel grid.
3. Adjust tone and color.
4. Dither pixels within the mask.
5. Apply post-effects.
6. Enlarge the result with nearest-neighbor sampling.

Width or height alone preserves the aspect ratio. Both values specify exact
dimensions. Pixel scale uses a `ceil(width/scale)` × `ceil(height/scale)` working
grid. The engine returns exactly the requested output dimensions. Mask
coordinates refer to the reduced grid. A crop must fit wholly within the input,
relative to `Bounds().Min`.

Default resizing uses bilinear interpolation that preserves alpha correctly.
The `nearest` filter is also available. Bilinear downsampling balances speed and
quality. It does not use area or Lanczos resampling.

Neutral brightness is 0. Neutral contrast, saturation, gamma, and strength are
1. Neutral threshold is 0.5. Optional float pointers distinguish omission from an
explicit zero.

Gamma applies `channel^(1/gamma)`. Brightness adds an sRGB channel offset.
Contrast pivots around 0.5. Saturation interpolates against Rec. 709 luminance.
Adjusted channels clamp to [0,1]. Threshold adds `0.5-threshold` to the
quantization vector, so higher thresholds darken the result.

Palette colors are opaque sRGB values. Quantization uses Euclidean RGB distance
in `srgb` by default. The `linear-rgb` option uses the standard sRGB transfer
function. Equal-distance ties select the earlier palette entry.

Error diffusion carries floating-point RGB error in the selected color space.
It discards error at image edges. The corrected quantization vector clamps to
[0,1] before nearest-color selection and error calculation. This bounds feedback
even when strength exceeds 1 or source colors lie outside the palette gamut.

Ordered algorithms select the two nearest palette entries. They project the
source onto the segment between those entries in the selected color space.
The threshold pattern selects an entry. This method supports arbitrary palettes
and keeps every output color within the palette. It approximates color mixtures
with two colors. It does not use Yliluoma's exhaustive palette-mixing method.

Unresized alpha values remain exact. Fully transparent pixels become transparent
black. Partially transparent pixels use their unassociated RGB. Resizing
interpolates premultiplied color and alpha. This prevents fringes from hidden RGB
in transparent pixels.

Masks retain the adjusted source outside the selection. Diffusion checks the
complete line to each destination. Wider kernels cannot cross a transparent or
unselected gap. Riemersma clears its history at unselected or transparent pixels.
Image masks select luminance × alpha ≥ 0.5.

Scanlines, CRT shading, and seeded additive noise run after dithering. They may
produce colors outside the palette. Glitch performs cyclic horizontal shifts of
seeded row bands. Pixel sorting orders visible contiguous luminance runs.
Moving effects preserve each row's multiset of alpha values.

Without post-effects or unselected source regions, every visible output RGB is a
palette entry.

## Named algorithms

| Family | Implementations | Reference and convention |
| --- | --- | --- |
| Classic diffusion | Floyd–Steinberg, false Floyd–Steinberg, Jarvis–Judice–Ninke, Atkinson, Stucki, Burkes, Sierra, two-row Sierra, Sierra Lite, simple 2D | The implementation follows [Go dither's explicit kernels](https://github.com/makeworld-the-better-one/dither/blob/master/error_diffusers.go) and [hitherdither's kernels](https://github.com/hbldh/hitherdither/blob/master/hitherdither/diffusion.py). Both implementations provide checks for the mathematical coefficients. Atkinson propagates 6/8 of the error, so its weights do not sum to 1. |
| Sparse diffusion | Steven Pigeon | The implementation follows [the author's proposal](https://hbfs.wordpress.com/2013/12/31/dithering/). It uses denominator 14, as recorded in the Go dither catalog. This sparse kernel propagates 12/14 of the error. |
| Extended diffusion | Fan, Shiau–Fan, Shiau–Fan 2 | The implementation follows [Shiau and Fan's original distribution sets](https://patents.google.com/patent/US5353127A/en). The extended variants use power-of-two shares and diffuse only to future pixels in scan order. Fan uses the four-neighbor 7:1:3:5 variant. |
| Hexagonal diffusion | Stevenson–Arce | The implementation follows [Ulichney's error diffusion discussion](https://cv.ulichney.com/papers/1988-blue-noise.pdf) and [hitherdither's explicit 7×4 stencil](https://github.com/hbldh/hitherdither/blob/master/hitherdither/diffusion.py). The denominator is 200. |
| Curve diffusion | Riemersma | The implementation follows [Thiadmer Riemersma's description](https://www.compuphase.com/riemer.htm). Hilbert traversal uses 16 historical errors. Their weights decay exponentially with an oldest:newest ratio of 1:16. History stores source-minus-chosen error. It does not recursively feed accumulated correction into the queue. Clipped rectangles prune empty Hilbert subtrees. |
| Ordered | Bayer 2, 4, 8, 16, 32 | The implementation uses recursive dispersed-dot Bayer construction. Tests check the 4×4 rank matrix against [ImageMagick's threshold catalog](https://github.com/ImageMagick/ImageMagick/blob/main/config/thresholds.xml). Ranks use midpoint thresholds `(rank+0.5)/(size²)`. |
| Clustered | Clustered 4, clustered 8 | The implementation follows [ImageMagick's angled 4×4 and 8×8 clustered screens](https://github.com/ImageMagick/ImageMagick/blob/main/config/thresholds.xml). These screens use repeated ranks within each tile. Their patterns differ from radial dot screens. |
| Noise | Blue noise | The implementation follows [Robert Ulichney's void-and-cluster construction](https://cv.ulichney.com/papers/1993-void-cluster.pdf). The original bounded 16×16 implementation uses toroidal Gaussian density (σ=1.5), prototype relaxation, and three ranking phases. The small periodic tile becomes visible at large pixel scales. It cannot provide infinitely nonrepeating stochastic texture. |
| Noise | Interleaved gradient | The implementation uses the fractional-coordinate threshold formula from [Jorge Jimenez's presentation](https://www.iryoku.com/next-generation-post-processing-in-call-of-duty-advanced-warfare/). |

All diffusion kernels support optional serpentine scanning. Odd rows reverse
both traversal and horizontal kernel offsets. Coordinate noise uses a specified
SplitMix64 integer hash. Identical images and recipes produce identical pixels
independently of Go's pseudorandom library version.

## Original screens and baseline

These algorithms use original screen styles. They do not claim to reproduce a
proprietary filter or a specific historical algorithm.

- `threshold` selects the nearest palette color as a baseline.
- `random` uses independent, uniformly distributed seeded coordinate thresholds.
- `arithmetic-add` and `arithmetic-xor` use explicit integer coordinate arithmetic.
- `halftone-4`, `halftone-8`, and `halftone-16` use dot cells ranked by radial distance.
- `horizontal-lines`, `vertical-lines`, and `diagonal-lines` use eight-pixel
  periodic thresholds that grow parallel lines.
- `crosshatch` and `diamond` use ranked crossed lines and Manhattan-distance cells.
- `spiral`, `dots`, and `waves` use ranked polar, multipoint, and sinusoidal screens.
- `checkerboard` alternates thresholds of 1/3 and 2/3.

A ranked screen changes local coverage according to each source pixel. Dot cells
may distort over rapidly varying detail. These image screens do not render
vector halftones, calibrate printer halftones, or simulate physical inks.
The catalog contains 41 distinct choices, including different Bayer and dot-cell
sizes. It counts no aliases as additional implementations.

## Export and palette extraction

PNG and SVG preserve alpha. SVG merges identical horizontal pixel runs into
paths. It preserves exact raster coverage with `shape-rendering="crispEdges"`.
JPEG (quality 95), PBM, and ASCII composite against white. PBM emits binary P4
bilevel data. It reduces colored output by luminance at 0.5.

ASCII uses one symbol per pixel. Its dark-to-light ramp is `@%#*+=-:. `.
It does not correct the terminal character aspect ratio.

GIF retains exact visible RGB colors when they fit into 256 palette slots.
It reserves one transparent slot where required. Larger color sets use
deterministic median cut and nearest-color mapping. This mapping adds no second
dither pattern.

Palette training uses the same binary-alpha rule as output mapping. Invisible
RGB therefore cannot bias visible colors. GIF makes alpha below 128 transparent.
It makes alpha at or above 128 opaque. Palette order and mapping remain
deterministic. `Palettize` exposes the same mapping for animations.

Palette extraction uses alpha-weighted median cut over at most 262,144 samples
at a fixed stride. It excludes fully transparent pixels. Sorting before
partitioning makes map iteration irrelevant. A single-color image produces one
extracted color even when the request specifies more. A fully transparent image
produces an error. All preset colors use portable opaque sRGB values, regardless
of their historical origin.

## Built-in color library

The embedded `palettes.json` uses schema version 1. It contains 256 reviewed
palettes across 16 categories:

- Architecture
- Botanical
- Cosmic
- Duotone
- Food
- Interface
- Landscape
- Mineral
- Neon
- Neutral
- Ocean
- Pastel
- Print
- Retro
- Seasonal
- Terminal

Each category has sixteen choices. Color lists contain 2–256 distinct normalized
`#rrggbb` values. Unique unordered color sets prevent color-order permutations
from increasing the count. Palette order affects nearest-color tie breaking.
An independent migration snapshot preserves the original twenty-two IDs and
ordered color lists.

The library includes 239 original designs for dither-mcp. These cover natural
subjects, printing treatments, compact duotones, and broad illustration sets.
The project's MIT license covers these designs. Every palette has a display
name, description, category, tags, and origin. `Palettes()` returns independent
copies of both color and tag slices. Catalog access and image processing require
no network connection.

Seven entries follow published reference color tables:

- PICO-8's sixteen base colors follow the [official manual](https://www.lexaloffle.com/dl/docs/pico-8_manual.html).
- [Solarized](https://ethanschoonover.com/solarized/),
  [Nord](https://www.nordtheme.com/docs/colors-and-palettes/), and
  [Dracula](https://draculatheme.com/contribute) identify their theme authors and
  published colors.
- `plan9-256` and `web-safe-216` follow Go's
  [`image/color/palette` API](https://pkg.go.dev/image/color/palette).
  Serialization preserved exact table order without copying implementation
  source. Tests compare every entry against the standard-library tables.
  The [Plan 9 color-map specification](https://9p.io/magic/man2html/6/color)
  describes RGBV's construction and sixteen gray levels.
- Teletext's eight ideal binary RGB combinations follow the character-generator
  color bits in [MAME's SAA5050 implementation](https://github.com/mamedev/mame/blob/master/src/devices/video/saa5050.cpp).

Ten entries label their historical display colors as `approximation`. These
cover Game Boy, Game Boy Pocket, two CGA modes, EGA's sixteen RGBI colors,
Commodore 64, Apple II, ZX Spectrum, MSX, and NES. Source URLs identify hardware
definitions or emulator context. They do not claim universally calibrated sRGB
measurements. Analog and reflective display appearance depends on the screen
and decoder.

The [Game Boy register definitions](https://github.com/gbdev/pandocs/blob/master/src/Palettes.md)
identify four monochrome levels.
The [IBM documentation archive](https://www.pcjs.org/machines/pcx86/ibm/video/)
and [Commodore's original user guide](https://commodore.ca/manuals/c64_users_guide/c64-users_guide.htm)
identify their color families.
Apple II follows the color family in [Apple's original reference manual](https://apple2history.org/dl/a2refmanorig.pdf).
Spectrum normal and bright levels use a conventional D7/FF mapping of the
[original manual's color definitions](https://worldofspectrum.org/ZXBasicManual/zxmanchap16.html).
The MSX entry matches [MAME's TMS9928A RGB approximation](https://github.com/mamedev/mame/blob/master/src/devices/video/tms9928a.cpp).

NES uses a conventional NTSC emulator interpretation of its base colors.
[FCEUX's palette documentation](https://fceux.com/web/help/Palette.html)
explains how palette and NTSC-decoder choices affect its appearance. Repeated
hardware black and gray indices appear once. Apple II, Spectrum, and MSX each
have fifteen unique colors. NES has fifty-five. These opaque color lists do not
emulate hardware indices, tile attributes, or transparency.

## Verification

`go test ./engine` checks these contracts:

- Floyd–Steinberg and Atkinson match fixed reference pixels.
- Bayer ranks and diffusion kernels follow their defined constructions and weights.
- All 41 algorithms retain determinism, palette membership, and alpha.
- Hilbert traversal covers thin rectangles.
- Masks and transparency isolate diffusion.
- Resizing preserves alpha correctly.
- Cancellation, recipe validation, extraction, and writer errors follow their contracts.
- GIF output retains exact colors when they fit.

Every built-in palette renders with ordered and diffusion algorithms in both
sRGB and linear RGB. Tests check determinism, alpha, and exact palette membership.
Catalog tests check metadata, provenance, unique color sets, category and size
breadth, defensive copies, and legacy recipe compatibility.
`go test -race ./engine` checks shared-state safety. The algorithm distinctness
test compares all 41 choices on a coverage ramp.

`FuzzProcess` checks bounded processing contracts across algorithms and valid
recipes. `FuzzParseHex` checks accepted hexadecimal round trips.

The engine has no nonstandard dependencies. Run these commands to reproduce the
benchmarks:

```sh
go test ./engine -run '^$' -bench 'Benchmark(FloydSteinberg|Bayer8)$' -benchtime=1x -benchmem
```

Local measurements used an Apple M3 Pro with Go 1.27 on darwin/arm64. Each
algorithm processed a 1024×768 NRGBA fixture into PICO-8 colors.
Floyd–Steinberg took 124 ms with 3.20 MB and 18 allocations.
Bayer 8 took 90 ms with 3.15 MB and 9 allocations.
These single-iteration measurements provide orientation, not throughput
guarantees.

Common Go image types use direct pixel readers to avoid per-pixel interface
allocations. Diffusion storage uses rolling rows instead of a full floating-point
image.
