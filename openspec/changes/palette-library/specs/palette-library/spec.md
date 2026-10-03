## ADDED Requirements

### Requirement: Stable existing palette identities

The library SHALL preserve the initial 22 built-in palette IDs and their exact ordered sRGB color arrays. It SHALL resolve those IDs identically after the expansion, including deterministic tie-breaking based on palette order. Adding metadata SHALL not alter rendered colors.

#### Scenario: Replay an existing Game Boy recipe
- **WHEN** a previously saved recipe names `gameboy`
- **THEN** it resolves to the same four colors in the same order as the initial release

#### Scenario: Compatibility fixture
- **WHEN** a test compares the expanded catalog with the initial palette fixture
- **THEN** all 22 IDs and ordered color arrays match exactly

### Requirement: Distinct curated palette breadth

The library SHALL include at least 256 built-in palettes with unique stable IDs. Each palette SHALL contain 2–256 distinct opaque colors normalized as lowercase `#rrggbb`. No two catalog entries SHALL have the same normalized unordered color set. A permutation SHALL not count as another palette.

#### Scenario: Reject an alias disguised as a new palette
- **WHEN** a proposed palette merely reverses another palette's colors
- **THEN** catalog validation rejects its duplicate color-set signature

#### Scenario: Resolve a catalog entry
- **WHEN** an agent supplies any listed palette ID to rendering
- **THEN** the engine resolves the exact listed colors without requiring external data

### Requirement: Meaningful metadata and provenance

Every palette SHALL include name, category, description, tags, and an `origin` of `original`, `reference`, or `approximation`. Reference and approximation entries SHALL identify a meaningful `source`. Original designs MAY omit an external source. Descriptions SHALL convey design intent and identify historical/device approximations. Metadata SHALL not trigger network access.

#### Scenario: Choose a print-inspired palette
- **WHEN** an agent browses a palette description and source information
- **THEN** it can distinguish an original print-inspired color choice from a historically sourced or device-accurate claim

### Requirement: Composable bounded filtering

Discovery SHALL limit queries to 256 UTF-8 bytes. It SHALL match case-insensitive substring terms across ID, name, description, category, tags, origin, and colors. Whitespace SHALL separate terms. Terms and filters SHALL combine with AND.

Discovery SHALL trim and lowercase categories, then require a known slug. Zero color-count bounds SHALL default to 2 and 256. Bounds SHALL lie within 2–256, with minimum no greater than maximum. Invalid bounds or unknown categories SHALL fail.

#### Scenario: Combine mood and color-count filters
- **WHEN** an agent searches for a mood within a category and a color-count interval
- **THEN** every returned palette satisfies all supplied filters

#### Scenario: Invalid range
- **WHEN** `min_colors` exceeds `max_colors` after default normalization
- **THEN** discovery returns a validation error rather than an unfiltered result

#### Scenario: No matching palettes
- **WHEN** a valid combination matches no entries
- **THEN** discovery returns an empty palette list and a matched count of zero

### Requirement: Deterministic bounded pagination

Discovery SHALL filter before slicing and sort by ID. It SHALL default limit zero to 32. It SHALL accept limits 1–256 and nonnegative offsets. Results SHALL include `palettes`, catalog `total`, filtered `matched`, page `count`, requested `offset`, and normalized `limit`. Results SHALL include `next_offset` only when matches remain. `categories` SHALL list all category IDs with unfiltered counts, sorted by ID.

#### Scenario: Enumerate a filtered library
- **WHEN** an agent starts at offset zero and repeatedly requests the returned next offset with unchanged filters and page size
- **THEN** concatenating pages returns every matching palette exactly once in the same order as the full filtered catalog

#### Scenario: Repeat a page
- **WHEN** identical discovery requests run against the same catalog version
- **THEN** they return identical palette IDs, ordering, counts, and navigation metadata

#### Scenario: Page beyond the end
- **WHEN** a nonnegative offset exceeds the matched count
- **THEN** the result preserves the requested offset, contains an empty palette list, and has no further next offset

### Requirement: Discovery parity across transports

The MCP palette tool and CLI palette command SHALL use the same application discovery logic. Both SHALL expose the same filters and pagination semantics. MCP input/output schemas and examples SHALL reflect the expanded result. An empty request SHALL remain valid and return a documented default page.

#### Scenario: Reproduce discovery on the CLI
- **WHEN** an agent supplies equivalent filters, page size, and offset through MCP and the CLI
- **THEN** both return the same palette entries and navigation metadata
