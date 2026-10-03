# A palette library agents can explore

## Why

An agent should be able to discover a color direction by subject, mood, color count, or design tradition. It should then be able to apply that direction immediately. The initial 22 presets provide a useful starting point. A broader searchable library adds creative range without image uploads or another interface.

## What changes

- Expand the built-in library to 256 distinct palettes while preserving the initial 22 identifiers and exact ordered colors.
- Add meaningful categories, descriptions, searchable tags, and truthful provenance to each palette.
- Extend the existing palette-discovery tool with bounded text/category/color-count filters and deterministic pagination.
- Expose the same discovery through MCP and the CLI. Preserve named-palette rendering and custom color recipes.
- Add an executable finite Quint model of discovery, page soundness, ordering, bounds, and traversal. Add Go regression tests for the actual catalog.
- Update palette documentation and the showcase around the added creative range.

## Capabilities

### New capabilities

- `palette-library`: curated catalog metadata, stable identities, set-level uniqueness, bounded filtered discovery, deterministic pagination, and transport parity.

### Modified capabilities

None. The initial `image-pipeline` palette-handling contract remains in force. This change adds discovery and breadth without reinterpreting existing recipe identifiers.

## Impact

The runtime stays local and the tool count remains unchanged. The `dither_palettes` request gains optional filters and pagination. Its result retains `palettes` and adds metadata for navigating results. Agents must follow `next_offset` to enumerate the library when the default page is smaller than the catalog. Existing named-palette recipes preserve their rendering colors.

The current TurboDither palette page lists 15 presets. A 256-entry library exceeds ten times that reference count. Count is only a floor. Duplicate sets, simple reorderings, fabricated source attribution, and undocumented generated variants do not count as additional curated choices.
