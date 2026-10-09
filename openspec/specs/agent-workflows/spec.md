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

The studio SHALL process local input, palette, engine options, and image masks under the shared normalization, path, and resource policies. It SHALL accept no output path and create no file. Default dimensions SHALL fit 512 by 512 pixels. Results SHALL fit 1,024 pixels per axis and 2 MiB of PNG. Results SHALL include source path, dimensions, MIME type, the versioned recipe used, and any image-mask path needed for replay.

#### Scenario: User previews processing settings
- **WHEN** the studio receives a valid input and supported processing settings
- **THEN** it returns the processed image and exact replay settings without publishing an artifact

#### Scenario: Processing exceeds preview limits
- **WHEN** explicit or derived dimensions exceed a preview bound
- **THEN** the tool returns an actionable error and creates no output

### Requirement: Explicit current-preview saves

The UI SHALL preview edits only on explicit request. It SHALL accept a result only while its request identity and control revision match. Save SHALL require a current successful preview, a destination, host tool-call support, and no pending operation. Save SHALL call `dither_render` only on user action with the accepted source, recipe settings, and mask. The UI SHALL preserve unexposed settings and reject values it cannot represent exactly.

#### Scenario: Controls change after a successful preview
- **WHEN** a user changes processing controls after receiving a preview
- **THEN** saving remains disabled until a successful preview accepts those settings

#### Scenario: An old preview finishes late
- **WHEN** a response arrives for a canceled, replaced, or edited preview request
- **THEN** it cannot replace the current preview or enable saving

#### Scenario: User saves the current preview
- **WHEN** a user enters a destination and requests save for a current successful preview
- **THEN** the UI submits the accepted preview's source, exact settings, and mask to `dither_render` with that destination
- **AND** existing rooted, bounded, no-clobber publication policies apply

### Requirement: Self-contained host-aware UI

The studio resource SHALL embed its script, styles, and pinned official MCP Apps SDK. It SHALL declare no external connection, resource, or frame domains in resource-content CSP metadata. The UI SHALL register lifecycle handlers before connecting, respond to theme context, and present tool failures or cancellations. It SHALL check host tool-call capability before making tool calls.

#### Scenario: Host cannot proxy tool calls
- **WHEN** the host omits `serverTools` from its UI capabilities
- **THEN** the UI displays available result content and explains why interactive preview and save controls are unavailable

#### Scenario: Host loads the resource without network access
- **WHEN** the host reads and renders the studio resource with its declared CSP
- **THEN** the UI initializes using only embedded application assets and the host bridge

#### Scenario: Preview is canceled or fails
- **WHEN** preview execution reports cancellation or a tool error
- **THEN** the UI shows the outcome and does not enable saving that failed request
