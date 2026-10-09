# Guarded studio exports and usable host setup

## Why

The bounded studio preview should remain useful when the final image needs more pixels. A save must also detect source or image-mask edits since previewing, rather than silently apply an accepted recipe to different bytes. Readers need an install path whose actual transport and host requirements are explicit, plus a reusable embedding example.

## What changes

- Preserve exact preview size as the default save mode. Add explicit source-size and custom-size export choices, changing only width and height in the accepted recipe.
- Return normalized, cropped source dimensions, export limits, and SHA-256 fingerprints with previews. Check optional expected source and mask fingerprints against the same bounded buffers used for decoding.
- Report a changed source or mask as a recoverable structured error and require a fresh accepted preview before another studio save.
- Document local stdio and MCP Apps host requirements, dated compatibility evidence, installation, fallback behavior, and transport exclusions. Explicitly permit `data:` resource URLs so hosts can display the inline PNG preview under the declared CSP.
- Provide a runnable embedding example using the official Apps host bridge and the real local stdio server through a loopback development adapter.
- Add finite export and input-snapshot contracts, regression tests, mutation checks, and explicit verification limits.

## Capabilities

### New capabilities

None.

### Modified capabilities

- `agent-workflows`: guarded previews and saves, independent export dimensions, and recoverable input-change errors.
- `delivery-quality`: reproducible host setup and embedding guidance with dated compatibility evidence.

## Impact

Preview metadata and optional render arguments are additive. Ordinary unguarded render clients keep their existing behavior. The studio sends guards for accepted previews. Preview limits stay at 1,024 pixels per axis and 2 MiB PNG; export uses normal engine admission limits. Choosing another resolution can change the dither pattern and is not an exact pixel replay. The released server remains a local stdio executable without a new production HTTP endpoint.
