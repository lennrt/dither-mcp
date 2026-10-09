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

Optional flags are `--mask masks/subject.png`, `--binary /absolute/path/to/dither-mcp`, and `--port 8788`. If a mask is selected, the session restricts all calls to that mask too. The example uses Atkinson, the `oat-and-ink` palette, and a 512px preview initially; use the programmatic `initial` option for other initial settings.

## Reuse the two modules

In the backend or desktop process:

```js
import { createStudioSession } from './examples/embedding/session.mjs';

const session = await createStudioSession({
  binary: '/absolute/path/to/dither-mcp',
  root: '/absolute/path/to/images',
  input: 'source.png',
  initial: { palette: 'oat-and-ink', options: { algorithm: 'atkinson', width: 512 } },
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

The supplied HTTP example listens only on `127.0.0.1`. Its host UI is addressed as `localhost`; its sandbox proxy is addressed as `127.0.0.1`, creating separate origins. It checks exact Host and Origin values, rejects cross-site browser requests, and requires a fresh per-process session token for data and tool calls. It exposes only `dither_studio`, `dither_render`, and the selected studio resource. It has no arbitrary server, tool, filesystem, upload, or static-directory proxy.

The inner app has an opaque sandbox origin and a CSP that permits its embedded JavaScript/CSS and PNG data images. Network connections, nested frames, forms, popups, and native file URLs are blocked. The sandbox proxy validates both adjacent windows and the host's origin. No CORS access is provided. The source image reaches the UI as a tool-result PNG; the browser never receives a native file URL.

If you adapt the example for a multi-user server, implement user/session authorization and isolated workspaces in your own backend before exposing it. This example is a local integration with one selected source, not a hosted service.

## Verify it

```sh
make embedding-test
make embedding-browser
```

The stdio tests negotiate Apps with the real binary, preserve metadata, save a byte-identical PNG, verify guarded render errors, and reject different sources and path escapes. The browser check exercises the official AppBridge handshake and teardown, an opaque sandbox, a real save, theme updates, mobile layout, Host/Origin/token restrictions, and absence of console errors, failed requests, or external requests. It writes local screenshots and a report under `work/embedding-browser-check/`. The [recorded browser results](embedding-browser-verification.json) list the checks. These checks verify this integration example; [host compatibility](hosts.md) distinguishes independent client tests from documented support.
