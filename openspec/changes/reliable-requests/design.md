# Reliable request boundaries

## Encoding cancellation

Expose `EncodeContext` and `PalettizeContext` alongside existing `Encode` and `Palettize` wrappers. Reject a nil context. Check cancellation before work, between handwritten image rows, and before successful return. Pass the context into GIF median-cut extraction. Preserve palette order, alpha handling, matting, and successful encoded bytes.

Use a context-aware writer for standard PNG, JPEG, and GIF encoding. Standard codecs can still perform work between writes. A blocked caller-provided writer cannot be interrupted by a context check. The application uses bounded in-memory buffers, so this limitation does not introduce filesystem writes before publication. Motion and print encoding also retain the request context.

The existing transaction still publishes only after encoding and its final cancellation check. Tests cancel deterministically within scans and writes. The publication model adds partial encode work and checks that its abstract canceled state cannot continue encoding. It does not model codec internals or promise a wall-clock cancellation bound.

## CLI flag admission

Use the existing parser's explicitly visited flags to check a compact command-specific allowlist. Reject unsupported flags before opening the workspace. A supplied zero or false value still counts as an explicit unsupported flag. Keep startup flags and accepted command arguments intact. Generic `call` receives tool arguments through JSON and does not silently consume image flags.

This is an intentional error-behavior change for previously ignored input. It does not alter shared-service JSON semantics or introduce a second CLI processing pipeline. Advanced options retain their documented merge behavior.

## Mixed studio edits

When a control changes after valid advanced JSON edits, merge that latest field into the parsed options and adopt the complete object. Preserve hidden crop, mask, and effect settings. If the JSON is malformed or not an object, keep its text and the draft unchanged, restore the control, and explain how to correct the JSON. Do not silently replace either representation. Direct Apply also rejects explicit null instead of coercing it to empty options; omission by the host still uses defaults.

The interaction model's revision abstracts both JSON and control edits. Its existing stale-result and save guards apply unchanged. Concrete helper and browser request-audit tests establish field precedence and preservation, which the finite model does not parse.
