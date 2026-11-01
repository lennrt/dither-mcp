## ADDED Requirements

### Requirement: Reusable comparison cell recipes

Every comparison cell SHALL include a valid version-1 recipe with its effective palette, algorithm, reusable options, and exact rendered dimensions. Recipes SHALL describe the cell's processing pixels before contact-sheet encoding. Output-format transformations SHALL remain separate from the reusable recipe. Recipes SHALL omit source, output, and mask paths. Image-mask recipes SHALL retain the external-mask requirement. Shared artifact recipe behavior SHALL remain compatible.

#### Scenario: Replay a selected comparison cell

- **WHEN** an agent saves a cell recipe from an algorithm or palette comparison and renders it with the same source and required mask
- **THEN** the render reproduces the cell's processing pixels before output encoding, including default comparison dimensions and advanced options

#### Scenario: Encode a comparison sheet

- **WHEN** a comparison uses JPEG, GIF, or another output with color or alpha transformations
- **THEN** the cell recipe retains its processing settings without promising identical decoded pixels after sheet encoding and separate cell encoding
- **AND** documentation identifies PNG and TIFF as lossless formats for pixel comparisons, JPEG block loss, GIF sheet-wide palette selection, and BMP white compositing

### Requirement: Complete still-format discovery

Every advertised still-image output format SHALL have exactly one catalog format role with its MIME type, representation, and alpha behavior. BMP SHALL report opaque RGB composited against white. TIFF SHALL report uncompressed unassociated RGBA preserving full alpha.

#### Scenario: Discover interchange output semantics

- **WHEN** an agent discovers BMP and TIFF in the still-image output formats
- **THEN** the corresponding format roles describe their MIME types and distinct alpha behavior

### Requirement: Unambiguous advanced CLI options

The CLI SHALL reject an engine setting supplied in both `--options` and an explicit dedicated flag before opening the workspace. Explicit zero and false flags and null JSON values SHALL count as supplied settings. Distinct settings SHALL combine.

#### Scenario: Reject duplicate geometry settings

- **WHEN** a render command supplies `--width 0` and `--options '{"width":320}'`
- **THEN** the CLI reports the conflicting flag and JSON option without workspace access or publication

#### Scenario: Combine distinct settings

- **WHEN** a command supplies `--width 320` and advanced JSON with algorithm and effects settings
- **THEN** the shared service receives all those settings

### Requirement: Complete admitted batch outcomes

An admitted batch SHALL return one indexed outcome per submitted item in input order. Cancellation observed during the batch SHALL produce errors for every unstarted item and preserve earlier published artifacts. Cancellation before admission MAY fail the call without item outcomes.

#### Scenario: Cancel after one published item

- **WHEN** a batch publishes its first item and observes cancellation before its remaining items publish
- **THEN** the response retains the first artifact and reports cancellation errors for all remaining indices
- **AND** canceled items create no artifacts
