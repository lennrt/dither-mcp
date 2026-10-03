## ADDED Requirements

### Requirement: Discoverable algorithm semantics

The suite SHALL expose stable identifiers, families, and descriptions for its implemented algorithms. These SHALL include error diffusion, ordered screens, threshold quantization, and artistic screens. Unknown identifiers SHALL produce errors. A catalog entry SHALL correspond to implemented behavior rather than an unsupported alias.

#### Scenario: Discover and select
- **WHEN** an agent lists the catalog and submits a listed algorithm identifier
- **THEN** the engine executes the corresponding kernel or screen

#### Scenario: Unknown algorithm
- **WHEN** an agent submits a misspelled or unsupported identifier
- **THEN** the engine rejects it and makes supported choices discoverable

### Requirement: Explicit palette handling

The suite SHALL support named palettes and custom hexadecimal colors. It SHALL support deterministic extraction of a bounded palette from a source image. Palette validation SHALL reject invalid colors and unsupported sizes. Quantization without region masking or post-effects SHALL use only the selected palette RGB values while preserving documented alpha behavior.

#### Scenario: Custom duotone
- **WHEN** an agent renders an opaque source with exactly two custom colors and no mask or post-effects
- **THEN** every output RGB triplet is one of those two colors

#### Scenario: Stable extraction
- **WHEN** the engine samples the same source with the same extraction size and settings
- **THEN** the extracted palette and its ordering are repeatable

### Requirement: Validated adjustments and geometry

The pipeline SHALL support documented crop/resize/pixel scaling, brightness, contrast, gamma, saturation, inversion, and strength controls. It SHALL reject non-finite numbers, invalid rectangles, invalid dimensions, and out-of-range options before expensive work.

#### Scenario: Palette-ready crop
- **WHEN** an agent supplies a valid crop, target width, and pixel scale
- **THEN** the result follows the documented crop, resize, adjustment, quantization, and scale order

#### Scenario: Invalid gamma
- **WHEN** gamma is zero, negative, non-finite, or outside its supported interval
- **THEN** the engine returns a validation error

### Requirement: Reproducible seeded processing

The engine SHALL produce identical pixels for equal decoded inputs, normalized configurations, seeds, and engine versions. Randomized effects SHALL use a request-local seeded generator. They SHALL not use process-global mutable randomness.

#### Scenario: Repeat seeded noise
- **WHEN** a stochastic algorithm renders the same image twice using the same seed
- **THEN** the resulting pixel buffers are identical

#### Scenario: Parallel reproducibility
- **WHEN** independent requests with the same source and settings execute concurrently
- **THEN** their outputs remain identical

### Requirement: Region masks and creative effects

The engine SHALL provide documented geometric region masks and composable creative controls sufficient for localized dithering and retro effects. The documentation SHALL identify which controls run before or after quantization. It SHALL state when final palette membership is no longer guaranteed.

#### Scenario: Dither a selected region
- **WHEN** a rectangle or circle mask selects a region
- **THEN** the engine applies the quantized result according to the documented inside/outside mask behavior

### Requirement: Explicit export semantics

The suite SHALL support PNG and the documented GIF, SVG, PBM, and ASCII export operations. It SHALL reject unsupported format combinations instead of silently renaming bytes. SVG output SHALL encode the raster result as safe generated geometry without user-controlled markup or external references. PBM SHALL follow a documented monochrome conversion.

#### Scenario: Export inspectable vector artwork
- **WHEN** an agent exports a small dither to SVG
- **THEN** the result contains generated geometry representing output pixels and no remote dependencies

#### Scenario: Unsupported format
- **WHEN** an agent requests an unsupported output format
- **THEN** the request fails before publication
