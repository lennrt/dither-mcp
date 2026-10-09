# Interactive studio

`dither_studio` processes a local image and returns a PNG preview with its exact recipe. An MCP Apps host can display the interactive studio. Other MCP clients receive the same preview and recipe through ordinary tool results.

The studio follows the [stable MCP Apps specification, dated 2026-01-26](https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/2026-01-26/apps.mdx). It bundles the official [`@modelcontextprotocol/ext-apps`](https://apps.extensions.modelcontextprotocol.io/) SDK at version 2.0.3.

## Open and use the studio

Start the server with an existing image workspace:

```sh
dither-mcp mcp --root /absolute/path/to/images
```

Ask the agent to open an image with `dither_studio`, or call the tool directly:

```json
{
  "input": "photo.png",
  "palette": "gameboy",
  "options": {
    "algorithm": "atkinson",
    "width": 480,
    "seed": 42
  }
}
```

Choose an algorithm, search the palette library, or enter custom hex colors. Adjust tone, dimensions, and texture. Advanced JSON exposes the complete engine options, including crop, masks, and effects. The source and image-mask paths come from the tool call.

Select **Apply preview** to process changed settings. Editing controls marks the displayed image as stale. Save stays disabled until those settings have a successful preview. Pending, canceled, and late results cannot authorize a save of unpreviewed settings.

Enter a new workspace-relative destination, then select **Save image**. Saving calls the existing `dither_render` tool. Choose **Preview**, **Source**, or **Custom dimensions** for the export. The filename extension selects the output format. The destination must not exist.

## Preview and save behavior

| Property | Behavior |
|---|---|
| Preview input | `input`, optional `palette` or `colors`, engine `options`, and optional `mask_input` |
| Default size | Fit within 512 × 512 pixels without enlargement when both dimensions are zero or omitted |
| Size limit | 1,024 pixels on each axis, including a dimension derived from aspect ratio |
| Encoded limit | 2 MiB of PNG bytes |
| Preview result | Source `path`, preview `width` and `height`, `mime_type`, versioned `recipe`, source and mask SHA-256 fingerprints, cropped source dimensions, and runtime export limits |
| Filesystem effect | Preview reads files and keeps its PNG in memory |
| Save input | Accepted preview source, recipe settings, mask path, expected fingerprints, chosen dimensions, and destination |
| Preview-size export | The accepted preview dimensions; the default |
| Source-size export | Upright source dimensions after cropping and before resizing |
| Custom-size export | Explicit positive integers for both width and height |
| Export limits | 16,384 pixels per axis and 16,777,216 pixels total; supplied by the server |
| Studio seed | Integer from −9,007,199,254,740,991 through 9,007,199,254,740,991, the exact JavaScript range |

The studio uses the shared image loader. EXIF orientation and supported input color normalization run before crop, resize, and dithering. Image masks use the same normalization policy. Preview failures create no artifact.

Save reprocesses the source with the accepted recipe. Preview-size PNG exports reproduce the preview pixels. Source and custom sizes change only width and height. Changing resolution can change the dither pattern; JPEG and other output formats can also change appearance.

Every studio save sends `expected_source_sha256` and, for image masks, `expected_mask_sha256` to `dither_render`. The service checks the exact byte buffers that it decodes. A mismatch returns `source_changed` or `mask_changed` and creates no artifact. The app keeps the displayed preview, disables Save, and asks you to apply a fresh preview. Files changed after capture cannot alter the bytes already being rendered. These checks do not lock the original files or retain a server-side preview session.

Use `dither_render` directly for the full signed 64-bit seed range or saved recipe files. Its usual limits apply. See the [MCP reference](mcp.md) for all engine options and the [security model](security.md) for rooted paths and atomic publication.

See [host setup and compatibility](hosts.md) for tested clients and installation instructions. The [embedding guide](embedding.md) provides a reusable local AppBridge host. The studio edits still images, including the first frame of a GIF. Use the MCP tools or CLI for animation and video.

## Host and resource contract

The host advertises the `io.modelcontextprotocol/ui` extension with the exact MIME type `text/html;profile=mcp-app`. The server advertises the extension and checks the client's MIME types. For supporting clients, `dither_studio` declares:

```json
{
  "_meta": {
    "ui": {
      "resourceUri": "ui://dither/studio.html",
      "visibility": ["model", "app"]
    }
  }
}
```

The public resource returns a complete HTML document through `resources/read`. Each returned content item carries `_meta.ui` with `resourceDomains: ["data:"]` for inline PNG data and empty connection, frame, and base-URI lists. No external resource origins are allowed. The explicit data scheme also supports hosts that require it in resource metadata, including MCP Inspector 2.10.1. It requests no browser permissions. The host controls sandboxing and CSP enforcement.

Scripts, styles, and the SDK are embedded in the Go binary. The running studio uses the host bridge and in-memory PNG data. It needs no CDN, HTTP listener, or external network access.

The UI registers lifecycle handlers before connecting. It handles input, result, cancellation, and theme changes. Hosts must advertise `serverTools` to enable interactive preview and save. Without that capability, the UI can show its supplied result and explains the unavailable controls.

For supporting clients, result `_meta.dither` supplies the canonical request and the algorithm/palette catalogs. The normal result still contains structured metadata, serialized JSON text, and native PNG image content. The server omits UI linkage and presentation metadata when the client does not advertise the supported MIME type.

The studio calls the same public tools available to the agent. Host consent and tool policies still apply to each call. No private save endpoint bypasses the existing service.

## Build and check

Released binaries contain the built UI. Contributors need the declared Node.js and pnpm versions to rebuild it:

```sh
pnpm install --frozen-lockfile --ignore-scripts
pnpm ui:build
pnpm ui:check
make ui-test          # build the Go binary and run UI state/bridge tests
pnpm spec:check
go test ./...
```

`ui:build` bundles the pinned SDK and application assets. `ui:check` verifies that the checked-in embedded document matches its sources. `make ui-test` builds the Go binary, exercises the concrete interaction state and official SDK bridge, and checks preview/save behavior through the real stdio server. After `make build`, `pnpm ui:harness` starts a loopback-only browser test host with an isolated copy of the original sample artwork. Open the localhost URL that it prints. Its outputs stay in an ignored `work/apps-host-*` directory. This development harness is not part of the production server.

The [OpenSpec change](../openspec/changes/mcp-apps/proposal.md) records the contract. The [Quint interaction model](../spec/mcp_apps.qnt) checks abstract preview, stale-result, and save sequencing. The [specification guide](../spec/README.md#mcp-apps-interaction-model) separates model evidence from implementation tests. Neither finite simulation nor a local browser harness proves compatibility with every MCP Apps host.

## Browser verification and screenshots

The optional browser check starts its own loopback host and Go stdio server with
an isolated copy of the original artwork. Install Chromium once, then run:

```sh
pnpm exec playwright install chromium
make ui-browser
```

The check uses the actual embedded HTML with external connections and native
form submission blocked. It covers initial rendering, palette search, changed
previews, exact PNG saving, overwrite rejection, custom-color validation,
keyboard saving, source-size and custom exports, changed-source rejection and recovery, cancellation, dark mode, narrow layouts, and hosts without
`serverTools`. Screenshots and a JSON report go to `work/browser-check/`.

To include the separate showcase, serve its directory locally and set
`DITHER_SITE_TEST_URL` to that loopback URL:

```sh
DITHER_SITE_TEST_URL=http://localhost:8080 make ui-browser
```

The browser check verifies navigation to the studio section, visible image
loading, and desktop/mobile overflow. It fails on browser script errors,
unexpected request failures, or external requests. On October 8, 2026, it passed
with Chromium 151.0.7922.34 and Playwright 1.62.1. The [browser report](mcp-apps-browser-verification.json) records the checks and published screenshot hashes. These results cover the local SDK test host. The separate [host guide](hosts.md) records the independent Inspector check and the limits of each compatibility claim.
