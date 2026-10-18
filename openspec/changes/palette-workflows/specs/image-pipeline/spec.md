## ADDED Requirements

### Requirement: Sparse visible palette extraction

If the normal fixed-stride extraction sample contains no visible pixels, the engine SHALL search each stride-sized block for a visible sample. It SHALL retain at most 262,144 alpha-weighted samples, preserve deterministic ordering, and observe cancellation. It SHALL reject full transparency only after confirming that the source has no visible pixels. Existing nonempty samples SHALL preserve their behavior.

#### Scenario: Visible pixels fall between sample positions
- **WHEN** every fixed-stride sample is transparent but the admitted source contains visible pixels
- **THEN** extraction returns the palette derived from bounded visible fallback samples

#### Scenario: Source is fully transparent
- **WHEN** neither the fixed-stride sample nor the fallback finds a visible pixel
- **THEN** extraction returns the full-transparency error

#### Scenario: Cancel the fallback scan
- **WHEN** cancellation is observed during sparse-image recovery
- **THEN** extraction returns cancellation without a palette
