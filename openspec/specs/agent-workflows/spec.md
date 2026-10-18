# Agent workflows specification

## Purpose

Make creative iteration inspectable, repeatable, and scriptable through MCP and the CLI.

## Requirements

### Requirement: Typed stdio MCP

The application SHALL implement MCP initialization and stdio tool discovery with explicit input schemas and useful structured success/error results. Stdout in server mode SHALL contain only protocol messages. Diagnostics SHALL use stderr. Tool annotations SHALL accurately distinguish reads from file writes.

#### Scenario: Agent connects
- **WHEN** a client initializes the stdio server and lists tools
- **THEN** it receives discoverable image-workflow tools with schemas

#### Scenario: Invalid tool arguments
- **WHEN** a client calls a tool with invalid fields or types
- **THEN** the tool returns an actionable error without terminating the server

### Requirement: Inspection and catalog context

The suite SHALL expose image metadata and catalogs without modifying the source. It SHALL supply agent-readable usage context through documented resources or prompts, including local-root rules and recommended discover-inspect-compare-render workflows.

#### Scenario: Plan a render
- **WHEN** an agent inspects a source and reads algorithm/palette context
- **THEN** it can choose valid settings and destination dimensions before rendering

#### Scenario: Discover supported media
- **WHEN** an agent reads the capability catalog before choosing a media operation
- **THEN** the catalog identifies still formats, animation outputs, video containers, video outputs, enablement, dependencies, and limits

### Requirement: Bounded image previews

The suite SHALL provide a read-only PNG preview with a default requested width of 512 pixels. The preview SHALL not exceed 1,024 pixels per axis or 2 MiB of encoded content. It SHALL deliver the preview as MCP image content without creating a file.

#### Scenario: Inspect a rendered candidate
- **WHEN** an agent requests a preview of a supported local image
- **THEN** it receives a bounded PNG image and dimensions without a new filesystem artifact

### Requirement: Comparable candidates

The suite SHALL render a bounded set of candidate configurations from one input and generate a comparison artifact. Candidate ordering and labels SHALL be deterministic and traceable to the supplied configurations.

#### Scenario: Compare three algorithms
- **WHEN** an agent submits three valid algorithm choices
- **THEN** it receives a comparison with one corresponding candidate per choice in request order

### Requirement: Versioned recipes

The suite SHALL accept versioned JSON recipes that represent reproducible processing options. It SHALL reject unsupported recipe versions. Recipes SHALL not grant access outside the configured root or bypass validation and limits.

#### Scenario: Replay a saved look
- **WHEN** an agent reuses a recipe with the same source and seed
- **THEN** the same normalized processing options produce the same pixels

#### Scenario: Unsupported recipe version
- **WHEN** a recipe declares an unsupported version
- **THEN** the suite rejects it with a version error

### Requirement: Explicit batch outcomes

The suite SHALL process bounded batches with per-item results. It SHALL describe whether processing stops or continues after an item fails. It SHALL never claim transaction-wide rollback after publishing earlier items.

#### Scenario: One item fails
- **WHEN** a batch contains a valid render and an invalid input
- **THEN** the result identifies artifacts the suite published and the item that failed according to the documented failure policy

### Requirement: CLI shares application semantics

The CLI SHALL expose the principal discovery, inspection, rendering, and workflow operations through the same service used by MCP. It SHALL provide useful help, a version command, nonzero failure exit status, and machine-readable output where documented.

#### Scenario: Reproduce a tool call locally
- **WHEN** an agent submits equivalent validated configuration through CLI and MCP
- **THEN** both paths apply the same engine options and filesystem policies

### Requirement: Negotiated MCP Apps studio

The suite SHALL expose a public read-only `dither_studio` tool and a bundled `ui://dither/studio.html` resource. The resource SHALL return a complete HTML document with MIME type `text/html;profile=mcp-app`. The server SHALL advertise the `io.modelcontextprotocol/ui` extension and associate the tool with its UI only for clients that advertise support for that MIME type. The tool SHALL remain usable with typed arguments, structured JSON, and native PNG image content in other MCP clients.

#### Scenario: Apps-capable host discovers the studio
- **WHEN** a client advertises the MCP Apps extension and the supported HTML MIME type
- **THEN** tool discovery links `dither_studio` to the declared UI resource through `_meta.ui.resourceUri`

#### Scenario: Client uses the tool without Apps support
- **WHEN** a client without the supported MCP Apps MIME type calls `dither_studio`
- **THEN** the tool returns its processed PNG and recipe through ordinary MCP content and structured results
- **AND** tool discovery omits the UI association

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

### Requirement: Explicit CLI flag admission

The CLI SHALL reject explicitly supplied flags that its selected command does not consume. It SHALL identify the unsupported flag and command before opening the workspace or creating an output. Supplying zero or false SHALL NOT make an unsupported flag valid. Supported command flags, common startup flags, help, and generic tool-call JSON SHALL retain their documented behavior.

#### Scenario: Inspection receives render settings
- **WHEN** a caller supplies `inspect --width 100` or an output path flag
- **THEN** the CLI returns an actionable error instead of silently ignoring those flags

#### Scenario: Zero-valued irrelevant flag
- **WHEN** a caller supplies `render --frames 0`
- **THEN** the CLI rejects the unsupported frame flag

#### Scenario: Valid palette comparison
- **WHEN** a caller supplies the supported palette comparison flags
- **THEN** the CLI forwards the intended comparison through the shared service

### Requirement: Preserve mixed studio option edits

When an engine control changes after pending valid advanced JSON, the studio SHALL merge the latest control field into those parsed options. It SHALL preserve all other options. If the JSON is malformed or not an object, the studio SHALL retain the raw text and draft, restore the control, and report the correction needed. Neither case SHALL call a tool without an explicit preview request. Direct Apply SHALL reject malformed or nonobject advanced JSON, including explicit `null`, without replacing the draft or calling a tool. Omitted host options SHALL continue to use defaults.

#### Scenario: Adjust tone after advanced settings
- **WHEN** a user enters valid JSON with an effect and then changes brightness
- **THEN** the next preview receives both the effect and the latest brightness value

#### Scenario: Control changes while JSON is invalid
- **WHEN** a user changes a control while advanced JSON is malformed or not an object
- **THEN** the studio preserves the JSON text and previous draft and explains the error
- **AND** it makes no tool call

#### Scenario: Apply explicit null options
- **WHEN** a user enters `null` in advanced JSON and selects Apply preview
- **THEN** the studio retains the text and existing draft, reports that options must be an object, and makes no tool call
- **AND** correcting the JSON allows preview again
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
