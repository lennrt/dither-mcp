## ADDED Requirements

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
