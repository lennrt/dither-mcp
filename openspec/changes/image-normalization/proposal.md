# Normalize source orientation and color

## Why

Camera files can store rotated pixels with an EXIF orientation tag. Color profiles can also give identical RGB values different appearances. Agents need consistent source pixels before they crop, extract palettes, compare algorithms, or render images.

## What changes

- Apply all eight EXIF orientations before crop and resize.
- Convert supported embedded RGB matrix/TRC ICC profiles to sRGB in pure Go.
- Interpret supported PNG color metadata with explicit precedence and errors.
- Use one bounded, context-aware still loader across tools, including image masks.
- Report stored dimensions, applied orientation, color source, and conversion status.
- Reject malformed or unsupported color transforms before processing or publication.
- Add a finite Quint model of normalization admission, geometry, alpha, and processing order.

## Capabilities

### New capabilities

None.

### Modified capabilities

- `image-pipeline`: source normalization precedes the existing geometry and quantization pipeline.

## Impact

The tool count and request options remain unchanged. Tagged sources can produce different pixels because the loader now honors their metadata. Untagged RGB sources retain the documented sRGB assumption. Unsupported selected profiles produce explicit errors instead of silent color reinterpretation.

This increment covers still-image metadata. It does not add video HDR color management, soft proofing, rendering-intent controls, or arbitrary external profiles. Existing GIF frame and output-format limits remain in force.
