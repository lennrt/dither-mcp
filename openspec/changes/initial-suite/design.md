# Design

## Context

A dithering agent discovers a catalog, inspects an input, and selects a palette. It then renders candidates, compares them, and saves a chosen recipe. The agent repeats this workflow as needed. Every step needs a reproducible contract. The filesystem stores durable artifacts. The workflow needs no database, daemon-side job queue, or cloud account.

## Goals and non-goals

The design has these goals:

- Provide a reusable Go image engine and discoverable typed tools.
- Make local processing deterministic and bounded.
- Support useful still, motion, and print workflows.
- Present an attractive, accurate showcase that contributors can replay.

The design excludes these capabilities:

- Reproduce every visual filter in every linked project.
- Capture webcams or route live audio.
- Use machine learning for background removal or object tracking.
- Render in a browser or store hosted state.
- Claim identical pixels with another implementation.

## Decisions

1. **Separate mechanism from authority.** `engine` accepts images and values. `internal/app` controls local path authorization, codecs, and operations. `internal/mcpserver` and `cmd/dither-mcp` adapt requests to the service. This makes network-free image processing reviewable and allows the CLI and MCP to share validation.
2. **Use declarative settings.** A JSON recipe carries a stable version and explicit options. The service expands named palettes deterministically and validates custom palettes. Unknown fields must fail where strict JSON decoding applies. Invalid values always fail instead of silently degrading.
3. **Constrain paths and resource use before work.** Require relative paths. Resolve them through Go `os.Root`. Reject traversal and symlink escapes from the configured root. Check compressed bytes and image headers before decoding. Limit destination pixels, animation frames, comparisons, batches, and duration. Go `os.Root` supplies a race-resistant traversal boundary. The service accepts only regular file inputs. Snapshot guarantees do not cover concurrent changes to their contents.
4. **Publish complete artifacts.** Encode to a temporary file in the same directory. Close it successfully. Publish it atomically. Publication always refuses an existing destination. There is no overwrite switch. Use a fresh output path for each iteration. An atomic hard link publishes the completed temporary file without a check-then-rename race. A batch is a sequence of independent artifact transactions, not an all-or-nothing multi-file transaction.
5. **Be deterministic by default.** A fixed seed, palette order, algorithm ID, options, decoded input, and engine version determine output pixels. Random effects use a local seeded generator. Encoders and FFmpeg may vary by version. Pixel determinism therefore differs from byte determinism across versions.
6. **Make motion finite.** The service composites GIF input in source order and honors disposal. Generated motion uses an explicit finite frame count and deterministic phase function. Video is available only when explicitly enabled and configured. It uses a local FFmpeg subprocess with bounded arguments. It never uses a shell or a remote URL.
7. **State the formal boundary.** Quint models publication and quantization safety at small finite bounds. They do not prove the Go implementation, codec correctness, floating-point accuracy, aesthetic quality, symlink race resistance, or unbounded liveness. The refinement requires Go tests and review.

## Risks and tradeoffs

- Many reference effects are approximations of print processes. Algorithm descriptions must explain their construction. They must not call hash noise optimized blue-noise.
- Error diffusion is scan-order-dependent. Serpentine traversal intentionally changes results. A seed makes results repeatable, not invariant under traversal changes.
- Post-processing and region masks can leave pixels outside the selected palette. Palette-membership guarantees apply to unmasked quantization without post-effects, and export formats can impose further restrictions.
- GIF palette limits and integer centisecond delays constrain fidelity. Video re-encoding is lossy unless its selected codec is lossless. Do not promise audio preservation if the output omits audio.
- MCP runs locally. The connected client receives returned paths, metadata, text, and optional preview images. Its model or transport may be remote.

## Validation plan

Run these checks:

1. Strict OpenSpec validation.
2. Quint typechecking, unit runs, and seeded safety simulation.
3. Go unit and integration tests with the race detector.
4. CLI smoke tests and a real MCP stdio transcript.
5. Adversarial input and path tests.
6. A GIF round trip and an optional video round trip.
7. Generated asset checks.
 Record commands and outcomes. Do not infer implementation verification from a successful model simulation.
