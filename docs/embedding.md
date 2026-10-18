# Embed the studio in your application

The [embedding example](../examples/embedding/server.mjs) runs the same studio resource as an MCP Apps host. It accepts your workspace and source explicitly and calls the real Go MCP server over stdio. Use it as a starting point for an existing desktop app or an application with a local backend.

The browser module uses the official [`AppBridge`](https://apps.extensions.modelcontextprotocol.io/api/classes/app-bridge.AppBridge.html). Browsers cannot launch a native stdio process or read arbitrary native paths; the Node backend or your desktop process must own the MCP connection.

## Run the example

From this repository, build the Go binary and install the pinned JavaScript development dependencies:

```sh
make build
pnpm install --frozen-lockfile --ignore-scripts
pnpm embedding:serve --root /absolute/path/to/images --input source.png
```

Open **http://localhost:8788**. `--root` must be an existing absolute directory, and `--input` is a file relative to that directory. The backend restricts the session to this one source. Save destinations are also relative to the workspace. Files already present are protected by the Go server's overwrite policy.

Optional flags are `--mask masks/subject.png`, `--binary /absolute/path/to/dither-mcp`, and `--port 8788`. If a mask is selected, preview and image-save calls use that mask. Saved recipes retain the mask requirement without its file path. Extraction reads only the selected source. The example starts with Atkinson and `oat-and-ink`. Omitted dimensions fit within 512 × 512 pixels without enlargement. Use the programmatic `initial` option for other settings. Supplying custom colors omits the default palette.

## Reuse the two modules

In the backend or desktop process:

```js
import { createStudioSession } from './examples/embedding/session.mjs';

const session = await createStudioSession({
  binary: '/absolute/path/to/dither-mcp',
  root: '/absolute/path/to/images',
  input: 'source.png',
  initial: { palette: 'oat-and-ink', options: { algorithm: 'atkinson' } },
});

// Deliver session.bootstrap to your trusted application UI.
// Forward only its validated studio calls, preserving cancellation:
const result = await session.callTool(params, { signal });
// Close the native connection when the view or application is closed:
await session.close();
```

Bundle the browser module with your app:

```js
import { mountDitherStudio } from './examples/embedding/studio-host.mjs';

const mounted = await mountDitherStudio({
  iframe: document.querySelector('#studio'),
  sandboxUrl: 'http://127.0.0.1:8788/sandbox',
  bootstrap,
  callTool: (params, { signal }) => yourBackend.callStudioTool(params, { signal }),
  hostContext: { theme: 'light', containerDimensions: { maxWidth: 1100 } },
});

mounted.setHostContext({ theme: 'dark' });
// Before removing the view:
await mounted.destroy();
```

The iframe must already be attached to the DOM. The sandbox proxy must run on a **different origin** from the host UI. In a desktop app, preserve that origin separation in your webview protocol/backend design; a plain HTML page opened with `file://` is insufficient. Bundle the SDK import with your normal browser build system. The example server uses esbuild from the repository's pinned dependencies.

`session.bootstrap` contains the original tool arguments, HTML resource, and full `CallToolResult`. Keep `content`, `structuredContent`, and `_meta` together: the app's catalogs and canonical request live in `_meta.dither`. `mountDitherStudio` forwards them unchanged through AppBridge. It also forwards `expected_source_sha256` and `expected_mask_sha256` unmodified on render. Those guards make source changes invalidate Save, including changed or removed mask files. Preview dimensions and export dimensions are independent; the app chooses the final render request.

## Local boundaries

The supplied HTTP example listens only on `127.0.0.1`. Its host UI is addressed as `localhost`; its sandbox proxy is addressed as `127.0.0.1`, creating separate origins. It checks exact Host and Origin values, rejects cross-site browser requests, and requires a fresh per-process session token for data and tool calls. It exposes only `dither_studio`, `dither_render`, `dither_palette_extract`, `dither_recipe_save`, `dither_random_style`, `dither_favorites_list`, `dither_favorite_load`, `dither_favorite_save`, and the selected studio resource. Extraction accepts only the selected `input` and a color `count` from 2 through 256. It has no arbitrary server, tool, filesystem, upload, or static-directory proxy.

Recipe saving accepts only `output` and `recipe`. Its destination must be a new
relative `.json` path, and its recipe must match the session's latest successful
preview. Starting a preview, extraction, favorite load, or random style revokes recipe permission until a
successful preview completes. Image saving keeps the accepted recipe available.
The host grants no arbitrary recipe-file load or arbitrary-tool access. The Go server validates
recipes and enforces rooted paths and no-overwrite publication.

Random generation accepts only an optional safe integer `seed`. A new random
request may cancel and supersede pending read-only work, including a random
preview. It cannot interrupt image, recipe, or favorite writes. The host tracks operation
identity so an old completion cannot clear a newer request or authorize an old
recipe. The UI additionally checks settings revisions before adopting a result.

History restoration uses the existing `dither_studio` call with exact stored
settings. It requires idle state, rereads the selected files, and renews recipe
authority only on success. No tool or permission is added. The app holds a
20-look in-memory history; the host retains no image history. Undo does not undo
published files. Random style can supersede a pending restoration, with the
same current-operation guards.

Favorites list accepts an empty object and preserves the accepted preview.
Favorite load accepts only a validated ID from the fixed workspace library;
it revokes recipe authority until a successful current preview completes.
Favorite save accepts a name and only the exact latest accepted recipe, with
no output path. It is an exclusive write. Closing and reopening the session
leaves the workspace library available. Source/mask bindings still apply to
previewing a loaded favorite. Metadata from unrelated workspace files remains
inaccessible through these tools.

The inner app has an opaque sandbox origin and a CSP that permits its embedded JavaScript/CSS and PNG data images. Network connections, nested frames, forms, popups, and native file URLs are blocked. The sandbox proxy validates both adjacent windows and the host's origin. No CORS access is provided. The source image reaches the UI as a tool-result PNG; the browser never receives a native file URL.

If you adapt the example for a multi-user server, implement user/session authorization and isolated workspaces in your own backend before exposing it. This example is a local integration with one selected source, not a hosted service.

## Verify it

```sh
make embedding-test
make embedding-browser
```

The stdio tests negotiate Apps with the real binary, preserve metadata, save a byte-identical PNG, save/load/replay accepted recipes and named favorites across sessions, verify guarded render errors, and reject different sources and path escapes. The browser check exercises the official AppBridge handshake and teardown, an opaque sandbox, real image and recipe saves, recipe authority checks, rapid random clicks and history restores with deliberately delayed replies, exact restored PNG/recipe bytes, theme updates, mobile layout, Host/Origin/token restrictions, and absence of console errors, failed requests, or external requests. It writes local screenshots and a report under `work/embedding-browser-check/`. The [recorded browser results](embedding-browser-verification.json) list the checks. These checks verify this integration example; [host compatibility](hosts.md) distinguishes independent client tests from documented support.
