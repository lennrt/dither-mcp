## ADDED Requirements

### Requirement: Reproducible random styles

The server SHALL provide a read-only `dither_random_style` tool returning a seed and a validated version-1 style recipe. An optional signed JavaScript-safe integer seed SHALL reproduce selection within the current engine and catalog version. An omitted seed SHALL choose new random input. Generation SHALL require no source image, create no file, and omit dimensions, pixel scale, resize filter, crop, mask and source paths from its recipe.

#### Scenario: Reproduce a generated look
- **WHEN** a client repeats a returned seed with the same engine and catalog version
- **THEN** the tool returns the same palette and engine settings
- **AND** the returned recipe can be rendered, previewed or explicitly saved using existing tools

#### Scenario: Validate seed boundaries
- **WHEN** a client supplies zero or either signed JavaScript-safe integer boundary
- **THEN** the tool accepts it as a reproducible seed
- **AND** null, fractional, string and out-of-range seeds are rejected

#### Scenario: Fresh selection without an image
- **WHEN** a client omits seed and source image
- **THEN** the tool returns a valid catalog palette and algorithm with moderate tone and texture settings
- **AND** it returns the chosen seed without reading or writing an image

### Requirement: Rapid random studio iteration

The studio SHALL prominently offer Random style as a one-click generation and preview action. It SHALL preserve geometry and mask settings, replace style settings only after a valid current preview, and permit repeated clicks to supersede read-only work. It SHALL NOT supersede image or recipe writes. Both app and embedding host SHALL reject stale generation and preview completions and authorize saving only after the current preview succeeds.

#### Scenario: Try a first look
- **WHEN** a user selects Random style
- **THEN** the studio generates and previews a new palette, algorithm, tone and texture without requiring Apply preview
- **AND** dimensions, pixel scale, resize filter, crop and mask retain their meaning
- **AND** no artifact is saved

#### Scenario: Rapid clicks finish out of order
- **WHEN** a user clicks again while generation, extraction or preview is pending
- **THEN** the latest random operation supersedes the read-only request
- **AND** older replies or cleanup cannot change the current controls, image, pending operation or authorized recipe

#### Scenario: Failure or cancellation
- **WHEN** generation or its preview fails, is canceled, or becomes stale after a settings edit
- **THEN** no generated settings are adopted and no save is authorized
- **AND** the last displayed image and user settings remain available

#### Scenario: Invalid pending options and explicit writes
- **WHEN** advanced JSON is invalid or an image or recipe save is pending
- **THEN** Random style makes no tool call and preserves the user's work

#### Scenario: Restricted random authority
- **WHEN** the embedding receives a random-style call
- **THEN** it admits only the optional seed argument
- **AND** it permits superseding read-only work while preserving exclusive explicit writes and latest-preview recipe authority
