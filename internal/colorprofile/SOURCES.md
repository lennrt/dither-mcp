# RGB normalization

This package converts source RGB to sRGB with a media-relative colorimetric transform. It clips destination channels to their representable range. It preserves source alpha and uses 16-bit unassociated RGB throughout conversion. It performs no perceptual gamut mapping or black-point compensation.

The parser supports ICC v2 and v4 input, display, and color-space profiles with RGB data and an XYZ D50 PCS. It reads the three XYZ colorant columns and the three tone curves. The colorants already contain source adaptation. The parser validates the optional `chad` matrix but never applies it again.

The parser limits profiles to 4 MiB, tags to 256, and sampled curves to 65,536 entries. It validates declared size, signatures, ranges, shared data, padding, required transform tags, and curve parameters. Tag data must form a contiguous sequence. It rejects LUT pipelines, multiprocess elements, HDR `cicp` tags, unsupported classes, and other source color spaces. Descriptive metadata never changes conversion behavior. The parser does not interpret descriptive tag payloads.

The numerical model follows [ICC.1:2022](https://www.color.org/specification/ICC.1-2022-05.pdf), especially sections 10.6 and 10.18 and Annexes E and F. Type-4 parametric curves use the corrected equation in Table 68. Sampled curves use linear interpolation. Parametric output values are clipped to `[0,1]`.

`FromPNG` derives a primary matrix from `cHRM` coordinates and applies linear Bradford adaptation to the ICC D50 white. Positive `gAMA` values use exponent `1/gamma`. Zero uses the sRGB transfer function. Nil chromaticities select sRGB primaries and D65. The caller applies container metadata precedence before constructing a transform.

The sRGB transfer function and primary coordinates follow the [ICC sRGB registry](https://registry.color.org/rgb-registry/srgb). Display P3 uses the [ICC Display P3 registry](https://registry.color.org/rgb-registry/displayp3). Adobe RGB uses the [ICC Adobe RGB registry](https://registry.color.org/rgb-registry/adobergb).

## Independent verification

The saved fixtures are original profiles generated from published primaries and transfer functions. `testdata/oracle.py` creates them through LittleCMS and reopens their serialized bytes before computing reference values. The saved table contains 45 RGB16 vectors from LittleCMS 2.19, with relative colorimetric intent and optimization disabled. Go tests consume these files without a native library.

The conversion tests allow 32 RGB16 steps for the independent LittleCMS comparison. This bound accounts for differences in the encoded destination matrix and remains below one eighth of an 8-bit channel step. Published leaf coordinates from [CSS Color 4](https://www.w3.org/TR/css-color-4/) provide a separate numerical check. Tests also verify PNG neutral values, supported curves, alpha precision, cancellation, and invalid input bounds.

Optional compatibility tests read installed Apple Display P3, Adobe RGB, and sRGB profiles without changing or redistributing them. The checked-in fixtures contain no Apple profile bytes. The runtime uses pure Go and requires no LittleCMS installation.

To regenerate the independent fixtures, install LittleCMS and run:

```sh
python3 internal/colorprofile/testdata/oracle.py
```

Set `LCMS_LIBRARY` to the shared-library path if Python cannot locate it. The script processes numeric RGB arrays and writes test profiles and reference values. It does not process images.
