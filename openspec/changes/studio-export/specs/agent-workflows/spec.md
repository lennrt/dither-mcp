## MODIFIED Requirements

### Requirement: Read-only processed studio previews

The studio SHALL process local input, palette, engine options, and image masks under the shared normalization, path, and resource policies. It SHALL accept no output path and create no file. Default dimensions SHALL fit 512 by 512 pixels. Results SHALL fit 1,024 pixels per axis and 2 MiB of PNG. Results SHALL include source path, preview dimensions, MIME type, the versioned recipe used, and any image-mask path needed for replay. They SHALL also include normalized source dimensions after the accepted crop and before resize, current export dimension/pixel limits, and SHA-256 fingerprints of the exact encoded source and image-mask buffers decoded.

#### Scenario: User previews processing settings
- **WHEN** the studio receives a valid input and supported processing settings
- **THEN** it returns the processed image and exact replay settings without publishing an artifact

#### Scenario: Processing exceeds preview limits
- **WHEN** explicit or derived dimensions exceed a preview bound
- **THEN** the tool returns an actionable error and creates no output

### Requirement: Explicit current-preview saves

The UI SHALL preview edits only on explicit request. It SHALL accept a result only while its request identity and control revision match. Save SHALL require a current successful preview, a destination, host tool-call support, and no pending operation. Save SHALL call `dither_render` only on user action with the accepted source, recipe settings, mask, and expected byte fingerprints. Save SHALL default to exact preview dimensions; only an explicit export-size choice may alter the accepted width and height. The UI SHALL preserve unexposed settings and reject values it cannot represent exactly.

#### Scenario: Controls change after a successful preview
- **WHEN** a user changes processing controls after receiving a preview
- **THEN** saving remains disabled until a successful preview accepts those settings

#### Scenario: An old preview finishes late
- **WHEN** a response arrives for a canceled, replaced, or edited preview request
- **THEN** it cannot replace the current preview or enable saving

#### Scenario: User saves the current preview
- **WHEN** a user enters a destination and requests save for a current successful preview
- **THEN** the UI submits the accepted preview's source, exact settings, mask, and expected byte fingerprints to `dither_render` with that destination
- **AND** existing rooted, bounded, no-clobber publication policies apply


### Requirement: Self-contained host-aware UI

The studio resource SHALL embed its script, styles, and pinned official MCP Apps SDK. It SHALL declare `data:` as its only resource source for inline PNG previews, with no external connection, resource, or frame domains in resource-content CSP metadata. The UI SHALL register lifecycle handlers before connecting, respond to theme context, and present tool failures or cancellations. It SHALL check host tool-call capability before making tool calls.

#### Scenario: Host cannot proxy tool calls
- **WHEN** the host omits `serverTools` from its UI capabilities
- **THEN** the UI displays available result content and explains why interactive preview and save controls are unavailable

#### Scenario: Host loads the resource without network access
- **WHEN** the host reads and renders the studio resource with its declared CSP
- **THEN** the UI initializes using only embedded application assets and the host bridge

#### Scenario: Host derives image policy from declared resource sources
- **WHEN** the host enforces an image policy using the resource-content CSP metadata
- **THEN** the declared `data:` resource source permits the inline PNG preview to display
- **AND** the resource requests no external resource, connection, or frame origins

#### Scenario: Preview is canceled or fails
- **WHEN** preview execution reports cancellation or a tool error
- **THEN** the UI shows the outcome and does not enable saving that failed request

## ADDED Requirements

### Requirement: Independent explicit export dimensions

The studio UI SHALL offer preview-size, source-size, and custom-size export modes separately from bounded preview controls. Preview-size SHALL be the default and replay the accepted dimensions. Source-size SHALL use accepted normalized dimensions after cropping and before preview resize. Custom-size SHALL require two positive integer dimensions. The UI SHALL validate against the returned `export_limits`, while the service SHALL enforce the normal engine limits. A changed export size SHALL alter only `width` and `height` in the accepted processing options. The UI SHALL explain that changing resolution can change the dither pattern.

#### Scenario: User exports the original cropped extent
- **WHEN** a user selects source-size export after a successful bounded preview
- **THEN** save uses the accepted normalized and cropped source width and height
- **AND** palette, seed, crop, effects, mask, and every other accepted option remain unchanged

#### Scenario: User specifies larger custom dimensions
- **WHEN** a user chooses valid custom dimensions larger than the preview limits but within export limits
- **THEN** save uses those dimensions without enlarging the in-memory preview
- **AND** the UI discloses the resolution-dependent dither pattern

#### Scenario: Export dimensions are invalid
- **WHEN** custom dimensions are missing, zero, fractional, or exceed dimension or pixel limits
- **THEN** the UI prevents the save call and describes how to correct the dimensions

### Requirement: Byte-consistent guarded saves

Studio results SHALL include `source_sha256` and optional `mask_sha256` computed from the exact bounded encoded-file buffers decoded for the preview. Render requests SHALL accept optional `expected_source_sha256` and `expected_mask_sha256`, each nonempty value containing exactly 64 hexadecimal digits. Omitted or empty expectations SHALL leave the corresponding input unguarded. A mask expectation SHALL require an image-mask path. The service SHALL compare supplied expectations to the same buffers subsequently decoded or staged for processing, without rereading the path after checking. Requests without expectations SHALL retain their existing behavior. This guard SHALL apply wherever the shared render request is accepted.

#### Scenario: Source changes between preview and save
- **WHEN** a guarded render reads source bytes whose SHA-256 differs from `expected_source_sha256`
- **THEN** the request fails before publication with recoverable code `source_changed`
- **AND** the MCP tool error contains `structuredContent.error` with `code`, `message`, `path`, `expected_sha256`, and `actual_sha256`, plus a text fallback beginning with that code

#### Scenario: Image mask changes between preview and save
- **WHEN** a guarded render reads mask bytes whose SHA-256 differs from `expected_mask_sha256`
- **THEN** the request fails before publication with the same structured shape and code `mask_changed`

#### Scenario: Guarded input can no longer be read
- **WHEN** a guarded source or mask is missing, unreadable, nonregular, or exceeds the read budget
- **THEN** the service fails before publication with the corresponding `source_changed` or `mask_changed` error
- **AND** `actual_sha256` is empty because no complete readable buffer is available

#### Scenario: File changes after the processing buffer is loaded
- **WHEN** a source or mask path changes after the service has captured a buffer whose fingerprint matches the expectation
- **THEN** processing uses that captured buffer and cannot decode replacement bytes from another read

#### Scenario: User recovers from changed input
- **WHEN** studio save returns `source_changed` or `mask_changed`
- **THEN** the UI invalidates the accepted preview and requires a new successful preview before another save
- **AND** it does not automatically retry with replacement fingerprints

#### Scenario: Preview lacks valid guard metadata
- **WHEN** a preview response omits or malforms a required source or image-mask fingerprint
- **THEN** the UI does not enable saving from that response
