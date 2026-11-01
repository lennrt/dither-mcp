# Add area resizing and interchange exports

## Why

Bilinear reduction samples only four pixels per footprint. Fine patterns can lose their average tone. BMP and TIFF inputs cannot currently export in their original format.

## What changes

- Add explicit alpha-correct `area` box sampling without changing the bilinear default.
- Average exact source coverage on shrinking and enlarged axes, including fractional boundaries.
- Add opaque white-matted BMP exports and lossless unassociated RGBA TIFF exports.
- Retain byte limits, cancellation, immutable publication, and PNG-only custom DPI.

## Capabilities

### New capabilities

None.

### Modified capabilities

- `image-pipeline`: area sampling and BMP/TIFF export semantics.

## Impact

Existing version-1 recipes can select `area` through the existing resize field. New exports use existing codec dependencies. Area buffers use two scanlines. TIFF encodes uncompressed to enforce the output byte budget while writing. No publication, cancellation, palette, or selection contract changes require a Quint update.
