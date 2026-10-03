# Image pipeline specification

## Purpose

Provide reproducible image quantization and expressive transformations through a discoverable Go engine.

## Requirements

### Requirement: Discoverable algorithm semantics

The suite SHALL expose stable identifiers, families, and descriptions for its implemented algorithms. These SHALL include error diffusion, ordered screens, threshold quantization, and artistic screens. Unknown identifiers SHALL produce errors. A catalog entry SHALL correspond to implemented behavior rather than an unsupported alias.

#### Scenario: Discover and select
- **WHEN** an agent lists the catalog and submits a listed algorithm identifier
- **THEN** the engine executes the corresponding kernel or screen

#### Scenario: Unknown algorithm
- **WHEN** an agent submits a misspelled or unsupported identifier
- **THEN** the engine rejects it and makes supported choices discoverable

### Requirement: Explicit palette handling

The suite SHALL support named palettes and custom hexadecimal colors. It SHALL support deterministic extraction of a bounded palette from a source image. Palette validation SHALL reject invalid colors and unsupported sizes. Quantization without region masking or post-effects SHALL use only the selected palette RGB values while preserving documented alpha behavior.

#### Scenario: Custom duotone
- **WHEN** an agent renders an opaque source with exactly two custom colors and no mask or post-effects
- **THEN** every output RGB triplet is one of those two colors

#### Scenario: Stable extraction
- **WHEN** the engine samples the same source with the same extraction size and settings
- **THEN** the extracted palette and its ordering are repeatable

### Requirement: Validated adjustments and geometry

The pipeline SHALL support documented crop/resize/pixel scaling, brightness, contrast, gamma, saturation, inversion, and strength controls. It SHALL reject non-finite numbers, invalid rectangles, invalid dimensions, and out-of-range options before expensive work.

#### Scenario: Palette-ready crop
- **WHEN** an agent supplies a valid crop, target width, and pixel scale
- **THEN** the result follows the documented crop, resize, adjustment, quantization, and scale order

#### Scenario: Invalid gamma
- **WHEN** gamma is zero, negative, non-finite, or outside its supported interval
- **THEN** the engine returns a validation error

### Requirement: Reproducible seeded processing

The engine SHALL produce identical pixels for equal decoded inputs, normalized configurations, seeds, and engine versions. Randomized effects SHALL use a request-local seeded generator. They SHALL not use process-global mutable randomness.

#### Scenario: Repeat seeded noise
- **WHEN** a stochastic algorithm renders the same image twice using the same seed
- **THEN** the resulting pixel buffers are identical

#### Scenario: Parallel reproducibility
- **WHEN** independent requests with the same source and settings execute concurrently
- **THEN** their outputs remain identical

### Requirement: Region masks and creative effects

The engine SHALL provide documented geometric region masks and composable creative controls sufficient for localized dithering and retro effects. The documentation SHALL identify which controls run before or after quantization. It SHALL state when final palette membership is no longer guaranteed.

#### Scenario: Dither a selected region
- **WHEN** a rectangle or circle mask selects a region
- **THEN** the engine applies the quantized result according to the documented inside/outside mask behavior

### Requirement: Explicit export semantics

The suite SHALL support PNG and the documented GIF, SVG, PBM, and ASCII export operations. It SHALL reject unsupported format combinations instead of silently renaming bytes. SVG output SHALL encode the raster result as safe generated geometry without user-controlled markup or external references. PBM SHALL follow a documented monochrome conversion.

#### Scenario: Export inspectable vector artwork
- **WHEN** an agent exports a small dither to SVG
- **THEN** the result contains generated geometry representing output pixels and no remote dependencies

#### Scenario: Unsupported format
- **WHEN** an agent requests an unsupported output format
- **THEN** the request fails before publication

### Requirement: Shared source orientation normalization

The shared still loader SHALL apply EXIF orientations 1 through 8 before crop, resize, palette extraction, and quantization. Missing orientation SHALL mean 1. Crop coordinates and working dimensions SHALL refer to normalized pixels. Image masks SHALL use the same policy. Orientation SHALL preserve pixel count and move alpha with its pixel.

#### Scenario: Crop an oriented source
- **WHEN** a source declares orientation 6 and an agent supplies a crop
- **THEN** the loader swaps the working dimensions and applies the crop to the clockwise-rotated image

#### Scenario: Preserve mirrored alpha
- **WHEN** a source declares a mirrored orientation and contains partial transparency
- **THEN** normalization moves each color and its alpha together without reducing alpha precision

#### Scenario: Reuse normalized source semantics
- **WHEN** inspection, preview, palette extraction, rendering, or a mask load reads the same source
- **THEN** each operation uses the same orientation and color-normalization policy

### Requirement: Bounded source color conversion

The loader SHALL convert selected supported RGB matrix/TRC ICC v2/v4 input/display/color-space profiles with XYZ PCS to sRGB. It SHALL support `curv` and `para` types 0–4. Conversion SHALL preserve source alpha precision. Untagged RGB/grayscale SHALL assume sRGB. Unsupported selected profiles and untagged CMYK SHALL fail explicitly. Source metadata SHALL NOT survive export as stale EXIF or ICC data.

#### Scenario: Convert a supported RGB profile
- **WHEN** source metadata selects a valid supported RGB matrix/TRC profile
- **THEN** the loader converts its channels through D50 PCS to sRGB before image processing and preserves alpha

#### Scenario: Reject a LUT profile
- **WHEN** a selected source profile contains an unsupported LUT transform, including one with incidental matrix tags
- **THEN** the loader rejects it instead of silently selecting the matrix tags

#### Scenario: Reject a linked BMP profile
- **WHEN** a BMP color declaration references an external profile
- **THEN** the loader rejects the declaration without opening the referenced path

#### Scenario: Export normalized values
- **WHEN** a normalized source renders successfully
- **THEN** the encoder receives sRGB values and does not copy the source orientation or ICC profile

### Requirement: Explicit PNG color metadata precedence

The PNG loader SHALL select supported cICP, ICC, sRGB, gamma/chromaticities, then assumed sRGB, in that order. It SHALL accept only cICP `[1, 13, 0, 1]`. It SHALL reject ICC combined with sRGB. It SHALL validate recognized container metadata and payload bounds. A supported cICP declaration SHALL supersede lower-priority ICC transforms. Selected malformed or unsupported profiles SHALL fail. Conversion SHALL preserve alpha.

#### Scenario: Honor PNG gamma and chromaticities
- **WHEN** valid gAMA or cHRM metadata determines a PNG source's color interpretation
- **THEN** the loader applies the supported conversion to sRGB before processing pixels

#### Scenario: Reject conflicting PNG declarations
- **WHEN** a PNG contains both an ICC profile and an sRGB chunk
- **THEN** the loader reports the conflict and returns no normalized image

#### Scenario: Select cICP before ICC
- **WHEN** a PNG contains supported cICP and a bounded ICC payload
- **THEN** the loader selects sRGB from cICP without parsing or applying the lower-priority ICC transform

#### Scenario: Reject an unsupported HDR declaration
- **WHEN** a PNG cICP tuple differs from `[1, 13, 0, 1]`
- **THEN** the loader returns an unsupported-color error instead of assuming sRGB

### Requirement: Metadata admission before processing

The loader SHALL limit expanded ICC profiles and EXIF payloads to 4 MiB each. It SHALL limit profiles to 256 tags, sampled curves to 65,536 entries, and EXIF first directories to 4,096 entries. It SHALL validate container lengths, offsets, orientation, and selected transforms before processing. Metadata loops and normalization pixel loops SHALL observe context cancellation. Malformed, unsupported, oversized, or canceled normalization SHALL return an error and SHALL NOT publish an output.

#### Scenario: Bound a compressed profile
- **WHEN** a compressed PNG profile expands beyond 4 MiB
- **THEN** normalization fails before allocating an unbounded profile or processing source pixels

#### Scenario: Reject malformed orientation
- **WHEN** recognized orientation metadata has an invalid type, value, or conflicting declaration
- **THEN** the loader returns a metadata error and the operation publishes nothing

#### Scenario: Observe cancellation during normalization
- **WHEN** a normalization loop observes context cancellation
- **THEN** it stops and does not return pixels for downstream processing

### Requirement: Discoverable normalization evidence

Inspection SHALL report stored dimensions, source orientation, orientation application, color source, sRGB working space, and conversion status. It SHALL include an ICC digest when an ICC profile determines color interpretation. Source hashes SHALL continue to describe original bytes. The catalog SHALL identify supported metadata, profile shapes, bounds, and exclusions. MCP and CLI SHALL expose the same normalization policy.

#### Scenario: Inspect normalized dimensions
- **WHEN** an agent inspects a source whose orientation swaps its axes
- **THEN** working dimensions describe the normalized image and normalization metadata retains the stored dimensions

#### Scenario: Discover the supported color subset
- **WHEN** an agent reads the catalog before processing a tagged source
- **THEN** it can identify supported orientation and profile behavior without assuming LUT, CMYK, or HDR support
