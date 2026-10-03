## ADDED Requirements

### Requirement: Animated GIF processing

The suite SHALL process multi-frame GIFs with bounded frame and canvas sizes. It SHALL composite frames in source order and honor disposal semantics. Exported GIF delays and loop behavior SHALL follow documented rules and format limits.

#### Scenario: Dispose-to-background source
- **WHEN** a GIF uses transparent partial frames and disposal-to-background
- **THEN** the suite composites each source frame correctly before processing and prevents obsolete pixels from leaking from the previous frame

### Requirement: Generated loops and sprite sheets

The suite SHALL generate deterministic finite animated variations from a still image. It SHALL support sprite-sheet export with an explicit layout. It SHALL reject excessive frame counts or sheet dimensions before allocation.

#### Scenario: Reproducible loop
- **WHEN** an agent requests a supported animation mode, frame count, seed, and palette
- **THEN** the resulting frames and their ordering are reproducible

#### Scenario: Sprite-sheet dimensions
- **WHEN** an agent requests N frames and C columns
- **THEN** the sheet uses C columns and enough rows for N frames, subject to configured dimensions and pixel limits

### Requirement: Optional local video

When video support is available and enabled, the suite SHALL use local FFmpeg/ffprobe processes. These processes SHALL have context cancellation and explicit duration, frame, and dimension limits. It SHALL reject remote and playlist inputs, avoid shell command construction, and report missing dependencies clearly. Documentation SHALL state output codec and audio behavior.

#### Scenario: Video dependency unavailable
- **WHEN** an agent requests video processing without a usable local FFmpeg dependency
- **THEN** the tool returns a capability or dependency error while still-image operations remain usable

#### Scenario: Local clip
- **WHEN** an agent supplies a supported local clip with bounded duration and output dimensions
- **THEN** the tool processes only the allowed frames and returns the resulting local video artifact

### Requirement: Print-oriented separation

The suite SHALL expose documented monochrome and per-palette-color separation artifacts suitable for further print preparation. It SHALL state that digital color separation is not a calibrated press profile or a physical print-quality guarantee.

#### Scenario: Separate a limited-color render
- **WHEN** an agent requests separations for a palette-limited image
- **THEN** it receives the documented mask artifact for each selected color with reproducible ordering
