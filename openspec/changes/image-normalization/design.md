# Source normalization design

## Pipeline boundary

The shared still loader owns metadata extraction and normalization. It checks encoded bytes and stored dimensions before full decoding. It validates relevant metadata before returning pixels to a caller.

The loader applies orientation and color conversion before crop, resize, palette extraction, masks, or quantization. These two normalization operations commute pointwise, so their relative order is an implementation choice. Each runs at most once per load. The image engine continues to accept already decoded images without filesystem access.

Rendering, preview, inspection, comparison, batch items, still-source animation, palette extraction, and separations use this loader. Image masks use the same orientation and color policy. GIF source animation keeps its existing logical-canvas composition and timing rules. Recognized GIF ICC extensions fail explicitly in this increment.

The loader accepts a context. Metadata loops and pixel loops check cancellation. Existing decoder calls remain cooperative boundaries rather than preemptible operations.

## EXIF orientation

Read orientation from JPEG APP1 EXIF, PNG eXIf, WebP EXIF, and the first TIFF image directory. Missing orientation means 1. A present orientation must have the expected scalar type and a value from 1 through 8. Malformed or conflicting orientation metadata fails.

For stored coordinates `(x, y)` and dimensions `(W, H)`, the source-to-output mapping is:

| Orientation | Output coordinate | Output dimensions |
|---|---|---|
| 1 | `(x, y)` | `(W, H)` |
| 2 | `(W-1-x, y)` | `(W, H)` |
| 3 | `(W-1-x, H-1-y)` | `(W, H)` |
| 4 | `(x, H-1-y)` | `(W, H)` |
| 5 | `(y, x)` | `(H, W)` |
| 6 | `(H-1-y, x)` | `(H, W)` |
| 7 | `(H-1-y, W-1-x)` | `(H, W)` |
| 8 | `(y, W-1-x)` | `(H, W)` |

Crop coordinates and reported working dimensions refer to the normalized image. Stored dimensions remain available in normalization metadata. Orientation preserves pixel count and moves each alpha value with its pixel.

## Color interpretation

The working space is gamma-encoded sRGB. Untagged RGB and grayscale pixels use the existing sRGB assumption. Untagged CMYK input fails because an arbitrary device-CMYK conversion would imply unsupported color accuracy.

Selected embedded ICC support covers version 2 and version 4 RGB input/display/color-space profiles with XYZ PCS and matrix/TRC transforms. Supported curves include identity, single-gamma and sampled `curv` tags, plus `para` function types 0 through 4. Each colorant forms one matrix column.

The transform decodes source channels through their curves, maps them to D50 PCS XYZ, adapts to D65 once, and encodes sRGB. Colorants already describe the D50 PCS. The implementation must not apply their profile chromatic-adaptation tag a second time. Output channel clipping provides the documented out-of-gamut behavior. This is a bounded colorimetric conversion without perceptual gamut mapping or black-point compensation.

Reject unsupported versions, non-RGB device spaces, non-XYZ PCS values, device-link profiles, and unsupported profile classes. Reject profiles with LUT transforms instead of ignoring those transforms and using incidental matrix tags. Malformed tag spans, conflicting entries, invalid curve domains, and unsupported curve types fail explicitly.

ICC extraction covers JPEG APP2 segment assembly, PNG iCCP decompression, WebP ICCP, and the first TIFF directory's profile tag. BMP V5 embedded profiles use the same ICC admission checks. BMP V4/V5 sRGB and Windows sRGB declarations select sRGB. Linked profiles, calibrated color spaces, and unknown color-space declarations fail. The loader never reads an external profile. BMP input remains subject to the existing decoder limitations.

Preserve source alpha precision through normalization, including 16-bit alpha. The existing engine later performs its documented 8-bit quantization. Color transforms operate on unassociated color channels. Encoders receive sRGB values and discard source EXIF and ICC metadata. This increment does not add output ICC embedding.

## PNG precedence

Apply this order to supported PNG metadata:

1. Supported cICP indicating `[1, 13, 0, 1]`.
2. Embedded ICC profile.
3. sRGB declaration.
4. gAMA and/or cHRM interpretation.
5. Assumed sRGB.

Reject any other cICP tuple, including HDR encodings. Reject an ICC profile combined with an sRGB chunk. Validate recognized container metadata and payload bounds even when a higher-priority source exists. A supported cICP declaration takes precedence over ICC profile internals. The loader does not parse or apply that lower-priority ICC transform. Selected malformed or unsupported profiles still fail.

Use the supported PNG gamma/chromaticity conversion when those chunks determine color interpretation. Preserve alpha and validate the resulting transform before processing pixels. Catalog metadata and documentation describe this precedence.

## Resource and error boundary

Limit expanded ICC profiles and EXIF payloads to 4 MiB each. Apply the profile limit after decompression and segment assembly. Validate lengths and offsets before slicing or allocating. Limit profiles to 256 tags and each sampled curve to 65,536 entries. Limit the EXIF first image directory to 4,096 entries. Existing encoded-byte, dimension, pixel, aggregate-frame, and timeout limits still apply.

Normalize before returning pixels to downstream operations. Failure returns an actionable error and no normalized image. An operation with failed normalization cannot publish an output. In a batch, other items retain their independent transaction semantics.

## Result metadata

Inspection exposes a `normalization` object with these fields:

| Field | Meaning |
|---|---|
| `stored_width`, `stored_height` | Dimensions in the source raster |
| `exif_orientation` | Source orientation, defaulting to 1 |
| `orientation_applied` | Whether normalization changed pixel orientation |
| `color_source` | Metadata source or the documented sRGB assumption |
| `working_space` | `srgb` |
| `color_converted` | Whether the loader applied a color transform |
| `icc_sha256` | Optional digest of the selected assembled source profile |

The source SHA-256 continues to describe original encoded bytes. Normalization does not change those bytes. The capability catalog exposes supported orientation values, color-profile shapes, metadata precedence, bounds, and exclusions. MCP and CLI use the same service policy.

## Dependency decision

The focused library review found no reason to broaden this increment's contract automatically.

- [prism](https://github.com/mandykoh/prism) provides useful pure-Go color utilities and metadata extraction. Its documented gaps include arbitrary ICC transforms and CMYK.
- [go-icc](https://pkg.go.dev/seehuhn.de/go/icc) implements broader transforms under GPL-3.0. Adding it would change the dependency licensing decision.
- [gfx/color v0.34.0](https://pkg.go.dev/github.com/go-gfx/gfx@v0.34.0/color) provides BSD-licensed matrix and LUT transforms. Source review found admission gaps for this service's contract. These include profile-header checks, duplicate-tag rejection, declared-size boundaries, and curve-domain validation.

A small bounded implementation keeps the supported subset explicit. Broad LUT/CMYK support requires separate parser, resource, reference-image, and licensing review. The library findings describe this integration decision rather than a general assessment of those projects.

Primary references are the [ICC profile specification](https://www.color.org/specification/ICC.1-2022-05.pdf), [CIPA EXIF standards](https://www.cipa.jp/e/std/std-sec.html), and [PNG specification](https://www.w3.org/TR/png-3/). Implementation tests must check numerical behavior independently of the finite Quint abstraction.

## Verification

Use asymmetric image fixtures to verify all eight orientations and crop ordering. Test alpha transport with transparent, partial, opaque, and 16-bit values. Test accepted ICC curve shapes and known color transforms. Cover malformed profiles, unsupported LUT/CMYK, segment assembly, compressed metadata limits, and PNG precedence.

Exercise the shared loader through multiple operations and both transports. Verify that metadata failures publish nothing and that old untagged fixtures remain stable. The Quint model checks admission, permutations, alpha transport, at-most-once normalization, and downstream readiness. It does not model floating-point color science or binary metadata parsers.
