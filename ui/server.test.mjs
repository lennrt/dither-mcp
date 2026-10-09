// Exercise the official app bridge against the compiled Go server over stdio.
import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, copyFile, readdir, readFile, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { Client } from '@modelcontextprotocol/client';
import { StdioClientTransport } from '@modelcontextprotocol/client/stdio';
import { App } from '@modelcontextprotocol/ext-apps';
import { AppBridge } from '@modelcontextprotocol/ext-apps/app-bridge';
import { StudioState } from './state.mjs';

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
function transports() {
  const app = { start: async () => {}, close: async () => {}, send: async (message) => { queueMicrotask(() => host.onmessage?.(structuredClone(message))); } };
  const host = { start: async () => {}, close: async () => {}, send: async (message) => { queueMicrotask(() => app.onmessage?.(structuredClone(message))); } };
  return [app, host];
}

test('official Apps bridge previews and saves through the real Go stdio server', { timeout: 30_000 }, async (t) => {
  const workspace = await mkdtemp(path.join(os.tmpdir(), 'dither-apps-integration-'));
  const client = new Client({ name: 'dither-apps-integration', version: '1' }, { capabilities: { extensions: { 'io.modelcontextprotocol/ui': { mimeTypes: ['text/html;profile=mcp-app'] } } } });
  const app = new App({ name: 'studio-test', version: '1' }, {}, { autoResize: false });
  const bridge = new AppBridge(null, { name: 'integration-host', version: '1' }, { serverTools: {} });
  t.after(async () => { await app.close(); await bridge.close(); await client.close(); await rm(workspace, { recursive: true, force: true }); });
  await copyFile(path.join(root, 'docs/assets/source/moon-garden.png'), path.join(workspace, 'source.png'));
  const transport = new StdioClientTransport({ command: path.join(root, 'bin', process.platform === 'win32' ? 'dither-mcp.exe' : 'dither-mcp'), args: ['mcp', '--root', workspace], stderr: 'pipe' });
  await client.connect(transport);
  const tools = await client.listTools();
  const studio = tools.tools.find((entry) => entry.name === 'dither_studio');
  assert.equal(studio?._meta?.ui?.resourceUri, 'ui://dither/studio.html');
  const resource = (await client.readResource({ uri: studio._meta.ui.resourceUri })).contents[0];
  assert.equal(resource.mimeType, 'text/html;profile=mcp-app');
  assert.ok(resource.text.includes('Dither studio'));
  assert.deepEqual(resource._meta.ui.csp.connectDomains, []);
  assert.deepEqual(resource._meta.ui.csp.resourceDomains, []);

  const state = new StudioState();
  const args = { input: 'source.png', palette: 'oat-and-ink', options: { algorithm: 'atkinson', width: 96 } };
  const initial = await client.callTool({ name: 'dither_studio', arguments: args });
  assert.equal(initial.isError, undefined);
  let ready;
  const received = new Promise((resolve) => { ready = resolve; });
  app.ontoolinput = ({ arguments: value }) => state.setInput(value);
  app.ontoolresult = (value) => { assert.equal(state.hostResult(value), true); ready(); };
  const calls = [];
  bridge.oncalltool = async (params) => {
    assert.ok(['dither_studio', 'dither_render'].includes(params.name));
    calls.push(structuredClone(params));
    return client.callTool(params);
  };
  bridge.oninitialized = async () => { await bridge.sendToolInput({ arguments: args }); await bridge.sendToolResult(initial); };
  const [view, host] = transports();
  await bridge.connect(host);
  await app.connect(view);
  await received;
  state.canCallTools = !!app.getHostCapabilities()?.serverTools;
  assert.equal(state.canSave, true);
  assert.equal(state.algorithms.length, 41);
  assert.equal(state.palettes.length, 256);
  assert.equal(state.rendered.width, 96);
  assert.deepEqual(await readdir(workspace), ['source.png']);

  const changed = structuredClone(state.draft);
  changed.options.algorithm = 'bayer-8';
  changed.options.strength = .7;
  state.edit(changed);
  assert.equal(state.canSave, false);
  const preview = state.beginPreview();
  const previewResult = await app.callServerTool({ name: 'dither_studio', arguments: preview.request });
  assert.equal(state.finishPreview(preview, previewResult), true);
  assert.deepEqual(await readdir(workspace), ['source.png']);
  const operation = state.beginSave('preview.png');
  assert.deepEqual(operation.request, { ...state.rendered.request, output: 'preview.png' });
  const saved = await app.callServerTool({ name: 'dither_render', arguments: operation.request });
  assert.equal(state.finishSave(operation, saved), true);
  const png = await readFile(path.join(workspace, 'preview.png'));
  assert.deepEqual(png, Buffer.from(previewResult.content.find((entry) => entry.type === 'image').data, 'base64'));
  const collision = state.beginSave('preview.png');
  const existing = await app.callServerTool({ name: 'dither_render', arguments: collision.request });
  assert.equal(existing.isError, true);
  assert.equal(state.finishSave(collision, existing), false);
  assert.deepEqual(await readFile(path.join(workspace, 'preview.png')), png);
  assert.deepEqual(calls.map((call) => call.name), ['dither_studio', 'dither_render', 'dither_render']);
});
