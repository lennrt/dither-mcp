# Compare and extract image palettes

## Why

The library provides 256 palettes, but contact sheets vary only algorithms. Studio users must copy extracted colors from a separate tool call. Fixed-stride extraction can also miss every visible pixel in a sparse image and incorrectly report full transparency.

## What changes

- Add bounded palette comparisons to the existing comparison tool and CLI command.
- Add explicit source-color extraction to the studio, followed by an explicit preview before saving.
- Recover visible samples when the existing extraction sample finds none.
- Initialize embedded studios with bounded aspect-preserving previews and compatible custom colors.
- Add regression checks, browser workflows, and finite contract tests.

## Capabilities

### New capabilities

None.

### Modified capabilities

- `agent-workflows`: palette comparisons, studio extraction, and embedding defaults.
- `image-pipeline`: bounded sparse-image extraction recovery.

## Impact

No new tools or runtime dependencies are required. Algorithm comparisons retain their existing behavior. Palette sheets identify each cell instead of claiming one shared palette recipe. Existing successful extraction samples retain their results. File publication, preview limits, and source guards remain enforced.
