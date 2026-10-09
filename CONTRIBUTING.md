# Contributing

Use Go 1.27. The runtime does not require Node.js. OpenSpec, Quint, and the MCP Apps
bundle use the development versions in `package.json` and `pnpm-lock.yaml`.

1. Describe the behavior change and failure cases in an OpenSpec change.
   Include concrete scenarios. When you adopt the change, update the baseline
   specifications.
2. If the change affects publication, cancellation, selection, or palette
   contracts, update the corresponding Quint model. Explain what the model
   includes and abstracts.
3. Implement MCP and CLI behavior in the shared service. Keep image algorithms
   in the `engine` package. The engine must remain independent of filesystems
   and external processes.
4. Add tests that can detect a plausible implementation error. For algorithm
   changes, include a small reference fixture.
5. Run `make spec-install` to install the pinned development dependencies.
   Then run `make verify`, `make spec-check`, and `make fuzz`. If dependencies change,
   run `make vuln`. Record affected demos again. Update the documented limits.

Follow the [writing style](docs/writing-style.md) for documentation, comments,
CLI help, MCP descriptions, and website text.

## Add palettes

The `engine/palettes.json` file defines the palette catalog. The runtime embeds
this file in the Go binary. Each palette needs these fields:

- A stable lowercase ID.
- An ordered array of 2–256 unique six-digit sRGB hex colors.
- A category, description, and searchable tags.
- An origin: `original`, `reference`, or `approximation`.
- A source link for a published reference or historical approximation.

Keep the color order intentional. Palette cycling and nearest-color ties use
that order. If hardware or calibration affects historical display colors,
describe the palette as an approximation.

Review the swatches and a rendered image together. Each palette must have a
distinct unordered color set. Its description should explain the intended use.
Keep existing IDs and ordered colors stable. Preserve the original compatibility
fixture in `engine/testdata/legacy-palettes.json`.

When you expand the catalog, update its count assertions, specifications, and
documentation. To refresh generated files, run these commands from the source
directory:

1. Run `make palette-docs` to update the Markdown reference.
2. Run `go run ./scripts/palette-atlas` to generate the atlas, image treatments,
   and website catalog.
3. Run `./scripts/showcase-site.sh` to copy the assets to the standalone site.

The [showcase guide](docs/showcase.md) describes each generated asset.
The `make verify` target checks palette data, rendering, discovery, and generated
reference content.

## Update the MCP Apps studio

Read the [MCP Apps guide](docs/mcp-apps.md) before changing the studio or its bridge.
Keep image processing in the shared Go service. The app provides controls and
calls tools through the official `@modelcontextprotocol/ext-apps` SDK.

1. Install pinned development dependencies with `make spec-install`.
2. Edit the source files in `ui/`.
3. Run `pnpm ui:build` to regenerate the self-contained HTML bundle.
4. Run `pnpm ui:check` and `make ui-test`. The test target builds the Go binary.
5. Run `pnpm exec playwright install chromium`, then `make ui-browser`.
   Review the generated desktop, mobile, and dark-mode screenshots.

Keep the HTML, CSS, SDK, and scripts bundled locally. Preserve host fallback to
PNG and structured JSON. Previews must remain read-only and bounded. Save image
must use `dither_render` with the exact recipe from the last successful preview
and a new relative output path. Document host requirements without implying that
every MCP client supports the Apps extension.

## Preserve runtime contracts

Do not add network imports, URL sources, overwrite flags, unbounded image-sized
allocations, user-supplied executable paths, or stdout logging in MCP mode.
Changes to these contracts require an explicit design review and specification.

## Prepare a release

The development version is `0.1.0-dev`. The project does not yet have a published
release or a stable public Go API. Before a release:

1. Test the supported operating systems.
2. Review dependency notices.
3. Repeat vulnerability checks.
4. Check that the source archive builds from a clean checkout.

Run the local verification commands before submitting changes. The repository does not configure hosted CI.
