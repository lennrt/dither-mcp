# Studio recipes and source-color coverage

## Accepted recipe authority

A separate recipe destination and action save the exact version-1 recipe from the last accepted preview. Image export dimensions do not modify this recipe. Dirty settings, pending operations, canceled previews, or missing tool capability prevent saving. A recipe save is explicit and creates no image.

The app validates a successful artifact receipt before reporting success. The path, JSON format, positive byte count, SHA-256 shape, and recipe must match the operation. Failures preserve the accepted preview for a new-destination retry. Superseded responses cannot update newer state.

Recipes preserve custom colors, hidden engine options, and image-mask settings. They omit source and mask file paths. The UI explains that image-mask recipes require an external mask when reused.

The embedding adds only `dither_recipe_save`. It admits exact output/recipe fields, a relative JSON destination, and the latest accepted preview recipe. Starting preview or extraction invalidates that authority until a successful preview supplies a new recipe. The server retains final validation, rooted paths, and atomic no-clobber publication.

## Full-source histogram

Scan every visible pixel, accumulating alpha-weighted exact RGB counts up to 262,144 distinct colors. If another distinct color would exceed the bound, fold into 6-bit-per-channel bins. Each bin retains original-RGB weighted sums and weights in 64-bit integers. Subsequent pixels contribute to those same bounded bins. Compute bin centroids after scanning to order and split boxes. Preserve original sums through final box averaging, avoiding repeated rounding. Use deterministic median cut and existing final color ordering.

This removes positional omission of visible pixels. It does not guarantee that all rare colors survive reduction to the requested 2–256 colors. Sources with more than 262,144 distinct colors merge nearby RGB values before median cut. Fully transparent pixels never contribute.

Images with at most 262,144 pixels retain exact histograms and successful extraction behavior. Larger sources may produce different palettes and overflow GIF colors. Full scans increase read work relative to stride sampling, within the existing 16,777,216-pixel limit. Cancellation checks bound scan and histogram-fold work. The histogram bound is independent of source pixel count.

## Verification boundaries

Concrete tests reproduce missed singleton and periodic colors, check alpha-weighted references, origins, overflow, cancellation, and successful encoder bytes. Measure large-source runtime and allocations without timing-based correctness assertions. Recipe tests use the real stdio server, accepted pixel replay, and restricted browser host.

The finite models abstract accepted recipe identity and histogram color keys. They check publication authorization, distinct-color bounds, and source contribution. They do not prove browser behavior, SHA-256 identity, numerical color quality, runtime, or memory allocation sizes. Implementation tests and measurements cover those separate claims.
