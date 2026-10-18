## ADDED Requirements

### Requirement: Bounded palette comparison

The comparison tool SHALL accept either 1–12 algorithm IDs or 1–12 named palette IDs. Palette comparisons SHALL use one shared algorithm and options. Palette mode SHALL reject simultaneous algorithm candidates, a shared palette, custom colors, or a saved recipe. Cells SHALL follow request order and identify each palette and algorithm. Palette sheets SHALL omit a single shared artifact recipe and retain complete operation parameters. Existing image limits, guards, cancellation, and publication rules SHALL apply.

#### Scenario: Compare three palettes
- **WHEN** an agent supplies three palette IDs and a shared algorithm
- **THEN** each cell matches a standalone render using that palette and the shared settings
- **AND** cell metadata identifies the corresponding palette and rectangle in request order

#### Scenario: Reject ambiguous candidates
- **WHEN** a request supplies both palette and algorithm shortlists or combines palette candidates with a saved recipe
- **THEN** comparison fails without publishing an artifact

#### Scenario: Preserve algorithm comparisons
- **WHEN** an existing caller supplies only algorithm candidates and one shared palette
- **THEN** the existing algorithm comparison behavior remains available

### Requirement: Explicit studio source-color extraction

The studio SHALL offer an explicit read-only source-color extraction action with a requested count from 2 through 256. It SHALL use the existing extraction tool on the complete normalized source. It SHALL accept at least two valid distinct colors as custom draft colors while preserving all non-palette settings. Successful extraction SHALL require a new explicit preview before saving. Results SHALL apply only to the current operation and control revision. Canceled, stale, failed, or malformed results SHALL NOT replace the draft or publish files.

#### Scenario: Use the image's colors
- **WHEN** a user requests extraction and receives a current valid palette
- **THEN** the studio selects those custom colors and preserves its other settings
- **AND** Save remains disabled until Apply preview succeeds

#### Scenario: User edits during extraction
- **WHEN** controls or the source change before extraction returns
- **THEN** the late response cannot replace the newer draft

#### Scenario: A canceled extraction responds late
- **WHEN** a completed extraction response arrives after local cancellation
- **THEN** the studio ignores its colors and keeps preview and extraction available
- **AND** a nonfatal SDK late-response diagnostic does not imply that the host connection closed

#### Scenario: A source cannot supply two colors
- **WHEN** the extraction tool rejects a transparent or single-color source
- **THEN** the studio explains the error and preserves the previous settings

### Requirement: Bounded embedding initialization and extraction authority

The embedding example SHALL use the studio's default fit when initial dimensions are omitted. It SHALL add its default palette only when neither a palette nor custom colors is supplied. It SHALL permit palette extraction only from the selected source with the documented extraction fields. Preview and save SHALL retain the selected source and mask restrictions.

#### Scenario: Open a tall portrait
- **WHEN** the embedding session opens a tall portrait without explicit dimensions
- **THEN** its initial preview fits within 512 × 512 pixels without enlargement

#### Scenario: Open with custom colors
- **WHEN** initial settings provide custom colors without a named palette
- **THEN** the host preserves those colors without adding a conflicting palette

#### Scenario: Reject extraction outside the selected source
- **WHEN** an embedded app requests extraction from another path or supplies unrelated fields
- **THEN** the host rejects the call before forwarding it to the server
