## ADDED Requirements

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
