# Studio export and input snapshot design

## Preview and export dimensions

`dither_studio` still returns bounded PNG dimensions in `width` and `height`, and stores those exact dimensions in `recipe.options`. Add `source_width` and `source_height` after orientation normalization and the validated crop, before resize. These are the dimensions used by the source-size export choice. Add `export_limits` with `max_width`, `max_height`, and `max_pixels`, derived from the actual engine limits.

The UI defaults to preview-size export. Source-size export uses the accepted source dimensions. Custom export requires two positive integer dimensions; zero does not mean automatic inference. Validate the chosen dimensions against the advertised limits before dispatch, while retaining backend validation as authoritative. Export controls are independent of preview controls and need no new preview merely to choose a final size.

Build save arguments from the accepted request. Preserve source, palette/colors, mask path, seed, crop, effects, and all other accepted options. Only the accepted width and height may change when the user explicitly selects another export mode. Explain that changing resolution can alter the dither pattern. The preview remains bounded independently of final output dimensions.

## Fingerprints bind processing to loaded bytes

Preview metadata adds `source_sha256` and, when using an image mask, `mask_sha256`. Each is a lowercase SHA-256 digest of the exact bounded encoded-file buffer used for decoding, including metadata bytes. A source is normalized and cropped after decoding; its digest identifies the encoded bytes before those transformations. A metadata-only or encoding-only edit therefore requires a new preview even when decoded pixels would be equal.

Render requests add optional `expected_source_sha256` and `expected_mask_sha256`. A nonempty supplied digest must contain exactly 64 hexadecimal digits, compared case-insensitively. An omitted or empty expectation leaves that input unguarded. A mask expectation requires an image mask path. Compute and check the digest on the same read buffer subsequently decoded or staged for processing. Do not perform a check-read followed by another source read. The common request/loader boundary must retain this behavior for workflows that accept render requests.

The fingerprint is a consistency guard, not authentication, an immutable filesystem snapshot, or an authorization token. If a file changes after its bytes were read, this call may safely complete from those captured bytes. Source and mask are separate reads; no multi-file atomic snapshot is promised. SHA-256 collision resistance is assumed and not proved by the model.

## Recoverable mismatch

On mismatch return a tool error with `structuredContent.error` containing `code`, `message`, `path`, `expected_sha256`, and `actual_sha256`. Codes are `source_changed` and `mask_changed`; text fallback starts with the same code. A guarded read failure, including a missing, unreadable, nonregular, or oversized input, returns the same corresponding code with empty `actual_sha256`, because a complete buffer is unavailable. No output is published from a mismatching or unreadable guarded request. The UI invalidates the accepted preview and explains that the user must preview again. It must not retry automatically or substitute newly read fingerprints while retaining the old preview.

On successful preview acceptance, the UI stores both fingerprints together with the recipe. Save sends `expected_source_sha256` and, for an image mask, `expected_mask_sha256`. Missing or malformed guard metadata cannot authorize a studio save. Ordinary render clients may omit guards for backward compatibility.

## Host and embedding boundary

Host guidance distinguishes local stdio access, core MCP tools, native image content, MCP Apps extension negotiation, the supported HTML MIME type, and app-to-host `serverTools` capability. A documented product name alone is not proof of each feature. Record documentation sources and dates separately from locally executed harness evidence. Remote connector URLs do not make a local stdio binary remotely reachable.

The embedding example uses the pinned official AppBridge and a sandboxed app resource. It launches the real server over stdio and adapts it through a loopback-only local development host. Document installation, workspace-root configuration, launch/stop commands, and how to reuse the bridge. This example is not a production remote transport or a substitute for authentication and deployment design.

## Verification boundary

Keep request identity, dirty controls, cancellation, and explicit save authorization in `mcp_apps.qnt`. Add `studio_export.qnt` for three abstract source/mask versions, one accepted recipe token, fixed preview/source geometry, bounded custom dimensions, independent disk mutations, captured buffers, validation, decode, and publication. The model checks that selected size is authorized and bounded, non-dimensional settings are preserved, and every published decode uses the accepted bytes. It also exercises a disk edit after loading to distinguish a captured-buffer guarantee from a check-read/reread race.

Finite run tests, seeded traces, completion witnesses, and negative mutations provide model evidence only. The model does not implement hashing, image codecs, geometry arithmetic, JSON, browser controls, filesystem races, host installation, or real host rendering. Backend, controller, browser, and embedding checks must test those implementation boundaries independently; no refinement proof or production-host certification follows from the model.

## Inline image CSP compatibility

The resource explicitly declares `resourceDomains: ["data:"]` for in-memory PNG previews. Connection, frame, and base-URI lists stay empty. This permits local encoded image data while requesting no external origins. MCP Inspector 2.10.1 requires this declaration because an empty resource list produces `img-src 'none'`. The integration test must exercise the published host with its normal sandbox policy.
