# Save studio treatments and include rare source colors

## Why

Studio users can save an image but cannot save the accepted treatment as a reusable recipe. Fixed-stride extraction can miss small accents or periodic colors even when its sample is nonempty.

## What changes

- Add explicit recipe saving from the last accepted preview through the existing recipe tool.
- Keep recipe dimensions independent of image export settings and preserve custom colors and advanced options.
- Restrict embedding recipe writes to the current accepted recipe and a new relative JSON destination.
- Scan every source pixel into a bounded alpha-weighted histogram instead of selecting periodic positions.
- Preserve exact RGB weights until the distinct-color bound requires deterministic coarse bins.
- Test real recipe replay, publication errors, host boundaries, rare-color recovery, bounded memory, and cancellation.

## Capabilities

### New capabilities

None.

### Modified capabilities

- `agent-workflows`: explicit studio recipe saving and restricted embedding authority.
- `image-pipeline`: full-source color coverage with bounded histogram storage.

## Impact

No tools or dependencies are added. Recipes retain their existing version-1 format and omit source and mask paths. Large-image palette extraction and overflow GIF quantization may change. Images within the previous sample budget retain exact histogram behavior. The runtime retains existing image and publication limits.
