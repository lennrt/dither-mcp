## ADDED Requirements

### Requirement: Bounded full-source palette coverage

Palette extraction SHALL inspect every admitted source pixel and include every nonzero-alpha pixel in an alpha-weighted histogram. It SHALL retain exact RGB weights up to 262,144 distinct colors. Beyond that bound it SHALL use at most 262,144 deterministic 6-bit-per-channel bins with original-RGB weighted centroids. Accumulation SHALL use weights and sums that cannot overflow at the admitted pixel limit. Median cut SHALL remain deterministic and return at most the requested count of distinct opaque colors. Transparent pixels SHALL NOT contribute. Scan and fold work SHALL observe cancellation.

#### Scenario: A rare accent lies between old sample positions
- **WHEN** a large image contains a rare visible accent and its distinct colors fit the requested count
- **THEN** extraction includes that accent exactly even if former stride samples missed it

#### Scenario: Periodic minority colors
- **WHEN** repeating source colors align with former sample gaps
- **THEN** every visible pixel contributes its alpha weight instead of selecting one periodic subset

#### Scenario: Many distinct source colors
- **WHEN** a source exceeds the exact histogram color bound
- **THEN** the histogram folds into bounded bins and preserves full-source alpha-weighted contributions
- **AND** the same source produces the same palette without guaranteeing preservation of every rare color after reduction

#### Scenario: Small-source compatibility
- **WHEN** the source contains at most 262,144 pixels
- **THEN** extraction uses the same exact alpha-weighted histogram and deterministic median cut as before

#### Scenario: Transparent or canceled input
- **WHEN** every pixel is transparent or cancellation occurs during scan or fold work
- **THEN** extraction returns the applicable error and no partial palette

## MODIFIED Requirements

### Requirement: Sparse visible palette extraction

The engine SHALL detect visible pixels regardless of their position in the source. It SHALL retain bounded alpha-weighted histogram storage, preserve deterministic ordering, and observe cancellation. It SHALL reject full transparency only after confirming that the source has no visible pixels. The full-source coverage requirement defines histogram bounds and small-source compatibility.

#### Scenario: Visible pixels fall between former sample positions
- **WHEN** an admitted source contains visible pixels between former fixed-stride positions
- **THEN** extraction includes their alpha-weighted contributions

#### Scenario: Source is fully transparent
- **WHEN** the full scan finds no visible pixel
- **THEN** extraction returns the full-transparency error

#### Scenario: Cancel a sparse source scan
- **WHEN** cancellation is observed during sparse-image scanning
- **THEN** extraction returns cancellation without a palette

