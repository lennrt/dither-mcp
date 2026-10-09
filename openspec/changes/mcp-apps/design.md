# MCP Apps studio design

## Protocol boundary

Implement the stable MCP Apps specification dated 2026-01-26. Advertise the `io.modelcontextprotocol/ui` extension with `text/html;profile=mcp-app` support. Check each client's extension MIME types before exposing the tool's UI association.

Register `ui://dither/studio.html` as a public resource. Return a complete HTML document with the exact MCP Apps MIME type. Put UI security metadata on the returned resource content. Declare empty external connection, resource, and frame domain lists. The host supplies and enforces the sandbox.

For capable clients, `dither_studio` declares `_meta.ui.resourceUri`. The tool remains visible to both the model and app. Existing `dither_render` remains available through normal app-to-host tool calls. There are no app-only server tools.

All clients receive meaningful content, including serialized structured JSON and native PNG image content. UI metadata is additional presentation data. It does not carry the only copy of the result or change tool authorization.

## Preview boundary

`dither_studio` accepts `input`, optional `palette` or `colors`, optional engine `options`, and optional `mask_input`. It accepts no output path. It uses the shared rooted loader, normalization, configuration validation, mask handling, processing engine, and resource admission checks.

When both dimensions are omitted or zero, fit the source within 512 by 512 pixels. The processed result must fit 1,024 pixels on each axis and 2 MiB of encoded PNG. Reject oversized explicit dimensions or derived geometry. Keep encoded bytes in memory.

Return `path`, `width`, `height`, `mime_type`, and a versioned `recipe`. Return `mask_input` when the processed request uses an image mask. The recipe records the dimensions and exact settings used for this preview. The path identifies the source, not a created artifact.

For capable hosts, result `_meta.dither` includes the canonical request and algorithm/palette catalog data. The UI uses these values to present supported settings. The service remains the authority for validating every subsequent request.

## UI lifecycle

Use the official `@modelcontextprotocol/ext-apps` package, pinned to an exact version. Bundle the SDK, application script, and styles into the embedded HTML. Register lifecycle handlers before connecting to the host.

Read initial arguments and the tool result from host notifications. Apply host theme context and subsequent context changes. Show tool errors and cancellation states. Check `serverTools` before enabling interactions that require host tool calls.

Changing processing controls makes the displayed preview stale. An explicit preview action calls `dither_studio` with the current controls. Track the request identity and control revision. A response may become current only when both still match. Cancellation or a replaced request invalidates its late response.

Keep save disabled while work is pending, while controls differ from the current successful preview, or when no valid preview exists. Require a destination entered by the user. A save action calls `dither_render` with the last accepted preview's source, palette/colors, recipe options, image-mask path, and the entered output path. It does not use a fresh reconstruction from edited controls.

The UI must preserve all returned processing settings, including options it does not expose as controls. JavaScript must reject settings it cannot represent exactly when they affect a tool call, including unsafe integer seeds. Ordinary MCP clients retain the existing full signed 64-bit seed contract.

Save reprocesses the source using the accepted recipe. The output matches the preview's dimensions and settings. Source or mask changes between calls can change the pixels. The UI does not claim a content-addressed source snapshot.

## Verification boundary

The finite Quint model represents settings with revision numbers and tool responses with request identities. It checks that previews cannot publish, accepted results correspond to their requests, and save calls use the current accepted preview after an explicit action. It also checks dirty controls, missing tool capability, cancellation, and stale responses.

Seeded simulation and model run tests do not establish browser behavior or Go refinement. Go tests must exercise the real preview path, schemas, negotiated metadata, image results, resource contents, bounds, and publication behavior. UI tests must exercise the real controller and browser lifecycle. A local bridge harness provides repeatable evidence. It does not establish compatibility with every MCP Apps host.
