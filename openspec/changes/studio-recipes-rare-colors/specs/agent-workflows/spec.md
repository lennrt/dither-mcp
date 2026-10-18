## ADDED Requirements

### Requirement: Explicit studio recipe saving

The studio SHALL offer an explicit recipe-save action using `dither_recipe_save`. It SHALL save the exact version-1 recipe from a current, clean, accepted preview. The recipe SHALL retain preview dimensions, custom colors, and advanced options independently of image export dimensions. Dirty settings, pending work, unavailable tools, or invalidated previews SHALL prevent recipe saving. Saving a recipe SHALL create no image and SHALL use a separate relative JSON destination.

#### Scenario: Preserve an extracted treatment
- **WHEN** a user extracts colors, accepts a preview, and saves its recipe to a new JSON destination
- **THEN** loading that recipe and rendering the same source reproduces the accepted preview pixels
- **AND** the saved recipe includes the accepted custom colors and advanced settings

#### Scenario: Export dimensions remain independent
- **WHEN** a user selects source-size or custom image export dimensions and saves a recipe
- **THEN** the recipe still contains the accepted preview dimensions

#### Scenario: Masked recipe reuse
- **WHEN** an accepted preview uses an external image mask
- **THEN** the saved recipe retains image-mask options without storing source or mask paths
- **AND** the studio explains that reuse needs an external mask

#### Scenario: Dirty or busy settings cannot save a recipe
- **WHEN** another studio operation is pending
- **THEN** the studio makes no recipe-save call until work is idle
- **AND** settings edits or preview/extraction invalidation require a fresh current preview, while completed image or recipe saves preserve the accepted preview

#### Scenario: Failed or stale recipe publication
- **WHEN** a recipe destination exists, the tool fails, or a receipt does not match the requested path and recipe
- **THEN** the studio reports failure rather than success and permits a new-destination retry
- **AND** a completion from a superseded operation cannot alter newer state

### Requirement: Restricted embedding recipe authority

The reusable embedding host SHALL permit recipe saving only for its latest successful studio recipe. Preview or extraction initiation SHALL invalidate recipe permission until another successful preview. It SHALL admit only output and recipe arguments, a workspace-relative JSON destination, and a recipe matching the accepted settings. It SHALL NOT grant arbitrary recipe loading or unrelated tool access.

#### Scenario: Save the accepted recipe
- **WHEN** the embedded studio saves its current accepted recipe
- **THEN** the host forwards that exact recipe to the existing server and preserves its artifact receipt

#### Scenario: Reject broader recipe authority
- **WHEN** a caller changes the recipe, supplies extra fields, or uses an absolute, traversal, source, or mask destination
- **THEN** the host rejects the call before forwarding it to the server

#### Scenario: Preview replacement revokes recipe permission
- **WHEN** another preview or extraction starts, including an operation that later fails
- **THEN** the host rejects recipe saving until a successful preview supplies the currently authorized recipe
