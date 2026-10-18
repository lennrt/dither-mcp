# Palette workflow design

## Comparison requests

`dither_compare` accepts either `algorithms` or `palettes`, with 1–12 entries. Palette mode takes named palette IDs and one shared algorithm through `options.algorithm`. It rejects simultaneous algorithm candidates, a shared palette, custom colors, and saved recipes. These exclusions prevent silent overrides. Duplicate candidates remain valid, as in algorithm mode.

The shared service decodes one source, resolves candidate palettes, and renders cells in request order. The existing geometry, mask, byte, cancellation, and no-clobber rules apply. Palette cells contain `palette`, `algorithm`, and their rectangles. Full operation parameters remain in artifact provenance. Palette sheets omit the single `artifact.recipe`, because one palette recipe cannot describe all cells. To replay a cell, use its palette and algorithm, set width and height to its cell dimensions, and retain the other original options and mask. This preserves the comparison default of width 320 when the request omitted dimensions.

The CLI adds `compare --palettes A,B,C`. Its default algorithm shortlist applies only when palette mode is absent. Explicitly supplying both shortlist flags fails. MCP and CLI call the same service.

## Studio extraction

An explicit action calls `dither_palette_extract` with the current input and a count from 2 through 256. Extraction uses the complete normalized source image before studio crop, adjustments, or masking. The existing tool requires at least two distinct colors. The app validates returned colors before adopting them as custom colors.

Extraction participates in the same single-operation, request-identity, and control-revision rules as previewing. A late response cannot replace edits or another source. Successful extraction preserves every non-palette setting, invalidates the accepted save state, and requires Apply preview. It creates no artifact and does not implicitly save or render. Errors preserve the draft and explain the failure.

The embedding host allows this third tool only for its selected source, with only `input` and `count` arguments. Extraction does not consume the selected image mask. Preview and render calls retain their source-and-mask restrictions. Omitted initial dimensions use the studio's fit within 512 × 512 pixels. The host adds its default palette only when neither a palette nor custom colors is supplied.

## Sparse sampling

Keep the current bounded, fixed-stride sample when it sees any visible pixel. This preserves successful extracted palettes. If the sample is empty, inspect each stride-sized block for its first visible pixel. Collect at most one alpha-weighted sample per block, then use the existing deterministic median cut and color ordering.

The fallback may read every admitted source pixel, but stores at most 262,144 samples. It checks cancellation during traversal. A truly transparent image still fails. This fallback corrects false full-transparency reports; it does not eliminate all periodic bias in nonempty fixed-stride samples.

## Verification

Reference renders check comparison pixels and metadata. Regression fixtures expose sparse sampling and embedding initialization failures on the original code. Controller, stdio, and browser checks cover extraction, stale results, validation, and explicit saving. Quint abstracts candidate order and bounds, empty-sample recovery, and asynchronous studio state. These models do not prove color quality, real geometry arithmetic, codecs, or implementation refinement.

## Late response recovery

The SDK can report an unknown response ID after local request cancellation. Its error callback does not establish transport closure. Ignore only the known stale-response/progress diagnostics, report other communication errors, and disable tool interaction when the transport actually closes. Browser tests deliver a canceled request's completed reply explicitly and require the next preview/save workflow to remain usable.
