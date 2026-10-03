# Source normalization checklist

## 1. Contracts

- [x] 1.1 Write and strictly validate the image-normalization OpenSpec change and baseline additions.
- [x] 1.2 Typecheck and execute the finite Quint normalization model and negative regression check.
- [x] 1.3 Record model boundaries, implementation-test mapping, and verification evidence.

## 2. Implementation

- [x] 2.1 Add bounded metadata extraction for supported EXIF and ICC containers.
- [x] 2.2 Normalize all eight EXIF orientations before geometry and preserve alpha precision.
- [x] 2.3 Convert supported matrix/TRC ICC profiles to sRGB and reject unsupported or malformed profiles.
- [x] 2.4 Implement PNG color-metadata precedence and gamma/chromaticity conversion.
- [x] 2.5 Integrate the context-aware loader across still operations and image masks.
- [x] 2.6 Expose normalization evidence and supported color behavior through inspection and the catalog.

## 3. Delivery and verification

- [x] 3.1 Add orientation, color-reference, alpha, metadata-boundary, and shared-loader regression tests.
- [x] 3.2 Verify malformed metadata cannot publish output and existing untagged sources remain stable.
- [x] 3.3 Update formats, pipeline, security, MCP, and CLI documentation with the supported subset.
- [x] 3.4 Run specification and implementation checks, then refresh delivered verification evidence.
