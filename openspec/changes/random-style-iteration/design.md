## API contract

`dither_random_style` accepts only an optional integer seed from -9007199254740991 through 9007199254740991. Omission chooses a cryptographically random seed; zero is reproducible and null is invalid. The response contains the seed and a validated version-1 recipe. The existing options serialization may omit a zero engine seed; omission means zero.

A local SplitMix64 stream chooses a catalog algorithm and palette, moderate discrete brightness/contrast/gamma/saturation/strength values, sRGB or linear RGB, diffusion direction when applicable, and at most one subtle texture effect. Recipes exclude source paths, dimensions, pixel scale, resize filter, crop and masks. Generation needs no source image and publishes no artifact. Seed reproducibility is scoped to the current engine and catalog version; save the complete recipe for later reuse.

## App interaction

Random style is a primary action above the preview with an invitation to try it again. One operation spans generation and preview. It preserves the current dimensions, pixel scale, resize filter, crop and mask, replacing style values only once the generated preview succeeds. Invalid pending advanced JSON is retained and causes no call. Pending custom palette text is intentionally replaced by a successful generated palette.

A new click supersedes read-only work, including either random stage. The app aborts the older request and rejects late results using operation identity and settings revision. Failed or canceled work preserves the displayed image and controls, but cannot authorize saving. Image and recipe writes remain exclusive and cannot be superseded.

The embedding host allows only the optional-seed random call. It tracks operation identity and cancellation rather than a bare busy flag. Only the latest successful preview can restore accepted recipe authority; an older finally block cannot clear a newer operation. Existing source confinement and explicit save gates remain.

## Verification boundaries

The finite app model adds generation and preview phases under one identity, supersession of pending reads, write exclusion, and stale/failure rejection. It abstracts generated colors and numerical processing. Concrete Go tests cover seeds, valid selection, actual rendering and schemas; browser and host tests exercise delayed two-stage races and real SDK calls. Image examples demonstrate specific seeds without promising all random looks suit every source.
