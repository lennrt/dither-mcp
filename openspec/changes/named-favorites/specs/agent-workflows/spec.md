## ADDED Requirements

### Requirement: Persistent named favorites

The shared service SHALL expose typed save, list and load tools for named favorites within the configured workspace. Each favorite SHALL retain one validated version-1 recipe without source, mask or output paths. Saved names SHALL be trimmed, case-sensitive, valid UTF-8 with 1–80 Unicode characters and no control characters. IDs SHALL be lowercase SHA-256 digests of the trimmed names. Saves SHALL publish complete bounded JSON without replacing an existing destination. Listing and loading SHALL create no files.

#### Scenario: Discover a look after restarting
- **WHEN** a client saves a named recipe and opens another server or studio session on the same workspace
- **THEN** listing returns its stable ID and name and loading returns the original recipe
- **AND** another workspace does not inherit the favorite

#### Scenario: Preserve a duplicate name
- **WHEN** two saves target the same trimmed name, including from independent services
- **THEN** at most one creates the destination and later saves report an error
- **AND** the successful recipe remains unchanged

#### Scenario: Bound and validate the library
- **WHEN** a favorite name, ID, envelope, recipe, file type or byte size is invalid
- **THEN** saving or loading rejects it without publishing an invalid favorite
- **WHEN** a recognized favorite file is unavailable during listing
- **THEN** other valid favorites remain visible and the result counts unavailable entries
- **WHEN** the library exceeds the 512-directory-entry scan limit
- **THEN** listing reports the limit rather than silently returning an incomplete library and direct loading remains available

#### Scenario: Keep library access within the workspace
- **WHEN** a client supplies a path in place of an ID or the library exposes a visible symlink or special file
- **THEN** the service rejects that access
- **AND** bounded root-relative operations prevent access outside the configured workspace

### Requirement: Explicit studio favorites and current-source loading

The studio SHALL discover saved favorites, offer refresh, and save the exact accepted preview under an explicit name. Saving SHALL require current clean settings, tool support and no pending operation. Loading SHALL retrieve a complete recipe and preview it on the current source and compatible mask before adopting controls, history or save authority. Metadata listing SHALL preserve existing preview authority. Favorites SHALL remain independent of session-only history and image export dimensions.

#### Scenario: Keep a named random look
- **WHEN** a random preview succeeds and the user saves it under a new name
- **THEN** the app sends the exact accepted recipe and reports success only for a matching receipt
- **AND** the library remains available after closing and reopening the studio

#### Scenario: Reload complete settings
- **WHEN** a saved favorite is loaded with compatible current source and mask files
- **THEN** the studio preserves its custom colors, dimensions, crop, masks and effects through a fresh preview
- **AND** only documented omitted defaults may be resolved and source fingerprints are refreshed before explicit image saving

#### Scenario: Preserve edits on rejected loading
- **WHEN** loading fails, is canceled, returns mismatched data, is superseded, or controls change while it runs
- **THEN** the old result cannot replace current controls, raw text, history or save authority
- **WHEN** a recipe needs a missing image mask, conflicts with the current external mask, or exceeds preview limits
- **THEN** the app reports the problem rather than silently changing the favorite

#### Scenario: Confine host authority
- **WHEN** the embedding app calls a favorite tool
- **THEN** list and load access only the fixed named library and save accepts only the host's latest accepted recipe
- **AND** Random style may supersede favorite reads but cannot interrupt a favorite write
