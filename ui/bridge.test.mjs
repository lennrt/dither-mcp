import test from 'node:test';
import assert from 'node:assert/strict';
import { App } from '@modelcontextprotocol/ext-apps';
import { AppBridge } from '@modelcontextprotocol/ext-apps/app-bridge';
import { StudioState } from './state.mjs';

function transports() {
  const a = { start: async () => {}, close: async () => {}, send: async (message) => { queueMicrotask(() => b.onmessage?.(structuredClone(message))); } };
  const b = { start: async () => {}, close: async () => {}, send: async (message) => { queueMicrotask(() => a.onmessage?.(structuredClone(message))); } };
  return [a, b];
}

test('official App and AppBridge deliver initial lifecycle data and proxy exact save arguments', async (t) => {
  const [viewTransport, hostTransport] = transports();
  const app = new App({ name: 'studio-test', version: '1' }, {}, { autoResize: false });
  const bridge = new AppBridge(null, { name: 'host-test', version: '1' }, { serverTools: {} }, { hostContext: { theme: 'light' } });
  const state = new StudioState();
  const request = { input: 'source.png', palette: 'mono', options: { algorithm: 'atkinson', width: 4, height: 2, seed: 7, effects: { noise: .2 } } };
  const initial = { content: [{ type: 'image', mimeType: 'image/png', data: 'iVBORw0KGgo=' }], structuredContent: { path: 'source.png', width: 4, height: 2, mime_type: 'image/png', recipe: { version: 1, palette: 'mono', options: request.options } }, _meta: { dither: { request, algorithms: [], palettes: [] } } };
  let ready;
  const initialReady = new Promise((resolve) => { ready = resolve; });
  app.ontoolinput = ({ arguments: value }) => state.setInput(value);
  app.ontoolresult = (value) => { state.hostResult(value); ready(); };
  app.onhostcontextchanged = (context) => { state.theme = context.theme; };
  app.ontoolcancelled = () => state.cancel('Host canceled');
  const calls = [];
  bridge.oncalltool = async (params) => { calls.push(params); return { content: [{ type: 'text', text: 'Saved' }], structuredContent: { path: params.arguments.output } }; };
  bridge.oninitialized = async () => { await bridge.sendToolInput({ arguments: request }); await bridge.sendToolResult(initial); };
  await bridge.connect(hostTransport); await app.connect(viewTransport); await initialReady;
  t.after(async () => { await app.close(); await bridge.close(); });
  state.canCallTools = !!app.getHostCapabilities()?.serverTools;
  assert.equal(state.canSave, true); assert.deepEqual(state.rendered.request, request);
  const operation = state.beginSave('exact.png');
  const saved = await app.callServerTool({ name: 'dither_render', arguments: operation.request });
  assert.equal(state.finishSave(operation, saved), true);
  assert.deepEqual(calls.map(({ name, arguments: args }) => ({ name, arguments: args })), [{ name: 'dither_render', arguments: { ...request, output: 'exact.png' } }]);
  const themeChanged = new Promise((resolve) => { app.addEventListener('hostcontextchanged', (context) => { state.theme = context.theme; resolve(); }); });
  bridge.setHostContext({ theme: 'dark' }); await themeChanged; assert.equal(state.theme, 'dark');
  const canceled = new Promise((resolve) => { app.addEventListener('toolcancelled', () => { state.cancel('Host canceled'); resolve(); }); });
  await bridge.sendToolCancelled({ reason: 'test cancellation' }); await canceled; assert.equal(state.canSave, false);
});

test('official host capability negotiation leaves unsupported tool actions disabled', async (t) => {
  const [viewTransport, hostTransport] = transports();
  const app = new App({ name: 'studio-test', version: '1' }, {}, { autoResize: false });
  const bridge = new AppBridge(null, { name: 'display-host', version: '1' }, {}, { hostContext: { theme: 'light' } });
  await bridge.connect(hostTransport); await app.connect(viewTransport);
  t.after(async () => { await app.close(); await bridge.close(); });
  const state = new StudioState(); state.canCallTools = !!app.getHostCapabilities()?.serverTools;
  state.setInput({ input: 'source.png', options: {} }); assert.equal(state.canPreview, false); assert.equal(state.canSave, false);
});
