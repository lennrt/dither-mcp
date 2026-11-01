## ADDED Requirements

### Requirement: Explicit area resizing

The engine SHALL accept `area` as a resize filter while retaining bilinear as the default. Area sampling SHALL average exact source-pixel coverage using premultiplied alpha in display-space RGB. Coverage-weighted alpha and RGB numerators SHALL remain exact integers until final conversion to 8-bit values, with half-integer ties rounded upward. Crop coordinates SHALL remain relative to source bounds. Each enlarged or reduced axis SHALL use box footprints. Fully transparent output pixels SHALL become transparent black. Sampling SHALL observe cancellation and use bounded scanline storage. Integer accumulation SHALL remain within uint64 for every admitted image geometry.

#### Scenario: Reduce fine stripes
- **WHEN** six opaque white pixels and two opaque black pixels reduce to one pixel with area sampling
- **THEN** the sample is opaque RGB `(191,191,191)` before adjustments and quantization

#### Scenario: Fractional transparent coverage
- **WHEN** an output footprint includes opaque red and partially transparent green source pixels
- **THEN** its RGB average weights source alpha and coverage, while its alpha averages coverage alone

#### Scenario: Enlarge one axis and reduce another
- **WHEN** area sampling enlarges width and reduces height
- **THEN** each axis follows exact box coverage, including blended fractional boundary pixels

#### Scenario: Preserve the GIF alpha threshold
- **WHEN** exact source coverage produces alpha 127.5 before 8-bit conversion
- **THEN** area sampling produces alpha 128, including when the separable axis order is transposed
- **AND** GIF export treats that output pixel as opaque

#### Scenario: Cancel area work
- **WHEN** cancellation occurs during source integration
- **THEN** the engine returns cancellation without a partial result

### Requirement: BMP interchange export

The suite SHALL export BMP as opaque uncompressed 24-bit RGB composited against white. It SHALL admit the `bmp` format and extension. Encoding SHALL obey the existing output byte budget, cancellation, and no-clobber publication contracts. Custom DPI SHALL remain unsupported for BMP.

#### Scenario: Export partial transparency to BMP
- **WHEN** a source with partial alpha exports as BMP
- **THEN** decoded pixels match white compositing and have opaque alpha

#### Scenario: BMP exceeds output budget
- **WHEN** BMP encoding exceeds the output byte budget
- **THEN** rendering fails without publishing an artifact

### Requirement: TIFF interchange export

The suite SHALL export TIFF as uncompressed unassociated 8-bit RGBA without color quantization. The `tiff` and `tif` format names and extensions SHALL select canonical `tiff`. Encoding SHALL obey the existing output byte budget, cancellation, and no-clobber publication contracts. TIFF SHALL retain no source EXIF or ICC metadata. Custom DPI SHALL remain unsupported for TIFF. Codec default resolution tags SHALL NOT imply calibrated physical dimensions.

#### Scenario: Export partially transparent TIFF
- **WHEN** source pixels with partial alpha export as TIFF
- **THEN** decoding preserves each 8-bit RGB channel and alpha value exactly

#### Scenario: TIFF alias and explicit format
- **WHEN** an agent selects `tif` explicitly or through the output extension
- **THEN** the artifact contains TIFF bytes and reports canonical format `tiff`

#### Scenario: TIFF exceeds output budget
- **WHEN** uncompressed TIFF encoding exceeds the output byte budget
- **THEN** rendering fails without publishing an artifact
